# 运行时业务与性能根治设计

## 背景与证据

2026-07-31 在 172.24.224.197 清洁部署 20,000 台模拟设备后，基础设施健康，
HTTP 502/503/504 为 0，NATS consumer pending/redelivery 为 0，但业务任务和
Redis 队列观测仍不达标：

- 最近 30 分钟 `UECountPolicy:GPV` 创建约 132,000 个任务，其中约 11,600 个
  `exceeded max retries`，约 10,100 个停留在 `sent`。
- 所有设备每 5 分钟同步 Periodic Inform，现有策略在每次 Inform 中为每台设备
  创建一个 UE Count GPV，形成约 240,000 任务/小时的固定波峰。
- UE Count 创建链路在 Inform request context 下串行执行 Redis 准入、PG 查重、
  产品路径解析、PG 持久化和 Redis 入队，并共享 500ms 超时。Redis 入队超时后，
  PG 回滚继续使用已经过期的 context。
- 合法的空 GPV ParameterList 被编码为 JSON `null`，设备响应消费者只接受数组，
  因此持续输出 `unexpected type <nil>` 告警。
- Redis 队列观测器扫描到 10,000 个 key 后主动失败；扫描过程中 key 被正常删除时，
  `TYPE none` 也被当作故障。20,000 设备负载会稳定触发误报。

## 目标

1. 20,000 台设备同步 Periodic Inform 时，UE Count 仍能周期刷新，但不再形成
   20,000 任务同刻波峰。
2. 服务端原因导致的 UE Count 任务失败率低于 0.1%，任务队列在一个 5 分钟周期内
   排空，ACS 不再出现 UE Count 入队超时。
3. 空 GPV 是合法空结果，不产生日志告警或无意义重投。
4. Redis 队列 key 超过 20,000 时观测仍保持 `up=1`；正常删除竞态不产生失败。
5. Redis 入队失败后 PG 不留下 pending 孤儿。
6. 不改变 Dashboard 只读取现有小时/天/周结果、每 5 分钟定时刷新的既定契约。

## 方案选择

### 方案 A：只提高 ACS/Redis/PG 配额

不采用。宿主机仍有 CPU 和内存余量，但任务生成率是固定的 240,000/小时；
提高配额只能推迟积压和失败，不能消除同步波峰。

### 方案 B：把每次 UE Count 探测搬到 parameter-sync durable readback

本轮不采用。它能把任务创建移出 Inform 热路径，但仍会生成同样数量的任务；
并且当前 durable completion projector 只对 full scope 做 `device_info` 投影，
直接迁移会破坏 UE Count 的即时投影语义。

### 方案 C：Redis 原子到期闸门 + 冷启动稳定分片

采用。每台设备在 Redis 保存 `next_due_at`：

- 默认探测周期 1 小时，冷启动按设备 SN 稳定散列到 12 个 5 分钟槽。
- 20,000 台设备同步上报时，每个槽约 1,667 台，1 小时内完整覆盖。
- 到期判断与推进 `next_due_at` 使用 Lua 原子执行，多 ACS 实例不会重复准入。
- 设备 Inform 周期不是 5 分钟时，也会在到期后的下一次 Inform 执行，不会永久错过
  固定槽。
- 创建失败将 `next_due_at` 调整为 5 分钟后重试；成功或已有 open task 时维持
  正常 1 小时周期。
- 继续创建普通系统 GPV，完整复用现有标准路径翻译、响应回译、
  `device_parameters` 入库和 `device_info` 刷新。

## 详细设计

### UE Count 到期闸门

`RedisUECountProbeGate.Acquire` 接收当前时间、正常周期和冷启动延迟，通过单 key
Lua 脚本完成：

1. key 不存在或是旧版本值 `"1"`：写入稳定散列得到的首次到期时间；延迟为 0
   的设备立即准入。
2. key 的 `next_due_at` 晚于当前时间：拒绝本次探测。
3. 已到期：原子推进到 `now + period` 并准入。
4. key TTL 为两个正常周期；长时间离线后重新接入会再次走稳定冷启动分片。

创建链路失败时调用 `RetryAfter(5m)`，原子把当前设备的下次到期时间缩短到
`now + 5m`。失败不会阻塞 InformResponse。

### 空 GPV

`decodeParameterValues` 将 `nil` 和 JSON `null` 统一解释为空切片；畸形的非数组值
仍返回错误。这样只消除合法空结果噪音，不吞掉真实 schema 错误。

### 任务双写回滚

Redis `Push` 失败后，PG `Delete` 使用脱离调用方取消信号、但有独立短超时的 context。
调用链 trace/value 保留，取消和 deadline 不继承，避免已经到期的 request context
让补偿操作必然失败。

### Redis 队列观测

- 移除按 key 数量判故障的 10,000 硬上限，扫描由调用 context 和串行采集周期约束。
- `TYPE none` 表示 key 在 SCAN 与读取之间自然消失，按空队列跳过。
- 其他未知类型仍报错，避免掩盖真实数据污染。
- 保持业务 gauge 的“采集失败保留上次值”语义。

## 验证与发布门禁

### 自动化

- 新测试必须先在旧实现上按预期失败，再写最小实现。
- 目标包：`internal/acs`、`internal/device`、`internal/task`。
- 后端全量：`go build ./...`、`go test ./... -count=1`。
- 发布包既有全量门禁全部通过。

### 20,000 设备线上验收

- 首次 5 分钟 UE Count 新任务不超过 2,000；任意连续 65 分钟覆盖 20,000 台设备。
- 稳态 UE Count server-attributable 失败率 `< 0.1%`。
- `pending + sent` 在每个 5 分钟槽结束后 5 分钟内回落，oldest age `< 300s`。
- `enqueue UE count GPV task failed` 为 0，PG pending/Redis queue 孤儿差异为 0。
- `decode parameter_values ... unexpected type <nil>` 为 0。
- `omc_redis_task_queue_up{queue_family="taskq"}` 持续为 1，扫描失败增量为 0。
- HTTP 502/503/504 为 0，容器 restart/OOM 为 0。
- 宿主机 CPU idle 保持余量，iowait `< 5%`，磁盘平均写延迟 `< 5ms`。

## 后续迭代

本 MR 部署稳定后，再基于真实 PM drift 明细修复指标库/白名单问题；关键 KPI
`K900010006`、`K900010076` 的依赖覆盖和小时/天/周结果继续按既定 12 分钟水位验收。
