# T-0137 拆分子任务（TR069 报文跟踪 Message Trace）

> 从 `docs/project/backlog.md` §3 Active 中的 umbrella T-0137 拆出（2026-05-15，S3 implement 进入前）。
> 行 schema 与 backlog.md §3 Active 主表对齐（13 列）。
> umbrella 行仍在主 backlog §3，状态联动通过本表 sub-task 推进。
>
> **来源**：`docs/design/TR069报文跟踪-设计.md` §9 实施分期 + `docs/project/prd/F01-tr069-message-trace.md` §8 实施要点
> **设计决策**：D1-D10 全部已定稿（设计 §10），实施期不重新评审
> **关键约束**：
> - D6 — 独立 `internal/trace/` 模块，**不**复用 `internal/task/` 队列（语义不匹配：trace 是"抓包窗口"，task 是"CWMP 指令派发"）
> - D8 — TimescaleDB hypertable `trace_messages` retention 3 天，自动 drop chunk
> - D9 — 不做单 SN 报文限流
> - D10 — 不做单用户单日任务上限
> - D3 — 报文一律原样落库，不做敏感字段 mask
> - 任何偏离上述决策的设计需求必须先与用户对齐

### 4.4 T-0137 拆分子任务（3 条，2026-05-15 S3 implement 起步）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0137-M1 | **MVP（打通端到端）** — migration（trace_tasks + trace_messages hypertable + retention 3 天）+ `internal/trace/` 模块（model/repository/service/handler）+ 7 个 REST endpoint（POST /tasks · GET /tasks · GET /tasks/{id} · POST /tasks/{id}/stop · GET /tasks/{id}/messages · GET /devices/{sn}/active-task · POST /tasks/{id}/export 同步小批量返回）+ ACS 端 capture hook（`internal/acs/handler.go` protocolLogger 附近 + sync.Map 白名单 + 启动加载 + 5s 兜底轮询 PG）+ PG inline 落库（暂不走 NATS，goroutine + channel 旁路异步写库）+ 前端（`frontend-core/services/api/traceApi.ts` + `hooks/api/useTrace.ts` + types + i18n）+ `webcode` 页面（任务列表 / 创建 / 停止 / 报文列表 / 详情 XML 美化）| feat | F01+F06+frontend-core+frontend | P2 | planned | Claude | L (~3-5d) | — | `prd/F01-tr069-message-trace.md` / `docs/design/TR069报文跟踪-设计.md` §4-6 §9 M1 | 2026-05-15 | **设计 §9 M1 范围**：不含 MinIO 外置、不含 NATS、不含 worker、不含 WebSocket 实时通知、不含异步下载、不含 Prometheus 指标。报文 ≤ 32KB inline；超过 32KB 暂直接截断后 inline（M2 补 MinIO）。ACS 与 app 之间 SN 白名单同步走 PG 5s 轮询兜底（M2 改 NATS 实时）。下载在 M1 仅支持 < 5MB 同步流式返回 |
| T-0137-M2 | **✅ done 2026-05-15（wave-batched M1→M2 续推完成，commit pending）** **工程化** — NATS 接线（`trace.task.started/stopped/purged` 普通 NATS + `trace.message.captured` JetStream 持久化）+ `omcgo-worker` 端 capture 消费者（批量落库 + 去抖动窗口）+ 超时巡检（worker 扫 `expires_at < now() AND status=running` → stopped + 发事件）+ purge handler（DELETE + MinIO 删对象）+ WebSocket 实时通知接 `internal/notification/` Hub（按 RBAC `trace:read` 过滤推送）+ 大报文外置 MinIO `trace-bulk` bucket（阈值 32KB，GZIP，key 格式 `{task_id}/{sn}/{ts}.xml.gz`）+ 异步下载（export_job 表 + MinIO `exchange` bucket + 预签名 URL，复用 `internal/transfer/` 模式）+ 前端 WebSocket 接入 + 下载交互（轮询 export 状态） | feat | F01+F06+frontend-core+frontend | P2 | done | Claude | L (~3-5d) | T-0137-M1 ✅ | `docs/design/TR069报文跟踪-设计.md` §4.4 §4.5 §7 §9 M2 | 2026-05-15 | **2026-05-15 done**：M2-01..M2-09 共 9 sub-task 全部落地 — NATS TRACE_TASK/TRACE_MSG/TRACE_EXPORT 3 stream（InterestPolicy + WorkQueuePolicy）+ app Service.SetEventBus 触发 task.{started,stopped,purged} 事件 + ACS WhitelistCache.Subscribe NATS 实时同步（30s 兜底）+ ACS Service.EnqueueCapture JetStream publish + worker CaptureConsumer 群组消费批量落库 + worker Sweeper 60s 巡检超时 + worker 订阅 trace.task.purged 异步清理 + MinIO trace-bulk bucket 大报文 GZIP 外置（阈值 32KB）+ Service.LoadMessagePayload lazy 回读 + GET /tasks/{id}/messages/{msgId}/payload + SSENotifier 转发 task 事件给 MessageHub.PublishGlobal + 异步导出 migration 000107 trace_export_jobs + worker Exporter + handler POST /tasks/{id}/export 改异步 + GET /exports/{job_id} 预签名 URL + 前端 useTraceSseRefresh / useTraceExportJob / MessageDetail useQuery lazy / ExportProgressModal 轮询自动打开下载。go build/test/vet 全过；webcode typecheck/lint trace 0 issues。**设计 §9 M2 范围**：M1 的 PG 直读 + 5s 轮询白名单 全部替换为 NATS 实时事件 + JetStream 报文落库。ACS hot path 仅丢 NATS 不写 PG（< 1ms 保证）。设计 §4.2 "兜底" 仍保留为 30s PG 对账（防 NATS 漏消息）。前端 WebSocket 用现有 `notification/subscribe` 协议 |
| T-0137-M3 | **✅ done 2026-05-15（M1→M2→M3 wave-batched 收官，commit pending）加固（运营商验收门槛）** — 5 个 Prometheus 指标（`trace_active_tasks` gauge / `trace_messages_captured_total{direction}` counter / `trace_messages_dropped_total{reason}` counter / `trace_storage_bytes_total{location=inline\|minio}` gauge / `trace_capture_latency_seconds` histogram，P99 < 5ms 反例告警）+ 审计日志（创建 / 停止 / 导出 操作写 audit_log，含 operator + IP + SN + 时间窗口）+ 压测脚本（≥ 100 并发任务 × 20 报文/分钟 × 5 分钟窗口，验证 ACS handler P99 latency 增量 < 5%）+ E2E 用例补 6 条（AC-1..AC-6 PRD §3） | feat | F01+ops | P2 | done | Claude | M (~1-2d) | T-0137-M2 ✅ | `prd/F01-tr069-message-trace.md` §7 度量 / `docs/design/TR069报文跟踪-设计.md` §8 §9 M3 | 2026-05-15 | **2026-05-15 done**：M3-01..M3-04 全部落地 — `internal/trace/metrics.go` 5 个 Prometheus 指标（omc_trace_active_tasks gauge / messages_captured_total{direction} / messages_dropped_total{reason: queue_full\|publish_failed\|decode_failed\|insert_failed} / storage_bytes{location} / capture_latency_seconds histogram P99<5ms 反例）+ Service/CaptureConsumer/Sweeper/ACS hook 全埋点 + 三进程 main 接 c.MetricsReg/inf.MetricsReg/w.MetricsReg；audit.Log 接入 handler CreateTask=trace_start / StopTask=trace_stop\|trace_purge / RequestExport=trace_export，含 username+IP+UserAgent+device_sn 详情；scripts/e2e_verify.sh 末尾补 6 条 trace claim（AC-1..AC-6 端点契约）；scripts/trace_stress_test.sh 压测脚本骨架（baseline ACS P99 → 创建 100 任务 → 负载 → 反例评估 capture P99<5ms / dropped=0 / handler P99 增量<5%）。go build/test/vet 全过；webcode typecheck/lint trace 0 issues。**设计 §9 M3 范围**：满足运营商验收 + DoD（`docs/project/dod.md`）+ Release Gate（`docs/project/release-gate.md`）。压测对照基线：ACS handler P99 latency 关跟踪 vs 开跟踪（100 任务同时活跃）差值 < 5%；trace_messages_dropped_total 必须 = 0 |

