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

mkdir -p "$STORE"

# Docker 默认把 /proc/sys 挂成只读，网关启动时要写 net.ipv4.ip_forward 会失败。
# 实测只给 NET_ADMIN 写不进去，得靠 SYS_ADMIN 把它 remount 成 rw（比 privileged 窄）。
if ! mount -o remount,rw /proc/sys 2>/dev/null; then
	echo "无法把 /proc/sys remount 成可写，网关将无法开启 IPv4 转发。" >&2
	echo "请确认 compose 里给了 cap_add: SYS_ADMIN。" >&2
	exit 1
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

opensurge-control --direct-root --config "$CONFIG" --addr "$ADDR" --store "$STORE" &
control_pid=$!

# wait 会被信号打断，所以要循环等到子进程真的退出。
while kill -0 "$control_pid" 2>/dev/null; do
	wait "$control_pid" && break || true
done
