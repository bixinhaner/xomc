# ACS 压力测试

对 ACS (TR069 Auto Configuration Server) 引擎进行全面的压力测试。支持 4 种测试模式：ACS 会话吞吐 / KPI (PM) 文件上传 / MR 文件上传 / 混合模式，以及 5 轮递增压测 (-ramp)。验证完整 TR069 会话生命周期、AutonomousTransferComplete 文件上传管线在高并发场景下的性能和稳定性。

## 环境信息

- **ACS 服务**: `omcgo/cmd/acs/main.go`
- **ACS 配置**: `omcgo/configs/acs.yaml`
- **压测工具**: `omcgo/scripts/loadtest/main.go`
- **ACS 端口**: 7547 (HTTP TR069)
- **依赖**: Redis (默认 localhost:6379), NATS (默认 localhost:4222)
- **预期报告**: `omcgo/doc/acs-benchmark-analysis.md`
- **预测报告**: `omcgo/doc/acs-stress-test-prediction.md`

## 执行步骤

### Step 1: 环境检查

并行执行以下检查：

1. **Redis 连通性**: `redis-cli ping` 返回 PONG
2. **NATS 连通性**: 检查 NATS 服务是否运行 (`nats-server` 进程)
3. **编译检查**: `cd omcgo && go build ./cmd/acs/ ./scripts/loadtest/`

如果任何检查失败，**立即停止并报告错误**。

### Step 2: 启动 ACS 服务

```bash
cd omcgo

# 编译 ACS
go build -o bin/omcgo-acs ./cmd/acs/

# 检查是否已在运行
lsof -ti:7547

# 如果端口被占用，终止旧进程
lsof -ti:7547 | xargs kill -9 2>/dev/null

# 启动 ACS (后台)
nohup ./bin/omcgo-acs --config configs/acs.yaml > /tmp/omcgo-acs.log 2>&1 &

# 等待启动
sleep 2

# 验证
curl -s -o /dev/null -w "%{http_code}" http://localhost:7547/acs
```

### Step 3: 生成预测报告

基于代码和硬件环境分析，保存到 `omcgo/doc/acs-stress-test-prediction.md`。

分析维度：
- 硬件环境 (CPU 核数 / 内存 / 同机部署)
- ACS 配置参数 (Redis 连接池 / Admission 限制 / Rate Limit / Session TTL)
- Redis 操作计数 (每会话 5-10 次 Redis 操作)
- 理论瓶颈计算 (CPU / Redis 连接池 / HTTP 调度)
- 分场景预期 (各并发级别的成功率、吞吐量、延迟)

### Step 4: 执行压测 (5 轮递增)

每轮运行 60 秒，使用 JSON 输出便于解析：

```bash
cd omcgo

# Round 1: 基线 (50 并发 / 500 设备)
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -devices 500 -concurrency 50 -duration 60s -json

# Round 2: 中等负载 (200 并发 / 2000 设备)
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -devices 2000 -concurrency 200 -duration 60s -json

# Round 3: 高负载 (500 并发 / 5000 设备)
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -devices 5000 -concurrency 500 -duration 60s -json

# Round 4: 压力上限 (1000 并发 / 10000 设备)
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -devices 10000 -concurrency 1000 -duration 60s -json

# Round 5: 极限 (3000 并发 / 30000 设备)
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -devices 30000 -concurrency 3000 -duration 60s -json
```

**每轮之间等待 5 秒**，让 ACS 的 admission slots 和 connSessions reaper 清理完毕。

#### KPI/MR 文件上传压测

```bash
cd omcgo

# 仅测 KPI (PM 文件上传): Inform → ATC(FileType=4) → session end
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -mode kpi -devices 1000 -concurrency 100 -duration 60s -json

# 仅测 MR (MR 文件上传): Inform → ATC(FileType=5) → session end
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -mode mr -devices 1000 -concurrency 100 -duration 60s -json

# 混合模式 (round-robin: acs/kpi/mr 按设备编号轮转)
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -mode all -devices 3000 -concurrency 300 -duration 60s -json
```

#### 5 轮递增压测 (一键 15 分钟 ~5 万请求)

```bash
# KPI 模式 5 轮递增
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -mode kpi -ramp -json

# 混合模式 5 轮递增
go run scripts/loadtest/main.go \
  -url http://localhost:7547/acs \
  -mode all -ramp -json
```

递增计划 (每轮 3 分钟):

| 轮次 | 并发 | 设备数 | 预估请求 |
|------|------|--------|---------|
| R1 warmup | 20 | 200 | ~2,400 |
| R2 light | 50 | 500 | ~6,000 |
| R3 medium | 100 | 1,000 | ~12,000 |
| R4 heavy | 200 | 2,000 | ~16,000 |
| R5 stress | 500 | 5,000 | ~20,000 |
| **合计** | - | - | **~56,000** |

### Step 5: 结果分析

对比预测 vs 实际，保存分析报告到 `omcgo/doc/acs-benchmark-analysis.md`。

分析维度：
1. **成功率趋势**: 随并发增加的变化曲线
2. **吞吐量拐点**: 找到性能开始下降的并发级别
3. **延迟分布**: p50/p90/p95/p99 ���负载变化
4. **瓶颈识别**: Rate Limiter? Redis? CPU? 连接池?
5. **生产环境换算**: 基于测试数据预估独立部署性能

### Step 6: 清理

```bash
# 停止 ACS 服务
pkill -f omcgo-acs
```

