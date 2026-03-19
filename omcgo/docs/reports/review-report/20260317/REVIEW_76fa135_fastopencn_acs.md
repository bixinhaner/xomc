# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-17 |
| 基线 | 76fa135 |
| 作者 | fastopencn |
| Scope | acs |
| 结论 | **PASS_WITH_WARNINGS** |

## 变更概述

将 ACS 设备级限流器从 `sync.Map`（无生命周期管理、无容量上限）重构为 `hashicorp/golang-lru/v2`，实现：
- O(1) 淘汰最久未访问设备（替代 O(n) 遍历）
- 后台定期清理不活跃设备，利用 LRU 有序 Keys() 提前终止
- 可配置的 burst、maxDevices、cleanup 间隔/超时
- Prometheus 指标：限流拒绝计数 + 追踪设备数
- 优雅关闭：stopCh 终止 cleanup 协程

## 变更文件 (13)

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/acs/ratelimit.go` | 重写 | lru.Cache 替代 sync.Map + mutex |
| `internal/acs/ratelimit_test.go` | 重写 | 12 个测试覆盖 LRU 淘汰/cleanup/并发 |
| `internal/acs/metrics.go` | 增强 | +2 Prometheus 指标 |
| `internal/acs/handler.go` | 增强 | 限流拒绝时 Inc metrics |
| `internal/acs/handler_test.go` | 适配 | 构造函数签名变更 |
| `internal/acs/server.go` | 增强 | 启动 cleanup / Shutdown 停止 / NewDefaultDeps 签名 |
| `internal/core/appconfig/config.go` | 增强 | RateLimitConfig 5 个字段 |
| `cmd/acs/main.go` | 适配 | 传入 cfg.RateLimit 整体 |
| `cmd/acs/etc/config.{dev,test,prod}.yaml` | 更新 | 新限流配置字段 |
| `go.mod` / `go.sum` | 依赖 | +golang-lru/v2 v2.0.7 |

## 审查详情

### PASS 项

- **资源泄漏**: `StartCleanup` goroutine 通过 `stopCh` 可停止，`Shutdown` 中调用 `Stop()`
- **并发安全**: `lru.Cache` 内部线程安全，`Stop()` 幂等（select + close 模式）
- **测试覆盖**: 12 个测试覆盖基础限流、LRU 淘汰顺序、cleanup 提前终止、并发 50 协程、默认值
- **配置完整性**: dev/test/prod 三环境一致，均有合理默认值
- **向后兼容**: `handler_test.go` 已适配新签名

### INFO 项

1. **`Allow()` Get+Add 非原子窗口**: 两个协程同时为同一新设备创建 limiter，后者覆盖前者。影响仅为多放行 1 次请求，限流场景可接受，无需加锁。
2. **`RateLimitDeviceCount` 指标已声明未运行时采集**: 当前仅注册了 Gauge，未在定时任务中调用 `DeviceCount()` 填充。可后续在 session reaper 或独立 goroutine 中采集。

### 无 CRITICAL / WARNING 项
