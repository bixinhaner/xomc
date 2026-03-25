# ACS EventCode 全链路性能分析与优化建议

> 分析 ACS 所有 EventCode 在不同设备规模下的数据库写入压力，评估是否需要批量更新机制，并提供优化建议。

---

## 1. 分析背景

### 1.1 系统链路

```
CPE → ACS (handleInform) → NATS EventBus → App/Worker Subscribers → PostgreSQL/Redis
```

ACS 本身**不直接写 PostgreSQL**，仅写 Redis（会话状态）。真正的 DB 写入压力来自下游 NATS 订阅者。

### 1.2 规模假设

| 规模 | 设备数 | 说明 |
|------|--------|------|
| 小型 | 10,000 | 当前开发目标 |
| 中型 | 50,000 | 近期生产目标 |
| 大型 | 100,000 | 设计容量 |

---

## 2. EventCode 逐项分析

### 2.1 `2 PERIODIC` — 周期心跳（高频，核心瓶颈）

**频率特征**：

| 参数 | 值 |
|------|-----|
| 典型 InformInterval | 60s - 300s |
| 运营商默认 | 移动 300s、电信 120s、联通 300s |
| 最坏场景 | 60s（部分设备配置） |

**QPS 计算**（按 InformInterval 分档）：

| 设备数 | 60s 间隔 | 120s 间隔 | 300s 间隔 |
|--------|---------|----------|----------|
| 10,000 | 167 QPS | 83 QPS | 33 QPS |
| 50,000 | 833 QPS | 417 QPS | 167 QPS |
| 100,000 | 1,667 QPS | 833 QPS | 333 QPS |

**每次 Periodic 触发的写操作**：

| 操作 | 目标 | SQL 类型 | 说明 |
|------|------|---------|------|
| ① GetBySerialNumber | Redis → PG | SELECT (cache hit) | Redis 命中率 >95% |
| ② deviceRepo.Update | PostgreSQL | UPDATE devices | 更新 LastInformAt, FirmwareVersion 等 |
| ③ cacheDevice | Redis | SET device:sn:{sn} | 写穿缓存 |
| ④ paramRepo.BatchUpsert | PostgreSQL | INSERT ON CONFLICT | 参数列表 10-50 行 |
| ⑤ heartbeat.RefreshHeartbeat | Redis | SET + EXPIRE | 心跳键 |

**关键代码路径**：
- `internal/device/inform_handler.go:122` → `handlePeriodic()`
- `internal/device/service.go:329` → `UpdateFromInform()`

**单次 Periodic 的 DB 写入量**：

| 操作 | 写入行数 | 耗时估算 |
|------|---------|---------|
| UPDATE devices | 1 行 | ~1ms |
| BatchUpsert device_parameters | 10-50 行 | ~3-10ms |
| **合计** | 11-51 行 | ~4-11ms |

**瓶颈分析（100K 设备，300s 间隔 = 333 QPS）**：

| 指标 | 值 | 评估 |
|------|-----|------|
| devices UPDATE QPS | 333 | ✅ PostgreSQL 轻松承受 |
| BatchUpsert QPS | 333 | ⚠️ 每次 10-50 行，总写入 3,330-16,650 rows/s |
| Redis SET QPS | 666 (cache + heartbeat) | ✅ Redis 轻松承受 |

**瓶颈结论**：`device_parameters` 的 BatchUpsert 是主要压力点。

---

### 2.2 `0 BOOTSTRAP` / `1 BOOT` — 设备注册（低频，重操作）

**频率特征**：

| 场景 | 频率 |
|------|------|
| 新设备入网 | 批量导入期：100-1000/天；稳态：0-10/天 |
| 设备重启 | 固件升级、断电恢复：0-100/天 |
| 批量重启 | 区域停电恢复：可瞬时 100-1000 台 |

**QPS 计算**：

| 场景 | 峰值 QPS |
|------|---------|
| 稳态 | < 1 |
| 批量入网 | 5-20 |
| 区域停电恢复 | 50-200（短时尖��） |

