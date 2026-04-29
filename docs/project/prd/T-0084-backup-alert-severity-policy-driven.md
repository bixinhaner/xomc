# PRD: backup AlertSeverity policy-driven 化（T-0084）

> **关联**: Backlog T-0084 / Sprint-08 / Domain=F06/backup+alarm / Type=feat / Prio=P3
> **作者**: Claude（代 Owner=电信+Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: 一并关闭 `policy_alarm_publisher.go` + `policy_storage_alarm.go` 两处 TODO；新增 schema 列 `alert_severity` 但**复用** T-0084 单一 column 服务两类告警（避免分别为 backup_task_failed 与 backup_storage_threshold_exceeded 各开一列）

---

## 1. 业务背景

R-102 备份模块已有两类 alarm 走 alarm.raised：
- `backup_task_failed`（T-0007 / T-0073 — backup 任务失败时由 executor 发布）
- `backup_storage_threshold_exceeded`（T-0082 — bucket 用量超阈值时由 PolicyMonitor 发布）

**两处 severity 都硬编码 `"major"`**：
- `policy_alarm_publisher.go:73-78` 行内 `// TODO(T-0084): policy-driven severity`
- `policy_storage_alarm.go:78` 行内 `// "major" — disk fill = data-loss precursor`（隐式 TODO，与 T-0084 配套）

**问题**：
- 不同运维场景对告警重要程度判断不同：测试环境磁盘满想降级 warning（避免 oncall 误打扰），生产环境想升级 critical（强制 PagerDuty）；运维**无法配置**
- T-0082 PRD §2.5 explicitly punted 此项到 T-0084："T-0082 disk-threshold 现在处理，正是联合设计窗口"

**不做会发生什么**：F04 alarm 引擎按 severity 路由（critical → SMS+phone / major → email / warning → log only），全锁死 major 等于剥夺运维分级响应能力。本任务把 severity 从代码常量提升为 policy 字段。

---

## 2. ULTRATHINK 决策

### 2.1 单 column 服务两类告警 vs 双 column 分开

**选项 A**：单字段 `alert_severity` 同时控制 backup_task_failed + backup_storage_threshold_exceeded 严重度
**选项 B**：双字段 `failure_alert_severity` + `storage_alert_severity` 分别控制

考虑：
- 实操中两类告警都源于同一备份策略实体，运维通常会一致设定（"我对备份事务的容忍度"是单一概念）
- 双字段会让运维 UI 多一项配置项，决策疲劳（"failure 配 major 但 storage 配 critical 行不行？"）
- 双字段的灵活性收益小，复杂性收益大

**采纳 A — 单 column**。如果未来有 split 需求，可加新列为 fallback override；不会被本任务的单字段卡死。

### 2.2 严重度等级集合

3GPP 32.111 告警标准：critical / major / minor / warning / indeterminate / cleared。

OMC 既有 alarm engine（F04）值域**未限制**，任意字符串都收下。但 EmailDispatcher 等下游可能假定标准等级。

**采纳缩集**：`warning | major | critical`（3 档）。理由：
- minor 与 major 在备份场景区分模糊（备份失败要么不重要要么很重要，"中等重要"少见）
- indeterminate / cleared 不应作为初始 severity，cleared 走 alarm.cleared subject 而非 raised
- 3 档够用：warning（测试环境 / 非关键备份）/ major（默认，生产标配）/ critical（关键运营商客户、PagerDuty 集成）

如未来有需求，可加 minor 或 emergency。CHECK 约束放宽容易。

### 2.3 默认值 & 既有行兼容

migration `ALTER TABLE ... ADD COLUMN alert_severity VARCHAR(16) NOT NULL DEFAULT 'major'` 在已有行上**自动 backfill 'major'**（PostgreSQL ADD COLUMN with NOT NULL DEFAULT 是常量填充，单事务原子）。

DefaultPolicy() 同设 'major'。代码常量 `alarmSeverityMajor = "major"` 保留为 fallback（runtime 防御）。

