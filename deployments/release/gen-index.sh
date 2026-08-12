#!/usr/bin/env bash
# =============================================================================
# OMC 离线版本下载索引生成
#
# 扫描 archive/project/ 与 archive/infra/ 下的版本目录，生成 archive/index.html
# —— 8000 端口下载页：项目交付包、基础设施下载各一张版本列表，并附完整交付侧
# 操作手册（文件清单 / 校验 / 安装 / 加速镜像 / 部署 / 验证 / 访问 / 账号）。
#
# 由 build-release.sh / build-images.sh 在归档后自动调用；serve.sh 启动时也会
# 调一次，确保索引与 archive/ 实际内容一致。也可手动运行刷新。
#
# 用法：
#   ./gen-index.sh                          # 默认 archive/
#   ./gen-index.sh --archive <dir>          # 自定义 archive 目录（测试用）
#   ./gen-index.sh -h | --help              # 本帮助
#
# 参数：
#   --archive <dir>   归档目录（默认 deployments/release/archive）
#   -h, --help        本帮助
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARCHIVE="$SCRIPT_DIR/archive"

while [ $# -gt 0 ]; do
  case "$1" in
    --archive) ARCHIVE="$2"; shift 2 ;;
    -h|--help) sed -n '3,21p' "$0"; exit 0 ;;
    *)         echo "未知参数：$1（-h 查看用法）" >&2; exit 1 ;;
  esac
done

mkdir -p "$ARCHIVE"

