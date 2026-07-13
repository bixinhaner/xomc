# 首页 KPI 时序接口粒度对齐方案

> 文档日期：2026-07-13
> 文档状态：已实现，待验收
> 代码基线：`ce479a0ff`
> 对应 Issue：首页 KPI 时序接口支持性能仪表板一致的粒度查询
> 本文只给出方案，不包含业务代码修改。

## 1. 目标与边界

扩展现有 `GET /api/v1/dashboard/kpi-time-series`，让首页可以显式读取与性能仪表板一致的 hourly/daily 聚合结果。

本 Issue 只提供公共取数能力，不改首页“天/周”UI。后续周视图和天视图分别通过独立 Issue 接入。

明确不改：

- KPI 指标公式、指标库元数据。
- PM hourly/daily 聚合任务和数据表。
- 性能仪表板接口及页面。
- 数据库结构和迁移。
- v2/v3 页面。

## 2. 当前问题

首页接口当前固定读取 `network + hourly`，并在最近完整小时缺点时使用 15min 尾部补点。即使调用方传入七天时间范围，也只能得到小时序列。

性能仪表板选择日粒度时读取 `network + daily`：先按自然日汇总基础 counter，再按指标公式重算 KPI。对速率、成功率、利用率等分式指标，daily 重算不等于 hourly KPI 的算术平均，因此首页无法仅靠前端聚合得到相同日值。

## 3. 接口设计

### 3.1 请求参数

在现有请求中增加可选参数：

```text
granularity=hourly | daily
```

兼容规则：

- 未传时默认 `hourly`，现有调用行为不变。
- 显式传 `hourly` 与未传等价。
- 传 `daily` 时读取 PM Aggregator 的 daily network 结果。
- 其他值返回 HTTP 400，并明确提示允许值。

响应结构保持不变：

```json
{
  "KGNB0516": [
    { "time": "2026-07-12T00:00:00+08:00", "value": 3.23 }
  ]
}
```

### 3.2 时间窗口

接口继续接受 RFC3339 `start_time/end_time`，采用半开窗口：

```text
[start_time, end_time)
```

调用方负责使用系统业务时区生成自然日边界。后端不得把 `23:59:59.999` 当作自然日结束边界。

### 3.3 粒度路由

```text
hourly
  -> DimensionNetwork + GranularityHourly
  -> pm_metrics_hourly
  -> counter 聚合后重算 KPI
  -> 只返回 Aggregator 已生成的完整 hourly 桶，不使用 15min 尾部补点

daily
  -> DimensionNetwork + GranularityDaily
  -> pm_metrics_daily
  -> daily counter 聚合后重算 KPI
  -> 禁止 15min 尾部补点
```

两种粒度都复用 PM Aggregator，不在 dashboard service 中实现第二套 KPI 公式。

## 4. 实现设计

### 4.1 Handler

解析 `granularity` 并转换为受限枚举。默认值在 Handler 层确定，Service 不接收任意字符串。

校验顺序：

1. 校验 `kpi_names`。
2. 校验 `start_time/end_time`。
3. 校验 `end_time > start_time`。
4. 校验 `granularity`。

### 4.2 Service

将当前固定 hourly 的查询入口扩展为显式粒度参数：

- hourly 使用桶起点，只返回 Aggregator 已生成的完整桶；缺失桶保持缺失，不进入 15min 尾部补点逻辑。
- daily 直接构造 daily network 查询，返回 daily 桶，不进入 hourly 尾部逻辑。
- 公共排序、按指标分组和去重逻辑可以复用。

### 4.3 查询构造

在 dashboard 的 PM 查询适配层增加 daily request builder。它与 hourly builder 使用相同：

- `DimensionNetwork`
- KPI code 列表
- 时间窗口
- KPI counter 依赖重算

唯一差异是 `GranularityDaily`。

### 4.4 前端共享 API

`frontend-core` 的 KPI 时序参数增加可选粒度类型：

```text
type DashboardKPIGranularity = 'hourly' | 'daily'
```

API client 仅在调用方传值时发送 `granularity`。现有 hooks 不传值时继续走默认 hourly。

## 5. 影响范围

预计修改：

- `omcgo/internal/dashboard/handler.go`
- `omcgo/internal/dashboard/service.go`
- `omcgo/internal/dashboard/kpi_network_query.go`
- `omcgo/internal/dashboard/handler_test.go`
- `omcgo/internal/dashboard/kpi_network_query_test.go`
- dashboard service 对应测试文件
- `omcmb/frontend-core/src/services/api/dashboardApi.ts`
- `omcmb/frontend-core/src/types/dashboard.ts`

不修改首页页面组件。

## 6. 测试方案

### 6.1 后端

- 未传粒度时构造 hourly network 查询。
- `granularity=hourly` 与默认行为一致。
- `granularity=daily` 构造 daily network 查询。
- 非法粒度返回 400。
- `end_time <= start_time` 返回 400。
- daily 路径不调用 15min 尾部补点。
- hourly 路径同样不调用 15min 尾部补点，最近完整桶尚未生成时返回缺点。
- daily 返回的时间使用自然日桶时间，不转换为小时桶结束时间。

### 6.2 口径一致性

以 `KGNB0516` 为代表，构造各小时业务时长不同的数据：

- daily 返回值等于全天 counter 汇总后按公式重算的值。
- daily 返回值不等于 hourly KPI 简单平均时，必须保留 daily 重算结果。
- 相同指标、制式、network 范围和自然日下，dashboard daily 与性能仪表板 daily 一致。

### 6.3 回归

- 现有首页 hourly 调用保持响应结构兼容；点位统一为桶起点，且不再混入 15min 尾部替代值。
- 前端现有不传粒度的调用参数不变。
- 后端执行 `go build ./...` 和 dashboard/PM 相关测试。
- 前端执行 typecheck。

## 7. 验收标准

1. 接口同时支持 hourly 和 daily，默认 hourly。
2. daily 与性能仪表板复用同一 PM Aggregator 计算链路。
3. daily 日值与性能仪表板在相同统计范围下严格一致。
4. daily 不混入 hourly 或 15min 补点。
5. 现有首页调用保持接口兼容；hourly 点位时间和完整桶策略以本文最终契约为准。
