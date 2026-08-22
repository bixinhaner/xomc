#!/usr/bin/env bash
# reset_password.sh —— 逐组件密码轮换 (#239)。内网交付侧 docker compose 部署专用。
#
# 背景：#175 自动生成强随机凭证到 etc/secrets.env（6 键唯一权威源），但只解决「生成」，
# 没解决「轮换/改密码」。本脚本交互式、**逐个组件**更新密码（不支持批量），密码可随机
# 生成、也可手动输入，并保证三处一致：secrets.env（权威）↔ 组件实际口令 ↔ .env/客户端。
#
# 关键：不同组件后端存口令的方式不同，没有一招通吃——
#   PostgreSQL：口令固化在数据卷 pg_authid（env 仅 initdb）→ 容器内 ALTER ROLE 改卷内口令。
#   MinIO     ：每次启动读 env（不入卷）            → 改 env + up -d 重建读新口令。
#   Grafana   ：口令在卷 grafana.db（env 仅首次）   → 容器内 grafana cli reset-admin-password。
#   JWT/共享密钥：无状态（签名/HMAC 密钥）          → 改 env + up -d 客户端重载。
#
# 这 6 个口令都只用于「鉴权/连接」，没有一个是数据加密密钥——轮换任何一个都不会导致旧数据
# 不可用（PG 表 / MinIO 对象 / 配置备份均不依赖这些口令加密；配置备份用独立的
# OMC_BACKUP_ENCRYPTION_KEY，不在本脚本范围）。
#
# 用法：
#   sudo bash reset_password.sh [pg|minio|grafana|jwt|shared] [--omc-root DIR]
#   不带组件参数时弹出菜单，必须单选 1 个。
#
# 测试：本脚本可被 source（main 受 BASH_SOURCE 守卫），见 reset_password_test.sh。

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=secrets-lib.sh
. "$DIR/secrets-lib.sh" || { echo "ERROR: 无法加载 secrets-lib.sh" >&2; exit 1; }
if [ -f "$DIR/docker-network-lib.sh" ]; then
  . "$DIR/docker-network-lib.sh"
elif [ -f "$DIR/../../../docker/docker-network-lib.sh" ]; then
  . "$DIR/../../../docker/docker-network-lib.sh"
else
  echo "ERROR: 缺少 docker-network-lib.sh，无法按 DOCKER_BIP 规划 Docker 网段" >&2
  exit 1
fi

# ---- 配置（可被环境变量 / 参数覆盖）----
OMC_ROOT="${OMC_ROOT:-/opt/omc}"
COMPOSE_PROJECT="omcgo"
# SECRETS_FILE / ENV_FILE / DEPLOY_DIR 在 main 解析参数后据 OMC_ROOT 计算。
SECRETS_FILE=""
ENV_FILE=""
DEPLOY_DIR=""
COMPOSE=""
SELECTED=""
NEW_VALUE=""
WAS_RANDOM=0

# ---- 日志/提示一律走 stderr，避免污染 $(...) 捕获的密码值 ----
log()  { printf '%s\n' "$*" >&2; }
warn() { printf 'WARN: %s\n' "$*" >&2; }
die()  { printf 'ERROR: %s\n' "$*" >&2; exit "${2:-1}"; }

# ── set_secret_key <KEY> <VALUE> ───────────────────────────────────────────────
# 改 secrets.env（权威）：备份 → 删旧 KEY= 行 → printf 字面追加新值（含特殊字符安全）→
# chmod 600 → secrets_apply_to_env 同步到 .env。printf 字面写入不经 shell 解释，= / $ / & 安全。
set_secret_key() {
  local key="$1" val="$2" tmp ts
  [ -f "$SECRETS_FILE" ] || die "secrets.env 不存在：$SECRETS_FILE"
  ts="$(date +%Y%m%d%H%M%S)"
  cp -p "$SECRETS_FILE" "$SECRETS_FILE.bak.$ts" || die "备份 secrets.env 失败"
  tmp="$(mktemp)" || die "mktemp 失败"
  # 删所有旧 KEY= 行（去重防御），其它行原样保留；KEY 均为 [A-Z_]+ 无正则元字符。
  grep -v "^${key}=" "$SECRETS_FILE" > "$tmp" 2>/dev/null || true
  printf '%s=%s\n' "$key" "$val" >> "$tmp"
  ( umask 077; cat "$tmp" > "$SECRETS_FILE" )
  rm -f "$tmp"
  chmod 600 "$SECRETS_FILE" 2>/dev/null || true
  # 同步到 .env（若存在）：secrets.env 权威覆盖其 6 键，保留其它行。
  if [ -f "$ENV_FILE" ]; then
    secrets_apply_to_env "$SECRETS_FILE" "$ENV_FILE" \
      || warn "同步 $ENV_FILE 失败，请人工核对其 6 个密钥键"
  fi
}

