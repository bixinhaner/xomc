# Issue #247 Dashboard 告警比较修复审查

## 结论

PASS_WITH_WARNINGS

## 修改范围

- `internal/dashboard/service.go`
- `internal/dashboard/service_test.go`

## 行为审查

Dashboard 在计算“活跃告警较昨日”前，先重建昨日同一时刻的设备快照。

- 昨日设备数为 0：不读取昨日告警快照，返回 `has_comparison=false`，前端显示 `N/A`。
- 昨日设备数大于 0：继续读取昨日告警快照，保留增量升级场景的真实比较行为。
- 昨日设备查询失败：记录带上下文的错误日志，并返回不可比较状态。
- 当前活跃告警数量不受影响，仍按当前告警统计结果展示。

该规则与 Issue #247 的部署语义一致：全量部署会清除历史设备数据，增量升级保留历史数据。

## 测试

通过：

- `go test ./internal/dashboard -count=1`
- `go build ./...`
- `npm test --workspace webcode -- --run src/pages/dashboard/index.test.tsx`
- `git diff --check`

全量 `go test ./...` 未完全通过，但失败与本次修改无关：

- `internal/device/TestBulkUpsertCommitsUncontendedDeviceBeforeWaitingForContendedDevice`
- 本地数据库报 `no partition of relation "devices" found`
- 部分 Redis 集成测试连接本机 `127.0.0.1:60431` 被拒绝

Dashboard 后端包和 Dashboard 前端测试均通过。

## 风险与建议

该修复使用昨日设备快照作为“是否存在可比较告警基线”的业务判断。若未来出现“设备数据保留但告警历史被清空”的部署模式，需要单独定义告警数据集的部署边界；当前 Issue #247 明确的全量/增量语义不包含该场景。