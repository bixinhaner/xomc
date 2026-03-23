# 参数发现优化方案

## 1. 问题分析

### 原始问题

当前 `DiscoveryService.HandleLevelGPNResponse` 对每个子对象都递归发起 GPN 请求，存在以下问题：

1. **多实例膨胀**：TR069 多实例对象（如 `FAPService.1.` ~ `FAPService.8.`）参数结构完全相同，但每个实例都被完整遍历
2. **跨会话重复**：discovery 跨多个 ACS 会话时，同一路径可能被重复入队
3. **空响应泄漏**：空 GPN 响应（parameter_count=0）被跳过，pending 计数永远无法归零
4. **级联 finalize**：finalize 后删除 Redis 键，后续迟到的 GPN 响应触发 Decr 返回 -1，再次进入 finalize
5. **竞态 force-finalize**：ACS 分发命令比 provision engine 通过 NATS 处理响应更快，队列暂时为空时误判为停滞

### 真实数据对比

| 指标 | 优化前 | 优化后 |
|------|--------|--------|
| GPN 请求数 | 4000+ 且持续增长 | 122 |
| 叶参数数 | 1769（含重复实例） | 948（去重后正确值） |
| 剩余 pending | 32 | 2 |
| 多实例跳过 | 0 | 84 |
| ACS 会话轮次 | 21 | ~5 |
| 发现耗时 | >2 分钟 | ~20 秒 |

## 2. 优化方案（共 5 项）

### 2.1 多实例过滤

> 遇到多实例对象时，只递归探索编号最小的实例，其余实例跳过。

**识别规则**：当同一父路径下存在多个以纯数字命名的子对象时，认定为多实例对象。

```
输入: ["Device.Services.FAPService.1.", "Device.Services.FAPService.2.", "Device.ManagementServer."]
输出: ["Device.Services.FAPService.1.", "Device.ManagementServer."], skipped=1
```

**实现**：`filterMultiInstanceObjects()` + `splitLastSegment()` 辅助函数。

### 2.2 Redis Set 去重

跨会话 GPN 命令去重，防止 discovery 跨多个 ACS 会话时同一路径被重复入队。

**实现**：`discoveryEnqueuedKey` Redis Set，入队前 `SAdd` 检查，已存在则跳过。

### 2.3 空 GPN 响应处理

移除 `engine.go` 中对 `parameter_count=0` 的 early return，确保空响应也能正确递减 pending 计数器。

### 2.4 迟到响应防护

在 `HandleLevelGPNResponse` 入口检查 `pendingKey` 是否存在。如果 discovery 已 finalize（键被删除），直接忽略迟到的 GPN 响应，避免级联 re-finalization。

### 2.5 延迟停滞检测

替换即时 force-finalize 为延迟检测机制：

1. 当 `remaining > 0 && queue == 0` 时，调用 `scheduleStallRecovery()`
2. 使用 Redis SETNX 互斥锁，确保每设备只有一个检测 goroutine
3. Goroutine 等待 15 秒后重新检查状态
4. 只有确认 `pending > 0 AND queue == 0` 持续 15 秒，才执行 force-finalize

**解决的竞态**：ACS 分发命令后，provision engine 通过 NATS 异步处理响应并入队新命令。在 NATS 处理完成前，队列暂时为空属于正常过渡状态，不应触发 force-finalize。

## 3. 其他改进

### 3.1 单次会话 RPC 限制

新增 `MaxRPCPerSession` 配置项，防止单次 ACS 会话发送过多 RPC 超出 CPE 承受能力。

- Session 结构体增加 `RPCCount` 字段
- `handleEmpty` 和 `handleRPCResponse` 两个分发路径都检查限制
- 达到限制时优雅结束会话，由 post-session wake 触发下一轮

### 3.2 速率限制调优

调整 per-device 速率限制从 10/min → 50/min，burst 从 5 → 50，避免 discovery 阶段频繁连接被限流。

### 3.3 restart-all.sh 增强

重启脚本增加 Redis 临时状态清理，避免残留的 discovery/session/cmdqueue 键影响新流程。

## 4. 影响范围

| 文件 | 修改内容 |
|------|---------|
| `internal/provision/discovery.go` | 多实例过滤、Redis 去重、迟到响应防护、延迟停滞检测 |
| `internal/provision/discovery_test.go` | 14 个单元测试（filterMultiInstanceObjects + splitLastSegment） |
| `internal/provision/engine.go` | 移除空 GPN 响应 early return |
| `internal/acs/handler.go` | 单次会话 RPC 限制检查 |
| `internal/acs/session.go` | 新增 RPCCount 字段 |
| `internal/acs/server.go` | 注入 maxRPCPerSession 配置 |
| `internal/core/appconfig/config.go` | 新增 MaxRPCPerSession 配置项 |
| `cmd/acs/etc/config.dev.yaml` | 速率限制和 RPC 限制配置 |

## 5. Redis 键设计

| 键 | 类型 | TTL | 用途 |
|----|------|-----|------|
| `provision:discovery:pending:{sn}` | String(int) | 1h | 待处理 GPN 计数器 |
| `provision:discovery:params:{sn}` | List(JSON) | 1h | 累积的叶参数 |
| `provision:discovery:enqueued:{sn}` | Set | 1h | 已入队的 GPN 路径去重 |
| `provision:discovery:stall_check:{sn}` | String | 30s | 停滞检测 goroutine 互斥锁 |

## 6. 实测结果（2026-03-23）

设备：1202000588233HB0039（48BF74 / FAP/BU1810 / FW: Vs11.23）

```
config: max_rpc_per_session=50, per_device_rate=50, burst=50

ACS sessions:      ~5 (productive: 3, wake-only: 2+)
GPN responses:     122
Leaf parameters:   948
Multi-instance:    84 instances skipped (20 groups)
Excluded paths:    1
Remaining pending: 2 (stall recovery after 15s)
Data model:        created and activated
```

948 为去重后的正确参数数量。之前 1769 参数包含了多实例的重复结构（84 个被跳过的实例路径各自下方有大量子参数）。