# ── random_for <KEY> ───────────────────────────────────────────────────────────
# 随机长度对齐 secrets_generate_to：PG/MinIO=rand_hex 24、JWT/SHARED=rand_hex 32、Grafana=16。
random_for() {
  case "$1" in
    POSTGRES_PASSWORD|MINIO_ROOT_PASSWORD) rand_hex 24 ;;
    OMCGO_JWT_SECRET|OMC_SHARED_SECRET)    rand_hex 32 ;;
    GRAFANA_ADMIN_PASSWORD)                rand_hex 16 ;;
    *) die "random_for：未知 key $1" ;;
  esac
}

# ── detect_compose ─────────────────────────────────────────────────────────────
detect_compose() {
  if docker compose version >/dev/null 2>&1; then
    COMPOSE="docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE="docker-compose"
  else
    die "未找到 docker compose（V2 plugin 或 V1 standalone）"
  fi
}

# ── build_dc ───────────────────────────────────────────────────────────────────
# 构造全局 DC 数组：cd 部署目录 + -p omcgo + 按存在拼 --env-file 与 -f 列表
# （兼容 --skip-web/--skip-monitoring 部署，不存在的 compose 文件自动跳过）。
build_dc() {
  cd "$DEPLOY_DIR" || die "进不去部署目录：$DEPLOY_DIR"
  docker_network_resolve_bip "$DEPLOY_DIR/.env" ||
    die "无法解析 Docker 网段规划（默认值或自定义 DOCKER_BIP 均不可用）"
  docker_network_plan || die "DOCKER_BIP 无效或无法派生 Docker 网段：${DOCKER_BIP:-<空>}"
  # shellcheck disable=SC2206  # $COMPOSE 需按词拆分（"docker compose" → 两元素）
  DC=( $COMPOSE -p "$COMPOSE_PROJECT" )
  [ -f .env ]           && DC+=( --env-file .env )
  [ -f resources.env ]  && DC+=( --env-file resources.env )
  local f
  for f in docker-compose.infra.yml docker-compose.app.yml \
           docker-compose.web.yml docker-compose.monitoring.yml; do
    [ -f "$f" ] && DC+=( -f "$f" )
  done
}

# ── require_running <container> ────────────────────────────────────────────────
require_running() {
  docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$1" \
    || die "容器未运行：$1（请先 install.sh 起栈，或该组件未部署）"
}

# ── preflight ──────────────────────────────────────────────────────────────────
preflight() {
  [ "$(id -u)" -eq 0 ] || die "请用 root 运行（sudo bash reset_password.sh ...）"
  command -v docker >/dev/null 2>&1 || die "未找到 docker"
  detect_compose
  [ -f "$SECRETS_FILE" ] \
    || die "未找到 $SECRETS_FILE —— 这不是 #175 初始化的部署，无法安全轮换"
}

# ── 组件选择菜单（必须单选）─────────────────────────────────────────────────────
select_component_menu() {
  {
    printf '请选择要轮换密码的组件（必须单选）：\n'
    printf '  1) PostgreSQL      (POSTGRES_PASSWORD)\n'
    printf '  2) MinIO           (MINIO_ROOT_PASSWORD)\n'
    printf '  3) Grafana         (GRAFANA_ADMIN_PASSWORD)\n'
    printf '  4) JWT 签名密钥     (OMCGO_JWT_SECRET)\n'
    printf '  5) TR-069 共享密钥  (OMC_SHARED_SECRET)\n'
    printf '选择 [1-5]: '
  } >&2
  local c
  read -r c || die "需交互式终端（或用位置参数指定组件：pg|minio|grafana|jwt|shared）"
  printf '%s' "${c:-}"
}

