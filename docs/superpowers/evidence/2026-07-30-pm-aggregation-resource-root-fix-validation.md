# PM 聚合与资源根治验收记录（2026-07-30）

## 验收结论

- 业务版本 `100.0.0-20260730-1528`（commit `abba2a55e`）已部署到
  `172.24.224.197`；资源告警标签热修复 commit `194c6e3fc` 已在不重启业务
  容器的前提下部署。
- 用户明确授权无备份清理测试环境，并明确接受 32 CPU / 32100 MiB 主机上的资源
  上限超配。仅删除了 OMC compose project 的五个数据卷，随后从空库完成安装、迁移
  和 20000 设备冷启动测试。
- 健康检查 73/73 通过；九个业务容器的 Docker CPU/内存限制、Go 运行时限制和
  PostgreSQL/Redis 参数与 `resources.env` 一致；容器 restart/OOM 均为 0。
- 20000 设备约 10 分钟完成注册。PM 与 PM_AGG 队列可以实时归零，聚合失败和 Redis
  写入错误均为 0。
- 第一个有效 16:15 数据批次已进入 Redis v2 和小时窗口；两个重点 KPI 的全部 Counter
  依赖在 16:00、16:15 两个槽均齐全。小时窗口仍处于自然进行中，按 12 分钟水位策略
  最早在 17:12 发布，所以当前没有最终小时、天、周结果是符合时间语义的，不是数据
  丢失。
- 仍有两个不能隐藏的外部/运行期问题：
  1. 测试源持续把 GSM 指标文件报给 LTE BLQ 设备，隔离与告警工作正常，但源数据
     必须由厂家/模拟器修正。
  2. COMMAND 队列消费正常且无 redelivery，但 16:32 的一分钟生产高峰超过消费能力，
     pending 短时继续增长；必须继续观察稳态并按真实业务节奏校准。

## 测试环境清理

在用户明确选择“无备份清空”后，先核对
`com.docker.compose.project=omcgo` 标签和实际挂载路径，只删除以下五个卷：

| Compose 卷 | 实际数据目录 |
| --- | --- |
| `omcgo_pgdata` | `/home/docker-data/volumes/omcgo_pgdata/_data` |
| `omcgo_tsdbdata` | `/home/docker-data/volumes/omcgo_tsdbdata/_data` |
| `omcgo_redisdata` | `/home/docker-data/volumes/omcgo_redisdata/_data` |
| `omcgo_natsdata` | `/home/docker-data/volumes/omcgo_natsdata/_data` |
| `omcgo_miniodata` | `/home/docker-data/volumes/omcgo_miniodata/_data` |

没有删除非 OMC 数据；`/opt/omc/data` 的模型库和字典由安装程序保留并刷新。

## 代码与数据库验证

通过：

- `cd omcgo && go build ./... && go test ./...`
- `cd omcmb && npm run typecheck`
- 完整资源契约 53/53
- 存储资源规划 19/19
- storage compose 152/152
- 配置升级与资源计划指标测试
- Grafana Dashboard JSON 解析
- Prometheus `promtool check rules`：24 条规则通过
- Prometheus resource-plan 规则单测通过

真实 PostgreSQL/TimescaleDB 验证：

1. `pm_aggregation_windows` 迁移在中断后可安全重入。
2. 两个独立连接用 `FOR UPDATE SKIP LOCKED` 能领取不同记录。
3. pgx + TimescaleDB hypertable 的 `ReplaceWindowResults` revision 替换和旧结果
   删除符合预期。

## 部署资源

完整 schema v2 资源文件已落地：

| 服务 | CPU | 内存 | 关键进程参数 |
| --- | ---: | ---: | --- |
| app | 2 | 1536 MiB | GOMAXPROCS=2，GOMEMLIMIT=1382 MiB |
| acs | 5 | 4096 MiB | GOMAXPROCS=5，GOMEMLIMIT=3686 MiB |
| worker | 8 | 2048 MiB | GOMAXPROCS=8，GOMEMLIMIT=1843 MiB |
| postgres | 10 | 7168 MiB | shared_buffers=1792MB，work_mem=8MB，max_connections=300 |
| postgres-tsdb | 16 | 7168 MiB | shared_buffers=1792MB，work_mem=8MB，max_connections=300 |
| redis | 2 | 8192 MiB | maxmemory=6144mb，noeviction |
| nats | 1 | 1024 MiB | max memory store=256 MiB |
| minio | 4 | 4096 MiB | — |
| web | 1 | 512 MiB | — |

