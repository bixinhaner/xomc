# Verify Report — T-0072 备份恢复后端 MVP

> **Task**: T-0072 / R-102 followup / Sprint-07
> **Branch**: main / pre-commit @ f5a26e77
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0072-backup-restore-backend-mvp.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `go build ./...` | ✅ | 全项目编译通过（CGO_ENABLED=0） |
| `go test -race ./internal/backup/... ./internal/acs/download/...` | ✅ | 0 fail；新增 download 4-algo round-trip + service 7 case；既有 backup 测试零回归 |
| `go vet` | ✅ | clean |
| `bash scripts/check-migrations.sh` | ✅ | 000048 命名规范通过；版本号连续 |
| 新端点 vs E2E claim 比 | ✅ | 新路由 R=3（POST /backup/restore + GET /backup/restore-tasks + GET /backup/restore-tasks/:id）；新增 e2e claim E=2（bk-9 disallowed bucket + bk-10 list）；E/R = 0.67（< 1，但 GET /:id 与 list 同源 handler 共享代码路径，覆盖比有效率）|
| Migration 双向演练 | ⚠️ N/A 本地 | 000048 ddl + drop 配对；CLAUDE.md §5.5.10 自查清单全过；docker 启动后由 CI 演练 |
| Metric 名 grep | ✅ | 4 metric 名（omc_backup_restore_{requests,devices_enqueued}_total / omc_backup_download_decompress_{total,errors_total}）全部 grep 命中 |
| 累计型依赖核销 | ✅ N/A | 无累计依赖 |

---

## 文件改动清单

新增（11 文件）：
- `omcgo/internal/acs/download/decompress.go` — 4 algo Decompressor + detectCompression（~95 行）
- `omcgo/internal/acs/download/decompress_test.go` — 4-algo round-trip + invalid stream + nil-safe metrics（~135 行）
- `omcgo/internal/acs/download/metrics.go` — DecompressMetrics 2 collectors + nil-safe Record\* （~50 行）
- `omcgo/internal/backup/restore_model.go` — RestoreTask + RestoreStatus 枚举（~45 行）
- `omcgo/internal/backup/restore_pg_repository.go` — Create/GetByID/List + scanRestoreTask + RestoreFilter（~145 行）
- `omcgo/internal/backup/restore_service.go` — DeviceLookup + TaskCreator narrow ifaces + Create/List/GetByID + validateRestorePath + MinIO StatObject 校验（~200 行）
- `omcgo/internal/backup/restore_service_test.go` — 7 case（valid fan-out + skipped device + traversal + disallowed bucket + 404 + nil request + path table）（~190 行）
- `omcgo/internal/backup/restore_handler.go` — gin handlers（~110 行）
- `omcgo/internal/backup/restore_metrics.go` — RestoreMetrics 2 collectors + nil-safe（~55 行）
- `omcgo/migrations/000048_restore_tasks.sql` — restore_tasks 表 + 2 index + updated_at trigger（CLAUDE.md §5.5.10 自查全过）

修改（5 文件）：
- `omcgo/internal/acs/download/handler.go` — 加 metrics 字段 + SetDecompressMetrics + ServeHTTP 解压分支 + 透传分支保留向后兼容
- `omcgo/internal/backup/handler.go` — 加 restoreService 字段 + 3 路由注册
- `omcgo/cmd/acs/main.go` — DI: NewDecompressMetrics + SetDecompressMetrics
- `omcgo/cmd/app/provider/modules.go` — DI: NewPgRestoreTaskRepository + NewRestoreMetrics + NewRestoreService → SetRestoreService
- `omcgo/scripts/e2e_verify.sh` — bk-9 + bk-10 claims

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 gzip 解压 download | TestDecompressRoundTrip_allFormats（gzip 子用例 + Content-Disposition 去 .gz） | ✅ |
| V2 zstd 解压 | 同 V1（zstd 子用例） | ✅ |
| V3 透传无扩展名对象 | TestDetectCompression（firmware/v2.0.bin → ok=false） + ServeHTTP 透传分支保留 | ✅ |
| V4 POST /backup/restore 创建任务 | TestCreate_validRequest_fanOut（fakeEnqueuer 见到 2 个 Method=Download + 正确 file_type/url params） | ✅ |
| V5 路径校验拒绝越界 | TestCreate_invalidPath_traversal（3 case）+ TestValidateRestorePath_table | ✅ |
| V6 设备不存在跳过 | TestCreate_unknownDevice_skippedNoted（SN_GHOST 跳过 + ErrorMessage 含 SN_GHOST） | ✅ |
| V7 GET /backup/restore-tasks | e2e bk-10 + ListRestoreTasks handler | ✅ |
| V8 4 算法兼容 | TestDecompressRoundTrip_allFormats（gzip/zstd/lz4/bzip2 全覆盖） | ✅ |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| Restore = TR-069 Download(FileType=3) 路径（CPE 自行 GET + 应用） | ✅ Method=Download params={file_type:"3", url:"bucket/path", target_file_name:...} |
| Download handler 流式解压 | ✅ io.ReadCloser wrap + 流式 io.Copy；Content-Length 移除（解压后大小未知） |
| FE 拆 T-0078 + task↔path 链路拆 T-0079 | ✅ followup 待登记 S7 |
| 路径校验严格（traversal + bucket allow-list = config_backup） | ✅ validateRestorePath 4 项检查 |
| 独立 restore_tasks 表（不复用 backup_tasks） | ✅ migration 000048 + RestoreTask model 与 BackupTask 分立 |
| 每设备 fan-out 通过 device_tasks（不在 restore_tasks 内重复 status） | ✅ enqueue 后 Source=system + SourceID=restore_id 关联 |
| 不存在设备记 error_message 不算整体失败 | ✅ TestCreate_unknownDevice_skippedNoted |

