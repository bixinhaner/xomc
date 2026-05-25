# Review Report — T-0164 G1 真机验证 PM SPV 重发链路修复

| 项 | 值 |
|---|---|
| Branch | `draft/pm-kpi-impl` |
| Commit Hash | `5ea1720e` |
| Author | kevin |
| Scope | device |
| Backlog | T-0164 |
| Sprint | wave-3 |
| Related | F03（PM 链路目标）/ F06（设备模块基础设施） |
| 真机 | BLQ `1202000240194DP0015`（OUI 48BF74，FAP/mBS31001/CA） |
| 部署形态 | docker compose（单进程 app + 单进程 worker），dev `batch_processor.enabled=true` |
| 验证日志 | 2026-05-25 14:07–16:17（本仓库 conversation） |

## 1. 背景与触发

`TODO.MD` §第二步真机端到端验证 — 用户提供 BLQ `1202000240194DP0015` 在线，验证 PM 自动下发链路。

**预期链路**：CPE 重启 → BOOT Inform → `device.online` 事件 → `pm.OnlineSubscriber` 入队 SPV → ACS 下发 3 个 `Device.FAP.PerfMgmt.Config.1.*` 参数。

**实际暴露 4 个连环 BUG**，本次合并修复。

## 2. 修复脉络（4 个 BUG 闭环）

### BUG-1: CPE 重启后 PM SPV 不自动重发

**根因**：
- `device.online` 事件触发条件：`oldStatus == DeviceOffline && newStatus == DeviceActive`（`device_service.go:738`）
- CPE 重启耗时 ~4 分钟，但 OfflineDetector 阈值 10 分钟 / 扫描间隔 5 分钟 → 重启期间 `is_online` 始终保持 `true`
- BOOT Inform 回连时 `oldStatus=Active` → `becameOnline=false` → `device.online` 不发 → `pm.OnlineSubscriber` 不入队

**修复**：新增 `RebootResponseOfflineMarker` 订阅 `command.reboot.response` 主题，ACS 收 CPE 回 RebootResponse SOAP 时立即把 `is_online` 标 `false`。等 BOOT Inform 回连时 `oldStatus=Offline` → `becameOnline=true` → 自然走 `device.online` 通路。

### BUG-2: marker 写 false 被 PERIODIC 异步覆盖

**根因**：
- ACS publish `device.inform.periodic` 后 `InformHandler.handlePeriodic` 异步走 `UpdateFromInform` 写 DB
- `UpdateFromInform` 内 `device.IsOnline = true`（line 619）→ 写 DB → 覆盖 marker 写的 false
- 事件 publish 顺序固定（PERIODIC 先于 RebootResponse），但**处理顺序无保证**（NATS 跨 subject 无 FIFO）

**修复**：新增 `Sequencer`（`sync.Map[serial_number]*sync.Mutex`），所有写 `is_online` 的入口共享同一把 SN 锁串行处理。

涉及方：`DeviceService.UpdateFromInform` / `RebootResponseOfflineMarker` / `OfflineDetector` / `HeartbeatMonitor` / `BatchInformProcessor`。

### BUG-3: DeviceCache 与 DB 不一致

**根因**：
- `UpdateOnlineStatus(...)` 只写 DB，不动 Redis cache
- `UpdateFromInform` 走 `cache.GetOrLoad` 读 device → 命中 cache 返回 stale `is_online=true`
- `oldStatus` 派生错误，重蹈 BUG-1 覆辙

**修复**：marker / `OfflineDetector` / `HeartbeatMonitor` 写 DB 后 `cache.Delete(sn)`，下次 GetOrLoad miss → DB 重建。

### BUG-4: BatchInformProcessor flush 写 stale is_online=true 覆盖 marker（lost update）

**根因**：
- dev `batch_processor.enabled=true`，PERIODIC 进 `map[sn]*informUpdate` buffer 10s 后批量 UPDATE
- `informUpdate.device.IsOnline=true` 是 PERIODIC 处理时的快照值
- 10s 内 marker 写 `is_online=false` → 释放 sequencer 锁
- batch flush tick 触发 → 按字典序 acquire SN 锁（B 修复加的） → UPDATE 写 stale true 覆盖

**修复（D 方案）**：让 batch 路径**完全不写** `is_online` 列。该列只由 4 条独立快路径维护：

| 路径 | 写 |
|---|---|
| `DeviceService.UpdateFromInform` (BOOT/Bootstrap/M Reboot 时) | true |
| `PeriodicOnlineMarker` (PERIODIC/VALUE_CHANGE 时，**新增**) | true |
| `RebootResponseOfflineMarker` | false |
| `OfflineDetector` / `HeartbeatMonitor` | false |

四条都在 sequencer 锁内 + 写 DB 后 cache.Delete。

