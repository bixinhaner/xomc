# Verify Report — T-0076 backup cleanup Phase 2 物理文件删除

> **Task**: T-0076 / R-102 / Sprint-07
> **Branch**: main / pre-commit @ fe64b7bc
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0076-backup-cleanup-phase2-physical-delete.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `go build ./...` | ✅ | clean |
| `go test -race ./internal/backup/... -count=1` | ✅ | 0 fail；新增 5 case (PhysicalDelete_AllSuccess / PartialFailure / MalformedPathSkipped / NoMinIONoCalls / classifyMinIORemoveErr) |
| `go vet` | ✅ | clean |
| 5 既有 mock 实现 TaskRepository | ✅ | execTaskRepo / fakeTaskRepo / mockTaskRepo / monTaskRepo / fakePrefixRepo 全更新 CleanupOldRows 签名 |
| 既有 4 PolicyMonitor 测试用例 | ✅ | cleanupFn 闭包返回 ([]string, int64, error) 同步更新 |
| Migration | ✅ N/A | 复用既有 backup_tasks.file_path 列；无 schema 变更 |
| Metric grep | ✅ | 2 新 metric 名（omc_backup_file_deleted_total + omc_backup_file_delete_errors_total）全 grep 命中（含 dashboard JSON） |
| Dashboard JSON | ✅ | node 解析通过；panels 12 → 18 (+6 backup panels) |

---

## 文件改动清单

修改：
- `omcgo/internal/backup/repository.go` — TaskRepository.CleanupOldRows 签名改为 `([]string, int64, error)` + 注释明示 file_path 过滤逻辑
- `omcgo/internal/backup/pg_repository.go` — DELETE ... RETURNING file_path 流式扫描；nullable 处理；filter NULL/empty 条目
- `omcgo/internal/backup/policy_monitor.go` — 包级注释 T-0073 → T-0073+T-0076；MinIOObjectRemover narrow interface + minio 字段 + SetMinIO；RunCleanupOnce 加物理删除 loop + 多设备 orphan info-log；physicallyDeleteOne + classifyMinIORemoveErr
- `omcgo/internal/backup/policy_metrics.go` — PolicyMetrics +2 collector (fileDeletedTotal / fileDeleteErrorsTotal) + RecordFileDeleted / RecordFileDeleteError 方法
- `omcgo/internal/backup/policy_alarm_publisher.go` — 更新 TODO 注释（T-0076 → T-0084；punted reasoning）
- `omcgo/cmd/app/provider/modules.go` — `backupPolicyMonitor.SetMinIO(c.MinIO)` 一行 DI
- `omcgo/internal/backup/policy_monitor_test.go` — +`fakeMinIORemover` + 5 新 case；既有 4 case 闭包签名同步
- `omcgo/internal/backup/executor_test.go` `handler_test.go` `service_test.go` `file_path_recorder_test.go` — 4 mock CleanupOldRows 签名同步
- `deployments/monitoring/grafana/dashboards/omc-overview.json` — +6 panels 在 y=36/44/52 三行（cleanup outcomes / rows deleted / files deleted / file delete errors / failure alarms / compression throughput）

无新增文件（除 PRD/verify 报告本身）。

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 CleanupOldRows 返回 deleted file_paths | 真实 PG 集成测试需 docker；schema-level 验证通过 squirrel ToSql + 流式 Rows scan 设计 | ✅ 设计 + mock |
| V2 PolicyMonitor 调 MinIO RemoveObject best-effort | TestPhysicalDelete_PartialFailure（NoSuchKey + network + ok 三种混合）| ✅ |
| V3 file_path NULL 跳过物理删除 | repo SQL filter at scan + TestNoMinIONoCalls 兜底 | ✅ |
| V4 Cleanup metrics 仪表盘可视 | omc-overview.json +6 panel；node JSON parse 通过 | ✅ |
| V5 Email routing 文档化 | PRD §9.4 给出 SQL 示例 + env vars 引用 | ✅ |
| V6 多设备 backup_task 物理删除部分成功 | RunCleanupOnce 输出 rows_without_file info-log + T-0083 followup 登记 | ✅ |
| V7 Backward compat | 既有 11 case + 4 cleanup case 全通过；mock 闭包签名同步 | ✅ |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| Scope 收紧（4 sub-component → 2）| ✅ 物理删除 + dashboard 合 1 task；磁盘阈值 / 多设备 reaper / email 拆为 N/A 或 followup |
| Email routing N/A — alarm 引擎已就绪 | ✅ PRD §9.4 仅 operator runbook |
| 多设备 orphan accept + 拆 T-0083 | ✅ RunCleanupOnce 加 rows_without_file info-log |
| Severity TODO punt（不 inline 修） | ✅ TODO(T-0076) → TODO(T-0084) + 注释解释 |
| MinIO field optional + nil-safe | ✅ TestNoMinIONoCalls；既有 T-0073 单元测试零回归 |
| classifyMinIORemoveErr 三类粒度 | ✅ not_found / network / other + dedicated unit test |

---

## 备注

- **Backward compat 关键设计**：MinIO 字段 optional + SetMinIO 后置 wire — 既有不传 MinIO 的测试场景零回归
- **best-effort 错误分类**：classifyMinIORemoveErr 用 `minio.ToErrorResponse` + 字符串匹配回退；3 类粒度足够 metric 仪表盘
- **多设备 orphan info-log**：每次 cleanup tick 一行，含 `rows_without_file` count；T-0083 实施后可移除
- **dashboard panels x/y 排布**：6 新 panels 排在 y=36..56 三行；不冲突现有 panels；保持 Grafana schema v39 不动

---

## S5 Code Review 复核（review-agent verdict: APPROVE — 0 critical / 0 high / 2 medium accept / 1 low optional）

| 严重度 | 问题 | 修复 | 状态 |
|--------|------|------|------|
| M1 | classifyMinIORemoveErr AccessDenied → `other`（不归 `auth`） | 接受；MVP 减少 reason cardinality；如 dashboard 显示 other 主要是 AccessDenied 再加 auth label | ⏸ accept |
| M2 | 多设备 orphan info-log 在 @daily cron 下 365 lines/year — 不噪 | 接受 | ⏸ accept |
| L1 | physicallyDeleteOne 返 int 而非 bool | 风格偏好；保留 int sum 习惯 | ⏸ accept |

**review-agent 7 项 verify 全部确认正确**：
1. Transaction 语义（DELETE → 后 RemoveObject）— 明确 trade-off：MinIO 不阻塞 DB 清理；orphan 通过 T-0082+T-0083 reaper 闭环
2. classifyMinIORemoveErr 三类粒度 + ctx.Canceled→network 分类合理
3. CleanupOldRows []string 内存压力 50k×80B ≈ 4MB @daily 可接受；100k+ 规模再加 BATCH_LIMIT
4. rows.Err() 检查存在（pg_repository.go:178-180）
5. 5 mock 签名更新行为不变
6. Dashboard PromQL 全部 metric name 与 code 1:1 对齐
7. SetMinIO timing 仅理论 race（DI 先于 Start，且 cron @daily 数小时后才触发）

**最终复核**：
- `go build ./...` ✅
- `go test -race ./internal/backup/... -count=1` ✅
- 既有 11 case + 4 cleanup case + 5 新 physical-delete case 全过
- Grafana dashboard JSON node parse 通过；6 panels 加入
- 无 P0/P1 阻塞项
