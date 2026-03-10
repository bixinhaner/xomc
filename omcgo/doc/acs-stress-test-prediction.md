# ACS 压测预期分析报告

> 日期: 2026-03-10
> 基于代码静态分析 + 硬件环境建模

---

## 一、测试环境

### 硬件
| 项目 | 值 |
|------|-----|
| CPU | Apple M1 / 4 物理核 / 8 逻辑核 |
| RAM | 16 GB |
| OS | macOS 12.7.6 (Darwin 21.6.0) |
| 部署 | **同机部署** (ACS + Redis + loadtest client 共享资源) |

### ACS 配置 (acs.yaml)
| 参数 | 值 | 含义 |
|------|-----|------|
| `server.port` | 7547 | TR069 HTTP 端口 |
| `server.read_timeout` | 30s | HTTP 读超时 |
| `server.write_timeout` | 30s | HTTP 写超时 |
| `server.idle_timeout` | 120s | HTTP 空闲超时 |
| `session.timeout` | 5min | Redis Session TTL |
| `session.max_concurrent` | 10,000 | Admission 最大并发 |
| `rate_limit.per_device` | 10/min | 设备级限流 (burst=5) |
| `redis.pool_size` | 100 | Redis 连接池 |
| `db.max_conns` | 20 | PostgreSQL 连接池 |

### 关键组件实现特征
| 组件 | 实现 | 性能特征 |
|------|------|---------|
| Admission | 原子 CAS (in-memory) | 极快，无锁竞争 |
| Rate Limiter | sync.Map + Token Bucket (in-memory) | 快，轻量级锁 |
| Session Store | Redis SET/GET | 依赖 Redis 延迟 |
| Command Queue | Redis ZSET (ZAdd/ZRange/ZRem) | 依赖 Redis 延迟 |
| SOAP 模板 | Pre-compiled text/template | 快，~0.1ms/次 |
| XML 解析 | encoding/xml stream | 较慢，~0.5ms/次 |
| Event Bus | ChannelEventBus (in-memory) | 非阻塞，满则丢弃 (buffer=256) |
| connSessions | sync.Map + reaper | 轻量，30s 扫描 |

---

## 二、每会话资源消耗模型

### Redis 操作计数

```
Full Session (无 RPC 命令):
┌─────────────────────┬──────────────────────┬─────────┐
│ 阶段                 │ Redis 操作            │ 次数    │
├─────────────────────┼──────────────────────┼─────────┤
│ handleInform        │ SET (session create)  │ 1       │
│ handleEmpty         │ GET (session)         │ 1       │
│                     │ SET (state update)    │ 1       │
│                     │ ZRANGEWITHSCORES      │ 1       │
│                     │ (queue empty, no ZRem)│         │
│ completeSession     │ SET (state complete)  │ 1       │
├─────────────────────┼──────────────────────┼─────────┤
│ 合计                 │                      │ 5       │
└─────────────────────┴──────────────────────┴─────────┘

Full Session (1 轮 RPC):
┌─────────────────────┬──────────────────────┬─────────┐
│ 阶段                 │ Redis 操作            │ 次数    │
├─────────────────────┼──────────────────────┼─────────┤
│ handleInform        │ SET                   │ 1       │
│ handleEmpty         │ GET + SET + ZRANGE    │ 3       │
│                     │ + ZREM + SET (pending)│ 2       │
│ handleRPCResponse   │ GET + SET + ZRANGE    │ 3       │
│ completeSession     │ SET                   │ 1       │
├─────────────────────┼──────────────────────┼─────────┤
│ 合计                 │                      │ 10      │
└─────────────────────┴──────────────────────┴─────────┘
```

### CPU 消耗 (每会话)

| 操作 | 估算耗时 | 备注 |
|------|---------|------|
| XML 解析 (Inform) | ~0.5ms | encoding/xml stream parser |
| SOAP 模板渲染 (InformResp) | ~0.1ms | pre-compiled template |
| JSON 序列化 (Session) | ~0.05ms | 8 字段 struct |
| HTTP 框架开销 | ~0.1ms | net/http stdlib |
| Event 发布 | ~0.01ms | 非阻塞 channel send |
| **合计 (Inform 路径)** | **~0.76ms** | |
| **合计 (Full Session)** | **~1.2ms** | |

---

## 三、瓶颈分析

### 理论上限