**Runtime fallback 决策**：当 publisher 收到 `policy.AlertSeverity == ""`（unlikely 但理论可能：legacy 测试构造 BackupPolicy literal 不设 AlertSeverity），publisher **fall back 到 `alarmSeverityMajor`**。理由：
- 无 alert 总比"无 severity 字段的歪歪扭扭 alarm"好
- 防御深度，不破坏既有测试

### 2.4 Validate 边界

`policy_service.go:validatePolicy` 加严重度白名单（mirror `validBackupPolicyEncryptionAlgorithms` 模式）：
- map 集合 `validBackupPolicyAlertSeverities = {warning, major, critical}`
- PUT 路径不在集合即 400 ErrInvalidInput

DB CHECK 约束 mirror map（双重防御）。

### 2.5 关闭两处 TODO 的精确清理

**policy_alarm_publisher.go:67-79**：
- 当前：`Severity: alarmSeverityMajor` + 6 行 TODO 注释
- 改后：`Severity: severityFromPolicy(policy.AlertSeverity)` + 删除 TODO 注释（实际行动落地了，不留口头承诺）

**policy_storage_alarm.go:78**：
- 当前：`Severity: alarmSeverityMajor`（无显式 TODO 但与 T-0084 配套）
- 改后：本函数从 `StorageInfo` 接 severity 字段（caller PolicyMonitor 从 policy 注入）

`alarmSeverityMajor` 常量保留：仍是 DefaultPolicy() 默认 + runtime fallback。

### 2.6 与 T-0082 决策的呼应

T-0082 PRD §2.5 punted 时记 "AlertSeverity column + schema migration 联合 disk-threshold 设计 schema"。本任务正是该联合设计 — 不在 T-0082 做是因为 schema migration 的影响面（policy_handler / policy_service / policy_pg_repository / 全部测试 / FE form）需要单独 PR 范围才好控质量。

T-0082 已经把 PolicyMonitor 改造为多职责载体（cleanup + storage check）。T-0084 在此基础上再加"policy 注入到 publishers"的小改动，blast radius 可控。

### 2.7 FE 影响

frontend-core BackupPolicy 类型 + form 需加 alert_severity 字段（select：warning/major/critical）。**本任务不改前端** — 后端 PUT 接受空字符串时返 400 (validate)，前端无后端改动也不会传该字段，DB 默认 'major'，behavior 不变。前端补完拆 follow-up（轻量任务，登记为本任务的子项 followup 记到 §10 即可）。

### 2.8 Test 影响范围

3 处 test 文件含 BackupPolicy literal 不 set AlertSeverity：
- policy_service_test.go（5+ literal）
- policy_monitor_test.go（8 既有 case literal）
- policy_storage_monitor_test.go（本会话刚创建的 makeStoragePolicy helper + 13+ 单测）

策略：
- `DefaultPolicy()` 升级 — 新增字段；测试通过 monPolicyRepo 返回的策略走 PolicyService.Get 路径，Get 路径不 validate（仅 Update 验证），所以 literal 不设 AlertSeverity 不会让 Get 失败
- runtime fallback 保证发出的 alarm 仍是 "major"，与既有断言兼容
- 仅给关键 case 加 AlertSeverity 字段（policy_service_test 验证 + 至少 1 个 publisher case 验证 policy-driven 走完整链路）

实测 fallback 后 0 既有 case 需修改；新加 ~3 测试覆盖新分支。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维（生产）| 备份失败应该 PagerDuty oncall 响铃；我把 backup policy.alert_severity = "critical" 让告警走 PD 通道 |
| 网管运维（测试）| 测试环境频繁备份失败但运维不希望被骚扰；我把 alert_severity = "warning" 走 log-only |
| 网管运维（默认）| 不主动配置时沿用 major（生产稳态默认），与未升级前行为一致 |
| 后端开发 | 两处 severity 硬编码 + TODO 注释终于关闭；新加告警类型也按同样字段消费 policy.AlertSeverity |
| QA | 看到 backup_storage_threshold_exceeded 与 backup_task_failed 走同一 severity 字段，确信运维一致性 |

---

## 4. 验收标准（GWT）