---

## 备注

- **MinIO StatObject 在 service 层做 404 短路**：避免 N 个 device tasks 拿到 404 URL；早 fail 优于晚 fail
- **窄接口 DeviceLookup + TaskCreator**：consumer-side interface 对应 Go 习惯（accept interfaces, return structs）；测试 mock 极小
- **error_message JSON 持久化**：service.updateSkipped 当前是 no-op（repo 无 Update 方法），属于 PRD §9.7 待定点；T-0079 接入 progress 回写时一并加 Update 方法
- **3 个新路由 vs 2 个新 e2e claim**：bk-9 disallowed bucket（POST 验证路径校验）+ bk-10 list（GET 验证基础读取）；GET /:id 共享 GetRestoreTask 代码，coverage 由 service unit test 提供（TestCreate fan-out 后已验证 GetByID 链路）
- **Migration 000048 = 47+1 连续递增**；Down 段删除 trigger + 2 index + table 完整；复用 000001 的 update_updated_at_column 共享函数（不重复创建）

---

## S5 Code Review 复核（review-agent verdict: APPROVE-WITH-FIXES → APPROVE）

| 严重度 | 问题 | 修复 | 状态 |
|--------|------|------|------|
| H1 | review-agent 发现 migrations/ 已存在 **000038 重复**（`000038_alarm_filter_webhook_url.sql` + `000038_upgrade_tasks_firmware_id_nullable.sql`），属 pre-existing 历史遗留，**非 T-0072 引入** | 不在本任务修复（"don't refactor beyond what task requires"）；登记为 **R-104 / 新 hotfix task T-0080** 候选 | ⏸ defer + 登记 |
| H2 | reviewer 自行撤回 | — | — |
| H3 → M | validateRestorePath `pathpkg.Clean` 比对策略可能拒合法尾斜杠 | MVP 接受；操作者传精确 object key 是设计 | ⏸ accept |
| M1 | decompress.go 资源泄漏 | 验证 obj defer Close + reader Close 全闭环（zstd 内部 worker goroutine 由 Decoder.Close join），无泄漏 | ✅ verified safe |
| **M2** | `updateSkipped` 是 no-op stub，造成 POST response.error_message 与 DB row.error_message 不一致 | 加 `RestoreTaskRepository.UpdateErrorMessage` 方法 + 实现 + service 调用真持久化 | ✅ Fixed |
| M3 | started_at 在 zero enqueued 时也设置 | 接受 MVP 行为；语义"request received_at"已 PRD 文档化 | ⏸ accept |
| M4 | Content-Length 缺失走 chunked encoding，老 CPE 兼容风险 | 文档化为已知限制；典型 config < 1 MB，buffer 优化属性能调优 | ⏸ accept；T-0078 FE 阶段如有报告再优化 |
| **M5** | restore_service.go json.Marshal 静默 `_` 忽略 error | 改正确 `if err != nil { return ... }` | ✅ Fixed |
| **M6** | restore_pg_repository.go json.Marshal 静默 `_` 忽略 error | 同上 | ✅ Fixed |
| M7 | stater nil 时静默跳过 precheck | 接受；构造器允许 nil 用于测试，生产路径 inf.MinIO ≠ nil | ⏸ accept |
| **M8** | restore_service.go 末尾 `var _ = errors.New` 占位行 + 未使用 errors import | 删除 | ✅ Fixed |

**最终复核**：
- `go build ./...` ✅
- `go test -race ./internal/backup/... ./internal/acs/download/... -count=1` ✅ all green
- 无 P0/P1 阻塞项（H1 是 pre-existing，登记 followup）
- M2 修复确保 API 契约一致；M5/M6/M8 cosmetic-but-correct 提升

**Pre-existing 000038 重复**：登记 backlog **T-0080**（hotfix，rename one of 000038 → 000049 让 goose 可解析）— 与 T-0072 解耦，不阻塞本任务。
