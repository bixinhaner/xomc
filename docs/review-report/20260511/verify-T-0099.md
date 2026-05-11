# Verify Report — T-0099 (北向 push engine 接入 active server)

**Task**: T-0099 F08 北向 push engine 接入 active server
**Sprint**: sprint-cleanup-2026-05
**Type**: feat (refactor character)
**Date**: 2026-05-11

## S3 出口门
- [x] `go build ./...` ✓
- [x] `go test ./internal/northbound/... -race` ✓（3 pre-existing failure 已经在 main 存在，与本变更无关）
- [N/A] 前端 typecheck（无前端改动）
- [x] 无新 TODO/FIXME/panic
- [x] 无 carrier 硬编码
- [x] 无新 any/interface{}

## S4 验证

### 新接口 / 端点
- N/A — 本任务为内部 reload 机制，REST API 未改动

### 新 metric / log（grep 命中数）
| 名字 | 类型 | 命中 |
|------|-----|------|
| `RefreshSuccessCount()` | metric 暴露 | 3+ |
| `RefreshFailureCount()` | metric 暴露 | 2+ |
| `active_target_refreshed` | log msg | 2 |
| `active_target_refresh_failed` | log msg | 1 |
| `northbound_server_changed_received` | log msg | 1 |
| `SubjectNorthboundServerChanged` | event subject | 10+ 处 |

### 测试覆盖
- `active_target_test.go` 6 个用例：Upsert / NoActive_RemovesExisting / ProviderError /
  NilProvider_Noop / SwitchUpdatesURL / HandleServerChanged_TriggersRefresh
- `server_service_test.go` 新增 3 个：SetActive_PublishesEvent / Update_PublishesEvent /
  GetActiveForPush（2 case）

### Loop closure 验证
| 场景 | 行为 |
|------|------|
| UI 切换主备 → ServerService.SetActive | 发 SubjectNorthboundServerChanged 事件（已测）|
| Engine 订阅事件 | handleServerChanged → RefreshActiveTarget（已测）|
| RefreshActiveTarget | GetActiveForPush → 更新 ActiveTargetID target URL（已测）|
| UI 编辑 host/port | 同链路：Update 发 event → Engine reload → 下次推送命中新 host |
| 启动期兜底 | modules.go 直接调用 pushEngine.RefreshActiveTarget(ctx) 一次 |

## 设计偏差

- **OSS 协议 / DataTypes 默认值**：Refresh 时构建的 target.DataTypes = `["alarm", "pm", "config"]` 是 sensible default；运营商 OSS 自己通过端侧过滤决定接收。如未来需要 UI 配置每个 active server 的 DataTypes，需扩 `northbound_servers` 表 + ActiveServerInfo 字段（未来 task）
- **URL scheme**：当前固定 `http://`；TLS 支持留待 R-001 闭环（不在本任务范围）

## Risk
- 无新风险；R-003（push engine 失效）状态保持