### V1 — DefaultPolicy().AlertSeverity = "major"
- **Given** backup_policies 表为空，PolicyService.Get(ctx) 调用
- **When** 返回 DefaultPolicy()
- **Then** policy.AlertSeverity == "major"

### V2 — Migration backfill 既有行
- **Given** migration 000050 在含 1 行（alert_severity 列尚未存在）的 backup_policies 上执行 Up
- **When** ALTER TABLE 完成
- **Then** 既有行 alert_severity = 'major'（NOT NULL DEFAULT backfill）；CHECK 约束生效

### V3 — Validate 接受白名单
- **Given** PolicyRequest{AlertSeverity: "warning"} OR "major" OR "critical"
- **When** PUT /api/v1/backup/policy
- **Then** 200 OK，policy 持久化；其余字段保持

### V4 — Validate 拒绝白名单外
- **Given** PolicyRequest{AlertSeverity: "emergency"} 或 ""（空字符串）
- **When** PUT /api/v1/backup/policy
- **Then** 400 ErrInvalidInput；错误消息含字段名和合法值

### V5 — backup_task_failed alarm 走 policy.AlertSeverity
- **Given** policy.AlertSeverity = "critical"，policy.AlertOnFailure = true，task 失败
- **When** PublishFailureAlarm 调用
- **Then** 发布的 FailureAlarmPayload.Severity == "critical"

### V6 — backup_storage_threshold_exceeded alarm 走 policy.AlertSeverity
- **Given** policy.AlertSeverity = "warning"，policy.AlertOnFailure = true，bucket 超阈值
- **When** PolicyMonitor.RunStorageCheckOnce 触发 raised
- **Then** 发布的 StorageThresholdAlarmPayload.Severity == "warning"

### V7 — alarm.cleared 也走 policy.AlertSeverity
- **Given** policy.AlertSeverity = "critical"，above→below 跃迁
- **When** PolicyMonitor.RunStorageCheckOnce 触发 cleared
- **Then** alarm.cleared payload.Severity == "critical"（与 raised 一致便于 alarm 引擎对账）

### V8 — Empty AlertSeverity runtime fallback
- **Given** policy.AlertSeverity == "" （legacy / test edge case）
- **When** publishFailureAlarm 或 PublishStorageThresholdAlarm 触发
- **Then** payload.Severity == "major"（fallback 到 alarmSeverityMajor）

### V9 — Backward compat
- **Given** 既有 backup 测试套（policy_service_test 18 case + policy_monitor_test 8 case + policy_storage_monitor_test 13+ case）
- **When** `go test -race ./internal/backup/...`
- **Then** 全部通过（runtime fallback 保证未显式 set AlertSeverity 的 literal 仍走 "major"）

---

## 5. 运营商差异矩阵

无差异。三家运营商对 OMC 内部 severity 配置无感知。运维侧对接的 PagerDuty / 短信 / 邮件路由逻辑可能不同，但归 alarm 引擎 filter 配置（T-0007 已处理），与本任务 severity 字段独立。

---

## 6. 非目标

| # | 非目标 | 原因 / 后续承接 |
|---|--------|----------------|
| N1 | 前端 BackupPolicy form 加 alert_severity select | 单独 followup（轻量 S 任务），不阻塞后端启用；DB 默认 'major' 后端 PUT 接受空时返 400，FE 无改动也不会传 |
| N2 | 双字段 severity（failure / storage 各自一个） | §2.1 决策接受单字段；未来有 split 需求时加 column override |
| N3 | minor / emergency 等级 | §2.2 缩集到 3 档；future 加值容易（CHECK 放宽 + map 加值） |
| N4 | severity-based alarm filter rule（如 critical 必走 SMS） | F04 alarm engine 既有 filter rule（match by source/identifier/severity）已能做；T-0084 仅提供输入字段，filter 配置归运维 |
| N5 | severity 历史变更审计 | 当前 backup_policies 无审计表；如需，未来加 policy_history 表 |
| N6 | 设备级覆盖 severity（按 device 单独）| 当前 policy 是单 instance；按设备 override 拆未来任务 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0082 disk threshold | ✅ done — 提供 storage alarm publisher 让本任务在其上加 severity 注入 |
| T-0073 cleanup + failure alarm | ✅ done — 提供 failure alarm publisher 让本任务删 TODO |
| T-0071 BackupPolicy 持久化 | ✅ done — schema 持久化层已就位，加列只需 ALTER |
| Goose migration max=000049 | ✅ — 本任务用 000050 |
| `event.SubjectAlarmRaised` / `event.SubjectAlarmCleared` | ✅ |

