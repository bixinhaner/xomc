# KPI 压测过载治理设计

## 背景与证据

目标机为 32 核、32 GiB 内存，但 PostgreSQL、TimescaleDB、Redis、NATS 和
MinIO 共用一块 1 TiB 旋转盘。持续 KPI 压测期间观测到：

- CPU 未耗尽，内存仍有约 18 GiB 可用；
- 块设备利用率约 95%，平均队列深度约 47，主机 IO wait 为 67%～86%；
- MinIO 出现约 2.1 万线程，其中约 3700 个线程阻塞于 `do_renameat2`；
- FileUpload 成功率低于 1%，大量 499/502，耗时 p95 超过 100 秒；
- PM durable consumer 的 pending 超过 82 万、ack pending 达默认上限 1000；
- 现有 ACS 背压仅检查磁盘空间使用率，空间使用 66% 时不会触发；
- `acs_active_sessions` 在进程重启后被历史会话清理减为负数；
- TimescaleDB 维表同步周期性执行 `TRUNCATE`，触发立即同步写；
- ACS 高频 access log 已增长到数 GiB，并与业务数据竞争同一块盘。

根因是缺少基于 IO 压力的闭环准入控制，导致对象存储的小文件落盘、数据库
WAL、Redis AOF、NATS JetStream 和日志同时制造随机写与日志提交。增加 CPU 或
内存不能消除该瓶颈。

## 方案选择

采用分层、有界、可观测的闭环治理：

1. ACS PM 上传同时受 IO PSI 高低水位与在途上传数量限制；
2. PM worker durable consumer 的 `MaxAckPending` 与实际处理并发绑定；
3. 移除无意义的高频 ACS access log 写盘；
4. 维表同步从 `TRUNCATE + COPY` 改为事务内 staging + upsert/delete，避免清空表；
5. NATS JetStream 进程内存上限由容器限额派生，避免配置看到宿主机内存；
6. 修复活跃会话指标，使进程只能递减本进程曾递增的会话。

不采用仅靠 Nginx 固定限速的方案，因为它无法随磁盘状态自动恢复，也无法区分
PM 上传与 CWMP 控制流量。不采用仅升级 SSD 的方案，因为软件仍会在任何较慢或
退化存储上形成无界排队。SSD/NVMe 仍是推荐的最终存储介质。

## 详细设计

### PM 上传准入

背压配置增加：

- `max_inflight`：默认 64；
- `io_some_high_pct`：默认 40；
- `io_some_low_pct`：默认 20。

watchdog 读取 `/proc/pressure/io` 的 `some avg10`。超过高水位进入背压，回落至
低水位才恢复；读取失败时 PSI 信号 fail-open，但 `max_inflight` 始终生效。
上传 handler 在写 MinIO 前获取准入令牌，并在所有返回路径释放。超过上限或
watchdog 已进入背压时快速返回 503，使设备按协议重试，避免请求在 Nginx/MinIO
中等待一分钟后变成 499/502。

新增指标记录 IO PSI、在途上传数和按原因区分的拒绝数。保留已有指标名兼容现有
面板和告警。

### NATS worker 背压

`QueueSubscribe` 增加按 subject 可配置的 queue tuning。PM worker 在订阅前把
`MaxAckPending` 设置为处理并发的 4 倍（最小 16），`AckWait` 保持 2 分钟，
`MaxDeliver` 保持 5。启动时更新已有 durable consumer，而不是只影响新建 consumer。

### 会话指标

ACS handler 维护本进程已计入指标的 session ID 集合。只有集合中存在的 session
完成时才递减 gauge。重启前遗留、TTL 清理和 nil session 不再递减本进程的 gauge。

### TSDB 维表同步

源数据先 COPY 到同事务临时 staging 表，再执行 `INSERT ... ON CONFLICT DO UPDATE`
和 anti-join DELETE。目标表始终可读，且不再执行 `TRUNCATE` 的强制同步写。若目标表
缺少可用于冲突判定的主键/唯一键，则保留原路径并记录告警，避免改变数据语义。

### 部署与日志

发布 compose 中 NATS 增加由 `NATS_MEM` 派生的 JetStream `max_memory_store`，默认
不超过容器内存的 25%。ACS access log 默认关闭，错误日志保留；管理面 access log
不变。部署检查发现 AIDE 正在扫描 OMC 数据目录时给出明确警告和排除目录建议，但
不擅自修改主机安全策略。

## 验收标准

部署后连续执行最多 10 个固定窗口。满足以下条件并连续两个窗口稳定即可判定本轮
发现的问题已解决：

- 容器无重启、无 OOM，健康检查全部通过；
- FileUpload 不再出现持续 499/502，快速 503 作为预期过载信号；
- 已接纳上传的 p95 明显下降，目标小于 15 秒；
- MinIO 线程数回落到 1000 以下，磁盘平均队列明显低于基线 47；
- PM durable consumer 的 ack pending 不超过配置上限，redelivery 不持续增长；
- PM pending 的增长斜率不再失控；
- `acs_active_sessions` 不小于 0；
- NATS 内存低于容器限额的 80%。

由于压测前已积累约 82 万 PM backlog，验收以队列斜率和稳定性为准，不要求在单次
验证中清空历史积压。

