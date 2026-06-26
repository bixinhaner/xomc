# 快速设置「刷新」无法结束 — 根因分析报告

- 报告日期：2026-06-26
- 现场环境：`http://172.24.224.78:8081`（Baicells 172.24.224.78，docker compose 编排栈）
- 复现设备：`120202662099BB01308`（device_id=`28847af7-6023-41f5-8261-bf591c0cdab2`）
- 触发路径：设备详情 → tab=quickSettings → 顶部「刷新」按钮
- 结论严重度：P1（功能性失败 + 用户长时间感知 5 min 卡顿）
- 修复状态：P0 已在本地分支落地（A.1 + A.2 + C + D）；P1/P2/P3 仍待独立推进与回归验收。

---

## 1. 现象

1. 用户在快速设置 tab 点「刷新」按钮，按钮进入 loading 状态。
2. 后端 4 个 GPV 任务在 ≤ 1 s 内全部执行完毕、CPE 已应答、result 写入 DB；但前端按钮持续 loading **直到 5 min 兜底超时**才解除。
3. 部分历史刷新呈现为 `last_param_sync_error="task expired"`，给人"设备超时"的错觉——但本次设备实际在线、所有 task `status=completed`，并非 expired。

## 2. 受影响范围

- 任意设备的 quickSettings「刷新」按钮（前端依赖统一字段终态信号）。
- 实测同环境 app 容器最近 15 分钟内累计 **3266 条** `command.get_parameters.response` 事件被 NATS 客户端 drop，问题处于活跃复发状态。

## 3. 调查时间线

| 时间 | 动作 | 关键证据 |
|---|---|---|
| t0 | 前端浏览器复现 + 抓 sync-status 响应 | `last_param_sync_at` 字段缺失，`last_param_sync_failed_at=2026-06-25` 是旧值 |
| t1 | 后端 `device_tasks` 直查（postgres） | 4 个 `sync-gpv-120202662099BB01308-{0,1,2,3}` 全 `status=completed`，sent→completed 在 100-300 ms 内 |
| t2 | 查 task 行 `result::text` | batch-0 raw_response 含 `ParameterValueStruct[15]`，并已写入 `standard_parameter_values`（含 HNBName="模拟站" 等真实值）→ ACS 解析完全成功 |
| t3 | 查 `devices.last_param_sync_at`（同一行 DB） | **NULL**；`last_param_sync_failed_at=2026-06-26 05:31:48`，`last_param_sync_error="task expired"`（早期残留） |
| t4 | 查 app 容器 12h 内日志 `received GPV response` / `path-b parameter values saved` | **0 条**——`provision/engine.go::handleGPVResponse` 从未被调到 |
| t5 | 查 app 容器 NATS 错误 | **4898 条 `nats: slow consumer, messages dropped on connection [770] for subscription on "command.get_parameters.response"`，最近 15 min 内 3266 条** |
| t6 | 查同窗口其他 subject 是否同样被 drop | 仅 `command.get_parameters.response` 单点过载；其他订阅未见 slow consumer |
| t7 | 旁证：app 容器同时段海量 `batch path InfoSyncer.SyncFromParameters failed (non-fatal): context deadline exceeded`（批量同步路径 [batch_processor.go:763](omcgo/internal/device/batch_processor.go) 在跑） | 后端 DB 处于过载，慢查导致 GPV handler 入库阻塞 |

## 4. 根因（Root Cause）

**链路证据指向单一根因**：

> ACS 应答事件 `command.get_parameters.response` 被 NATS 客户端的 push delivery 内部 pending buffer **丢弃**（slow consumer drop）；下游 `provision-gpv` queue subscriber 的 `handleGPVResponse` 因此**一次都没被调到**，所以 `HandleSyncResultPathB → finalizePathBSync → UpdateLastParamSyncAt` 整条链跑不到，`devices.last_param_sync_at` 永久保持 NULL。前端 `QuickSettingsSyncWatcher` 严格依赖该字段的时间戳跳变作为成功终态信号，等不到 → 只能依赖 `staleMonitorTimeoutMs=5min` 兜底清理。

### 4.1 端到端链路对照

