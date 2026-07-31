# Code Review Report: Issue #234 异频载波 Q-RxLevMin 校验

## 结论

PASS

- CRITICAL: 0
- WARNING: 0
- INFO: 1

## 审查范围

- `data/quicksettings/BLN.xml`
- `data/quicksettings/BLQ.xml`
- `data/quicksettings/BM.xml`
- `data/quicksettings/MLN.xml`
- `data/quicksettings/MLQ.xml`
- `internal/quicksettings/loader_test.go`

## 变更摘要

- 为五个 LTE 参数模型中的 `QRxLevMinSIB5` 显式声明整数类型及 `-70` 到 `-22` 的数值范围，避免负数上下限被按字符串长度规则校验。
- 新增内置快速设置加载回归测试，确保五个型号都保留一致的数值约束。

## 审查发现

### INFO

- 全量扫描快速设置 XML 后，其他带上下限但未声明类型的字段仅为列表容量配置，现有列表组件按容量语义消费，不属于本缺陷的数值输入场景。

## 风险与兼容性

- 仅修正快速设置元数据，不改变数据库、API 或设备参数路径。
- 影响 BLN、BLQ、BM、MLN、MLQ 的异频载波快速设置表单。
- 约束范围与页面既有提示和设备参数定义一致。

## 验证

- `go build ./...` — 通过
- `go test ./...` — 通过
- `go test ./internal/quicksettings -count=1` — 通过
- `git diff --check --staged` — 通过
- 本地部署后真实浏览器显示 `Q-RxLevMin [-70 ~ -22]`，合法值 `-23` 可通过表单校验，越界值 `-21` 被拒绝。
