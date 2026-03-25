# ACS completeSession 会话结束机制分析

> 分析 TR069 ACS 引擎中 `completeSession` 的完整调用链路，覆盖所有会话结束场景及 postSessionWake 续唤机制。

---

## 1. 概述

`completeSession` 是 ACS 会话生命周期的**唯一出口**。无论会话以何种方式结束，都必须经过此函数统一收口，确保资源释放、指标记录和命令队列续唤。

**核心职责**：

```go
// handler.go:775
func (h *Handler) completeSession(ctx context.Context, session *Session) {
    h.admission.Release()                    // 1. 释放准入槽位
    h.metrics.ActiveSessions.Dec()           // 2. 递减活跃计数
    h.metrics.SessionDuration.Observe(...)   // 3. 记录会话时长
    session.State = StateComplete            // 4. 标记状态完成
    h.sessionStore.DeleteByID(...)           // 5. 清除 Redis 会话
    h.deviceSessions.Delete(...)             // 6. 清除设备→会话映射
    go h.postSessionWake(session.DeviceSN)   // 7. 异步续唤（检查队列是否还有命令）
}
```

---

## 2. 全部调用点

共 **7 个调用点**，覆盖 4 大类场景：

| # | 位置 | 函数 | 场景 | 触发条件 |
|---|------|------|------|---------|
| 1 | `handler.go:165` | `reapOrphanedSession` | 孤儿会话清理（Redis 存在） | 新 Inform 到达 / 定时 reaper |
| 2 | `handler.go:503` | `handleEmpty` | RPC 限额耗尽 | `RPCCount >= maxRPCPerSession` |
| 3 | `handler.go:599` | `handleEmpty` | 队列排空（正常结束） | TaskQueue + CommandQueue 均为空 |
| 4 | `handler.go:689` | `handleRPCResponse` | RPC 限额耗尽 | 处理 RPC 响应后 RPCCount 达上限 |
| 5 | `handler.go:766` | `handleRPCResponse` | 队列排空（正常结束） | 处理 RPC 响应后队列为空 |
| 6 | `handler.go:1018` | `handleSOAPFault` | SOAP Fault 后队列排空 | CPE 拒绝命令且无后续命令 |
| 7 | `handler.go:173-179` | `reapOrphanedSession` | 孤儿会话（Redis 已过期） | 手动释放资源 + postSessionWake |

---

## 3. 四大场景详解

### 3.1 场景一：正常结束 — 队列排空

**调用点**：`handleEmpty:599` + `handleRPCResponse:766`

**占比**：~90% 的会话通过此路径结束。

**流程**：

```
CPE 发送 Inform
     ↓
ACS 返回 InformResponse
     ↓
CPE 发送 Empty POST（准备接收命令）
     ↓
handleEmpty() → PopTask() / CommandQueue.Pop()
     ↓
[有命令] → 下发 RPC → CPE 返回响应 → handleRPCResponse()
     ↓                                    ↓
     ↓                              PopTask() / Pop()
     ↓                                    ↓
     ↓                              [有命令] → 继续循环
     ↓                              [无命令] → completeSession() ← 调用点 5
     ↓
[无命令] → completeSession() ← 调用点 3
     ↓
204 No Content + Connection: close
     ↓
CPE 关闭 TCP 连接，会话结束
```

**TR069 协议要求**：ACS 无更多命令时必须返回空 HTTP 响应（204 或空 body），CPE 收到后关闭连接。

**postSessionWake 行为**：检查队列为空 → 跳过，不发 Connection Request。

---

### 3.2 场景二：RPC 限额截断

**调用点**：`handleEmpty:503` + `handleRPCResponse:689`

**触发条件**：`session.RPCCount >= maxRPCPerSession`

**背景**：部��低端 CPE ��单次会话中的 RPC 交互次数有硬件/固件限制。超过限制可能导致 CPE 内存溢出或行为异常。ACS 通过 `maxRPCPerSession` 配置项主动控制。

**流程**：

```
会话已执行 N 次 RPC（如 maxRPCPerSession=10）
     ↓
handleEmpty() / handleRPCResponse() 检查 RPCCount
     ↓
RPCCount >= 10 → 队列中还有 50 条命令未执行
     ↓
completeSession() ← 主动结束当前会话
     ↓
204 No Content + Connection: close
     ↓
postSessionWake() → 检查队列 remaining=50 > 0
     ↓
发送 Connection Request → CPE 秒级重连
     ↓
新会话 → 继续处理剩余 50 条命令（分多个会话完成）
```

**两个检查时机的区别**：

| 调用点 | 时机 | 说明 |
|--------|------|------|
| `handleEmpty:503` | Empty POST 到达，准备下发第一条命令前 | 防止从上一会话累计的 RPCCount 继续 |
| `handleRPCResponse:689` | 收到 RPC 响应，准备下发下一条命令前 | 每次 RPC 响应后递增 RPCCount 并检查 |

