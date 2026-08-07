# fnOS App Packaging

飞牛 NAS 应用商店打包目录。

## 快速构建

```bash
cd packaging/fnos
./build-fpk.sh
```

生成 `opensurge_0.1.1_all.fpk` (约 41KB)

## 目录结构

```
packaging/fnos/
├── fnos/                           # 应用包内容
│   ├── manifest                   # 应用元数据
│   ├── ICON.PNG                   # 128x128 应用图标
│   ├── ICON_256.PNG               # 256x256 应用图标
│   ├── cmd/
│   │   ├── main                   # 启停脚本
│   │   └── service-setup          # 安装后钩子
│   ├── docker/
│   │   ├── docker-compose.yaml    # Compose 模板
│   │   └── config.fnos.example.yaml  # mihomo 配置示例
│   ├── ui/
│   │   └── opensurge.Application  # 桌面入口定义
│   └── wizard                     # 安装向导配置
├── scripts/
│   ├── meta.env                   # 构建元数据
│   └── build.sh                   # app.tgz 构建脚本
├── build-fpk.sh                   # fpk 打包脚本
├── app.tgz                        # UI 和 docker 资源包
├── opensurge_0.1.1_all.fpk        # 最终安装包
└── INSTALL.md                     # 安装说明
```

## 工作流程

1. **准备资源**
   - 编辑 `fnos/manifest` 修改版本号、描述等
   - 更新 `fnos/docker/docker-compose.yaml` 镜像标签
   - 修改 `fnos/wizard` 调整安装向导

2. **构建应用包**
   ```bash
   # 构建 app.tgz（包含 docker 和 ui 目录）
   ./scripts/build.sh
   
   # 打包成 fpk
   ./build-fpk.sh
   ```

3. **安装测试**
   - 上传 fpk 到飞牛 NAS
   - 通过应用商店本地安装
   - 验证服务启动和 Web 界面

## 版本发布清单

发布新版本时需要同步修改：

- [ ] `fnos/manifest` - version 字段
- [ ] `fnos/docker/docker-compose.yaml` - 镜像标签
- [ ] `scripts/meta.env` - VERSION 变量
- [ ] `INSTALL.md` - 版本号和更新说明

## 参考

- [conversun/fnos-apps](https://github.com/conversun/fnos-apps) - 官方应用示例
- fnOS 应用打包规范：fpk 实际上是 tar.gz 格式，包含特定目录结构