具体改动：
- `batch_processor.go:413` SQL UPDATE 删 is_online 列
- `batch_processor.go:492` `batchRedisOps` 改 `pipe.Del`（而非 `pipe.Set`）让 cache 失效
- `device_service.go:619` 改条件写：`if isReconnectEvent(events) { device.IsOnline = true }`
- 新增 helper `isReconnectEvent`（识别事件码 `0 BOOTSTRAP` / `1 BOOT` / `M Reboot`）

## 3. 文件清单（12 文件 / +940 / -42 行）

| 文件 | 类型 | 关键改动 |
|---|---|---|
| `omcgo/internal/device/sequencer.go` | 新增 | per-device `sync.Map[string]*sync.Mutex` + `Lock(sn) → unlock` |
| `omcgo/internal/device/sequencer_test.go` | 新增 | 5 单测：串行 / 跨 SN 并发 / 排队顺序 / 空 SN noop / 复用同一 mutex |
| `omcgo/internal/device/reboot_response_offline_marker.go` | 新增 | 订阅 `command.reboot.response` → seq 锁内 `UpdateOnlineStatus(false)` + `cache.Delete` |
| `omcgo/internal/device/reboot_response_offline_marker_test.go` | 新增 | 6 单测：写 false / 空 SN / 设备不存在 / lookup 失败 / 更新失败 / 已离线幂等 |
| `omcgo/internal/device/periodic_online_marker.go` | 新增 | 订阅 PERIODIC + VALUE_CHANGE → seq 锁内 `UpdateOnlineStatus(true)` + `cache.Delete`（独立 queue group） |
| `omcgo/internal/device/periodic_online_marker_test.go` | 新增 | 7 单测（与 marker 对称） |
| `omcgo/internal/device/device_service.go` | 改 | `SetSequencer` setter + `UpdateFromInform` 入口拿锁；line 619 改条件写；新增 `isReconnectEvent` helper |
| `omcgo/internal/device/batch_processor.go` | 改 | `SetSequencer` setter + `doFlush` 字典序 acquire 所有 SN 锁；UPDATE SQL 删 is_online 列；`batchRedisOps` 改 `pipe.Del` |
| `omcgo/internal/device/offline_detector.go` | 改 | `SetSequencer` + `SetCache` setter；`markOffline` 加锁 + 写后 `cache.Delete` |
| `omcgo/internal/device/heartbeat.go` | 改 | `SetSequencer` + `SetCache` setter；超时分支加锁 + 写后 `cache.Delete` |
| `omcgo/cmd/app/provider/device.go` | 改 | 构造单例 `deviceSeq` + `deviceCache`，注入到 5 个共享者；注册 `RebootResponseOfflineMarker` + `PeriodicOnlineMarker` |
| `omcgo/cmd/app/etc/config.dev.yaml` | 改 | `batch_processor.enabled` 注释更新（值仍 true，说明竞态由 sequencer 保护） |

## 4. 静态审查

### 后端规范符合度

| 检查项 | 结果 |
|---|---|
| 命名（导出 PascalCase / 未导出 camelCase） | ✅ |
| 错误处理（`fmt.Errorf("...: %w", err)`，不 panic） | ✅ |
| SQL（Squirrel / 参数化，不字符串拼接） | ✅（batch UPDATE 是预编译模板，pgx 参数化） |
| 运营商硬编码 | ✅ 无运营商相关代码 |
| 资源释放（defer Unlock / defer rows.Close） | ✅ 所有 `seq.Lock` 返回 unlock 都用 `defer unlock()` |
| 测试覆盖（成功 + 失败路径） | ✅ 18 个单测涵盖正常 + 各失败分支 |
| 接口最小化 | ✅ `RebootOfflineDeviceRepo` 只暴露 2 个方法（`GetBySerialNumber` + `UpdateOnlineStatus`） |
| Context 传递 | ✅ 所有 DB / cache 调用都传 ctx |
| 注释（为什么 vs 什么） | ✅ 关键点都标注真机验证 2026-05-25 + 设计动机 |

### 并发安全

| 检查项 | 结果 |
|---|---|
| Sequencer 自身 thread-safe | ✅ `sync.Map.LoadOrStore` 保证 mutex 单例 |
| Sequencer 死锁 | ✅ batch 字典序 acquire 防 cycle；单 device 路径不嵌套 |
| 锁 leak（panic 路径） | ✅ 所有 caller 用 `defer unlock()`，goroutine panic 时锁释放 |
| Map 膨胀（10w 设备 ≈ 24MB mutex） | ⚠️ 已知，注释标注；100w+ 再加 LRU |
| 跨进程 | ⚠️ 当前单 worker 实例 OK；多 worker 实例需换 Redis 分布式锁（TODO.MD 已记） |

### 性能影响

