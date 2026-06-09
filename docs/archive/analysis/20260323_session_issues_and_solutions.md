# ACS 会话管理问题分析与解决方案

**日期**: 2026-03-23
**状态**: 待实施
**基于**: 基站 `1202000588233HB0039` 实际连接日志分析（16 次 Inform，3 分钟，645 次 HTTP 交互）

---

## 问题一：随机测试任务注入导致队列无法清空

### 现象

每次 Inform 都会注入 6-10 个随机测试任务（Download、Upload、Reboot、GPN、GPV、SPV 等），16 次 Inform 共创建 72 个任务。GPN 任务触发参数树发现后还会递归产生数百条 cmdq 子命令。结果：

| 指标 | 数值 |
|------|------|
| 任务创建 | 72 个 |
| cmdq 残留 | 375 条（持续增长） |
| Redis task keys 残留 | 72 个 |

队列永远无法清空，因为注入速度 > 消费速度。

### 根因

`handler.go:322-323` — `injectRandomTestTasks()` 在每次非 TC/ATC 的 Inform 都会执行：

```go
if !isTC && !isATC {
    h.injectRandomTestTasks(r, deviceSN, log)
}
```

这是一个测试功能，生产环境中不应存在。但当前阶段作为开发验证工具，问题是**无法控制注入频率和总量**。

### 解决方案

**方案 A（推荐）：仅在首次 BOOT 时注入 + 配置开关**

```go
// config.dev.yaml
acs:
  test_task_injection:
    enabled: true           # 总开关
    trigger_events:         # 仅在这些事件时注入
      - "0 BOOTSTRAP"
      - "1 BOOT"
    max_per_device: 50      # 每设备最大注入总量
    cooldown: 5m            # 同一设备注入冷却时间
```

实现要点：
1. 在 `Handler` 增加 `testInjectionCfg` 配置字段
2. 注入前检查事件类型是否在 `trigger_events` 列表中
3. 用 Redis 计数器 `acs:test_inject_count:{deviceSN}` 跟踪已注入数量
4. 用 Redis TTL key `acs:test_inject_cd:{deviceSN}` 实现冷却时间

**方案 B（快速方案）：移除 `injectRandomTestTasks` 调用**

直接注释掉 `handler.go:323` 的调用。测试任务改为通过 API 手动创建。

**方案 C：限制每次注入数量**

```go
// 每次 Inform 最多注入 2 个任务，且不包含 Reboot/FactoryReset
// 这样队列增长可控
```

### 建议

开发阶段选 **方案 A**，通过配置灵活控制。生产部署前切换为 `enabled: false`。

---

## 问题二：会话孤立导致 Post-Session Wake 丢失

### 现象

16 次 Inform 中，仅 4 次触发了 `postSessionWake` goroutine，其余 12 个会话的 `completeSession()` **从未被调用**。

关键证据：
```
12:29:31.641  最后一条 GPN 响应处理完成（parameter_count=2）
              handleRPCResponse 从 cmdq Pop 出新命令，发送 GPN 请求给 CPE
              handler 返回，等待 CPE 下一次 HTTP 请求

12:30:30.217  CPE 发送 PERIODIC Inform（59 秒后！）
              handleInform 创建全新 session
              旧 session 永远不会被 completeSession()
```

### 根因分析

**会话孤立的三个场景**：

#### 场景 1：CPE 不响应 RPC 请求（占多数）

```
ACS 发送 GPN/GPV 请求 → CPE 应该回 GPN/GPV 响应
但 CPE 等到 PERIODIC 定时器到期 → 发送新 Inform
旧会话悬挂，completeSession 永远不调用
```

时间线证据：
- 12:29:31 → 12:30:30（59 秒空白，旧 session 孤立）
- 12:30:33 → 12:31:30（57 秒空白，同样孤立）

#### 场景 2：Reboot 命令导致 CPE 断开

