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

<h2>🚚 5. 一键部署 OMC</h2>
<p class="lead">所有场景都使用 <code>deploy.sh</code>，脚本会自动检测已有镜像并智能跳过重复加载。</p>

<h3>场景 A：首次部署（全新服务器）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;
sudo bash deploy/deploy.sh</pre>
<p class="lead">全量执行：load 所有镜像 → 建目录 → 启动基础设施 → migrate/seed → 启动全栈。</p>

<h3>场景 B：升级业务版本（最常用）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;新版本&gt;-&lt;架构&gt;
sudo bash deploy/deploy.sh --skip-infra</pre>
<p class="lead">跳过基础设施镜像加载，仅加载新业务镜像 → 执行新迁移 → 重建业务容器（app/acs/worker/web），基础设施容器保持运行不受影响。</p>

<h3>场景 C：重复部署同版本（修复/重启）</h3>
<pre>cd /opt/omc/releases/omc-&lt;test|release&gt;-&lt;当前版本&gt;-&lt;架构&gt;
sudo bash deploy/deploy.sh</pre>
<p class="lead">脚本检测到所有镜像已存在 → 自动跳过 load → 重启容器 → 重跑 migrate（幂等）→ 健康检查。适用于服务异常需要完整重启的场景。</p>

<h3>其他参数</h3>
<pre>sudo bash deploy/deploy.sh --check-only             # 仅检查环境，不动手
sudo bash deploy/deploy.sh --skip-migrate            # 不跑 migrate / seed
sudo bash deploy/deploy.sh --skip-monitoring         # 不起监控栈
sudo bash deploy/deploy.sh -h                        # 查看所有参数</pre>

