#!/usr/bin/env sh
# 生成一个可用的 Web GUI 登录链接。
#
# 控制面的 API 全部要 session cookie，浏览器必须先走 /bootstrap?code=... 换。
# 启动时打印的那个链接 30 秒就过期，在 Docker 里基本抓不住，所以用持久化的
# control-token 现场换一个。
#
# 用法（在 NAS 上跑）：./scripts/fnos-gui-url.sh [容器名] [地址]
set -eu

CONTAINER="${1:-${OPENSURGE_CONTAINER:-opensurge}}"
BASE="${2:-${OPENSURGE_BASE_URL:-http://127.0.0.1:61767}}"
STORE="${OPENSURGE_STORE:-/var/lib/opensurge/store}"

token="$(docker exec "$CONTAINER" cat "$STORE/control-token")"
if [ -z "$token" ]; then
	echo "拿不到 control-token，容器起来了吗？" >&2
	exit 1
fi

response="$(curl -fsS -X POST "$BASE/api/v1/session/bootstrap" \
	-H "Authorization: Bearer $token" \
	-H 'Content-Type: application/json' \
	-d '{"path":"dashboard"}')"

# ponytail: 用 sed 抠 url 字段，不引入 jq —— 飞牛上不一定装了。
url="$(printf '%s' "$response" | sed -n 's/.*"url":"\([^"]*\)".*/\1/p')"
if [ -z "$url" ]; then
	echo "响应里没有 url：$response" >&2
	exit 1
fi

# 链接里的主机名来自控制面的监听地址，从别的机器访问要换成 NAS 的 IP。
echo "$url"
echo "（30 秒内打开。从别的设备访问就把 127.0.0.1 换成 NAS 的 IP。）" >&2