```
[CPE]  ──TR-069 GPV Response (15 ParameterValueStruct) ──▶ [omcgo-acs-1]
                                                                │
                                                ① 解码 OK,翻译标准 path
                                                ② taskService.MarkTaskCompleted (status=completed,result 写入)
                                                ③ publishRPCResponseEvent → NATS subject "command.get_parameters.response"
                                                                │
                                                                ▼
[omcgo-nats-1 JetStream]  ← stream OK,服务端持久化
                                                                │
                                                ④ push deliver → omcgo-app-1 (nats.go client subscription pending buffer)
                                                                │
                                            ❌ slow consumer → 客户端 buffer 满 → 丢弃消息(累计 4898 条)
                                                                │
[omcgo-app-1 provision-gpv subscriber]
   handleGPVResponse          ← 从未被调用 (零条日志)
     ├─ HandleSyncResultPathB ← 未调
     │   ├─ finalizePathBSync
     │   │   ├─ deviceInfoRefresher.SyncFromParameters
     │   │   └─ paramSyncWriter.UpdateLastParamSyncAt(time.Now())  ← 这一步永远到不了
     │   └─ ...
     └─ HandleSyncResult
                                                                │
                                                                ▼
[postgres.devices]
   last_param_sync_at = NULL  ← 永不被更新
                                                                │
                                                                ▼
[frontend QuickSettingsSyncWatcher]
   isCurrentSyncSuccess() → false (依赖 last_param_sync_at 时间戳跳变)
   isCurrentSyncFailure() → false (依赖 last_param_sync_failed_at 时间戳跳变)
   ──▶ 永远命中不到终态 → 等待 staleMonitorTimeoutMs = 5 min 兜底超时
```

### 4.2 为何 `provision-gpv` 是 slow consumer

- `handleGPVResponse` 同步调用 path-b 翻译 + 写库 + finalize 时 deviceInfoRefresher 再次读写 DB。
- 同一 app 容器同期跑 `batch_processor` 对全网设备做 `InfoSyncer.SyncFromParameters` 批量写，DB 长时间高负载 → 单次 GPV handler 落库慢 → JetStream push 投递的 nats.go 客户端 pending buffer（默认 64 MB / 65 536 msg）撑爆 → 客户端 drop。
- JetStream 服务端无消息丢失（持久化的），会重投；但只要客户端持续慢就持续 drop，事实上整批 `command.get_parameters.response` 被吞掉。

### 4.3 旁证

| 旁证 | 解释 |
|---|---|
| 同样 device 早上 05:31 出现 `last_param_sync_failed_at + task expired` | 该次有 batch 真的 expired（另一条路径写 failed 字段），与本次 NATS drop 不冲突，是同根因（DB 过载导致 task 来不及发或 finalize 不及时）的不同表象。 |
| 12h 内 app 容器**仅 4 条** 该 SN 的非批量日志，全部是 `inform_handler:auto group-assign failed (heartbeat): get tree with counts: context deadline exceeded` | 继续印证 DB 过载是底层背景。 |
| ACS 容器 12h 内对该 SN 仅见上传报错（MR/PM minio），无 GPV 日志 | ACS 业务层只对该 SN 做了上传/CWMP 协议处理，未额外打印 task 完成 Info，符合现状代码（acs handler 只在 RPC 命中时 log "task completed"，default log level 不一定打出）。task DB 已 `completed` 充分说明 ACS 这段正确。 |

### 4.4 缺陷性质与复现条件

**这是代码缺陷，不是配置/部署/环境问题；属于"负载触发型必现"——低负载下潜伏看似正常，压力一来稳定复发。**

#### 4.4.1 涉及的 4 处独立代码缺陷（叠加放大）

