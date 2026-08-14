#!/usr/bin/env bash
# configure-smtp.sh —— OMC SMTP 配置与 app/worker 重建入口。
#
# 用法：
#   sudo bash configure-smtp.sh --config /secure/omc-smtp.env
#   sudo bash configure-smtp.sh --check
#   sudo bash configure-smtp.sh --disable
#
# --config 文件必须 chmod 600/400，脚本不会 source 它，也不会输出密码。
# 正常 apply 会用 `svc.sh start app worker`（docker compose up -d）重建容器，
# 让新环境变量生效；不能用 docker compose restart，它不会重读环境变量。

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$SCRIPT_DIR"
TARGET_ENV=""
CONFIG_FILE=""
ACTION="apply"
NO_RECREATE=0
CONFIG_CHANGED=0
GENERATED_FILE=""
MERGE_TEMP=""

SMTP_KEYS="OMCGO_NOTIFICATION_SMTP_ENABLED OMCGO_NOTIFICATION_SMTP_HOST OMCGO_NOTIFICATION_SMTP_PORT OMCGO_NOTIFICATION_SMTP_USERNAME OMCGO_NOTIFICATION_SMTP_PASSWORD OMCGO_NOTIFICATION_SMTP_FROM OMCGO_NOTIFICATION_SMTP_TLS_MODE OMCGO_NOTIFICATION_SMTP_TIMEOUT OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES"

log()  { printf '[smtp] %s\n' "$*"; }
warn() { printf '[smtp][WARN] %s\n' "$*" >&2; }
die()  { printf '[smtp][ERROR] %s\n' "$*" >&2; exit "${2:-1}"; }

cleanup() {
  [ -z "$GENERATED_FILE" ] || rm -f "$GENERATED_FILE"
  [ -z "$MERGE_TEMP" ] || rm -f "$MERGE_TEMP"
}
trap cleanup EXIT

usage() {
  sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

file_mode() {
  stat -f '%Lp' "$1" 2>/dev/null || stat -c '%a' "$1" 2>/dev/null
}

require_private_file() {
  local file="$1" mode
  [ -f "$file" ] || die "配置文件不存在：$file"
  [ ! -L "$file" ] || die "配置文件不能是符号链接：$file"
  mode="$(file_mode "$file")" || die "无法读取配置文件权限：$file"
  case "$mode" in
    400|600) ;;
    *) die "配置文件权限必须是 600 或 400，当前为 ${mode}：chmod 600 '$file'" ;;
  esac
}

raw_env_value() { # raw_env_value <key> <file>
  local key="$1" file="$2"
  awk -v key="$key" '
    index($0, "=") > 0 {
      pos=index($0, "="); name=substr($0, 1, pos-1)
      if (name == key) { count++; value=substr($0, pos+1) }
    }
    END {
      if (count == 1) print value
      else if (count > 1) exit 2
      else exit 1
    }
  ' "$file"
}

decode_env_value() { # decode_env_value <raw>
  local value="$1" first last len
  len="${#value}"
  if [ "$len" -ge 2 ]; then
    first="${value:0:1}"
    last="${value:$((len - 1)):1}"
    if [ "$first" = "'" ] && [ "$last" = "'" ]; then
      value="${value:1:$((len - 2))}"
    elif [ "$first" = '"' ] && [ "$last" = '"' ]; then
      value="${value:1:$((len - 2))}"
    fi
  fi
  printf '%s' "$value"
}

config_value() { # config_value <key> <default> <required:0|1>
  local key="$1" default="$2" required="$3" raw value rc
  raw="$(raw_env_value "$key" "$CONFIG_FILE" 2>/dev/null)"; rc=$?
  if [ "$rc" -eq 2 ]; then
    die "配置键重复：$key"
  fi
  if [ "$rc" -ne 0 ]; then
    if [ "$required" -eq 1 ]; then
      die "配置缺少必填键：$key"
    fi
    printf '%s' "$default"
    return 0
  fi
  value="$(decode_env_value "$raw")"
  printf '%s' "$value"
}

