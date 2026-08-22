# OMC 仪表板与后端架构分析与优化方案

**日期：** 2026-07-06  
**主题：** 全局仪表板（Dashboard）、GIS 地图告警数据一致性、时序数据查询性能及系统启动流程的架构瓶颈分析与优化方案。

---

## 1. 数据高并发一致性瓶颈：告警反范式设计 (Denormalization)

### 1.1 具体问题与现象
在查询 GIS 地图设备列表时，如果发现“当前告警”状态并未准确展现，原因往往是数据层面的脱节。
为了保证 GIS 地图在展示数万个基站节点时的高性能，系统放弃了在地图接口里实时进行 `COUNT(*)` 扫描 `alarms_active` 表，而是采用了**反范式设计**，在 `device_info` 详情表中维护了一个额外的冗余字段：`active_alarm_count`。
*   **现象**：绕过 Go 业务层代码（例如执行测试模拟数据直插数据库）时，只写了 `alarms_active` 没更新 `device_info`，导致地图前端气泡显示告警为 0，而详情页却有告警。
*   **并发隐患**：正常情况下 Go 应用依靠 DB 事务进行 `UPDATE device_info SET active_alarm_count = active_alarm_count + 1` 操作。在极端“网络风暴”场景下，数千条事件并发写入会导致严重的行锁（Row-level lock）竞争与请求排队。

### 1.2 优化改动细节方案
**方案A：轻量级后台对账（Reconciler Job）**
*   **逻辑**：新增一个异步定时 Worker，每天业务低谷期利用数据库真实的 `alarms_active` 表，通过 `GROUP BY device_id` 对 `device_info` 中的告警计数字段做一次对账与强刷，达到“最终一致性”。
*   **改动点**：在 Go 后端新增定时任务调度模块。不改变现有主链路的写入性能。

**方案B：转移热点至 Redis (推荐)**
*   **逻辑**：彻底将动态变化的“活动告警数”移出 PostgreSQL 的热点更新行列。将每个设备的告警数写入 Redis 的 Hash 或单个 Key 中。
*   **改动点**：
    1.  基站告警发生时，事务仅写 PG 的 `alarms_active`，随后发送消息。
    2.  Redis 服务监听后执行原子指令 `HINCRBY device_alarms <device_id> 1` 或 `-1`。
    3.  GIS 地图聚合查询时，先从 PG 中拉取所有设备的拓扑信息，紧接着用管道（Pipeline）或批量指令 `MGET / HMGET` 从 Redis 拿到对应的告警数值进行融合再下发。

---

## 2. 边缘缝补逻辑繁重：未完全释放 TimescaleDB 新特性

### 2.1 具体问题与现象深度剖析
前端首页的 KPI 时间折线图不仅会查询历史聚合结果，还能显示最后一刻（例如 `12:15`）的准实时数据。

*   **当前实现（繁重的手动打补丁逻辑）**：
    为了解决大盘按小时聚合 `pm_adhoc_aggregation_results` 带来的延迟空窗（例如 `11:00-12:15` 的缺口），Go 代码在 `service.go` 的 `fetchNetworkKCodeSeries` 方法中硬编码了极度复杂的“三段式”缝合策略：
    1.  **主查询**：先查询小时级的预聚合表。
    2.  **缺口计算**：在代码内存中遍历所有指标（代码中需应对 LTE/NR/GSM 等不同制式聚合延迟不一致的情况，维护 `latestByCode` 寻找全局最小的“尾部查询起点” `trailingStart`）。
    3.  **明细回退补点**：通过 `buildRawNetworkKPISeriesQuery` 拼装动态 SQL，再次穿透到 `pm_metrics` 这个底层超大明细表（15分钟级），强制执行全网数据的 `GROUP BY metric_path, time` 和 `CASE MIN(statis_type)` 聚合计算。
    4.  **数组拼接**：最后在 Go 内存中把查出的 15 分钟点与之前的 1 小时点做数组 `append` 及时间窗口覆盖过滤。
*   **核心隐患**：
    *   **性能消耗转移**：虽然大头时间走了预聚合表，但每次刷新页面，仍至少触发一次面对百万级 `pm_metrics` 裸表的全网 `SUM/AVG`，属于在联机分析处理（OLAP）场景硬查联机事务处理（OLTP）底表。
    *   **代码坏味道**：Go 层查询路由逻辑长达百余行，耦合了太多本该属于数据库引擎的计算逻辑，极易在跨制式指标合并时产生双重计数的 Bug。
    *   **技术栈闲置**：我们选型使用了专业的时序数据库 TimescaleDB，但在 `migrations/tsdb` 的初始化 DDL 中，目前只是把它当普通 PostgreSQL 用。

### 2.2 优化改动细节方案与实施路径
利用 TimescaleDB 自带的**实时连续聚合（Continuous Aggregates with Real-Time logic）**功能，将合并负担下移给数据库引擎。

*   **1. 数据库层改造（创建 Caggs 视图）**：
    在 `omcgo/migrations/tsdb/` 新增一个迁移文件，基于原始 `pm_metrics` 建立小时级的连续聚合视图。
    ```sql
    -- 创建连续聚合视图，按小时进行通用聚合（SUM/AVG等）
    CREATE MATERIALIZED VIEW pm_metrics_hourly_cagg
    WITH (timescaledb.continuous) AS
    SELECT 
        time_bucket('1 hour', time) AS bucket_time,
        metric_path,
        statis_type,
        SUM(metric_value) as sum_val,
        AVG(metric_value) as avg_val,
        MAX(metric_value) as max_val,
        MIN(metric_value) as min_val
    FROM pm_metrics
    GROUP BY bucket_time, metric_path, statis_type;
    
    -- 设定自动刷新策略（比如每15分钟刷新一次过去2小时的数据）
    SELECT add_continuous_aggregate_policy('pm_metrics_hourly_cagg',
        start_offset => INTERVAL '2 hours',
        end_offset => INTERVAL '15 minutes',
        schedule_interval => INTERVAL '15 minutes');
    ```

