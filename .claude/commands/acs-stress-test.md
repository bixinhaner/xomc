# ACS 全链路压力测试

对 OMC 系统进行全链路压力测试，覆盖完整数据管线。支持 4 种测试模式 (acs/kpi/mr/all) 和 5 轮递增压测 (-ramp)。
**所有压测必须启动完整数据管线**，包括 Web 前端。

## 系统架构

```
                        ┌─────────────────────────────────────────┐
                        │              完整数据管线                 │
                        │                                         │
 Load Test ──► ACS ──► NATS(JetStream) ──► TransferBridge        │
 (压测工具)    :7547    :4222               │  (Worker 进程)       │
                                           ▼                      │
                                         MinIO ──► PM/MR Collector│
                                         :9000     (Worker 进程)  │
                                                      │           │
                                                      ▼           │
                                                  PostgreSQL      │
                                                   :5432          │
                        └─────────────────────────────────────────┘
                                                      │
                                                      ▼
                                              Main App (REST API)
                                                   :8080
                                                      │
                                                      ▼
                                              Web Frontend (Vite)
                                                   :3000
```

## 组件清单

| 组件 | 二进制 / 命令 | 端口 | 配置文件 | 日志 |
|------|-------------|------|---------|------|
| **ACS** | `bin/omcgo-acs` | 7547 | `configs/acs-stress.yaml` | `/tmp/omcgo-acs.log` |
| **Worker** | `bin/omcgo-worker` | 9092 (metrics) | `cmd/worker/etc/config.dev.yaml` | `/tmp/omcgo-worker.log` |
| **Main App** | `bin/omcgo-app` | 8080 | `cmd/app/etc/config.dev.yaml` | `/tmp/omcgo-app.log` |
| **Frontend** | `npm run dev` | 3000 | `omcmb/webcode/vite.config.ts` | `/tmp/omcmb-dev.log` |
| **NATS** | `nats-server -js` | 4222 | (内置 JetStream) | `/tmp/nats-server.log` |
| **Redis** | `redis-server` | 6379 | 默认 | - |
| **MinIO** | `minio server` | 9000 | 默认 | - |
| **PostgreSQL** | `postgres` | 5432 | 默认 | - |

> **重要**: Worker 进程包含 TransferBridge + PM Collector + MR Collector，是数据管线的核心。
> **重要**: NATS 必须启用 JetStream (`nats-server -js`)，否则 Worker 订阅全部失败。

## 执行步骤

### Step 1: 环境检查

并行检查所有 8 个组件：

```bash
# 1. 基础设施
redis-cli ping                                                    # → PONG
pgrep -f nats-server                                              # → PID (必须带 -js)
curl -s -o /dev/null -w "%{http_code}" http://localhost:9000/minio/health/live  # → 200
# PostgreSQL: 通过应用连接验证

# 2. 编译 3 个 Go 二进制
cd omcgo
go build -o bin/omcgo-acs ./cmd/acs/
go build -o bin/omcgo-worker ./cmd/worker/
go build -o bin/omcgo-app ./cmd/app/
go build -o bin/loadtest ./scripts/loadtest/

# 3. 前端依赖
cd ../omcmb/webcode && npm install --silent
```

如果任何检查失败，**立即停止并报告错误**。

**NATS JetStream 检查**：如果 NATS 未启用 JetStream，必须重启：
```bash
pkill -f nats-server
nohup nats-server -js -sd /tmp/nats-data > /tmp/nats-server.log 2>&1 &
```

### Step 2: 启动完整管线

**按顺序**启动 (有依赖关系):

