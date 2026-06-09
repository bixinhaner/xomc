# ACS 会话结束后自动续唤分析报告

> 日期：2026-03-23
> 状态：**已实施并验证** (2026-03-23)
> 问题：会话结束时命令队列仍有积压，系统不会主动发送 Connection Request 唤醒设备
> 影响：3994 条命令只能依赖设备周期性 Inform（~60s/次）逐批消化，严重影响任务下发效率
>
> **实施摘要**：在 `completeSession()` 中增加异步续唤机制，会话结束后检查队列深度，
> 非空则发送 Connection Request（支持 STUN UDP + HTTP 两种方式，自动回退）。
> Redis 计数器防止无限循环，handleInform 时重置。从 Inform 参数自动缓存设备 CR 地址。
> 涉及文件：`handler.go`, `task_service.go`, `metrics.go`, `server.go`, `config.go`, `cmd/acs/main.go`

---

## 1. 问题现状

### 1.1 实际观测数据

| 指标 | 数值 |
|------|------|
| 设备 SN | `1202000588233HB0039` |
| 命令队列积压 | 3994 条（GPN 参数发现命令） |
| Inform 周期 | ~60 秒 |
| 每会话处理量 | 约 20-50 条（取决于会话时长） |
| 预计清空时间 | **80-200 分钟**（远超预期） |
| STUN 地址 | `127.0.0.1:3479`（已缓存，可用） |

### 1.2 日志证据

```
10:30:16  HandleEmpty → 弹出命令 → 下发 GPN
10:30:19  HandleEmpty → PopTask returned nil → 会话结束（HTTP 204）
          ↑ 此处会话结束，但 cmdq 仍有 ~3970 条命令

[等待 ~57 秒]

10:31:16  新 Inform → 新会话 → 继续处理...
10:32:16  新 Inform → 新会话...
...
10:37:19  大量 GPN Response 连续处理（每条 ~10ms）
```

**关键问题**：每次会话结束后，**系统没有任何动作**主动触发设备重新连接。

---

## 2. 当前架构分析

### 2.1 会话生命周期

```
CPE 发送 Inform
    ↓
ACS handleInform() → 创建 Session → 发送 InformResponse
    ↓
CPE 发送 Empty POST
    ↓
ACS handleEmpty()
    ├── PopTask() 有任务 → 发送 RPC → 等待响应
    │       ↓
    │   handleRPCResponse()
    │       ├── PopTask() 有任务 → 继续发送 RPC（循环）
    │       └── PopTask() 无任务 → completeSession() → HTTP 204
    │
    └── PopTask() 无任务 → completeSession() → HTTP 204
                                ↓
                    *** 会话结束，无后续动作 ***
                                ↓
                    等待下一次周期性 Inform（60s+）
```

### 2.2 Connection Request 触发点（仅一处）

**当前代码** (`task/service.go:84`)：

```go
func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
    // 1. 持久化到 PostgreSQL
    // 2. 推入 Redis 队列
    // 3. 唤醒设备（仅此一次）
    s.wakeDevice(task.DeviceSN)  // ← 唯一的 CR 触发点
    return task, nil
}
```

**问题**：`wakeDevice()` 只在**任务创建时**调用一次。后续无论队列积压多少命令，都不会再次触发。

### 2.3 completeSession() 当前实现

**文件**：`omcgo/internal/acs/handler.go:586-611`

```go
func (h *Handler) completeSession(ctx context.Context, session *Session) {
    h.admission.Release()              // 释放准入控制槽位
    h.metrics.ActiveSessions.Dec()     // 指标递减
    h.metrics.SessionDuration.Observe(duration)  // 记录时长
    session.State = StateComplete
    h.sessionStore.DeleteByID(ctx, session.ID)   // 删除会话
    // ← 结束。没有检查队列、没有触发 CR
}
```

### 2.4 Dedup 机制（30 秒窗口）

**文件**：`connreq/client.go:46-58`

HTTP Connection Request 有 30 秒去重窗口：
```go
dedupKey := "acs:connreq:pending:" + deviceSN
set, _ := c.redis.SetNX(ctx, dedupKey, "1", 30*time.Second).Result()
if !set {
    return nil  // 30 秒内已发送过，跳过
}
```

UDP Connection Request（STUN）**没有去重机制**。

---

## 3. 根因分析

### 3.1 设计缺陷：单次唤醒 vs 持续驱动

| 维度 | 当前设计 | 应有设计 |
|------|---------|---------|
| CR 触发时机 | 仅任务创建时 | 任务创建 + 会话结束时队列非空 |
| CR 触发次数 | 1 次 | 直到队列清空 |
| 会话结束动作 | 仅清理资源 | 检查残余 → 触发续唤 |
| 驱动模型 | 被动等待（靠周期 Inform） | 主动驱动（队列驱动） |