# ── 取新值：随机（默认）或手动（双次确认、非空、JWT/SHARED≥32）─────────────────────
# 结果写全局 NEW_VALUE / WAS_RANDOM，避免 $(...) 捕获时的提示串扰。
acquire_value() {
  local key="$1" choice val val2 min=0
  case "$key" in OMCGO_JWT_SECRET|OMC_SHARED_SECRET) min=32 ;; esac
  {
    printf '取新值方式：\n'
    printf '  1) 随机生成（默认/推荐）\n'
    printf '  2) 手动输入\n'
    printf '选择 [1]: '
  } >&2
  read -r choice || die "需交互式终端读取选择"
  case "${choice:-1}" in
    2)
      while :; do
        printf '输入新密码（不回显）: ' >&2; read -rs val  || die "输入中断"; printf '\n' >&2
        printf '再次确认: '             >&2; read -rs val2 || die "输入中断"; printf '\n' >&2
        if [ -z "$val" ]; then warn "不能为空，请重试"; continue; fi
        if [ "$val" != "$val2" ]; then warn "两次输入不一致，请重试"; continue; fi
        if [ "$min" -gt 0 ] && [ "${#val}" -lt "$min" ]; then
          warn "$key 至少 $min 字符（当前 ${#val}），请重试"; continue
        fi
        break
      done
      NEW_VALUE="$val"; WAS_RANDOM=0
      ;;
    *)
      NEW_VALUE="$(random_for "$key")"; WAS_RANDOM=1
      ;;
  esac
}

# ── 各组件轮换 ─────────────────────────────────────────────────────────────────
# PostgreSQL：口令固化在卷。顺序——先 ALTER（即时生效、旧连接不断），成功后再改
# secrets.env/.env，再 up -d 客户端用新口令重连。ALTER 失败则中止、不动文件，
# 避免「DB 旧、客户端新」锁死。
rotate_pg() {
  local new="$1" pguser pgdb cname
  cname="${COMPOSE_PROJECT}-postgres-1"
  require_running "$cname"
  pguser="$(secrets_get_val POSTGRES_USER "$ENV_FILE")"; pguser="${pguser:-omcgo}"
  pgdb="$(secrets_get_val POSTGRES_DB "$ENV_FILE")";     pgdb="${pgdb:-omcgo}"
  log "PostgreSQL：容器内 ALTER ROLE 改卷内口令（本地 socket trust，:'pw' 参数化防注入）..."
  # 关键：psql 的 :'pw' 变量插值只在 stdin/脚本模式生效；用 -c 传 SQL 时【不展开】，
  # 会把字面 :'pw' 原样发给服务端 → `syntax error at or near ":"`（pg16 实测），PG 改密全程失败。
  # 故必须走 printf 管道喂 stdin。-v pw=NEW + :'pw'：psql 对变量值做安全单引号转义防注入；
  # ON_ERROR_STOP 失败即非零退出。docker exec 需带 -i 才能接收 stdin。
  printf 'ALTER ROLE "%s" WITH PASSWORD :%spw%s;\n' "$pguser" "'" "'" \
    | docker exec -i "$cname" psql -U "$pguser" -d "$pgdb" -v ON_ERROR_STOP=1 -v pw="$new" \
    || die "ALTER ROLE 失败——未改 secrets.env/.env，DB 口令保持原值"
  set_secret_key POSTGRES_PASSWORD "$new"
  build_dc
  "${DC[@]}" up -d app acs worker || die "up -d app/acs/worker 失败（请人工核对并重试）"
  log "PostgreSQL：完成——app/acs/worker 已用新口令重连。"
}

# MinIO：root 口令每次启动读 env（不入卷）。改 env → up -d minio 重建读新 env → up -d 客户端。
# 注意：用旧 secret 签的预签名下载 URL 立即失效（TTL≤1h），重新生成即可。
rotate_minio() {
  local new="$1"
  set_secret_key MINIO_ROOT_PASSWORD "$new"
  build_dc
  "${DC[@]}" up -d minio          || die "up -d minio 失败"
  "${DC[@]}" up -d app acs worker || die "up -d app/acs/worker 失败"
  log "MinIO：完成——minio 重建读新口令，app/acs/worker 重连。"
  warn "用旧 secret 签发的预签名下载 URL（MML/PM 导出、备份下载，TTL≤1h）立即失效，重新生成即可。"
}

# Grafana：口令在卷 grafana.db（env 仅首次）。容器内 grafana cli reset-admin-password 改卷内口令。
rotate_grafana() {
  local new="$1" cname
  cname="${COMPOSE_PROJECT}-grafana-1"
  require_running "$cname"
  log "Grafana：容器内 reset-admin-password ..."
  docker exec "$cname" grafana cli admin reset-admin-password "$new" \
    || die "grafana reset-admin-password 失败——未改 secrets.env"
  set_secret_key GRAFANA_ADMIN_PASSWORD "$new"
  log "Grafana：完成——:3030 用新口令可登。"
}