```bash
cd omcgo

# ① 清理旧进程
lsof -ti:7547 | xargs kill -9 2>/dev/null   # ACS
pkill -f omcgo-worker 2>/dev/null             # Worker
# 注意: 不要杀 Main App (:8080) 和 Frontend (:3000)，除非需要

# ② 启动 ACS (stress 配置 — 放宽 rate limit)
nohup ./bin/omcgo-acs --config configs/acs-stress.yaml > /tmp/omcgo-acs.log 2>&1 &
sleep 2
curl -s -o /dev/null -w "ACS: %{http_code}\n" http://localhost:7547/acs

# ③ 启动 Worker (TransferBridge + PM/MR Collector)
nohup ./bin/omcgo-worker --config cmd/worker/etc/config.dev.yaml > /tmp/omcgo-worker.log 2>&1 &
sleep 3
pgrep -f omcgo-worker && echo "Worker: OK"
# 检查日志确认无 panic 或 subscribe 失败
grep -c "panic\|FATAL" /tmp/omcgo-worker.log

# ④ 确认 Main App 运行中
curl -s -o /dev/null -w "App: %{http_code}\n" http://localhost:8080/api/v1/auth/login

# ⑤ 确认 Frontend 运行中 (如未启动则启动)
curl -s -o /dev/null -w "Frontend: %{http_code}\n" http://localhost:3000
# 如需启动: cd ../omcmb/webcode && nohup npm run dev > /tmp/omcmb-dev.log 2>&1 &
```

### Step 3: 基线监控 (压测前快照)

**压测前**采集所有进程的 CPU / 内存基线：

```bash
echo "=== 压测前基线 ===" && date
ps -p $(pgrep -f omcgo-acs) -o pid,pcpu,pmem,rss,vsz,command 2>/dev/null | head -2
ps -p $(pgrep -f omcgo-worker) -o pid,pcpu,pmem,rss,vsz,command 2>/dev/null | head -2
ps -p $(pgrep -f nats-server) -o pid,pcpu,pmem,rss,vsz,command 2>/dev/null | head -2
ps -p $(pgrep -f omcgo-app) -o pid,pcpu,pmem,rss,vsz,command 2>/dev/null | head -2
ps -p $(pgrep -f redis-server) -o pid,pcpu,pmem,rss,vsz,command 2>/dev/null | head -2
ps -p $(pgrep -f minio) -o pid,pcpu,pmem,rss,vsz,command 2>/dev/null | head -2
```

记录 RSS (物理内存 KB) 和 %CPU 作为基线。

### Step 4: 执行压测

**每轮压测期间**监控 CPU/内存，**每轮结束后**检查进程状态。使用预编译二进制 `bin/loadtest` 而非 `go run`。

#### 4.1 混合模式递增压测 (推荐)

```bash
cd omcgo

# R1: 5000 设备 / 1000 并发 / 60s
./bin/loadtest -url http://localhost:7547/acs \
  -mode all -devices 5000 -concurrency 1000 -duration 60s -interval 100ms -json

sleep 5  # 等待 admission 清理

# R2: 10000 设备 / 2000 并发 / 60s
./bin/loadtest -url http://localhost:7547/acs \
  -mode all -devices 10000 -concurrency 2000 -duration 60s -interval 100ms -json

sleep 5

# R3: 30000 设备 / 3000 并发 / 60s
./bin/loadtest -url http://localhost:7547/acs \
  -mode all -devices 30000 -concurrency 3000 -duration 60s -interval 100ms -json

sleep 5

# R4: 50000 设备 / 5000 并发 / 60s
./bin/loadtest -url http://localhost:7547/acs \
  -mode all -devices 50000 -concurrency 5000 -duration 60s -interval 100ms -json
```

#### 4.2 单模式压测

```bash
# 纯 ACS 心跳
./bin/loadtest -url http://localhost:7547/acs \
  -mode acs -devices 5000 -concurrency 1000 -duration 60s -json

# 纯 KPI (PM 文件上传)
./bin/loadtest -url http://localhost:7547/acs \
  -mode kpi -devices 5000 -concurrency 1000 -duration 60s -json

# 纯 MR (MR 文件上传)
./bin/loadtest -url http://localhost:7547/acs \
  -mode mr -devices 5000 -concurrency 1000 -duration 60s -json
```

#### 4.3 一键 5 轮递增 (15 分钟 ~5.6 万请求)

```bash
./bin/loadtest -url http://localhost:7547/acs -mode all -ramp -json
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

### Step 5: 压测后监控 (关键!)

**每轮压测结束后**，必须检查所有进程 CPU/内存是否恢复正常：

```bash
echo "=== 压测后监控 ===" && date

