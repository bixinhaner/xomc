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
#   ./serve.sh start [-p PORT|PORT]  # 启动并 detach
#   ./serve.sh stop                  # 停止后台进程
#   ./serve.sh restart [-p PORT]     # 重启（不传端口时沿用上次端口）
#   ./serve.sh status                # 查看运行状态
#
# 后台保活策略（start / restart 自动选择）：
#   · root + systemd 机器 → 用 systemd 瞬态服务 omc-serve-auto 托管，后台进程
#     脱离登录会话，关 ssh / 退出登录都不掉（挺过 logind KillUserProcesses=yes）。
#     这是 build-release.sh 末尾自动 restart 后下载服务能持续在线的关键。
#     日志走 journald：journalctl -u omc-serve-auto -f
#   · 非 root / 无 systemd → 回退 setsid+nohup（写 .serve.pid/.serve.log）。
#     注意：开了 KillUserProcesses=yes 的 systemd 系统上，此回退路径登出仍可能
#     被带走；需要登出后/重启后保活请装静态 unit（见 README「下载服务长期运行」）。
#
# 后台模式相关文件（隐藏在脚本目录）：
#   .serve.unit ：  systemd 瞬态服务模式标记（内容=unit 名；存在即走 systemd）
#   .serve.pid  ：  setsid/nohup 模式的后台进程 PID
#   .serve.port ：  上次启动端口（restart 沿用）
#   .serve.log  ：  setsid/nohup 模式的请求日志（systemd 模式日志在 journald）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARCHIVE="$SCRIPT_DIR/archive"
PID_FILE="$SCRIPT_DIR/.serve.pid"
PORT_FILE="$SCRIPT_DIR/.serve.port"
LOG_FILE="$SCRIPT_DIR/.serve.log"
UNIT_FILE="$SCRIPT_DIR/.serve.unit"   # 存在则后台用 systemd 瞬态服务托管；内容=unit 名
RUN_UNIT="omc-serve-auto"             # systemd 瞬态服务名（区别于仓库内的静态 omc-serve.service）
DEFAULT_PORT=8000

die()   { echo "错误：$*" >&2; exit 1; }
usage() { sed -n '3,31p' "$0"; }

# 是否具备 systemd 托管条件：root + systemd-run 可用 + systemd 是 1 号进程。
# 满足时后台进程交给 systemd 瞬态服务（脱离登录会话），可挺过 SSH 退出 /
# logind 的 KillUserProcesses=yes —— 这是 setsid/nohup 单独做不到的。
systemd_capable() {
  [ "$(id -u)" = 0 ] || return 1
  command -v systemd-run >/dev/null 2>&1 || return 1
  [ -d /run/systemd/system ] || return 1
  return 0
}

is_running() {
  # systemd 瞬态服务模式
  if [ -f "$UNIT_FILE" ]; then
    local unit; unit="$(cat "$UNIT_FILE" 2>/dev/null || true)"
    [ -n "$unit" ] || return 1
    systemctl is-active --quiet "$unit" 2>/dev/null
    return
  fi
  # 传统 setsid/nohup 模式
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
import os, sys, http.server
PORT = int(sys.argv[1])
DOWNLOAD_EXTS = ('.tar.xz', '.tar.gz', '.tar.zst', '.tgz', '.sha256')

def _is_download(url_path):
    p = url_path.split('?', 1)[0].split('#', 1)[0]
    return any(p.endswith(e) for e in DOWNLOAD_EXTS)

class DownloadHandler(http.server.SimpleHTTPRequestHandler):
    # 连接级 socket 超时（#209）：慢/半开客户端（连上不读）最多占一条线程，
    # 到点 socket 操作抛 timeout 自动释放，整服务不再被一条僵死连接拖垮。
    # StreamRequestHandler.setup() 据此对连接 settimeout(self.timeout)。
    timeout = 120

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

# 多线程下载服务（#209）：每请求一线程，一次大/慢的 omc-*.tar.xz 下载不再独占唯一 worker、
# 把 index.html 与其它下载全部排队拖死（单线程 TCPServer 的老问题，重启才恢复）。
class ThreadingServer(http.server.ThreadingHTTPServer):
    daemon_threads = True          # 进程退出不被在传下载线程阻塞
    allow_reuse_address = True     # 重启立即重新 bind，不卡 TIME_WAIT
    request_queue_size = 128       # 加大 listen backlog，突发并发不被 connection refused

with ThreadingServer(('', PORT), DownloadHandler) as srv:
    print(f'Serving HTTP on 0.0.0.0 port {PORT} (threading) ...', flush=True)
    srv.serve_forever()
PYEOF
}