| # | 缺陷位置 | 性质 | 风险 |
|---|---|---|---|
| ① | [omcgo/internal/core/event/nats_bus.go](omcgo/internal/core/event/nats_bus.go#L81) `QueueSubscribe` 未调 `sub.SetPendingLimits`，沿用 nats.go 默认（64 MB / 65 536 msg） | **设计缺陷** — 缺少抗背压配置 | 客户端 buffer 一满即 drop，且无指标/告警 → **静默丢消息**（4898 / 12 h 全无告警） |
| ② | [omcgo/internal/provision/engine.go](omcgo/internal/provision/engine.go#L1003) `handleGPVResponse` 在 NATS 回调内**同步**执行 path-b 翻译 + DB 写 + finalize（再读 DB） | **设计缺陷** — 慢消费天生体质 | DB 慢直接放大成 NATS backlog；handler 应是"解析 + 入有界队列"轻动作 |
| ③ | [omcgo/internal/provision/sync_pathb.go](omcgo/internal/provision/sync_pathb.go) `finalizePathBSync` 中 `deviceInfoRefresher.SyncFromParameters` 错误**短路 return**，导致 `UpdateLastParamSyncAt` 不被调 | **次要逻辑缺陷** — 终态语义耦合 | 本次被 ①② 遮蔽；一旦 ①② 修了就立刻显形 |
| ④ | 前端 [omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx](omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx) 把 `last_param_sync_at` 字段时间戳跳变作为**唯一**成功终态信号 | **设计缺陷** — 单点依赖 + 兜底 5 min 过长 | 后端任何一处不回写该字段，前端必卡 5 min；本身即脆弱设计 |

NATS 服务端、JetStream 持久化、docker compose 编排、PG/Redis 配置均无问题——根因纯在进程内代码路径的多重设计缺陷。

#### 4.4.2 复现矩阵：低负载偶现 / 压力下必现

| 触发条件 | 行为 |
|---|---|
| DB 低负载（dev 单机 / 早期商用 / 设备数 ≤ 数百）| `handleGPVResponse` 入库速率 ≥ 投递速率 → buffer 永远不满 → **看似永远正常** |
| 大批量后台同步并发（本现场 `batch_processor` 全网刷 1000+ 设备 `SyncFromParameters`）| 单次入库 100 ms ~ s 级 → 投递速率瞬间淹没消费速率 → **稳定丢消息 + 稳定卡 5 min** |
| 设备规模 10 万级 / 商用满负荷 | 即使无大批量 cron，常态 inform 流量也让 GPV 响应速率高于单 goroutine 同步 handler 上限 → **必现** |

#### 4.4.3 现场必现性证据

- 启动以来累计 **4898 条** GPV drop；**最近 15 min 增量 3266 条**（≈ 3.6 条/秒持续丢弃）→ 当前现场已是必现。
- 同时段 `batch_processor.go:763` `context deadline exceeded` 出现在大量不同 SN 上 → 压力源全局；**任何设备此时点「刷新」都必现**，并非本设备特有。
- DB 字段 `devices.last_param_sync_at = NULL` 表示**该字段引入以来该设备从未成功 finalize 过**，并非今日才坏。
- `last_param_sync_failed_at = 2026-06-26 05:31:48` 是同根因（DB 过载）走另一条路径写 failed 时留下的早期残留，问题已**持续至少 12 h+**。

#### 4.4.4 git 历史可追溯性

- `provision-gpv` 订阅注册写法（push consumer + 默认 pending limits + 同步 handler）自 T-0098 path-b 引入起即为现状，**从未加过抗背压**——属于「在低负载下被压制、负载上来即暴露」的**潜伏型缺陷**，不是回归。
- 不能通过"维持低负载"或重启 app 长期回避；必须把 ①②④ 修掉。重启虽能让 backlog 暂时清空、按钮一时恢复，但负载再起即复发。

#### 4.4.5 git 时间线 — 4 处缺陷的具体引入提交

> 通过 `git log -S` 字符串变化定位每处缺陷的首次出现版本。**不是一直存在的单一 bug，也不是某次回归——是 4 个不同时间点叠加形成的复合潜伏缺陷**，最后由 2026-06-24 的前端全局化提交首次把它"以 UI 卡顿"形式暴露给最终用户。

| 缺陷 | 引入提交 | 日期 | 提交说明 | 性质 |
|---|---|---|---|---|
| **① NATS QueueSubscribe 未配 `SetPendingLimits`** | `d45abcc1` | 2026-03-05 | `feat: 完成 Phase 1 基础建设全部开发（Sprint 1.1-1.5）` | JetStream event bus 首版即如此；`git log -S 'SetPendingLimits'` 全仓库**无任何匹配**——历史上从未配置过 |
| **② `provision-gpv` 订阅 + 同步 `handleGPVResponse`** | `18e0dd38` | 2026-03-23 | `feat(provision): 实现参数树自动发现与同步全链路` | 首次注册 push subscriber + 同步业务 handler；之后 `01ca538c`(5-9) / `fdaa3113`(5-29) 多次重构均未改其同步性质 |
| **③ `finalizePathBSync` 内 `SyncFromParameters` 错误短路 return** | `fdaa3113` | 2026-05-29 | `修复 NR/LTE 设备状态展示与参数同步一致性问题` | 首次定义 `finalizePathBSync` 函数即自带 `return fmt.Errorf("refresh device_info from parameters: %w", err)`；后续 `0fc4e6f5`(6-23) 大改 sync_pathb 也未修该 return |
| **④ 前端 `QuickSettingsSyncWatcher` 单点依赖 `last_param_sync_at`** | `0af257aa` | 2026-06-24 | `fix(acs,device,layout): … + quickSettings 同步全局化` | 把"等 `last_param_sync_at` 跳变 + 5 min staleMonitorTimeoutMs"机制提到 AppShell 全局层；**这是把 ①②③ 的潜伏后端缺陷首次以"按钮转圈无法结束"形式暴露给用户的最后一根稻草** |

#### 4.4.6 演化时序与暴露窗口

```
2026-03-05  d45abcc1  ① NATS 抗背压缺失 (出生)        ┐
2026-03-23  18e0dd38  ② 同步 GPV handler 注册 (出生)  │  ①+② 决定了"后端慢即丢消息"
2026-05-29  fdaa3113  ③ finalize 短路 return (出生)   │  ③ 让"即使 GPV 跑到也可能不写 last_param_sync_at"
                                                       ┤  此阶段：丢消息表现为"参数显示不更新",用户感知弱
2026-06-24  0af257aa  ④ Watcher 全局化 (出生)         ┘  ④ 把后端潜伏缺陷首次以"UI 卡 5min"形式呈现
2026-06-26  现场                                          ①②(④) 三者全部触发,3266 drop/15min,必现
```

**结论**：缺陷链 ①②③ 自 2026-03 ~ 2026-05 陆续埋下、潜伏 1-3 个月，业务侧仅以"参数偶尔显示不更新"轻度症状存在；2026-06-24 引入 ④ 后，用户首次看到"按钮卡 5 min"。也就是说，**用户感知是 2 天前才出现的，但根因从 3 个月前就在了**。修复时 ①② 必须修，③ 一并修（顺手），④ 同步加固兜底（不然修了后端，前端任何一处后续不写 `last_param_sync_at` 字段仍可能再卡 5 min）。

### 4.5 DB 吞吐健康度量化（现场实测，2026-06-26 11:30）

> 用户追问"DB 吞吐是否正常"。结论：**不正常，且是 NATS 慢消费的次级放大因子**——DB 现在被多条 SQL 反模式持续拖在 60-70% 烧 CPU/扫表状态，导致 GPV handler 入库慢，反过来撑爆 NATS pending buffer 触发 ①②。

#### 4.5.1 实测指标（5 s 窗口）

| 指标 | 实测值 | 健康基线 | 评估 |
|---|---|---|---|
| 主机 load (1/5/15 min) | 5.78 / 3.86 / 3.67 | < CPU 核数 50% | 32 核机 = 18%，**主机本身有余量** |
| 连接数 in_use / max_connections | 106 / 300 | < 50% | OK |
| xact_commit/s | 1 097 | 视业务 | 中等偏高（5005 设备 × inform 300s 应 ≈ 17 笔/s，**实际 65×**） |
| UPDATE rows/s | **6 037** | inform 频率 × 1-2 列 ≈ 50/s | **120×，严重异常** |
| rows_returned/s | **503 641** | < 5 万/s | **10×，严重异常**（每事务平均返回 459 行，正常应 1-100） |
| blks_read/s | 1 835（≈ 14 MB/s） | < 100 MB/s | OK |
| `track_io_timing` | **off** | 应 on（生产可观测必备） | **盲点** |
| `pg_stat_statements` | **未启用** | 应启用 | **盲点**——没有 top-N SQL 视图 |
| 长查询 > 30 s | **2 条**（`device_tasks` select 33 s；`autovacuum: ANALYZE device_tasks_p10` 23 s） | 0 | 应用层早 5-10 s 超时但 DB 仍在算 |
| 长查询 5-15 s | 3 条 `WITH cell_id ... WHERE parameter_path LIKE '%.CellCon%'` 并发 | 0 | **同一查询并发跑 3 份，全跨 32 分区扫表** |
| 行锁等 (`transactionid`) | 1 条（`INSERT ... ON CONFLICT` 7 s 等锁） | 0 | UPSERT 同行竞争 |

#### 4.5.2 三大 SQL 反模式（DB 烧灼源）

| # | 反模式 | 实测代价 | 来源 |
|---|---|---|---|
| **a** | `devices_cmcc` 单表分区（5 005 行）被 UPDATE **1300 万次**，dead_tup 占比 **77.4 %**，平均每次 seq_scan 读 4 999 行（≈ 全表） | 总返回 **310 亿行**；UPDATE/s 6037 占总 UPDATE 99% | `inform_handler` 每次心跳都更新 `last_inform_at` / `is_online` / `updated_at`；`batch_processor`、`HandleSyncResultPathB`、`UpdateLastParamSyncAt` 各自再独立 UPDATE 一次同一行 → HOT update 失败 → 索引膨胀 |
| **b** | `device_parameters` 分区表（32 分区，每分区 ≈ 10 万行）被 `parameter_path LIKE '%.CellCon%'` 通配前缀查询 | 单查询触发 32 个分区全表扫，每分区平均 seq_tup_read 2.7-3.2 万行 | `WITH cell_id ... LIKE '%.CellConfig.LTE.RAN.Common.CellIdentity'` 之类——前缀通配符让 PG 必须 seq scan，仪表盘/拓扑/devsweep 路径频繁触发 |
| **c** | `device_parameters` 高频 UPSERT，多 goroutine 抢同一 `(device_id, parameter_path)` 行 | 行锁等 7 s+；transactionid wait_event | `batch_processor` 全网并发 `SyncFromParameters` + `path-b` 单设备并发，无任何 device_sn-shard 串行化 |

#### 4.5.3 为什么 DB 是"次级放大因子"而不是根因

把 NATS 抗背压（①+②）修了，即使 DB 仍慢，handler 也只是把消息排队到本进程 channel，**不会再丢消息**，按钮就能 ≤ 5 s 给终态——这就是为什么本报告把 DB 列为 §6.5 独立工作流（P3）而不是 P0。

但**DB 必须独立立项治**，否则：

- 即便 NATS 不丢消息，单次 finalize 写库 1-3 s，前端体感仍差；
- `devices_cmcc` 索引膨胀 + WAL 写入压力 = autovacuum 长期失血，几天后会触发 wraparound 风险；
- 设备规模从 5 k 到 10 万时，**a/b/c 三个反模式都是 N² 级别放大**，必现拖垮整库。

#### 4.5.4 DB 优化目标（落到 §6.5 P3）

| 优化项 | 预期收益 |
|---|---|
| **去掉 `parameter_path LIKE '%.xxx%'` 通配前缀**：要么改成精确 `parameter_path = $1`，要么建 `pg_trgm` GIN 索引（仅在确需子串搜索时） | 消除每次几十亿行 seq scan |
| **`devices_cmcc` UPDATE 收敛**：每次 inform 同一事务内合并所有字段为 **1 个 UPDATE**（当前是 3-4 个独立 UPDATE）；对 `last_inform_at` 类纯时间戳字段考虑 ≥ 1 s 节流（去重 noop write） | UPDATE/s 从 6 000+ 降到 100 以内；dead_tup 不再爆涨 |
| **`device_parameters` UPSERT 按 device_sn 分片串行化**（在应用层 channel 分桶，§6.2 A 方案天然带来）| 行锁等待归零 |
| **打开 `track_io_timing = on`，启用 `pg_stat_statements`** | 获得 top-N SQL 视图（当前可观测性盲点）|
| **`batch_processor` 并发上限收敛 + 单事务规模封顶**（当前疑似无上限，凭 `context deadline exceeded` 自然反压）| 给 GPV handler 留出 DB 余量 |

## 5. 不是根因（排除项）

| 假设 | 排除理由 |
|---|---|
| 设备掉线/超时 | DB 内 4 个 task `status=completed`，sent→completed ≤ 0.4 s；当前 `stats.completed=199, queue_length=0, expired=65` 但本次 0 expired |
| CPE 返回空 ParameterList | batch-0 raw_response 实有 `ParameterValueStruct[15]`，含 HNBName/EARFCNDL 等真实值 |
| `paramSyncWriter` 在 DI 时为 nil | 即使为 nil 也应在 `finalizePathBSync` 内有 Warn 日志；前提是 handler 跑到，但日志中 `received GPV` 完全为 0 → handler 根本没跑 |
| `deviceInfoRefresher.SyncFromParameters` 内部错误吞掉了 `UpdateLastParamSyncAt` | 同上，handler 都没进，不存在"前半部分错误"的可能 |
| watcher 时间戳比较精度问题 | 后端字段始终 NULL，不是 ms 级误差问题 |

## 6. 修复方案

### 6.1 设计原则（澄清"不是多线程"）

修复目标不是"加并发"，而是**「接收侧轻量化 + 快进慢出解耦」**：

1. **发送侧（ACS publish）不动** — 本来就是异步非阻塞。
2. **接收侧 NATS 回调极简化** — 只做"解析 + 入有界 channel + 立即 Ack"，不阻塞 nats.go 客户端的 push delivery 循环。这是核心，**单线程也能解决丢消息问题**，只要 channel 够大、消费速率慢 ≠ 消息丢失。
3. **Worker pool 是放大手段，不是核心** — N 个 worker 让 DB 吞吐跟上常态投递速率；**必须按 `device_sn` 分片**保序（同设备 batch-0/1/2/3 不能乱序，否则 finalize 提前触发即 bug）。
4. **不要无脑加并发** — 同进程的 `batch_processor` 已经把 DB 压到 `context deadline exceeded`，加并发只会进一步压垮 DB。正确顺序：**先解耦（保不丢消息）→ 再 worker 化（提吞吐）→ DB 优化独立 issue 推**。

### 6.2 方案备选对比

| 方案 | 核心做法 | 抗背压 | 改动量 | 风险 | 推荐度 |
|---|---|---|---|---|---|
| **A. push + SetPendingLimits + worker pool** | 现有 push consumer 保留；nats.go 客户端调大 pending buffer + handler 改异步 + worker 按 device_sn 分片 | 中——extreme spike 仍可能撑爆 buffer | 小（nats_bus.go + engine.go） | 低 | ⭐⭐⭐⭐ 快速落地 |
| **B. JetStream Pull Consumer + MaxAckPending** | 客户端主动 `Fetch` 一批批拉；服务端按 `MaxAckPending` 限 in-flight | 高——**消费速率由消费者决定，物理上不会 drop**；服务端持久 backlog | 中（订阅模型重写，EventBus 接口补 `PullSubscribe`） | 中（要重写测试、考虑 AckWait/MaxDeliver） | ⭐⭐⭐⭐⭐ 治本 |
| **C. handler 极致瘦身（finalize 异步化为独立 reconciler）** | GPV handler 只做"参数翻译入库 + `UpdateLastParamSyncAt`"；`SyncFromParameters`/`device_info` 刷新挪到独立周期 reconciler（依赖 [PeriodicSyncer](omcgo/internal/provision/periodic_syncer.go)）| handler 落库快，间接缓解 backlog | 中（拆代码 + 新 reconciler）| 低 | ⭐⭐⭐⭐ 长期清洁，建议和 A/B 一起 |
| **D. 仅前端兜底** | watcher 加 `lastSyncGpv.lastCompletedAt > startedAt` 终态 + 60s 兜底 + "等待 OMC 入库" 文案 | 不解决后端丢消息，只是不再卡 5min | 小（三皮肤前端）| 低 | ⭐⭐⭐ **必做**但只是兜底 |
| **E. 给 GPV subject 单独 NATS 连接 + 独立 io goroutine** | 把热 subject 隔离到专属连接，避免被其他订阅拖慢 | 低——本质还是 push，慢消费照样丢 | 小 | 治标不治本 | ⭐⭐ 不建议 |
| **F. 合并 ACS 与 App 进程，绕过 NATS** | ACS handler 完成后直接同进程函数调 provision | 没 NATS 就没 NATS 丢消息 | 极大（违背 deployable units 设计）| 高——ACS-App 强耦合，丢失横向扩展能力 | ⭐ 不推荐 |
| **G. 增大 DB / 砍 `batch_processor` 并发**（治 DB 过载，不动 NATS）| 让 handler 不慢 → 自然不丢 | 间接 | 中（排 SQL 慢查、调连接池）| 中——治本但与本 issue 是独立工作（详见原 §6.B）| ⭐⭐⭐ 应做但属另一 issue |

### 6.3 推荐组合

**短期（同一 PR，止血 + 大部分治本）= A + C + D**

```
A 改 NATS client pending limits + handler 异步 + worker pool 分片
  ↓
C finalize 中 SyncFromParameters 错误改 warn-log 不 return；
  并准备把 deviceInfoRefresher 移到独立 reconciler
  ↓
D 前端 watcher 加 lastSyncGpv 终态 + 60s 兜底 + "等待 OMC 入库" 文案
```

- 落地最快，对现有 `EventBus` 抽象零破坏；
- 解决 99% 的丢消息（剩 1% 极端尖峰由 D 兜底，用户不感知）。

**中期（独立 PR，收口防御）= B**

```
B 把 provision-gpv 及其他高频热 subject 从 push 改 Pull Consumer
  + MaxAckPending=N、AckWait=30s、MaxDeliver=5
```

- 物理上消除 client-side drop 的可能；
- 与 worker pool 天然契合（pull 的 batch 直接喂给 worker）；
- 在 `EventBus` 接口加 `PullSubscribe`，现有 subscriber 不需要动，逐步迁移。

**独立工作流（不在本 PR 范围）= G**

- DB 过载是另一个系统性问题（详见 §6.5）；与本 NATS 丢消息问题正交，必须先把 NATS 这条路径堵住，再单独立项推进。

### 6.4 修复方案 → 缺陷提交映射

| 修复方案 | 主修哪次提交的问题 | 修复后效果 |
|---|---|---|
| A.1 nats.go SetPendingLimits | `d45abcc1`（2026-03-05）NATS event bus 首版无抗背压 | NATS client buffer 扩 10×，常态尖峰不再 drop |
| A.2 handler 异步化 + worker pool | `18e0dd38`（2026-03-23）`provision-gpv` 同步 handler 注册 | NATS 回调零阻塞，DB 慢不再倒灌到 NATS |
| C finalize 不 return | `fdaa3113`（2026-05-29）`finalizePathBSync` 短路 return | `UpdateLastParamSyncAt` 链路打通后必跑 |
| D 前端兜底 | `0af257aa`（2026-06-24）watcher 单点依赖 + 5min 兜底 | 即使后端某处不写字段也不再卡 5min |

**注意**：四处都是"补缺失逻辑"而非"撤销错改"，**不需要 revert 任何提交**，向前兼容。

### 6.5 不在本 PR 范围但需独立跟进（DB 治本工作，详证据见 §4.5）

> 现场实测：**UPDATE 6 037 行/s、rows_returned 50 万/s、`devices_cmcc` dead_tup 77.4%**，三大 SQL 反模式（§4.5.2）正在持续烧 DB。NATS 修好后体验立刻好，但 DB 不治会随设备规模 N² 放大。独立 issue 跟踪：

1. **`device_parameters` 通配前缀 LIKE 查询定位与改写**（§4.5.2-b）——`grep -rn "parameter_path LIKE '%" omcgo/internal/` 找全部使用点；要么改精确匹配，要么建 `pg_trgm` GIN 索引。
2. **`devices_cmcc` UPDATE 合并 + 节流**（§4.5.2-a）——每次 inform / batch sync / path-b finalize 在同一事务内合并为 1 个 UPDATE；`last_inform_at` 等纯时间戳字段加 ≥ 1 s 节流（避免 noop write 制造 dead tuple）。
3. **`device_parameters` UPSERT 按 device_sn 分片串行化**（§4.5.2-c）——天然由 §6.2 方案 A 的 worker pool device_sn-shard 带来，无需单独改。
4. **`batch_processor` 并发上限收敛 + 单事务规模封顶**——日志 `batch path InfoSyncer.SyncFromParameters failed (non-fatal): context deadline exceeded` 在数十设备/秒上同时出现，疑似无并发上限。
5. **可观测性补齐**：开 `track_io_timing = on`、启用 `pg_stat_statements` 扩展（当前都未启用，§4.5.1 列为盲点）；Grafana 加 `xact_commit/s`、`UPDATE rows/s`、`top-5 dead_tup_pct 表` 三个 panel。
6. **TimescaleDB chunk 大小与索引覆盖率审计**（如适用——本环境 `pg_extension` 未见 timescaledb 扩展，可能用纯 PG 分区，需在独立 issue 确认）。

### 6.6 不推荐方案及原因

- **E（单独连接隔离）**：只是把丢消息延后一点，本质还是 push + 默认 buffer，不彻底。
- **F（合并进程）**：跟项目「ACS 独立水平扩展」的部署原则冲突，是回头路。
- **单独做 G（不动 NATS）**：DB 过载是另一个系统性问题，等 DB 修好需要数周；同期 NATS 丢消息这条路径必须先堵住。

## 7. 建议交付节奏

| 阶段 | 内容 | 验收 |
|---|---|---|
| **P0 同一 PR**（短期，当前执行） | A.1 + A.2 + C + D | 1) 现场 15min 内 `nats: slow consumer` 计数 = 0；2) 点「刷新」按钮 ≤ 5 s 给出正确终态；3) `devices.last_param_sync_at` 在 GPV 完成后立即被写 |
| **P1 已完成** | Prometheus 指标 `nats_client_pending_drops_total{subject}` + 告警（metrics.go 已实现）| EventBusMetrics 投递结果分类计数；单测覆盖完整 |
| **P2 独立 PR**（中期） | B：高频热 subject 迁移 Pull Consumer | 极端尖峰下 client-side drop 物理不可能；服务端 backlog 受 MaxAckPending 控制 |
| **P3 独立 issue**（治本） | §6.5 DB 过载根因排查 | `batch_processor` 不再产生 `context deadline exceeded`；DB p99 latency 回到健康水位 |

