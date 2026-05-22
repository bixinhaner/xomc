# PM / KPI 流水线改进设计方案

> 文档日期：2026-05-19
> 作者：syb + Claude
> 关联：T-0098（已收官，搭好"产品 → KPI 平台"静态关联）/ T-0121（PM 端到端链路缺口）/ §16.4 电信业务专家、§16.5 数据与存储专家
> 关联现有设计：`docs/design/参数-KPI-告警-整合设计方案.md`（数据字典平台化）

---

# 0. 背景与问题陈述

T-0098 收官后，OMC 在"数据字典层面"完成了 **产品 → 指标平台公式** 的装配（products 表的 `indicator_device_type` + `indicator_platform` 字符串软引用到 `rela_platform_indicator_formula_{enb,gsm,gnb}`）。同时 T-0098 的 indicator loader 已在启动期把 XML 指标库（含 `statisType` 字段，取值 `sum/avg/max/pct`）灌入 `perf_indicators_{enb,gsm,gnb}` 表。

但 **PM 文件采集 / KPI 计算 / 聚合** 这条运行时流水线**整条都没用上这套新装配**——它跑的还是"运营商适配器代码里硬编码的一份 KPI 列表"。当前并存两套互不相通的指标体系：

| 维度 | 老一套（KPIEngine 实际在用） | 新一套（T-0098 装好但 KPI 计算未消费） |
|---|---|---|
| 指标来源 | 运营商适配器代码中硬编码的 `KPIDefinitions` | XML 指标库 → `perf_indicators_*` + `rela_platform_indicator_formula_*` |
| 路由方式 | 按"运营商 + 制式"全量加载内存 | 应走"设备 → product → indicator_platform → 平台公式" |
| 聚合方式记录 | 无字段 | `perf_indicators_*.statis_type`（sum / avg / max / pct） |
| 在线调整 | 改代码 + 重启 | 改 XML / 改表 + 缓存失效 |

由此具体衍生出五个缺口：

| # | 缺口 | 现状 | 期望 |
|---|------|------|------|
| 1 | KPI 路由按"运营商+制式"全量加载，不经产品；且不消费 DB 中的新指标库 | KPIEngine 启动时把适配器代码里所有 carrier×tech 的公式 load 进内存，跑时按 carrier+tech 过滤；DB 中 `perf_indicators_*` / `indicator_platform` 链路被绕过 | 按设备的 productClass → product → indicator_platform → 平台公式集合路由，**只算这台设备所属产品声明支持的指标**；指标元数据完全来自 DB |
| 2 | KPI 时序表无过期保留策略 | `pm_metrics` 已是 hypertable，1 天 chunk；但无 compression、无 retention，长期单调增长 | 5 级粒度（含原始 15min + G5 引入的小时 / 日 / 周 / 月聚合表）统一挂 compression + retention policy；默认值按金字塔保留（详见 §4.2），运维可在系统设置页改 |
| 3 | KPI 计算的"counter 聚合"硬编码 SUM；新指标库 `statis_type` 字段被忽略 | TS 物化视图 `pm_counters_hourly` 写死 SUM/AVG/MIN/MAX 四列；KPIEngine 从 `pm_metrics` 取 counter 聚合值时硬编码 `SUM(counter_value)`；`perf_indicators_*.statis_type` 取值（sum/avg/max/pct）虽已落库但无消费方 | 聚合方式由指标元数据 `statis_type` 驱动；运维改 XML / 改表即生效，无需改 SQL / 重启 |
| 4 | PM 文件采集时间窗的起止时间未完整存储；OMC 入库本地时间也未单独记录 | `parser.go:110` 只读了 measInfo 段的 `granPeriod.endTime`，没读 PM 文件级的 `fileHeader/measCollec/@beginTime` 与 `fileFooter/measCollec/@endTime`；`pm_metrics` 只存一个 `time` 字段；OMC 本地处理 / 入库时间也没字段 | `pm_metrics` 显式存三个时间：`start_time`（基站采集起，直接读自 fileHeader）、`end_time`（基站采集止，直接读自 fileFooter）、`ingest_time`（OMC 入库时刻），用于区分基站时钟与服务时钟、排查上报延迟 |
| 5 | 缺少自然日历桶（小时 / 日 / 周 / 月）的预聚合 | 现状只有 15min 粒度原始数据；小时报 / 日报 / 月报需要每次现场扫大量原始行；删除物化视图后唯一兜底是 `QueryAggregated` 现场聚合，量大时性能不可接受 | OMC 后台定时任务按整点对齐的自然日历桶预聚合到独立的聚合表（小时 / 日 / 周 / 月），按 `statis_type` 元数据驱动聚合方式，报表接口直查聚合表 |

共同根因：**T-0098 完成了"配置态 + 数据态"，但"运行态"（PMCollector + KPIEngine）还停留在 T-0095 时代——从代码读 KPI 定义、按 carrier+tech 过滤、SUM 聚合**。在此基础上，PM 流水线还缺时间窗完整记录（缺口 4）与自然日历桶预聚合（缺口 5）。

---

# 1. 目标 / 非目标

## 1.1 目标

- **G1**：PM/KPI 流水线在运行时按"设备 → 产品 → 平台公式"动态路由，让"基站只算它该算的指标"成立；KPIEngine 完全消费 DB 中的 `perf_indicators_*`，不再依赖代码硬编码 KPI 列表
- **G2**：为 PM 时序数据（pm_metrics）建立统一的压缩 + 保留策略，挂到系统设置页 UI 可调；存储容量可预测
- **G3**：聚合函数对齐 XML 指标库现有取值（`sum / avg / max / pct`）由指标元数据 `statis_type` 驱动；合并 `pm_counters` / `kpi_values` 为单表 `pm_metrics`（含 `metric_type` 字段）；删除 TimescaleDB 物化视图，简化分层
- **G4**：在 pm_metrics 中显式存储 PM 采集时间窗（`start_time` / `end_time`，基站时钟）+ 入库时间（`ingest_time`，OMC 时钟），支持上报延迟监控、时钟漂移排查、补传识别
- **G5**：OMC 后台定时任务按整点对齐的自然日历桶（小时 / 日 / 周 / 月）预聚合到独立聚合表（设备维度 + 设备组维度），按 `statis_type` 元数据驱动；报表 / 趋势查询按粒度 + 行维度路由到对应聚合表，避免大窗口扫原始表
- **G6**：前端"性能管理"模块改为单 tab"性能查看"+ 仪表盘（Dashboard）+ 面板（Panel）两概念模型（替代原"概览 / 趋势探查 / 报表"三 tab + 报表模板）+ 制式顶层切换 + 对比双模式 + 点对点用户分享 / 协作 + 派生（fork）支持；KPI 卡片配置持久化到 user_preferences
- **G7**：用户即兴选 N 个设备做异步聚合（不沉淀设备组）；支持一次性 / 持续两种模式；任务式提交 + 进度可见；结果落 `pm_adhoc_aggregation_results` 快照表；全局保留期统一管理（drop_chunks 整块清）；走 PM 模块的 `pm_tasks` 表（per-module 模式），不进 G8 通用任务框架
- **G8**：通用任务框架——`async_jobs` 表 + 心跳 + 僵尸检测 + cron 触发补跑 + 分布式锁；承载 G5 cron 实例 / 未来新增的长流程异步计算任务；进程重启后任务可续跑（G7 自定义聚合改走 `pm_tasks` per-module 模式，详见 §4.7 / §4.8）

## 1.2 非目标

- 不替换 TimescaleDB 本身
- 不重构数据字典加载器（dictloader）
- 不动 PM 文件采集前段（ACS → MinIO → NATS 这段保持原样）
- 不在本次涉及 MR / 告警的聚合（如适用，本设计模式可作为参考）
- 不为 100 万规模做存储分片设计（属另一议题）
- 不引入 fallback / 灰度 / 兼容路径（项目未上生产，直接切换）

---

# 2. 当前流水线快照

> **现状基线**：CPE → ACS upload → MinIO → PMCollector 解析 → 写库的**完整链路已投产**，关键代码：
> - `internal/acs/upload/handler.go`（HTTP `/smallcell/FileUploadService` + token 鉴权 + 流式存 MinIO）
> - `internal/acs/handler.go` `handleTransferComplete`（CPE SOAP/CWMP "上传完成"通知）
> - `internal/transfer/bridge.go`（MinIO 入库 → 发布 `pm.file.received` 事件桥）
> - `internal/pm/collector/`（订阅事件 → 下载 MinIO → 解析 3GPP 32.435 XML → 写 counter / 算 KPI）
> - `internal/pm/counter/` + `internal/pm/kpi/`（落库到 `pm_counters` / `kpi_values`，G3 后合并为 `pm_metrics`）
>
> 本设计文档 G1-G5 **重构数据流的元数据驱动方式**（路由 / 聚合 / 保留 / 时间窗 / 预聚合），不是从零搭建接收链路；G6-G8 在已有数据之上增加用户面（仪表盘 + 面板）与任务编排（自定义聚合 + 通用任务框架）。

## 2.1 运行时基础设施（背景）

PM 流水线整体是**事件驱动 + 多消费者**模型（类比 Spring 的 `@KafkaListener`），而非"定时器扫表"。理解本设计 8 个改进点前需先对齐几个关键基础设施：

- **NATS JetStream 作为事件总线**：ACS 收到 CPE Upload 后落 MinIO，发 `transfer.file.uploaded` 事件；worker 进程订阅。
- **TransferBridge 中转**：worker 内的 TransferBridge 订阅 `transfer.file.uploaded`，识别文件类型后转发 `pm.file.received`（解耦"文件落地"与"PM 解析"）。
- **NATS queue group 模式**：PMCollector 用 `QueueSubscribe(..., "pm-workers", ...)` 订阅 `pm.file.received`——多个 worker 实例共享同一事件流，NATS 自动负载均衡（等价 Kafka consumer group）。**这是 worker 水平扩展的核心机制**。
- **Retry + DLQ 装饰器**：每个 handler 由 Retry+DLQ Runner 包裹（cmd/worker/main.go），失败自动重试，耗尽进死信表 `dead_letters`（等价 Spring Cloud Stream 的 DLQ）。

**对本设计的影响**：
- G1 路由：多 worker 并发取同一份"产品 → 指标清单"，路由结果必须可缓存且支持 cache_version 失效（详见 §4.1）
- G2 保留策略：hypertable（15min / hourly）由 TimescaleDB policy 自治；普通表（daily / weekly / monthly）由 worker 内的 G5 cron 清理任务负责；UI 改动通过 sys_configs 联动到两条路径
- G3 动态聚合：聚合元数据（statis_type）变更也走同一缓存失效协议，多 worker 一致生效
- G4 时间窗：parser 解析的三时间字段在 PMCollector handler 内一次性写入；多 worker 并发不冲突
- G5 预聚合：cron 任务运行在 worker 进程内，多 worker 部署的分布式锁与重启续跑由 G8 通用任务框架承担
- G6 前端：与 worker 无直接关系，纯前端 + app 层接口扩展
- G7 自定义聚合：PM 模块独立的 worker 池（`pm_tasks` per-module 模式），cron 调度时刻与 G5 对齐（详见 §4.7）
- G8 通用任务框架：worker 内承载所有长流程异步任务的执行器（含心跳、僵尸检测、补跑、分布式锁）

## 2.2 流水线视图

```
 CPE ──Upload──> ACS ──MinIO──> NATS ──> PMCollector
                                                │
                                                ├─ 解析 XML，写 pm_counters（现状表，G3 后合并入 pm_metrics）
                                                │
                                                ├─ 调 KPIEngine 算 KPI
                                                │     └─ 输入：运营商适配器代码里硬编码的 KPI 列表（按 carrier+tech 过滤）
                                                │        DB 中的 perf_indicators_* / indicator_platform 被绕过
                                                │        counter 聚合：固定 SUM
                                                │        输出：写 kpi_values（现状表，G3 后合并入 pm_metrics）
                                                │
                                                └─ 发 pm.file.parsed
                          ┌──────────────────────┴─────────┐
                          │ TimescaleDB 后台增量计算         │
                          │ 物化视图 pm_counters_hourly      │
                          │ （SUM/AVG/MIN/MAX 各列写死）      │
                          └────────────────────────────────┘
```

关键问题点：
- KPI 选谁：从代码硬编码列表里按 carrier+tech 过滤；未经产品；不消费 T-0098 后新填充的 `perf_indicators_*` / `rela_platform_indicator_formula_*`
- 聚合方式：物化视图硬编码；KPI 计算层硬编码 SUM；新指标库的 `statis_type` 取值（sum/avg/max/pct）已落库但无消费方
- 保留策略：旧 pm_counters 有（7 天压缩 / 90 天删）；旧 kpi_values 无

---

# 3. 目标流水线

```
 CPE ──Upload──> ACS ──MinIO──> NATS ──> PMCollector
                                                │
                                                ├─ 解析 XML（含 granPeriod.endTime + duration）
                                                │     → 算出 start_time / end_time，附加 ingest_time（G4）
                                                │
                                                ├─ 写 pm_metrics（TS hypertable，三时间字段）
                                                │
                                                ├─ 路由（G1）：device → product → indicator_platform → 指标子集
                                                │
                                                └─ 算 KPI：counter 字典 + arithmetic 四则运算 → 写 pm_metrics
                                                          （无窗口聚合；KPI 自身 statis_type 在 G5 cron 聚合时才用上）

  后台 cron 任务（worker 进程内）：                                            ┐
   - 每小时 :05 → 聚合上一小时 → 写 pm_metrics_hourly      │
   - 每天 00:05 → 聚合昨天    → 写 pm_metrics_daily       │
   - 每周一 00:10 → 聚合上周  → 写 pm_metrics_weekly      │  G5
   - 每月 1 日 00:15 → 聚合上月 → 写 pm_metrics_monthly   │
   - 任务两步：A) 从 15min 聚合 counter；B) KPI 按 statis_type 分两路           │
     · sum / avg / max KPI 从 pm_metrics 15min 直接聚合                         │
     · pct KPI 从当前级 counter 聚合表取值代入 arithmetic                       ┘

  报表 / 趋势查询接口（QueryAggregated）：按用户选的粒度路由到对应聚合表
  TimescaleDB 物化视图 pm_counters_hourly：删除（被 G5 应用层聚合表取代）
```

