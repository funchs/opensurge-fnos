#!/bin/bash
# 飞牛应用打包脚本 - 生成双平台 fpk 文件
#
# Usage: ./build-fpk.sh [x86|arm|all]
#   镜像是 amd64 + arm64 多架构，两个 fpk 只有 manifest 的 platform 字段不同。
#   x86 = Intel/AMD, arm = ARM64（瑞芯微 / 飞牛 ARM 机型）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

APP_NAME="opensurge"
VERSION="0.1.1"

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

    cp app.tgz "${build_dir}/"

    # cmd/*：生命周期脚本（可执行位在 tar 里保留）
    cp -a fnos/cmd/. "${build_dir}/cmd/"
    # main 被 fnOS 直接调用；wrapper 入口也必须可执行
    chmod a+x "${build_dir}/cmd/"* 2>/dev/null || true

    cp -a fnos/config "${build_dir}/"
    cp -a fnos/ui "${build_dir}/"
    cp -a fnos/wizard "${build_dir}/"

    # 端口转发描述（*.sc），没有也不失败
    if compgen -G "fnos/*.sc" > /dev/null; then
        cp fnos/*.sc "${build_dir}/"
    fi

    cp fnos/ICON.PNG "${build_dir}/"
    cp fnos/ICON_256.PNG "${build_dir}/"

    # 参考 conversun/fnos-apps：ui/images/256.png 与 ICON_256 对齐
    if [ -d "${build_dir}/ui/images" ]; then
        cp fnos/ICON_256.PNG "${build_dir}/ui/images/256.png"
    fi

    cp fnos/manifest "${build_dir}/manifest"
    sed -i.tmp "s/^checksum.*/checksum        = ${CHECKSUM}/" "${build_dir}/manifest"
    sed -i.tmp "s/^platform.*/platform        = ${platform}/" "${build_dir}/manifest"
    rm -f "${build_dir}/manifest.tmp"

    grep -q "^platform[[:space:]]*=[[:space:]]*${platform}$" "${build_dir}/manifest" \
        || { echo "Error: failed to set platform=${platform} in manifest" >&2; return 1; }
    grep -q "^checksum[[:space:]]*=[[:space:]]*${CHECKSUM}$" "${build_dir}/manifest" \
        || { echo "Error: failed to set checksum in manifest" >&2; return 1; }

    # 基本结构校验
    for need in app.tgz manifest cmd config ui wizard ICON.PNG ICON_256.PNG; do
        [ -e "${build_dir}/${need}" ] || { echo "Error: package missing ${need}" >&2; return 1; }
    done
}

BUILT=""

for PLATFORM in ${PLATFORMS}; do
    FPK_NAME="${APP_NAME}_${VERSION}_${PLATFORM}.fpk"
    echo "Building ${FPK_NAME}..."

    BUILD_DIR=$(mktemp -d)
    # shellcheck disable=SC2064
    trap "rm -rf '${BUILD_DIR}'" RETURN

    assemble_package "${BUILD_DIR}" "${PLATFORM}"

    rm -f "${SCRIPT_DIR}/${FPK_NAME}"
    tar czf "${SCRIPT_DIR}/${FPK_NAME}" -C "${BUILD_DIR}" .
    rm -rf "${BUILD_DIR}"
    trap - RETURN

    echo "✓ Package created: ${FPK_NAME} ($(du -h "${SCRIPT_DIR}/${FPK_NAME}" | cut -f1))"
    BUILT="${BUILT} ${FPK_NAME}"
done

echo "  Checksum (app.tgz): ${CHECKSUM}"
echo ""
echo "Installation:"
echo "  1. 按 NAS CPU 架构选择安装包:${BUILT}"
echo "     x86 = Intel/AMD, arm = ARM64 (瑞芯微/飞牛 ARM 机型)"
echo "  2. 飞牛商店 > 本地安装 > 上传 fpk"
echo "  3. 在向导里配置服务端口、网卡和局域网 IP，然后启动服务"