> 执行状态：P0 代码改动已完成并进入评审/回归阶段；P1 已完成（Prometheus 指标实现）；P2/P3 仍待独立推进。

### 7.1 P0 一天落地清单（建议）

> 目标：在 1 个工作日内完成 A.1 + A.2 + C + D 的开发、联调与回归，恢复 quickSettings 刷新可用性。

| 时段 | 负责人建议 | 任务 | 交付物 | 完成标准 |
|---|---|---|---|---|
| 09:30-10:30 | 后端 A | A.1：在 NATS 订阅侧补 pending limits 配置（仅改 provision-gpv 路径） | 可配置项 + 默认值 + 代码注释 | 本地编译通过；配置生效可观测（日志或启动打印） |
| 10:30-12:00 | 后端 A | A.2：将 GPV 回调改为“轻回调 + 有界队列 + worker 消费”，并按 device_sn 分片保序 | 异步消费代码 + 队列满保护策略 | 压测或回放下不再阻塞回调线程；同设备批次顺序正确 |
| 13:30-14:30 | 后端 B | C：修复 finalizePathBSync 的错误短路，确保 SyncFromParameters 失败不阻断 UpdateLastParamSyncAt | finalize 逻辑修复 + 单测 | 即便 deviceInfoRefresher 失败，last_param_sync_at 仍写入 |
| 14:30-15:30 | 前端 A（含三皮肤） | D：Watcher 成功终态加 lastSyncGpv 完成时间兜底，超时从 5min 缩到 60s，并优化提示文案 | 三皮肤代码改动 | 三皮肤点击刷新均在 60s 内收敛，不再长时间转圈 |
| 15:30-16:30 | 前后端联合 | 联调现场设备 120202662099BB01308，连续执行 10 次刷新 | 联调记录 | 10 次刷新均在 ≤5s 进入成功或明确失败态 |
| 16:30-17:30 | QA/研发 | 回归与发布前检查（编译、测试、日志巡检） | 验收记录 + 风险说明 | 满足下方 P0 验收清单全部条目 |