PM 流水线后端核心改进（G1-G5）对应：
- **G1（指标路由）** → 替换 PMCollector 选指标这一步
- **G2（保留策略）** → 5 级粒度统一挂 UI 可配的 compression + retention
- **G3（动态聚合 + 合并表）** → 按 statis_type 驱动；KPIEngine 实时计算简化（无 SQL 聚合）；合并 pm_counters / kpi_values 为 pm_metrics + metric_type；删除物化视图
- **G4（时间窗）** → 写入新增 start_time / end_time / ingest_time 三字段
- **G5（预聚合）** → cron 任务写 8 张聚合表（设备级 + 设备组级）；查询路由按粒度 + 行维度

附加改进（G6 前端 / G7 用户任务 / G8 通用任务框架）分别在 §4.6 / §4.7 / §4.8 详述，本节流水线视图侧重 G1-G5 的后端数据流。

## 3.1 表设计约定（贯穿 G3 / G4 / G5 / G7）

- **合并 `pm_counters` + `kpi_values` → `pm_metrics`**：counter（设备上报事实）与 KPI（公式导出值）字段结构完全相同（device_sn / cell_id / metric_name / time / value / 三时间），合并为单表；新增 `metric_type ∈ {counter, kpi}` 字段区分；原 `counter_name` / `kpi_name` 统一为 `metric_name`
- **元数据表保留分开**：`perf_counters_*`（描述 counter，无公式）与 `perf_indicators_*`（描述 KPI，含 arithmetic / statis_type）不合并——描述字段差异大
- **命名空间唯一**：counter 名 / KPI 名加载入库时校验，确保不撞名（同 metric_name 不能既是 counter 又是 KPI）
- **聚合表沿用合并设计**：G5 设备级 4 张 + 设备组级 4 张，共 8 张，全部含 `metric_type` 字段
- **查询时按 `metric_type` 过滤**：报表 / 趋势 / 下钻接口在 WHERE 子句加 `metric_type = ?`；G6 折线 panel 的"下钻 counter"从切表变成切过滤
- **重算的影响**：G3 改 statis_type / arithmetic 后重算历史 KPI 只需 `DELETE WHERE metric_type='kpi' AND time BETWEEN ...`，不影响 counter 行

---

# 4. 设计要点

## 4.1 指标路由：从"按 carrier+tech 扫全表"切到"按产品路由"（G1）

### 设计原则

- 路由结果是 **一份指标清单**（即"这台设备该算哪些指标"），由设备模型决定，可缓存
- 路由发生在 PMCollector 拿到设备身份的那一刻，不在 KPIEngine 内部硬塞 if/else
- **路由失败直接报错暴露**（不做 fallback）：开发期数据可控，遇到未匹配产品 / 空 indicator_platform 直接当种子数据 / 装配件配置缺陷修，比让它静默走旧路径更利于定位

### 路由链条（业务视图）

```
Device   ──product_class──>   ProductRegistry
                                    │
                                    ▼
                             Product 实例
                                    │
                          indicator_device_type + indicator_platform
                                    │
                                    ▼
                          指标库的"平台公式"集合
                                    │
                                    ▼
                             该设备应采集的指标清单
```

### 与 T-0098 现状的差异

- T-0098 已经搭好两件事：
  1. `products.indicator_device_type + indicator_platform` 字符串软引用到 `rela_platform_indicator_formula_{enb,gsm,gnb}`（设计/配置态）
  2. indicator loader 把 XML 指标库灌入 `perf_indicators_*`，含 `statis_type` 字段（数据态）
- 缺的是：**让 KPIEngine 在运行时按这条路找 KPI 列表 + 读元数据**，而非启动期把代码硬编码的 carrier KPI 列表 load all + 运行时仅按 carrier 过滤

### 缓存与失效

- 指标清单按产品维度缓存（与 T-0098 的 paramModel/indicator/alarm registry 一致），L1 内存 + L2 Redis + cache_version 失效
- 指标管理 UI 编辑后通过 cache_version 失效，PMCollector 下次取最新清单

### 异常暴露

- 设备未匹配产品 / indicator_platform 为空 → 跳过该设备的 KPI 计算 + 记 Warn 日志（含 device_sn / productClass）+ Prometheus 计数器 `pm_kpi_skip_total{reason=...}`
- 不引入 fallback 路径；开发期看到计数器涨 = 种子数据 / 产品装配件配置漏了，直接修

---

## 4.2 PM 数据保留策略（G2）

### 设计原则

- G5 引入后，PM 数据按粒度分 5 级（原始 15min + 小时 + 日 + 周 + 月）；**每一级各自挂 compression + retention policy**
- 同一级的 `pm_metrics_<G>` 不区分 counter / KPI，共享一组保留 / 压缩配置（一份配置覆盖一张表）
- 全部挂到系统设置页 UI 可调；迁移仅安装默认值
- **设备级 5 张表（pm_metrics + 4 聚合）与设备组级 4 张表（pm_metrics_group_*）共享同一份保留配置**——按粒度共用，不分维度；UI 上仍是 5 行（每粒度一行），保存时同时应用到该粒度的设备级和设备组级表

### 保留策略（默认值，运维可改）

| 粒度 | 表 | 压缩 | 保留 | 说明 |
|------|------|------|------|------|
| 原始 15min | pm_metrics | 7 天后压缩 | 32 天后删除 | 保证月桶聚合（每月 1 日跑、聚合上月 31 天）时原始数据仍在；长期趋势走聚合表 |
| 小时 | pm_metrics_hourly | 30 天后压缩 | 90 天后删除 | 近期小时报表 |
| 日 | pm_metrics_daily | 90 天后压缩 | 1 年后删除 | 季度同环比 |
| 周 | pm_metrics_weekly | 180 天后压缩 | 2 年后删除 | 长期趋势 |
| 月 | pm_metrics_monthly | 1 年后压缩 | 5 年后删除 | 历史归档 |

> 默认值按"层级越粗、保留越长"金字塔设计；运维可在 UI 改每一项

### 配置承载

- 复用已有 `sys_configs` 通用 KV 表（admin 模块）；新增 5 组 KV（每级一组 = 2 个 key：压缩天数 + 保留天数）
- 启动期：app 读取 `sys_configs`，按表的存储形态分两条路径同步：
  - **hypertable（15min / hourly）**：调 TimescaleDB `add_compression_policy` + `add_retention_policy`，policy 由 TS 自治执行
  - **普通表（daily / weekly / monthly）**：把保留天数写入 G5 cron 清理任务的运行参数；压缩天数对普通表不适用（由 PG TOAST 自动处理），UI 上该字段可置灰或显示"不适用"
- 运维改某一级保存：服务端按上述路径同步生效；hypertable 通过 `remove_*_policy` + `add_*_policy` 立即重置，普通表 cron 任务下一次跑就按新阈值
- 不需要重启

### 前端入口

复用 `system/SystemConfig/BasicSettings.tsx` 现有"系统设置"页，新增一个 **"PM 数据保留"分组**，含 5 行（每级粒度一行），每行两列（压缩天数、保留天数）：

| 粒度 | 压缩天数 | 保留天数 | 校验 |
|------|--------|---------|------|
| 原始 15min（hypertable） | 7（默认） | **32**（默认，保证月桶聚合时上月 31 天的原始数据仍在） | 压缩 ≥ 1 且 < 保留 |
| 小时（hypertable） | 30 | 90 | 同上 |
| 日（普通表） | —（不适用） | 365 | 保留 ≥ 1 |
| 周（普通表） | —（不适用） | 730 | 保留 ≥ 1 |
| 月（普通表） | —（不适用） | 1825 | 保留 ≥ 1 |

- 字段标签自带"PM 数据"前缀，避免与未来其他域（trace / syslog 等）的保留策略字段混淆
- 同分组、同保存按钮；点保存 → 后端同时更新对应 sys_configs KV + 对应表的 TimescaleDB policy
- 取值不预设上限（与项目"不预设保护性限额"风格一致）；不强制要求高层级保留期长于低层级（运维自负）
- 改动立即生效，无需重启

### 理由

- 5 级金字塔覆盖从 15min 到月的全粒度，运维按业务需求调
- 原始 15min 默认保留 **32 天 = 31（一个月最长天数）+ 1（buffer）**——保证月桶 cron 任务（每月 1 日凌晨跑）能扫到上月所有 15min 原始数据；G5 聚合任务统一从 15min 算（详见 §4.5），所以原始保留期必须 ≥ 最远聚合任务对历史数据的依赖
- 现状 7 天压缩 / 90 天保留 → 调整为 7 天 / 32 天（短保留期，因为 G5 聚合表已经承担长期趋势查询）
- 不同部署环境的存储预算 / 业务回溯期望不同，做成 UI 可配比写死更稳；改了不用发版

---

## 4.3 聚合元数据驱动 + 删除物化视图（G3）

### 设计原则

- 聚合方式由指标元数据 `statis_type` 驱动，对齐 XML 指标库现有取值（`sum / avg / max / pct`）；不再硬编码 SUM
- **删除 `pm_counters_hourly` 物化视图**：被 G5 应用层聚合表取代——物化视图固定列、不按 statis_type、无法跨产品分支；G5 的 cron 任务能完整满足这些需求
- 直接消费已有的 `statis_type` 字段；不新增 schema

### 元数据现状与启用方式（不引入新字段）

`perf_indicators_*` 已经有 `statis_type` 字段，XML 指标库现存取值：

| 取值 | 含义 |
|------|------|
| `sum` | 累加 |
| `avg` | 均值 |
| `max` | 最大 |
| `pct` | 百分率（percent），结果是比率值，单位通常为 % |

四个取值**平级**。`statis_type` 描述指标自身的聚合方式，与"是否原始 counter"（`isCounter`）是两个独立维度——任何组合都合法：

- 原始 counter（isCounter=1）：sum / avg / max / pct 都可以  
  示例：`RRC.AttConnEstab`(sum)、`RRC.SetupTimeMean`(avg)、`USER.ActMax`(max)
- 派生 KPI（isCounter=0）：sum / avg / max / pct 都可以  
  示例：`K900010001` RACH 接入成功率（pct，arithmetic = `C19/C18*100`）；其他派生 KPI 也可能是 sum / avg / max——取决于公式与业务含义

G3 范围严格对齐 XML 现有 4 个取值，不预设 `min / weighted_avg / percentile`。

### 三个环节：实时计算 / 定时聚合 / 用户查询

PM 数据从基站到用户看见，经过三个环节。`statis_type` 只在中间环节（定时聚合）参与计算：

| 环节 | 时机 | 主体 | 是否聚合 | statis_type 作用 |
|------|------|------|---------|------------------|
| 1. 实时计算 | 每 15 分钟收到 PM 文件 | KPIEngine | 否 | 不参与（仅前端展示用） |
| 2. 定时聚合 | cron 整点对齐 | G5 cron 任务 | 是 | 决定 SQL 聚合函数 |
| 3. 用户查询 | 任意时刻 | 报表 / 趋势接口 | 否 | 不参与（按粒度选表直读） |

---

### 环节 1：实时计算（KPIEngine 处理单个 PM 文件）

15 分钟上报一份文件，每个 counter 在这份文件里**只有一个值**——根本不存在聚合：

```
收到 PM 文件 →
    1) 解析 XML，得到 counter_name → value 字典
    2) 把每个 counter 写入 pm_metrics（一对一）
    3) 对路由进来的每个 KPI 指标 I：
         · 按 I.arithmetic 从 counter 字典里取出公式引用的 counter 值
         · 直接代入 arithmetic 做四则运算 → 得到 KPI 值
         · 写一行 pm_metrics
```

完全不需要 SQL 聚合，counter 值在内存里代入公式即可。

---

### 环节 2：定时聚合（G5 定时任务）

每个定时任务（小时 / 日 / 周 / 月）分**两步**执行：

- **步骤 A：聚合 counter** — 从 pm_metrics 15min 原始表读取桶内 counter 行，按 counter 自身 `statis_type` 选 SQL 聚合函数（SUM / AVG / MAX）→ 写入当前级 counter 聚合表
- **步骤 B：聚合 KPI** — 按 KPI 自身 `statis_type` 分两条路径：
  - `sum / avg / max` 类 KPI：从 **pm_metrics 15min 原始表** 读取桶内 I 的多行，直接做 SUM / AVG / MAX——与 counter 聚合走同一条逻辑
  - `pct` 类 KPI：从 **pm_metrics_<G>（步骤 A 已聚合好的当前级 counter 表）** 取 arithmetic 引用的 counter 值代入公式——pct 不能对 kpi 值直接聚合（详见 §4.5）

**关键**：所有跨窗口聚合都在定时任务里**预先算好存表**，用户查询不重复计算。

---

### 环节 3：用户查询（报表 / 趋势接口）

按用户选的粒度路由到对应聚合表，直接 `SELECT ... WHERE time BETWEEN ...`，**不做 SQL 聚合**：

| 用户选的粒度 / 时间窗 | 行维度 = 设备 / 小区 | 行维度 = 设备组 |
|---|---|---|
| 15min（或时间窗 ≤ 1 小时） | pm_metrics | pm_metrics（现场 GROUP BY device_group_id） |
| 小时 | pm_metrics_hourly | pm_metrics_group_hourly |
| 日 | pm_metrics_daily | pm_metrics_group_daily |
| 周 | pm_metrics_weekly | pm_metrics_group_weekly |
| 月 | pm_metrics_monthly | pm_metrics_group_monthly |

设备组级聚合表设计详见 §4.5 末尾"设备组维度聚合"。

查询接口性能稳定——任何粒度都只是按时间索引读取，不扫海量原始行。

---

### 与现状代码的差异

