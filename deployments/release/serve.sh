#!/usr/bin/env bash
# =============================================================================
# OMC 离线版本下载 HTTP 服务（支持前台 / 后台守护两种模式）
#
# 用法（前台，旧用法保留兼容）：
#   ./serve.sh                       # 默认端口 8000
#   ./serve.sh 9000                  # 位置参数指定端口
#   ./serve.sh -p 9000               # 选项形式
#   ./serve.sh --port 9000           # 长选项
#   ./serve.sh -h | --help           # 本帮助
#
# 用法（后台守护）：
#   ./serve.sh start [-p PORT|PORT]  # 启动并 detach；写 PID/端口到隐藏文件
#   ./serve.sh stop                  # 停止后台进程
#   ./serve.sh restart [-p PORT]     # 重启（不传端口时沿用上次端口）
#   ./serve.sh status                # 查看运行状态
#
# 后台模式相关文件：
#   PID 文件 ：  <脚本目录>/.serve.pid
#   端口缓存 ：  <脚本目录>/.serve.port
#   日志文件 ：  <脚本目录>/.serve.log
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARCHIVE="$SCRIPT_DIR/archive"
PID_FILE="$SCRIPT_DIR/.serve.pid"
PORT_FILE="$SCRIPT_DIR/.serve.port"
LOG_FILE="$SCRIPT_DIR/.serve.log"
DEFAULT_PORT=8000

die()   { echo "错误：$*" >&2; exit 1; }
usage() { sed -n '3,23p' "$0"; }

is_running() {
  [ -f "$PID_FILE" ] || return 1
  local pid; pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  [ -n "$pid" ] || return 1
  kill -0 "$pid" 2>/dev/null
}

# ── 参数解析 ────────────────────────────────────────────────────────────
# 第一个位置参数若是子命令（start|stop|restart|status）则作为 ACTION，否则
# ACTION=foreground 走老用法。__serve 是 start 内部用来 detach 出来的真正 worker。
ACTION="foreground"
case "${1:-}" in
  start|stop|restart|status|__serve) ACTION="$1"; shift ;;
  -h|--help) usage; exit 0 ;;
esac

PORT=""
while [ $# -gt 0 ]; do
  case "$1" in
    -p|--port) PORT="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    -*)        die "未知参数：$1（-h 查看用法）" ;;
    *)         [ -z "$PORT" ] || die "重复指定端口：$1"
               PORT="$1"; shift ;;
  esac
done

# 各 action 拿端口的策略不同：
#   foreground / __serve / start ：命令行 → DEFAULT_PORT
#   restart                       ：命令行 → 上次端口缓存 → DEFAULT_PORT
#   stop / status                 ：用不到端口
case "$ACTION" in
  restart) PORT="${PORT:-$(cat "$PORT_FILE" 2>/dev/null || echo "")}" ;;
esac
PORT="${PORT:-$DEFAULT_PORT}"

# ── 共用：archive 内容扫描 + 启动横幅 ────────────────────────────────────
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

print_banner() {
  local port="$1" stop_hint="$2"
  local ip; ip="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
  echo "──────────────────────────────────────────────"
  echo " OMC 离线版本下载服务"
  echo "   根目录   ： $ARCHIVE"
  echo "   地址     ： http://${ip:-<构建机IP>}:$port/"
  list_archive project "项目版本下载" "运行 ./build-release.sh 生成"
  list_archive infra   "基础设施下载" "运行 ./build-images.sh 生成"
  echo "   下载行为 ： .tar.xz / .tar.gz / .tar.zst / .tgz / .sha256 自动加"
  echo "              Content-Disposition: attachment（强制浏览器下载而非内嵌展示）"
  echo "   日志     ： $stop_hint"
  echo "──────────────────────────────────────────────"
}

prepare_env() {
  mkdir -p "$ARCHIVE"
  command -v python3 >/dev/null 2>&1 || die "需要 python3 提供 HTTP 服务"
  # 启动前刷新下载索引，确保 index.html 与 archive/ 实际内容一致。
  [ -x "$SCRIPT_DIR/gen-index.sh" ] && "$SCRIPT_DIR/gen-index.sh" >/dev/null 2>&1 || true
}