#### P0 验收清单（当日必须）

1. 现场连续观察 15 分钟，app 日志中不再出现 command.get_parameters.response 的 nats slow consumer。
2. 设备详情 quickSettings 刷新按钮在 ≤5 秒内结束 loading（成功或失败均需有明确终态）。
3. GPV 完成后，devices.last_param_sync_at 可实时更新，不再长期为 NULL。
4. 三皮肤类型检查通过：webcode、webcode-v2、webcode-v3。
5. 变更说明中明确“本轮仅 P0，P1 保持独立阶段”。

#### 当日风险兜底

1. 若 A.2 在半天内无法稳定落地，先保留 A.1 + C + D 发 hotfix，A.2 当晚继续完成。
2. 若三皮肤联调时间不足，禁止仅发布单皮肤修复，至少保证三皮肤同等终态行为。
3. 若现场 DB 抖动导致个别刷新 >5 秒，允许短时放宽到 ≤10 秒，但需在发布说明中标注并挂 P3 跟进。

## 8. 涉及代码与现场资料索引

- 后端
  - [omcgo/internal/core/event/nats_bus.go](omcgo/internal/core/event/nats_bus.go) — `QueueSubscribe` 注册 JetStream push 订阅
  - [omcgo/internal/provision/engine.go](omcgo/internal/provision/engine.go#L304) — `provision-gpv` 订阅注册点
  - [omcgo/internal/provision/engine.go](omcgo/internal/provision/engine.go#L1003) — `handleGPVResponse`
  - [omcgo/internal/provision/sync_pathb.go](omcgo/internal/provision/sync_pathb.go#L132) — `HandleSyncResultPathB` / `shouldFinalizePathBSync` / `finalizePathBSync`
  - [omcgo/internal/task/pg_repository.go](omcgo/internal/task/pg_repository.go#L290) — `HasIncompleteSyncGPVTasksByDevice`
  - [omcgo/internal/device/batch_processor.go](omcgo/internal/device/batch_processor.go) — 旁证：全网批量同步压力源
- 前端
  - [omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx](omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx)
  - [omcmb/frontend-core/src/store/quickSettingsFeedbackStore.ts](omcmb/frontend-core/src/store/quickSettingsFeedbackStore.ts)
  - [omcmb/webcode/src/pages/device/DeviceDetail/index.tsx](omcmb/webcode/src/pages/device/DeviceDetail/index.tsx)
- 现场证据
  - DB：`devices` 行 `last_param_sync_at=NULL`，`last_param_sync_failed_at=2026-06-26 05:31:48 +08`；`device_tasks` 4 行 `status=completed`，source_id=`2a2330bd-ea65-4122-bed5-923f59dcbd5f`
  - 日志：app 容器最近 15 min `nats: slow consumer` × **3266**，全部命中 subject `command.get_parameters.response`；累计 4898
  - 业务对照：batch-0 result 含 `ParameterValueStruct[15]`，HNBName="模拟站"、EARFCNDL=55340、PhyCellID=117 等

---
报告人：GitHub Copilot Agent（在 wangyong 引导下完成现场取证）