- **环节 1**：现状 `pg_repository.go:172` `QueryForKPI` 写的是 `SUM(counter_value) as total`，但 15 分钟窗口内每个 counter 只有 1 个值，SUM 是退化的。改造后这一步**完全去掉 SQL 聚合**，从 PMCollector 解析的 counter 字典里直接取值代入 arithmetic
- **环节 2**：现状没有 cron 聚合任务，由 TS 物化视图后台增量刷新承担（固定 4 列、不按 statis_type）。改造后由 G5 cron 任务接管，按 statis_type 聚合
- **环节 3**：现状 `QueryAggregated` 直查物化视图。改造后按粒度路由到 G5 聚合表，接口签名不变
- **结论**：`statis_type` 的实际消费方是**环节 2 的 cron 任务**，不是 KPIEngine、也不是查询接口

> 字段命名、缓存键、SQL 构造方式属实施细节，本设计不固化。

### 物化视图 vs G5 应用层聚合表

| 维度 | TS 物化视图（删） | G5 应用层 cron + 聚合表（采用） |
|------|----------------|--------------------------------|
| 改聚合方式 | DROP + CREATE 视图（停机风险） | 改一条元数据记录，下次 cron 跑即生效 |
| 时间粒度 | 固定按小时桶 | 小时 / 日 / 周 / 月 4 级 |
| 多产品差异 | 不可行（视图无法按产品分支） | 路由即支持（G1） |
| 写入时机 | TS 后台增量刷新，分钟级延迟 | cron 整点对齐 + 容忍延迟，可精确控制 |
| 按 statis_type 聚合 | 不可行（固定 sum/avg/min/max 4 列） | 支持（按各指标元数据动态选聚合函数） |
| pct 类正确聚合 | 不可行（视图无法两步） | 支持（先 counter 后 KPI 两步） |
| 维护成本 | DDL 复杂、压缩 / 保留单独配 | cron 任务 + 普通 hypertable，工程视角更直接 |

结论：物化视图在所有维度都被 G5 方案覆盖，本期 `DROP MATERIALIZED VIEW pm_counters_hourly`。

### counter 趋势接口的迁移

- 现状：前端"counter 按小时曲线"调 `QueryAggregated`，背后查 `pm_metrics_hourly`（物化视图）
- 改造：`QueryAggregated` 接口 URL / 入参 / 返回结构不变；底层按调用方传入的粒度路由到对应聚合表（详见 §4.5）
  - 短时间窗（如近 6 小时按 15min）→ 查 `pm_metrics`
  - 按小时 → 查 `pm_metrics_hourly`（G5 维护）
  - 按日 / 周 / 月 → 查对应聚合表
- 前端无感

---

## 4.4 PM 时间窗与入库时间存储（G4）

### PM 文件结构（3GPP 32.435）

> 以下结构基于真实的 ENB PM 文件（vendor Baicells、Baicells fileFormatVersion 32.435 v9.0）实际验证；GSM / GNB 文件格式基本类似（待 vendor 样本补充后验证细节差异）。

```
measCollecFile
├── fileHeader (vendorName, fileSender.elementType)
│     └── measCollec @beginTime           ← 文件级"采集窗口起"
├── measData
│   └── managedElement @localDn (Station=...)
│         └── measInfo
│               ├── granPeriod @duration @endTime  (采集任务粒度，常与 fileFooter 一致)
│               ├── repPeriod @duration            (上报周期)
│               ├── measType × N                   (counter 名定义，按 p="1..N" 编号)
│               └── measValue × M                  (M 个小区，每个 measObjLdn="Cellid=...")
│                     └── r p="X"                  (counter 值，按 p 对应到 measType 名)
└── fileFooter
      └── measCollec @endTime              ← 文件级"采集窗口止"
```

### 问题

现状 `parser.go:110` 只读了 measInfo 段的 `granPeriod.endTime`，写入 `pm_metrics.time`——丢了两类信息：
1. **文件级时间窗的 begin**：fileHeader 里就有，不需要靠 `endTime - duration` 算
2. **OMC 服务时钟的入库时刻**：无法区分"基站 8:00 的数据 8:02 收到 vs 9:30 收到"

### 设计

合并后的 `pm_metrics` 表补齐三个时间字段：

| 字段 | 来源 | 含义 |
|------|------|------|
| `start_time` | 基站时钟，**直接读** `fileHeader/measCollec/@beginTime` | 采集窗口起 |
| `end_time` | 基站时钟，**直接读** `fileFooter/measCollec/@endTime` | 采集窗口止 |
| `ingest_time` | OMC 服务时钟 = `time.Now()`（PMCollector 处理时） | OMC 入库时刻 |

- TimescaleDB hypertable 的分区列保持为 `end_time`（取代现有 `time` 字段，语义对齐）
- `parser.go` 改造：扩展解析 fileHeader / fileFooter，把 begin/endTime 一起放进 `PMFileContent`；不再依赖"endTime − duration"反推
- `PMCollector` 写入 `pm_metrics` 时填三个时间（counter 行与 KPI 行继承同一 PM 文件的 start/end）；`ingest_time` 用当前 wall clock

### 三时间字段的用途

- 前端时间轴展示：可选"按基站采集窗口"或"按 OMC 入库时间"画图
- 上报延迟监控：`ingest_time - end_time` 是上报延迟，超阈值可告警
- 时钟漂移排查：基站 endTime 比 OMC 本地时间提前 / 落后明显时报警
- 数据补传识别：补传的 PM 文件 `end_time` 远早于 `ingest_time`

### 时间字段的额外校验

PM 文件名编码也带起止时间（如 `A20260521.0445-0400-0500-0400_<vendorTag>.<stationSN>.xml`）；可在 parser 层校验文件名时间 vs 文件内 measCollec 时间一致性，发现 vendor 实现 bug 时告警。但以 XML 内容为权威数据源，不用文件名做业务取值。

### pm_metrics 行的唯一性维度

一份 PM 文件覆盖**多个小区**（每个 measValue 对应一个 measObjLdn），同一个 counter 在不同小区各有一个值；KPI 行同理。所以一行唯一性的维度是：

```
(device_sn, cell_id, metric_type, metric_name, end_time)
```

其中 `metric_type ∈ {counter, kpi}`，`metric_name` 对应原 `counter_name` / `kpi_name` 命名空间（XML 指标库 + 配置入库时保证 counter 与 KPI 不撞名）。

cell_id 也是 G5 聚合任务的分组维度之一（聚合按 device + cell 各算各的）。

pm_metrics 系列共 9 张表（原始 1 + 设备级 4 + 设备组级 4）统一用 `device_sn` 关联，不存 device_id。需要 JOIN 设备属性时按 `device_sn = devices.serial_number`。

### measObjLdn 解析（LDN 通用结构）

`measObjLdn` 是 3GPP 32.300 定义的 LDN（Local Distinguished Name）相对可辨别名，统一格式为 `<对象类型>=<对象ID>`，与 `managedElement.localDn`（如 `Station=eNb-...`）组合形成对象的完整路径。

不同制式 / vendor 的对象类型标签不同：

| 制式 | 典型类型标签（待 vendor 样本逐一验证） |
|------|-----------------------------------|
| ENB（LTE） | `Cellid=...`（Baicells 实测） |
| GNB（5G NR） | 推测 `NRCellCU=...` / `NRCellDU=...`，待样本验证 |
| GSM | 推测 `Cell=...` / `BTS=...`，待样本验证 |

**parser 改造要点**：cell_id 提取不能硬编码 `Cellid=` 前缀。改成按 LDN 通用结构 split by `=` 取 value 段——既能兼容已有 ENB 文件，也对 GSM / GNB 适配天然友好。具体类型标签的统一映射（ENB → cell、GNB → cell、GSM → cell 等）由 parser 层维护一张白名单，未识别的类型标签记 Warn 日志并 skip。

> LDN 的高阶用途（多级嵌套 / 类型校验）属实施细节，本设计不固化。LTE 的 ECI 28-bit 拆解（高 20 bit eNB ID + 低 8 bit 小区编号）属业务规则，与本设计无关，按需放业务层处理。

> 字段是否对所有查询都加 `start_time` 索引、`ingest_time` 是否压缩属实施细节，本设计不固化。

---

## 4.5 自然日历桶定时预聚合（G5）

> **术语说明**：本节"桶（bucket）"= 时间窗口。按粒度切成等长的时间格子，每个格子叫一个桶。表示法用数学的左闭右开区间 `[start, end)`——左端 **包含**、右端 **不包含**，保证相邻桶不重叠不漏数。例如 14 点的小时桶 = `[14:00:00, 15:00:00)`：包含 14:00:00 ~ 14:59:59，不含 15:00:00（属于下一桶）。

### 问题

15min 粒度数据细但量大。报表场景（小时报、日报、月报、按区域汇总）每次扫原始表性能不可接受；删除物化视图后 `QueryAggregated` 现场聚合只能撑得住短时间窗（24 小时）。需要稳定、可控、按 statis_type 聚合的预聚合层。

### 为什么不用 TimescaleDB Continuous Aggregate（物化视图）做

- 物化视图固定列（sum/avg/min/max），不能按 `statis_type` 选聚合函数
- 物化视图刷新是 TS 内部异步，难以精确控制"什么时候完成上一桶"
- pct 类 KPI 需要"先聚合 counter、再代入公式"两步，物化视图固定输出列做不到

所以本设计走**应用层 cron 任务 + 独立聚合表**。

### 聚合表与粒度

为合并后的 pm_metrics 表各级粒度创建聚合表，schema 形态与原表对齐（含 `metric_type` 字段），且都带 `start_time` / `end_time` / `ingest_time`（与 G4 一致）：

| 粒度 | 聚合表 | 桶范围 | 存储形态 |
|------|--------|--------|---------|
| 15min（原始） | `pm_metrics` | `[T, T+15min)` | TimescaleDB hypertable，`chunk_time_interval = 1 day` |
| 1 小时 | `pm_metrics_hourly` | `[H:00:00, (H+1):00:00)` | TimescaleDB hypertable，`chunk_time_interval = 1 day` |
| 1 天 | `pm_metrics_daily` | `[00:00:00, 24:00:00)` | 普通 PostgreSQL 表（不分 chunk） |
| 1 周 | `pm_metrics_weekly` | `[周一 00:00, 下周一 00:00)` | 普通 PostgreSQL 表（不分 chunk） |
| 1 月 | `pm_metrics_monthly` | `[1 日 00:00, 下月 1 日 00:00)` | 普通 PostgreSQL 表（不分 chunk） |

### 为什么 daily / weekly / monthly 不用 hypertable

它们数据量小（按 device × cell 计：daily 每天 1 行、weekly 每 7 天 1 行、monthly 每 30 天 1 行），TimescaleDB chunk 机制反而带来不必要的元数据开销。普通 PG 表 + 时间列索引即可。

### 存储形态对照管理路径

| 形态 | compression | retention | 谁来管 |
|------|------------|-----------|--------|
| Hypertable（15min / hourly） | TimescaleDB `add_compression_policy` | TimescaleDB `add_retention_policy` | TS 自治 |
| 普通表（daily / weekly / monthly） | PG TOAST 自动（无需配） | G5 cron 清理任务 `DELETE WHERE end_time < NOW() - interval` | 应用层 cron |

> 被删除的旧物化视图叫 `pm_counters_hourly`；G5 新建的 hypertable 叫 `pm_metrics_hourly`——名字不同避免误解。

### 调度时机（程序内定时任务，不依赖系统 crond）

调度机制采用项目现有的 `robfig/cron/v3` 库——**纯 Go 进程内调度**，跑在 worker 进程的 goroutine 里。表达式是 cron 语法（5 字段），但执行体完全在程序内部，部署 / 容器 / 镜像不需要任何系统 crontab 配置。

**触发后走 G8 通用任务框架**：cron 表达式触发不直接执行聚合，而是**写一条 `async_jobs` 记录入库**（type = `kpi_rollup_<G>` / `kpi_rollup_group_<G>`），由 worker 池拉起执行——这样进程重启、心跳超时、补跑、分布式锁都由 G8 统一承担。详见 §4.8。

| 任务 | 调度表达式 | 容忍延迟 | 聚合范围 |
|------|-----------|---------|---------|
| 小时桶 | `5 * * * *`（每小时 05 分） | 5 分钟 | 上一个完整小时 |
| 日桶 | `5 0 * * *`（每天 00:05） | 5 分钟 | 昨天整天 |
| 周桶 | `10 0 * * MON`（周一 00:10） | 10 分钟 | 上周整周 |
| 月桶 | `15 0 1 * *`（每月 1 日 00:15） | 15 分钟 | 上月整月 |

容忍延迟用来等"晚到的 15min PM 文件入库"。延迟超出容忍的数据需要"重算最近 N 个桶"——本设计不固化具体策略（看运营经验调整），由 §4.8 G8 通用任务框架统一承担"指定桶时间区间补跑"入口（无论手动触发还是 cron 错过自动补跑都走同一套）。

**聚合方法**（每个定时任务两步执行；**数据源统一是 15min 原始表**）

```
任务（粒度 G，桶时间窗 [start, end)）：
    步骤 A：聚合 counter（从 pm_metrics WHERE metric_type='counter' 算）
        对每个 counter c：按 c.statis_type 选聚合函数
              · sum → SUM(value)
              · avg → AVG(value)
              · max → MAX(value)
              · pct → 同 sum 兜底（counter 标 pct 罕见）
        → 写一行到 pm_metrics_<G>（metric_type='counter'）

    步骤 B：聚合 KPI（按 KPI.statis_type 分两条路径）
        对每个 KPI I：
            ┌── I.statis_type ∈ {sum, avg, max} ─────────────────────────────
            │  从 pm_metrics WHERE metric_type='kpi' 读取桶内 I 的多行
            │  按 I.statis_type 做 SUM / AVG / MAX
            │  (与 counter 聚合同样的直接聚合——对 15min 桶里的 KPI 值做)
            └────────────────────────────────────────────────────────────────
            ┌── I.statis_type = pct ──────────────────────────────────────────
            │  从 pm_metrics_<G> WHERE metric_type='counter'（步骤 A 刚写好的）
            │  取 I.arithmetic 引用的各 counter 值
            │  代入 arithmetic 求结果
            │  (pct 不能直接对 kpi 值聚合——见下方"为什么 pct 走特殊路径")
            └────────────────────────────────────────────────────────────────
        → 写一行到 pm_metrics_<G>（metric_type='kpi'）

所有写入都附带 start_time / end_time / ingest_time
```

