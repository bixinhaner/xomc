#!/usr/bin/env bash
# =============================================================================
# secrets-lib.sh —— 部署凭证治本（#175）的纯函数库
#
# 由 install.sh `source`，并由 secrets-lib_test.sh 单测。这里只放**无副作用 / 可测**的
# 凭证生成 / 导入 / 合并工具；编排（docker 卷探测、日志、落盘时机）在 install.sh::ensure_secrets。
#
# 设计目标：密钥不再发默认值，首次安装自动生成强随机并存到 etc/secrets.env（uninstall 保留、
# --purge 删），secrets.env 作为 6 个密钥键的**唯一权威源**，根除「默认口令」与「.env.saved
# 快照继承被默认值污染」两个老坑。值任何时候**不打印**。
# =============================================================================

# 由 secrets.env 权威管理的密钥键。POSTGRES_TSDB_PASSWORD（时序库口令，#347）独立于主库
# POSTGRES_PASSWORD —— 跨版本新增键，旧 secrets.env 缺它时由 secrets_ensure_keys 幂等补齐。
SECRET_KEYS="POSTGRES_PASSWORD POSTGRES_TSDB_PASSWORD MINIO_ROOT_USER MINIO_ROOT_PASSWORD OMCGO_JWT_SECRET OMC_SHARED_SECRET GRAFANA_ADMIN_PASSWORD"

# rand_hex <字节数> —— 生成 hex 随机串。优先 openssl，回退 /dev/urandom（不依赖 openssl 存在）。
rand_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$1"
  else
    head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n'
  fi
}

# secrets_get_val <key> <env_file> —— 取 `key=value` 的 value（文件不存在 / 无该键 → 空）。
secrets_get_val() {
  [ -f "$2" ] || return 0
  awk -F= -v k="$1" '$1==k{sub(/^[^=]*=/,"");print;exit}' "$2" 2>/dev/null
}

# secrets_bind_data_exists <env_file> —— 检测 plan-resources.sh 配置的 bind-mount
# PostgreSQL 数据目录是否已经初始化。Docker volume inspect 无法覆盖这种部署方式。
secrets_bind_data_exists() {
  local env_file="$1" key path
  for key in POSTGRES_DATA_PATH TSDB_DATA_PATH; do
    path="$(secrets_get_val "$key" "$env_file")"
    [ -f "$path/PG_VERSION" ] && return 0
  done
  return 1
}

# secrets_is_default_value <key> <value> —— value 是否为已知默认 / 占位 / 空。
# 用于「是否需新生成 / 该 .env 能否当迁移源」判定。返回 0=是默认（不可信），1=非默认（可信）。
secrets_is_default_value() {
  case "$2" in ""|REPLACE_ME*) return 0 ;; esac
  case "$1=$2" in
    "POSTGRES_PASSWORD=omcgo123") return 0 ;;
    "MINIO_ROOT_USER=minioadmin") return 0 ;;
    "MINIO_ROOT_PASSWORD=minioadmin") return 0 ;;
    "OMC_SHARED_SECRET=dps") return 0 ;;
    "GRAFANA_ADMIN_PASSWORD=admin") return 0 ;;
    "OMCGO_JWT_SECRET=8f7a9b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6") return 0 ;;
    "POSTGRES_TSDB_PASSWORD=omcgo123") return 0 ;;
  esac
  return 1
}

# secrets_generate_to <file> —— 生成强随机 6 键写入 file（umask 077 → 600）。**全新部署用**。值不打印。
secrets_generate_to() {
  ( umask 077; {
      echo "POSTGRES_PASSWORD=$(rand_hex 24)"
      echo "POSTGRES_TSDB_PASSWORD=$(rand_hex 24)"   # 时序库口令，独立于主库（#347）
      echo "MINIO_ROOT_USER=omcadmin"
      echo "MINIO_ROOT_PASSWORD=$(rand_hex 24)"
      echo "OMCGO_JWT_SECRET=$(rand_hex 32)"
      echo "OMC_SHARED_SECRET=$(rand_hex 32)"
      echo "GRAFANA_ADMIN_PASSWORD=$(rand_hex 16)"
    } > "$1" )
}

