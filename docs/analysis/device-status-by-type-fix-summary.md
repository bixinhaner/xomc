# 设备状态分布（按类型）修复总结

> **修复日期**: 2026-05-28
> **问题接口**: `GET /api/v1/dashboard/device-status-by-type`
> **相关文档**: [dashboard-gis-map-devices-fix-20260528.md](./dashboard-gis-map-devices-fix-20260528.md)

## 问题描述

接口 `GET /api/v1/dashboard/device-status-by-type` 返回 500 错误：

```
ERROR: column "status" does not exist (SQLSTATE 42703)
```

## 根本原因

在 migration 000137 中，`devices` 表的 `status` 列被删除，替换为 `lifecycle_state` 和 `is_online` 两个正交字段：

- `lifecycle_state`: 业务流程进度（discovered/registered/provisioning/commissioned/maintenance/decommissioned）
- `is_online`: 实时在线状态（true/false）

但 `dashboard/service.go` 中的查询仍在使用已删除的 `status` 列。

## 修复内容

### 文件: `omcgo/internal/dashboard/service.go`

#### 修复 1: GetDeviceStatusByType 函数（647-662 行）

**修改前**:
```go
query := `
    SELECT
        technology,
        COUNT(*) FILTER (WHERE status = 'active') AS online,
        COUNT(*) FILTER (WHERE status IN ('offline', 'decommissioned')) AS offline,
        COUNT(*) FILTER (WHERE EXISTS (
            SELECT 1 FROM alarms_active aa WHERE aa.device_id = devices.id
        )) AS alarm
    FROM devices
    WHERE deleted_at IS NULL
    GROUP BY technology
    ORDER BY technology`
```

**修改后**:
```go
// T-0162: 使用 is_online 字段（migration 000137 替换了原 status 列）
query := `
    SELECT
        technology,
        COUNT(*) FILTER (WHERE is_online = TRUE) AS online,
        COUNT(*) FILTER (WHERE is_online = FALSE) AS offline,
        COUNT(*) FILTER (WHERE EXISTS (
            SELECT 1 FROM alarms_active aa WHERE aa.device_id = devices.id
        )) AS alarm
    FROM devices
    WHERE deleted_at IS NULL
    GROUP BY technology
    ORDER BY technology`
```

#### 修复 2: GetRegionStats 函数（446-454 行）

**修改前**:
```go
// Count online devices (status = 'active') in this group
Where(sq.Eq{"status": "active"}).
```

**修改后**:
```go
// Count online devices (is_online = TRUE) in this group
// T-0162: 使用 is_online 字段（migration 000137 替换了原 status 列）
Where(sq.Eq{"is_online": true}).
```

## 验证结果

### API 响应

```json
{
  "data": {
    "lte": {"online": 13147, "offline": 12573, "alarm": 0},
    "nr": {"online": 3860, "offline": 4140, "alarm": 0}
  },
  "msg": "ok",
  "ret": 1
}
```

### 日志验证

```
{"level":"info","method":"GET","path":"/api/v1/dashboard/device-status-by-type","status":200,...}
```

## 相关迁移

- **Migration 000137**: `device_lifecycle_online_decouple.sql` - 删除 `status` 列，新增 `lifecycle_state` 和 `is_online`
- **Migration 000159**: `add_devices_other_partition.sql` - 新增国际运营商分区

## 技术说明

### 状态映射关系

| 旧 status | 新 is_online | 新 lifecycle_state |
|-----------|--------------|-------------------|
| active | TRUE | commissioned |
| offline | FALSE | commissioned |
| provisioning | - | provisioning |
| maintenance | - | maintenance |
| decommissioned | - | decommissioned |

### SQL 查询变更

| 场景 | 旧查询 | 新查询 |
|------|--------|--------|
| 在线设备 | `status = 'active'` | `is_online = TRUE` |
| 离线设备 | `status IN ('offline', 'decommissioned')` | `is_online = FALSE` |
| 告警设备 | `EXISTS (SELECT 1 FROM alarms_active...)` | （不变） |

---

**修复状态**: ✅ 已完成并验证
