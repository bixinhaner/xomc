# Issue #220 UE Count 同步实施计划

1. 在 `device_info_calc_ue_test.go` 先增加多小区、去重和回退边界用例，确认旧实现失败。
2. 最小修改 `CalcUECount`，按物理小区聚合标准路径。
3. 为 ACS 新增 `UECountPolicy` 测试，覆盖产品映射筛选、Periodic 触发、GPV 参数和
   pending/sent 去重，确认测试先失败。
4. 在 `PathTranslationService` 暴露设备支持的 UE Count 标准路径解析，并实现
   `UECountPolicy`。
5. 将策略注入 `ServerDeps`，在 `handleInform` 的 InformResponse 前执行。
6. 为多实例并发增加 Redis 原子租约，为同步依赖增加短超时，并给 active task
   精确查重 SQL 增加部分索引。
7. 运行 gofmt、目标包测试、`go build ./...` 和串行 `go test -p 1 ./...`，审查最终 diff。
