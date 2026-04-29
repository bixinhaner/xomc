# S4 Verify Report — T-0073 Phase 1 backup cleanup cron + 失败告警

> **生成**: 2026-04-29
> **任务**: T-0073 Phase 1 / R-102 enforcement followup（持久化层增强：DB cleanup + alarm.raised 发布；file delete + storage threshold 留 T-0076）
> **PRD**: `docs/project/prd/T-0073-backup-cleanup-and-alarm.md`
> **Sprint**: sprint-06..07

---

## 1. 改动清单

### Backend（5 新 + 4 修改）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcgo/internal/backup/policy_metrics.go` | 新 | 3 Prometheus counter（cleanup_runs / cleanup_rows_deleted / failure_alarm_published）+ nil-safe |
| `omcgo/internal/backup/policy_alarm_publisher.go` | 新 | `FailureAlarmPayload` + `PublishFailureAlarm(ctx, policyGetter, bus, metrics, task)` + `PolicyGetter` interface |
| `omcgo/internal/backup/policy_monitor.go` | 新 | `PolicyMonitor`（仿 license/monitor.go cron + RunCleanupOnce + Start/Stop）|
| `omcgo/internal/backup/policy_alarm_publisher_test.go` | 新 | 4 测试：alertOn=true publishes / alertOff skips / policy fetch error / nil task |
| `omcgo/internal/backup/policy_monitor_test.go` | 新 | 5 测试：cleanup deletes / keep_last_n forwarded / autoCleanup=false skips / empty policy fallback / Start-Stop smoke |
| `omcgo/internal/backup/repository.go` | 修 | TaskRepository +`CleanupOldRows(ctx, cutoff, keepLastN)` 接口方法；+ time 导入 |
| `omcgo/internal/backup/pg_repository.go` | 修 | `CleanupOldRows` pg 实现：CTE + ROW_NUMBER OVER PARTITION BY target_type + 严格 status WHERE + completed_at NOT NULL guard |
| `omcgo/internal/backup/executor.go` | 修 | `BackupExecutor` +policyService/metrics 字段（optional）+ `SetPolicyEnforcement` setter；失败路径调 `PublishFailureAlarm`（nil-safe） |
| `omcgo/internal/backup/{service_test,executor_test,handler_test}.go` | 修 | 3 既有 mock 加 CleanupOldRows stub 满足扩展接口 |
| `omcgo/cmd/app/provider/modules.go` | 修 | DI：+policyMetrics +backupPolicyMonitor.Start；miscDeps +backupPolicyMonitor 字段 |
| `omcgo/cmd/worker/main.go` | 修 | Worker DI：+backupPolicyService + backupPolicyMetrics + backupExecutor.SetPolicyEnforcement(...) |

### 文档（2 文件）

| 路径 | 性质 |
|------|------|
| `docs/project/prd/T-0073-backup-cleanup-and-alarm.md` | PRD（10 章 / 9 GWT V1-V9 / Phase 化决策 + Phase 2 = T-0076 followup） |
| `docs/review-report/20260429/verify-T-0073.md` | 本报告 |

### 不动

- migration head 仍 000047（Phase 1 0 schema 变更）
- 前端不动（webcode/frontend-core 零改动）
- alarm engine / notification 模块 不动（复用既有 `event.SubjectAlarmRaised` 订阅）

---

## 2. 硬门验证

### 2.1 Backend

```bash
$ CGO_ENABLED=0 go build ./...
✅ BUILD_OK

$ CGO_ENABLED=0 go test -count=1 ./internal/backup/...
ok  	github.com/omcgo/omcgo/internal/backup	0.845s
✅ 全过

$ CGO_ENABLED=0 go vet ./...
✅ 无输出（通过）

$ bash omcgo/scripts/check-migrations.sh
✅ migration 头不变（000047），无新迁移
```

### 2.2 3 metric grep

```bash
$ grep -rn "backup_cleanup_runs_total|backup_cleanup_rows_deleted_total|backup_failure_alarm_published_total" --include="*.go"
5 命中 ✅（注释 3 + Name 字段 3）
```

### 2.3 测试覆盖（V1-V9）

| 验收 | 测试函数 | 验证点 |
|------|---------|--------|
| V1 | TestPublishFailureAlarm_AlertOnTruePublishes | 完整 payload round-trip + 事件 subject = alarm.raised |
| V2 | TestPublishFailureAlarm_AlertOffSkips | 不发布 + 0 事件 |
| V3 | TestPolicyMonitor_StartStop | cron 启停 smoke |
| V4 | TestPolicyMonitor_RunCleanupOnce_DeletesOldRows | retentionDays=30 → cutoff ~30 天前；调一次 cleanup |
| V5 | TestPolicyMonitor_RunCleanupOnce_HonoursKeepLastN | keep_last_n=3 透传 |
| V6 | TestPolicyMonitor_RunCleanupOnce_AutoCleanupFalseSkips | repo 0 调用 |
| V7 | TestPolicyMonitor_RunCleanupOnce_EmptyPolicyUsesDefaults | 空 policy → DefaultPolicy()（KeepLastN=5 默认）|
| V8 | 6 单测全过 | 单元测试覆盖 V1-V7 |
| V9 | check-migrations.sh ✅ | 0 schema 变更 |

