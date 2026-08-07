#!/bin/bash
# 按飞牛 fpk 应用图标规范导出圆角图标。
#
# 规范（与 build-fpk / 应用中心约定一致）：
#   ICON.PNG     64×64  PNG，圆角矩形 + 四角透明
#   ICON_256.PNG 256×256 PNG，同上
#   ui/images/{64,256,icon-64,icon-256}.png  与上相同
#
# 圆角半径约 22% 边长（与常见 fnOS / 桌面圆角应用图标一致），
# 外角必须透明，勿输出直角满铺方块。
#
# 色板与 Web GUI 统一：
#   浅色卡底  #f2f7f4 / #e8f5ef / #d0e5dc
#   符号      #9af0c7 / #79e6b3 / #3ca67a
# 主源稿：assets/opensurge-fnos-icon-source.png（薄荷绿 fusion C）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS="${PKG_DIR}/assets"
OUT="${PKG_DIR}/fnos"
UI="${OUT}/ui/images"

SOURCE="${ASSETS}/opensurge-fnos-icon-source.png"
[ -f "$SOURCE" ] || SOURCE="${ASSETS}/opensurge-fnos-icon-256.png"
[ -f "$SOURCE" ] || SOURCE="${ASSETS}/fusion-variant-C-wave-horns.png"
[ -f "$SOURCE" ] || { echo "missing icon source" >&2; exit 1; }
command -v magick >/dev/null || { echo "ImageMagick required" >&2; exit 1; }
mkdir -p "$UI" "$ASSETS"

# 圆角半径比例（相对边长）
CORNER_RATIO="${CORNER_RATIO:-22}"

# 从源图取内容底色（若源已是圆角则取中心附近）
sample_bg() {
  local img="$1"
  local w h x y hex
  w=$(magick identify -format '%w' "$img")
  h=$(magick identify -format '%h' "$img")
  x=$((w / 5))
  y=$((h / 5))
  hex=$(magick "$img" -format "%[hex:u.p{${x},${y}}]" info:)
  # 若角落是透明/白，改采中心偏上
  if [ "${hex:0:6}" = "000000" ] || [ "${hex:6:2}" = "00" ]; then
    hex=$(magick "$img" -format "%[hex:u.p{$((w/2)),$((h/5))}]" info:)
  fi
  echo "#${hex:0:6}"
}

# 生成指定边长的圆角图标（RGBA，角外透明）
make_rounded() {
  local src="$1"
  local size="$2"
  local dest="$3"
  local r=$((size * CORNER_RATIO / 100))
  local last=$((size - 1))

  magick "$src" -resize "${size}x${size}^" -gravity center -extent "${size}x${size}" \
    -colorspace sRGB PNG32:"$WORK/base-${size}.png"

  # 圆角遮罩：白=保留，黑=透明
  magick -size "${size}x${size}" xc:none \
    -fill white -draw "roundrectangle 0,0 ${last},${last} ${r},${r}" \
    PNG32:"$WORK/mask-${size}.png"

  magick "$WORK/base-${size}.png" "$WORK/mask-${size}.png" \
    -compose DstIn -composite \
    -colorspace sRGB \
    PNG32:"$dest"
}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# 若源是带白边的圆角图，先用背景色填满再重新切圆角，避免双重圆角白环
BG=$(sample_bg "$SOURCE")
echo "source=$SOURCE bg=${BG} corner=${CORNER_RATIO}%"

magick "$SOURCE" -resize 512x512 \
  -background "$BG" -alpha remove -alpha off \
  -fuzz 8% -fill "$BG" -opaque white \
  -fuzz 5% -fill "$BG" -opaque "rgb(252,252,252)" \
  -colorspace sRGB -type TrueColor -depth 8 \
  PNG32:"$WORK/flat.png"

# 导出各尺寸圆角
make_rounded "$WORK/flat.png" 256 "$ASSETS/icon-square-256.png"
make_rounded "$WORK/flat.png" 128 "$ASSETS/icon-square-128.png"
make_rounded "$WORK/flat.png" 90  "$ASSETS/icon-square-90.png"
make_rounded "$WORK/flat.png" 64  "$ASSETS/icon-square-64.png"
make_rounded "$WORK/flat.png" 48  "$ASSETS/icon-square-48.png"
make_rounded "$WORK/flat.png" 32  "$ASSETS/icon-square-32.png"

cp "$ASSETS/icon-square-256.png" "$ASSETS/opensurge-fnos-icon-256.png"
cp "$ASSETS/icon-square-256.png" "$ASSETS/preview-256.png"
# 保留一份无圆角满铺源，便于以后改 radius 重导
cp "$WORK/flat.png" "$ASSETS/opensurge-fnos-icon-source-flat.png"

# fpk / 桌面
cp "$ASSETS/icon-square-64.png"  "$OUT/ICON.PNG"
cp "$ASSETS/icon-square-256.png" "$OUT/ICON_256.PNG"
cp "$ASSETS/icon-square-64.png"  "$UI/64.png"
cp "$ASSETS/icon-square-256.png" "$UI/256.png"
cp "$ASSETS/icon-square-64.png"  "$UI/icon-64.png"
cp "$ASSETS/icon-square-256.png" "$UI/icon-256.png"

# Web：侧栏 / favicon 同一套圆角图标（与应用中心一致）
if [ -d "${PKG_DIR}/../../web/public" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../web/public/opensurge-icon.png"
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../web/public/opensurge-mark.png"
fi
if [ -d "${PKG_DIR}/../../internal/webui/dist" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../internal/webui/dist/opensurge-icon.png"
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../internal/webui/dist/opensurge-mark.png"
fi

echo "fnOS rounded icons ready (64/256, r≈${CORNER_RATIO}%, RGBA corners transparent)"
magick "$OUT/ICON.PNG" -format 'ICON64  %wx%h TL=%[pixel:p{0,0}] C=%[pixel:p{32,32}]\n' info:
magick "$OUT/ICON_256.PNG" -format 'ICON256 %wx%h TL=%[pixel:p{0,0}] C=%[pixel:p{128,128}]\n' info:
