#!/bin/bash
# 飞牛应用打包脚本 - 生成双平台 fpk 文件
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

APP_NAME="opensurge"
VERSION="0.1.1"

# 平台参数：x86 | arm | all（默认 all）
# 镜像是 amd64 + arm64 多架构，两个 fpk 只有 manifest 的 platform 字段不同。
TARGET="${1:-all}"
case "${TARGET}" in
    x86)  PLATFORMS="x86" ;;
    arm)  PLATFORMS="arm" ;;
    all)  PLATFORMS="x86 arm" ;;
    *)    echo "Usage: $0 [x86|arm|all]" >&2; exit 1 ;;
esac

# 检查必要文件
if [ ! -f "app.tgz" ]; then
    echo "Error: app.tgz not found. Run scripts/build.sh first."
    exit 1
fi

if [ ! -f "fnos/ICON.PNG" ] || [ ! -f "fnos/ICON_256.PNG" ]; then
    echo "Error: Icon files missing."
    exit 1
fi

if [ ! -d "fnos/config" ]; then
    echo "Error: config/ directory missing."
    exit 1
fi

if [ ! -d "fnos/cmd" ]; then
    echo "Error: cmd/ directory missing."
    exit 1
fi

# 计算 app.tgz 的 MD5 校验和
if command -v md5sum >/dev/null 2>&1; then
    CHECKSUM=$(md5sum app.tgz | cut -d' ' -f1)
else
    CHECKSUM=$(md5 -q app.tgz)
fi

BUILT=""

for PLATFORM in ${PLATFORMS}; do
    FPK_NAME="${APP_NAME}_${VERSION}_${PLATFORM}.fpk"
    echo "Building ${FPK_NAME}..."

    BUILD_DIR=$(mktemp -d)

    # 复制所有文件到构建目录
    cp app.tgz "${BUILD_DIR}/"
    cp -r fnos/cmd "${BUILD_DIR}/"
    cp -r fnos/config "${BUILD_DIR}/"
    cp -r fnos/ui "${BUILD_DIR}/"
    cp fnos/ICON.PNG "${BUILD_DIR}/"
    cp fnos/ICON_256.PNG "${BUILD_DIR}/"
    cp fnos/manifest "${BUILD_DIR}/"
    [ -f fnos/wizard ] && cp fnos/wizard "${BUILD_DIR}/"

    # 更新 manifest 中的 checksum 和 platform
    sed -i.tmp "s/^checksum.*/checksum        = ${CHECKSUM}/" "${BUILD_DIR}/manifest"
    sed -i.tmp "s/^platform.*/platform        = ${PLATFORM}/" "${BUILD_DIR}/manifest"
    rm -f "${BUILD_DIR}/manifest.tmp"

    grep -q "^platform[[:space:]]*=[[:space:]]*${PLATFORM}$" "${BUILD_DIR}/manifest" \
        || { echo "Error: failed to set platform=${PLATFORM} in manifest"; rm -rf "${BUILD_DIR}"; exit 1; }

    # 打包成 .fpk (tar.gz 格式)
    rm -f "${SCRIPT_DIR}/${FPK_NAME}"
    tar czf "${SCRIPT_DIR}/${FPK_NAME}" -C "${BUILD_DIR}" .
    rm -rf "${BUILD_DIR}"

    echo "✓ Package created: ${FPK_NAME} ($(du -h "${SCRIPT_DIR}/${FPK_NAME}" | cut -f1))"
    BUILT="${BUILT} ${FPK_NAME}"
done

echo "  Checksum (app.tgz): ${CHECKSUM}"
echo ""
echo "Installation:"
echo "  1. 按 NAS CPU 架构选择安装包:${BUILT}"
echo "     x86 = Intel/AMD, arm = ARM64 (瑞芯微/飞牛 ARM 机型)"
echo "  2. 飞牛商店 > 本地安装 > 上传 fpk"
echo "  3. 在向导里配置服务端口和数据目录，然后启动服务"
