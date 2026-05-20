#!/usr/bin/env bash
# ubuntu-dev-bootstrap.sh
# Ubuntu 开发环境一键初始化：git + GitHub SSH key + voidint/g + nvm
#
# 适配中国大陆服务器：自动检测 GitHub 直连，不通时切到镜像代理；
# 同时为 git / SSH / Go / Node 配置中国镜像源。幂等可重跑。
#
# 用法：
#   bash ubuntu-dev-bootstrap.sh
#   # 自定义：
#   GIT_USER_NAME="Your Name" GIT_USER_EMAIL="you@example.com" \
#     bash ubuntu-dev-bootstrap.sh
#
# 可选环境变量：
#   GIT_USER_NAME / GIT_USER_EMAIL  git 全局 user.name / user.email
#   SSH_KEY_PATH                    SSH 私钥路径（默认 ~/.ssh/id_ed25519_github）
#   SSH_KEY_PASSPHRASE              SSH 私钥密码短语（默认空，便于非交互）
#   NVM_VERSION                     nvm 版本（默认 v0.40.1）
#   NODE_LTS_INSTALL                安装 nvm 后是否 install --lts（1/0，默认 1）
#   GH_PROXY                        首选 GitHub 代理（默认 https://gh-proxy.com）
#   FORCE_PROXY                     1=强制走代理，0=自动探测（默认 0）

set -euo pipefail

# ─── 配置项 ──────────────────────────────────────────────────────────────
GIT_USER_NAME="${GIT_USER_NAME:-}"
GIT_USER_EMAIL="${GIT_USER_EMAIL:-}"
SSH_KEY_PATH="${SSH_KEY_PATH:-$HOME/.ssh/id_ed25519_github}"
SSH_KEY_PASSPHRASE="${SSH_KEY_PASSPHRASE:-}"
NVM_VERSION="${NVM_VERSION:-v0.40.1}"
NODE_LTS_INSTALL="${NODE_LTS_INSTALL:-1}"
GH_PROXY="${GH_PROXY:-https://gh-proxy.com}"
FORCE_PROXY="${FORCE_PROXY:-0}"

# gh-proxy.com 偶发不稳定，备选若干（按顺序探测）
GH_PROXY_CANDIDATES=(
  "${GH_PROXY}"
  "https://ghfast.top"
  "https://ghproxy.net"
  "https://mirror.ghproxy.com"
)
# Go / Node 中国镜像
G_MIRROR_CN="https://golang.google.cn/dl/"
NVM_NODE_MIRROR="https://npmmirror.com/mirrors/node/"

USE_PROXY=0
PUB_KEY=""

# ─── 日志 ────────────────────────────────────────────────────────────────
log()   { printf '\033[0;32m[INFO]\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m[WARN]\033[0m %s\n' "$*"; }
error() { printf '\033[0;31m[ERR ]\033[0m %s\n' "$*" >&2; }

# ─── 基础设施 ────────────────────────────────────────────────────────────
require_ubuntu() {
  if ! command -v apt-get >/dev/null 2>&1; then
    error "此脚本仅支持 Debian/Ubuntu (apt-get)"
    exit 1
  fi
}

APT() {
  if [[ $EUID -eq 0 ]]; then
    DEBIAN_FRONTEND=noninteractive apt-get "$@"
  else
    sudo DEBIAN_FRONTEND=noninteractive apt-get "$@"
  fi
}

check_url() { curl -fsS --max-time 6 -o /dev/null "$1" 2>/dev/null; }

# ─── 1. 网络探测 ────────────────────────────────────────────────────────
detect_network() {
  if [[ $FORCE_PROXY -eq 1 ]]; then
    warn "FORCE_PROXY=1 → 跳过探测，强制走代理"
    USE_PROXY=1
  elif check_url https://github.com \
    && check_url https://raw.githubusercontent.com/voidint/g/master/README.md; then
    log "GitHub / raw 直连可达"
    return
  else
    warn "GitHub 直连不通，进入代理选择"
    USE_PROXY=1
  fi

  # 在候选中挑第一个能拉到 raw 资源的
  local probe="https://raw.githubusercontent.com/voidint/g/master/README.md"
  for cand in "${GH_PROXY_CANDIDATES[@]}"; do
    if check_url "${cand}/${probe}"; then
      GH_PROXY="$cand"
      log "选用代理：$GH_PROXY"
      return
    fi
  done
  warn "所有候选代理均不可达；继续尝试 $GH_PROXY（后续步骤可能失败）"
}

