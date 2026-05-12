# T-0101..T-0112 拆分子任务（F06 运维管理）

> **2026-05-11 状态更新**：12 umbrella 全部已 done（commits ce36108f/6e42a664/c0485129 MVP 落地，详 `docs/project/backlog/done/2026Q2.md`）；本表下方 66 sub-task 仍处 proposed，待 sprint planning 拍板 Q1-Q6 后升 planned 推进生产级实施（剩余 ~20-25 工作日）。MVP 边界与二期清单见每个 umbrella 在 done/2026Q2.md 的 Closing Evidence 行。
>
> 从 `docs/project/backlog.md` §5 Proposed 12 个 umbrella 拆出（2026-05-10）。
> 来源 PRD: `docs/project/prd/F06-ops-management.md`（847 行）
> 推进计划: `docs/project/F06-ops-management-implementation-plan.md`（347 行 / 8 Sprint × 16 周 / ~38 工作日）
>
> umbrella 行仍在主 backlog §5 Proposed，sub-task 状态联动 Sprint planning 推进。
>
> **行 schema** 与 backlog.md §3 Active 主表对齐（13 列）。State 默认 proposed；Sprint/Owner 待下周 Triage 拍板 Q1-Q6 后回填。
>
> **数量**：12 umbrella + 66 sub-task。详情：
> - T-0112 Foundation: 4 (a..d)
> - T-0101 调度引擎: 9 (a..i) — W1 P0 关键路径
> - T-0102 SSE 通道: 5 (a..e) — W1 P0
> - T-0103 前端补齐: 6 (a..f) — W1 P0
> - T-0104 诊断: 9 (a..i) — W3 P1
> - T-0105 下载: 7 (a..g) — W3 P1
> - T-0106 审批流: 5 (a..e) — W4 P1
> - T-0107 维护窗口: 5 (a..e) — W4 P2
> - T-0108 健康巡检: 4 (a..d) — W5 P2
> - T-0109 审计归档: 4 (a..d) — W4 P2
> - T-0110 知识库: 4 (a..d) — W5 P3
> - T-0111 break-glass: 4 (a..d) — W6 P3

---

## T-0112 Foundation（Sprint 1，~5 天，P0 解锁全部）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0112-a | migrations/000080_ops_extensions.sql — ops_templates + ops_tasks 扩展 12 列 | feat | F06/ops | P0 | proposed | — | S | Q3,Q4 | — | PRD §7.1 | 2026-05-10 | risk_level/version/change_log/owner_user_id/rollback_steps/target_carriers + template_snapshot/approval_state/approver_user_id/approved_at/batch_config/failure_policy |
| T-0112-b | migrations/000081_ops_new_tables.sql — 6 张新表 + 索引 + 触发器 | feat | F06/ops | P0 | proposed | — | M | — | — | PRD §7.2 | 2026-05-10 | ops_task_executions / ops_diagnostics / ops_downloads / ops_audit_logs / ops_maintenance_windows / ops_playbooks |
| T-0112-c | seed/000082_seed_ops_permission_points.sql — 8 个新权限点 menus + role_menus 调整 | feat | F06/ops+admin | P0 | proposed | — | S | T-0112-a | — | PRD §8.1 | 2026-05-10 | ops:command:safe/cautious/dangerous + task:approve + diagnostic:run + download:trigger + maintenance:plan/approve + audit:view + break_glass |
| T-0112-d | seed/000083_seed_ops_builtin_templates.sql + Carrier.OpsAdapter 接口骨架 | feat | F06/ops+carrier | P0 | proposed | — | M | T-0112-a | — | PRD §4.1.3 + §9 | 2026-05-10 | 6 内置 OEM 模板（GPS 失锁恢复 / 时间同步 / 计划重启 / 设备隔离 / 现场协助包 / 软重启）+ OpsAdapter 接口（GetLogPath / MapDiagnosticType / DefaultApprovalPolicy）+ 3 空实现 |

---

