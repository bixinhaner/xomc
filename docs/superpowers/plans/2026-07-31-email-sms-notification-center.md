# OMC 邮件短信统一通知中心实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建成以可靠告警生命周期事件为事实入口、完成邮件试点，并为契约确认后的短信
Adapter 提供统一规则、调度、投递和审计基础。

**Architecture:** 主 PostgreSQL 中的告警变更与 `alarm_event_outbox` 原子提交，由 Relay
发布到独立的 `DOMAIN_ALARM` JetStream；历史投影、北向和通知中心使用各自 durable
消费。通知中心以 Inbox、occurrence 顺序投影、持久化 schedule 和逐收件人 delivery
驱动邮件 Worker，并为后续 SMS Adapter 保留窄接口；TimescaleDB 仅作为可重放历史投影。

**Tech Stack:** Go 1.25、pgx、Squirrel、PostgreSQL、TimescaleDB、NATS JetStream、
Gin、Prometheus、React 19、TypeScript、TanStack Query、Ant Design、Vitest、
Playwright。

## 执行状态（2026-08-04）

| Task | 状态 | 证据 |
| --- | --- | --- |
| 1 | 已完成 | `f112d0529` |
| 2 | 已完成 | `512a8aba1` |
| 3 | 已完成 | `2147c4133`、`6a3e8371b` |
| 4 | 已完成 | `586a90099` |
| 5 | 已完成 | `da50a3a63` |
| 6 | 进行中 | Shadow、canonical、北向真实报文及 NATS 进程级故障恢复已在隔离环境通过；现场 2 倍峰值容量和 Shadow 回退演练仍待执行，见[本地验证记录](../reviews/2026-08-04-email-sms-milestone-a-local-validation.md) |
| 7–14 | 未开始 | 里程碑 A 通过前不得进入 |

下文保留原始 RED/GREEN checkbox 作为实施步骤模板；是否完成以本表、提交和验证记录共同
判断，不能只按 checkbox 推断。

## Global Constraints

- OMC 面向小基站运维，按运营商级告警可靠性和可审计性处理。
- SQL 使用 Squirrel + pgx，禁止 ORM 和运行期字符串拼接 SQL。
- 后端错误使用 `fmt.Errorf("context: %w", err)` 包装。
- 运营商差异只能进入 `internal/core/carrier/`。
- 数据库变化只折回三个 `000001` 基线，不创建 `000002+`。
- 当前前端只维护 V1；所有用户可见文案必须进入中英文 i18n。
- 用户、角色和投递历史必须通过 `GetUserVisibleDeviceGrants` 同时服从设备组和制式权限；
  `carrier` 不是租户或数据权限来源。
- `domain.alarm.lifecycle.*` 是唯一标准生命周期事实；legacy `alarm.*` 不得被通知中心消费。
- 备份、磁盘和渠道故障等无受管网元身份的事件属于 system incident/health 域，禁止伪造
  `device_id` 迁入 AlarmEngine。
- SMTP accepted 不等于 delivered；Kafka Ack 只能记为 handoff。
- 直连短信不在本计划实施范围，必须在供应商协议和回执契约冻结后单独设计。
- 通知维护窗口复用 `ops_maintenance_windows`：不抑制告警事实，只为命中范围的外部投递
  记录可审计的 suppressed 原因。
- 事件、schedule 和 delivery 统一以 UTC 保存；quiet hours、摘要和周期提醒绑定 IANA
  时区，禁止依赖进程本地时区。
- 老 `operator_code` 与当前 `carrier` 不直接等价。运营商硬隔离如仍为独立业务边界，
  必须由统一 PermissionService 承载，通知规则不能以 carrier 匹配替代权限。
- 每个任务先写失败测试，再写最小实现，并使用 Conventional Commits 中文提交。

## 交付边界与顺序

本计划分为两个独立验收里程碑：

1. **A：可靠告警事件基础**——不发送通知，可单独上线 Shadow。
2. **B：通知核心与邮件试点**——完成规则、模板、调度、邮件、历史和迁移预览。
短信 Adapter 不进入本轮可执行代码里程碑：当前项目没有 Kafka 客户端或外部短信供应商
契约。任何里程碑未通过其验收门禁，不进入下一里程碑。

---

## 里程碑 A：可靠告警事件基础

### Task 1: 定义告警生命周期契约与 JetStream

**Files:**
- Create: `omcgo/internal/core/event/alarm_lifecycle.go`
- Modify: `omcgo/internal/core/event/subjects.go`
- Modify: `omcgo/internal/core/components/nats/nats.go`
- Test: `omcgo/internal/core/event/alarm_lifecycle_test.go`
- Test: `omcgo/internal/core/components/nats/streams_test.go`

**Interfaces:**
- Produces: `event.AlarmLifecyclePayload`, `event.SubjectDomainAlarmLifecycle*`
- Consumes: `model.Alarm`

- [ ] **Step 1: 写生命周期载荷往返测试**

```go
func TestAlarmLifecyclePayload_RoundTrip(t *testing.T) {
    previous := model.Alarm{Severity: model.AlarmMajor}
    alarm := model.Alarm{
        ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "SN-1",
        Severity: model.AlarmCritical,
    }
    occurredAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
    payload, err := event.NewAlarmLifecyclePayload(
        event.AlarmLifecycleUpdated, alarm, &previous, 2, occurredAt,
        []event.AlarmChangeField{event.AlarmChangeSeverity},
    )
    require.NoError(t, err)
    require.Equal(t, alarm.ID, payload.OccurrenceID)
    require.Equal(t, alarm.ID, payload.Snapshot.AlarmID)
    require.Equal(t, int64(2), payload.AlarmVersion)
    require.Equal(t, model.AlarmMajor, *payload.PreviousSeverity)
    require.Equal(t, occurredAt, payload.OccurredAt)
    require.Equal(t, 1, payload.SchemaVersion)
}
```