proxy_url() {
  local url="$1"
  if [[ $USE_PROXY -eq 1 ]]; then
    echo "${GH_PROXY}/${url}"
  else
    echo "$url"
  fi
}

# ─── 2. git ──────────────────────────────────────────────────────────────
install_git() {
  if command -v git >/dev/null 2>&1; then
    log "git 已存在：$(git --version)"
    return
  fi
  log "apt-get install git curl ca-certificates openssh-client"
  # apt-get update 在私有/本地 repo 损坏时会以 exit=100 失败（典型：内网源
  # Packages 缺失、_apt 用户读不到 /home 下的本地 repo）。这种 update 失败
  # 不应致命——缓存里通常已经有 git/curl，install 仍能成功；只有 install
  # 真的拿不到包才视为硬错误。
  APT update -y \
    || warn "apt-get update 部分源失败（多见于本地/私有 repo 损坏），继续使用缓存"
  APT install -y git curl ca-certificates openssh-client
  log "git 安装完成：$(git --version)"
}

configure_git() {
  [[ -n $GIT_USER_NAME  ]] && git config --global user.name  "$GIT_USER_NAME"
  [[ -n $GIT_USER_EMAIL ]] && git config --global user.email "$GIT_USER_EMAIL"
  git config --global init.defaultBranch main
  git config --global pull.rebase true

  # 清掉之前可能写入的代理 insteadOf（重跑场景）
  git config --global --unset-all url."https://gh-proxy.com/https://github.com/".insteadOf 2>/dev/null || true
  git config --global --unset-all url."https://gh-proxy.com/https://raw.githubusercontent.com/".insteadOf 2>/dev/null || true
  git config --global --unset-all url."https://gh-proxy.com/https://codeload.github.com/".insteadOf 2>/dev/null || true

  if [[ $USE_PROXY -eq 1 ]]; then
    log "配置 git insteadOf → $GH_PROXY/"
    git config --global url."${GH_PROXY}/https://github.com/".insteadOf            "https://github.com/"
    git config --global url."${GH_PROXY}/https://raw.githubusercontent.com/".insteadOf "https://raw.githubusercontent.com/"
    git config --global url."${GH_PROXY}/https://codeload.github.com/".insteadOf   "https://codeload.github.com/"
  fi
}

# ─── 3. SSH 密钥 ─────────────────────────────────────────────────────────
generate_ssh_key() {
  mkdir -p "$HOME/.ssh"
  chmod 700 "$HOME/.ssh"

  if [[ -f $SSH_KEY_PATH ]]; then
    log "SSH 私钥已存在：$SSH_KEY_PATH（跳过生成）"
  else
    log "生成 ed25519 SSH key → $SSH_KEY_PATH"
    ssh-keygen -t ed25519 \
      -C "$(whoami)@$(hostname)-$(date +%Y%m%d)" \
      -f "$SSH_KEY_PATH" \
      -N "$SSH_KEY_PASSPHRASE"
  fi

  local cfg="$HOME/.ssh/config"
  touch "$cfg"
  chmod 600 "$cfg"

  if ! grep -qE "^Host github\.com\$" "$cfg" 2>/dev/null; then
    # 探测 22 端口；不通则走 ssh.github.com:443
    local ssh_host="github.com" ssh_port=22
    if ! timeout 5 bash -c '</dev/tcp/github.com/22' 2>/dev/null; then
      warn "github.com:22 不通，改用 ssh.github.com:443"
      ssh_host="ssh.github.com"
      ssh_port=443
    fi
    cat >> "$cfg" <<EOF

Host github.com
  HostName ${ssh_host}
  Port ${ssh_port}
  User git
  IdentityFile ${SSH_KEY_PATH}
  IdentitiesOnly yes
  ServerAliveInterval 60
EOF
    log "已追加 ~/.ssh/config: Host github.com → ${ssh_host}:${ssh_port}"
  else
    log "~/.ssh/config 中已存在 Host github.com（保留不变）"
  fi

  PUB_KEY=$(cat "${SSH_KEY_PATH}.pub")
}

