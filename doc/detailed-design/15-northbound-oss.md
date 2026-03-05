# DD-15: 北向/OSS 接口（F08）

> 关联功能域：F08（北向/OSS 接口）
> 关联 backend-design.md 章节：第十一章（API 设计）
> 实施阶段：Phase 4（北向与规模化）
> 依赖文档：DD-11, DD-12, DD-08

---

## 1. 概述

### 1.1 模块定位

北向/OSS 接口（`internal/northbound/`）是 OMC 向上层运维支撑系统（OSS）开放的数据接口，负责 PM 数据、告警数据和配置数据的外部暴露。

### 1.2 核心职责

- 性能数据接口（PM 统计推送、文件导出）
- 配置数据接口（设备配置快照、变更推送）
- 告警数据接口（实时推送、全量/增量同步）
- 数据推送引擎（HTTP 回调、文件推送）
- 全量/增量同步设计

---

## 2. 接口设计

### 2.1 Northbound API — `internal/northbound/api/`

```go
type NorthboundRouter struct {
    pmHandler     *PMHandler
    alarmHandler  *AlarmHandler
    configHandler *ConfigHandler
}

func (r *NorthboundRouter) RegisterRoutes(engine *gin.Engine)
```

### 2.2 数据推送引擎 — `internal/northbound/push/`

```go
type PushEngine struct {
    targets  []PushTarget
    eventBus event.EventBus
    logger   *zap.Logger
}

type PushTarget struct {
    URL        string
    AuthType   string // "api_key", "basic", "tls_mutual"
    DataTypes  []string // "alarm", "pm", "config"
    Format     string // "json", "xml"
    BatchSize  int
    RetryCount int
}

// PushAlarm 推送告警到 OSS
func (e *PushEngine) PushAlarm(ctx context.Context, alarm *model.Alarm) error
// PushPMData 推送 PM 数据
func (e *PushEngine) PushPMData(ctx context.Context, data []model.KPIValue) error
```

### 2.3 同步服务 — `internal/northbound/sync/`

```go
type SyncService struct {
    deviceRepo  device.DeviceRepository
    alarmStore  alarm.AlarmStore
    pmRepo      pm.CounterRepository
}

// FullSync 全量同步
func (s *SyncService) FullSync(ctx context.Context, dataType string) (io.Reader, error)
// IncrementalSync 增量同步（基于时间戳）
func (s *SyncService) IncrementalSync(ctx context.Context, dataType string, since time.Time) (io.Reader, error)
```

---

## 3. REST API

```
# 北向数据查询（复用主 API）
GET  /api/v1/devices              设备列表
GET  /api/v1/pm/kpi               KPI 数据
GET  /api/v1/alarms/active        活跃告警

# 北向专用接口
POST /api/v1/northbound/push/configure   配置推送目标
GET  /api/v1/northbound/sync/full        全量同步
GET  /api/v1/northbound/sync/incremental 增量同步
POST /api/v1/northbound/export/pm        导出 PM 数据
POST /api/v1/northbound/export/alarms    导出告警数据
```

---

## 4. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 北向 OSS 独立规范 | ❌ | ❌ | ✅ (V1.0) |
| 性能配置接口 | — | — | 技术规范 + 测试规范 |
| 告警接口 | — | — | 技术规范 + 测试规范 |

联通规范作为基线实现，移动/电信可按需对齐。

---

## 5. 实施子阶段

### 阶段 15a：REST API 骨架（Phase 4）
### 阶段 15b：告警推送（Phase 4）
### 阶段 15c：PM 数据导出 + 配置快照（Phase 4）
### 阶段 15d：全量/增量同步（Phase 4）

---

## 6. 文件清单

```
internal/northbound/api/router.go
internal/northbound/api/pm_handler.go
internal/northbound/api/alarm_handler.go
internal/northbound/api/config_handler.go
internal/northbound/push/engine.go
internal/northbound/push/targets.go
internal/northbound/sync/sync.go
```

---

## 7. 参考

- backend-design.md 第十一章：API 设计（11.1 北向 REST API）
- doc/features/08-northbound-oss.md：F08 全部子功能
