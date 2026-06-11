#!/usr/bin/env bash
# 一键清理 KPI 上传压测产生的测试数据：
#   - devices     （SN 前缀 LIKE 命中的测试设备）
#   - pm_files    （这些设备的 PM 文件元数据）
#   - pm_metrics  （这些设备的 counter + KPI 行）
#
# 用法：
#   bash test/pmperf/cleanup.sh                      # 清理默认前缀 KPILT
#   SN_PREFIX=FOO bash test/pmperf/cleanup.sh        # 清理自定义前缀
#   DB_DSN='postgres://u:p@host:5432/db?sslmode=disable' bash test/pmperf/cleanup.sh
#   bash test/pmperf/cleanup.sh -sn-prefix BAR       # 透传任意 kpiperf 参数（覆盖默认）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OMCGO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"   # test/pmperf -> omcgo
cd "$OMCGO_DIR"

PREFIX="${SN_PREFIX:-KPILT}"
DSN="${DB_DSN:-postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable}"

echo "→ 清理 SN 前缀 '${PREFIX}' 的测试设备与 KPI 数据"
echo "  DB: ${DSN}"

# 优先用已编译的 bin/kpiperf，没有则 go run（首次会编译，稍慢）。
if [[ -x "$OMCGO_DIR/bin/kpiperf" ]]; then
  "$OMCGO_DIR/bin/kpiperf" -mode cleanup -sn-prefix "$PREFIX" -db "$DSN" "$@"
else
  go run ./test/pmperf -mode cleanup -sn-prefix "$PREFIX" -db "$DSN" "$@"
fi
