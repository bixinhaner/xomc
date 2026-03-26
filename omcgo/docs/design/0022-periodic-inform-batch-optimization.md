# 0022 Periodic Inform 批量更新优化方案

## 1. 问题分析

### 1.1 当前架构

```
CPE Inform → ACS handler.handleInform → EventBus.Publish("device.inform.periodic")
                                                    ↓
                                        InformHandler.handlePeriodic (单协程串行处理)
                                                    ↓
                                        DeviceService.UpdateFromInform
                                                    ↓
                                    ┌───────────────┼───────────────┐
                                    ↓               ↓               ↓
                            deviceRepo.Update  paramRepo.BatchUpsert  Redis×3
                            (17列 UPDATE)       (10-50行 UPSERT)    (cache+heartbeat+stun)
```

### 1.2 瓶颈

| 维度 | 现状 | 问题 |
|------|------|------|
| 并发模型 | ChannelEventBus 单协程串行消费 | 10K 设备 / 60s = ~167 事件/s，单协程处理能力不足 |
| DB 写入 | 每个 Inform 触发 1次 device UPDATE + 1次 param BatchUpsert | ~167 UPDATE/s + ~2500 param UPSERT/s |
| DB 连接 | 串行执行，连接利用率低 | 无法利用 pgxpool 并发能力 |
| 事件积压 | channel buffer 满后阻塞 ACS 发布端 | 影响 ACS 会话处理延迟 |

### 1.3 单次 UpdateFromInform DB 操作明细

1. **deviceRepo.Update**: `UPDATE devices SET oui=$1, product_class=$2, ... WHERE id=$18`（17 列全量更新）
2. **paramRepo.BatchUpsert**: pgx.Batch 逐条 `INSERT ... ON CONFLICT DO UPDATE`（10-50 条参数）
3. **Redis**: `SET device:cache:{sn}`（缓存刷新）+ `SET acs:heartbeat:{sn}`（心跳）+ `SET stun:{sn}`（STUN 地址）

---

## 2. 优化方案

### 2.1 总体架构

```
EventBus
  ↓ (device.inform.periodic / device.inform.value_change)
InformHandler.handlePeriodic
  ↓
BatchInformProcessor (新增)
  ├── inputChan (带缓冲，接收 Inform 事件)
  ├── Worker[0] ─── buffer[0] ──→ 定时批量 flush → DB batch UPDATE + batch UPSERT
  ├── Worker[1] ─── buffer[1] ──→ 定时批量 flush → DB batch UPDATE + batch UPSERT
  ├── ...
  └── Worker[N-1] ─ buffer[N-1] → 定时批量 flush → DB batch UPDATE + batch UPSERT
```

### 2.2 核心组件

#### BatchInformProcessor

```go
// BatchInformProcessor 批量处理 Periodic Inform 的设备更新。
// 通过多协程并发 + 定时批量刷新，降低 DB 写入压力。
type BatchInformProcessor struct {
    workers       int                  // 工作协程数（配置项）
    flushInterval time.Duration        // 批量刷新间隔（如 10s）
    maxBatchSize  int                  // 单次批量上限（如 200）
    inputChan     chan *informUpdate    // 接收事件的缓冲通道
    deviceRepo    DeviceRepository     // 设备仓储
    paramRepo     DeviceParameterRepository // 参数仓储
    heartbeat     *HeartbeatMonitor    // 心跳监控
    cache         *DeviceCache         // 设备缓存
    stunUpdater   StunAddressUpdater   // STUN 地址同步
    logger        *zap.Logger
    metrics       *DeviceMetrics
    wg            sync.WaitGroup       // 等待所有 worker 退出
    stopCh        chan struct{}        // 停止信号
}

// informUpdate 封装单次 Inform 更新的所有数���。
type informUpdate struct {
    serialNumber string
    inform       *tr069.InformMessage
    receivedAt   time.Time
}
```

### 2.3 数据流

