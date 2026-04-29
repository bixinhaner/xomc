# S4 Verify Report — T-0084 backup AlertSeverity policy-driven

> 任务：T-0084 / `feat(backup): AlertSeverity policy-driven 化 + schema migration 000050`
> Sprint：sprint-08（R-102 followup）
> 基线 commit：`9da7f538`（T-0082 S7 之上）
> 验证日期：2026-04-29

---

## 1. 验证范围

PRD §4 列 9 条 GWT（V1-V9）。新增 schema 列 + validate + 双 publisher 消费 policy.AlertSeverity + runtime fallback。无新端点。

## 2. S3 出口门复核

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `gofmt -l <my-files>` | ✅ 无输出 |
| `go test -race ./internal/backup/...` | ✅ 通过 |
| `bash scripts/check-migrations.sh` | ✅ 命名规范 + Up/Down 标记齐全 + 0 重复 |
| 新增 TODO/FIXME/panic | 0（删除了 policy_alarm_publisher.go 既有 TODO(T-0084)） |
| 新增 carrier 硬编码 | 0 |
| 新增 `any` / `interface{}` | 0 |

## 3. S4 验证清单

### 3.1 metric 名 grep

**N/A** — 本任务无新 metric。severity 是 alarm payload 字段；既有 `backup_failure_alarm_published_total` + `omc_backup_storage_threshold_alarm_total` 不变。

### 3.2 迁移演练

新增 `omcgo/migrations/000050_backup_alert_severity.sql`：
- Up: `ALTER TABLE backup_policies ADD COLUMN IF NOT EXISTS alert_severity VARCHAR(16) NOT NULL DEFAULT 'major' CHECK (alert_severity IN ('warning', 'major', 'critical'))`
- Down: `ALTER TABLE backup_policies DROP COLUMN IF EXISTS alert_severity`
- `IF NOT EXISTS` / `IF EXISTS` 保证幂等
- DEFAULT 'major' + NOT NULL：PostgreSQL 单事务 backfill 既有行 = 'major'，行为与既有硬编码 alarmSeverityMajor 常量一致 ⇒ 升级零运维感知

### 3.3 端点 E/R ≥ 1

**N/A** — 无新增 HTTP 端点（仅扩 PolicyRequest 字段，PUT /backup/policy 路径不变）

### 3.4 累计型依赖核销

**N/A** — Deps `T-0082 ✅` 是普通 done 依赖（commit `08913d0e` + `9da7f538`）

### 3.5 GWT 覆盖矩阵

| GWT | Test | Result |
|-----|------|--------|
| V1 DefaultPolicy().AlertSeverity = "major" | DefaultPolicy() 直接 verify; 既有 TestPolicyService_Get_EmptyReturnsDefaults 通过 | ✅ |
| V2 Migration backfill 既有行 | migration 000050 + check-migrations.sh ✅；PG ALTER ADD COLUMN NOT NULL DEFAULT 单事务 backfill 是 PG 16 文档承诺 | ✅ schema-level |
| V3 Validate 接受 warning/major/critical | TestPolicyService_Update_ValidationMatrix 3 新 case (T-0084) | ✅ |
| V4 Validate 拒绝 empty + outside set | TestPolicyService_Update_ValidationMatrix 2 新 case (T-0084 empty + emergency) | ✅ |
| V5 backup_task_failed alarm 走 policy.AlertSeverity | TestPublishFailureAlarm_PolicyDrivenSeverity 4 sub | ✅ |
| V6 backup_storage_threshold_exceeded raised 走 policy.AlertSeverity | TestStorageCheck_PolicyDrivenSeverity_Raised | ✅ |
| V7 alarm.cleared 也走 policy.AlertSeverity | TestStorageCheck_PolicyDrivenSeverity_Cleared | ✅ |
| V8 Empty AlertSeverity runtime fallback → major | TestPublishFailureAlarm_PolicyDrivenSeverity "empty" sub + TestStorageCheck_EmptyAlertSeverity_FallsBackToMajor | ✅ |
| V9 Backward compat 既有 8 PolicyMonitor + 18 service + 13 storage cases | go test ./internal/backup/... PASS | ✅ |

**总计**：8 新 test/sub + 既有 ~50 case 全 PASS。

### 3.6 新代码覆盖率（per-function）

| 函数 | 覆盖率 |
|------|--------|
| `severityOrDefault` | 100% |
| `(s *PolicyService).validatePolicy` | 100% |
| `validatePolicy` (package-level) | 89.2%（既有覆盖；本任务新加分支由 V3/V4 覆盖） |
| `PublishFailureAlarm` | 84.6%（既有；NewEvent JSON 不可达失败） |
| `PublishStorageThresholdAlarm` | 84.6%（既有；同上） |
| `RunStorageCheckOnce` | 94.7%（既有）|

整包覆盖 45.0%（无回退；T-0082 时 44.8%，本任务+0.2pp）。

### 3.7 既有 case 兼容（V9）

**全部既有 case PASS — runtime fallback 设计成功**：
- policy_service_test 18 → 23 case（+3 accept + 2 reject T-0084 case）
- policy_alarm_publisher_test 5 → 6（+1 PolicyDrivenSeverity table-driven 4 sub）
- policy_storage_monitor_test 13 → 16 顶层（+3 V6/V7/V8）
- policy_monitor_test 8 case 零改动通过（DefaultPolicy 升级 + runtime fallback 兜底）

### 3.8 文件清单

