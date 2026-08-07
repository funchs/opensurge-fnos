# OpenSurge 飞牛应用中心安装使用手册

面向 **在飞牛 NAS（fnOS）上通过应用中心本地安装 `.fpk` 包** 的用户。  
如果你更习惯纯 Docker Compose / SSH，请看 [DEPLOY.md](./DEPLOY.md)。

| 项 | 说明 |
| --- | --- |
| 产品 | OpenSurge fnOS Edition |
| 形态 | 旁路由透明代理网关（Docker host 网络） |
| 安装方式 | 飞牛应用中心 → 本地安装（`.fpk`） |
| 当前版本 | **0.1.1**（fpk 与 Docker 镜像 `v0.1.1` 一致） |
| 镜像 | `ghcr.io/funchs/opensurge-fnos:v0.1.1`（`linux/amd64` + `linux/arm64`） |
| Web GUI 默认端口 | **61767** |
| 推荐入口 | `http://<NAS-IP>:61767/enter` |
| 应用下载 | [GitHub Releases（最新）](https://github.com/funchs/opensurge-fnos/releases/latest) |

---

## 1. 下载应用安装包

普通用户请从 **GitHub Release** 下载，不要自己编译：

- **发布页（推荐）**  
  <https://github.com/funchs/opensurge-fnos/releases/latest>

- **当前版本 v0.1.1 直接下载**

| 架构 | 安装包 | 直链 |
| --- | --- | --- |
| Intel / AMD（N100、N305 等） | `opensurge_0.1.1_x86.fpk` | [下载](https://github.com/funchs/opensurge-fnos/releases/download/v0.1.1/opensurge_0.1.1_x86.fpk) |
| ARM64（瑞芯微等） | `opensurge_0.1.1_arm.fpk` | [下载](https://github.com/funchs/opensurge-fnos/releases/download/v0.1.1/opensurge_0.1.1_arm.fpk) |

> 仓库源码：<https://github.com/funchs/opensurge-fnos>  
> 默认开发分支：`fnos-port`（与 `master` 同步）。

下载后用浏览器或网盘传到本机，在飞牛网页「应用中心 → 本地安装」中选中该 `.fpk` 即可。

---

## 2. 你将得到什么

- 在 NAS 上跑 mihomo + dnsmasq + nftables 的旁路由网关
- 浏览器 Web GUI：总览、网络、订阅源、设备、策略、连通性、诊断
- 设备侧：把 **网关 + DNS** 指到 NAS IP 即可走透明代理（不接管主路由 DHCP）

**不会做：**

- 不改 fnOS 系统网卡设置（请在「系统设置 → 网络设置」里改 NAS IP）
- 默认不接管全屋 DHCP（需要的设备手动指网关/DNS）

---

## 3. 安装前准备

### 3.1 确认 NAS CPU 架构

| 架构 | 选用安装包 |
| --- | --- |
| Intel / AMD（多数 N100、N305 等） | `opensurge_0.1.1_x86.fpk` |
| ARM64（瑞芯微、部分 ARM 飞牛机型等） | `opensurge_0.1.1_arm.fpk` |

两个 fpk 内容相同，仅 `manifest` 的 `platform` 字段不同；镜像是多架构的，运行时按 NAS 架构拉对应层。

### 3.2 网络与权限前提

- NAS 已接入局域网，并有稳定 IPv4（建议固定 IP 或 DHCP 预留）
- 能访问 Docker Hub / ghcr.io（若不能，见下文「离线导入镜像」）
- 应用会以 Docker 项目方式运行，需要：
  - `network_mode: host`
  - `NET_ADMIN` / `NET_RAW` / `SYS_ADMIN`
  - `/dev/net/tun`

### 3.3 自己构建安装包（可选）

普通用户请直接使用 [§1 下载](#1-下载应用安装包) 的 Release。  
开发者可在本仓库构建：

```bash
cd packaging/fnos
./scripts/gen-icon.sh    # 可选：刷新图标
./scripts/build.sh       # 生成 app.tgz
./build-fpk.sh all       # 生成 x86 + arm 两个 fpk
```

产物位于 `packaging/fnos/opensurge_0.1.1_{x86,arm}.fpk`。

---

## 4. 安装步骤（应用中心）

### 4.1 拉取 Docker 镜像

**在线（推荐）**

安装后容器会按 compose 拉取：

```text
ghcr.io/funchs/opensurge-fnos:v0.1.1
```

package 为 public，一般 **无需登录 ghcr**。

**离线**

在能访问 ghcr 的机器上：

```bash
# 按 NAS 架构二选一
docker pull --platform linux/amd64 ghcr.io/funchs/opensurge-fnos:v0.1.1
# docker pull --platform linux/arm64 ghcr.io/funchs/opensurge-fnos:v0.1.1

docker save ghcr.io/funchs/opensurge-fnos:v0.1.1 -o opensurge-v0.1.1.tar
```

传到 NAS 后：

```bash
docker load -i opensurge-v0.1.1.tar
```

### 4.2 本地安装 fpk

1. 打开 **飞牛应用中心**
2. 左下角 **本地安装** / **手动安装**
3. 选择已下载的 `opensurge_0.1.1_x86.fpk` 或 `_arm.fpk`
4. 按安装向导填写：

| 字段 | 说明 | 建议 |
| --- | --- | --- |
| **Web GUI 端口** | 控制面监听端口 | 默认 `61767`，冲突再改 |
| **网关网卡名** | 旁路由绑定的网卡 | 向导默认 `eth0`；**若未改**，安装钩子会按 NetworkManager / 默认路由自动探测；也可填 `enp0s5` 等 |
| **NAS 局域网 IPv4** | 与浏览器访问 NAS 的 IP 一致 | 默认占位；**若未改**会自动探测；**必须与实际访问 IP 一致**，否则 Web 登录会失败 |

5. 完成安装后，在应用列表中 **启动** OpenSurge

安装钩子会：

- 在数据目录种子化 `config.yaml`（已有配置不会覆盖）
- 写入 compose 中的 `OPENSURGE_ADDR=0.0.0.0:<端口>`
- 尝试同步应用图标到包目录（见「图标」一节）

---

## 5. 首次打开 Web GUI

### 5.1 推荐入口

```text
http://<NAS-IP>:61767/enter
```

端口以向导填写为准。桌面图标默认也打开 **`/enter`**。

### 5.2 为什么不能只打开根路径？

控制面 API 需要 **session cookie**。  
`/enter` 在局域网模式（已配置 BaseURL / `gateway.lan_ip`）下会：

1. 校验浏览器 `Host` 是否允许  
2. 签发 `opensurge_session` cookie  
3. 跳转到控制台  

直接打开 `http://<NAS-IP>:61767/` 时，若尚未登录，前端收到 401 会 **自动再跳一次 `/enter`**。  
若仍提示「安全连接已过期」，请检查：

| 检查项 | 说明 |
| --- | --- |
| `gateway.lan_ip` | 必须等于你在地址栏用的 NAS IP（例如 `10.211.55.9`） |
| 端口 | 向导端口与 compose / 访问端口一致 |
| 容器是否运行 | `docker ps \| grep opensurge` |
| 浏览器 | 换无痕窗口，避免旧 cookie |

### 5.3 调试：手工签发 30 秒 bootstrap 链接

```bash
# 在装有仓库脚本的机器上，或把 scripts/fnos-gui-url.sh 拷到 NAS
./scripts/fnos-gui-url.sh opensurge http://<NAS-IP>:61767
```

原理：读取容器内 `state/store/control-token`，调用  
`POST /api/v1/session/bootstrap`。日常使用优先 `/enter`，不必每次跑脚本。

### 5.4 安全说明

- 61767 在 host 网络上对局域网可见，属于 **内网可信** 模型  
- **不要**把该端口端口转发到公网（HTTP 明文）  
- 运维级 `control-token` 在容器卷 `.../state/store/control-token`（权限 0600）

---

## 6. 日常使用

### 6.1 启动 / 停止

- 应用中心：打开 / 停止 OpenSurge  
- 或 Docker：对容器 `start` / `stop` / `restart`  
  容器名默认 `opensurge`

停止时应走正常 stop（会清理网关状态）；避免 `docker kill`，以免留下 nft 表或 `ip_forward`。

### 6.2 Web GUI 功能概览

| 页面 | 用途 |
| --- | --- |
| 总览 | 网关状态、流量趋势、快速启停入口 |
| 网络设置 | 拓扑与网关控制（宿主机网卡由 fnOS 管理，页内会有相应提示） |
| 代理与规则源 | 订阅 / profile 导入与刷新 |
| 设备 | 每设备策略登记与观察 |
| 策略 | 策略组、节点健康 |
| 连通性 | 本机 applied 路径探测 |
| 诊断 | doctor / 日志类信息 |

### 6.3 客户端（旁路由）

需要走代理的设备上设置：

- **网关** = NAS 局域网 IP  
- **DNS** = NAS 局域网 IP  

不要改主路由 DHCP 全局下发，除非你有意做全屋接管（进阶，见上游文档中的 DHCP 接管说明；fpk 默认旁路由种子配置）。

### 6.4 配置文件位置

以实际安装卷为准，常见：

| 内容 | 典型路径 |
| --- | --- |
| 应用数据（config / state） | `/var/apps/opensurge/var/` 或卷上 `@appdata/opensurge` |
| 运行目标（compose、ui） | `/vol1/@appcenter/opensurge/` 或 `/var/apps/opensurge/target/` |
| mihomo 配置 | 数据目录下 `config/config.yaml` |
| control-token | 数据目录 `state/store/control-token` |

首次安装若无配置，会从包内 `config.fnos.example.yaml` 种子化，并写入向导里的网卡名与局域网 IP。

### 6.5 修改端口 / 网卡 / IP

- **应用设置 / 配置向导**：可改 Web 端口；网卡与 IP 留空占位会重新自动探测，填写则覆盖  
- 改完通常需要 **重启容器 / 应用** 才完全生效  
- `gateway.lan_ip` 变更后，请用 **新 IP** 访问 `/enter`

---

## 7. 升级与重装

1. 应用中心卸载（可选择保留数据）  
2. 安装新 fpk（版本号与镜像 tag 应一致，当前均为 `0.1.1` / `v0.1.1`）  
3. 若镜像有更新：

   ```bash
   docker pull ghcr.io/funchs/opensurge-fnos:v0.1.1
   # 然后重启 OpenSurge 容器 / 应用
   ```

4. **图标**：桌面图标一般随 `ui/images` 更新；应用中心列表若仍显示旧图标，多为 UI 缓存——强制刷新浏览器或无痕窗口。磁盘上确认：

   ```bash
   ls -la /var/apps/opensurge/ICON*.PNG /vol1/@appcenter/opensurge/ICON*.PNG
   # 新 64px 约 3–6KB；旧 OS 字标约 20KB
   ```

   也可用包内脚本（安装后路径可能在 `@appcenter`）：

   ```bash
   bash /vol1/@appcenter/opensurge/cmd/fix-appcenter-icons
   # 或
   bash packaging/fnos/scripts/fix-appcenter-icons.sh   # 拷到 NAS 后执行
   ```

---

## 8. 卸载

1. 应用中心 → 卸载 OpenSurge  
2. 向导可选择是否删除数据目录（默认保留配置便于重装）  
3. 可选清理镜像：

   ```bash
   docker rm -f opensurge 2>/dev/null || true
   docker rmi ghcr.io/funchs/opensurge-fnos:v0.1.1
   ```

---

## 9. 故障排查

### 9.1 打开 GUI 提示「安全连接已过期」

| 步骤 | 操作 |
| --- | --- |
| 1 | 访问 `http://<NAS-IP>:<端口>/enter` |
| 2 | 确认 `config.yaml` 中 `gateway.lan_ip` = 地址栏 IP |
| 3 | `docker ps` / `docker logs opensurge` 看容器是否 healthy |
| 4 | 换无痕窗口，清掉旧 `opensurge_session` cookie |
| 5 | 镜像是否含 `/enter`：需使用已推送的 `v0.1.1` 新构建，`docker pull` 后再 `up -d` |

### 9.2 容器起不来 / 不断重启

```bash
docker logs opensurge --tail 100
```

常见原因：

- 缺少 `config.yaml` 或 `lan_ip` / 网卡名错误  
- 无 `SYS_ADMIN` 且宿主 `ip_forward=0`  
- 镜像未拉取成功  

### 9.3 网关启不来 / 无流量

```bash
docker exec opensurge omg doctor --config /etc/opensurge/config.yaml
docker exec opensurge omg status --config /etc/opensurge/config.yaml
ss -lnup | grep -E ':53|:61767'
```

- 53 被占：检查 fnOS 自身 DNS 与 `dns.listen` 是否只绑 NAS IP  
- TUN：确认设备映射 `/dev/net/tun`  
- 客户端是否已把网关/DNS 指到 NAS  

### 9.4 应用中心图标仍是旧的，但桌面已是新的

磁盘 `ICON.PNG` 已更新时，属于 **应用中心 UI 缓存**（飞牛已知现象）。  
强刷 / 无痕 / 重新登录；必要时执行第 6 节图标修复命令。

---

## 10. 路径与组件速查

| 组件 | 作用 |
| --- | --- |
| mihomo | 代理引擎、TUN、external-controller |
| dnsmasq | 旁路由下 DNS 转发（非全屋 DHCP 服务器） |
| nftables | NAT / 转发相关规则 |
| opensurge-control | Web GUI + Control API（默认 `0.0.0.0:61767`） |

安装包结构（简化）：

```text
opensurge_0.1.1_x86.fpk   # tar.gz
├── manifest
├── ICON.PNG / ICON_256.PNG      # 应用中心图标
├── cmd/                         # 生命周期 + fix-appcenter-icons
├── config/ privilege, resource
├── ui/ config + images          # 桌面入口
├── wizard/ install|config|uninstall
├── OpenSurge.sc
└── app.tgz                      # docker-compose + 配置模板 + ui
```

---

## 11. 相关文档

| 文档 | 内容 |
| --- | --- |
| [DEPLOY.md](./DEPLOY.md) | 纯 Docker Compose / 开发机推镜像 / 虚拟机验证 |
| [DESIGN.md](./DESIGN.md) | 移植方案与技术决策 |
| [packaging/fnos/README.md](../../packaging/fnos/README.md) | 开发者如何打 fpk |
| [仓库 README](../../README.md) | 产品能力与 CLI |

---

## 12. 快速检查清单

- [ ] 已从 [Releases](https://github.com/funchs/opensurge-fnos/releases/latest) 下载对应架构 fpk  
- [ ] 选对 x86 / arm fpk  
- [ ] 镜像 `v0.1.1` 已 pull 或 docker load  
- [ ] 向导端口、网卡、**lan_ip = 访问 IP**  
- [ ] 应用已启动，`docker ps` 可见 `opensurge`  
- [ ] 浏览器打开 `http://<NAS-IP>:61767/enter` 能进总览  
- [ ] 测试设备网关/DNS 已指向 NAS  
- [ ] 不要把 61767 暴露公网  
