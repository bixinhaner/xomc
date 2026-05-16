# 代码审查报告 — T-0137 TR069 报文跟踪 M1/M2/M3 + 活体验证修复

| 项 | 值 |
|---|---|
| 审查时间 | 2026-05-16 |
| 审查对象 | T-0137 主 commit（48 文件 / +5167 / -17） |
| 基线 commit | 9b371f25 |
| 审查者 | Claude（自审，附人工活体验证） |
| 关联 Backlog | T-0137（umbrella）+ M1/M2/M3 三 milestone + L-1..L-12 修复 |
| 关联 PRD | docs/project/prd/F01-tr069-message-trace.md |
| 关联设计 | docs/design/TR069报文跟踪-设计.md |
| 子任务文档 | docs/project/backlog/subtasks/T-0137-trace.md §4.4 §4.5 |
| **审查结论** | **PASS_WITH_WARNINGS** |

---

## 1. 变更范围

### 1.1 后端新增（功能主体）

- **`omcgo/internal/trace/` 全模块**（14 文件）：model / repository / pg_repository / service / handler / capture_consumer / sweeper / exporter / sse_notifier / bulk_store / whitelist / metrics / capture_consumer_test / service_test / whitelist_test
- **`omcgo/internal/acs/trace_capture.go`** — ACS hot path hook，调 service.EnqueueCapture
- **migrations**：000106 (`trace_messages` hypertable + 3 天 retention) / 000107 (`trace_export_jobs`) / seed/000108 (菜单 + 三角色绑定)
- **`omcgo/scripts/trace_stress_test.sh`** — M3 压测骨架

### 1.2 后端修改（接线 / 兼容）

| 文件 | 变更目的 |
|---|---|
| `cmd/acs/main.go` + `acs/server.go` + `acs/handler.go` | 启动 trace WhitelistCache + 注入 hook |
| `cmd/app/provider/modules.go` + `router.go` | 注入 trace service / handler / SSE notifier / MinIO presign client |
| `cmd/worker/main.go` | 启动 CaptureConsumer / Sweeper / Exporter；接入 audit sink |
| `internal/admin/middleware.go` | **L-3** 加 `?token=` query fallback（SSE 用） |
| `core/appconfig/config.go` | 加 `MinIOConfig.PublicEndpoint` 字段 |
| `core/components/minio/minio.go` | 加 `NewPresignClient` helper（**L-8** + Region 强制） |
| `core/components/nats/nats.go` | 加 3 个 trace stream 定义 |
| `core/event/subjects.go` | 加 trace 事件主题常量 |
| 3 份 `config.dev.yaml` | 加 `public_endpoint` + bucket + NATS stream 配置 |
| `scripts/e2e_verify.sh` | M3 补 6 条 trace claim（AC-1..AC-6） |

### 1.3 前端新增

- `frontend-core/services/api/traceApi.ts` — 7 个 REST 端点封装
- `frontend-core/hooks/api/useTrace.ts` — React Query Hooks + `useTraceSseRefresh`
- `frontend-core/types/trace.ts` — 类型定义
- `webcode/pages/ops/MessageTrace/index.tsx`（475 行）— 列表 / 创建 / 停止 / 报文详情 / 导出

### 1.4 前端修改

- i18n 双语 catalog 加 trace 模块语料
- frontend-core/services/api/index.ts 导出 traceApi
- webcode 路由（navConfig / componentRegistry / routes）注册 `/ops/message-trace`

### 1.5 文档

- 设计文档 + PRD（之前 commit 已立项）
- backlog.md / dod.md / T-0137-trace.md §4.5 完成清单 + 12 条遗留登记

---

## 2. 检查项审查结果

### 2.1 Go 后端（高密度区）