- [ ] **Step 2: 运行测试并确认因类型不存在而失败**

Run: `cd omcgo && go test ./internal/core/event -run AlarmLifecycle -count=1`

Expected: FAIL，提示 `AlarmLifecyclePayload` 或 Subject 未定义。

- [ ] **Step 3: 实现版本化契约和五个新 Subject**

```go
const (
    SubjectDomainAlarmLifecycleRaised       = "domain.alarm.lifecycle.raised"
    SubjectDomainAlarmLifecycleUpdated      = "domain.alarm.lifecycle.updated"
    SubjectDomainAlarmLifecycleAcknowledged = "domain.alarm.lifecycle.acknowledged"
    SubjectDomainAlarmLifecycleUnacknowledged = "domain.alarm.lifecycle.unacknowledged"
    SubjectDomainAlarmLifecycleCleared      = "domain.alarm.lifecycle.cleared"
)

type AlarmLifecyclePayload struct {
    SchemaVersion    int                    `json:"schema_version"`
    EventID          uuid.UUID              `json:"event_id"`
    LifecycleType    AlarmLifecycleType     `json:"lifecycle_type"`
    OccurredAt       time.Time              `json:"occurred_at"`
    OccurrenceID     uuid.UUID              `json:"occurrence_id"`
    AlarmVersion     int64                  `json:"alarm_version"`
    ChangeMask       []AlarmChangeField     `json:"change_mask,omitempty"`
    PreviousSeverity *model.AlarmSeverity   `json:"previous_severity,omitempty"`
    Snapshot         AlarmLifecycleSnapshot `json:"snapshot"`
}
```

`AlarmLifecycleSnapshot` 显式声明设计文档列出的稳定字段，禁止嵌套 `model.Alarm`。
Builder 只复制审核后的 extensions 键并执行长度限制，避免内部模型变化或调用方后续修改
map 改写快照；`occurredAt` 必须由事务存储传入，builder 不自行取时间。构造器拒绝未知
生命周期类型、零 occurrence/device ID、非正版本、零变更时间，以及缺旧快照的 severity
变更。

- [ ] **Step 4: 为标准事件新增独立 LimitsPolicy 流测试**

测试 `DefaultStreams()` 包含：

```go
StreamDef{
    Name: "DOMAIN_ALARM",
    Subjects: []string{"domain.alarm.>"},
    Retention: nats.LimitsPolicy,
    AllowDirect: true,
    MaxAge: 7 * 24 * time.Hour,
    MaxBytes: alarmLifecycleStreamMaxBytes,
    Compression: nats.S2Compression,
}
```

初始 `alarmLifecycleStreamMaxBytes` 为 10 GiB 安全硬上限；测试还须验证该值大于零，并与
legacy `ALARM` 的 `alarm.>` 不重叠。Task 4 接入部署配置覆盖。MaxAge 不是 7 天容量承诺，
正式启用前必须完成 Task 14 容量计算和提前淘汰门禁。

- [ ] **Step 5: 运行契约和流定义测试**

Run: `cd omcgo && go test ./internal/core/event ./internal/core/components/nats -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/core/event omcgo/internal/core/components/nats
git commit -m "feat(alarm): 定义可靠告警生命周期事件"
```

### Task 2: 建立告警版本与领域 Outbox 基线

**Files:**
- Modify: `omcgo/migrations/000001_init_schema.sql`
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Modify: `omcgo/internal/core/model/alarm.go`
- Create: `omcgo/internal/alarm/outbox_model.go`
- Test: `omcgo/internal/alarm/outbox_schema_test.go`

**Interfaces:**
- Produces: `model.Alarm.Version`, `alarm.OutboxRecord`
- Consumes: Task 1 生命周期载荷

- [ ] **Step 1: 写基线结构守卫测试**

测试读取两个基线 SQL，断言主库含：

```sql
ALTER-compatible final schema:
alarms_active.alarm_version bigint NOT NULL DEFAULT 1
alarm_event_outbox.event_id uuid UNIQUE NOT NULL
alarm_event_outbox.aggregate_version bigint NOT NULL
alarm_event_outbox UNIQUE (aggregate_id, aggregate_version)
alarm_event_outbox.payload jsonb NOT NULL
alarm_event_outbox.status varchar(16) NOT NULL DEFAULT 'pending'
```

同时断言 TSDB `alarms_history` 含 `alarm_version bigint NOT NULL`。

- [ ] **Step 2: 运行结构测试并确认失败**

Run: `cd omcgo && go test ./internal/alarm -run OutboxSchema -count=1`

Expected: FAIL，指出缺少字段或表。

- [ ] **Step 3: 折回主库和 TSDB 基线**

`alarm_event_outbox` 至少包含：

```sql
id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
event_id uuid NOT NULL UNIQUE,
aggregate_type varchar(32) NOT NULL,
aggregate_id uuid NOT NULL,
aggregate_version bigint NOT NULL,
subject varchar(128) NOT NULL,
payload jsonb NOT NULL,
status varchar(16) NOT NULL DEFAULT 'pending',
attempt_count integer NOT NULL DEFAULT 0,
next_attempt_at timestamptz NOT NULL DEFAULT now(),
locked_by varchar(128),
locked_at timestamptz,
published_at timestamptz,
last_error text,
created_at timestamptz NOT NULL DEFAULT now(),
updated_at timestamptz NOT NULL DEFAULT now()
```

增加 `(status, next_attempt_at)` 索引和 `(aggregate_id, aggregate_version)` 唯一约束；不新增
迁移文件。

- [ ] **Step 4: 将版本字段接入模型和行扫描**

修改 `model.Alarm`、`SaveActive`、活动/历史查询列及 scanner，使 `alarm_version` 全链路可读写。

- [ ] **Step 5: 运行告警存储测试**

