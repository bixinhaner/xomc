#!/usr/bin/env bash
# =============================================================================
# smoke_retention.sh — 资源保留与上传背压冒烟（#318-321）
#
# 覆盖四个后端特性，核心保证「配置全部落 sys_configs、前端系统配置页可见可改」：
#   #318 ACS PM 上传背压 watchdog（磁盘%+CPU 双阈迟滞，停收/恢复）
#   #319 MinIO 原始件 ILM 保留期可配（默认 60 天）
#   #320 基站日志按时间保留（默认 60 天）+ 文件数配额可配并存
#   #836 入库后原始 XML 一次性压缩回写 MinIO 省盘
#
# 分层：
#   A. 系统配置可见可改（纯 app HTTP API，任何部署形态都跑）—— 头号验收项
#   B. #318 背压：ACS 指标暴露 + 功能 engage/release（容器栈，可达才跑）
#   C. #319 ILM：pm-files/mr-files 桶 60 天过期规则（MinIO mc，容器可用才跑）
#   D. #836 一次性压缩：worker raw_archive 指标暴露（worker 指标口，可达才跑）
#   E. #320 基站日志保留：时间保留 + 文件数配额配置并存（纯 app HTTP API）
#
# 容器栈额外端点（可用环境变量覆盖）：
#   OMC_ACS_URL（默认 http://localhost:7557）          ACS 上传/连接口
#   OMC_ACS_METRICS_URL（默认 http://localhost:9095）   ACS Prometheus 指标
#   OMC_WORKER_METRICS_URL（默认 http://localhost:9092）worker Prometheus 指标
#   OMC_MINIO_CONTAINER（默认 omc-minio-1）             MinIO 容器名（mc 查 ILM）
#   OMC_MINIO_USER/OMC_MINIO_PASS（默认 minioadmin）    MinIO 根凭据
#   SMOKE_BP_FUNCTIONAL（默认 1）                        是否跑背压 engage/release 功能用例
#
# 危险操作禁区：本脚本只临时改 acs.backpressure 阈值做功能验证，结束（含异常退出）
# 必恢复原值（EXIT trap）；不触碰真实基站，上传的 PM 文件名带 SMOKE_TAG 便于识别。
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "保留与上传背压(#318-321)" "$@"
smoke_login

ACS_UPLOAD_URL="${OMC_ACS_URL:-http://localhost:7557}"
ACS_METRICS_URL="${OMC_ACS_METRICS_URL:-http://localhost:9095}"
WORKER_METRICS_URL="${OMC_WORKER_METRICS_URL:-http://localhost:9092}"
MINIO_CONTAINER="${OMC_MINIO_CONTAINER:-omc-minio-1}"
MINIO_USER="${OMC_MINIO_USER:-minioadmin}"
MINIO_PASS="${OMC_MINIO_PASS:-minioadmin}"
BP_FUNCTIONAL="${SMOKE_BP_FUNCTIONAL:-1}"

# ---------------------------------------------------------------------------
# 辅助：从 sysConfig API 按 (category,key) 取值（设置全局 BODY）
# ---------------------------------------------------------------------------
cfg_get_value() {
    local category="$1" key="$2"
    req GET "/api/v1/admin/sysConfig?category=${category}"
    printf '%s' "$BODY" | KEY="$key" python3 -c "
import sys, json, os
try:
    d = json.load(sys.stdin)
except Exception:
    print(''); sys.exit(0)
arr = d.get('data') or []
k = os.environ['KEY']
for it in arr:
    if isinstance(it, dict) and it.get('key') == k:
        print(it.get('value', '')); break
" 2>/dev/null
}

# check_cfg_key CATEGORY KEY [期望值] —— 断言该键在系统配置中可见（可选校验值）
check_cfg_key() {
    local category="$1" key="$2" expect="${3:-}" v
    v=$(cfg_get_value "$category" "$key")
    if [ -z "$v" ]; then
        fail "系统配置可见 ${category}.${key}" "未在 sysConfig 中找到该键（配置未落 sys_configs）"
    elif [ -n "$expect" ] && [ "$v" != "$expect" ]; then
        fail "系统配置可见 ${category}.${key}" "值=${v} 期望=${expect}"
    else
        pass "系统配置可见 ${category}.${key} (=${v})"
    fi
}

# bp_batch ITEMS_JSON —— 批量 upsert acs.backpressure 一组键（value_type 必带）
bp_batch() {
    req POST "/api/v1/admin/sysConfig/batch" \
        "{\"category\":\"acs.backpressure\",\"items\":[$1]}"
}

