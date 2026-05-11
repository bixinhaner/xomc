# F06 运维管理 Wave 决议纪要与合规登记（2026-05-10）

**记录日期**：2026-05-11（追认）
**Wave 编号**：F06 ops-management 12 umbrella + 66 sub-task wave
**关联 PRD**：[`docs/project/prd/F06-ops-management.md`](../prd/F06-ops-management.md)（847 行）
**关联推进计划**：[`docs/project/F06-ops-management-implementation-plan.md`](../F06-ops-management-implementation-plan.md)（347 行）
**关联 Backlog Task**：T-0113（合规债清账）
**记录人**：Claude（代 Owner = PM + 架构专家 + QA/发布经理）

> **目的**：本 wave 的 12 umbrella 在 2026-05-10/11 走 dev-pipeline §A4 Path A
> 快速通道 commits ce36108f / 6e42a664 / c0485129 一次性 MVP 落地，PRD §13
> 列出的 6 个待决议项实际**已按默认建议落地**但未走正式决议会议。本文档
> **追认**这些决议 + 登记 dev-pipeline §B6 footer 偏差 + 列出 wave 期间漏项
> audit，关闭 T-0113 合规债。

---

## 第一部分：PRD §13 Q1-Q6 决议追认

每条决议结构：议题 → 采纳选项 → 理由 → 落地证据 → 反向影响。

### Q1. MML 模块与运维命令如何融合

**议题文本**：合并成一个"命令中心" / 双入口共存（建议）/ 隐藏 MML 入口

**采纳**：**B 双入口共存**

**理由**：
- MML 是按运营商规范文本协议执行（华为风格 LST/DSP/SET），有独立命令语法、参数体系、运营商验收要求
- 运维命令是按 TR-069 RPC 即时操作（reboot/factory_reset/get_param 等）
- 二者面向场景不同（MML 偏深度调试，Ops 偏一线即时处置），合并会产生"既不像 MML 也不像 RPC"的妥协 UX
- 已落地：保留 `internal/mml/` 独立模块 + 5 ops 页 / MML 控制台 + 5 mml 页 并行

**落地证据**：
- `internal/mml/`（MML 控制台 + 脚本任务，独立保留）
- `internal/ops/`（新增 25 ops 端点，commit `c0485129`）
- 前端 `pages/mml/*` + `pages/ops/*` 双导航入口（commit `6e42a664` 菜单 seed）

**反向影响**：用户需理解两套入口的边界 — 文档由各功能页 Tooltip 承担，不在本 wave 范围。

---

### Q2. 任务调度引擎放哪

**议题文本**：复用 `internal/task/` + 上层应用 orchestrator（建议）/ 独立队列

**采纳**：**A 复用 internal/task**

**理由**：
- `internal/task/` 已是统一任务队列（Redis + PG 双写、CWMP ID ↔ Task 映射、completion router）
- 独立队列 = 重复造轮子 + 双重 metrics + 双重故障面
- OpsTask 编排器作为应用层，调用底层 `task.CreateTask`，不接管派发原语

**落地证据**：
- `internal/ops/service_ext.go` TaskExecutor MVP 设计为 `internal/task` 上层 orchestrator
- DI provider/modules.go 已接 opsTaskRepo + opsExecRepo + opsAuditSvc，未额外创建 task 队列

**反向影响**：T-0101-d（步骤路由）实施时需 grep `internal/acs/rpc/` 接口可调用面（W1 待定点，sprint-11 D1 起手）。

---

### Q3. 高风险 4 眼审批是否首版必须

**议题文本**：必须 / 首版仅记录 + 警告

**采纳**：**A 必须**

**理由**：
- 等保 2.0 三级合规要求"重要操作可追溯"+ 高风险操作需双人审批
- 商用网管面向运营商 QA / 客户验收，"仅记录"会被等保 audit 卡
- 首版必须比后期补改成本低（涉及流程改造）

**落地证据**：
- `internal/ops/service_ext.go` ApprovalService.Approve 内 ErrSelfApprovalForbidden 校验（creator≠approver）
- `internal/ops/service_ext.go` EvaluateRiskLevel：> 50 设备 OR dangerous=L3 / > 10 OR cautious=L2 / 其他 L1
- `migration 000080` ops_tasks 加 approval_state / approver_user_id / approved_at 列

