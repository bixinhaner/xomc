# Sprint-09 规划 & 回顾

> 创建于 2026-04-30；S1 触发（T-0027 拓扑自动分组规则引擎激活）。

---

**Sprint ID**：09
**窗口**：2026-05-01 ~ 2026-05-14（2 周）
**主题**：拓扑自动分组规则引擎激活，关闭 R-104（P1）
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`

---

## 1. Sprint 目标（Sprint Goal）

把现有 80% 已建的 rule_engine 接通最后 4 处断点（getAllDevices 实现 + EventBus 订阅 device.inform.bootstrap + cron @hourly 重评 + device_group_members source_type 区分），让规则一次配置 → 全网设备自动归位。

完成后："OMC 在 100 万设备规模下，分组管理工作量从 30% 工时降到 < 5%。"

---

## 2. 承诺项（Committed）

| # | 工作项 | Owner | 估算 | 关联 | 优先级 |
|---|-------|-------|------|------|--------|
| 1 | T-0027 拓扑自动分组规则引擎激活（4 接线断点 + migration 000051 + EventBus 订阅 + cron） | Claude | M（1-3 人日） | `prd/F06-topology-auto-grouping.md` / R-104 | P1 |

**容量**：单人 2 周可用 10 人日 × 80% buffer = 8 人日有效容量。Committed 1 项 M（≤ 3 人日）+ stretch 留 5 人日给 T-0090a / T-0097 reproduce 后续 / 合理插入项。

---

## 3. Stretch Goals（如有余力）

| # | 工作项 | Owner | 估算 |
|---|-------|-------|------|
| 1 | T-0090a — MML 控制台公私命令新增页面 FE-only 子集（① 操作类型差异化 + ③ textarea + ④ 自定义限制 + ② UI section 删除） | Claude | S |
| 2 | T-0097 reproduce 后处理（reject 或 fix） | Claude | S |
| 3 | T-0035 ID 冲突修正（§8 Rejected vs §4 Triaged 同号 — 类比 T-0027 修法） | Claude | S |

---

## 4. 依赖与阻塞

| 依赖项 | 阻塞什么 | 预计解除 | Owner |
|-------|---------|---------|-------|
| 无 | — | — | — |

T-0027 deps：原 T-0094 已 rejected（misdiagnosis），T-0090 链断开，无任何外部阻塞。完全独立可推。

---

## 5. 每日进展（Daily Standup）

**2026-04-30 周 X**
- 完成：S0 PRD 起草（commit `689207ad`）+ S1 排期（本文件）+ S2 设计备忘补完（PRD §12）
- 进行中：S3 接线（待 user 确认进入）
- 阻塞：无

---

## 6. 回顾（Retro，Sprint 最后一天填写）

待 2026-05-14 填写。

---

## 7. 度量

| 指标 | 本 Sprint 目标 | 里程碑目标 |
|------|---------|----------|
| E2E 用例增量 | +5（A1-A5 GWT 各 1 条 e2e claim） | 累计 ≥ 549（已超目标） |
| 单元测试覆盖率 | rule_service / matcher 新代码 ≥ 80% | 70% |
| Bug 合入数 | 0（理想） | — |
| Hotfix 次数 | 0 | < 1 |
| **R-104 关闭** | ✅ | P0/P1 风险 ≤ 5 |

---

## 8. 附录：PR 清单（自动生成 — Sprint 结束补）

待 Sprint 结束按 git 日志生成。
