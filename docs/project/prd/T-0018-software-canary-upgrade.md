# PRD: Software 灰度升级策略（T-0018 / R-101）

> **关联**: Backlog T-0018 / Sprint-04..05 / Domain=F06/software / Type=feat
> **作者**: Claude（代 Owner=电信业务专家）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施（A 方案：主会话全程深度协作）

---

## 1. 业务背景

OMC 当前 `internal/software/BatchUpgrade(req)` 只支持**全量并发控制** (`MaxConcurrent`)，一次性将所有 sub-tasks 提交给设备升级。R-101 风险登记册描述：

> "批量升级仅支持全量并发控制，无分组/延时/百分比策略；大规模升级无法分批验证，失败面影响全量"

商用场景：单次升级 1000+ 基站，全网同步升级一旦 firmware 有 bug → **整个网络瘫痪**。运营商要求**灰度（canary）发布**：先升 1% 设备 → 验证健康 → 10% → 50% → 100%，每阶段失败率超阈值自动暂停告警。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望大批量升级走 1%/10%/50%/100% 四阶段，每阶段验证后再推进，发现问题立即暂停 |
| 运营商客户 | 我希望失败率超 5%（首阶段）/ 10%/20%/30% 自动暂停 + 告警，不擅自回滚（回滚是独立动作）|
| 软件 PM | 我希望可定制阶段（不仅 1/10/50/100，可配 5/25/100 等）|
| dev/test | 我希望 `auto_advance` 模式自动推进（无需人工干预）做 CI 测试 |

---

## 3. 验收标准（Given-When-Then）

### V1 — Strategy 路由分支
- **Given** `BatchUpgrade(req)` 入参 `req.Strategy="full"` 或缺省
- **When** 调用
- **Then** 走原 BatchUpgrade 路径（向后兼容）

### V2 — Canary 创建第一阶段
- **Given** `req.Strategy="canary"` + 100 设备 + 默认 stages [1, 10, 50, 100]
- **When** 调用
- **Then** 创建 100 个 sub_tasks（全量），但**仅**第 1 阶段（1 个设备 = 1%）的 sub_tasks 进入执行（其余 stage_status='pending'）

### V3 — 阶段失败率阈值触发自动暂停
- **Given** stage 1 1 设备升级失败率 100% > 阈值 5%
- **When** cron monitor 检查
- **Then** task.stage_status='paused' + 告警写入 + 不推进下一阶段

### V4 — 手动 advance
- **Given** stage 1 完成（失败率 < 阈值）
- **When** `POST /upgrade-tasks/:id/advance`
- **Then** 推进 stage 2（10% 设备进入执行）

### V5 — 手动 pause / resume / abort
- **Given** canary 任务正在 running
- **When** `POST /upgrade-tasks/:id/pause` (或 resume / abort)
- **Then** stage_status 更新对应

### V6 — auto_advance 自动推进
- **Given** `auto_advance=true` + auto_advance_minutes=5
- **When** 当前 stage 完成 + 5min 间隔后 cron 触发
- **Then** 自动推进下一阶段

### V7 — 4 metric 落地
- **Given** canary 流程跑完
- **When** `grep -rn` 指标名
- **Then** 4 个 metric 都 ≥1

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 灰度策略 | 一致（默认 1/10/50/100）| 一致 | 一致 |
| 失败率阈值 | 一致（默认 5/10/20/30%）| 一致 | 一致 |
| 实际差异 | **无**（升级流程通用）| **无** | **无** |

**结论**：T-0018 对运营商透明，不需要 Carrier 适配点。

---

## 5. 非目标

- ❌ 不实现自动回滚（T-0021 专责）
- ❌ 不补齐 UI（T-0019 前端 Software 后续 PR 消费此 task 的 API）
- ❌ 不实现复杂"健康度"指标（仅失败率；后续可扩展 RTT、connection request 成功率等）
- ❌ 不实现 stage 内 device 选择策略（按 ID 排序前 N% 即可，无 stratified sampling）
- ❌ 不实现 Web UI 强类型 stages 编辑（前端 textarea 输入 JSON）

---

## 6. 依赖

- 既有 `software/` 模块 + state_machine
- `github.com/robfig/cron/v3`（已有）
- migration 000045（新增）
- `internal/alarm`（告警告知接口；本任务通过 logger.Warn 占位，后续接 alarm.RaiseAlarm）