# ① CPU / 内存快照
ps -p $(pgrep -f omcgo-acs) -o pid,%cpu,%mem,rss,command 2>/dev/null | head -2
ps -p $(pgrep -f omcgo-worker) -o pid,%cpu,%mem,rss,command 2>/dev/null | head -2
ps -p $(pgrep -f nats-server) -o pid,%cpu,%mem,rss,command 2>/dev/null | head -2
ps -p $(pgrep -f minio) -o pid,%cpu,%mem,rss,command 2>/dev/null | head -2

# ② Worker 存活检查 + 日志检查
pgrep -f omcgo-worker > /dev/null && echo "Worker: ALIVE" || echo "Worker: DEAD !!!"
grep -c "panic" /tmp/omcgo-worker.log

# ③ Worker 日志尾部 (检查是否还在处理消息)
# 注意: 如果 worker 日志仍在快速增长，说明 NATS 积压消息仍在消费
ls -lh /tmp/omcgo-worker.log
sleep 3
ls -lh /tmp/omcgo-worker.log
# 两次大小相同 → 队列已清空; 仍在增长 → 管线仍在处理

# ④ NATS JetStream 队列深度 (如 nats CLI 可用)
# nats stream ls
# nats consumer ls DEVICE
```

**异常判断标准**:

| 指标 | 正常 | 异常 (需排查) |
|------|------|-------------|
| Worker CPU | <5% (压测后 30s 内回落) | >50% 持续不降 |
| NATS CPU | <5% (压测后 30s 内回落) | >50% 持续不降 |
| Worker 日志 | 不再增长 | 持续增长 (MB/s) |
| Worker 进程 | 存活 | 崩溃 (panic) |
| ACS CPU | <5% | >20% 持续不降 |

**如果 Worker/NATS CPU 压测后不降**，参见 FAQ Q5。

### Step 6: 结果分析

对比各轮数据，保存分析报告到 `omcgo/doc/acs-benchmark-analysis.md`。

分析维度:
1. **成功率趋势**: 随并发增加的变化曲线，找到 >99% → <80% 的拐点
2. **吞吐量拐点**: 性能开始下降的并发级别
3. **延迟分布**: p50/p90/p95/p99 随负载变化
4. **文件上传速率**: kpi/mr/all 模式的 upload_rate 趋势
5. **管线吞吐**: TransferBridge → MinIO → Collector 的处理能力
6. **资源消耗**: CPU/内存在各轮次的变化对比
7. **瓶颈识别**: Rate Limiter / Redis / CPU / TCP 连接 / MinIO
8. **生产环境换算**: 基于同机测试数据预估独立部署性能

### Step 7: 清理

```bash
# 停止压测相关进程
pkill -f omcgo-acs
pkill -f omcgo-worker

# 清理 NATS JetStream 残留消息 (可选)
# pkill -f nats-server && rm -rf /tmp/nats-data/jetstream
# nohup nats-server -js -sd /tmp/nats-data > /tmp/nats-server.log 2>&1 &

# 注意: Main App 和 Frontend 通常保持运行，不要关闭
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

## ACS Stress 配置 (configs/acs-stress.yaml)

| 参数 | 值 | 说明 |
|------|-----|------|
| `rate_limit.per_device` | 60000 | 压测专用，等于禁用 |
| `rate_limit.global_burst` | 100000 | 全局突发限制 |
| `session.max_concurrent` | 50000 | Admission 最大并发 |
| `redis.pool_size` | 200 | Redis 连接池 |
| `log.level` | warn | 减少日志开销 |

> **禁止**使用默认 `acs.yaml` 压测 — `per_device: 10` 会导致几乎 100% 限流。

## 数据管线详解

### 会话流程

```
mode=acs:  Inform → InformResponse → Empty → RPC/Empty → session end
mode=kpi:  Inform → InformResponse → ATC(FileType=4) → ATCResponse → Empty → session end
mode=mr:   Inform → InformResponse → ATC(FileType=5) → ATCResponse → Empty → session end
mode=all:  按设备编号 mod 3 轮转 (0=acs, 1=kpi, 2=mr)
```

