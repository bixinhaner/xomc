#!/usr/bin/env bash
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
RELEASE_DEPLOY="$REPO_ROOT/deployments/release/bundle/deploy"
RELEASE_COMPOSE="$RELEASE_DEPLOY/docker-compose.infra.yml"
DEV_COMPOSE="$REPO_ROOT/deployments/docker/docker-compose.yml"
INSTALL="$RELEASE_DEPLOY/install.sh"
SVC="$RELEASE_DEPLOY/svc.sh"
BUILD="$REPO_ROOT/deployments/release/build-release.sh"

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
contains() {
  local name="$1" pattern="$2" file="$3"
  if grep -Fq "$pattern" "$file"; then ok; else bad "$name: $file 未包含 [$pattern]"; fi
}

echo "── release compose 五个 bind mount ──"
contains "PostgreSQL 可配置挂载" '${POSTGRES_DATA_PATH:-pgdata}:/var/lib/postgresql/data' "$RELEASE_COMPOSE"
contains "TimescaleDB 可配置挂载" '${TSDB_DATA_PATH:-tsdbdata}:/var/lib/postgresql/data' "$RELEASE_COMPOSE"
contains "Redis 可配置挂载" '${REDIS_DATA_PATH:-redisdata}:/data' "$RELEASE_COMPOSE"
contains "NATS 可配置挂载" '${NATS_DATA_PATH:-natsdata}:/data' "$RELEASE_COMPOSE"
contains "MinIO 可配置挂载" '${MINIO_DATA_PATH:-miniodata}:/data' "$RELEASE_COMPOSE"

echo "── release .env 模板和升级继承 ──"
for key in POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH; do
  contains "$key 模板" "$key=" "$BUILD"
  contains "$key 升级继承" "$key" "$INSTALL"
done

echo "── install/svc 启动前准备路径 ──"
contains "install 加载存储库" 'storage-paths-lib.sh' "$INSTALL"
contains "install 准备目录" 'storage_prepare_configured_env_paths "$ENV_FILE"' "$INSTALL"
contains "svc 加载存储库" 'storage-paths-lib.sh' "$SVC"
contains "svc 准备目录" 'storage_prepare_configured_env_paths ".env"' "$SVC"

echo "── 开发 compose 保留命名卷默认值 ──"
contains "开发 PostgreSQL 默认命名卷" '${POSTGRES_DATA_PATH:-pgdata}:/var/lib/postgresql/data' "$DEV_COMPOSE"
contains "开发 TimescaleDB 默认命名卷" '${TSDB_DATA_PATH:-tsdbdata}:/var/lib/postgresql/data' "$DEV_COMPOSE"
contains "开发 Redis 默认命名卷" '${REDIS_DATA_PATH:-redisdata}:/data' "$DEV_COMPOSE"
contains "开发 NATS 默认命名卷" '${NATS_DATA_PATH:-natsdata}:/data' "$DEV_COMPOSE"
contains "开发 MinIO 默认命名卷" '${MINIO_DATA_PATH:-miniodata}:/data' "$DEV_COMPOSE"

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]
