#!/bin/bash
# 从融合主图导出各尺寸应用图标。
# 主图: assets/opensurge-fnos-icon-256.png
# 设计: Surge 波浪 + 飞牛牛角合成单一符号（非角标裁切）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS="${PKG_DIR}/assets"
OUT="${PKG_DIR}/fnos"
UI="${OUT}/ui/images"
MASTER="${ASSETS}/opensurge-fnos-icon-256.png"
[ -f "$MASTER" ] || { echo "missing $MASTER" >&2; exit 1; }
command -v magick >/dev/null || { echo "ImageMagick (magick) required" >&2; exit 1; }
mkdir -p "$UI"
magick "$MASTER" -depth 8 PNG32:"$OUT/ICON_256.PNG"
magick "$MASTER" -resize 90x90 -depth 8 PNG32:"$OUT/ICON.PNG"
magick "$MASTER" -resize 64x64 -depth 8 PNG32:"$UI/64.png"
magick "$MASTER" -depth 8 PNG32:"$UI/256.png"
echo "Exported ICON.PNG / ICON_256.PNG / ui/images/{64,256}.png from fused master"