## 压测工具参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-url` | `http://localhost:7547/acs` | ACS 端点 URL |
| `-mode` | `acs` | 测试模式: `acs` / `kpi` / `mr` / `all` |
| `-devices` | 1000 | 模拟设备数量 |
| `-concurrency` | 100 | 最大并发会话数 |
| `-duration` | 2m | 测试持续时间 |
| `-interval` | 1s | 设备发送 Inform 的间隔 |
| `-full-session` | true | 完整 TR069 会话模式 (kpi/mr/all 强制 true) |
| `-json` | false | JSON 格式输出 |
| `-ramp` | false | 5 轮递增压测 (覆盖 devices/concurrency/duration) |
| `-file-port` | 9876 | 内置文件服务器端口 (kpi/mr/all 模式) |
| `-file-host` | `localhost` | TransferURL 主机名 (需 TransferBridge 可达) |

## ACS 关键配置 (acs.yaml)

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| `session.max_concurrent` | 10000 | Admission 最大并发会话数 |
| `rate_limit.per_device` | 10 | 每设备每分钟最大请求数 (压测时可调至 600) |
| `redis.pool_size` | 100 | Redis 连接池 (建议 >= 并发数) |
| `server.read_timeout` | 30s | HTTP 读超时 |
| `server.write_timeout` | 30s | HTTP 写超时 |

## Prometheus 监控指标

压测期间关注以下指标 (端口见 acs.yaml 的 metrics.port):

| 指标 | 类型 | 含义 |
|------|------|------|
| `acs_active_sessions` | Gauge | 当前活跃会话数 (应 <= max_concurrent) |
| `acs_session_duration_seconds` | Histogram | 会话耗时分布 |
| `acs_inform_total{event_type}` | Counter | Inform 消息计数 |
| `acs_rpc_duration_seconds{method}` | Histogram | RPC 方法耗时 |
| `acs_rpc_errors_total{method}` | Counter | RPC 错误计数 |

## 性能基准 (4 核 / 同机部署)

| 指标 | 基准值 | 模式 | 说明 |
|------|--------|------|------|
| 会话速率 | >2000 sessions/s | acs | R4 (1000 并发) |
| 请求吞吐 | >5000 req/s | acs | R4 |
| p99 延迟 | <500ms | acs | R4 (同机竞争) |
| 成功率 | >70% | acs | R4 (含 Rate Limiter 影响) |
| 文件上传速率 | >500 uploads/s | kpi/mr | R3 (100 并发) |
| 混合吞吐 | >3000 req/s | all | R4 (acs+kpi+mr 混合) |

## KPI/MR 模式说明

### 会话流程

```
mode=acs:  Inform → InformResponse → Empty → RPC/Empty → session end
mode=kpi:  Inform → InformResponse → ATC(FileType=4) → ATCResponse → Empty → session end
mode=mr:   Inform → InformResponse → ATC(FileType=5) → ATCResponse → Empty → session end
mode=all:  按设备编号 mod 3 轮转 (0=acs, 1=kpi, 2=mr)
```

### 内置文件服务器

kpi/mr/all 模式下自动启动 HTTP 文件服务器 (默认端口 9876):
- `GET /pm/{sn}.xml` → 3GPP 32.435 PM XML (4 counters × 2 cells)
- `GET /mr/{sn}.xml` → MRO XML (RSRP/RSRQ/SINR × 2 cells)

ATC 消息中的 TransferURL 指向此服务器。TransferBridge 从此 URL 下载文件 → MinIO → PM/MR Collector。

### 数据管线 (需完整基础设施)

若 NATS + MinIO + Collectors 运行中，KPI/MR 模式将触发完整数据管线:
```
Load Test → ACS (ATC) → NATS → TransferBridge → MinIO → PM/MR Collector → PostgreSQL
```
若只有 ACS 运行，仍可测试 ACS 的 ATC 处理吞吐量。

## 常见问题

### Q1: 成功率很低 (<50%)

**检查**:
1. Rate Limiter: 设备数不够 → 增加 `-devices` 或降低 `rate_limit.per_device`
2. Redis 连接: `redis-cli ping` 确认连通
3. ACS 日志: `tail -100 /tmp/omcgo-acs.log` 检查错误

### Q2: 大量 errors (非 failures)

**原因**: 通常是 TCP 连接超时或文件描述符耗尽
**解决**: 降低 `-concurrency`，或 `ulimit -n 65535` 增加 fd 限制

### Q3: 压测后 active_sessions 不归零

**原因**: connSessions 泄漏，等待 reaper 清理 (每 30s 扫描, 5min TTL)
**验证**: 等待 5 分钟后再检查 Prometheus 指标

### Q4: Rate Limiter 警告

压测工具会自动检测 Rate Limiter 是否会成为瓶颈并打印警告。
公式: `concurrency * 2 / devices * 60 > 10` 时触发限流。
解决: 确保 `devices >= concurrency * 12`。

## 相关文档

- **压测预期分析**: `omcgo/doc/acs-stress-test-prediction.md`
- **压测结果分析**: `omcgo/doc/acs-benchmark-analysis.md`
- **修复方案记录**: `omcgo/doc/acs-stress-test-fix-plan.md`
- **就绪评估**: `omcgo/doc/acs-stress-test-readiness.md`
- **ACS 架构**: `omcgo/internal/acs/` (handler.go, server.go)
- **压测工具源码**: `omcgo/scripts/loadtest/main.go`
