#!/usr/bin/env bash
# =============================================================================
# 系统加速设置 — Docker / npm / Golang 三合一
#
# 一次性配置 3 类常见加速器（每项独立可选，可选"不设置"=使用官方）：
#   · Docker registry-mirrors          → /etc/docker/daemon.json
#   · npm registry + disturl           → /etc/npmrc（系统级，影响所有用户）
#   · Golang GOPROXY + GOSUMDB         → /etc/profile.d/goproxy.sh（新 shell 生效）
#
# 不下载任何东西，只改本地配置文件。
#
# 用法：
#   sudo bash setup-mirrors.sh                                     # 交互：逐项询问 3 项
#   sudo bash setup-mirrors.sh --docker daocloud                   # 仅设 Docker
#   sudo bash setup-mirrors.sh --npm taobao                        # 仅设 npm
#   sudo bash setup-mirrors.sh --golang goproxycn                  # 仅设 Go
#   sudo bash setup-mirrors.sh --docker daocloud --npm taobao --golang goproxycn
#   sudo bash setup-mirrors.sh --show                              # 仅展示当前 3 项配置
#   sudo bash setup-mirrors.sh --remove                            # 取消全部加速（=官方×3）
#   sudo bash setup-mirrors.sh -h | --help
#
# 参数：
#   --docker <official|daocloud|xuanyuan>
#       Docker registry-mirrors。official=不设置（走 docker hub 官方）。
#       daocloud=https://docker.m.daocloud.io（推荐）。
#       xuanyuan=https://docker.xuanyuan.me。
#   --npm <official|taobao>
#       npm registry。official=不设置（走 registry.npmjs.org）。
#       taobao=https://registry.npmmirror.com（推荐，对应 disturl 也设）。
#   --golang <official|goproxycn>
#       Golang 模块代理。official=不设置（走 proxy.golang.org）。
#       goproxycn=https://goproxy.cn,direct（推荐，配套 GOSUMDB=sum.golang.google.cn）。
#   --mirror <name>          【已废弃】等价 --docker <name>，仅为兼容老调用
#   --show                   仅展示当前 3 项配置，不修改任何文件（不需要 root）
#   --remove                 取消全部加速（等价 --docker official --npm official --golang official）
#   -h | --help              本帮助
#
# 安全性：
#   · 修改前自动备份目标文件 → <path>.bak.<时间戳>
#   · daemon.json 的其它键不动；/etc/npmrc 仅替换 registry= / disturl= 行
#   · 仅在内容确变时 systemctl restart docker；npm / Golang 配置不需重启服务
# =============================================================================
set -euo pipefail

DAEMON_JSON="/etc/docker/daemon.json"
NPMRC="/etc/npmrc"
GOPROXY_FILE="/etc/profile.d/goproxy.sh"

log()  { echo -e "\033[1;36m[mirrors]\033[0m $*"; }
warn() { echo -e "\033[1;33m[mirrors][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[mirrors][错误]\033[0m $*" >&2; exit 1; }

show_help() { sed -n '3,43p' "$0"; exit 0; }

# ── 参数解析 ─────────────────────────────────────────────────────────────
DOCKER_CHOICE=""
NPM_CHOICE=""
GOLANG_CHOICE=""
SHOW=0
REMOVE_ALL=0
while [ $# -gt 0 ]; do
  case "$1" in
    --docker) DOCKER_CHOICE="$2"; shift 2 ;;
    --npm)    NPM_CHOICE="$2";    shift 2 ;;
    --golang) GOLANG_CHOICE="$2"; shift 2 ;;
    --mirror) DOCKER_CHOICE="$2"; shift 2
              warn "--mirror 已废弃，等价 --docker $DOCKER_CHOICE，请改用 --docker" ;;
    --show)   SHOW=1; shift ;;
    --remove) REMOVE_ALL=1; shift ;;
    -h|--help) show_help ;;
    *) die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# 校验 choice 取值
case "$DOCKER_CHOICE"  in official|daocloud|xuanyuan|"") ;; *) die "--docker 取值非法：$DOCKER_CHOICE（应为 official/daocloud/xuanyuan）" ;; esac
case "$NPM_CHOICE"     in official|taobao|"") ;;            *) die "--npm 取值非法：$NPM_CHOICE（应为 official/taobao）" ;; esac
case "$GOLANG_CHOICE"  in official|goproxycn|"") ;;         *) die "--golang 取值非法：$GOLANG_CHOICE（应为 official/goproxycn）" ;; esac

# --remove：等价于 3 个 official
if [ "$REMOVE_ALL" = 1 ]; then
  DOCKER_CHOICE="official"; NPM_CHOICE="official"; GOLANG_CHOICE="official"
