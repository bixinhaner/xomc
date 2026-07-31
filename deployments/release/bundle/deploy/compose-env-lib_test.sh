#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

if [ ! -f "$SCRIPT_DIR/compose-env-lib.sh" ]; then
  echo "FAIL: compose-env-lib.sh 不存在" >&2
  exit 1
fi
. "$SCRIPT_DIR/compose-env-lib.sh"

cat >"$TMP/.env" <<'EOF'
IMAGE_APP=omcgo/app:test
OMC_PUBLIC_HOST=
PM_AGGREGATION_FINALIZE_CONCURRENCY=4
EOF
cat >"$TMP/resources.env" <<'EOF'
OMC_PUBLIC_HOST=172.24.224.197
PM_AGGREGATION_FINALIZE_CONCURRENCY=32
EOF

unset OMC_PUBLIC_HOST PM_AGGREGATION_FINALIZE_CONCURRENCY
if deploy_env_load "$TMP/.env" "$TMP/resources.env"; then
  ok
else
  bad "双 env 文件加载失败"
fi
[ "${OMC_PUBLIC_HOST:-}" = "172.24.224.197" ] && ok ||
  bad "resources.env 应覆盖 .env 的空公网地址，实际=${OMC_PUBLIC_HOST:-<unset>}"
[ "${PM_AGGREGATION_FINALIZE_CONCURRENCY:-}" = "32" ] && ok ||
  bad "resources.env 应覆盖 .env 的聚合并发，实际=${PM_AGGREGATION_FINALIZE_CONCURRENCY:-<unset>}"

effective="$(deploy_env_effective_value OMC_PUBLIC_HOST "$TMP/.env" "$TMP/resources.env")"
[ "$effective" = "172.24.224.197" ] && ok ||
  bad "effective value 应遵循 Compose env-file 顺序，实际=$effective"

cat >"$TMP/host.env" <<'EOF'
OMC_PUBLIC_HOST=172.24.224.197
EOF
cat >"$TMP/no-host.env" <<'EOF'
PM_AGGREGATION_FINALIZE_CONCURRENCY=32
EOF
effective="$(deploy_env_effective_value OMC_PUBLIC_HOST "$TMP/host.env" "$TMP/no-host.env")"
[ "$effective" = "172.24.224.197" ] && ok ||
  bad "后置 env 不含目标键时必须保留前置值，实际=$effective"

if deploy_env_public_host_valid "$effective"; then
  ok
else
  bad "合法公网地址被拒绝"
fi
for invalid in "" localhost 127.0.0.1 0.0.0.0 'http://1.2.3.4' 'host/path'; do
  if deploy_env_public_host_valid "$invalid"; then
    bad "不可供基站访问的地址不应通过: <$invalid>"
  else
    ok
  fi
done

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]
