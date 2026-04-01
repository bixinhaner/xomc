# Code Review Report

| Field | Value |
|-------|-------|
| Date | 2026-04-01 |
| Scope | pm (docs) |
| Type | docs |
| Conclusion | **PASS** |

## Changed Files

| File | Changes |
|------|---------|
| `files/Back-end/组数据汇总与查询.md` | +531 lines (new) |
| `files/Back-end/设备数据查询.md` | +604 lines (new) |

## Review Summary

纯文档变更，新增两篇 PM/KPI 模块后端功能说明文档。

### 组数据汇总与查询

- 完整描述了组维度 KPI 数据的定时聚合流程（15min/小时/天/周/月五种粒度）
- 覆盖 ENB/GSM 和 GNB 两种设备类型的差异处理
- 详细说明了 MongoDB 聚合管道（$match → $group → $sort）的两层聚合算法
- 包含查询接口定义和 group_id 过滤逻辑

### 设备数据查询

- 完整描述了设备维度 KPI 数据的表格查询和图表查询两种视图
- 覆盖 ENB/GSM（RPC 转发）和 GNB（本地处理）两种处理路径
- 详细说明了分表规则、时间过滤、混合查询等复杂逻辑
- 包含 uniqueId 计算规则和时间缺口填充逻辑

## Findings

无 CRITICAL 或 WARNING 级别问题。

| Level | Count |
|-------|-------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 0 |
