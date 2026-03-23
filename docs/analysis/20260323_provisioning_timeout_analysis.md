# Provisioning 任务超时机制分析报告

> 日期：2026-03-23
> 状态：当前方案已实现，增强方案待评估

---

## 1. 问题背景

Provisioning Task 采用去重机制：每个设备同一时间只允许一个非终态任务。当设备离线导致任务卡在 `discovering`/`verifying`/`syncing` 等状态时，该设备后续的 BOOT 将被 HandleBootstrap 的去重检查阻断，无法触发新的自动开站流程。

```
HandleBootstrap (engine.go:139-148):
    existingTask, _ := e.taskRepo.GetByDeviceID(ctx, evt.DeviceID)
    if existingTask != nil && !IsTerminal(existingTask.Status) {
        // → 跳过，不创建新任务
        return nil
    }
```

---

## 2. 当前方案：Task Reaper（已实现）

### 机制

| 参数 | 值 | 说明 |
|------|-----|------|
| timeout | 15 分钟 | 非终态任务超时阈值（基于 `updated_at`） |
| interval | 7.5 分钟 | 扫描周期 = timeout / 2 |
| 触发 | `FailStale()` | 将超时任务标记为 `failed` |

### 工作流程

```
[Task 卡住] ──── 0~7.5min ────→ [Reaper 扫描] ──→ [标记 failed]
                                                          │
[设备重连 BOOT] ────────────────────────────────────────→ [去重通过] → [新 Task]
```

### 最坏等待时间分析

```
场景1: 任务刚超时 → 下次扫描在 0~7.5min 内 → 最坏 7.5min
场景2: 任务在扫描间隔中段超时 → 等下次扫描 → 最坏 15min (timeout)
场景3: 设备重连时 Reaper 恰好刚扫描完 → 等下个周期 → 最坏 ~22min
```

**平均等待**: 约 11 分钟（从任务实际卡住到被清理）

### 优点

- **极其简单**: 一个后台 goroutine + 一条 SQL UPDATE
- **无侵入**: 不修改任何现有流程，纯粹是兜底安全网
- **可配置**: `task_timeout` 配置项可调
- **覆盖全面**: 无论什么原因导致任务卡住都能最终清理

### 缺点

- **不够及时**: 最坏 22 分钟的空窗期，设备重连后不能立即开始新流程
- **盲目清理**: 无法区分"设备离线导致卡住"和"任务确实还在执行中"
- **发现进度丢失**: 如果 auto-discovery 已探索了 80% 的参数树，Reaper 失败任务后重来

---

## 3. Discovery 流程深度分析

### 状态分布

Discovery（Path C）是一个**跨多个 TR-069 会话、多轮 RPC** 的异步流程：

```
StartDiscovery
  ├── 创建 ParameterDiscoveryLog (DB)
  ├── 设置 Redis: pending=1, params=[]
  └── 入队 GPN(Device., NextLevel=true) 到 cmdqueue

    ┌─── 会话1: Inform → GPN(Device.) Response ────┐
    │  HandleLevelGPNResponse:                      │
    │    - 收集 leaf params → Redis RPush           │
    │    - 发现 sub-objects → 入队更多 GPN          │
    │    - pending: +N(子对象) -1(本次完成)          │
    └───────────────────────────────────────────────┘

    ┌─── 会话2: Inform → GPN(Device.ManagementServer.) Response ─┐
    │    ... 同上，递归展开 ...                                    │
    └──────────────────────────────────────────────────────────────┘

    ┌─── 会话N: pending=0 → finalizeDiscovery ─────┐
    │  - Redis LRange 读取全部 leaf params          │
    │  - buildParameterTree → 构建参数树 JSON       │
    │  - 创建 DataModel (PostgreSQL)                │
    │  - 清理 Redis tracking keys                   │
    │  - 如果 auto_sync 开启 → 自动触发参数同步     │
    └───────────────────────────────────────────────┘
```

### 关键特征

| 特征 | 说明 |
|------|------|
| **跨会话** | 每次 GPN Response 在独立的 TR-069 会话中返回 |
| **状态在 Redis** | `pending` 计数器 + `params` 列表，TTL 1 小时 |
| **结果持久化** | DataModel 写入 PostgreSQL，后续设备可直接使用 |
| **增量积累** | 每次 GPN Response 的 leaf params 追加到 Redis list |
| **终止条件** | `pending` 计数器归零 = 所有层级探索完毕 |

### 中断场景

当设备在 discovery 过程中离线：

