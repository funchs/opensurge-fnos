# 在飞牛 NAS (fnOS) 上部署（Docker Compose）

> 前置：读过 `DESIGN.md`。这是旁路由形态——**设备手动指网关 + DNS，不接管 DHCP，
> 不修改 fnOS 的网卡配置**。

**普通用户更推荐用飞牛应用中心安装 `.fpk`：**  
→ **[飞牛应用中心安装使用手册（FPK-USER-GUIDE.md）](./FPK-USER-GUIDE.md)**

本文面向：**SSH / Docker 项目导入、开发机推镜像、虚拟机验证**。

## 一、准备镜像

镜像同时提供 `linux/amd64` 和 `linux/arm64`，`docker pull` 会自动选对应架构：

```
ghcr.io/funchs/opensurge-fnos:v0.1.2             # 推荐：固定版本，多架构
ghcr.io/funchs/opensurge-fnos:latest             # 跟着主线走
ghcr.io/funchs/opensurge-fnos:v0.1.0             # 仅 amd64（多架构支持之前的版本）
```

- **amd64** —— 飞牛主流的 N100 / N305 等 x86 机型
- **arm64** —— 飞牛 ARM 版（瑞芯微 / 全志 / 鲲鹏等），以及**在 Apple Silicon 虚拟机里
  跑 fnOS ARM 做验证**的场景（见文末「先在虚拟机里验证」）

生产环境用 `v0.1.2` 这样的固定版本，别用 `latest`——它会随下次推送变化。

package 已设为 **public**，NAS 上直接 `docker compose pull` 即可，**不需要登录**。

镜像里不含任何私密信息：只打包了一份占位示例配置（`/usr/share/opensurge/`），
真实配置和 `control-token` 都在运行时挂载的卷里。

### 重新构建并推送

```bash
# 开发机上，token 需要 write:packages；用 gh CLI 最省事：
gh auth refresh -h github.com -s write:packages
gh auth token | docker login ghcr.io -u funchs --password-stdin

make docker-push                            # 单架构（构建机的架构）
make docker-push-multiarch VERSION=v0.1.2   # 双架构 + 合并 manifest，并把 latest 指过去
```

> **为什么多架构不用 `docker buildx --platform a,b` 一步到位**：buildx 的容器化
> builder 有自己的 buildkitd 配置，**继承不到 daemon 的 registry mirror**，
> 国内网络下拉 `golang` / `debian` 基础镜像会 EOF 失败。所以改成用默认 builder
> 分架构构建（走得到 mirror），再用 `imagetools create` 合并 manifest。

## 二、NAS 上准备目录

```bash
mkdir -p /vol1/docker/opensurge/{config,state}
cd /vol1/docker/opensurge
```

把 `examples/config.fnos.example.yaml` 复制成 `config/config.yaml`，改三处：

| 字段 | 怎么填 |
|---|---|
| `gateway.interface` / `gateway.upstream_interface` | `ip -br link` 看网卡名，两个填一样的 |
| `gateway.lan_ip` / `dns.listen` | NAS 在局域网里的 IPv4，两个填一样的 |

## 三、compose

把仓库的 `docker-compose.yml` 放到同目录，改掉 `image:`，然后：

- **走 fnOS 界面**：Docker → 项目 → 新建 → 导入 `docker-compose.yml`
- **走 SSH**：`docker compose up -d`

关键字段和它们为什么必须在：

| 字段 | 原因 |
|---|---|
| `network_mode: host` | 旁路由要看见真实网卡、要建 TUN、nftables 规则要作用在宿主网络栈上 |
| `cap_add: NET_ADMIN` | nftables 写规则、TUN 建接口 |
| `cap_add: NET_RAW` | ping 探测网关 |
| `cap_add: SYS_ADMIN` | **只为**把 `/proc/sys` remount 成可写。Docker 默认挂只读，网关启动时要写 `net.ipv4.ip_forward`；host 网络下不能用 `sysctls:`，`privileged` 又太宽 |
| `devices: /dev/net/tun` | mihomo 的 TUN 栈 |

## 四、打开控制面

推荐访问：

```text
http://<NAS-IP>:61767/enter
```

在 **BaseURL / 局域网模式**下，`/enter` 会校验 Host 后签发 `opensurge_session`
cookie 并跳到控制台。直接打开 `/` 时，前端若收到 401 会自动再跳一次 `/enter`。

compose 需监听 `0.0.0.0`，且配置中 `gateway.lan_ip`（或环境变量
`OPENSURGE_BASE_URL`）与浏览器地址一致，否则 Host 校验失败。

手工签发 30 秒 bootstrap（调试）：

```bash
./scripts/fnos-gui-url.sh
```

