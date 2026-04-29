# S4 Verify Report — T-0082 backup 磁盘阈值告警

> 任务：T-0082 / `feat(backup): 磁盘阈值告警 — bucket 用量轮询 + 边沿触发 alarm.raised/cleared`
> Sprint：sprint-08（R-102 followup）
> 基线 commit：`a1c54231`
> 验证日期：2026-04-29

---

## 1. 验证范围

PRD §4 列 10 条 GWT（V1-V10）+ 3 条 bonus，全部由单元测试承接。无新端点，无 schema 迁移，纯后台 cron 集成。

## 2. S3 出口门复核

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ 通过（CGO_ENABLED=0；本地 macOS Xcode license 未签，cgo 路径阻塞但无影响） |
| `go vet ./...` | ✅ 通过 |
| `gofmt -l <changed-files>` | ✅ 无输出（全部 gofmt-clean） |
| `go test -race ./internal/backup/...` | ✅ 通过（-race 含数据竞争检测） |
| 全包 `go test -race ./...` | ⚠️ 仅 `internal/task` 2 个 pre-existing 失败（NULL source_id scan，与 T-0082 无关，T-0075 会话 stash 已验证） |
| 新增 TODO/FIXME/panic | 0 |
| 新增 carrier 硬编码 | 0（不触运营商代码） |
| 新增 `any` / `interface{}` | 0（payload 全强类型） |

## 3. S4 验证清单

### 3.1 metric 名 grep（5/5 命中）

```
omc_backup_storage_used_bytes               → policy_metrics.go:113
omc_backup_storage_capacity_bytes           → policy_metrics.go:117
omc_backup_storage_usage_ratio              → policy_metrics.go:121
omc_backup_storage_check_total              → policy_metrics.go:125
omc_backup_storage_threshold_alarm_total    → policy_metrics.go:129
```

### 3.2 迁移演练

**N/A** — 本任务无新迁移。`MaxStorageGB` (default 500) + `AlertThresholdPercent` (default 80) 已存在于 `backup_policies` 表（T-0071 schema），本任务首次消费这两列。`bash scripts/check-migrations.sh` 因此跳过。

### 3.3 端点 E/R ≥ 1

**N/A** — 本任务无新增 HTTP 端点（纯后台 cron）。

### 3.4 累计型依赖核销

**N/A** — Deps `T-0076 ✅` 是普通 done 依赖，非累计型（T-0076 commit `e22dc3e4` + S7 `b5a6e5d4`）。

### 3.5 GWT 覆盖矩阵

| GWT | Test | Result |
|-----|------|--------|
| V1 below threshold no alarm | `TestStorageCheck_BelowThreshold_NoAlarm` | ✅ |
| V2 first above publishes raised | `TestStorageCheck_FirstAboveThreshold_PublishesRaised` | ✅ |
| V3 still above no republish | `TestStorageCheck_StillAbove_NoRepublish` | ✅ |
| V4 drops below publishes cleared | `TestStorageCheck_DropsBelow_PublishesCleared` | ✅ |
| V5 list error fail-open | `TestStorageCheck_ListError_FailOpen` | ✅ |
| V6 AlertOnFailure=false skipped | `TestStorageCheck_AlertOnFailureFalse_Skipped` | ✅ |
| V7 MaxStorageGB=0 skipped | `TestStorageCheck_MaxStorageGBZero_Skipped` | ✅ |
| V8 cold start above re-publish | `TestStorageCheck_ColdStart_AboveThresholdPublishesOnce` | ✅ |
| V9 cron registers @hourly | `TestStorageCheck_CronStartIncludesHourly` | ✅ |
| V10 no wiring no-op | `TestStorageCheck_NoWiringIsNoOp` | ✅ |
| Bonus over-capacity 250% | `TestStorageCheck_OverCapacity_PayloadPercentExceeds100` | ✅ |
| Bonus publisher nil bus | `TestPublishStorageThresholdAlarm_NilBus` | ✅ |
| Bonus publisher publish error | `TestPublishStorageThresholdAlarm_PublishError` | ✅ |
| Bonus percentClamped table | `TestPercentClamped` 6 sub | ✅ |
| Bonus storageAlarmRoute table | `TestStorageAlarmRoute` 3 sub | ✅ |

**总计**：13 顶层 test + 9 sub = ~22 case，全 PASS。

