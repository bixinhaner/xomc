#!/usr/bin/env bash
# T-0137 M3-04: TR069 报文跟踪压测脚本骨架
#
# 验证目标（设计 §7 度量 + PRD §7）：
#   1. ACS handler P99 latency 增量 < 5%（开 100 任务前后对比）
#   2. trace_messages_dropped_total 增量 = 0（零丢失）
#   3. 报文采集端到端延迟（NATS → worker → PG）P95 < 5s
#
# 前置条件：
#   - app/acs/worker 三进程已启动
#   - 至少 100 台设备在 ACS 上（用 omcgo/bin/loadtest 持续打 Inform）
#   - $API_URL / $METRICS_URL_ACS / $METRICS_URL_WORKER 三个端点可达
#
# 使用：
#   bash scripts/trace_stress_test.sh [N_TASKS=100] [DURATION_MIN=5]

set -euo pipefail

# ----- 配置 -----
API_URL="${API_URL:-http://localhost:8081/api/v1}"
METRICS_URL_ACS="${METRICS_URL_ACS:-http://localhost:9090/metrics}"
METRICS_URL_WORKER="${METRICS_URL_WORKER:-http://localhost:9092/metrics}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
N_TASKS="${1:-100}"
DURATION_MIN="${2:-5}"

GREEN=$(tput setaf 2 2>/dev/null || echo "")
RED=$(tput setaf 1 2>/dev/null || echo "")
YELLOW=$(tput setaf 3 2>/dev/null || echo "")
NC=$(tput sgr0 2>/dev/null || echo "")

log()  { echo -e "${GREEN}[trace-stress]${NC} $*"; }
warn() { echo -e "${YELLOW}[trace-stress]${NC} $*"; }
fail() { echo -e "${RED}[trace-stress]${NC} $*" >&2; exit 1; }

auth_header=()
if [ -n "$AUTH_TOKEN" ]; then
    auth_header=(-H "Authorization: Bearer $AUTH_TOKEN")
fi

# ----- Step 0: 健康检查 -----
log "step 0: 检查端点可达性"
curl -fsS -o /dev/null "$API_URL/../healthz" || fail "app /healthz 不可达 ($API_URL)"
curl -fsS -o /dev/null "$METRICS_URL_ACS" || fail "ACS /metrics 不可达 ($METRICS_URL_ACS)"
curl -fsS -o /dev/null "$METRICS_URL_WORKER" || fail "worker /metrics 不可达 ($METRICS_URL_WORKER)"

