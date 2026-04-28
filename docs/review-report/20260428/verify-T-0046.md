# T-0046 — `internal/events` 增加 `EventService` facade 与单元测试

> Wave 2 Block B.2 / Backlog T-0046
> Worktree: `agent-a2be7eaf`
> 任务：补充 `internal/events` 的高层服务接口（facade）+ 把覆盖率拉到 ≥ 60%

---

## 1. Baseline（执行前现状）

`internal/events/` 下仅 3 个源码文件、**无任何测试**：

```
internal/events/
├── handler.go        # SSEHandler — Gin SSE 端点，依赖 admin.JWTService
├── hub.go            # MessageHub — 内存 channel 分发
└── store.go          # MessageStore + RedisMessageStore
```

执行 `go test -cover ./internal/events/...` 之前：覆盖率 = **0.0%**（无测试文件）。

---

## 2. EventService 接口设计

`internal/events/service.go` 新增。

### 接口（小而紧凑，3 个方法，符合 Go "small interface" 风格）

```go
type EventService interface {
    Publish(ctx context.Context, userID string, msg *SSEMessage) error
    SubscribeQuery(ctx context.Context, userID string) (<-chan *SSEMessage, error)
    QueryHistory(ctx context.Context, userID string, afterID string, limit int) ([]*SSEMessage, error)
}
```

### 实现 `EventServiceImpl`

- `NewEventService(hub *MessageHub, store MessageStore, logger *zap.Logger) *EventServiceImpl`
- 遵循 "Accept interfaces, return structs"：构造函数返回具体类型，调用方按需依赖接口
- 仅薄封装现有 `MessageHub` / `MessageStore`，**不引入新状态、不修改原语义**
- 编译期断言 `var _ EventService = (*EventServiceImpl)(nil)`

### 附加便利方法（在具体类型上，不进 interface）

| 方法 | 用途 |
|------|------|
| `PublishJSON(ctx, userID, eventType, payload any)` | 自动 marshal payload，省掉调用方一次序列化 |
| `PublishGlobal(ctx, msg)` | 全网广播（不持久化） |
| `Unsubscribe(userID)` | 解订阅，HTTP handler defer 用 |
| `Hub() / Store()` | 给 `SSEHandler` 等需要直接操作底层组件的内部使用，明确加注释"业务模块勿用" |

### 不强制注入到 `modules.go`

按 Wave 2 Block B.2 路径互斥要求，**未触碰** `cmd/app/provider/modules.go`，
由主会话整合时决定何时注入。

---

## 3. 测试改动

### 新增 4 个测试文件

| 文件 | 测试数 | 说明 |
|------|-------|------|
| `service_test.go` | 18 | EventService 接口契约 + 边界条件 + nil store 兜底 |
| `hub_test.go` | 8 | MessageHub Publish/Subscribe/Unsubscribe/PublishGlobal/吞吐削峰 |
| `store_test.go` | 5 | RedisMessageStore（用 miniredis）— Store/GetSince/afterID/limit/TTL |
| `handler_test.go` | 8 | SSEHandler — 路由注册、401 路径、Bearer & query token、消息分发、Last-Event-ID 回放、parseTokenSimple |

### 关键测试列表

- `TestService_Publish_Delivers` — 端到端发布订阅
- `TestService_Publish_ValidatesArgs` — 空 userID / nil msg 报错路径
- `TestService_PublishJSON_MarshalErrorPropagates` — channel 不可序列化的失败路径
- `TestService_PublishGlobal_DeliversToAllAndAutoIDs` — 多订阅者广播
- `TestService_QueryHistory_AfterID` — 断线重连过滤
- `TestService_QueryHistory_DefaultsLimitTo50` — limit≤0 默认值
- `TestService_QueryHistory_NilStore` — store 缺省时返回 `nil, nil`
- `TestService_QueryHistory_StoreErrorPropagates` — 错误 wrap 路径
- `TestNewEventService_PanicsOnNilHub/Logger` — 强制 fail-fast
- `TestHub_Publish_FullChannelDropsOldest` — 缓冲满 256 时丢老消息不阻塞
- `TestHub_Subscribe_KicksExistingChannel` — 同 user 重订阅踢老连接
- `TestRedisStore_TTLApplied` — Pipeline `Expire` 命中
- `TestSSEHandler_Stream_DeliversPublishedMessage` — handler→hub→ch 真链路（用 200ms ctx 截断）
- `TestSSEHandler_Stream_LastEventID_TriggersReplay` — 回放分支
- `TestParseTokenSimple_*` — fallback JWT 解析（含错误签名算法路径）

### 测试基础设施

- `fakeStore` — service_test/hub_test 共用的 in-memory MessageStore（线程安全）
- `miniredis.RunT` — store_test 用真实 Redis 协议但内存运行
- `httptest.NewRecorder` + 200ms `context.WithTimeout` — handler_test 切断 SSE 长连接
- 全程 `-race` 通过

---

## 4. 最终覆盖率

```
$ go test -coverprofile=cov_events.out ./internal/events/...
ok  	github.com/omcgo/omcgo/internal/events	1.347s	coverage: 87.6% of statements

$ go tool cover -func=cov_events.out | tail -1
total:							(statements)		87.6%
```

**结果：87.6% ≥ 60% 阈值**（远超章程 W2.B.2 Pass 标准）。

### 各文件覆盖：

| 文件 | 覆盖率 |
|------|-------|
| service.go | 95%+（除 panic 兜底外全覆盖） |
| hub.go | 100% / 100% / 67% / 75% / 100% / 100%（已订阅时 PublishSimple 内部 select 的"channel 满"分支没专门测，可接受） |
| store.go | 100% / 100% / 82% / 84% |
| handler.go | 73% / 92% / 67% / 100% / 91% |

### Race & build

```
$ go test -race -count=1 ./internal/events/...
ok  	github.com/omcgo/omcgo/internal/events	2.482s

$ go build ./...   # 全仓 build
（无输出，通过）

$ go vet ./internal/events/...
（无输出，通过）
```

---

## 5. 路径互斥/隔离遵守情况

- 仅修改 `internal/events/`（新增 4 文件 + 1 service.go）
- **未触碰** `cmd/app/provider/modules.go`
- **未触碰** `internal/core/event/`、`internal/{task,mr,syslog,provision,interop,notification}` 等其他模块
- 未新增 `go.mod` 依赖（miniredis、testify、jwt、uuid、zap、gin 全部已存在）
- 未 commit/push/pull/动 backlog 或 charter

---

## 6. 章程 W2.B.2 Pass 自检

| 条件 | 结果 |
|------|------|
| `ls internal/events/*service*.go` ≥ 1 文件 | ✅ 2 个（service.go + service_test.go） |
| 覆盖率 ≥ 60% | ✅ 87.6% |
| `go build ./...` 通过 | ✅ |
| `go test -race -count=1 ./internal/events/...` 通过 | ✅ |
| 接口小（1-3 method） | ✅ 3 个方法 |
| 不强制改 modules.go | ✅ 由主会话整合 |

DONE.