# secrets_ensure_keys <file> —— 保证 SECRET_KEYS 每个键在 file 中都存在且非空（幂等）。
# 跨版本【新增密钥键】（如 #347 的 POSTGRES_TSDB_PASSWORD）时，复用旧 secrets.env 或从旧 .env
# 导入都可能缺该键 / 写成空值；这里补齐：缺失或空值 → 生成强随机（MINIO_ROOT_USER 用固定
# omcadmin）。**已有非空值绝不改动**（保护现行口令）。umask 077 → 600。
secrets_ensure_keys() {
  local file="$1" k v
  [ -f "$file" ] || return 0
  ( umask 077
    for k in $SECRET_KEYS; do
      v="$(secrets_get_val "$k" "$file")"
      [ -n "$v" ] && continue
      # 删掉可能存在的空值行（key= 无值），避免与即将追加的 key=val 并存
      if grep -q "^${k}=" "$file" 2>/dev/null; then
        grep -v "^${k}=" "$file" > "${file}.tmp" 2>/dev/null && mv "${file}.tmp" "$file"
      fi
      case "$k" in
        MINIO_ROOT_USER) printf '%s=omcadmin\n' "$k" >> "$file" ;;
        *)               printf '%s=%s\n' "$k" "$(rand_hex 24)" >> "$file" ;;
      esac
    done )
}

# secrets_import_to <src_env> <file> —— 从现行有效凭证 src 抽 6 键写入 file（**存量机迁移用**，
# 绝不新生成，保证与已存在数据卷口令一致）。umask 077 → 600。
secrets_import_to() {
  local src="$1" out="$2" k
  ( umask 077; : > "$out"
    for k in $SECRET_KEYS; do
      printf '%s=%s\n' "$k" "$(secrets_get_val "$k" "$src")" >> "$out"
    done )
}

# secrets_apply_to_env <secrets_file> <target_env> —— 用 secrets 的 6 键覆盖 target_env（secrets
# 为权威源）。awk 把值当**数据**处理、不经 shell 展开（含 = / 特殊字符的口令安全）；保留 target
# 其它行与原顺序；secrets 有而 target 没有的键追加到末尾。成功 0 / 失败 1。
secrets_apply_to_env() {
  local secrets="$1" target="$2" tmp
  [ -f "$secrets" ] && [ -f "$target" ] || return 1
  tmp="$(mktemp)" || return 1
  if awk -v keys="$SECRET_KEYS" '
      BEGIN { n=split(keys,A," "); for(i=1;i<=n;i++) want[A[i]]=1 }
      FNR==NR {
        if ($0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
          p=index($0,"="); k=substr($0,1,p-1)
          if (k in want) { val[k]=substr($0,p+1); have[k]=1 }
        }
        next
      }
      {
        if ($0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
          p=index($0,"="); k=substr($0,1,p-1)
          if ((k in want) && (k in have)) { print k"="val[k]; seen[k]=1; next }
        }
        print
      }
      END { for (k in have) if (!(k in seen)) print k"="val[k] }
    ' "$secrets" "$target" > "$tmp"; then
    cat "$tmp" > "$target"   # 覆写内容保留 target 原 inode / 权限
    rm -f "$tmp"
    return 0
  fi
  rm -f "$tmp"
  return 1
}

