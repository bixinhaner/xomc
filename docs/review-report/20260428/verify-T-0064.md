# T-0064 Verify Report — 敏感信息脱敏（W3.G.3）

> 章程：Wave 3 G.3 — 日志/错误响应不含明文 password/Token/设备密钥
> Sub-agent worktree：`agent-a6eaa975`，分支 `worktree-agent-a6eaa975`
> 日期：2026-04-28

## 1. 改动范围

| 类型 | 路径 | 说明 |
|------|------|------|
| 新建 package | `omcgo/internal/core/redact/redact.go` | 脱敏核心：`SensitiveKeys` / `IsSensitive` / `MaskString` / `MaskAny` / `RedactMap` / `RedactJSON` / zap field helpers (`String`/`Stringp`/`Any`) / `FieldEncoder` |
| 新建测试 | `omcgo/internal/core/redact/redact_test.go` | 18 个 table 用例 + 边界 + zap 集成 |
| 新建桥接 | `omcgo/internal/core/components/logger/redact.go` | 在 logger 包再导出 `RedactString` / `RedactStringp` / `RedactAny`，方便已 import logger 的调用方原地切换 |
| 新建测试 | `omcgo/internal/core/components/logger/redact_test.go` | 6 个用例覆盖 logger 桥接 |
| 修改 | `omcgo/internal/core/errors/errors.go` | `AbortWithError` 在写出 body 前先 marshal → `redact.RedactJSON` → 直接写。当 `err.Error()` 自身是 JSON（上游 SDK 错误常见），也会被解析后脱敏 |
| 修改 | `omcgo/internal/core/errors/errors_test.go` | 新增 `TestAbortWithError_RedactsSensitiveJSONInDetails` / `TestAbortWithError_BusinessError_NoLeakWhenMessageIsSafe` |

## 2. 设计要点

### 2.1 `MaskString` 风格
- 长度 ≤ 6：全部替换为 `*`（保留长度信息便于关联）
- 长度 > 6：`<head 2><***><tail 2>`（如 `secretvalue` → `se***ue`），保留少量信号用于跨日志关联，又不泄露中段
- 同时提供 `MaskAny`：非字符串一律返回 `"***"` 占位，避免 PIN/[]byte/嵌套结构泄露

### 2.2 SensitiveKeys 覆盖
覆盖项目实际使用 + OWASP 推荐：
- 通用：`password` / `passwd` / `pwd` / `secret` / `token` / `bearer` / `jwt`
- HTTP：`authorization` / `auth` / `api_key` / `apikey` / `x-api-key`
- 凭证：`access_token` / `refresh_token` / `id_token` / `client_secret` / `private_key` / `credential` / `credentials` / `session_key` / `session_token`
- 设备/SNMP：`device_password` / `device_secret` / `snmp_community` / `community`

匹配通过 `normalize` 归一化（小写 + 去 `_` / `-`），所以 `Password` / `pass_word` / `X-API-Key` / `refreshToken` 都会命中。

### 2.3 三层脱敏策略
1. **MaskString / MaskAny**：单值层，调用方直接调
2. **RedactMap / RedactJSON**：结构层，递归 + 不变性（不修改入参）
3. **zap.Field 包装**：日志层，`redact.String(key, val)` 在 field 构造时即脱敏，无需改 logger encoder

### 2.4 错误响应集成
`AbortWithError` 改为先 `json.Marshal(resp)` → `redact.RedactJSON(body)` → `Writer.Write`。当 `err` 是被序列化的上游 JSON（如第三方 API 返回的错误 payload），其内嵌的 `password` / `token` 字段也会被脱敏。这是关键的 defense-in-depth 层。

## 3. 严禁项核查

| 严禁 | 检查 |
|------|------|
| 改 `cmd/app/provider/{modules,router}.go` | ✅ 未改 |
| 改 `omcgo/internal/admin/audit/*` | ✅ 未改（只改 `core/errors` 和新建 `core/redact` / `core/components/logger/redact*`） |
| 改 `.github/workflows/*` | ✅ 未改 |
| 改 `omcmb/*` | ✅ 未改 |
| 新增 go.mod 依赖 | ✅ 未改 go.mod；只用 `encoding/json` / `strings` / `go.uber.org/zap`（已有） |
| commit / push / pull / 改 backlog | ✅ 未做 |