```
状态1 (Redis):  pending=5, params=[...300个参数...]
状态2 (DB):     provisioning_task.status = "discovering"
状态3 (DB):     parameter_discovery_logs.status = "discovering"
状态4 (cmdqueue): 还有 5 个 GPN 命令在队列中

设备离线 →
  - cmdqueue 中的 GPN 命令等不到设备来取
  - Redis pending 永远不会归零
  - Redis TTL 1小时后自动过期
  - provisioning_task 卡在 "discovering"
  - Reaper 15分钟后清理 task → 但不清理 Redis discovery 状态
```

---

## 4. 增强方案评估

### 方案 A: Reset-on-BOOT（推荐的下一步增强）

**核心思路**: HandleBootstrap 发现已有非终态任务时，不再跳过，而是**失败旧任务并重新开始**。

```go
// HandleBootstrap 中的修改（伪代码）
existingTask, _ := e.taskRepo.GetByDeviceID(ctx, evt.DeviceID)
if existingTask != nil && !IsTerminal(existingTask.Status) {
    e.logger.Warn("cancelling stale task for reconnected device",
        zap.String("existing_status", string(existingTask.Status)))
    _ = e.failTask(ctx, existingTask,
        fmt.Errorf("device reconnected with new BOOT, cancelling stale task"))
    // 清理可能残留的 discovery Redis 状态
    e.cleanupDiscoveryState(ctx, evt.SerialNumber)
    // 继续执行，创建新任务
}
```

| 维度 | 评估 |
|------|------|
| 复杂度 | **低** — 修改 HandleBootstrap ~15 行 + 新增 cleanupDiscoveryState ~10 行 |
| 及时性 | **立即** — 设备重连的 BOOT Inform 即刻触发新流程 |
| 兼容性 | **好** — Reaper 仍作为兜底（设备不再重连的场景） |
| 风险 | **低** — 最坏情况是多做一次 discovery，不会丢数据 |

**适用场景**：
- 设备断电重启 → 新 BOOT → 立即重新开站 ✓
- 设备网络闪断 → 新 BOOT → 立即重新开站 ✓
- 设备永久离线 → 无新 BOOT → Reaper 兜底 ✓

### 方案 B: Discovery 与 Provisioning 解耦

**核心思路**: 将 parameter model discovery 从 provisioning task 中独立出来，作为一个独立的子系统。

```
当前:
  ProvisioningTask(discovering) → DiscoveryService → DataModel

解耦后:
  DiscoveryTask（独立生命周期）→ DataModel
  ProvisioningTask（仅在有 DataModel 时创建）
```

| 维度 | 评估 |
|------|------|
| 复杂度 | **高** — 新的 Task 类型、新的状态机、新的去重逻辑 |
| 收益 | Discovery 结果与 Provisioning 生命周期解耦 |
| 必要性 | **当前不高** — ParameterDiscoveryLog 已在追踪 discovery 状态 |
| 风险 | 两个 Task 类型的交互和竞态条件更复杂 |

**结论**: 过早抽象，当前阶段不推荐。

### 方案 C: Discovery 进度持久化 + 断点续传

**核心思路**: 将 Redis 中的 discovery 中间状态（已发现的参数）持久化到 DB，设备重连时从断点继续。

```
当前: 中间参数在 Redis → 设备离线 → Redis TTL 过期 → 丢失
增强: 中间参数同步写 DB → 设备重连 → 从 DB 恢复 → 只探索剩余层级
```

| 维度 | 评估 |
|------|------|
| 复杂度 | **很高** — 需要追踪哪些路径已探索、哪些未探索 |
| 收益 | 避免重复探索已发现的 80% 参数 |
| 必要性 | **低** — 完整 discovery 通常 3-5 分钟，重做成本可接受 |
| 风险 | 部分状态恢复可能导致数据不一致（设备固件可能已变更） |

**结论**: 工程代价远大于收益，不推荐。

---

## 5. 方案对比矩阵

| 维度 | 当前(Reaper) | +方案A(Reset) | 方案B(解耦) | 方案C(断点续传) |
|------|:---:|:---:|:---:|:---:|
| 代码复杂度 | ★☆☆ | ★★☆ | ★★★★ | ★★★★★ |
| 设备重连恢复速度 | 11min avg | **即时** | **即时** | **即时+续传** |
| 设备永久离线处理 | ✓ | ✓ | ✓ | ✓ |
| Discovery 结果保留 | ✗ | ✗ | ✓ | ✓ |
| Redis 状态清理 | ✗(靠TTL) | ✓ | ✓ | ✓ |
| 实现周期 | **已完成** | 1小时 | 1-2天 | 3-5天 |
| 维护成本 | 极低 | 低 | 中 | 高 |

---

