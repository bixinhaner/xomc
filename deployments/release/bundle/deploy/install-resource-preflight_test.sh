#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

write_complete_env() {
  cat >"$1" <<'EOF'
APP_CPUS=2
APP_MEM=2048m
APP_GOMEMLIMIT=1800MiB
APP_GOMAXPROCS=2
ACS_CPUS=2
ACS_MEM=2048m
ACS_GOMEMLIMIT=1800MiB
ACS_GOMAXPROCS=2
WORKER_CPUS=4
WORKER_MEM=4096m
WORKER_GOMEMLIMIT=3600MiB
WORKER_GOMAXPROCS=4
POSTGRES_CPUS=4
POSTGRES_MEM=4096m
PG_SHARED_BUFFERS=1024MB
PG_EFFECTIVE_CACHE_SIZE=3072MB
PG_MAX_CONNECTIONS=180
PG_WORK_MEM=16MB
PG_MAINTENANCE_WORK_MEM=256MB
PG_MAX_WAL_SIZE=2048MB
TSDB_CPUS=4
TSDB_MEM=4096m
TSDB_SHARED_BUFFERS=1024MB
TSDB_EFFECTIVE_CACHE_SIZE=3072MB
TSDB_MAX_CONNECTIONS=180
TSDB_WORK_MEM=16MB
TSDB_MAINTENANCE_WORK_MEM=256MB
TSDB_MAX_WAL_SIZE=2048MB
REDIS_CORE_CPUS=2
REDIS_CORE_MEM=4g
REDIS_CORE_MAXMEMORY=3gb
REDIS_PM_CPUS=2
REDIS_PM_MEM=8g
REDIS_PM_MAXMEMORY=6gb
NATS_CPUS=1
NATS_MEM=512m
NATS_MAX_MEMORY_STORE=134217728
MINIO_CPUS=1
MINIO_MEM=1024m
WEB_CPUS=1
WEB_MEM=512m
OMC_RESOURCE_SCHEMA_VERSION=3
OMC_RESOURCE_PLAN_HOST_CPU=32
OMC_RESOURCE_PLAN_HOST_MEM_MIB=32768
EOF
}

write_legacy_env() {
  cat >"$1" <<'EOF'
REDIS_CPUS=2
REDIS_MEM=5g
REDIS_MAXMEMORY=4gb
EOF
}

prepare_package() {
  local pkg="$1"
  mkdir -p "$pkg/deploy" "$pkg/etc" "$pkg/images"
  cp -a "$SCRIPT_DIR/." "$pkg/deploy/"
  printf 'OMC_PUBLIC_HOST=10.0.0.1\n' >"$pkg/deploy/.env"
  : >"$pkg/deploy/docker-compose.infra.yml"
  : >"$pkg/deploy/docker-compose.app.yml"
  cat >"$pkg/etc/worker.prod.yaml" <<'EOF'
tsdb:
  max_conns: 128
EOF
  printf 'project_version=preflight-test\n' >"$pkg/VERSION"
}

prepare_stubs() {
  local bin="$1"
  mkdir -p "$bin"
  cat >"$bin/id" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = "-u" ] && { echo 0; exit 0; }
exec /usr/bin/id "$@"
EOF
  cat >"$bin/docker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${DOCKER_LOG:?}"
case "${1:-} ${2:-}" in
  "compose version") exit 0 ;;
  "image inspect")
    [ -n "${MISSING_IMAGE:-}" ] && [ "${3:-}" = "$MISSING_IMAGE" ] && exit 1
    exit 0
    ;;
  "info ") exit 0 ;;
esac
exit 0
EOF
  cat >"$bin/systemctl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${SYSTEMCTL_LOG:?}"
exit 1
EOF
  cat >"$bin/sha256sum" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$bin/id" "$bin/docker" "$bin/systemctl" "$bin/sha256sum"
}

run_install() {
  local pkg="$1" root="$2" bin="$3" output="$4"
  shift 4
  env \
    PATH="$bin:$PATH" \
    DOCKER_LOG="$TMP/docker.log" \
    SYSTEMCTL_LOG="$TMP/systemctl.log" \
    REAL_CP="${REAL_CP:-}" \
    bash "$pkg/deploy/install.sh" \
      --skip-infra --skip-monitoring --skip-web --yes \
      --omc-root "$root" "$@" >"$output" 2>&1
}

