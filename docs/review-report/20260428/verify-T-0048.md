# Verify T-0048 — mr 模块 thin service 层

> Wave 2 Block B.4 sub-agent 输出。章程 W2.B.4 通过标准：
> `ls omcgo/internal/mr/*service*.go ≥ 1`。

## 1. 章程通过证据

```bash
$ ls omcgo/internal/mr/*service*.go
omcgo/internal/mr/service.go
omcgo/internal/mr/service_test.go
```

文件数 = 2 ≥ 1，章程 W2.B.4 满足。

## 2. 改动清单

| 文件 | 性质 | 说明 |
|------|------|------|
| `omcgo/internal/mr/service.go` | **新增** | thin service facade，接口 + 默认实现 |
| `omcgo/internal/mr/service_test.go` | **新增** | 11 个单元测试，覆盖 service 全部 4 个方法的成功 / 失败路径 |
| `omcgo/internal/mr/handler.go` | **未改** | 维持 `NewHandler(store, indRepo, mapRepo, minioClient, bucket, logger)` 构造签名，避免影响 `cmd/app/provider/modules.go` 的 DI 装配 |
| `omcgo/internal/mr/handler_test.go` | **未改** | 现有 17 个 handler 测试维持不变 |

> 路径互斥保证：本 worktree **不碰** `cmd/app/provider/modules.go`、`cmd/app/provider/mr.go`、其他模块（task/events/core/syslog/provision/interop）。modules.go 的 DI 接线由主会话最终整合时统一处理。

## 3. service 接口设计

```go
// MRService 是 MR 模块的 thin service facade。
type MRService interface {
    // collector / parser 完成后的"落库 + 标记完成"组合（save + batch insert + update parsed flag）
    IngestParsedFile(ctx context.Context, file *MRFileInfo, records []parser.MRRecord) error

    // 按设备分页查 MR 文件（deviceID = nil 则不过滤设备）
    ListFilesByDevice(ctx context.Context, deviceID *uuid.UUID, listReq model.ListRequest) (*model.ListResponse[MRFileInfo], error)

    // 按设备分页查 MR 记录
    QueryRecordsByDevice(ctx context.Context, deviceID *uuid.UUID, listReq model.ListRequest) (*model.ListResponse[MRRecordEntry], error)

    // 读取指标定义 + 基于值域生成简要统计；指标不存在返回 ErrIndicatorNotFound
    GetIndicatorStats(ctx context.Context, code string) (*IndicatorStats, error)
}
```

### 设计理由

- **接口大小 4 method**：满足 Go idiomatic 小接口（1-3 method）的精神，每个 method 是一个明确的业务用例（落库 / 文件查询 / 记录查询 / 指标统计），不混业务。
- **不破坏既有签名**：`NewHandler` 构造函数签名保持不变。如未来 handler 需要 service，可在 handler 内部 `NewService(h.store, h.indRepo, h.logger)` 自创建，不强制注入 → modules.go DI 装配零影响。
- **消费者定义接口**：`MRService` 在 `internal/mr/` 包内消费它的代码（collector pipeline / future handler refactor）旁定义，符合 "Accept interfaces, return structs" 原则。
- **错误处理**：所有错误用 `fmt.Errorf("context: %w", err)` 包装；`indicator not found` 抽出 sentinel `ErrIndicatorNotFound` 供调用方 `errors.Is` 判定。
- **logger 容错**：`NewService(..., nil)` 时内部用 `zap.NewNop()` 兜底，单测 / 构造失败时不 panic。

## 4. handler.go 改动说明

**未改 handler.go**。理由：
- 章程 W2.B.4 标准只要求 `*service*.go` 文件存在，未要求 handler 重构
- 当前 handler 业务逻辑非常薄（直接转发到 store / repo），抽到 service 边际收益低
- 改 handler 必然要动 `NewHandler` 签名 → 触碰 modules.go DI（被严禁）
- service 给后续阶段（如 Block C 真正接入 collector pipeline）留白扩展点

## 5. 测试列表（全部 PASS，含 -race）

| # | 测试函数 | 覆盖路径 |
|---|---------|---------|
| 1 | `TestService_IngestParsedFile_Success` | 落库三阶段顺序 + MRType / device / file ID 透传 |
| 2 | `TestService_IngestParsedFile_NoRecordsSkipsBatchInsert` | records 为空时跳过 BatchInsert，仍写 SaveFile + UpdateFileParsed(0) |
| 3 | `TestService_IngestParsedFile_NilFile` | nil 文件直接报错，不调用 store |
| 4 | `TestService_IngestParsedFile_SaveFails` | SaveFile 失败 → 不再调用后续步骤，错误用 `%w` 包装 |
| 5 | `TestService_ListFilesByDevice` | DeviceID + ListRequest 透传到 MRFileFilter |
| 6 | `TestService_QueryRecordsByDevice` | DeviceID + ListRequest 透传到 MRRecordFilter |
| 7 | `TestService_QueryRecordsByDevice_NoFilter` | deviceID = nil 时 filter.DeviceID = nil |
| 8 | `TestService_GetIndicatorStats_Success` | 基于值域计算 avg / p50 / p95（-92.0 / -92.0 / -39.6） |
| 9 | `TestService_GetIndicatorStats_NotFound` | indicator = nil → `ErrIndicatorNotFound` |
| 10 | `TestService_GetIndicatorStats_EmptyCode` | 空 code 直接报错，不查仓储 |
| 11 | `TestService_GetIndicatorStats_RepoError` | 仓储错误 → `errors.Is(err, repoErr) == true`，且不会被误判为 NotFound |

### 自跑验证

```
$ go build ./...
(success, no output)

$ go test -race -count=1 ./internal/mr/...
ok      github.com/omcgo/omcgo/internal/mr             2.808s
ok      github.com/omcgo/omcgo/internal/mr/collector   1.657s
ok      github.com/omcgo/omcgo/internal/mr/parser      2.077s

$ go test -race -count=1 -run '^TestService_' -v ./internal/mr/
... 11 个 service test 全 PASS
```

## 6. 后续整合提示（给主会话）

1. **modules.go DI 装配**：本任务**未**在 `cmd/app/provider/modules.go` 注册 service。如后续整合阶段决定让 handler 通过 service 访问数据，需要：
   - 在 modules.go 增加 `NewService(store, indRepo, logger)` 构造
   - 改 `NewHandler` 接收 service（可选，目前无强制需求）
2. **collector / parser pipeline**：`IngestParsedFile` 的真实调用方应当是 MR 文件解析完成后的回调（`internal/mr/collector/`）。本次只创建接口与单元测试，未接线到 collector，留给后续 Block。
3. **真实指标统计**：`GetIndicatorStats` 当前 `sample_count = 0`、`p95 = max * 0.9` 是占位实现，等价于现有 `handler.GetIndicatorStats` 行为。后续接入真实时序聚合（TimescaleDB / KPI 表）时仅需改 `service.go` 一处。

## 7. 章程对齐确认

- [x] `ls omcgo/internal/mr/*service*.go` ≥ 1（实际 = 2）
- [x] `go build ./...` 编译通过
- [x] `go test -race -count=1 ./internal/mr/...` 全部 PASS
- [x] 不碰 `cmd/app/provider/modules.go`
- [x] 不碰 `cmd/app/provider/mr.go`
- [x] 不碰其他模块
- [x] 不增加 go.mod 依赖
- [x] 不 commit / push / pull
