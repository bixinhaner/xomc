# 设备列表首次进入耗时过大定位与优化方案

> 状态：调查结论 + 优化方案  日期：2026-06-29  作者：GitHub Copilot
>
> **结论先行：本次“第一次点进设备列表很慢”不是单一接口天然慢，也不是前端懒加载本身导致，而是共享环境中 PostgreSQL 在后台时序影子维表同步与设备参数相关负载下进入高压窗口，设备列表首屏请求与 Header 告警统计叠加后，被同一数据库争用拖慢。**

---

## 1. 背景

用户反馈：访问 `http://172.24.224.78:8081/login` 后，第一次进入“设备列表”页面耗时明显偏大。

本次分析限定为：

- 以真实运行环境为准，不在本地假设
- 优先定位根因，不直接修改代码
- 允许通过浏览器、API、SSH、数据库快照与 access log 交叉取证

---

## 2. 最终结论

### 2.1 根因不是“设备列表 SQL 固有 30 秒慢”

设备列表相关接口在空闲窗口下并不慢：

- `GET /api/v1/device-groups/tree` 常见 `0.3s ~ 1s`
- `GET /api/v1/devices?page=1&page_size=20` 常见 `0.8s ~ 1.3s`
- `GET /api/v1/alarms/statistics` 空闲窗口可在 `0.3s ~ 0.7s`

但在高压窗口中，同一时间窗会出现：

- `device-groups/tree` 上升到 `4s ~ 18s`
- `devices` 上升到 `5s ~ 10s`
- `alarms/statistics` 上升到 `10s ~ 30s`，甚至出现 `499` 超时和 `500`

这说明问题是**数据库争用导致的整组接口一起变慢**，不是单条接口固有慢。

### 2.2 主要压力源是 worker 的 tsdb shadow-dim sync

worker 中存在一个固定 60 秒周期的时序影子维表同步任务：

- 启动点：`omcgo/cmd/worker/main.go`
- 周期定义：`omcgo/internal/tsdbsync/runner.go`
- 目的：把主库维度表同步到时序库影子表，供 PM / 告警时序查询本库 JOIN

其中最重的一段是 `cell_band_dim` 派生同步：

- 位置：`omcgo/internal/tsdbsync/runner.go`
- 行为：从 `device_parameters` 按多个参数路径做筛选、`UNION ALL`、按 `(device_id, fap_instance)` 配对后生成 `(device_id, cell_id, band)`

这段逻辑不会去连设备拉参数，但会重扫 `device_parameters`，对主库造成明显压力。

### 2.3 首进设备列表会把问题放大

首次进入设备列表时，前端会叠加发起多类请求：

- Header 常驻请求：`GET /api/v1/alarms/statistics`
- 页面首屏请求：`GET /api/v1/device-groups/tree`
- 页面列表请求：`GET /api/v1/devices?page=1&page_size=20`
- 辅助请求：`GET /api/v1/products`、字典批量加载等

在数据库正被后台同步任务压住的时间窗内，这组请求会同时被拖慢，于是用户体感就是“第一次点进设备列表特别慢”。

---

## 3. 证据链

## 3.1 前端触发路径

- Header 告警统计轮询：`omcmb/webcode/src/components/Layout/Header/index.tsx`
- 轮询 hook：`omcmb/frontend-core/src/hooks/api/useAlarms.ts`
- 设备列表分组树 hook：`omcmb/frontend-core/src/hooks/api/useDevices.ts`
- 设备列表页面：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

已在真实浏览器页面上验证：整页刷新设备列表时，`alarms/statistics`、`device-groups/tree`、`products`、`devices` 会在同一时间窗内一起发出。

## 3.2 后端代码路径

- 设备分组树查询：`omcgo/internal/topology/pg_repository.go`
- 告警统计查询：`omcgo/internal/alarm/pg_store.go`
- tsdb 影子维表同步：`omcgo/internal/tsdbsync/runner.go`

关键说明：

- `alarms/statistics` 自身 SQL 在单独 `EXPLAIN ANALYZE` 下通常只有数百毫秒，不是天然 30 秒慢
- `device-groups/tree` 单独 `EXPLAIN ANALYZE` 也通常只有数百毫秒
- 因此真正问题不是单条 SQL 的静态复杂度，而是运行时资源争用

## 3.3 PostgreSQL 活跃会话与锁快照

线上数据库快照显示：

- `device_parameters` 有 32 个分区
- `devices` 有 4 个分区
- `tsdbsync_cell_band_dim` 活跃查询单条可持有 165 个 relation lock
- `device_group_tree` 活跃查询单条常见持有 96 个 relation lock
- 同时可见 `LWLock / LockManager` 等待

这说明慢点主要来自：

- 分区表访问导致大量 relation lock
- 多类查询在同一时窗内争用 PostgreSQL lock manager / buffer / CPU
- 不是传统意义上的“一个事务锁死另一个事务”