### 为什么 pct 走特殊路径，其他 KPI 不需要

pct 类 KPI（如 RACH 接入成功率 = 成功次数 / 尝试次数 × 100）跨桶聚合**数学上不能直接对 kpi 值聚合**。例：

- 14:00 这个 15min 桶：成功 9 / 尝试 10 → pct = 90%
- 14:15 这个 15min 桶：成功 0 / 尝试 100 → pct = 0%
- 1 小时桶的真值：`(9+0) / (10+100) × 100 = 8.18%`
- 1 小时桶若直接 AVG kpi 值：`(90% + 0%) / 2 = 45%`——**大错**

所以 pct 必须"回 counter 维度重算"——把成功次数与尝试次数各自 SUM 到小时桶（步骤 A 已经做了），再代入公式得 8.18%。

`sum / avg / max` 类 KPI 没有这个陷阱：
- sum 类（如 K_total = C_a + C_b）：跨多个 15min 桶 SUM(K_total) = SUM(sum_Ca + sum_Cb) = 总和正确
- max 类：MAX 满足结合律
- avg 类：每个 15min 桶里 K_avg 就是一个确定值，跨多桶 AVG 就是数学真值（不存在 avg-of-avg 偏差，因为 15min 桶里 KPI 没经过 avg 操作）

所以只对 pct 单独走"counter 重算"路径。

### 为什么不走"金字塔"（上层基于下一层算）

直觉上让小时桶基于 15min、日桶基于小时、月桶基于日，扫描量最小，性能最好。但有数学风险：

- **sum / max** counter：满足结合律，逐级聚合等价
- **avg** counter：`AVG(各小时 AVG) ≠ AVG(所有 15min 原始)`，除非每小时桶样本数完全相等
  - 例：14 点 3 个样本 `[10,20,30]` avg=20；15 点 4 个样本 `[5,5,5,5]` avg=5；两小时金字塔 avg=`(20+5)/2 = 12.5`，但原始 avg=`80/7 ≈ 11.43`——偏差 ~9%
- **pct 类 KPI**：如果公式引用 avg 类 counter，继承上述偏差

avg 类指标库里普遍存在（`RRC.SetupTimeMean`、`ERAB.EstabTimeMean`、`USER.ActAve` 等），金字塔会累积偏差。

### 统一从 15min 原始表算的代价与收益

- 代价：日桶 / 月桶要扫更多原始行（月桶 ≈ 2880 行/cell × cell 数）；但 TimescaleDB chunk 索引 + 列式压缩对这种 GROUP BY 是秒级响应，且月桶任务一个月才跑一次
- 收益：数学正确、补传 / 异常时只重算单桶、单一数据源、错误处理简单

**前置依赖**：原始 15min 保留期 ≥ 月桶聚合需要的最远历史数据时间（默认 32 天，G2 §4.2）。这是 G2 / G5 共同设计点：G2 默认值不是凭感觉拍的，是 G5 聚合源选择的反推结果。

### 保留期（与 G2 统一管理）

各级表的压缩 + 保留策略与原始表一起，全部纳入 G2 的"5 级金字塔"管理（详见 §4.2）；本节不重复，仅强调一点：**原始 15min 保留期是 G5 聚合任务的硬约束**——必须 ≥ 月桶聚合需要的最远历史数据时间。默认 32 天恰好覆盖"一个月最长 31 天 + 1 天 buffer"，UI 调整时要遵守这条约束。

### 与查询接口的关系

报表 / 趋势查询接口（如 `QueryAggregated`）根据用户选的时间粒度选表：
- 用户选"按 15min"或时间窗 ≤ 6 小时 → 查 pm_metrics
- 用户选"按小时" → 查 _hourly
- 用户选"按日 / 周 / 月" → 查对应聚合表

避免大时间窗扫原始表，性能可预期。

### 设备组维度聚合（G5 扩展）

> 由 G6 仪表盘"行维度 = 设备组"驱动：月报 × 大组（如华东全网 1 万设备）若现场 GROUP BY device_group_id 扫聚合表，扫描量 = 设备数 × 时间桶数，性能不可控。设备维度聚合表是必选预聚合层。

为每个时间粒度新增一张设备组聚合表（共 4 张，沿用 §4.5 的合并表设计）：

| 粒度 | 设备组聚合表 | 存储形态 |
|------|------------|---------|
| 1 小时 | `pm_metrics_group_hourly` | TimescaleDB hypertable |
| 1 天 | `pm_metrics_group_daily` | 普通 PG 表 |
| 1 周 | `pm_metrics_group_weekly` | 普通 PG 表 |
| 1 月 | `pm_metrics_group_monthly` | 普通 PG 表 |

**行键**：`(device_group_id, metric_type, metric_name, start_time, end_time, ingest_time, value)`，`metric_type ∈ {counter, kpi}`

**为什么没有 15min 粒度的设备组表**：设备组通常用于报表（小时 / 日 / 周 / 月），15min 粒度对设备组无业务意义（噪声大、无报表场景）；如有特例查询接口现场算。

**调度**（紧接设备级聚合任务之后跑，确保数据源已就位；触发后同样写入 G8 async_jobs，type = `kpi_rollup_group_<G>`）：

| 任务 | 调度表达式 | 设备级任务时点 | 数据源 |
|------|-----------|--------------|--------|
| 设备组小时桶 | `15 * * * *` | `5 * * * *`（已完成） | pm_metrics 15min 原始表 |
| 设备组日桶 | `15 0 * * *` | `5 0 * * *`（已完成） | 同上 |
| 设备组周桶 | `20 0 * * MON` | `10 0 * * MON`（已完成） | 同上 |
| 设备组月桶 | `25 0 1 * *` | `15 0 1 * *`（已完成） | 同上 |

**聚合方法**（与设备维度同源，仅 GROUP BY 维度不同）：
- **步骤 A**：从 pm_metrics 15min 原始表 `WHERE metric_type='counter'` JOIN device_group_members → GROUP BY `(device_group_id, metric_name, time_bucket)`，按 counter 的 `statis_type` 选 SQL 聚合函数 → 写 `pm_metrics_group_<G>`（metric_type='counter'）
- **步骤 B**：sum / avg / max 类 KPI 从 pm_metrics 15min 原始表 `WHERE metric_type='kpi'` 同样 JOIN + GROUP BY；pct 类 KPI 从步骤 A 刚写好的设备组 counter 行取值代入 arithmetic（与 §4.5 同理：pct 必须回 counter 重算，业务语义是"网络整体接入成功率"而非"每设备成功率平均"）

**为什么坚持从 15min 原始表算（不从设备级聚合表算）**：
- 设备级聚合表已按 statis_type 算好，从它再做设备组维度聚合理论上算"双层聚合"
- 但 avg 类指标双层聚合会累积偏差（与 §4.5 时间维度金字塔同理）
- 月桶任务 × 大组扫描量虽大，但 TimescaleDB chunk + 列式压缩 + 一个月才跑一次，可接受
- 与 §4.5 保持同源、同方法，工程简洁

**设备组成员变更的语义**：cron 任务执行时按"当时的成员快照"算；加入 / 移除设备后的下一个聚合周期生效，历史数据反映"那个时点的成员"——不重算、不补偿。

**保留策略**：
- 沿用 G2 五级金字塔（设备组级与设备级保留期同表配置；后续若发现量级差异再分级）
- `*_group_hourly` 是 hypertable，挂 TS compression + retention policy；`*_group_daily / weekly / monthly` 是普通表，靠 cron 清理

**查询接口路由**：
- 报表"行维度 = 设备组" → 直查 `pm_metrics_group_<G>`
- 报表"行维度 = 设备" → 仍走 §4.5 的 `pm_metrics_<G>`
- 报表"行维度 = 小区" → 同上（cell_id 在 §4.5 表里）

> SQL 实现、cron 任务的失败重试与告警、聚合表的索引设计属实施细节，本设计不固化。

---

## 4.6 前端 KPI 图表：仪表盘 + 面板模型（G6）

> **设计简化路径**：原方案为"三视图分层（概览 / 趋势探查 / 报表）+ 三种模板概念（内置模板 / 自定义模板 / 自定义聚合）"，存在 tab 与概念冗余。**本次简化为"一个 tab + 仪表盘 + 面板"两概念**（沿用 Grafana / BI 行业标准模型），用户认知负担降一档。

### 设计原则

- 一个菜单"性能管理"下**一个 tab —— "性能查看"**（合并原"概览 / 趋势探查 / 报表"三 tab）
- 结构：**左侧仪表盘列表 + 右侧当前仪表盘渲染**
- **核心概念简化为两个**：仪表盘（Dashboard）+ 面板（Panel）
- **制式作为顶层强制单选**（GSM / LTE-ENB / 5G NR-GNB）保留：三种制式的指标差异极大，不在同一张图 / 同一张表展示
  - 切换制式时整页刷新——KPI 卡片配置、内置仪表盘、设备组列表、筛选状态都按制式各自独立
  - 制式选择持久化到 `user_preferences`，下次进入恢复
- 仪表盘内顶部共享筛选条（时间窗 / 设备组 / 设备多选），多 panel 共享
- 粒度切换前端不感知表名，由 `QueryAggregated` 后端按"粒度 + 行维度"路由到对应聚合表（详见 §4.3 + §4.5）

### 核心概念

| 概念 | 含义 |
|------|------|
| **仪表盘（Dashboard）** | 多个面板的有序组合 + 共享筛选条；可保存、可分享给指定用户（见本节"分享 / 协作"） |
| **面板（Panel）** | 单个展示单元，类型可选：折线图 / 柱状图 / 表格 / KPI 卡片（多指标拼盘）/ TopN / 数值大屏（单数值大字号展示） |

仪表盘分两类：

- **系统仪表盘**：内置（GSM/ENB/GNB × 全网概览 / 日报 / 周报 / 月报），承担"角色页 / 对账报告"职责，全员公开 + readonly
- **我的仪表盘**：用户自建保存，含"一次性查询不保存"模式

### 布局

```
菜单 / 性能查看（一个 tab）
├── 顶部：制式（顶层强制单选）
├── 左侧：仪表盘列表
│       ├── 📌 系统仪表盘（GSM/ENB/GNB × 全网概览 / 日报 / 周报 / 月报）
│       ├── ⭐ 我的仪表盘（用户保存）
│       │   + 新建仪表盘  + 自定义聚合（一次性 / 持续，见 §4.7）
│       └── 🤝 分享给我的（其他用户分享的仪表盘 / 快照，见本节"分享 / 协作"）
└── 右侧：当前仪表盘渲染
        ├── 顶部：时间窗 + 设备组/设备多选（共享筛选条）
        ├── 多 panel 网格混排（每个 panel 自带"输出粒度"，多选；切换 / 下钻无需重跑）：
        │     ├── KPI 卡片 panel（× N）
        │     ├── 折线图 panel（多 KPI 叠加 + 添加对比时段）
        │     ├── 表格 panel（行 = 设备 / 小区 / 设备组 / 自选 N 设备 × 列 = 时间桶）
        │     └── TopN 表 panel
        └── [+ 添加面板] [另存为我的仪表盘] [导出 Excel/PDF] [分享给用户]
```

### 面板属性

```
panel = {
  type: 折线 | 柱状 | 表格 | KPI 卡片 | TopN | 数值大屏,
  devices: 设备集 | 设备组 | 自定义 N 设备,
  kpis: [KPI ID 集],                              ← API 字段 kpis；DB 层在 pm_tasks.kpi_codes
  granularities: [15min | 小时 | 日 | 周 | 月],   ← 数组，至少一项；API 字段 granularities；DB 层在 pm_tasks.granularity（JSONB 数组，schema 改造后承载多值）
  timeWindow: 共享筛选条 | 覆盖共享筛选条,
  compareWith: [对比时段 | 快照引用]?            ← 支持对比双模式（同快照 / 跨快照，见下文"对比功能"）
}
```

**输出粒度多选**：用户常见诉求是"既要趋势细节（小时），又要日报对账（天）"——一次扫描覆盖多视角。快照页支持按粒度切换 tab / 下钻，无需重跑。详细规则见 §4.7 "多粒度多选"段。

### 数据加载分流（按 panel 行维度静态分流）

| 场景 | 数据来源 | 加载方式 |
|------|---------|---------|
| 内置仪表盘 panel（行维度 ∈ 设备 / 小区 / 已有设备组）| G5 预聚合表 `pm_metrics_<G>` / `pm_metrics_group_<G>` | **同步 SELECT**，毫秒级返回 |
| 我的仪表盘 panel（用户保存，行维度命中预聚合）| 同上 | 同步 SELECT |
| 自选 N 设备聚合（行维度 = 用户即兴选定，不沉淀为设备组）| `pm_metrics` 15min 原始表，按 KPI 公式重算（见 §4.7）| **走 pm_tasks 任务式**（异步，进度可见，结果落 `pm_adhoc_aggregation_results`）|
| 跨多个已有设备组的临时分析（命中预聚合）| 预聚合表 UNION | 同步 SELECT |

**判断逻辑**：
- 行维度 ∈ {设备, 小区, 已有设备组} → 命中预聚合 → 同步 SELECT
- 行维度 = 用户自选 N 设备 → 走 pm_tasks 任务式（无论粒度、无论行数）

**关键收益**：
- 99% 仪表盘 panel 走同步，体验快（无任务列表干扰）
- 万级设备 / 上千指标的自定义聚合走任务式，HTTP 不阻塞、进度可见
- 用户只在"自定义聚合"场景才看到"我的任务"列表概念