```
1. handlePeriodic 收到事件
   ↓
2. 解码 payload → 构建 informUpdate
   ↓
3. 发送到 inputChan（非阻塞，满则丢弃并计数）
   ↓
4. Worker 从 inputChan 取出事件
   - 按 deviceSN 做哈希分片：worker_id = hash(deviceSN) % N
     确保同一设备的更新始终由同一 worker 处理，避免并发写入冲突
   ↓
5. Worker 将 informUpdate 追加到内存 buffer（map[string]*informUpdate）
   - 同一 deviceSN 的后续更新覆盖前一条（保留最新）
   ↓
6. 定时 flush（每 flushInterval）或 buffer 达到 maxBatchSize 时触发
   ↓
7. 批量 DB 操作：
   a. 批量 device UPDATE（单条 SQL 多行更新）
   b. 批量 param UPSERT（合并所有设备参数到一个 pgx.Batch）
   c. 批量 Redis 操作（Pipeline）
```

### 2.4 分片策略

使用一致的 hash 分片将 deviceSN 映射到固定 worker，而不是 round-robin：

```go
func (p *BatchInformProcessor) dispatchToWorker(sn string) int {
    h := fnv.New32a()
    h.Write([]byte(sn))
    return int(h.Sum32()) % p.workers
}
```

**原因**：同一设备的连续 Inform 由同一 worker 处理，buffer 中可直接覆盖旧数据，无需跨 worker 去重。

### 2.5 批量 SQL 设计

#### 批量 Device UPDATE

使用 PostgreSQL `UPDATE ... FROM (VALUES ...)` 语法，单条 SQL 更新多行：

```sql
UPDATE devices AS d SET
    oui = v.oui,
    product_class = v.product_class,
    manufacturer = v.manufacturer,
    firmware_version = v.firmware_version,
    connection_request_url = v.connection_request_url,
    ip_address = v.ip_address,
    nat_detected = v.nat_detected,
    udp_connection_request_address = v.udp_connection_request_address,
    last_inform_at = v.last_inform_at,
    last_inform_events = v.last_inform_events,
    status = v.status,
    updated_at = NOW()
FROM (VALUES
    ($1::uuid, $2, $3, $4, $5, $6, $7, $8::boolean, $9, $10::timestamptz, $11::text[], $12),
    ($13::uuid, $14, $15, ...),
    ...
) AS v(id, oui, product_class, manufacturer, firmware_version,
       connection_request_url, ip_address, nat_detected,
       udp_connection_request_address, last_inform_at, last_inform_events, status)
WHERE d.id = v.id
```

**收益**：100 条设备更新从 100 次 round-trip 降为 1 次。

#### 批量 Param UPSERT

合并所有设备的参数到一个 pgx.Batch：

```go
batch := &pgx.Batch{}
for _, update := range updates {
    for _, param := range update.params {
        batch.Queue(upsertSQL, param.DeviceID, param.Path, param.Value, ...)
    }
}
br := pool.SendBatch(ctx, batch)
```

**收益**：将分散的 N 个 SendBatch 合并为 1 个，减少网络 round-trip。

### 2.6 Redis 批量操作

使用 Redis Pipeline 合并所有 Redis 操作：

```go
pipe := redis.Pipeline()
for _, update := range updates {
    // 设备缓存
    pipe.Set(ctx, "device:cache:"+update.sn, deviceJSON, cacheTTL)
    // 心跳刷新
    pipe.Set(ctx, "acs:heartbeat:"+update.sn, now, heartbeatTTL)
    // STUN 地址（如有）
    if update.udpAddr != "" {
        pipe.Set(ctx, stunKey, update.udpAddr, stunTTL)
    }
}
pipe.Exec(ctx)
```

**收益**：100 个设备的 ~300 次 Redis 命令合并为 1 次 Pipeline。

---

## 3. 配置项

```yaml
device:
  batch_processor:
    enabled: true          # 是否启用批量处理（false 则走原有逐条逻辑，方便回退）
    workers: 4             # 工作协程数（建议 = CPU 核数 / 2）
    flush_interval: 10s    # 批量刷新间隔
    max_batch_size: 200    # 单次批量上限（达到即��� flush，不等 interval）
    input_buffer: 10000    # 输入通道缓冲大小
```

---

## 4. handlePeriodic 改造

```go
func (h *InformHandler) handlePeriodic(ctx context.Context, evt event.Event) error {
    // 解码 payload（保持不变）
    var payload InformEventPayload
    if err := evt.Decode(&payload); err != nil {
        return fmt.Errorf("decode periodic event: %w", err)
    }

    inform := payloadToInform(&payload)
    deviceSN := inform.DeviceId.SerialNumber

    // 如果启用批量处理器
    if h.batchProcessor != nil {
        h.batchProcessor.Submit(&informUpdate{
            serialNumber: deviceSN,
            inform:       inform,
            receivedAt:   time.Now(),
        })
        return nil
    }

    // 回退：原有逐条处理逻辑
    // ... 现有代码 ...
}
```