# 真正的 HTTP 服务（前台 / 后台 worker 共用此函数）
run_http_server() {
  local port="$1"
  cd "$ARCHIVE"
  exec python3 - "$port" <<'PYEOF'
import os, sys, http.server, socketserver
PORT = int(sys.argv[1])
DOWNLOAD_EXTS = ('.tar.xz', '.tar.gz', '.tar.zst', '.tgz', '.sha256')

def _is_download(url_path):
    p = url_path.split('?', 1)[0].split('#', 1)[0]
    return any(p.endswith(e) for e in DOWNLOAD_EXTS)

class DownloadHandler(http.server.SimpleHTTPRequestHandler):
    def guess_type(self, path):
        if any(path.endswith(e) for e in DOWNLOAD_EXTS):
            return 'application/octet-stream'
        return super().guess_type(path)

    def end_headers(self):
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
}

# ── action 路由 ─────────────────────────────────────────────────────────
case "$ACTION" in
  status)
    if is_running; then
      pid="$(cat "$PID_FILE")"
      port="$(cat "$PORT_FILE" 2>/dev/null || echo "?")"
      echo "运行中：PID=$pid，端口=$port"
      echo "日志：$LOG_FILE"
    else
      echo "未运行"
      [ -f "$PID_FILE" ] && echo "（残留 PID 文件 $PID_FILE，可手动清理）"
    fi
    exit 0
    ;;

  stop)
    if is_running; then
      pid="$(cat "$PID_FILE")"
      kill "$pid" 2>/dev/null || true
      for _ in 1 2 3 4 5 6 7 8 9 10; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.5
      done
      if kill -0 "$pid" 2>/dev/null; then
        echo "10s 内未退出，发送 SIGKILL"
        kill -9 "$pid" 2>/dev/null || true
        sleep 0.5
      fi
      rm -f "$PID_FILE"
      echo "已停止 (PID=$pid)"
    else
      echo "未运行"
      [ -f "$PID_FILE" ] && rm -f "$PID_FILE"
    fi
    exit 0
    ;;

  restart)
    "$0" stop || true
    exec "$0" start -p "$PORT"
    ;;

  start)
    if is_running; then
      echo "已在运行 (PID=$(cat "$PID_FILE"))，请先 stop 或 restart" >&2
      exit 1
    fi
    prepare_env
    print_banner "$PORT" "$LOG_FILE（tail -f 查看）"
    # 用 setsid 摆脱终端组；nohup 也行但 setsid 更彻底，关 ssh 不带走进程。
    if command -v setsid >/dev/null 2>&1; then
      setsid bash "$0" __serve -p "$PORT" >>"$LOG_FILE" 2>&1 </dev/null &
    else
      nohup bash "$0" __serve -p "$PORT" >>"$LOG_FILE" 2>&1 </dev/null &
    fi
    pid=$!
    disown "$pid" 2>/dev/null || true
    echo "$pid"   > "$PID_FILE"
    echo "$PORT"  > "$PORT_FILE"
    sleep 0.5
    if kill -0 "$pid" 2>/dev/null; then
      echo "已启动 (PID=$pid，端口 $PORT)"
      echo "停止：$0 stop   |   状态：$0 status   |   日志：tail -f $LOG_FILE"
    else
      rm -f "$PID_FILE" "$PORT_FILE"
      echo "启动失败，查看 $LOG_FILE 末尾错误：" >&2
      tail -n 20 "$LOG_FILE" >&2 || true
      exit 1
    fi
    exit 0
    ;;

  __serve)
    # 后台 worker：start 通过 setsid/nohup 唤起，stdin/stdout/stderr 已重定向到 $LOG_FILE
    prepare_env
    print_banner "$PORT" "本日志（$LOG_FILE）"
    run_http_server "$PORT"
    ;;

  foreground)
    prepare_env
    print_banner "$PORT" "每次 HTTP 请求打到本终端 stderr（Ctrl-C 停止）"
    run_http_server "$PORT"
    ;;
esac
