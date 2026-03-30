#!/bin/bash
# scripts/minio_lifecycle.sh — MinIO 生命周期策略配置
# 使用前需确保 mc 已安装并配置 alias:
#   mc alias set omc http://localhost:9000 minioadmin minioadmin
#
# 用法: bash scripts/minio_lifecycle.sh [alias]
#   alias 默认为 "omc"

set -euo pipefail

MC="${MC:-mc}"
ALIAS="${1:-omc}"

echo "Configuring MinIO lifecycle rules for alias: ${ALIAS}"

# PM 数据: 90 天
$MC ilm rule add ${ALIAS}/omc-pm --expiry-days 90
echo "  omc-pm: 90 days"

# MR 数据: 90 天
$MC ilm rule add ${ALIAS}/omc-mr --expiry-days 90
echo "  omc-mr: 90 days"

# 日志分级过期
$MC ilm rule add ${ALIAS}/omc-logs --prefix "running/" --expiry-days 30
$MC ilm rule add ${ALIAS}/omc-logs --prefix "security/" --expiry-days 30
$MC ilm rule add ${ALIAS}/omc-logs --prefix "fault/" --expiry-days 180
$MC ilm rule add ${ALIAS}/omc-logs --prefix "pcap/" --expiry-days 7
echo "  omc-logs: running=30d, security=30d, fault=180d, pcap=7d"

# 报表: 180 天
$MC ilm rule add ${ALIAS}/omc-reports --expiry-days 180
echo "  omc-reports: 180 days"

# 交换区: 7 天（import/export 临时文件）
$MC ilm rule add ${ALIAS}/omc-exchange --prefix "import/" --expiry-days 7
$MC ilm rule add ${ALIAS}/omc-exchange --prefix "export/" --expiry-days 7
echo "  omc-exchange: import=7d, export=7d"

# 配置备份: 365 天
$MC ilm rule add ${ALIAS}/omc-config --prefix "backup/" --expiry-days 365
echo "  omc-config: backup=365 days"

# 固件和数据模型: 不设过期（永久保留）
echo "  omc-firmware: no expiry (permanent)"

echo ""
echo "MinIO lifecycle rules configured successfully."
