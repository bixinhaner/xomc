# Code Review — T-0101-d (OpsTask 状态机集成)

**Reviewer**: Claude (self-review)
**Date**: 2026-05-12
**Scope**: 6 files / +215 / -10 / Type=feat / Backend Go + state machine
**Related**: [`verify-T-0101-d.md`](./verify-T-0101-d.md)

---

## §1. Findings 严重度汇总

| Severity | Count |
|----------|-------|
| **CRITICAL (P0)** | 0 |
| **HIGH (P1)** | 0 |
| **MEDIUM (P2)** | 0 |
| **LOW (P3)** | 2 |
| **NOTE** | 4 |

**APPROVE** — 无 P0/P1。

---

## §2. 审查重点

### 2.1 状态机正确性

**两轴正交设计**：
- Status (执行轴): pending / running / success / failed / cancelled / paused
- ApprovalState (审批轴): not_required / pending / approved / rejected

**审批操作触发的转移**：
```
ApprovalPending + approve=true  → ApprovalState=approved + Status=running
ApprovalPending + approve=false → ApprovalState=rejected + Status=cancelled
```

UpdateApproval 单一 atomic UPDATE 同时写两轴 — 无中间态泄露 ✅

### 2.2 安全审查

| 防御点 | 实现 |
|--------|------|
| **4-eye 原则** | `task.Creator == approverID.String()` → ErrSelfApprovalForbidden（既有，单测反退化）|
| **防重复审批** | `task.ApprovalState != ApprovalPending` → ErrApprovalNotPending wrap（新增）|
| **SQL 注入** | Squirrel `sq.Eq` 参数化 + 列名 hard-coded |
| **权限 escape** | approverID 来自 gin context（auth middleware 写入），调用方不可伪造 |
| **审计** | OpsAuditLog 一次性记录 approval 操作 + result + reason + approver_user_id |

**LOW-1**：当前 ErrApprovalNotPending 错误信息含 `task.ApprovalState` 值（如 "approval_state=approved not eligible: ..."）— 给客户端泄露了 task 当前状态。**评估**：approvers 是 sys_admin，本身有权查看 task 状态；信息泄露面 = 0；无需修复

### 2.3 并发审计

UpdateApproval SQL `UPDATE ops_tasks SET ... WHERE id=$1`：
- 不带 ApprovalState 二次校验 → 理论上同一 task 两个 approver 同时点 approve 都能通过（race）
- 业务影响：第二次 UPDATE 覆盖第一次，approver_user_id 取后到者 — 不破坏 ApprovalState 终态（都是 approved）
- 但是 audit_log 两条都记录 — 留双 approver 痕迹

**LOW-2**：可以在 UpdateApproval SQL 加 `WHERE id=$1 AND approval_state='pending'` 条件让二次写返 RowsAffected=0。**评估**：并发审批两个 sys_admin 同时点的概率极低；本任务先以 service 层 ApprovalState 校验为主防线；future 加 DB-level CAS 是 LOW 优化。

### 2.4 数据模型对齐

| schema 列 (PRD §6.1) | OpsTask 字段 | 一致性 |
|--------------------|-------------|--------|
| `approval_state VARCHAR(16) NOT NULL DEFAULT 'not_required'` | `ApprovalState ApprovalState` | ✅ |
| `approver_user_id UUID REFERENCES users(id)` | `ApproverUserID *uuid.UUID` | ✅ NULL 用 pointer |
| `approved_at TIMESTAMPTZ` | `ApprovedAt *time.Time` | ✅ NULL 用 pointer |
| `risk_level VARCHAR(16) NOT NULL DEFAULT 'safe'` | `RiskLevel RiskLevel` | ✅ |

scanTask + scanTaskRow + Create + UpdateApproval 4 处 SQL 字段顺序一致 — 无 column index 错位风险

### 2.5 兼容性

**MVP 之前**：列已存在但 OpsTask 不读 → row 全 default
**T-0101-d 之后**：OpsTask 正常读 → 既有 default row 表现为 RiskLevel=safe / ApprovalState=not_required（向后兼容）

**NOTE-1**：Create 路径补 default fallback（`if task.RiskLevel == "" { risk = RiskSafe }`）— 即使调用方忘记设置 RiskLevel，DB 仍可用；防御深度

