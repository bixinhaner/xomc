#!/usr/bin/env bash
# =============================================================================
# OMC 交付包下载 HTTP 服务
#
# 在构建机上起一个 HTTP 服务，把 archive/ 目录（build-release.sh 按版本归档的
# 交付包）暴露出来，使用者用浏览器（或 wget/curl）访问即可下载。
#
# 用法： ./serve.sh [端口]            # 端口缺省 8000
#
# 后台常驻可用： nohup ./serve.sh 8000 >/tmp/omc-serve.log 2>&1 &
# 或做成 systemd 服务长期运行。
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT="${1:-8000}"
ARCHIVE="$SCRIPT_DIR/archive"

mkdir -p "$ARCHIVE"
command -v python3 >/dev/null 2>&1 || { echo "错误：需要 python3 提供 HTTP 服务"; exit 1; }

# 启动前刷新下载索引，确保 index.html 与 archive/ 实际内容一致。
[ -x "$SCRIPT_DIR/gen-index.sh" ] && "$SCRIPT_DIR/gen-index.sh" >/dev/null 2>&1 || true

IP="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
echo "──────────────────────────────────────────────"
echo " OMC 交付包下载服务"
echo "   根目录： $ARCHIVE"
echo "   地址  ： http://${IP:-<构建机IP>}:$PORT/"
echo "   说明  ： 浏览器打开上面地址即可看到版本列表并下载"
echo "   停止  ： Ctrl-C"
echo "──────────────────────────────────────────────"

cd "$ARCHIVE"
# 有 index.html 时 http.server 自动作为首页；子目录无 index 时自动列目录。
exec python3 -m http.server "$PORT"
