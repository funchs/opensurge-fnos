# OpenSurge fnOS 应用包

## 安装说明

### 方式一：通过飞牛商店本地安装（推荐）

1. **下载镜像**（如果 NAS 无法直接拉取 ghcr.io）
   ```bash
   # 在有网络的机器上
   docker pull ghcr.io/funchs/opensurge-fnos:v0.1.1
   docker save ghcr.io/funchs/opensurge-fnos:v0.1.1 -o opensurge-v0.1.1.tar
   ```

2. **传输文件到 NAS**
   - 将 `opensurge_0.1.1_all.fpk` 上传到 NAS
   - 如需要，将 `opensurge-v0.1.1.tar` 也上传到 NAS

3. **加载 Docker 镜像**（如果在步骤1中导出了 tar）
   ```bash
   # SSH 登录 NAS
   docker load -i opensurge-v0.1.1.tar
   ```

4. **安装应用**
   - 打开飞牛商店
   - 点击「本地安装」
   - 选择 `opensurge_0.1.1_all.fpk`
   - 按向导配置：
     - **服务端口**：默认 61767（Web 管理界面）
     - **配置目录**：存放 mihomo 配置文件，默认 `/volume1/app-data/opensurge/config`
     - **状态目录**：存放运行时数据，默认 `/volume1/app-data/opensurge/state`

5. **启动服务**
   - 安装完成后，在应用列表中启动 OpenSurge
   - 点击应用图标打开 Web 管理界面

### 方式二：手动安装（高级用户）

如果飞牛商店本地安装不可用，可以手动解包：

```bash
# 解压 fpk
mkdir opensurge-pkg
cd opensurge-pkg
tar xzf ../opensurge_0.1.1_all.fpk

# 查看包内容
ls -la
# manifest - 应用元数据
# cmd/main - 启停脚本
# cmd/service-setup - 安装钩子
# docker/ - compose 模板和配置示例
# ui/ - 桌面入口定义
# wizard - 安装向导配置
# app.tgz - UI 和 docker 资源包

# 手动部署 compose
mkdir -p /volume1/app-data/opensurge/{config,state}
cp docker/docker-compose.yaml /volume1/app-data/opensurge/
cd /volume1/app-data/opensurge
docker compose up -d
```

## 配置

### 首次配置

1. 上传 mihomo 配置文件到 `{配置目录}/config.yaml`
2. 参考 `docker/config.fnos.example.yaml` 调整配置：
   - `external-controller` 必须监听 `0.0.0.0:61767`
   - `secret` 设置 API 认证密钥
   - `tun.enable` 启用 TUN 模式
   - `dns` 配置 DNS 分流规则

### 客户端设置

OpenSurge 以旁路由网关模式运行，设备需手动配置：

- **网关**：NAS IP 地址
- **DNS**：NAS IP 地址（dnsmasq 监听 53 端口）

## 验证

访问 `http://{NAS_IP}:61767/dashboard` 进入 Web 管理界面：

- 查看当前连接的设备
- 实时监控流量和连接
- 测试代理节点连通性
- 按设备策略分流（全局/直连/代理）

## 故障排查

```bash
# 查看容器状态
docker ps | grep opensurge

# 查看日志
docker logs opensurge-opensurge-1

# 重启服务
docker compose -f /volume1/app-data/opensurge/docker-compose.yaml restart

# 验证网络模式
docker inspect opensurge-opensurge-1 | grep NetworkMode
# 应该显示 "NetworkMode": "host"

# 验证权限
docker inspect opensurge-opensurge-1 | grep -A5 CapAdd
# 应该包含 NET_ADMIN 和 SYS_ADMIN
```

## 卸载

通过飞牛商店卸载应用会自动停止容器，但不会删除数据目录。如需完全清理：

```bash
# 删除容器和镜像
docker compose -f /volume1/app-data/opensurge/docker-compose.yaml down
docker rmi ghcr.io/funchs/opensurge-fnos:v0.1.1

# 删除数据（谨慎操作）
rm -rf /volume1/app-data/opensurge
```

## 已知限制

- 不修改 fnOS 宿主机网卡配置（由系统管理）
- 不接管 DHCP（设备手动指定网关和 DNS）
- 需要客户端主动配置才能使用代理功能

## 技术栈

- mihomo (clash meta core) - 透明代理引擎
- dnsmasq - DNS 分流
- nftables - 流量劫持规则
- React + TypeScript - Web 管理界面
- Docker host 网络模式 - 直接访问宿主机网络栈