**右上角"任务托盘"**：只显示在跑的自定义聚合任务（不显示仪表盘 panel 查询）。

### 全局设计约定（所有 panel 一致）

- **pct 类 KPI**：单位 %、独立 Y 轴；支持叠加阈值线（红 / 黄 / 绿带；阈值挂在 `perf_indicators_*` 元数据上）
- **sum / avg / max 类 KPI**：原始数值，可选同比 / 环比角标
- **时间轴**：默认按 `end_time`（基站时钟）；右上角小开关切 `ingest_time`（OMC 时钟）—— 直接消费 G4 三时间设计
- **上报延迟提示**：数据点 hover 显示 `ingest_time − end_time`；超阈值的点加角标
- **空值 / 缺采**：明确显示"缺采"，不画 0；避免误判
- **未匹配产品的设备**：前端按"无数据"处理，后端 G1 的 `pm_kpi_skip_total` 计数器与 Warn 日志承接异常排查

### 对比功能（双模式）

折线图 panel 支持"+ 添加对比时段"按钮，叠加多条对比线。两种模式：

**模式 A：同快照内多组对比（主路径，覆盖 80% 场景）**

- 自定义聚合按大范围跑（例如 3 个月）→ 一份快照覆盖整个范围
- 展示时在快照内挑任意两个时段做对比，例如"本周一 vs 上周一"——两天数据都在同一份快照里
- **零额外任务成本**：对比 = 快照表内 SQL 取两个时间段并列展示

**模式 B：跨快照对比（进阶，应对事后才发现的对比需求）**

- 用户已有快照 X（3 月数据），又跑了快照 Y（4 月数据）→ 想对比"3 月 vs 4 月"
- 系统支持挑选两个或多个已有快照做对比 SELECT，免重跑大任务
- **维度对齐校验**：两份快照的 KPI 集 / 设备集 / 粒度需有交集；不一致时只在交集上做对比 + 给提示

**预设对比项**：

| 预设对比项 | 计算方式 |
|----------|---------|
| 对比昨天 | 主时段整体平移 -1 天 |
| 对比上周同期 | 主时段整体平移 -7 天 |
| 对比上月同期 | 主时段整体平移 -1 个自然月（按天数对齐）|
| 对比上年同期 | 主时段整体平移 -1 年 |
| 对比紧邻前段 | 主时段长度的等长前段紧邻拼接 |
| 自定义时段 | 用户挑选起止 |

**对齐与约束**：
- X 轴按"相对偏移"对齐——主时段第 N 个时间点和对比时段第 N 个时间点画在同一个 X 位置（主流做法，Grafana / Datadog 同理）
- 对比时段长度 = 主时段长度，对比时段粒度 = 主时段粒度
- 对比时段超出全局保留期（§4.7）→ 显示"无数据"或自动降级提示

**视觉规范**：
- 主线：粗实线
- 对比线：虚线 + 半透明，颜色按对比项区分（同色系不同明度）
- 图例显式标注"主时段"与每个对比项的时段范围

### 仪表盘"另存为派生"

用户打开"全网概览"系统仪表盘，调整几个 panel 配置，希望"另存为我的仪表盘"——基于现有仪表盘派生一份个性化版本，是 Grafana / BI 系统的标配交互。

- **入口**：当前仪表盘工具栏 → "另存为..." → 弹窗输入新仪表盘名 → 复制全部 panel + 共享筛选条到新 dashboard 记录
- **派生关系**：派生后两份独立——系统仪表盘日后升级（新增 panel / 改默认 KPI）**不自动同步**到派生版本；不维护"基于哪个系统仪表盘派生"的引用（保持简单；若未来有"同步更新"诉求再加）

### 分享 / 协作（点对点用户分享）

**模型**：**点对点用户分享**。A 用户把仪表盘 / 快照显式分享给 B 用户，B 用户额外可见。无 public 概念，无 team 概念。

**用户能看到什么**：
- 系统内置仪表盘（全员公开 + readonly）
- 自己创建的仪表盘 / 快照
- 别人分享给自己的仪表盘 / 快照
- admin 角色可看全部（运维 / 故障排查需要）

**分享操作**：
- 仪表盘工具栏 / "我的任务"列表条目 → [分享 ▼] → 弹窗搜索 + 选择用户 → 确认
- **仅 owner 可分享 / 撤销**；被分享方不能转分享给第三方
- 权限层级：**viewer-only**（默认且唯一，不区分 editor）

**为什么不做 editor 权限**：
- 多人改同一份引入版本冲突
- 想"基于别人的改造一份" → 走"另存为派生" fork 成 B 私有，独立演化

**撤销 / 失效语义**：

| 触发 | 行为 |
|---|---|
| owner 在分享列表移除某用户 | B 立即不可见 |
| owner 删除仪表盘 / 快照 | 所有分享自然失效 |
| 全局保留期到期（§4.7） | 快照分享同时失效 |

**数据模型**（细节移至 plan）：

```
dashboard_shares (dashboard_id, shared_to, granted_at, granted_by)
snapshot_shares  (job_id,        shared_to, granted_at, granted_by)
```

或合并为单表 `shares + entity_type` 列；二选一移至 plan 阶段决定。查询"我可见的仪表盘" = `owner = me` ∪ `shared_to 含 me` ∪ 系统内置；admin → 全部。

**审计**（运营商合规需要）：分享 / 撤销操作进系统审计日志（granted_by + shared_to + entity + timestamp），复用现有审计基础设施，无需新建。

**不做的（YAGNI）**：editor 权限 / 分享给"角色 / 团队" / 链接分享 / 二维码 / 公开链接 / 失效期分享。

### KPI 卡片可配置

- 三种制式（GSM / ENB / GNB）**各内置一套默认卡片集**（前端常量定义；KPI 取自 `perf_indicators_{gsm,enb,gnb}`）
- 用户可对当前制式添加 / 移除 / 排序卡片
- 偏好持久化到后端 `user_preferences` 表，**每用户 × 每制式各一份**（结构按 `{ enb: [...], gnb: [...], gsm: [...] }` 组织）；本期不做团队 / 全局共享

### 与原 G6 设计的对照

| 维度 | 原 G6（三视图分层） | 简化后（仪表盘 + 面板） |
|------|---------------------|------------------------|
| Tab 数 | 3（概览 / 探查 / 报表） | 1（性能查看） |
| 核心概念 | 三 tab 各有角色 + 内置/自定义报表模板 + 自定义聚合任务 = 多个 | 仪表盘 + 面板 = 2 个（行业标准模型） |
| 角色定位 | 运维人员有"概览 tab"作为角色页 | 系统内置"全网概览"仪表盘充当角色页（默认打开） |
| 报表"日 / 周 / 月" | 9 套内置报表模板（GSM / ENB / GNB × 日 / 周 / 月） | 9 套**内置仪表盘**承担"对账 / 报告"角色 |
| 自定义模板 | 用户保存的报表模板（DB 表 `report_templates`） | 用户的"我的仪表盘"（`dashboards` + `panels`），与报表模板合并为同一模型 |
| 同步 / 异步 | G6 同步、G7 异步（写死边界） | 按 panel 行维度静态二分：命中预聚合走同步；自选 N 设备走 pm_tasks |

> 卡片配置 schema、模板存储 schema、URL 参数命名、分享 token 撤销机制属实施细节，本设计不固化。

---

## 4.7 自定义聚合任务：用户即兴选设备做异步聚合（G7）

> **命名约定**：前端中文文案统一叫 **"自定义聚合"**；后端 task_type 字段值用业界 BI 标准词 `adhoc_aggregation`；菜单 / 按钮 / 任务列表标题中不出现"adhoc"。
>
> **任务表归属**：本次自定义聚合复用 PM 模块现有的 **`pm_tasks`** 表（migration 000005，per-module 模式），**不进 §4.8 G8 通用任务框架的 `async_jobs`**——后者继续承担 G5 cron 等系统驱动的后台任务。两表分流原因：pm_tasks 更贴近"用户驱动 / 业务领域绑定"，async_jobs 更贴近"系统驱动 / 跨模块通用"。

### 定位与边界

与 G5 设备组定时聚合**互补**——G5 跑常驻设备组、整点对齐、长期保留；G7 让用户**当场选 N 个设备**（不沉淀设备组）、提交后异步算、看进度。

| 维度 | G5 设备组聚合 | G7 自定义聚合任务 |
|------|--------------|----------------|
| 触发 | cron（小时 / 日 / 周 / 月） | 用户即兴提交 |
| 设备集合 | topology 设备组（持久、共享）| 用户选定（不沉淀） |
| 计算时机 | 整点对齐 + 容忍延迟 | 提交后立即跑（oneshot）或周期跑（continuous）|
| 数据源 | pm_metrics 15min 原始表 | pm_metrics 15min 原始表（与 §4.5 同源） |
| 结果存储 | 长期聚合表（保留期跟 G2） | `pm_adhoc_aggregation_results` 快照表（全局保留期，见本节）|
| 任务表 | `async_jobs`（§4.8 G8 框架）| **`pm_tasks`**（per-module，本节）|
| 适用场景 | 例行报表 | 排障 / 即兴分析 / 跨组对比 / 长期跟踪自选 N 设备 |

### 两种任务模式

| 模式 | 时间窗 | 行为 | 数据增长 |
|------|--------|------|---------|
| **oneshot（一次性）** | startTime + endTime 必填 | 跑一次 → 快照定格 | 有上界 |
| **continuous（持续）** | startTime 必填，endTime 留空 | cron 按粒度周期跑，增量追加快照 | 滚动追加；旧 chunk 按全局保留期持续 `drop_chunks` 清，不依赖用户停止 |

**保留期统一遵守全局保留期**（见本节"全局保留期"段）——用户创建任务时无保留期字段、无决策点；oneshot 与 continuous 共用同一系统级参数。

### 任务模型（复用 pm_tasks 表）

**复用现有 PM 任务表**（migration 000005，schema：id / task_name / task_type / device_sns / kpi_codes / granularity / time_range / status / progress / creator / created_at / updated_at）。

**字段映射**：

| 维度 | pm_tasks 字段 |
|------|--------------|
| 输入设备 SN 列表 | `device_sns` JSONB |
| KPI 集 | `kpi_codes` JSONB |
| 时间窗（startTime / endTime） | `time_range` JSONB |
| 粒度多选（本次扩展为多值数组）| `granularity`（schema 改造） |
| 状态机 | `status`（pending → running → completed / failed / cancelled） |
| 进度 | `progress`（JSON：已完成桶 / 总桶 + 已完成设备 / 总设备）|
| 归属 | `creator` |

**本次扩展字段**（具体 schema 移至 plan）：
- `mode`：oneshot / continuous
- `cursor`：continuous 增量游标（记录"上次 cron 跑到的时间点"）
- `result_rows_count`：审计便利

**task_type 枚举**：
- **本次新增**：`'adhoc_aggregation'`
- **现有占位枚举清理**：`'extraction'` / `'report'` / `'threshold-check'` 当前无执行端逻辑、生产基本未用，本次清理废弃（VARCHAR(30) 字段开放，清理只是代码侧不再使用，无需表结构改动）
- **未来扩展开放**：task_type 字段保持开放，PM 模块新增后台任务（如性能门限自动检查、PM 抽样诊断、按需文件抽取等）直接新增枚举值即可，无需新建表
- **与 G5 cron 边界**：G5 cron 预聚合任务继续走 §4.8 G8 通用任务框架的 `async_jobs`，本次**不**迁移到 pm_tasks

### 数据源统一：pm_metrics 15min 原始表

**规则**：自定义聚合（行维度 = 自选 N 设备）**一律读 pm_metrics 15min 原始表，按 KPI 公式重算到目标粒度**——即使预聚合表已有对应粒度也不复用。

**为什么不能复用预聚合表**：很多 KPI 是公式类（接通率 = 成功 / 尝试、掉话率、时延 P95、加权 AVG 等）。从"已聚合数据"再聚合会引入误差：

| KPI 类型 | 再聚合是否安全 |
|---------|---------------|
| 纯计数器 SUM / MAX / MIN | ✅ 安全（SUM-of-SUM = SUM、MAX-of-MAX = MAX）|
| 简单 AVG | ⚠️ 需要分子分母都保留才安全；只存 AVG 列再 AVG 会丢权重 |
| Ratio / 百分比（接通率、掉话率等）| ❌ 必须从分子分母原始计数器算 |
| Percentile（P95、P99）| ❌ 无法从已聚合 P95 再算 P95 |
| 加权指标 | ❌ 必须从原始数据带权重重算 |

**反例**（接通率为何不能 AVG）：

| 设备 | 尝试 | 成功 | 接通率 |
|------|-----|------|-------|
| A | 100 | 95 | 95% |
| B | 10 | 5 | 50% |
| **真值（从原始算）** | 110 | 100 | **90.9%** |
| **AVG 错算** | — | — | **72.5%** ❌ |

**为什么内置 / 我的仪表盘 panel 不受影响**：单设备 / 设备组 panel 行维度是"预定义"的，预聚合阶段就按正确口径从原始计数器算好结果——不存在"跨设备再聚合"。

**代价与权衡**：
- 读取量大（输出量的几十到几千倍）—— 任务式不阻塞 HTTP，**几十分钟级耗时可接受**（已与用户确认），因为用户不必等待，可切走做其他事
- 不复用预聚合 = 牺牲一部分性能换正确性，符合"商用级品质"原则
- 简化架构：自定义聚合只有一个数据源（pm_metrics），没有"按粒度选表"的分支

**算法方向**（细节移 plan）：任务执行器对时间窗内自选 N 设备的 15min 原始计数器做一次扫描，按目标粒度 GROUP BY + 按 KPI 公式计算 → 写入快照表。KPI 公式定义需在系统中可查（参考 G2 / G3 的 KPI 字典）。

### 多粒度多选

**规则**：