# 读 ACS/worker 指标值（gauge/counter 行第二列）；不可达回空
metric_value() {
    local url="$1" name="$2"
    curl -s --max-time 5 "$url/metrics" 2>/dev/null \
        | awk -v n="$name" '$1==n {print $2; exit}'
}
metric_has() {
    local url="$1" pat="$2"
    curl -s --max-time 5 "$url/metrics" 2>/dev/null | grep -qE "$pat"
}
endpoint_up() { curl -s --max-time 3 -o /dev/null "$1/metrics" 2>/dev/null; }

# wait_bp_active EXPECT [TIMEOUT_SEC] —— 轮询背压指标直到 active==EXPECT 或超时。
# 背压 watchdog 每个采样周期才刷新一次配置（首次把 interval 30→1 要等满旧周期），故用
# 轮询而非固定 sleep：状态一翻立即返回，最坏只等到 watchdog 下一拍。返回 0=已达期望。
wait_bp_active() {
    local expect="$1" timeout="${2:-35}" waited=0 av
    while [ "$waited" -lt "$timeout" ]; do
        av=$(metric_value "$ACS_METRICS_URL" "acs_pm_upload_backpressure_active")
        [ "$av" = "$expect" ] && return 0
        sleep 2; waited=$((waited + 2))
    done
    return 1
}

# 上传 PM 文件到 ACS，回显 HTTP 状态码
acs_upload_pm() {
    local fname="$1" body="$2"
    curl -s --max-time 8 -o /dev/null -w '%{http_code}' \
        -X POST "$ACS_UPLOAD_URL/smallcell/FileUploadService?fileType=PM&filename=${fname}" \
        --data-binary "$body" 2>/dev/null || echo "000"
}