**每次 Bootstrap 触发的写操作**：

| 操作 | 目标 | SQL 类型 | 说明 |
|------|------|---------|------|
| ① GetBySerialNumber | Redis → PG | SELECT | 查重 |
| ② deviceRepo.Create | PostgreSQL | INSERT devices | 新设备（已存在走 Update） |
| ③ cacheDevice | Redis | SET | 写穿缓存 |
| ④ stunUpdater.SetFromInform | Redis | SET | STUN 地址 |
| ⑤ storeInformParameters | PostgreSQL | INSERT ON CONFLICT | 参数 10-50 行 |
| ⑥ heartbeat.RefreshHeartbeat | Redis | SET + EXPIRE | 心跳键 |
| ⑦ PublishDeviceRegistered | NATS | Publish | 触发 Provision Engine |

**关键代码路径**：
- `internal/device/inform_handler.go:69` → `handleBootstrap()`
- `internal/device/service.go:208` → `RegisterFromInform()`

**瓶颈结论**：低频操作，不需要批量优化。区域停电恢复时的短时尖峰由 NATS Queue Group 自动均衡。

---

### 2.3 `4 VALUE CHANGE` — 参数变更（中低频）

**频率特征**：

| 场景 | 频率 |
|------|------|
| 稳态 | 每设备 0-2 次/天（IP 变更、小区重配等） |
| 批量配置下发后 | 短时尖峰 |

**QPS 计算**：

| 设备数 | 稳态 QPS | 配置下发尖峰 |
|--------|---------|------------|
| 10,000 | < 1 | 10-50 |
| 100,000 | 1-5 | 50-200 |

**写操作**：与 `2 PERIODIC` 完全相同（复用 `handlePeriodic`）。

**瓶颈结论**：低频，无需优化。

---

### 2.4 `6 ALARM` — 告警事件（突发性高）

**频率特征**：

| 场景 | 频率 |
|------|------|
| 稳态 | 每设备 0-5 次/天 |
| 告警风暴 | 区域故障：1000-5000 条/分钟 |

**QPS 计算**：

| 场景 | QPS |
|------|-----|
| 稳态（100K 设备） | 5-50 |
| 告警风暴 | 100-500 |

**每次 Alarm 触发的写操作**：

| 操作 | 目标 | SQL 类型 | 说明 |
|------|------|---------|------|
| ① Redis 去重检查 | Redis | HGET alarm:active:{sn} | O(1) |
| ②a 新告警 | PostgreSQL | INSERT alarms_active | 1 行 |
| ②b 重复告警 | PostgreSQL | UPDATE alarms_active | 更新时间/计数 |
| ③ Redis 去重设置 | Redis | HSET alarm:active:{sn} | 设置 alarm_code → id |
| ④ Publish alarm.raised | NATS | Publish | → PushEngine |

**关键代码路径**：
- `internal/alarm/receiver.go:52` → `handleAlarmEvent()`
- `internal/alarm/engine.go:54` → `Process()`

**瓶颈分析**：
- Redis 去重有效减少 DB 写入（重复告警仅 UPDATE，不 INSERT）
- 告警风暴时 INSERT 量 = 唯一 alarm_code 数量 × 设备数，通常远小于总告警数
- `alarms_active` 表数据量有限（活跃告警），UPDATE 性能好

**瓶颈结论**：Redis 去重机制已有效控制。告警风暴时可考虑批量 INSERT，但当前优先级低。

---

### 2.5 `5 TRANSFER COMPLETE` — 文件传输完成（低频）

**频率特征**：固件升级完成时触发，频率极低。

| 场景 | 频率 |
|------|------|
| 稳态 | 0-5/天 |
| 批量升级期间 | 50-500/天 |

**写操作**：
- `upgrade_tasks` UPDATE（1 行/次）

**瓶颈结论**：无需优化。

---

### 2.6 `10 AUTONOMOUS TRANSFER COMPLETE` — 设备主动上传（中频）

**频率特征**：PM/MR 文件定时上传。

