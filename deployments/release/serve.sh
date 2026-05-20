#!/usr/bin/env bash
# =============================================================================
# OMC 交付包下载 HTTP 服务
#
# 在构建机上起一个 HTTP 服务，把 archive/ 目录（build-release.sh 按版本归档的
# 交付包）暴露出来，使用者用浏览器（或 wget/curl）访问即可下载。
#
# 用法：
#   ./serve.sh                       # 默认端口 8000
#   ./serve.sh 9000                  # 位置参数指定端口（向后兼容）
#   ./serve.sh -p 9000               # 选项形式
#   ./serve.sh --port 9000           # 长选项
#   ./serve.sh -h | --help           # 本帮助
#
# 参数：
#   -p, --port <PORT>   监听端口（默认 8000）
#   -h, --help          本帮助
#
# 后台常驻：
#   nohup ./serve.sh 8000 >/tmp/omc-serve.log 2>&1 &
# 或做成 systemd 服务长期运行。
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── 参数解析 ────────────────────────────────────────────────────────────
PORT=""
while [ $# -gt 0 ]; do
  case "$1" in
    -p|--port) PORT="$2"; shift 2 ;;
    -h|--help) sed -n '3,21p' "$0"; exit 0 ;;
    -*)        echo "未知参数：$1（-h 查看用法）" >&2; exit 1 ;;
    *)         # 位置参数兼容老用法 ./serve.sh 8000
               [ -z "$PORT" ] || { echo "重复指定端口：$1" >&2; exit 1; }
               PORT="$1"; shift ;;
  esac
done
PORT="${PORT:-8000}"

ARCHIVE="$SCRIPT_DIR/archive"

mkdir -p "$ARCHIVE"
command -v python3 >/dev/null 2>&1 || { echo "错误：需要 python3 提供 HTTP 服务"; exit 1; }

# 启动前刷新下载索引，确保 index.html 与 archive/ 实际内容一致。
[ -x "$SCRIPT_DIR/gen-index.sh" ] && "$SCRIPT_DIR/gen-index.sh" >/dev/null 2>&1 || true

# ── archive 内容扫描（每类一行 "版本号 (N 个文件)"；让用户启动时就能确认
# 自己的 build-release.sh / build-images.sh 产物是否被检测到）────────────────
list_archive() {
  local kind="$1" label="$2" empty_hint="$3"
  local root="$ARCHIVE/$kind"
  local n=0 d v files
  if [ -d "$root" ]; then
    for d in "$root"/*/; do
      [ -d "$d" ] || continue
      n=$((n+1))
    done
  fi
  if [ "$n" = 0 ]; then
    printf '   %s ： 0 个    （%s）\n' "$label" "$empty_hint"
    return
  fi
  printf '   %s ： %d 个\n' "$label" "$n"
  for d in "$root"/*/; do
    [ -d "$d" ] || continue
    v=$(basename "$d")
    files=$(find "$d" -maxdepth 1 -type f -name 'omc-*.tar.*' ! -name '*.sha256' 2>/dev/null | wc -l | tr -d ' ')
    printf '       · %s   (%s 个交付文件)\n' "$v" "$files"
  done
}

IP="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
echo "──────────────────────────────────────────────"
echo " OMC 交付包下载服务"
echo "   根目录   ： $ARCHIVE"
echo "   地址     ： http://${IP:-<构建机IP>}:$PORT/"
list_archive project "项目版本下载" "运行 ./build-release.sh 生成"
list_archive infra   "基础设置下载" "运行 ./build-images.sh 生成"
echo "   下载行为 ： .tar.xz / .tar.gz / .tar.zst / .tgz / .sha256 自动加"
echo "              Content-Disposition: attachment（强制浏览器下载而非内嵌展示）"
echo "   日志     ： 每次 HTTP 请求打到本终端 stderr（404 / 200 可一眼分辨）"
echo "   说明     ： 浏览器打开上面地址即可看到版本列表并下载"
echo "   停止     ： Ctrl-C"
echo "──────────────────────────────────────────────"

cd "$ARCHIVE"
# 内联 Python HTTP 服务：强制为下载类后缀（.tar.* / .tgz / .sha256）发
# Content-Disposition: attachment，并把 Content-Type 强制成 octet-stream，
# 避免浏览器把交付包"在新 tab 里渲染成乱码"或被中间件接管而无法下载。
# 不依赖额外文件，纯 stdlib，与 `python3 -m http.server` 100% 等价行为外延。
exec python3 - "$PORT" <<'PYEOF'
import os, sys, http.server, socketserver
PORT = int(sys.argv[1])
DOWNLOAD_EXTS = ('.tar.xz', '.tar.gz', '.tar.zst', '.tgz', '.sha256')

def _is_download(url_path):
    p = url_path.split('?', 1)[0].split('#', 1)[0]
    return any(p.endswith(e) for e in DOWNLOAD_EXTS)

class DownloadHandler(http.server.SimpleHTTPRequestHandler):
    def guess_type(self, path):
        # 凡是下载类后缀都返回 octet-stream，避免被浏览器 / 中间件按
        # application/x-xz / application/x-tar 等做奇怪处理
        if any(path.endswith(e) for e in DOWNLOAD_EXTS):
            return 'application/octet-stream'
        return super().guess_type(path)

    def end_headers(self):
        # 在 SimpleHTTPRequestHandler.send_head() 已发完 Content-Type / Length
        # 之后追加 Content-Disposition: attachment
        try:
            if _is_download(self.path):
                fn = os.path.basename(self.translate_path(self.path))
                if fn:
                    self.send_header(
                        'Content-Disposition',
                        f'attachment; filename="{fn}"')
        except Exception:
            pass
        super().end_headers()

socketserver.TCPServer.allow_reuse_address = True
with socketserver.TCPServer(('', PORT), DownloadHandler) as srv:
    print(f'Serving HTTP on 0.0.0.0 port {PORT} ...', flush=True)
    srv.serve_forever()
PYEOF
