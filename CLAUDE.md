# CLAUDE.md

## 这是什么

把 [OpenSurge-for-Mac](https://github.com/YTwsy/OpenSurge-for-Mac)（GPL-3.0）移植到
**飞牛 NAS (fnOS，Debian 12 底)**，以旁路由网关形态运行，Docker 交付。

**开工前先读 `docs/fnos-port/DESIGN.md`** —— 方案已确认，包含逐步实施计划、
必须匹配的类型签名、验收标准和已知降级。不要重新设计。

## 仓库现状

- 这是上游仓库的 clone，`origin` 指向上游，**默认分支是 `master`（不是 main）**
- 开发在 `fnos-port` 分支
- Go module 名：`open-mihomo-gateway`
- 不要 push 到 origin（那是别人的仓库）

## 移植的核心策略

**build tag 拆分，不重构。** 上游已经把网络能力做成了依赖注入点
（`controlapi.Options` 的函数字段、`gateway` 的工厂函数），所以：

- 保持包名和函数签名**完全不变**
- 现有 macOS 实现加 `//go:build darwin`
- 新增同签名的 `*_linux.go`
- 调用方一行不改

需要 Linux 实现的只有三个包：`internal/pf`（→ nftables）、
`internal/macosnetwork`（→ `ip` 命令）、`internal/sysctl`（→ 改常量）。
约 350 行新代码，其余是删 macOS 打包产物。

### 包名 `macosnetwork` 不改

名字在 Linux 上误导，但要长期 rebase 上游，改名会让每次同步在 7 个 import 点全冲突。

## 约束

- **不修改宿主机网卡配置**。fnOS 的网络归 fnOS 系统设置管，
  `SetManual` / `SetDHCP` / `SystemProxy` 等在 Linux 上返回 `ErrManagedByFnOS`。
- **不接管 DHCP**。旁路由模式，设备手动指网关 + DNS。
  `internal/dhcp` 保留代码但不启用。
- 不改 `Snapshot` / `Neighbor` / `RouteSelection` 等结构体 —— Web GUI 的 JSON 依赖它们。
- 新写的 parser 必须有表驱动测试（`ip neigh` / `ip -j route get` 输出格式固定）。

## 常用命令

```bash
GOOS=linux GOARCH=amd64 go build ./...      # 交叉编译验证
GOOS=linux go test ./internal/...           # Linux 单测
git fetch origin && git rebase origin/master # 同步上游
```

## 上游文档

`AGENTS.md` 和 `docs/` 下的中文文档是上游作者写的，讲的是 macOS 版的用法和架构，
读的时候注意区分——涉及 pf / launchd / networksetup 的部分本项目不适用。
