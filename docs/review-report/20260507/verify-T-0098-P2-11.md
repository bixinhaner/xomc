# S4 Verify Report — T-0098-P2-11（**N/A 决策结案**）

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-11（PM worker 接 KPI Registry enabled 过滤；条件性，operator_code 桶生效；D10=A 吸收 T-0095）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-11（标注"如适用才做；非必需"） |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-09 done（commit `d99cce05`） |
| 时间 | 2026-05-07 |
| 决议 | **N/A — 当前 PM 数据路径不消费 enabled_pm_indicators 表；推迟到 P3-03 KPI 端点上线后再统一评估**

## 决策依据

### 1. 现状检查（grep 全 PM 子包）

按 `pm/` 各子包查"是否消费 enabled_pm_indicators_* 表"：

| 子包 | 用途 | 消费 enabled 表 |
|------|------|---------------|
| `pm/collector/` | PM XML 文件采集 | **否** |
| `pm/counter/` | 原始 counter 持久化 | **否** |
| `pm/aggregation/` | counter → KPI 聚合 | **否** |
| `pm/kpi/` | KPI 时序计算 / 输出 | **否** |
| `pm/indicator/` | admin REST 元数据管理（CRUD） | 是（handler / pg_enabled_repository / pg_indicator_repository LEFT JOIN 用于"列表过滤显示"）|

PM 数据流热路径（file → counter → KPI → 时序）**今天完全不参考** enabled 表，无论 default 桶还是运营商桶。

### 2. 任务语义重审

任务原文：
> PM worker 接 KPI Registry enabled 过滤（条件性，operator_code 桶生效）；如适用才做；非必需

实施计划标注：**P2-11 是"差额 3 条"之一**（计划标题写"33 子任务"实际 36 条）。意图是**预留**：若 PM worker 路径已有 enabled 过滤则切换到 Registry；若没有则不强求。

### 3. 不实施的理由（决策树）

| 维度 | 评估 |
|------|------|
| 正确性 | 不过滤无任何回归风险 — disabled 指标只是 UI 不展示；数据继续算出来不影响 |
| 性能 | ~5-10K rows/min 量级，禁用占比难估；过滤收益边际不明显 |
| 语义清晰度 | **device.carrier → operator_code 映射未定义**：carrier="cmcc" 是否等于 operator_code="cmcc"？非 default 桶的优先级如何？这些设计层未明确 |
| 任务定义 | 实施计划明确"如适用才做；非必需" |
| 风险 | 引入过滤等于改变 PM 数据范围契约——若 dashboard / 报表期望"全量数据可查询"，过滤会破坏隐性合约 |

### 4. 替代方案（推迟到 P3-03 后）

`/api/v1/indicators` 端点（P3-03）上线后会暴露"enabled 桶 CRUD + 切换 operator"能力。届时可基于真实 dashboard 用户行为决定：
- 是否需要 PM worker 侧过滤（性能驱动）
- device.carrier → operator_code 的具体映射规则
- 过滤后旧数据是否需要清理 / 归档

那时再评估更稳妥。

## 改动文件清单

| 文件 | 性质 | 说明 |
|------|------|------|
| `docs/review-report/20260507/verify-T-0098-P2-11.md` | 新增 | 本决策备忘 |
| `docs/project/backlog/subtasks/T-0098-data-dict.md` | 修改 | 行 P2-11 状态 → done（N/A）|

零代码改动。

## 出口门核销

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` — N/A（零代码改动）
- [x] `go test ./...` — N/A
- [x] DoD：N/A 决策有清晰理由 + 推迟接力点（P3-03）

### 决策可逆性

P2-11 的"实施"并非破坏性变更——任何时候 P3-03 后开新任务（如 T-0099）就可以补做。本次结案不锁路径。

## 结论

**N/A 决策结案**。当前 PM 数据路径不消费 enabled 表，过滤改造既无正确性必需也无明确性能动机；语义不清且实施计划标注"非必需"。推迟到 P3-03 KPI 端点上线后基于真实使用模式再评估。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