Run: `cd omcgo && go test ./internal/alarm -run 'OutboxSchema|PgAlarmStore|AlarmRow' -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add omcgo/migrations omcgo/internal/core/model/alarm.go omcgo/internal/alarm
git commit -m "feat(alarm): 增加生命周期版本和事件Outbox"
```

### Task 3: 将所有活动告警变更与 Outbox 原子提交

**Files:**
- Modify: `omcgo/internal/alarm/store.go`
- Modify: `omcgo/internal/alarm/pg_store.go`
- Modify: `omcgo/internal/alarm/engine.go`
- Modify: `omcgo/internal/alarm/handler.go`
- Test: `omcgo/internal/alarm/pg_store_lifecycle_test.go`
- Test: `omcgo/internal/alarm/engine_lifecycle_test.go`
- Test: `omcgo/internal/alarm/handler_test.go`

**Interfaces:**
- Produces: `PersistRaised`, `PersistUpdated`, `PersistAcknowledged`,
  `PersistUnacknowledged`, `PersistCleared`
- Consumes: Task 1 payload、Task 2 Outbox

- [ ] **Step 1: 写事务回滚测试**

覆盖五类状态变化，至少断言：

```text
活动行成功 + Outbox 失败 => 整个主库事务回滚
活动行失败 => 不存在 Outbox
清除成功 => 活动行删除且 cleared Outbox 含完整告警快照
每次变化 => alarm_version 严格 +1
并发更新 => 版本在事务锁内串行分配，不产生相同或跳跃版本
新告警 auto_clear => 同一事务产生 raised v1 和 cleared v2
```

- [ ] **Step 2: 运行测试并确认旧存储无法保证原子性**

Run: `cd omcgo && go test ./internal/alarm -run 'LifecycleTx|OutboxRollback' -count=1`

Expected: FAIL。

- [ ] **Step 3: 实现窄生命周期存储接口**

```go
type LifecycleStore interface {
    PersistRaised(context.Context, LifecycleMutation) (event.AlarmLifecyclePayload, error)
    PersistUpdated(context.Context, LifecycleMutation) (event.AlarmLifecyclePayload, error)
    PersistAcknowledged(context.Context, LifecycleMutation) (event.AlarmLifecyclePayload, error)
    PersistUnacknowledged(context.Context, LifecycleMutation) (event.AlarmLifecyclePayload, error)
    PersistCleared(context.Context, LifecycleMutation) (event.AlarmLifecyclePayload, error)
}
```

每个方法内部只使用 `s.pool.Begin(ctx)`；清除事务删除 `alarms_active` 并写 Outbox，不访问
`s.tsPool`。存储方法先 `SELECT ... FOR UPDATE`（或使用等价 expected-version 条件）取得旧
快照，在事务内分配下一版本、确定变更时间、构造 payload 并写 Outbox。AlarmEngine 不得
在事务外传入已计算的版本、previous severity 或最终 payload。

- [ ] **Step 4: AlarmEngine 发起变更并由存储事务构造事件**

产生、重复更新、severity 更新、确认、取消确认、自动清除、人工清除和同步对账全部使用
同一 builder。
使用单一 `alarm.lifecycle_mode=legacy|shadow|canonical` 配置，默认 `legacy`。`shadow` 保留
legacy 行为并生成 canonical 对比数据；`canonical` 只能在 Relay、历史投影和北向消费者
全部就绪后启用。

- [ ] **Step 5: 将批量活动告警操作改为经过 AlarmEngine**

`Handler.BatchAcknowledge`、`BatchUnacknowledge`、`BatchClear` 不再调用 store 批量 SQL。
Engine 逐 occurrence 执行生命周期事务并返回：

```go
type BatchMutationResult struct {
    Succeeded []uuid.UUID
    Failed    map[uuid.UUID]string
}
```

接口响应必须如实返回部分失败，不能继续固定返回请求数量。

- [ ] **Step 6: 运行告警包测试**

Run: `cd omcgo && go test ./internal/alarm -count=1`

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/alarm
git commit -m "refactor(alarm): 原子提交告警变更和领域事件"
```

### Task 4: 实现 Outbox Relay 与运维指标

**Files:**
- Create: `omcgo/internal/alarm/outbox_repository.go`
- Create: `omcgo/internal/alarm/outbox_relay.go`
- Create: `omcgo/internal/alarm/outbox_relay_test.go`
- Modify: `omcgo/internal/alarm/metrics.go`
- Modify: `omcgo/cmd/app/provider/alarm.go`
- Modify: `omcgo/cmd/worker/main.go`
- Modify: `omcgo/internal/core/appconfig/config.go`
- Modify: `omcgo/internal/core/components/persistent_queue_metrics.go`

**Interfaces:**
- Produces: `OutboxRelay.Start`, `OutboxRelay.Stop`, `Replay(eventID)`
- Consumes: Task 2 Outbox、Task 1 Subject

- [ ] **Step 1: 写 claim、重试、重复发布测试**

测试 SQL 必须包含 `FOR UPDATE SKIP LOCKED`，并验证：

```text
pending/failed 且 next_attempt_at 到期才能领取
发布失败 => attempt_count+1、脱敏 last_error、指数退避
崩溃遗留 locked 行 => 租约到期后重新领取
发布成功 => published_at 非空
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd omcgo && go test ./internal/alarm -run OutboxRelay -count=1`

Expected: FAIL。

- [ ] **Step 3: 参考 PM Outbox 模式实现 Relay**

复用 `internal/pm/stream/outbox_relay.go` 的 claim/mark/requeue 结构，但保持 alarm 包独立。
NATS `Event.ID` 必须等于 Outbox `event_id`，不能由 `event.NewEvent` 再生成新 ID。
只有收到 JetStream PubAck 后才允许将 Outbox 标为 published。

- [ ] **Step 4: 增加指标和 30 天保留清理**

至少暴露 backlog、oldest age、publish total、retry total、dead total；清理只删除
`published_at < now()-30d` 的已发布记录。`Replay` 只能把指定已发布记录安全重置为 pending。

- [ ] **Step 5: 在 app 和 worker 两种部署入口装配并注册优雅停止**

保证同一进程只启动一个 Relay；多副本依赖 `SKIP LOCKED` 分担。健康检查只报告积压和
最老延迟，不因短暂 NATS 故障阻止告警写入。

- [ ] **Step 6: 接入单一模式、容量配置和起始 sequence**

`alarm.lifecycle_mode` 默认 `legacy`，只有 `shadow|canonical` 运行 Relay；
`alarm.lifecycle_stream_max_bytes` 覆盖 Task 1 的 10 GiB 代码默认值。创建 DOMAIN_ALARM 流
后记录 `alarm.lifecycle_start_sequence`；历史、北向和通知 durable 必须显式使用该
sequence，不能依赖新 consumer 从尾部开始。超过流保留期时只能从已发布 Outbox 受控重放。

- [ ] **Step 7: 运行测试**

Run: `cd omcgo && go test ./internal/alarm ./internal/core/components -count=1`

Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/alarm omcgo/internal/core/appconfig omcgo/cmd omcgo/internal/core/components
git commit -m "feat(alarm): 发布可靠生命周期Outbox事件"
```