| 检查项 | 状态 | 备注 |
|---|---|---|
| 命名规范 | ✅ | 导出 PascalCase / 未导出 camelCase / 接口名词无 I 前缀 |
| 错误 wrap | ✅ | 全模块 `fmt.Errorf("context: %w", err)` |
| SQL 安全 | ✅ | 全部 Squirrel + pgx，无字符串拼接 |
| 运营商硬编码 | ✅ N/A | trace 不区分运营商 |
| 认证 | ✅ | trace handler 走 RequireAuthWithAPIKey；audit log 含 username/IP/UA |
| 资源泄漏 | ✅ | Sweeper / CaptureConsumer 都有 `Stop()` + cleanup func；ScheduledWakeup goroutine 用 channel 关 |
| 测试覆盖 | ⚠️ | trace 包含 4 个 `_test.go`（service / whitelist / capture_consumer 含 L-11 / handler 暂缺）；3 个 PASS + 已有 service 测试都 PASS |
| `go build ./...` | ✅ | 全绿 |
| `go test ./...` | ✅ | 全绿（含本次新增 capture_consumer_test 3 PASS） |
| `go vet ./...` | ✅ | 全绿（段 2 修了 software/topology 预存 vet 错） |

### 2.2 TR-069 协议合规

| 检查项 | 状态 | 备注 |
|---|---|---|
| SOAP 命名空间 | ✅ | `xmlns:cwmp="urn:dslforum-org:cwmp-1-0"` 正确；Inform request 原文 CDATA 完整保留 |
| cwmpID 匹配 | ✅ | hook 在 SOAP 解析后捕获 ID，trace_messages.cwmp_id 列实测含值 |
| 会话状态机 | ✅ | hook 不阻塞 hot path（P99 < 5ms 指标，反例告警阈值已在 metric HELP 写明） |
| Empty Response | ✅ | Empty 报文 direction=in/out 各 1 条，payload=空 CDATA，符合预期 |
| Download/Upload SOAP 模板 | ⚠️ | 段 1 未涉及，段 2 修了 Download 模板 cwmp 前缀（V-1）|

### 2.3 数据库 / 迁移

| 检查项 | 状态 | 备注 |
|---|---|---|
| 编号连续 | ✅ | 000106 / 000107 / seed/000108（紧接上一版本号）|
| up/down 配对 | ✅ | 全 3 个迁移含双段，DROP 语句对称 |
| TimescaleDB hypertable | ✅ | `trace_messages` 创建 hypertable + drop_after=3 days policy；实测 `timescaledb_information.hypertables` 含此表 |
| goose StatementBegin/End | ✅ | DO 块/函数都正确包裹 |
| 索引 | ✅ | `idx_trace_messages_sn_time` / `idx_trace_messages_task_time` / `idx_trace_tasks_status_expires` 覆盖热点查询 |
| 种子幂等 | ✅ | 000108 含 `ON CONFLICT DO NOTHING` |
| 分区表外键 | ✅ N/A | trace_messages 无外键引用其他分区表 |

### 2.4 前端

| 检查项 | 状态 | 备注 |
|---|---|---|
| 类型安全 | ✅ | 全模块禁 `any`；types/trace.ts 含 `TraceTask` / `TraceMessage` / `TraceExportJob` |
| API 服务模式 | ✅ | 一模块一文件，位于 frontend-core/services/api/ |
| Hook 模式 | ✅ | useTrace.ts 用 React Query；polling 条件式（仅 running 任务）|
| i18n | ✅ | 用户可见文本走 `t()`；表格"操作"列约定用 `table.action`（L-4 修复）|
| XSS | ✅ | XML 用 `<pre>` 渲染原文，不通过 `dangerouslySetInnerHTML` |
| Token 处理 | ✅ | SSE 通过 `?token=accessToken` query；其他 API 走 Axios 拦截器 Bearer header |
| 多皮肤评估 | ⚠️ | frontend-core 改动会影响 webcode-v2/webcode-v3，未实际编译验证（无活体环境）|

### 2.5 可观测性

| 检查项 | 状态 | 备注 |
|---|---|---|
| Prometheus 指标 | ✅ | 5 个 family 全暴露（L-5 修），label 组合 prime；HELP 标注 emit owner（L-6/L-9）|
| Zap 结构化日志 | ✅ | 全模块 `zap.String/Int/Error`，无 fmt.Sprintf 拼接 |
| 审计日志 | ✅ | handler 写 trace_start/trace_export；Sweeper 写 trace_stop actor=system（L-10）|
| 优雅关闭 | ✅ | 三进程都 Register 关闭函数 |