fi

# ── --show 仅展示，不需要 root ──────────────────────────────────────────
if [ "$SHOW" = 1 ]; then
  echo "── Docker registry-mirrors ──"
  if [ -f "$DAEMON_JSON" ]; then
    sed 's/^/  /' "$DAEMON_JSON"
  else
    echo "  (未配置，$DAEMON_JSON 不存在)"
  fi
  echo
  echo "── npm registry ──"
  if [ -f "$NPMRC" ]; then
    if grep -qE '^(registry|disturl)=' "$NPMRC"; then
      grep -E '^(registry|disturl)=' "$NPMRC" | sed 's/^/  /'
    else
      echo "  (registry / disturl 未配置)"
    fi
  else
    echo "  (未配置，$NPMRC 不存在)"
  fi
  echo
  echo "── Golang GOPROXY ──"
  if [ -f "$GOPROXY_FILE" ]; then
    sed 's/^/  /' "$GOPROXY_FILE"
  else
    echo "  (未配置，$GOPROXY_FILE 不存在 — 走 Go 默认 proxy.golang.org)"
  fi
  exit 0
fi

# ── 其它操作需要 root ───────────────────────────────────────────────────
[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

# ── 交互式补齐缺失的选择 ─────────────────────────────────────────────────
# 仅在「3 个 --xxx 都没传 且 不是 --remove」时进入逐项询问
if [ -z "$DOCKER_CHOICE$NPM_CHOICE$GOLANG_CHOICE" ]; then
  cat <<MENU

─────────────────────────────────────────────────
 系统加速设置 — 共 3 项可选（每项可独立"不设置"=用官方）
─────────────────────────────────────────────────

[1/3] Docker 镜像加速
  1) official  不设置（走 docker hub 官方）
  2) daocloud  https://docker.m.daocloud.io      (推荐)
  3) xuanyuan  https://docker.xuanyuan.me
MENU
  read -rp "选择 [1-3]，默认 2： " c
  case "${c:-2}" in
    1) DOCKER_CHOICE=official ;;
    2) DOCKER_CHOICE=daocloud ;;
    3) DOCKER_CHOICE=xuanyuan ;;
    *) die "无效选择：$c" ;;
  esac

  cat <<MENU

[2/3] npm 镜像加速
  1) official  不设置（走 registry.npmjs.org）
  2) taobao    https://registry.npmmirror.com    (推荐)
MENU
  read -rp "选择 [1-2]，默认 2： " c
  case "${c:-2}" in
    1) NPM_CHOICE=official ;;
    2) NPM_CHOICE=taobao ;;
    *) die "无效选择：$c" ;;
  esac

  cat <<MENU

[3/3] Golang 模块代理（GOPROXY）
  1) official  不设置（走 proxy.golang.org）
  2) goproxycn https://goproxy.cn,direct          (推荐)
MENU
  read -rp "选择 [1-2]，默认 2： " c
  case "${c:-2}" in
    1) GOLANG_CHOICE=official ;;
    2) GOLANG_CHOICE=goproxycn ;;
    *) die "无效选择：$c" ;;
  esac
  echo
fi

# ============================================================================
# 1. Docker registry-mirrors
# ============================================================================
configure_docker() {
  local choice="$1"
  [ -z "$choice" ] && { log "Docker：未指定 --docker，跳过"; return; }
  command -v docker >/dev/null 2>&1 || { warn "Docker：未检测到 docker，跳过本项"; return; }

  local urls=""
  case "$choice" in
    official) urls="" ;;
    daocloud) urls="https://docker.m.daocloud.io" ;;
    xuanyuan) urls="https://docker.xuanyuan.me" ;;
  esac
  log "Docker：目标 = $choice ${urls:+($urls)}"

  mkdir -p /etc/docker
  [ -f "$DAEMON_JSON" ] && cp -a "$DAEMON_JSON" "$DAEMON_JSON.bak.$(date +%Y%m%d%H%M%S)"

  local OLD_HASH=""
  [ -f "$DAEMON_JSON" ] && OLD_HASH="$(sha256sum "$DAEMON_JSON" | cut -d' ' -f1)"

  if command -v python3 >/dev/null 2>&1; then
    python3 - "$DAEMON_JSON" "$urls" <<'PYEOF'
import json, os, sys
p, urls = sys.argv[1], sys.argv[2]
data = {}
if os.path.exists(p) and os.path.getsize(p) > 0:
    try:
        data = json.load(open(p))
    except json.JSONDecodeError:
        data = {}
if urls.strip():
    data['registry-mirrors'] = [u.strip() for u in urls.split(',') if u.strip()]
else:
    data.pop('registry-mirrors', None)