| 场景 | 频率 |
|------|------|
| PM 文件 | 每设备 4-6 次/天（每 4-6 小时） |
| MR 文件 | 每设备 1-4 次/天 |

**QPS 计算**：

| 设备数 | PM 上传 | MR 上传 | 合计 QPS |
|--------|--------|--------|---------|
| 10,000 | 0.5-0.7 | 0.1-0.5 | ~1 |
| 100,000 | 5-7 | 1-5 | ~10 |

**写操作链（TransferBridge → PM/MR Collector）**：

| 阶段 | 操作 | 目标 |
|------|------|------|
| Bridge | MinIO PUT | 对象存储 |
| Bridge | Publish pm/mr.file.received | NATS |
| PM Collector | INSERT pm_files | PostgreSQL |
| PM Collector | COPY pm_counters | TimescaleDB（1万-10万行/文件）|
| PM Collector | KPI 计算 INSERT | TimescaleDB |
| MR Collector | INSERT mr_files | PostgreSQL |
| MR Collector | COPY mr_records | TimescaleDB（100-1万行/文件）|

**关键代码路径**：
- `internal/transfer/bridge.go:83` → `handleAutonomousTransferComplete()`
- `internal/pm/collector/collector.go:71` → `handleFileReceived()`
- `internal/mr/collector/collector.go:76` → `handleFileReceived()`

**瓶颈分析**：
- COPY FROM 批量写入已是 PostgreSQL 最优策略
- PM 计数器数据量大但写入 TimescaleDB hypertable，自动分区
- 文件解析是 CPU 密集操作，由 Worker 进程隔离处理

**瓶颈结论**：已使用最优策略（COPY FROM + TimescaleDB）。Worker 水平扩展可解决吞吐瓶颈。

---

### 2.7 `7 CONNECTION REQUEST` — 连接请求（低频）

**频率**：极低，ACS 主动触发时的回调。无 DB 写入。

**瓶颈结论**：无需关注。

---

### 2.8 `8 M Reboot` — 重���完成（低频）

**频率**：仅设备重启后触发。当前无订阅者消费。

**瓶颈结论**：无需关注。

---

### 2.9 `command.*.response`（11 种）— RPC 响应（按需）

**频率**：取决于 ACS 下发的 RPC 命令量。

| 场景 | 频率 |
|------|------|
| 稳态 | 0-10/分钟 |
| 批量配置下发 | 100-1000/分钟 |
| 自动开站 | 50-200/分钟 |

**写操作（ProvisionEngine 消费）**：
- `provisioning_tasks` UPDATE
- `device_parameters` BatchUpsert（GPV 响应时）
- `parameter_discovery_logs` INSERT（GPN 响应时）

**瓶颈结论**：低频，无需批量优化。

---

## 3. 综合写入压力矩阵

### 3.1 按设备规模（稳态，InformInterval=300s）

| 写入目标 | 操作 | 10K 设备 | 50K 设备 | 100K 设备 |
|---------|------|---------|---------|----------|
| **devices** UPDATE | Periodic | 33 QPS | 167 QPS | 333 QPS |
| **device_parameters** BatchUpsert | Periodic | 33 QPS × 10-50 行 | 167 QPS × 10-50 行 | 333 QPS × 10-50 行 |
| **alarms_active** INSERT/UPDATE | Alarm | 1-10 QPS | 5-30 QPS | 10-50 QPS |
| **pm_counters** COPY | PM 文件 | ~1 QPS | ~5 QPS | ~10 QPS |
| **mr_records** COPY | MR 文件 | ~0.5 QPS | ~2 QPS | ~5 QPS |
| **Redis** SET/GET | 缓存+心跳 | 100 QPS | 500 QPS | 1000 QPS |

### 3.2 写入热度排序

```
🔴 高压: device_parameters BatchUpsert  — 333 QPS × 10-50 rows = 3,330-16,650 rows/s (100K)
🟡 中压: devices UPDATE                 — 333 QPS (100K)
🟡 中压: pm_counters COPY               — 10 QPS × 10K-100K rows/batch
🟢 低压: alarms_active INSERT/UPDATE    — 10-50 QPS (稳态)
🟢 低压: mr_records COPY                — 5 QPS × 100-10K rows/batch
🟢 低压: 其他                            — < 10 QPS
```