## T-0101 TaskExecutor 调度引擎（Sprint 2，~5 天，P0）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0101-a | TaskExecutor 接口设计（small interface）+ DI 注入 — **done 2026-05-12**（commit `747552e5`；TaskExecutorEngine 6 方法接口（5 控制 Run/Pause/Resume/Cancel/Rollback + 1 查询 ListExecutions）+ 编译期断言 + handler 改依赖接口；Pause/Resume/Cancel 委托 T-0101-g TransitionStatus / Rollback 返 ErrNotImplemented 留 T-0101-h future）| feat | F06/ops | P0 | done | Claude | S | T-0112-a ✅, Q2=A ✅ | sprint-10 (pull-forward) | PRD §5.3.2 / R-O01 mitigation 接口抽象 | 2026-05-12 | Closing Evidence → `docs/review-report/20260512/verify-T-0101-a.md` + `REVIEW_T-0101-a_chenbo01_ops.md` |
| T-0101-b | 步骤路由器 — step.type → rpc/mml/wait/loop/branch dispatcher — **done 2026-05-12**（commit `90874970`；StepRouter + 5 StepType + ErrUnknownStepType + 5 默认 stub handler；StepWait 真实有效 time.After+ctx；6 单测含反退化）| feat | F06/ops+acs+mml | P0 | done | Claude | S | T-0101-a ✅, T-0112-d ✅ (MVP) | sprint-10 (pull-forward) | PRD §5.3.2 | 2026-05-12 |
| T-0101-c | 并发控制（max=20）+ 节流 — **done 2026-05-12**（commit `9c66867c`；ConcurrencyLimiter buffered chan 信号量 + rate.Limiter 防 ACS 雪崩；4 单测含 race + ctx 超时）| feat | F06/ops+task | P0 | done | Claude | S | T-0101-a ✅ | sprint-10 (pull-forward) | PRD §5.3.2 | 2026-05-12 |
| T-0101-d | 状态机集成（pending → approved → running → success/failed/...）— **done 2026-05-12**（commit `1743e8d1` 8 文件 +548/-32；ApprovalService.Approve 删 TODO 兑现承诺；OpsTask 扩 4 字段 + UpdateApproval 单 SQL atomic UPDATE + 5 unit test 反退化 + R-O01 mitigation 完整闭环）| feat | F06/ops | P0 | done | Claude | S | T-0112-a ✅ (MVP/live DB v87), Q3=A ✅ | sprint-10 (pull-forward) | PRD §5.3.1+§6.1+§8.2 / R-O01 mitigation 完整 | 2026-05-12 | Closing Evidence → `docs/review-report/20260512/verify-T-0101-d.md` + `REVIEW_T-0101-d_chenbo01_ops.md`；T-0101 umbrella 9 sub-task 进度 1/9（d 完成 / a/b/c/e/f/g/h/i 仍 proposed）|
| T-0101-e | 结果聚合 + ops_task_executions 写入（device × step 明细）— **done 2026-05-12**（commit `1e9d79dd`；TaskExecutionRepository +CountByTaskStatus GROUP BY SQL + TaskExecutor +RecordExecution dispatcher 调用面 + AggregateForTask PRD §5.3.3 进度公式 progress=(success+failed+skipped)*100/total + 0 除防御 + 4 新单测）| feat | F06/ops | P0 | done | Claude | S | T-0112-b ✅ (MVP) | sprint-10 (pull-forward) | PRD §5.3.3 + §7.2 | 2026-05-12 | Closing Evidence → `docs/review-report/20260512/verify-T-0101-e.md` + `REVIEW_T-0101-e_chenbo01_ops.md` |
| T-0101-f | 失败策略实现（abort / continue / retry-N / rollback）— **done 2026-05-12**（commit `799fc062`；PolicyDecider 无状态决策器 + 4 FailureAction 常量 + 指数退避 30s 钳位 + 重试用尽降级 abort + 未知 policy 走 abort 默认；8 单测全策略 happy/edge case 覆盖）| feat | F06/ops | P0 | done | Claude | M | T-0101-e ✅ | sprint-10 (pull-forward) | PRD §5.3.2 | 2026-05-12 |
| T-0101-g | 暂停/恢复/取消 dispatcher 集成 — **done 2026-05-12**（commit `02d01eea` 7 文件 +483/-78；TransitionStatus 单 SQL CAS 替换 GetByID+UpdateStatus 三步；Cancel 允许 from paused 行为扩展；disambiguate helper 保 NotFound vs 400 契约；dispatcher 协作 hook 留 T-0101-a/b future；11 反退化测试）| feat | F06/ops | P0 | done | Claude | S | T-0101-d ✅ | sprint-10 (pull-forward) | PRD §5.3.1 / 并发 race 防御完整 | 2026-05-12 | Closing Evidence → `docs/review-report/20260512/verify-T-0101-g.md` + `REVIEW_T-0101-g_chenbo01_ops.md`；T-0101 umbrella 9 sub-task 进度 **2/9**（d+g 完成 / a/b/c/e/f/h/i 仍 proposed）|
| T-0101-h | 回滚引擎（执行 rollback_steps 反向序列）— **done 2026-05-12**（commit `2baeb8a5`；TaskExecutor +stepRouter 字段 + SetStepRouter setter + Rollback 替换 ErrNotImplemented stub + DispatchRollbackSteps 反向 iterate + fail-tolerant + RecordExecution 持久化；4 单测含反向顺序+失败容忍）| feat | F06/ops+alarm | P0 | done | Claude | S | T-0101-f ✅ | sprint-10 (pull-forward) | PRD §4.1.1 / §5.3.2 | 2026-05-12 |
| T-0101-i | 断点续传（进程重启 / 节点切换从 ops_task_executions 恢复）— **done 2026-05-12**（commit `7ae9a401`；TaskExecutor +RecoverPendingTasks 启动期扫描 running task → MVP 保守路线标 failed + 写 audit "task_recovered_as_failed" + 容忍 partial failure；4 单测覆盖恢复/空列表/单失败容忍/List 错误传播）| feat | F06/ops | P0 | done | Claude | S | T-0101-e ✅ | sprint-10 (pull-forward) | PRD §5.3.2 | 2026-05-12 |

