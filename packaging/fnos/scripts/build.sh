#!/bin/bash
# 生成 app.tgz —— fnOS 安装时把它解压到 TRIM_APPDEST。
# Docker 模式下里面只有 docker/（compose + 配置模板）和 ui/（桌面入口）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/meta.env"

# fpk / manifest 版本（无 v 前缀）
if [ -z "${VERSION:-}" ] && [ -f "${PKG_DIR}/fnos/manifest" ]; then
    VERSION="$(grep -E '^version[[:space:]]*=' "${PKG_DIR}/fnos/manifest" | head -1 | sed 's/.*=[[:space:]]*//' | tr -d '[:space:]')"
fi
VERSION="${VERSION:-0.1.2}"

# Docker 镜像 tag 可与 fpk 版本解耦（图标热修时只升 fpk 版本）
IMAGE_VERSION="${IMAGE_VERSION:-0.1.1}"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

mkdir -p "${WORK_DIR}/docker"
cp "${PKG_DIR}/fnos/docker/docker-compose.yaml" "${WORK_DIR}/docker/"
cp "${PKG_DIR}/fnos/docker/config.fnos.example.yaml" "${WORK_DIR}/docker/"

# 只替换 ${VERSION} → 镜像 tag 用 IMAGE_VERSION。
# ${DOCKER_MIRROR} / ${TRIM_PKGVAR} / ${TRIM_SERVICE_PORT} 必须原样留给 fnOS。
sed -i.bak "s|\${VERSION}|${IMAGE_VERSION}|g" "${WORK_DIR}/docker/docker-compose.yaml"
rm -f "${WORK_DIR}/docker/docker-compose.yaml.bak"

if grep -q '\${VERSION}' "${WORK_DIR}/docker/docker-compose.yaml"; then
    echo "Error: \${VERSION} substitution failed" >&2
    exit 1
fi
for keep in 'DOCKER_MIRROR' 'TRIM_PKGVAR' 'TRIM_SERVICE_PORT'; do
    grep -q "\${${keep}" "${WORK_DIR}/docker/docker-compose.yaml" \
        || { echo "Error: \${${keep}} was lost from compose" >&2; exit 1; }
done

# 不用 cp -a，避免 macOS xattr 进包
mkdir -p "${WORK_DIR}/ui"
cp -R "${PKG_DIR}/fnos/ui/." "${WORK_DIR}/ui/"
find "${WORK_DIR}" \( -name '._*' -o -name '.DS_Store' \) -delete 2>/dev/null || true

export COPYFILE_DISABLE=1
export COPY_EXTENDED_ATTRIBUTES_DISABLE=1
tar czf "${PKG_DIR}/app.tgz" -C "${WORK_DIR}" docker ui

echo "Built app.tgz for ${FILE_PREFIX} fpk=${VERSION} image=v${IMAGE_VERSION}"
tar tzf "${PKG_DIR}/app.tgz"