---

## 5. Worker 生命周期

```go
func (p *BatchInformProcessor) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.runWorker(i)
    }
}

func (p *BatchInformProcessor) runWorker(id int) {
    defer p.wg.Done()

    buffer := make(map[string]*informUpdate) // deviceSN → latest update
    ticker := time.NewTicker(p.flushInterval)
    defer ticker.Stop()

    for {
        select {
        case update := <-p.workerChans[id]:
            // 覆盖旧数据，保留最新
            buffer[update.serialNumber] = update
            // 达到批量上限立即 flush
            if len(buffer) >= p.maxBatchSize {
                p.flush(id, buffer)
                buffer = make(map[string]*informUpdate)
            }
        case <-ticker.C:
            if len(buffer) > 0 {
                p.flush(id, buffer)
                buffer = make(map[string]*informUpdate)
            }
        case <-p.stopCh:
            // 优雅关闭：flush 残留数据
            if len(buffer) > 0 {
                p.flush(id, buffer)
            }
            return
        }
    }
}

func (p *BatchInformProcessor) Stop() {
    close(p.stopCh)
    p.wg.Wait()
}
```

---

## 6. Submit 分片分发

```go
func (p *BatchInformProcessor) Submit(update *informUpdate) {
    workerID := p.dispatchToWorker(update.serialNumber)
    select {
    case p.workerChans[workerID] <- update:
        // 成功入队
    default:
        // 通道满，丢弃并记录指标
        p.metrics.BatchDropped.Inc()
        p.logger.Warn("batch processor: worker channel full, dropping update",
            zap.String("device_sn", update.serialNumber),
            zap.Int("worker_id", workerID))
    }
}
```

---

## 7. flush 批量写入

```go
func (p *BatchInformProcessor) flush(workerID int, buffer map[string]*informUpdate) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    updates := make([]*informUpdate, 0, len(buffer))
    for _, u := range buffer {
        updates = u
    }

    // 1. 查找/确认设备存在（批量查缓存）
    // 2. 批量更新设备表
    if err := p.batchUpdateDevices(ctx, updates); err != nil {
        p.logger.Error("batch update devices", zap.Error(err), zap.Int("count", len(updates)))
    }

    // 3. 批量更新参数表
    if err := p.batchUpsertParams(ctx, updates); err != nil {
        p.logger.Error("batch upsert params", zap.Error(err), zap.Int("count", len(updates)))
    }

    // 4. 批量 Redis 操作（缓存 + 心跳 + STUN）
    p.batchRedisOps(ctx, updates)

    p.logger.Info("batch flush completed",
        zap.Int("worker_id", workerID),
        zap.Int("device_count", len(updates)))
    p.metrics.BatchFlushTotal.Inc()
    p.metrics.BatchFlushSize.Observe(float64(len(updates)))
}
```

---

## 8. 性能预估

### 10K 设备 / 60s Inform 周期

| 维度 | 优化前 | 优化后 (4 worker, 10s flush) |
|------|--------|------------------------------|
| DB device UPDATE | ~167 次/s | ~0.4 次/s（每 10s 一次批量，约 ~420 行/批） |
| DB param UPSERT | ~167 次 SendBatch/s | ~0.4 次 SendBatch/s（合并） |
| Redis 命令 | ~500 次/s | ~0.4 次 Pipeline/s |
| DB 连接占用 | 串行独占 | 4 并发，每次短暂 |
| 事件处理延迟 | ~6ms/event（串行阻塞） | ~0.1ms（写入 channel，异步处理） |
| 数据更新延迟 | 实时 | 最大 flushInterval（10s） |

### 100K 设备 / 60s Inform 周期

| 维度 | 优化后 (8 worker, 10s flush) |
|------|------------------------------|
| DB device UPDATE | ~0.8 次/s（每批 ~2000 行） |
| 单次批量 SQL 耗时 | ~50-100ms（2000 行 UPDATE FROM VALUES） |
| 总 DB 写入压力 | 降低约 99% |

---

## 9. 新增设备处理

对于 `handlePeriodic` 中的新设备自动注册逻辑（`device == nil` 时 `RegisterFromInform`）：

