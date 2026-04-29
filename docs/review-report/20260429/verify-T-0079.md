# Verify Report — T-0079 backup_task → file_path 链路回填

> **Task**: T-0079 / R-102 / Sprint-07
> **Branch**: main / pre-commit @ cd086009
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0079-backup-task-filepath-linkage.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `go build ./...` | ✅ | clean |
| `go test -race ./internal/backup/... ./internal/acs/upload/... ./internal/acs/download/...` | ✅ | 0 fail；新增 6 case file_path_recorder + 5 case restore_service CreateByTaskID |
| `go vet` | ✅ | clean on touched packages |
| 4 既有 mock 实现 TaskRepository | ✅ | execTaskRepo / fakeTaskRepo / mockTaskRepo / monTaskRepo 全加 UpdateFilePath + FindByIDPrefix stub |
| 新端点 vs E2E claim 比 | ✅ | 1 新路由 (POST /backup/restore/by-task-id) + 1 claim (bk-11)；E/R = 1 |
| Migration | ✅ N/A | 复用现有 backup_tasks.file_path 字段，无 schema 变更 |
| Metric grep | ✅ | 2 metric 名 (omc_backup_filepath_recorded_total + omc_backup_restore_by_task_total) 全部 grep 命中 |

---

## 文件改动清单

新增：
- `omcgo/internal/backup/file_path_recorder.go`（~135 行）— FilePathRecorder + Subscribe + handleFileReceived（解码 → 找 task by prefix → first-write-wins → UpdateFilePath）
- `omcgo/internal/backup/file_path_recorder_test.go`（~165 行）— 6 case：recorded / skipped_already_set / empty prefix / no-match / db error propagate / update race vs cleanup

修改：
- `omcgo/internal/core/event/subjects.go` — +`SubjectBackupFileReceived` 常量 + 注释
- `omcgo/internal/backup/executor.go` — `taskIDPrefix` helper + Upload params 加 `target_file_name="backup-{taskID8}-{SN}.xml"` + `CommandKey="backup-{taskID8}"` + `SourceID=task.ID.String()`
- `omcgo/internal/backup/executor_test.go` — 加断言 target_file_name + SourceID + CommandKey + 加 `strings` import + 4 mock 加 stub
- `omcgo/internal/backup/handler_test.go` — fakeTaskRepo 加 stub
- `omcgo/internal/backup/service_test.go` — mockTaskRepo 加 stub
- `omcgo/internal/backup/policy_monitor_test.go` — monTaskRepo 加 stub
- `omcgo/internal/backup/repository.go` — TaskRepository interface +UpdateFilePath +FindByIDPrefix
- `omcgo/internal/backup/pg_repository.go` — 实现 UpdateFilePath（squirrel UPDATE）+ FindByIDPrefix（`REPLACE(id::text, '-', '') LIKE prefix%`）
- `omcgo/internal/backup/restore_metrics.go` — RestoreMetrics +2 collector（filePathRecorded + restoreByTaskTotal）+ 2 Record 方法
- `omcgo/internal/backup/restore_service.go` — `BackupTaskFinder` narrow interface + `SetBackupTaskFinder` + `CreateByTaskID` + `splitBucketAndPath` 工具 + `errors` import
- `omcgo/internal/backup/restore_service_test.go` — 5 新 case (CreateByTaskID + splitBucketAndPath)
- `omcgo/internal/backup/restore_handler.go` — 新 `CreateRestoreByTaskID` handler
- `omcgo/internal/backup/handler.go` — 注册 `POST /backup/restore/by-task-id` 路由
- `omcgo/internal/acs/upload/handler.go` — `regexp` import + `publishBackupFileReceivedEvent` + `parseBackupFilename` + 在 FileType=3 分支调用（与 FileType=11 datamodel 完全对称）
- `omcgo/cmd/app/provider/modules.go` — DI: SetBackupTaskFinder(backupTaskRepo) + NewFilePathRecorder(...).Subscribe(EventBus)
- `omcgo/scripts/e2e_verify.sh` — 加 W2D bk-11 claim

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 executor 设置 target_file_name | TestHandleTask_PushUploadCommand 断言 `target_file_name` 含 `backup-` 前缀 + `-SN001.xml` 后缀 + SourceID + CommandKey | ✅ |
| V2 upload handler 发布事件 | publishBackupFileReceivedEvent + parseBackupFilename regex `^backup-([0-9a-f]{8})-(.+)\.xml(\.[a-z0-9]+)?$` | ✅ |
| V3 backup module 订阅 + UpdateFilePath | TestHandleFileReceived_recorded（end-to-end mock） | ✅ |
| V4 multi-device first-write-wins | TestHandleFileReceived_skippedAlreadySet（已 set FilePath 不再覆盖）| ✅ |
| V5 POST /backup/restore/by-task-id 新 endpoint | TestCreateByTaskID_resolvesAndDispatches | ✅ |
| V6 by-task-id file_path NULL 拒绝 | TestCreateByTaskID_filePathNullRejected → ErrNotFound | ✅ |
| V7 multi-device task warning | TestCreateByTaskID_multiDeviceWarning（warning 含 "multi-device" + "first-write-wins"）| ✅ |
| V8 backward compat | T-0072 既有 explicit-path endpoint + e2e bk-9/bk-10 不变 | ✅ |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| EventBus pub-sub（不直接耦合 ACS↔backup） | ✅ acs/upload 仅 import `internal/core/event` + backup 模块 import `internal/core/event` 订阅；零跨模块直连 |
| backup_task_id 嵌入 filename `backup-{taskID8}-{SN}.xml` | ✅ executor.go 设 target_file_name + handler 用 regex 解析 + 兼容 .gz/.zst/.lz4/.bz2 压缩扩展名 |
| first-write-wins 多设备语义 | ✅ FilePathRecorder.handleFileReceived 检测 FilePath != nil → skip + 记 metric |
| restore_by_task_id 新 endpoint | ✅ POST /backup/restore/by-task-id；FilePath null → 404；multi-device → response.warning |
| 跨进程事件一致性 | ✅ Subscribe 用 QueueSubscribe queue group "backup-file-recorder"；多 App 实例只一个 consumer |

