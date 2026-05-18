#!/usr/bin/env bash
# =============================================================================
# OMC 交付包下载索引生成
#
# 扫描 archive/project/ 与 archive/infra/ 下的版本目录，生成 archive/index.html
# —— 8000 端口下载页：项目交付包、基础设施包各一张版本列表，并附操作步骤。
#
# 由 build-release.sh / build-images.sh 在归档后自动调用；serve.sh 启动时也会
# 调一次，确保索引与 archive/ 实际内容一致。也可手动运行刷新。
#
# 用法： ./gen-index.sh
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARCHIVE="$SCRIPT_DIR/archive"
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
    for f in "$d"omc-*.tar.*; do
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
[ -n "$PROJECT_ROWS" ] || PROJECT_ROWS='<tr><td colspan="4" class="empty">暂无项目交付包 —— 运行 ./build-release.sh 生成</td></tr>'
[ -n "$INFRA_ROWS" ]   || INFRA_ROWS='<tr><td colspan="4" class="empty">暂无基础设施包 —— 运行 ./build-images.sh 生成</td></tr>'

cat > "$ARCHIVE/index.html" <<HTML
<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>OMC 离线交付包下载</title>
<style>
body{font-family:-apple-system,"Segoe UI",sans-serif;margin:0;color:#1f2937;background:#f5f6f8}
.wrap{max-width:1080px;margin:0 auto;padding:2rem 1.5rem}
h1{font-size:1.4rem;margin:0 0 .3rem}
h2{font-size:1.08rem;margin:1.8rem 0 .6rem;padding-left:.55rem;border-left:4px solid #1668dc}
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
ol.steps{background:#fff;border:1px solid #e5e7eb;border-radius:6px;padding:1rem 1rem 1rem 2.5rem;font-size:.9rem;line-height:1.7;margin:0}
ol.steps li{margin:.55rem 0}
code{background:#f0f2f5;padding:.1rem .35rem;border-radius:3px;font-size:.85em}
pre{background:#1f2937;color:#e5e7eb;padding:.7rem .9rem;border-radius:5px;overflow-x:auto;font-size:.82rem;margin:.4rem 0;line-height:1.5}
.note{color:#9ca3af;font-size:.8rem;margin-top:2.2rem}
</style></head><body><div class="wrap">
<h1>OMC 离线交付包下载</h1>
<p class="lead">内网离线部署交付包。<b>项目包</b>与<b>基础设施包</b>相互独立、各自版本号：
首次部署两个都要下载；之后日常升级通常只需更新项目包。</p>

<h2>📋 操作步骤</h2>
<ol class="steps">
<li><b>确认目标机架构</b>：在目标机执行 <code>uname -m</code>。结果 <code>x86_64</code> 下载 <code>amd64</code> 包；<code>aarch64</code> 下载 <code>arm64</code> 包。两个包必须取<b>同一架构</b>。</li>
<li><b>下载交付包</b>（首次部署两个都下，连同各自的 <code>.sha256</code>）：<br>
&nbsp;&nbsp;· 基础设施包 <code>omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz</code> —— Docker 引擎 + 基础镜像（PostgreSQL / Redis / NATS / MinIO / Nginx）<br>
&nbsp;&nbsp;· 项目包 <code>omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz</code> —— OMC 二进制 + 前端 + 配置 + 数据库迁移</li>
<li><b>校验完整性</b>（防止下载损坏）：
<pre>sha256sum -c omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz.sha256
sha256sum -c omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz.sha256</pre></li>
<li><b>解压两个压缩包</b>：
<pre>tar -xJf omc-infra-&lt;版本&gt;-&lt;架构&gt;.tar.xz
tar -xJf omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;.tar.xz</pre></li>
<li><b>安装 Docker</b>（目标机<b>未装 Docker</b> 时执行；已装可跳过）：
<pre>cd omc-infra-&lt;版本&gt;-&lt;架构&gt;/docker
sudo bash install-docker.sh</pre></li>
<li><b>导入基础镜像</b>（从基础设施包）：
<pre>docker load -i omc-infra-&lt;版本&gt;-&lt;架构&gt;/images/infra-images-&lt;架构&gt;.tar</pre></li>
<li><b>部署 OMC</b>：进入项目包目录 <code>omc-&lt;test|release&gt;-&lt;版本&gt;-&lt;架构&gt;/</code>，按
<code>docs/OMC内网离线部署手册（运维侧）.md</code> 执行（修改默认口令 → 起基础设施容器 → 执行数据库迁移 → 启动 app/acs/worker）。</li>
</ol>

<h2>📦 项目交付包</h2>
<p class="lead">OMC 二进制 + 前端 + 配置 + 数据库迁移 + 部署模板。发版频繁。
渠道：<span class="ch ch-test">test</span> 测试阶段　<span class="ch ch-release">release</span> 正式发布。</p>
<table><thead><tr><th>项目版本</th><th>渠道</th><th>构建时间</th><th>下载</th></tr></thead>
<tbody>$PROJECT_ROWS</tbody></table>

<h2>🛠️ 基础设施包</h2>
<p class="lead">Docker 引擎离线安装包 + 基础镜像。不常变更，仅基础设施升级时更新。</p>
<table><thead><tr><th>基础设施版本</th><th>Docker 版本</th><th>构建时间</th><th>下载</th></tr></thead>
<tbody>$INFRA_ROWS</tbody></table>

<p class="note">索引刷新于 $(date -Is)　·　架构对照 <code>uname -m</code>：x86_64 → amd64，aarch64 → arm64</p>
</div></body></html>
HTML
