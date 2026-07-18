# KPI 压测过载治理实施计划

## Task 1：扩展 PM 上传背压

1. 在 `backpressure_test.go` 增加 PSI 解析、双信号迟滞、并发准入与释放测试。
2. 运行目标测试，确认因缺少实现失败。
3. 实现 PSI 配置、探测、决策、准入令牌和指标。
4. 在 `handler_test.go` 增加 PM 才使用准入且所有路径释放的测试。
5. 接入 handler 与 seed 配置，运行 upload 包测试。

## Task 2：限制 PM durable consumer 在途消息

1. 在 `nats_bus_test.go` 增加 queue tuning 默认、覆盖及已有 consumer 更新测试。
2. 运行目标测试，确认失败。
3. 实现 `QueueTuning`、`SetQueueTuning` 和 `MaxAckPending` reconciliation。
4. worker 按 PM collector concurrency 配置 tuning。
5. 运行 event 与 worker 相关测试。

## Task 3：修复活跃会话指标

1. 增加重启遗留会话与重复 complete 不会递减的测试。
2. 运行测试，确认现有负数行为。
3. 仅对本进程登记的 session 执行 gauge decrement。
4. 运行 ACS handler 测试。

## Task 4：削减同步与日志 IO

1. 为 TSDB mirror SQL 增加 staging/upsert/delete 行为测试。
2. 实现事务内临时 staging 同步；无唯一键时安全回退。
3. 增加配置测试，要求 ACS access log 默认关闭。
4. 修改开发与发布 Nginx 配置，并运行配置测试。

## Task 5：对齐 NATS 内存配置

1. 增加 shell 测试，要求 planner 输出 `NATS_MAX_MEMORY_STORE`，compose 使用该值。
2. 修改资源规划脚本与开发/发布 compose。
3. 运行发布部署脚本测试。

## Task 6：完整验证、提交和部署

1. 执行 gofmt、目标包测试、`go build ./...`、`go test ./...`。
2. 执行部署 shell 测试及 compose 配置校验。
3. 检查 diff，提交 Conventional Commit，推送现有 MR。
4. 构建 release，复制到目标机并用 compose 更新服务。

## Task 7：最多 10 轮复测与迭代

每轮固定采集健康、HTTP 状态/延迟、磁盘、MinIO 线程、NATS consumer、内存和会话
指标。若未达到设计文档标准，则只针对证据暴露的新瓶颈补测试、修复、提交、部署后
继续下一轮；连续两个窗口稳定则提前结束。所有迭代继续提交到同一个 MR。