```
任务定义时：输出粒度多选（至少一个，默认勾选小时）
任务跑完：    每个选中粒度独立产出一份数据（同一 job，按 granularity 列区分）
快照页查看：  按粒度切换 tab / 下钻，无需重跑
```

**为什么多选**：
- 用户常见诉求"既要趋势细节（小时），又要日报对账（天）"——拆成两个独立任务跑很笨
- 一次扫描覆盖多视角，省任务次数
- 下钻天然支持：日级概览 → 点开某天看小时分布，切 tab 即可

**算法**：每次扫描 15min 原始一遍，按每个目标粒度独立 GROUP BY + KPI 公式算（公式类 KPI 不能"先算小时再 SUM 到天"，必须从原始算）。多粒度共享一次扫描，比 N 个独立任务省 N-1 次 I/O。

### 持续模式 cron 调度

**cron 调度时刻沿用 §4.5 G5 规则**——continuous 自定义聚合本质是用户级的 G5 类预聚合，与系统级 G5 cron 共用同一套调度时刻 + 容忍延迟约定。

| 最细粒度 | 调度表达式 | 容忍延迟 | 聚合范围 |
|---------|-----------|---------|---------|
| 15min | `5,20,35,50 * * * *`（与 G5 同步对齐）| 5 分钟 | 上一个 15min 桶 |
| 小时 | `5 * * * *`（每小时 05 分）| 5 分钟 | 上一个完整小时 |
| 日 | `5 0 * * *`（每天 00:05）| 5 分钟 | 昨天整天 |
| 周 | `10 0 * * MON`（周一 00:10）| 10 分钟 | 上周整周 |
| 月 | `15 0 1 * *`（每月 1 日 00:15）| 15 分钟 | 上月整月 |

cron 频率与所选粒度集中**最细的一档**对齐（每周期一次扫描共算所有选中粒度）。容忍延迟用于等"晚到的 15min PM 文件入库"，与 G5 同语义。延迟超出容忍的桶补跑机制由 **PM 模块自管**（参考 §4.5 G5 "指定桶时间区间补跑"思路实现；不复用 §4.8 G8 框架，因为 G7 不进 `async_jobs`）。

**首次创建行为**：先回填 [startTime, now] 作为初始数据 → 之后每周期跑增量 [cursor, now]，cursor 记录"上次 cron 跑到的时间点"。

### 结果落地：pm_adhoc_aggregation_results

**问题**：自定义聚合任务跑完后，结果是丢弃还是落地？

**结论**：**必须落地存储为快照**，理由：
1. 任务可能跑几分钟，用户大概率切走再回来——结果丢了体验差
2. 进度条达 100% 后没东西可取就尴尬
3. 分享 URL / 导出 Excel 都要基于一份"快照"
4. 重新打开"我的任务"列表要能直接展示历史结果，不重跑

**专门表**（不用 JSON 列、不用 MinIO）：

| 方案 | 为什么不选 |
|------|----------|
| `pm_tasks.result_json` | 大 JSON 列 toast 拖 PG 性能；前端在快照页要切粒度 / 排序 / 筛选 / 翻页，JSON 不能局部查询 |
| MinIO 对象 | 适合整文件 download；但快照页的"按粒度查 / 按时间筛 / 按 KPI 排序"做不到 |
| **专门表（采用）** | 结构化查询；可走 TimescaleDB hypertable，分区 + 索引天然支持；流式导出 Excel / CSV 不撑内存；"另存为 panel" 顺（结构已表化，元数据迁移即可）|

**字段维度**（具体 schema / 索引 / 分区策略移至后续 plan）：

```
pm_adhoc_aggregation_results = (job_id, granularity, device_sn, time_bucket, metric_name, metric_type, value)
```

走 TimescaleDB hypertable 按 time_bucket 分区，复用 pm_metrics 已有的分区 / 压缩策略。

**pm_tasks 与结果表的分工**：

- `pm_tasks` 存任务元数据：复用现有字段（task_type / device_sns / kpi_codes / time_range / status / progress / creator / created_at）+ 本次扩展（`granularity` 字段单值改为多值数组 / 新增 `mode` / `cursor` / `result_rows_count`）；保留期参数走全局，任务表无过期字段
- `pm_adhoc_aggregation_results` 存数据本身
- 关系：1 条 pm_tasks 记录（task_type='adhoc_aggregation'）↔ N 行结果（按 granularity × device_sn × time_bucket × metric_name 展开，与 §3.1 表设计约定一致）

**为什么是"快照"而非"实时"**：
- 自定义聚合通常是**分析场景**（"上月这 50 台设备 KPI 表现"）——分析对象是过去时段，数据不会变
- 真有"我要最新"诉求 → 点刷新重跑（成本可控）
- 这区分了"仪表盘 panel（实时查询）" vs "自定义聚合结果（快照）"两种心智

### 全局保留期：系统统一管理

**规则**：所有自定义聚合（oneshot + continuous）共用一个系统级保留期参数 `pm.adhoc_retention_days`，**默认 1 年**。用户创建任务时无保留期字段、无决策点。

**为什么不做任务级保留期**：
- 用户决策点消失（"30 还是 90 天？"不再困扰用户）
- 任务表 schema 简化（无 retentionDays 列）
- UI 表单干净（一次性 / 持续都无保留期字段）
- "数据真丢"风险**有限**：`pm_metrics` 原始数据仍在 §4.2 G2 保留期内（默认 32 天，超期被自动清）；想重做老于 G2 保留期的 oneshot 无法做（属可接受代价——若有长期跟踪需求应建 continuous 而非依赖事后重做）

**清理路径**（数据走 drop_chunks，任务记录单独处理）：

| 对象 | 清理判据 | 清理动作 |
|------|---------|---------|
| `pm_adhoc_aggregation_results`（数据本身）| chunk 最新 time_bucket < now - globalRetention | TimescaleDB **`drop_chunks` 整块删**（高效；oneshot / continuous 数据共用一张 hypertable，不区分 mode 一起清）|
| `pm_tasks`（任务记录，endTime 有值）| `(time_range->>'endTime') + globalRetention < now`（包括 oneshot 任务 + 已停止的 continuous 任务，两者语义等同）| 行级 DELETE 任务记录（任务记录量小，无性能问题；目的是避免列表残留指向空数据）|
| `pm_tasks`（运行中 continuous，endTime 空）| 不自动清（cron 还在跑就有新数据）| 用户停止时填入 endTime → 进入上一条路径 |

**配置位置 / 谁能改**：
- 系统设置页（管理员角色权限）
- DB 存储（修改无需重启进程）
- 默认 1 年（覆盖运营商年度对账场景）

**改值后的行为**：

| 操作 | 行为 | UI |
|------|------|-----|
| 改大（1 年 → 2 年）| 已清不恢复；新数据按新值留 | 提示"已过期数据不可恢复" |
| 改小（1 年 → 90 天）| 下次 drop_chunks 立即多删一批 chunks + 清一批已结束任务记录（oneshot + 已停止 continuous）| **强 confirm**："将清理 N 天前的所有数据 + 约 X 个已结束任务记录，不可恢复，确认？" |

预删数据范围 / 任务数由后台先估算（按 chunk 边界 + pm_tasks COUNT），confirm 弹窗带数字展示。

**长期保留的补偿路径**（针对失去任务级保留期控制的用户）：
- **想长期跟踪某 N 设备** → 建一个 continuous 任务（持续追加快照），并基于该任务的快照创建 panel 长期查看（任务与 panel 独立生命周期，见下节"与 panel 的关系"）
- **想留作历史档案** → 导出 Excel / PDF 落地到本地或外部系统
- **想看老数据** → 重新跑 oneshot（pm_metrics 原始数据始终在）

**实施要点**（细节移至后续 plan）：
- 清理 cron 日级跑：数据用 `SELECT drop_chunks('pm_adhoc_aggregation_results', INTERVAL 'globalRetention')` 整块删（chunk 时间边界自然对齐 globalRetention）；可直接走 TimescaleDB 自带的 `add_retention_policy` 自治
- 任务记录用一条 SQL 按 `task_type='adhoc_aggregation' AND (time_range->>'endTime') IS NOT NULL AND (time_range->>'endTime')::timestamptz + globalRetention < now()` 清理——覆盖 oneshot + 已停止的 continuous（运行中 continuous 的 endTime 为空，自然不被命中；pm_tasks 量小，无需分批）
- 跨保留期查询的边界处理：panel 时间窗 / 跨快照对比 → 仅返回有数据部分 + UI 横条提示"超出保留期的数据已清理"
- 审计：`pm_tasks` 加 `last_cleanup_at`；配套 Prometheus metric（`pm_adhoc_dropped_chunks_total` / `pm_adhoc_tasks_cleaned_total`）

**不做的（YAGNI）**：
- 任务级保留期 override（"这个任务我想留更久"）
- 按 KPI 类型差异化保留
- 冷数据归档到 MinIO
- 多保留窗口（细粒度短保留 + 粗粒度长保留）

### 生命周期操作

| 操作 | 行为 |
|------|------|
| **停止** | 停 cron，把停止时刻写入 `time_range.endTime`，从此**语义等同 oneshot**（按 endTime + 全局保留期自动清）；想恢复持续跟踪 → 复制为新任务 |
| **删除** | 注销 cron + 删全部历史数据 |
| **改设备 / KPI / 粒度** | 不支持原地改；引导用户复制为新任务（避免 cursor 与历史数据语义混乱）|

### 与 panel 的关系（独立生命周期）

- **panel ≠ pm_tasks**：panel 是仪表盘上的视图配置；pm_tasks 是任务记录 + 数据源。两者**完全独立**，各自管理生命周期，互不影响。
- **基于 task 创建 panel**：在快照页点"另存为我的仪表盘 panel" → 创建新 panel，配置上引用 task 的 `(job_id, granularity)`；panel 自此独立运转，渲染时从 `pm_adhoc_aggregation_results` SELECT。
- **删 panel**：仅删 panel 记录；**不动 pm_tasks**，不停 cron，不删数据——任务继续按自己的生命周期运转（continuous 持续追加 / oneshot 静止）。
- **删 task**：panel 仍存在；panel 加载时若源 task 已不存在或数据已被 `drop_chunks` 清，UI 显示"数据源已清理或被删除"提示，引导用户重建关联或删 panel。
- **task 自动清理触发**：仅取决于全局保留期 + endTime 是否有值（见上节"全局保留期"），**与 panel 是否引用无关**——不引入引用计数管理。

### UI 集成点

- 在 §4.6 仪表盘工具栏 / 我的仪表盘区加"+ 自定义聚合"入口
- 自定义聚合表单加 toggle `[一次性 / 持续]`，默认一次性
- 选"持续"时：endTime 输入框隐藏；"cron 频率"作为只读说明显示（联动粒度）
- 表单**不出现**保留期字段；统一在系统设置页（管理员）控制
- 粒度选择改为 checkbox 多选；至少选一个；多选时提示"任务数据量随粒度数线性增长"
- "我的任务"列表区分一次性 / 持续（图标 + 持续任务显示运行状态：运行中 / 已停止）

### 进度上报

后端每完成一个聚合单元（一个时间桶 or 一批设备）就 UPDATE `pm_tasks.progress` JSON，前端用轮询拉进度（SSE 看后续基础设施可不可用，本期不强制）。

### 结果消费

- 完成后用户在"我的任务"列表点入
- 复用 §4.6 仪表盘 panel 渲染组件——快照结构与 G5 聚合表对齐
- 支持导出 Excel / PDF（与 §4.6 仪表盘导出同口子）；分享走 §4.6 点对点用户分享机制

### 异常

- 任务失败：记录失败原因（缺采、设备未匹配产品、KPI 不存在等）；UI 展示
- 任务取消：用户主动取消 → 中断执行 + 清理中间状态
- 进度卡住：worker 心跳超时机制由 PM 模块自管（不复用 §4.8 G8 通用监控）；细节移 plan

> 任务并发上限、单任务设备数 / 时间窗上限、cron 表达式细节、SSE 是否启用、心跳阈值属实施细节，本设计不固化。

---

## 4.8 通用任务框架（G8）

### 定位

把 G5 cron 触发的聚合实例、未来新增的长流程异步计算任务，统一进同一张 **`async_jobs`** 表 + 同一套**持久化 / 重启续跑 / 心跳 / 幂等**机制。

> **G7 自定义聚合任务不在 G8 范围内**：G7 自定义聚合改走 PM 模块的 `pm_tasks` 表（per-module 模式，与 `mml_tasks` / `upgrade_tasks` / `backup_tasks` 等并列），不进 `async_jobs`。原因：G7 是"用户驱动 + 业务领域绑定"，`pm_tasks` 已有 device_sns / kpi_codes / time_range / granularity 等 PM 专属字段；G8 `async_jobs` 更适合"系统驱动 + 跨模块通用"的任务。详见 §4.7。

### 职责边界

| 进 `async_jobs`（A 层，本设计承载） | per-module 独立表（B 层，业务领域绑定） |
|---|---|
| G5 cron 聚合任务实例（每次触发生成一条 job） | `pm_tasks`（PM 自定义聚合 / 未来 PM 后台任务，见 §4.7） |
| 未来：批量报表生成、离线统计、批量诊断等 | `device_tasks`（RPC 派发，Redis 双写）|
|  | `upgrade_tasks` / `upgrade_sub_tasks`（固件升级）|
|  | `backup_tasks`（备份） |
|  | `mml_tasks`（MML 批量） |

### `async_jobs` 表字段

