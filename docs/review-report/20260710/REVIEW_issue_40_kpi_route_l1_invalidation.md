# Review：Issue #40 KPI 路由 L1 热更新

- 日期：2026-07-10
- 固定点：`main`
- 审查范围：当前分支 `fix/40-kpi-route-l1-invalidation` 相对 `main` 的工作区 diff
- 规格源：GitLab Issue #40；父 PRD #39
- 结论：PASS with one known integration gap；CRITICAL = 0

## Standards

初审发现的三项硬违规均已修复：

- singleflight 改为 `DoChan`，每个等待者响应自己的 context；共享 route 构建使用保留 values 的独立 10 秒上限 context。
- 并发测试移除 `time.Sleep`，改用明确的进入信号和仅用于失败保护的 timeout。
- 本次新增测试统一为 `Test_LookupByDevice_<Scenario>`。

剩余判断项：

1. 两个新增 Prometheus 指标本 MR 未配告警规则。处理：人工恢复、重试和告警属于后续 Issue #41，本切片只提供读侧原始信号。
2. L1 热路径增加一次 Redis 版本读取，未更新压测基线。处理：这是 #40 锁定的一致性契约；同 product miss 已用 singleflight 防止 DB 放大，后续如观察到 Redis RTT 成为瓶颈再单独建立性能任务。
3. 版本读取失败 WARN 没有 request/trace 字段。处理：Router 当前没有请求日志字段 seam；日志已包含版本/产品上下文，未扩大公共接口。本次不为日志字段引入跨模块耦合。

未发现运营商硬编码、字符串 SQL、裸 panic、接口扩为 `any`、删除测试或安全性降低。

## Spec

已满足：

- app/worker 两个 Router 使用两个 RedisCache 实例共享同一 Redis；远端 bump 后 worker 下一次查询清除旧 L1。
- 新 report_key counter 进入准入集合，新 KPI 进入求值集合。
- Redis 版本读取失败 fail-open，恢复后追平，并产生 WARN/Prometheus 指标。
- 任意版本不相等均失效，包括版本回退。
- DB 构建期间 bump 时旧快照不会以新版本写入 L2；连续变化采用三次有界重试。
- 同 product 并发 miss 合并，等待者可独立取消，race 测试通过。
- route 重建日志包含来源层级、cache version、counter 数和 KPI 数。

已知中等级缺口：没有在单条普通测试中真正执行 `indicator.Loader.LoadOnce/Reload` 的完整 PostgreSQL 写入。现有证据采用组合 seam：既有 Loader 测试证明成功 reload 调用注入 bumper；本 MR 双 Router 测试证明该 bumper 到 worker 新 route 的完整读侧行为。真实 Loader 会事务重写 ENB/GSM/GNB 指标表，专用数据库集成验收留在写路径 Issue #42，不在 #40 单测中引入破坏性依赖。

未发现 scope creep 或明确的错误实现。

## DoD 与验证

- `go build ./...`：PASS
- `go test -race ./internal/pm/kpi/router -count=1`：PASS
- PMCollector/KPI/aggregator、worker、app provider 定向测试：PASS
- Router 覆盖率：82.3%
- `go test ./...`：除既有 `internal/core/carrier.TestRegistryResolveByOUI` 非确定性失败外通过；用户明确授权本次忽略。该失败来自共享 OUI + map 遍历，与本 diff 无关。
- `golangci-lint`：N/A，本机未安装（`command not found`）；未伪报执行成功。
- 前端、新端点、迁移、安全敏感路径：N/A

汇总：Standards 0 个硬违规、3 个已记录判断项；Spec 0 个错误实现、1 个已记录集成缺口；最严重项为真实 Loader SQL reload 未形成单条自动化集成测试，无 CRITICAL。
