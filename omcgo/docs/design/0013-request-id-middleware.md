# 请求 ID 中间件设计

## 1. 需求背景

在分布式系统中，一个请求可能经过多个服务和处理层。为了便于问题排查和日志追踪，需要为每个请求生成唯一标识符，并在所有相关日志中记录该标识符。

## 2. 设计目标

1. 为每个外部请求生成唯一的 Request ID
2. 支持从上游服务接收 Request ID（链路追踪）
3. 在所有日志中自动包含 Request ID
4. 在响应头中返回 Request ID，便于客户端追踪
5. 适用于所有对外服务（App REST API、ACS TR069）

## 3. 技术方案

### 3.1 Request ID 生成策略

```
优先级：
1. 如果请求头中已存在 X-Request-ID，直接使用（支持链路追踪）
2. 否则，生成新的 Request ID

格式：req-{timestamp}-{random}
示例：req-20260319150430-a1b2c3d4
```

### 3.2 Context 存储

使用 Go context 存储 Request ID，确保在整个请求生命周期内可访问。

```go
// internal/core/context/request_id.go
package context

import "context"

type ctxKey int

const (
    requestIDKey ctxKey = iota
)

// WithRequestID 将 Request ID 存入 context
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID 从 context 获取 Request ID
func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(requestIDKey).(string); ok {
        return id
    }
    return ""
}
```

### 3.3 日志集成

创建 Request ID 感知的日志辅助函数。

```go
// internal/core/logger/context_logger.go
package logger

import (
    "context"
    "go.uber.org/zap"
)

// L 从 context 获取带有 Request ID 的 logger
func L(ctx context.Context) *zap.Logger {
    logger := zap.L() // 或从全局获取

    if requestID := GetRequestID(ctx); requestID != "" {
        return logger.With(zap.String("request_id", requestID))
    }
    return logger
}

// 便捷方法
func Info(ctx context.Context, msg string, fields ...zap.Field) {
    L(ctx).Info(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
    L(ctx).Error(msg, fields...)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
    L(ctx).Debug(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
    L(ctx).Warn(msg, fields...)
}
```

### 3.4 Gin 中间件（App REST API）

```go
// internal/core/middleware/request_id.go
package middleware

import (
    "crypto/rand"
    "encoding/hex"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/omcgo/omcgo/internal/core/context"
)

const (
    RequestIDHeader = "X-Request-ID"
)

// RequestID Gin 中间件，为每个请求生成或传递 Request ID
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 尝试从请求头获取
        requestID := c.GetHeader(RequestIDHeader)

        // 2. 如果不存在，生成新的
        if requestID == "" {
            requestID = generateRequestID()
        }

        // 3. 存入 context
        ctx := context.WithRequestID(c.Request.Context(), requestID)
        c.Request = c.Request.WithContext(ctx)

        // 4. 设置响应头
        c.Header(RequestIDHeader, requestID)

        // 5. 设置到 gin context（便于直接访问）
        c.Set("request_id", requestID)

        c.Next()
    }
}

// GenerateRequestID 生成 Request ID（导出供 ACS 等其他服务使用）
func GenerateRequestID() string {
    return generateRequestID()
}

func generateRequestID() string {
    timestamp := time.Now().Format("20060102150405")
    random := make([]byte, 4)
    rand.Read(random)
    return "req-" + timestamp + "-" + hex.EncodeToString(random)
}
```

### 3.5 ACS Handler 集成

```go
// internal/acs/handler.go 中添加

import (
    "github.com/omcgo/omcgo/internal/core/context"
    "github.com/omcgo/omcgo/internal/core/middleware"
)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. 生成或获取 Request ID
    requestID := r.Header.Get("X-Request-ID")
    if requestID == "" {
        requestID = middleware.GenerateRequestID()
    }

    // 2. 存入 context
    ctx := context.WithRequestID(r.Context(), requestID)

    // 3. 更新请求
    r = r.WithContext(ctx)

    // 4. 设置响应头
    w.Header().Set("X-Request-ID", requestID)

    // 后续处理使用 ctx...
    h.handleRequest(ctx, w, r)
}
```

### 3.6 配置项（可选）

```yaml
# 可选：是否启用 Request ID（默认启用）
server:
  request_id:
    enabled: true
    header_name: "X-Request-ID"
```

## 4. 日志输出示例

启用前：
```json
{"level":"info","ts":"2026-03-19T15:04:30.123+0800","msg":"device registered","serial_number":"BCTest00123456"}
```

启用后：
```json
{"level":"info","ts":"2026-03-19T15:04:30.123+0800","msg":"device registered","request_id":"req-20260319150430-a1b2c3d4","serial_number":"BCTest00123456"}
```

## 5. 文件清单

| 文件 | 说明 |
|------|------|
| `internal/core/context/request_id.go` | 新建 - Request ID context 工具 |
| `internal/core/logger/context_logger.go` | 新建 - Context 感知的日志封装 |
| `internal/core/middleware/request_id.go` | 新建 - Gin 中间件 + 生成函数 |
| `internal/acs/handler.go` | 修改 - 添加 Request ID 处理 |
| `cmd/app/router/router.go` | 修改 - 注册中间件 |

## 6. 实施步骤

1. **Phase 1**: 创建基础设施
   - 创建 `internal/core/context/request_id.go`
   - 创建 `internal/core/logger/context_logger.go`
   - 创建 `internal/core/middleware/request_id.go`

2. **Phase 2**: 集成到服务
   - App: 在 router 中注册 Gin 中间件
   - ACS: 在 handler 中添加 Request ID 处理

3. **Phase 3**: 更新关键路径日志
   - 更新 ACS 关键日志点使用 context logger
   - 更新 Device 管理关键日志点

4. **Phase 4**: 全面推广
   - 逐步更新所有日志调用

## 7. 测试验证

```bash
# 测试 Request ID 生成
curl -v http://localhost:8081/api/v1/devices
# 响应头应包含: X-Request-ID: req-xxxxxxxxx

# 测试 Request ID 传递
curl -v -H "X-Request-ID: my-trace-123" http://localhost:8081/api/v1/devices
# 响应头应包含: X-Request-ID: my-trace-123

# 测试日志包含 Request ID
tail -f ~/data/logs/omcgo/app/app.log | grep request_id
```

## 8. 与 SQL 日志的联动

当 SQL 日志功能启用时，SQL 日志也会包含当前请求的 `request_id`，实现全链路追踪：

```json
{"level":"debug","logger":"sql","msg":"SQL","request_id":"req-20260319150430-a1b2c3d4","sql":"SELECT * FROM devices","duration":"2.345ms"}
```

详见 `sql-logging.md` 文档。
