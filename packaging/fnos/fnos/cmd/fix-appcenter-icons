#!/bin/bash
# 在 NAS 上运行：强制把磁盘上的新图标覆盖到应用中心/桌面所有路径。
#
# 用法（SSH 登录飞牛后）:
#   bash /vol1/@appcenter/opensurge/cmd/fix-appcenter-icons
# 或拷贝本脚本:
#   APPNAME=opensurge bash fix-appcenter-icons.sh
#
# 新版薄荷绿满铺图标特征：ICON.PNG ≈ 3–5KB，ICON_256.PNG ≈ 20–30KB，
# 四角接近 rgb(208,229,220)。若 ls 体积仍是 ~20KB/50KB 的旧 OS/蓝标，
# 说明装到的不是最新 fpk——请装 opensurge_0.1.1_*.fpk。
set -euo pipefail

APPNAME="${APPNAME:-opensurge}"

echo "==> Looking for installed OpenSurge under common fnOS paths..."

SRC64=""
SRC256=""
# Prefer package-root ICON*.PNG (what fpk ships), then ui/images.
while IFS= read -r -d '' f; do
  case "$(basename "$f")" in
    ICON.PNG)
      # Prefer larger? No — prefer paths under @appcenter, then by mtime later
      if [ -z "$SRC64" ]; then SRC64="$f"; fi
      # Prefer newer/mtime by comparing
      if [ -n "$SRC64" ] && [ "$f" -nt "$SRC64" ]; then SRC64="$f"; fi
      ;;
    ICON_256.PNG)
      if [ -z "$SRC256" ]; then SRC256="$f"; fi
      if [ -n "$SRC256" ] && [ "$f" -nt "$SRC256" ]; then SRC256="$f"; fi
      ;;
  esac
done < <(find \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  "/var/apps/${APPNAME}" \
  2>/dev/null \( -name 'ICON.PNG' -o -name 'ICON_256.PNG' \) -print0)

# Fallback to ui/images
if [ -z "$SRC64" ] || [ -z "$SRC256" ]; then
  while IFS= read -r -d '' f; do
    case "$(basename "$f")" in
      icon-64.png|64.png)  [ -z "$SRC64" ] && SRC64="$f" ;;
      icon-256.png|256.png) [ -z "$SRC256" ] && SRC256="$f" ;;
    esac
  done < <(find \
    "/vol1/@appcenter/${APPNAME}" \
    "/vol2/@appcenter/${APPNAME}" \
    "/volume1/@appcenter/${APPNAME}" \
    "/var/apps/${APPNAME}" \
    2>/dev/null \( -name 'icon-64.png' -o -name '64.png' -o -name 'icon-256.png' -o -name '256.png' \) -print0)
fi

if [ -z "$SRC64" ] || [ -z "$SRC256" ]; then
  echo "ERROR: could not find ICON / ui/images for ${APPNAME}"
  echo "Reinstall opensurge_0.1.1_x86.fpk or _arm.fpk first."
  exit 1
fi

echo "    SRC64  = $SRC64  ($(wc -c <"$SRC64" | tr -d ' ') bytes)"
echo "    SRC256 = $SRC256 ($(wc -c <"$SRC256" | tr -d ' ') bytes)"
if command -v md5sum >/dev/null; then
  echo "    md5 64  = $(md5sum "$SRC64" | awk '{print $1}')"
  echo "    md5 256 = $(md5sum "$SRC256" | awk '{print $1}')"
elif command -v md5 >/dev/null; then
  echo "    md5 64  = $(md5 -q "$SRC64")"
  echo "    md5 256 = $(md5 -q "$SRC256")"
fi

echo "==> Overwriting ICON*.PNG + ui/images everywhere..."
n=0
for root in \
  "/var/apps/${APPNAME}" \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  "/usr/local/apps/${APPNAME}"
do
  [ -d "$root" ] || continue
  cp -f "$SRC64"  "${root}/ICON.PNG"
  cp -f "$SRC256" "${root}/ICON_256.PNG"
  touch "${root}/ICON.PNG" "${root}/ICON_256.PNG"
  echo "    wrote ${root}/ICON.PNG + ICON_256.PNG"
  n=$((n + 1))
  if [ -d "${root}/ui/images" ] || mkdir -p "${root}/ui/images" 2>/dev/null; then
    cp -f "$SRC64"  "${root}/ui/images/64.png"
    cp -f "$SRC256" "${root}/ui/images/256.png"
    cp -f "$SRC64"  "${root}/ui/images/icon-64.png"
    cp -f "$SRC256" "${root}/ui/images/icon-256.png"
    touch "${root}/ui/images/"*.png 2>/dev/null || true
  fi
done

while IFS= read -r -d '' p; do
  case "$p" in
    */ICON.PNG)
      cp -f "$SRC64" "$p" && touch "$p" && echo "    overwrite $p" && n=$((n + 1))
      ;;
    */ICON_256.PNG)
      cp -f "$SRC256" "$p" && touch "$p" && echo "    overwrite $p" && n=$((n + 1))
      ;;
  esac
done < <(find \
  "/var/apps/${APPNAME}" \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  2>/dev/null \( -name 'ICON.PNG' -o -name 'ICON_256.PNG' \) -print0)

# Best-effort cache purge
echo "==> Clearing loose cache files (ignore errors)..."
find /usr/trim/www /var/cache /tmp 2>/dev/null \
  \( -iname "*${APPNAME}*icon*" -o -iname "*${APPNAME}*ICON*" \) \
  -type f -print -delete 2>/dev/null || true

echo "==> Updated ${n} path(s). Current icons:"
find \
  "/var/apps/${APPNAME}" \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  2>/dev/null \( -name 'ICON.PNG' -o -name 'ICON_256.PNG' \) -ls || true

echo "==> Restarting appcenter services (ignore errors if unit names differ)..."
for svc in trim_app_center appcenter fn-appcenter trim-appcenter; do
  if systemctl restart "$svc" 2>/dev/null; then
    echo "    restarted $svc"
  fi
done

echo ""
echo "Done."
echo "1) Browser: hard-refresh App Center (Ctrl/Cmd+Shift+R) or use a private window"
echo "2) Or log out of fnOS web UI and log back in"
echo "3) Confirm disk icons are NEW: ICON.PNG ~3–5KB, ICON_256 ~20–30KB"
echo "   If still ~20KB/50KB → you installed an old fpk; install opensurge_0.1.1_*.fpk"