echo "── check-only 校验当前或 saved 资源候选 ──"
PKG_CHECK="$TMP/pkg-check"
ROOT_CHECK="$TMP/root-check"
BIN_CHECK="$TMP/bin-check"
prepare_package "$PKG_CHECK"
prepare_stubs "$BIN_CHECK"
mkdir -p "$ROOT_CHECK/releases/old/deploy" "$ROOT_CHECK/etc"
ln -s "$ROOT_CHECK/releases/old" "$ROOT_CHECK/current"
write_legacy_env "$ROOT_CHECK/releases/old/deploy/resources.env"
: >"$TMP/docker.log"
: >"$TMP/systemctl.log"
before_link="$(readlink "$ROOT_CHECK/current")"
if run_install "$PKG_CHECK" "$ROOT_CHECK" "$BIN_CHECK" "$TMP/check-legacy.out" --check-only; then
  bad "check-only 不得接受仅三行 Redis 的 current resources.env"
elif grep -Fq 'resources.env' "$TMP/check-legacy.out"; then
  ok
else
  bad "check-only 应明确报告 resources.env 校验失败: $(cat "$TMP/check-legacy.out")"
fi
[ "$(readlink "$ROOT_CHECK/current")" = "$before_link" ] && ok || bad "check-only 失败不得切换 current"
if grep -Eq '(^| )(restart|up)( |$)' "$TMP/docker.log"; then
  bad "check-only 失败不得重启或启动容器"
else
  ok
fi

: >"$TMP/docker.log"
if run_install "$PKG_CHECK" "$ROOT_CHECK" "$BIN_CHECK" "$TMP/install-legacy.out"; then
  bad "正常安装不得接受仅三行 Redis 的 current resources.env"
elif grep -Fq 'resources.env' "$TMP/install-legacy.out"; then
  ok
else
  bad "正常安装预检应明确报告 resources.env 校验失败"
fi
[ "$(readlink "$ROOT_CHECK/current")" = "$before_link" ] && ok || bad "三行旧资源文件不得触发 current 切换"
if grep -Eq '(^| )(restart|up)( |$)' "$TMP/docker.log"; then
  bad "三行旧资源文件不得触发容器重启或启动"
else
  ok
fi

rm -f "$ROOT_CHECK/releases/old/deploy/resources.env"
write_complete_env "$ROOT_CHECK/etc/resources.env.saved"
if run_install "$PKG_CHECK" "$ROOT_CHECK" "$BIN_CHECK" "$TMP/check-saved.out" --check-only; then
  ok
else
  bad "check-only 应接受完整的 resources.env.saved 候选: $(cat "$TMP/check-saved.out")"
fi

rm -f "$ROOT_CHECK/etc/resources.env.saved"
if run_install "$PKG_CHECK" "$ROOT_CHECK" "$BIN_CHECK" "$TMP/check-missing.out" --check-only; then
  bad "check-only 不得在 resources.env 缺失时静默回退默认值"
elif grep -Fq 'resources.env' "$TMP/check-missing.out"; then
  ok
else
  bad "缺失资源契约时应给出重新规划提示"
fi

echo "── 正常安装在复制后复验，失败不切软链、不重启 ──"
PKG_COPY="$TMP/pkg-copy"
ROOT_COPY="$TMP/root-copy"
BIN_COPY="$TMP/bin-copy"
prepare_package "$PKG_COPY"
prepare_stubs "$BIN_COPY"
mkdir -p "$ROOT_COPY/releases/old/deploy" "$ROOT_COPY/etc"
ln -s "$ROOT_COPY/releases/old" "$ROOT_COPY/current"
printf 'OMC_PUBLIC_HOST=10.0.0.1\n' >"$ROOT_COPY/releases/old/deploy/.env"
write_complete_env "$ROOT_COPY/releases/old/deploy/resources.env"
REAL_CP="$(command -v cp)"
cat >"$BIN_COPY/cp" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  */current/deploy/resources.env)
    destination="${@: -1}"
    printf 'REDIS_CPUS=2\nREDIS_MEM=5g\nREDIS_MAXMEMORY=4gb\n' >"$destination"
    exit 0
    ;;
esac
exec "${REAL_CP:?}" "$@"
EOF
chmod +x "$BIN_COPY/cp"
: >"$TMP/docker.log"
: >"$TMP/systemctl.log"
before_link="$(readlink "$ROOT_COPY/current")"
if REAL_CP="$REAL_CP" run_install "$PKG_COPY" "$ROOT_COPY" "$BIN_COPY" "$TMP/copy-invalid.out"; then
  bad "继承复制后被破坏的 resources.env 必须终止安装"
elif grep -Fq '复制/继承后的 resources.env' "$TMP/copy-invalid.out" ||
  grep -Fq 'Copied/inherited resources.env failed complete resource-plan validation' "$TMP/copy-invalid.out"; then
  ok
else
  bad "复制后复验失败应明确指出候选已损坏: $(tail -5 "$TMP/copy-invalid.out")"
