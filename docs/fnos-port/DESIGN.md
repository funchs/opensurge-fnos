# OpenSurge for fnOS — 移植设计

> 状态：已确认，待实施
> 上游：https://github.com/YTwsy/OpenSurge-for-Mac (GPL-3.0, 默认分支 `master`)
> 本项目分支：`fnos-port`
> Go module：`open-mihomo-gateway`

## 目标

把 OpenSurge 的 Go 控制面 + Web GUI 跑在飞牛 NAS (fnOS，Debian 12 底) 上，
以 **旁路由网关** 形态提供按设备分流策略、设备发现和实时流量监控，
交付形式为 Docker 镜像 + `docker-compose.yml`。

### 已确认的三个方向

| 维度 | 选择 |
|---|---|
| 范围 | 移植 Go 控制面（保留按设备策略 / 设备发现 / Web GUI） |
| 网络角色 | 旁路由网关（设备手动指网关 + DNS，不接管 DHCP） |
| 交付 | Docker 镜像 + compose.yml，从 fnOS「Docker → 项目」导入 |

### 非目标（明确不做）

- DHCP 接管（`internal/dhcp` 保留代码但不启用）
- Linux 桌面托盘（对应上游 Swift 菜单栏）
- 原生 deb 包 / systemd unit
- 修改 fnOS 自身网卡配置（那是 fnOS 系统设置的职责）

---

## 核心洞察：缝已经开好了

上游控制面已经把网络能力做成了依赖注入点，**不需要重构**：

- `internal/controlapi/server.go:129-138` — `DiscoverNetwork` / `ListInterfaces` /
  `DiscoverNeighbors` / `PingRouter` 是 `controlapi.Options` 上的函数字段，
  默认值指向 `macosnetwork` 包。
- `internal/gateway/manager.go:104,110,133` — `pf.New(...)` 和 `macosnetwork.SystemProxy{}`
  从工厂函数返回。
- `internal/gateway/status.go:94` — `pf.New(...).Loaded()`。
- `internal/mihomo/manager.go:198` — `macosnetwork.LookupRoute(...)`。

因此移植策略是 **build tag 拆分**：保持包名和函数签名完全不变，
新增 `*_linux.go` 实现，调用方一行不改。

### 为什么不改包名 `macosnetwork`

名字在 Linux 上确实误导，但本项目要长期 `git rebase` 上游拿更新，
改名会让每次同步在 7 个 import 点全面冲突。**保留原名，在 `doc.go` 加一行说明。**

---

## 工作量（数出来的）

| 包 | 非测试 LOC | 处置 |
|---|---|---|
| `controlapi` | 5002 | 原样复用 |
| `mihomo` | 2492 | 原样复用 |
| `device` | 1099 | 原样复用 |
| `config` | 914 | 原样复用 |
| `gateway` | 846 | 原样复用 |
| `dhcp` | 282 | 保留不启用 |
| `runtime` / `process` / `doctor` / `webui` | ~500 | 原样复用 |
| **`internal/pf`** | 172 | **写 `*_linux.go` → nftables** |
| **`internal/macosnetwork`** | 776 | **写 `*_linux.go`，3 实现 + 6 stub** |
| **`internal/sysctl`** | 59 | **拆常量到 `key_{darwin,linux}.go`** |
| `apps/menubar` (Swift) | — | 删 |
| `packaging/launchd` `packaging/pkg-scripts` | — | 删 |
| `scripts/build-*installer.sh` 等 macOS 打包脚本 | — | 删 |

**净新增约 350 行 Go**，其余是删代码和加打包文件。

---

## 实施步骤

### 步骤 1 — `internal/sysctl` build tag 拆分（最小，先验证工具链）

把常量抽出来，两个文件各 5 行：

```go
// key_darwin.go
//go:build darwin
package sysctl
const keyIPForwarding = "net.inet.ip.forwarding"

// key_linux.go
//go:build linux
package sysctl
const keyIPForwarding = "net.ipv4.ip_forward"
```

`ipforward.go` 删掉原常量声明，其余不动。

**验收**：`GOOS=linux go build ./internal/sysctl/` 通过。

---

### 步骤 2 — `internal/macosnetwork` Linux 实现

现有 `network.go` / `neighbors.go` / `tun_routes.go` / `dhcp.go` / `system_proxy.go`
全部加 `//go:build darwin`，新增对应 `*_linux.go`。

