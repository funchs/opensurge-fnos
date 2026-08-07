# OpenSurge 配置文件目录

此目录用于存放应用安装时需要的默认配置文件。

fnOS 打包规范要求每个应用必须包含 `config/` 目录，即使为空。

对于 OpenSurge，实际的 mihomo 配置文件由用户在安装后上传到数据目录（`TRIM_PKGVAR/config/config.yaml`），而不是打包到应用中。
