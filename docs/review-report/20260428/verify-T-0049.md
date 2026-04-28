# Verify T-0049 — syslog Service Layer

- **Backlog**: T-0049
- **Charter**: Wave 2 / Block B.4 — syslog 模块 service 层
- **Worktree**: `agent-ad9e836f`
- **Branch**: `worktree-agent-ad9e836f`
- **Date**: 2026-04-28

## Charter Pass Criterion

```
ls omcgo/internal/syslog/*service*.go ≥ 1
```

实际产出 **2** 个匹配文件（达标）：

```
omcgo/internal/syslog/service.go
omcgo/internal/syslog/service_test.go
```

## Scope

为 `internal/syslog/` 新增 thin service 层，封装 syslog 模块的高层业务（系统日志查询、网元消息日志查询、按设备查询便利接口），并提供 `SyslogService` 接口供下游 mock。**未改动** `handler.go` / `handler_test.go` / `model.go` / `repository.go` / `pg_repository.go`，因此 `cmd/app/provider/syslog.go`（如存在）的 DI 调用不受影响。

## Design Notes

| 维度 | 决策 |
|------|------|
| 接口规模 | `SyslogService` 暴露 3 个方法（list system logs、list NE messages、按设备查询便利接口） |
| 命名 | `Service` 为导出 struct（贴齐 license / dashboard 同 F06 兄弟模块）；`SyslogService` 为消费方接口 |
| 错误处理 | `fmt.Errorf("...: %w", err)` 包装；非法输入走 `commonerrors.NewBusinessError(4001, ..., ErrInvalidInput)` |
| 输入校验 | `TimeWindow.Validate()`、`validateRange()`、`uuid.Nil` 检查在 service 边界 fail-fast |
| 不可变 | service 不修改入参 filter；构造 `NEMessageLogFilter` 时拷贝 deviceID 到本地变量再取地址 |
| 编译期断言 | `var _ SyslogService = (*Service)(nil)` 防止后续接口/实现漂移 |

## Self-Verification

```bash
$ cd omcgo
$ go build ./...                                 # PASS（无输出）
$ go test -race -count=1 ./internal/syslog/...   # ok  github.com/omcgo/omcgo/internal/syslog 1.456s
$ ls internal/syslog/*service*.go                # 2 files
```

### Test Inventory

| Test | Coverage |
|------|----------|
| `TestService_ListSystemLogs_Success` | repo 直通 |
| `TestService_ListSystemLogs_InvalidRange` | start > end → ErrInvalidInput |
| `TestService_ListSystemLogs_RepoError` | repo 错误用 `%w` 包装 |
| `TestService_ListNEMessageLogs_Success` | filter 正确传递 |
| `TestService_ListNEMessageLogs_InvalidRange` | 时间窗校验 |
| `TestService_ListNEMessagesByDevice_Success` | filter 组装正确 |
| `TestService_ListNEMessagesByDevice_NilDeviceID` | uuid.Nil → BusinessError(4001) |
| `TestService_ListNEMessagesByDevice_InvalidWindow` | TimeWindow.Validate 联动 |
| `TestNewService_NilLogger` | nil logger 不 panic |
| `TestTimeWindow_Validate` | 5 sub-tests，覆盖 both nil / only start / only end / 正常 / 反向 |

合计 10 个顶层测试，含 5 个 sub-test（合计 15 个 -run 单元），远超章程 ≥ 3 要求。所有测试通过 `-race` 检测。

## Out-of-Scope（严守互斥）

- 未碰 `cmd/app/provider/modules.go`、`cmd/app/provider/syslog.go`（主会话整合时统一处理）
- 未改其他模块（task/events/core/mr/provision/interop）
- 未新增 go.mod 依赖
- 未 commit / push / pull

## Status

`.wave-status.txt` = `DONE`