```
ACS 发送 Reboot → CPE 回 RebootResponse → CPE 立即重启
handleRPCResponse 处理完 RebootResponse → Pop 下一个命令 → 发送
但 CPE 已经断开，响应写入失败或 CPE 无法读取
CPE 重启后发 M Reboot Inform → 新 session
旧 session 孤立
```

#### 场景 3：Inform 打断活跃会话

```
当前 session 正在 RPC 交互中
CPE 因某些原因（STUN CR、Periodic等）发起新 Inform
handleInform 创建新 session，旧 session 无人清理
```

### 影响

| 泄漏资源 | 后果 |
|----------|------|
| Admission 槽位 | 并发上限逐渐被占满，新设备无法接入 |
| Redis session 数据 | 内存浪费（等 TTL 过期自动清理） |
| PostSessionWake 丢失 | 队列中的命令只能等下一个 PERIODIC Inform 处理 |
| ActiveSessions 指标 | 只增不减，监控数据失真 |

### 解决方案

**方案：handleInform 接管旧会话 + Session Reaper 双保险**

#### 修改 1：handleInform 中清理旧会话

在 `handleInform()` 创建新 session 之前，检查设备是否有活跃 session，如有则先调用 `completeSession()`：

```go
// handler.go — handleInform() 中，Admission 之后、创建新 session 之前

// 清理该设备的旧会话（防止会话孤立）
if oldSessions := h.sessionStore.FindByDeviceSN(ctx, deviceSN); len(oldSessions) > 0 {
    for _, old := range oldSessions {
        log.Info("cleaning orphaned session before new Inform",
            zap.String("device_sn", deviceSN),
            zap.String("old_session_id", old.ID),
            zap.String("old_state", string(old.State)),
            zap.Duration("age", time.Since(old.StartedAt)),
        )
        h.completeSession(ctx, old)
    }
}

// Create new session
sessionID := generateSessionID()
session := &Session{...}
```

**需要新增 SessionStore 方法**：

```go
// SessionStore 接口新增
FindByDeviceSN(ctx context.Context, deviceSN string) []*Session
```

Redis 实现：维护一个 `acs:device_session:{deviceSN}` 键存储当前 session ID，在 `CreateWithID` 时更新，在 `DeleteByID` 时清除。

#### 修改 2：简化方案（无需修改 SessionStore）

利用现有的 Cookie-based session 机制。由于 CPE 发新 Inform 时，旧 session 的 Cookie 已失效（新请求没带旧 Cookie），所以旧 session 已经"不可达"。

**替代方案：在 handleInform 中直接触发 postSessionWake**

```go
// handleInform() 中，resetContinuousWake 之前
// 每次 Inform 都检查队列，如果有残留则触发 wake（处理上一个孤立会话的遗留）
go h.postSessionWake(deviceSN)
```

这个方案更简单：不需要找旧 session，不需要修改 SessionStore。只是在每次 Inform 时异步检查队列并续唤。但需要处理 Admission 泄漏问题。

#### 修改 3：Session Reaper（兜底保险）

后台 goroutine 定期扫描活跃 session，清理超时的：

```go
func (h *Handler) startSessionReaper(interval, maxAge time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        ctx := context.Background()
        stale := h.sessionStore.FindStale(ctx, maxAge) // 更新时间 > maxAge 的
        for _, s := range stale {
            h.logger.Warn("reaping stale session",
                zap.String("device_sn", s.DeviceSN),
                zap.String("session_id", s.ID),
                zap.Duration("age", time.Since(s.UpdatedAt)))
            h.completeSession(ctx, s)
        }
    }
}
```

配置建议：
```yaml
acs:
  session_reaper:
    enabled: true
    interval: 30s       # 扫描间隔
    max_idle: 60s       # session 最大空闲时间（> Periodic Inform 间隔）
```

#### 修改 4：修复 Admission 泄漏

在 `handleInform()` 中，如果检测到设备已有活跃 session，**不重复 Acquire admission**，而是复用旧槽位：