## 3.4 nginx access log 证据

线上 `omcgo-web-1` 的 `/var/log/nginx/app_access.log` 已直接证明：

- 正常窗口中，`device-groups/tree`、`devices`、`alarms/statistics` 都可在 1 秒左右完成
- 慢窗口中，它们会在同一时间段一起上升到 `4s ~ 30s`

典型时间窗：

- `10:52:55 ~ 10:52:57`
  - `device-groups/tree` 约 `5.052s`
  - `devices?page=1&page_size=20` 约 `5.151s`
  - `alarms/statistics` 约 `22.957s`
- `11:16:18 ~ 11:16:24`
  - `device-groups/tree` 约 `4.816s`
  - `devices?page=1&page_size=1&sn=...` 约 `3.080s`
  - `alarms/statistics` 约 `9.768s`
  - 设备详情相关接口同窗也被拖到 `4s ~ 6s`

这条证据是最终定性的关键：**慢点是共享数据库压力窗口，不是单页独有 bug。**

## 3.5 运行态侧证据

调查时采样到：

- `omcgo-postgres-1` CPU 约 `238%`
- `omcgo-postgres-1` 内存约 `5.4GiB / 7GiB`
- web nginx error log 多次出现 `worker_connections are not enough`

其中 nginx 的连接数不足不是本次首要根因，但会放大高峰时的不稳定性，属于次级风险。

---

## 4. 问题拆解

### 4.1 不是“后台每 60 秒同步所有设备参数”

必须明确区分两件事：

1. **设备参数同步**
   - 是按设备触发的业务动作
   - 前端只在设备处于 `syncing` 时以 3 秒轮询状态
   - 不是全局 60 秒后台任务

2. **tsdb shadow-dim sync**
   - 是 worker 固定周期后台任务
   - 本质是“查主库、写时序库影子维表”
   - 最重部分会读取 `device_parameters`

本次性能问题针对的是第 2 类，不是设备参数同步业务本身。

### 4.2 为什么“只是查数据库”也会这么慢

因为这里不是普通小表查询，而是：

- 大分区表访问
- 参数路径过滤 + `UNION ALL` + JOIN 派生
- 多请求同时争用同一库
- 前端常驻轮询与设备页首屏请求叠加

所以准确表述应为：

> 不是数据库天生慢，而是这类查询在共享环境的高并发窗口下会一起变慢。

---

## 5. 优化原则

1. 先止血，再做结构优化
2. 优先处理最重的 `cell_band_dim` 派生同步，不一刀切改所有维表
3. 不能只盯前端，要同时降低数据库热点和首屏叠加效应
4. 优化应尽量做成配置化、可灰度、可回滚

---

## 6. 优化方案

## 6.1 P0：快速止血

### 方案 A：降低 `cell_band_dim` 同步频率

当前 `tsdbsync.DefaultInterval = 60s`。

建议：

- 不直接把整套维表同步全部改成 1 小时
- 优先把 `cell_band_dim` 从主循环中拆出，单独使用更长周期
- 推荐初始周期：`10 ~ 15 分钟`

理由：

- 真正重的是 `cell_band_dim`，不是所有维表
- 直接 1 小时虽然更猛，但会让 PM / 告警查询看到的 band 维度陈旧太久
- `10 ~ 15 分钟` 更适合作为第一轮线上止血值

### 方案 B：把同步周期做成配置项

当前同步周期写死在代码常量中。

建议：

- 增加 `tsdb.shadow_dim_sync_interval` 类配置项
- 支持按环境调整
- 支持线上逐步调参，而非每次改代码发版

收益：

- 降低实验成本
- 便于灰度对比 `60s / 5m / 15m / 1h`

## 6.2 P1：拆分重同步与轻同步

### 方案 C：`cell_band_dim` 独立 runner

当前所有影子维表同步都在同一个 runner 串行执行。

建议：

- 轻量表继续高频同步，例如 `device_dim`、`product_dim`、`device_group_dim`
- `cell_band_dim` 独立 runner、独立 interval、独立日志打点

收益：

- 降低重任务拖累全部维表同步的概率
- 便于单独观察 `cell_band_dim` 的执行时长与资源占用

## 6.3 P2：避免全量重算 `cell_band_dim`

### 方案 D：从全量派生改为增量维护

这是长期最有效的优化方向。

可选路径：

- 仅处理最近发生参数变化的设备
- 在设备参数写入链路中同步维护 band 映射
- 单独落一张 `device_cell_band_cache`/`cell_band_dim_source` 类中间表
ngyong
目标：

- 不再每轮全扫 `device_parameters`
- 将热点从“大表派生”转为“设备级增量更新”

收益：

- 这是根因级修复，不只是降频止血

风险：

- 实施复杂度高于单纯改 interval
- 需要补一致性与回填策略

