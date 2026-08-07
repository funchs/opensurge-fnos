# fnOS App Packaging

飞牛 NAS 应用商店打包目录。

## 快速构建

```bash
cd packaging/fnos
./scripts/build.sh    # 生成 app.tgz（docker + ui）
./build-fpk.sh        # 默认同时打 x86 / arm 两个包
# 或只打一个架构：
# ./build-fpk.sh x86
# ./build-fpk.sh arm
```

产物：

- `opensurge_0.1.1_x86.fpk` — Intel/AMD NAS
- `opensurge_0.1.1_arm.fpk` — ARM64 NAS（瑞芯微等）

两个 fpk 共享同一份多架构 Docker 镜像（`ghcr.io/funchs/opensurge-fnos:v0.1.1`），差异只在 `manifest` 的 `platform` 字段。

## 目录结构

```
packaging/fnos/
├── fnos/                              # 应用包源内容
│   ├── manifest                       # 应用元数据
│   ├── ICON.PNG                       # 90x90 应用图标
│   ├── ICON_256.PNG                   # 256x256 应用图标
│   ├── OpenSurge.sc                   # 端口转发描述
│   ├── cmd/                           # 生命周期脚本（自包含，无外部 shared）
│   │   ├── main                       # 启停 / status
│   │   ├── common / installer         # 框架
│   │   ├── service-setup              # 安装钩子（种子配置、写端口）
│   │   └── {install,upgrade,config,uninstall}_{init,callback}
│   ├── config/
│   │   ├── privilege
│   │   └── resource                   # docker-project + port-config
│   ├── docker/
│   │   ├── docker-compose.yaml        # Compose 模板
│   │   └── config.fnos.example.yaml   # mihomo 配置示例
│   ├── ui/
│   │   ├── config                     # 桌面入口（opensurge.Application）
│   │   └── images/{64,256}.png
│   └── wizard/{install,config,uninstall}
├── scripts/
│   ├── meta.env
│   └── build.sh                       # 生成 app.tgz
├── build-fpk.sh                       # 打双平台 fpk
├── app.tgz                            # 构建产物
├── opensurge_0.1.1_x86.fpk
├── opensurge_0.1.1_arm.fpk
└── INSTALL.md
```

## 工作流程

1. **准备资源**
   - 改 `fnos/manifest` 版本 / 描述
   - 改 `fnos/docker/docker-compose.yaml` 镜像与能力
   - 改 `fnos/wizard/*` 安装向导字段

2. **构建**
   ```bash
   ./scripts/build.sh
   ./build-fpk.sh all
   ```

3. **安装测试**
   - 上传对应架构的 fpk 到飞牛 NAS
   - 应用商店 → 本地安装
   - 向导填写端口、网卡名、局域网 IP
   - 启动后打开 Web GUI

## 版本发布清单

- [ ] `fnos/manifest` — version
- [ ] `fnos/docker/docker-compose.yaml` — 镜像标签（`v${VERSION}` 由 build.sh 替换）
- [ ] `scripts/meta.env` / `build-fpk.sh` — VERSION
- [ ] `INSTALL.md` — 版本号
- [ ] 多架构镜像推送到 `ghcr.io/funchs/opensurge-fnos`

## 参考

- [conversun/fnos-apps](https://github.com/conversun/fnos-apps) — 官方应用示例与 `scripts/build-fpk.sh`
- fpk 实际是 tar.gz，根目录含 `manifest` / `app.tgz` / `cmd` / `config` / `ui` / `wizard` / 图标 / `*.sc`