---

## 4. 优化建议

### 4.1 `device_parameters` BatchUpsert 优化（优先级：高）

**现状问题**：每次 Periodic Inform 立即执行 BatchUpsert（INSERT ON CONFLICT），100K 设备下 333 QPS × 10-50 行。

**方案 A：参数变更检测（推荐，短期）**

只有参数值真正发生变化时才写入，避免无变更的重复写入。

```go
// service.go — UpdateFromInform 中
func (s *DeviceService) storeInformParameters(ctx context.Context, deviceID uuid.UUID, params []tr069.ParameterValueStruct) {
    // 1. 从缓存/DB 获取当前参数值
    // 2. 比较差异，仅 Upsert 变更的参数
    // 3. 大多数 Periodic 参数不变化 → 写入量减少 80-95%
}
```

**预估收益**：写入量从 333 QPS 降至 15-60 QPS（仅变更参数）。

**方案 B：内存聚合 + 定时批量写入（中期）**

```
Periodic Inform → 内存 map[deviceID][]Param → 每 10s 批量 COPY INTO device_parameters
```

| 优点 | 缺点 |
|------|------|
| 写入 QPS 从 333 降至 ~1（每 10s 一次 COPY） | 参数更新延迟 0-10s |
| COPY 性能远优于逐行 INSERT ON CONFLICT | 内存占用增加 |
| 可合并同一设备多次写入 | 进程重启丢失未刷盘数据 |

**方案 C：异步队列写入（中期备选）**

```
Periodic Inform → Redis List (LPUSH) → Writer goroutine (BRPOP + 批量 COPY)
```

利用 Redis 作为缓冲区，Writer 固定频率批量消费。

**推荐路径**：先实施方案 A（投入小、收益大），100K 规模后评估是否需要方案 B。

---

### 4.2 `devices` UPDATE 优化（优先级：中）

**现状**：每次 Periodic 更新 `last_inform_at`、`firmware_version`、`connection_request_url` 等字段。

**方案 A：选择性更新（推荐，短期）**

仅当字段值变化时才 UPDATE。`last_inform_at` 可改为 Redis 存储（已有 heartbeat 键），不必每次写 PostgreSQL。

```go
// 仅在关键字段变化时 UPDATE
needsUpdate := device.FirmwareVersion != newFirmware ||
               device.ConnectionRequestURL != newURL ||
               device.Status != model.DeviceActive
if needsUpdate {
    deviceRepo.Update(ctx, device)
}
// last_inform_at 由 heartbeat Redis 键替代
```

**预估收益**：UPDATE QPS 从 333 降至 ~10-30（仅真正变化时写入）。

**方案 B：UpdateLastInform 轻量操作**

如果必须记录 `last_inform_at`，使用专用 SQL 仅更新时间字段：

```sql
UPDATE devices SET last_inform_at = $1, updated_at = $2 WHERE serial_number = $3
```

��免全行 UPDATE 的 WAL 日志开销。

---

### 4.3 Redis 操作优化（优先级：低）

**现状**：每次 Periodic 执行 3 次 Redis 操作（GET cache, SET cache, SET heartbeat）。

100K 设备下约 1000 QPS，Redis 轻松承受（单实例 100K+ QPS）。

**无需优化**，但可以通过 Pipeline 合并减少网络往返：

```go
pipe := redis.Pipeline()
pipe.Set(ctx, cacheKey, deviceJSON, 10*time.Minute)
pipe.Set(ctx, heartbeatKey, timestamp, heartbeatTTL)
pipe.Exec(ctx)
```

---

### 4.4 ACS 层面的保护机制（已有，无需额外优化）