必须匹配的现有类型（**不要改这些结构体**，Web GUI 的 JSON 依赖它们）：

```go
type Neighbor struct {
    IP        string `json:"ip"`
    MAC       string `json:"mac"`
    Interface string `json:"interface"`
}

type RouteSelection struct {
    Interface string `json:"interface"`
    Gateway   string `json:"gateway,omitempty"`
}

type Snapshot struct {
    NetworkService string   `json:"network_service"`
    Interface      string   `json:"interface"`
    IPv4Mode       string   `json:"-"`
    HardwareAddr   string   `json:"hardware_address,omitempty"`
    IPv4           string   `json:"ipv4,omitempty"`
    SubnetMask     string   `json:"subnet_mask,omitempty"`
    Router         string   `json:"router,omitempty"`
    DNS            []string `json:"dns"`
    IPv6Default    bool     `json:"ipv6_default"`
}

type ManualConfig struct { NetworkService, Interface, IPv4, SubnetMask, Router string; DNS []string }
type InterfaceOption struct { Interface, NetworkService string }
```

#### 2a. 真实现（3 个）

| 函数 | Linux 后端 | 备注 |
|---|---|---|
| `DiscoverNeighbors(ctx, ifaceName) ([]Neighbor, error)` | `ip neigh show dev <iface>` | 只取 `REACHABLE`/`STALE`/`DELAY` 且有 `lladdr` 的 IPv4 条目；MAC 统一大写以匹配上游 `parseNeighbors` 的行为 |
| `LookupRoute(ctx, destination) (RouteSelection, error)` | `ip -j route get <dest>` | JSON 输出，取 `dev` 和 `gateway` |
| `ListInterfaces(ctx) ([]InterfaceOption, error)` | `net.Interfaces()` 过滤 down/loopback | Linux 无「网络服务」概念，`NetworkService` 填成和 `Interface` 相同的值 |

另外 `Discover(ctx, service, iface) (Snapshot, error)` 需要**只读**实现（Web GUI 状态页要显示当前网络）：
`ip -j addr show dev <iface>` 拿 IPv4/掩码/MAC，`ip -j route show default` 拿 Router，
`/etc/resolv.conf` 拿 DNS，`IPv4Mode` 恒为 `IPv4ModeDHCP`（fnOS 管，控制面不判断）。

`PingRouter(ctx, router) error` — `ping -c1 -W1`，Linux 参数和 macOS 一致，可直接复用逻辑。

#### 2b. Stub（6 个，返回明确错误）

```go
var ErrManagedByFnOS = errors.New("网卡配置由 fnOS 系统设置管理，OpenSurge 不修改宿主机网络")
```

`SetManual` / `SetDHCP` / `VerifyManual` / `ProbeDHCPServers` /
`ServiceInterface` / `NetworkServiceForInterface` / `SystemProxy` 的三个方法
→ 返回 `ErrManagedByFnOS`。

`ValidateManual(cfg)` 是纯函数（无系统调用），**移到 `validate.go` 不加 build tag**，两平台共用。

> **设计理由**：这 6 个函数的作用是「把本机网卡改成静态 IP」和「设置系统代理」。
> 在 fnOS 上网卡归 fnOS 管，控制面碰它会和系统设置打架，而且容器内也改不动宿主机。
> Web GUI 的网络配置页会收到这个错误并显示提示，而不是崩溃。

**验收**：每个 parser 一个表驱动测试（`ip neigh` / `ip -j route get` / `ip -j addr` 输出格式固定，好测），
`GOOS=linux go test ./internal/macosnetwork/` 通过。

---

### 步骤 3 — `internal/pf` → nftables

`manager.go` / `template.go` 加 `//go:build darwin`，新增 `manager_linux.go` / `template_linux.go`。

保持 `Manager` 的方法集不变：`New(cfg, paths) Manager`、`Check()`、`WriteAnchor()`、
`Enabled()`、`Load(bool)`、`Unload(bool)`、`Loaded()`。

| 方法 | Linux 实现 |
|---|---|
| `Check()` | `exec.LookPath("nft")` |
| `WriteAnchor()` | 渲染 nft 规则写到 `paths` 下的 `opensurge.nft` |
| `Load(_)` | `nft -f <file>`（参数 `enablePF` 在 Linux 无对应，忽略） |
| `Unload(_)` | `nft delete table inet opensurge`（表不存在不算错） |
| `Loaded()` | `nft list tables` 输出含 `inet opensurge` |
| `Enabled()` | 恒 `true, nil`（nftables 无「全局开关」概念） |

