#!/bin/bash
# 合成 OpenSurge 应用图标：Surge 波浪标 + 左上角融合的飞牛牛头
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS="${PKG_DIR}/assets"
OUT="${PKG_DIR}/fnos"
UI="${OUT}/ui/images"
SURGE="${ASSETS}/source-surge-wave.png"
FNOS="${ASSETS}/source-fnos-favicon.png"
[ -f "$SURGE" ] && [ -f "$FNOS" ] || { echo "missing source icons in assets/" >&2; exit 1; }
command -v magick >/dev/null || { echo "ImageMagick (magick) required" >&2; exit 1; }
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT

magick "$SURGE" -resize 256x256 PNG32:"$WORK/base.png"
magick "$WORK/base.png" -alpha extract PNG32:"$WORK/mask.png"
magick "$FNOS" -colorspace sRGB -type TrueColorAlpha -resize 200x200 PNG32:"$WORK/fnos.png"
magick "$WORK/fnos.png" \
  \( +clone -threshold 101% -fill white -draw 'circle 100,100 100,1' \) \
  -alpha off -compose CopyOpacity -composite PNG32:"$WORK/circle.png"
magick "$WORK/circle.png" -channel A -blur 0x2.0 +channel PNG32:"$WORK/soft.png"
magick "$WORK/soft.png" \
  \( +clone -fill '#6fd9cc' -colorize 40 \) \
  -compose Blend -define compose:args=40 -composite PNG32:"$WORK/tinted.png"
magick "$WORK/tinted.png" \
  \( +clone -background '#8aeee3' -shadow 40x6+0+0 \) \
  +swap -background none -layers merge +repage PNG32:"$WORK/glow.png"
magick "$WORK/glow.png" -resize 72x72 PNG32:"$WORK/badge.png"
magick "$WORK/base.png" \
  \( "$WORK/badge.png" -channel A -evaluate multiply 0.88 +channel \) \
  -gravity NorthWest -geometry +14+14 \
  -compose Over -composite PNG32:"$WORK/composed.png"
magick "$WORK/composed.png" "$WORK/mask.png" \
  -alpha off -compose CopyOpacity -composite PNG32:"$WORK/final.png"

mkdir -p "$UI" "$ASSETS"
magick "$WORK/final.png" -depth 8 PNG32:"$OUT/ICON_256.PNG"
magick "$WORK/final.png" -resize 90x90 -depth 8 PNG32:"$OUT/ICON.PNG"
magick "$WORK/final.png" -resize 64x64 -depth 8 PNG32:"$UI/64.png"
magick "$WORK/final.png" -depth 8 PNG32:"$UI/256.png"
cp "$WORK/final.png" "$ASSETS/opensurge-fnos-icon-256.png"
echo "Wrote icons (fnOS badge fused top-left, clipped to app shape)"