| 字段 | 说明 |
|------|------|
| `id` | 主键 UUID |
| `type` | 任务类型（`kpi_rollup_hourly` / `kpi_rollup_daily` / `kpi_rollup_weekly` / `kpi_rollup_monthly` / `kpi_rollup_group_*` 同上 / 未来扩展。**注**：自定义聚合 `adhoc_aggregation` 不在此表，走 `pm_tasks`，见 §4.7） |
| `payload` | JSON 输入参数（业务自定义） |
| `status` | `pending` / `running` / `succeeded` / `failed` / `cancelled` |
| `progress` | JSON：当前阶段、已完成 / 总桶数、已完成 / 总设备数 |
| `worker_id` | 占有当前任务的 worker 实例 ID（重启后用于识别遗留任务） |
| `last_heartbeat` | 最后心跳时间 |
| `result_pointer` | 结果数据库主键或 MinIO 对象 key（按需） |
| `owner_user_id` | 用户任务关联（cron 实例为空） |
| `created_at` / `started_at` / `ended_at` | 生命周期时间戳 |
| `retry_count` / `max_retries` | 重试计数与上限 |
| `error_message` | 失败原因（可选） |

**状态机**：`pending → running → succeeded / failed / cancelled`

### 持久化与重启续跑设计

1. **任何阶段都写库**
   - 任务创建 → 立即写 `pending`
   - worker 拉起 → 写 `running` + `started_at` + `worker_id` + `last_heartbeat`
   - 进度变化 → UPDATE `progress` + `last_heartbeat`
   - 结束 → 写 `succeeded` / `failed` + `ended_at` + `result_pointer`

2. **心跳 + 僵尸检测**（默认值，挂 sys_configs 可调）
   - worker 心跳间隔：**60 秒**
   - 僵尸判定阈值：**5 分钟**（5 倍心跳间隔，给长 SQL 足够缓冲，避免误判）
   - 监控线程定时扫 `status='running' AND last_heartbeat < NOW() - 5min` → 判定 worker 挂了 → 重置为 `pending`（若任务幂等可续跑）或标 `failed`（retry_count++ 进重试队列）

3. **幂等性 + 续跑**
   - G5 聚合任务天然幂等：按 `(time_bucket, device_sn / device_group_id, metric_type, metric_name)` UPSERT，重跑覆盖结果
   - 续跑策略：重置 `pending` 后由其他 worker 拉起重头跑（聚合任务跑得快，分钟级，重头跑可接受）
   - 不实现"断点续传"——简化设计

4. **cron 错过触发时点的补跑**
   - robfig/cron/v3 进程内触发器不持久化，进程死了会错过下次触发
   - 每个 cron 任务记录"最后成功触发的桶时间"在 `async_jobs_cron_state` 单独小表（一行一种 cron 类型）
   - worker 启动时检查：上次触发时间 vs 当前时间，期间漏触发的桶**自动补跑**——生成对应 `async_jobs` 记录入队
   - 同一入口还供运维手动重算（HTTP / gRPC / CLI 任选实现）；§4.5 提到的"指定桶时间区间重算"需求即由此承担

5. **分布式锁（多 worker 部署）**
   - 任务从 `pending` 切到 `running`：用 `SELECT ... FOR UPDATE SKIP LOCKED` 加行锁，防两个 worker 抢同一条
   - cron 触发本身：用 PG advisory lock 防多实例重复触发同一桶

**进度上报机制**
- worker 在执行循环中每 N 秒（或每完成一个聚合单元后，取近者）调一次 `UpdateProgress(job_id, progress_json)`
- 前端轮询 GET `/api/v1/async-jobs/{id}` 读最新状态 + progress

**取消与清理**
- 用户取消：API 标 `cancelled` → worker 下一次心跳前读到状态变化 → 主动中断 + 清理中间状态
- TTL 清理：`async_jobs` 的清理由后台 cron 任务（自身也是一条 async_job）扫 `status in (succeeded, failed, cancelled) AND ended_at < NOW() - ttl` → 删除 + 关联的 MinIO 对象
- G5 cron 实例 TTL 较长（与 G2 数据保留期对齐，方便审计）
- G7 自定义聚合的清理走 §4.7 全局保留期（drop_chunks + 任务记录单删），不在本节范围

**Prometheus 指标**
- 任务队列深度（按 type 分桶）
- 平均执行耗时
- 失败率
- 僵尸任务计数

> 任务清理 cron 表达式、Prometheus 标签命名、心跳调用频率细节、SSE 是否启用属实施细节，本设计不固化。

---

# 5. 改造影响范围

| 模块 | G1 路由 | G2 保留 | G3 聚合 | G4 时间窗 | G5 预聚合 | G6 前端 KPI 图表 | G7 自定义聚合 | G8 通用任务框架 |
|------|---------|---------|---------|----------|----------|------------------|------------|----------------|
| `internal/pm/collector/` | ✅ 路由插入点 | — | — | ✅ 解析 start/end + 填 ingest_time | — | — | — | — |
| `internal/pm/kpi/` | ✅ 引擎签名扩展 | — | ✅ 主战场 | ✅ 写 pm_metrics 时填三时间 | — | — | — | — |
| `internal/pm/aggregation/` | — | — | ⚠️ 删除物化视图依赖 | ⚠️ 查询参数按 end_time 还是 ingest_time 由调用方选 | ✅ 主战场：四级 cron 聚合任务 ×（设备 + 设备组）= 8 个任务 + 接口路由（粒度 + 行维度→表） | ⚠️ QueryAggregated 增加 device_group 维度路由 | ✅ 主战场：oneshot / continuous 执行器 + 进度上报 + 写 `pm_adhoc_aggregation_results`；任务走 `pm_tasks` 表 + PM 自管 cron 补跑 + 心跳超时（不复用 G8）| ⚠️ G5 聚合 cron 实例改为写 async_jobs；执行体接受 G8 心跳 / 取消信号（G7 不在范围内）|
| `internal/pm/indicator/`（指标管理） | ✅ 查询入口 | — | ✅ statis_type 启用 | — | — | — | — | — |
| `internal/product/` | ✅ 路由消费方 | — | — | — | — | — | — | — |
| `internal/admin/`（sys_configs / user_preferences） | — | ✅ 10 条保留期 KV（5 级粒度 × 压缩/保留） | — | — | — | ✅ user_preferences（KPI 卡片配置）+ dashboards / panels / shares 系列表 | ✅ 1 条 `pm.adhoc_retention_days` KV（全局保留期） | ✅ 2 条 KV（心跳间隔 60s / 僵尸阈值 5min） |
| `internal/topology/` | — | — | — | — | — | ✅ 设备组 API 供报表范围筛选复用（沿用现有，不动 schema） | — | — |
| `internal/task/` | — | — | — | — | — | — | — | ✅ 不动（B 层，与 ACS 会话耦合） |
| `internal/job/`（新增） | — | — | — | — | — | — | — | ✅ 主战场：async_jobs 通用框架 + 心跳 / 僵尸检测 / 补跑 / 分布式锁 |
| `migrations/` | — | ✅ hypertable 装 TS policy（15min + hourly + group_hourly）；普通表的 retention 走 cron | ✅ `DROP MATERIALIZED VIEW pm_counters_hourly` + 合并 pm_counters / kpi_values 为 pm_metrics + 加 metric_type 字段 | ✅ 加 start/end/ingest_time 字段 + 重建分区列 + 业务键改 device_sn | ✅ 新增 8 张聚合表：设备级 4 张 + 设备组级 4 张；`*_hourly / *_group_hourly` 是 hypertable，其余是普通表 | ✅ 新增 user_preferences + dashboards + panels + 分享表（dashboard_shares / snapshot_shares 或合表） | ✅ 扩展 pm_tasks 表（加 mode / cursor / result_rows_count；granularity 改多值；新增 task_type='adhoc_aggregation'）+ 新建 pm_adhoc_aggregation_results hypertable | ✅ 新增 async_jobs + async_jobs_cron_state 两张表 |
| 前端指标库 UI | — | — | ✅ statis_type 编辑入口 | — | — | — | — | — |
| 前端 `system/SystemConfig/BasicSettings.tsx` | — | ✅ "PM 数据保留"分组 5 行 × 2 列（5 级粒度的压缩天数 / 保留天数） | — | — | — | — | — | — |
| 前端 PM 模块 | — | — | — | ✅ 时间轴 end_time / ingest_time 切换、缺采显示、延迟角标 | ✅ 粒度切换映射聚合表 | ✅ 主战场：仪表盘 + 面板 + KPI 卡片配置持久化 + 分享 / 协作 UI | ✅ "+ 自定义聚合"入口（含一次性 / 持续 toggle）+ "我的任务"列表（含进度条、状态、停止 / 删除）| — |
| 监控 / Prometheus | ✅ skip 计数 | ✅ 容量告警 | ✅ 聚合切换告警 | ✅ 上报延迟阈值告警 | ✅ rollup 任务成功率 / 耗时 / 落后桶数 | — | ✅ 任务队列深度 / 平均耗时 / 失败率 | ✅ 任务总览（含僵尸计数 / 重试率 / 补跑次数） |
| 文档 / 验收 | ✅ E2E 用例 | ✅ DoD | ✅ Runbook | ✅ E2E 校验三字段 | ✅ E2E 校验自然桶聚合结果 + 重算入口 | ✅ E2E + Playwright（三 tab 联动、模板渲染、配置持久化） | ✅ E2E + Playwright（提交 / 进度 / 结果渲染） | ✅ E2E 校验 kill -9 续跑 + 多 worker 不重复 + cron 错过补跑 |

---

# 6. 迁移动作

> 项目处于初期开发阶段，未上生产。无需双轨 / 灰度 / 废弃路径，直接切换；表结构调整随迁移文件一次性生效。

本期 G2 + G4 + G5 的 schema 部分**合到同一份迁移文件**，一次性完成：

- **合并 `pm_counters` + `kpi_values` → 新建 `pm_metrics`（原始 15min）**：
  - 项目未上线，**不做数据搬运**：迁移文件里直接 `DROP TABLE pm_counters` + `DROP TABLE kpi_values` + `CREATE TABLE pm_metrics`，一次完成；不写 `INSERT INTO pm_metrics SELECT ... FROM ...` 这种数据迁移 DML
  - `pm_metrics` schema：业务键 `device_sn VARCHAR(64)` + `cell_id` + `metric_type VARCHAR(16)`（`counter` / `kpi`）+ `metric_name` + 三时间字段 `start_time` / `end_time` / `ingest_time` + `value`
  - hypertable 分区列为 `end_time`（不再保留 `time`）
  - compression policy = 7 天，retention policy = **32 天**（G2 默认；保证月桶聚合所需的最远 31 天数据仍在；UI 可改但要 ≥ 32）
- **新增 8 张聚合表（G5）**：
  - 设备级 4 张：`pm_metrics_{hourly,daily,weekly,monthly}`
  - 设备组级 4 张：`pm_metrics_group_{hourly,daily,weekly,monthly}`
  - `*_hourly` / `*_group_hourly`：hypertable（chunk_time_interval = 1 day），挂 TS compression + retention policy
  - 其余（daily / weekly / monthly 各一对）：普通 PG 表（不分 chunk），靠 G5 cron 任务做 retention 清理
- **物化视图 `pm_counters_hourly`**：`DROP MATERIALIZED VIEW`（被 G5 应用层聚合表 `pm_metrics_hourly` 取代——名字不同避免误解）
- **sys_configs 种子**：5 级 × 2 个 key = 10 条 KV 记录默认值
- **指标元数据**：`perf_indicators_*.statis_type` 已就位，KPIEngine 切到消费这套表即可，不动 schema
- **老 `kpi_definitions` 表 + 适配器代码硬编码 KPI 列表**：本期直接下线（drop 表 + 删代码），不留废弃路径
- **`QueryAggregated` 接口**：签名扩展支持 `device_group` 维度筛选（供 G6 panel 使用），底层按调用方粒度路由到对应聚合表（短窗口仍查 pm_metrics）
- **G6 新增表**：
  - `user_preferences`（user_id + 偏好 JSON，承载 KPI 卡片配置；JSON 按制式分键 `{ enb: [...], gnb: [...], gsm: [...] }`；后续其他用户偏好可复用同张表）
  - `dashboards` + `panels`（取代原 `report_templates`，承载"仪表盘 + 面板"两概念模型；系统内置仪表盘 + 我的仪表盘共用此 schema，详见 §4.6）
  - `dashboard_shares` + `snapshot_shares`（或合并为 `shares + entity_type` 单表，二选一移至 plan）——点对点用户分享
- **G7 复用 pm_tasks 表（per-module）**：
  - `pm_tasks` 表扩展：新增 `mode` / `cursor` / `result_rows_count` 字段；`granularity` 从单值改多值；task_type 新增枚举 `'adhoc_aggregation'`（占位枚举 `extraction` / `report` / `threshold-check` 本次清理废弃）
  - `pm_adhoc_aggregation_results` 新建（TimescaleDB hypertable，专门表存自定义聚合结果，详见 §4.7）
  - `sys_configs` 新增 1 条 KV：`pm.adhoc_retention_days`（全局保留期，默认 1 年）
- **G8 新增两张表**：
  - `async_jobs`（通用任务表：id / type / payload / status / progress / worker_id / last_heartbeat / result_pointer / owner_user_id / 生命周期时间戳 / 重试计数）
  - `async_jobs_cron_state`（cron 触发状态：type + 最后成功触发时间，用于重启后补跑）
  - sys_configs 新增 2 条 KV：心跳间隔（默认 60s）+ 僵尸阈值（默认 5min）

---

# 7. 验收（DoD 草案）