```go
// 如果旧 session 存在且 admission 已 acquired，不需要再 acquire
if oldSession != nil {
    // 旧 session 的 admission 会在 completeSession 中 Release
    // 新 session 需要自己的 admission
    // 但由于 completeSession 先 Release，所以这里 Acquire 不会阻塞
}
```

或者更简单：在 `completeSession` 之前先 Release 旧的，再 Acquire 新的。

### 推荐实施顺序

| 步骤 | 修改 | 复杂度 | 效果 |
|------|------|--------|------|
| 1 | handleInform 中触发 postSessionWake | 低 | 解决 wake 丢失 |
| 2 | handleInform 中清理旧 session（加设备→session 映射） | 中 | 解决 admission 泄漏 |
| 3 | Session Reaper 后台扫描 | 中 | 兜底保险 |
| 4 | 测试任务注入配置化 | 低 | 队列可控 |

### 步骤 1 的最小改动

```go
// handler.go handleInform() 中，现有 resetContinuousWake 之前加一行：

// 触发 post-session wake — 处理上一个会话可能遗留的队列命令
// （当 CPE 不响应 RPC 或因 Reboot 断开时，旧会话的 completeSession 不会被调用）
if h.connReqSender != nil && h.postSessionWakeCfg.Enabled {
    // 不需要异步，因为 Inform 本身就意味着设备在线
    // 检查队列即可，如果有残留说明旧会话遗留了未处理的命令
    // 但设备已经在线了（正在发 Inform），所以不需要发 CR
    // 只需要确保本次会话结束后能触发 wake
    // 实际上什么都不用做 — 设备已经连上了，本次会话会处理队列
}
```

等等，仔细想：设备已经在发 Inform 了，说明设备在线。本次 Inform 后的 HandleEmpty 会 Pop 队列中的命令并处理。所以**队列命令不会丢失**，只是要等到本次会话处理。

**真正的问题是**：如果本次会话又因为同样的原因（CPE 不响应 RPC）而孤立，命令又会积压。这是一个循环问题。

**根本解决方案是 Session Reaper**：确保任何孤立会话都被及时清理，触发 completeSession → postSessionWake。

---

## 综合实施计划

### Phase 1：快速修复（预计 1 小时）

1. **测试任务注入**：注释掉 `injectRandomTestTasks` 调用，或添加 `enabled` 配置开关
2. **Admission 泄漏**：在 `handleInform` 创建新 session 时，用 `sync.Map` 记录 `deviceSN → sessionID` 映射，新 Inform 到达时清理旧 session

### Phase 2：Session Reaper（预计 2 小时）

1. 实现 `startSessionReaper` goroutine
2. 在 `server.go` 中启动 reaper
3. reaper 检测 `UpdatedAt` 超过 `max_idle` 的 session，调用 `completeSession`
4. 添加 `acs_session_reaped_total` Prometheus 指标

### Phase 3：设备→会话映射优化（预计 2 小时）

1. `SessionStore` 增加 `FindByDeviceSN` 方法
2. 维护 Redis 二级索引 `acs:device_session:{deviceSN} → sessionID`
3. `handleInform` 中先查旧 session → `completeSession` → 创建新 session
4. 优雅处理 admission 槽位转移

---

## 附录：会话结束路径分析

```
handleEmpty (line 476)     — "no tasks found" → completeSession ✅ 有日志
handleRPCResponse (line 621) — pop empty → completeSession ✅ 无日志（需补充）
CPE 断开 / 不响应         — handler 已返回 → ❌ completeSession 永远不调用
Session TTL 过期           — Redis 自动删除 → ❌ 不触发 completeSession
PERIODIC Inform 覆盖       — 新 session 覆盖旧 → ❌ 旧 session 不清理
```

正常结束的会话只有前两种路径，后三种都会导致会话孤立。Session Reaper 是唯一能覆盖所有异常路径的兜底方案。