with open(p, 'w') as f:
    json.dump(data, f, indent=2, ensure_ascii=False)
    f.write('\n')
PYEOF
  else
    # 降级：无 python3 时只在 daemon.json 不存在/空文件场景生成最小配置
    if [ -s "$DAEMON_JSON" ]; then
      warn "Docker：未装 python3 且 daemon.json 已有内容；请手动编辑添加 registry-mirrors，
            然后 systemctl restart docker"
      return
    fi
    if [ -z "$urls" ]; then
      : > "$DAEMON_JSON"
    else
      printf '{\n  "registry-mirrors": ["%s"]\n}\n' "$urls" > "$DAEMON_JSON"
    fi
  fi

  local NEW_HASH=""
  [ -f "$DAEMON_JSON" ] && NEW_HASH="$(sha256sum "$DAEMON_JSON" | cut -d' ' -f1)"
  log "Docker：$DAEMON_JSON 当前内容："
  sed 's/^/    /' "$DAEMON_JSON" 2>/dev/null || echo "    (空文件)"

  if [ "$OLD_HASH" = "$NEW_HASH" ]; then
    log "Docker：配置未变化，跳过 docker 重启"
  elif systemctl is-active --quiet docker 2>/dev/null; then
    log "Docker：重启 docker 让加速配置生效 ..."
    systemctl restart docker
    sleep 1
  else
    log "Docker：当前未运行，启动它 ..."
    systemctl start docker || warn "Docker：启动失败，请手动 systemctl status docker"
  fi
}

# ============================================================================
# 2. npm registry
# ============================================================================
configure_npm() {
  local choice="$1"
  [ -z "$choice" ] && { log "npm：未指定 --npm，跳过"; return; }
  log "npm：目标 = $choice"

  mkdir -p "$(dirname "$NPMRC")"
  [ -f "$NPMRC" ] && cp -a "$NPMRC" "$NPMRC.bak.$(date +%Y%m%d%H%M%S)"

  # 先删除既有 registry= / disturl= 行（保留其它）
  if [ -f "$NPMRC" ]; then
    sed -i.tmp -e '/^registry=/d' -e '/^disturl=/d' "$NPMRC"
    rm -f "$NPMRC.tmp"
  else
    : > "$NPMRC"
  fi

  case "$choice" in
    official) ;; # 已清空 registry= 即官方
    taobao)
      cat >> "$NPMRC" <<EOF
registry=https://registry.npmmirror.com
disturl=https://npmmirror.com/mirrors/node
EOF
      ;;
  esac
  log "npm：$NPMRC 内容："
  if [ -s "$NPMRC" ]; then
    sed 's/^/    /' "$NPMRC"
  else
    echo "    (空文件 — 即使用官方 registry.npmjs.org)"
  fi
  log "npm：当前用户若已有 ~/.npmrc 中 registry= 会**优先**生效，请按需同步修改"
}

# ============================================================================
# 3. Golang GOPROXY
# ============================================================================
configure_golang() {
  local choice="$1"
  [ -z "$choice" ] && { log "Golang：未指定 --golang，跳过"; return; }
  log "Golang：目标 = $choice"

  case "$choice" in
    official)
      if [ -f "$GOPROXY_FILE" ]; then
        cp -a "$GOPROXY_FILE" "$GOPROXY_FILE.bak.$(date +%Y%m%d%H%M%S)"
        rm -f "$GOPROXY_FILE"
        log "Golang：已删除 $GOPROXY_FILE（回归官方 proxy.golang.org）"
      else
        log "Golang：$GOPROXY_FILE 不存在，无需操作"
      fi
      ;;
    goproxycn)
      [ -f "$GOPROXY_FILE" ] && cp -a "$GOPROXY_FILE" "$GOPROXY_FILE.bak.$(date +%Y%m%d%H%M%S)"
      cat > "$GOPROXY_FILE" <<'EOF'
# 由 setup-mirrors.sh 生成 — Golang 模块代理（七牛 goproxy.cn）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
EOF
      chmod 0644 "$GOPROXY_FILE"
      log "Golang：写入 $GOPROXY_FILE："
      sed 's/^/    /' "$GOPROXY_FILE"
      log "Golang：**新登录的 shell 才生效**；当前 shell 立即生效请运行："
      log "          source $GOPROXY_FILE   # 或：go env -w GOPROXY=https://goproxy.cn,direct"
      ;;
  esac
}

# ── 顺序应用 3 项 ───────────────────────────────────────────────────────
configure_docker  "$DOCKER_CHOICE"
configure_npm     "$NPM_CHOICE"
configure_golang  "$GOLANG_CHOICE"

echo
log "完成。下次可单独运行：sudo bash $0 --show 查看当前 3 项配置。"