current_value() { # current_value <key> <default>
  local key="$1" default="$2" raw rc
  raw="$(raw_env_value "$key" "$TARGET_ENV" 2>/dev/null)"; rc=$?
  [ "$rc" -ne 2 ] || die "目标环境文件中的配置键重复：$key"
  [ "$rc" -eq 0 ] || raw="$default"
  decode_env_value "$raw"
}

validate_no_control_or_quote() {
  local name="$1" value="$2"
  case "$value" in
    *"'"*) die "$name 不能包含单引号；请让邮件管理员生成不含单引号的 SMTP 授权码" ;;
  esac
  if printf '%s' "$value" | LC_ALL=C grep -q '[[:cntrl:]]'; then
    die "$name 不能包含换行或控制字符"
  fi
}

validate_settings() {
  local enabled="$1" host="$2" port="$3" username="$4" password="$5" from="$6" tls_mode="$7" timeout="$8" max_bytes="$9"
  case "$enabled" in true|false) ;; *) die "ENABLED 只能是 true 或 false" ;; esac
  [ "$enabled" = false ] && return 0
  [ -n "$host" ] || die "SMTP HOST 不能为空"
  case "$host" in *[[:space:]]*|*://*|*/*) die "SMTP HOST 只能填写主机名/IP，不能包含协议、路径或空格" ;; esac
  case "$port" in ''|*[!0-9]*) die "SMTP PORT 必须是 1-65535 的整数" ;; esac
  [ "$port" -ge 1 ] && [ "$port" -le 65535 ] || die "SMTP PORT 必须是 1-65535 的整数"
  case "$tls_mode" in implicit|starttls|none) ;; *) die "TLS_MODE 只能是 implicit、starttls 或 none" ;; esac
  [ -n "$from" ] || die "SMTP FROM 不能为空"
  case "$from" in *@*.*) ;; *) die "SMTP FROM 不是有效邮箱格式" ;; esac
  case "$from" in *[[:space:]]*) die "SMTP FROM 不能包含空格" ;; esac
  if { [ -n "$username" ] && [ -z "$password" ]; } || { [ -z "$username" ] && [ -n "$password" ]; }; then
    die "USERNAME 与 PASSWORD 必须同时填写；内网免认证 Relay 则两者都留空"
  fi
  case "$password" in REPLACE_ME|CHANGE_ME|REPLACE_WITH_AUTH_CODE) die "SMTP PASSWORD 仍是占位值" ;; esac
  case "$timeout" in ''|*[!0-9a-zA-Z.]*) die "SMTP TIMEOUT 格式无效（示例 10s）" ;; esac
  case "$max_bytes" in ''|*[!0-9]*) die "MAX_ATTACHMENT_BYTES 必须是正整数" ;; esac
  [ "$max_bytes" -gt 0 ] || die "MAX_ATTACHMENT_BYTES 必须大于 0"
  validate_no_control_or_quote HOST "$host"
  validate_no_control_or_quote USERNAME "$username"
  validate_no_control_or_quote PASSWORD "$password"
  validate_no_control_or_quote FROM "$from"
}

write_setting() { # write_setting <file> <key> <value>
  local file="$1" key="$2" value="$3"
  # 单引号防止 Compose/shell 对密码中的 $, #, 空格和双引号二次展开。
  printf "%s='%s'\n" "$key" "$value" >> "$file"
}

