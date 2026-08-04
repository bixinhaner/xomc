#!/usr/bin/env bash
# OMC ACS 阶梯式压力测试
# 用法: ./scripts/loadtest-benchmark.sh
set -euo pipefail

LOADTEST="./bin/loadtest"
ACS_URL="http://localhost:7547/acs"
DURATION="60s"
REPORT_DIR="test"
REPORT_FILE="$REPORT_DIR/benchmark-report-$(date '+%Y%m%d-%H%M%S').txt"

# 4 轮测试参数: "并发 设备数"
ROUNDS=(
    "100 2000"
    "500 5000"
    "1000 10000"
    "3000 30000"
)

# ---- 前置检查 ----
if [ ! -f "$LOADTEST" ]; then
    echo "Error: $LOADTEST not found. Run: go build -o bin/loadtest ./scripts/loadtest/"
    exit 1
fi

acs_code=$(curl -s -o /dev/null -w "%{http_code}" "$ACS_URL" -X POST -H "Content-Type: text/xml" -d '<test/>' 2>/dev/null || true)
if [[ "$acs_code" != "200" && "$acs_code" != "400" ]]; then
    echo "Error: ACS not reachable at $ACS_URL (HTTP $acs_code)"
    exit 1
fi

mkdir -p "$REPORT_DIR"

# ---- 输出头 ----
header() {
    cat <<'HEADER'
============================================
       OMC ACS 压力测试报告
============================================
HEADER
    echo "日期:   $(date '+%Y-%m-%d %H:%M:%S')"
    echo "目标:   $ACS_URL"
    echo "每轮:   $DURATION"
    echo "环境:   $(uname -s) $(uname -m), $(sysctl -n hw.ncpu 2>/dev/null || nproc) CPU"
    echo "Go:     $(go version 2>/dev/null | awk '{print $3}')"
    echo ""
}

# ---- 收集系统快照 ----
sys_snapshot() {
    local label=$1
    echo "--- $label 系统快照 ---"
    echo "  Redis 内存: $(redis-cli info memory 2>/dev/null | grep used_memory_human | tr -d '\r' | cut -d: -f2)"
    echo "  Redis 连接: $(redis-cli info clients 2>/dev/null | grep connected_clients | tr -d '\r' | cut -d: -f2)"
    local acs_sessions=$(curl -sf http://localhost:9090/metrics 2>/dev/null | grep "^acs_global_active_sessions " | awk '{print $2}')
    echo "  ACS 活跃会话: ${acs_sessions:-0}"
    echo ""
}

# ---- 单轮测试 ----
run_round() {
    local round=$1
    local concurrency=$2
    local devices=$3

    echo "============================================"
    echo "  Round $round: ${concurrency} 并发 / ${devices} 设备 / ${DURATION}"
    echo "============================================"
    echo ""

    sys_snapshot "测试前"

    echo ">>> 开始测试..."
    local json_result
    json_result=$("$LOADTEST" \
        -url "$ACS_URL" \
        -devices "$devices" \
        -concurrency "$concurrency" \
        -duration "$DURATION" \
        -interval 1s \
        -json 2>/dev/null || echo '{"error":true}')

    # Parse JSON output
    local total success failed errors throughput success_rate p50 p90 p95 p99 max_lat duration_s
    total=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total',0))" 2>/dev/null || echo "0")
    success=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('success',0))" 2>/dev/null || echo "0")
    failed=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('failed',0))" 2>/dev/null || echo "0")
    errors=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('errors',0))" 2>/dev/null || echo "0")
    throughput=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('throughput',0):.1f}\")" 2>/dev/null || echo "0.0")
    success_rate=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('success_rate',0):.1f}\")" 2>/dev/null || echo "0.0")
    p50=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('p50_ms',0):.1f}\")" 2>/dev/null || echo "0")
    p90=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('p90_ms',0):.1f}\")" 2>/dev/null || echo "0")
    p95=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('p95_ms',0):.1f}\")" 2>/dev/null || echo "0")
    p99=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('p99_ms',0):.1f}\")" 2>/dev/null || echo "0")
    max_lat=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('max_ms',0):.1f}\")" 2>/dev/null || echo "0")
    duration_s=$(echo "$json_result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d.get('duration_ms',0)/1000:.1f}\")" 2>/dev/null || echo "0")

    echo "  Duration:    ${duration_s}s"
    echo "  Total:       $total"
    echo "  Success:     $success (${success_rate}%)"
    echo "  Failed:      $failed"
    echo "  Errors:      $errors"
    echo "  Throughput:  ${throughput} req/s"
    echo ""
    echo "  Latency Percentiles:"
    echo "    p50: ${p50}ms  p90: ${p90}ms  p95: ${p95}ms  p99: ${p99}ms  max: ${max_lat}ms"
    echo ""

    sys_snapshot "测试后"

    # Store results for summary
    RESULTS+=("$concurrency|$devices|$total|$success|${success_rate}|${throughput}|${p50}|${p90}|${p95}|${p99}|${max_lat}")
}

# ---- 汇总表 ----
print_summary() {
    echo ""
    echo "============================================"
    echo "  对比汇总"
    echo "============================================"
    echo ""
    printf "| %-6s | %-8s | %-10s | %-11s | %-7s | %-7s | %-7s | %-8s |\n" \
        "并发" "设备数" "总请求" "吞吐(req/s)" "p50" "p90" "p99" "成功率"
    printf "|--------|----------|------------|-------------|---------|---------|---------|----------|\n"
    for result in "${RESULTS[@]}"; do
        IFS='|' read -r conc devs total succ rate tput p50 p90 p95 p99 maxl <<< "$result"
        printf "| %-6s | %-8s | %-10s | %-11s | %-5sms | %-5sms | %-5sms | %-6s%% |\n" \
            "$conc" "$devs" "$total" "$tput" "$p50" "$p90" "$p99" "$rate"
    done
    echo ""
}

# ---- 主流程 ----
RESULTS=()

{
    header

    round_num=0
    for params in "${ROUNDS[@]}"; do
        round_num=$((round_num + 1))
        read -r conc devs <<< "$params"
        run_round "$round_num" "$conc" "$devs"

        # 轮间冷却
        if [ "$round_num" -lt "${#ROUNDS[@]}" ]; then
            echo ">>> 冷却 5s..."
            sleep 5
            echo ""
        fi
    done

    print_summary
    echo "报告已保存: $REPORT_FILE"

} 2>&1 | tee "$REPORT_FILE"
