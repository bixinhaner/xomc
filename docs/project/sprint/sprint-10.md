# Sprint-10 规划 & 回顾

> Sprint Planning by `/dev-pipeline plan sprint-10` Path A 保守方案。
> 守护人：项目经理 + 自我管理（Claude 作为 Owner）

---

**Sprint ID**：10
**窗口**：2026-05-12 ~ 2026-05-25（2 周）
**主题**：Wave-3 finishing 后清账 — F10 互操作扩充 + MML 控制台 UX 整改前期拆分
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`（GA 准备期）

---

## 1. Sprint 目标（Sprint Goal）

本 Sprint 让 OMC 在 **F10 互操作测试覆盖度 + MML 控制台用户体验整改路线图** 两个方向有可衡量推进。具体：
- F10 互操作用例库扩展到 RC 级质量（关闭 R-202）
- T-0090 MML UX 整改 7 子项完成 S2 设计拆分（产出 4 sub-task 待 sprint-11 执行）
- T-0097 MML 保存脚本 toast bug 复现 + 修复或证伪

---

## 2. 承诺项（Committed）

| # | ID | 工作项 | Owner | 估算 | 关联 | 优先级 |
|---|----|--------|-------|------|------|-------|
| 1 | T-0030 | F10 互操作用例库扩充（GA 级质量补强）| Claude | L (3-4 天) | R-202 | P2 |
| 2 | T-0097 | MML console "保存脚本" toast 失败提示 bug | Claude | S (0.5-1 天) | R-NEW（toast util 场景失效面）| P2 |
| 3 | T-0090 | MML 控制台 UX 整改 — **本 Sprint 仅 S2 拆分**（产出 a/b/c/d 4 sub-task 待 sprint-11 执行）| Claude | S (0.5 天) | R-NEW（productTypes 删除 + RBAC 私有命令分组）| P2 |

**容量小计**：~4-5 工作日 / 6 可用工作日（buffer 20%）

**工作量规则**：
- T-0030: L = 6-10 工作日规则下属 borderline；本 Sprint 内做"扩充"具体到几个用例库，剩余可延入 sprint-11
- T-0097: S 类 bug；pre-pick 必须先复现（dev real-API 模式 + 浏览器 console 实测）
- T-0090: 仅做 S2 设计拆分，不实际执行 a/b/c/d 实现（XL 不应整体进 Sprint）

---

## 3. Stretch Goals（如有余力）

| # | ID | 工作项 | 估算 |
|---|----|--------|------|
| 1 | T-0096 | MML script 弹窗取消产品类型字段（跟随 T-0090a，本 Sprint 设计阶段一并 review）| S |

---

## 4. 依赖与阻塞

| 依赖项 | 阻塞什么 | 预计解除 | Owner |
|-------|---------|---------|-------|
| T-0097 复现步骤 | T-0097 pick S3 实施 | Sprint-10 D1 复现尝试 | Claude（dev 实测）|
| T-0090 S2 拆分 owner 评审 | T-0090 a/b/c/d 实施进 sprint-11 | Sprint-10 D5 拆分完成 + Claude self-review | Claude |

**外部 trigger 任务**（不进 sprint-10）：
- T-0091 KMS 适配器 — 等运维选定 AWS/Vault/HSM
- T-0093 SFTP/FTPS auth probe — 等实际部署需求
- T-0035 多皮肤 Phase 3-7 — XL 任务，需独立 S2 拆 P3/P4/P5/P6/P7 后再排

---

## 5. 每日进展（Daily Standup，可选）

**2026-05-11 周日**（Sprint 启动前预热）：T-0090 S2 拆分完成（commit `a71ab4a9`）；T-0097 pre-pick 调研 + S3 实施 + S4 typecheck + S6 commit `5b281b06`（AddTemplateModal 切 App.useApp() scoped messageApi）+ S7 backlog state-writeback → done。两个 sprint-10 deliverable 提前完成；剩余 sprint 工作日全部留给 T-0030 F10 用例库扩充
**2026-05-12 周一**：T-0030 F10 用例库扩充 D1
**2026-05-13 周二**：T-0030 D2
**2026-05-14 周三**：T-0030 D3
...

---

## 6. Sprint 回顾（最后一天填写）

- 完成情况：
- 未完成项：
- 经验教训：
- 改进点：

---

## 7. 关联文档

- Backlog 主表：`docs/project/backlog.md`
- T-0090 sub-task 表（拆分产出）：`docs/project/backlog/subtasks/T-0090-mml-ux-rework.md`（拆分后创建）
- F06-license PRD（关联 review report）：`docs/project/prd/F06-license.md`