---

## 7. 度量（4 个 Prometheus 指标）

```
software_canary_stage_advance_total{result}            counter (advanced/paused/aborted/threshold_exceeded)
software_canary_stage_failure_rate{stage}              gauge   (0-1)
software_canary_active_tasks                           gauge
software_canary_devices_in_stage{stage}                gauge
```

---

## 8. 设计备忘

### 8.1 6 项决策（默认 ULTRATHINK 推荐）

| D | 选项 | 实施 |
|---|------|------|
| D1 | B 自定义 stages JSON | `canary_stages JSONB`，默认 `[{percent:1,threshold:5},{10,10},{50,20},{100,30}]` |
| D2 | C hybrid | `auto_advance BOOLEAN` + `auto_advance_minutes INT` |
| D3 | B 每阶段独立阈值 | 嵌入 stages JSON 数组 |
| D4 | A 自动暂停 + 告警 | logger.Warn + metric `result=threshold_exceeded` |
| D5 | B 仅核心调度第一版 | schema + service + 4 API + cron + 测试，UI 后续 |
| D6 | C 加 strategy 参数 | `BatchUpgradeRequest.Strategy` 默认 `"full"` |

### 8.2 Migration 000045 schema

```sql
ALTER TABLE upgrade_tasks
    ADD COLUMN strategy VARCHAR(16) NOT NULL DEFAULT 'full'
        CHECK (strategy IN ('full', 'canary')),
    ADD COLUMN canary_stages JSONB,
    ADD COLUMN current_stage INT NOT NULL DEFAULT 0,
    ADD COLUMN stage_status VARCHAR(20) DEFAULT 'pending'
        CHECK (stage_status IN ('pending','running','paused','aborted','completed')),
    ADD COLUMN stage_history JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN auto_advance BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN auto_advance_minutes INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_canary_active
    ON upgrade_tasks(strategy, stage_status)
    WHERE strategy='canary' AND stage_status IN ('running','paused');
```

### 8.3 接口

```go
// canary.go
type CanaryStage struct {
    Percent          int `json:"percent"`            // 累计百分比
    FailureThreshold int `json:"failure_threshold"`  // 该阶段失败率阈值（%）
}
type CanaryStrategy struct {
    Stages              []CanaryStage `json:"stages"`
    AutoAdvance         bool          `json:"auto_advance"`
    AutoAdvanceMinutes  int           `json:"auto_advance_minutes,omitempty"`
}
var DefaultCanaryStages = []CanaryStage{
    {Percent: 1, FailureThreshold: 5},
    {Percent: 10, FailureThreshold: 10},
    {Percent: 50, FailureThreshold: 20},
    {Percent: 100, FailureThreshold: 30},
}

// service.go (扩展)
func (s *SoftwareService) AdvanceStage(ctx, taskID) error
func (s *SoftwareService) PauseStage(ctx, taskID) error
func (s *SoftwareService) ResumeStage(ctx, taskID) error
func (s *SoftwareService) AbortCanary(ctx, taskID) error
```

### 8.4 4 新端点

```
POST /api/v1/upgrade-tasks/:id/advance
POST /api/v1/upgrade-tasks/:id/pause-canary
POST /api/v1/upgrade-tasks/:id/resume-canary
POST /api/v1/upgrade-tasks/:id/abort-canary
```

(命名 `pause-canary` / `resume-canary` 避免与既有 `suspend` / `resume` 冲突)

### 8.5 Cron monitor

每 1 min 扫 `strategy='canary' AND stage_status='running'`：
- 当前 stage 失败率 > 阈值 → pause + metric + log
- 当前 stage 完成（success+fail==stage_devices）+ failure_rate < threshold + auto_advance + 间隔到 → advance

---

## 9. DoD

- [ ] PRD 七要素全 + 运营商差异矩阵
- [ ] migration 000045 up/down 配对
- [ ] go build / test -race / vet 全过
- [ ] canary 单测覆盖 ≥ 70%
- [ ] 4 metric grep ≥1
- [ ] e2e_verify.sh +1 claim
- [ ] backlog T-0018 → done
- [ ] R-101 → Closed

---

*本 PRD 由 dev-pipeline /pick T-0018 ULTRATHINK A 方案生成。*
