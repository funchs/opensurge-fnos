# 在飞牛 NAS (fnOS) 上部署

> 前置：读过 `DESIGN.md`。这是旁路由形态——**设备手动指网关 + DNS，不接管 DHCP，
> 不修改 fnOS 的网卡配置**。

## 一、准备镜像

镜像只出 `linux/amd64`（飞牛主流的 N100 / N305 都是 x86）。

```bash
# 在开发机上构建并推到你的 registry
make docker-push IMAGE=ghcr.io/你的账号/opensurge-fnos:v0.1.0
```

`IMAGE` 不给就用 `ghcr.io/OWNER/opensurge-fnos:latest`——那是占位，必须改。

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

```
http://<NAS-IP>:61767
```

**控制面没有密码**（上游的设计是只监听 loopback + 一次性链接，这里为了局域网可用改成了
`0.0.0.0`）。别把 61767 端口映射到公网，也别在不可信的网络里这么跑。
要收紧就把 compose 里的 `OPENSURGE_ADDR` 改回 `127.0.0.1:61767`，用 SSH 隧道访问。

## 五、设备侧

在要走代理的设备上手动设置：

- 网关 → NAS 的 IP
- DNS → NAS 的 IP

不要改路由器的 DHCP 下发（那会让全屋设备都走过来）。

## 排查

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
