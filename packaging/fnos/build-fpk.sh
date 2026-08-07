#!/bin/bash
# 飞牛应用打包脚本 - 生成 .fpk 文件
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

APP_NAME="opensurge"
VERSION="0.1.1"
FPK_NAME="${APP_NAME}_${VERSION}_all.fpk"

echo "Building ${FPK_NAME}..."

# 检查必要文件
if [ ! -f "app.tgz" ]; then
    echo "Error: app.tgz not found. Run scripts/build.sh first."
    exit 1
fi

if [ ! -f "fnos/ICON.PNG" ] || [ ! -f "fnos/ICON_256.PNG" ]; then
    echo "Warning: Icon files missing. Package will have placeholder icons."
fi

# 创建临时目录
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

# 复制所有文件到构建目录
cp -r fnos/* "${BUILD_DIR}/"
cp app.tgz "${BUILD_DIR}/"

# 打包成 .fpk (实际上就是 tar.gz)
cd "${BUILD_DIR}"
tar czf "${SCRIPT_DIR}/${FPK_NAME}" *

echo "✓ Package created: ${FPK_NAME}"
echo ""
echo "Installation:"
echo "  1. Upload ${FPK_NAME} to fnOS via App Store > 本地安装"
echo "  2. Configure service port and paths in wizard"
echo "  3. Start OpenSurge service"
