# docs/archive/ — 历史归档区（只读）

> 本目录是**历史快照归档**，仅供溯源，**不代表当前架构/现状**。重建文档体系（2026-06-09）时从 `docs/` 各处下沉至此，与活文档物理隔离。
> 阅读现状请走 [`docs/README.md`](../README.md)（文档总索引）。

| 子目录 | 内容 | 为何归档 |
|--------|------|---------|
| `analysis/` | 早期阶段性分析、Sprint/Phase 完成度快照、数据流向/容量分析、20260323 会话诊断 | point-in-time 分析，已被 backlog/sprint 体系与现行设计取代 |
| `reports/` | 2026-03 九专家审查（×3）、专家角色提案、实现完整度报告、CPE/合规一次性报告、RBAC 设计、设备管理调研（×3）、菜单表设计 SQL、基站上线数据流程图 | 一次性评审/调研/验证产物，结论已落地 |
| `legacy-handoff/` | 接手时的原始交付/迁移参考资产（UED 页面规格、早期后端模块设计、前端 GIS 选型与模板） | 从遗留 JSP/Java OMC 反推的迁移参考，非当前实现 |
| `skins/` | 设计系统灵感参考（Apple/Stripe/Vercel/Linear 等），原 `docs/claude/` | 与 OMC 业务无关的视觉参考，原路径名易误解为 Claude 配置 |

**注**：每提交代码评审/验证快照（`docs/review-report/**`）**未迁入此处**——它是 append-only 审计日志，被 backlog 约 65 处 closing-evidence 引用，故原地保留只读。已从仓库删除的 9MB 遗留 MySQL/KPI SQL（旧 `docs/files/db/`、`docs/files/Back-end/seed-data/`）可经 git 历史恢复（指标已由 `omcgo/data/indicator-library/` XML 驱动取代）。