无新外部依赖。

---

## 8. 度量

无新增 metric — severity 是 alarm payload 字段，从 alarm 引擎下游 filter / dispatcher 角度可见，无独立 backup-side metric 需求。

既有 `backup_failure_alarm_published_total{kind}` + `omc_backup_storage_threshold_alarm_total{kind}` 不变（kind 是 raised/cleared/skipped/error，与 severity 正交）。

**反例监控**：
- alarm 引擎下游若收到 severity ∉ {warning, major, critical}，记日志 — 但归 alarm 引擎职责，不在本任务

---

## 9. 设计备忘（S2）

### 9.1 Schema migration 000050

```sql
-- +goose Up
-- T-0084 / R-102 followup: BackupPolicy.AlertSeverity column
--
-- 新增 alert_severity 列让运维分级响应：warning / major / critical。
-- 关闭 policy_alarm_publisher.go + policy_storage_alarm.go 两处 severity TODO。
-- DEFAULT 'major' 让既有行 backfill 与硬编码常量行为一致 — 升级零运维感知。

ALTER TABLE backup_policies
    ADD COLUMN IF NOT EXISTS alert_severity VARCHAR(16) NOT NULL DEFAULT 'major'
    CHECK (alert_severity IN ('warning', 'major', 'critical'));

-- +goose Down
ALTER TABLE backup_policies DROP COLUMN IF EXISTS alert_severity;
```

`ADD COLUMN IF NOT EXISTS` 让重跑安全（与 5.5 §幂等性要求一致）。

### 9.2 BackupPolicy struct + DefaultPolicy

`policy_model.go`：
```go
// 告警（enforcement = T-0073/T-0082，severity = T-0084）
AlertOnFailure        bool   `json:"alert_on_failure"`
AlertEmail            string `json:"alert_email"`
AlertThresholdPercent int    `json:"alert_threshold_percent"`
AlertSeverity         string `json:"alert_severity"` // T-0084: warning|major|critical
```

`DefaultPolicy()` += `AlertSeverity: "major"`。

```go
var validBackupPolicyAlertSeverities = map[string]struct{}{
    "warning": {}, "major": {}, "critical": {},
}
```

### 9.3 Service validate

`validatePolicy(p *BackupPolicy)` 末尾追加：
```go
if _, ok := validBackupPolicyAlertSeverities[p.AlertSeverity]; !ok {
    return fmt.Errorf("invalid alert_severity %q (must be warning|major|critical): %w",
        p.AlertSeverity, commonerrors.ErrInvalidInput)
}
```

### 9.4 Repository

`policy_pg_repository.go`：
- `policyColumns` 末尾追加 `"alert_severity"`（在 timestamps 之前）
- `scanPolicy` 末尾追加 `&p.AlertSeverity`（顺序对齐）
- `Upsert` SQL：INSERT 列表 + 占位符 + DO UPDATE SET 三处加 `alert_severity`

### 9.5 Handler

`policy_handler.go`：
- `PolicyRequest` 末尾追加 `AlertSeverity string`
- `toModel` 同步加字段拷贝

### 9.6 Publishers

`policy_alarm_publisher.go`：
```go
payload := FailureAlarmPayload{
    Source:     alarmSourceBackup,
    Severity:   severityOrDefault(policy.AlertSeverity),  // was alarmSeverityMajor + TODO
    Identifier: "backup_task_failed",
    ...
}
```

`policy_storage_alarm.go`：
```go
type StorageInfo struct {
    BucketName       string
    UsedBytes        int64
    CapacityBytes    int64
    UsagePercent     int
    ThresholdPercent int
    Severity         string  // T-0084: caller injects from policy.AlertSeverity
    AlertEmail       string
}
// in publisher:
payload := StorageThresholdAlarmPayload{
    Source:     alarmSourceBackup,
    Severity:   severityOrDefault(info.Severity),  // was alarmSeverityMajor
    ...
}
```