## 4. 自跑验证

### 4.1 `go build ./...`
```
（无输出，编译通过）
```

### 4.2 `go test -race -count=1 ./internal/core/redact/... ./internal/core/errors/... ./internal/core/components/logger/...`
```
ok  	github.com/omcgo/omcgo/internal/core/redact	1.285s
ok  	github.com/omcgo/omcgo/internal/core/errors	1.403s
ok  	github.com/omcgo/omcgo/internal/core/components/logger	1.870s
```

### 4.3 `go test -count=1 ./internal/core/...` 全 core 包
```
ok  	github.com/omcgo/omcgo/internal/core/appconfig
ok  	github.com/omcgo/omcgo/internal/core/carrier
ok  	github.com/omcgo/omcgo/internal/core/carrier/cmcc
ok  	github.com/omcgo/omcgo/internal/core/carrier/ctcc
ok  	github.com/omcgo/omcgo/internal/core/carrier/cucc
ok  	github.com/omcgo/omcgo/internal/core/components
ok  	github.com/omcgo/omcgo/internal/core/components/logger
ok  	github.com/omcgo/omcgo/internal/core/components/redisx
ok  	github.com/omcgo/omcgo/internal/core/errors
ok  	github.com/omcgo/omcgo/internal/core/event
ok  	github.com/omcgo/omcgo/internal/core/health
ok  	github.com/omcgo/omcgo/internal/core/middleware
ok  	github.com/omcgo/omcgo/internal/core/model
ok  	github.com/omcgo/omcgo/internal/core/redact
ok  	github.com/omcgo/omcgo/internal/core/reliability
ok  	github.com/omcgo/omcgo/internal/core/storage
ok  	github.com/omcgo/omcgo/internal/core/tracing
```
所有 17 个 core 包 PASS。

### 4.4 `go vet ./...`
全树通过，无警告。

### 4.5 章程 grep
```
$ grep -rn "redact|sanitize|maskPassword|maskToken|MaskString" internal/core/middleware/ internal/core/components/logger/ internal/core/redact/
internal/core/components/logger/redact.go        ✅ 桥接
internal/core/components/logger/redact_test.go   ✅ 测试
internal/core/redact/redact.go                   ✅ 主实现
internal/core/redact/redact_test.go              ✅ 测试

$ grep -rn "TestRedact|TestMask" internal/
internal/core/redact/redact_test.go              ✅
internal/core/components/logger/redact_test.go   ✅
```

## 5. 章程 W3.G.3 Pass 标准

- [x] 脱敏 helper 存在：`internal/core/redact/redact.go`
- [x] 集成测验证不出明文：
  - `TestRedactJSON_Object` / `TestRedactJSON_Nested` 验证 plaintext 不出现在字节流
  - `TestZapIntegration_LoggerNeverEmitsPlaintext` 通过自定义 `zapcore.Core` 捕获字段，逐字段断言无 plaintext
  - `TestAbortWithError_RedactsSensitiveJSONInDetails` 模拟上游 JSON 错误透传，断言 `password` / `token` 不进响应
  - `TestMaskString_NoLeakOfMiddle` 防止"长 mask 露中段"回归
- [x] 关键路径调用：
  - `errors.AbortWithError` → `redact.RedactJSON` （所有 API 错误响应自动脱敏）
  - `logger.RedactString` / `logger.RedactStringp` / `logger.RedactAny`（业务调用方按需用）

## 6. 后续 follow-up（不在本任务范围）

- 在 admin/auth handler 实际改造一两处明文打印为 `redact.String(...)`，作为示例（属于 W3 后续审计范畴）。
- T-0063 业务审计日志（admin audit log）落 PG 是另一层级，本任务只覆盖结构化日志 / 错误响应；两者互补不重复。

## 7. 状态

`.wave-status.txt = DONE`
