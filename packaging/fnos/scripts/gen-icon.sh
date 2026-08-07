#!/bin/bash
# 完整正方形直角图标：去掉预渲染圆角白边（系统 UI 再裁圆角）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS="${PKG_DIR}/assets"
OUT="${PKG_DIR}/fnos"
UI="${OUT}/ui/images"
MASTER="${ASSETS}/opensurge-fnos-icon-256.png"
[ -f "$MASTER" ] || { echo "missing $MASTER" >&2; exit 1; }
command -v magick >/dev/null || { echo "ImageMagick required" >&2; exit 1; }
mkdir -p "$UI" "$ASSETS"

BG=$(magick "$MASTER" -format '%[hex:u.p{48,48}]' info:)
R=$((16#${BG:0:2})); G=$((16#${BG:2:2})); B=$((16#${BG:4:2}))
if [ "$R" -gt 245 ] && [ "$G" -gt 245 ] && [ "$B" -gt 245 ]; then
  BG_COLOR="#DDE7F0"
else
  BG_COLOR="#${BG:0:6}"
fi

WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
magick "$MASTER" -resize 256x256 \
  -fuzz 15% -fill "$BG_COLOR" -opaque white \
  -fuzz 15% -fill "$BG_COLOR" -opaque "rgb(250,250,250)" \
  -background "$BG_COLOR" -alpha remove -alpha off \
  -colorspace sRGB -type TrueColor -depth 8 \
  PNG32:"$WORK/full.png"

cp "$WORK/full.png" "$ASSETS/opensurge-fnos-icon-256.png"
cp "$WORK/full.png" "$ASSETS/icon-square-256.png"
cp "$WORK/full.png" "$ASSETS/preview-256.png"

for size in 32 48 64 90 128 256; do
  magick "$WORK/full.png" -resize ${size}x${size} \
    -colorspace sRGB -type TrueColor -alpha off -depth 8 \
    PNG32:"$ASSETS/icon-square-${size}.png"
done

cp "$ASSETS/icon-square-64.png"  "$OUT/ICON.PNG"
cp "$ASSETS/icon-square-256.png" "$OUT/ICON_256.PNG"
cp "$ASSETS/icon-square-64.png"  "$UI/64.png"
cp "$ASSETS/icon-square-256.png" "$UI/256.png"
cp "$ASSETS/icon-square-64.png"  "$UI/icon-64.png"
cp "$ASSETS/icon-square-256.png" "$UI/icon-256.png"

# web / dist
if [ -d "${PKG_DIR}/../../web/public" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../web/public/opensurge-icon.png"
fi
if [ -d "${PKG_DIR}/../../internal/webui/dist" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../internal/webui/dist/opensurge-icon.png"
fi

echo "Full-bleed square icons bg=${BG_COLOR}"
magick "$OUT/ICON_256.PNG" -format 'TL=%[pixel:p{1,1}] edge=%[pixel:p{1,128}] center=%[pixel:p{128,128}]\n' info:
magick identify "$OUT/ICON.PNG" "$OUT/ICON_256.PNG"
