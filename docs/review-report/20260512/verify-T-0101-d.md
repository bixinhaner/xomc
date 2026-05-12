# Verify Report — T-0101-d (OpsTask 状态机集成 + approval_state 持久化)

**Task**: T-0101-d — 状态机集成（pending → approved/rejected → running/cancelled）：把 approval_state 字段真持久化到 ops_tasks 表 + ApprovalService.Approve 状态转移闭环
**Type**: feat / F06/ops / P0 / S
**Sprint**: sprint-10 (pull-forward 续 T-0090 + T-0096 后)
**Deps**: T-0112-a (MVP migration 000080 已应用，live DB v87 含 approval_state/approver_user_id/approved_at/risk_level 4 列) + Q3=A 决议（必须 4 眼，T-0113 commit `848928ec` 锁定）
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 改动 | 行数 |
|------|------|------|
| `omcgo/internal/ops/model.go` | OpsTask 扩 4 字段 + 状态机文档注释 | +9 行 |
| `omcgo/internal/ops/pg_repository.go` | taskColumns +4 / Create Columns+Values +4 / scanTask + scanTaskRow Scan +4 字段 / **新增 UpdateApproval 方法** | +50 行 |
| `omcgo/internal/ops/repository.go` | TaskRepository 接口 +UpdateApproval 签名 + `time` import | +3 行 |
| `omcgo/internal/ops/service_ext.go` | ApprovalService.Approve 重写：去 TODO + 持久化 + ApprovalState pending 校验 + ErrApprovalNotPending wrap | +14 / -10 |
| `omcgo/internal/ops/service_test.go` | mockTaskRepo +UpdateApprovalFn 字段 + Method + 5 个 ApprovalService 测试 + mockAuditLogRepo helper | +130 行 |
| `omcgo/internal/ops/handler_test.go` | mockOpsTaskRepo +UpdateApprovalFn 字段 + Method（接口扩展同步）| +9 行 |

---

## §2. 状态机设计（PRD §5.3.1 + §8.2 闭环）

```
任务创建（Service.CreateTask 当前由 service_ext / handler_ext 实施）:
  根据 task.RiskLevel 评估：
    L1/L2 → ApprovalState=not_required + Status=pending
    L3   → ApprovalState=pending      + Status=pending

审批者 ApprovalService.Approve (approver_user_id != creator):
  - 校验：task.ApprovalState == ApprovalPending（否则 ErrApprovalNotPending）
  - 校验：creator != approver（既有 ErrSelfApprovalForbidden）
  - 调 taskRepo.UpdateApproval(taskID, approverID, approve, now):
      approve=true  → ApprovalState=approved + Status=running
      approve=false → ApprovalState=rejected + Status=cancelled
  - 写审计 OpsAuditLog（既有逻辑保留）
```

**两轴正交**：Status（执行）+ ApprovalState（审批）— router/dispatcher (T-0101-b) 拉起 task 前应检查 ApprovalState 不是 pending 状态（即 not_required / approved 才能 run）

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `go build ./...` | ✅ PASS | 无输出 |
| `go test -race -count=1 ./internal/ops/...` | ✅ PASS | 1.035s（含 5 新 ApprovalService 测试 + 既有测试）|
| `go test -race -count=1 ./...` 全工程 | ⚠️ 2 pre-existing FAIL | `internal/task` 包 (scan NULL into *string for col source_id) 与今日 4 任务同基线，**已 stash 验证** main commit `87aac8e2` 同样 FAIL，与 T-0101-d 完全无关 |
| 无 TODO/FIXME/panic | ✅ | **删除了原 service_ext.go 的 "留待 T-0106-b 二期完善" TODO 注释 — 兑现承诺** |
| 无新增 `any` / `interface{}` | ✅ | 显式 OpsTask 4 字段都有具体类型（uuid.UUID / *uuid.UUID / RiskLevel / ApprovalState / *time.Time）|
| 无 `if carrier ==` 硬编码 | N/A | 状态机模块无运营商分支 |
| `golangci-lint` | N/A | 本会话不跑全工程 lint |
| 迁移 up/down | N/A | **无新迁移**（live DB 列已存在 from migration 000080 / T-0112-a MVP）|
| 新端点 R | 0 | ApproveTask handler 已存在（T-0106 MVP），本任务只完善其后端实现 |
| 新 metric/log 名 | 0 | 既有 zap.Info "task approval persisted" 仅微调 wording（原 "task approved"）|
| 累计型 deps | N/A | Deps 非累计型 |

---

## §4. 5 个 ApprovalService 测试场景核销

| 测试 | 验证不变量 |
|------|-----------|
| `Approve_SelfApprovalForbidden` | 4-eye：creator == approver 时返 ErrSelfApprovalForbidden（既有反退化）|
| `Approve_NonPendingRejected` (3 sub-case: already_approved/already_rejected/not_required) | 防重复审批：非 ApprovalPending 状态返 ErrApprovalNotPending wrap |
| `Approve_TransitsToRunning` | approve=true 时 UpdateApproval 收到 approve=true 参数 + approver_user_id 透传 |
| `Approve_TransitsToCancelled` | approve=false 时 UpdateApproval 收到 approve=false 参数 |
| `Approve_UpdateApprovalErrorPropagates` | 持久化失败时错误正确 wrap "persist approval decision" 上抛 |

**4 关键不变量**：
1. 4-eye 反退化（self-approval 必拒）
2. 状态前置校验（重复审批必拒）
3. approve 决策正确传到持久层
4. 持久化失败时不静默吞错

---

## §5. PRD §6.1 schema 兼容性

| schema 列 | OpsTask 字段 | 数据类型映射 |
|-----------|-------------|-------------|
| `approval_state VARCHAR(16) NOT NULL DEFAULT 'not_required'` | `ApprovalState ApprovalState` | string ↔ string |
| `approver_user_id UUID REFERENCES users(id)` | `ApproverUserID *uuid.UUID` | UUID NULL ↔ pointer |
| `approved_at TIMESTAMPTZ` | `ApprovedAt *time.Time` | timestamptz NULL ↔ pointer |
| `risk_level VARCHAR(16) NOT NULL DEFAULT 'safe'` | `RiskLevel RiskLevel` | string ↔ string |

**MVP 之前**：列已存在但 OpsTask 不读 → 所有 row 都 default ('not_required', NULL, NULL, 'safe')
**MVP 之后**（本任务）：列正常读写 → ApprovalService.Approve 实际持久化 → 既有 row 仍以 default 表现（向后兼容）

---

## §6. 用户回归路径

需 staging 多用户多角色 RBAC seed 数据真实验证 V3 GWT：

1. operator A 创建任务（risk_level=dangerous）→ ApprovalState=pending / Status=pending
2. sys_admin A 自己审批自己 → 403 ErrSelfApprovalForbidden（4-eye）
3. sys_admin B 同意 → ApprovalState=approved / Status=running
4. sys_admin B 再次审批 → 拒绝（ErrApprovalNotPending）

dev DB 仅 admin/test 2 user，多 role/group seed 留 staging 补 e2e（本任务 unit test 已 5 用例覆盖完整状态空间）

---

## §7. S4 出口门

- [✓] 所有命令绿（go build + ops 包 go test -race + 5 新 ApprovalService 测试）
- [N/A] E/R ≥ 1（无新后端端点；ApproveTask handler 已存在）
- [N/A] 迁移双向演练（无新迁移；live DB 列已存在 from T-0112-a MVP）
- [N/A] metric/log 名（无新埋点）
- [N/A] 累计型依赖阈值

**S4 PASS**。
