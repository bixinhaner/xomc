# Sprint-11 规划 & 回顾（DRAFT）

> **状态**：DRAFT — sprint-10 close 前的"前置 planning 备忘"，不是正式 committed。
> 等 sprint-10 close（2026-05-25）走一次正式 `/dev-pipeline plan sprint-11` 拍板后，本文件 §2 候选项才升为 Committed 并改 State→planned。
> 守护人：项目经理 + 自我管理（Claude 作为 Owner）
> 起草日期：2026-05-11（sprint-10 D-1 启动前夜）

---

**Sprint ID**：11
**窗口**：2026-05-26 ~ 2026-06-08（2 周）
**主题候选**：MML UX 主线下沉（T-0090 sub-task 全推）+ F06 ops MVP→生产级 P-1 通路启动
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`（GA 准备期）
**关联依据**：
- F06 ops 需求现状分析 → 推荐方案 B（4 sprint GA 路线图）
- sprint-10.md §2 — T-0090 仅做 S2 拆分，sub-task 在 sprint-11 执行
- `docs/project/backlog/subtasks/T-0090-mml-ux-rework.md`（4 sub-task）
- `docs/project/backlog/subtasks/T-0101-ops-management.md`（66 sub-task）

---

## 0. Draft 状态说明

| 项 | 状态 | 备注 |
|----|------|------|
| §2 候选承诺项 State | 仍为 `triaged` / `proposed`（未升 `planned`）| 等 sprint-10 close 时正式 plan |
| Sprint Goal 措辞 | 暂定 | sprint-10 close 时可调 |
| Owner | 默认 Claude | 同 |
| Stretch 候选 | T-0113 + T-0103 多页 + T-0102-b/c | 2026-05-11 D-1 扩展（sprint-10 提前完成 → buffer 增大）|

**为什么提前 draft**：方案 B（F06 ops 4 sprint GA 路线图）已锁定，sprint-11 候选项只待拍板；提前固化避免 sprint planning 当场争论 1-2 小时。

### 2026-05-11 D-1 进度更新（sprint-10 启动前夜）

| 事件 | 影响 sprint-11 draft |
|------|---------------------|
| **sprint-10 三 deliverable 全闭单晚完成** — T-0090 S2 拆分 / T-0097 fix / T-0030 双 Phase（用例 14→37 + R-202 Closed）+ Phase 3 三 task MVP（T-0114/T-0115/T-0116） | sprint-10 14 天窗口全部 buffer；sprint-11 候选项不变（T-0090 sub-task / T-0101-d / T-0102-a 仍待 sprint-11 执行），但 stretch 候选可大幅扩展 |
| **R-202 P2 风险已 Closed** | sprint-11 §4 依赖表删除"sprint-10 T-0030 完成"行（已 done）|
| **T-0115 CSV report MVP 已交付** | sprint-11 §8 sprint-13 路线"P-2 下载+诊断"中 T-0105-b transfer.Upload 真触发仍待做（CSV 是 export 报告，与 transfer.Upload 收日志/取配置无关），路线图不变 |
| **today 累计 17 commits 已 push** | origin/main 已含全部本会话产物，无 sprint-10 残余推送动作 |

**sprint-11 主线候选项稳定**：T-0090 a/b/c/d 4 sub-task + T-0101-d + T-0102-a 仍是 ~8-10d 容量主体；新 stretch 候选见 §3 扩展。

---

## 1. Sprint 目标候选（Sprint Goal）

本 Sprint 让 OMC 在 **MML 控制台 UX 整改 + F06 运维管理 MVP→生产级 P-1 通路启动** 两个方向同步推进：

- **T-0090 主线**：MML 控制台公/私命令新增页面 UX 整改 4 sub-task（a FE-only / b DB drop / c backend RBAC / d FE 私有页）全部完成 → 关闭 T-0090 umbrella + R-NEW-1..4 闭环
- **F06 ops P-1 通路启动**：完成 OpsTask 真执行链路最关键的两条 sub-task（T-0101-d 步骤路由 + T-0102-a 实 RPC 派发），让 GWT V1 单命令即时回执"从 stub 到真通过"，为后续 4 个 sprint 的 W1-W6 路线图开局

成功定义：
- T-0090 a/b/c/d 全 done + umbrella in_design → done
- T-0101-d 步骤路由实现并能调用 `internal/acs/rpc/` 派发；OpsTask 创建后 step.type=rpc_* 可真发到设备
- T-0102-a `/ops/commands/rpc` 端点实际派发 RPC（替换 stub 占位），SSE 流推送真 `command.completed{success:true}`
- GWT V1 在 dev real-API 环境实测可通过

---

## 2. 候选承诺项（DRAFT — Committed 待 sprint-10 close 时升）

| # | ID | 工作项 | Owner | Est | 关联 | Prio | State 候选 |
|---|----|--------|-------|------|------|-------|------|
| 1 | T-0090-a | MML 控制台 FE-only UX 调整（操作类型差异化展示 + 自定义命令限制） | Claude | S (~0.5d) | T-0090 主线 / R-NEW-1 | P2 | **done 2026-05-12（pull-forward 进 sprint-10 buffer 执行；commit `bf71e518`；详 `backlog/done/2026Q2.md`）** |
| 2 | T-0090-b | DB drop product_types 列（migration up/down + 数据迁移）| Claude | M (~1-2d) | T-0090 主线 / R-NEW-2 | P2 | **done 2026-05-12（pull-forward 进 sprint-10 buffer 执行；commit `86b19c3f`；详 `backlog/done/2026Q2.md`）** |
| 3 | T-0090-c | Backend RBAC 私有命令查询过滤（admin context + repo where 子句） | Claude | M-L (~2-3d) | T-0090 主线 / R-NEW-3 | P2 | **done 2026-05-12（pull-forward 进 sprint-10 buffer 执行；commit `2c114a40`；/security-review 0 P0/P1；详 `backlog/done/2026Q2.md`）** |
| 4 | T-0090-d | FE 私有命令页（复用公有页面组件 + 切换私有/公有 tab） | Claude | S (~0.5d) | T-0090 主线 / R-NEW-4 | P2 | **done 2026-05-12（pull-forward sprint-10 buffer 单晚；commit `396cb76d`；T-0090 umbrella 4/4 全闭）** |
| **5** | **T-0101-d** | **状态机集成（pending → approved/rejected → running/cancelled）** ← sprint-11 draft 原误描述为 "步骤路由"（实为 T-0101-b） | Claude | S | F06 ops 4 眼审批闭环 / R-O01 mitigation | P0 | **done 2026-05-12（pull-forward sprint-10 buffer；commit `1743e8d1`；ApprovalService.Approve 删 TODO 兑现；详 `backlog/done/2026Q2.md`）** |
| **6** | **T-0102-a** | **`/ops/commands/rpc` 实 RPC 派发到 acs**（替换 stub 占位） | Claude | M (~2d) | F06 ops P-1 通路 / unlock GWT V1 | P1 | planned |

**容量小计**：~8-10 工作日 / 11 可用工作日（buffer 20%，含 1 工作日 buffer）

**工作量规则**：
- T-0090 a/b/c/d 4 sub-task 估算合计 ~4-6d
- T-0101-d / T-0102-a 各 ~2d；优先实现"acs/rpc/ 现有 RPC handler 调用面是否能直接对接"调研（W1 待定点）→ 不行则需再加 adapter ~0.5d
- Sprint 容量不应再加 P-1 通路其他 sub-task（T-0101-b 并发控制 / T-0102-b MML 流式等），留 sprint-12 推

---

## 3. Stretch Goals 候选（如有余力）

> **2026-05-11 D-1 扩展**：sprint-10 三 deliverable 单晚完成后整个 14 天窗口富余，sprint-11 也可视情况吸纳更多 stretch — 但仍受 11d 总容量约束。优先级排序：T-0103-c（unlock 用户感知）> T-0113（合规债）> T-0103-b/d/e（更多页面接 API）> T-0102-b/c（MML 流式 + per-device 限流）。

| # | ID | 工作项 | 估算 | 触发条件 | 优先级 |
|---|----|--------|------|---------|------|
| 1 | T-0103-c | CommandManagement 页接 useOpsExt 新 API（替换 mock）| M (~1d) | T-0102-a 真 RPC 派发完成后立即做 — 让用户能"看到" P-1 通路价值 | **高**（升级候选 committed） |
| 2 | T-0113 | F06 ops wave 合规债清账（PRD §13 Q1-Q6 决议纪要补齐 + dev-pipeline §B6 footer 偏差登记）| S (~0.5d) | T-0090 + T-0101-d + T-0102-a 提早 D8 闭环时立即做 | 中 |
| 3 | T-0103-b | TaskManagement 页接 API + 设备级流水视图 | M (~1d) | T-0103-c 完成 + 还有余力 | 中 |
| 4 | T-0103-d | Templates 页接 API + Edit modal（模板导入导出可后置）| M (~1d) | 同上 | 中 |
| 5 | T-0102-b | MML 命令流式输出（多帧推送 SSE）| M (~1.5d) | sprint-12 主线，可仅做调研放 stretch | 低（sprint-12 主线，不进 sprint-11 committed）|
| 6 | T-0102-c | per-device RPC 限流（rate.Limiter）| S (~0.5d) | 同 T-0102-b | 低 |

**Stretch 容量约束**：sprint-11 主线 ~8-10d / 总 11d → 实际 stretch 余量 1-3d。**建议吸纳顺序**：T-0103-c（同 sprint 配对 T-0102-a 天然，1d 内可做）→ T-0113 清账（0.5d）→ T-0103-b 或 T-0103-d（1d）。**不应**今晚就升 committed — 等 sprint-11 D5 看主线进度再加塞。

T-0113 强调：**纯文档/纪要清账**，无代码改动，可在 sprint 任意阶段穿插做。

---

## 4. 依赖与阻塞

| 依赖项 | 阻塞什么 | 预计解除 | Owner |
|-------|---------|---------|-------|
| ~~sprint-10 T-0030 完成~~ | ~~不阻塞 sprint-11~~ | **✅ 已 done 2026-05-11 D-1** | — |
| T-0090 sub-task State 升 planned | 等本 draft sprint planning 拍板（T-0090 a/b/c/d 当前 triaged 在 `subtasks/T-0090-mml-ux-rework.md`） | 2026-05-25 sprint planning | Claude |
| T-0101-d / T-0102-a State 升 planned | 等本 draft sprint planning 拍板（66 sub-task 当前 proposed 在 `subtasks/T-0101-ops-management.md`）| 2026-05-25 sprint planning | Claude |
| W1 待定点：`internal/acs/rpc/` 是否提供 step.type 派发接口 | T-0101-d 实施时长上限 | sprint-11 D1 起手 grep | Claude |

**外部 trigger 任务**（不进 sprint-11）：T-0091 KMS / T-0093 SFTP / T-0035 多皮肤 P3-7（与 sprint-10 一致）

---

## 5. 每日进展（Daily Standup）— 待 sprint 启动后填

**2026-05-26 周一**：sprint-11 启动；T-0090-a 起手（最小 sub-task，热身）
**2026-05-27 周二**：T-0090-b DB drop migration（依赖 a，必须串行）
**2026-05-28 周三**：T-0090-c backend RBAC D1
**2026-05-29 周四**：T-0090-c D2 + T-0090-d FE 私有页（并行）
**2026-05-30 周五**：T-0090 收尾 + umbrella close + R-NEW 闭环；起 T-0101-d 调研
**2026-06-02 周一**：T-0101-d 实施 D1（`internal/acs/rpc/` 接口对接）
**2026-06-03 周二**：T-0101-d D2 收尾
**2026-06-04 周三**：T-0102-a 实施 D1
**2026-06-05 周四**：T-0102-a D2 收尾 + GWT V1 实测
**2026-06-06 周五**：Stretch（T-0113 / T-0103-c）或 buffer / sprint 回顾准备
**2026-06-08 周一**：sprint-11 close + sprint-12 plan（F06 ops P-1 通路收尾 + T-0103 前端 UI 整组）

---

## 6. Sprint 回顾（最后一天填写）

- 完成情况：
- 未完成项：
- 经验教训：
- 改进点：

---

## 7. 关联文档

- Backlog 主表：`docs/project/backlog.md`
- T-0090 sub-task 表：`docs/project/backlog/subtasks/T-0090-mml-ux-rework.md`
- T-0101..T-0112 sub-task 表：`docs/project/backlog/subtasks/T-0101-ops-management.md`
- F06 ops PRD：`docs/project/prd/F06-ops-management.md`
- F06 ops 实施推进计划：`docs/project/F06-ops-management-implementation-plan.md`
- F10 互操作 PRD（sprint-10 主线，参考用）：`docs/project/prd/F10-interop-testing-coverage.md`
- 推荐方案 B（F06 ops 4 sprint GA 路线图）：本文件 §1 + dev-pipeline 会话备忘（2026-05-11）

---

## 8. Sprint-12+ 后续路线图（参考，方案 B）

> **2026-05-11 D-1 提前完成红利**：sprint-10 富余 ~10 工作日（窗口 14 天，原计划只用 3-4 天给 T-0030 等）— 红利可流向：① sprint-11 stretch 候选区（见 §3）；② sprint-10 期间 ad-hoc 启动 sprint-11 主线候选项；③ buffer 留给突发 hotfix / 客户反馈。**不建议**今晚就把 sprint-11 候选项升 committed（违反"Sprint 不加塞"原则）。

| Sprint | 窗口 | 主线 |
|--------|------|------|
| 12 | 6/9-6/22 | F06 ops P-1 通路收尾：T-0101-b/c/e + T-0102-b/c + T-0103-b..f 5 ops 页 UI 全接 |
| 13 | 6/23-7/6 | F06 ops P-2 诊断+下载：T-0104-b/c/d + T-0105-b/c + T-0106-c/d |
| 14 上 | 7/7-7/15 | F06 ops P-3 治理+合规：T-0107-b/c/d + T-0109-c（**等保合规闭环**）|
| 14 下 + 15 | 7/15-8/3 | 体验优化：T-0108 巡检 + T-0110 知识库 + T-0111-c/d break-glass + 长尾 |

→ **3 sprint = 6 周到 GA 门槛**（2026-07-06）；含体验全完 2026-08-03。

**T-0030 + Phase 3 三 task 已 done 不进路线图**（F10 互操作模块已完工，R-202 Closed；剩余 PDF/markdown export / 设备×型号矩阵用例填充等 Phase 2 polish 项作长尾任务）。

详 dev-pipeline 会话记录 2026-05-11 ULTRATHINK 决策分析。
