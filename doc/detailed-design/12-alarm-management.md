# DD-12: 告警管理（F04）

> 关联功能域：F04（告警管理）
> 关联 backend-design.md 章节：第六章（告警管线）
> 实施阶段：Phase 3（数据管线）
> 依赖文档：DD-02, DD-04, DD-07

---

## 1. 概述

### 1.1 模块定位

告警管理（`internal/alarm/`）负责接收、处理和转发基站设备上报的告警事件。告警通过 TR069 Inform ALARM 事件上报，经过去重、关联分析后存储并可选转发至 OSS。

### 1.2 告警管线

```
ACS 解析 Inform ALARM 事件
  → alarm.raised (EventBus)
  → Alarm Receiver（提取告警参数）
  → Alarm Engine（去重、关联、抑制）
  → Alarm Store（Redis 活跃 + PostgreSQL + TimescaleDB 历史）
  → 可选: Alarm Forward（OSS 北向转发）
  → 清除时: alarm.cleared → 更新生命周期
```

---

## 2. 接口设计

### 2.1 Alarm Receiver — `internal/alarm/receiver.go`

```go
type AlarmReceiver struct {
    eventBus event.EventBus
    engine   *AlarmEngine
    logger   *zap.Logger
}

// Start 订阅告警事件
func (r *AlarmReceiver) Start() error
// handleAlarmEvent 处理 device.inform.alarm 事件
func (r *AlarmReceiver) handleAlarmEvent(ctx context.Context, evt event.Event) error
```

### 2.2 Alarm Engine — `internal/alarm/engine.go`

```go
type AlarmEngine struct {
    store      AlarmStore
    forwarder  *AlarmForwarder
    carrierReg *carrier.CarrierRegistry
    logger     *zap.Logger
}

// Process 处理告警事件
func (e *AlarmEngine) Process(ctx context.Context, alarm *model.Alarm) error
// Deduplicate 去重检查
func (e *AlarmEngine) Deduplicate(ctx context.Context, alarm *model.Alarm) (bool, error)
// Correlate 关联分析
func (e *AlarmEngine) Correlate(ctx context.Context, alarm *model.Alarm) error
// Suppress 抑制检查
func (e *AlarmEngine) Suppress(ctx context.Context, alarm *model.Alarm) (bool, error)
```

### 2.3 Alarm Lifecycle — `internal/alarm/lifecycle.go`

```go
// Acknowledge 确认告警
func (s *AlarmService) Acknowledge(ctx context.Context, alarmID uuid.UUID, by string) error
// Clear 清除告警
func (s *AlarmService) Clear(ctx context.Context, alarmID uuid.UUID) error
// AutoClear 自动清除（收到对应 clear 事件时）
func (s *AlarmService) AutoClear(ctx context.Context, deviceSN string, alarmCode string) error
```

### 2.4 Alarm Store — `internal/alarm/store.go`

```go
type AlarmStore interface {
    // 活跃告警
    SaveActive(ctx context.Context, alarm *model.Alarm) error
    GetActive(ctx context.Context, deviceID uuid.UUID) ([]model.Alarm, error)
    GetActiveByCode(ctx context.Context, deviceSN, alarmCode string) (*model.Alarm, error)
    RemoveActive(ctx context.Context, alarmID uuid.UUID) error
    ListActive(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error)

    // 历史告警
    Archive(ctx context.Context, alarm *model.Alarm) error
    ListHistory(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error)

    // 统计
    Statistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error)
}

type AlarmStatistics struct {
    TotalActive    int64
    BySeverity     map[model.AlarmSeverity]int64
    ByType         map[string]int64
    TopDevices     []DeviceAlarmCount
}
```

### 2.5 Alarm Forwarder — `internal/alarm/forward.go`

```go
type AlarmForwarder struct {
    eventBus event.EventBus
    enabled  bool
}

// Forward 转发告警到北向接口
func (f *AlarmForwarder) Forward(ctx context.Context, alarm *model.Alarm) error
```

---

## 3. 数据模型

### 3.1 数据库 Schema

```sql
-- 活跃告警表
CREATE TABLE alarms_active (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id        UUID NOT NULL REFERENCES devices(id),
    severity         SMALLINT NOT NULL,
    alarm_type       VARCHAR(64),
    alarm_code       VARCHAR(32) NOT NULL,
    description      TEXT,
    status           VARCHAR(16) NOT NULL DEFAULT 'active',
    raised_at        TIMESTAMPTZ NOT NULL,
    acknowledged_at  TIMESTAMPTZ,
    acknowledged_by  VARCHAR(128),
    additional_info  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alarms_active_device ON alarms_active (device_id);
CREATE INDEX idx_alarms_active_severity ON alarms_active (severity);

-- 历史告警超表
CREATE TABLE alarms_history (
    time             TIMESTAMPTZ NOT NULL,
    alarm_id         UUID NOT NULL,
    device_id        UUID,
    severity         SMALLINT,
    alarm_type       VARCHAR(64),
    alarm_code       VARCHAR(32),
    description      TEXT,
    status           VARCHAR(16),
    acknowledged_at  TIMESTAMPTZ,
    cleared_at       TIMESTAMPTZ
);
SELECT create_hypertable('alarms_history', 'time');
```

### 3.2 Redis 数据结构

```
alarm:active:{device_serial} → Hash {alarm_code → 告警 JSON}
```

### 3.3 REST API

```
GET  /api/v1/alarms/active          活跃告警列表
GET  /api/v1/alarms/history         历史告警查询
POST /api/v1/alarms/{id}/acknowledge 确认告警
POST /api/v1/alarms/{id}/clear      手动清除
GET  /api/v1/alarms/statistics      告警统计
```

---

## 4. 详细设计

### 4.1 告警去重规则

相同设备 + 相同告警码 = 重复告警，更新时间戳，不创建新记录。

### 4.2 告警生命周期状态机

```
active → acknowledged → cleared
active → cleared（自动清除）
```

### 4.3 告警严重级别映射

各运营商告警码到统一严重级别的映射通过 `Carrier.AlarmSeverityMapping()` 实现。

---

## 5. 实施子阶段

### 阶段 12a：Receiver + 基础存储（Phase 3）
### 阶段 12b：去重 + 关联 + 生命周期（Phase 3）
### 阶段 12c：转发 + 历史归档（Phase 3/4）

---

## 6. 文件清单

```
internal/alarm/receiver.go
internal/alarm/engine.go
internal/alarm/lifecycle.go
internal/alarm/store.go
internal/alarm/forward.go
internal/alarm/service.go
```

---

## 7. 参考

- backend-design.md 第六章：告警管线
- doc/features/04-alarm-management.md：F04 全部子功能