<p class="tip">deploy.sh 自动：环境检查 → 目录布局 → 智能 load 镜像（已有则跳过并重启）→ 默认口令检查 → 启动 infra → 等就绪 → migrate → seed → <code>docker compose up -d</code> 全栈 → 健康检查。<b>全 docker compose 部署，宿主机不再放业务二进制。</b></p>
<div class="danger">⚠️ 生产环境首次部署前请编辑 <code>/opt/omc/current/deploy/.env</code> 与 <code>/opt/omc/etc/*.prod.yaml</code>，改 <b>PostgreSQL / MinIO / Grafana / JWT</b> 默认口令为强口令。</div>

<h2>✅ 6. 验证部署</h2>
<pre>bash /opt/omc/current/deploy/healthcheck.sh</pre>
<p class="lead">应输出全部 <code>[OK]</code>：业务容器（app/acs/worker）+ 基础设施容器（postgres/redis/nats/minio）+ 监控容器（prometheus/grafana/loki/...）+ 4 个健康端点（app /health, acs /healthz, app /metrics, 前端首页）。</p>
</div>

<div class="tab-content" id="tab-config">
<h2>🌐 7. 部署后访问地址</h2>
<div class="kv">
<b>Web 管理页：</b>http://&lt;服务器IP&gt;:8080<br>
<b>MinIO Console：</b>http://&lt;服务器IP&gt;:9001<br>
<b>App 健康端点：</b>http://&lt;服务器IP&gt;:8081/health<br>
<b>ACS 健康端点：</b>http://&lt;服务器IP&gt;:9090/healthz<br>
<b>Grafana（监控）：</b>http://&lt;服务器IP&gt;:3000　<span class="sz">仅启用监控栈时</span>
</div>

<h2>👤 8. 初始账号 / 口令</h2>
<div class="danger">⚠️ 全部默认口令<b>首次登录后必须改</b>。生产部署前需重新生成强口令并同步到 deploy/.env 与 *.prod.yaml。</div>
<div class="kv">
<b>Web 管理员：</b><code>admin</code> / <code>admin123</code><br>
<b>MinIO Console：</b><code>minioadmin</code> / <code>minioadmin</code><br>
<b>PostgreSQL：</b><code>omcgo</code> / <code>omcgo123</code><br>
<b>Grafana：</b><code>admin</code> / <code>admin</code>
</div>

<h2>🔑 9. 配置文件修改指南（账号 / 口令 / JWT）</h2>
<p class="lead">默认口令在两处出现、必须<b>同步修改</b>，否则 OMC 进程连不上 PostgreSQL / MinIO：</p>
<ul class="list">
<li><code>/opt/omc/current/deploy/.env</code> —— docker compose 起容器时的<b>初始口令 / 镜像版本</b>（仅首次 <code>volumes</code> 创建时生效）</li>
<li><code>/opt/omc/etc/{app,acs,worker}.prod.yaml</code> —— OMC 三进程连接中间件时的<b>客户端口令</b></li>
</ul>

<h3>9.1 需同步修改的口令对应表</h3>
<table>
<thead><tr><th>项</th><th>deploy/.env</th><th>etc/app.prod.yaml</th><th>说明</th></tr></thead>
<tbody>
<tr><td>PostgreSQL 账号</td><td><code>POSTGRES_USER=omcgo</code></td><td><code>db.dsn</code> / <code>tsdb.dsn</code> 里的 <code>omcgo</code></td><td>DSN 格式：<code>postgres://<b>账号</b>:<b>口令</b>@postgres:5432/omcgo?sslmode=disable</code></td></tr>
<tr><td>PostgreSQL 口令</td><td><code>POSTGRES_PASSWORD=omcgo123</code></td><td><code>db.dsn</code> / <code>tsdb.dsn</code> 里的 <code>omcgo123</code></td><td>同上，出现两次（db + tsdb）</td></tr>
<tr><td>PostgreSQL 库名</td><td><code>POSTGRES_DB=omcgo</code></td><td>DSN 路径部分 <code>/omcgo</code></td><td>一般不改</td></tr>
<tr><td>MinIO 账号</td><td><code>MINIO_ROOT_USER=minioadmin</code></td><td><code>minio.access_key</code></td><td>三个 yaml（app/acs/worker）都要改</td></tr>
<tr><td>MinIO 口令</td><td><code>MINIO_ROOT_PASSWORD=minioadmin</code></td><td><code>minio.secret_key</code></td><td>同上</td></tr>
<tr><td>JWT 密钥</td><td><code>OMCGO_JWT_SECRET=...</code></td><td><code>jwt.secret</code>（同值）</td><td>必须 ≥ 32 字符；产生：<code>openssl rand -base64 48</code></td></tr>
<tr><td>Grafana 管理员</td><td><code>GRAFANA_ADMIN_PASSWORD=admin</code></td><td>—</td><td>仅监控栈使用；首次登录 :3000 也会强制提示改口令</td></tr>
<tr><td>Web 管理员 admin</td><td>—</td><td>—</td><td>首次登录 <code>http://&lt;IP&gt;:8080</code> 后在「个人中心 → 修改密码」里改，<b>不需改配置文件</b></td></tr>
</tbody>
</table>

<h3>9.2 修改步骤（首次部署、<code>deploy.sh</code> 起 infra 之前）</h3>
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

# 3) 改 etc/*.prod.yaml（OMC 进程以这里为准连接中间件）
sudo vi /opt/omc/etc/app.prod.yaml      # db.dsn / tsdb.dsn / minio.* / jwt.secret
sudo vi /opt/omc/etc/acs.prod.yaml      # db.dsn / minio.* （按需）
sudo vi /opt/omc/etc/worker.prod.yaml   # db.dsn / minio.* （按需）

# 4) 一键部署（自动 load 镜像 + up 全栈）
sudo bash /opt/omc/current/deploy/deploy.sh</pre>

<div class="danger">⚠️ <b>volume 已创建后改口令无效</b>：PostgreSQL / MinIO 只在首次创建 <code>pgdata</code> / <code>miniodata</code> volume 时读取环境变量。若发现初始口令错了，需重应。</div>

<h3>9.3 已跑起来后改口令（volume 已创建）</h3>
<pre># PostgreSQL — 在容器内改
sudo docker exec -it omcgo-postgres-1 psql -U omcgo -d omcgo \
    -c "ALTER USER omcgo WITH PASSWORD '新口令';"
# 同步改 etc/*.prod.yaml 中的 dsn 口令部分；重启业务容器
cd /opt/omc/current/deploy
sudo docker compose -p omcgo -f docker-compose.app.yml restart app acs worker

# MinIO — 使用 mc 客户端（或重建 volume）。参 MinIO 官方文档。</pre>

<h3>9.4 调口令后验证</h3>
<pre># PG 可连
PGPASSWORD='新口令' psql -h 127.0.0.1 -U omcgo -d omcgo -c 'select 1'
# OMC 服务全部 OK
bash /opt/omc/current/deploy/healthcheck.sh</pre>
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
<li>查看脚本帮助：<code>bash &lt;脚本&gt; -h</code>（install-docker.sh / setup-mirrors.sh / deploy.sh / healthcheck.sh 均支持）</li>
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
