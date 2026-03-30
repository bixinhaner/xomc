# Code Review Report

| Item | Value |
|------|-------|
| Date | 2026-03-30 |
| Author | chenbo01 |
| Reviewer | Claude Opus 4.6 |
| Base Commit | 46ebed9 |
| Scope | storage (MinIO file storage architecture) |
| Verdict | **PASS_WITH_WARNINGS** |

## Summary

Implementation of MinIO file storage architecture design (doc 0024). Creates unified `storage` package for bucket routing and path construction, standardizes TR069 FileType constants, refactors 3 consumer modules to use the new package, and adds 7th bucket (`omc-exchange`) across all configs.

## Changes Overview

| Category | Files | Lines |
|----------|-------|-------|
| New packages | `pkg/tr069/filetype.go`, `internal/core/storage/{path,router}.go` | +202 |
| Design doc | `docs/design/0024-*.md` | +1662 |
| Config updates | 12 YAML + `appconfig/config.go` + `minio.go` | +14 |
| Refactor: upload handler | `acs/upload/handler.go`, `token_test.go` | +76/-80 |
| Refactor: transfer bridge | `transfer/bridge.go`, `bridge_test.go`, `cmd/worker/main.go` | +67/-89 |
| Refactor: software service | `software/service.go` | +3/-4 |
| Script | `scripts/minio_lifecycle.sh` | +48 |

## Findings

### WARNING (2)

**W1. `ObjectPath()` 不验证空参数**
- File: `internal/core/storage/path.go:12`
- 当 `carrier` 或 `deviceSN` 为空时会产生双斜杠路径（如 `running//2026/03/30//file.xml`）
- MinIO 可接受此类路径，但影响可读性和 ListObjects 前缀过滤
- 建议: 后续迭代添加参数验证或调用方保证非空

**W2. 固件路径前缀变更**
- File: `internal/software/service.go:61`
- 路径前缀从 `firmware/` 变更为 `img/`（符合设计文档 0024 §4.3）
- 如有已存储的固件文件需注意迁移（当前项目阶段无历史数据，影响极小）

### INFO (3)

**I1. 未知 FileType 默认行为变更**
- `upload/handler.go`: 未知类型从默认到 PMFiles bucket 改为 Logs/running bucket
- 更合理的默认行为，未知文件归入日志而非 PM 数据

**I2. `normalizeFileType` 丢弃了 "Log" 别名（首字母大写）**
- 原代码有 `case "6", "Log", "LOG":`，新代码 `case "6", "LOG":`
- 由于先执行了 `strings.ToUpper()`，"Log" 会转为 "LOG" 再匹配，行为不变

**I3. `classifyFileType` 新增了 FileType 1/2/9 识别**
- `transfer/bridge.go`: 新增固件、补丁、抓包文件类型识别，扩展了覆盖范围

## Checklist

- [x] Go 命名规范 (PascalCase/camelCase)
- [x] 错误处理 (`fmt.Errorf` wrap)
- [x] 无 SQL 注入风险 (无 SQL 变更)
- [x] 无运营商硬编码 (`if carrier == "cmcc"`)
- [x] 无 ORM 使用
- [x] 测试覆盖 (upload/transfer 测试已更新并通过)
- [x] 接口设计合理 (`BucketAndCategory` 纯函数，易测试)
- [x] 配置一致性 (12 YAML + K8s 全部更新)
- [x] 无安全漏洞 (路径遍历保护保留)
- [x] 编译通过 (`go build ./...`)
- [x] 测试通过 (`go test` 相关包全部通过)

## Build & Test

```
go build ./...          → PASS
go test ./...           → PASS (provision 4 failures pre-existing, unrelated)
```
