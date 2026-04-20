# Sprint-NN 规划 & 回顾模板

> 复制此文件为 `sprint-01.md`、`sprint-02.md` 使用。  
> 规划在 Sprint 第一天产出；执行中逐日补充；回顾在 Sprint 最后一天完成。  
> 守护人：项目经理（`CLAUDE.md §16.11`）

---

**Sprint ID**：NN  
**窗口**：YYYY-MM-DD ~ YYYY-MM-DD（2 周）  
**主题**：一句话  
**关联 Milestone**：`docs/project/milestone/YYYY-QN-*.md`

---

## 1. Sprint 目标（Sprint Goal）

一句话。完成后应能说："本 Sprint 让 OMC 在 X 方面可以 Y 了"。

---

## 2. 承诺项（Committed）

列出本 Sprint 必达的工作项。每项需有 Owner、预计工作量、关联 Issue/PRD。

| # | 工作项 | Owner | 估算 | 关联 | 优先级 |
|---|-------|-------|------|------|--------|
| 1 | {任务描述} | {角色} | S/M/L | PRD or Issue | P0/P1 |
| 2 | | | | | |

**工作量规则**：
- S（Small）：≤ 2 人日
- M（Medium）：3-5 人日
- L（Large）：6-10 人日（> 10 人日必须拆分）

**容量预留**：总估算 ≤ 工作日数 × 80%（留 20% buffer 应对意外）

---

## 3. Stretch Goals（如有余力）

（选做项，不影响 Sprint 达成判定）

| # | 工作项 | Owner | 估算 |
|---|-------|-------|------|
| | | | |

---

## 4. 依赖与阻塞

| 依赖项 | 阻塞什么 | 预计解除 | Owner |
|-------|---------|---------|-------|
| | | | |

---

## 5. 每日进展（Daily Standup，可选）

**{YYYY-MM-DD 周 N}**
- 完成：…
- 进行中：…
- 阻塞：…

**{YYYY-MM-DD 周 N}**
- …

---

## 6. 回顾（Retro，Sprint 最后一天填写）

### 6.1 完成情况

| 承诺项 | 状态 | 备注 |
|-------|------|------|
| #1 | ✅ Done / ⚠️ Partial / ❌ Not Done | 原因 |
| #2 | | |

**承诺完成率**：M/N = XX%（目标 > 80%）

### 6.2 What Went Well（继续保持）
- …

### 6.3 What Didn't Go Well（需要改进）
- …

### 6.4 Action Items（下 Sprint 落地）
- [ ] {具体动作} — Owner {角色} — 截止 {日期}
- [ ] 

### 6.5 DoD 调整（如有）
- {哪条 DoD 项需要补充/修订}

### 6.6 Risk Register 更新（如有新风险或风险关闭）
- R-NNN：{状态变更}
- 新登记：R-NNN：{描述}

---

## 7. 度量

| 指标 | 本 Sprint | 里程碑目标 |
|------|---------|----------|
| E2E 用例增量 | +N | 累计 N |
| 单元测试覆盖率 | XX% | 70% |
| Bug 合入数 | N | — |
| Hotfix 次数 | N | < 1 |

---

## 8. 附录：PR 清单（可选）

本 Sprint 合入的 PR 链接列表（从 git 日志自动生成）：
- #NNN: {title}
- …