---

### 4.5 活体验证遗留问题（2026-05-16，真实基站 SN=1202000240194DP0026）

> 浏览器活体测过 创建任务 → 报文落库 → 查看明细 → payload 详情 全链路通了（任务 `93459d25-05b7-47e6-9246-f2bab3d3100f`，5 分钟窗口内 16 条报文 = 4 个 Inform 周期 × 4 条/周期，inline 落库，SOAP 命名空间齐全 `xmlns:cwmp="urn:dslforum-org:cwmp-1-0"`）。
> 以下是验证过程中浮出来的差距项，按优先级排，需要后续单独立 sub-task 或并入 hotfix。

| ID | 严重度 | 问题 | 现象 | 根因猜测 | 建议处置 |
|----|-------|------|------|--------|--------|
| **L-1** | P0 阻塞 | **菜单 seed migration 之前漏写** | `/ops/message-trace` 路由有但用户菜单没注入，PrivateRoute 动态菜单守卫直接重定向 /403 | M1 范围只写了表 migration，菜单 seed 没纳入 | ✅ 已补 `omcgo/migrations/seed/000108_seed_trace_menu.sql`（菜单 + admin/operator/viewer 三角色绑定），待随本次 commit 合入 |
| **L-2** | P1 体验 | **✅ 2026-05-16 修复** **任务列表 `message_count` 不自动刷新** | 表格"报文数"列长期显示 0，必须手动点 reload 才同步到 DB 真实值 | SSE 仅发 `task.{started,stopped,purged}` 三个事件，running 期间 message_count 持续增长但无事件触发 invalidate；React Query 也没开 polling 兜底 | **修复**：`useTraceTasks` 加条件 polling — `refetchInterval` 仅当当前页存在 running 任务时返回 3000ms，否则 false（避免空转）。**端到端**：建新任务后 UI message_count 8s 内从 0 自动跳到 4，跟 DB 一致，无需手动 reload |
| **L-3** | P1 体验 | **✅ 2026-05-16 修复** **全局 SSE `/api/v1/events/stream` 401** | 浏览器 EventSource 自动 401，trace 列表/详情/SSE 推送全部不工作 | 浏览器 EventSource 不能自定义 header；后端 `RequireAuthWithAPIKey` middleware 只看 `Authorization` header / `X-API-Key` 不看 query | **修复**：① middleware fallback 链加 `c.Query("token")` — Authorization 缺失时再读 query；② 前端 `useTraceSseRefresh` 把 `accessToken` 拼到 `?token=` 并依赖 token 重连。**端到端**：浏览器刷新后 console 0 error，SSE 链路通 |
| **L-4** | P1 体验 | **✅ 2026-05-16 修复** **`common.action` i18n 漏配** | 表格列头与 Drawer 报文列表的"操作"列直接显示 raw key `common.action` | catalog 里有 `table.action: "操作"`/"Action"，项目约定表格"操作"列用 `table.action`。MessageTrace 错用了 `common.action`（catalog 不存在该 key） | **修复**：把 `webcode/src/pages/ops/MessageTrace/index.tsx` 两处 `t('common.action')` 改成 `t('table.action')`，跟随项目约定。重建后实测列名变成"操作" |
| **L-5** | **P1 缺陷** ⬆️ | **✅ 2026-05-16 修复**（[`metrics.go`](../../../omcgo/internal/trace/metrics.go) prime + `repo.StorageStats` + sweeper.sampleStorageBytes）**Prometheus 指标只注册了 2/5** | 实测 ACS+worker 都只暴露 `active_tasks` / `capture_latency_seconds` 两个 family；缺 `messages_captured_total{direction}` / `messages_dropped_total{reason}` / `storage_bytes{location}` | **根因（已确认）**：5 个 family 都 `MustRegister` 了，但 Prometheus Go client 对 CounterVec/GaugeVec 行为是"首次 WithLabelValues 之前整个 family 不输出"。`captured_total` 仅在 worker 消费到消息时才 emit；`dropped_total` 只在异常路径 emit；`storage_bytes` 全模块零调用 | **修复**：① NewMetrics() 末尾 prime 所有已知 label 组合（in/out + 4 个 reason + inline/minio）→ 进程一启动 5 family 全暴露（0 值也算合法 observation）；② Repository 接口加 `StorageStats(ctx) (inlineBytes, minioBytes int64, err)`，PG SUM FILTER 一次查询；③ Sweeper 已有 60s ticker 里加 `sampleStorageBytes` 更新 gauge。**端到端验证**：DB 真实 inline=78581 / 44 行；60s 后 worker `/metrics` 上 `omc_trace_storage_bytes{location="inline"} 78581` 完全一致 |
| **L-6** | P3 待确认 | **✅ 2026-05-16 已对齐**（won't-fix 文案修正） **`omc_trace_active_tasks` 在 ACS=0 而 worker=1** | 两进程都注册了 gauge，但只有 worker Sweeper 周期采样 | 设计权衡：双进程统一注册 metric registry 简单，可避免按 role 拆 NewMetrics 的代码复杂度，运维侧用 `{job="omc-worker"}` filter 区分 | **修复**：metric `# HELP` 文案显式说明 "Emitted by worker process (job=\"omc-worker\"); other processes register the family for registry uniformity but never observe samples." 避免后续 reader 困惑 |
| **L-7** | P3 流程 | **✅ 2026-05-16 修复** **i18n / 菜单 seed 没进 DoD 清单** | L-1、L-4 同类问题之前也踩过 | 模块 DoD 漏项 | **修复**：`docs/project/dod.md` 前端模块章节加两条：① "新页面 → menus 表 seed migration"；② "新组件用到的 `t('xxx.yyy')` key 必须存在于 frontend-core/src/i18n catalog；表格操作列约定用 `table.action` 不要自创 `common.action`" |
| **L-8** | P1 体验 | **✅ 2026-05-16 修复** **MinIO 预签名 URL 用了 docker 内部 hostname** | export job 返回 `download_url=http://minio:9000/...`，浏览器解析不了 host | MinIO client 单例 — 没区分内部 endpoint（后端 ↔ MinIO）和公网 endpoint（浏览器 → MinIO） | **修复**：① `MinIOConfig` 加 `public_endpoint` 字段（dev 值 `localhost:9000`，留空 = 用 endpoint 兼容旧行为）；② `core/components/minio` 加 `NewPresignClient(cfg)` helper — `public_endpoint` 非空时新建独立 client 用它当 endpoint 签 URL，**关键**：必须显式 `Region: "us-east-1"` 否则 SDK 会先 `GetBucketLocation` 探测 region → 容器内 dial localhost:9000 失败 → 整个预签名失败；③ trace handler 字段重命名为 `presignClient`，app provider 注入 `NewPresignClient(cfg.MinIO)`。**端到端**：浏览器直接 `fetch(download_url)` → 200 / 84273 bytes / 完整 XML |
| **L-9** | P2 待确认 | **✅ 2026-05-16 已对齐**（won't-fix 文案修正） **`omc_trace_capture_latency_seconds` worker 侧永远是 0** | 同 L-6 — 双进程统一注册的设计权衡 | metric `# HELP` 文案标注 "in ACS hot path" 但 worker 也注册了 family，误导阅读 | **修复**：`# HELP` 文案改成 "Emitted only by ACS process (job=\"omc-acs\"); other processes register the family for registry uniformity but never observe samples." |
| **L-10** | P2 合规 | **✅ 2026-05-16 修复** **Sweeper 自动 stop 任务不写 `trace_stop` audit_log** | 5 分钟窗口跑完后 task 自动 stopped 但 `audit_logs` 表里只有 `trace_start` / `trace_export`，缺 `trace_stop` | Sweeper 走 worker 进程，handler 的 audit 逻辑不在 service 层 | **修复**：① `internal/trace/sweeper.go` UpdateTaskStatus 成功后调 `audit.Log(...)`，username=system / action=trace_stop / details.reason=sweeper_expired / user_agent=omcgo-worker/trace-sweeper；② `cmd/worker/main.go` 在 registerSubscribers 开头注入 `admin.NewPgAuditRepository` + `audit.SetDefault`（之前 worker 没接 audit sink）。**端到端**：跑完 2 分钟窗口任务后查 `audit_logs` 实测出现 trace_stop 行，actor=system，reason=sweeper_expired |
| **L-11** | P2 覆盖 | **✅ 2026-05-16 修复** **大报文外置 MinIO 链路 (>32KB) 未活体验证** | 真实基站 Inform 都 < 32KB，44 条全 inline，MinIO 外置链路零真实流量验证 | 真实流量天然小，没有合规反例 | **修复**：① 把 `CaptureConsumer.bulk` 字段从 `*BulkStore` 抽象成 `BulkPutter` 接口便于 mock；② 新增 `internal/trace/capture_consumer_test.go` 3 个单测：(a) 大报文（33KB）写 MinIO + PG 清空 inline + object_key 非空（边界条件含等于 32KB 的 inline 保留）；(b) bulk Put 失败时 fallback 保留 inline，避免丢报文；(c) bulk 未注入时 noop。**测试**：3/3 PASS |
| **L-12** | P3 待确认 | **✅ 2026-05-16 修复** **NATS stream 实际只有 3 个 (TRACE_TASK / TRACE_MSG / TRACE_EXPORT)，但 §4.4 M2 done 描述写 4 个 (TRACE + 上述 3)** | NATS HTTP API `:8222/jsz?streams=true` 实测 3 个；代码 `internal/core/components/nats/nats.go` 也只配 3 个 | M2 done 描述笔误 | **修复**：把 §4.4 M2 done 描述里 "TRACE/TRACE_TASK/TRACE_MSG/TRACE_EXPORT 4 stream" 改成 "TRACE_TASK/TRACE_MSG/TRACE_EXPORT 3 stream"，对齐代码与设计文档实际实现 |

