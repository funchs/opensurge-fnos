#!/bin/bash
# 生成 app.tgz —— fnOS 安装时把它解压到 TRIM_APPDEST。
# Docker 模式下里面只有 docker/（compose + 配置模板）和 ui/（桌面入口）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/meta.env"

# fpk / manifest / Docker 镜像共用同一版本号（无 v 前缀；镜像 tag 为 v${VERSION}）
if [ -z "${VERSION:-}" ] && [ -f "${PKG_DIR}/fnos/manifest" ]; then
    VERSION="$(grep -E '^version[[:space:]]*=' "${PKG_DIR}/fnos/manifest" | head -1 | sed 's/.*=[[:space:]]*//' | tr -d '[:space:]')"
fi
VERSION="${VERSION:-0.1.1}"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

mkdir -p "${WORK_DIR}/docker"
cp "${PKG_DIR}/fnos/docker/docker-compose.yaml" "${WORK_DIR}/docker/"
cp "${PKG_DIR}/fnos/docker/config.fnos.example.yaml" "${WORK_DIR}/docker/"

# 只替换 ${VERSION}。${DOCKER_MIRROR} / ${TRIM_PKGVAR} / ${TRIM_SERVICE_PORT}
# 必须原样留给 fnOS。
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

# 不用 cp -a，避免 macOS xattr 进包
mkdir -p "${WORK_DIR}/ui"
cp -R "${PKG_DIR}/fnos/ui/." "${WORK_DIR}/ui/"
find "${WORK_DIR}" \( -name '._*' -o -name '.DS_Store' \) -delete 2>/dev/null || true

export COPYFILE_DISABLE=1
export COPY_EXTENDED_ATTRIBUTES_DISABLE=1

# app.tgz 的 MD5 要写进 manifest 的 checksum，所以打包必须可复现：
# 否则源码没动、重跑一次 build 就换个 checksum，让 fpk 产生假 diff。
# 三个不确定性来源：gzip 的时间戳头、文件 mtime、目录遍历顺序。
find "${WORK_DIR}" -exec touch -t 202001010000.00 {} +
# gzip -n 不写时间戳和原文件名；bsdtar/GNU tar 都没有直接开关，所以分两步。
# --no-xattrs 去掉 macOS 的 com.apple.* pax 头（GNU tar 解包会刷警告）。
tar cf - --no-xattrs -C "${WORK_DIR}" docker ui | gzip -n9 > "${PKG_DIR}/app.tgz"

echo "Built app.tgz for ${FILE_PREFIX} ${VERSION} (image=v${VERSION})"
tar tzf "${PKG_DIR}/app.tgz"