### Task 5: TimescaleDB 历史投影与受控切换

**Files:**
- Create: `omcgo/internal/alarm/history_projector.go`
- Create: `omcgo/internal/alarm/history_projector_test.go`
- Modify: `omcgo/internal/alarm/pg_store.go`
- Modify: `omcgo/cmd/app/provider/alarm.go`
- Modify: `omcgo/cmd/worker/main.go`
- Modify: `omcgo/internal/core/appconfig/config.go`

**Interfaces:**
- Produces: `HistoryProjector.Subscribe`, `HistoryProjector.SetShadow`
- Consumes: `domain.alarm.lifecycle.cleared`

- [ ] **Step 1: 写重复、失败和 Shadow 测试**

测试同一 `alarm_id + alarm_version` 重复消息只形成一条历史；TSDB 失败返回 error 让
JetStream 重投；Shadow 模式只比较、不写入。

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd omcgo && go test ./internal/alarm -run HistoryProjector -count=1`

Expected: FAIL。

- [ ] **Step 3: 增加 TSDB 幂等键**

TimescaleDB 超表唯一约束必须包含分区时间。为投影使用确定性的 `time=cleared_at`，增加：

```sql
UNIQUE (time, alarm_id, alarm_version)
```

投影使用 Squirrel `INSERT ... ON CONFLICT (time, alarm_id, alarm_version) DO NOTHING`。

- [ ] **Step 4: 实现独立 durable 消费者**

durable 名称固定为 `alarm-history-projector-v1`。仅处理 cleared；schema version 不支持时
拒绝 Ack 并产生结构化错误。

- [ ] **Step 5: 实现单一受控切换**

历史写路径服从全局生命周期模式：

```text
alarm.lifecycle_mode=legacy|shadow|canonical
```

`legacy` 保留原同步 Archive；`shadow` 保留原同步 Archive 并让 projector 只比较；
`canonical` 禁止原同步 Archive，改由 projector 写入。启动时发现配置形成两个正式写路径
必须失败关闭 canonical，不能双写。projector 未追平或不可用时不得切换 canonical。

- [ ] **Step 6: 运行告警集成测试**

Run: `cd omcgo && go test ./internal/alarm -count=1`

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/alarm omcgo/internal/core/appconfig omcgo/cmd omcgo/migrations/tsdb
git commit -m "feat(alarm): 投影清除告警到TimescaleDB"
```

### Task 6: 接入标准消费者并完成 legacy 受控切换

**Files:**
- Create: `omcgo/internal/northbound/push/alarm_lifecycle_consumer.go`
- Create: `omcgo/internal/northbound/push/alarm_lifecycle_consumer_test.go`
- Modify: `omcgo/internal/northbound/push/engine.go`
- Modify: `omcgo/internal/core/event/subjects.go`
- Modify: `omcgo/internal/alarm/engine.go`
- Modify: `omcgo/internal/northbound/push/engine_test.go`
- Modify: `omcgo/internal/alarm/engine_test.go`

**Interfaces:**
- Produces: 单一标准生命周期发布链
- Consumes: Task 4 Relay、Task 5 Projector

- [ ] **Step 1: 写系统事件领域边界守卫测试**

盘点并断言备份失败、存储阈值和通知渠道故障没有被转换为要求受管网元 `device_id` 的
`AlarmLifecyclePayload`。这些现有 legacy `alarm.*` 自定义载荷不属于本轮 canonical 事实，
不得被历史投影、北向生命周期消费者或通知中心订阅。

- [ ] **Step 2: 保持系统事件独立并记录后续迁移边界**

备份、磁盘和渠道健康继续使用各自 system incident/health 路径及站内通知/指标；移除
`alert_email` 等直连收件人提示应作为独立兼容任务处理。只有未来建立具备独立 identity、
生命周期、权限和查询模型的 system alarm 域后，才允许迁移，不能伪造基站设备。

- [ ] **Step 3: 将北向消费者接入标准 payload**

北向使用独立 durable `northbound-alarm-lifecycle-v1` 和明确的 start sequence，映射
`payload.Snapshot` 后直接写现有 `northbound_outbox`，不能先 Ack 生命周期事件再依赖一次
不可靠的二次 NATS 发布。consumer 不同时订阅 legacy 和标准 Subject。

- [ ] **Step 4: 删除 AlarmEngine legacy 直接发布**

只有在 Shadow 计数、历史投影和北向对比通过后执行。删除四处直接
`eventBus.Publish(alarm.*)`，保留 legacy Subject 常量一段兼容期但标注 deprecated。

- [ ] **Step 5: 运行受影响测试**