*   **2. 实时合并引擎机制（Real-Time Aggregation）**：
    TimescaleDB 默认开启实时聚合(`timescaledb.materialized_only=false`)。当我们读取 `pm_metrics_hourly_cagg` 视图时：
    * 对于已固化的历史数据（如 `11:00` 之前），它直接极速读取已物化的结果（毫秒级）。
    * 对于 `11:00-12:15` 的缺口数据，TSDB 引擎会在幕后自动读取原始表 `pm_metrics`，并将其透明地叠加到底层返回结果中。无需应用层再去算缺口缝补！

*   **3. Go 代码层的终极“减肥”**：
    *   **删除** `buildRawNetworkKPISeriesQuery` 等回退拼装 SQL 函数。
    *   **删除** `fetchNetworkKCodeSeries` 里面阶段 2、阶段 3 的所有 `trailingStart` 测算、去重与数组 `append` 逻辑。
    *   **替换为** 单一且清爽的视图扫表：
    ```go
    // 伪代码：彻底抛弃手动汇总逻辑
    func (s *Service) fetchNetworkKCodeSeries(...) {
        query := storage.Psql.Select("bucket_time as time", "metric_path", "GET_PROPER_AGG_VAL(statis_type) as metric_value").
            From("pm_metrics_hourly_cagg").
            Where(sq.GtOrEq{"bucket_time": startTime}).
            Where(sq.LtOrEq{"bucket_time": endTime})
        // 引擎自动保证返回的是 聚合好的历史 + 准实时的尾巴 两者融合的结果
        return s.scanNetworkSeries(ctx, query, args)
    }
    ```

---

## 3. 前端过度轮询：告警大盘未能与实时推送对接

### 3.1 具体问题与现象
监控大盘往往是常开挂在大屏上的视觉窗口。
*   **当前实现**：在前端 React 代码中，`useDashboard` 钩子大量使用了诸如 `refetchInterval: 60000` 甚至 `30000` 的轮询机制，强制每分钟向后端发压以同步图表与告警统计的状态。
*   **隐患**：高频的无效请求不仅耗费 Nginx 网关和 PG 数据库资源的计算量；对于紧急告警这类场景，“分钟级”的同步通常还意味着事件传递可能存在秒到分钟级不等的滞后。其实后端内部针对 MML 操作等其实已经拥有 SSE (Server‑Sent Events) 双向通信模块的设计基础，却没有覆盖给大盘统计。

### 3.2 优化改动细节方案
*   **应用 SSE 实时事件更新取代短轮询**：
    1.  **后端网关推送**：当设备告警状态发生新增、清除时以及当某定时 PM 时序数据分析落网完成后。Go 层利用现有的事件发布流，往指定的频道推一条状态指令：例如 `{"topic": "dashboard_update", "type": "alarm", "sn": "CMCC-011903"}`。
    2.  **前端监听**：取消前端硬编码的 `refetchInterval`，改为当接收到特定的 SSE Topic 时，调用 `queryClient.invalidateQueries({ queryKey: ['dashboard'] })` 或更细粒度地 `setQueryData` 进行界面的差量更新。
    3.  **收益**：使监控真正达到毫秒级别的准确实时率；解决轮询开销。

---

## 4. 冗杂的本地开发与启动流（Developer Experience）

### 4.1 具体问题与现象
*   **当前的开发启动体验**：我们在刚刚修复环境启动时，需要穿梭并手动运行诸多脚本组合（`stop-all.sh`, `start-deps.sh`, `dc.sh up`, `reset_and_import_seed_data.sh`），且在前端使用 Webpack/Vite 本地反代，后端某些模块跑容器并占据由 `DOCKER_BIP` 派生的私有网桥网段。
*   **隐患**：造成极大的记忆负担。极容易发生“忘记配 VITE_API_PROXY_TARGET”、“网段地址被占用冲突（Pool overlaps）”、“没有正确打好测试种子数据只导一半的情况”等各类联调报错。

### 4.2 优化改动细节方案
*   **重塑 Docker Compose 与统一声明**：
    1.  利用一套标准的 `docker-compose.dev.yml` 文件涵盖全部本地应用。
    2.  彻底发挥 Docker 中的 Healthcheck 级联特性 (`depends_on: -> condition: service_healthy`)。保证后端的 App 进程仅在 Postgres 跟 TimescaleDB 通过探针确认完全连通后再拉起。
    3.  针对测试 Mock 数据，做统一化 `Init-Container`。把 `seed_dev_data.sql` 等封装进独立的初始化镜像或放到 Postgres 官方库内置的 `/docker-entrypoint-initdb.d/` 下，确保容器每次 `down / up` 都全自动生成 100% 同步的结构化测试库，规避两边数量不同等杂乱情况。
    4.  引入项目级的 `Makefile` 顶层统一抽象，做到只敲 `make dev` 即可拉起前后端全部依赖并安全连通进行沉浸式开发。 
