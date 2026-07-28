# PM 上卷设备查询优化设计

## 背景与现象

设备 `120200024118AA01241` 的 15 分钟查询稳定在 0.18～0.30 秒，而小时查询稳定在
2 秒以上，已有 KPI 在服务器负载下达到 7.4 秒。

小时结果已经预计算并写入 `pm_aggregation_results`，问题不在聚合计算或队列。当前设备
维度小时查询仍读取兼容视图 `pm_metrics_hourly`。该视图先对所有设备结果执行
`DISTINCT ON (dimension_key, metric_id, window_start, object_ldn)`，设备编号、指标和时间
条件位于视图外层，不能在去重前下推。

线上执行计划显示：

- 兼容视图扫描并去重 440,000 行，最终返回目标设备 4 行，SQL 约 5.56 秒。
- 按目标设备 UUID 直接读取 `pm_aggregation_results`，现有
  `(dimension, dimension_key, window_start DESC)` 索引只检查 44 行，SQL 约 0.46 毫秒。

## 目标

- 设备维度的小时、日、周、月查询直接读取 `pm_aggregation_results`。
- 设备、指标、粒度、时间和测量对象过滤必须在 `DISTINCT ON` 前下推。
- 页面、导出、权限、透视分页、补空和返回字段契约保持不变。
- 不修改 15 分钟查询、聚合写入、NATS、任务定义和其他聚合维度。

## 方案比较

### 方案 A：继续使用兼容视图并增加索引

不采用。底表已有维度索引，但视图的全局 `DISTINCT ON` 是优化屏障。增加
`device_sn` 索引不能保证外层条件进入视图内部，也会增加 44 万行及后续数据的写放大。

### 方案 B：新增设备小时物化查询表

不采用。它可以获得稳定查询性能，但会重复保存现有上卷结果，增加磁盘、写入、保留策略
和版本一致性成本。

### 方案 C：直接查询统一上卷结果表

采用。根据设备 OUI/SN/制式从 TimescaleDB `device_dim` 解析设备 UUID，以 UUID 字符串
过滤 `pm_aggregation_results.dimension_key`，在底表过滤后执行与兼容视图相同的最新结果
去重。该方案复用已有索引，不增加表和数据。

## 查询架构

### 数据源路由

- `15min × device`：保留现有稀疏锚点和值表读取路径。
- `hourly/daily/weekly/monthly × device`：直接读取 `pm_aggregation_results r`。
- `device_group/product/band/network`：保持当前实现，不在本次修改范围内。

新增一个只负责上卷设备结果的查询构造单元，投影现有 `deviceTableColumns` 契约：

- `device_oui`、`device_sn`
- `metric_path`、`metric_type`、`metric_value`
- `aggregation_op AS statis_type`
- `granularity`
- `window_start AS time/start_time`
- `window_end AS end_time`
- `created_at AS ingest_time`
- `object_ldn`
- 与兼容视图一致的 `extra`

### 设备键解析

请求中的 OUI、SN 和制式先在同一 TimescaleDB 的 `device_dim` 中解析为设备 ID。上卷设备
任务的 `dimension_key` 使用设备 ID 字符串，因此查询条件为：

```sql
r.dimension = 'device'
AND r.granularity = $granularity
AND r.dimension_key IN (
  SELECT id::text
  FROM device_dim
  WHERE ...
)
AND r.metric_id = ANY($metric_ids)
AND r.window_start >= $start_time
AND r.window_start < $end_time
AND ($object_ldns IS NULL OR r.object_ldn = ANY($object_ldns))
```

设备权限继续使用现有可见分组生成的 SN 过滤。设备 ID 预过滤是等价的性能谓词，不能替代
权限过滤，也不能让无权限设备进入结果。

### 去重和任务版本

必须保留兼容视图的业务语义：

```sql
DISTINCT ON (dimension_key, metric_id, window_start, object_ldn)
ORDER BY dimension_key, metric_id, window_start, object_ldn, created_at DESC
```

区别只是设备、粒度、指标、时间、对象和权限条件先作用于底表，再对命中的小集合去重。
这样仍然能在任务版本变更、补报或重复写入时返回最新结果。

### 透视分页和补空

对象发现、透视键和骨架计数继续复用已经上线的 `newDeviceObjectSource`。同时将上卷粒度的
设备过滤转换为 `dimension_key` 预过滤，避免对象发现按 `device_sn` 并行扫描全部结果。

`page_by=pivot_row`、`count_mode=n_plus_one`、`fill_empty=true` 的调用顺序及返回口径不变：

1. 发现对象；
2. 获取当前页透视键；
3. 读取当前设备的真实指标；
4. 补齐已存在时间桶中缺少的所选指标。

## 影响范围

直接受益：

- KPI 查询页面的设备小时、日、周、月数据；
- 使用相同查询服务的页面导出和后台导出；
- 空指标和已有指标的上卷粒度查询。

不受影响：

- 前端代码和 API 请求/响应格式；
- 15 分钟 PM 入库和查询；
- 逐级聚合计算与任务版本生成；
- NATS 流、Redis 和数据保留策略；
- 全网、产品、频段、设备组查询。

优化不会补算历史缺失指标。历史版本没有生成的指标仍为空，新窗口按当前任务版本正常生成。

## 错误处理

- 设备在 `device_dim` 中不存在时返回空结果，不回退到全表扫描。
- 数据库错误继续包装查询上下文并返回现有 500 错误契约。
- 不支持的粒度和维度继续由现有 `SelectTable` 校验。
- 不新增静默降级到兼容视图的路径，避免性能问题重新出现。

## 测试与验收

单元测试必须证明：

- 小时、日、周、月设备查询 SQL 读取 `pm_aggregation_results`，不读取
  `pm_metrics_hourly/daily/weekly/monthly`。
- 设备 ID、粒度、指标、时间、对象过滤位于底表 `DISTINCT ON` 内部。
- 最新 `created_at` 去重、权限、分页和返回投影保持一致。
- 15 分钟路径继续读取稀疏原始表。
- 设备组、产品、频段、全网路由不变。

统一验证：

- `cd omcgo && go build ./... && go test ./...`
- `cd omcmb && npm run typecheck`
- 部署交付包并执行 25 项健康检查。
- 同一设备、指标和时间窗对比 15 分钟与小时接口。
- 对小时查询执行 `EXPLAIN (ANALYZE, BUFFERS)`，确认按
  `dimension_key` 命中索引，不再扫描并去重全体设备结果。
- 核验页面与导出返回行数、分页、补空和字段结构不变。
