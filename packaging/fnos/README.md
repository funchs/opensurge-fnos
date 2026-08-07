# fnOS App Packaging

飞牛 NAS 应用商店（`.fpk`）**开发者打包**目录。

- **终端用户安装与使用** → 请读  
  **[docs/fnos-port/FPK-USER-GUIDE.md](../../docs/fnos-port/FPK-USER-GUIDE.md)**
- **纯 Docker Compose 部署** →  
  [docs/fnos-port/DEPLOY.md](../../docs/fnos-port/DEPLOY.md)

## 快速构建

版本号以 `fnos/manifest` 的 `version` 为准（当前 **0.1.1**），与镜像 tag
`v0.1.1` **保持一致**（`scripts/build.sh` 会把 compose 里的 `${VERSION}` 替换成同一数字）。

仓库根目录（推荐）：

```bash
make fpk          # app.tgz + x86 & arm 两个 fpk
make fpk-x86      # 只出 x86
make fpk-arm      # 只出 arm
make fpk-clean    # 删掉 app.tgz 和 *.fpk
```

或直接在本目录跑：

```bash
cd packaging/fnos
./scripts/gen-icon.sh    # 可选：从 assets 主图导出 ICON / ui 图标
./scripts/build.sh       # 生成 app.tgz（compose + 配置模板 + ui）
./build-fpk.sh all       # 生成 x86 + arm 两个 fpk
```

产物：

| 文件 | 说明 |
| --- | --- |
| `opensurge_0.1.1_x86.fpk` | Intel / AMD NAS |
| `opensurge_0.1.1_arm.fpk` | ARM64 NAS |
| `app.tgz` | 中间产物，被打进 fpk |

镜像（需另推）：`ghcr.io/funchs/opensurge-fnos:v0.1.1`  
`make docker-push-multiarch VERSION=v0.1.1`

打包是**可复现**的：固定 mtime + `gzip -n`，同样的输入产出同样的字节。
`app.tgz` 的 MD5 要写进 manifest 的 `checksum`，不固定的话源码没动、重跑一次
就会换 checksum，让签入的 fpk 产生假 diff。

macOS 打包时脚本会设 `COPYFILE_DISABLE=1` 并给 tar 传 `--no-xattrs`，
避免把 `._*` AppleDouble 和 `com.apple.*` 扩展头打进 fpk。

## 目录结构

```text
packaging/fnos/
├── fnos/
│   ├── manifest                 # appname / version / service_port …
│   ├── ICON.PNG                 # 64×64，应用中心
│   ├── ICON_256.PNG             # 256×256
│   ├── OpenSurge.sc             # 端口转发描述
│   ├── cmd/                     # 生命周期（含 fix-appcenter-icons）
│   ├── config/ privilege, resource
│   ├── docker/                  # compose 模板 + 示例配置
│   ├── ui/ config + images/     # 桌面入口 → /enter
│   └── wizard/ install|config|uninstall
├── assets/                      # 图标源稿与尺寸阶梯
├── scripts/
│   ├── build.sh                 # app.tgz
│   ├── gen-icon.sh
│   ├── fix-appcenter-icons.sh   # 可在 NAS 上手动修图标缓存
│   └── meta.env
├── build-fpk.sh
├── app.tgz
├── opensurge_0.1.1_*.fpk
├── INSTALL.md                   # 短版说明（指向 FPK-USER-GUIDE）
└── README.md                    # 本文件
```

## 发布清单

改版本时 **只改 `fnos/manifest` 的 version**，然后：

- [ ] `make fpk`（等价于 `./scripts/build.sh && ./build-fpk.sh all`）
- [ ] `make docker-push-multiarch VERSION=v<同一版本>`
- [ ] 更新 `docs/fnos-port/FPK-USER-GUIDE.md` / `INSTALL.md` 中的版本号
- [ ] 在真机或虚拟机：本地安装 → `/enter` → 启停网关冒烟

## 参考

- [conversun/fnos-apps](https://github.com/conversun/fnos-apps)
- fpk = tar.gz：`manifest`、`app.tgz`、`cmd`、`config`、`ui`、`wizard`、`ICON*.PNG`、`*.sc`