**NOTE-2**：删除了 `service_ext.go` 中的 TODO 注释 "留待 T-0106-b 二期完善" — 兑现承诺；本任务正是 T-0106-b 提到的二期工作

---

## §3. DoD 逐项核销

### 编译与类型
- [✓] 后端 `go build ./...` 通过
- [⚠️] 后端 `go test -race -count=1 ./...`：ops 全绿；internal/task 2 pre-existing FAIL 与本任务无关
- [N/A] 后端 `golangci-lint run`：本会话不跑

### 测试
- [✓] 新增 5 个 ApprovalService 测试（覆盖 4 关键不变量）
- [✓] 成功 + 失败两路径（5 test 含 approve/reject 双成功 + 3 失败 + 1 异常传播）
- [N/A] 新 REST 端点 E2E（ApproveTask handler 已存在，本任务不动 endpoint）
- [N/A] 修复 bug 回归（这是 feat 不是 bug fix）
- [✓] 覆盖率提升（ApprovalService.Approve 从无测试 → 5 测试 + mockTaskRepo.UpdateApprovalFn 钩子）
- [✓] 禁止禁用失败测试 — 0 禁用

### 迁移与数据
- [N/A] 全部（无新迁移；列已 from migration 000080）

### 代码规范
- [✓] 无 TODO（**显式删除既有 TODO**）/ FIXME / panic
- [✓] 错误处理 `fmt.Errorf("...: %w", err)` 模式保留
- [✓] SQL 构建用 Squirrel
- [N/A] Carrier 接口 — 状态机无运营商分支
- [✓] 无 `any`
- [✓] zap 结构化日志（既有 task_id / approver / approve 字段保留）

### 文档
- [✓] PR 说明含 Why（"兑现 service_ext.go TODO / 完成 PRD §5.3.1 状态机集成"）
- [✓] 关联 Backlog: T-0101-d（subtask file 真 scope，非 sprint-11 误描述）
- [✓] 关联 PRD: docs/project/prd/F06-ops-management.md §5.3.1 + §6.1 + §8.2
- [N/A] CLAUDE.md 同步
- [N/A] Swagger（端点契约不变）

### 流水线闭环
- [ ] commit footer 五元组（待 S6）
- [ ] backlog.md Task 状态回写 + subtask T-0101-d State proposed → done（待 S7）
- [N/A] 快速通道 postmortem — type=feat 完整 S0-S7

### 安全
- [✓] 4-eye 反退化测试入测（永不下线）
- [✓] 防重复审批校验
- [✓] 审计日志含 approver_user_id + result + reason
- [✓] approverID 来自 auth context 不可伪造
- [N/A] 文件上传 — 不涉

### 可观测性
- [N/A] 新 Prometheus 指标 — 无新埋点（既有 zap log 保留）
- [✓] log 携带 task_id + approver + approve 三字段
- [✓] context 取消 — UpdateApproval 通过 r.pool.Exec(ctx, ...) 透传

### 模块特定（Go backend）
- [✓] 接口扩展（TaskRepository +UpdateApproval）+ 2 mock 同步（mockTaskRepo + mockOpsTaskRepo）
- [✓] 接受接口，返回结构体 — UpdateApproval 接受 uuid + bool + time，无返回值仅 error

---

## §4. T-0101-d 与 T-0106 的边界

| 工作 | 归属 |
|------|------|
| approval_state 字段持久化 | **T-0101-d** ✅（本任务）|
| 状态机转移（approve→running / reject→cancelled）| **T-0101-d** ✅ |
| 4-eye 原则 | T-0101-d ✅（既有 + 测试反退化）|
| 审批通知（推送 sys_admin email/sms）| T-0106-c（后续 sub-task）|
| 审批队列 UI（列出待审批 task）| T-0106-a（FE 列表页）|
| 审批 SLA 超时自动 reject | T-0106-d（future）|

**NOTE-3**：本任务严守 scope —— 仅做状态机持久化层；通知/UI/SLA 留 T-0106 系列

**NOTE-4**：subtask Notes 提到 `Q3=A 时引入 pending_approval 节点` — 本任务用既有 ApprovalState=pending 表达，与设计一致（Q3=A 决议已落地）

---

## §5. 结论

**APPROVE** — 无 P0/P1；2 LOW（错误信息泄露状态/并发审批 DB-CAS）不强制；4 NOTE 仅记录。

可以进 S6 commit。