merge_settings() { # merge_settings <generated-file> <keys>
  local generated="$1" keys="$2" backup stamp suffix
  [ -f "$TARGET_ENV" ] || die "目标环境文件不存在：$TARGET_ENV"
  [ -w "$TARGET_ENV" ] || die "目标环境文件不可写：${TARGET_ENV}（请用 sudo）"
  MERGE_TEMP="$(mktemp "${TARGET_ENV}.smtp.XXXXXX")" || die "创建临时文件失败"
  if ! awk -v keys="$keys" '
    BEGIN { n=split(keys, list, " "); for (i=1; i<=n; i++) managed[list[i]]=1; pending_blank=0 }
    {
      pos=index($0, "=")
      if (pos > 0 && substr($0, 1, pos-1) in managed) next
      if ($0 == "# OMC 邮件通知（由 configure-smtp.sh 管理）") next
      if ($0 == "") { pending_blank++; next }
      while (pending_blank > 0) { print ""; pending_blank-- }
      print
    }
  ' "$TARGET_ENV" > "$MERGE_TEMP"; then
    rm -f "$MERGE_TEMP"
    MERGE_TEMP=""
    die "过滤旧 SMTP 配置失败"
  fi
  printf '\n# OMC 邮件通知（由 configure-smtp.sh 管理）\n' >> "$MERGE_TEMP"
  cat "$generated" >> "$MERGE_TEMP"
  chmod 600 "$MERGE_TEMP" 2>/dev/null || true
  if cmp -s "$TARGET_ENV" "$MERGE_TEMP"; then
    rm -f "$MERGE_TEMP"
    MERGE_TEMP=""
    CONFIG_CHANGED=0
    log "SMTP 配置未变化，跳过写入和容器重建"
    return 0
  fi
  stamp="$(date +%Y%m%d%H%M%S)"
  backup="${TARGET_ENV}.smtp.bak.${stamp}"
  suffix=0
  while [ -e "$backup" ]; do
    suffix=$((suffix + 1))
    backup="${TARGET_ENV}.smtp.bak.${stamp}.${suffix}"
  done
  cp -p "$TARGET_ENV" "$backup" || die "备份失败：$backup"
  chmod 600 "$backup" 2>/dev/null || true
  mv "$MERGE_TEMP" "$TARGET_ENV" || die "原子替换 $TARGET_ENV 失败；备份在 $backup"
  MERGE_TEMP=""
  CONFIG_CHANGED=1
  log "已更新 ${TARGET_ENV}（备份：${backup}）"
}

recreate_services() {
  if [ "$CONFIG_CHANGED" -eq 0 ]; then
    return 0
  fi
  if [ "$NO_RECREATE" -eq 1 ]; then
    log "已跳过容器重建；稍后必须执行：bash '$DEPLOY_DIR/svc.sh' start app worker"
    return 0
  fi
  [ -f "$DEPLOY_DIR/svc.sh" ] || die "缺少 $DEPLOY_DIR/svc.sh，无法重建 app/worker"
  log "重建 app/worker 以加载新 SMTP 环境变量..."
  (cd "$DEPLOY_DIR" && bash svc.sh start app worker) || die "app/worker 重建失败；可用备份恢复 $TARGET_ENV"
}

show_summary() {
  local enabled="$1" host="$2" port="$3" username="$4" from="$5" tls_mode="$6"
  log "enabled=$enabled host=${host:-<empty>} port=$port tls=$tls_mode from=${from:-<empty>} auth=$([ -n "$username" ] && printf configured || printf none)"
}