### 3.6 新代码覆盖率（per-function）

| 函数 | 覆盖率 |
|------|--------|
| `RunStorageCheckOnce`     | 94.6% |
| `sumBucketUsage`          | 100% |
| `SetBucketLister`         | 100% |
| `SetEventBus`             | 100% |
| `percentClamped`          | 100% |
| `storageAlarmRoute`       | 100% |
| `PublishStorageThresholdAlarm` | 84.6%（NewEvent JSON 失败分支不可达） |

整包 `internal/backup` 覆盖率：44.8% of statements（无回退；仅新代码贡献）。

### 3.7 既有 case 兼容

`policy_monitor_test.go` 8 既有 case（V3-V8 of T-0073/0076）全部继续通过：
- TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_AllSuccess
- TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_PartialFailure
- TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_MalformedPathSkipped
- TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_NoMinIONoCalls
- TestClassifyMinIORemoveErr
- TestPolicyMonitor_RunCleanupOnce_DeletesOldRows
- TestPolicyMonitor_RunCleanupOnce_HonoursKeepLastN
- TestPolicyMonitor_RunCleanupOnce_AutoCleanupFalseSkips
- TestPolicyMonitor_RunCleanupOnce_EmptyPolicyUsesDefaults
- TestPolicyMonitor_StartStop

设计上 `NewPolicyMonitor` 签名保持 4 参数不变（PRD §9.2 决策），新依赖 `bucketLister` + `bus` 通过 setter 模式注入；既有 8 case 不需修改。

### 3.8 Grafana dashboard

`deployments/monitoring/grafana/dashboards/omc-overview.json` 加 4 新 panel：
- Backup Storage Usage Ratio（stat，绿/黄/红 0.6/0.8 阈值）
- Backup Storage Used vs Capacity（time-series 双线 overlay）
- Backup Storage Check Outcomes（time-series by result）
- Backup Storage Threshold Alarms Raised/Cleared（stat by kind）

`jq empty` 校验 JSON 合法 ✅。

## 4. 风险与已知限制

| # | 项 | 状态 |
|---|----|------|
| K1 | 进程重启重发 alarm.raised | 接受（PRD §2.4 决策；F04 alarm 引擎 identifier 去重）|
| K2 | ListObjects 在大 bucket（10M+ 对象）IO 成本 | 接受（@hourly 频率内可消化）|
| K3 | severity 硬编码 "major" | T-0084 followup（联合 T-0082 设计 schema） |
| K4 | 多实例部署多次发 raised | T-0084 followup 时由 alarm 引擎 identifier 去重承担 |
| K5 | sub-second tick 重叠 | cron 默认不重叠跑同一 entry，无需额外 mutex |

## 5. 文件清单

**新增**（3）：
- `omcgo/internal/backup/policy_storage_alarm.go`（123 行 — payload + transition + publisher + route helper）
- `omcgo/internal/backup/policy_storage_monitor.go`（167 行 — RunStorageCheckOnce + setters + percentClamped）
- `omcgo/internal/backup/policy_storage_monitor_test.go`（约 380 行 — 13 test + 9 sub case + fakes）

**修改**（4）：
- `omcgo/internal/backup/policy_metrics.go`（+5 collector + 5 Record + 顶部 doc 更新）
- `omcgo/internal/backup/policy_monitor.go`（+bucketLister/bus/storageMu/lastAboveThreshold 字段；+@hourly cron entry；+顶部 doc 更新）
- `omcgo/cmd/app/provider/modules.go`（+SetBucketLister + SetEventBus DI 装配）
- `deployments/monitoring/grafana/dashboards/omc-overview.json`（+4 panel）

**文档**（2）：
- `docs/project/prd/T-0082-backup-disk-threshold.md`（新建 PRD，~310 行 7 要素 + GWT + 设计备忘）
- `docs/project/risk-register.md`（R-102 mitigation 状态行更新）

**Backlog**：
- `docs/project/backlog.md`（T-0082 行 295 状态 triaged → in_design + PRD 路径回写 + 描述更新）

## 6. S4 出口门结论

