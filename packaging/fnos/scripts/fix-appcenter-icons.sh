#!/bin/bash
# 在 NAS 上运行：用桌面已生效的新图标，覆盖应用中心仍在用的旧 ICON.PNG。
#
# 用法（SSH 登录飞牛后）:
#   bash fix-appcenter-icons.sh
# 或指定应用名:
#   APPNAME=opensurge bash fix-appcenter-icons.sh
set -euo pipefail

APPNAME="${APPNAME:-opensurge}"

echo "==> Looking for NEW icons under ${APPNAME} install tree..."

SRC64=""
SRC256=""
while IFS= read -r -d '' f; do
  case "$(basename "$f")" in
    icon-64.png|64.png)
      [ -z "$SRC64" ] && SRC64="$f"
      ;;
    icon-256.png|256.png)
      [ -z "$SRC256" ] && SRC256="$f"
      ;;
  esac
done < <(find \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  "/var/apps/${APPNAME}" \
  2>/dev/null \( -name 'icon-64.png' -o -name '64.png' -o -name 'icon-256.png' -o -name '256.png' \) -print0)

if [ -z "$SRC64" ] || [ -z "$SRC256" ]; then
  echo "ERROR: could not find ui/images icons for ${APPNAME}"
  echo "Desktop icon sources missing — reinstall the fpk first."
  exit 1
fi

echo "    SRC64 = $SRC64  ($(wc -c <"$SRC64") bytes)"
echo "    SRC256= $SRC256 ($(wc -c <"$SRC256") bytes)"

# 若 64 源看起来仍是旧 OS 暗图（体积很小且路径可疑），仍继续覆盖
echo "==> Overwriting ICON.PNG / ICON_256.PNG everywhere under ${APPNAME}..."

n=0
while IFS= read -r -d '' root; do
  # package roots that already exist
  :
done < <(true)

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
  echo "    wrote ${root}/ICON.PNG + ICON_256.PNG"
  n=$((n + 1))
  if [ -d "${root}/ui/images" ] || mkdir -p "${root}/ui/images" 2>/dev/null; then
    cp -f "$SRC64"  "${root}/ui/images/64.png"
    cp -f "$SRC256" "${root}/ui/images/256.png"
    cp -f "$SRC64"  "${root}/ui/images/icon-64.png"
    cp -f "$SRC256" "${root}/ui/images/icon-256.png"
  fi
done

# 再扫一遍，覆盖所有嵌套的 ICON*.PNG
while IFS= read -r -d '' p; do
  case "$p" in
    */ICON.PNG)
      cp -f "$SRC64" "$p" && echo "    overwrite $p" && n=$((n + 1))
      ;;
    */ICON_256.PNG)
      cp -f "$SRC256" "$p" && echo "    overwrite $p" && n=$((n + 1))
      ;;
  esac
done < <(find \
  "/var/apps/${APPNAME}" \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  2>/dev/null \( -name 'ICON.PNG' -o -name 'ICON_256.PNG' \) -print0)

echo "==> Updated ${n} path(s)."
echo "==> Current ICON.PNG files:"
find \
  "/var/apps/${APPNAME}" \
  "/vol1/@appcenter/${APPNAME}" \
  "/vol2/@appcenter/${APPNAME}" \
  "/volume1/@appcenter/${APPNAME}" \
  2>/dev/null \( -name 'ICON.PNG' -o -name 'ICON_256.PNG' \) -ls || true

# 尝试重启应用中心相关服务（失败可忽略）
echo "==> Restarting appcenter services (ignore errors if unit names differ)..."
for svc in trim_app_center appcenter fn-appcenter trim-appcenter; do
  if systemctl restart "$svc" 2>/dev/null; then
    echo "    restarted $svc"
  fi
done
# 部分版本用 trim 脚本
if [ -x /usr/local/bin/appcenter-cli ]; then
  /usr/local/bin/appcenter-cli 2>/dev/null | head -5 || true
fi

echo ""
echo "Done. In the browser: hard-refresh App Center (Ctrl/Cmd+Shift+R) or open an incognito window."
echo "If still old: log out of fnOS web UI and log back in."
