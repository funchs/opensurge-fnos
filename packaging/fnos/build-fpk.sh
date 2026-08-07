#!/bin/bash
# 飞牛应用打包脚本 - 生成双平台 fpk 文件
#
# Usage: ./build-fpk.sh [x86|arm|all]
#   镜像是 amd64 + arm64 多架构，两个 fpk 只有 manifest 的 platform 字段不同。
#   x86 = Intel/AMD, arm = ARM64（瑞芯微 / 飞牛 ARM 机型）
set -euo pipefail

# macOS tar 默认会把 AppleDouble (._*) 打进包，Linux/fnOS 解析时可能读错图标。
export COPYFILE_DISABLE=1
export COPY_EXTENDED_ATTRIBUTES_DISABLE=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

APP_NAME="opensurge"
# 读 manifest 版本，避免脚本与 manifest 漂移
VERSION="$(grep -E '^version[[:space:]]*=' fnos/manifest | head -1 | sed 's/.*=[[:space:]]*//' | tr -d '[:space:]')"
VERSION="${VERSION:-0.1.1}"

TARGET="${1:-all}"
case "${TARGET}" in
    x86)  PLATFORMS="x86" ;;
    arm)  PLATFORMS="arm" ;;
    all)  PLATFORMS="x86 arm" ;;
    *)    echo "Usage: $0 [x86|arm|all]" >&2; exit 1 ;;
esac

# 检查必要文件
if [ ! -f "app.tgz" ]; then
    echo "Error: app.tgz not found. Run scripts/build.sh first." >&2
    exit 1
fi

for required in \
    fnos/ICON.PNG \
    fnos/ICON_256.PNG \
    fnos/manifest \
    fnos/cmd \
    fnos/config \
    fnos/ui \
    fnos/wizard
do
    if [ ! -e "${required}" ]; then
        echo "Error: missing ${required}" >&2
        exit 1
    fi
done

# ICON.PNG 规范为 64x64（安装进度 / 应用中心列表）
ICON_W=$(magick identify -format '%w' fnos/ICON.PNG 2>/dev/null || echo 0)
ICON_H=$(magick identify -format '%h' fnos/ICON.PNG 2>/dev/null || echo 0)
if [ "${ICON_W}" != "64" ] || [ "${ICON_H}" != "64" ]; then
    echo "Warning: ICON.PNG is ${ICON_W}x${ICON_H}, official size is 64x64" >&2
fi

# 计算 app.tgz 的 MD5 校验和
if command -v md5sum >/dev/null 2>&1; then
    CHECKSUM=$(md5sum app.tgz | cut -d' ' -f1)
else
    CHECKSUM=$(md5 -q app.tgz)
fi