---

## T-0102 即时命令 SSE 通道（Sprint 3，~3 天，P0）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0102-a | SSE Hub + 路由 — **done 2026-05-12**（commit `1de0a16d`；SSEHub +Stats() observability + 5 个新单测覆盖 Subscribe/Publish/Unsubscribe 生命周期 + fan-out 正确性 + 防内存泄漏；MVP 既有 streamSSE handler 含 nginx X-Accel-Buffering + 15s 心跳 + flusher + ctx cancel 长连接 robust 处理已完整）| feat | F06/ops | P0 | done | Claude | S | — | sprint-10 (pull-forward) | PRD §5.2.3 / R-O02 mitigation | 2026-05-12 | Closing Evidence → `docs/review-report/20260512/verify-T-0102-a.md` + `REVIEW_T-0102-a_chenbo01_ops.md`；T-0102 umbrella 5 sub-task 进度 1/5 |
| T-0102-b | command-execute endpoint（POST → 入队 → 返 task_id） | feat | F06/ops | P0 | proposed | — | S | T-0101-*, T-0102-a | — | PRD §7.2 §4.2.2 | 2026-05-10 | /api/v1/ops/commands/rpc；可批量发起 |
| T-0102-c | RPC 命令支持 — reboot/factory_reset/get_param/set_param/get_rpc_methods | feat | F06/ops+acs | P0 | proposed | — | M | T-0102-b | — | PRD §4.2.1 | 2026-05-10 | 风险等级 L1/L2/L3 自动评估 |
| T-0102-d | MML 流式输出（多帧）通过 SSE 推送 | feat | F06/ops+mml | P0 | proposed | — | S | T-0102-c | — | PRD §4.2.3 | 2026-05-10 | 设备返多帧时实时推；保持 MML 模块独立入口（Q1=B 双通道共存）|
| T-0102-e | 超时 + 重试 + per-device 限流 | feat | F06/ops | P0 | proposed | — | S | T-0102-c | — | PRD §4.2.3 | 2026-05-10 | 单命令默认 60s 超时；批量场景设备级独立超时 |

---

