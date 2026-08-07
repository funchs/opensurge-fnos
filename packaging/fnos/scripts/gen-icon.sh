#!/bin/bash
# 从「渐变融合 C」主图导出直角满铺图标（保留浪/角渐变，去掉预渲染圆角白边）。
#
# 色板与 Web GUI 卡片统一（web/src/styles.css）：
#   浅色卡底  #f2f7f4 / #e8f5ef / #d0e5dc（page + light card mint）
#   符号高光  #9af0c7 / #8ef0c2 / #79e6b3（primary / orb / letter-spacing accents）
#   符号中调  #6fcfa6 / #58b98f / #3ca67a
#   深色卡参考 assets/fusion-variant-C-wave-horns-mint-dark.png
#               （#0d1b17 + #153c31 glow + mint ribbon，默认不导出）
# 主源稿：assets/opensurge-fnos-icon-source.png（薄荷绿浅卡版）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS="${PKG_DIR}/assets"
OUT="${PKG_DIR}/fnos"
UI="${OUT}/ui/images"
# 优先用未铺边的设计稿，否则用当前 256
SOURCE="${ASSETS}/opensurge-fnos-icon-source.png"
[ -f "$SOURCE" ] || SOURCE="${ASSETS}/opensurge-fnos-icon-256.png"
[ -f "$SOURCE" ] || { echo "missing icon source" >&2; exit 1; }
command -v magick >/dev/null || { echo "ImageMagick required" >&2; exit 1; }
mkdir -p "$UI" "$ASSETS"

BG=$(magick "$SOURCE" -format '%[hex:u.p{40,40}]' info:)
R=$((16#${BG:0:2})); G=$((16#${BG:2:2})); B=$((16#${BG:4:2}))
if [ "$R" -gt 248 ] && [ "$G" -gt 248 ] && [ "$B" -gt 248 ]; then
  BG=$(magick "$SOURCE" -format '%[hex:u.p{60,60}]' info:)
fi
BG_COLOR="#${BG:0:6}"

WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
magick "$SOURCE" -resize 256x256 \
  -fuzz 6% -fill "$BG_COLOR" -opaque white \
  -fuzz 4% -fill "$BG_COLOR" -opaque "rgb(252,252,252)" \
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
if [ -d "${PKG_DIR}/../../web/public" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../web/public/opensurge-icon.png"
fi
if [ -d "${PKG_DIR}/../../internal/webui/dist" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../internal/webui/dist/opensurge-icon.png"
fi
echo "Gradient fusion C → full-bleed square (bg=${BG_COLOR})"
magick "$OUT/ICON_256.PNG" -format 'TL=%[pixel:p{1,1}] center=%[pixel:p{128,128}]\n' info:
