# 自定义聚合导出缺值占位审查

## 范围

- 分支：`fix/pm-adhoc-export-placeholder`
- Issue：[#879](https://github.com/569423176-sketch/goomc/issues/879)
- 对比基线：`origin/main`
- 变更路径：
  - `omcgo/internal/pm/export/csv.go`
  - `omcgo/internal/pm/export/generator.go`
  - `omcgo/internal/pm/export/runner.go`
  - `omcgo/internal/pm/export/csv_test.go`
  - `omcgo/internal/pm/export/generator_test.go`

## Standards

未发现违反项目硬性规范的问题。

- Go 后端改动保持在 `internal/pm/export` 模块内，未跨层访问。
- 未新增 SQL、迁移、REST 端点、日志、metric 或运营商分支逻辑。
- `NewWideCSVWriter` 原公开调用口径保持不变；缺值占位通过内部 layout 配置，只在 adhoc 导出路径启用。
- 已补回归测试，并保留默认 CSV 空单元格行为测试，避免扩大 dashboard 导出口径。

DoD 备注：

- `golangci-lint run ./...` 未运行：本机未安装 `golangci-lint`。
- 前端、迁移、安全敏感路径、新端点：N/A。

## Spec

规格来源：本轮需求“性能管理-自定义聚合页面，图表无值显示为 `-`，导出为空字符串；希望都显示为 `-`”。

未发现规格缺口。

- 自定义聚合导出路径在构造 `csvLayout` 时设置缺值占位为 `-`。
- 横表 writer 在单元格无对应指标点时使用配置占位符，有真实数值时仍覆盖为数值，`0` 不会被误判为空。
- 默认 writer 行为保持空字符串，避免影响非 adhoc 导出。

## 验证

- `cd omcgo && go test ./internal/pm/export -count=1`
- `cd omcgo && go build ./...`
- `cd omcgo && go test ./...`
- `git diff --check`