业务容器上限合计 35840 MiB，监控固定上限约 4224 MiB，超过 32100 MiB 物理
内存；这是用户明确接受的测试超配，不应作为生产主机容量依据。

16:18 聚合高峰快照：

- 主机 CPU idle 70%～76%，iowait 1%。
- 可用内存 20.9 GiB，swap 使用约 23 MiB，采样期间无 swap in/out。
- app 约 137% CPU / 203 MiB；ACS 433% / 110 MiB；worker 38% / 210 MiB。
- 主 PostgreSQL 243% / 3.23 GiB；TimescaleDB 16% / 2.27 GiB。
- Redis 45% / 1.17 GiB；NATS 28% / 232 MiB；MinIO 9% / 1.04 GiB。
- 所有业务容器 restart=0、OOMKilled=false；内核日志没有 OOM。

## 20000 设备与队列

- 15:36 左右业务启动，15:46:30 达到 20000 台，约 10 分钟。
- 设备最终均为 `commissioned`。
- 冷启动 `DEVICE/device-mgr-periodic` pending 峰值 68861，15:49 归零，
  redelivery=0。
- `PM/pm-workers` 与 `PM_AGG_15M/pm-aggregation-workers-pull` 持续回到 0，
  redelivery=0。
- 16:32 最终快照：PM_AGG pending=0；PM 仅有 12 条 ack in-flight；
  聚合失败=0，Redis 写错误=0。

COMMAND 只有一个实际消息主题 `command.get_parameters.response`。积压关联两个
必须独立消费全部消息的消费者：

- App 的随机 durable `Lo0yTlW5`：`RPCResponseSubscriber`，负责上行路径翻译、
  参数入库和设备信息刷新。
- `provision-gpv-pull`：`ProvisioningEngine` pull consumer，当前默认
  concurrency=1。

任务库拆解结果：

- 最近 5 分钟 49737 个任务，覆盖 19998 台设备。
- 100% 是 `Baicells / BLQ / FAP/BAIBLQ/SC`、
  `GetParameterValues / source=system / UECountPolicy:GPV`。
- 参数只有 `Device.DeviceInfo.UE_Count` 和
  `Device.DeviceInfo.2.UE_Count`。
- 15:37～16:38 共 452025 个、20000 台设备，均值约 7410/min，
  平均每设备 22.6 次。
- 当前 pending/sent 每台设备最多一条，数量与 distinct device 相等；Redis gate
  和 `LatestOpenTaskByDeviceAndMethod` 的并发合并有效，没有同设备并发重复入队。
- 设备库 `inform_interval=300`，理论约 4000 个 Periodic Inform/min；测试源实际触发
  约 7410/min，接近声明频率的两倍。相同参数的历史重复是“每次 Periodic Inform
  查询一次 UE count”的设计行为，不是同一个周期重复调度。

吞吐：

- 16:13:34～16:14:34 的低波峰中，Lo0yTlW5 pending -3473/min，
  provision-gpv-pull pending -2992/min。
- 16:39:39～16:40:14 的 35 秒波峰生产 12831 条，约 22000/min；
  Lo0yTlW5 消费约 6495/min，pending 增至 152006；
  provision-gpv-pull 消费约 5445/min，pending 增至 187028；redelivery 均为 0。

若测试源立即停止，按当前吞吐估算 Lo0yTlW5 约 23 分钟、
provision-gpv-pull 约 34 分钟清空；若维持一小时平均 7410/min，后者会持续净增长，
无法清空。结论是没有重复调度 bug，但测试源 Periodic Inform 频率与声明不一致，
且两个 GPV 消费者单实例吞吐低于该输入均值，这是明确的容量缺口。

## 冷启动 DLQ 恢复

空环境启动时 PM 文件先于设备注册到达，产生 9502 条
`pm.file.received: device not found`。设备注册完成后：

- 只选择设备现已存在、主题为 `pm.file.received` 的记录。
- 用 `minio_path` 与 TSDB `pm_files` 去重。
- 分批限速 6 req/s 重放，429/502/503/504 自动退避重试。
- 8551 条由审计脚本确认成功；另 951 条已由此前发布/重试处理并被去重。
- 最终 `eligible=0`，覆盖全部 9502 个唯一文件。
- `worker_dlq_replays_total{result="published"}=9551` 比唯一文件多 49，来自早期限流/
  502 后的重复发布；TSDB 以 `minio_path` 去重，没有重复业务数据。

DLQ 记录作为审计数据保留，没有 purge。技术制式不匹配隔离记录从未重放。

## 制式隔离与指标库漂移

