# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-13 22:34 |
| 提交 | 31bc3708 (基线) |
| 作者 | shangyingbin |
| 范围 | fullstack-dashboard-pm (dashboard / pm·aggregator / pm·adhoc / migration / 前端 dashboard·components·i18n) |
| 变更文件数 | 16（含收口修复） |
| 新增行数 | +593（含收口 +222 子集） |
| 删除行数 | -107 |

## 变更概要

首页 KPI 配置放开全部指标（#328）：后端把内置全网聚合任务放开到全库（清空指标列表＝全聚），并在全聚时连派生 KPI 一起重算落库（收口修复）；首页取数改读全网预聚合结果表（按编号 + 老别名映射兼容）；前端把「指标查询」页表格弹窗抽成共享件供配置页选全库指标，面板名字/单位改从指标库元数据取（自动翻页取整库，收口修复）。

## 审查发现

### 🔴 CRITICAL (严重)
无

### 🟡 WARNING (警告)
无

### 🔵 INFO (建议)
- 全库每小时全网聚合 + 全库 KPI 重算的耗时随指标规模上升，设计已标「实测一轮、过慢则后续优化」，非本次阻塞。
- useAllIndicators 翻页防御上限 100 页（远超任何单制式库规模），库规模异常增长时需关注。

## 详细分析

- `internal/dashboard/service.go`：GetKPITimeSeries / GetKPITrendComparison 改读全网预聚合表，统一走 buildNetworkKPISeriesQuery（Squirrel 参数化）；错误均 `fmt.Errorf("...: %w")` 包装；none 项返回空序列不报错。✓
- `internal/dashboard/kpi_network_query.go`：纯函数构 SQL，三条 network 任务 ID 固定常量，编号唯一定位制式，注释充分。✓
- `internal/pm/aggregator/{query,kpi_recompute}.go`（收口）：新增 RecomputeAllKPIs 开关 + queryFullLibraryWithKPIs（拉全 counter + 动态枚举全库派生 KPI + 复用 recomputeKPIs），跨制式靠编号全局唯一+缺 deps 跳过；查询失败降级不丢 counter。✓
- `internal/pm/adhoc/executor.go`（收口）：仅 network 维度 + 空指标列表时置 RecomputeAllKPIs，scope 收窄。✓
- `migrations/seed/000004_*.sql`：goose Up/Down 配对、幂等 UPDATE、Down 精确还原精选子集、只动 network 三条、版本号连续。✓
- 前端 useIndicatorsLibrary.ts（收口）：useAllIndicators 循环翻页（1000/页）取整库，复用 api.list、不动后端分页上限。✓
- 前端 useMetricMetadata.ts / LayoutKPIPanel.tsx / KpiConfig/index.tsx：元数据从指标库取（中文名+单位），共享 MetricPickerModal 接入，今日/昨日对比保留，强类型无 any，@core 跨包引用，文本走 react-intl。✓

## 业务完整性检查
业务链路完整：后端取数→聚合→落库闭环；前端选择器→面板渲染闭环；迁移 up/down 配对；新增能力均有单测（aggregator 重算 2 条、dashboard 网络查询、useMetricMetadata、MetricPickerModal）。

## 业务影响范围检查
变更范围可控。QueryRequest 新增字段（向后兼容，默认 false 不改既有行为）；全聚 network 任务行为增强（多产 KPI 行，counter 行不变）；指标库列表接口零改动（仅前端多翻一页）。未发现跨模块破坏。

## 前后端一致性检查
后端 kpi-time-series 接口契约未变（仍按编号/别名查、按 key 回填）；前端按编号传、元数据从指标库取，与后端编号化存储一致。端到端实测三段对齐通过。

## 代码质量回退检查
未发现代码质量回退：无删测试、无移除错误处理、无降级安全、无引入 any/interface{}、无硬编码替代配置、无 ORM 替代 Squirrel。

## 配套更新提醒
- **文档**：无需更新（设计/计划在 notes，不入 git；Issue #328 已引用）。
- **单元测试**：已覆盖（后端重算成功+降级路径、网络查询；前端元数据解析）。
- **端到端测试**：已人工三段对齐验（DB/接口/DOM）；e2e_verify.sh 未加用例（首页 KPI 接口需登录态，既有脚本不覆盖登录链路），非回归。

## 安全检查
未发现安全问题（不触 auth/middleware；无敏感数据日志；SQL 全参数化）。

## 性能检查
全库全网聚合 + KPI 重算耗时随规模上升，已标后续优化项；首页元数据多翻 1 页（1408→2 页）开销可忽略。

## 测试覆盖
后端 go build/test 全过（含新增 4 测）；前端 webcode typecheck=0、useMetricMetadata 测试 4 通过。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 2 |

**审查结论**: `PASS`