## 6.4 P3：减小设备列表首屏叠加效应

### 方案 E：设备列表加载期暂缓 Header 告警统计

建议：

- 首次进入设备列表的前 `2 ~ 5 秒`，暂停 `alarms/statistics` 轮询
- 或者设备列表主请求完成后再恢复 Header 轮询

收益：

- 降低首屏阶段的同窗竞争
- 直接改善用户体感

风险：

- 告警角标在极短时间内不是绝对实时
- 但对用户体验的影响远小于页面整体卡顿

### 方案 F：`device-groups/tree` 短 TTL 缓存

建议：

- 后端给全量分组树加 `30 ~ 60 秒` 缓存
- 尤其针对超级管理员的全量树

收益：

- 减少重复访问 `devices` + `device_group_members` 的重计数逻辑
- 首次进入与多标签切换都可受益

风险：

- 分组变更后最多短时陈旧
- 可以通过分组变更后主动失效缓存缓解

### 方案 G：设备列表先出表，再补树

建议：

- 页面先展示列表，再异步补充分组树
- 避免树和列表成为同一首屏关键路径

收益：

- 降低首屏阻塞感
- 有助于把“卡住不动”改成“先可用再补全”

## 6.5 P4：数据库与网关配套优化

### 方案 H：为 `device_parameters` 派生路径做专项优化

建议：

- 复核 `parameter_path` 相关索引是否匹配当前查询模式
- 避免长期依赖多段 `LIKE` 后缀匹配
- 对常用派生字段做预计算或结构化存储

### 方案 I：上线 `pg_stat_statements`

当前排查依赖：

- `pg_stat_activity` 即时快照
- nginx access log
- 手工 `EXPLAIN ANALYZE`

建议上线 `pg_stat_statements`：

- 统计慢 SQL 频率、总耗时、平均耗时
- 长期确认 `cell_band_dim`、`device_group_tree`、`alarms/statistics` 的真实开销分布

### 方案 J：提高 nginx 并发容量

调查期间 web error log 多次出现：

- `10240 worker_connections are not enough`

建议：

- 复核 `worker_processes`、`worker_connections`、`worker_rlimit_nofile`
- 配合宿主机 fd 限制一起调整

说明：

- 这不是本次首要根因
- 但属于应一并修复的运行态隐患

---

## 7. 推荐实施顺序

### 7.1 今日可做

1. 把 `cell_band_dim` 从统一 60 秒循环中拆出来
2. 先将 `cell_band_dim` 同步周期调整到 `10 ~ 15 分钟`
3. 给 `device-groups/tree` 增加 `30 ~ 60 秒` 短缓存

### 7.2 本周内建议完成

1. 把同步 interval 改成配置项
2. 进入设备列表时，延后 Header 告警统计轮询
3. 接入 `pg_stat_statements`
4. 处理 nginx `worker_connections` 容量告警

### 7.3 中期正确方案

1. 将 `cell_band_dim` 改为增量维护，不再每轮全扫 `device_parameters`
2. 审核并优化 `device_parameters` 派生路径的索引 / 存储结构
3. 重新评估设备列表首屏编排，减少关键路径并行争用

---

## 8. 不推荐的方案

### 8.1 不建议直接把整套 shadow-dim sync 一刀切改成 1 小时

原因：

- 真正重的是 `cell_band_dim`，不是所有维表
- `device_dim` / `product_dim` / `device_group_dim` 等轻量镜像没必要一起降到 1 小时
- 会让时序侧维度陈旧过久，影响 PM / 告警侧查询结果一致性

### 8.2 不建议只从前端绕开

例如：

- 只改页面懒加载
- 只把告警轮询关掉

这些只能减轻放大效应，不能解决数据库高压窗口本身。

---

## 9. 验收标准

优化后建议以真实环境验收以下指标：

1. 首次进入设备列表：P95 小于 `3s`
2. `GET /api/v1/device-groups/tree`：P95 小于 `1s`
3. `GET /api/v1/alarms/statistics`：P95 小于 `2s`，不再出现连续 `499/500`
4. PostgreSQL 活跃窗口中，`LockManager` 等待显著下降
5. nginx 不再出现频繁 `worker_connections are not enough`

额外建议：

- 灰度比较 `60s / 5m / 15m` 下的 access log 与数据库 CPU
- 对照业务可接受的时序维度陈旧时长，决定最终 interval

---

## 10. 一句话对外口径

> 本次“第一次点设备列表慢”不是设备列表页面自身 bug，而是共享环境中 PostgreSQL 在后台时序影子维表同步与参数相关负载下进入高压窗口，设备列表首屏请求与 Header 告警统计叠加后，被同一数据库争用拖慢；优化重点应放在拆分/降频 `cell_band_dim` 同步、缓存 `device-groups/tree`、降低首屏请求叠加。 
