#!/bin/bash
# 生成 app.tgz —— fnOS 安装时把它解压到 TRIM_APPDEST。
# Docker 模式下里面只有 docker/（compose + 配置模板）和 ui/（桌面入口）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/meta.env"

# 镜像 tag 是 v 前缀（ghcr.io/funchs/opensurge-fnos:v0.1.1），
# 而 manifest / fpk 文件名里的版本不带 v，所以这里只传裸版本号。
VERSION="${VERSION:-0.1.1}"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

mkdir -p "${WORK_DIR}/docker"
cp "${PKG_DIR}/fnos/docker/docker-compose.yaml" "${WORK_DIR}/docker/"
cp "${PKG_DIR}/fnos/docker/config.fnos.example.yaml" "${WORK_DIR}/docker/"

# 只替换 ${VERSION}。${DOCKER_MIRROR} / ${TRIM_PKGVAR} / ${TRIM_SERVICE_PORT}
# 必须原样留着——那是 fnOS 起 compose 时才注入的。
sed -i.bak "s|\${VERSION}|${VERSION}|g" "${WORK_DIR}/docker/docker-compose.yaml"
rm -f "${WORK_DIR}/docker/docker-compose.yaml.bak"

if grep -q '\${VERSION}' "${WORK_DIR}/docker/docker-compose.yaml"; then
    echo "Error: \${VERSION} substitution failed" >&2
    exit 1
fi
for keep in 'DOCKER_MIRROR' 'TRIM_PKGVAR' 'TRIM_SERVICE_PORT'; do
    grep -q "\${${keep}" "${WORK_DIR}/docker/docker-compose.yaml" \
        || { echo "Error: \${${keep}} was lost from compose" >&2; exit 1; }
done

cp -a "${PKG_DIR}/fnos/ui" "${WORK_DIR}/ui"

tar czf "${PKG_DIR}/app.tgz" -C "${WORK_DIR}" docker ui

echo "Built app.tgz for ${FILE_PREFIX} ${VERSION}"
tar tzf "${PKG_DIR}/app.tgz"