Run: `cd omcgo && go test ./internal/alarm ./internal/northbound/... ./internal/core/event -count=1`

Expected: PASS。

- [ ] **Step 6: 里程碑 A 验收**

验证 NATS 中每个 `event_id` 唯一、所有已启用 durable 独立收到事件、TSDB 中清除历史不
重复、NATS 中断不阻塞告警事务且恢复后 Relay 补发；历史和北向两个 durable 在本 Task
验收；通知 Shadow durable 在通知 Inbox 建立后补齐，属于后续生产 canonical 切换门禁，
不反向阻塞 Task 7。三者全部通过前不得在生产切 canonical。同时验证 stream bytes、最老
消息和 durable lag 满足容量门禁，且系统事件未混入受管网元生命周期。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/alarm omcgo/internal/northbound omcgo/internal/core/event
git commit -m "refactor(alarm): 统一标准告警生命周期事件"
```

---

## 里程碑 B：通知核心与邮件试点

### Task 7: 建立通知中心领域基线

**Files:**
- Modify: `omcgo/migrations/000001_init_schema.sql`
- Create: `omcgo/internal/notification/domain_model.go`
- Create: `omcgo/internal/notification/domain_repository.go`
- Create: `omcgo/internal/notification/pg_domain_repository.go`
- Test: `omcgo/internal/notification/domain_schema_test.go`
- Test: `omcgo/internal/notification/pg_domain_repository_test.go`

**Interfaces:**
- Produces: Inbox、Occurrence、RuleVersion、TemplateVersion、Schedule、Delivery、Attempt 仓储
- Consumes: `event.AlarmLifecyclePayload`

- [ ] **Step 1: 写表、约束和索引守卫测试**

覆盖设计中的 `notification_events`、`notification_occurrences`、规则及版本、模板及版本、
收件人、联系组、渠道、健康、schedule、delivery、attempt、aggregation bucket。

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd omcgo && go test ./internal/notification -run DomainSchema -count=1`

Expected: FAIL。

- [ ] **Step 3: 折回主库基线**

关键唯一约束必须包括：

```sql
notification_events(event_id)
notification_occurrences(occurrence_id)
notification_deliveries(event_id, dispatch_kind, sequence_no, channel, recipient_fingerprint)
notification_delivery_attempts(delivery_id, attempt_no)
notification_schedules(occurrence_id, rule_version_id, channel, recipient_fingerprint,
                       schedule_kind, sequence_no, generation)
```

地址快照保存 `ciphertext + key_version + hmac_fingerprint`，不保存明文。

- [ ] **Step 4: 实现小型仓储接口**

按 Inbox、Rules、Schedules、Deliveries 分文件和接口，禁止构造一个覆盖所有表的巨型
repository。所有 claim 使用 `FOR UPDATE SKIP LOCKED` 和 lease。

- [ ] **Step 5: 运行 repository 测试**

Run: `cd omcgo && go test ./internal/notification -run 'DomainSchema|Repository' -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add omcgo/migrations/000001_init_schema.sql omcgo/internal/notification
git commit -m "feat(notification): 建立通知中心领域模型"
```

### Task 8: 实现 Inbox 顺序投影与持久化调度

**Files:**
- Create: `omcgo/internal/notification/lifecycle_consumer.go`
- Create: `omcgo/internal/notification/lifecycle_consumer_test.go`
- Create: `omcgo/internal/notification/scheduler.go`
- Create: `omcgo/internal/notification/scheduler_test.go`
- Modify: `omcgo/cmd/app/provider/modules.go`

**Interfaces:**
- Produces: `LifecycleConsumer.Handle`, `Scheduler.RunOnce`
- Consumes: Task 7 仓储、Task 1 标准事件

- [ ] **Step 1: 写版本状态机测试**

覆盖：

```text
version=1 创建 occurrence
重复/旧版本只审计不重复编排
版本缺口进入 waiting
缺失版本补齐后顺序继续
ack/clear 递增 schedule_generation 并取消旧 schedule
unack 不重发 initial，只在规则上限内恢复未来 repeat
```

- [ ] **Step 2: 写重启和领取竞争测试**

两个 Scheduler 实例只能领取一次 due schedule；lease 到期可恢复；claim 后 clear 导致
generation fence 失败，不产生 delivery。

- [ ] **Step 3: 实现 lifecycle durable**

durable 固定为 `notification-lifecycle-v1`，按 `occurrence_id` 进行有界 hash 分片，保证同一
occurrence 串行，不同基站可并发。首次创建 durable 时使用里程碑 A 记录的 start sequence；
超过 DOMAIN_ALARM 保留期的缺口先从 Outbox 重放。

- [ ] **Step 4: 实现 schedule worker**

`initial_gate`、`repeat`、`digest_flush`、`quiet_hours_release` 分别由显式 handler 处理。
领取和状态转换必须是条件更新；业务 handler 不直接 sleep。

- [ ] **Step 5: 运行测试**

Run: `cd omcgo && go test ./internal/notification -run 'LifecycleConsumer|Scheduler' -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/notification omcgo/cmd/app/provider/modules.go
git commit -m "feat(notification): 编排有序告警事件和持久化调度"
```

### Task 9: 实现规则、版本、模板和收件人 API

**Files:**
- Create: `omcgo/internal/notification/rule_service.go`
- Create: `omcgo/internal/notification/rule_handler.go`
- Create: `omcgo/internal/notification/recipient_resolver.go`
- Create: `omcgo/internal/notification/contact_group_service.go`
- Create: `omcgo/internal/notification/contact_group_handler.go`
- Create: `omcgo/internal/notification/channel_handler.go`
- Modify: `omcgo/internal/notification/template_model.go`
- Modify: `omcgo/internal/notification/template_repository.go`
- Modify: `omcgo/internal/notification/pg_template_repository.go`
- Modify: `omcgo/internal/notification/template_service.go`
- Modify: `omcgo/internal/notification/template_handler.go`
- Modify: `omcgo/cmd/app/provider/router.go`
- Create: `omcgo/internal/notification/rule_service_test.go`
- Create: `omcgo/internal/notification/rule_handler_test.go`
- Create: `omcgo/internal/notification/recipient_resolver_test.go`
- Create: `omcgo/internal/notification/contact_group_service_test.go`
- Create: `omcgo/internal/notification/channel_handler_test.go`
- Modify: `omcgo/internal/notification/template_service_test.go`
- Modify: `omcgo/internal/notification/template_handler_test.go`