截至 16:30：

- 正常 PM 文件 69321 个、19937 台设备，`minio_path` 全部唯一。
- 技术制式隔离 5595 个文件、3358 台设备。
- 所有抽样均为设备身份 LTE、实际 XML GSM，设备族为
  `Baicells / BLQ / FAP/BAIBLQ/SC / 48BF74`。
- XML 中存在明确 GSM 指标族，例如 `Call.*`、`SDCCH.*`、`TCH.*`、`BTS.*`。
- `PMTechnologyMismatch` 正确触发；错误文件未进入 LTE 指标库或聚合。

`PMReportKeysMissingFromLibrary` 与 `PMKnownIndicatorsDisabled` 仍在触发，分别对应
已经拆开的 `whitelist_miss` 和 `known_but_disabled`，不再与 technology mismatch
重复计数。这说明隔离/分类代码工作正常，但测试源上报名与当前启用指标库仍有真实
覆盖差异，不能解释成“数据已保留且无影响”。

## 16:15 聚合、KPI 与小时/天/周

第一个有效任务版本从 16:00 生效。16:15 文件到达后：

- Redis `pmagg:*` 从 0 增长到 112426 个键，内部内存约 1.42 GiB / 6 GiB，
  `maxmemory-policy=noeviction`。
- 样例 accumulator 是 hash，HLEN=108，MEMORY USAGE=7304 bytes，所有值使用
  `v2|...` 编码。
- TSDB 有 18700 个 16:00～17:00 `hourly/open` 设备流水线窗口，
  `expected_slots=4`、`received_slots=1..2`，符合两个 15 分钟源槽逐步到达。
- `pm_aggregation_results=0`、`pm_aggregation_counter_rollups=0` 是当前时点的
  正常状态：小时自然窗口尚未结束，12 分钟水位关闭后才会发布并逐级生成天、周结果。

KPI 依赖闭包实测：

| 槽 | K900010006 的 6 个依赖 | K900010076 的 2 个依赖 |
| --- | ---: | ---: |
| 16:00 | 每项覆盖 18465 个唯一源文件 | 每项覆盖 18465 个唯一源文件 |
| 16:15 | 每项覆盖 8605 个唯一源文件 | 每项覆盖 8605 个唯一源文件 |

其中 `C000010070`、`C000010080` 已按修正后的 BLQ 映射入库。两个 KPI 当前没有最终
小时结果的原因是窗口未到关闭时间，不是依赖 Counter 缺失。天、周进行中结果必须在
首个小时发布后才能验证数值；本次在 16:33 截止，不能伪造自然周期完整性结论。

## ACS 与 HTTP

- ACS 全局会话上限 30000；负载期 active sessions 约 16800～17700。
- `acs_pm_upload_backpressure_rejected_total=0`。
- `acs_rate_limit_rejected_total=0`。
- ACS 日志中 `admission denied`、rate-limit、device-swap 失败均为 0。
- 部署后 App Nginx 没有 503。DLQ 早期高并发重放有 394 个 429 和 1 个瞬时 502，
  限速和重试后不再影响业务。

## 资源告警标签根因与修复

实机发现 node-exporter scrape target 自带 `service="node"`，而 textfile 资源计划也
使用 `service="app|..."`。Prometheus 为避免冲突把样本标签改成
`exported_service`，旧规则仍按 `service` 与 cAdvisor 连接，造成 27 条 CPU quota、
CPU period、memory limit “序列缺失”假告警。

修复把资源计划业务标签改为 `compose_service`，并同步 absent、drift 规则与 Grafana
图例。热更新后实机验证：

- 资源计划 CPU baseline：9 个服务。
- 资源计划 memory baseline：9 个服务。
- cAdvisor quota / period / memory limit：各 9 个服务。
- 三类 absent 查询：均为 0。
- drift 查询：0。
- `ALERTS{alertname=~"OMCResourcePlan.*"}`：0。
- Dashboard 两个资源计划查询：均返回完整 9 个服务。

## 18:30 Redis 无重启临时保护

GPV 旧消费者积压尚未部署根治版本时，Redis 数据集继续增长。18:27～18:30 CST
只读采样：

- `used_memory=3.98 GiB`，原 `maxmemory=6 GiB`，占上限约 66.3%；
  `mem_fragmentation_ratio=1.03`、`lazyfree_pending_objects=0`。
- `DBSIZE=2122116`，其中 `acs:task:*` 约 1332137 个、
  `acs:taskq:*` 约 6847 个、`dedup:acs-pm-upload:*` 约 223855 个。