**反向影响**：单兵运维场景需"应急通道"— 由 T-0111 break-glass 服务承担（已 MVP 落地）。

---

### Q4. 诊断结果是否进入 KPI / MR 库

**议题文本**：独立 `ops_diagnostics` 表 / 进 PM 库

**采纳**：**A 独立 ops_diagnostics 表**

**理由**：
- 诊断结果（IPPing / TraceRoute / Throughput 等）是事件型短期数据，与 PM 性能时序数据（KPI 长期聚合）语义不同
- 进 PM 库会污染 KPI 计算引擎、模糊 PM/MR 模块边界
- 独立表后期可加 retention policy（90 天保留）独立于 PM 配置

**落地证据**：
- `migration 000081` 新建 `ops_diagnostics` 表（含 CHECK initiator ∈ device/omc / status 5 枚举 / 复合索引）
- `internal/ops/` DiagnosticService 三方法 Ping/Traceroute/Throughput 写本表，未触碰 PM 库

**反向影响**：诊断与 PM 数据关联分析需在应用层 JOIN（如"PM 异常时段诊断结果汇总"），由后续 T-0104-b..i 二期实现。

---

### Q5. 模板 steps[] schema 是否引入 JSON Schema 严格校验

**议题文本**：引入 / 仅文档约定（首版）/ 用 protobuf

**采纳**：**B 仅文档约定（首版）**

**理由**：
- 引入 JSON Schema 严格校验首版成本高（schema 定义 + validator + 错误码 + 前端校验同步）
- 首版 6 内置 OEM 模板均为 hardcoded（seed/000082），不存在用户输入风险
- protobuf 与 OpsTemplate 的 JSON-first API 不兼容
- V2 用户开放自定义模板时再引入

**落地证据**：
- `internal/ops/model_ext.go` TestCase.Steps 字段为 `JSONB` 无 validator
- `seed/000082` 6 内置模板的 steps[] 直接硬编码 JSON，无 schema check

**反向影响**：用户开放自定义模板时（V2）需补 schema 引入路径 — 登记为 known debt，未来 sub-task 独立登记不计入本 wave。

---

### Q6. 跨 OMC 实例（多集群）共享模板

**议题文本**：支持 / 不支持（首版）

**采纳**：**B 不支持（首版）**

**理由**：
- 100K 规模单 Go 进程 / 单集群已足够（333 sessions/s）
- 跨集群同步 = 分布式一致性问题（CRDT / 中心同步服务），首版 ROI 低
- 当前 OMC 部署模式为单运营商单集群

**落地证据**：
- `ops_templates` 表无 cluster_id / origin_instance / sync_status 等跨集群字段
- `internal/ops/pg_repo_ext.go` 查询条件无 cluster filter

**反向影响**：未来 100 万规模拆分微服务时同步问题作为架构演进议题，不在 F06 ops scope。

---

### 6 决议汇总

| Q# | 议题 | 采纳 | 落地 commit | 实施状态 |
|----|------|------|-----------|--------|
| Q1 | MML vs Ops 融合 | B 双入口 | `c0485129` + `6e42a664` | ✅ |
| Q2 | 调度引擎 | A 复用 internal/task | `c0485129` (MVP) | 🟡 MVP；T-0101-d 二期实接 |
| Q3 | 4 眼审批首版 | A 必须 | `c0485129` | ✅ ApprovalService |
| Q4 | 诊断结果库 | A 独立表 | `c0485129` migration 000081 | ✅ |
| Q5 | steps schema | B 仅文档约定 | `c0485129` | ✅ |
| Q6 | 跨集群共享 | B 不支持 | `c0485129` | ✅ |

**全部决议按 PRD §13 默认建议追认**。如未来 Q5/Q6 需翻盘（V2 自定义模板 / 100 万规模拆分），走独立决议会议 + 新 PRD 章节 / 新 ADR 文件，不在本 wave 范围。

---

## 第二部分：dev-pipeline §B6 footer 五元组偏差登记

### 偏差描述

dev-pipeline §D3 规定 commit footer 含正式五元组 `PRD: ... / Sprint: ... / Risk: ... / Backlog: ... / Review: ...`。F06 ops wave 3 commits 实际 footer **未完整**。

### 实测 footer 内容

