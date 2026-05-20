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
    # 仅显示 amd64 包（release 工具链 amd64-only；历史 *-arm64.tar.* 物理保留但不进列表）
    for f in "$d"omc-*-amd64.tar.*; do
      [ -f "$f" ] || continue
      case "$f" in *.sha256) continue ;; esac
      bn="$(basename "$f")"
      sz="$(du -h "$f" | cut -f1)"
      links="${links}<a href=\"${kind}/${v}/${bn}\">${bn}</a> <span class=\"sz\">(${sz})</span><br>"
    done
    links="${links}<a href=\"${kind}/${v}/RELEASE.txt\">RELEASE.txt</a>"
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
</style></head><body><div class="wrap">
<h1>OMC 离线版本下载</h1>
<p class="lead">内网离线部署交付包。<b>项目包</b>与<b>基础设施下载</b>相互独立、各自版本号：
首次部署两个都要下载；之后日常升级通常只需更新项目包。</p>

<h2>📋 你应该下载哪些文件？</h2>
<div class="tabs">
<div class="tab"><b>🆕 首次部署</b>
基础设施下载 <code>omc-infra-*.tar.xz</code> + <code>.sha256</code><br>
项目包 <code>omc-&lt;test|release&gt;-*.tar.xz</code> + <code>.sha256</code></div>
<div class="tab"><b>♻️ 日常升级</b>
仅项目包 <code>omc-&lt;test|release&gt;-*.tar.xz</code> + <code>.sha256</code><br>
<span class="sz">基础设施已部署、Docker 已装时，只更新项目包</span></div>
</div>

<h2>🔐 1. 校验完整性</h2>
<pre>sha256sum -c omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz.sha256
sha256sum -c omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz.sha256</pre>

<h2>📦 2. 解压交付包</h2>
<pre># 推荐目录布局：/opt/omc/infra/ 装一次；/opt/omc/releases/&lt;版本&gt;/ 按版本独立
sudo mkdir -p /opt/omc/infra /opt/omc/releases
sudo tar -xJf omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz -C /opt/omc/infra --strip-components=1
sudo tar -xJf omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz -C /opt/omc/releases</pre>

<h2>🐳 3. 安装 Docker（仅首次部署）</h2>
<p class="lead">目标机已装 Docker（<code>docker --version</code> 返 ≥ 20.10）时跳过本步。</p>
<pre>cd /opt/omc/infra/docker
sudo bash install-docker.sh                         # 交互式：装完会引导选加速镜像
sudo bash install-docker.sh --mirror daocloud       # 非交互：装完直接配 DaoCloud 加速
sudo bash install-docker.sh -h                      # 查看所有参数</pre>
<p class="tip">install-docker.sh 自动：解压二进制 → 写 containerd / docker 的 systemd 单元 → <code>enable --now</code> 开机自启 → 验证 → 引导加速镜像。</p>

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

<h2>🚚 5. 一键部署 OMC（推荐）</h2>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;
sudo bash deploy/deploy.sh                          # 全套首次部署
sudo bash deploy/deploy.sh --skip-infra             # 日常升级（基础设施已装）
sudo bash deploy/deploy.sh --check-only             # 仅检查环境，不动手
sudo bash deploy/deploy.sh -h                       # 查看所有参数</pre>
<p class="tip">deploy.sh 自动：环境检查 → 建立目录布局 → load 基础镜像 → 默认口令检查 → 启动 infra 容器 → 等就绪 → migrate → seed → 装 systemd → 起 web → 健康检查。</p>
<div class="danger">⚠️ 生产环境首次部署前请编辑 <code>/opt/omc/current/deploy/docker-compose.infra.yml</code> 与 <code>/opt/omc/etc/*.prod.yaml</code>，改 <b>PostgreSQL / MinIO / JWT</b> 默认口令为强口令。</div>

<h2>✅ 6. 验证部署</h2>
<pre>bash /opt/omc/current/deploy/healthcheck.sh</pre>
<p class="lead">应输出全部 <code>[OK]</code>：3 个 systemd 服务（app/acs/worker）+ 4 个健康端点（app /health, acs /healthz, app /metrics, 前端首页）。</p>

<h2>🌐 7. 部署后访问地址</h2>
<div class="kv">
<b>Web 管理页：</b>http://&lt;服务器IP&gt;:8080<br>
<b>MinIO Console：</b>http://&lt;服务器IP&gt;:9001<br>
<b>App 健康端点：</b>http://&lt;服务器IP&gt;:8081/health<br>
<b>ACS 健康端点：</b>http://&lt;服务器IP&gt;:9090/healthz<br>
<b>Grafana（监控）：</b>http://&lt;服务器IP&gt;:3000　<span class="sz">仅启用监控栈时</span>
</div>

<h2>👤 8. 初始账号 / 口令</h2>
<div class="danger">⚠️ 全部默认口令<b>首次登录后必须改</b>。生产部署前需重新生成强口令并同步到 docker-compose.infra.yml 与 *.prod.yaml。</div>
<div class="kv">
<b>Web 管理员：</b><code>admin</code> / <code>admin123</code><br>
<b>MinIO Console：</b><code>minioadmin</code> / <code>minioadmin</code><br>
<b>PostgreSQL：</b><code>omcgo</code> / <code>omcgo123</code><br>
<b>Grafana：</b><code>admin</code> / <code>admin</code>
</div>

<h2>🔧 故障排查</h2>
<ul class="list">
<li>容器状态：<code>docker compose -f /opt/omc/current/deploy/docker-compose.infra.yml ps</code></li>
<li>服务日志：<code>journalctl -u omcgo-app -n 200 --no-pager</code>（acs / worker 同样）</li>
<li>容器日志：<code>docker logs &lt;容器名&gt; --tail 200</code></li>
<li>重跑健康检查：<code>bash /opt/omc/current/deploy/healthcheck.sh</code></li>
<li>查看脚本帮助：<code>bash &lt;脚本&gt; -h</code>（install-docker.sh / setup-mirrors.sh / deploy.sh / healthcheck.sh 均支持）</li>
<li>完整运维手册：见随项目包附带 <code>docs/OMC内网离线部署手册（运维侧）.md</code></li>
</ul>

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
<p class="note">索引刷新于 $(date -u '+%Y-%m-%dT%H:%M:%SZ')</p>
</div></body></html>
HTML