## T-0103 前端 W1 MVP 补齐（Sprint 3，~2 天，P0）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0103-a | Templates Edit modal + version + change_log UI | feat | frontend | P0 | proposed | — | S | T-0112-a | — | PRD §4.1.1 | 2026-05-10 | 当前只有 Create modal，缺 Edit；version 自增 + change_log 追加 |
| T-0103-b | 模板复制（POST /duplicate）+ JSON 导入导出 | feat | frontend+ops | P0 | proposed | — | S | T-0112-a, Q6 | — | PRD §4.1.2 | 2026-05-10 | Q6=B 首版不支持跨实例同步，仅本实例导入导出 |
| T-0103-c | 风险等级 UI（safe/cautious/dangerous Tag + 编辑入口）| feat | frontend | P0 | proposed | — | S | T-0112-a, Q5 | — | PRD §4.2.2 | 2026-05-10 | Q5=B 仅文档约定，UI 用枚举下拉，不上 JSON Schema |
| T-0103-d | Tasks 设备级流水视图（ops_task_executions 数据可视化）| feat | frontend | P0 | proposed | — | S | T-0101-e | — | PRD §5.3.3 | 2026-05-10 | 详情抽屉新增"执行流水"Tab；展示 device × step 表格 |
| T-0103-e | Tasks SSE 实时进度页（进度条 + 设备状态实时）| feat | frontend | P0 | proposed | — | S | T-0102-a | — | PRD §5.3.3 | 2026-05-10 | 订阅 /api/v1/ops/tasks/:id/events SSE |
| T-0103-f | CommandManagement 新建即时命令 UI（按钮 + Modal + SSE 实时输出）| feat | frontend | P0 | proposed | — | M | T-0102-* | — | PRD §4.2.4 | 2026-05-10 | 快速操作菜单 + 批量操作；输出窗口仿终端样式 |

---

## T-0104 网络诊断子系统（Sprint 4-5，~5 天，P1）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0104-a | DiagnosticsService 骨架 + ops_diagnostics 表接入 | feat | F06/ops | P1 | proposed | — | XS | T-0112-b, Q4 | — | PRD §5.4 | 2026-05-10 | schema 已在 T-0112-b；本任务建 service + repo |
| T-0104-b | TR-181 IPPingDiagnostics（SPV → 等事件码 8 → GPV 拉结果） | feat | F06/ops+acs | P1 | proposed | — | M | T-0104-a, T-0112-d | — | PRD §5.4.1 | 2026-05-10 | 流程：写 DiagnosticsState=Requested → 设备 inform 8 → 拉结果 |
| T-0104-c | TraceRouteDiagnostics 接入 | feat | F06/ops+acs | P1 | proposed | — | S | T-0104-b | — | PRD §5.4.1 | 2026-05-10 | RouteHops[] 解析 |
| T-0104-d | Download/UploadDiagnostics（测带宽）| feat | F06/ops+acs | P1 | proposed | — | M | T-0104-b | — | PRD §5.4.1 | 2026-05-10 | TotalBytesReceived/Sent + Throughput |
| T-0104-e | UDPEchoDiagnostics | feat | F06/ops+acs | P1 | proposed | — | S | T-0104-b | — | PRD §5.4.1 | 2026-05-10 | RTT 测试 |
| T-0104-f | OMC 侧诊断 — reachability / 同站连通 / GPS 聚合 / 邻区一致 / 告警风暴 | feat | F06/ops | P1 | proposed | — | M | T-0104-a | — | PRD §5.4.2 | 2026-05-10 | 不依赖设备执行 |
| T-0104-g | 诊断结果时序图可视化 + 包导出（PDF/JSON）| feat | F06/ops+frontend | P1 | proposed | — | M | T-0104-b..f | — | PRD §5.4.3 | 2026-05-10 | 多 hop / 多 packet 时序 |
| T-0104-h | 厂商扩展 DiagnosticsAdapter（Carrier.MapDiagnosticType）| feat | F06/ops+carrier | P1 | proposed | — | S | T-0112-d | — | PRD §5.4.4 + §9 | 2026-05-10 | 防止 if carrier == "..." 硬编码 |
| T-0104-i | 前端 NetworkDiagnosis 接 API（替换占位卡）| feat | frontend | P1 | proposed | — | M | T-0104-b..g | — | PRD §5.4 | 2026-05-10 | 当前 6e42a664 已 stub 占位卡 |

---

