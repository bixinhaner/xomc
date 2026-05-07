# Backlog Bootstrap 遗留事项

> 从 `docs/project/backlog.md` §12 拆出（2026-05-07，纯搬运）。
> 初始化阶段（2026-04-20）的待办清单；多数已闭环，剩余待 owner 推进。

## 12. 待办（Bootstrap 遗留）

初版仅把 P0/P1 主干登入，以下为后续 7 天内完善事项：

- [ ] 为 T-0007 / T-0010 / T-0013（P0 且复杂）建 `docs/project/backlog/T-NNNN-*.md` 详情文件
- [x] ~~补齐所有 Active 条目的 `Owner` 字段~~ — 2026-04-20 已填**角色级**占位（电信/Go/PM/架构/运维/前端/QA），Sprint Planning 时由 PgM 替换为具体人名
- [ ] 所有 `待产出` PRD（infra-event-bus / F08-oss-protocol）启动 S0
- [ ] `sprint-01.md` 从 `TEMPLATE.md` 拉一版初稿，引用本表 T-0005~T-0009
- [ ] 每周一 09:30 Triage 例会时段确认
- [ ] `/dev-pipeline status` skill 子命令实现（自动读本表输出仪表盘）
- [ ] 累计型上游 Task（当前仅 T-0006）每 Sprint 回顾更新 `Notes: Progress: 累计 NN`（供下游 S4 核销用，规则见设计 §11.7.1）

---

**当前版本**：v1.2（2026-04-20 Bootstrap Day 1-3；补 Owner 角色占位 + 累计型依赖豁免 + T-0005 S7 回写关闭）
**维护节奏**：每日（状态回写）· 每周一（Triage）· 每 2 周（Sprint Planning + 仪表盘刷新）· 每季度（深度清理）

---

← 返回 [`docs/project/backlog.md`](../../backlog.md)