# ─── 4. voidint/g（Go 多版本管理） ──────────────────────────────────────
install_g() {
  if command -v g >/dev/null 2>&1; then
    log "g 已存在：$(g version 2>/dev/null | head -1 || echo '?')"
    return
  fi
  log "安装 voidint/g …"
  # g 的 install.sh 读 GHPROXY 给 release 二进制下载用
  export GHPROXY="$([[ $USE_PROXY -eq 1 ]] && echo "$GH_PROXY" || true)"
  export G_MIRROR="${G_MIRROR:-$G_MIRROR_CN}"

  local installer
  installer=$(proxy_url "https://raw.githubusercontent.com/voidint/g/master/install.sh")
  if ! curl -fsSL "$installer" | bash; then
    error "g 安装失败：检查 $GH_PROXY 或参考 https://github.com/voidint/g"
    return 1
  fi
  # shellcheck disable=SC1091
  [[ -f $HOME/.g/env ]] && . "$HOME/.g/env" || true
  log "g 安装完成（GHPROXY=${GHPROXY:-直连}, G_MIRROR=${G_MIRROR}）"
}

# ─── 5. nvm ──────────────────────────────────────────────────────────────
install_nvm() {
  export NVM_DIR="$HOME/.nvm"
  if [[ -s $NVM_DIR/nvm.sh ]]; then
    log "nvm 已存在"
  else
    log "安装 nvm $NVM_VERSION …"
    # nvm install.sh 默认 git clone https://github.com/nvm-sh/nvm
    # → 已被 git insteadOf 透出，无需 NVM_SOURCE 额外指定
    local installer
    installer=$(proxy_url "https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_VERSION}/install.sh")
    if ! curl -fsSL "$installer" | bash; then
      error "nvm 安装失败：检查 $GH_PROXY 或参考 https://github.com/nvm-sh/nvm"
      return 1
    fi
  fi
  # shellcheck disable=SC1091
  [[ -s $NVM_DIR/nvm.sh ]] && . "$NVM_DIR/nvm.sh" || true

  if [[ $USE_PROXY -eq 1 ]]; then
    local rc="$HOME/.bashrc"
    if ! grep -q "NVM_NODEJS_ORG_MIRROR" "$rc" 2>/dev/null; then
      {
        echo ""
        echo "# 由 ubuntu-dev-bootstrap.sh 添加 — Node 中国镜像"
        echo "export NVM_NODEJS_ORG_MIRROR=${NVM_NODE_MIRROR}"
      } >> "$rc"
      log "写入 $rc：NVM_NODEJS_ORG_MIRROR=$NVM_NODE_MIRROR"
    fi
    export NVM_NODEJS_ORG_MIRROR="$NVM_NODE_MIRROR"
  fi

  if [[ $NODE_LTS_INSTALL -eq 1 ]] && command -v nvm >/dev/null 2>&1; then
    log "nvm install --lts …"
    nvm install --lts || warn "nvm install --lts 失败，可重启 shell 后手动重试"
  fi
}

# ─── 6. 总结 ─────────────────────────────────────────────────────────────
summary() {
  echo
  log "═══════════════════ 安装完成 ═══════════════════"
  log "git: $(git --version 2>/dev/null || echo 未安装)"
  log "g:   $(command -v g >/dev/null && (g version 2>/dev/null | head -1) || echo '未生效（重启 shell 后可用）')"
  log "nvm: $([[ -s $HOME/.nvm/nvm.sh ]] && echo 已安装 || echo 未安装)"
  log "代理: $([[ $USE_PROXY -eq 1 ]] && echo "$GH_PROXY" || echo '直连')"
  log "SSH 私钥: $SSH_KEY_PATH"
  echo
  warn "请手动完成："
  warn "  ① 复制下方公钥到 GitHub → Settings → SSH and GPG keys → New SSH key"
  printf '\n%s\n\n' "$PUB_KEY"
  warn "  ② 验证连接：ssh -T git@github.com  （首次会问 yes/no）"
  warn "  ③ 重启 shell 或：source ~/.bashrc  以加载 g / nvm 环境变量"
  warn "  ④ HTTPS push 如需 PAT：到 https://github.com/settings/tokens 自助生成"
}

main() {
  require_ubuntu
  detect_network
  install_git
  configure_git
  generate_ssh_key
  install_g
  install_nvm
  summary
}

main "$@"
