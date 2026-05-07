# S4 Verify Report — T-0098-P1-05

| 字段 | 值 |
|------|-----|
| Task | T-0098-P1-05（KPI 表名对齐文档；D1=A 决策落地，docs only） |
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 1 §6 决策 D1=A |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| 时间 | 2026-05-07 |
| 类型 | docs（无代码 / 无迁移） |

## 改动文件清单

| 文件 | 性质 | 改动 |
|------|------|------|
| `docs/design/参数-KPI-告警-整合设计方案.md` | 修改 | §2.3 顶部加 D1=A 命名对齐表（17 行 footnote） |
| `docs/project/参数-KPI-告警-整合-实施计划.md` | 修改 | §2 Phase 1 任务行 P1-05 描述更新（决策已定，明确为 docs 任务，**不写迁移**） |

## 出口门核销

| 门 | 命令 | 结果 |
|----|------|------|
| 实际表名验证 | `psql ... SELECT to_regclass(...)` × 3 | ✅ `indicator_unit` / `rela_platform_indicator_formula_enb` / `enabled_pm_indicators_enb` 三表全部 exists=true |
| 设计 footnote 与生产对齐 | grep 7 行别名映射 | ✅ 单数 `indicator_unit` / `rela_*_formula_{enb,gsm,gnb}` / `enabled_pm_indicators_{enb,gsm,gnb}` 全部对上 |
| 无 schema 变更 | `ls omcgo/migrations/*060*` | ✅ 无 000060_kpi_rename.sql 创建 |
| go build | 无需 | N/A（docs only） |

## D1=A 决议要点

| 角度 | 论证 |
|------|------|
| **改名影响面** | 17 张 KPI 相关迁移已落地（000035 + 后续）；`internal/pm/indicator/` 15 个 Go 文件 + handler + repo 全部按历史命名引用；重命名链式触发大规模 churn |
| **价值** | 设计文档名（`indicator_units` 复数 / `platform_indicator_formulas_*` / `enabled_indicators_*`）只是命名风格更"齐"；生产历史命名（`indicator_unit` / `rela_*_formula_*` / `enabled_pm_indicators_*`）含语义信息（`rela_` = 关系表前缀，`pm_` = 性能管理域），**信息密度更高** |
| **风险/收益** | 改名风险高（多模块同步变更 + 既有脚本/SQL 适配）、收益小（仅命名美观）；保留现状风险零、收益是设计文档对齐生产 |
| **决议** | **D1=A 保留现状**；设计 §2.3 顶部加命名对齐表说明历史命名 ≠ 设计目标名但语义等价；P1 阶段不再写 `000060_kpi_rename.sql` |

## 不在本任务交付范围

- 任何 KPI schema 变更 — 已决议不做
- KPI Loader 增强（`enabled` 属性 / 多文件 OR 合并 / `operator_code='default'` 行刷新语义）→ Phase 2 P2-09
- KPI handler / REST API → Phase 3 P3-03

## 结论

**S4 出口门通过**（docs-only，仅检验文档与生产一致性）。S5 wave-batched 模式以本 verify-md 为审查证据，可直接进入 S6 commit。
