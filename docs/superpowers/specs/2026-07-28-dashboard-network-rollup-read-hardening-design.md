# Dashboard 全网预聚合直读与线上保护设计

**日期：** 2026-07-28

**状态：** 已确认方向，待实施计划

**范围：** 首页 `/api/v1/dashboard/summary`、`/api/v1/dashboard/kpi-time-series` 及其监控保护

## 1. 背景与问题

生产环境 `172.24.224.197` 在首页被多个客户端打开后出现 TSDB CPU、磁盘读写和 PostgreSQL 临时文件突增。诊断闭环显示：

- 首页 KPI 时序查询进入 PM Aggregator 的 `network` 即席聚合路径；
- 公式 KPI 会从设备级聚合数据读取依赖 Counter 后现场重算；
- `sum/avg/min/max` 类型的直接 KPI 会进入 `queryDirectRollupKPIs`，重新扫描 15 分钟原始 PM 数据；
- 首页 summary 直接连接 `pm_metric_values`、`pm_measurement_anchors`、`pm_metric_dictionary`，在最近 24 小时原始数据上执行 `DISTINCT ON`；
- 多个客户端和前端固定轮询使同类重查询并发叠加，最终产生长查询、约 95 GB PostgreSQL 临时写入以及明显 CPU、磁盘尖峰。

系统已经存在按制式生成的全网预聚合结果：

- eNB 对应 LTE；
- gNB 对应 NR；
- GSM 对应 GSM；
- 已有小时、天、周粒度。

因此根治方案不是建设新的聚合体系，也不是继续优化原始表扫描，而是让首页直接读取现有全网预聚合结果。

## 2. 目标

1. 首页 KPI 查询完全退出原始 PM 明细扫描和在线全网汇总链路。
2. 首页保持现有产品口径，只开放小时、天粒度；不增加 15 分钟或周粒度入口。
3. 直接复用现有 eNB、gNB、GSM 小时、天、周全网聚合结果，不新增第二套结果表或聚合任务。
4. 为 Dashboard 增加查询超时、并发限制、同请求合并和短缓存，限制异常请求的影响范围。
5. 补齐慢查询、Dashboard 延迟、聚合完整性和 TSDB 资源告警。
6. 在生产规模数据下证明查询成本与“指标数 × 时间桶数”相关，不再与设备数或原始样本量相关。

## 3. 非目标

- 不改变首页 KPI 公式、统计类型或指标编号。
- 不新增 15 分钟首页数据。
- 不在本次改造中重写 eNB、gNB、GSM 聚合生产链路。
- 不新增 Dashboard 专用聚合表。
- 不允许 Dashboard 因预聚合数据缺失而回退扫描 `pm_metrics`、anchor 或 value 明细表。
- 不改变性能管理页面的其他设备、设备组、产品、频段等查询功能。

## 4. 方案选择

### 4.1 否决：继续优化原始表 SQL

可以通过提前过滤 metric ID、增加索引、减少 `DISTINCT ON` 展开来降低单次开销，但成本仍随原始数据量和设备数增长。多个客户端并发后仍可能再次形成资源尖峰，不能根治。

### 4.2 采用：直接读取现有全网预聚合结果

首页按照制式、粒度、指标编号和时间范围读取已经发布的 network 维度结果。请求成本只与返回的 KPI 和桶数量相关，符合首页读模型需求。

### 4.3 否决：新建 Dashboard 专用汇总表

这会复制现有全网聚合能力，形成第二套口径、补数、版本和完整性维护机制，增加长期一致性风险。

## 5. 数据读取设计

### 5.1 唯一数据源

Dashboard 新增专用只读 Repository，唯一读取源为现有全网聚合结果：

- `dimension = 'network'`；
- 制式映射为 `eNB → lte`、`gNB → nr`、`GSM → gsm`；
- 使用现有内置全网任务 ID，不在 Dashboard 中重复硬编码；
- 粒度仅接受 `hourly`、`daily`；
- 指标通过 `metric_path` 精确过滤；
- 时间范围通过 `window_start >= start`、`window_start < end` 过滤。

现有 `weekly` 结果继续保留并由原有功能使用，但本次不暴露到首页。

### 5.2 版本去重

同一内置任务可能因任务版本更新或窗口重算产生多份逻辑结果。Repository 对同一：

```text
technology + granularity + metric_path + window_start
```

只选择 `created_at` 最新的一条。版本选择在 SQL 内完成，禁止把重复行返回到 Service 后再随意覆盖。

全网结果发布与窗口状态更新保持现有事务原子性。Dashboard 只读取已经提交可见的结果，不连接主库任务版本表，不引入跨库查询。

### 5.3 KPI 时序接口

`/api/v1/dashboard/kpi-time-series` 保持当前请求与响应契约：

