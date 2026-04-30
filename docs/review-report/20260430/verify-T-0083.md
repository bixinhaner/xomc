# T-0083 verify — backup multi-device orphan 文件 reaper

> **Backlog**: T-0083 (P3 / F06/backup / sprint-08 / R-102)
> **PRD**: `docs/project/prd/T-0083-backup-orphan-reaper.md`
> **Date**: 2026-04-30

---

## 1. 改动摘要

| 文件 | 性质 | 行数变动 |
|------|------|---------|
| `omcgo/internal/backup/policy_orphan_reaper.go` | 新增 | +175 |
| `omcgo/internal/backup/policy_orphan_reaper_test.go` | 新增（V1-V12 + parseBackupFilename 表测） | +302 |
| `omcgo/internal/backup/policy_monitor.go` | 加 `taskIDLister` field + `@weekly` cron AddFunc | +20 / -3 |
| `omcgo/internal/backup/policy_metrics.go` | +2 collector + +2 Record helper | +30 |
| `omcgo/internal/backup/repository.go` | 注释说明 narrow iface 设计 | +5 |
| `omcgo/internal/backup/pg_repository.go` | `ListAllTaskIDPrefixes` 实施 | +28 |
| `omcgo/internal/backup/policy_storage_monitor_test.go` | V9 cron 数量 2→3 | ±2 |
| `omcgo/cmd/app/provider/modules.go` | DI: `SetTaskIDLister(backupTaskRepo)` | +4 |

无新迁移、无新端点、无前端改动、无运营商分支。

## 2. 关键设计决策（与 PRD §2 ULTRATHINK 对齐）

- §2.1 list-prefix scan 而非新表 `backup_files` — scope 控制
- §2.2 `@weekly` 频率 — housekeeping 性质足够
- §2.3 `RetentionDays + 1` 天 age 安全网 — 与 cleanup 的 retention 锁步联动，避免 race 误删
- §2.4 `maxReapPerRun = 1000` — 单 tick 上限防 API 流量爆发
- §2.5 regex `^.*backup-([0-9a-f]{8})-.+?\.xml(\.[a-z0-9]+)?(\.[a-z0-9]+)?$` — 容许 `.gz/.zst/.enc` 双 suffix
- §2.6 一次性构建 live set（≤ 8MB heap @ 1M tasks）
- §2.7 2 metric：`omc_backup_orphan_reaped_total` + `omc_backup_orphan_skipped_total{reason}`
- §2.8 与 T-0076 cleanup 无 race — cleanup 删 `file_path`-known 集合，reaper 删 live-set 之外 + 30+ 天对象，目标不重叠
- **narrow consumer iface**（PRD §9.2）：`TaskIDLister` 在 `policy_orphan_reaper.go` 声明，避免给 `TaskRepository` 加方法 → 5 个既有 mock（`monTaskRepo` / `mockTaskRepo` / `fakeTaskRepo` / `fakePrefixRepo` / `execTaskRepo`）零改动

## 3. 验证命令

```bash
CGO_ENABLED=0 go build ./...                                # ✅ pass
CGO_ENABLED=0 go vet ./...                                  # ✅ pass (0 issue)
gofmt -l <changed files>                                    # ✅ 0 file (post-normalize)
CGO_ENABLED=0 go test -count=1 -race ./internal/backup/...  # ✅ ok 2.711s
CGO_ENABLED=0 go test -count=1 -race ./...                  # ✅ all pass except 2 pre-existing
```

### 3.1 全量 -race 失败项（pre-existing on `main`，T-0083 未引入）

```
FAIL  github.com/omcgo/omcgo/internal/task
  TestPgRepo_Integration_ListPendingAllDevices
  TestService_PG_RestorePendingQueues
  → can't scan into dest[20] (col: source_id): cannot scan NULL into *string
```

`git stash && go test ./internal/task/...` 在 `main` 干净状态同样失败 — 与本次改动无关。已记录留待 task 模块 owner 处理。

### 3.2 验收标准（PRD §4）