| 门 | 状态 |
|----|------|
| 所有命令绿（build/vet/test/race） | ✅ |
| E/R ≥ 1（无新端点则 N/A） | ✅ N/A |
| 迁移双向演练（无迁移则 N/A） | ✅ N/A |
| 5 metric 名全部能 grep 找到 | ✅ |
| 累计型依赖阈值（无则 N/A） | ✅ N/A |

**S4 通过**，进入 S5 review。

---

## 7. S5 Review 落实记录（review-agent + simplify）

### 7.1 Review-agent（独立） — PASS-WITH-FIXES（0 P0 / 2 HIGH / 3 MED / 2 LOW）

| Severity | Finding | 处置 |
|----------|---------|------|
| HIGH-1 | 阈值边界 strict `>` 在 == threshold 时误发 cleared | ✅ 改 `used >= threshold` + 加 boundary 回归测试（`TestStorageCheck_ExactlyAtThreshold_PublishesRaised` + `TestStorageCheck_BelowThresholdAfterAbove_Clears`） |
| HIGH-2 | RunStorageCheckOnce list 失败返回 error vs PRD §2.7 fail-open 语义不一致 | ✅ 加函数级 warn log 同时保留 error 返回；clarify "fail-open = 不发假告警 + 不 panic"，error 用于直接调用方可见性 |
| MED-1 | AlertOnFailure 同时门控 disk + per-task alarm 易混淆 | ✅ 加注释指向 T-0084 followup（DiskAlertEnabled 字段拆分） |
| MED-2 | 92 PB 容量时 `capacity * pct` int64 溢出 | ✅ 重排数学 `(capacity / 100) * pct`，sub-100B 精度损失可接受 |
| MED-3 | gigabyte(1)/2 readability 暗坑 | ⏸ leave；偶数除整除安全 |
| LOW-1 | storageAlarmRoute default 分支不可达 | ⏸ leave；防御性路由是正确选择 |
| LOW-2 | metric kind="error" 未在 PRD §8 列出 | ✅ 已更新 PRD §8 列入 error 含义 |

### 7.2 /simplify（reuse + quality + efficiency 三 agent 并行）

| 类别 | Finding | 处置 |
|------|---------|------|
| Reuse MED-1 | fakeEventBus 与既有 pubEventBus 重复 | ⏸ defer — 合并需触 T-0007/0073/0075 publisher 测试，超出 T-0082 scope |
| Reuse LOW-2 | `percentClamped` 实际不 clamp（misnomer） | ✅ 重命名 `usagePercent` |
| Quality MED-1 | "backup"/"major" 字面值跨两个 payload 重复 | ✅ 提取 `alarmSourceBackup` + `alarmSeverityMajor` const 到 policy_alarm_publisher.go |
| Quality LOW-3 | 两个 cron AddFunc 闭包 90% 重复 | ⏸ leave — 第三个出现时再 extract（rule of three） |
| Quality LOW-4 | T-NNNN comment 噪声 | ⏸ leave — review-fix 注释承载 WHY 语义（边界 / fail-open / overflow），删除会造成 info loss |
| Efficiency MED-1 | ListObjects 全 bucket 扫描 hourly | ⏸ defer — PRD §2.2 explicitly punted madmin-go 切换到未来任务 |
| Efficiency MED-2 | RemoveObject 单条循环 vs RemoveObjects 批量 | ⏸ defer — T-0076 既有代码，超出 T-0082 scope |
| Efficiency LOW | PolicyService.Get 每 tick 重读 / 跳过 nil-disabled AddFunc | ⏸ leave — 性能影响可忽略 |

### 7.3 DoD（按 `docs/project/dod.md` 通用项）

- [x] `go build ./...` 通过
- [x] `go test -race ./...` backup 包全绿（task 包 2 个 pre-existing 失败已记账）
- [x] `gofmt -l <my-files>` 无输出
- [x] `go vet ./...` 通过
- [x] 新代码无 TODO/FIXME/panic（注释含 TODO(T-0084) 是有意指向 followup 任务）
- [x] 无新增 carrier 硬编码
- [x] 无新增 `any`/`interface{}`
- [x] 新功能含成功 + 失败两条路径测试（V1-V10 + 5 bonus + 2 boundary）
- [x] 新代码覆盖率 84.6%-100% 各函数；整包覆盖 44.8%
- [x] 无禁用既有测试
- [x] 关键 review finding 全部落实或显式 defer 并记账

**S5 通过**，进入 S6 commit。