apply_config() {
  local enabled=true host port username password from tls_mode timeout max_bytes
  require_private_file "$CONFIG_FILE"
  host="$(config_value OMCGO_NOTIFICATION_SMTP_HOST '' 1)" || exit $?
  port="$(config_value OMCGO_NOTIFICATION_SMTP_PORT 465 0)" || exit $?
  username="$(config_value OMCGO_NOTIFICATION_SMTP_USERNAME '' 0)" || exit $?
  password="$(config_value OMCGO_NOTIFICATION_SMTP_PASSWORD '' 0)" || exit $?
  from="$(config_value OMCGO_NOTIFICATION_SMTP_FROM '' 1)" || exit $?
  tls_mode="$(config_value OMCGO_NOTIFICATION_SMTP_TLS_MODE implicit 0)" || exit $?
  timeout="$(config_value OMCGO_NOTIFICATION_SMTP_TIMEOUT 10s 0)" || exit $?
  max_bytes="$(config_value OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES 20971520 0)" || exit $?
  validate_settings "$enabled" "$host" "$port" "$username" "$password" "$from" "$tls_mode" "$timeout" "$max_bytes"
  GENERATED_FILE="$(mktemp)" || die "创建 SMTP 临时配置失败"
  chmod 600 "$GENERATED_FILE" 2>/dev/null || true
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_ENABLED "$enabled"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_HOST "$host"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_PORT "$port"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_USERNAME "$username"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_PASSWORD "$password"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_FROM "$from"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_TLS_MODE "$tls_mode"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_TIMEOUT "$timeout"
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES "$max_bytes"
  merge_settings "$GENERATED_FILE" "$SMTP_KEYS"
  rm -f "$GENERATED_FILE"
  GENERATED_FILE=""
  show_summary "$enabled" "$host" "$port" "$username" "$from" "$tls_mode"
  recreate_services
}

disable_smtp() {
  GENERATED_FILE="$(mktemp)" || die "创建 SMTP 临时配置失败"
  chmod 600 "$GENERATED_FILE" 2>/dev/null || true
  write_setting "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_ENABLED false
  merge_settings "$GENERATED_FILE" OMCGO_NOTIFICATION_SMTP_ENABLED
  rm -f "$GENERATED_FILE"
  GENERATED_FILE=""
  log "SMTP 已禁用；原主机/账号参数保留，便于再次启用"
  recreate_services
}

check_current() {
  local enabled host port username password from tls_mode timeout max_bytes
  enabled="$(current_value OMCGO_NOTIFICATION_SMTP_ENABLED false)" || exit $?
  host="$(current_value OMCGO_NOTIFICATION_SMTP_HOST '')" || exit $?
  port="$(current_value OMCGO_NOTIFICATION_SMTP_PORT 465)" || exit $?
  username="$(current_value OMCGO_NOTIFICATION_SMTP_USERNAME '')" || exit $?
  password="$(current_value OMCGO_NOTIFICATION_SMTP_PASSWORD '')" || exit $?
  from="$(current_value OMCGO_NOTIFICATION_SMTP_FROM omc-alert@example.com)" || exit $?
  tls_mode="$(current_value OMCGO_NOTIFICATION_SMTP_TLS_MODE implicit)" || exit $?
  timeout="$(current_value OMCGO_NOTIFICATION_SMTP_TIMEOUT 10s)" || exit $?
  max_bytes="$(current_value OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES 20971520)" || exit $?
  validate_settings "$enabled" "$host" "$port" "$username" "$password" "$from" "$tls_mode" "$timeout" "$max_bytes"
  show_summary "$enabled" "$host" "$port" "$username" "$from" "$tls_mode"
  log "当前 SMTP 配置格式有效（未执行网络认证/投递）"
}

main() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --config) [ $# -ge 2 ] || die "--config 缺少文件"; CONFIG_FILE="$2"; shift 2 ;;
      --deploy-dir) [ $# -ge 2 ] || die "--deploy-dir 缺少目录"; DEPLOY_DIR="$2"; shift 2 ;;
      --disable) ACTION="disable"; shift ;;
      --check) ACTION="check"; shift ;;
      --no-recreate) NO_RECREATE=1; shift ;;
      -h|--help) usage 0 ;;
      *) die "未知参数：$1" ;;
    esac
  done
  TARGET_ENV="$DEPLOY_DIR/.env"
  [ -f "$TARGET_ENV" ] || die "未找到部署环境文件：$TARGET_ENV"
  case "$ACTION" in
    apply) [ -n "$CONFIG_FILE" ] || die "apply 需要 --config /secure/omc-smtp.env"; apply_config ;;
    disable) disable_smtp ;;
    check) check_current ;;
  esac
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  main "$@"
fi
