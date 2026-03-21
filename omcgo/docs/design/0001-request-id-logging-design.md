# 方案文档：request_id 日志链路追踪完善

> 版本: 1.0
> 日期: 2026-03-21
> 状态: 待确认

---

## 1. 问题分析

### 1.1 现状

当前日志系统已具备 request_id 基础设施：

```go
// logger/logger.go - 已实现
func L(ctx context.Context) *zap.Logger {
    if requestID := GetRequestID(ctx); requestID != "" {
        return logger.With(zap.String("request_id", requestID))
    }
    return logger
}
```

但实际日志输出中缺少 request_id：

```json
{"level":"info","timestamp":"2026-03-21T02:54:43.952Z","caller":"acs/handler.go:202","msg":"ACS Inform processing","device_sn":"1202000752246FD0899","oui":"48BF74","product_class":"FAP/BAIBLQ/SC","events":["2 PERIODIC"],"param_count":37}
```

### 1.2 根本原因

1. **Handler 使用了 `log` (context-aware) 但 SQL Tracer 使用了 `t.logger` (全局 logger)**
   - `log := logger.L(ctx)` 包含 request_id
   - `t.logger` 是独立实例，不自动携带 context 信息

2. **部分代码直接使用 `h.logger` 而非 context-aware logger**
   - 如 `handler.go:69`, `handler.go:663`, `handler.go:670`

3. **Context 未正确传递**
   - SQL 执行时 context 可能丢失 request_id

---

## 2. 解决方案

### 2.1 核心原则

**所有日志必须通过 `logger.L(ctx)` 获取，确保自动携带 request_id**

### 2.2 改造点

#### 2.2.1 ACS Handler 改造

**文件**: `internal/acs/handler.go`

**改造前**:
```go
h.logger.Warn("reaped stale session", ...)
h.logger.Debug("no session cookie", ...)
h.logger.Error("get session by cookie", ...)
```

**改造后**:
```go
// 在 startSessionReaper 中无法获取 context，使用带前缀的 logger
h.logger.Named("reaper").Warn("reaped stale session",
    zap.String("device_sn", entry.DeviceSN),
    zap.String("remote_addr", key.(string)),
    zap.Duration("age", now.Sub(entry.CreatedAt)))

// 其他位置使用 context-aware logger
log := logger.L(r.Context())
log.Debug("no session cookie", zap.Error(err), ...)
```

#### 2.2.2 SQL Tracer 改造

**文件**: `internal/core/components/postgres/tracer.go`

**问题**: `t.logger` 是初始化时创建的命名 logger，不携带 context 信息

**方案**: 保持现有架构，但确保从 context 提取 request_id

当前实现已正确处理：
```go
// TraceQueryEnd 已实现
if requestID := logger.GetRequestID(ctx); requestID != "" {
    fields = append(fields, zap.String("request_id", requestID))
}
```

**需要验证**: 确保 repository 层调用时传递了正确的 context

#### 2.2.3 Repository 层改造

**原则**: 所有数据库操作必须传递 context

**示例** (`internal/device/pg_repository.go`):
```go
// 改造前
func (r *pgRepository) GetByID(id string) (*Device, error) {
    return r.getByID(context.Background(), id)  // 丢失 request_id
}

// 改造后
func (r *pgRepository) GetByID(ctx context.Context, id string) (*Device, error) {
    return r.getByID(ctx, id)  // 保留 request_id
}
```

#### 2.2.4 Gin 中间件改造

**文件**: `internal/core/middleware/requestid.go`

**已有实现** (需确认):
```go
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        ctx := logger.WithRequestID(c.Request.Context(), requestID)
        c.Request = c.Request.WithContext(ctx)
        c.Set("request_id", requestID)
        c.Next()
    }
}
```

### 2.3 日志格式规范

#### 2.3.1 必需字段

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| level | string | 日志级别 | `info`, `error`, `warn`, `debug` |
| timestamp | string | ISO8601 时间 | `2026-03-21T02:54:43.952Z` |
| caller | string | 调用位置 | `acs/handler.go:217` |
| msg | string | 日志消息 | `ACS Inform processing` |
| request_id | string | 请求追踪 ID | `acs-abc123` |

#### 2.3.2 SQL 日志额外字段

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| sql | string | SQL 语句 | `SELECT * FROM devices WHERE id = $1` |
| duration | string | 执行时长 | `1.234ms` |
| params | array | SQL 参数 (可选) | `["device-001"]` |
| rows | int | 影响行数 | `1` |

#### 2.3.3 完整日志示例

```json
{
  "level": "info",
  "timestamp": "2026-03-21T02:54:43.952Z",
  "caller": "acs/handler.go:217",
  "msg": "ACS Inform processing",
  "request_id": "acs-550e8400-e29b-41d4-a716-446655440000",
  "device_sn": "1202000752246FD0899",
  "oui": "48BF74",
  "product_class": "FAP/BAIBLQ/SC",
  "events": ["2 PERIODIC"],
  "param_count": 37
}
```

```json
{
  "level": "debug",
  "timestamp": "2026-03-21T02:54:43.955Z",
  "caller": "postgres/tracer.go:93",
  "msg": "SQL",
  "request_id": "acs-550e8400-e29b-41d4-a716-446655440000",
  "sql": "SELECT * FROM devices WHERE serial_number = $1",
  "duration": "1.234ms",
  "params": ["1202000752246FD0899"],
  "rows": 1
}
```

---

## 3. 实施计划

### 3.1 改造范围

| 模块 | 文件 | 改动量 | 优先级 |
|------|------|--------|--------|
| ACS | `internal/acs/handler.go` | 小 | P0 |
| ACS | `internal/acs/*.go` | 中 | P0 |
| 中间件 | `internal/core/middleware/` | 小 | P0 |
| Repository | `internal/*/pg_*.go` | 中 | P1 |
| Service | `internal/*/*_service.go` | 小 | P1 |

### 3.2 验证方法

1. **单元测试**: 验证 context 中 request_id 正确传递
2. **集成测试**: 检查日志输出包含 request_id
3. **手动验证**:
   ```bash
   # 启动服务
   make run-app

   # 发送请求并检查日志
   curl -H "X-Request-ID: test-123" http://localhost:8080/api/v1/devices

   # 检查日志
   tail -f logs/app.log | grep "test-123"
   ```

---

## 4. 影响评估

### 4.1 风险点

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| context 传递断裂 | request_id 丢失 | 代码审查 + 单元测试 |
| 性能影响 | 极小 (string 传递) | 无需特殊处理 |

### 4.2 兼容性

- **向后兼容**: 是
- **配置变更**: 否
- **数据库变更**: 否

---

## 5. 总结

本方案通过统一使用 `logger.L(ctx)` 获取 context-aware logger，确保所有日志自动携带 request_id，实现完整的请求链路追踪。

**核心改动**:
1. 将 `h.logger` 替换为 `logger.L(ctx)`
2. 确保 repository 方法签名包含 `context.Context` 参数
3. 验证 SQL Tracer 正确提取 request_id

---

## 附录：代码检查清单

```bash
# 检查所有直接使用 h.logger 的地方（应该使用 log = logger.L(ctx)）
grep -rn "h\.logger\." internal/acs/

# 检查所有 repository 调用是否传递 context
grep -rn "context\.Background()" internal/

# 检查所有 SQL 日志是否包含 request_id（运行时验证）
```