| Case | 测试 | 状态 |
|------|------|------|
| V1 reap 成功 | `TestOrphanReaper_V1_SuccessPath` | ✅ |
| V2 live task 不 reap | `TestOrphanReaper_V2_LiveTaskSkipped` | ✅ |
| V3 < retention+1 天不 reap | `TestOrphanReaper_V3_AgeRecentSkipped` | ✅ |
| V4 pattern 不匹配不 reap | `TestOrphanReaper_V4_PatternMismatchSkipped` | ✅ |
| V5 max-K 上限 | `TestOrphanReaper_V5_MaxKCap` | ✅（1500 candidate → 1000 reap） |
| V6 API error 不 abort | `TestOrphanReaper_V6_RemoveObjectErrorContinues` | ✅ |
| V7 list err 不 panic | `TestOrphanReaper_V7_ListBucketErrorFailOpen` | ✅ |
| V8 nil-safe | `TestOrphanReaper_V8_NilSafeNoop`（3 子用例）| ✅ |
| V9 cron 3 entries | `TestOrphanReaper_V9_CronLifecycle3Entries` + `TestStorageCheck_CronStartIncludesHourly` | ✅ |
| V10 既有 11 case 不破 | 全部 `policy_monitor_test.go` + `policy_storage_monitor_test.go` 维持 | ✅ |
| V11 taskIDLister err | `TestOrphanReaper_V11_TaskIDListerErrorPropagates`（PRD §9.3 显式 err 路径） | ✅ |
| V12 parseBackupFilename | `TestParseBackupFilename`（8 用例覆盖 hex / case / suffix 组合） | ✅ |

### 3.3 metric 名 grep 验证

```bash
grep -rn "omc_backup_orphan_reaped_total\|omc_backup_orphan_skipped_total" .
→ docs/project/prd/T-0083-backup-orphan-reaper.md      # PRD §2.7
→ omcgo/internal/backup/policy_metrics.go              # collector 注册
→ omcgo/internal/backup/policy_orphan_reaper.go        # （间接通过 RecordOrphanReaped/RecordOrphanSkipped）
```

## 4. E/R 比

无新端点，N/A。

## 5. 累计型依赖核销

不适用。

## 6. 已识别风险与已采取措施

| 风险 | 缓解 |
|------|------|
| ListObjects 期间新建任务被误判 orphan | RetentionDays+1 天 age 安全网（PRD §2.3） |
| 8-char prefix 命名碰撞导致 live-set 误命中 | 误命中 = 减少 reap，不会误删（保守方向） |
| 单 tick 大量 reap 压垮 MinIO | maxReapPerRun=1000 上限 + 余量留下次 tick |
| RemoveObject 中途 err 中断扫描 | best-effort：err 记 metric + log 后继续 |
| AccessDenied / IAM 配置 | 走 `api_error` 计数器，cron 下次重试；ops dashboard 可见 |
| 与 T-0076 cleanup race | 目标集不交：cleanup 处理 `file_path`-known，reaper 处理 live-set 外 + 30+ 天对象 |

## 7. 待办与未覆盖

- 端到端实测（与 docker-compose MinIO 真容器 + 真 backup 流量）— Wave 期暂用单元 + race 覆盖
- Grafana panel 接入 `omc_backup_orphan_reaped/skipped` — 后续 dashboard 维护任务
- 多皮肤前端：N/A（纯后端改动）

## 8. DoD 自查

- [x] `go build ./...`
- [x] `go test -race ./internal/backup/...`
- [x] `gofmt -l` 干净
- [x] `go vet` 干净
- [x] 新代码覆盖（V1-V12 GWT 一对一）
- [x] 公共接口无 `any` / `interface{}`
- [x] 无新增 `if carrier == "..."` 硬编码
- [x] 无 TODO/FIXME 残留
- [x] 接口 narrow（`TaskIDLister` 1 method）
- [x] 错误 `fmt.Errorf("ctx: %w", err)` wrap
- [x] 不阻塞 S4 全量 (pre-existing failure 在 task 模块，与 T-0083 无关，已 stash 验证)

## 9. Reviewer 备忘

Approve 建议。Highlight：
1. `TaskIDLister` 是 narrow consumer iface，不入 `TaskRepository` — 避免 5 mock 同步成本
2. Age cutoff 联动 `policy.RetentionDays`：运维如改 retention=60 天，reaper 自动 push 到 61 天
3. 仅 `RemoveObject` 用 `obj.Key` 作 object path（无 splitBucketAndPath round-trip）— ListObjects 返回的 key 已是 in-bucket 路径
4. `maxReapPerRun` 当前为常量；如未来需 env var 可调，参考 §N4 非目标延后处理