| Commit | Date | Footer 状态 | 偏差描述 |
|--------|------|------------|---------|
| `ce36108f` | 2026-05-10 | 含 Impact + Related: F06 | 缺正式 PRD / Sprint / Risk / Backlog / Review 五元组 |
| `6e42a664` | 2026-05-10 | 含 Impact + Related + PRD reference | 缺 Sprint / Risk / Backlog / Review 四项 |
| `c0485129` | 2026-05-10 | 含 Impact + Related + 推进计划 + PRD | 缺 Sprint / Risk / Backlog / Review 四项 |

### 合理化（dev-pipeline §C.1 wave-batched 准入）

- 本 wave 触发 §C.1 wave-batched 模式（Wave 整改专用，准入 commits 跳过 S0/S1）
- §C.1 允许 `Skip: S0,S1 (per §C wave-batched)` 但仍要求挂 PRD/Sprint/Risk/Backlog/Review
- 实际 commits 由 dev-pipeline §A4 Path A 路径执行（Proposed → Done 跳过 S0-S5），**超出 §C.1 准入范围**（§C.1 仅允许 skip S0/S1）
- 属于"流程偏差"但不是"代码缺陷"：PRD/推进计划/sub-task 文件齐全，仅 footer 字段不符规范

### 后续防范

- 本文件登记为偏差记录
- 后续 wave 严格遵守 §B6 footer 五元组要求（即使 wave-batched 也补全）
- T-0113 关闭后不另起追溯 commit（追溯 amend 3 个 commit 成本高于价值，且改写历史 commit 风险大）

---

## 第三部分：Wave 期间漏项 Audit

### Audit 清单

| 项 | 状态 | 备注 |
|----|------|------|
| PRD 完整性 | ✅ | docs/project/prd/F06-ops-management.md 847 行齐全 |
| 推进计划 | ✅ | docs/project/F06-ops-management-implementation-plan.md 347 行 |
| 12 umbrella 登记 | ✅ | backlog.md + backlog/done/2026Q2.md |
| 66 sub-task 登记 | ✅ | backlog/subtasks/T-0101-ops-management.md |
| Risk 登记 | ✅ | R-T0098 系列 wave 自带 12 风险已在 risk-register.md |
| 12 umbrella Closing Evidence | ✅ | backlog/done/2026Q2.md 行 76-87 完整 |
| State writeback | ✅ | umbrella State proposed → done（commits `ce36108f`/`6e42a664`/`c0485129` 后）|
| 数据库迁移 | ✅ | migrations/000080/000081 + seed/000082 |
| 测试覆盖 | 🟡 | service_ext_test 覆盖 ApprovalService / DiagnosticService / DownloadService 等；MVP stub 部分（实 RPC 派发 / TR-181 Diagnostics）未单测，留 sub-task 二期 |
| 安全审查 | 🟡 | RBAC 8 权限点 seed/000082 全部接入 admin/operator/viewer 三角色；break-glass 强制告警通知 + 24h 复盘评注留 T-0111-c/d 二期 |
| dev-pipeline footer 五元组 | ❌ | 本文件第二部分登记，T-0113 关闭 |
| Q1-Q6 决议纪要 | ❌ | 本文件第一部分追认，T-0113 关闭 |

### 总结

- **完整项 7/12** + 部分项 2 + 缺项 3
- 3 缺项中 2 项（footer / 决议纪要）由 T-0113 本文件关闭
- 1 项（测试覆盖 stub 部分）由二期 sub-task 自然消化（T-0101-b..i / T-0104-b..i 等）
- 1 项（安全审查）部分项 = MVP 已合规，break-glass 二期补强后完整

---

## 关闭依据

- 本文件追认 6 决议（Q1-Q6）+ 登记 3 commits footer 偏差 + 完成 wave audit
- T-0113 backlog state triaged → done

## 参考资料

- PRD：`docs/project/prd/F06-ops-management.md` §13 待决议
- 推进计划：`docs/project/F06-ops-management-implementation-plan.md`
- Wave commits：`ce36108f` / `6e42a664` / `c0485129`
- Wave 收官 changelog：`docs/project/backlog/changelog.md` 2026-05-11 12 umbrella 行
- dev-pipeline §B6 + §C.1 + §D3：`.claude/commands/dev-pipeline.md`
- T-0113 backlog 行：`docs/project/backlog.md` §4 Triaged
