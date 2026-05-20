#!/usr/bin/env bash
# =============================================================================
# Docker 加速镜像（registry-mirrors）配置
#
# 配置 /etc/docker/daemon.json 的 registry-mirrors 字段并 restart docker。
# 不下载任何东西，只改本地 daemon 配置。
#
# 设计来源：docs/design/deployments-release-enhancements-20260520.md §3.1
#
# 用法：
#   sudo bash setup-docker-mirror.sh                          # 交互式选单
#   sudo bash setup-docker-mirror.sh --mirror aliyun          # 非交互（推荐脚本里调）
#   sudo bash setup-docker-mirror.sh --mirror custom --url https://my-mirror.example.com
#   sudo bash setup-docker-mirror.sh --remove                 # 取消加速（回归官方）
#   sudo bash setup-docker-mirror.sh --show                   # 仅展示当前配置
#   sudo bash setup-docker-mirror.sh -h | --help              # 本帮助
#
# 参数：
#   --mirror <name>   加速器名称：official / aliyun / tencent / ustc / netease / baidu / custom
#                     · official：清空 registry-mirrors，回归 docker hub 官方
#                     · custom：必须配合 --url <URL>（可多次，逗号分隔多 URL）
#   --url <URL>       仅 --mirror custom 时使用，自定义加速 URL
#   --remove          删除 registry-mirrors 字段（等价 --mirror official）
#   --show            仅展示当前 /etc/docker/daemon.json 的 registry-mirrors，不修改
#   -h | --help       本帮助
#
# 内置加速 URL（每项之后会被原样写入 daemon.json）：
#   aliyun   → https://registry.aliyuncs.com  +  https://hub-mirror.c.163.com（双备份）
#   tencent  → https://mirror.ccs.tencentyun.com
#   ustc     → https://docker.mirrors.ustc.edu.cn
#   netease  → https://hub-mirror.c.163.com
#   baidu    → https://mirror.baidubce.com
#
# 安全性：
#   · 仅修改 /etc/docker/daemon.json 的 registry-mirrors 一项，其它键原样保留
#   · 修改前自动备份 daemon.json.bak.<时间戳>
#   · 仅在内容确有变化时才 systemctl restart docker，避免无谓抖动
# =============================================================================
set -euo pipefail

DAEMON_JSON="/etc/docker/daemon.json"

log()  { echo -e "\033[1;36m[mirror]\033[0m $*"; }
warn() { echo -e "\033[1;33m[mirror][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[mirror][错误]\033[0m $*" >&2; exit 1; }

# ── 帮助 ────────────────────────────────────────────────────────────────
show_help() { sed -n '3,38p' "$0"; exit 0; }

# ── 参数解析 ─────────────────────────────────────────────────────────────
MIRROR=""
URL=""
REMOVE=0
SHOW=0
while [ $# -gt 0 ]; do
  case "$1" in
    --mirror) MIRROR="$2"; shift 2 ;;
    --url)    URL="$2"; shift 2 ;;
    --remove) REMOVE=1; shift ;;
    --show)   SHOW=1; shift ;;
    -h|--help) show_help ;;
    *) die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# ── --show 仅展示，不需要 root ──────────────────────────────────────────
if [ "$SHOW" = 1 ]; then
  if [ ! -f "$DAEMON_JSON" ]; then
    log "$DAEMON_JSON 不存在（当前未配置任何 daemon 选项）"
    exit 0
  fi
  log "当前 $DAEMON_JSON："
  cat "$DAEMON_JSON"
  exit 0
fi

# ── 其它操作需要 root ───────────────────────────────────────────────────
[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

# ── Docker 必须已安装 ───────────────────────────────────────────────────
command -v docker >/dev/null 2>&1 || die "未检测到 docker。请先安装 Docker（参考 install-docker.sh）。"

# ── 加速器名 → URL 列表 ─────────────────────────────────────────────────
mirror_urls() {
  case "$1" in
    official) echo "" ;;
    aliyun)   echo "https://registry.aliyuncs.com,https://hub-mirror.c.163.com" ;;
    tencent)  echo "https://mirror.ccs.tencentyun.com" ;;
    ustc)     echo "https://docker.mirrors.ustc.edu.cn" ;;
    netease)  echo "https://hub-mirror.c.163.com" ;;
    baidu)    echo "https://mirror.baidubce.com" ;;
    custom)   echo "$URL" ;;
    *) die "未知加速器：$1（应为 official/aliyun/tencent/ustc/netease/baidu/custom）" ;;
  esac
}