### 内置文件服务器

kpi/mr/all 模式下自动启动 HTTP 文件服务器 (默认端口 9876):
- `GET /pm/{sn}.xml` → 3GPP 32.435 PM XML (4 counters x 2 cells)
- `GET /mr/{sn}.xml` → MRO XML (RSRP/RSRQ/SINR x 2 cells)

ATC 消息中的 TransferURL 指向此服务器。

### 完整管线流转

```
Load Test → ACS → NATS(JetStream) → TransferBridge → MinIO → PM/MR Collector → PostgreSQL
  :9876      :7547    :4222           (Worker)         :9000   (Worker)           :5432
  文件服务    SOAP     事件总线        下载+存储         对象存储  解析+入库          持久化
                                                                    │
                                                              Main App (REST API) → Frontend
                                                                :8080               :3000
```

- **TransferBridge**: 订阅 `device.inform.autonomous_transfer_complete`，从 TransferURL 下载文件存入 MinIO，发布 `pm.file.received` / `mr.file.received`
- **PM Collector**: 订阅 `pm.file.received`，解析 3GPP 32.435 XML，批量写入 TimescaleDB，计算 KPI
- **MR Collector**: 订阅 `mr.file.received`，解析 MRO/MRS/MRE XML，批量写入 TimescaleDB
- **Main App**: REST API 提供设备管理、PM/MR 查询、KPI 报表等接口
- **Frontend**: React 前端通过 `/api` 代理到 Main App，展示 PM/MR 数据和 KPI 仪表盘

> **注意**: TransferBridge 会对 DB 中不存在的设备 SN 跳过下游事件 (直接 return nil)，不会产生毒丸消息。

## 进程监控命令速查

```bash
# 全部进程 CPU/内存一览
ps aux | grep -E "(omcgo-acs|omcgo-worker|omcgo-app|nats-server|redis-server|minio)" | grep -v grep

# 单进程详细 (替换 PID)
ps -p <PID> -o pid,%cpu,%mem,rss,vsz,etime,command

# 持续监控 (每 2 秒刷新)
watch -n 2 'ps aux | head -1; ps aux | grep -E "(omcgo|nats-server)" | grep -v grep'

# Worker 日志大小 (判断是否还在处理消息)
ls -lh /tmp/omcgo-worker.log

# Worker 日志尾部 (非阻塞，用 dd 避免管道阻塞)
dd if=/tmp/omcgo-worker.log bs=1 skip=$(($(stat -f%z /tmp/omcgo-worker.log) - 3000)) count=3000 2>/dev/null
```

## 性能基准 (同机部署 / 完整管线)

| 指标 | R1 (5K/1K) | R2 (10K/2K) | R3 (30K/3K) | R4 (50K/5K) | 模式 |
|------|------------|-------------|-------------|-------------|------|
| 成功率 | 99.73% | 99.59% | 79.34% | 2.58% | all |
| 吞吐量 (req/s) | 2,992 | 2,108 | 1,388 | 1,384 | all |
| 会话速率 (sess/s) | 1,790 | 1,260 | 659 | 0.13 | all |
| 文件上传速率 (/s) | 1,193 | 840 | 440 | 14 | all |
| p50 延迟 | 142ms | 390ms | 956ms | 2,050ms | all |
| p99 延迟 | 657ms | 2,549ms | 9,598ms | 6,854ms | all |
| 会话 p50 | 529ms | 1,403ms | 3,002ms | 10,701ms | all |

**甜区**: 1000 并发 / 5000 设备 (成功率 99.7%, 吞吐 3K req/s, 上传 1.2K/s)
**拐点**: 3000 并发 / 30000 设备 (成功率降至 79%, p99 飙升至 9.6s)
**崩溃**: 5000 并发 / 50000 设备 (同机部署资源耗尽)

## 常见问题

### Q1: 成功率很低 (<50%)