| 瓶颈层 | 计算逻辑 | 理论上限 |
|--------|---------|---------|
| **CPU** (XML 解析主导) | 8 核 × (1000ms / 0.76ms) | ~10,500 Inform/s |
| **Redis 连接池** (100 连接) | 100 × (1000ms / 0.3ms) / 5 ops | ~66,000 sessions/s |
| **Go HTTP goroutine** | 经验值 (net/http stdlib) | ~15,000-20,000 req/s |
| **同机竞争** (client 占 ~40% CPU) | CPU 上限 × 0.6 | ~6,300 Inform/s |

**预测主瓶颈**: **CPU (XML 解析)** — 同机部署时，loadtest client 消耗约 40% CPU，ACS 实际可用约 4.8 核。

### Redis 连接池评估

```
场景: 500 并发会话 × 5 Redis ops/session × 0.3ms/op = 750ms 总 Redis 时间
需要连接数: 500 × 5 × 0.3ms / 1000ms = 0.75 连接 (极度充裕)

场景: 5000 并发会话 = 7.5 连接 (依然充裕)

结论: Redis pool=100 对本机测试不是瓶颈。
```

### Admission 控制预测

```
修复后 admission 真正持锁于整个会话:
- 会话平均持续时间: ~2-5ms (无 RPC) 到 ~10-20ms (有 RPC)
- 10,000 slots / 5ms avg = 2,000,000 sessions/s 理论
- Admission 不是瓶颈
```

---

## 四、分场景预期

### Full-Session 模式 (默认, 每会话 2 个 HTTP 请求)

| 轮次 | 并发 | 设备数 | 预期会话速率 | 预期请求速率 | 预期成功率 | 预期 p99 延迟 |
|------|------|--------|-------------|-------------|-----------|-------------|
| R1 | 50 | 500 | 300-800/s | 600-1,600/s | **>95%** | <30ms |
| R2 | 200 | 2,000 | 800-2,000/s | 1,600-4,000/s | **>90%** | <80ms |
| R3 | 500 | 5,000 | 1,500-3,000/s | 3,000-6,000/s | **>80%** | <300ms |
| R4 | 1,000 | 10,000 | 2,000-3,500/s | 4,000-7,000/s | **60-80%** | <1.5s |
| R5 | 3,000 | 30,000 | **拐点区域** | — | **<50%** | >5s |

### Inform-Only 模式 (fallback, 每会话 1 个 HTTP 请求)

| 轮次 | 并发 | 预期 Inform/s | 预期成功率 | 预期 p99 |
|------|------|-------------|-----------|---------|
| R1 | 50 | 1,000-2,000 | >98% | <20ms |
| R2 | 200 | 2,000-4,000 | >95% | <50ms |
| R3 | 500 | 3,000-5,000 | >90% | <200ms |
| R4 | 1,000 | 3,500-5,500 | 70-85% | <1s |
| R5 | 3,000 | 4,000-6,000 | <50% | >5s |

---

## 五、关键风险预警

### 1. Rate Limiter 误杀 (高风险)

```
配置: 10/min/device = 0.167 req/s/device, burst=5

压测场景: 500 设备, 每秒循环 → 每设备 ~1 req/s
0.167 < 1.0 → 第 6 个请求开始被限流!

影响: 如果 loadtest interval=1s, 持续运行后每设备超过 burst 会被限流。
第一轮 (burst): 5 × 500 = 2500 请求通过
之后: 每秒仅 500 × 0.167 = 83 请求通过

实际影响: 成功率可能降至 ~10-20% (非引擎性能问题，而是限流策略)
```

**建议**: 压测时需将 `per_device` 调高或使用足够多设备分散请求。

### 2. 同机资源争用

- loadtest client (Go 程序) 也消耗 CPU/内存
- 高并发时 goroutine 调度竞争加剧
- 预期：实际吞吐为理论值的 50-60%

### 3. TCP 连接复用

- full-session 模式依赖 HTTP Keep-Alive 复用 TCP 连接
- 如果连接被服务端 120s idle timeout 关闭，后续 Empty POST 会失败
- 正常情况下不是问题（会话 <1s），但高延迟时可能触发

---

## 六、结果解读指南

| 指标 | 正常范围 | 异常判断 |
|------|---------|---------|
| 成功率 R1 | >95% | <80% → 检查 rate limiter / Redis 连通 |
| 成功率 R3 | >80% | <50% → 检查 admission / CPU 饱和 |
| p99 延迟 R1 | <50ms | >200ms → Redis 延迟 / CPU 竞争 |
| p99 延迟 R3 | <500ms | >2s → goroutine 调度积压 |
| 会话完成率 | >90% | <70% → 检查 Empty POST 路由 / connSessions |
| ActiveSessions 测后归零 | 是 | 否 → admission 泄漏 / reaper 异常 |