- `evicted_keys=0`、无 OOM errorstat，策略为 `noeviction`。
- COMMAND stream 398014 条 / 207222218 bytes；旧 RPC ephemeral
  `Lo0yTlW5` 为 pending=389761、ack_pending=1000；
  `provision-gpv-pull` 为 pending=398013、ack_pending=1，二者 redelivery=0。

为避免旧版本清积压期间撞到 6 GiB 写入拒绝线，在不重启、不重建 Redis 的条件下执行：

- 先备份 `.env.bak.20260730T103051Z` 与
  `resources.env.bak.20260730T103051Z`。
- 运行时 `CONFIG SET maxmemory 7gb`，保持
  `maxmemory-policy=noeviction`。
- `.env` 与 `resources.env` 均持久化 `REDIS_MAXMEMORY=7gb`；
  `resources.env` 保持 `REDIS_MEM=8192m`，容器硬限仍为 8589934592 bytes。
- Redis 容器 `StartedAt=2026-07-30T07:35:27.964475849Z`，确认未重启。
- `resource_env_validate` 通过，资源计划指标已刷新，完整 healthcheck
  73/73 通过。
- 调整后 `used_memory=3.96 GiB`、`maxmemory=7.00 GiB`、
  fragmentation=1.04、`evicted_keys=0`、无 OOM。

这是等待 GPV round 2 部署期间的临时容量保护，不替代固定 durable、同设备 lane
头阻塞与终态 ACS task TTL 根治。

## 最终判断

代码、部署资源、PM 技术隔离、Redis v2 写入、KPI 依赖闭包和小时水位状态符合当前
验收时点预期。资源序列缺失假告警已根治。

不能宣称整个环境“完全无异常”：厂家/模拟器仍持续上报 LTE 身份下的 GSM 文件；
指标库仍存在真实 whitelist/disabled 覆盖差异；COMMAND 在最新一分钟波峰下尚未净
消化；首个小时尚未到 17:12 关闭点，因此小时最终结果以及后续天、周结果仍需续验。

## 20:12 根治版本部署与 15 分钟续验

### 发布与接力

- 从干净 worktree 的精确 commit `c77d57b55` 构建
  `100.0.0-20260730-2012`；归档 SHA-256 校验通过，`VERSION` 中
  `git_commit=c77d57b55`。发布门禁 187/187、真实 NATS 回归 13/13 通过。
- 第一次尝试的 `100.0.0-20260730-1952` 在停止旧 app 前被接力门禁安全中止：
  接力工具复用完整 `AppConfig` 校验，但 one-shot 容器没有注入 JWT，报
  `jwt.secret must not be empty`。旧 app 未停止。随后以专用最小 NATS+GPV
  配置加载根治该耦合，重新构建新包，不复用失败包。
- 12:19:00 UTC 切换前，旧 ephemeral `Lo0yTlW5` 的 AckFloor stream sequence
  为 `1564590`、pending `95290`、ack pending `1000`、redelivery `0`。
  正式 handoff 在流继续前进后返回
  `SourceConsumer=Lo0yTlW5, StartSequence=1568773, Created=true`。
- 固定 `device-rpc-gpv` 实际配置为 `by_start_sequence`、
  `opt_start_seq=1568773`、filter subject
  `command.get_parameters.response`、固定 deliver subject/group，
  AckWait 30 秒、MaxDeliver 5、MaxAckPending 2000。旧 consumer 随旧 app
  退出后消失，新 consumer 没有跳读或重投。

### 部署过程偏差

- NATS、主 PostgreSQL、TimescaleDB、MinIO 均保持原 StartedAt，没有重建；
  app、ACS、worker、web 按计划切换新镜像。
- Redis 被 Compose 重建一次。原因是 18:30 只用运行时 `CONFIG SET` 把
  maxmemory 从 6 GiB 提到 7 GiB，虽然 `resources.env` 已持久化 7 GiB，旧容器的
  Compose config hash/启动命令仍是 6 GiB；本次 `up -d redis` 消除了该漂移。
- Redis 重建后加载约 5 GiB RDB，业务容器启动时收到 `LOADING`，各重启 15 次。
  12:27:16 UTC 加载完成后 app/ACS/worker 自动恢复。安装器因固定 90 秒窗口先以
  healthcheck exit 4 结束；加载完成后人工重跑完整 healthcheck 为 73/73。
- `errorstat_LOADING=41215` 全部来自这次恢复窗口；之后
  `total_error_replies=41276` 不再增长。`evicted_keys=0`、
  `rejected_connections=0`、无 OOM errorstat。所有容器最终 running，
  `OOMKilled=false`。

