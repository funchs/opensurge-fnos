#!/bin/bash
# 按飞牛 fpk 应用图标规范导出圆角图标（高清导出链路）。
#
# 规范：
#   ICON.PNG     64×64  PNG，圆角矩形 + 四角透明
#   ICON_256.PNG 256×256 PNG，同上
#   ui/images/{64,256,icon-64,icon-256}.png  与上相同
#
# 清晰度：
#   - 优先用 ≥1024 源稿，工作分辨率 1024
#   - Lanczos 缩放；4× 超采样再缩到目标尺寸，圆角更利落
#   - 64px 轻微锐化，避免应用中心列表糊成一团
#
# 圆角半径约 22% 边长。色板与 Web GUI 薄荷绿一致。
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS="${PKG_DIR}/assets"
OUT="${PKG_DIR}/fnos"
UI="${OUT}/ui/images"

# 源稿优先级：扁平满铺高清稿 > 设计稿 > 其它
SOURCE=""
for cand in \
  "${ASSETS}/opensurge-fnos-icon-source-flat.png" \
  "${ASSETS}/opensurge-fnos-icon-source.png" \
  "${ASSETS}/fusion-variant-C-wave-horns-mint-light.png" \
  "${ASSETS}/fusion-variant-C-wave-horns-mint-unified.png" \
  "${ASSETS}/fusion-variant-C-wave-horns.png" \
  "${ASSETS}/opensurge-fnos-icon-256.png"
do
  if [ -f "$cand" ]; then
    SOURCE="$cand"
    break
  fi
done
[ -n "$SOURCE" ] || { echo "missing icon source" >&2; exit 1; }
command -v magick >/dev/null || { echo "ImageMagick required" >&2; exit 1; }
mkdir -p "$UI" "$ASSETS"

CORNER_RATIO="${CORNER_RATIO:-22}"
MASTER_SIZE="${MASTER_SIZE:-1024}"
# 浪角在画布中的占比（略放大，小尺寸更清晰）
LOGO_SCALE="${LOGO_SCALE:-92}"

sample_bg() {
  local img="$1"
  local w h x y hex
  w=$(magick identify -format '%w' "$img")
  h=$(magick identify -format '%h' "$img")
  x=$((w / 5))
  y=$((h / 5))
  hex=$(magick "$img" -format "%[hex:u.p{${x},${y}}]" info:)
  if [ "${#hex}" -lt 6 ] || [ "${hex:6:2}" = "00" ]; then
    hex=$(magick "$img" -format "%[hex:u.p{$((w/2)),$((h/5))}]" info:)
  fi
  # 纯白/近白时用薄荷绿卡底
  case "${hex:0:6}" in
    FFFFFF|FEFEFE|FDFDFD|FCFCFC|FAFAFA) echo "#E6F2EE" ;;
    *) echo "#${hex:0:6}" ;;
  esac
}

# 4× 超采样圆角 → Lanczos 落到目标尺寸
make_rounded() {
  local src="$1"
  local size="$2"
  local dest="$3"
  local sharpen="${4:-0}"
  local ss=$((size * 4))
  local r=$((ss * CORNER_RATIO / 100))
  local last=$((ss - 1))

  magick "$src" \
    -filter Lanczos -resize "${ss}x${ss}^" \
    -gravity center -extent "${ss}x${ss}" \
    -colorspace sRGB \
    PNG32:"$WORK/base-${size}.png"

  magick -size "${ss}x${ss}" xc:none \
    -fill white -draw "roundrectangle 0,0 ${last},${last} ${r},${r}" \
    PNG32:"$WORK/mask-${size}.png"

  magick "$WORK/base-${size}.png" "$WORK/mask-${size}.png" \
    -compose DstIn -composite \
    -filter Lanczos -resize "${size}x${size}" \
    -colorspace sRGB \
    PNG32:"$WORK/out-${size}.png"

  if [ "$sharpen" = "1" ]; then
    magick "$WORK/out-${size}.png" \
      -unsharp 0x0.6+0.6+0.02 \
      PNG32:"$dest"
  else
    # 256 也极轻锐化，抵消缩放发软
    magick "$WORK/out-${size}.png" \
      -unsharp 0x0.35+0.35+0.02 \
      PNG32:"$dest"
  fi
}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

BG=$(sample_bg "$SOURCE")
echo "source=$SOURCE ($(magick identify -format '%wx%h' "$SOURCE")) bg=${BG} master=${MASTER_SIZE} corner=${CORNER_RATIO}% logo=${LOGO_SCALE}%"