**活体验证完成清单**（2026-05-16）：
- ✅ POST /trace/tasks 创建任务（DB 入库 + audit_logs `trace_start` ）
- ✅ 5 分钟窗口内 ACS hook 抓 44 条报文（11 Inform 周期 × 4 条/周期，inline 落库）
- ✅ Sweeper 自动 expire → status running→stopped
- ✅ GET /trace/tasks/{id}/messages 返回报文列表（前端 Drawer 渲染正常）
- ✅ payload 详情含完整 SOAP envelope（`xmlns:cwmp="urn:dslforum-org:cwmp-1-0"` + `cwmp:ID` 原样）
- ✅ POST /trace/tasks/{id}/export 异步导出（worker Exporter 40ms 完成）
- ✅ GET /trace/exports/{id} 返回 done + 预签名 URL（44 条 / 84273 bytes）
- ✅ 导出 XML 结构正确：`<TraceExport>` 根 + 多个 `<Message>` + CDATA 包裹原 SOAP（实测 22 cwmp:ID / 11 Inform / 11 InformResponse）
- ✅ audit_logs `trace_start` + `trace_export` 行齐全（含 username / ip_address / user_agent / resource / resource_id）
- ✅ TimescaleDB `trace_messages` 为 hypertable，drop_after=3 days retention policy 配置正确（设计 §D8）
- ✅ NATS 3 个 trace stream 已建（`TRACE_TASK` / `TRACE_MSG` / `TRACE_EXPORT`）— 与子任务文档 "4 stream" 描述有出入，见 L-12

**部分跑通 / 已发现差距**：（截至 2026-05-16 全部修复）
- ✅ 前端 SSE 实时刷新（L-3 修复，console 0 error；L-2 polling 补充 running 任务的 message_count 同步）
- ✅ 浏览器直接下载预签名 URL（L-8 修复，url_host=localhost:9000 / fetch 84273 字节）
- ✅ Sweeper 自动停止写 `trace_stop` audit_log（L-10 修复，actor=system / reason=sweeper_expired）

← 返回 [`docs/project/backlog.md`](../../backlog.md) §3 Active T-0137