**Interfaces:**
- Produces: `/api/v1/notification-rules`、版本化模板、联系组、渠道配置、preview、publish、
  enable
- Consumes: Task 7 仓储、现有 authz 数据权限

- [ ] **Step 1: 写规则匹配和冲突测试**

测试字段内 OR、字段间 AND、优先级、具体度、稳定 ID 决胜，以及多规则合并同一地址时只
产生一个候选投递。

- [ ] **Step 2: 写乐观锁与版本测试**

覆盖缺少 `If-Match` 返回 428、旧 revision 返回 412、发布后不可变、启用必须绑定已发布
rule version、模板新版本不自动改变启用规则。

- [ ] **Step 3: 实现 draft/publish/enable 服务**

Handler 只做绑定和鉴权，事务及版本不变量在 service/repository。继续使用现有
`/notifications/templates` 时提供兼容只读响应；新写路径必须创建版本。

- [ ] **Step 4: 实现收件人解析和权限求交**

固定联系人、用户、角色、默认 NOC 组最终统一为：

```go
type ResolvedRecipient struct {
    UserID      *uuid.UUID
    Channel     Channel
    Ciphertext  []byte
    KeyVersion  int
    Fingerprint []byte
    Locale      string
}
```

通过 `PermissionService.GetUserVisibleDeviceGrants` 取得用户或角色的授权，告警设备必须
同时命中设备组和制式 grant；不能沿用已删除的 `users.carrier`，也不能只判断设备组。
无权限结果记录排除原因。

- [ ] **Step 5: 实现联系组、渠道配置与写审计**

联系组成员仍经过设备权限求交。渠道 API 只返回非敏感参数和 secret reference 状态，
不回显密码、Token 或私钥。规则、模板、联系组和渠道的创建、发布、启停、归档及秘密
变更全部写现有操作审计，敏感字段只记录“已变更”。

- [ ] **Step 6: 运行 API 和服务测试**

Run: `cd omcgo && go test ./internal/notification -run 'Rule|Template|Recipient' -count=1`

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/notification omcgo/cmd/app/provider/router.go
git commit -m "feat(notification): 增加版本化通知规则和模板"
```

### Task 10: 实现通知编排、恢复配对和风暴控制

**Files:**
- Create: `omcgo/internal/notification/orchestrator.go`
- Create: `omcgo/internal/notification/orchestrator_test.go`
- Create: `omcgo/internal/notification/aggregation.go`
- Create: `omcgo/internal/notification/rate_limit.go`
- Create: `omcgo/internal/notification/delivery_service.go`
- Create: `omcgo/internal/notification/delivery_handler.go`
- Modify: `omcgo/cmd/app/provider/router.go`
- Create: `omcgo/internal/notification/aggregation_test.go`
- Create: `omcgo/internal/notification/rate_limit_test.go`
- Create: `omcgo/internal/notification/delivery_service_test.go`
- Create: `omcgo/internal/notification/delivery_handler_test.go`

**Interfaces:**
- Produces: schedule 与 delivery
- Consumes: Task 8 occurrence、Task 9 规则及收件人

- [ ] **Step 1: 写产生、升级、确认、清除测试**

覆盖瞬时告警 suppressed、首次门槛、普通重复不重发、跨阈值升级、确认停止 repeat、
取消确认按原 sequence 恢复未来 repeat，以及仅为 accepted/handoff 的原收件人渠道发送恢复。

- [ ] **Step 2: 写风暴和聚合测试**

Critical 绕过普通摘要但受最大次数和收件人速率限制；Minor/Warning 进入 bucket；超限必须
产生 suppressed 或 digest 事实，不能静默丢弃。

增加维护窗口场景：已审批且生效的窗口命中设备时保留 occurrence 和事件审计，但外部
delivery 记录为 `suppressed` 并关联窗口；窗口不命中、未审批或已结束时不得抑制。规则
预览同时展示业务时区、下一次实际执行时间以及夏令时边界下的唯一执行结果。

- [ ] **Step 3: 实现纯规则决策与事务写入**

将 matcher、policy evaluator 设计为纯函数；Orchestrator 在一个主库事务中保存匹配解释、
schedule、bucket 或 delivery。

- [ ] **Step 4: 实现外呼前栅栏**

```go
func (s *Service) AuthorizeSend(
    ctx context.Context, deliveryID uuid.UUID,
) (AuthorizedDelivery, error)
```

该方法原子校验 occurrence 状态、事件版本、schedule generation、delivery lease 和渠道
熔断状态。失败时写 cancelled/suppressed 原因。

- [ ] **Step 5: 实现投递、attempt、轨迹和人工重试 API**

提供设计中的 delivery 列表/详情/attempt/retry API。查询必须按关联告警的设备组 + 制式
grant 过滤，地址默认脱敏；完整地址需要独立权限。批量重试最多 100 条、必须填写原因，只允许永久配置已
修复后的 dead_letter，操作写审计。告警详情轨迹只返回当前用户可见 occurrence。

- [ ] **Step 6: 运行测试**

Run: `cd omcgo && go test ./internal/notification -run 'Orchestrator|Aggregation|RateLimit|AuthorizeSend' -count=1`

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/notification omcgo/cmd/app/provider/router.go
git commit -m "feat(notification): 编排告警通知和风暴控制"
```