# JWT：无状态签名密钥。改 env → up -d app。已签发 token 立即失效。
rotate_jwt() {
  local new="$1"
  set_secret_key OMCGO_JWT_SECRET "$new"
  build_dc
  "${DC[@]}" up -d app || die "up -d app 失败"
  log "JWT：完成——所有已登录用户 token 立即失效，需重新登录。"
  warn "内部 API key（.api-key omk_）是独立体系，不受 JWT 轮换影响。"
}

# TR-069 共享密钥：STUN/UDP ConnReq 的 HMAC-SHA1 共享密钥 + ConnReq 鉴权，ACS 与基站共用。
# 改 env → up -d acs app；基站侧需经 TR-069 SetParameterValues 同步，否则 ACS 主动触达失败。
rotate_shared() {
  local new="$1"
  set_secret_key OMC_SHARED_SECRET "$new"
  build_dc
  "${DC[@]}" up -d acs app || die "up -d acs/app 失败"
  log "TR-069 共享密钥：完成——acs/app 已用新密钥。"
  print_shared_warning
}

print_shared_warning() {
  cat >&2 <<'WARN'

⚠️  TR-069 共享密钥（OMC_SHARED_SECRET）已在 OMC 侧轮换，但基站侧尚未同步！
    必须经 TR-069 SetParameterValues 把新共享密钥下发到所有基站（ConnReqPassword/STUN 相关参数），
    各基站下次 Inform 时生效。建议分批灰度推送 + 监控 ConnReq 成功率。

    过渡窗口（未同步基站）：
      · ACS 用新 secret 发起 UDP CR/ConnReq，旧 secret 基站 HMAC 校验失败 → ConnReq 失败，
        ACS 无法主动触达（下发配置/重启/升级/即时 GPV/MML 全部做不了）。
      · 基站周期 Inform 不受影响（走 CPE Digest/Basic 认证）→ 设备仍在线、仍能上报；
        仅 ACS→设备的主动控制通道在该设备同步前断开，拿到新 secret 后自动恢复。
      · NAT 后设备 UDP CR 绑定 HMAC 失败 → 主动触达同样中断。
      · 无数据丢失，属控制面可达性的过渡性降级。无法零窗口（除非 ACS 支持新旧双 secret 并存，当前不支持）。
WARN
}

# ── 主流程 ─────────────────────────────────────────────────────────────────────
parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      pg|minio|grafana|jwt|shared|1|2|3|4|5) SELECTED="$1"; shift ;;
      --omc-root) OMC_ROOT="${2:?--omc-root 需要参数}"; shift 2 ;;
      -h|--help) sed -n '2,30p' "${BASH_SOURCE[0]}"; exit 0 ;;
      *) die "未知参数：$1（-h 查看用法）" ;;
    esac
  done
}

main() {
  set -euo pipefail
  parse_args "$@"
  SECRETS_FILE="$OMC_ROOT/etc/secrets.env"
  ENV_FILE="$OMC_ROOT/current/deploy/.env"
  DEPLOY_DIR="$OMC_ROOT/current/deploy"
  preflight

  local comp key
  comp="$SELECTED"
  [ -n "$comp" ] || comp="$(select_component_menu)"
  case "$comp" in
    pg|1)      key=POSTGRES_PASSWORD ;;
    minio|2)   key=MINIO_ROOT_PASSWORD ;;
    grafana|3) key=GRAFANA_ADMIN_PASSWORD ;;
    jwt|4)     key=OMCGO_JWT_SECRET ;;
    shared|5)  key=OMC_SHARED_SECRET ;;
    *) die "未选择有效组件（pg|minio|grafana|jwt|shared 或 1-5）" ;;
  esac

  acquire_value "$key"
  case "$key" in
    POSTGRES_PASSWORD)      rotate_pg "$NEW_VALUE" ;;
    MINIO_ROOT_PASSWORD)    rotate_minio "$NEW_VALUE" ;;
    GRAFANA_ADMIN_PASSWORD) rotate_grafana "$NEW_VALUE" ;;
    OMCGO_JWT_SECRET)       rotate_jwt "$NEW_VALUE" ;;
    OMC_SHARED_SECRET)      rotate_shared "$NEW_VALUE" ;;
  esac

  # 随机生成的新值在终端打印一次（运维登录 PG/MinIO/Grafana 需要）。手动输入则用户已知，不再打印。
  if [ "$WAS_RANDOM" = 1 ]; then
    {
      printf '\n=== 新口令（随机生成，仅显示一次，请妥善保存）===\n'
      printf '%s = %s\n' "$key" "$NEW_VALUE"
    } >&2
  fi
  log "完成。secrets.env（权威）↔ 组件口令 ↔ .env 已保持一致。"
}

# 仅在直接执行时跑 main；被 source（测试）时只加载函数。
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