# ── action 路由 ─────────────────────────────────────────────────────────
case "$ACTION" in
  status)
    port="$(cat "$PORT_FILE" 2>/dev/null || echo "?")"
    if is_running; then
      if [ -f "$UNIT_FILE" ]; then
        unit="$(cat "$UNIT_FILE")"
        echo "运行中（systemd 瞬态服务 ${unit}），端口=$port"
        echo "日志：journalctl -u $unit -f"
      else
        echo "运行中：PID=$(cat "$PID_FILE")，端口=$port"
        echo "日志：$LOG_FILE"
      fi
    else
      echo "未运行"
      [ -f "$UNIT_FILE" ] && echo "（残留 unit 标记 ${UNIT_FILE}，可手动清理）"
      [ -f "$PID_FILE" ]  && echo "（残留 PID 文件 ${PID_FILE}，可手动清理）"
    fi
    exit 0
    ;;

  stop)
    # systemd 瞬态服务模式
    if [ -f "$UNIT_FILE" ]; then
      unit="$(cat "$UNIT_FILE" 2>/dev/null || true)"
      if [ -n "$unit" ] && systemctl is-active --quiet "$unit" 2>/dev/null; then
        systemctl stop "$unit" 2>/dev/null || true
        echo "已停止（systemd 瞬态服务 ${unit}）"
      else
        echo "未运行"
      fi
      systemctl reset-failed "$unit" 2>/dev/null || true
      rm -f "$UNIT_FILE"
      exit 0
    fi
    # 传统 setsid/nohup 模式
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
      if [ -f "$UNIT_FILE" ]; then
        echo "已在运行（systemd 瞬态服务 $(cat "$UNIT_FILE")），请先 stop 或 restart" >&2
      else
        echo "已在运行 (PID=$(cat "$PID_FILE"))，请先 stop 或 restart" >&2
      fi
      exit 1
    fi
    prepare_env
    print_banner "$PORT" "见下方提示"

    # 优先：systemd 瞬态服务托管（root + systemd）。后台进程进入独立的
    # systemd cgroup，脱离登录会话 —— 关 ssh / 退出登录都不会被 logind 的
    # KillUserProcesses 带走（setsid/nohup 在开了该选项的系统上挺不过登出）。
    if systemd_capable; then
      systemctl reset-failed "$RUN_UNIT" 2>/dev/null || true
      if systemd-run --quiet --unit="$RUN_UNIT" \
           --description="OMC 离线版本下载服务 (serve.sh :$PORT)" \
           --working-directory="$ARCHIVE" \
           bash "$0" __serve -p "$PORT" 2>/dev/null; then
        echo "$RUN_UNIT" > "$UNIT_FILE"
        echo "$PORT"     > "$PORT_FILE"
        rm -f "$PID_FILE"
        sleep 0.5
        if systemctl is-active --quiet "$RUN_UNIT"; then
          echo "已启动（systemd 瞬态服务 ${RUN_UNIT}，端口 ${PORT}）—— 关 ssh / 退出登录不掉。"
          echo "停止：$0 stop   |   状态：$0 status   |   日志：journalctl -u $RUN_UNIT -f"
          echo "（开机自启 / 崩溃重启请装静态 unit：见 deployments/release/README.md「下载服务长期运行」）"
          exit 0
        fi
        rm -f "$UNIT_FILE"
        echo "systemd 瞬态服务未就绪，回退到 setsid/nohup ..." >&2
        echo "排查：journalctl -u $RUN_UNIT --no-pager | tail -n 20" >&2
      else
        echo "systemd-run 启动失败，回退到 setsid/nohup（关 ssh 可能不保活）..." >&2
      fi
    fi

    # 回退：setsid 摆脱终端组（nohup 兜底）。非 root / 无 systemd 时用此路径；
    # 注意：systemd 系统若开启 KillUserProcesses=yes，登出仍可能带走本进程，
    # 此时建议改用静态 systemd unit（见 README）。
    rm -f "$UNIT_FILE"
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
      echo "已启动 (PID=${pid}，端口 $PORT)"
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
    print_banner "$PORT" "本日志（${LOG_FILE}）"
    run_http_server "$PORT"
    ;;

  foreground)
    prepare_env
    print_banner "$PORT" "每次 HTTP 请求打到本终端 stderr（Ctrl-C 停止）"
    run_http_server "$PORT"
    ;;
esac