### Task 11: 统一 SMTP 渠道、Worker、重试和熔断

**Files:**
- Modify: `omcgo/internal/notification/email_sender.go`
- Create: `omcgo/internal/notification/email_worker.go`
- Create: `omcgo/internal/notification/email_worker_test.go`
- Create: `omcgo/internal/notification/channel_health.go`
- Modify: `omcgo/internal/notification/channel_handler.go`
- Modify: `omcgo/internal/notification/renderer.go`
- Modify: `omcgo/cmd/app/provider/modules.go`
- Modify: `omcgo/cmd/app/provider/alarm.go`

**Interfaces:**
- Produces: Email delivery `completed+accepted`
- Consumes: Task 10 authorized delivery

- [ ] **Step 1: 写逐收件人、安全模板和状态测试**

SMTP 每个信封只有一个目标地址；缺变量渲染失败；成功只记 accepted；鉴权/证书失败打开
熔断；网络/5xx 临时错误退避；永久地址错误不重试。

- [ ] **Step 2: 写未知结果和幂等测试**

每次 attempt 使用稳定 delivery ID 作为日志关联键；外呼超时但结果未知时不盲目立即重发，
进入 `unknown` 或人工确认策略。

- [ ] **Step 3: 实现 Worker**

Worker 流程固定为 claim → `AuthorizeSend` → attempt start → SMTP → attempt/result update。
禁止在 AlarmEngine 或 Orchestrator 中直接外呼。

- [ ] **Step 4: 收敛两套 SMTP**

删除 `cmd/app/provider/alarm.go` 的 `OMC_SMTP_*` 告警 dispatcher 装配。默认邮件渠道复用
当前 `c.Cfg.Notification.SMTP`，秘密只通过运行配置/Secret 引用进入 adapter。

- [ ] **Step 5: 增加渠道故障系统事件与健康信号**

产生 `NOTIFICATION_CHANNEL_UNAVAILABLE` system incident/health event、指标和站内通知；
不得伪造成带 `device_id` 的网元告警。若由其他健康渠道或北向转发，必须设置
`origin=notification` 并递归排除故障渠道。

- [ ] **Step 6: 实现连接验证与测试发送**

连接验证只检查配置和握手；测试发送必须使用有独立权限的明确测试地址，并经过
`AuthorizeSend` 等价的渠道状态检查。两种操作都写审计，API 响应和日志不回显秘密。

- [ ] **Step 7: 运行测试**

Run: `cd omcgo && go test ./internal/notification ./internal/alarm -count=1`

Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/notification omcgo/cmd/app/provider
git commit -m "feat(notification): 增加可靠邮件投递Worker"
```

### Task 12: 迁移预览、兼容 barrier 与 Shadow 门禁

**Files:**
- Create: `omcgo/internal/notification/migration_service.go`
- Create: `omcgo/internal/notification/migration_service_test.go`
- Modify: `omcgo/internal/alarm/filter_model.go`
- Modify: `omcgo/internal/alarm/filter_engine.go`
- Modify: `omcgo/internal/alarm/pg_filter_repository.go`
- Modify: `omcgo/migrations/000001_init_schema.sql`

**Interfaces:**
- Produces: legacy rule preview、overlap findings、compatibility barrier
- Consumes: 当前 `alarm_filters.notify_email`

- [ ] **Step 1: 写首条命中回归测试**

构造高优先级 notify_email 和低优先级 ignore/auto_clear，断言迁移后 barrier 仍阻止低优先级
规则；无 barrier 的测试必须先展示行为回归。

- [ ] **Step 2: 写迁移预览测试**

预览输出命中范围、收件人、重叠低优先级规则、Tolerance Duration 未确认标志；生成的新
通知规则必须默认 disabled。

- [ ] **Step 3: 实现显式 compatibility action**

使用 `legacy_notification_barrier`，只返回 Handled，不发邮件、不改变告警。该 action 只能由
迁移服务创建，普通创建 API 不暴露。

- [ ] **Step 4: 实现分范围切换**

只有 overlap findings 已确认、Shadow 数量一致、规则明确启用时，才把对应 notify_email
切换成 barrier。`notify_webhook` 保持原状。

- [ ] **Step 5: 运行测试**

Run: `cd omcgo && go test ./internal/alarm ./internal/notification -run 'Migration|Barrier|FirstMatch' -count=1`

Expected: PASS。

- [ ] **Step 6: 里程碑 B 后端验收**

使用测试 SMTP 验证 raised、minimum duration、repeat、ack cancel、clear recovery、
熔断和逐收件人历史；对比 legacy 命中结果。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/alarm omcgo/internal/notification omcgo/migrations/000001_init_schema.sql
git commit -m "feat(notification): 安全迁移旧邮件告警规则"
```

### Task 13: 建设 V1 通知管理前端

