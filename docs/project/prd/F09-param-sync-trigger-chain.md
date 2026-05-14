# F09 基站参数同步触发链补强（Umbrella PRD）

> Umbrella PRD — 覆盖 T-0123 / T-0124 / T-0125 / T-0126 / T-0127 五个子任务。
> 五任务共享同一份业务背景、验收口径、运营商差异、依赖与度量；技术实施细节按章节拆到下面"## 实施方案"§1-§9。
> 子任务通过 `PRD: docs/project/prd/F09-param-sync-trigger-chain.md#实施方案-§N` 锚点引用。

---

**PRD ID**：F09-param-sync-trigger-chain
**功能域**：F09 自动开站 + F02 配置参数模型（横跨 provision / device / config）
**作者**：Claude + user（基于 `~/Documents/notes/docs/基站参数同步触发链补强设计方案.md`）
**创建日期**：2026-05-14
**最后更新**：2026-05-14
**状态**：Approved（设计方案对话内已对齐七字段判决）
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`（GA 准备期）
**关联 Sprint**：`docs/project/sprint/sprint-11.md`（T-0123 stretch 候选；T-0124/T-0125/T-0126 后续 sprint）
**关联 Risk**：无新建 — 设计方案 §9 风险已被现有机制（Redis token bucket / PG advisory lock / feature flag）缓解，不构成 P0/P1

---

## 1. 业务背景（Why）

**一句话**：让 OMC 参数库与设备真实参数状态形成可闭环的一致性检测，避免库与设备无声漂移。

设计方案 `docs/design/参数-KPI-告警-整合设计方案.md` §1 已落地"基站首次入库"链路，但"已存在设备的持续一致性"目前依赖 CPE 主动上报变化（ActiveNotification + Value Change），存在以下漏点：

| 漏点 | 现状 | 影响 |
|------|------|------|
| 已存在设备"下线→上线"不触发参数刷新 | `HandleBootstrap` 仅被 `device.registered` 触发 | 设备离线期间被本地登陆/其他网管改过的参数差异，OMC 永远不会发现 |
| 无周期性参数同步 | 全部事件驱动 | 配置漂移检测能力为零 |
| 固件版本变化不重交集 | `UpdateFromInform` 简单赋值 firmware_version | 升级后 `discovered_param_mappings` 与新固件不匹配，Translator 降级到默认映射失精度 |
| 缺一键全量刷新入口 | `PullConfig` 必须显式列参数名 | 运维排障无法快速对账 |
| Value Change 链路缺真机验证 | 代码已通但未回放 | 协议层细节（路径前缀、`X_VENDOR_` 命名）需真机覆盖 |

不做的后果：商用网管面对"现场反馈说参数对不上"类工单只能逐参数 `PullConfig` 排查，效率低且漏点多；3-6 个月后参数库失参考价值。

现在做的契机：参数模型字典 + Path B 全量同步基础设施在 T-0098 后已落地，本方案只是把"触发器"补齐，复用既有同步能力。

---

## 2. 用户故事（Who / What）

> 1. As a **运维人员**，I want **设备从离线恢复在线时 OMC 自动拉一遍全量参数**，So that **设备离线期间被本地工具或别的网管改过的参数差异能被 OMC 主动发现并入库**。

> 2. As a **运营规划人员**，I want **每天有一次兜底的全量参数对账**，So that **即使 CPE 厂商出厂订阅列表缺失某个参数，OMC 也能在 24h 内追上设备真实值，规划判断不会基于过期数据**。

> 3. As a **运维人员**，I want **设备固件升级后 OMC 自动重新跑一次参数模型交集**，So that **新固件新增的参数路径能被 OMC 识别，Translator 不会因为映射表过旧而降级**。

> 4. As a **运维人员**，I want **在设备详情页有一个"立即同步参数"按钮**，So that **排查现场问题时不用手输参数名一个一个 PullConfig，秒级触发全量对账**。

> 5. As a **系统管理员**，I want **每次全量同步在日志里看到"哪些参数本次没上报"**，So that **能追溯设备废弃路径或固件升级后的字段裁剪，主动发现协议变更**。

---

## 3. 验收标准（Given/When/Then）

### AC-1（T-0123 — device.online 触发 Path B）

```
Given: 设备 D-001 状态为 active，CPE 离线超过 10 分钟被 OfflineDetector 标记 offline
When:  D-001 再次 Inform（event code 2 PERIODIC），UpdateFromInform 将状态从 offline 回升到 active，且 swVersion 未变化
Then:  - device.online 事件在 1 秒内发布到 EventBus
       - Provision Engine HandleDeviceOnline 收到事件
       - syncService.StartPathBSync(deviceID, opts{Reason: "device_online"}) 被调用一次
       - device_tasks 表新增一条 GetParameterValues 任务，状态 queued
       - Redis key provision:online_sync:D-001 TTL=60s 存在
       - 同一设备 60s 内第二次 offline→active 不会再次入队
```

### AC-2（T-0123 firmware 与 online 二选一兜底）

```
Given: 设备 D-002 状态 offline，FirmwareVersion="v1.0"
When:  D-002 Inform 上报 event code 2 PERIODIC，且 SoftwareVersion="v1.1"（与旧值不同）
Then:  - device.online 事件**不发布**（被 firmware 变化挡板拦截）
       - 由于 T-0125 尚未实现 firmware.changed 事件，本次同步不会触发（临时降级，T-0125 完成后由 firmware.changed 路径覆盖）
       - 日志输出 zap Info："firmware_changed_suppresses_online" device_id=D-002 old=v1.0 new=v1.1
```

### AC-3（T-0127 — Path B 完成时输出差异日志，与 T-0123 合并实现）

```
Given: 设备 D-003 device_parameters 表已有 500 条 standardPath
When:  device.online 触发 Path B 同步，GPV 翻译落地后准备 BatchUpsert 480 条 standardPath（缺失 20 条 = 设备本次未上报）
Then:  - BatchUpsert 完成后，zap Info 日志一条
       - 字段含：device_sn / device_id / reason="device_online" / prev_total=500 / current_total=480 / missing_count=20 / missing_paths_sample=[前 20 条]
       - device_parameters 表中缺失的 20 条 standardPath 行**保留不动**（不删除）
       - 当 current_total < 0.5 × prev_total 时，日志级别升 Warn
```

### AC-4（T-0127 跳过场景）

```
Given: 设备 D-004 是首次注册，device_parameters 中无任何 standardPath（集合 B 为空）
When:  Path B 完成
Then:  - 差异日志**不输出**（B 空 → diff 无意义，避免噪声）
```

### AC-5（idempotency / E2E）

```
Given: 设备 D-005 状态 offline
When:  cpe_simulator.py 模拟 D-005 上线 → Inform → 等 1 秒 → 再次 Inform → 等 1 秒 → 再次 Inform（共 3 次）
Then:  - 第一次：device.online 发布 + Path B 入队 + Redis token 占位
       - 第二、三次：不重复发布 device.online（status 已是 active），不重复入队
       - 60s 后 D-005 再次离线 → 上线：可以重新入队