- [ ] **G1**：选一个 productClass，UI 上编辑其产品的 indicator_platform → PMCollector 重新路由 → pm_metrics 写入的 KPI 名集合发生预期变化
- [ ] **G1**：未匹配产品 / 空 platform 的设备触发 `pm_kpi_skip_total` 计数器 + Warn 日志
- [ ] **G1**：老 `kpi_definitions` 表 + 适配器硬编码 KPI 列表已下线，全链路只剩 `perf_indicators_*` 一条路
- [ ] **G2**：5 级粒度共 5 张表（合并后）都装上 retention 机制——hypertable（15min + hourly）用 TS policy，普通表（daily / weekly / monthly）用 G5 cron 清理任务；E2E 验证每级各自的 N+1 天数据消失
- [ ] **G2**：系统设置页"PM 数据保留"分组显示 5 行 × 2 列（每级粒度的压缩天数 + 保留天数），可编辑保存
- [ ] **G2**：保存某级 → 服务端立即重置该级对应表的 compression + retention policy（不重启）→ 新值生效
- [ ] **G3**：把某个 KPI 的 statis_type 从 sum 改为 avg → 下一次计算结果切换到 avg，无需重启
- [ ] **G3**：物化视图 pm_counters_hourly 已删除；counter 趋势查询接口（QueryAggregated）接口契约不变，底层在 G5 落地后改为按粒度路由
- [ ] **G4**：pm_metrics 每行三字段（start_time / end_time / ingest_time）非空写入；E2E 校验 start_time = parser 解的 `fileHeader/measCollec/@beginTime`、end_time = parser 解的 `fileFooter/measCollec/@endTime`、ingest_time 落在合理偏差内
- [ ] **G4**：一份覆盖多小区的 PM 文件解析后 pm_metrics 按 (device_sn, cell_id, metric_type, metric_name, end_time) 唯一；counter 行数 = counter 数 × cell 数；KPI 行数 = KPI 数 × cell 数
- [ ] **G4**：pm_metrics 系列共 9 张表（原始 1 + 设备级 4 + 设备组级 4）全部用 device_sn 关联，不存 device_id
- [ ] **G4**：合并表 `pm_metrics` 含 `metric_type` 字段；旧表 `pm_counters` / `kpi_values` 已 DROP；counter 与 KPI 名字命名空间不冲突（XML 指标库入库时校验）
- [ ] **G4**：上报延迟监控（ingest_time − end_time）有 Prometheus 直方图 / 计数器
- [ ] **G5**：四级 × 两维度 = 8 个 cron 任务（设备级 + 设备组级 × 小时 / 日 / 周 / 月）按时跑；设备组任务在对应设备级任务之后启动；E2E 验证整点桶范围正确（`[H:00, H+1:00)`、`[00:00, 24:00)` 等）
- [ ] **G5**：每级聚合任务两步执行——
  - 步骤 A：从 pm_metrics 15min 原始表聚合 counter 写 pm_metrics_<G>（设备维度）/ pm_metrics_group_<G>（设备组维度，JOIN device_group_members）
  - 步骤 B：sum/avg/max 类 KPI 从 pm_metrics 15min 原始表直接聚合；pct 类 KPI 从同级 counter 聚合表（设备级或设备组级）取 counter 值代入公式
- [ ] **G5**：E2E 验证 pct 类 KPI 的日桶值 = sum(分子 counter) / sum(分母 counter) × 100；同时校验若按"对 kpi 值直接 AVG"会得到错误结果（证明 pct 走特殊路径的必要性）
- [ ] **G5**：手动重算入口（指定桶时间区间，针对晚到数据 / 补传场景）可用，覆盖设备级 + 设备组级；rollup 任务 Prometheus 指标（成功率 / 耗时 / 落后桶数）齐全
- [ ] **G5**：报表 / 趋势查询接口按粒度 + 行维度路由到对应聚合表（设备维度 → `pm_metrics_<G>`，设备组维度 → `pm_metrics_group_<G>`）
- [ ] **G6**：性能管理菜单下单 tab"性能查看"；左侧仪表盘列表 + 右侧仪表盘渲染；共享筛选条（时间窗 / 设备组 / 设备多选）在 panel 间共享
- [ ] **G6**：制式（GSM / ENB / GNB）作为顶层强制单选；切换制式时整页刷新——KPI 卡片配置、仪表盘、设备组列表、筛选状态都按制式各自独立；选择持久化到 `user_preferences`
- [ ] **G6**：系统内置仪表盘可见且 readonly（GSM / ENB / GNB × 全网概览 / 日报 / 周报 / 月报）；用户"另存为派生"可 fork 一份成自己的仪表盘（派生后独立演化，不同步原版）
- [ ] **G6**：我的仪表盘可新建 / 保存 / 删除；panel 类型覆盖 折线图 / 柱状图 / 表格 / KPI 卡片 / TopN / 数值大屏
- [ ] **G6**：panel 输出粒度支持多选（至少一项，默认勾选小时）；同一 job 内按粒度 tab 切换 / 下钻无需重跑
- [ ] **G6**：panel 数据加载分流正确——行维度 ∈ {设备 / 小区 / 已有设备组} 走同步 SELECT；行维度 = 自选 N 设备走 `pm_tasks` 任务式
- [ ] **G6**：折线图 panel 对比双模式可用——模式 A（同快照内多组对比，本周一 vs 上周一）；模式 B（跨快照对比，3 月快照 vs 4 月快照）；维度交集校验 + UI 提示
- [ ] **G6**：点对点分享可用——分享给指定用户（viewer-only，不区分 editor）;"分享给我的"子区显示其他用户分享的仪表盘 / 快照；owner 可撤销；分享 / 撤销有审计日志
- [ ] **G6**：KPI 卡片配置（添加 / 移除 / 排序）持久化到 `user_preferences`（按制式分键）；切制式显示对应卡片集
- [ ] **G6**：pct 类 KPI 显示带 % 单位 + 阈值线；时间轴可切 `end_time` / `ingest_time`；缺采点显示"缺采"不画 0；超延迟阈值的点有角标
- [ ] **G6**：仪表盘导出 Excel / PDF；URL 分享可复现筛选；不做邮件订阅
- [ ] **G7**：从仪表盘工具栏可进入"+ 自定义聚合"入口；选 N 个设备 + KPI 集 + 时间窗 + 粒度多选后提交，返回 task_id；制式与所在仪表盘一致
- [ ] **G7**：任务模式双选可用——oneshot（startTime + endTime 必填）/ continuous（endTime 留空，cron 周期跑）；持续模式表单**不显示**保留期字段（统一走全局）
- [ ] **G7**：任务状态机正确（pending → running → completed / failed / cancelled）；进度条实时刷新；用户取消可中断
- [ ] **G7**：cron 调度时刻与 §4.5 G5 对齐（小时 `5 * * * *` / 日 `5 0 * * *` 等）；cron 频率与所选最细粒度对齐
- [ ] **G7**：完成后结果可视化走 G6 panel 组件；快照页支持按粒度切换 tab；可导出 Excel / PDF；分享走 G6 点对点用户分享
- [ ] **G7**：全局保留期（`pm.adhoc_retention_days`，默认 1 年）配置可改；改大不恢复已清，改小弹强 confirm 显示预删数据范围
- [ ] **G7**：自动清理验证——`pm_adhoc_aggregation_results` 走 `drop_chunks` 整块清；`pm_tasks` endTime 有值且超期的任务记录被一条 SQL 清；运行中 continuous（endTime 空）不被清
- [ ] **G7**：停止 continuous 任务时把停止时刻写入 `time_range.endTime`，此后语义等同 oneshot 走自动清；删除则注销 cron + 删全部历史数据
- [ ] **G7**：panel ↔ task **独立生命周期**——删 panel 不动 task；删 task 后 panel 加载显示"数据源已清理或被删除"提示
- [ ] **G7**："我的任务"列表只显示当前用户的任务；区分一次性 / 持续（图标 + 状态：运行中 / 已停止）；admin 可看全部
- [ ] **G7**：cron 错过触发时点（worker 停机覆盖触发时间）后，启动时自动补跑漏掉的桶（PM 自管，参考 §4.5 G5 思路实现，不复用 G8）
- [ ] **G8**：`async_jobs` 表承载 G5 cron 实例 + 未来批量计算任务（G7 自定义聚合不在范围内，走 `pm_tasks`，见 §4.7）；状态机 pending → running → succeeded / failed / cancelled
- [ ] **G8**：重启续跑可用——kill -9 worker 进程后，僵尸任务在阈值（默认 5min）后被重置为 pending，由其他 worker 接管完成
- [ ] **G8**：多 worker 部署下同一任务不会被两个 worker 同时执行（`FOR UPDATE SKIP LOCKED` 验证）
- [ ] **G8**：cron 错过触发时点（worker 停机覆盖触发时间）后，启动时自动补跑漏掉的桶；`async_jobs_cron_state` 表正确记录最后成功触发时间
- [ ] **G8**：心跳间隔与僵尸阈值通过 sys_configs 可调；改后立即生效
- [ ] **G8**：Prometheus 任务总览指标齐全（队列深度按 type 分桶 / 平均耗时 / 失败率 / 僵尸计数 / 补跑次数）
- [ ] 八项均覆盖到 `scripts/e2e_verify.sh` + Playwright（G6 / G7 含 UI 交互断言；G8 含进程重启续跑断言）
- [ ] 文档更新：本设计 → 实施计划 → backlog 登记 → release-gate.md 检查项

---

# 8. 风险

| ID | 描述 | 缓解 |
|----|------|------|
| R-PMK-02 | G3 后某 KPI 算法切换（statis_type / arithmetic 变更）导致历史趋势断层 | 指标定义表 `perf_indicators_*` 保留变更日志（updated_at + updator）；前端在异常断点处可点查"该指标元数据何时被改过" |
| R-PMK-03 | retention 默认值与实际业务回溯 / 同比需求冲突 | 保留期挂到系统设置页可改；不写死在代码 |
| R-PMK-04 | G5 cron 任务失败 / 多 worker 重复执行导致聚合表数据缺失或重复 | cron 任务带分布式锁；失败有 Prometheus 计数器 + 重试；提供"指定时间区间手动重算"入口 |
| R-PMK-05 | indicator_platform 软引用空 / 指向不存在的 platform | 路由层抛错 + skip 计数器；开发期直接修种子数据 / 产品装配件 |
| R-PMK-06 | 上报延迟超过 G5 cron 的容忍窗口，数据漏聚合 | 上报延迟 Prometheus 直方图监控（G4）；超容忍窗口触发"重算最近 N 个桶"任务 |
| R-PMK-07 | G3 合并表后 counter 名 / KPI 名命名空间冲突（同名既是 counter 又是 KPI） | XML 指标库 + 配置入库时校验唯一性；冲突报错阻止加载 |
| R-PMK-08 | G6 仪表盘"设备规模过大"——用户选了上万设备 + 月粒度 + 多 KPI，查询慢、导出大 | UI 软提示"建议改用设备组汇总 / 设备组聚合表"；不强制限额 |
| R-PMK-09 | G7 任务无限堆积——用户大量提交自定义聚合任务把 worker 池打满 | sys_configs 暴露"单用户并发任务上限"（默认 3，可调）；超出排队；后端 Prometheus 队列深度告警 |
| R-PMK-10 | G8 cron 补跑期间产生大量积压任务（如停机一周后启动），冲击 worker | 补跑生成的 async_jobs 按桶时间顺序入队；worker 池并发上限保护；监控积压队列长度 |

---

# 9. 切分（建议进 backlog 的子任务）

> 本设计文档不固定具体编号，由 PgM 在 Triage 时分配。建议八条独立任务：

1. **G8 通用任务框架**（**P0**，async_jobs + async_jobs_cron_state 两张表 + 心跳 / 僵尸检测 / 补跑 / 分布式锁；G5 / G7 的前置依赖）
2. **G4 时间窗 + 入库时间存储**（P1，schema 加 3 字段 + parser / collector 改造 + 上报延迟监控）
3. **G2 保留策略 + 系统设置 UI**（P1，5 级 policy + 10 条 sys_configs KV + 前端 5 行 2 列分组）
4. **G3 动态聚合元数据 + 删除物化视图 + 合并 pm_counters/kpi_values + 下线老 KPI 表**（P1，KPIEngine 重构消费 `perf_indicators_*` + 指标 UI 编辑 statis_type）
5. **G5 自然日历桶预聚合**（P1，**8 张聚合表（设备级 4 + 设备组级 4）+ 8 个 cron 任务（含设备组级，走 G8 async_jobs）+ 查询路由（按粒度 + 行维度）**）
6. **G1 产品路由**（P2，PMCollector / KPIEngine 按 product → indicator_platform 路由 + skip 计数）
7. **G6 前端 KPI 图表：仪表盘 + 面板模型**（P1，单 tab"性能查看" + dashboards / panels / 分享表 + KPI 卡片配置持久化 + user_preferences + 对比双模式 + 点对点用户分享 / 协作 + 派生 fork）
8. **G7 自定义聚合任务**（P2，扩展 pm_tasks 表 + 新建 pm_adhoc_aggregation_results hypertable + oneshot / continuous 双模式 + 全局保留期 drop_chunks 清理 + PM 自管 cron 补跑；结果渲染复用 G6 panel 组件）

排期建议：
- **schema 合并**：G8 加 2 张任务表、G3 合并 pm_counters / kpi_values 为 pm_metrics（含 metric_type 字段）+ 删物化视图、G4 加三时间字段、G2 装 policy、G5 建 8 张聚合表、G6 加 dashboards / panels / 分享表 / user_preferences、G7 扩展 pm_tasks + 新建 pm_adhoc_aggregation_results hypertable——全部合到**同一份迁移**，一次性完成
- **代码依赖链**：**G8 是 G5 / G7 的前置**（任务持久化 + 续跑 / 心跳能力先就位）；G3（消费 statis_type）→ G5（按 statis_type 聚合）→ G7（沿用 G5 同套聚合方法）；G1 与上述独立可并行；G6 依赖 G4（时间字段切换）+ G5（粒度路由），可在 G4 / G5 schema 落地后并行启动前端
- **任务并行度**：G4 schema 部分一定先于 G3 / G5 代码改造（其他模块写入三时间字段要前置）；G6 后端接口（QueryAggregated 扩展 device_group 维度、user_preferences / report_templates CRUD）与前端三 tab 可前后端并行；G7 后端任务执行器与前端任务列表 / 提交弹窗可前后端并行

---

# 10. 与现有文档的关系

- 数据字典平台化基础：`docs/design/参数-KPI-告警-整合设计方案.md`（T-0098 收官设计）
- PM 端到端缺口背景：`docs/project/backlog.md` T-0121
- productClass pattern 治理：`docs/project/backlog.md` T-0142
- 实施计划格式参考：`docs/project/参数-KPI-告警-整合-实施计划.md`