**Files:**
- Modify: `omcmb/frontend-core/src/types/notification.ts`
- Modify: `omcmb/frontend-core/src/services/api/notificationApi.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useNotifications.ts`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`
- Modify: `omcmb/webcode/src/pages/notifications/index.tsx`
- Create: `omcmb/webcode/src/pages/notifications/RuleList.tsx`
- Create: `omcmb/webcode/src/pages/notifications/RuleDrawer.tsx`
- Create: `omcmb/webcode/src/pages/notifications/ChannelHealth.tsx`
- Modify: `omcmb/webcode/src/pages/notifications/TemplateForm.tsx`
- Modify: `omcmb/webcode/src/pages/notifications/TemplateList.tsx`
- Modify: `omcmb/webcode/src/pages/notifications/HistoryList.tsx`
- Modify: `omcmb/webcode/src/pages/notifications/HistoryDetail.tsx`
- Modify: `omcmb/webcode/src/pages/alarm/AlarmDetail/index.tsx`
- Create: `omcmb/frontend-core/src/services/api/__tests__/notificationApi.test.ts`
- Create: `omcmb/webcode/src/pages/notifications/RuleDrawer.test.tsx`
- Create: `omcmb/webcode/src/pages/notifications/ChannelHealth.test.tsx`
- Create: `omcmb/webcode/e2e/notification-management.spec.ts`

**Interfaces:**
- Produces: 规则、模板、联系组、渠道健康、delivery 历史、告警通知轨迹
- Consumes: Tasks 9–12 API

- [ ] **Step 1: 写 API 类型和 optimistic concurrency 测试**

客户端保存 `revision/ETag`，所有修改请求发送 `If-Match`；412 显示“配置已被其他管理员
修改，请刷新后重试”，不能静默覆盖。

- [ ] **Step 2: 写规则表单组件测试**

覆盖告警范围、收件人、渠道、minimum duration、摘要、repeat、recovery、quiet hours、
preview 和影响范围确认。所有断言使用 i18n key。

- [ ] **Step 3: 实现通知管理页面**

沿用现有 V1 `pages/notifications`，不恢复其他皮肤。旧模板和历史页迁移到新版本/逐投递
语义，legacy 历史明确标注批次数据。

- [ ] **Step 4: 实现告警详情通知轨迹**

只读展示命中规则、脱敏地址、状态、attempt、suppressed 原因和 recovery pairing；无编辑
入口。

- [ ] **Step 5: 运行单元、类型和构建验证**

Run:

```bash
cd omcmb/webcode
npm test -- --run
npm run typecheck
npm run build
```

Expected: 全部退出 0。

- [ ] **Step 6: 使用真实浏览器验证**

启动真实后端和 V1 前端，使用 Playwright 或应用内浏览器逐项验证创建、preview、发布、
启用、412 冲突、渠道健康、历史筛选、地址脱敏、人工重试和告警轨迹的真实请求参数。

- [ ] **Step 7: 运行 E2E**

Run: `cd omcmb/webcode && npx playwright test e2e/notification-management.spec.ts`

Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add omcmb/frontend-core omcmb/webcode
git commit -m "feat(notification): 建设V1通知管理界面"
```

---

## 最终验证与上线门禁

### Task 14: 全链路验证、容量基线与回退演练

**Files:**
- Create: `omcgo/internal/notification/e2e_test.go`
- Create: `docs/operations/notification-center-runbook.md`
- Modify: `omcgo/cmd/app/etc/config.dev.yaml`
- Modify: `omcgo/cmd/app/etc/config.prod.yaml`
- Modify: `omcgo/cmd/app/etc/config.test.yaml`
- Modify: `omcgo/cmd/worker/etc/config.dev.yaml`
- Modify: `omcgo/cmd/worker/etc/config.prod.yaml`
- Modify: `omcgo/cmd/worker/etc/config.test.yaml`

**Interfaces:**
- Produces: 上线证据和运行手册
- Consumes: 里程碑 A–B

- [ ] **Step 1: 运行后端全量相关测试**

Run:

```bash
cd omcgo
go test ./internal/alarm ./internal/notification ./internal/backup ./internal/northbound/... ./internal/core/event ./internal/core/components/nats -count=1
```

Expected: PASS。

- [ ] **Step 2: 运行前端全量验证**

Run:

```bash
cd omcmb/webcode
npm test -- --run
npm run lint
npm run typecheck
npm run build
```

Expected: 全部退出 0。

- [ ] **Step 3: 执行故障注入**

依次中断 NATS、TimescaleDB、SMTP，验证：

```text
主库告警写入/确认/清除继续可用
Outbox、history projection、delivery 均可恢复
没有重复邮件
ack/clear 后没有迟到提醒
渠道故障不会递归通知自身
```

- [ ] **Step 4: 建立容量基线**

以现场预估峰值至少 2 倍压测，记录事件处理 P50/P95/P99、Outbox oldest age、schedule
延迟、SMTP 吞吐、数据库锁等待和最大积压。实时通知内部处理 P95 必须小于 30 秒。根据
平均/峰值事件大小、7 天目标窗口、最长消费者中断和磁盘预算计算 DOMAIN_ALARM
`MaxBytes`；验证容量至少覆盖目标窗口，并对 stream bytes、最老消息、durable lag 和
提前淘汰风险配置告警。容量门禁未通过不得启用 Relay/canonical。

- [ ] **Step 5: 编写运行手册**

手册必须包含 Shadow 对比、开关顺序、durable/stream 检查、Outbox 重放、死信重试、
渠道熔断恢复、投影积压、数据脱敏检查和一键停止外部渠道但保持告警主流程的回退步骤。

- [ ] **Step 6: 最终真实浏览器验收**

按设计的浏览器验收矩阵执行并保存截图/请求证据；不得用 Mock 模式代替。

- [ ] **Step 7: 提交**

```bash
git add omcgo/internal/notification/e2e_test.go omcgo/cmd/app/etc omcgo/cmd/worker/etc docs/operations
git commit -m "test(notification): 完成通知中心全链路验收"
```

## 短信 Adapter 的后续入口

短信渠道只在下面两类外部契约之一冻结后另写设计与实施计划：

- Kafka Handoff：必须取得 Broker 地址、安全协议、认证方式、Topic、分区键、消息
  Schema、Ack 等级、幂等键、失败重试归属、容量和测试环境。实现后只能记录
  `handoff_only`，不能记录 delivered。
- 直连供应商：必须取得 API、鉴权、签名、请求幂等键、回执签名、回执乱序规则、状态
  查询、限流和沙箱环境，才能实现 accepted/delivered/failed/unknown。

两者都复用本计划的 Rule、Schedule、Delivery、Attempt、Circuit 和 `AuthorizeSend`，
仅新增独立 Adapter；直连模式额外增加 Receipt Processor。若外部平台不要求 Kafka，
不得仅为复刻老 OMC 而给当前项目引入 Kafka 运行依赖。