# ── 交互选单（无 --mirror、无 --remove 时）───────────────────────────────
if [ -z "$MIRROR" ] && [ "$REMOVE" = 0 ]; then
  cat <<MENU
─────────────────────────────────────────────────
 请选择 Docker 加速镜像（输入数字）：
   1) official  官方源（不配置加速）
   2) aliyun    阿里云（推荐 / 国内默认）
   3) tencent   腾讯云
   4) ustc      中国科学技术大学
   5) netease   网易
   6) baidu     百度云
   7) custom    自定义 URL
─────────────────────────────────────────────────
MENU
  read -rp "选择 [1-7]，默认 2： " choice
  case "${choice:-2}" in
    1) MIRROR=official ;;
    2) MIRROR=aliyun ;;
    3) MIRROR=tencent ;;
    4) MIRROR=ustc ;;
    5) MIRROR=netease ;;
    6) MIRROR=baidu ;;
    7) MIRROR=custom
       read -rp "请输入自定义加速 URL（多个用逗号分隔）： " URL
       [ -n "$URL" ] || die "custom 模式必须输入 URL"
       ;;
    *) die "无效选择：$choice" ;;
  esac
fi

[ "$REMOVE" = 1 ] && MIRROR="official"
[ "$MIRROR" = "custom" ] && [ -z "$URL" ] && die "--mirror custom 必须配合 --url <URL>"

URLS="$(mirror_urls "$MIRROR")"
log "目标加速器：$MIRROR ${URLS:+($URLS)}"

# ── 合并 daemon.json：保留其它键，仅改 registry-mirrors ──────────────────
mkdir -p /etc/docker
[ -f "$DAEMON_JSON" ] && cp -a "$DAEMON_JSON" "$DAEMON_JSON.bak.$(date +%Y%m%d%H%M%S)"

write_with_python() {
  python3 - "$DAEMON_JSON" "$URLS" <<'PYEOF'
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
}

write_without_python() {
  # 降级：仅在 daemon.json 不存在或为空时写一份只含 registry-mirrors 的最小文件
  # 已存在其它键时拒绝写入，引导用户手动 edit。
  if [ -s "$DAEMON_JSON" ]; then
    die "未安装 python3，且 $DAEMON_JSON 已有自定义内容。请手动编辑添加：
      \"registry-mirrors\": [\"${URLS//,/\",\"}\"]
    然后执行：systemctl restart docker"
  fi
  if [ -z "$URLS" ]; then
    : > "$DAEMON_JSON"
    return
  fi
  {
    echo "{"
    local first=1 u
    echo -n '  "registry-mirrors": ['
    IFS=, read -ra ARR <<<"$URLS"
    for u in "${ARR[@]}"; do
      [ -n "$u" ] || continue
      [ "$first" = 1 ] || echo -n ", "
      echo -n "\"$u\""
      first=0
    done
    echo "]"
    echo "}"
  } > "$DAEMON_JSON"
}

OLD_HASH=""
[ -f "$DAEMON_JSON" ] && OLD_HASH="$(sha256sum "$DAEMON_JSON" | cut -d' ' -f1)"

if command -v python3 >/dev/null 2>&1; then
  write_with_python
else
  warn "未安装 python3，使用降级写入路径（仅在 daemon.json 为空时可用）"
  write_without_python
fi

NEW_HASH=""
[ -f "$DAEMON_JSON" ] && NEW_HASH="$(sha256sum "$DAEMON_JSON" | cut -d' ' -f1)"

log "$DAEMON_JSON 内容："
cat "$DAEMON_JSON" 2>/dev/null || echo "(空文件)"

# ── 仅在内容确变 / 服务正在跑时 restart ─────────────────────────────────
if [ "$OLD_HASH" = "$NEW_HASH" ]; then
  log "配置未变化，跳过 docker 重启"
else
  if systemctl is-active --quiet docker 2>/dev/null; then
    log "重启 docker 让加速配置生效 ..."
    systemctl restart docker
    sleep 1
    docker info 2>/dev/null | grep -A 5 -i "registry mirror" || true
  else
    log "docker 当前未运行，启动它 ..."
    systemctl start docker || warn "docker 启动失败，请手动 systemctl status docker"
  fi
fi

log "完成。当前生效的加速镜像："
docker info 2>/dev/null | awk '/Registry Mirrors/{flag=1; next} flag && /^[[:space:]]/ {print "  " $0; next} flag {exit}' \
  || echo "  (未配置任何加速镜像 = 走 docker.io 官方)"