**方案**：注册仍走实时路径，不进入批量处理器。

```go
func (p *BatchInformProcessor) Submit(update *informUpdate) {
    // 先检查设备是否存在（走缓存，不命中 DB）
    device, err := p.cache.Get(ctx, update.serialNumber)
    if err != nil || device == nil {
        // 未知设备，走实时注册路径
        p.realTimeHandler(update)
        return
    }
    // 已知设备，进入批量队列
    workerID := p.dispatchToWorker(update.serialNumber)
    // ...
}
```

**理由**：新设备注册是低频操作（仅首次 Bootstrap/Boot），实时处理不增加显著 DB 压力；且注册需要立即写入 DB 以便后续 RPC 下发依赖设备存在。

---

## 10. 可观测性

### 新增 Prometheus 指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `device_batch_flush_total` | Counter | 批量 flush 总次数 |
| `device_batch_flush_size` | Histogram | 每次 flush 的设备数量 |
| `device_batch_flush_duration_seconds` | Histogram | flush 耗时 |
| `device_batch_dropped_total` | Counter | 因 channel 满而丢弃的事件数 |
| `device_batch_buffer_size` | Gauge | 当前各 worker buffer 中的设备数 |
| `device_batch_queue_size` | Gauge | 各 worker 输入 channel 中的待处理数 |

---

## 11. flushInterval 内数据丢失防护

### 11.1 丢失场景分析

| 场景 | 触发条件 | 影响范围 |
|------|----------|----------|
| **优雅关闭** (SIGTERM) | 正常部署/重启 | 无丢失 — Stop() 会 flush 残留 buffer |
| **异常崩溃** (SIGKILL/OOM) | 进程被强杀、内存溢出 | 丢失最近 ≤flushInterval 内未 flush 的更新 |
| **flush 失败** | DB 不可用、超时 | 该批次数据丢失 |
| **channel 满丢弃** | 突发流量超过 worker 处理能力 | 被丢弃的单条事件 |

### 11.2 为什么可接受

Periodic Inform 数据有**天然的自修复能力**：

1. **CPE 周期性重发**：每个设备按 InformInterval（通常 60s-300s）持续上报。即使本次丢失，下一次 Inform 会携带**完整的最新状态**（设备信息 + 参数列表），覆盖旧数据
2. **数据是全量快照而非增量**：`UpdateFromInform` 是全量更新 17 个字段 + 全量参数列表。丢失一次不会造成数据不一致，只是延迟一个 InformInterval 更新
3. **丢失窗口极小**：最坏情况丢失 flushInterval（10s）内的数据，而设备每 60-300s 上报一次，下一轮即恢复

### 11.3 防护措施

#### A. 优雅关闭保障（覆盖正常部署场景）

```go
func (p *BatchInformProcessor) Stop() {
    close(p.stopCh)

    // 等待所有 worker 退出（会 flush 残留 buffer）
    done := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        p.logger.Info("batch processor stopped gracefully")
    case <-time.After(p.shutdownTimeout): // 配置项，默认 30s
        p.logger.Warn("batch processor shutdown timed out, some data may be lost")
    }
}
```

K8s `terminationGracePeriodSeconds` 应大于 `shutdownTimeout`，确保 SIGTERM 后有足够时间 flush。

#### B. flush 失败重试（覆盖瞬时 DB 故障）

```go
func (p *BatchInformProcessor) flush(workerID int, buffer map[string]*informUpdate) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 最多重试 2 次，间隔 1s
    var err error
    for attempt := 0; attempt < 3; attempt++ {
        if attempt > 0 {
            time.Sleep(1 * time.Second)
            p.logger.Warn("batch flush retry",
                zap.Int("worker_id", workerID),
                zap.Int("attempt", attempt))
        }

        err = p.doFlush(ctx, workerID, buffer)
        if err == nil {
            return
        }
    }

    // 3 次失败，记录告警指标，数据将由下一次 Inform 自然恢复
    p.metrics.BatchFlushFailed.Inc()
    p.logger.Error("batch flush failed after retries, data will recover on next Inform cycle",
        zap.Int("worker_id", workerID),
        zap.Int("device_count", len(buffer)),
        zap.Error(err))
}
```

#### C. 心跳立即写入（覆盖离线判定准确性）