### 3.2 影响评估

**场景：批量参数发现（当前场景）**

```
创建 1 个发现任务 → 展开为 3994 条 GPN 命令 → wakeDevice() 触发 1 次 CR
    ↓
会话 1：处理 ~30 条（~3 秒）→ 结束 → 等 57 秒
会话 2：处理 ~30 条（~3 秒）→ 结束 → 等 57 秒
...
会话 133：处理最后 ~30 条 → 完成

总耗时 ≈ 133 × 60 秒 ≈ 133 分钟（2.2 小时）
```

**如果有续唤机制**：

```
会话 1：处理 ~30 条 → 结束 → 立即发 CR
    ↓ （设备 ~1-3 秒后重新 Inform）
会话 2：处理 ~30 条 → 结束 → 立即发 CR
...
会话 133：处理最后 ~30 条 → 完成（队列空，不发 CR）

总耗时 ≈ 133 × 4 秒 ≈ 9 分钟
```

**效率提升：~15 倍**

### 3.3 为什么每会话只处理 ~30 条？

从日志看，设备会在以下情况结束会话：
1. **CPE 主动断开**：某些 CPE 实现有最大 RPC 数限制或会话超时
2. **TCP 连接中断**：网络抖动导致连接断开
3. **CPE 重新 Inform**：CPE 在处理过程中触发了新的 Inform 事件，中断当前会话

实际上 ACS 侧对单会话 RPC 数量**没有上限**，瓶颈在 CPE 侧。

---

## 4. 解决方案设计

### 4.1 方案总览：会话结束续唤（Post-Session Wake）

在 `completeSession()` 中增加**队列检查 + 异步 CR 触发**逻辑：

```
completeSession(ctx, session)
    ├── 释放准入控制
    ├── 更新指标
    ├── 删除会话
    └── [新增] postSessionCheck(session.DeviceSN)
                ├── 检查 cmdq + task queue 深度
                ├── 如果 > 0 → 延迟发送 CR
                └── 如果 = 0 → 不做任何事
```

### 4.2 核心改动点

#### 改动 1：Handler 增加 ConnectionRequester 依赖

**文件**：`omcgo/internal/acs/handler.go`

```go
type Handler struct {
    // ... 现有字段
    connReqSender  ConnectionRequester  // [新增] 会话结束后续唤
    connReqConfig  PostSessionCRConfig  // [新增] 续唤配置
}

type PostSessionCRConfig struct {
    Enabled       bool          // 是否启用续唤
    DelayAfter    time.Duration // 会话结束后延迟多久发 CR（建议 1-2s）
    MaxContinuous int           // 单设备最大连续续唤次数（防止无限循环）
    CooldownTTL   time.Duration // 连续续唤冷却时间（防过热）
}
```

#### 改动 2：completeSession() 增加续唤逻辑

```go
func (h *Handler) completeSession(ctx context.Context, session *Session) {
    // ... 现有清理逻辑不变 ...

    // [新增] 异步检查队列并续唤
    if h.connReqSender != nil && h.connReqConfig.Enabled {
        go h.postSessionWake(session.DeviceSN)
    }
}

func (h *Handler) postSessionWake(deviceSN string) {
    ctx := context.Background()

    // 1. 检查队列深度
    remaining := h.getQueueDepth(ctx, deviceSN)
    if remaining == 0 {
        return  // 队列空，无需续唤
    }

    // 2. 检查连续续唤计数（防止无限循环）
    count, _ := h.incrContinuousWake(ctx, deviceSN)
    if count > h.connReqConfig.MaxContinuous {
        h.logger.Warn("post-session wake: max continuous reached",
            zap.String("device_sn", deviceSN),
            zap.Int("remaining", remaining),
            zap.Int64("continuous_count", count))
        return
    }

    // 3. 短暂延迟（给 CPE 喘息时间）
    time.Sleep(h.connReqConfig.DelayAfter)

    // 4. 发送 Connection Request
    if err := h.connReqSender.Send(ctx, deviceSN); err != nil {
        h.logger.Warn("post-session wake: CR failed",
            zap.String("device_sn", deviceSN),
            zap.Int("remaining", remaining),
            zap.Error(err))
    } else {
        h.logger.Info("post-session wake: CR sent",
            zap.String("device_sn", deviceSN),
            zap.Int("remaining", remaining),
            zap.Int64("continuous_count", count))
        h.metrics.PostSessionWakeTotal.Inc()
    }
}
```

#### 改动 3：连续续唤计数器（Redis）

