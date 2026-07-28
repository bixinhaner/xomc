# PM 测量对象发现查询优化设计

## 背景

KPI 查询页在单设备、`fill_empty=true` 且未显式选择 `object_ldns` 时，会调用
`Aggregator.DiscoverObjectLDNs` 自动发现时间窗内的测量对象。当前实现从
`pm_metrics` 兼容视图执行 `SELECT DISTINCT object_ldn`。该视图会展开指标集并关联
指标值、PM 文件、入库批次和设备维度，导致一个仅需读取测量对象的查询扫描并展开
数百万行，触发 PostgreSQL 并行查询和 Docker 默认 64 MiB `/dev/shm` 耗尽。

## 目标

- 15 分钟粒度直接从 `pm_measurement_anchors` 发现测量对象。
- 小时、日、周、月直接从 `pm_aggregation_results` 发现测量对象。
- 透视分页键和空指标骨架计数复用同一轻量数据源，不再回读指标兼容视图。
- 保持设备、制式、时间窗和可见设备组权限过滤语义不变。
- 保持 `/api/v1/pm/metrics/aggregated` 请求和响应契约不变。
- 保持 KPI 查询导出任务参数、CSV 格式和页面/导出对象集合一致。
- 为时序库容器配置可调的共享内存兜底，默认 512 MiB。

## 方案

`DiscoverObjectLDNs` 保留现有函数签名，内部按粒度路由查询源：

- `15min`：以 `pm_measurement_anchors a` 为主表，通过 `device_dim` 解析设备并使用
  `a.device_dim_id + a.time` 索引过滤。
- `hourly/daily/weekly/monthly`：直接查询 `pm_aggregation_results r`，限定
  `dimension='device'`、目标粒度、设备、制式和窗口。

查询仍返回排序去重后的 `[]string`。请求显式携带 `object_ldns` 时继续跳过自动发现。
页面 handler 和导出 runner 都继续调用同一个公共函数，因此无需新增接口或修改前端。

同一来源投影还用于透视分页的 `(device, object_ldn, granularity, time)` 键。指标数据查询
本身仍只读取用户选择的 Counter/KPI；分页骨架无需展开指标字典、指标值、PM 文件或入库批次。

## 权限与正确性

对象发现必须复用现有设备可见范围约束，不能因为绕开兼容视图而绕过
`VisibleGroups`。设备 SN、OUI 和制式过滤保持与 `applyDeviceFilters` 一致。时间范围
使用左闭右开 `[start_time, end_time)`，与指标查询一致。

## 部署防护

在 `postgres-tsdb` Compose 服务增加：

```yaml
shm_size: ${TSDB_SHM_SIZE:-512m}
```

仓库不跟踪环境专属的 `resources.env`；需要覆盖默认值时可在部署环境设置
`TSDB_SHM_SIZE`。该配置用于防护其他合法并行查询，不替代 SQL 优化。

## 验证

- 单元测试验证 15 分钟 SQL 不再引用 `pm_metrics`、PM 文件或指标表。
- 单元测试验证聚合粒度直接查询 `pm_aggregation_results`。
- 单元测试验证 15 分钟和上卷粒度的透视分页键、骨架计数不再展开指标表。
- 现有“忽略所选指标发现对象全集”测试继续成立。
- 导出 runner 测试验证仍通过同一公共函数获得对象骨架。
- Compose 配置渲染成功。
- 运行 `go test` 覆盖 aggregator、export 和 PM handler，再运行后端全量构建测试。
- 服务器真实页面请求以同一设备、指标和时间窗验证响应时间及返回数据。
