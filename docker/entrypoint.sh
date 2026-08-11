#!/bin/sh
set -eu

CONFIG="${OPENSURGE_CONFIG:-/etc/opensurge/config.yaml}"
ADDR="${OPENSURGE_ADDR:-0.0.0.0:61767}"
# 上游默认的 store 目录是 ~/Library/Application Support/OpenSurge，Linux 上必须显式给。
STORE="${OPENSURGE_STORE:-/var/lib/opensurge/store}"

if [ ! -f "$CONFIG" ]; then
	echo "缺少配置文件 $CONFIG" >&2
	echo "把 examples/config.fnos.example.yaml 复制成 ./config/config.yaml 并改好网卡名和 IP。" >&2
	exit 1
fi

# 浏览器实际用的地址。控制面拿它做 Origin 校验，填错的话 GUI 会变成只读
# （所有 POST 返回 403）。不给就从配置里的 gateway.lan_ip 推——那就是 NAS 的局域网 IP。
#
# FN Connect 外网入口固定为 https://opensurge.<FNID>.fnos.net/ ，控制面在 LAN 模式下
# 自动放行该 Host 形态，一般不必改 OPENSURGE_BASE_URL。
# 其它自定义域名：设 OPENSURGE_BASE_URL，或 OPENSURGE_ALLOWED_HOSTS（逗号分隔）。
if [ -z "${OPENSURGE_BASE_URL:-}" ]; then
	lan_ip="$(sed -n 's/^[[:space:]]*lan_ip:[[:space:]]*"\{0,1\}\([0-9.]\{7,\}\)"\{0,1\}.*/\1/p' "$CONFIG" | head -1)"
	if [ -z "$lan_ip" ]; then
		echo "配置里读不到 gateway.lan_ip，请显式设置 OPENSURGE_BASE_URL。" >&2
		exit 1
	fi
	OPENSURGE_BASE_URL="http://${lan_ip}:${ADDR##*:}"
fi

ALLOWED_HOSTS_ARGS=""
if [ -n "${OPENSURGE_ALLOWED_HOSTS:-}" ]; then
	ALLOWED_HOSTS_ARGS="--allowed-hosts=${OPENSURGE_ALLOWED_HOSTS}"
fi

mkdir -p "$STORE"

# Docker 默认把 /proc/sys 挂成只读，网关启动时要写 net.ipv4.ip_forward 会失败。
# 实测只给 NET_ADMIN 写不进去，得靠 SYS_ADMIN 把它 remount 成 rw（比 privileged 窄）。
# 部分环境（fnOS）即使给了 SYS_ADMIN 也禁止 remount，这种情况下只要宿主机已经开启
# ip_forward 就不影响网关运行——只打警告继续。
if ! mount -o remount,rw /proc/sys 2>/dev/null; then
	current="$(cat /proc/sys/net/ipv4/ip_forward 2>/dev/null || echo 0)"
	if [ "$current" != "1" ]; then
		echo "无法把 /proc/sys remount 成可写，且 ip_forward 当前是 $current。" >&2
		echo "请在宿主机上执行：sudo sysctl -w net.ipv4.ip_forward=1" >&2
		echo "或在 compose 里加 privileged: true（不推荐）。" >&2
		exit 1
	fi
	echo "警告：/proc/sys 只读，但宿主机已开启 ip_forward，继续启动。" >&2
fi

# 容器被停掉时把网关也拆干净：mihomo / dnsmasq 进程、nftables 表、ip_forward 都要还原，
# 否则下次启动会撞上残留的 state.json 和 nft 表。
shutdown() {
	echo "收到停止信号，正在停止网关…"
	omg stop --config "$CONFIG" || true
	if [ -n "${control_pid:-}" ]; then
		kill -TERM "$control_pid" 2>/dev/null || true
		wait "$control_pid" 2>/dev/null || true
	fi
	exit 0
}
trap shutdown TERM INT

echo "Web GUI: $OPENSURGE_BASE_URL/enter （浏览器打开；FN Connect: https://opensurge.<FNID>.fnos.net/ ）"
if [ -n "${OPENSURGE_ALLOWED_HOSTS:-}" ]; then
	echo "Extra allowed hosts: $OPENSURGE_ALLOWED_HOSTS"
fi
echo "若需一次性 bootstrap 链接：scripts/fnos-gui-url.sh"

# shellcheck disable=SC2086
opensurge-control --direct-root --config "$CONFIG" --addr "$ADDR" \
	--base-url "$OPENSURGE_BASE_URL" --store "$STORE" ${ALLOWED_HOSTS_ARGS} &
control_pid=$!

# wait 会被信号打断，所以要循环等到子进程真的退出。
while kill -0 "$control_pid" 2>/dev/null; do
	wait "$control_pid" && break || true
done