- 一个请求批量携带多个 KPI 编号；
- 当前期和对比期分别形成有界查询；
- 小时请求只读全网小时结果；
- 天请求只读全网天结果；
- 返回前按 `metric_path, window_start` 排序；
- 缺失桶不填造假零值，沿用空点语义。

以下旧路径不再被首页调用：

- `queryNetworkTable` 的设备级现场全网 `GROUP BY`；
- `queryDirectRollupKPIs` 的 15 分钟原始 KPI 回扫；
- Dashboard 请求内的公式 KPI 即席重算。

公式 KPI 和直接 KPI 都使用现有全网任务已经产出的最终值。

### 5.4 Summary 接口

`/api/v1/dashboard/summary` 的 KPI overview 改为从三种制式最近一个可见的小时窗口读取：

- 查询最近 24 小时的全网小时结果；
- 每个 `technology + metric_path` 选最新窗口；
- KPI 编号在三种制式中全局唯一时直接写入现有 map；
- 如果未来发现同编号跨制式复用，必须先在指标契约层明确合并规则，禁止隐式覆盖。

删除 summary 当前对原始 value、anchor、dictionary 的连接查询。设备数、告警统计、最近告警等非 PM 子查询保持不变。

### 5.5 索引

为 Dashboard 的读取形状增加 network 维度局部索引，候选结构为：

```sql
CREATE INDEX ... ON pm_aggregation_results
    (task_id, granularity, technology, metric_path, window_start DESC, created_at DESC)
WHERE dimension = 'network';
```

最终列顺序以隔离的生产规模副本执行 `EXPLAIN (ANALYZE, BUFFERS)` 后确定。索引验收要求：

- 只访问 `pm_aggregation_results` 和目标索引；
- 不扫描原始 PM 表和兼容视图；
- 不产生临时文件；
- 指标数、时间范围变化时扫描行数近似线性增长。

## 6. 缺失与完整性语义

切换前必须核验首页使用的全部 KPI 在 LTE、NR、GSM 对应小时和天结果中的覆盖情况。

- 结果存在：直接返回。
- 单个 KPI 或窗口缺失：返回空点，同时增加缺失指标并触发告警。
- 聚合窗口标记为不完整：保留现有可用值，但记录不完整指标；API 不伪造完整状态。
- 全部结果缺失：返回可识别的服务错误或空数据契约，并告警。
- 任何缺失场景都禁止在线回退原始明细查询。

如果覆盖核验发现指标未被现有内置全网任务纳入，只修复现有任务的指标选择或映射配置；不在 Dashboard 内补算，也不创建新聚合体系。

## 7. Dashboard 在线保护

### 7.1 超时

- Dashboard PM 查询应用层超时默认 3 秒，可配置；
- 数据库侧设置略短于应用层的 `statement_timeout`，确保请求断开后 SQL 不继续运行；
- 超时错误独立计数，不再仅作为普通 context cancellation 静默处理；
- 超时不触发原始表降级。

### 7.2 并发限制

- Dashboard PM 查询每实例并发上限默认 4，可配置；
- 等待并发名额默认最多 100 ms；
- 超过等待时间且无可用缓存时快速返回 `503 Service Unavailable` 和 `Retry-After`；
- 容量规划按“实例数 × 单实例上限”校验，防止水平扩容后无意放大 TSDB 并发。

### 7.3 同请求合并与缓存

- 对 endpoint、制式、粒度、排序后的 KPI 列表和时间范围生成规范化 key；
- 同一 key 的并发请求只执行一次数据库查询；
- summary 使用 30 秒短缓存；
- KPI 时序使用 60 秒短缓存；
- 新全网窗口发布后主动失效相关缓存；
- 查询失败时只允许返回不超过 5 分钟的最近成功缓存，并通过响应头标记 stale；
- 缓存不是数据源，进程重启后允许自然丢失。

### 7.4 前端请求治理

- 删除 KPI 历史查询当前 60 秒固定轮询；
- 页面重新聚焦或收到新聚合窗口通知时刷新；
- 当前期和对比期分别批量查询，不按 KPI 面板拆成多组重复请求；
- 超时、503 和 429 不进行自动重试风暴；
- 保持已有小时、天选择和响应展示契约。

## 8. 可观测性

### 8.1 应用指标

新增低基数 Prometheus 指标：

```text
dashboard_kpi_query_duration_seconds{endpoint,granularity,status}
dashboard_kpi_query_inflight
dashboard_kpi_query_timeout_total{endpoint}
dashboard_kpi_query_rejected_total{endpoint}
dashboard_kpi_query_cache_total{endpoint,result}
dashboard_kpi_query_coalesced_total{endpoint}
dashboard_kpi_missing_result_total{technology,granularity}
dashboard_kpi_incomplete_window_total{technology,granularity}
pm_network_rollup_lag_seconds{technology,granularity}
```

指标标签不得包含 KPI 编号、SQL 文本、请求 ID或时间范围，避免高基数。