| 机制 | 实现 | 效果 |
|------|------|------|
| 设备级限流 | `rate.Limiter` 10 req/min | 防止单设备 Inform 洪泛 |
| 全局准入控制 | `AdmissionController` 10K 并发 | 限制同时处理的 Inform 数 |
| NATS Queue Group | `device-mgr-periodic` | 多实例负载均衡 |
| Redis 设备缓存 | `DeviceCache` TTL 10min | 减少 PG 读取 |

---

## 5. 实施优先级

| 优先级 | 优化项 | 目标规模 | 预估收益 | 实施复杂度 |
|--------|--------|---------|---------|-----------|
| **P0** | device_parameters 变更检测 | 10K+ | 写入量减少 80-95% | 低 |
| **P1** | devices 选择性 UPDATE | 50K+ | UPDATE QPS 减少 90% | 低 |
| **P2** | Redis Pipeline 合并 | 50K+ | 网络往返减少 50% | 低 |
| **P3** | device_parameters 内存聚合批量写入 | 100K+ | 写入 QPS 从 333 降至 ~1 | 中 |
| **P4** | last_inform_at 迁移至 Redis | 100K+ | 消除最高频 UPDATE | 中 |

---

## 6. PostgreSQL 连接池配置建议

### 6.1 当前推荐（10K 设备）

```yaml
postgres:
  max_conns: 30        # CPU 核数 × 2 + 磁盘数
  min_conns: 10
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
```

### 6.2 100K 设备推荐

```yaml
postgres:
  max_conns: 60        # 需要更多并发写入连接
  min_conns: 20
  max_conn_lifetime: 1h
  max_conn_idle_time: 15m
```

### 6.3 关键索引检查

```sql
-- device_parameters 的 BatchUpsert 需要高效的 ON CONFLICT 索引
CREATE UNIQUE INDEX idx_device_params_device_path
  ON device_parameters (device_id, parameter_path);

-- devices 表的高频查询
CREATE UNIQUE INDEX idx_devices_serial_number
  ON devices (serial_number);

-- alarms_active 的去重查询
CREATE INDEX idx_alarms_active_device_code
  ON alarms_active (device_sn, alarm_code);
```

---

## 7. 监控指标建议

为了在生产环境及时发现瓶颈，建议添加以下 Prometheus 指标：

```go
// device_parameters 写入
device_param_upsert_total          // 总写入次数
device_param_upsert_rows_total     // 总写入行数
device_param_upsert_duration_seconds // 写入耗时直方图
device_param_upsert_skipped_total  // 变更检测跳过次数（方案 A 实施后）

// devices 更新
device_update_total                // UPDATE 总次数
device_update_skipped_total        // 选择性 UPDATE 跳过次数

// DB 连接池
pgx_pool_acquired_conns            // 当前占用连接数
pgx_pool_idle_conns                // 空闲连接数
pgx_pool_max_conns                 // 最大连接数
pgx_pool_acquire_duration_seconds  // 获取连接耗时
```

---

## 8. 结论

| 事件 | 频率类型 | 当前是否有瓶颈 | 是否需要批量机制 |
|------|---------|-------------|--------------|
| `2 PERIODIC` | **高频持续** | 100K 时有压力 | ✅ 需要（参数变更检测 + 选择性 UPDATE） |
| `0 BOOTSTRAP` / `1 BOOT` | 低频突发 | 无 | ❌ |
| `4 VALUE CHANGE` | 低频 | 无 | ❌ |
| `6 ALARM` | 低频/突发 | 风暴时有压力 | ❌ Redis 去重已足够 |
| `5 TRANSFER COMPLETE` | 极低频 | 无 | ❌ |
| `10 AUTONOMOUS TC` | 中频 | 无 | ❌ COPY FROM 已是最优 |
| `command.*.response` | 按需 | 无 | ❌ |

**核心结论**：只有 `2 PERIODIC` 需要优化，且优化重点在 `device_parameters` BatchUpsert 和 `devices` UPDATE 两个写入点。通过参数变更检测和选择性 UPDATE（P0 + P1），可将 100K 设备的 DB 写入压力降低 80-95%，无需引入复杂的批量聚合机制。
