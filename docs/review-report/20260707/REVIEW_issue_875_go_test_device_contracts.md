# Review: Issue #875 go test device contracts

| 项目 | 值 |
|------|----|
| 日期 | 2026-07-07 |
| 范围 | `git diff main -- omcgo/internal/.../*_test.go` |
| Issue | #875 修复 go test 因设备仓库接口与测试契约漂移失败 |
| 结论 | PASS |

## Standards

未发现违反仓库标准的问题。

- 改动保持在后端测试文件与本审查记录内，生产代码未变更。
- 测试替身同步当前 `DeviceRepository` 接口，没有扩宽生产接口。
- 未引入 ORM、字符串拼接 SQL、运营商硬编码、裸 `panic` 或安全敏感路径变更。
- 已执行 `gofmt`。

## Spec

未发现与 Issue #875 不一致的问题。

- 已同步手写 `DeviceRepository` mock 的 `GetDeletedBySerialNumber`。
- 已将测试 mock 的 `SearchDevices` 签名同步为 `[]model.DeviceVisibilityGrant`。
- 已将 `scanGeoDeviceRow` fixture 同步到 19 列，并覆盖可空 `highest_alarm_severity` 扫描。
- 已将参数 schema 测试调整为验证 model-only 参数展示，避免回退到“只展示实际上报参数”的旧契约。

## DoD

- 后端 `go build ./...`: PASS
- 后端 `go test ./...`: PASS
- 后端 `golangci-lint run`: 未运行，本机缺少 `golangci-lint`（`command not found`）
- 前端改动: N/A
- 新 REST 端点: N/A
- 数据库迁移: N/A
- 安全敏感路径: N/A

## Summary

Standards findings: 0. Spec findings: 0. Worst issue: none.