# ensure_secrets —— 编排（#175）：在 source .env / 起 infra 之前确保 6 个密钥就位。
# 依赖 sourcer 提供：log() warn() docker  以及变量 OMC_ROOT、COMPOSE_PROJECT。
# etc/secrets.env 是密钥唯一权威源：① 已存在→复用（幂等，绝不重生成）；② 存量机（数据卷已存在
# 或现行 .env/.env.saved 有非默认 PG 口令）→从现行凭证导入（绝不新生成，保证与旧卷口令一致）；
# ③ 全新部署→生成强随机。最后用 secrets.env 覆盖 current/deploy/.env 的 6 键。值任何时候不打印。
ensure_secrets() {
  local SECRETS_FILE="$OMC_ROOT/etc/secrets.env"
  local target_env="$OMC_ROOT/current/deploy/.env"
  mkdir -p "$OMC_ROOT/etc"
  [ -f "$target_env" ] || { warn "secrets：$target_env 不存在，跳过凭证就位" "Secrets: $target_env does not exist; skipping credential setup"; return 0; }

  if [ -f "$SECRETS_FILE" ]; then
    log "secrets：复用已存在 ${SECRETS_FILE}（幂等，绝不重生成）" "Secrets: reusing existing ${SECRETS_FILE} (idempotent; never regenerated)"
  else
    local have_vol=0 have_bind_data=0 src="" f pw
    docker volume inspect "${COMPOSE_PROJECT}_pgdata"    >/dev/null 2>&1 && have_vol=1
    docker volume inspect "${COMPOSE_PROJECT}_miniodata" >/dev/null 2>&1 && have_vol=1
    if secrets_bind_data_exists "$target_env"; then
      have_bind_data=1
      have_vol=1
    fi
    # 现行有效凭证源：current .env 优先，退 etc/.env.saved；取「PG 口令非默认」者
    for f in "$target_env" "$OMC_ROOT/etc/.env.saved"; do
      [ -f "$f" ] || continue
      pw="$(secrets_get_val POSTGRES_PASSWORD "$f")"
      if ! secrets_is_default_value POSTGRES_PASSWORD "$pw"; then src="$f"; break; fi
    done
    if [ "$have_bind_data" = 1 ] && [ -z "$src" ]; then
      warn "secrets：检测到已初始化的 PostgreSQL bind-mount 数据目录，但没有可继承的非默认口令" "Secrets: initialized PostgreSQL bind-mount data found, but no non-default credential can be inherited"
      warn "secrets：为避免新口令与现有数据不一致，已停止部署；请恢复旧 .env/secrets.env，或确认测试数据后清空数据目录再重试" "Secrets: deployment stopped to avoid a credential mismatch; restore the old .env/secrets.env, or clear the data directory after confirming it is test data"
      return 1
    fi
    if [ "$have_vol" = 1 ] || [ -n "$src" ]; then
      [ -n "$src" ] || src="$target_env"   # 卷在但没找到非默认源：用现值（绝不瞎生成，否则连不上旧卷）
      log "secrets：检测到存量数据卷/现行凭证，从现行凭证导入 → ${SECRETS_FILE}（不新生成，保证与旧卷一致）" "Secrets: existing data or credentials found; importing current credentials into ${SECRETS_FILE} (not generating new credentials)"
      secrets_import_to "$src" "$SECRETS_FILE"
    else
      log "secrets：首次部署，生成强随机凭证 → ${SECRETS_FILE}（PG/MinIO/Grafana 登录所需，请妥善备份；值不打印）" "Secrets: fresh install; generating strong random credentials -> ${SECRETS_FILE} (required for PG/MinIO/Grafana login; values are never printed)"
      secrets_generate_to "$SECRETS_FILE"
    fi
  fi
  # 跨版本新增密钥键补齐（#347 POSTGRES_TSDB_PASSWORD）：复用旧 secrets.env / 从旧凭证导入
  # 都可能缺新键；幂等补强随机（已有值不动），保证 .env 不会拿到空口令 / REPLACE_ME。
  secrets_ensure_keys "$SECRETS_FILE"
  chmod 600 "$SECRETS_FILE" 2>/dev/null || true

  # secrets.env 为密钥键权威源 → 覆盖进 current/deploy/.env（取代 .env.saved 对密钥的脆弱继承）
  secrets_apply_to_env "$SECRETS_FILE" "$target_env" \
    || warn "secrets：覆盖 $target_env 失败，请人工核对其 6 个密钥键" "Secrets: failed to apply credentials to $target_env; verify its six secret keys"
  # .env.saved 同步刷新（供 uninstall→reinstall 一致；但密钥真权威是 secrets.env）
  cp -f "$target_env" "$OMC_ROOT/etc/.env.saved" 2>/dev/null || true
}
