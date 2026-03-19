# DD-19: 可观测性（日志/指标/追踪）

> 关联功能域：全部
> 关联 backend-design.md 章节：第三章（技术栈选型）
> 实施阶段：Phase 1 ~ Phase 4（贯穿全部阶段）
> 依赖文档：DD-01

---

## 1. 概述

### 1.1 模块定位

可观测性是系统运维的基础能力，贯穿所有模块。包含三大支柱：结构化日志（zap）、指标暴露（Prometheus）和分布式追踪（OpenTelemetry）。

### 1.2 核心职责

- 统一日志格式与级别策略
- Prometheus 指标注册与暴露
- OpenTelemetry Span 传播与采样
- 健康检查端点
- 监控面板设计

---

## 2. 详细设计

### 2.1 结构化日志 — zap

**日志配置**：

```go
type LogConfig struct {
    Level  string `yaml:"level"`  // debug, info, warn, error
    Format string `yaml:"format"` // json, console
    Output string `yaml:"output"` // stdout, file path
}

func NewLogger(cfg LogConfig) (*zap.Logger, error)
```

**日志字段规范**：

| 字段 | 说明 | 示例 |
|------|------|------|
| `component` | 模块名 | `acs`, `pm`, `alarm` |
| `device_sn` | 设备序列号 | `ABC123456` |
| `carrier` | 运营商 | `cmcc` |
| `session_id` | 会话 ID | `uuid` |
| `rpc_method` | RPC 方法 | `GetParameterValues` |
| `duration_ms` | 耗时（毫秒）| `150` |
| `error` | 错误信息 | `connection refused` |

**日志级别策略**：

| 级别 | 使用场景 |
|------|---------|
| `Debug` | 开��调试：SOAP 报文内容、参数值详情 |
| `Info` | 正常业务事件：设备注册、Inform 接收、命令执行 |
| `Warn` | 可恢复异常：限流触发、缓存未命中、重试 |
| `Error` | 不可恢复异常：DB 连接失败、RPC 超时、数据解析错误 |

### 2.2 Prometheus 指标

**全局指标**：

```go
// ACS 指标
acs_active_sessions         gauge       // 当前活跃会话数
acs_inform_total            counter     // Inform 总数（按 event_type 标签）
acs_rpc_duration_seconds    histogram   // RPC 执行耗时
acs_rpc_errors_total        counter     // RPC 错误数（按 method 标签）
acs_session_duration_seconds histogram  // 会话持续时间

// 设备指标
device_total                gauge       // 设备总数（按 carrier, tech, status 标签）
device_online               gauge       // 在线设备数

// PM 指标
pm_files_processed_total    counter     // PM 文件处理数
pm_parse_duration_seconds   histogram   // PM 解析耗时
kpi_calculation_duration    histogram   // KPI 计算耗时

// 告警指标
alarm_active_total          gauge       // 活跃告警数（按 severity 标签）
alarm_raised_total          counter     // 告警产生数
alarm_cleared_total         counter     // 告警清除数

// 基础设施指标
db_query_duration_seconds   histogram   // DB 查询耗时
redis_operation_duration    histogram   // Redis 操作耗时
nats_publish_total          counter     // NATS 发布数
nats_consume_total          counter     // NATS 消费数
```

**Metrics 端点**：

| 进程 | 端口 | 路径 |
|------|------|------|
| omcgo-acs | 9090 | `/metrics` |
| omcgo-app | 9091 | `/metrics` |
| omcgo-worker | 9092 | `/metrics` |

### 2.3 OpenTelemetry 追踪

**Span 命名规范**：`{component}.{operation}`

| Span 名称 | 说明 |
|-----------|------|
| `acs.handle_inform` | Inform 处理 |
| `acs.execute_rpc` | RPC 方法执行 |
| `datamodel.resolve` | 数据模型三级回退解析 |
| `pm.parse_file` | PM 文件解析 |
| `alarm.process` | 告警处理 |

**采样策略**：
- 开发环境：全量采样
- 生产环境：尾部采样（错误全采、成功 1%）

### 2.4 健康检查端点

```
GET /healthz    → 存活探针（Liveness）
GET /readyz     → 就绪探针（Readiness）
```

---

## 3. 实施子阶段

### 阶段 19a：zap 日志 + Prometheus 基础指标（Phase 1）

**交付物**：日志初始化、metrics HTTP 端点、基础指标注册
**验证**：启动后 `/metrics` 可访问

### 阶段 19b：OpenTelemetry 追踪（Phase 3）

**交付物**：Tracer 初始化、关键路径 Span 注入
**验证**：Jaeger 中可看到 trace 链

### 阶段 19c：Grafana 面板 + 告警规则（Phase 4）

**交付物**：Grafana dashboard JSON、Prometheus AlertManager rules
**验证**：面板数据正常展示

---

## 4. 文件清单

```
internal/infra/logger.go          # zap 初始化
internal/infra/metrics.go         # Prometheus 注册表
internal/infra/tracer.go          # OpenTelemetry 初始化
deployments/grafana/               # Grafana dashboard JSON
deployments/prometheus/            # AlertManager rules
```

---

## 5. 参考

- CLAUDE.md 第 3 节：技术栈（zap、prometheus、otel）
- backend-design.md 第九章：部署架构