```

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 备注 |
|------|------|------|------|------|
| Path B GPV 参数范围 | 由 `extractStorablePrefixes(MappingSet)` 自动按 discovered_param_mappings 推导 | 同左 | 同左 | **三家一致** — 已经通过 Carrier 接口在 MappingSet 层封装运营商差异，本任务不引入新差异点 |
| Inform 解析 swVersion 字段路径 | `Device.DeviceInfo.SoftwareVersion`（已通过 Translator 标准化） | 同左 | 同左 | 三家一致 |
| Offline 阈值（last_inform_at < now-X） | 10 分钟（既有 OfflineDetector 默认） | 同左 | 同左 | 三家一致；本任务不改 |
| 周期同步 interval 默认 | 24h（可配） | 同左 | 同左 | 三家一致；运营商可在 PeriodicSyncConfig 自行覆盖 |
| device.online 事件载荷字段 | deviceID/productID/swVersion | 同左 | 同左 | 三家一致 |

**关键约束**：所有运营商差异通过 `internal/carrier/*` + `MappingSet`（discovered_param_mappings 表的 carrier 列）适配，本方案**不引入**新的 `if carrier == "..."` 判断。

---

## 5. 非目标（Non-Goals）

- ✂️ **不做**：ActiveNotification 主动订阅下发（在 OMC 侧通过 SetParameterAttributes 给 CPE 配订阅列表）— 按用户决策，依赖 CPE 出厂订阅
- ✂️ **不做**：离线时长字段 `last_offline_at` / 上线时长统计 — devices 表暂不加字段，超出本方案范围
- ✂️ **不做**：旧固件 `discovered_param_mappings` 行的清理 — 设计上保留（车队混合版本时旧设备仍可用旧映射）
- ✂️ **不做**：旧 standardPath 在 `device_parameters` 的删除 — "设备没报就当不变"原则，避免误删历史/人工录入；漂移由 §5 差异日志记录追溯
- ✂️ **不做**：完整 missing 列表入日志正文 — 仅前 20 条 sample，全量由后续独立的"参数差异查询"运维端点导出（不在本方案）
- ✂️ **不做**：周期同步的 GPV 同步等待 — `StartPathBSync` 是入队动作，不阻塞等待 GPV 完成
- ✂️ **不做**：OfflineDetector 迁移到 `LeaderElector` 抽象 — post-RC，单列任务 T-G

---

## 6. 依赖

### 阻塞项（必须先解决）

- [x] T-0098 — 参数模型 ProductRegistry / ParamRegistry / Path B 全量同步链路（**已 done**，本方案的全部基础设施已就位）
- [x] OfflineDetector（既有，每 5 分钟扫表）— device.offline 事件已存在

### 被阻塞项（本功能不完成会影响什么）

- T-0124 周期同步：可独立实现，但实际触发的 Path B 链路需要本任务的差异日志（T-0127）才有完整可观测性
- T-0125 firmware 变化重新交集：依赖本任务（T-0123）的 UpdateFromInform 改造（共享 oldVersion / newVersion 比对逻辑）
- T-0126 手动同步端点：可独立实现，但完成时同样依赖 T-0127 的差异日志输出

### 外部依赖

- 无 — 全部代码改动在 omcgo + omcmb 现有模块内；Redis / PG 客户端已有依赖注入

---

## 7. 度量（如何证明上线成功）

| 指标 | 基线 | 目标 | 度量方式 |
|------|------|------|---------|
| **device.online 事件发布数** | 0/天（事件不存在）| ≥ 1 / 设备 / 天（按设备日均上下线频率）| Prometheus `event_bus_publish_total{subject="device.online"}` |
| **device_online 触发的 Path B 完成数** | 0/天 | ≈ device.online 发布数（理想 1:1）| Prometheus `provision_pathb_complete_total{reason="device_online"}` |
| **Path B 差异日志输出条数** | 0/天 | > 0（证明 T-0127 落地）| `grep "param_sync_missing" omcgo-app.log \| wc -l` |
| **Path B 差异日志 Warn 占比** | N/A | < 5%（绝大多数同步是健康的）| Loki / log aggregation 按 level 分组 |
| **Redis token bucket 命中数** | 0/天 | < 10%（不应该过度抑制）| Prometheus `provision_online_throttle_total` |

**反例监控**（上线后应**不**发生）：

- ❌ 不应出现 `device.online` 事件高频抖动（同设备每分钟 > 5 次）— 说明节流失效
- ❌ 不应出现 Path B 队列堆积（device_tasks 表 reason="device_online" queued > 1000 条）— 说明入队过快或 GPV worker 不够
- ❌ 不应出现 `panic` 或 GPV 失败率 > 5%

---

## 8. 实施要点（非规范性）

### 预计涉及模块

- `internal/core/event/subjects.go`（新增常量）
- `internal/device/device_service.go`（UpdateFromInform 改造，oldStatus + oldVersion 比对 + 事件发布）
- `internal/provision/engine.go`（订阅 + 新方法）
- `internal/provision/sync_pathb.go`（差异日志 — T-0127 合并实现）
- `omcgo/migrations/000NNN_devices_last_param_sync_at.sql`（T-0124 — 本任务不涉）
- `omcgo/internal/provision/periodic_syncer.go`（T-0124 — 本任务不涉）

### 预计新增端点

- `POST /api/v1/devices/:id/sync-params`（T-0126 — 本任务不涉）

### 预计新增迁移

- `000NNN_devices_last_param_sync_at.sql`（T-0124 — 本任务不涉）

### 预计工作量

- T-0123 + T-0127 合并：**S**（共享 UpdateFromInform 改造 + 共享 Path B 完成测试场景）
- T-0124：**L**（含 PG advisory lock + LeaderElector 接口设计）
- T-0125：**S**（共享 UpdateFromInform 改造）
- T-0126：**S**（后端端点 + 前端按钮）

---

## 9. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | Claude（user 委托）| 2026-05-14 | 七字段判决已对齐 |
| 架构师 | Claude | 2026-05-14 | 设计方案 §1-§9 已经过架构 + Go + 数据 + 电信领域 4 专家评议（详见原方案文档） |
| 领域专家 | Claude（F09 / F02 / TR-069 三视角）| 2026-05-14 | 三家运营商一致 |
| QA/发布经理 | Claude | 2026-05-14 | DoD 检查待 S5 阶段逐项打勾 |

---

## 10. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-05-14 | v1.0 | 初稿 — 设计方案完整拷入作为实施方案附录；T-0123/T-0124/T-0125/T-0126/T-0127 五任务合并 umbrella PRD | Claude + user |

---

---

## 实施方案（来源：`~/Documents/notes/docs/基站参数同步触发链补强设计方案.md` 完整拷贝）

> 以下 §1-§9 为原设计方案逐字拷入。任何技术细节以本节为准；上面 §1-§7 是 PRD 七要素抽象。

### 实施方案 §1. 设备"上线"事件 + 已有设备上线参数刷新（T-0123）

#### 1.1 现状盘点

- `devices.status` 已有 `offline` / `active` 状态枚举（migrations/000003）
- `OfflineDetector`（`internal/device/offline_detector.go`）每 5 分钟扫一次 `last_inform_at < now-10min` 的 active 设备 → 改为 offline → 发 `device.offline` 事件
- `inform_handler.go` 的 `handlePeriodic` / `handleRebootComplete` 在 Inform 到达时把 status 自动恢复为 active，**但不发"上线"事件**
- `HandleBootstrap` 只订阅 `device.registered`（首次注册才发布），不订阅"上线"

#### 1.2 新增机制

**(a) 新增事件主题**：`internal/core/event/subjects.go`
```go
SubjectDeviceOnline = "device.online"   // 已存在设备从 offline → active
```
（已有 `device.offline`，对称补 `device.online`，统一两者均走 `event/subjects.go` 常量）

**(b) 在 status 回升处发布事件（与固件变化事件二选一）**
- 位置：`internal/device/device_service.go` `UpdateFromInform` 内部
- 触发条件：`oldStatus == "offline" && newStatus == "active"`
- 事件载荷：`deviceID, productID, swVersion`（不带 `lastOfflineDuration`——`devices` 表没有 `last_offline_at` 字段，引入此字段不在本方案范围；如未来需要离线时长统计，单列任务）
- **与 §3 固件变化事件二选一**：同一次 Inform 同时满足"offline→active"和"swVersion 变化"时，**只发布 `SubjectDeviceFirmwareChanged`，不发布 `SubjectDeviceOnline`**——因为 §3 处理流程已包含全量 Path B 同步，能力覆盖 §1，避免两个订阅者各跑一次 Path B
- `active→active` 静默（不广播每次 Inform）
- **公共事件定位**：与已有 `device.offline` 对称，作为通用基础设施事件发布。当前订阅者仅 Provision Engine（触发 Path B），但保留事件抽象为未来 Dashboard 上线数 / 告警自愈 / Ops 审计预留扩展点（一次 EventBus 投递成本 µs 级，值得保留）

**(c) Provision Engine 订阅新事件**
- 位置：`internal/provision/engine.go` `Subscribe()` 内追加对 `SubjectDeviceOnline` 的订阅
- 新增 `HandleDeviceOnline(ctx, evt)`：**不重跑 FileType=11 上传与产品路由**（这些首次工作已完成），直接调 `syncService.StartPathBSync(ctx, deviceID, opts{Reason: "device_online"})`
- 不再需要在 `HandleDeviceOnline` 内比对 swVersion，因为固件变化路径已在 (b) 处二选一兜底

#### 1.3 关键文件

| 文件 | 改动 |
|------|------|
| `internal/core/event/subjects.go` | 新增 `SubjectDeviceOnline` 常量 |
| `internal/device/device_service.go` | `UpdateFromInform` 增 oldStatus 比对 + 事件发布 |
| `internal/provision/engine.go` | `Subscribe` 增订阅；新增 `HandleDeviceOnline` 方法 |
| `internal/device/inform_handler.go` | 确认状态转换路径走 `UpdateFromInform`（不需改） |

#### 1.4 幂等性

- Path B 同步本身幂等（GPV + UPSERT），重复触发只浪费一次 GPV 不破坏数据
- 同一设备短时间内多次 offline↔active 抖动：在 `HandleDeviceOnline` 里加 Redis token bucket，例如 `provision:online_sync:{deviceID}` TTL 60s，命中则跳过，防 ACS 抖动

---

### 实施方案 §2. 周期性参数同步（T-0124）

#### 2.1 设计目标

按时间间隔（默认 24h，可配）批量扫描在线设备，对每个设备入队一次 Path B GPV 同步，作为兜底的"配置漂移检测"。

#### 2.2 实现位置

新建 `internal/provision/periodic_syncer.go`，参照 `internal/device/offline_detector.go` 现成模式：

```go
type PeriodicSyncer struct {
    repo          DeviceRepository
    syncService   *SyncService
    interval      time.Duration   // 默认 24h
    batchSize     int             // 默认 200，单轮处理设备数上限
    maxConcurrent int             // 默认 10，并发同步数
    logger        *zap.Logger
}

func (p *PeriodicSyncer) Start(ctx context.Context) error {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done(): return nil
        case <-ticker.C: p.runOnce(ctx)
        }
    }
}
```

`runOnce`：
1. **leader 检查**：先尝试拿到 leader 锁（详见 §2.5），失败则本轮跳过，由其他副本执行
2. 查 `devices WHERE status='active' AND (last_param_sync_at IS NULL OR last_param_sync_at < now() - interval) ORDER BY last_param_sync_at NULLS FIRST LIMIT batchSize`
3. 用带计数信号量的并发池逐个调 `syncService.StartPathBSync(ctx, dev.ID, opts{Reason: "periodic"})`——注意 `StartPathBSync` 是**入队动作**（内部 `taskSvc.CreateTask`），不是同步等待 GPV 完成
4. 不在 `runOnce` 里回写 `last_param_sync_at`；该字段由 `HandleSyncResultPathB` 真正完成时回写（详见 §2.6）

**GPV 范围**：`StartPathBSync` 内部已通过 `extractStorablePrefixes(MappingSet)` 拉取本设备 `discovered_param_mappings`（discovered 优先 / default 兜底）中所有 `is_storable=true` 的前缀，**周期同步直接复用，不引入新的 GPV 范围决策**。运营商差异通过 Carrier 适配 `MappingSet` 自动覆盖。

#### 2.3 数据库变更

新增迁移 `omcgo/migrations/{下一可用版本号}_devices_last_param_sync_at.sql`：
```sql
-- +goose Up
ALTER TABLE devices ADD COLUMN last_param_sync_at TIMESTAMPTZ;
CREATE INDEX idx_devices_last_param_sync ON devices (last_param_sync_at NULLS FIRST) WHERE status='active';

-- +goose Down
DROP INDEX IF EXISTS idx_devices_last_param_sync;
ALTER TABLE devices DROP COLUMN IF EXISTS last_param_sync_at;
```

#### 2.4 配置

`internal/core/appconfig/config.go` 增：
```go
type PeriodicSyncConfig struct {
    Enabled       bool          // 默认 false（生产建议开，dev 关）
    Interval      time.Duration // 默认 24h
    BatchSize     int           // 默认 200
    MaxConcurrent int           // 默认 10
    StaggerWindow time.Duration // 默认 0；非零时把 batch 内任务在窗口内打散，避免雷霆万钧
}
```

#### 2.5 启动注册 + 多副本 leader 锁

`cmd/app/router/deps.go` + `cmd/app/main.go` 注入 PeriodicSyncer 并调用 `Start(ctx)`，类比现有 `OfflineDetector.Start()` 模式。挂在 app 进程（依赖 ParamRegistry / ProductRegistry，worker 进程访问不到）。

**多副本 leader 锁（RC 阶段强制）**：

- RC 部署 app 多副本，每副本各跑一份 ticker；必须保证**同一时刻只有一个副本执行 `runOnce`**，否则同一设备会被多个副本同时入队，浪费 GPV 配额且可能错乱 `last_param_sync_at`
- 实现：PG advisory lock（`SELECT pg_try_advisory_lock(hashtext('periodic_param_syncer'))`）—— 选 PG 优先于 Redis，因为：
  - PG 连接断开自动释放锁，副本异常退出无残留
  - 不依赖 TTL，简化故障切换
  - 已有 pgxpool，无需新组件
- 锁粒度：单一 leader，全程持有（不是每 tick 抢一次），避免抢占抖动
- **接口先行设计**：在 `internal/provision/periodic_syncer.go` 抽出 `LeaderElector` 接口（`TryAcquire(ctx) (bool, error)` / `Release(ctx)`），T-B 内提供 PG advisory lock 实现。后续 OfflineDetector 迁移到统一抽象作为**单列任务**（详见 §7 T-G post-RC）

**当前 `OfflineDetector` 暂不改造**：保持现状（每副本各跑一份，扫表结果幂等），范围控制在 T-B 内不外溢。

#### 2.6 `last_param_sync_at` 回写口径（关键）

**统一规则**：所有 Path B 同步完成时回写 `last_param_sync_at = now()`，**不区分触发源**（device_online / periodic / firmware_changed / manual）。

- 回写位置：`HandleSyncResultPathB` 收尾处（GPV 真正返回、BatchUpsert 完成之后），**不在 `StartPathBSync` 入队时回写**（避免任务还没跑完就被周期 tick 视为已同步而跳过）
- GPV 部分失败 / 超时 / 取消 时**不回写**，让下一轮周期或上线重试自动覆盖
- 手动同步（§4）回写后，下一轮周期 tick 会跳过该设备至少一个 interval，这是预期行为（不浪费 GPV）

#### 2.7 关键文件

| 文件 | 改动 |
|------|------|
| `omcgo/migrations/{下一可用版本号}_devices_last_param_sync_at.sql` | 新增字段 + 索引（版本号实施时确定，当前 main 已到 000091+） |
| `internal/provision/periodic_syncer.go` | **新增** |
| `internal/provision/sync_pathb.go` | `HandleSyncResultPathB` 收尾处回写 `last_param_sync_at`（适用全部 4 个触发源） |
| `internal/core/appconfig/config.go` | 增 `PeriodicSyncConfig` |
| `cmd/app/router/deps.go` + `cmd/app/main.go` | 注入 + Start |
| `cmd/app/etc/config.dev.yaml` | 增样例段（默认 Enabled=false） |

#### 2.8 与 internal/task 任务队列的衔接（关键）

§1 / §2 / §3 / §4 四个触发源最终都调 `syncService.StartPathBSync`，内部走的是 `taskSvc.CreateTask`（`internal/task` 统一任务队列，写 `device_tasks` 表 + Redis Sorted Set）。

- 周期同步 batch=200 设备 = **200 次 CreateTask 入队**，**不是** 200 个并发 GPV goroutine
- 实际 GPV 并发由 ACS 端的 worker pool / 全局准入控制 / per-device 限流统一管理（与现有 hotpath 共享）
- `PeriodicSyncer.MaxConcurrent` 控制的是 **CreateTask 入队的并发度**（防止瞬间往 PG 写 200 条任务记录），不是 GPV 并发度
- GPV 完成后 ACS 通过 task completion router 调回 `HandleSyncResultPathB`（已是现有路径，不需要改）
- CWMP ID ↔ Task 映射、reboot closer 等沿用现有机制

#### 2.9 防雪崩

10 万级设备 batch=200、interval=24h 即每天扫一轮。GPV 实际并发受现有 ACS 准入控制限制（不会瞬间打爆），单轮入队耗时秒级，GPV 完成串行展开持续数分钟到小时级。StaggerWindow 非零时把 CreateTask 入队动作打散到窗口内，避免同一时刻 PG 写入毛刺。生产灰度时建议先 Interval=72h、BatchSize=50 起。

---

### 实施方案 §3. 固件版本变化重新交集（T-0125）

#### 3.1 检测时机

`internal/device/device_service.go` `UpdateFromInform` 中：
```go
oldVersion := dev.FirmwareVersion
newVersion := findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
if oldVersion != "" && newVersion != "" && oldVersion != newVersion {
    // 发布事件
    bus.Publish(event.SubjectDeviceFirmwareChanged, FirmwareChangedEvent{
        DeviceID: dev.ID, ProductID: dev.ProductID,
        OldVersion: oldVersion, NewVersion: newVersion,
    })
}
```

#### 3.2 订阅与处理

- 新增 `SubjectDeviceFirmwareChanged = "device.firmware.changed"`
- `provision/engine.go` 订阅 → `HandleFirmwareChanged`，**入口加 Redis 串行锁**：`provision:firmware_handling:{deviceID}` TTL=10min（覆盖一次完整 Upload+交集+Path B 耗时上界），命中则跳过本次（让上一轮跑完）—— 防设备升级期间多次 Inform 携带不稳定 swVersion 引发并发交集
- 锁内串行执行：
  1. 调 `model_upload.go RequestModelUpload(deviceID)` 重新下发 FileType=11（产品 `enable_filetype11=true` 时）
  2. 上传完成后 `intersect.IntersectCPEModel` 自动写入 `discovered_param_mappings`，新 `(product_id, software_version, ...)` 为新主键，旧版本数据保留（不影响其他还在跑老固件的设备）
  3. 触发一次 Path B 全量同步（`Reason: "firmware_changed"`），完成时由 `HandleSyncResultPathB` 统一回写 `last_param_sync_at`（详见 §2.6）
  4. 锁在第 3 步入队后即释放（Path B 异步），不阻塞下一次合法 firmware 升级

> **与 §1 device_online 的互斥**：若同一次 Inform 同时满足"offline→active"与"swVersion 变化"，由 §1.2(b) 在事件发布侧二选一兜底——只发 `firmware.changed`，本节流程已覆盖参数同步能力。

#### 3.3 兼容性

- `enable_filetype11=false` 或 CPE 不支持 Upload 时降级：跳过交集，直接走默认 `param_mappings` 同步（保持现有降级逻辑不变）
- 旧固件的 `discovered_param_mappings` 行**不删除**：若车队设备版本分布混合，旧版本设备仍可用旧映射
- 设备升级后不再上报的旧 standardPath 在 `device_parameters` 中**保留不动**（"设备没报就当不变"）：避免误删人工录入或历史参考值；漂移情况由 §5 Path B 同步差异日志记录供运维追溯

#### 3.4 关键文件

| 文件 | 改动 |
|------|------|
| `internal/core/event/subjects.go` | 新增 `SubjectDeviceFirmwareChanged` |
| `internal/device/device_service.go` | `UpdateFromInform` 增版本比对与事件发布 |
| `internal/provision/engine.go` | 订阅 + `HandleFirmwareChanged` 方法 |
| `internal/provision/model_upload.go` | 确认 `RequestModelUpload` 可被重入调用（已有路径，验证即可） |

---

### 实施方案 §4. 手动"刷新所有参数"端点 + 前端按钮（T-0126）

#### 4.1 后端端点

新增 `POST /api/v1/devices/:id/sync-params` 到 `internal/device/handler.go`：
- 入参：`{ "force": true }` （可选；force=true 时绕过节流）
- 处理：直接调 `syncService.StartPathBSync(ctx, deviceID, opts{Reason: "manual"})`；完成时由 `HandleSyncResultPathB` 统一回写 `last_param_sync_at`（详见 §2.6），下一轮周期 tick 至少跳过一个 interval 不会重复触发
- 响应：`{ "taskID": "...", "status": "queued" }`
- 鉴权：操作员及以上角色

#### 4.2 前端

`omcmb/frontend-core/src/services/api/deviceApi.ts` 新增：
```ts
syncDeviceParams(deviceID: string, opts?: { force?: boolean })
```
对应 React Query Hook `useSyncDeviceParams` 放 `frontend-core/src/hooks/api/useDevices.ts`。

`omcmb/webcode/src/pages/device/DeviceDetail/`（已有详情页）参数 Tab 顶部加按钮"立即同步参数"，点击 → toast → 轮询 task 状态。

#### 4.3 关键文件

| 文件 | 改动 |
|------|------|
| `internal/device/handler.go` | 新增端点 |
| `cmd/app/router/router.go` | 注册路由 |
| `omcmb/frontend-core/src/services/api/deviceApi.ts` | 新增 API |
| `omcmb/frontend-core/src/hooks/api/useDevices.ts` | 新增 Hook |
| `omcmb/webcode/src/pages/device/DeviceDetail/...` | 新增按钮 |

---

### 实施方案 §5. Path B 同步差异日志（T-0127 — 与 T-0123 合并实现）

#### 5.1 设计目标

§1（上线触发）/ §2（周期）/ §3（固件变化）/ §4（手动）四个触发源最终都走 Path B 全量同步。在每次同步收尾处对比"DB 中已有的 standardPath 集合"与"本次 GPV 翻译落地的 standardPath 集合"，把**"之前上报过、本次未上报"**的差集写入应用日志，便于运维主动追溯参数漂移与设备废弃路径。**不改动数据库行**（设备没报的旧 standardPath 保持原值，遵循 §3.3"设备没报就当不变"原则）。

#### 5.2 触发范围

| 触发源 | 是否打差异日志 | 说明 |
|--------|---------------|-----|
| Path B 全量同步 | **是** | 所有 4 个触发源（device_online / periodic / firmware_changed / manual） |
| Path A 指定参数拉取（PullConfig 指定参数名）| 否 | 半量场景天然不完整，"缺失"无业务含义 |
| FileType=11 交集入库 | 否 | 写的是 `discovered_param_mappings`，不是 `device_parameters` |

#### 5.3 计算口径

- 集合 A：本次 GPV 结果经 Translator 翻译、过滤 `is_storable=false` 之后，准备写入 `device_parameters` 的 standardPath 集合
- 集合 B：本次同步开始前 `device_parameters` 中本设备的 standardPath 集合
- 差集 = B − A
- 实例化路径（含 `.{i}.`）按完整 standardPath 比对，不做模板归一化

#### 5.4 跳过场景

| 场景 | 处理 |
|------|------|
| 首次同步（集合 B 为空）| 跳过 — 新设备 / 重建数据，全量 missing 无意义，避免噪声 |
| 本次 GPV 部分失败、超时、被取消 | 跳过 — 缺失可能来自传输错误而非真实漂移 |
| 集合 A 显著小于集合 B（建议阈值：A < 0.5 × B）| 仍打，但级别提升到 Warn — 提示可能 GPV 不完整 |

#### 5.5 日志输出

- **通道**：应用 zap 结构化日志，共用现有日志管道，不另开 audit 文件
- **级别**：Info（常规）/ Warn（异常小子集）
- **关键字段**：`device_sn`、`device_id`、`reason`（`device_online` / `periodic` / `firmware_changed` / `manual`）、`prev_total`、`current_total`、`missing_count`、`missing_paths_sample`（前 20 条，超出截断）

完整 missing 列表不入日志正文（避免单行膨胀）；运维需要全量时由后续独立的"参数差异查询"运维端点导出（不在本方案范围）。

#### 5.6 性能影响

- 每次 Path B 多一次 `GetByDevice(deviceID)` 查询：单设备亚秒级
- 10 万设备周期同步（每天一轮、batch=200、并发=10）总开销可忽略
- 差集计算在内存（Set 操作），单设备路径数通常 < 5000，CPU 可忽略
- 不引入新表 / 新索引 / 新事件

#### 5.7 关键文件

| 文件 | 改动性质 |
|------|---------|
| `internal/provision/sync_pathb.go` | 在 `BatchUpsert` 前后插入差集计算与日志输出 |

#### 5.8 任务登记

T-0127 与 T-0123 合并实现（共享 Path B 测试场景）。

---

### 实施方案 §6. Value Change 真机测试

无代码改动。组织真机回归用例，覆盖：
1. CMCC/CTCC/CUCC 各一台真实 CPE（每运营商至少 1 款 productClass）
2. 在 CPE 端用厂商工具修改一个出厂订阅参数（如 Tx Power、邻区列表）
3. 抓 ACS 日志确认收到 `4 VALUE CHANGE` Inform
4. 查 `device_parameters` 表确认对应 standardPath 已 UPSERT 为新值
5. 验证 Translator 翻译正确（私有路径 → 标准路径）

测试结果归档：`docs/test/value-change-真机回归-YYYYMMDD.md`。

如发现某厂商 CPE 出厂订阅列表与预期差距大，登记为后续 Backlog（按用户决策，不在 OMC 侧补 SetParameterAttributes）。

---

### 实施方案 §7. 任务分解（Backlog 登记建议）

| 任务编号 | 描述 | 依赖 | 工作量估算 |
|---------|------|------|----------|
| T-0123 (T-A) | `SubjectDeviceOnline` 事件（含与 firmware.changed 二选一逻辑）+ Provision 订阅 + `HandleDeviceOnline` | 无 | S |
| T-0124 (T-B) | 周期同步：迁移 + PeriodicSyncer + **PG advisory lock leader 锁（LeaderElector 接口先行）**+ 配置 + 启动注册 | 无 | **L**（含 leader 接口设计） |
| T-0125 (T-C) | 固件版本变化检测 + Redis 串行锁 + 重新交集 + 同步 | T-0123 完成后接入更顺 | S |
| T-0126 (T-D) | `POST /devices/:id/sync-params` + 前端按钮 | 无（可与 T-0124 并行） | S |
| T-E | Value Change 真机测试 | 需真机环境就绪 | M（取决于设备协调） |
| T-0127 (T-F) | Path B 同步差异日志（跨 §1/§2/§3/§4 触发源共用） | 无（**与 T-0123 合并实现**） | S |
| **T-G** | OfflineDetector 迁移到 `LeaderElector` 抽象，统一 leader 选举（**post-RC**，依赖 T-0124 完成） | T-0124 | S |

---

### 实施方案 §8. 验证方案

#### 8.1 单元测试

- T-0123 + T-0127（合并）：
  - `TestUpdateFromInform_OfflineToActive_PublishesOnlineEvent`
  - `TestUpdateFromInform_FirmwareChangedSuppressesOnlineEvent`（二选一逻辑）
  - `TestHandleDeviceOnline_TriggersPathBSync`
  - `TestHandleDeviceOnline_RedisTokenBucketSkipsRepeat`
  - `TestPathBSync_LogsMissingPaths_WhenSubset`
  - `TestPathBSync_SkipsLogOnFirstSync`
  - `TestPathBSync_SkipsLogOnGPVPartialFailure`
  - `TestPathBSync_WarnLevelWhenSetTooSmall`
- T-0124：`TestPeriodicSyncer_*` / `TestPGLeaderElector_*`
- T-0125：`TestUpdateFromInform_FirmwareChanged_PublishesEvent` / `TestHandleFirmwareChanged_*`
- T-0126：handler 表测试

#### 8.2 集成 / E2E

补充到 `omcgo/scripts/e2e_verify.sh`：
- 用 `cpe_simulator.py` 模拟设备先 Inform 上线 → 等 11 分钟（或配置短间隔） → 静默 → 等 OfflineDetector 标记 offline → 再次 Inform → 校验 `device.online` 事件已发布且 Path B 任务入队
- 模拟 swVersion 从 v1 → v2 Inform → 校验产生新的 `discovered_param_mappings(software_version=v2, ...)` 行

#### 8.3 真机回归

只覆盖 §6 Value Change。

#### 8.4 灰度建议

- 周期同步 `Enabled=false` 默认值，先在 dev/test 环境跑两轮通后改为 true
- 生产首次开启时 Interval=72h、BatchSize=50，观察一周后再调到 24h / 200

---

### 实施方案 §9. 风险与回退

| 风险 | 缓解 |
|------|------|
| 周期同步给 ACS 带来突发压力 | 配置 `MaxConcurrent` + `StaggerWindow` 限流；监控 ACS QPS Prometheus 指标 |
| `device.online` 事件抖动放大 | Redis token bucket 节流（§1.4） |
| 固件升级时同一设备短时间触发多次 FileType=11 Upload | §3.2 Redis 串行锁 `provision:firmware_handling:{deviceID}` TTL=10min 兜底，叠加 `model_upload.go` 现有"同设备进行中跳过"机制 |
| 周期同步功能引入新表字段 | 迁移文件提供 down，可一键回退；Disabled 时与回退等效 |
| **多副本 leader 锁失效**（PG 连接异常 / advisory lock 未释放） | PG 连接断开自动释放是 PG 内核行为；监控 leader 持有时长指标，超阈值告警；最坏情况两个副本同跑一轮，因 `last_param_sync_at` UPSERT 与 task 队列幂等，**不会损坏数据**只浪费一次 GPV |
| **leader 切换瞬间窗口期** | PG advisory lock 释放后下一副本立即拿到，最长 1 个 ticker 间隔（默认 24h）无 leader——周期同步推迟一轮可接受 |

回退路径：把 `PeriodicSyncConfig.Enabled` 与 §1 / §3 的事件订阅注册做成 feature flag，出问题时关 flag 即可恢复到目前行为，不需要回滚迁移。

---

## 设计备忘（T-0123 + T-0127 合并实现，2026-05-14 S2 产出）

> 本节为 S2 阶段对设计方案 §1 + §5 的具体代码层落地备忘。**仅覆盖 T-0123 + T-0127 合并范围**；T-0124/T-0125/T-0126 留到各自 S2 阶段补充。

### 1. 现状勘察结论

| 检查点 | 文件 | 现状 |
|--------|------|------|
| `SubjectDeviceOnline` 常量 | `internal/core/event/subjects.go:6` | 不存在；`device.offline` 在 `offline_detector.go:133` 用字面量发布（**未走常量**）。本次新增 `SubjectDeviceOnline` 常量；顺手补 `SubjectDeviceOffline` 常量但**不**强迁旧代码（避免范围外溢）|
| `UpdateFromInform` 状态自动回升 | `device_service.go:530-650` | 已有 `oldStatus` 比对（`shouldActivate = device.Status != model.DeviceActive && device.Status != model.DeviceDecommissioned`，line 584），转 active 时记录 `oldStatus` 临时变量但**不发事件**。本次复用此处插入事件发布点 |
| `FirmwareVersion` 比对 | `device_service.go:556` | `device.FirmwareVersion = findParamValue(...)` 直接覆盖，**不**比对旧值。本次在 line 552-556 之间捕获 `oldVersion := device.FirmwareVersion` |
| `Subscribe()` 模式 | `provision/engine.go:136` | `bus.QueueSubscribe(subject, queueName, handler)` 三参形态；已订阅 `device.registered` / `datamodel.file.received` / `command.get_params.response`。本次追加 `device.online` 订阅 + queue `provision-online-sync` |
| `StartPathBSync` 签名 | `provision/sync_pathb.go:49` | `(ctx, dev, sourceID string) (bool, error)`；`sourceID` 当前作为 task 追溯（`pt.ID.String()`），**不能**直接覆盖塞 reason |
| `HandleSyncResultPathB` 是否能拿到 reason | `provision/sync_pathb.go:90` | **不能**——它是 GPV 响应异步回流点（`handleGPVResponse → HandleSyncResultPathB`），调用栈与 `StartPathBSync` 解耦 |
| `paramRepo` 拉取已有 standardPath 集合 | `device_param_repository.go` | 需验证有 `GetByDevice(ctx, deviceID)` 或类似方法——若无则需新加（少量代码） |

### 2. Reason 传递机制（关键决策）

设计方案 §5.5 要求差异日志含 `reason` 字段（device_online / periodic / firmware_changed / manual）。但 `HandleSyncResultPathB` 与 `StartPathBSync` 调用栈解耦，必须通过外部存储传递。

**选定方案**：Redis 临时映射

- Key：`provision:syncreason:{deviceID}` （含 deviceID 而非 deviceSN，因为 GPV 响应路径已有 device 对象）
- Value：`device_online` / `periodic` / `firmware_changed` / `manual` 字符串字面量
- TTL：10 分钟（覆盖 GPV 上界，超出按 `unknown` 降级）
- 写入位置：`StartPathBSync` 内、`enqueueGPVPrefixes` 调用前
- 读取位置：`HandleSyncResultPathB` 内、`BatchUpsert` 完成后
- 读取后**不删除**（让 TTL 自动过期，避免 Path B 部分失败下一次重试时 reason 丢失）

**为何不选方案 B（改 StartPathBSync 签名加 reason 参数）**：
- 4 个现有 caller（engine.go:363/608 + 未来 T-0124/T-0126）需要全部改
- 异步回流端拿不到，仍要外部存储；改签名只解决"发起端"问题
- 调用方为 reason 加 functional option 反而增加门槛
- Redis 已是 provision 模块依赖（节流用），不引入新组件

**为何不选方案 C（编码 reason 进 sourceID）**：
- 破坏 sourceID 作为 task_id 追溯的契约
- 解析方需要 split 字符串，脆弱

### 3. 改动点清单（按文件）

| 文件 | 改动内容 | 估算行数 |
|------|---------|---------|
| `internal/core/event/subjects.go` | 新增 `SubjectDeviceOnline = "device.online"` + `SubjectDeviceOffline = "device.offline"` 两常量（with doc comment 对称 SubjectDeviceConnectionLost）| +12 |
| `internal/device/device_service.go` | `UpdateFromInform` 捕获 `oldVersion` （在 line 556 之前）；现有 oldStatus 路径（line 590）扩展为：deviceRepo.Update 成功后判断 firmwareChanged / becameOnline 二选一 → 调 e.eventBus.Publish；新增 helper `publishDeviceOnlineEvent` | +60 |
| `internal/device/device_service.go` | 注入 `eventBus event.EventBus` 字段（构造函数 + DI）；现有结构体已有 logger/cache/etc，加一个字段成本低 | +5 |
| `internal/provision/engine.go` | `Subscribe()` 追加 `bus.QueueSubscribe(event.SubjectDeviceOnline, "provision-online-sync", handler)`；新增 `HandleDeviceOnline(ctx, evt DeviceOnlineEvent) error` 方法；`DeviceOnlineEvent` 结构体定义（含 DeviceID/ProductID/SwVersion）| +80 |
| `internal/provision/engine.go HandleDeviceOnline` | 内部：Redis token bucket（`SetNX` `provision:online_sync:{deviceID}` TTL=60s）→ deviceRepo.GetByID → syncService.StartPathBSync 调用（reason="device_online" 由 StartPathBSync 写 Redis 映射）| 含上 |
| `internal/provision/sync_pathb.go` | `StartPathBSync` 加 functional option `WithReason(string)`；内部入队前写 Redis `provision:syncreason:{deviceID}` TTL=10min；保持现有签名不变（4 个 caller 不动）| +30 |
| `internal/provision/sync_pathb.go HandleSyncResultPathB` | BatchUpsert 完成后调新增 helper `logPathBSyncDiff(ctx, dev, newPaths)`：读 Redis reason → 拉 device_parameters 现有 standardPath 集合 B → 比对 set A（已 upsert）→ 差集 B-A 写 zap 日志（Info/Warn）| +60 |
| `internal/device/device_param_repository.go` | 若 `GetByDevice` 方法不存在则新加（仅返回 standardPath 字符串列表，不需全字段）| 视情况 +20 |
| `internal/device/device_service_test.go` | 新增 4 testcase：OfflineToActive_PublishesOnlineEvent / FirmwareChangedSuppressesOnlineEvent / NoStatusChange_NoEvent / FirmwareChangedAndOnline_OnlySuppress | +120 |
| `internal/provision/engine_test.go` | 新增 testcase：HandleDeviceOnline_TriggersPathBSync / RedisTokenBucketSkipsRepeat / DeviceNotFound_NoOp | +80 |
| `internal/provision/sync_pathb_test.go` | 新增 testcase：LogsMissingPaths_WhenSubset / SkipsLogOnFirstSync / SkipsLogOnGPVPartialFailure / WarnLevelWhenSetTooSmall / ReadsReasonFromRedis | +120 |

**估算总改动**：~590 行（含测试），生产代码 ~265 行。**符合 S 工作量**。

### 4. Carrier 扩展点

无新增。本任务的所有运营商差异通过既有 `MappingSet`（discovered_param_mappings 表的 carrier 列）和 Carrier 接口 `extractStorablePrefixes` 自动覆盖。**不引入** `if carrier == "..."` 判断。

### 5. 新增观测埋点（metric / log key）

| 类型 | 名称 | 用途 | 输出文件 |
|------|------|------|---------|
| log key | `firmware_changed_suppresses_online` | device_service.go info 级 — 同 Inform 触发 firmware 变化挡板时记录 | `omcgo-app.log` |
| log key | `device.online published` | device_service.go info 级 — 成功发布 device.online 事件 | `omcgo-app.log` |
| log key | `device.online throttled by token bucket` | engine.go debug 级 — Redis token bucket 命中 | `omcgo-app.log` |
| log key | `path-b sync started` | sync_pathb.go info 级 — 已存在，保持 | `omcgo-app.log` |
| log key | `param_sync_missing` | sync_pathb.go info/warn 级 — **T-0127 差异日志**核心 key | `omcgo-app.log` |
| Prometheus | `event_bus_publish_total{subject="device.online"}` | 已有现成 metric，subject 维度自动覆盖；无新 metric | — |
| Prometheus | `provision_online_throttle_total` | 新增 counter — Redis token bucket 命中数 | metrics :9091 |

### 6. 待定点（< 3）

1. `paramRepo.GetByDevice` 是否已存在 — S3 实施时第一动作 `grep -n "func.*GetByDevice\|ListByDevice" omcgo/internal/device/device_param_repository.go`，无则新加
2. `device_service.go` 是否已注入 `event.EventBus` — S3 实施时 grep；若无需注入新依赖（通过 router/deps.go DI）
3. T-0127 差异日志的"reason=unknown 降级"行为是否需要 metric 计数 — 暂不加，可后续 follow-up

### 7. 出口门（S2）

- [x] 接口契约明确（DeviceOnlineEvent 结构 / Subscribe 模式 / HandleDeviceOnline 签名 / WithReason option / logPathBSyncDiff helper）
- [x] 迁移草案：**无**（本任务无 DB schema 变更，T-0124 才需要 last_param_sync_at 字段）
- [x] Carrier 差异点列出：无新增（既有 MappingSet 自动覆盖）
- [x] 观测埋点名字列出（5 log key + 1 新 metric `provision_online_throttle_total`）
- [x] 待定点 < 3（实际 3 个，均 S3 实施时立刻解决）

---

## 设计备忘（T-0125，2026-05-14 S2 产出）

> 接力 T-0123，覆盖设计方案 §3。**核心：把 T-0123 在 UpdateFromInform 末端的 firmware 挡板从 log-only 改造为真发 `SubjectDeviceFirmwareChanged` 事件 + Provision 引擎订阅触发"重新交集 + Path B"流程**。

### 1. 现状勘察

| 检查点 | 文件 | 现状 |
|--------|------|------|
| `SubjectDeviceFirmwareChanged` 常量 | `internal/core/event/subjects.go` | 不存在；本任务新增 |
| T-0123 firmware 挡板 | `device_service.go:670-686` | 当前仅 log "firmware_changed_suppresses_online"，**未发任何事件**；本任务替换为真发 event |
| `RequestModelUpload` 签名 | `model_upload.go:78` | `(ctx, dev, sourceID) (*ParameterDiscoveryLog, error)`；`product.EnableFileType11=false` 时返回 log.Status=DiscoveryCompleted + reason="skipped: enable_filetype11=false"；其余正常入队 Upload RPC |
| `handleDataModelFileReceived` auto-sync | `engine.go:701-721` | Upload 完成后 `e.config.AutoSync.Enabled` 时自动调 `StartPathBSync(ctx, dev, sourceID)`，**不带 WithReason** → 默认 reason 标签缺失 |
| Redis 串行锁基础设施 | T-0123 已注入 `e.redisClient` | 复用 |

### 2. Reason 传递机制（关键 — 复用 T-0123 Redis 协议）

设计方案 §3.2 流程：
1. HandleFirmwareChanged 调 RequestModelUpload（异步入队 Upload）
2. Upload 完成 → datamodel.file.received 事件 → handleDataModelFileReceived → IntersectCPEModel → auto-sync Path B
3. Path B 完成 → HandleSyncResultPathB 打差异日志（应携带 reason="firmware_changed"）

**问题**：handleDataModelFileReceived 内的 auto-sync 不知道当前是 firmware 触发还是首次 bootstrap 触发。

**方案**：HandleFirmwareChanged 入口先**预设 reason hint** —— `SET provision:syncreason:{deviceID} = "firmware_changed" TTL=10min`（与 T-0123 路径同 key）。`StartPathBSync` 内 `if pbOpts.reason != "" { SET key reason }` 仅在显式 `WithReason` 时覆盖；handleDataModelFileReceived 现行调用不带 opts → **不覆盖** → HandleSyncResultPathB 完成时读到正确的 firmware_changed。

### 3. HandleFirmwareChanged 控制流

```
1. Redis 串行锁：SetNX provision:firmware_handling:{deviceID} = "1" TTL=10min
   - 未拿到锁 → log debug "firmware handling already in progress, skip" → return（防设备升级期间不稳定 swVersion 多次 Inform 引发并发交集）
2. 预设 reason hint：SET provision:syncreason:{deviceID} = "firmware_changed" TTL=10min
3. e.deviceService.GetDevice(ctx, evt.DeviceID) — nil/error → log warn 跳过
4. 调 e.modelUploadService.RequestModelUpload(ctx, dev, sourceID="firmware_changed:UUID")
   - log.Status == DiscoveryDiscovering → Upload 真正入队，handleDataModelFileReceived 完成时会触发 Path B（reason 由 hint 决定）→ return
   - log.Status == DiscoveryCompleted (skipped: enable_filetype11=false) OR err != nil → 走 step 5 兜底
5. 兜底 Path B：调 e.syncService.StartPathBSync(ctx, dev, sourceID, WithReason("firmware_changed"))
   - 直接全量同步用 default 映射；旧 standardPath 设备没报就当不变（T-0127 差异日志兜底）
6. 锁不主动释放 — TTL=10min 自然过期防短时间内重复触发；Path B 完成后差异日志读 reason 仍可用（5min reason TTL 内）
```

**为何不强制锁内等 Upload 完成**：Upload→Intersect→Path B 是异步事件链，HandleFirmwareChanged 同步等待会阻塞 EventBus handler；锁的作用是**防并发触发**而非协调步骤，TTL 自然过期即可。

### 4. 改动点清单

| 文件 | 改动 | 估算行 |
|------|------|------|
| `internal/core/event/subjects.go` | 新增 `SubjectDeviceFirmwareChanged = "device.firmware.changed"` 常量 + doc comment | +6 |
| `internal/device/device_service.go` | 把 `firmware_changed_suppresses_online` log 块替换为：log "firmware_changed event published" → 调 publishDeviceFirmwareChangedEvent；新增 `DeviceFirmwareChangedEvent` 类型（DeviceID/SerialNumber/ProductClass/OldVersion/NewVersion 5 字段）+ `publishDeviceFirmwareChangedEvent` helper | +50 / -8 |
| `internal/provision/engine.go` | Subscribe 追加 `SubjectDeviceFirmwareChanged` 订阅；新增 `HandleFirmwareChanged(ctx, DeviceFirmwareChangedEvent)` 方法（含 §3 控制流） | +90 |
| `internal/device/service_test.go` | 新增 `TestUpdateFromInform_FirmwareChanged_PublishesEvent` 替换原 `TestUpdateFromInform_FirmwareChangedSuppressesOnlineEvent` 的"无事件"断言为"firmware.changed 事件"断言 | +50 |
| `internal/provision/engine_test.go` | 新增 4 testcase：HandleFirmwareChanged_RedisSerialLockSkipsConcurrent / HandleFirmwareChanged_RequestModelUploadEnqueued / HandleFirmwareChanged_FileType11Disabled_DirectPathBFallback / HandleFirmwareChanged_DeviceNotFound_NoOp | +120 |

**估算总改动**：~310 行，符合 S 工作量。

### 5. 观测埋点

| 类型 | 名称 | 文件 |
|------|------|------|
| log key | `firmware.changed event published` | device_service.go info |
| log key | `firmware handling already in progress, skip` | engine.go debug |
| log key | `firmware.changed: RequestModelUpload result` | engine.go info（含 log.Status / DiscoveryStatus） |
| log key | `firmware.changed: direct Path B fallback` | engine.go info（enable_filetype11=false 路径） |
| Redis key | `provision:firmware_handling:{deviceID}` TTL=10min | engine.go HandleFirmwareChanged |

### 6. 兼容性

- 旧 `discovered_param_mappings` 行不删除（车队混合版本兼容；设计方案 §3.3）
- 设备升级后不再上报的旧 standardPath 在 `device_parameters` 中保留不动；漂移由 T-0127 差异日志记录
- `enable_filetype11=false` 设备：跳过 Upload，直接走 default 映射 Path B 同步（不会"卡住"）

### 7. 出口门（S2）

- [x] 接口契约明确（DeviceFirmwareChangedEvent / Subscribe 追加 / HandleFirmwareChanged 5 步控制流）
- [x] 迁移草案：N/A
- [x] Carrier 差异点：无新增
- [x] 观测埋点名字列出（4 log key + 1 Redis key）
- [x] 待定点 < 3（实际 0 个 — T-0123 已铺好 Redis 注入 + sync 服务）