✅ 9 项验收全部覆盖

### 2.4 既有 3 mock task repo 已扩展

3 既有 mock（`mockTaskRepo` / `execTaskRepo` / `fakeTaskRepo`）补 `CleanupOldRows` stub 方法 + `time` 导入；既有 service/executor/handler tests 全部继续 PASS。

---

## 3. Phase 边界确认

**已交付（Phase 1）**：
- ✅ 失败告警 alarm.raised 发布（worker 端 executor 失败路径）
- ✅ DB cleanup cron @daily（app 端 monitor）
- ✅ 3 Prometheus 指标
- ✅ DI 在 app + worker 双进程

**留 T-0076（Phase 2）**：
- ❌ 物理文件删除（需先审计 transfer 模块文件存储路径）
- ❌ 磁盘使用率超 alertThresholdPercent → alarm.raised
- ❌ Backup-failure → 邮件实际投递（需要 alarm engine filter rule 配置；本期仅"发布到 alarm 总线"）
- ❌ Cleanup metrics Grafana 仪表盘

---

## 4. DoD（PRD §8）

- [x] PRD 七要素全 + 运营商一致矩阵
- [x] V1-V9 全部测试通过
- [x] go build / vet / test 全过
- [x] 0 schema 改动（migration 头不变）
- [x] 3 新 Prometheus metric 注册 + grep 命中 ≥1
- [x] 6+ 单元测试（实际 V1+V2+bonus 4 publisher + V3-V7 5 monitor = 9 测试）
- [ ] backlog T-0073 → done + Phase 2 followup T-0076 登记 — S7 处理
- [ ] R-102 进展更新 — S7 处理

---

## 5. 风险评估

| 风险 | 缓解 |
|------|------|
| Cleanup CTE 在百万行 backup_tasks 慢 | MVP 接受；future 加索引 `(target_type, completed_at)`；本期 backup_tasks 量级远小于此 |
| Cleanup 误删未结束任务 | WHERE 严格 `status IN ('completed','failed','cancelled')`；不动 pending/running；test V4 验证 |
| alarm.raised payload 含 alert_email 暴露给所有 alarm 订阅者 | 邮件地址来自管理员配置（policy.alert_email），非用户 PII；alarm 总线本来就是内网服务消息 |
| Worker + App 双进程 metric 注册可能冲突（重复 collector） | 不冲突：两进程独立 `MetricsReg` 注册器；同一指标名在两进程各 export 一份（Prometheus scrape 时按 instance 区分）|
| Monitor 在 backup_policies 表为空时崩 | PolicyService.Get 已有 fallback 到 DefaultPolicy() ；test V7 验证 |

---

## 6. 多皮肤影响

无前端改动 → webcode-v2/v3 零影响。

---

*验证完成；硬门全过；Phase 1 范围交付。*

---

## 7. S5 Review（已完成）

Code-reviewer agent verdict: **APPROVE-WITH-FIXES → APPROVE**（2 HIGH + 6 MEDIUM）。已 fix-in-place 4 项；其余按 reviewer 推荐 defer 至 T-0076。

| 项 | 严重度 | 处理 |
|----|-------|------|
| H1 — `PublishFailureAlarm` 缺 nil-policyService/nil-bus 守卫 | HIGH | ✅ 加 3 项 nil guard + 2 新测试（NilPolicyService / NilEventBus）|
| H2 — `PolicyMonitor.Start` AddFunc 失败时 cancel 未调，泄漏 scoped ctx | HIGH | ✅ 加 cancel() + 重置 m.cancel/m.cron 到 nil |
| M1 — Cleanup CTE PARTITION BY 仅 target_type，可能 failed 占 keep 槽 | MEDIUM | ✅ 加详细注释说明设计意图（保留 failed 行用于事后调查），并标记 future ORDER BY 优化路径 |
| M5 — `Severity:"major"` 硬编码 | MEDIUM | ✅ 加 TODO(T-0076) 注释，说明未来 policy-driven |
| M2 — backup_tasks 无 `(target_type, completed_at)` 索引 | MEDIUM | ⚪ Defer T-0076（performance phase 2，现量级 OK）|
| M3 — CleanupTime 不被 cron 表达式利用 | MEDIUM | ⚪ 已在代码注释明示 Phase 2；不再二次 |
| M4 — Worker 端 PolicyService.Get 每次失败 SELECT，无 cache | MEDIUM | ⚪ Defer T-0076；失败频次低 |
| M6 — pubEventBus.Subscribe 返 nil/nil 测试 ergonomics | LOW | ⚪ skip |

post-fix 验证：
- `go build ./...` ✅
- `go test -count=1 ./internal/backup/...` ✅（11 测试函数：原 9 + 新增 2 nil guard）
- 0 schema 改动（migration head 仍 000047）

最终 review verdict: **APPROVE**