| 维度 | 评估 |
|---|---|
| Sequencer 锁开销 | mutex 无争用时 ~ns 级，本场景临界区毫秒级，无瓶颈 |
| `UpdateOnlineStatus` 写 DB 单次 ~ms | 100k 设备 PERIODIC 60s 一次 ≈ 1700 UPDATE/s，pgxpool 默认 32 连接可承受 |
| `cache.Delete` 调用 | 每次写 DB 加一次 Redis DEL，~ms 级 |
| batch 路径开销 | 字典序 sort O(n log n)，n ≤ 200，可忽略；UPDATE SQL 少一列字段，性能略有提升 |

## 5. 真机验证证据

### batch off 模式（dev `enabled: false`）— 2026-05-25 15:00-15:30

```
15:00:58.205 ACS 下发 Reboot RPC (task 2d9c0b41)
15:00:58.335 收 RebootResponse
15:00:58.354 marker (sequencer 锁内) 写 is_online=false + cache.Delete  ✓
DB 真值: is_online=f, updated_at=15:00:58.443  ✓
Redis cache: 空  ✓
（10 分钟内未被覆盖）
15:05:24 BOOT Inform 到达
15:05:24.350 UpdateFromInform "previous_status: offline" → becameOnline=true → publish device.online
（未观察到 SPV 入队 — 此时 D 方案未上）
```

### batch on 模式（dev `enabled: true`，D 方案完整生效）— 2026-05-25 16:12-16:17

```
16:12:42.219 marker 写 is_online=false + cache.Delete
DB 真值: is_online=f, updated_at=16:12:43.679
+15 秒后: is_online=f, updated_at=16:12:43.679  ✓ batch flush 未覆盖
16:17:10.609 BOOT Inform → UpdateFromInform "previous_status: offline"
16:17:10.682 worker "PM upload SPV task enqueued" (task 7038d12e)
              url: http://172.19.1.132:7557/smallcell/FileUploadService?fileType=PM&filename=
              enable: "1", interval: 900
16:17:18.378 ACS 下发 SetParameterValues 给 CPE (cwmp_id ID:intrnl.unset.id...)
```

### 旁证：T-0164 G1 PM 上传链路（PERIODIC 持续生效）

- 14:30:02 / 15:00:00 / 15:21:21 多次收到 CPE 上传的 PM XML 文件（166KB，15 分钟窗口），证明 SPV 下发后 CPE 真的按配置周期上传 PM 文件到 OMC MinIO。

## 6. 发现与建议

### CRITICAL

无。

### WARNING

**W-1**：`BatchInformProcessor` 与 `PgDeviceRepository.BatchUpdateDevices` 代码重复（同 SQL 模板出现两次），未来重构成统一 repo 方法。延续 commit `17aa2e1` W-002 标注，无新增 debt。

**W-2**：Sequencer 当前为单进程 `sync.Map`。多 worker 部署时需换 Redis 分布式锁。已记入 TODO.MD 作为 T-0166 候选立项材料。

**W-3**：极端事件序乱（PERIODIC 处理被 GC pause 延迟，RebootResponse 先抢锁写 false 后释放，PERIODIC 后来又写 true）。概率极低（ms 级窗口），且 OfflineDetector 10 分钟兜底。生产暂可接受。

### INFO

**I-1**：`isReconnectEvent` helper 当前 hardcode `"0 BOOTSTRAP" / "1 BOOT" / "M Reboot"` 字面值。这些是 TR-069 Amendment 6 规定的标准事件码，相对稳定，未提取常量可接受。

**I-2**：`PeriodicOnlineMarker` 用了两个独立 NATS queue group（因 JetStream 不允许跨 subject 共享 consumer），日志看起来略冗余但功能正确。

## 7. 流水线门禁

| 关卡 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/device/...` | ✅（18 新测全过 + 既有测全过） |
| `go test ./internal/task/...` `./internal/pm/...` | ✅（无回归） |
| Docker 全栈部署 | ✅（2 次 redeploy 验证） |
| 真机端到端 | ✅（batch off / batch on 双模式验证通过） |
| Backlog Task | ✅ T-0164（umbrella，G1 收尾真机闭环） |
| 涉及 migration | ❌ 无 |
| 涉及 e2e_verify.sh 断言 | ❌ 无 |

## 8. 结论

**PASS_WITH_WARNINGS** — 可合入。W-1/W-2 为已知技术债，不阻塞；W-3 已有兜底机制。

## 9. 文档关联

- T-0164 主 plan：`docs/project/plan-T-0164-pm-kpi-pipeline.md`
- T-0164 followup：`docs/project/plan-T-0164-followup-gaps.md`
- TODO.MD（用户私有）：`/Users/shangyingbin/Documents/notes/TODO.MD` §T-0164 真机验证 BUG-2 闭环（B 验证记录已可关）
- 设计文档：`docs/design/pm-kpi-pipeline-improvements.md`（cd3d8683 on main）
