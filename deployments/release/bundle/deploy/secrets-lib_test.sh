#!/usr/bin/env bash
# secrets-lib_test.sh —— secrets-lib.sh 单测（#175）。运行：bash secrets-lib_test.sh
set -uo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$DIR/secrets-lib.sh"

PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); }
bad()  { FAIL=$((FAIL+1)); echo "  ✗ FAIL: $*"; }
chk()  { if [ "$2" = "$3" ]; then ok; else bad "$1: got '$2' want '$3'"; fi; }
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

echo "── secrets_get_val ──"
printf 'A=1\nPOSTGRES_PASSWORD=p@ss=w0rd\nB=2\n' > "$TMP/e1"
chk "get value with = in it" "$(secrets_get_val POSTGRES_PASSWORD "$TMP/e1")" "p@ss=w0rd"
chk "get missing key" "$(secrets_get_val NOPE "$TMP/e1")" ""
chk "get from missing file" "$(secrets_get_val A "$TMP/none")" ""

echo "── secrets_is_default_value ──"
secrets_is_default_value POSTGRES_PASSWORD omcgo123 && ok || bad "omcgo123 应判默认"
secrets_is_default_value MINIO_ROOT_PASSWORD minioadmin && ok || bad "minioadmin 应判默认"
secrets_is_default_value OMC_SHARED_SECRET dps && ok || bad "dps 应判默认"
secrets_is_default_value POSTGRES_PASSWORD "" && ok || bad "空 应判默认"
secrets_is_default_value POSTGRES_PASSWORD REPLACE_ME && ok || bad "REPLACE_ME 应判默认"
secrets_is_default_value POSTGRES_PASSWORD "a1b2c3d4e5f6strongrand" && bad "强随机不应判默认" || ok

echo "── secrets_generate_to（强随机 + 权限 600 + 非默认）──"
secrets_generate_to "$TMP/sec"
chk "MINIO_ROOT_USER 固定 omcadmin" "$(secrets_get_val MINIO_ROOT_USER "$TMP/sec")" "omcadmin"
chk "文件权限 600" "$(stat -f '%Lp' "$TMP/sec" 2>/dev/null || stat -c '%a' "$TMP/sec" 2>/dev/null)" "600"
PW="$(secrets_get_val POSTGRES_PASSWORD "$TMP/sec")"
[ -n "$PW" ] && ! secrets_is_default_value POSTGRES_PASSWORD "$PW" && ok || bad "生成的 PG 口令应非空非默认"
JWT="$(secrets_get_val OMCGO_JWT_SECRET "$TMP/sec")"
[ "${#JWT}" -eq 64 ] && ok || bad "JWT 应 32 字节 hex=64 字符，got ${#JWT}"
# 两次生成应不同（随机性）
secrets_generate_to "$TMP/sec2"
[ "$(secrets_get_val POSTGRES_PASSWORD "$TMP/sec")" != "$(secrets_get_val POSTGRES_PASSWORD "$TMP/sec2")" ] && ok || bad "两次生成应不同"

echo "── secrets_import_to（存量迁移：从现行 .env 抽 6 键）──"
printf 'POSTGRES_PASSWORD=oldstrong123\nMINIO_ROOT_USER=opsuser\nMINIO_ROOT_PASSWORD=oldminio\nOMCGO_JWT_SECRET=oldjwt\nOMC_SHARED_SECRET=oldshared\nGRAFANA_ADMIN_PASSWORD=oldgraf\nOMC_PUBLIC_HOST=1.2.3.4\n' > "$TMP/old"
secrets_import_to "$TMP/old" "$TMP/imp"
chk "import PG" "$(secrets_get_val POSTGRES_PASSWORD "$TMP/imp")" "oldstrong123"
chk "import MINIO user" "$(secrets_get_val MINIO_ROOT_USER "$TMP/imp")" "opsuser"
chk "import 不带非密钥键" "$(secrets_get_val OMC_PUBLIC_HOST "$TMP/imp")" ""

echo "── secrets_apply_to_env（覆盖 .env 的 6 键，保留其它行）──"
printf 'IMAGE_APP=x\nPOSTGRES_USER=omcgo\nPOSTGRES_PASSWORD=REPLACE_ME\nMINIO_ROOT_USER=REPLACE_ME\nOMC_PUBLIC_HOST=10.0.0.1\n' > "$TMP/env"
secrets_apply_to_env "$TMP/imp" "$TMP/env"
chk "apply 覆盖 PG" "$(secrets_get_val POSTGRES_PASSWORD "$TMP/env")" "oldstrong123"
chk "apply 覆盖 MINIO user" "$(secrets_get_val MINIO_ROOT_USER "$TMP/env")" "opsuser"
chk "apply 保留非密钥 IMAGE_APP" "$(secrets_get_val IMAGE_APP "$TMP/env")" "x"
chk "apply 保留 OMC_PUBLIC_HOST" "$(secrets_get_val OMC_PUBLIC_HOST "$TMP/env")" "10.0.0.1"
# secrets 有但 env 没有的键应追加
chk "apply 追加缺失的 OMC_SHARED_SECRET" "$(secrets_get_val OMC_SHARED_SECRET "$TMP/env")" "oldshared"