```go
// Redis Key: acs:continuous_wake:{deviceSN}
// TTL: CooldownTTL（例如 5 分钟）
// 每次续唤 INCR，超过 MaxContinuous 停止
// 当 CPE 长时间不来（TTL 过期）自动重置

func (h *Handler) incrContinuousWake(ctx context.Context, deviceSN string) (int64, error) {
    key := "acs:continuous_wake:" + deviceSN
    pipe := h.redis.Pipeline()
    incrCmd := pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, h.connReqConfig.CooldownTTL)
    pipe.Exec(ctx)
    return incrCmd.Val(), nil
}

// 在 handleInform() 中重置计数器（设备主动来了，重置续唤计数）
func (h *Handler) resetContinuousWake(ctx context.Context, deviceSN string) {
    h.redis.Del(ctx, "acs:continuous_wake:"+deviceSN)
}
```

#### 改动 4：Dedup 窗口调整

当前 HTTP CR 的 30 秒 dedup 窗口会阻止续唤 CR 的发送（因为上一个会话刚结束不到 30 秒）。

**方案**：续唤 CR 走 UDP 通道（无 dedup），或为续唤 CR 使用独立的 dedup key：

```go
// 续唤 CR 用独立的 dedup key
dedupKey := "acs:connreq:postsession:" + deviceSN
// TTL 设为 DelayAfter 的 2 倍即可（例如 3 秒）
```

#### 改动 5：handleInform 重置续唤计数

```go
func (h *Handler) handleInform(w http.ResponseWriter, r *http.Request, ...) {
    // ... 现有逻辑 ...

    // [新增] 设备主动 Inform，重置续唤计数
    h.resetContinuousWake(r.Context(), deviceSN)

    // ... 后续处理 ...
}
```

### 4.3 配置建议

```yaml
# config.yaml
acs:
  post_session_wake:
    enabled: true
    delay_after: 1s          # 会话结束后等 1 秒再发 CR
    max_continuous: 200       # 单设备最多连续续唤 200 次
    cooldown_ttl: 5m          # 5 分钟无活动重置计数
```

**参数说明**：

| 参数 | 建议值 | 说明 |
|------|--------|------|
| `delay_after` | 1s | 给 CPE 处理上一会话的时间 |
| `max_continuous` | 200 | 安全阀：200 次 × 30 条/次 = 6000 条，足以覆盖参数发现 |
| `cooldown_ttl` | 5m | 超过 5 分钟没有会话则重置计数，允许重新续唤 |

### 4.4 状态机扩展

```
                                    ┌──────────────────────┐
                                    │                      │
CPE Inform → INFORM_RECEIVED        │                      ▼
                ↓                   │              [Post-Session CR]
           PROCESSING              │                      │
                ↓                   │              CPE 收到 CR
           RPC_PENDING             │                      │
                ↓                   │              CPE 发 Inform
           RPC_RESPONSE            │                      │
                ↓                   │              新 Session
           (more tasks?)            │              ┌───────┘
            ├── Yes → RPC_PENDING   │              │
            └── No  → COMPLETE ─────┘              │
                         │                          │
                    [队列非空?]                      │
                    ├── Yes → 延迟发 CR ────────────┘
                    └── No  → 结束（不发 CR）
```

---

## 5. 需要关注的边界条件

### 5.1 防止 CR 风暴

**风险**：如果续唤 CR 发出后设备没响应（网络不可达），会导致无效 CR 堆积。

**防护措施**：
- `max_continuous` 限制单设备续唤次数上限
- `cooldown_ttl` 超时自动重置
- 续唤 CR 失败后不再重试（不像 HTTP CR 的 3 次重试）

### 5.2 防止会话重叠

**风险**：CR 发出后 CPE 立即 Inform，但旧会话的清理还没完成。

**防护措施**：
- `delay_after = 1s` 确保旧会话完全清理
- Admission Control 已有机制阻止同设备并发会话
- Session Cookie 机制确保请求归属正确会话

### 5.3 CPE 侧压力

**风险**：连续不断的会话可能导致 CPE 资源紧张。

**防护措施**：
- `delay_after` 提供最小间隔
- CPE 可以拒绝 CR（返回非 200），此时停止续唤
- 大多数 CPE 设计上支持持续 TR-069 会话

### 5.4 队列深度检查的性能

**Redis 命令**：`ZCARD acs:cmdq:{deviceSN}` + `ZCARD acs:task:queue:{deviceSN}`

两个 `O(1)` 操作，性能无影响。

### 5.5 UDP vs HTTP 选择

| 方式 | 续唤场景适用性 | 延迟 | 可靠性 |
|------|--------------|------|--------|
| UDP（STUN） | 优先 | <10ms | 较低（需重试） |
| HTTP | 备选 | 50-500ms | 较高 |

续唤场景建议**优先 UDP**：延迟低，且绕过 HTTP 的 30 秒 dedup。

---