---

## 备注

- **8 hex char prefix collision 概率**：~1 in 4 billion；在 65k 任务规模才有 50% 冲突；FindByIDPrefix limit=2 + 拿最新 created_at 已防御
- **filename pattern 兼容压缩扩展**：regex 末尾 `(\.[a-z0-9]+)?` 兜底 .gz/.zst/.lz4/.bz2（T-0074/T-0077 产物）
- **T-0078 FE 升级路径明确**：T-0078 N3 (file 浏览器升级) 现可消费 backup_task.file_path 列；不在本任务内做（FE-only 任务）
- **历史 backup_tasks 不回填**：forward-only — T-0079 之前的老任务 file_path 仍 NULL；老任务用 explicit-path 模式 restore（T-0072 已存在）
- **跨进程 consumer 一致性**：app 进程 + worker 进程都可订阅，但本任务只在 app DI 中 Subscribe（worker 不重复订阅，避免 NATS queue group 同名导致一致性混乱）

---

## S5 Code Review 复核（review-agent verdict: APPROVE-WITH-FIXES → APPROVE）

| 严重度 | 问题 | 修复 | 状态 |
|--------|------|------|------|
| **HIGH** | first-write-wins 是 TOCTOU 假命题：read FilePath nil → 检查通过 → 同时另一 recorder 也 read 到 nil → 双方都 UpdateFilePath，第二个静默覆盖第一个，实际是 last-write-wins。注释与实现矛盾 | UpdateFilePath 改原子 CAS：`UPDATE backup_tasks SET file_path=$1 WHERE id=$2 AND file_path IS NULL`；0 行受影响 → 二次 SELECT 区分 NotFound vs AlreadySet，返回 sentinel `ErrFilePathAlreadySet`；recorder switch err 路径 + 加 CAS-lost test case | ✅ Fixed |
| M1 | 正则边界 device SN 含 ".xml" 字面 | greedy `(.+)` + 后置 `\.xml` literal anchor 已正确处理 — accept | ⏸ accept |
| M2 | FindByIDPrefix `REPLACE(id::text,'-','') LIKE ?` 走 seq scan | squirrel 参数化 → 无注入；MVP 规模可接受 | ⏸ accept |
| M3 | 空 prefix 仍 publish event 走 decode | 一次 decode 成本极小；保 event 语义清晰；OK | ⏸ accept |
| L1-L4 | warning 仅信息 / bk-11 4xx 表面 / Subscribe 未单测 / target_file_name basename 已正确 / queue group ChannelEventBus 单订阅者 no-op | 全部 verify safe | ⏸ accept |

**最终复核**：
- `go build ./...` ✅
- `go test -race ./internal/backup/... -count=1` ✅ all green（含新增 TestHandleFileReceived_updateAlreadySetIsSkip）
- 7 case file_path_recorder + 5 case CreateByTaskID 全过
- 无 P0/P1 阻塞项

**HIGH fix 关键设计**：
- 把"first-write-wins"语义从应用层错位的 read-then-write 移到 DB 层 CAS（`WHERE file_path IS NULL`）— 唯一正确的并发控制位置
- recorder.go 的 fast-path read（`if FilePath != nil`）保留为优化（避免无谓 UPDATE round-trip），但**不是**正确性边界
- ErrFilePathAlreadySet sentinel 让消费者区分 CAS-lost（idempotent 跳过）vs. 真错误
- 0 接口签名变更（避免 4 mock 重写）— 通过 sentinel error 优雅扩展