echo '── apply 特殊字符口令(含 = 和 美元符 & )安全 ──'
printf 'POSTGRES_PASSWORD=a=b$c&d\nMINIO_ROOT_USER=u\nMINIO_ROOT_PASSWORD=m\nOMCGO_JWT_SECRET=j\nOMC_SHARED_SECRET=s\nGRAFANA_ADMIN_PASSWORD=g\n' > "$TMP/sp"
printf 'POSTGRES_PASSWORD=REPLACE_ME\n' > "$TMP/env2"
secrets_apply_to_env "$TMP/sp" "$TMP/env2"
chk "特殊字符口令原样" "$(secrets_get_val POSTGRES_PASSWORD "$TMP/env2")" 'a=b$c&d'

echo "── ensure_secrets 编排（3 场景 + 迁移安全）──"
log()  { :; }   # 桩
warn() { :; }
COMPOSE_PROJECT=omcgo
DOCKER_VOL_EXISTS=0
docker() {  # 桩：仅模拟 `docker volume inspect`
  if [ "${1:-}" = "volume" ] && [ "${2:-}" = "inspect" ]; then [ "$DOCKER_VOL_EXISTS" = "1" ]; return $?; fi
  return 0
}
setup_omc() {  # $1 = 写到 current/deploy/.env 的内容；清空 secrets.env / .env.saved
  OMC_ROOT="$TMP/omc.$$.$RANDOM"; mkdir -p "$OMC_ROOT/current/deploy" "$OMC_ROOT/etc"
  printf '%s' "$1" > "$OMC_ROOT/current/deploy/.env"
}

# A. 全新装：无 secrets.env、无卷、.env=REPLACE_ME → 生成强随机
DOCKER_VOL_EXISTS=0
setup_omc 'POSTGRES_USER=omcgo
POSTGRES_PASSWORD=REPLACE_ME
MINIO_ROOT_USER=REPLACE_ME
MINIO_ROOT_PASSWORD=REPLACE_ME
OMCGO_JWT_SECRET=REPLACE_ME
OMC_SHARED_SECRET=REPLACE_ME
GRAFANA_ADMIN_PASSWORD=REPLACE_ME
OMC_PUBLIC_HOST=1.2.3.4
'
ensure_secrets
[ -f "$OMC_ROOT/etc/secrets.env" ] && ok || bad "A: secrets.env 应生成"
PWA="$(secrets_get_val POSTGRES_PASSWORD "$OMC_ROOT/etc/secrets.env")"
{ [ -n "$PWA" ] && ! secrets_is_default_value POSTGRES_PASSWORD "$PWA"; } && ok || bad "A: 应生成强随机 PG 口令"
chk "A: .env 被覆盖为强随机" "$(secrets_get_val POSTGRES_PASSWORD "$OMC_ROOT/current/deploy/.env")" "$PWA"
chk "A: .env 非密钥键保留" "$(secrets_get_val OMC_PUBLIC_HOST "$OMC_ROOT/current/deploy/.env")" "1.2.3.4"

# B. 存量卷迁移：卷存在、.env 有旧强口令 → 导入，绝不新生成
DOCKER_VOL_EXISTS=1
setup_omc 'POSTGRES_PASSWORD=oldStrongVolPw
MINIO_ROOT_USER=opsuser
MINIO_ROOT_PASSWORD=oldminio
OMCGO_JWT_SECRET=oldjwt
OMC_SHARED_SECRET=oldshared
GRAFANA_ADMIN_PASSWORD=oldgraf
'
ensure_secrets
chk "B: 导入旧卷口令(不新生成)" "$(secrets_get_val POSTGRES_PASSWORD "$OMC_ROOT/etc/secrets.env")" "oldStrongVolPw"

# C. 幂等：secrets.env 已存在 → 复用，不变
DOCKER_VOL_EXISTS=0
setup_omc 'POSTGRES_PASSWORD=REPLACE_ME
'
printf 'POSTGRES_PASSWORD=existingSecret\nMINIO_ROOT_USER=omcadmin\nMINIO_ROOT_PASSWORD=m\nOMCGO_JWT_SECRET=j\nOMC_SHARED_SECRET=s\nGRAFANA_ADMIN_PASSWORD=g\n' > "$OMC_ROOT/etc/secrets.env"
ensure_secrets
chk "C: 复用既有 secrets.env" "$(secrets_get_val POSTGRES_PASSWORD "$OMC_ROOT/etc/secrets.env")" "existingSecret"
chk "C: .env 被既有 secrets 覆盖" "$(secrets_get_val POSTGRES_PASSWORD "$OMC_ROOT/current/deploy/.env")" "existingSecret"

# D. 迁移安全（关键）：卷存在 + .env 是默认口令 → 仍导入默认(匹配旧卷)，绝不生成新强随机(否则连不上)
DOCKER_VOL_EXISTS=1
setup_omc 'POSTGRES_PASSWORD=omcgo123
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
OMCGO_JWT_SECRET=REPLACE_ME
OMC_SHARED_SECRET=dps
GRAFANA_ADMIN_PASSWORD=admin
'
ensure_secrets
chk "D: 卷在时不瞎生成,导入默认匹配旧卷" "$(secrets_get_val POSTGRES_PASSWORD "$OMC_ROOT/etc/secrets.env")" "omcgo123"

echo ""
echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]