### GPV、任务 TTL 与 Redis

12:29:28～12:45:03 UTC：

- COMMAND stream last sequence `1673119 → 1737770`。
- `device-rpc-gpv` consumer sequence `9980 → 73888`；
  `provision-gpv-pull` `9809 → 73717`，两条链都处理 `63908` 条目标主题消息，
  约 `4100/min`。二者终点 pending `0`、redelivery `0`，已跟上实时输入。
- 新 RPC consumer 在 12:29 已把接力后的历史 backlog 清到流尾；进入稳态后吞吐与
  输入相等，不再积压。
- 20 个 PostgreSQL 已确认终态任务抽样，Redis 状态 20/20 一致（19 completed、
  1 failed），TTL `896～897s`；活跃任务样本 TTL `14396～14400s`。终态 15 分钟、
  活跃态 4 小时均已在实机生效。
- `acs:task:transition:pending` 在持续负载中出现短脉冲
  `1,14,4,0,9,3,0,0,0,82,30,11`，多次回到 0；终点瞬时值 50 是新转换流入，
  不是单调累积。
- Redis used memory `5046259408 → 4335832592` bytes，15 分钟净下降约
  677 MiB；DBSIZE `2595517 → 2235742`，净下降 359775。
  `acs:task:*` 精确扫描在 12:32 为 1292998，较 11:50 旧版的 1511785
  减少 218787。终点全库扫描因耗时超过两分钟主动停止，避免继续给 Redis 增压。

### 资源、HTTP、队列和数据库

- 复验资源：app 2 CPU/1536 MiB、ACS 5 CPU/4 GiB、worker 8 CPU/2 GiB、
  Redis 2 CPU/8 GiB 且 maxmemory 7 GiB/noeviction；PM finalize concurrency 32，
  Redis v2 write 为 true。
- 12:36 快照主机可用内存 16 GiB、swap 419 MiB、磁盘 35%；load average
  `10.42/13.52/12.54`（32 CPU）。app 29%、ACS 199%、worker 150%、
  PostgreSQL 280%、TimescaleDB 141%、Redis 39% CPU，主机仍有明显 CPU 余量。
- 主库和时序库均无等待锁。主库当时约 8 个 active、74 个 idle；时序库约
  2 个 active、26 个 idle。
- 切换后 Nginx 503 为 0。ACS active sessions 12405，低于全局上限 30000；
  rate-limit 和 PM upload backpressure rejection 均为 0。
- DEVICE、PM、固定 PM_AGG consumer 终点 pending 均为 0、redelivery 均为 0。
  PM_AGG 中周期出现的随机 consumer 是 `ack_policy=none`、
  `inactive_threshold=5s` 的历史快照扫描器；其单批 pending 持续下降，退出后自动
  删除，不属于固定业务队列积压。

### KPI、小时/天/周与未关闭项

- 已发布小时窗口覆盖 16:00～19:00：complete 39907、timeout 13595，
  聚合结果共 7142770 行。20:00 小时窗口仍 open，符合 12 分钟关闭水位语义。
- `K900010006`、`K900010076` 各 53599 行，覆盖 21774 台设备，最新窗口 19:00，
  最新写入约 20:31。两个重点 KPI 已有小时结果，不再是“无数据”。
- 当前自然日 daily 窗口已创建但仍 open；自然日尚未结束，因此没有 daily final，
  weekly 也未到可发布点。不能把进行中周期解释成完整自然周期。
- `PMRebuildSnapshotScanSlow` 曾短暂 pending，随后清除；快照扫描样本 1 次、
  57.42 秒。固定 PM/PM_AGG 消费无积压，但终点
  `PMFinalizeOldestDueHigh` 正在 firing，oldest due 约 2166 秒，
  finalize errors 为 0、claim conflicts 为 0、inflight 为 1。这说明历史窗口修复
  读取仍拖慢关闭新鲜度，必须继续部署有界历史扫描/公平轮转修复后复验，不能宣称
  PM 聚合完全无异常。
- 两个既有外部业务告警仍为 `PMReportKeysMissingFromLibrary` 和
  `PMKnownIndicatorsDisabled`；其含义与前述厂家上报名/指标库覆盖漂移一致。
- 终点另有两个 `ContainerMemoryNearLimit` 处于 pending：PostgreSQL 约 92.3%，
  Tempo 约 97.6%。尚未持续到 firing，但资源告警有效，需继续观察并调整对应上限或
  工作集。
