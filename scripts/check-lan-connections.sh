#!/usr/bin/env bash
# Check whether mihomo is seeing real LAN client sourceIPs (side-router health).
# Usage:
#   ./scripts/check-lan-connections.sh
#   MIHOMO_API=http://127.0.0.1:9090 LAN_PREFIX=192.168.1 ./scripts/check-lan-connections.sh
set -euo pipefail

API="${MIHOMO_API:-http://127.0.0.1:9090}"
LAN_PREFIX="${LAN_PREFIX:-192.168.1}"
SECRET="${MIHOMO_SECRET:-}"

auth_header=()
if [[ -n "$SECRET" ]]; then
  auth_header=(-H "Authorization: Bearer ${SECRET}")
fi

json="$(curl -fsS "${auth_header[@]}" "${API}/connections")" || {
  echo "FAIL: cannot reach ${API}/connections" >&2
  exit 2
}

export LAN_PREFIX
export CONNECTIONS_JSON="$json"
python3 - <<'PY'
import json, os, sys

lan_prefix = os.environ["LAN_PREFIX"].rstrip(".") + "."
data = json.loads(os.environ["CONNECTIONS_JSON"])
conns = data.get("connections") or []
if not conns:
    print("WARN: 0 active connections — generate traffic from a LAN client and retry")
    sys.exit(1)

sources = []
lan = []
fake = []
other = []
for c in conns:
    meta = c.get("metadata") or {}
    src = (meta.get("sourceIP") or "").strip()
    typ = (meta.get("type") or "").strip()
    host = (meta.get("host") or meta.get("destinationIP") or "").strip()
    sources.append((src, typ, host))
    if src.startswith(lan_prefix):
        lan.append(src)
    elif src.startswith("198.18."):
        fake.append(src)
    else:
        other.append(src or "(empty)")

print(f"connections: {len(conns)}")
print(f"LAN sourceIP ({lan_prefix}*): {len(lan)} unique={sorted(set(lan))}")
print(f"fake-ip 198.18.* (host TUN): {len(fake)}")
print(f"other/empty: {len(other)} samples={sorted(set(other))[:8]}")
print("--- sample ---")
for src, typ, host in sources[:12]:
    print(f"  {(src or '—'):15}  type={(typ or '—'):8}  host={host or '—'}")

if lan:
    print("OK: LAN client traffic is visible to mihomo — active devices should populate")
    sys.exit(0)

print("FAIL: no LAN sourceIP — clients are not entering TUN (check tun_auto_redirect / gateway route)")
sys.exit(1)
PY
