# #372 COMMAND/GPV redelivery 真实夹具

测试目标：

- 证明 COMMAND/GPV 慢设备或 ACK 断连窗口下，JetStream 的发布、投递、ACK 进度可被观察。
- 证明 parameter-sync 结果在 PostgreSQL 已提交但 NATS ACK 失败后被重投时，不会重复创建 run、task、result 或 device 参数投影。
- 证明本次夹具不引入 `device_sn`、`object_path`、完整 Redis key 等高基数指标标签。

自动化夹具：

- 覆盖矩阵

| 场景 | 夹具 | 关键断言 |
|---|---|---|
| COMMAND/GPV 成功 | `internal/core/event.TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles` | 释放慢处理后 `AckSequence=LastSequence=3`，`AckGap=0` |
| 瞬时失败 | `internal/core/event.TestKeyedQueueHandlerFailureNaksThenSuccessAcks` | 第一次 NAK、第二次 ACK，最终 `AckGap=0` |
| 永久失败 | `internal/core/event.TestKeyedQueueTerminalHeadFailureExhaustsMaxDeliverBeforeFollower` | head 消息耗尽 `MaxDeliver` 后 follower 才执行，避免同设备乱序 |
| ACK 断连 | `internal/paramsync.TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection` | PG commit 后模拟 ACK 链路失败，NATS redelivery 后业务投影仍只一份 |
| redelivery 幂等 | `internal/paramsync.TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection` | `parameter_sync_task_results`、`device_parameters`、run、task 计数均为 1 |
| 慢设备 | `internal/core/event.TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles` | 慢处理期间 `DeliverySequence > AckConsumerSequence` 且 `AckGap > 0` |

- `internal/core/event.TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles`
  - 使用真实 NATS JetStream，创建唯一 `test.command.gpv.issue372.<suffix>` subject、stream 和 durable，避免与本地真栈既有 `command.get_parameters.response` stream 重叠。
  - 发布 3 条 GPV 响应，第一条模拟慢设备处理。
  - 慢处理期间断言 `LastSequence=3`、`DeliverySequence > AckConsumerSequence`、`AckGap > 0`、`AckPending > 0`。
  - 释放处理后断言 `AckSequence=3`、`Pending=0`、`AckPending=0`、`AckGap=0`。

- `internal/paramsync.TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection`
  - 使用真实 NATS JetStream + PostgreSQL。
  - 构造一条 parameter-sync request/run/device_task/device 投影。
  - 第一次消费先让 `PGResultProcessor` 提交事务，再模拟 ACK 链路失败并返回错误，触发 NATS redelivery。
  - 第二次消费同一事件时必须命中重复结果保护。
  - 断言 `parameter_sync_task_results`、`parameter_sync_staging_values`、`device_parameters`、`parameter_sync_runs`、`device_tasks` 都只有 1 份业务记录，request/run 进入 succeeded，`active_run_id` 清空，`last_param_sync_at` 更新，旧失败字段清空。

本地 Docker 真栈执行：

```bash
cd /Users/shangyingbin/xomc-project/xomc/omcgo

# 指向 Docker 真栈里的 NATS 与 PostgreSQL；如端口不同，按本机 compose 映射调整。
export GPV_NATS_TEST_URL=nats://127.0.0.1:4222
export TEST_PG_URL='postgres://omcgo:omcgo123@127.0.0.1:5432/omcgo?sslmode=disable'

go test -count=1 ./internal/core/event \
  -run 'TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles'

go test -count=1 ./internal/paramsync \
  -run 'TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection'
```

隔离和清理：

- 两个 NATS 夹具都使用纳秒后缀的唯一 stream/subject/durable，并在 `t.Cleanup` 中删除 stream。
- PostgreSQL 夹具使用唯一 UUID request/run/task/device，并在 `t.Cleanup` 中删除本测试写入的 task、run、staging 和 device parameter；request/device 清理由既有测试 helper 负责。
- 夹具不读取浏览器、Cookie、token 或共享业务环境数据。

指标标签检查：

- 新增测试没有新增指标名或标签。
- `EventBusMetrics` 已有 `TestEventBusMetrics_ContractUsesOnlyLowCardinalityLabels`，明确拒绝把 `device_sn`、`redis_key`、`object_path` 或包含设备 SN / Redis key 的值暴露为指标标签。
- COMMAND/GPV 队列观测仍只使用固定业务 subject 和 durable 维度。