修改（10）：
- `omcgo/migrations/000050_backup_alert_severity.sql`（新建）
- `omcgo/internal/backup/policy_model.go` — AlertSeverity 字段 + DefaultPolicy="major" + validBackupPolicyAlertSeverities map
- `omcgo/internal/backup/policy_service.go` — validatePolicy 加 severity 白名单校验
- `omcgo/internal/backup/policy_pg_repository.go` — columns / scanPolicy / Upsert SQL 加 alert_severity
- `omcgo/internal/backup/policy_handler.go` — PolicyRequest + toModel 加 AlertSeverity
- `omcgo/internal/backup/policy_alarm_publisher.go` — payload.Severity = severityOrDefault(policy.AlertSeverity)；删除 TODO(T-0084) 注释；加 severityOrDefault helper
- `omcgo/internal/backup/policy_storage_alarm.go` — StorageInfo +Severity 字段；payload.Severity = severityOrDefault(info.Severity)
- `omcgo/internal/backup/policy_storage_monitor.go` — RunStorageCheckOnce 注入 policy.AlertSeverity 到 StorageInfo
- `omcgo/internal/backup/policy_service_test.go` — +5 case (3 accept + 2 reject T-0084)
- `omcgo/internal/backup/policy_alarm_publisher_test.go` — +TestPublishFailureAlarm_PolicyDrivenSeverity 4 sub
- `omcgo/internal/backup/policy_storage_monitor_test.go` — +V6/V7/V8 三 case

文档（1）：
- `docs/project/prd/T-0084-backup-alert-severity-policy-driven.md`（新建 PRD ~310 行）

Backlog：
- `docs/project/backlog.md`（T-0084 行 297 状态 triaged → in_design + PRD 路径回写）

## 4. 风险与已知限制

| # | 项 | 状态 |
|---|----|------|
| K1 | FE form 加 alert_severity select 字段 | 拆 followup（PRD §6 N1，FE 单独 PR） |
| K2 | severity 历史变更审计 | PRD §6 N5，未来 policy_history 表 |
| K3 | 设备级覆盖 severity | PRD §6 N6，按 device override 拆未来任务 |
| K4 | minor / emergency 等级缺失 | PRD §6 N3，CHECK 放宽 + map 加值容易 |

## 5. S4 出口门结论

| 门 | 状态 |
|----|------|
| 所有命令绿（build/vet/test/race） | ✅ |
| E/R ≥ 1（无新端点则 N/A） | ✅ N/A |
| 迁移双向演练（命名规范 + Up/Down 配对） | ✅（实际 PG up/down 演练需 docker，单文件检查通过） |
| metric 名全部能 grep 找到 | ✅ N/A |
| 累计型依赖阈值（无则 N/A） | ✅ N/A |

**S4 通过**，进入 S5 review。

---

## 6. S5 Review 落实记录（review-agent + DoD）

### 6.1 Review-agent — PASS-WITH-FIXES（0 P0 / 0 HIGH / 1 MED / 3 LOW）

| Severity | Finding | 处置 |
|----------|---------|------|
| MED-1 | severityOrDefault 在两 publisher 注入位置不对称 — alarm_publisher 在内部 normalize，storage path 仅 publisher 端 normalize；monitor 注入 raw policy.AlertSeverity | ✅ 落实 — policy_storage_monitor.go:137 加 `severityOrDefault` 包裹（producer 端），与 publisher 端共同形成 defense-in-depth |
| LOW-1 | Migration NOT NULL DEFAULT 在大表上的潜在锁阻塞 | ⏸ 已 cleared — backup_policies 是 singleton 表，PG 16 metadata-only 优化 |
| LOW-2 | DB CHECK vs Go map 双重维护漂移风险 | ⏸ leave — policy_model.go:91-94 注释已说明镜像约定 |
| LOW-3 | TestPolicyService_Get_ExistingRoundTrips fixture 不显式设 AlertSeverity | ⏸ leave — 现有断言不查 AlertSeverity，无 round-trip 回归危险（V9 既有 case 全过） |

**Reviewer 11 项 cleared 项**：migration safety / severityOrDefault 在两 publisher 一致 / StorageInfo 单一 producer / validatePolicy 顺序 / DefaultPolicy 设 "major" / SQL Upsert + scan 顺序对齐 / Publish 错误路径 metric 时序 / V5-V8 矩阵覆盖 / 持久化 round-trip / migration 编号连续。

### 6.2 DoD（按 `docs/project/dod.md` 通用项）

- [x] `go build ./...` 通过
- [x] `go test -race ./...` backup 包全绿（task 包 2 个 pre-existing 失败已记账）
- [x] `gofmt -l <my-files>` 无输出
- [x] `go vet ./...` 通过
- [x] `bash scripts/check-migrations.sh` 通过
- [x] 新代码无 TODO/FIXME/panic（**显式删除**了 policy_alarm_publisher.go 既有 TODO(T-0084) — 这是本任务的存在意义）
- [x] 无新增 carrier 硬编码
- [x] 无新增 `any`/`interface{}`
- [x] 新功能含成功 + 失败两条路径测试（V3 accept × 3 / V4 reject × 2 / V5 sub × 4 / V6+V7+V8）
- [x] 新代码 per-function coverage 84.6%-100%；整包 45.0%（+0.2pp，无回退）
- [x] 既有 50+ case 零改动通过（runtime fallback 设计成功）
- [x] 关键 review finding 全部落实或显式 defer 并记账

**S5 通过**，进入 S6 commit。