# 组装单个 platform 的 fpk 内容到临时目录
assemble_package() {
    local build_dir="$1"
    local platform="$2"

    mkdir -p "${build_dir}/cmd"

    # 用 cp 而不是 cp -a，避免把 macOS xattr / resource fork 带进包
    cp app.tgz "${build_dir}/"

    # cmd/*：生命周期脚本
    cp fnos/cmd/* "${build_dir}/cmd/"
    chmod a+x "${build_dir}/cmd/"* 2>/dev/null || true

    mkdir -p "${build_dir}/config" "${build_dir}/ui" "${build_dir}/wizard"
    cp -R fnos/config/. "${build_dir}/config/"
    cp -R fnos/ui/. "${build_dir}/ui/"
    cp -R fnos/wizard/. "${build_dir}/wizard/"

    # 清掉可能混入的 AppleDouble / .DS_Store
    find "${build_dir}" \( -name '._*' -o -name '.DS_Store' \) -delete 2>/dev/null || true

    if compgen -G "fnos/*.sc" > /dev/null; then
        cp fnos/*.sc "${build_dir}/"
    fi

    cp fnos/ICON.PNG "${build_dir}/ICON.PNG"
    cp fnos/ICON_256.PNG "${build_dir}/ICON_256.PNG"

    # 桌面入口图标：conversun 用 images/{0}.png → 64.png / 256.png
    mkdir -p "${build_dir}/ui/images"
    cp fnos/ICON.PNG "${build_dir}/ui/images/64.png"
    cp fnos/ICON_256.PNG "${build_dir}/ui/images/256.png"
    # 官方文档备选命名
    cp fnos/ICON.PNG "${build_dir}/ui/images/icon-64.png"
    cp fnos/ICON_256.PNG "${build_dir}/ui/images/icon-256.png"

    cp fnos/manifest "${build_dir}/manifest"
    # 用 sed 写 platform / checksum（在 macOS 与 Linux 都可用）
    if [[ "$(uname -s)" == "Darwin" ]]; then
        sed -i '' "s/^checksum.*/checksum        = ${CHECKSUM}/" "${build_dir}/manifest"
        sed -i '' "s/^platform.*/platform        = ${platform}/" "${build_dir}/manifest"
        sed -i '' "s/^version.*/version         = ${VERSION}/" "${build_dir}/manifest"
    else
        sed -i "s/^checksum.*/checksum        = ${CHECKSUM}/" "${build_dir}/manifest"
        sed -i "s/^platform.*/platform        = ${platform}/" "${build_dir}/manifest"
        sed -i "s/^version.*/version         = ${VERSION}/" "${build_dir}/manifest"
    fi

    grep -q "^platform[[:space:]]*=[[:space:]]*${platform}$" "${build_dir}/manifest" \
        || { echo "Error: failed to set platform=${platform} in manifest" >&2; return 1; }
    grep -q "^checksum[[:space:]]*=[[:space:]]*${CHECKSUM}$" "${build_dir}/manifest" \
        || { echo "Error: failed to set checksum in manifest" >&2; return 1; }

    for need in app.tgz manifest cmd config ui wizard ICON.PNG ICON_256.PNG; do
        [ -e "${build_dir}/${need}" ] || { echo "Error: package missing ${need}" >&2; return 1; }
    done
}

BUILT=""

for PLATFORM in ${PLATFORMS}; do
    FPK_NAME="${APP_NAME}_${VERSION}_${PLATFORM}.fpk"
    echo "Building ${FPK_NAME}..."

    BUILD_DIR=$(mktemp -d)

    assemble_package "${BUILD_DIR}" "${PLATFORM}"

    rm -f "${SCRIPT_DIR}/${FPK_NAME}"
    # 与 conversun/fnos-apps 一致：在包根目录 tar *，成员名无 ./ 前缀、无 ._*
    (
        cd "${BUILD_DIR}"
        # 同 scripts/build.sh：固定 mtime + gzip -n，让同样的输入产出同样的字节，
        # 避免源码没动、重跑一次就给签入的 fpk 造出假 diff。
        find . -exec touch -t 202001010000.00 {} +
        # --no-xattrs：macOS bsdtar 默认把 com.apple.quarantine / provenance 写成
        # pax 扩展头，GNU tar 解包时会刷一堆 "Ignoring unknown extended header keyword"。
        # shellcheck disable=SC2035
        tar cf - --no-xattrs * | gzip -n9 > "${SCRIPT_DIR}/${FPK_NAME}"
    )
    rm -rf "${BUILD_DIR}"

    # 校验：包内不得含 AppleDouble，且 ICON 存在
    if tar tzf "${SCRIPT_DIR}/${FPK_NAME}" | grep -E '(^|/)\._' >/dev/null; then
        echo "Error: fpk still contains AppleDouble ._* entries" >&2
        exit 1
    fi
    tar tzf "${SCRIPT_DIR}/${FPK_NAME}" | grep -qx 'ICON.PNG' \
        || { echo "Error: ICON.PNG missing or wrong path in fpk" >&2; exit 1; }
    tar tzf "${SCRIPT_DIR}/${FPK_NAME}" | grep -qx 'ICON_256.PNG' \
        || { echo "Error: ICON_256.PNG missing or wrong path in fpk" >&2; exit 1; }

    echo "✓ Package created: ${FPK_NAME} ($(du -h "${SCRIPT_DIR}/${FPK_NAME}" | cut -f1))"
    BUILT="${BUILT} ${FPK_NAME}"
done

echo "  Version: ${VERSION}"
echo "  Checksum (app.tgz): ${CHECKSUM}"
echo ""
echo "Installation:"
echo "  1. 先在飞牛商店卸载旧版 OpenSurge（同名会缓存旧图标）"
echo "  2. 上传新包:${BUILT}"
echo "     x86 = Intel/AMD, arm = ARM64"
echo "  3. 本地安装（请确认文件名含版本 ${VERSION}）"