**检查**:
1. Rate Limiter: 确认使用 `configs/acs-stress.yaml` 而非默认 `acs.yaml`
2. Redis 连接: `redis-cli ping` 确认连通
3. ACS 日志: `tail -100 /tmp/omcgo-acs.log` 检查错误
4. 设备数不够: 增加 `-devices`，确保 `devices >= concurrency * 5`

### Q2: 大量 errors (非 failures)

**原因**: TCP 连接超时或文件描述符耗尽
**解决**: 降低 `-concurrency`，或 `ulimit -n 65535` 增加 fd 限制

### Q3: 压测后 active_sessions 不归零

**原因**: connSessions 泄漏，等待 reaper 清理 (每 30s 扫描, 5min TTL)
**验证**: 等待 5 分钟后再检查 Prometheus 指标

### Q4: Rate Limiter 警告

压测工具会自动检测 Rate Limiter 瓶颈。
公式: `concurrency * 2 / devices * 60 > per_device` 时触发限流。
解决: 确保使用 stress 配置或 `devices >= concurrency * 12`。

### Q5: 压测后 Worker/NATS CPU 持续高 (重要!)

**症状**: 压测结束后 Worker >100% CPU, NATS >100% CPU, worker 日志 >100MB 且持续增长

**根因链**:
1. 压测虚拟设备 (LT-000xxx) 不在 DB 中 → TransferBridge 查设备返回 nil
2. 旧版本: dev=nil 时仍发布 `pm/mr.file.received` 事件，`device_id: ""`
3. PM/MR Collector 解析 `uuid.Parse("")` 失败 → handler 返回 error
4. `wrapHandler` 调用 `msg.Nak()` → JetStream 立即重投 → 无限循环

**已修复** (3 处):
- `bridge.go`: `dev == nil` 时 `return nil`，不发布下游事件
- `nats_bus.go`: `wrapHandler` 增加 `maxDeliveries=5` + 指数退避 + `msg.Term()` 终止毒丸消息

**如果仍发生** (旧消息残留):
```bash
pkill -f omcgo-worker
pkill -f nats-server
rm -rf /tmp/nats-data/jetstream   # 清除 JetStream 积压
nohup nats-server -js -sd /tmp/nats-data > /tmp/nats-server.log 2>&1 &
sleep 2
nohup ./bin/omcgo-worker --config cmd/worker/etc/config.dev.yaml > /tmp/omcgo-worker.log 2>&1 &
```

### Q6: Worker 启动时 subscribe 失败

**症状**: `nats: jetstream not enabled` 或 `nats: no responders`
**原因**: NATS 未启用 JetStream
**解决**: `pkill -f nats-server && nohup nats-server -js -sd /tmp/nats-data > /tmp/nats-server.log 2>&1 &`

### Q7: Worker panic (nil pointer dereference)

**症状**: worker 日志出现 `panic: runtime error: invalid memory address`
**原因**: 旧版 `bridge.go` 未检查 `dev == nil`
**解决**: 确保 `bridge.go` 中 `GetBySerialNumber` 返回 nil 后直接 `return nil`

## 相关文件

| 文件 | 说明 |
|------|------|
| `omcgo/scripts/loadtest/main.go` | 压测工具源码 |
| `omcgo/configs/acs-stress.yaml` | ACS 压测配置 (放宽 rate limit) |
| `omcgo/cmd/worker/etc/config.dev.yaml` | Worker 开发配置 |
| `omcgo/cmd/app/etc/config.dev.yaml` | Main App 开发配置 |
| `omcgo/internal/acs/handler.go` | ACS 会话处理 (Inform / ATC) |
| `omcgo/internal/transfer/bridge.go` | TransferBridge (文件下载 + MinIO 存储) |
| `omcgo/internal/pm/collector/collector.go` | PM 文件解析 + KPI 计算 |
| `omcgo/internal/mr/collector/collector.go` | MR 文件解析 + 入库 |
| `omcgo/internal/event/nats_bus.go` | NATS JetStream 事件总线 (含重试策略) |
| `omcgo/doc/acs-benchmark-analysis.md` | 压测结果分析报告 |
| `omcgo/doc/acs-stress-test-prediction.md` | 压测预期分析 |