# 1) 高分辨率满铺底（直角），去掉旧圆角白边/角渣
magick "$SOURCE" \
  -filter Lanczos -resize "${MASTER_SIZE}x${MASTER_SIZE}^" \
  -gravity center -extent "${MASTER_SIZE}x${MASTER_SIZE}" \
  -background "$BG" -alpha remove -alpha off \
  -fuzz 12% -fill "$BG" -opaque white \
  -fuzz 10% -fill "$BG" -opaque "rgb(252,252,252)" \
  -fuzz 10% -fill "$BG" -opaque "rgb(245,248,246)" \
  -fuzz 8% -fill "$BG" -opaque "rgb(238,245,241)" \
  -fuzz 8% -fill "$BG" -opaque "rgb(230,240,234)" \
  -colorspace sRGB -type TrueColor -depth 8 \
  PNG32:"$WORK/flat-pre.png"
# 四角 floodfill，避免残留直角白边
ms=$((MASTER_SIZE + 1))
magick "$WORK/flat-pre.png" -bordercolor white -border 1 \
  -fill "$BG" -fuzz 12% \
  -draw "color 0,0 floodfill" \
  -draw "color ${ms},0 floodfill" \
  -draw "color 0,${ms} floodfill" \
  -draw "color ${ms},${ms} floodfill" \
  -shave 1x1 \
  PNG32:"$WORK/flat-raw.png"

# 2) 略放大中心内容（裁掉多余留白），小尺寸更清晰
pad=$(( (100 - LOGO_SCALE) * MASTER_SIZE / 200 ))
if [ "$pad" -gt 0 ]; then
  magick "$WORK/flat-raw.png" \
    -gravity center -crop "$((MASTER_SIZE - 2 * pad))x$((MASTER_SIZE - 2 * pad))+0+0" +repage \
    -filter Lanczos -resize "${MASTER_SIZE}x${MASTER_SIZE}" \
    -background "$BG" -alpha remove -alpha off \
    PNG32:"$WORK/flat.png"
else
  cp "$WORK/flat-raw.png" "$WORK/flat.png"
fi

cp "$WORK/flat.png" "$ASSETS/opensurge-fnos-icon-source-flat.png"
# 更新可再导出的源（满铺高清）
cp "$WORK/flat.png" "$ASSETS/opensurge-fnos-icon-source.png"

# 3) 各尺寸圆角（64 加强锐化）
make_rounded "$WORK/flat.png" 256 "$ASSETS/icon-square-256.png" 0
make_rounded "$WORK/flat.png" 128 "$ASSETS/icon-square-128.png" 0
make_rounded "$WORK/flat.png" 90  "$ASSETS/icon-square-90.png" 0
make_rounded "$WORK/flat.png" 64  "$ASSETS/icon-square-64.png" 1
make_rounded "$WORK/flat.png" 48  "$ASSETS/icon-square-48.png" 1
make_rounded "$WORK/flat.png" 32  "$ASSETS/icon-square-32.png" 1

cp "$ASSETS/icon-square-256.png" "$ASSETS/opensurge-fnos-icon-256.png"
cp "$ASSETS/icon-square-256.png" "$ASSETS/preview-256.png"

cp "$ASSETS/icon-square-64.png"  "$OUT/ICON.PNG"
cp "$ASSETS/icon-square-256.png" "$OUT/ICON_256.PNG"
cp "$ASSETS/icon-square-64.png"  "$UI/64.png"
cp "$ASSETS/icon-square-256.png" "$UI/256.png"
cp "$ASSETS/icon-square-64.png"  "$UI/icon-64.png"
cp "$ASSETS/icon-square-256.png" "$UI/icon-256.png"

if [ -d "${PKG_DIR}/../../web/public" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../web/public/opensurge-icon.png"
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../web/public/opensurge-mark.png"
fi
if [ -d "${PKG_DIR}/../../internal/webui/dist" ]; then
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../internal/webui/dist/opensurge-icon.png"
  cp "$ASSETS/icon-square-256.png" "${PKG_DIR}/../../internal/webui/dist/opensurge-mark.png"
fi

echo "fnOS HD rounded icons ready (64/256, r≈${CORNER_RATIO}%, 4× supersample + Lanczos)"
magick "$OUT/ICON.PNG" -format 'ICON64  %wx%h bytes=%b TL=%[pixel:p{0,0}] C=%[pixel:p{32,32}]\n' info:
magick "$OUT/ICON_256.PNG" -format 'ICON256 %wx%h bytes=%b TL=%[pixel:p{0,0}] C=%[pixel:p{128,128}]\n' info:
