#!/bin/bash
# 飞牛应用打包脚本 - 生成 .fpk 文件
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

APP_NAME="opensurge"
VERSION="0.1.1"
PLATFORM="x86"
FPK_NAME="${APP_NAME}_${VERSION}_${PLATFORM}.fpk"

echo "Building ${FPK_NAME}..."

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

# 创建临时目录
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

# 复制所有文件到构建目录
cp app.tgz "${BUILD_DIR}/"
cp -r fnos/cmd "${BUILD_DIR}/"
cp -r fnos/config "${BUILD_DIR}/"
cp -r fnos/ui "${BUILD_DIR}/"
cp fnos/ICON.PNG "${BUILD_DIR}/"
cp fnos/ICON_256.PNG "${BUILD_DIR}/"
cp fnos/manifest "${BUILD_DIR}/"
[ -f fnos/wizard ] && cp fnos/wizard "${BUILD_DIR}/"

# 更新 manifest 中的 checksum
sed -i.tmp "s/^checksum.*/checksum        = ${CHECKSUM}/" "${BUILD_DIR}/manifest"
rm -f "${BUILD_DIR}/manifest.tmp"

# 打包成 .fpk (tar.gz 格式)
cd "${BUILD_DIR}"
tar czf "${SCRIPT_DIR}/${FPK_NAME}" *

echo "✓ Package created: ${FPK_NAME}"
echo "  Checksum: ${CHECKSUM}"
echo ""
echo "Installation:"
echo "  1. Upload ${FPK_NAME} to fnOS via App Store > 本地安装"
echo "  2. Configure service port and paths in wizard"
echo "  3. Start OpenSurge service"