# 扫描一类交付包目录，输出 HTML 表格行（按目录 mtime 倒序）。
#   $1 = 版本目录根（archive/project 或 archive/infra）
#   $2 = 类型（project|infra）—— 决定列结构与下载链接前缀
scan_rows() {
  local root="$1" kind="$2" d v rel rows=""
  [ -d "$root" ] || return 0
  for d in $(ls -dt "$root"/*/ 2>/dev/null); do
    v="$(basename "$d")"
    rel="${d}RELEASE.txt"
    [ -f "$rel" ] || continue
    local bt links="" f bn sz
    bt="$(grep -E '^build_time=' "$rel" | cut -d= -f2- || true)"
    # 历史项目包曾写入 UTC 的 Z；索引统一按构建机当地时区展示，避免项目与
    # 基础设施列表出现一列两种时区格式。新包已直接写入带 %z 的本地时间。
    if [[ "$bt" == *Z ]]; then
      bt_epoch=""
      bt_local=""
      if bt_epoch="$(date -d "$bt" +%s 2>/dev/null)" ||
         bt_epoch="$(date -j -f '%Y-%m-%dT%H:%M:%SZ' "$bt" +%s 2>/dev/null)"; then
        if bt_local="$(date -d "@$bt_epoch" '+%Y-%m-%dT%H:%M:%S%z' 2>/dev/null)" ||
           bt_local="$(date -r "$bt_epoch" '+%Y-%m-%dT%H:%M:%S%z' 2>/dev/null)"; then
          bt="${bt_local:0:19}${bt_local:19:3}:${bt_local:22:2}"
        fi
      fi
    fi
    # 仅显示 amd64 包（release 工具链 amd64-only；历史 *-arm64.tar.* 物理保留但不进列表）
    # RELEASE.txt 仅作为构建系统内部元数据，不进下载列表
    for f in "$d"omc-*-amd64.tar.*; do
      [ -f "$f" ] || continue
      case "$f" in *.sha256) continue ;; esac
      bn="$(basename "$f")"
      sz="$(du -h "$f" | cut -f1)"
      links="${links}<a href=\"${kind}/${v}/${bn}\">${bn}</a> <span class=\"sz\">(${sz})</span>"
      if [ -f "${f}.sha256" ]; then
        links="${links} <a class=\"sha\" href=\"${kind}/${v}/${bn}.sha256\">[SHA256]</a>"
      fi
      links="${links}<br>"
    done
    links="${links%<br>}"
    if [ "$kind" = project ]; then
      local pv ch
      pv="$(grep -E '^project_version=' "$rel" | cut -d= -f2- || true)"
      ch="$(grep -E '^channel=' "$rel" | cut -d= -f2- || true)"
      rows="${rows}<tr><td>${pv:-$v}</td><td><span class=\"ch ch-${ch:-test}\">${ch:-test}</span></td><td>${bt}</td><td>${links}</td></tr>"
    else
      local iv dv
      iv="$(grep -E '^infra_version=' "$rel" | cut -d= -f2- || true)"
      dv="$(grep -E '^docker_version=' "$rel" | cut -d= -f2- || true)"
      rows="${rows}<tr><td>${iv:-$v}</td><td>${dv:-—}</td><td>${bt}</td><td>${links}</td></tr>"
    fi
  done
  printf '%s' "$rows"
}

PROJECT_ROWS="$(scan_rows "$ARCHIVE/project" project)"
INFRA_ROWS="$(scan_rows "$ARCHIVE/infra" infra)"
[ -n "$PROJECT_ROWS" ] || PROJECT_ROWS='<tr><td colspan="4" class="empty">暂无项目版本下载 —— 运行 ./build-release.sh 生成</td></tr>'
[ -n "$INFRA_ROWS" ]   || INFRA_ROWS='<tr><td colspan="4" class="empty">暂无基础设施下载 —— 运行 ./build-images.sh 生成</td></tr>'

# 检测非 amd64 历史归档（amd64-only refactor 之前留下的物理文件，已不进下载列表）
LEGACY_COUNT=$(find "$ARCHIVE" -type f -name 'omc-*-arm64.tar.*' ! -name '*.sha256' 2>/dev/null | wc -l | tr -d ' ')
LEGACY_NOTE=""
if [ "${LEGACY_COUNT:-0}" -gt 0 ]; then
  LEGACY_NOTE="<p class=\"note\">检测到 ${LEGACY_COUNT} 个非 amd64 历史归档（不在下载列表显示，物理文件仍在磁盘）。清理：<code>find archive/ -name 'omc-*-arm64.tar.*' -delete</code></p>"
fi

cat > "$ARCHIVE/index.html" <<HTML
<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>OMC 离线版本下载</title>
<style>
body{font-family:-apple-system,"Segoe UI",sans-serif;margin:0;color:#1f2937;background:#f5f6f8}
.wrap{max-width:1080px;margin:0 auto;padding:2rem 1.5rem}
h1{font-size:1.4rem;margin:0 0 .3rem}
h2{font-size:1.08rem;margin:1.8rem 0 .6rem;padding-left:.55rem;border-left:4px solid #1668dc}
h3{font-size:.95rem;margin:1rem 0 .35rem;color:#374151}
.lead{color:#6b7280;font-size:.88rem;margin:.2rem 0 1rem;line-height:1.6}
table{border-collapse:collapse;width:100%;background:#fff;box-shadow:0 1px 3px rgba(0,0,0,.08)}
th,td{border:1px solid #e5e7eb;padding:.55rem .75rem;text-align:left;font-size:.87rem;vertical-align:top}
th{background:#f0f2f5;font-weight:600}
a{color:#1668dc;text-decoration:none}a:hover{text-decoration:underline}
.sz{color:#9ca3af;font-size:.8rem}
a.sha{color:#6b7280;font-size:.75rem;font-family:ui-monospace,Menlo,monospace;margin-left:.25rem}
a.sha:hover{color:#1668dc}
.ch{display:inline-block;padding:.05rem .45rem;border-radius:3px;font-size:.78rem;font-weight:600}
.ch-test{background:#fff7e6;color:#d46b08}
.ch-release{background:#f6ffed;color:#389e0d}
.empty{color:#9ca3af;text-align:center}
code{background:#f0f2f5;padding:.1rem .35rem;border-radius:3px;font-size:.85em}
pre{background:#1f2937;color:#e5e7eb;padding:.7rem .9rem;border-radius:5px;overflow-x:auto;font-size:.82rem;margin:.4rem 0;line-height:1.5}
.note{color:#9ca3af;font-size:.8rem;margin-top:2.2rem}
.danger{background:#fef2f2;border-left:4px solid #dc2626;padding:.6rem .9rem;border-radius:4px;font-size:.85rem;color:#991b1b;margin:.5rem 0}
.tip{background:#f0f9ff;border-left:4px solid #0284c7;padding:.6rem .9rem;border-radius:4px;font-size:.85rem;color:#075985;margin:.5rem 0}
.kv{background:#fff;border:1px solid #e5e7eb;border-radius:6px;padding:.8rem 1rem;font-size:.88rem;line-height:1.8}
.kv b{display:inline-block;min-width:140px;color:#374151}
.tabs{display:flex;gap:.5rem;margin:.5rem 0 1rem}
.tab{flex:1;background:#fff;border:1px solid #e5e7eb;border-radius:6px;padding:.7rem .9rem;font-size:.88rem}
.tab b{display:block;color:#1668dc;margin-bottom:.3rem;font-size:.95rem}
ul.list{font-size:.88rem;line-height:1.8;margin:.3rem 0;padding-left:1.5rem}
.tab-nav{display:flex;gap:0;border-bottom:2px solid #e5e7eb;margin:1rem 0 1.5rem}
.tab-btn{padding:.6rem 1.2rem;cursor:pointer;border:none;background:none;font-size:.92rem;color:#6b7280;border-bottom:2px solid transparent;margin-bottom:-2px}
.tab-btn:hover{color:#1668dc}
.tab-btn.active{color:#1668dc;border-bottom-color:#1668dc;font-weight:600}
.tab-content{display:none}
.tab-content.active{display:block}
</style></head><body><div class="wrap">
<h1>OMC 离线版本下载</h1>
<p class="lead">内网离线部署交付包。<b>项目包</b>与<b>基础设施下载</b>相互独立、各自版本号：
首次部署两个都要下载；之后日常升级通常只需更新项目包。</p>

<nav class="tab-nav">
<button class="tab-btn active" data-tab="download">📦 下载</button>
<button class="tab-btn" data-tab="deploy">🚀 部署</button>
<button class="tab-btn" data-tab="config">⚙️ 配置</button>
<button class="tab-btn" data-tab="ops">🛠️ 运维</button>
</nav>

<div class="tab-content active" id="tab-download">
<h2>📋 你应该下载哪些文件？</h2>
<div class="tabs">
<div class="tab"><b>🆕 首次部署</b>
基础设施下载 <code>omc-infra-*.tar.xz</code> + <code>.sha256</code><br>
项目包 <code>omc-&lt;test|release&gt;-*.tar.xz</code> + <code>.sha256</code></div>
<div class="tab"><b>♻️ 日常升级</b>
仅项目包 <code>omc-&lt;test|release&gt;-*.tar.xz</code> + <code>.sha256</code><br>
<span class="sz">基础设施已部署、Docker 已装时，只更新项目包</span></div>
</div>

<h2>📦 项目版本下载</h2>
<p class="lead">OMC 二进制 + 前端 + 配置 + 数据库迁移 + 部署模板。发版频繁。
渠道：<span class="ch ch-test">test</span> 测试阶段　<span class="ch ch-release">release</span> 正式发布。</p>
<table><thead><tr><th>项目版本</th><th>渠道</th><th>构建时间</th><th>下载</th></tr></thead>
<tbody>$PROJECT_ROWS</tbody></table>

<h2>🛠️ 基础设施下载</h2>
<p class="lead">Docker 引擎离线安装包 + 基础镜像。不常变更，仅基础设施升级时更新。</p>
<table><thead><tr><th>基础设施版本</th><th>Docker 版本</th><th>构建时间</th><th>下载</th></tr></thead>
<tbody>$INFRA_ROWS</tbody></table>

$LEGACY_NOTE
</div>

<div class="tab-content" id="tab-deploy">
<h2>🧰 0. 系统准备(最小化系统必看)</h2>
<p class="lead">最小化安装的 Linux(尤其 Ubuntu Server / Debian netinst / RHEL minimal /
cloud-image)往往不带 <code>iptables</code>,Docker daemon 启动 bridge 驱动时会直接 panic:
<code>failed to start daemon: Error initializing network controller:
failed to register "bridge" driver: failed to create NAT chain DOCKER: iptables not found</code>。
请在装 Docker <b>之前</b>先执行对应发行版的命令补齐内核网络工具。</p>
<pre># Ubuntu / Debian
sudo apt update
sudo apt install -y iptables nftables bridge-utils

# RHEL / CentOS / Rocky / openEuler
sudo yum install -y iptables nftables bridge-utils
# 或 sudo dnf install -y iptables nftables bridge-utils</pre>
<p class="tip">标准 Server / Desktop ISO 装出来的系统通常已带这些工具,可
<code>which iptables nft brctl</code> 检查后再决定是否跳过本步。</p>

<h2>🔐 1. 校验完整性</h2>
<p class="tip">每个交付包对应 <b>2 个文件</b>:<code>.tar.xz</code>(产物) +
<code>.tar.xz.sha256</code>(校验和),两个都要下载;校验命令里的
<code>.sha256</code> 后缀不能省。</p>
<pre>sha256sum -c omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz.sha256
sha256sum -c omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz.sha256</pre>

<h2>📦 2. 解压交付包</h2>
<pre># 推荐目录布局：/opt/omc/infra/ 装一次；/opt/omc/releases/&lt;版本&gt;/ 按版本独立
sudo mkdir -p /opt/omc/infra /opt/omc/releases
sudo tar -xJf omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz -C /opt/omc/infra --strip-components=1
sudo tar -xJf omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz -C /opt/omc/releases</pre>

<h2>🐳 3. 安装 Docker（仅首次部署）</h2>
<p class="lead">目标机已装 Docker（<code>docker --version</code> 返 ≥ 20.10）时跳过本步。</p>

<h3>3.1 install-docker.sh 用法</h3>
<pre>cd /opt/omc/infra/docker
sudo bash install-docker.sh                         # 交互式:装完引导选加速镜像;/var &lt; 15G 询问切到 /home
sudo bash install-docker.sh --mirror daocloud       # 装完直接配 DaoCloud 加速(https://docker.m.daocloud.io)
sudo bash install-docker.sh --mirror xuanyuan       # 装完直接配轩辕加速(https://docker.xuanyuan.me)
sudo bash install-docker.sh --mirror official       # 装完不配镜像,回归 Docker Hub 官方
sudo bash install-docker.sh --no-mirror             # 装完不动 daemon.json,跳过加速引导
sudo bash install-docker.sh --skip-if-installed     # 已装 docker 时静默 0 退出(脚本里调用)
sudo bash install-docker.sh --uninstall             # 卸载 docker 引擎(dry-run,仅列 9 步计划;不动 OMC 业务数据)
sudo bash install-docker.sh --uninstall --force     # 真删:dockerd/二进制/systemd unit + apt/yum 系统包
sudo bash install-docker.sh --uninstall --force --keep-data  # 真删但保留数据目录,日后重装可复用镜像
sudo bash install-docker.sh -h                      # 查看所有参数</pre>
<p class="tip">install-docker.sh 自动:解压二进制 → 断言 docker0 网段(bip <code>173.17</code>/自动池 <code>173.19</code>,避开公司 <code>172</code> 内网)→ 写 containerd / docker 的 systemd 单元 → <code>enable --now</code> 开机自启 → 验证 → 引导加速镜像。<br>
<b>已装 docker 时</b>:跳过 dockerd 安装,但**仍补装** docker compose V2 + buildx plugin 到 <code>/usr/local/lib/docker/cli-plugins/</code>,解决系统 apt 装的 V1 Python compose 不识别 v3.x 写法问题;网段断言照跑。<br>
<b>卸载 docker 引擎</b>用 <code>--uninstall</code>(默认 dry-run,加 <code>--force</code> 真删,<code>--keep-data</code> 保留镜像数据);它只清 docker 本身,<b>不删</b> <code>/opt/omc</code> 等 OMC 业务数据 —— 先 <code>uninstall.sh</code> 再 <code>install-docker.sh --uninstall</code>。</p>

<h3>3.2 卸载 OMC（uninstall.sh）</h3>
<p class="lead"><code>uninstall.sh</code> 只负责卸载 OMC 业务栈，不卸载 Docker 引擎（如需卸引擎见 3.1 的 <code>install-docker.sh --uninstall</code>）。默认是 dry-run，仅列出计划；加 <code>--force</code> 才会实际执行，并会进行确认。</p>
<pre>cd /opt/omc/current/deploy

sudo bash uninstall.sh                              # dry-run：列出"保留数据"卸载计划
sudo bash uninstall.sh --force                      # 真卸载 OMC，保留数据卷 + 凭据（可重装复用）
sudo bash uninstall.sh --force --keep-images        # 同上但保留 omcgo/* 业务镜像
sudo bash uninstall.sh --purge                      # dry-run：列出"彻底清除"计划
sudo bash uninstall.sh --purge --force              # 彻底清除：删数据卷 + 整个 /opt/omc，不可恢复
sudo bash uninstall.sh --purge --force --yes        # 跳过二次确认（CI / 批处理）
sudo bash uninstall.sh --purge --force --keep-data  # 兼容旧命令；--keep-data 为 no-op，--purge 仍删数据</pre>
<p class="tip"><b>默认卸载：</b>删除 OMC 容器、网络、业务镜像和代码运行目录，保留数据库、MinIO、Redis、NATS、监控数据卷，以及 <code>/opt/omc/data</code>、<code>/opt/omc/etc</code> 和凭据，便于后续重新部署。<br>
<b>彻底清除：</b><code>--purge</code> 会连同所有 OMC 数据卷和整个 <code>/opt/omc</code> 一并删除，数据不可恢复。<code>--keep-data</code> 仅为兼容旧调用保留，不能覆盖 <code>--purge</code> 的删除行为。</p>

<div class="danger">⚠️ 若 <code>docker.service</code> 启动报
<code>failed to create NAT chain DOCKER: iptables not found</code>,
说明系统缺 iptables —— 回到 <b>0. 系统准备</b> 跑一遍 apt/yum 命令,然后
<code>sudo systemctl start docker</code> 即可继续。</div>

<h2>⚡ 4. 系统加速设置（可选）— Docker / npm / Golang 三合一</h2>
<p class="lead"><code>setup-mirrors.sh</code> 位于 infra 包顶层（非 Docker 专属），一次性配置 3 类加速器（每项可独立选择"不设置 = 走官方"）：</p>
<pre>cd /opt/omc/infra
sudo bash setup-mirrors.sh                          # 交互：逐项询问 3 项
sudo bash setup-mirrors.sh --docker daocloud --npm taobao --golang goproxycn   # 一气呵成
sudo bash setup-mirrors.sh --show                   # 看当前 3 项配置
sudo bash setup-mirrors.sh --remove                 # 全部取消，回归官方</pre>
<p class="tip">
推荐配置：<code>--docker daocloud</code>（<code>https://docker.m.daocloud.io</code>）／
<code>--npm taobao</code>（<code>https://registry.npmmirror.com</code>）／
<code>--golang goproxycn</code>（<code>https://goproxy.cn,direct</code>）。
每项 <code>official</code> = 不设置（走该工具官方源）。
</p>

<h2>📊 4.5 部署前：资源规划（plan-resources.sh — 必需）</h2>
<p class="lead">在跑 <code>install.sh</code> <b>之前</b>，先按目标服务器的<b>空闲资源</b>规划各容器的 CPU / 内存限额，生成 <code>deploy/resources.env</code>。
首次安装不运行会被预检拒绝；升级可继承上一版完整契约。脚本以 <code>MemAvailable</code> 为基准并扣除其它项目已占用 / 预留，避免超分压垮别的业务，也避免大机闲置或小机静默 OOM。</p>
<p><b>🆕 首次部署</b>（全新服务器）：在<b>解压出的交付包目录</b>跑（此时 <code>/opt/omc/current</code> 软链尚未创建——它由 install.sh 部署时才建，故首次<b>不能</b> <code>cd current/deploy</code>）：</p>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;      # 与下方 install.sh 同目录

bash deploy/plan-resources.sh --dry-run          # 只预览规划，不写文件（先看数字是否合理）
bash deploy/plan-resources.sh                     # 探测主机 + 计算 + 写 deploy/resources.env
bash deploy/plan-resources.sh --skip-monitoring   # 不部署监控栈时，降低最低配门槛
bash deploy/plan-resources.sh --tier medium       # 手动指定档位（默认按空闲内存自动判定 small/medium/large）
bash deploy/plan-resources.sh --assume-dedicated  # 本机 OMC 独占时，不扣其它容器预留
bash deploy/plan-resources.sh --floor-tolerance-pct 30  # 门禁容忍度（默认30，缺口在此百分比内降级WARN按下限分配）
bash deploy/plan-resources.sh -h                  # 全部参数</pre>
<p class="tip">随后 <code>sudo bash deploy/install.sh</code> 会把包内 <code>deploy/resources.env</code> 一并带入部署（拷进 <code>current/deploy/</code>）。首次安装缺少该文件会直接失败，不再回退 compose 内置默认限额。</p>
<p><b>⬆️ 升级部署</b>：<code>resources.env</code> 由 install.sh <b>自动从上一版继承</b>（<code>current/deploy/resources.env</code> 或 <code>etc/resources.env.saved</code>），<b>通常无需重跑</b>。仅当目标主机资源变化需<b>重新规划</b>时，在<b>新版本包目录</b>跑 <code>bash deploy/plan-resources.sh</code>（会覆盖继承值）。</p>
<p class="tip">脚本做三件事：① 探测 CPU / 内存 / 负载 / 其它容器占用；② 算「空闲预算」；③ <b>floor-first</b> 分配（每组件先发 100k 基线下限，剩余按权重分到上限）并<b>联动派生</b> <code>GOMEMLIMIT</code> / Postgres <code>shared_buffers·max_connections</code> / Redis <code>maxmemory</code>，写入带注释的 <code>resources.env</code>。主机低于最低配会<b>清晰报错并给建议最低配</b>（全栈约 ≥24 GiB，<code>--skip-monitoring</code> 约 20 GiB）。算法与档位详见交付包内 <code>deploy/RESOURCE-PLANNING.md</code>。</p>
<div class="tip">生成后请<b>检视 / 按需微调</b> <code>resources.env</code>，务必遵守文件头注释的约束：<code>GOMEMLIMIT &lt; *_MEM</code>、核心 Redis 保留 1GiB / PM Redis 保留 2GiB AOF COW 余量、<code>PG_MAX_CONNECTIONS ≥ Go 端连接池总和（当前 180）</code>。<br>
下游消费：<code>install.sh</code> 与 <code>svc.sh</code> 均以 <code>--env-file resources.env</code> 读取本文件，compose 用 <code>${VAR:-默认}</code> 套入限额。改完 <code>resources.env</code> 后跑 <code>bash svc.sh restart</code> 即按新限额有序重建生效。</div>

<h2>🚚 5. 一键部署 OMC</h2>
<p class="lead">所有场景都使用 <code>install.sh</code>，脚本会自动检测已有镜像并智能跳过重复加载；安装前必须存在完整 <code>resources.env</code>（见 4.5），脚本会在切换版本和重启容器前校验，并以 <code>--env-file</code> 套用其资源限额。所有示例均可加 <code>--lang cn</code> 切换中文安装提示（默认 <code>en</code>，亦可 <code>OMC_LANG=cn</code>）。</p>

<h3>场景 A：首次部署（全新服务器）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;
sudo bash deploy/install.sh</pre>
<p class="lead">全量执行：load 所有镜像 → 建目录 → 启动基础设施 → migrate/seed → 启动全栈。</p>

<h3>场景 B：升级业务版本（最常用）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;新版本&gt;-&lt;架构&gt;
sudo bash deploy/install.sh --skip-infra</pre>
<p class="lead">跳过基础设施镜像加载，仅加载新业务镜像 → 执行新迁移 → 重建业务容器（app/acs/worker/web），基础设施容器保持运行不受影响。</p>

<h3>场景 C：重复部署同版本（修复/重启）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;当前版本&gt;-&lt;架构&gt;
sudo bash deploy/install.sh</pre>
<p class="lead">脚本检测到所有镜像已存在 → 自动跳过 load → 重启容器 → 重跑 migrate（幂等）→ 健康检查。适用于服务异常需要完整重启的场景。</p>

<h3>场景 D：清理旧数据后全新部署（危险·不可恢复）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;
sudo bash deploy/install.sh --fresh-install --yes --public-host &lt;基站可达IP，如:11.22.33.44&gt;</pre>
<p class="lead">先停旧 OMC 栈 → <b>永久删除</b>所有 bind-mount 数据目录（PG/TSDB/Redis/Redis-PM/NATS/MinIO）+ <code>/opt/omc/{data,etc,current,run/logs}</code> + 项目全部 Docker volumes → 按本机重新规划 <code>resources.env</code>（见 4.5）→ 走正常首次安装流程。等效 <code>uninstall.sh --purge --force</code> 后再 <code>install.sh</code>，一步完成。</p>
<div class="danger">⚠️ <code>--fresh-install</code> 数据删除<b>不可恢复</b>。全新安装需要基站可达地址：通过 <code>--public-host</code> 传入，或 <code>deploy/.env</code> 已配有效的 <code>OMC_PUBLIC_HOST</code>（不能用 localhost/127.0.0.1，详 §9.5）；两者皆无则报错退出。省略 <code>--yes</code> 会在删除前二次交互确认。低内存主机可追加 <code>--skip-monitoring</code>，或 <code>--floor-tolerance-pct</code>（0-99，默认 60）放宽组件下限缺口容忍度。</div>

<h3>其他参数</h3>
<pre>sudo bash deploy/install.sh --check-only             # 仅检查环境，不动手
sudo bash deploy/install.sh --skip-migrate            # 不跑 migrate / seed
sudo bash deploy/install.sh --skip-monitoring         # 不起监控栈
sudo bash deploy/install.sh --lang cn                 # 中文安装提示（默认 en；亦可 OMC_LANG=cn）
sudo bash deploy/install.sh --overwrite-etc           # 用新包 etc/ 模板覆盖 /opt/omc/etc（旧自动备份）
sudo bash deploy/install.sh -h                        # 查看所有参数</pre>

<p class="tip">install.sh 自动：环境检查 → 目录布局 → 智能 load 镜像（已有则跳过并重启）→ 默认口令检查 → 启动 infra → 等就绪 → migrate → seed → <code>docker compose up -d</code> 全栈 → 健康检查。<b>全 docker compose 部署，宿主机不再放业务二进制。</b></p>
<div class="danger">🔐 凭证（#175 治本后）：首次安装 <code>install.sh</code> <b>自动生成强随机凭证</b>（PostgreSQL / MinIO / Grafana / JWT / TR-069 共享密钥）→ <code>/opt/omc/etc/secrets.env</code>（<code>600</code>/root；uninstall 保留、<code>--purge</code> 删，与数据卷同生命周期），<b>无需手工改默认口令</b>。MinIO / Grafana 登录口令在该文件，请<b>妥善备份</b>；<b>轮换 / 改口令</b>用 <code>deploy/reset_password.sh</code>（逐组件单选，见 §9.3）。<br>首次部署仍需手工填的只有 <code>OMC_PUBLIC_HOST</code>（基站可达 IP，见 §9.5）——在解压包的 <code>deploy/.env</code> 里填（此时 <code>current/deploy</code> 尚不存在）。</div>

<h3>5.4 svc.sh — 日常服务控制(部署完成后用)</h3>
<p class="lead">部署完成后,用 <code>svc.sh</code> 做日常启停 / 重启 / 查日志,无须再跑 install.sh。
脚本必须从 <code>/opt/omc/current/deploy/</code>(含 4 个 compose 文件那层)运行,自动按存在性拼 4 个 compose 文件,compose project 名固定 <code>omcgo</code>(与 install.sh 一致)。
不需要 root(除非 docker daemon 本身需 sudo)。</p>

<pre>cd /opt/omc/current/deploy

# ── 状态 / 启停 ──
bash svc.sh status                   # 查所有服务状态(默认子命令,等价 ps)
bash svc.sh start                    # 启全栈
bash svc.sh start app                # 只启 app(可跟多个服务名,如 app acs worker)
bash svc.sh stop                     # 停全栈
bash svc.sh restart                  # 重启全栈
bash svc.sh restart app worker       # 只重启指定服务
bash svc.sh up                       # 同 start,等价 docker compose up -d
bash svc.sh down                     # 关栈 + 清容器(保留数据卷)

# ── 日志 ──
bash svc.sh logs app                 # 看 app 最近 50 行
bash svc.sh logs app --tail 200      # 看 app 最近 200 行
bash svc.sh logs app -f              # 跟随 app 日志(Ctrl-C 退出)

# ── 选项(任意子命令前后皆可) ──
bash svc.sh restart --skip-monitoring        # 重启时不动监控栈
bash svc.sh status --skip-web                # 不算 web compose
bash svc.sh -h                                # 完整帮助</pre>
<p class="tip">常用服务名:<code>app</code> / <code>acs</code> / <code>acs-candidate</code> / <code>worker</code>(业务);<code>web</code>(web 层);<code>postgres</code> / <code>postgres-tsdb</code> / <code>redis-core</code> / <code>redis-pm</code> / <code>nats</code> / <code>minio</code>(基础设施);<code>prometheus</code> / <code>alertmanager</code> / <code>grafana</code> / <code>loki</code> / <code>tempo</code> / <code>otelcol</code> / <code>node-exporter</code> / <code>cadvisor</code> / <code>nats-exporter</code> / <code>nginx-exporter</code>(监控栈)。<br>
要看完整 compose ps 列表(默认全栈约 21 个容器;加 <code>--skip-monitoring</code> 约 11 个),直接跑 <code>bash svc.sh status</code>。</p>
<div class="tip">📊 <b>资源限额</b>:<code>svc.sh</code> 与 <code>install.sh</code> 一样会读取并校验同目录 <code>resources.env</code>(见 4.5)。改完该文件后,<code>bash svc.sh restart</code> 会按新限额有序重建容器(经 depends_on + 健康门控),无需重跑 install.sh。缺少或残缺 <code>resources.env</code> 时会拒绝重启，请先运行 <code>plan-resources.sh</code>。</div>

<h2>✅ 6. 验证部署</h2>
<pre>bash /opt/omc/current/deploy/healthcheck.sh</pre>
<p class="lead">应输出全部 <code>[OK]</code>：业务容器（app/acs/acs-candidate/worker）+ 基础设施容器（postgres/postgres-tsdb/redis-core/redis-pm/nats/minio）+ 监控容器（prometheus/grafana/loki/...）+ 5 个健康端点（app /healthz, acs /healthz, worker /healthz, app /metrics, 前端首页）。</p>
<p class="lead">脚本还会核对 Redis 业务路由隔离（redis-core / redis-pm 不同 run_id、prod.yaml 路由指向）、ACS 高并发网络参数（somaxconn / tcp_max_syn_backlog）、web nginx upstream 与临时端口范围、<code>OMC_PUBLIC_HOST</code> 在 app/acs/worker 容器内的实际取值，以及 <code>resources.env</code> 限额与容器渲染配置（GOMEMLIMIT / Redis maxmemory / PG shared_buffers·max_connections 等）是否一致——任一项漂移都会判 <code>[FAIL]</code>。</p>
</div>

<div class="tab-content" id="tab-config">
<h2>🌐 7. 部署后访问地址</h2>

<h3>7.1 内网可达(0.0.0.0 绑定 — 运营 / 客户 / 集成方使用)</h3>
<table>
<thead><tr><th>端口</th><th>用途</th><th>URL / 接入方式</th><th>使用方</th></tr></thead>
<tbody>
<tr><td><b>8081</b></td><td>Web 管理界面 + REST + SSE</td><td>http://&lt;服务器IP&gt;:8081</td><td>运维浏览器登录(默认 admin/admin123)</td></tr>
<tr><td><b>8080</b></td><td>基站连接(TR-069 ACS,nginx 反代 → ACS:7557)</td><td>http://&lt;服务器IP&gt;:8080</td><td>基站入口之一(等价 7557);人不浏览</td></tr>
<tr><td>7547</td><td>ACS CWMP 标准 Inform</td><td>基站填 <code>http://&lt;OMC_PUBLIC_HOST&gt;:7547</code></td><td><b>基站设备侧</b> CPE 发 TR-069 Inform 的标准端口</td></tr>
<tr><td>7557</td><td>ACS connection-request / 文件上传</td><td><code>http://&lt;OMC_PUBLIC_HOST&gt;:7557/smallcell/FileUploadService</code></td><td><b>基站设备侧</b>回传 PM/MR、ACS 主动触达(详 §9.5)</td></tr>
<tr><td>5432</td><td>PostgreSQL</td><td><code>psql -h &lt;IP&gt; -p 5432 -U omcgo omcgo</code></td><td>数据中台拉数据 / 备份回填 / 跨机调试</td></tr>
<tr><td>5433</td><td>TimescaleDB 时序库</td><td><code>psql -h &lt;IP&gt; -p 5433 -U omcgo omcgo</code></td><td>PM/KPI 超表查询(独立实例,避开主库 5432)</td></tr>
<tr><td>6379</td><td>Redis</td><td><code>redis-cli -h &lt;IP&gt; -p 6379</code></td><td>缓存监控 / 跨机调试</td></tr>
<tr><td>4222</td><td>NATS 客户端</td><td>nats CLI / SDK 连 <code>&lt;IP&gt;:4222</code></td><td>外部消费 JetStream / 跨机集成</td></tr>
<tr><td>9000</td><td>MinIO S3 API</td><td>mc / S3 SDK 连 <code>http://&lt;IP&gt;:9000</code></td><td>S3 客户端;Console 浏览器上传/下载也依赖此端口</td></tr>
<tr><td>9001</td><td>MinIO Console UI</td><td>http://&lt;服务器IP&gt;:9001</td><td>对象存储管理(默认 omcadmin,口令见 secrets.env)</td></tr>
<tr><td>9090</td><td>Prometheus</td><td>http://&lt;服务器IP&gt;:9090</td><td>指标查询 / 告警规则</td></tr>
<tr><td>9093</td><td>Alertmanager</td><td>http://&lt;服务器IP&gt;:9093</td><td>告警静默 / receiver 状态</td></tr>
<tr><td>3030</td><td>Grafana 监控大盘</td><td>http://&lt;服务器IP&gt;:3030 <span class="sz">仅启用监控栈时</span></td><td>默认 admin/admin,宿主 3030 → 容器 3000</td></tr>
<tr><td>3100</td><td>Loki</td><td>http://&lt;服务器IP&gt;:3100</td><td>日志 API,一般通过 Grafana 查询不直浏览</td></tr>
</tbody>
</table>
<div class="danger">⚠️ <b>5432 / 5433 / 6379 / 4222 / 9000 / 9001 对内网全开</b> — 部署前必须改强口令(详 §9)。Redis 当前无密码,仅受信任内网可接受;公网 / DMZ 须配 <code>requirepass</code> 同步 etc/*.prod.yaml。防火墙 / 安全组在出公网前必须 deny 这 6 个端口。</div>

<h3>7.2 仅本机回环 127.0.0.1(从工作机访问需 SSH 隧道)</h3>
<table>
<thead><tr><th>端口</th><th>服务</th><th>接入方式</th></tr></thead>
<tbody>
<tr><td>9091</td><td>app 健康 / metrics</td><td><code>curl 127.0.0.1:9091/healthz</code> 或 <code>/metrics</code></td></tr>
<tr><td>9095</td><td>acs 健康 / metrics</td><td><code>curl 127.0.0.1:9095/healthz</code>(容器内是 9090)</td></tr>
<tr><td>9092</td><td>worker 健康 / metrics</td><td><code>curl 127.0.0.1:9092/healthz</code></td></tr>
<tr><td>8222</td><td>NATS HTTP 监控</td><td>http://127.0.0.1:8222(server info / JetStream 状态)</td></tr>
<tr><td>13133</td><td>otelcol 健康探针</td><td><code>curl 127.0.0.1:13133/</code>(health extension)</td></tr>
</tbody>
</table>
<p class="tip">健康检查一键过:<code>bash /opt/omc/current/deploy/healthcheck.sh</code></p>

<h2>👤 8. 初始账号 / 口令</h2>
<div class="danger">⚠️ 全部默认口令<b>首次登录后必须改</b>。生产部署前需重新生成强口令并同步到 deploy/.env 与 *.prod.yaml。</div>
<div class="kv">
<b>Web 管理员：</b><code>admin</code> / <code>admin123</code><br>
<b>MinIO Console：</b><code>omcadmin</code> / <code>口令见 secrets.env(首次随机)</code><br>
<b>PostgreSQL：</b><code>omcgo</code> / <code>omcgo123</code><br>
<b>Grafana：</b><code>admin</code> / <code>admin</code>
</div>

<h2>🔑 9. 配置文件修改指南（账号 / 口令 / JWT）</h2>
<p class="lead">#175 治本后：凭证<b>唯一权威源</b>是 <code>/opt/omc/etc/secrets.env</code>（首次部署 <code>install.sh</code> 自动生成强随机，<code>600</code>/root）。<code>etc/*.prod.yaml</code> 已改用 <code>\${VAR}</code> 占位、从 <code>deploy/.env</code> 读取，<code>.env</code> 的 6 个密钥键由 <code>secrets.env</code> 自动同步覆盖——<b>无需再逐处手改口令</b>。改口令 / 轮换走 §9.3 的 <code>reset_password.sh</code>：</p>
<ul class="list">
<li><code>/opt/omc/etc/secrets.env</code> —— <b>唯一权威源</b>（7 键：PG 主库 / PG 时序库 / MinIO / Grafana / JWT / TR-069 共享密钥）</li>
<li><code>/opt/omc/current/deploy/.env</code> —— compose 起容器用；6 密钥键由 <code>secrets.env</code> 同步，<b>非密钥</b>键（镜像版本 / <code>OMC_PUBLIC_HOST</code>）在此手改</li>
<li><code>/opt/omc/etc/{app,acs,worker}.prod.yaml</code> —— OMC 三进程连接中间件；已是 <code>\${VAR}</code> 占位，<b>无口令可手改</b></li>
</ul>

<h3>9.1 凭证权威源与对应键</h3>
<p>所有口令唯一权威源是 <code>etc/secrets.env</code>。<b>下表「默认占位」仅历史值——首次部署已被 install.sh 随机值取代，不再是默认口令。</b>改口令 = 改 <code>secrets.env</code> 对应键（推荐用 §9.3 <code>reset_password.sh</code>），<code>*.prod.yaml</code> 走 <code>\${VAR}</code> 不需手改。</p>
<table>
<thead><tr><th>项</th><th>secrets.env 键</th><th>prod.yaml 引用 / 默认占位</th><th>说明</th></tr></thead>
<tbody>
<tr><td>PostgreSQL 口令（主库）</td><td><code>POSTGRES_PASSWORD</code></td><td><code>db.dsn</code> 的 <code>\${POSTGRES_PASSWORD}</code>（旧默认 <code>omcgo123</code>，首次已随机）</td><td>轮换：§9.3 <code>reset_password.sh pg</code></td></tr>
<tr><td>PostgreSQL 口令（时序库）</td><td><code>POSTGRES_TSDB_PASSWORD</code></td><td><code>tsdb.dsn</code> 的 <code>\${POSTGRES_TSDB_PASSWORD}</code>（#347，独立随机，不复用主库）</td><td>reset_password.sh 暂不轮换此键</td></tr>
<tr><td>PostgreSQL 账号 / 库</td><td>—（<code>.env</code> 的 <code>POSTGRES_USER</code>/<code>POSTGRES_DB</code>，非密钥）</td><td>DSN 账号 / <code>/库名</code></td><td>一般不改；reset_password.sh 不轮换账号</td></tr>
<tr><td>MinIO 账号</td><td><code>MINIO_ROOT_USER</code></td><td><code>minio.access_key</code>（首次 = <code>omcadmin</code>）</td><td><b>建议只改口令不改账号</b>（access_key 改名牵连引用）</td></tr>
<tr><td>MinIO 口令</td><td><code>MINIO_ROOT_PASSWORD</code></td><td><code>minio.secret_key</code>（旧默认 <code>minioadmin</code>，首次已随机）</td><td>轮换：§9.3 <code>reset_password.sh minio</code></td></tr>
<tr><td>JWT 密钥</td><td><code>OMCGO_JWT_SECRET</code></td><td><code>jwt.secret</code></td><td>≥32 字符；轮换后已签发 token 失效，需重登</td></tr>
<tr><td>TR-069 共享密钥</td><td><code>OMC_SHARED_SECRET</code></td><td>ACS ConnReq / STUN HMAC-SHA1</td><td>轮换需经 SetParameterValues 同步到基站，见 §9.3 告警</td></tr>
<tr><td>Grafana 管理员</td><td><code>GRAFANA_ADMIN_PASSWORD</code></td><td>—（容器 <code>GF_SECURITY_ADMIN_PASSWORD</code>，旧默认 <code>admin</code>，首次已随机）</td><td>仅监控栈；登录 :3030（宿主 3030 → 容器 3000）</td></tr>
<tr><td><b>基站可达地址</b></td><td>—（<code>.env</code> 的 <code>OMC_PUBLIC_HOST</code>，非密钥）</td><td>—（自动注入 app/acs/worker）</td><td><b>必填</b>：基站回传 PM 文件地址（<code>http://&lt;OMC_PUBLIC_HOST&gt;:7557/...</code>），不能用 localhost / 127.0.0.1，详见 §9.5</td></tr>
<tr><td>Web 管理员 admin</td><td>—</td><td>—</td><td>首次登录 <code>http://&lt;IP&gt;:8081</code> 在「个人中心 → 修改密码」改，<b>不动配置文件</b>（:8080 是 ACS 入口，人不要去登）</td></tr>
</tbody>
</table>

<h3>9.2 首次部署改口令（可选——默认已自动随机）</h3>
<p>#175 后<b>默认无需手工改口令</b>：<code>install.sh</code> 首次部署自动生成强随机凭证到 <code>secrets.env</code> 并同步 <code>.env</code>。仅当要用<b>自定义</b>口令时（首次 <code>install.sh</code> 起 infra 之前，直接写 <code>etc/secrets.env</code> 的 7 键，<code>chmod 600</code>；非密钥 <code>OMC_PUBLIC_HOST</code> 改解压包内 <code>deploy/.env</code>，见 §4.5/§9.5，再跑 <code>install.sh</code> 复用 secrets.env）。下方为底层等价步骤（仅参考）：</p>
<pre># 1) 生成强口令（示例）
openssl rand -base64 24    # PostgreSQL 口令
openssl rand -base64 24    # MinIO 口令
openssl rand -base64 24    # Grafana 口令
openssl rand -base64 48    # JWT 密钥

# 2) 改 deploy/.env（compose 初始口令 / 镜像版本变量）
sudo vi /opt/omc/current/deploy/.env
#     POSTGRES_PASSWORD=         → 刚生成的 PG 强口令
#     MINIO_ROOT_USER=           → 新账号（如仍用 minioadmin 则不改）
#     MINIO_ROOT_PASSWORD=       → 刚生成的 MinIO 强口令
#     GRAFANA_ADMIN_PASSWORD=    → 刚生成的 Grafana 口令
#     OMCGO_JWT_SECRET=          → 刚生成的 JWT 密钥
#     OMC_SHARED_SECRET=         → TR-069 共享密钥（≥32 字符）
#     POSTGRES_TSDB_PASSWORD=    → 时序库强口令（#347，独立于主库）
#     OMC_PUBLIC_HOST=           → 本机对外 IP（基站可达，如 172.19.1.132），必填，见 §9.5

# 3) （#175 后无需此步）etc/*.prod.yaml 已是 \${VAR} 占位、从 .env 读取，无明文口令可改；下面三行仅历史参考
sudo vi /opt/omc/etc/app.prod.yaml      # db.dsn / tsdb.dsn / minio.* / jwt.secret
sudo vi /opt/omc/etc/acs.prod.yaml      # db.dsn / minio.* （按需）
sudo vi /opt/omc/etc/worker.prod.yaml   # db.dsn / minio.* （按需）

# 4) 一键部署（自动 load 镜像 + up 全栈）
sudo bash /opt/omc/current/deploy/install.sh</pre>

<div class="danger">⚠️ <b>volume 已创建后，仅改 .env / secrets.env 不会改 PG / MinIO 卷内口令</b>：PG 口令固化在 <code>pgdata</code>（env 仅首次 initdb），MinIO 每次启动读 env。跑起来后改口令必须用 §9.3 的 <code>reset_password.sh</code>（自动处理各组件后端差异），不要只改文件。</div>

<h3>9.3 已跑起来后改口令 / 轮换（volume 已创建）—— 主路径</h3>
<p class="lead"><b>用 <code>reset_password.sh</code> 逐组件单选轮换</b>：自动处理「不同组件后端存口令方式不同」（PG 卷内 <code>ALTER ROLE</code> / MinIO 重建读 env / Grafana 容器内 reset / JWT·共享密钥 <code>up -d</code>），并保证 <code>secrets.env</code>（权威）↔ 组件口令 ↔ <code>.env</code> 三处一致。<b>不支持批量</b>，一次一个组件。</p>
<pre>cd /opt/omc/current/deploy
sudo bash reset_password.sh                   # 弹菜单，必须单选 1 个组件
# 或直接指定组件：pg | minio | grafana | jwt | shared
sudo bash reset_password.sh pg
# 每组件可选「随机生成（推荐）」或「手动输入」（JWT/共享密钥 ≥32 字符）；
# 随机值在终端显示一次，请妥善保存。</pre>
<p>各组件动作：<b>PG</b> 容器内 <code>ALTER ROLE</code> 改卷内口令 + 改 secrets.env + <code>up -d app acs worker</code>；<b>MinIO</b> 改 secrets.env + <code>up -d minio app acs worker</code>（旧预签名 URL ≤1h 内失效，重新生成即可）；<b>Grafana</b> 容器内 <code>grafana cli admin reset-admin-password</code> + 改 secrets.env；<b>JWT</b> 改 secrets.env + <code>up -d app</code>（已签发 token 失效，需重登）；<b>TR-069 共享密钥</b> 改 secrets.env + <code>up -d acs app</code> + 打印基站同步告警。</p>
<div class="danger">⚠️ <b>TR-069 共享密钥轮换</b>后，未同步基站的 ACS 主动触达（下发 / 重启 / 升级 / 即时 GPV / MML）会失败，直到经 TR-069 <b>SetParameterValues</b> 把新密钥下发到所有基站（下次 Inform 生效）；周期 Inform 不受影响、设备仍在线、无数据丢失。建议分批灰度 + 监控 ConnReq 成功率。脚本会自动打印该告警。</div>
<p class="tip">改后再跑 <code>install.sh</code> 会复用 <code>secrets.env</code>（不会把新口令冲回旧值）。<b>不要</b>再改 <code>*.prod.yaml</code>（已是 <code>\${VAR}</code>）；<b>不要</b>用 <code>docker compose restart</code>（不重读 <code>.env</code>），轮换一律 <code>up -d</code>（配置变更触发重建）。</p>

<h3>9.4 调口令后验证</h3>
<pre># PG 可连
PGPASSWORD='新口令' psql -h 127.0.0.1 -U omcgo -d omcgo -c 'select 1'
# OMC 服务全部 OK
bash /opt/omc/current/deploy/healthcheck.sh</pre>

<h2>🌍 9.5 OMC_PUBLIC_HOST —— 基站可达地址（env 文件必填项）</h2>
<p class="lead"><code>OMC_PUBLIC_HOST</code> 是本机对外的 IP / 域名（<b>基站侧能访问到的地址</b>）。
worker 用它拼 PM 文件上传 URL（<code>http://&lt;OMC_PUBLIC_HOST&gt;:7557/smallcell/FileUploadService?...</code>）
下发给基站；填 localhost / 127.0.0.1 基站将无法回传文件。</p>
<h3>9.5.1 env 文件在哪、怎么配</h3>
<ul class="list">
<li><b>文件位置：</b><code>/opt/omc/current/deploy/.env</code>（即解压出来的交付包 <code>deploy/.env</code>；<code>current</code> 软链指向当前版本目录）</li>
<li><b>改哪一行：</b>把 <code>OMC_PUBLIC_HOST=</code> 填成本机对外 IP，例如 <code>OMC_PUBLIC_HOST=172.19.1.132</code>（多网卡填基站能路由到的那个；有域名可填域名）</li>
<li><b>谁读它：</b>compose 把它注入 <code>app</code> / <code>acs</code> / <code>worker</code> 三个容器；容器内配置 <code>worker.prod.yaml</code> 的 <code>upload_url_template</code> 用 <code>\${OMC_PUBLIC_HOST}</code> 展开</li>
</ul>
<pre>sudo vi /opt/omc/current/deploy/.env
#   OMC_PUBLIC_HOST=172.19.1.132          ← 改成本机对外 IP

# 改完重启业务容器使其生效（或重跑 install.sh）
cd /opt/omc/current/deploy
bash svc.sh restart app acs worker

# 验证容器内已拿到（应回显你填的 IP）
docker exec omcgo-worker-1 printenv OMC_PUBLIC_HOST</pre>
<div class="tip">✅ <b>升级自动继承</b>：<code>install.sh</code> 升级时会把上一版 <code>deploy/.env</code> 里的运维自定义值
（PostgreSQL / MinIO / Grafana 口令、JWT、<code>OMC_PUBLIC_HOST</code>）合并进新包 <code>.env</code>，<b>镜像 tag 仍用新包</b> ——
所以<b>升级无需重填</b>，仅<b>首次部署</b>需手动填一次。升级日志会打印「.env：已从上一版继承运维自定义值…」。
如需改值，编辑 <code>/opt/omc/current/deploy/.env</code> 后 <code>bash svc.sh restart app acs worker</code>。</div>
</div>

<div class="tab-content" id="tab-ops">
<h2>🛠️ 10. 日常运维</h2>
<p class="lead">所有命令在 <code>/opt/omc/current/deploy/</code> 目录下执行，项目名 <code>omcgo</code>。</p>

<h3>10.1 服务状态与启停</h3>
<pre># 查看状态
docker compose -p omcgo ps

# 启动全部服务
docker compose -p omcgo start

# 停止全部服务
docker compose -p omcgo stop

# 重启全部服务
docker compose -p omcgo restart

# 重启单个服务（如 app）
docker compose -p omcgo restart app

# 停止并移除容器（数据保留）
docker compose -p omcgo down

# 重新创建并启动（如镜像更新后）
docker compose -p omcgo up -d</pre>

<h3>10.2 日志与健康检查</h3>
<pre># 查看日志
docker compose -p omcgo logs -f app

# 查看最近 100 行日志
docker compose -p omcgo logs --tail 100 app

# 健康检查
bash /opt/omc/current/deploy/healthcheck.sh</pre>

<h3>10.3 进入容器 / 连接数据库</h3>
<pre># 进入容器
docker compose -p omcgo exec app /bin/sh

# 连接数据库
docker compose -p omcgo exec postgres psql -U omcgo -d omcgo</pre>

<h2>🔧 故障排查</h2>
<ul class="list">
<li>容器状态：<code>cd /opt/omc/current/deploy && docker compose -p omcgo -f docker-compose.infra.yml -f docker-compose.app.yml -f docker-compose.web.yml -f docker-compose.monitoring.yml ps</code></li>
<li>业务日志：<code>docker compose -p omcgo logs -f app acs worker</code>（在 deploy/ 目录）</li>
<li>基础设施日志：<code>docker compose -p omcgo logs -f postgres redis nats minio</code></li>
<li>容器日志：<code>docker logs &lt;容器名&gt; --tail 200</code></li>
<li>重跑健康检查：<code>bash /opt/omc/current/deploy/healthcheck.sh</code></li>
<li>停止全栈：<code>cd /opt/omc/current/deploy && docker compose -p omcgo -f docker-compose.infra.yml -f docker-compose.app.yml -f docker-compose.web.yml -f docker-compose.monitoring.yml down</code></li>
<li>查看脚本帮助：<code>bash &lt;脚本&gt; -h</code>（install-docker.sh / setup-mirrors.sh / install.sh / uninstall.sh / healthcheck.sh 均支持）</li>
<li>Docker 装好却起不来报 <code>iptables not found</code>：最小化系统漏装 iptables，跑 <code>sudo apt install -y iptables nftables bridge-utils</code>（Ubuntu/Debian）或 <code>sudo yum install -y iptables nftables bridge-utils</code>（RHEL 系），再 <code>sudo systemctl start docker</code>。</li>
<li>cAdvisor 默认使用 DaoCloud 镜像站：<pre>export IMAGE_CADVISOR=gcr.m.daocloud.io/cadvisor/cadvisor:v0.55.1
bash build-images.sh        # 打包；本机起栈则 docker compose ... up -d --build</pre>如需手工预拉缓存：<pre>docker pull --platform linux/amd64 gcr.m.daocloud.io/cadvisor/cadvisor:v0.55.1
docker tag gcr.m.daocloud.io/cadvisor/cadvisor:v0.55.1 gcr.m.daocloud.io/cadvisor/cadvisor:v0.55.1-amd64-saved</pre></li>
<li>完整运维手册：见随项目包附带 <code>docs/OMC内网离线部署手册（运维侧）.md</code></li>
</ul>
</div>

<p class="note">索引刷新于 $(date -u '+%Y-%m-%dT%H:%M:%SZ')</p>
</div>
<script>
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
  });
});
</script>
</body></html>
HTML