`severityOrDefault`：
```go
// severityOrDefault returns the policy-supplied severity, falling back to
// alarmSeverityMajor when empty. Defends against legacy/test BackupPolicy
// literals that don't set AlertSeverity (Update-path validation prevents
// empty values reaching DB; this is for runtime resilience).
func severityOrDefault(s string) string {
    if s == "" {
        return alarmSeverityMajor
    }
    return s
}
```

### 9.7 PolicyMonitor 注入

`policy_storage_monitor.go:RunStorageCheckOnce`：
```go
info := StorageInfo{
    BucketName:       CanonicalRestoreBucket,
    UsedBytes:        used,
    CapacityBytes:    capacity,
    UsagePercent:     usagePercent(used, capacity),
    ThresholdPercent: policy.AlertThresholdPercent,
    Severity:         policy.AlertSeverity,  // T-0084
    AlertEmail:       policy.AlertEmail,
}
```

### 9.8 文件清单

修改：
- `omcgo/migrations/000050_backup_alert_severity.sql`（新建）
- `omcgo/internal/backup/policy_model.go` — 字段 + DefaultPolicy + valid set
- `omcgo/internal/backup/policy_service.go` — validate 加 severity 校验
- `omcgo/internal/backup/policy_pg_repository.go` — columns/scan/upsert SQL
- `omcgo/internal/backup/policy_handler.go` — PolicyRequest + toModel
- `omcgo/internal/backup/policy_alarm_publisher.go` — payload 用 policy.AlertSeverity，删 TODO；加 severityOrDefault helper
- `omcgo/internal/backup/policy_storage_alarm.go` — StorageInfo + payload 用 info.Severity
- `omcgo/internal/backup/policy_storage_monitor.go` — RunStorageCheckOnce 注入 policy.AlertSeverity
- `omcgo/internal/backup/policy_service_test.go` — +3 case validate severity 白名单
- `omcgo/internal/backup/policy_alarm_publisher_test.go` — +1 case policy-driven severity
- `omcgo/internal/backup/policy_storage_monitor_test.go` — +1 case storage severity propagation

无新增 fields/接口签名变更（StorageInfo 是 internal struct，只在 policy_storage_*.go 互通）。

### 9.9 待定点

| 待定 | 决策 |
|------|------|
| FE form select 加 alert_severity 字段 | 拆 followup（轻量 S，FE 单独 PR） |
| alert_severity 与 alert_threshold_percent 联动（usage > 95% 时强制 critical）| 不做，违反"运维显式控制"原则；如需实现也是 T-0082 disk-threshold 升级版 |
| Carrier 适配点（不同运营商默认 severity 不同）| 三家运营商默认 'major'，无差异；如未来有差异通过 carrier registry 注入 DefaultPolicy |

---

## 10. 实施要点（非规范性）

预计涉及模块：
- `omcgo/migrations/000050_*.sql`（新建）
- `omcgo/internal/backup/`（修改 8 文件）

预计新增端点：无（仅扩 PolicyRequest 字段，PUT /backup/policy 路径不变）

预计工作量：S（约 0.5 人日）—— migration + 8 文件修改 + ~5 新测试 case

---

## 11. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | Claude（PM 代签）| 2026-04-29 | 单字段决策、3 档值集 |
| 架构师 | Claude（架构代签）| 2026-04-29 | runtime fallback + DB validate 双层防御 |
| 数据专家 | Claude（数据代签）| 2026-04-29 | ADD COLUMN NOT NULL DEFAULT backfill 安全；CHECK mirror map |
| 电信业务 | Claude（电信代签）| 2026-04-29 | 3GPP 32.111 缩集合理 |
| QA/发布经理 | Claude（QA 代签）| 2026-04-29 | 9 GWT 全可测；既有 case runtime fallback 兼容 |

---

## 12. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-04-29 | v1.0 | 初稿；S0 起草，单字段决策 + 3 档值集 + runtime fallback | Claude |