## 6. 涉及文件清单

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `internal/acs/handler.go` | **核心改动** | completeSession() 增加续唤逻辑 |
| `internal/acs/handler.go` | **核心改动** | handleInform() 增加计数重置 |
| `internal/acs/handler.go` | **结构改动** | Handler 新增 connReqSender 字段 |
| `internal/core/appconfig/config.go` | **配置扩展** | 新增 PostSessionWakeConfig |
| `cmd/acs/main.go` | **DI 注入** | 将 connReqSender 注入 Handler |
| `cmd/acs/etc/config.dev.yaml` | **配置** | 添加 post_session_wake 配置段 |
| `internal/acs/connreq/dispatcher.go` | **可选** | 增加续唤专用发送方法 |
| `internal/acs/metrics.go` | **指标** | 新增 PostSessionWakeTotal 等指标 |

---

## 7. 实施计划

### 阶段 1：核心机制（预计改动量：~150 行）

1. 在 `Handler` 结构体中增加 `ConnectionRequester` 接口和配置
2. 实现 `postSessionWake()` 方法
3. 在 `completeSession()` 中调用
4. 在 `handleInform()` 中重置续唤计数
5. 配置文件增加 `post_session_wake` 段

### 阶段 2：DI 注入与测试

1. `cmd/acs/main.go` 注入依赖
2. 单元测试：`postSessionWake` 在队列非空时触发 CR
3. 单元测试：`postSessionWake` 在队列空时不触发 CR
4. 单元测试：超过 `max_continuous` 时停止续唤
5. E2E 测试：参数发现场景下的连续会话执行

### 阶段 3：监控与调优

1. Prometheus 指标：续唤触发次数、成功率
2. Grafana 面板：队列消化速度对比（有续唤 vs 无续唤）
3. 根据实际设备表现调优 `delay_after` 和 `max_continuous`

---

## 8. 预期效果

| 指标 | 改进前 | 改进后 |
|------|--------|--------|
| 3994 条命令消化时间 | ~133 分钟 | ~9 分钟 |
| 会话间等待时间 | ~57 秒（周期 Inform） | ~2 秒（CR 延迟 + Inform） |
| 每分钟处理命令数 | ~30 条 | ~450 条 |
| 效率提升 | 基准 | **~15x** |

---

## 9. 结论

当前 ACS 的 Connection Request 仅在**任务创建时**触发一次，会话结束后完全依赖设备的周期性 Inform 来消化积压命令。这在大量命令（如参数发现）场景下导致严重的效率瓶颈。

**核心修复**：在 `completeSession()` 中增加队列深度检查和异步 CR 触发，实现"队列不空则续唤"的驱动模型。改动量小（~200 行），风险可控（有 max_continuous 安全阀），预期效率提升 ~15 倍。

---

## 10. 实施验证记录 (2026-03-23)

### 10.1 验证过程

| 步骤 | 结果 |
|------|------|
| 编译 `go build ./...` | 通过 |
| 测试 `go test ./internal/acs/...` | 全部通过（8 个包） |
| 重启服务并观察日志 | postSessionWake 正常触发 |

### 10.2 日志验证

```
12:13:17 post-session wake: started (device_sn=1202000588233HB0039)
12:13:18 post-session wake: queue check after delay (cmd_queue_len=8899, remaining=8899)
12:13:28 connection request failed (device=10.10.3.64:7547, timeout)
```

### 10.3 发现的问题与修复

| 问题 | 原因 | 修复 |
|------|------|------|
| postSessionWake 日志不出现 | 队列深度检查在 delay 之前，异步命令入队尚未完成导致 remaining=0 提前返回 | 将 `time.Sleep(delay)` 移到队列检查之前 |
| CR 发送失败：no connection request method | Dispatcher 只有 UDP sender（无 STUN 绑定），没有 HTTP client | 增加 HTTP client 到 Dispatcher |
| CR 发送失败：设备不可达 | 设备 `10.10.3.64` 在 NAT 后面，HTTP CR 超时 | ① 减少超时至 5s ② 从 Inform 缓存 UDPConnectionRequestAddress 到 STUN Store |
| 设备无 STUN 绑定 | 设备未配置 STUN keepalive 指向 ACS 的 STUN 服务器 | 需设备侧配置（非代码问题） |

### 10.4 当前状态

- **代码层面**：Post-Session Wake 机制完整实现，支持 STUN UDP + HTTP 双通道
- **运行时**：依赖设备网络可达性
  - 如果设备可直接访问（同网段/端口映射）→ HTTP CR 立即生效
  - 如果设备发送 STUN keepalive → UDP CR 穿越 NAT 生效
  - 如果两者都不可达 → 回退到被动等待周期 Inform（原有行为）
- **从 Inform 自动缓存 CR 地址**：handler 从 Inform 参数中提取 `ConnectionRequestURL` 和 `UDPConnectionRequestAddress`，无需额外配置
