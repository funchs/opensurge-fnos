#!/usr/bin/env sh
# 在飞牛 NAS 上验证部署。默认只做只读检查，不碰宿主网络。
#
#   ./scripts/smoke-fnos.sh                  只读检查
#   ./scripts/smoke-fnos.sh --start-gateway  额外启动网关，验 nftables 和 ip_forward
#
# 为什么不默认启动网关：那会改宿主的转发状态和防火墙规则，得你明确同意。
set -eu

CONTAINER="${OPENSURGE_CONTAINER:-opensurge}"
STORE="${OPENSURGE_STORE:-/var/lib/opensurge/store}"
CONFIG="${OPENSURGE_CONFIG:-/etc/opensurge/config.yaml}"
START_GATEWAY=false
[ "${1:-}" = "--start-gateway" ] && START_GATEWAY=true

failures=0
pass() { printf '  PASS  %s\n' "$1"; }
fail() { printf '  FAIL  %s\n' "$1"; failures=$((failures + 1)); }
skip() { printf '  SKIP  %s\n' "$1"; }

echo "== 1. 容器在跑 =="
if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null)" = "true" ]; then
	pass "容器 $CONTAINER 运行中"
else
	fail "容器 $CONTAINER 没在运行，先 docker compose up -d"
	exit 1
fi

echo "== 2. 依赖齐全 =="
for binary in omg opensurge-control mihomo nft ip ping sysctl dnsmasq; do
	if docker exec "$CONTAINER" sh -c "command -v $binary >/dev/null"; then
		pass "$binary"
	else
		fail "$binary 不在镜像里"
	fi
done

echo "== 3. /proc/sys 可写（ip_forward 要写它）=="
if docker exec "$CONTAINER" sh -c 'mount | grep -q "on /proc/sys type proc (rw"'; then
	pass "/proc/sys 已 remount 成 rw"
else
	fail "/proc/sys 只读，compose 里缺 cap_add: SYS_ADMIN"
fi

echo "== 4. 控制面认证链路 =="
base="$(docker exec "$CONTAINER" sh -c 'sed -n "s/^[[:space:]]*lan_ip:[[:space:]]*\"\{0,1\}\([0-9.]\{7,\}\)\"\{0,1\}.*/\1/p" '"$CONFIG"' | head -1')"
base="http://${base}:61767"
token="$(docker exec "$CONTAINER" cat "$STORE/control-token" 2>/dev/null || true)"
if [ -z "$token" ]; then
	fail "读不到 control-token"
else
	pass "control-token 已生成"
	if [ "$(curl -s -o /dev/null -w '%{http_code}' "$base/")" = "200" ]; then
		pass "Web GUI 静态页可达 ($base)"
	else
		fail "Web GUI 静态页不可达 ($base)"
	fi
	if [ "$(curl -s -o /dev/null -w '%{http_code}' "$base/api/v1/overview")" = "401" ]; then
		pass "无凭证访问 API 被拒（401）"
	else
		fail "无凭证访问 API 没有返回 401，认证可能失效"
	fi
	url="$(curl -fsS -X POST "$base/api/v1/session/bootstrap" -H "Authorization: Bearer $token" \
		-H 'Content-Type: application/json' -d '{"path":"dashboard"}' 2>/dev/null |
		sed -n 's/.*"url":"\([^"]*\)".*/\1/p')"
	if [ -n "$url" ]; then
		pass "能签发 bootstrap 链接"
		cookie="$(mktemp)"
		curl -s -c "$cookie" -o /dev/null "$url"
		if [ "$(curl -s -b "$cookie" -o /dev/null -w '%{http_code}' "$base/api/v1/overview")" = "200" ]; then
			pass "换取 session 后能读 overview"
		else
			fail "换取 session 后仍读不到 overview"
		fi
		rm -f "$cookie"
	else
		fail "签发 bootstrap 链接失败"
	fi
fi

echo "== 5. 设备发现数据源 =="
if [ -n "$(docker exec "$CONTAINER" ip -j neigh show 2>/dev/null | tr -d '[]' | tr -d ' ')" ]; then
	pass "ip neigh 有邻居条目"
else
	fail "ip neigh 空，检查 network_mode: host"
fi

echo "== 6. nftables / ip_forward =="
if [ "$START_GATEWAY" = true ]; then
	if docker exec "$CONTAINER" omg start --config "$CONFIG"; then
		pass "omg start 成功"
	else
		fail "omg start 失败，看 docker logs"
	fi
fi
if docker exec "$CONTAINER" nft list tables 2>/dev/null | grep -q 'inet opensurge'; then
	pass "nftables 里有 table inet opensurge"
elif [ "$START_GATEWAY" = true ]; then
	fail "网关已启动但看不到 table inet opensurge"
else
	skip "网关未启动，加 --start-gateway 才验这项"
fi
forwarding="$(docker exec "$CONTAINER" cat /proc/sys/net/ipv4/ip_forward 2>/dev/null || echo '?')"
if [ "$forwarding" = "1" ]; then
	pass "ip_forward = 1"
elif [ "$START_GATEWAY" = true ]; then
	fail "网关已启动但 ip_forward = $forwarding"
else
	skip "ip_forward = $forwarding（网关未启动时正常）"
fi

echo
if [ "$failures" -gt 0 ]; then
	echo "$failures 项失败。"
	exit 1
fi
echo "自动检查全部通过。"
echo
echo "剩下这步只能手动验："
echo "  在另一台设备上把网关和 DNS 都指向 NAS 的 IP，确认能出网，"
echo "  并且 Web GUI 的设备列表里看得到这台设备。"
