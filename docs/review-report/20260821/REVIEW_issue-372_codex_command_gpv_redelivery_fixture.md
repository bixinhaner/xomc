# Review #372 COMMAND/GPV redelivery 夹具

基准：`main` + 当前工作区相关改动

## Standards

结论：通过，未发现 CRITICAL 或硬性规范违背。

- 分支不是 `main` / `master`。
- 后端 SQL 仍使用既有 repository/processor 路径；本次没有新增业务 SQL 拼接入口。
- 新增 NATS 夹具使用唯一 stream、subject、durable，并通过 `t.Cleanup` 删除 stream，符合 #372 对共享 NATS 状态隔离的要求。
- `QueueStats` 在 oldest pending direct-get 返回 `ErrNoResponders` 时仍返回核心 consumer 统计，避免真实 NATS 管理面瞬时不可用遮蔽 ACK gap。
- `ResultConsumer.WithSubscription` 保持生产默认 `param_sync.task.result` / `param-sync-results` 不变，只给测试夹具提供隔离入口。
- 指标没有新增 label 或 metric；既有低基数测试继续覆盖 `device_sn`、`redis_key`、`object_path` 不进入 Prometheus label。
- 文档记录了本地 Docker 真栈执行命令和清理边界。

## Spec

结论：通过，#372 验收项已有对应自动化或记录。

- COMMAND/GPV 成功、瞬时失败、永久失败、ACK 断连、redelivery、慢设备均在验证矩阵中落到具体测试。
- `TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles` 覆盖慢设备下 ACK gap 可见，以及释放后 ACK 追平发布/投递进度。
- `TestKeyedQueueHandlerFailureNaksThenSuccessAcks` 补强了 keyed lane 本地重试不提前 NAK、最终 ACK 的计数断言。
- `TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection` 覆盖 PG commit 后 ACK 失败再投递的真实组合，并断言 result、device parameter、run、task 均只保留一份，staging 最多一份且终态可清理为 0。
- 验收文档说明了本地 Docker 真栈运行所需的 `GPV_NATS_TEST_URL` 和 `TEST_PG_URL`。

## Verification

- `GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test -count=1 ./internal/paramsync -run 'TestResultConsumer_|TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection'`
- `GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test -count=1 ./internal/core/event -run 'TestEventBusMetrics_ContractUsesOnlyLowCardinalityLabels|TestKeyedQueueHandlerFailureNaksThenSuccessAcks|TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles'`
- `GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test -count=1 ./internal/paramsync`
- `GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test -count=1 ./internal/core/event`
- `GOCACHE=/Users/shangyingbin/tmp/go-build-cache go build ./...`
- `GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test ./...`
- `GPV_NATS_TEST_URL=nats://127.0.0.1:4222 TEST_PG_URL='postgres://omcgo:omcgo123@127.0.0.1:5432/omcgo?sslmode=disable' GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test -count=1 ./internal/core/event -run 'TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles' -v`
- `GPV_NATS_TEST_URL=nats://127.0.0.1:4222 TEST_PG_URL='postgres://omcgo:omcgo123@127.0.0.1:5432/omcgo?sslmode=disable' GOCACHE=/Users/shangyingbin/tmp/go-build-cache go test -count=1 ./internal/paramsync -run 'TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection' -v`

备注：第一次 `go test ./...` 中 `TestResultProcessingDoesNotBackfillLegacyTaskOutbox` 出现一次 `conn closed`，该用例单独重跑通过；第二次全仓测试通过。
