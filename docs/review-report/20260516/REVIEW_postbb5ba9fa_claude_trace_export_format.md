# 代码审查报告 — 报文导出 XML 可读性 + nginx index.html 禁缓存

| 项 | 值 |
|---|---|
| 审查时间 | 2026-05-16 |
| 审查对象 | 3 文件 / +187 / -5 |
| 基线 commit | bb5ba9fa（段 3 UX follow-up） |
| 关联 Backlog | T-0137（活体试用反馈：导出 XML 可读性 + 部署期缓存陷阱） |
| 审查者 | Claude |
| **审查结论** | **PASS** |

---

## 1. 变更范围

### 1.1 报文导出 XML 可读性增强（用户反馈）

**`omcgo/internal/trace/exporter.go`**：

- 文件头加任务元信息块（task_id / device_sn / 抓包窗口 / 导出时间 / 总条数），用 ═ 装饰边框
- 每条 Message 上方插：
  - 横向分隔线 `<!-- ──── -->`
  - 摘要注释行：`<!-- #001  CPE→ACS  Inform  2026-05-16 15:15:52.218  cwmp_id=676475467 -->` — 运维滚浏览时一眼定位时间和方向
- CDATA 内 SOAP 原文做 `prettyXML` 按 tag 缩进美化（与 webcode 前端 prettyXML 镜像实现）
- `prettyXML` 用 3 个编译期初始化的正则：`cdataRE`（保护 CDATA 不被切）/ `tagBoundaryRE`（在 `>...<` 边界拆行）/ `inlineTagRE`（识别 `<tag>text</tag>` 单行不增缩进）

**新增 `omcgo/internal/trace/exporter_test.go`** — 6 个 unit test 覆盖：
- 空输入 / 单行变多行 / CDATA 保留 / inline tag 缩进 / `<?xml ?>` 声明不计深度 / 自闭合 tag

### 1.2 nginx 部署期缓存陷阱（root cause hotfix）

**`deployments/docker/default.conf`**：给 `location = /index.html` 加 `Cache-Control: no-cache, no-store, must-revalidate` + `Pragma: no-cache`（用 `add_header ... always`，**不能用 `expires 0;`** — 后者会自动设 `max-age=0` 覆盖 add_header）。

## 2. 根因分析（部署期缓存）

排查"下载又失败"过程中浮出来的真根因：vite 构建的 dist 入口 `index.html` 通过 nginx 服务，但旧 nginx 配置**对 index.html 没有 no-cache 头**。浏览器默认按 HTTP cache 语义缓存：

- 用户首次访问 → 拿到 index.html-v1，引用 `index-AAAA.js`
- 服务器 docker-run.sh 重建 → 镜像里 index.html-v2 引用 `index-BBBB.js`，但旧 v1 在浏览器 cache
- 用户再次访问 → 浏览器 304 验证后仍用 cached v1 → 加载 `index-AAAA.js` → 但 nginx 镜像里这个 chunk 已删除 / 内容是旧版本 → 行为按旧代码走 → 下载失败、用户看到旧 UI、修复看上去"未生效"

修复后任何访问 index.html 都会向服务器重新拉，hash JS chunks 保持 `expires 30d immutable`（这是对的，因为 hash 文件名变化自然 invalidate）。

## 3. 端到端验证

| 项 | 证据 |
|---|---|
| `go build ./...` | ✅ |
| `go test -run TestPrettyXML ./internal/trace/...` | ✅ 6/6 PASS |
| nginx 响应头 | curl `http://localhost:8081/index.html` 返回 `Cache-Control: no-cache, no-store, must-revalidate` + `Pragma: no-cache` ✅ |
| 实际导出文件可读性 | 41933 bytes 含元信息块 + 16 条 Message 都有 `#NNN  方向  RPC  时间` 摘要 + SOAP 多层缩进 |
| 用户实测 | 用户确认"现在可以下载文件了" ✅ |

## 4. 检查项

| 检查项 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./...` | ✅ |
| `go vet ./...` | ✅ |
| 性能（正则编译期初始化） | ✅ `var (cdataRE = regexp.MustCompile(...))` 避免热路径反复编译 |
| 单元测试覆盖 | ✅ 6 个 case 覆盖关键边界 |
| 文档（why 注释） | ✅ exporter.go 头部 + nginx config 都注释了"为什么不能用 expires 0" |
| 破坏性变更 | ✅ N/A — 输出文件结构兼容（仍然是 `<TraceExport>` 根，仍含 `<Message>` 列表） |

## 5. 发现

### 5.1 CRITICAL / WARNING

**无**。

### 5.2 INFO

- exporter prettyXML 与 webcode `MessageTrace` 前端 prettyXML 是**双语言镜像实现**，未来如有 XML 格式特殊 case 需要两边同步修。可考虑后续把 Go 版本移到 `pkg/xmlutil` 给其他模块复用。
- nginx no-cache 修复对**已访问过用户**有缓存惯性，需要清一次缓存后再生效（之后永久 OK）。

## 6. 结论

**PASS** — 可合入。