规则内容（旁路由 + mihomo TUN 模式，转发由 mihomo 的 TUN 栈处理，nftables 只做出口 NAT 和放行）：

```
table inet opensurge {
    chain postrouting {
        type nat hook postrouting priority srcnat; policy accept;
        oifname != "<tun-name>" masquerade
    }
    chain forward {
        type filter hook forward priority filter; policy accept;
    }
}
```

**验收**：`GOOS=linux go build ./internal/pf/`；模板渲染有单测；
容器内 `nft list ruleset` 能看到 `table inet opensurge`。

---

### 步骤 4 — 删 macOS 专属产物

```
apps/menubar/
packaging/launchd/
packaging/pkg-scripts/
packaging/gui-components.plist
scripts/build-gui-installer.sh
scripts/build-menubar-app.sh
scripts/check-gui-packaging.sh
scripts/check-menubar.sh
scripts/notarize-gui-installer.sh
scripts/prepare-gui-release-deps.sh
scripts/uninstall-gui.sh
scripts/verify-unsigned-gui-installer.sh
```

`Makefile` 里对应的 target 一起删。`cmd/opensurge-helper`（macOS 特权 helper）
在 Docker 里容器已有 `NET_ADMIN`，不需要 helper 提权 —— 但 `controlapi` 的
`HelperClient` 分支还引用它，**先保留 `cmd/opensurge-helper` 不删**，
让 `DirectRunner` 路径生效即可。等确认无引用再删。

---

### 步骤 5 — Docker 打包

`Dockerfile`（多阶段）：

```
FROM golang:1.2x AS build      # 编译 opensurge-control + omg，webui/dist 已 embed，不需要 node
FROM debian:12-slim            # + nftables iproute2 iputils-ping ca-certificates
                               # + 固定版本 mihomo 二进制（从官方 release 下载，校验 sha256）
```

`docker-compose.yml` 要点：

```yaml
services:
  opensurge:
    network_mode: host                    # 旁路由必须
    cap_add: [NET_ADMIN, NET_RAW]
    devices: ["/dev/net/tun:/dev/net/tun"]
    sysctls: ...                          # 或在 entrypoint 里 sysctl -w（host 网络下生效于宿主机）
    volumes:
      - ./config:/etc/opensurge
      - ./state:/var/lib/opensurge
    restart: unless-stopped
```

架构：`linux/amd64` + `linux/arm64`（飞牛主流 N100/N305 是 amd64，也有 ARM 型号）。

`configs/fnos.example.yaml` — 旁路由预设，基于 `examples/config.same-lan.example.yaml` 改。

---

### 步骤 6 — 冒烟验证

`scripts/smoke-fnos.sh`：

1. `docker compose up -d`
2. `curl -fsS localhost:<port>/api/status` 返回 200
3. `docker exec ... nft list ruleset | grep -q 'table inet opensurge'`
4. `docker exec ... ip neigh show` 有输出 → 设备发现数据源可用
5. 手动：另一台设备把网关 + DNS 指向 NAS，能出网，Web GUI 里看得到这台设备

---

## 已知降级

| 项 | 影响 | 何时解决 |
|---|---|---|
| 不跑 dnsmasq，无 DHCP lease | `internal/device/scanner.go` 的 lease 数据源为空，**设备名降级为 IP + MAC + 厂商**，没有主机名 | 想要主机名时启用 `internal/dhcp` + 加一个 dnsmasq 容器 |
| 网络配置页不可用 | Web GUI 里改本机 IP / 系统代理的功能返回 `ErrManagedByFnOS` | 不打算解决，这是设计决策 |
| `Discover` 的 `IPv4Mode` 恒为 dhcp | 状态页不区分 fnOS 是静态还是 DHCP 拿的 IP | 需要时读 `/etc/network/interfaces` 或 netplan |

---

## 上游同步

`origin` 保持指向上游 `YTwsy/OpenSurge-for-Mac`，本项目在 `fnos-port` 分支开发。
拉更新：`git fetch origin && git rebase origin/master`。
因为所有改动都是**新增 `*_linux.go` 文件 + 给现有文件加一行 build tag**，
冲突面被压到最小。
