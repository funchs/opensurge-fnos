#!/bin/bash
# 合成 OpenSurge 应用图标：Surge 波浪标 + 飞牛官方牛头徽章
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

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

magick "$SURGE" -resize 256x256 PNG32:"$WORK/base.png"
magick "$FNOS" -colorspace sRGB -type TrueColorAlpha -resize 200x200 PNG32:"$WORK/fnos.png"
magick "$WORK/fnos.png" \
  \( +clone -threshold 101% -fill white -draw 'circle 100,100 100,1' \) \
  -alpha off -compose CopyOpacity -composite PNG32:"$WORK/circle.png"
magick -size 200x200 xc:none -fill none -stroke white -strokewidth 14 \
  -draw 'circle 100,100 100,8' PNG32:"$WORK/ring.png"
magick "$WORK/circle.png" \( +clone -background black -shadow 50x5+2+3 \) \
  +swap -background none -layers merge +repage PNG32:"$WORK/shadowed.png"
magick "$WORK/shadowed.png" "$WORK/ring.png" -gravity center -compose Over -composite \
  PNG32:"$WORK/badge-hi.png"
magick "$WORK/badge-hi.png" -resize 80x80 PNG32:"$WORK/badge.png"
magick "$WORK/base.png" "$WORK/badge.png" -gravity SouthEast -geometry +8+8 \
  -compose Over -composite PNG32:"$WORK/final.png"

mkdir -p "$UI" "$ASSETS"
magick "$WORK/final.png" -depth 8 PNG32:"$OUT/ICON_256.PNG"
magick "$WORK/final.png" -resize 90x90 -depth 8 PNG32:"$OUT/ICON.PNG"
magick "$WORK/final.png" -resize 64x64 -depth 8 PNG32:"$UI/64.png"
magick "$WORK/final.png" -depth 8 PNG32:"$UI/256.png"
cp "$WORK/final.png" "$ASSETS/opensurge-fnos-icon-256.png"
echo "Wrote ICON.PNG / ICON_256.PNG / ui/images/{64,256}.png"
