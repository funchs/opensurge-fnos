#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/meta.env"

VERSION="${VERSION:-v0.1.1}"
WORK_DIR=$(mktemp -d)
trap "rm -rf $WORK_DIR" EXIT

# 复制 docker 目录（包含 compose 和示例配置）
mkdir -p "${WORK_DIR}/docker"
cp "${SCRIPT_DIR}/../fnos/docker/docker-compose.yaml" "${WORK_DIR}/docker/"
cp "${SCRIPT_DIR}/../fnos/docker/config.fnos.example.yaml" "${WORK_DIR}/docker/"

# 复制 UI 定义
cp -a "${SCRIPT_DIR}/../fnos/ui" "${WORK_DIR}/ui"

# 打包
cd "${WORK_DIR}"
tar czf "${SCRIPT_DIR}/../app.tgz" docker/ ui/

echo "Built app.tgz for opensurge ${VERSION}"
