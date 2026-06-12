#!/usr/bin/env bash
# reset_password_test.sh —— reset_password.sh 纯函数单测（#239）。运行：bash reset_password_test.sh
# 只测可隔离的 set_secret_key / random_for（文件改写 + 随机长度）；交互/docker 路径靠
# bash -n + shellcheck + 人工/灰度部署验证（见 issue #239 验收）。
set -uo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# source 被守卫的脚本：只加载函数，不跑 main（BASH_SOURCE != $0）。
. "$DIR/reset_password.sh"

PASS=0; FAIL=0
ok()  { PASS=$((PASS+1)); }
bad() { FAIL=$((FAIL+1)); echo "  ✗ FAIL: $*"; }
chk() { if [ "$2" = "$3" ]; then ok; else bad "$1: got '$2' want '$3'"; fi; }
perm_of() { stat -f '%Lp' "$1" 2>/dev/null || stat -c '%a' "$1" 2>/dev/null; }
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

echo "── random_for（随机长度对齐 secrets_generate_to）──"
len_of() { local v; v="$(random_for "$1")"; printf '%s' "${#v}"; }
chk "PG hex 长度 48"      "$(len_of POSTGRES_PASSWORD)"      "48"
chk "MinIO hex 长度 48"   "$(len_of MINIO_ROOT_PASSWORD)"    "48"
chk "JWT hex 长度 64"     "$(len_of OMCGO_JWT_SECRET)"       "64"
chk "SHARED hex 长度 64"  "$(len_of OMC_SHARED_SECRET)"      "64"
chk "Grafana hex 长度 32" "$(len_of GRAFANA_ADMIN_PASSWORD)" "32"
[ "$(random_for POSTGRES_PASSWORD)" != "$(random_for POSTGRES_PASSWORD)" ] && ok || bad "两次随机应不同"

echo "── set_secret_key（改 secrets.env 权威 + 同步 .env，保留其它键/行）──"
# 全局：set_secret_key 读 $SECRETS_FILE / $ENV_FILE
SECRETS_FILE="$TMP/secrets.env"
ENV_FILE="$TMP/.env"
( umask 077; printf 'POSTGRES_PASSWORD=oldpg\nMINIO_ROOT_USER=omcadmin\nMINIO_ROOT_PASSWORD=oldminio\nOMCGO_JWT_SECRET=oldjwt\nOMC_SHARED_SECRET=oldshared\nGRAFANA_ADMIN_PASSWORD=oldgraf\n' > "$SECRETS_FILE" )
printf 'IMAGE_APP=x\nPOSTGRES_USER=omcgo\nPOSTGRES_PASSWORD=oldpg\nMINIO_ROOT_PASSWORD=oldminio\nOMC_PUBLIC_HOST=10.0.0.1\n' > "$ENV_FILE"

set_secret_key POSTGRES_PASSWORD 'n3w-pg-pass'
chk "secrets.env PG 改新值"      "$(secrets_get_val POSTGRES_PASSWORD "$SECRETS_FILE")" "n3w-pg-pass"
chk "secrets.env 其它键保留(JWT)" "$(secrets_get_val OMCGO_JWT_SECRET "$SECRETS_FILE")" "oldjwt"
chk "secrets.env 权限保持 600"    "$(perm_of "$SECRETS_FILE")" "600"
chk ".env 同步 PG 新值"          "$(secrets_get_val POSTGRES_PASSWORD "$ENV_FILE")" "n3w-pg-pass"
chk ".env 非密钥行保留(IMAGE_APP)" "$(secrets_get_val IMAGE_APP "$ENV_FILE")" "x"
chk ".env 非密钥行保留(PUBLIC_HOST)" "$(secrets_get_val OMC_PUBLIC_HOST "$ENV_FILE")" "10.0.0.1"
# 备份文件应生成
ls "$SECRETS_FILE".bak.* >/dev/null 2>&1 && ok || bad "应生成 secrets.env.bak.<ts> 备份"
# 不应产生重复 KEY 行
chk "PG 行恰好 1 行(无重复)" "$(grep -c '^POSTGRES_PASSWORD=' "$SECRETS_FILE")" "1"

echo "── set_secret_key 特殊字符口令（含 = \$ &）字面安全 ──"
set_secret_key OMC_SHARED_SECRET 'a=b$c&d=e'
chk "特殊字符 secrets.env 原样" "$(secrets_get_val OMC_SHARED_SECRET "$SECRETS_FILE")" 'a=b$c&d=e'

echo "── set_secret_key 连续两次（第二次覆盖，仍单行）──"
set_secret_key POSTGRES_PASSWORD 'second'
chk "第二次覆盖生效" "$(secrets_get_val POSTGRES_PASSWORD "$SECRETS_FILE")" "second"
chk "覆盖后仍单行"   "$(grep -c '^POSTGRES_PASSWORD=' "$SECRETS_FILE")" "1"

echo "── set_secret_key 无 .env 时也不报错（仅改 secrets.env）──"
rm -f "$ENV_FILE"
set_secret_key GRAFANA_ADMIN_PASSWORD 'graf2' && ok || bad "无 .env 时应仍成功"
chk "无 .env 仍改 secrets.env" "$(secrets_get_val GRAFANA_ADMIN_PASSWORD "$SECRETS_FILE")" "graf2"

echo ""
echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]
