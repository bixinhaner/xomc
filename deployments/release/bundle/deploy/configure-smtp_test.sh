#!/usr/bin/env bash
set -uo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$DIR/configure-smtp.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }

mkdir -p "$TMP/deploy"
cat > "$TMP/deploy/.env" <<'EOF'
IMAGE_APP=omcgo/app:test
OMC_PUBLIC_HOST=192.0.2.10
OMCGO_NOTIFICATION_SMTP_ENABLED=false
EOF
chmod 600 "$TMP/deploy/.env"

cat > "$TMP/smtp.env" <<'EOF'
OMCGO_NOTIFICATION_SMTP_HOST=smtp.qiye.example.com
OMCGO_NOTIFICATION_SMTP_PORT=465
OMCGO_NOTIFICATION_SMTP_USERNAME=omc-alert@example.com
OMCGO_NOTIFICATION_SMTP_PASSWORD=p@ss$word#123
OMCGO_NOTIFICATION_SMTP_FROM=omc-alert@example.com
OMCGO_NOTIFICATION_SMTP_TLS_MODE=implicit
OMCGO_NOTIFICATION_SMTP_TIMEOUT=10s
OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES=20971520
EOF
chmod 600 "$TMP/smtp.env"

output="$(bash "$SCRIPT" --deploy-dir "$TMP/deploy" --config "$TMP/smtp.env" --no-recreate 2>&1)"
rc=$?
[ "$rc" -eq 0 ] && ok || bad "合法配置 apply 失败：$output"
if printf '%s' "$output" | grep -Fq 'p@ss$word#123'; then
  bad "脚本输出泄露 SMTP 密码"
else
  ok
fi
[ "$(grep -c '^OMCGO_NOTIFICATION_SMTP_PASSWORD=' "$TMP/deploy/.env")" -eq 1 ] && ok || bad "SMTP password 键未去重"
grep -Fq "OMCGO_NOTIFICATION_SMTP_PASSWORD='p@ss\$word#123'" "$TMP/deploy/.env" && ok || bad "特殊字符密码未按字面量写入"
grep -Fq 'IMAGE_APP=omcgo/app:test' "$TMP/deploy/.env" && ok || bad "更新 SMTP 时破坏了其它 env 键"

before="$(cksum "$TMP/deploy/.env")"
backup_before="$(find "$TMP/deploy" -name '.env.smtp.bak.*' -type f | wc -l | tr -d ' ')"
repeat_output="$(bash "$SCRIPT" --deploy-dir "$TMP/deploy" --config "$TMP/smtp.env" --no-recreate 2>&1)"
after="$(cksum "$TMP/deploy/.env")"
[ "$before" = "$after" ] && ok || bad "重复 apply 不是幂等的"
backup_after="$(find "$TMP/deploy" -name '.env.smtp.bak.*' -type f | wc -l | tr -d ' ')"
[ "$backup_before" = "$backup_after" ] && ok || bad "无变化 apply 不应产生新备份"
printf '%s' "$repeat_output" | grep -Fq '跳过写入和容器重建' && ok || bad "无变化 apply 未明确跳过重建"

bash "$SCRIPT" --deploy-dir "$TMP/deploy" --disable --no-recreate >/dev/null 2>&1
grep -Fq "OMCGO_NOTIFICATION_SMTP_ENABLED='false'" "$TMP/deploy/.env" && ok || bad "disable 未关闭 SMTP"
grep -Fq "OMCGO_NOTIFICATION_SMTP_PASSWORD='p@ss\$word#123'" "$TMP/deploy/.env" && ok || bad "disable 不应删除已有参数"
bash "$SCRIPT" --deploy-dir "$TMP/deploy" --check >/dev/null 2>&1 && ok || bad "禁用状态 check 应通过"

chmod 644 "$TMP/smtp.env"
if bash "$SCRIPT" --deploy-dir "$TMP/deploy" --config "$TMP/smtp.env" --no-recreate >/dev/null 2>&1; then
  bad "权限过宽的配置文件不应通过"
else
  ok
fi

chmod 600 "$TMP/smtp.env"
sed 's/TLS_MODE=implicit/TLS_MODE=invalid/' "$TMP/smtp.env" > "$TMP/invalid.env"
chmod 600 "$TMP/invalid.env"
if bash "$SCRIPT" --deploy-dir "$TMP/deploy" --config "$TMP/invalid.env" --no-recreate >/dev/null 2>&1; then
  bad "非法 TLS_MODE 不应通过"
else
  ok
fi

printf "OMCGO_NOTIFICATION_SMTP_PORT='587'\n" >> "$TMP/deploy/.env"
if bash "$SCRIPT" --deploy-dir "$TMP/deploy" --check >/dev/null 2>&1; then
  bad "重复的目标环境键不应通过 check"
else
  ok
fi

printf '════ Results: PASS=%s FAIL=%s ════\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
