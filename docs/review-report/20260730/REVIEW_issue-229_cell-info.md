# Code Review: Issue 229 设备详情小区信息

## 结论

`PASS_WITH_WARNINGS`

## 变更范围

- 后端详情小区组装：按产品类型限制物理小区数量，并过滤无真实小区配置数据的额外行。
- 前端设备详情：小区激活状态只依据小区自身 `opState`，不再使用设备在线状态覆盖空值。
- 增加 SC 单小区虚假载波回归测试。

## 审查发现

### CRITICAL

无。

### WARNING

`NumOfCells` 仍可能与设备参数实际同步状态短暂不一致；本次通过真实 Cell ID、PCI、频点、带宽、频段字段过滤额外行，避免将仅有 FAPControl 状态的实例显示为小区。若后续需要展示“已配置但尚未同步完整”的小区，应另行定义状态模型，不应恢复无条件补行。

### INFO

- 未传入产品类型时保留 `AssembleCells` 原有稀疏实例行为，避免影响已有调用方和历史测试。
- `FAP/MLN/SC` 限制为 1 行，`FAP/MLN/DC` 限制最多 2 行。
- 设备级离线状态展示逻辑未改变。

## 验证结果

- `go test ./internal/device` 通过
- `go build ./...` 通过
- 前端激活/射频状态测试：17 项通过
- `npm run typecheck` 通过
- `git diff --check` 通过
- 编辑器错误检查无错误
