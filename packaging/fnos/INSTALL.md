# OpenSurge fnOS 应用包 — 短说明

> **完整用户手册（推荐）**  
> [docs/fnos-port/FPK-USER-GUIDE.md](../../docs/fnos-port/FPK-USER-GUIDE.md)

## 一分钟安装

1. 选架构包：`opensurge_0.1.3_x86.fpk` 或 `opensurge_0.1.3_arm.fpk`
2. 飞牛应用中心 → **本地安装**
3. 向导：端口（默认 61767）、网卡、**NAS 局域网 IP（须与浏览器访问 IP 一致）**
4. 启动应用
5. 浏览器打开：`http://<NAS-IP>:61767/enter`

镜像：`ghcr.io/funchs/opensurge-fnos:v0.1.3`（与 fpk 版本号一致）。  
若 NAS 拉不到 ghcr，在可联网机器上 `docker pull` + `docker save`，再在 NAS 上 `docker load`。

## 旁路由客户端

设备侧手动设置：

- 网关 = NAS IP  
- DNS = NAS IP  

## 常见问题

| 现象 | 处理 |
| --- | --- |
| 「安全连接已过期」 | 用 `/enter`；检查 `gateway.lan_ip`；`docker pull` 新镜像后重建容器 |
| 应用中心图标仍旧、桌面已新 | 强刷/无痕；或见手册中的 `fix-appcenter-icons` |
| 容器反复退出 | `docker logs opensurge`；检查配置与 `ip_forward` |

## 相关链接

- [飞牛应用中心安装使用手册](../../docs/fnos-port/FPK-USER-GUIDE.md)
- [Docker / 开发部署](../../docs/fnos-port/DEPLOY.md)
- [打包说明（开发者）](./README.md)