fi
[ "$(readlink "$ROOT_COPY/current")" = "$before_link" ] && ok || bad "复制后复验失败不得切换 current"
if grep -Eq '(^| )(restart|up)( |$)' "$TMP/docker.log"; then
  bad "资源复验失败不得重启或启动容器"
else
  ok
fi

echo "── skip-infra 在切换 current 前校验基础设施/监控镜像 ──"
PKG_IMAGE="$TMP/pkg-image"
ROOT_IMAGE="$TMP/root-image"
BIN_IMAGE="$TMP/bin-image"
prepare_package "$PKG_IMAGE"
prepare_stubs "$BIN_IMAGE"
: >"$PKG_IMAGE/deploy/docker-compose.monitoring.yml"
cat >>"$PKG_IMAGE/deploy/.env" <<'EOF'
IMAGE_POSTGRES=postgres:test
IMAGE_POSTGRES_TSDB=timescaledb:test
IMAGE_REDIS=redis:test
IMAGE_NATS=nats:test
IMAGE_MINIO=minio:test
IMAGE_PROMETHEUS=prometheus:test
IMAGE_ALERTMANAGER=alertmanager:test
IMAGE_GRAFANA=grafana:test
IMAGE_LOKI=loki:test
IMAGE_TEMPO=tempo:test
IMAGE_OTELCOL=otelcol:test
IMAGE_NATS_EXPORTER=nats-exporter:test
IMAGE_NGINX_EXPORTER=nginx-exporter:test
IMAGE_NODE_EXPORTER=node-exporter:test
IMAGE_CADVISOR=mirror/cadvisor:v1
EOF
mkdir -p "$ROOT_IMAGE/releases/old/deploy" "$ROOT_IMAGE/etc"
ln -s "$ROOT_IMAGE/releases/old" "$ROOT_IMAGE/current"
write_complete_env "$ROOT_IMAGE/releases/old/deploy/resources.env"
: >"$TMP/docker.log"
: >"$TMP/systemctl.log"
before_link="$(readlink "$ROOT_IMAGE/current")"
if env \
  PATH="$BIN_IMAGE:$PATH" \
  DOCKER_LOG="$TMP/docker.log" \
  SYSTEMCTL_LOG="$TMP/systemctl.log" \
  MISSING_IMAGE="mirror/cadvisor:v1" \
  bash "$PKG_IMAGE/deploy/install.sh" \
    --skip-infra --skip-web --yes --check-only \
    --omc-root "$ROOT_IMAGE" >"$TMP/image-missing.out" 2>&1; then
  bad "skip-infra check-only 不得接受缺失的基础设施/监控镜像"
elif grep -Fq 'mirror/cadvisor:v1' "$TMP/image-missing.out"; then
  ok
else
  bad "缺失镜像预检应明确列出镜像名: $(tail -5 "$TMP/image-missing.out")"
fi
[ "$(readlink "$ROOT_IMAGE/current")" = "$before_link" ] && ok || bad "缺失镜像预检失败不得切换 current"
if grep -Eq '(^| )(restart|up)( |$)' "$TMP/docker.log"; then
  bad "缺失镜像预检失败不得重启或启动容器"
else
  ok
fi

echo "── svc 缺失资源契约时禁止 restart ──"
SVC_DIR="$TMP/svc"
mkdir -p "$SVC_DIR"
for file in svc.sh storage-paths-lib.sh resource-env-lib.sh resource-plan-metrics.sh monitoring-profile-lib.sh gpv-handoff-lib.sh; do
  cp "$SCRIPT_DIR/$file" "$SVC_DIR/$file"
done
: >"$SVC_DIR/docker-compose.app.yml"
cat >"$SVC_DIR/.env" <<EOF
OMCGO_SKIP_MONITORING=1
POSTGRES_DATA_PATH=$TMP/data/postgres
TSDB_DATA_PATH=$TMP/data/timescaledb
REDIS_DATA_PATH=$TMP/data/redis
REDIS_PM_DATA_PATH=$TMP/data/redis-pm
NATS_DATA_PATH=$TMP/data/nats
MINIO_DATA_PATH=$TMP/data/minio
EOF
: >"$TMP/docker.log"
if env PATH="$BIN_CHECK:$PATH" DOCKER_LOG="$TMP/docker.log" SYSTEMCTL_LOG="$TMP/systemctl.log" \
  bash "$SVC_DIR/svc.sh" restart >"$TMP/svc-missing.out" 2>&1; then
  bad "svc restart 不得在 resources.env 缺失时使用 Compose 默认值"
elif grep -Fq 'resources.env' "$TMP/svc-missing.out"; then
  ok
else
  bad "svc 缺失资源契约时应给出重新规划提示"
fi
if grep -Eq '(^| )restart( |$)' "$TMP/docker.log"; then
  bad "svc 资源预检失败不得调用 compose restart"
else
  ok
fi

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]
