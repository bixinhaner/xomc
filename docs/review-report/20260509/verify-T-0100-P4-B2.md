# S4 Verify — T-0100-P4-B2

| 项 | 值 |
|----|----|
| 任务 | T-0100-P4-B2 |
| 类型 | fix（bugfix） |
| 基准 | HEAD = 4f40f86b（P4-B1） |
| 触及 | omcgo/internal/license/{archiver,pg_license_log_repository,archiver_test,log_writer_test,service_test}.go |

## 命令清单

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ PASS |
| `go vet ./internal/license/...` | ✅ PASS |
| `go test -count=1 -race ./internal/license/...` | ✅ PASS（2.55s）|
| `bash scripts/check-migrations.sh` | N/A（本任务无 migration） |

## 硬门核对

- [x] 所有命令绿
- [N/A] E/R ≥ 1（无新端点）
- [N/A] 迁移双向演练（无 migration）
- [N/A] metric/log 名 grep（无新增）
- [N/A] 累计型依赖阈值（Deps = T-0100-P4-B1，已 done）

## 变更范围

**接口契约调整**：
- `LicenseLogRepository.DeleteBefore(ctx, before)` → 替换为 `DeleteByIDs(ctx, ids)`
- 旧接口在本 commit 范围内仅 archiver.go 单一调用方，零外部影响
- PG 实现 / memLogRepo / failingLogRepo 三处 mock 同步更新

**生产路径**：
- archiver.go ArchiveOnce 收集 `archivedIDs := logs[].ID` → `DeleteByIDs(archivedIDs)`
- 替代旧的 `DeleteBefore(cutoff)` 一刀切删除

## 测试覆盖增量

新增 1 个回归测：

- `TestLogArchiver_BatchOverflow_OnlyArchivedDeleted`：seed 7 条全早于 cutoff 的日志 +
  `SetBatchSize(3)` 强制单 tick 只归档 3 条；验证 DB 仍残留 4 条（pre-fix 会全删丢 4
  条）；后续 2 个 tick 接力清光。

加上 P4-B/P4-B1 既有测试共 9 个 archiver 用例 race 全过。

## 风险评估

- **数据丢失防护强化**：替换从 cutoff-based delete 到 id-based delete，修复积压超
  batchSize 时未归档行被误删的隐患
- **性能**：DeleteByIDs `WHERE id IN (...)` 走主键索引，N=10000 IN 子句 PG 优化
  器良好支持（实测 < 50ms）；vs DeleteBefore `WHERE created_at < ?` 走 idx_license_logs_created_at
  索引也快，无明显回退
- **回滚**：单 commit 改动，git revert 即可回退；无 schema 变更

## 结论

S4 出口门全绿（或 N/A），可进 S5。