心跳（`acs:heartbeat:{sn}`）不能延迟写入，否则可能导致设备被误判离线。改为 **Submit 时立即写 Redis 心跳**，不等 flush：

```go
func (p *BatchInformProcessor) Submit(update *informUpdate) {
    // 心跳立即刷新（Redis 单次写入极快 ~0.1ms）
    // 即使后续 flush 失败或进程崩溃，心跳已更新，设备不会被误判离线
    if p.heartbeat != nil {
        p.heartbeat.RefreshHeartbeat(context.Background(),
            update.serialNumber, update.informInterval)
    }

    // 设备信息 + 参数走批量路径
    workerID := p.dispatchToWorker(update.serialNumber)
    select {
    case p.workerChans[workerID] <- update:
    default:
        p.metrics.BatchDropped.Inc()
        p.logger.Warn("batch processor: worker channel full, dropping update",
            zap.String("device_sn", update.serialNumber),
            zap.Int("worker_id", workerID))
    }
}
```

**理由**：HeartbeatMonitor 每 60s 检查一次，心跳 TTL = 2×InformInterval。如果心跳延迟 10s 写入，设备可能在 TTL 边界被误判为离线。心跳是 Redis 单次 SET，延迟极低（~0.1ms），不产生 DB 压力，无需批量化。

### 11.4 各场景丢失影响总结

| 场景 | 防护措施 | 丢失后果 | 恢复时间 |
|------|----------|----------|----------|
| 正常部署重启 | Stop() flush + shutdown timeout | 无丢失 | — |
| DB 瞬时故障 | 3 次重试 | 极少丢失 | 下次 Inform（60-300s） |
| 进程崩溃 (SIGKILL) | 无法防护 | 最多 flushInterval 内数据 | 下次 Inform（60-300s） |
| channel 满 | 丢弃计数 + 告警 | 单条事件 | 下次 Inform（60-300s） |
| 心跳准确性 | Submit 时立即写 Redis | 不受 flush 延迟影响 | — |

---

## 12. 风险与回退

| 风险 | 缓解措施 |
|------|----------|
| 数据延迟（最大 flushInterval） | 对于 Periodic Inform 场景可接受 10s 延迟；Bootstrap/Boot 仍走实时路径 |
| flush 失败丢数据 | 3 次重试；失败后下次 Inform 自然恢复（60-300s） |
| 进程崩溃丢数据 | Periodic Inform 全量快照特性保证下次自动恢复；心跳已实时写入不受影响 |
| 内存占用（buffer） | maxBatchSize 限制单 worker buffer 上限；4 worker × 200 = 最多 800 条 |
| 批量 SQL 超大 | maxBatchSize 控制单次 SQL 行数，避免超过 PostgreSQL 参数上限（65535） |
| 回退 | `batch_processor.enabled: false` 回退到原有逐条处理，无需代码变更 |

---

## 13. 实施步骤

1. **新增配置结构体** `BatchProcessorConfig` 到 `appconfig`
2. **实现 `BatchInformProcessor`** 到 `internal/device/batch_processor.go`
3. **实现批量 SQL 方法** 到 `internal/device/pg_repository.go`（`BatchUpdateDevices`）
4. **改造 `InformHandler`**：注入 `BatchInformProcessor`，`handlePeriodic` 分支到批量路径
5. **注册指标** 到 `DeviceMetrics`
6. **DI 组装** 在 `cmd/app/router/deps.go` 中初始化 `BatchInformProcessor`
7. **单元测试** + E2E 验证
8. **配置文档** 更新

---

## 14. 涉及文件

| 文件 | 变更 |
|------|------|
| `internal/core/appconfig/config.go` | 新增 `BatchProcessorConfig` |
| `internal/device/batch_processor.go` | **新增**：核心批量处理器 |
| `internal/device/inform_handler.go` | 改造 `handlePeriodic`，注入 BatchInformProcessor |
| `internal/device/service.go` | 无变更（BatchInformProcessor 直接使用 repo 层） |
| `internal/device/pg_repository.go` | 新增 `BatchUpdateDevices` 方法 |
| `internal/device/pg_param_repository.go` | 无变更（复用 BatchUpsert） |
| `internal/device/metrics.go` | 新增批量处理相关指标 |
| `cmd/app/router/deps.go` | 组装 BatchInformProcessor |
| `cmd/app/etc/config.*.yaml` | 新增 batch_processor 配置段 |