## 6. 推荐路径

### 当前阶段（已实现）✅

**Reaper 机制**足够作为 MVP：
- 覆盖了最危险的场景（永久阻塞）
- 15 分钟的超时在基站运维场景下是完全可接受的
- 代码量极小，可审计，不引入新的复杂度

### 下一步（建议实施）🔜

**方案 A: Reset-on-BOOT** — 在 HandleBootstrap 中增加"重连即重置"逻辑：
- 消除 11 分钟平均等待，设备重连即刻恢复
- 与 Reaper 互补（一个处理重连，一个处理永久离线）
- 实现成本极低（~25 行代码）

### 暂不考虑 ❌

方案 B（解耦）和方案 C（断点续传）在当前阶段属于过度工程：
- 基站 discovery 耗时 3-5 分钟，重做成本低
- DataModel 创建后是**跨设备复用**的（同 OUI+ProductClass 的设备共享）
- 只有第一台同型号设备需要 discovery，后续设备直接命中缓存走 Path B

---

## 7. DataModel 复用机制（核心优势）

一个经常被忽略但非常重要的事实：**Discovery 的结果是跨设备共享的**。

DataModel 的唯一键是 **Carrier + Technology + OUI + ProductClass + FirmwareVersion**（五元组）。
同一型号同一固件版本的设备共享同一个 DataModel。

```
设备 A (OUI=001122, ProductClass=FAP-LTE, FW=V2.0.1)
  → 首次 BOOT → Path C (auto-discovery) → 创建 DataModel
  → 耗时 3-5 分钟探索参数树

设备 B (同型号同固件: OUI=001122, ProductClass=FAP-LTE, FW=V2.0.1)
  → 首次 BOOT → dmRegistry.ResolveForDevice() → 命中已有 DataModel
  → Path B (auto-sync) → 直接同步参数，无需重新 discovery

设备 C (同型号新固件: OUI=001122, ProductClass=FAP-LTE, FW=V2.0.2)
  → 首次 BOOT → ResolveWithFirmware 四级回退：
    Level 1: carrier+tech+oui+pc+fw=V2.0.2 → 未命中
    Level 2: carrier+tech+oui+pc (firmware IS NULL) → 未命中
    Level 3-4: oui/carrier_default → 未命中
  → Path C (auto-discovery) → 创建新的 FW=V2.0.2 DataModel
```

这意味着：
- Discovery 只需对**每种设备型号+固件版本组合**执行一次
- 固件升级后需要重新 discovery（因为参数树可能变化）
- 即使某次 discovery 中断需要重来，也只是这一台设备的一次性开销
- 一旦 DataModel 入库，后续同型号同固件设备全部走 Path B（秒级）
- **不存在"反复 discovery 浪费时间"的系统性问题**

---

## 8. 总结

| 结论 | 说明 |
|------|------|
| 当前 Reaper 是否足够？ | **是**，作为 MVP 完全够用 |
| 是否需要更复杂方案？ | **不需要** — Discovery 结果跨设备复用，重做成本低 |
| 推荐的增强？ | 方案 A (Reset-on-BOOT)，~25 行代码，消除重连等待 |
| 是否需要分离 Discovery Task？ | **不需要** — 过早抽象，当前 ParameterDiscoveryLog 已足够 |

**核心判断依据**：参数模型发现是一个**低频一次性操作**（每种设备型号+固件版本组合只做一次），而非高频重复操作。在这个前提下，简单的超时兜底 + 重连重置就是最优解。把精力放在更有业务价值的功能上。

---

## 9. 已修复的缓存失效 Bug

**问题**：`InvalidateCache` 使用 `localCacheKey(carrier, tech, oui, productClass)` 删除 L1 缓存，但 `ResolveWithFirmware` 使用 `localCacheKeyWithFirmware(carrier, tech, oui, productClass, firmwareVersion)` 写入 L1 缓存。Key 格式不匹配导致带 firmware 的缓存条目永远不会被主动失效，只能等进程重启。

**修复**：`InvalidateCache` 现在同时删除两种格式的 key。

```go
// 修复前
key := localCacheKey(dm.Carrier, dm.Technology, dm.OUI, dm.ProductClass)
r.localCache.Delete(key)

// 修复后
key := localCacheKey(dm.Carrier, dm.Technology, dm.OUI, dm.ProductClass)
r.localCache.Delete(key)
if dm.FirmwareVersion != "" {
    keyWithFW := localCacheKeyWithFirmware(dm.Carrier, dm.Technology, dm.OUI, dm.ProductClass, dm.FirmwareVersion)
    r.localCache.Delete(keyWithFW)
}
```
