# OpenSurge fnOS 应用包

## 安装说明

### 方式一：通过飞牛商店本地安装（推荐）

1. **确认 NAS 架构**
   - x86 / Intel / AMD → 使用 `opensurge_0.1.1_x86.fpk`
   - arm64 / 瑞芯微等 → 使用 `opensurge_0.1.1_arm.fpk`

2. **下载镜像**（如果 NAS 无法直接拉取 ghcr.io）
   ```bash
   # 在有网络的机器上（按 NAS 架构选 platform）
   docker pull --platform linux/amd64 ghcr.io/funchs/opensurge-fnos:v0.1.1
   # 或
   docker pull --platform linux/arm64 ghcr.io/funchs/opensurge-fnos:v0.1.1

   docker save ghcr.io/funchs/opensurge-fnos:v0.1.1 -o opensurge-v0.1.1.tar
   ```

3. **传输文件到 NAS**
   - 将对应架构的 `.fpk` 上传到 NAS
   - 如需要，将 `opensurge-v0.1.1.tar` 也上传到 NAS

4. **加载 Docker 镜像**（如果在步骤 2 中导出了 tar）
   ```bash
   # SSH 登录 NAS
   docker load -i opensurge-v0.1.1.tar
   ```

5. **安装应用**
   - 打开飞牛商店
   - 点击「本地安装」
   - 选择 `opensurge_0.1.1_x86.fpk` 或 `opensurge_0.1.1_arm.fpk`
   - 按向导配置：
     - **Web GUI 端口**：默认 `61767`
     - **网关网卡名**：如 `eth0`（`ip -br link` 可查）
     - **NAS 局域网 IPv4**：fnOS 系统设置里显示的地址

6. **启动服务**
   - 安装完成后，在应用列表中启动 OpenSurge
   - 点击应用图标打开 Web 管理界面

### 方式二：手动安装（高级用户）

如果飞牛商店本地安装不可用，可以手动解包：

```bash
# 解压 fpk
mkdir opensurge-pkg
cd opensurge-pkg
tar xzf ../opensurge_0.1.1_x86.fpk

# 查看包内容
ls -la
# manifest / cmd / config / ui / wizard / OpenSurge.sc / ICON*.PNG / app.tgz

# app.tgz 含 docker 与 ui，安装时解压到 TRIM_APPDEST
tar tzf app.tgz
```

## 配置

### 首次配置

1. 安装向导会根据网卡名和局域网 IP 种子化 `config.yaml`
2. 也可手动参考 `docker/config.fnos.example.yaml` 调整：
   - `external-controller` 监听 `0.0.0.0:61767`（或你改的端口）
   - `secret` 设置 API 认证密钥
   - `tun.enable` 启用 TUN 模式
   - `dns` / `gateway` 与旁路由拓扑一致

### 客户端设置

OpenSurge 以旁路由网关模式运行，设备需手动配置：

- **网关**：NAS IP 地址
- **DNS**：NAS IP 地址（dnsmasq 监听 53 端口）

## 验证

访问 `http://{NAS_IP}:61767/enter`（端口以向导为准；桌面图标也会打开 `/enter`）。
直连根路径 `/` 时，Web GUI 会在需要登录时自动跳转到 `/enter` 建立会话。

进入后：

- 查看当前连接的设备
- 实时监控流量和连接
- 测试代理节点连通性
- 按设备策略分流（全局/直连/代理）

## 故障排查

```bash
# 查看容器状态
docker ps | grep opensurge

# 查看日志
docker logs opensurge

# 重启服务
docker compose -f /var/apps/opensurge/target/docker/docker-compose.yaml restart
# 实际路径以 fnOS 安装目录为准，常见在 /vol*/@appstore/opensurge 或类似位置

# 验证网络模式
docker inspect opensurge | grep NetworkMode
# 应该显示 "NetworkMode": "host"

# 验证权限
docker inspect opensurge | grep -A8 CapAdd
# 应包含 NET_ADMIN / NET_RAW / SYS_ADMIN
```

## 卸载

通过飞牛商店卸载时，向导可选择是否删除数据目录。默认保留配置与状态，便于重装恢复。

```bash
# 删除容器和镜像（数据目录按需）
docker rm -f opensurge 2>/dev/null || true
docker rmi ghcr.io/funchs/opensurge-fnos:v0.1.1
```

## 已知限制

- 不修改 fnOS 宿主机网卡配置（由系统管理）
- 不接管 DHCP（设备手动指定网关和 DNS）
- 需要客户端主动配置才能使用代理功能

## 技术栈

- mihomo (clash meta core) — 透明代理引擎
- dnsmasq — DNS 分流
- nftables — 流量劫持规则
- React + TypeScript — Web 管理界面
- Docker host 网络模式 — 直接访问宿主机网络栈