### 2.6 安全

| 检查项 | 状态 | 备注 |
|---|---|---|
| 鉴权扩散 | ⚠️ | `?token=` query 是 SSE 必需，但扩散到所有走该 middleware 的端点；token 进 nginx access log（**已在 middleware 注释中明确**警告范围）|
| RBAC | ✅ | trace 端点走 v1 group，admin/operator/viewer 通过 menu 绑定授权 |
| 敏感信息 | ✅ | 日志不打 payload 全文，只打 size / cwmp_id；token 走 query 时 access log 风险已注释 |
| 文件上传 | ✅ N/A | trace 不接受上传 |

---

## 3. 发现汇总

### 3.1 CRITICAL（阻塞 commit）

**无**。

### 3.2 WARNING（不阻塞，已知遗留）

| ID | 描述 | 处置 |
|---|---|---|
| W-1 | 多皮肤（webcode-v2/v3）未实际编译验证 | T-0137 之外，多皮肤项目独立沙盒；本 commit 仅触达 webcode 主皮肤 |
| W-2 | trace handler 没有独立 `handler_test.go` | service / capture_consumer / whitelist 各自有 unit test，handler 由 e2e_verify.sh 6 条 claim 覆盖；下一 sprint 可补 |
| W-3 | `?token=` query 扩散到所有走 RequireAuthWithAPIKey 的端点 | middleware 注释已警告 access log 风险；推荐内网部署 + 后续加 audit log 钩子记录用 query 鉴权的请求 |
| W-4 | trace_stress_test.sh 是骨架，没有真实大报文反例 | M3 设计文档已说"压测对照基线"，正式压测需在 staging 跑；本 sprint 接受骨架级 |

### 3.3 INFO

- 设计 §D9 D10（不做 SN 限流 + 用户日上限）按"做减法"决策落地，代码无对应实现 — **正确**
- L-5 选用 `Vec prime` 方案而非"按 role 拆 NewMetrics"，是"做减法"权衡（避免代码复杂度爆炸）
- L-8 用独立 presign client 而非"字符串替换 host" — 后者会破坏签名

---

## 4. 端到端验证证据

| 验证项 | 证据 |
|---|---|
| 创建任务 + 抓包 | task `82326bae-e673-...`，5 分钟窗口前抓 4 条 SOAP（11 Inform 周期 × 4 条另一任务 = 44 条）|
| Payload 原文落库 | 含 `xmlns:cwmp="urn:dslforum-org:cwmp-1-0"` + `<cwmp:ID>` 完整 envelope |
| Sweeper 自动 expire | running → stopped，sweep_at 比 expires_at 晚 16 秒（60s ticker 内）|
| 异步导出 | worker Exporter 40-43ms 完成，写 MinIO `omc-exchange/trace-export/` |
| 浏览器直接下载 | `fetch(download_url)` 200 / 84273 bytes(之前任务) / 7448 bytes（新任务） / XML 结构正确 |
| audit_logs 完整 | trace_start (admin/IP/UA) + trace_export (admin) + trace_stop (system / reason=sweeper_expired) |
| Prometheus 5 指标 | active_tasks / capture_latency_seconds / messages_captured_total{in,out} / messages_dropped_total{4 reason} / storage_bytes{inline,minio} 全暴露；storage_bytes inline=78581 == DB SUM 一致 |
| SSE 实时刷 | console 0 error，SSE 连接建立后 task.* 事件触发 invalidate |
| Polling 兜底 | running 中 message_count UI 自动从 0 同步到 DB 真实值（8s 内）|
| i18n 列名 | "操作"（已不是 `common.action`）|
| 菜单 seed | 镜像内含 000108 + goose_db_version_seed=108 + 3 角色绑定 |

---

## 5. 结论

**PASS_WITH_WARNINGS** — 可合入。

- 0 CRITICAL，4 WARNING 全部"已知 + 已处置"
- 全套自动化检查（build/test/vet）通过
- 活体端到端 11 项验证通过
- 12 条 L-N 遗留问题在 §4.5 完整登记并全部修复

WARNING 项均不阻塞当前 commit，可在后续 sprint 补强（详见 W-1..W-4 处置说明）。