**postSessionWake 行为**：检测到 remaining > 0 → 发 CR → 设备重连 → 新会话继续。

---

### 3.3 场景三：SOAP Fault 后排空

**调用点**：`handleSOAPFault:1018`

**触发条件**：CPE 返回 SOAP Fault（如 `Method not supported`、`Internal error`、`Invalid arguments`）。

**流程**：

```
ACS 下发 RPC 命令（如 SetParameterValues）
     ↓
CPE 返回 SOAP Fault（拒绝执行）
     ↓
handleSOAPFault()
     ↓
标记当前 Task 为 Failed
     ↓
尝试 Pop 下一条命令
     ↓
[有命令] → 继续下发（不结束会话，一条 Fault 不阻塞后续命令）
[无命令] → completeSession() ← 调用点 6
     ↓
204 No Content + Connection: close
```

**设计决策**：SOAP Fault **不立即终止会话**。ACS 跳过失败命令，继续执行队列中的后续命令。这保证了批量操作中单条失败不会阻塞其余命令。

**postSessionWake 行为**：取决于队列是否还有命令。

---

### 3.4 场景四：异常清理 — 孤儿会话

**调用点**：`reapOrphanedSession:165`（Redis 存在）+ `:173-179`（Redis 过期）

**触发条件**：CPE 崩溃、断电、网络中断，导致 ACS 侧的会话永远等不到后续 HTTP 请求。

**触发方式**：

| 触发源 | 函数 | 条件 |
|--------|------|------|
| 新 Inform | `handleInform()` → `deviceSessions.LoadAndDelete()` → `reapOrphanedSession("new_inform")` | 同一设备发起新 Inform，发现旧会话映射仍在 |
| 定时 Reaper | `startSessionReaper()` → `deviceSessions.Range()` → `reapOrphanedSession("reaper")` | 后台 goroutine 扫描超�� `maxAge` 的会话 |

**流程（新 Inform 触发）**：

```
CPE-A 会话进行中（等待 RPC 响应）
     ↓
CPE-A 断电重启
     ↓
CPE-A 发送新 Inform（0 BOOTSTRAP / 1 BOOT）
     ↓
handleInform() → 检查 deviceSessions["CPE-A"]
     ↓
发现旧会话映射 → LoadAndDelete
     ↓
reapOrphanedSession(旧会话, "new_inform")
     ↓
┌─ Redis 中旧会话还在（TTL 未过期）
│  → 加载完整 Session → completeSession(旧 Session)  ← 调用点 1
│  → 释放 admission + 清除映射 + postSessionWake
│
└─ Redis 中旧会话已过期（TTL 5min 已到）
   → session == nil，无法走 completeSession
   → 手动释放：admission.Release() + ActiveSessions.Dec()  ← 调用点 7
   → 仍然 go postSessionWake()（队列可能还有命令）
```

**流程（定时 Reaper 触发）**：

```
startSessionReaper(interval=30s, maxAge=5min)
     ↓
每 30s 扫描 deviceSessions
     ↓
发现某 entry 的 CreatedAt 超过 5 分钟
     ↓
deviceSessions.Delete(deviceSN)
     ↓
reapOrphanedSession(entry, "reaper") → 同上两条分支
```

**解决的问题**：

| 问题 | 不清理的后果 |
|------|------------|
| admission 槽位泄漏 | 并发上限被僵尸会话占满，新设备无法接入 |
| ActiveSessions 虚高 | 监控指标失真，无法反映真实负载 |
| 命令队列停滞 | 设备有待下发命令但无人触发续唤 |
| 内存泄漏 | `deviceSessions` sync.Map 条目持续增长 |

---

## 4. postSessionWake 续唤机制

`completeSession` 的最后一步是异步触发 `postSessionWake`，这是 ACS 命令下发效率的关键优化。

### 4.1 解决的核心问题

TR069 协议中，CPE 按 `InformInterval`（60-300s）周期性连接 ACS。如果会话结束时队列还有命令，传统方式需要等待下一个 Periodic Inform：

```
无 postSessionWake:  100 条命令 ÷ 5 条/会话 × 300s = 6000s ≈ 100 分钟
有 postSessionWake:  100 条命令 ÷ 5 条/会话 × ~2s  = ~40s
```

### 4.2 执行流程

```
completeSession()
  └── go postSessionWake(deviceSN)
        │
        ├── 1. panic recover（防止 goroutine 静默死亡）
        │
        ├── 2. Sleep(DelayAfter = 1s)
        │      等待 CPE 关闭 TCP + EventBus 订阅者处理完响应并追加新命令
        │
        ├── 3. 检查队列深度
        │      TaskService.GetQueueLength + CommandQueue.Len
        │      remaining == 0 → return（无需续唤）
        │
        ├── 4. 连续续唤计数器
        │      Redis INCR acs:continuous_wake:{sn}
        │      count > MaxContinuous(200) → return（退避防死循环）
        │
        └── 5. 发送 Connection Request
               connReqSender.Send(deviceSN, httpURL)
               → CPE 收到后立即发 Inform → 新会话开始
```