## T-0105 运维下载子系统（Sprint 5，~5 天，P1）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0105-a | DownloadService 骨架 + ops_downloads 表接入 | feat | F06/ops | P1 | proposed | — | XS | T-0112-b | — | PRD §5.5 | 2026-05-10 | schema 已在 T-0112-b |
| T-0105-b | 7 类内容采集（config/log/pm/mr/诊断包/pcap/gps）via transfer.Upload | feat | F06/ops+transfer | P1 | proposed | — | L | T-0105-a | — | PRD §5.5.1 | 2026-05-10 | 复用 internal/transfer 文件桥 |
| T-0105-c | 打 zip 包（并发取多文件 → 合包 → MinIO 存）| feat | F06/ops+minio | P1 | proposed | — | M | T-0105-b | — | PRD §5.5.2 | 2026-05-10 | OMC 端 zip，不在设备打 |
| T-0105-d | MinIO presigned URL（1h TTL）+ 浏览器 Range 请求 | feat | F06/ops+minio | P1 | proposed | — | S | T-0105-b | — | PRD §5.5.2 | 2026-05-10 | 限时签名 URL，避免公网读权限 |
| T-0105-e | 过期清理 cron（默认 90 天归档冷存 / 删除）| feat | F06/ops | P1 | proposed | — | S | T-0105-a | — | PRD §5.5.2 | 2026-05-10 | 可配置；config.dev.yaml retention_days |
| T-0105-f | 断点续传（transfer 失败重传）| feat | F06/ops+transfer | P1 | proposed | — | S | T-0105-b | — | PRD §5.5.2 | 2026-05-10 | 复用 transfer 既有能力 |
| T-0105-g | 前端 Downloads 接 API（替换占位卡）+ 下载篮 | feat | frontend | P1 | proposed | — | M | T-0105-b..d | — | PRD §5.5.3 | 2026-05-10 | 跨设备多选下载篮 |

---

## T-0106 4 眼审批流（Sprint 6，~4 天，P1）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0106-a | ops_tasks 审批列接入（approval_state/approver_user_id/approved_at）| feat | F06/ops | P1 | proposed | — | XS | T-0112-a | — | PRD §8.2 | 2026-05-10 | schema 已在 T-0112-a |
| T-0106-b | ApprovalService（approver ≠ creator）+ risk_level 自动评估 | feat | F06/ops+admin | P1 | proposed | — | M | T-0106-a | — | PRD §8.2 / R-O05 | 2026-05-10 | 4 眼原则：creator 不能 self-approve |
| T-0106-c | 审批通知接入（notification 中心，邮件 + UI 弹窗）| feat | F06/ops+notification | P1 | proposed | — | M | T-0106-b | — | PRD §8.2 | 2026-05-10 | 推送给非创建者 sys_admin |
| T-0106-d | 前端审批页（列表 / 详情 / 同意 / 拒绝 + Reason）| feat | frontend | P1 | proposed | — | M | T-0106-b | — | PRD §8.2 | 2026-05-10 | 详情含完整任务 + 影响面 + rollback 方案 |
| T-0106-e | 任务创建侧自动 risk 评估 + UI 提示"此任务需审批" | feat | frontend+ops | P1 | proposed | — | M | T-0106-b | — | PRD §8.1 | 2026-05-10 | 创建前预检 + 显示审批者候选 |

---

## T-0107 维护窗口（Sprint 6，~3 天，P2）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0107-a | ops_maintenance_windows 表接入 + Service | feat | F06/ops | P2 | proposed | — | XS | T-0112-b | — | PRD §6.2 | 2026-05-10 | schema 已在 T-0112-b |
| T-0107-b | 事件机制 — start/end 发 maintenance.window.* 事件 | feat | F06/ops+events | P2 | proposed | — | S | T-0107-a | — | PRD §6.2 | 2026-05-10 | 复用 internal/core/event EventBus |
| T-0107-c | 告警抑制集成（alarm 模块订阅 → suppressed=true）| feat | F06/ops+alarm | P2 | proposed | — | S | T-0107-b | — | PRD §6.2 / R-O06 | 2026-05-10 | 告警入库但不进活跃表 |
| T-0107-d | 暂停自动开站 + 允许高风险动作旁路 | feat | F06/ops+provision | P2 | proposed | — | S | T-0107-b | — | PRD §6.2 | 2026-05-10 | provision 订阅 → pause；ops 订阅 → allow_dangerous |
| T-0107-e | 前端 UI（列表 / 创建 / 审批 / active 列表）| feat | frontend | P2 | proposed | — | M | T-0107-a | — | PRD §6.2 | 2026-05-10 | 维护窗口列表 + 创建表单 + active 卡片 |

---