# ----- Step 1: 采集基线（无 trace 任务时 ACS handler latency P99）-----
log "step 1: 采集 ACS handler latency 基线"
baseline_metrics=$(curl -fsS "$METRICS_URL_ACS")
baseline_p99=$(echo "$baseline_metrics" | awk '
    /^omc_acs_request_duration_seconds{quantile="0.99"/ { print $2; exit }
') || baseline_p99="N/A"
log "  baseline ACS handler P99 = ${baseline_p99}s"

baseline_dropped=$(echo "$baseline_metrics" | awk '
    /^omc_trace_messages_dropped_total/ { sum += $2 } END { print sum+0 }
')
log "  baseline trace dropped = ${baseline_dropped}"

# ----- Step 2: 批量创建 N 个抓包任务 -----
log "step 2: 创建 $N_TASKS 个抓包任务（每个 ${DURATION_MIN} 分钟）"
created_task_ids=()
for i in $(seq 1 "$N_TASKS"); do
    sn="STRESS-$(printf '%04d' "$i")"
    resp=$(curl -fsS -X POST "$API_URL/trace/tasks" \
        "${auth_header[@]}" \
        -H "Content-Type: application/json" \
        -d "{\"device_sn\":\"$sn\",\"duration_minutes\":$DURATION_MIN}" \
        2>&1) || { warn "创建 $sn 任务失败：$resp"; continue; }
    task_id=$(echo "$resp" | sed -E 's/.*"id":"([^"]+)".*/\1/')
    created_task_ids+=("$task_id")
done
log "  实际创建 ${#created_task_ids[@]} / $N_TASKS"

# ----- Step 3: 等待白名单同步（NATS 实时 + 30s 兜底）-----
log "step 3: 等待 ACS 白名单同步（5s）"
sleep 5

# ----- Step 4: 触发负载 -----
warn "step 4: 现在手动启动 omcgo/bin/loadtest 或 cpe_simulator.py 制造负载"
warn "  推荐命令：omcgo/bin/loadtest -mode acs -devices 100 -url http://localhost:7547/acs"
warn "  让负载持续运行 ${DURATION_MIN} 分钟，期间 ACS 应命中白名单并 publish JetStream"
read -r -p "  按回车键继续到 step 5 ..."

# ----- Step 5: 采集压测结束后指标 -----
log "step 5: 采集 ACS handler latency 压测后值"
stress_metrics=$(curl -fsS "$METRICS_URL_ACS")
stress_p99=$(echo "$stress_metrics" | awk '
    /^omc_acs_request_duration_seconds{quantile="0.99"/ { print $2; exit }
') || stress_p99="N/A"
log "  with-capture ACS handler P99 = ${stress_p99}s"

stress_dropped=$(echo "$stress_metrics" | awk '
    /^omc_trace_messages_dropped_total/ { sum += $2 } END { print sum+0 }
')
log "  with-capture trace dropped = ${stress_dropped}"

capture_p99=$(echo "$stress_metrics" | awk '
    /^omc_trace_capture_latency_seconds{quantile="0.99"/ { print $2; exit }
') || capture_p99="N/A"
log "  trace capture hook P99 = ${capture_p99}s（反例阈值 0.005s）"

captured_total=$(echo "$stress_metrics" | awk '
    /^omc_trace_messages_captured_total/ { sum += $2 } END { print sum+0 }
')
log "  trace messages captured = ${captured_total}"

# worker 端落库统计
worker_metrics=$(curl -fsS "$METRICS_URL_WORKER")
worker_active=$(echo "$worker_metrics" | awk '
    /^omc_trace_active_tasks/ { print $2; exit }
') || worker_active=0
log "  worker active_tasks gauge = ${worker_active}"

# ----- Step 6: 评估 -----
log "step 6: 评估反例"
fail_count=0

# 反例 1: capture hook P99 > 5ms
if [[ "$capture_p99" =~ ^[0-9.]+$ ]]; then
    if awk "BEGIN{exit !($capture_p99 > 0.005)}"; then
        warn "❌ capture hook P99 ${capture_p99}s > 0.005s (5ms) — 违反反例阈值"
        fail_count=$((fail_count+1))
    else
        log "✓ capture hook P99 ${capture_p99}s ≤ 0.005s"
    fi
fi

# 反例 2: dropped 增量 > 0
delta_dropped=$((stress_dropped - baseline_dropped))
if [ "$delta_dropped" -gt 0 ]; then
    warn "❌ dropped 增量 ${delta_dropped} > 0 — 报文丢失"
    fail_count=$((fail_count+1))
else
    log "✓ dropped 增量 0"
fi

# 反例 3: ACS handler P99 增量 > 5%
if [[ "$baseline_p99" =~ ^[0-9.]+$ ]] && [[ "$stress_p99" =~ ^[0-9.]+$ ]]; then
    delta_ratio=$(awk "BEGIN{printf \"%.4f\", ($stress_p99 - $baseline_p99) / $baseline_p99}")
    if awk "BEGIN{exit !($delta_ratio > 0.05)}"; then
        warn "❌ ACS handler P99 增量 ${delta_ratio} > 5% — 影响主路径性能"
        fail_count=$((fail_count+1))
    else
        log "✓ ACS handler P99 增量 ${delta_ratio} ≤ 5%"
    fi
else
    warn "⊘ ACS handler P99 比对跳过（基线或压测值无效）"
fi

echo ""
if [ "$fail_count" -gt 0 ]; then
    fail "压测发现 $fail_count 处反例，详见上文"
fi
log "压测通过 ✓ captured=${captured_total} dropped=${delta_dropped} active=${worker_active}"
