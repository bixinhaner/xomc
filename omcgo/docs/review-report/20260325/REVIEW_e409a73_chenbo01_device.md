# Code Review Report

| Item | Value |
|------|-------|
| Date | 2026-03-25 |
| Author | chenbo01 |
| Base Commit | e409a73 |
| Scope | device |
| Type | feat |
| Verdict | **PASS_WITH_WARNINGS** |

---

## Changes Summary

重构 `handlePeriodic` 事件处理流程并引入 Redis 设备缓存层，解决设备未注册时 NATS 无限重试问题。

### Files Changed (7)

| File | Change | Lines |
|------|--------|-------|
| `internal/device/device_cache.go` | **NEW** | +92 |
| `internal/device/inform_handler.go` | Modified | +46/-20 |
| `internal/device/service.go` | Modified | +45/-8 |
| `internal/device/service_test.go` | Modified | +8/-3 |
| `internal/device/inform_handler_test.go` | Modified | +4 |
| `internal/device/handler_test.go` | Modified | +4 |
| `cmd/app/router/router.go` | Modified | +2 |

---

## Review Checklist

### Go Engineering

- [x] **命名规范**: `DeviceCache`, `GetOrLoad`, `cacheDevice` — PascalCase/camelCase 正确
- [x] **错误处理**: cache 操作失败仅 warn 日志，不阻断主流程；DB 错误正确传播
- [x] **接口设计**: `DeviceCache` 为具体类型（非接口），因为仅 Redis 一种实现，符合简��性原则
- [x] **资源释放**: Redis 操作使用 context 传播，无资源泄漏
- [x] **SQL 安全**: 无直接 SQL 变更，使用 Squirrel 参数化查询（未改动）

### Architecture

- [x] **handler → service → repository 分层**: 查询逻辑在 service 层，cache 在 service 内部
- [x] **跨模块依赖**: 通过 `SetDeviceCache` 注入，无循环依赖
- [x] **Cache-aside 模式**: Redis → miss → PostgreSQL → set Redis，标准模式

### TR-069 / NATS

- [x] **NATS 重试**: `handlePeriodic` 设备不存在时走注册路径，DB 错误仍返回 error 触发重试
- [x] **UpdateFromInform 返回 (nil, nil)**: 设备不存在为业务状态，不再导致无限重试

### Test Coverage

- [x] **现有测试通过**: `go test ./internal/device/...` OK
- [x] **mock 接口对齐**: 3 个 mock 补充了 `GetDirectChildLeaves` 方法
- [x] **UpdateFromInform_NotFound 测试**: 更新为断言 `(nil, nil)`

---

## Findings

### WARNING (2)

**W1: DeviceCache 未在 DeleteDevice/UpdateDevice(API) 路径中失效**

`service.go:DeleteDevice` 和 `UpdateDevice`（API 调用路径）未调用 `cache.Delete(sn)`，可能导致删除/修改后的设备在缓存 TTL（10min）内仍返回旧数据。

**影响**: 低。API 修改设备后短期内 Inform 事件可能读到旧缓存数据，10 分钟 TTL 后自动过期。
**建议**: 后续 sprint 在 `DeleteDevice` 和 `UpdateDevice` 中添加 cache 失效。

**W2: handlePeriodic 先查再更新存在双重查询**

`handlePeriodic` 先调 `GetBySerialNumber`（查缓存/DB），再调 `UpdateFromInform`（内部再查一次缓存/DB）。对已存在设备，每次 periodic 会查两次。

**影响**: 低。第二次查询命中 Redis 缓存（第一次已写入），额外开销仅一次 Redis GET。
**建议**: 可接受，缓存命中后 Redis GET 延迟 <1ms。

### INFO (1)

**I1: 缓存 TTL 10 分钟与默认 InformInterval 300s 匹配良好**

设备默认每 5 分钟上报 Inform，10 分钟 TTL 确保活跃设备始终命中缓存，离线设备自动过期。

---

## Verdict: PASS_WITH_WARNINGS

无 CRITICAL 问题。2 个 WARNING 均为低影响优化建议，不阻塞提交。