## T-0108 健康巡检（Sprint 7，~3 天，P2）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0108-a | cron 调度器（默认每周一 02:00）+ scope 选择 | feat | F06/ops | P2 | proposed | — | M | T-0104-* | — | PRD §6.1 | 2026-05-10 | 复用 robfig/cron/v3；scope=group/全网 |
| T-0108-b | 报告生成（PDF + CSV，存 MinIO）| feat | F06/ops+minio | P2 | proposed | — | M | T-0108-a | — | PRD §6.1 | 2026-05-10 | fpdf 库（已用于 license 导出）|
| T-0108-c | 邮件 / SMS 通知（notification 接入）| feat | F06/ops+notification | P2 | proposed | — | S | T-0108-b | — | PRD §6.1 | 2026-05-10 | 沿用 alarm 通知通道 |
| T-0108-d | 前端配置 UI（定时设置 / 历史报告下载）| feat | frontend | P2 | proposed | — | S | T-0108-b | — | PRD §6.1 | 2026-05-10 | cron 表达式编辑器 + 报告列表 |

---

## T-0109 审计归档（Sprint 6，~2 天，P2，建议先做）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0109-a | ops_audit_logs 表接入（独立于通用 audit_logs）| feat | F06/ops | P2 | proposed | — | XS | T-0112-b | — | PRD §6.3 | 2026-05-10 | schema 已在 T-0112-b；高频写隔离 |
| T-0109-b | 全量 hook — 所有 op_type 写日志（template/command/task/diagnostic/download/approval/break_glass）| feat | F06/ops+all | P2 | proposed | — | M | T-0109-a | — | PRD §6.3 | 2026-05-10 | 在每个 service 路径加 LogWriter |
| T-0109-c | 12 月保留 cron（超期归档到 MinIO，按月 JSONL.gz）| feat | F06/ops | P2 | proposed | — | S | T-0109-b | — | PRD §6.3 | 2026-05-10 | 复用 license_logs 归档器模式 |
| T-0109-d | 前端审计查询页（多维过滤 + CSV/JSON 导出）| feat | frontend | P2 | proposed | — | S | T-0109-b | — | PRD §6.3 | 2026-05-10 | 时间窗 / 操作者 / 设备 / 命令类型 / 结果 |

---

## T-0110 知识库 Playbook（Sprint 7，~2 天，P3）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0110-a | ops_playbooks 表接入 + Service | feat | F06/ops | P3 | proposed | — | XS | T-0112-b | — | PRD §6.5 | 2026-05-10 | schema 已在 T-0112-b |
| T-0110-b | 匹配引擎（alarm_pattern → recommended_templates，按 success_rate 排序）| feat | F06/ops+alarm | P3 | proposed | — | S | T-0110-a, T-0101-* | — | PRD §6.5 | 2026-05-10 | 用模板使用统计 |
| T-0110-c | 告警详情页"推荐处置"区块 + 一键应用模板 | feat | frontend | P3 | proposed | — | M | T-0110-b | — | PRD §6.5 | 2026-05-10 | 跳转到 Templates 详情 / 创建 Task |
| T-0110-d | 知识库管理 UI（PM/SysAdmin 维护）| feat | frontend | P3 | proposed | — | S | T-0110-a | — | PRD §6.5 | 2026-05-10 | CRUD playbook |

---

## T-0111 紧急响应 break-glass（Sprint 7-8，~3 天，P3）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0111-a | 临时权限激活流（工单号 + 理由 + 30 min 自动失效）| feat | F06/ops+admin | P3 | proposed | — | M | T-0109-* | — | PRD §6.6 | 2026-05-10 | 中间件层注入临时权限 grant |
| T-0111-b | 全程录像（命令 + 输出按时间戳进 ops_audit_logs.break_glass=true）| feat | F06/ops | P3 | proposed | — | S | T-0109-b | — | PRD §6.6 | 2026-05-10 | break_glass 字段在 T-0112-b schema |
| T-0111-c | 强制告警通知给安全负责人 | feat | F06/ops+notification | P3 | proposed | — | S | T-0111-a | — | PRD §6.6 / R-O07 | 2026-05-10 | severity=critical alarm + email/SMS |
| T-0111-d | 24h 复盘评注 + 未评注阻塞下次激活 | feat | F06/ops+frontend | P3 | proposed | — | M | T-0111-b | — | PRD §6.6 | 2026-05-10 | 阻断下次 break-glass 直至复盘 |

---

## 关联文档

- PRD：`docs/project/prd/F06-ops-management.md`
- 推进计划：`docs/project/F06-ops-management-implementation-plan.md`
- MVP 起点 commit：`6e42a664`
- DoD：`docs/project/dod.md`
- Risk Register（R-O01..R-O08 待登记）：`docs/project/risk-register.md`
- 待决议 Q1-Q6：PRD §13 / 推进计划 §1