### 8.2 慢查询

复用现有 `SlowQueryTracer`：

- Dashboard Repository 查询超过阈值时记录 query hash、表名、耗时和 request ID；
- Dashboard 超时另记专用 counter，弥补现有 tracer 对 context cancellation 静默的行为；
- TSDB 开启 `track_io_timing`；
- 将 `log_min_duration_statement` 设置为可配置的 1 秒默认值；
- 分阶段启用 `pg_stat_statements`，保留 TimescaleDB 已有 preload 配置并验证重启；
- OTel PostgreSQL receiver 继续作为数据库指标采集入口，不新增 postgres-exporter。

### 8.3 告警

新增或补齐以下默认规则：

| 告警 | 默认条件 | 级别 |
|---|---|---|
| DashboardKPIQuerySlow | P95 > 2s 持续 10m | warning |
| DashboardKPIQueryTimeout | 5m 内出现超时 | critical |
| DashboardKPIQueryRejected | 5m 内出现并发拒绝 | warning |
| DashboardKPIResultMissing | 任一制式/粒度缺失持续 10m | warning |
| PMNetworkRollupLagHigh | 小时结果延迟 > 90m 持续 10m | warning |
| PMNetworkRollupLagCritical | 小时结果延迟 > 120m 持续 5m | critical |
| TSDBTempWriteHigh | 临时写入速率 > 10 MiB/s 持续 5m | warning |
| TSDBTempWriteCritical | 临时写入速率 > 50 MiB/s 持续 5m | critical |

资源层继续使用现有 node-exporter、cAdvisor 和 OTel PostgreSQL 指标，并补充：

- TSDB 容器 CPU 持续超过配额 70%/90%；
- TSDB 容器内存持续超过限制 85%；
- PostgreSQL 活跃查询和连接池占用异常；
- 现有 `HostDiskIOSaturated` 保持为磁盘饱和主告警；
- Grafana 增加 TSDB 容器读写速率、临时写入、Dashboard 并发和延迟关联面板。

绝对磁盘吞吐阈值依赖硬件，不单独作为告警条件；使用磁盘活动率、await、队列深度组成的现有复合告警，避免硬件差异导致误报。

## 9. 测试与验收

### 9.1 自动化测试

- Repository SQL 形状测试：必须过滤 network、task、technology、granularity、metric path 和窗口；
- 版本重叠测试：同一窗口只返回最新结果；
- LTE、NR、GSM 映射测试；
- hourly、daily 路由测试；15min、weekly 在首页接口继续被拒绝；
- summary 每指标最新窗口测试；
- 缺失、不完整和空结果测试；
- timeout、并发拒绝、singleflight、缓存命中和 stale 返回测试；
- 前端假定时器测试：不存在 60 秒固定轮询，不自动重试重型请求；
- API 契约回归测试：字段、时间和空数组语义不变。

### 9.2 性能门禁

在隔离的生产规模副本验证：

- 原始规模至少等价于 10,000 台设备和 3 亿条 value；
- 查询 16 个 KPI 的今天/昨天小时对比；
- 查询相同 KPI 的天粒度范围；
- 数据库暖态 P95 < 200 ms；
- API P95 < 1 s；
- 10 个客户端并发时无原始表访问、无临时文件、无查询堆积；
- 查询计划只访问全网预聚合结果及其索引；
- TSDB CPU 和磁盘增量处于可接受范围，不出现 incident 同类尖峰。

禁止在生产库执行高风险大范围 `EXPLAIN ANALYZE`。

## 10. 上线与回滚

1. 先上线指标、告警和 Dashboard 读路径开关，保持旧路径不被流量调用。
2. 在生产只读核验 LTE、NR、GSM 小时/天结果覆盖和时间新鲜度。
3. 在隔离副本完成数值、执行计划和并发门禁。
4. 小流量启用新 Repository，观察 Dashboard 延迟、缺失率、TSDB CPU、临时写入和磁盘 I/O。
5. 全量切换后删除旧首页原始查询入口，避免未来误用。

回滚仅关闭新版 Dashboard KPI 接口或恢复上一应用版本。不得把实时流量重新切回原始 PM 明细扫描；若全网结果不可用，应展示数据暂不可用并触发告警。

## 11. 成功判定

本改造只有同时满足以下条件才算根治：

1. Dashboard KPI 两个接口的执行路径中不存在原始 PM 明细访问。
2. eNB、gNB、GSM 现有全网小时/天结果成为首页唯一数据源。
3. 首页 16 个 KPI 当前期和对比期数据正确且无粒度变化。
4. 并发刷新不会造成 TSDB CPU、临时写入或磁盘 I/O 尖峰。
5. 查询超时、并发拒绝、慢查询、聚合延迟和资源异常均可观测并可告警。
6. 预聚合结果缺失时系统快速失败或返回受控空数据，不触发原始扫描。
