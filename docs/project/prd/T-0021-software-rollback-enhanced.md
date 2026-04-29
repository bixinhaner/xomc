# PRD: Software 回退能力增强（T-0021 / R-101 完整关闭）

> **关联**: Backlog T-0021 / Sprint-05 / Domain=F06/software / Type=feat
> **作者**: Claude（代 Owner=电信业务专家）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施（A 方案：主会话全程深度协作）

---

## 1. 业务背景

T-0018 已落地灰度升级（commit `fd71e4f3`），明确"不擅自回滚（T-0021 专责）"。T-0019 落地前端 canary 消费（commit `4ebdc91c`）。

当前 `internal/software/RollbackDevices` 已能批量回退，但：
- ❌ **无回退原因 / 触发源记录** — 难以审计"为什么回退"
- ❌ **无 canary failure → 自动回退链路**（缺 opt-in 配置）
- ❌ **回退目标版本固定为 OriVersion** — 无法指定回退到任意旧版本
- ❌ **回退后无健康检查记录**

R-101 风险登记册已部分关闭（T-0018 灰度），本任务关闭剩余"回退增强"部分。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望回退时记录原因（"灰度阶段失败率超 5%"），事后审计可查 |
| 运营商客户 | 我希望 canary 阶段失败 → 暂停 → 运维评估 → **可选**自动回退（默认手动）|
| 软件 PM | 我希望回退能指定目标 firmware_id（不仅是 OriVersion），用于跳过有问题的中间版本 |
| QA | 我希望区分手动回退 / 自动回退 / 计划回退，metric 三类计数 |

---

## 3. 验收标准（GWT）

### V1 — RollbackRequest 增强字段
- **Given** RollbackRequest 接受 +3 字段（reason / source / target_firmware_id）
- **When** API 提交完整 JSON
- **Then** UpgradeTask 持久化新字段；source 缺省='manual'

### V2 — Canary auto-rollback opt-in
- **Given** CanaryStrategy.RollbackOnFailure=true + auto_advance=true
- **When** 阶段失败率 > 阈值
- **Then** 自动暂停 + **额外**触发 RollbackDevices(source='canary_failure')；metric `software_canary_stage_advance_total{result="auto_rollback"}` +1

### V3 — Canary auto-rollback opt-out（默认）
- **Given** CanaryStrategy.RollbackOnFailure=false（默认）
- **When** 阶段失败率 > 阈值
- **Then** 仅暂停 + 告警，**不**自动回退（与 T-0018 D4 一致）

### V4 — 指定目标 firmware
- **Given** RollbackRequest.TargetFirmwareID = uuid
- **When** RollbackDevices 执行
- **Then** sub_tasks 的 dest_version 取该 firmware version；OriVersion 取设备当前

### V5 — 缺省回退（向后兼容）
- **Given** RollbackRequest 不含 reason/source/target_firmware_id
- **When** RollbackDevices 执行
- **Then** source='manual'；target=OriVersion（旧路径不变）

### V6 — 3 metric grep
- `software_rollback_total{source}` counter
- `software_rollback_devices_total{source}` counter
- `software_rollback_with_target_total` counter

### V7 — 测试覆盖
- table-driven test 覆盖 5 source 值 + 4 target 路径

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 回退原因记录 | 一致 | 一致 | 一致 |
| auto-rollback 默认开关 | 一致（默认 false）| 一致 | 一致 |
| 实际差异 | **无** | **无** | **无** |

---

## 5. 非目标

- ❌ 不实现完整回退后健康检查（仅记 source/reason，inform 验证后续 PR）
- ❌ 不接前端 UI（T-0019 已建框架；此任务后续 PR 可加 reason/source 选择）
- ❌ 不实现回退失败重试（既有 RetryUpgrade 路径已支持）
- ❌ 不改 RollbackExecutor 单设备执行逻辑（仅元数据增强）
- ❌ 不实现兼容性触发回退（compatibility source 留 enum，业务逻辑后续）

---

## 6. 依赖

- T-0018 ✅ done — canary state machine
- T-0019 ✅ done — 前端可消费（本任务字段加后前端可扩展）
- migration 000046（新增）

---

## 7. 设计备忘

### 7.1 Schema 改动 (migration 000046)

```sql
ALTER TABLE upgrade_tasks
    ADD COLUMN IF NOT EXISTS rollback_reason TEXT,
    ADD COLUMN IF NOT EXISTS rollback_source VARCHAR(32) DEFAULT 'manual'
        CHECK (rollback_source IN ('manual', 'canary_failure', 'compatibility', 'scheduled')),
    ADD COLUMN IF NOT EXISTS rollback_target_firmware_id UUID REFERENCES firmware_versions(id);

CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_rollback_source
    ON upgrade_tasks(rollback_source) WHERE task_type = 2; -- TaskTypeRollback
```

### 7.2 Type 扩展

```go
// model.go RollbackRequest +3 字段
type RollbackRequest struct {
    DeviceIDs        []uuid.UUID `json:"device_ids" binding:"required,min=1"`
    TaskName         string      `json:"task_name" binding:"required"`
    CreateUser       string      `json:"create_user" binding:"required"`
    CreateSuspended  bool        `json:"create_suspended"`
    Reason           string      `json:"reason,omitempty"`
    Source           string      `json:"source,omitempty"`           // default "manual"
    TargetFirmwareID *uuid.UUID  `json:"target_firmware_id,omitempty"`
}

// canary.go CanaryStrategy +RollbackOnFailure
type CanaryStrategy struct {
    Stages              []CanaryStage `json:"stages"`
    AutoAdvance         bool          `json:"auto_advance"`
    AutoAdvanceMinutes  int           `json:"auto_advance_minutes,omitempty"`
    RollbackOnFailure   bool          `json:"rollback_on_failure,omitempty"` // T-0021 opt-in
}
```

### 7.3 RollbackSource 常量

```go
const (
    RollbackSourceManual         = "manual"
    RollbackSourceCanaryFailure  = "canary_failure"
    RollbackSourceCompatibility  = "compatibility"
    RollbackSourceScheduled      = "scheduled"
)
```

### 7.4 Canary monitor auto-rollback 钩子

```go
// canary_monitor.go evaluateOne 新增分支：
//   if rate > threshold && rollback_on_failure { trigger RollbackDevices(source=canary_failure) }
//   既有 pause 路径不变（互补，不替换）
```

### 7.5 3 新 Prometheus metric

```
software_rollback_total{source}                  counter (manual/canary_failure/...)
software_rollback_devices_total{source}          counter (devices count by source)
software_rollback_with_target_total              counter (rollback specifying target firmware)
```

---

## 8. DoD

- [ ] PRD 七要素全 + 运营商差异矩阵
- [ ] migration 000046 up/down 配对
- [ ] go build / test -race / vet 全过
- [ ] 3 metric grep ≥1
- [ ] e2e_verify.sh +1 claim "rollback with reason/source/target"
- [ ] backlog T-0021 → done
- [ ] R-101 完整关闭（T-0018+T-0021 合力）

---

*PRD by /dev-pipeline pick T-0021 ULTRATHINK A 方案。*
