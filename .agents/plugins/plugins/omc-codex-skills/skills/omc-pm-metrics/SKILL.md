---
name: omc-pm-metrics
description: 当处理 OMC 性能管理 PM、性能仪表盘、设备性能查看、KPI/counter、pm_metrics、pm_adhoc_aggregation_results、指标查询、导出、statis_type、pct/avg/sum/max/min、15 分钟点、缺点或 hourly/daily/weekly/monthly 聚合口径问题时使用。用于触发 PM 知识库、入口矩阵、页面验证和真实数据验证规则。
---

# OMC PM Metrics

## 必读知识库

处理 PM 相关任务时，先阅读仓库文档：

- `docs/ref/pm-metrics-knowledge.md`

## 执行要求

- 先确认统计口径，再看代码入口。
- P4.5/P7 必须列出页面查询、定时聚合、自定义聚合、导出、启用指标旁路是否同源。
- 涉及页面行为时，按知识库要求扫描真实页面结构和真实请求参数。
- 涉及真实数据时，从源表重新计算期望值，记录样本数和接口返回值。
- 涉及 PM 保存、导出、聚合口径时，把知识库对应检查点写入账本、验证计划和审查 brief。
