# 代码审查报告：Issue #287 BM MAC 地址投影

- 日期：2026-08-12
- 分支：`fix/287-bm-mac-template-path`
- 范围：`internal/device`
- 结论：PASS_WITH_WARNINGS

## 变更摘要

设备信息同步兼容 BM 固件将实例占位符原样保存为
`Device.Ethernet.Interface.{i}.MACAddress` 的情况，并增加相同输入形态的回归测试。

## 审查结果

### CRITICAL

无。

### WARNING

- 后端全量测试存在两个与本次改动无关的既有失败：
  - `internal/product`：BLN 参数模型 XML 元数据期望 380，实际 329。
  - `internal/provision`：BaiBNQ 模板字段 `INTERFACE.Gateway` 无法解析。

### INFO

- 修复仅扩展 MAC 参数候选路径，不改变数据库结构、API 或前端字段。
- 直接路径优先级保持不变；字面量 `{i}` 路径在数字实例遍历前作为兼容兜底。
- 未引入运营商硬编码、SQL、认证、资源管理或公共接口变化。

## 验证

- `go build ./...`：通过。
- `go test ./internal/device/... -count=1`：通过。
- `go test ./...`：未全绿，失败项见 WARNING，均不在本次改动范围。
- `git diff --check`：通过。
