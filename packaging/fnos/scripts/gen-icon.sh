#!/bin/bash
# 从融合主图导出符合 fnOS 规范的应用图标。
# 主图: assets/opensurge-fnos-icon-256.png（C 方案：浪 + 牛角）
# 规范: ICON.PNG=64x64, ICON_256.PNG=256x256, 完整不透明正方形
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
# 铺白底去透明，避免安装器解码半透明圆角失败
for size in 64 256; do
  magick "$MASTER" -resize ${size}x${size} \
    -background white -alpha remove -alpha off \
    -colorspace sRGB -type TrueColor -depth 8 \
    PNG32:"$ASSETS/icon-square-${size}.png"
done

magick "$ASSETS/icon-square-64.png"  -depth 8 PNG32:"$OUT/ICON.PNG"
magick "$ASSETS/icon-square-256.png" -depth 8 PNG32:"$OUT/ICON_256.PNG"
magick "$ASSETS/icon-square-64.png"  -depth 8 PNG32:"$UI/64.png"
magick "$ASSETS/icon-square-256.png" -depth 8 PNG32:"$UI/256.png"
magick "$ASSETS/icon-square-64.png"  -depth 8 PNG32:"$UI/icon-64.png"
magick "$ASSETS/icon-square-256.png" -depth 8 PNG32:"$UI/icon-256.png"

echo "Wrote ICON.PNG (64x64) / ICON_256.PNG (256x256) / ui/images/*"
magick identify "$OUT/ICON.PNG" "$OUT/ICON_256.PNG"