### 4.3 三大防护机制

| 机制 | 配置项 | 默认值 | 防止的问题 |
|------|--------|--------|-----------|
| 延迟检查 | `delay_after` | 1s | EventBus 异步订阅者还没追加新命令，导致误判队列为空 |
| 连续唤醒上限 | `max_continuous` | 200 | 命令持续失败重入队列导致的无限唤醒循环 |
| 冷却 TTL | `cooldown_ttl` | 5min | 超上限后自动恢复，TTL 过期后计数器重置 |

### 4.4 计数器重置

`resetContinuousWake()`（`handler.go:936`）在每次设备发起新 Inform 时调用：

```go
func (h *Handler) resetContinuousWake(ctx context.Context, deviceSN string) {
    h.redisClient.Del(ctx, "acs:continuous_wake:"+deviceSN)
}
```

设备主动连接说明它工作正常，可以安全重置计数器。

---

## 5. 各场景 postSessionWake 行为对照

| 场景 | 队列状态 | postSessionWake 行为 |
|------|---------|---------------------|
| 正常结束（队列排空） | remaining = 0 | 检查后跳过，不发 CR |
| RPC 限额截断 | remaining > 0 | 发 CR，设备秒级重连继续 |
| SOAP Fault 后排空 | remaining = 0 | 检查后跳过 |
| SOAP Fault 但还有命令 | remaining > 0 | 发 CR，继续处理 |
| 孤儿会话清理 | 取决于队列 | 队列有命令则发 CR |
| 自动开站（大量命令） | remaining >> 0 | 持续续唤直到队列排空或达上限 |

---

## 6. 会话结束的完整状态流转

```
                         TR069 会话状态机
                         ══════════════

   IDLE ──Inform──→ INFORM_RECEIVED ──Empty──→ PROCESSING
                                                    │
                                              ┌─────┴──────┐
                                              │            │
                                         [有命令]     [无命令]
                                              │            │
                                              ▼            │
                                        RPC_PENDING        │
                                              │            │
                                       CPE 响应/Fault      │
                                              │            │
                                              ▼            │
                                        RPC_RESPONSE       │
                                              │            │
                                        ┌─────┴──────┐    │
                                        │            │    │
                                   [有命令]     [无命令]  │
                                        │            │    │
                                        ▼            │    │
                                   RPC_PENDING       │    │
                                     (循环)          │    │
                                                     ▼    ▼
                                                  COMPLETE
                                                     │
                                              completeSession()
                                                     │
                                         ┌───────────┴───────────┐
                                         │                       │
                                    释放资源                postSessionWake
                                  admission.Release()           │
                                  ActiveSessions.Dec()    ┌─────┴─────┐
                                  Redis 会话删除          │           │
                                  设备映射清除        [队列空]   [队列有命令]
                                                      skip      发 CR
                                                              设备重连
                                                              → 新 IDLE
```

---

## 7. 关键代码索引

| 函数 | 文件位置 | 说明 |
|------|---------|------|
| `completeSession` | `internal/acs/handler.go:775` | 统一会话结束出口 |
| `postSessionWake` | `internal/acs/handler.go:819` | 异步续唤检查与 CR 发送 |
| `resetContinuousWake` | `internal/acs/handler.go:936` | Inform 时重置续唤计数器 |
| `reapOrphanedSession` | `internal/acs/handler.go:145` | 孤儿会话清理 |
| `startSessionReaper` | `internal/acs/handler.go:106` | 后台定时清理 goroutine |
| `handleEmpty` | `internal/acs/handler.go:453` | 处理 CPE Empty POST |
| `handleRPCResponse` | `internal/acs/handler.go:607` | 处理 CPE RPC 响应 |
| `handleSOAPFault` | `internal/acs/handler.go:945` | 处理 CPE SOAP Fault |
| `PostSessionWakeConfig` | `internal/core/appconfig/config.go:46` | 续唤配置结构体 |

---

## 8. 配置参考

```yaml
acs:
  max_rpc_per_session: 50          # 单会话最大 RPC 次数（0=不限）
  session_ttl: 5m                   # Redis 会话 TTL
  session_reaper_interval: 30s      # 孤儿会话扫描间隔
  session_reaper_max_age: 5m        # 会话最大存活时间

  post_session_wake:
    enabled: true                   # 续唤总开关
    delay_after: 1s                 # 会话结束后延迟检查时间
    max_continuous: 200             # 单设备最大连续续唤次数
    cooldown_ttl: 5m                # 续唤计数器冷却 TTL
```
