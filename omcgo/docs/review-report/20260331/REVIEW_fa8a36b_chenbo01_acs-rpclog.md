# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-31 |
| 基准提交 | fa8a36b |
| 作者 | chenbo01 |
| Scope | acs (protocol logging) |
| 审查结论 | **PASS** |

## 变更概述

为 ACS 引擎新增独立的协议交互日志功能。每次 ACS-CPE 的 HTTP 交互都会记录完整的请求/响应原始 XML 到专用日志文件，独立于应用日志，使用 lumberjack 轮转。

### 文件列表

| 文件 | 操作 |
|------|------|
| `internal/acs/rpclog/rpclog.go` | 新增 — LogEntry 上下文元数据 + ResponseCapturer 响应捕获器 |
| `internal/core/appconfig/config.go` | 修改 — 新增 ProtocolLogConfig 结构体 |
| `cmd/acs/etc/config.dev.yaml` | 修改 — 新增 protocol_log 配置段 (enabled: true) |
| `cmd/acs/etc/config.test.yaml` | 修改 — 新增 protocol_log 配置段 (enabled: true) |
| `cmd/acs/etc/config.prod.yaml` | 修改 — 新增 protocol_log 配置段 (enabled: false) |
| `internal/acs/handler.go` | 修改 — ServeHTTP 入口包装 ResponseCapturer + 6 个 handler 填充元数据 |
| `internal/acs/server.go` | 修改 — ServerDeps 新增 ProtocolLogger/MaxBodySize 字段 |
| `cmd/acs/main.go` | 修改 — newProtocolLogger 函数 + deps 注入 |

## 审查检查项

### Go 工���规范

- [x] 错误处理：logger 创建失败只打警告不阻断启动
- [x] 并发安全：LogEntry 为单请求独占，无并发访问
- [x] 资源管理：lumberjack 自动管理文件轮转和压缩
- [x] Context 传递：LogEntry 通过 context.Value 正确传播

### 架构设计

- [x] 非侵入式：ResponseCapturer 在 ServeHTTP 入口统一包装，不修改各 handler 的响应逻辑
- [x] 零开销关闭：protocolLogger == nil 时完全跳过（生产环境默认关闭）
- [x] 配置完善：独立控制开关、文件路径、截断阈值、轮转参数
- [x] 复用 RotationConfig：与应用日志轮转配置结构一致

### 安全审查

- [x] 协议日志可能包含敏感信息（设备密码等），生产环境默认 enabled: false
- [x] 文件权限：目录 0755，文件由 lumberjack 管理
- [x] XML 截断支持：max_body_size 防止内存和磁盘溢出

### 性能

- [x] ResponseCapturer.Write() 使用 bytes.Buffer 追加，开销可控
- [x] 日志写入在 defer 中执行，不影响响应延迟
- [x] JSON 编码写入独立文件，不阻塞应用日志

### TR-069 协议

- [x] 覆盖所有 6 个 handler：Inform, Empty, RPCResponse, SOAPFault, TransferComplete, AutonomousTransferComplete
- [x] 正确记录 8 处元数据填充点（包括 TaskService 和旧 cmdqueue 两条路径）
- [x] cwmpID 正确捕获（用于关联请求/响应配对）

### 发现

无 CRITICAL 或 WARNING 级问题。

| 级别 | 说明 |
|------|------|
| INFO | 非轮转模式下打开的文件句柄 (`os.OpenFile`) 生命周期等同进程，不会泄漏 |
| INFO | `capturer.Body()` 返回完整响应字节的拷贝，大 XML 场景可配置 max_body_size 截断 |
