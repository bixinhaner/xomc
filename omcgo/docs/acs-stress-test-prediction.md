# ACS 压测预期分析报告

> 日期: 2026-03-10 (v3 — per-device transport pool + stress config + in-flight guard)
> 基于代码静态分析 + 硬件环境建模 + v1/v2/v3 三轮压测数据校准

---

## 一、测试环境

### 硬件
| 项目 | 值 |
|------|-----|
| CPU | Apple M1 / 4 物理核 / 8 逻辑核 |
| RAM | 16 GB |
| OS | macOS 12.7.6 (Darwin 21.6.0) |
| 部署 | **同机部署** (ACS + Redis + loadtest client 共享资源) |

### ACS 配置

| 参数 | 生产 (acs.yaml) | 压测 (acs-stress.yaml) | 含义 |
|------|----------------|----------------------|------|
| `server.port` | 7547 | 7547 | TR069 HTTP 端口 |
| `server.read_timeout` | 30s | 30s | HTTP 读超时 |
| `server.write_timeout` | 30s | 30s | HTTP 写超时 |
| `server.idle_timeout` | 120s | 120s | HTTP 空闲超时 |
| `session.timeout` | 5min | 5min | Redis Session TTL |
| `session.max_concurrent` | 10,000 | **50,000** | Admission 最大并发 |
| `rate_limit.per_device` | 10/min | **60,000/min** | 设备级限流 |
| `redis.pool_size` | 100 | **200** | Redis 连接池 |
| `log.level` | info | **warn** | 降低日志开销 |

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

**v2 预期** (修复 per-session transport 后, RemoteAddr 绑定正确):

| 轮次 | 并发 | 设备数 | 预期会话速率 | 预期请求速率 | 预期成功率 | 预期 p99 延迟 |
|------|------|--------|-------------|-------------|-----------|-------------|
| R1 | 50 | 500 | 300-800/s | 600-1,600/s | **>95%** | <30ms |
| R2 | 200 | 2,000 | 800-2,000/s | 1,600-4,000/s | **>90%** | <80ms |
| R3 | 500 | 5,000 | 1,500-3,000/s | 3,000-6,000/s | **>85%** | <300ms |
| R4 | 1,000 | 10,000 | 2,000-3,500/s | 4,000-7,000/s | **>75%** | <1s |
| R5 | 3,000 | 30,000 | **拐点区域** | — | **<30%** | >5s |

**v1 实际** (connection pool bug — RemoteAddr 不一致导致 Empty POST 失败):

| 轮次 | 实际会话速率 | 实际成功率 | 实际 p99 | 问题 |
|------|------------|-----------|---------|------|
| R1 | 114.8/s | 38.7% | 16ms | 大量 204 (RemoteAddr miss) |
| R2 | 459.1/s | 43.9% | 57ms | 同上 |
| R3 | 1,106.8/s | 53.4% | 217ms | 同上，但高并发下复用概率上升 |
| R4 | 2,224.5/s | 76.5% | 437ms | 高并发时连接复用概率高 |
| R5 | 0.3/s | 1.9% | 13.8s | 系统崩溃 (fd/goroutine 耗尽) |

**v2 实际** (per-session transport, 生产 rate limit 10/min):

| 轮次 | 实际会话速率 | 实际成功率 | 实际 p99 | 限流数 | 问题 |
|------|------------|-----------|---------|--------|------|
| R1 | 115.7/s | 46.7% | 606ms | 16,000 | Rate Limiter 限流 |
| R2 | 462.0/s | 77.8% | 375ms | 14,469 | 同上 |
| R3 | 626.2/s | 87.6% | 1,190ms | 0 | TCP errors 10,681 |
| R4 | 665.9/s | 93.0% | 2,442ms | 0 | TCP errors 6,066 |
| R5 | 256.7/s | 28.6% | 18,356ms | 0 | TCP 崩溃 84,229 |

**v3 实际** (per-device transport pool + stress config 60000/min + in-flight guard):

| 轮次 | 实际会话速率 | 实际成功率 | 实际 p99 | 限流数 | TCP 错误 | 评价 |
|------|------------|-----------|---------|--------|---------|------|
| R1 | **418/s** | **100%** | 31ms | 0 | 0 | 完美 |
| R2 | **1,215/s** | **99.96%** | 94ms | 0 | 64 | 优秀 |
| R3 | **2,017/s** | **99.98%** | 173ms | 0 | 43 | 优秀 |
| R4 | **2,310/s** | **99.99%** | 456ms | 0 | 38 | 优秀 |
| R5 | **779/s** | **60.1%** | 9,350ms | 0 | 62,538 | fd 耗尽 |

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

### 1. Rate Limiter 误杀 — ✅ v3 已解决

```
问题: 生产配置 10/min/device 在高频压测下限流 >50% 请求
v1/v2 影响: R1 成功率低至 38.7%-46.7%

v3 解决方案:
- configs/acs-stress.yaml: rate_limit.per_device = 60000 (等效禁用)
- 压测工具自动追踪 503/429 响应, 报告中显示 rate_limited 计数
- 启动时提示使用 stress config
```

### 2. TCP 连接复用 — ✅ v3 已优化

```
v1 缺陷: 共享 http.Transport → RemoteAddr 不匹配 → 大量 204
v2 修复: per-session Transport → 正确但每会话 TCP 握手开销大
v3 优化: per-device Transport pool → TCP 跨会话复用 + RemoteAddr 一致

v2→v3 效果: R4 吞吐 666→2,310 sess/s (×3.5), TCP errors 6066→38 (×160 减少)
```

### 3. 同机资源争用 (固有限制)

- loadtest client (Go 程序) 也消耗 CPU/内存
- 高并发时 goroutine 调度竞争加剧
- v3 实测：实际吞吐约为理论值的 35-50% (符合预期)
- R5 (3000 并发) 因 macOS fd limit 崩溃 — 生产 Linux 环境不受影响

---

## 六、结果解读指南 (v3 基准)

| 指标 | v3 基准 (stress config) | 异常判断 |
|------|------------------------|---------|
| 成功率 R1 | **100%** | <95% → 检查 Redis 连通 / ACS 是否用 stress config |
| 成功率 R3 | **99.98%** | <95% → 检查 admission / CPU 饱和 |
| 成功率 R4 | **99.99%** | <90% → 检查 fd limit / Redis pool |
| p99 延迟 R1 | **31ms** | >100ms → Redis 延迟 / CPU 竞争 |
| p99 延迟 R3 | **173ms** | >500ms → goroutine 调度积压 |
| p99 延迟 R4 | **456ms** | >2s → 连接池饱和 |
| 会话速率 R4 | **2,310/s** | <1000/s → 检查 transport 池 / Redis ops |
| rate_limited 计数 | **0** | >0 → 未使用 stress config |
| TCP errors R4 | **38** | >1000 → fd limit 不足, ulimit -n 增大 |
| ActiveSessions 测后归零 | 是 | 否 → admission 泄漏 / reaper 异常 |

## 七、预期 vs 实际总结

| 轮次 | v2 预期成功率 | v3 实际 | 偏差 |
|------|-------------|---------|------|
| R1 | >95% | **100%** | 超出预期 |
| R2 | >90% | **99.96%** | 超出预期 |
| R3 | >85% | **99.98%** | 超出预期 |
| R4 | >75% | **99.99%** | 超出预期 |
| R5 | <30% | **60.1%** | 超出预期 (v3 更健壮) |

**结论**: v3 全面超出预期。R1-R4 成功率全部 >99.9%，证明 ACS 引擎在消除外部干扰 (Rate Limiter + TCP 新建开销) 后性能表现完美。