# ---------------------------------------------------------------------------
# 背压配置原值保存 + 异常恢复（EXIT trap 与 lib 的 tmpdir 清理串联）
# ---------------------------------------------------------------------------
BP_SAVED=0
BP_ORIG_ENABLED=""; BP_ORIG_HIGH=""; BP_ORIG_LOW=""; BP_ORIG_INTERVAL=""
BP_ORIG_CPUHIGH=""; BP_ORIG_CPULOW=""
restore_backpressure() {
    [ "$BP_SAVED" = "1" ] || return 0
    BP_SAVED=0
    bp_batch "{\"key\":\"enabled\",\"value\":\"${BP_ORIG_ENABLED:-true}\",\"value_type\":\"bool\"},
              {\"key\":\"disk_high_pct\",\"value\":\"${BP_ORIG_HIGH:-85}\",\"value_type\":\"int\"},
              {\"key\":\"disk_low_pct\",\"value\":\"${BP_ORIG_LOW:-75}\",\"value_type\":\"int\"},
              {\"key\":\"cpu_high_per_core\",\"value\":\"${BP_ORIG_CPUHIGH:-0.9}\",\"value_type\":\"float\"},
              {\"key\":\"cpu_low_per_core\",\"value\":\"${BP_ORIG_CPULOW:-0.7}\",\"value_type\":\"float\"},
              {\"key\":\"check_interval_sec\",\"value\":\"${BP_ORIG_INTERVAL:-30}\",\"value_type\":\"int\"}" \
        >/dev/null 2>&1
}
trap 'restore_backpressure; rm -rf "$SMOKE_TMPDIR"' EXIT

# ===========================================================================
section "A. 系统配置可见可改（配置全部落 sys_configs，前端系统配置页可改）"
# ===========================================================================
req GET "/api/v1/admin/sysConfig"
check_ret_ok "系统配置全量列表可查"

# #318 背压 6 键
check_cfg_key "acs.backpressure" "enabled"
check_cfg_key "acs.backpressure" "disk_high_pct"
check_cfg_key "acs.backpressure" "disk_low_pct"
check_cfg_key "acs.backpressure" "cpu_high_per_core"
check_cfg_key "acs.backpressure" "cpu_low_per_core"
check_cfg_key "acs.backpressure" "check_interval_sec"
# #319 ILM 保留天数
check_cfg_key "minio.retention" "raw_object_days"
# #836 入库后一次性压缩开关
check_cfg_key "raw_archive" "compress_after_ingest"
# #320 基站日志保留（时间 + 文件数配额并存）
check_cfg_key "stationlog.retention" "max_retention_days"
check_cfg_key "stationlog.retention" "max_file_count"

section "A2. 系统配置可改（batch 写值→热加载不报错）"
# 用 batch 把 minio.retention.raw_object_days 改成同值（幂等，验证写路径 + 热加载 hook 不炸）
req POST "/api/v1/admin/sysConfig/batch" \
    "{\"category\":\"minio.retention\",\"items\":[{\"key\":\"raw_object_days\",\"value\":\"60\",\"value_type\":\"int\"}]}"
check_ret_ok "系统配置 batch upsert（minio.retention.raw_object_days=60）可写"

# ===========================================================================
section "B. #318 PM 上传背压（ACS 指标 + engage/release 功能）"
# ===========================================================================
if ! endpoint_up "$ACS_METRICS_URL"; then
    skip "ACS 指标不可达（$ACS_METRICS_URL）" "非容器栈或端口未映射，跳过背压功能用例"
else
    # 指标暴露存在性
    if metric_has "$ACS_METRICS_URL" '^acs_pm_upload_backpressure_active'; then
        pass "#318 背压指标暴露 acs_pm_upload_backpressure_active"
    else
        fail "#318 背压指标暴露" "未在 ACS 指标中找到 acs_pm_upload_backpressure_active"
    fi
    disk_pct=$(metric_value "$ACS_METRICS_URL" "acs_data_disk_usage_percent")
    if [ -n "$disk_pct" ]; then
        pass "#318 数据盘使用率指标可读 acs_data_disk_usage_percent=${disk_pct}"
    else
        fail "#318 数据盘使用率指标" "acs_data_disk_usage_percent 缺失或为空"
    fi

    # 稳态（默认高阈）下背压应为 0、PM 上传应放行
    av0=$(metric_value "$ACS_METRICS_URL" "acs_pm_upload_backpressure_active")
    if [ "$av0" = "0" ]; then pass "#318 稳态背压未触发 active=0"; else fail "#318 稳态 active=0" "实际=${av0}"; fi

    if [ "$BP_FUNCTIONAL" != "1" ]; then
        skip "#318 背压 engage/release 功能用例" "SMOKE_BP_FUNCTIONAL!=1"
    elif [ "$(acs_upload_pm "pmwarm-${SMOKE_TAG}.xml" "warmup")" = "000" ]; then
        skip "#318 背压 engage/release 功能用例" "ACS 上传口 $ACS_UPLOAD_URL 不可达"
    else
        # 保存原值
        BP_ORIG_ENABLED=$(cfg_get_value acs.backpressure enabled)
        BP_ORIG_HIGH=$(cfg_get_value acs.backpressure disk_high_pct)
        BP_ORIG_LOW=$(cfg_get_value acs.backpressure disk_low_pct)
        BP_ORIG_CPUHIGH=$(cfg_get_value acs.backpressure cpu_high_per_core)
        BP_ORIG_CPULOW=$(cfg_get_value acs.backpressure cpu_low_per_core)
        BP_ORIG_INTERVAL=$(cfg_get_value acs.backpressure check_interval_sec)
        BP_SAVED=1

        # ---- engage：阈值压到 1%（实测盘 ~28% > 1% 必触发）、采样 1s 快刷 ----
        # 同时把 CPU 阈值架高（high=100/low=99）中和负载信号：迟滞解除要求「全部信号回落」，
        # 本机宿主负载/核 ~0.6 紧贴默认低阈 0.7、繁忙时会越过 → 解除变不确定。
        # 中和 CPU 后 engage/release 完全由我可控的磁盘阈值驱动，断言才确定性。
        bp_batch "{\"key\":\"enabled\",\"value\":\"true\",\"value_type\":\"bool\"},
                  {\"key\":\"check_interval_sec\",\"value\":\"1\",\"value_type\":\"int\"},
                  {\"key\":\"cpu_high_per_core\",\"value\":\"100\",\"value_type\":\"float\"},
                  {\"key\":\"cpu_low_per_core\",\"value\":\"99\",\"value_type\":\"float\"},
                  {\"key\":\"disk_high_pct\",\"value\":\"1\",\"value_type\":\"int\"},
                  {\"key\":\"disk_low_pct\",\"value\":\"0\",\"value_type\":\"int\"}"
        check_ret_ok "背压配置下发（engage：disk_high=1%，CPU 信号中和）"
        # 轮询到 active=1（首拍要等满旧采样周期，故超时给到 35s）
        if wait_bp_active 1 35; then
            pass "#318 engage：背压指标 active=1（watchdog 已采样生效）"
        else
            fail "#318 engage active=1" "35s 内背压未触发，实际=$(metric_value "$ACS_METRICS_URL" acs_pm_upload_backpressure_active)"
        fi
        code_engaged=$(acs_upload_pm "pmbp-${SMOKE_TAG}.xml" "<measCollecFile/>")
        if [ "$code_engaged" = "503" ]; then
            pass "#318 engage：PM 上传被背压拒收 → 503"
        else
            fail "#318 engage 503" "实际 HTTP=${code_engaged}（期望 503）"
        fi

        # ---- release：阈值恢复高水位（28% < low=75% 回落）。此时 interval 已=1s，回落很快 ----
        bp_batch "{\"key\":\"disk_high_pct\",\"value\":\"85\",\"value_type\":\"int\"},
                  {\"key\":\"disk_low_pct\",\"value\":\"75\",\"value_type\":\"int\"}"
        check_ret_ok "背压配置下发（release：disk_high=85%）"
        if wait_bp_active 0 15; then
            pass "#318 release：背压指标 active=0（watchdog 已回落）"
        else
            fail "#318 release active=0" "15s 内未回落，实际=$(metric_value "$ACS_METRICS_URL" acs_pm_upload_backpressure_active)"
        fi
        code_released=$(acs_upload_pm "pmbp2-${SMOKE_TAG}.xml" "<measCollecFile/>")
        if [ "$code_released" = "200" ] || [ "$code_released" = "204" ] || [ "$code_released" = "201" ]; then
            pass "#318 release：PM 上传恢复放行 → ${code_released}"
        else
            fail "#318 release 2xx" "实际 HTTP=${code_released}（期望 2xx）"
        fi

        restore_backpressure
        pass "#318 背压阈值已恢复原值（enabled=${BP_ORIG_ENABLED} high=${BP_ORIG_HIGH} low=${BP_ORIG_LOW})"
    fi
fi

# ===========================================================================
section "C. #319 MinIO 原始件 ILM 保留期（pm-files/mr-files 60 天过期规则）"
# ===========================================================================
if ! command -v docker >/dev/null 2>&1 || ! docker inspect "$MINIO_CONTAINER" >/dev/null 2>&1; then
    skip "#319 ILM 规则核查" "docker/MinIO 容器 $MINIO_CONTAINER 不可用（非容器栈），跳过"
else
    for bucket in pm-files mr-files; do
        ilm_out=$(docker exec "$MINIO_CONTAINER" sh -c \
            "mc alias set lo http://localhost:9000 ${MINIO_USER} ${MINIO_PASS} >/dev/null 2>&1; \
             mc ilm rule ls lo/${bucket} 2>/dev/null || mc ilm ls lo/${bucket} 2>/dev/null" 2>/dev/null)
        if printf '%s' "$ilm_out" | grep -qE 'omc-raw-expire-[0-9]+d'; then
            rule=$(printf '%s' "$ilm_out" | grep -oE 'omc-raw-expire-[0-9]+d' | head -1)
            pass "#319 ILM：${bucket} 桶含过期规则 ${rule}"
        else
            fail "#319 ILM：${bucket} 桶过期规则" "未找到 omc-raw-expire-*d 规则"
        fi
    done
fi

# ===========================================================================
section "D. #836 入库后一次性压缩（worker raw_archive 指标暴露）"
# ===========================================================================
if ! endpoint_up "$WORKER_METRICS_URL"; then
    skip "#836 raw_archive 指标核查" "worker 指标不可达（$WORKER_METRICS_URL），跳过"
else
    if metric_has "$WORKER_METRICS_URL" '^omc_raw_archive_(total|saved_bytes_total)'; then
        saved=$(metric_value "$WORKER_METRICS_URL" "omc_raw_archive_saved_bytes_total")
        pass "#836 worker raw_archive 指标暴露（saved_bytes_total=${saved:-0}）"
    else
        fail "#836 raw_archive 指标暴露" "未在 worker 指标中找到 omc_raw_archive_*"
    fi
fi

# ===========================================================================
section "E. #320 基站日志保留：时间保留 + 文件数配额并存（已在 A 验证键存在）"
# ===========================================================================
# 头号语义：max_retention_days（时间）与 max_file_count（配额，0=禁用）两旋钮并存可配。
days=$(cfg_get_value stationlog.retention max_retention_days)
cnt=$(cfg_get_value stationlog.retention max_file_count)
if [ -n "$days" ] && [ -n "$cnt" ]; then
    pass "#320 时间保留($days 天)与文件数配额($cnt)两旋钮并存"
else
    fail "#320 保留双旋钮并存" "max_retention_days=$days max_file_count=$cnt（应均可配）"
fi

smoke_summary