### 安全边界

- **局域网可信**：`/enter` 允许访问该 Host 的设备进入控制台；**不要端口转发到公网**。
- `control-token` 在 `state/store/control-token`（0600），用于运维 bootstrap API。
- 更严：只绑 `127.0.0.1` + SSH 隧道。

> 上游默认只绑 loopback；本项目用可选 `BaseURL` 支持「控制面在 NAS、浏览器在
> 另一台机器」。entrypoint 会从 `gateway.lan_ip` 自动推导 BaseURL。

## 五、设备侧

在要走代理的设备上手动设置：

- 网关 → NAS 的 IP
- DNS → NAS 的 IP

不要改路由器的 DHCP 下发（那会让全屋设备都走过来）。

## 排查

**打开 GUI 提示安全连接过期** —— 用 `/enter`；核对 `lan_ip`；`docker pull` 最新
`v0.1.2` 后重建容器。详见 [FPK-USER-GUIDE.md §8.1](./FPK-USER-GUIDE.md)。

**`ip_forward` 写入失败** —— compose 里少了 `cap_add: SYS_ADMIN`，或者 fnOS 的 Docker
版本不允许 remount。容器日志会直接打出提示。

**dnsmasq 起不来 / 53 端口被占** —— fnOS 自己可能有 DNS 服务占着 `0.0.0.0:53`。
本项目的 dnsmasq 用 `bind-interfaces` 只绑 `dns.listen` 那一个 IP，正常不冲突；
真冲突了先 `ss -lnup | grep :53` 看是谁占的。

**设备名只显示 IP + MAC，没有主机名** —— 已知降级，旁路由不接管 DHCP 就没有 lease
数据源。见 `DESIGN.md` 的「已知降级」。

**Web GUI 的网络配置页报 `网卡配置由 fnOS 系统设置管理`** —— 这是设计决策，不是 bug。
改 NAS 的 IP 请去 fnOS 系统设置。

## 停止

```bash
docker compose down
```

entrypoint 收到 SIGTERM 会先跑 `omg stop`，把 mihomo / dnsmasq 进程、nftables 表和
`ip_forward` 都还原。**别用 `docker kill`**——那会跳过清理，宿主上会留下
`table inet opensurge` 和打开的 `ip_forward`。真留下了就手动清：

```bash
nft delete table inet opensurge
sysctl -w net.ipv4.ip_forward=0
rm /vol1/docker/opensurge/state/state.json
```

---

## 附：先在虚拟机里验证（推荐首次部署前做）

不想拿真机冒险的话，可以先在 Apple Silicon Mac 上装一台 fnOS ARM 虚拟机跑一遍。
这比装普通 Debian 虚拟机更有价值——能一并验到 fnOS 特有的部分（Docker 项目导入、
fnOS 网络设置与 `ErrManagedByFnOS` 的交互）。

飞牛官方的 ARM 版明确支持苹果 M 系列的虚拟化环境，下载页有一个
**「UEFI ARM 安装镜像 ISO」**（标注「支持 此芯 P1 / 苹果M系列（虚拟化）」）：
<https://www.fnnas.com/download-arm>

> ARM 版目前是公测版，官方声明「请勿用于生产环境或存储唯一重要数据」。
> 拿来做一次性验证正合适，别拿它当真机用。

### 虚拟机配置

```bash
brew install --cask utm     # 免费开源
```

| 项 | 选什么 | 为什么 |
|---|---|---|
| 类型 | **虚拟化**（Virtualize），不是模拟 | 选错就是龟速，等于白验 |
| 内存 / CPU / 磁盘 | 4 GB / 4 核 / 40 GB | 够装系统 + 镜像 + 日志 |
| 网络 | **桥接**（Bridged） | 旁路由验的就是网络转发行为，NAT 会掩盖问题 |

桥接后虚拟机会从你家路由器拿到真实局域网 IP，和真机环境一致。

### 在虚拟机里跑

和真机流程完全相同，只是 `config.yaml` 里的网卡名和 IP 换成虚拟机的
（`ip -br addr` 查）。镜像是多架构的，`docker compose pull` 会自动拉 arm64 那份。

```bash
docker compose up -d
./scripts/smoke-fnos.sh --start-gateway
```

搞砸了删掉虚拟机重建即可，真机零风险。

### 虚拟机验不到的部分

- **真实硬件网卡的驱动行为**（虚拟机用 virtio）
- **x86 与 arm64 的架构差异**——不过本项目的 Go 代码不含架构相关逻辑，
  依赖的 `nft` / `ip` / `dnsmasq` 也都是 Debian 同源包
- **你家网络的实际拓扑**（虚拟机和真机可能在不同网段）
