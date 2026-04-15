# 告警管理模块重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按设计文档 v2.0 重构告警管理模块，新增告警库、增强活动/历史告警字段、新增过滤规则引擎、实现数据权限控制

**Architecture:** 在现有 alarm 模块基础上增强。保留现有 engine/store 双层存储架构，新增 FilterEngine 处理告警过滤，新增 DataPermissionChecker 实现基于设备组的数据权限。API 路径从 `/alarms/rules` 迁移到独立的 `/alarm-libraries`、`/alarm-filters` 路径。

**Tech Stack:** Go 1.25, PostgreSQL 16, TimescaleDB, Squirrel, Gin, NATS JetStream, Zap

---

## 现有代码与设计文档差异对照

| 设计要求 | 现有实现 | 处理方式 |
|---------|---------|---------|
| alarm_libraries 表 | 无 | **新增** |
| alarm_library_i18n 国际化 | 无 | **新增** |
| alarms_active 新字段 | 缺少 device_name, technology, alarm_source, event_type, alarm_type, probable_cause, network_location, description, explicit_cause, is_read, ack_count, first_raised_at, last_updated_at | **ALTER TABLE 增强** |
| alarms_history 新字段 | 缺少 device_name, technology, alarm_source, event_type, alarm_type, network_location, description, explicit_cause, ack_count, cleared_at | **ALTER TABLE 增强** |
| alarm_filters 表 (新) | 有 alarm_rules 表 | **新增，替代 alarm_rules** |
| FilterEngine 过滤引擎 | 无 | **新增** |
| DataPermissionChecker 数据权限 | 无 | **新增** |
| API 路径重构 | /alarms/rules | /alarm-libraries, /alarm-filters | **迁移** |
| 批量操作 API | 无 | **新增** batch/acknowledge, batch/clear |
| 标记已读 API | 无 | **新增** /active/:id/read |
| 导出 API | 无 | **新增** export |

## 文件结构

### 新建文件
| 文件 | 职责 |
|------|------|
| `omcgo/migrations/000013_alarm_libraries.up.sql` | 告警库 + 国际化表 + 过滤规则表 + 活动告警增强 + 历史告警增强 |
| `omcgo/internal/alarm/library_model.go` | 告警库模型定义 |
| `omcgo/internal/alarm/library_repository.go` | 告警库仓储接口 |
| `omcgo/internal/alarm/pg_library_repository.go` | 告警库 PostgreSQL 实现 |
| `omcgo/internal/alarm/library_handler.go` | 告警库 HTTP 处理器 |
| `omcgo/internal/alarm/library_service.go` | 告警库业务逻辑 |
| `omcgo/internal/alarm/filter_model.go` | 过滤规则模型定义 |
| `omcgo/internal/alarm/filter_repository.go` | 过滤规则仓储接口 |
| `omcgo/internal/alarm/pg_filter_repository.go` | 过滤规则 PostgreSQL 实现 |
| `omc/go/internal/alarm/filter_engine.go` | 告警过滤引擎 |
| `omcgo/internal/alarm/data_permission.go` | 数据权限控制 |
| `omcgo/internal/alarm/library_handler_test.go` | 告警库测试 |
| `omcgo/internal/alarm/filter_engine_test.go` | 过滤引擎测试 |
| `omcgo/internal/alarm/data_permission_test.go` | 数据权限测试 |

### 修改文件
| 文件 | 修改内容 |
|------|---------|
| `omcgo/internal/core/model/alarm.go` | Alarm 结构体新增字段 |
| `omcgo/internal/alarm/store.go` | AlarmStore 接口新增方法签名 |
| `omcgo/internal/alarm/pg_store.go` | 适配新字段、新增方法实现 |
| `omcgo/internal/alarm/engine.go` | 集成 FilterEngine |
| `omcgo/internal/alarm/handler.go` | 新增批量操作、标记已读、导出 API |
| `omcgo/internal/alarm/receiver.go` | 适配增强后的 Alarm 结构体 |
| `omcgo/internal/alarm/metrics.go` | 新增过滤引擎指标 |
| `omcgo/cmd/app/provider/router.go` | 注册新路由组 |
| `omcgo/cmd/app/provider/deps.go` | 新增依赖注入 |

### 删除文件
| 文件 | 原因 |
|------|------|
| `omcgo/internal/alarm/rule_model.go` | 被 filter_model.go 替代 |
| `omcgo/internal/alarm/rule_handler.go` | 被 filter_handler.go 替代 |
| `omcgo/internal/alarm/rule_repository.go` | 被 filter_repository.go 替代 |
| `omcgo/internal/alarm/pg_rule_repository.go` | 被 pg_filter_repository.go 替代 |
| `omcgo/internal/alarm/rule_handler_test.go` | 被 filter_handler_test.go 替代 |

---

## Task 1: 数据库迁移

**Files:**
- Create: `omcgo/migrations/000013_alarm_management_enhancement.up.sql`
- Create: `omcgo/migrations/000013_alarm_management_enhancement.down.sql`

- [ ] **Step 1: 编写迁移文件 (UP)**

```sql
-- +goose Up
-- ============================================================
-- 000013_alarm_management_enhancement.up.sql
-- 告警管理模块增强：告警库、过滤规则、字段增强
-- ============================================================

-- 1. 告警库表
CREATE TABLE alarm_libraries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alarm_code      VARCHAR(128) NOT NULL UNIQUE,
    alarm_source    VARCHAR(64) NOT NULL,
    event_type      VARCHAR(64) NOT NULL,
    severity        SMALLINT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    probable_cause  TEXT NOT NULL,
    explanation     TEXT,
    additional_info JSONB DEFAULT '{}',
    carrier         VARCHAR(4),
    technology      VARCHAR(16),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alarm_libraries_source ON alarm_libraries(alarm_source);
CREATE INDEX idx_alarm_libraries_code ON alarm_libraries(alarm_code);
CREATE INDEX idx_alarm_libraries_severity ON alarm_libraries(severity);
CREATE INDEX idx_alarm_libraries_carrier ON alarm_libraries(carrier) WHERE carrier IS NOT NULL;

-- 2. 告警库国际化表
CREATE TABLE alarm_library_i18n (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    library_id      UUID NOT NULL REFERENCES alarm_libraries(id) ON DELETE CASCADE,
    locale          VARCHAR(16) NOT NULL,
    probable_cause  TEXT NOT NULL,
    explanation     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(library_id, locale)
);

CREATE INDEX idx_alarm_library_i18n_library ON alarm_library_i18n(library_id);
CREATE INDEX idx_alarm_library_i18n_locale ON alarm_library_i18n(locale);

-- 3. 活动告警表增强
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS device_name VARCHAR(128);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS technology VARCHAR(16);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS alarm_source VARCHAR(64);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS event_type VARCHAR(64);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS alarm_type VARCHAR(32);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS probable_cause TEXT NOT NULL DEFAULT '';
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS network_location TEXT;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS explicit_cause TEXT;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS ack_count INT NOT NULL DEFAULT 0;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS first_raised_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_alarms_active_device_name ON alarms_active(device_name);
CREATE INDEX IF NOT EXISTS idx_alarms_active_is_read ON alarms_active(is_read);
CREATE INDEX IF NOT EXISTS idx_alarms_active_alarm_type ON alarms_active(alarm_type);

-- 4. 历史告警表增强
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS device_name VARCHAR(128);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS technology VARCHAR(16);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS alarm_source VARCHAR(64);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS event_type VARCHAR(64);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS alarm_type VARCHAR(32);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS network_location TEXT;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS explicit_cause TEXT;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS ack_count INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_alarms_history_alarm_type ON alarms_history(alarm_type);

-- 5. 告警过滤规则表
CREATE TABLE alarm_filters (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL,
    filter_type         VARCHAR(32) NOT NULL,
    alarm_sources       VARCHAR(64)[] DEFAULT '{}',
    alarm_codes         VARCHAR(128)[] DEFAULT '{}',
    device_ids          UUID[] DEFAULT '{}',
    device_group_ids    UUID[] DEFAULT '{}',
    action              VARCHAR(32) NOT NULL,
    acknowledge_desc    TEXT,
    priority            INT NOT NULL DEFAULT 0,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_by          VARCHAR(128),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(128),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alarm_filters_enabled ON alarm_filters(enabled, priority);
CREATE INDEX idx_alarm_filters_action ON alarm_filters(action);
CREATE INDEX idx_alarm_filters_sources ON alarm_filters USING GIN(alarm_sources);
CREATE INDEX idx_alarm_filters_codes ON alarm_filters USING GIN(alarm_codes);
CREATE INDEX idx_alarm_filters_devices ON alarm_filters USING GIN(device_ids);
CREATE INDEX idx_alarm_filters_groups ON alarm_filters USING GIN(device_group_ids);

-- 6. 种子数据：标准告警库
INSERT INTO alarm_libraries (alarm_code, alarm_source, event_type, severity, probable_cause, explanation) VALUES
    ('DEVICE_OFFLINE', 'Device', 'communications', 2, '设备离线', '设备与网管系统之间的连接中断'),
    ('DEVICE_RESTART', 'Device', 'equipment', 3, '设备重启', '设备已重新启动'),
    ('LINK_FAILURE', 'Device', 'communications', 2, '链路中断', '基站回传链路异常中断'),
    ('S1_INTERFACE_ERROR', 'TR069', 'communications', 2, 'S1接口异常', 'TR069 ACS与CPE的S1接口通信异常'),
    ('CPU_OVERLOAD', 'Device', 'processing', 3, 'CPU过载', '设备CPU使用率超过阈值'),
    ('MEMORY_OVERLOAD', 'Device', 'processing', 3, '内存过载', '设备内存使用率超过阈值'),
    ('TEMP_HIGH', 'Device', 'environment', 3, '温度过高', '设备温度超过告警阈值'),
    ('POWER_FAILURE', 'Device', 'power', 1, '电源故障', '设备供电中断或电源模块故障'),
    ('FIRMWARE_UPGRADE_FAILED', 'Device', 'software', 2, '固件升级失败', '设备固件升级过程中发生错误'),
    ('GPS_LOSS', 'Device', 'environment', 4, 'GPS信号丢失', '设备GPS模块无法获取卫星信号'),
    ('VSWR_MISMATCH', 'Device', 'quality', 3, 'VSWR不匹配', 'VSWR值超出正常范围'),
    ('RSRP_LOW', 'Device', 'quality', 4, 'RSRP偏低', '参考信号接收功率低于阈值'),
    ('SINR_LOW', 'Device', 'quality', 4, 'SINR偏低', '信号与干扰噪声比低于阈值'),
    ('PCI_CONFLICT', 'Device', 'quality', 3, 'PCI冲突', '物理小区标识冲突'),
    ('NEIGHBOR_CELL', 'Device', 'configuration', 4, '邻区检测', '检测到未配置的邻区关系')
ON CONFLICT (alarm_code) DO NOTHING;

-- 英文国际化
INSERT INTO alarm_library_i18n (library_id, locale, probable_cause, explanation)
SELECT al.id, 'en-US',
    CASE al.alarm_code
        WHEN 'DEVICE_OFFLINE' THEN 'Device Offline'
        WHEN 'DEVICE_RESTART' THEN 'Device Restart'
        WHEN 'LINK_FAILURE' THEN 'Link Failure'
        WHEN 'S1_INTERFACE_ERROR' THEN 'S1 Interface Error'
        WHEN 'CPU_OVERLOAD' THEN 'CPU Overload'
        WHEN 'MEMORY_OVERLOAD' THEN 'Memory Overload'
        WHEN 'TEMP_HIGH' THEN 'High Temperature'
        WHEN 'POWER_FAILURE' THEN 'Power Failure'
        WHEN 'FIRMWARE_UPGRADE_FAILED' THEN 'Firmware Upgrade Failed'
        WHEN 'GPS_LOSS' THEN 'GPS Signal Loss'
        WHEN 'VSWR_MISMATCH' THEN 'VSWR Mismatch'
        WHEN 'RSRP_LOW' THEN 'Low RSRP'
        WHEN 'SINR_LOW' THEN 'Low SINR'
        WHEN 'PCI_CONFLICT' THEN 'PCI Conflict'
        WHEN 'NEIGHBOR_CELL' THEN 'Neighbor Cell Detected'
    END,
    CASE al.alarm_code
        WHEN 'DEVICE_OFFLINE' THEN 'Connection between device and NMS is interrupted'
        WHEN 'DEVICE_RESTART' THEN 'Device has been restarted'
        WHEN 'LINK_FAILURE' THEN 'Backhaul link is abnormally interrupted'
        WHEN 'S1_INTERFACE_ERROR' THEN 'TR069 S1 interface communication error between ACS and CPE'
        WHEN 'CPU_OVERLOAD' THEN 'Device CPU usage exceeds threshold'
        WHEN 'MEMORY_OVERLOAD' THEN 'Device memory usage exceeds threshold'
        WHEN 'TEMP_HIGH' THEN 'Device temperature exceeds alarm threshold'
        WHEN 'POWER_FAILURE' THEN 'Device power supply interrupted or power module failure'
        WHEN 'FIRMWARE_UPGRADE_FAILED' THEN 'Error occurred during device firmware upgrade'
        WHEN 'GPS_LOSS' THEN 'GPS module cannot acquire satellite signal'
        WHEN 'VSWR_MISMATCH' THEN 'VSWR value is out of normal range'
        WHEN 'RSRP_LOW' THEN 'Reference Signal Received Power is below threshold'
        WHEN 'SINR_LOW' THEN 'Signal to Interference plus Noise Ratio is below threshold'
        WHEN 'PCI_CONFLICT' THEN 'Physical Cell Identifier conflict detected'
        WHEN 'NEIGHBOR_CELL' THEN 'Unconfigured neighbor cell relation detected'
    END
FROM alarm_libraries al
ON CONFLICT (library_id, locale) DO NOTHING;
```

- [ ] **Step 2: 编写迁移文件 (DOWN)**

```sql
-- +goose Down
DROP TABLE IF EXISTS alarm_filters CASCADE;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS ack_count;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS explicit_cause;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS description;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS network_location;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS alarm_type;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS event_type;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS alarm_source;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS technology;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS device_name;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS last_updated_at;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS first_raised_at;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS ack_count;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS is_read;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS explicit_cause;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS description;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS network_location;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS alarm_type;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS event_type;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS alarm_source;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS technology;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS device_name;
DROP TABLE IF EXISTS alarm_library_i18n CASCADE;
DROP TABLE IF EXISTS alarm_libraries CASCADE;
```

- [ ] **Step 3: 验证迁移语法**

Run: `cd /home/baicells/goomc/omcgo && docker exec docker-postgres-1 psql -U omcgo -c "\echo 'BEGIN; ROLLBACK;' -f /dev/stdin <<'EOF'
-- 粘贴 Step 1 中的完整 UP SQL
EOF"`

Expected: 无语法错误

- [ ] **Step 4: 执行迁移**

Run: `cd /home/baicells/goomc/omcgo && docker cp ../migrations/000013_alarm_management_enhancement.up.sql docker-migrate-schema-1:/etc/omcgo/migrations/ && docker exec docker-migrate-schema-1 /usr/local/bin/omcgo-migrate --dsn "postgres://omcgo:omcgo123@postgres:5432/omcgo?sslmode=disable" --path /etc/omcgo/migrations up`

Expected: OK 000013_alarm_management_enhancement.up.sql

- [ ] **Step 5: 验证数据**

Run: `docker exec docker-postgres-1 psql -U omcgo -c "SELECT count(*) FROM alarm_libraries; SELECT count(*) FROM alarm_library_i18n; SELECT column_name FROM information_schema.columns WHERE table_name='alarms_active' AND column_name='device_name';"`

Expected: alarm_libraries=16, alarm_library_i18n=16, device_name 列存在

- [ ] **Step 6: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/migrations/000013_alarm_management_enhancement.up.sql omcgo/migrations/000013_alarm_management_enhancement.down.sql
git commit -m "feat(alarm): 新增告警库、过滤规则表及活动/历史告警字段增强

Task 1: 数据库迁移
- 新增 alarm_libraries + alarm_library_i18n 表
- 新增 alarm_filters 表
- ALTER alarms_active/alarms_history 增加新字段
- 种子数据：16 条标准告警 + 英文国际化"
```

---

## Task 2: 增强核心模型

**Files:**
- Modify: `omcgo/internal/core/model/alarm.go`
- Create: `omcgo/internal/alarm/library_model.go`
- Create: `omcgo/internal/alarm/filter_model.go`

- [ ] **Step 1: 增强 Alarm 模型**

修改 `omcgo/internal/core/model/alarm.go`，在 Alarm 结构体中新增字段。在 `ClearedAt` 字段后添加：

```go
	// 增强字段
	DeviceName       string            `json:"device_name,omitempty" db:"device_name"`
	Technology      string            `json:"technology,omitempty" db:"technology"`
	AlarmSource     string            `json:"alarm_source,omitempty" db:"alarm_source"`
	EventType       string            `json:"event_type,omitempty" db:"event_type"`
	NetworkLocation string            `json:"network_location,omitempty" db:"network_location"`
	Description     string            `json:"description,omitempty" db:"description"`
	ExplicitCause    string            `json:"explicit_cause,omitempty" db:"explicit_cause"`
	IsRead          bool              `json:"is_read" db:"is_read"`
	AckCount        int               `json:"ack_count" db:"ack_count"`
	FirstRaisedAt   time.Time         `json:"first_raised_at,omitempty" db:"first_raised_at"`
	LastUpdatedAt   time.Time         `json:"last_updated_at,omitempty" db:"last_updated_at"`
```

注意：`AlarmType` 字段已存在于现有模型中，无需新增。`ProbableCause` 字段如果不存在也需添加。

- [ ] **Step 2: 创建告警库模型**

创建 `omcgo/internal/alarm/library_model.go`，包含 AlarmLibrary、AlarmLibraryI18n、AlarmLibraryFilter、CreateAlarmLibraryRequest、UpdateAlarmLibraryRequest 结构体。代码见设计文档第三章 3.1 节。

- [ ] **Step 3: 创建过滤规则模型**

创建 `omcgo/internal/alarm/filter_model.go`，包含 AlarmFilter、CreateAlarmFilterRequest、UpdateAlarmFilterRequest 结构体及常量定义。代码见设计文档第三章 3.3 节。

- [ ] **Step 4: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

Expected: 编译通过

- [ ] **Step 5: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/core/model/alarm.go omcgo/internal/alarm/library_model.go omcgo/internal/alarm/filter_model.go
git commit -m "feat(alarm): 新增告警库模型、过滤规则模型及Alarm结构体增强字段"
```

---

## Task 3: 告警库仓储层

**Files:**
- Create: `omcgo/internal/alarm/library_repository.go`
- Create: `omcgo/internal/alarm/pg_library_repository.go`

- [ ] **Step 1: 定义告警库仓储接口**

创建 `omcgo/internal/alarm/library_repository.go`：

```go
package alarm

import (
	"context"
	"github.com/google/uuid"
)

type AlarmLibraryRepository interface {
	Create(ctx context.Context, lib *AlarmLibrary) error
	GetByID(ctx context.Context, id uuid.UUID) (*AlarmLibrary, error)
	GetByCode(ctx context.Context, code string) (*AlarmLibrary, error)
	Update(ctx context.Context, lib *AlarmLibrary) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter AlarmLibraryFilter) (*model.ListResponse[AlarmLibrary], error)
	CreateI18n(ctx context.Context, i18n *AlarmLibraryI18n) error
	UpdateI18n(ctx context.Context, i18n *AlarmLibraryI18n) error
	DeleteI18n(ctx context.Context, id uuid.UUID) error
	ListI18n(ctx context.Context, libraryID uuid.UUID) ([]AlarmLibraryI18n, error)
}
```

- [ ] **Step 2: 实现 PostgreSQL 仓储**

创建 `omcgo/internal/alarm/pg_library_repository.go`，使用 Squirrel 实现 CRUD。参考现有 `pg_rule_repository.go` 的模式。

- [ ] **Step 3: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/library_repository.go omcgo/internal/alarm/pg_library_repository.go
git commit -m "feat(alarm): 实现告警库仓储接口及PostgreSQL实现"
```

---

## Task 4: 过滤规则仓储层

**Files:**
- Create: `omcgo/internal/alarm/filter_repository.go`
- Create: `omcgo/internal/alarm/pg_filter_repository.go`

- [ ] **Step 1: 定义过滤规则仓储接口**

创建 `omcgo/internal/alarm/filter_repository.go`：

```go
package alarm

import (
	"context"
	"github.com/google/uuid"
)

type AlarmFilterRepository interface {
	Create(ctx context.Context, filter *AlarmFilter) error
	GetByID(ctx context.Context, id uuid.UUID) (*AlarmFilter, error)
	Update(ctx context.Context, filter *AlarmFilter) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter AlarmFilterFilter) (*model.ListResponse[AlarmFilter], error)
	Toggle(ctx context.Context, id uuid.UUID) error
	ListEnabled(ctx context.Context) ([]AlarmFilter, error)
	GetDeviceGroups(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error)
}
```

- [ ] **Step 2: 实现 PostgreSQL 仓储**

创建 `omcgo/internal/alarm/pg_filter_repository.go`。ListEnabled 需按 priority ASC 排序。GetDeviceGroups 需 JOIN device_group_members 和 device_group_members 获取设备所属的设备组 ID 列表。

- [ ] **Step 3: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/filter_repository.go omcgo/internal/alarm/pg_filter_repository.go
git commit -m "feat(alarm): 实现过滤规则仓储接口及PostgreSQL实现"
```

---

## Task 5: 增强 Store 接口和实现

**Files:**
- Modify: `omcgo/internal/alarm/store.go`
- Modify: `omcgo/internal/alarm/pg_store.go`

- [ ] **Step 1: 扩展 AlarmStore 接口**

在 `store.go` 的 AlarmStore 接口中新增方法：

```go
	// 新增方法
	BatchAcknowledge(ctx context.Context, ids []uuid.UUID, desc string) error
	BatchClear(ctx context.Context, ids []uuid.UUID) error
	MarkRead(ctx context.Context, id uuid.UUID) error
	GetActiveByDeviceAndCode(ctx context.Context, deviceSN string, alarmCode string) (*model.Alarm, error)
```

注意：`GetActiveByDeviceAndCode` 可能已存在，检查后再决定是否新增。

- [ ] **Step 2: 修改 AlarmFilter 结构体**

在 `store.go` 的 AlarmFilter 中新增字段：

```go
type AlarmFilter struct {
	// 现有字段...
	// 新增过滤字段
	AlarmCodes     []string           `form:"alarm_codes"`
	AlarmSources   []string           `form:"alarm_sources"`
	AlarmType      *string           `form:"alarm_type"`
	IsRead         *bool              `form:"is_read"`
	DeviceName     *string           `form:"device_name"`
	StartTime     *time.Time         `form:"start_time"`
	EndTime       *time.Time         `form:"end_time"`
	// 数据权限
	DeviceIDs      []uuid.UUID        `form:"-"`
	Technologies   []string           `form:"-"`
	Carrier        *model.CarrierCode `form:"-"`
}
```

- [ ] **Step 3: 修改 pg_store.go 适配新字段**

更新 `alarmColumns` 列表，修改 SELECT/INSERT/UPDATE 语句以包含新字段。新增 BatchAcknowledge、BatchClear、MarkRead 方法实现。

- [ ] **Step 4: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 5: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/store.go omcgo/internal/alarm/pg_store.go
git commit -m "feat(alarm): 增强AlarmStore接口和实现，新增批量操作和标记已读"
```

---

## Task 6: 告警过滤引擎

**Files:**
- Create: `omcgo/internal/alarm/filter_engine.go`
- Create: `omcgo/internal/alarm/filter_engine_test.go`

- [ ] **Step 1: 实现过滤引擎**

创建 `omcgo/internal/alarm/filter_engine.go`，包含 FilterEngine 结构体和 ProcessAlarm、Match、executeFilterAction、processAutoAcknowledge、processAutoClear、processDefault、enrichFromLibrary 方法。代码见设计文档第五章 5.3 节。

- [ ] **Step 2: 编写过滤引擎单元测试**

创建 `omcgo/internal/alarm/filter_engine_test.go`，测试用例：
- TestMatch_IgnoreAction — 匹配 ignore 规则返回不处理
- TestMatch_AutoAcknowledge — 匹配自动确认规则
- TestMatch_AutoClear — 匹配自动清除规则
- TestMatch_NoMatch — 无匹配走默认流程
- TestProcessAlarm_NewAlarm_Default — 默认流程新建活动告警
- TestProcessAlarm_NewAlarm_AutoAck — 自动确认流程
- TestProcessAlarm_ExistingAlarm_Default — 默认流程更新 count
- TestProcessAlarm_ExistingAlarm_AutoClear — 自动清除流程
- TestMatchFilter_DeviceType — 设备类型过滤匹配
- TestMatchFilter_DeviceGroup — 设备组过滤匹配
- TestEnrichFromLibrary — 告警库补充信息

- [ ] **Step 3: 运行测试**

Run: `cd /home/baicells/goomc/omcgo && go test ./internal/alarm/ -run TestMatch -v && go test ./internal/alarm/ -run TestProcessAlarm -v && go test ./internal/alarm/ -run TestEnrichFromLibrary -v`

Expected: 全部 PASS

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/filter_engine.go omcgo/internal/alarm/filter_engine_test.go
git commit -m "feat(alarm): 实现告警过滤引擎，支持ignore/auto_acknowledge/auto_clear三种过滤动作"
```

---

## Task 7: 数据权限控制

**Files:**
- Create: `omcgo/internal/alarm/data_permission.go`
- Create: `omcgo/internal/alarm/data_permission_test.go`

- [ ] **Step 1: 实现数据权限检查器**

创建 `omcgo/internal/alarm/data_permission.go`，包含 DataPermissionChecker 结构体和 ApplyDataPermission 方法。代码见设计文档第五章 5.4 节。

- [ ] **Step 2: 编写数据权限单元测试**

创建 `omcgo/internal/alarm/data_permission_test.go`，测试用例：
- TestApplyDataPermission_SuperAdmin — 超管不限制
- TestApplyDataPermission_NormalUser — 普通用户限制设备组
- TestApplyDataPermission_WithCarrier — 运营商过滤

- [ ] **Step 3: 运行测试**

Run: `cd /home/baicells/goomc/omcgo && go test ./internal/alarm/ -run TestDataPermission -v`

Expected: 全部 PASS

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/data_permission.go omcgo/internal/alarm/data_permission_test.go
git commit -m "feat(alarm): 实现告警数据权限控制，基于角色设备组和运营商过滤"
```

---

## Task 8: 告警库 API Handler

**Files:**
- Create: `omcgo/internal/alarm/library_handler.go`
- Create: `omcgo/internal/alarm/library_service.go`

- [ ] **Step 1: 实现告警库 Service**

创建 `omcgo/internal/alarm/library_service.go`，封装业务逻辑：Create/GetByID/List/Update/Delete/CreateI18n。

- [ ] **Step 2: 实现告警库 Handler**

创建 `omcgo/internal/alarm/library_handler.go`，实现 HTTP 处理器。API 路径：
- `GET /api/v1/alarm-libraries` — 列表查询
- `GET /api/v1/alarm-libraries/:id` — 获取详情
- `POST /api/v1/alarm-libraries` — 创建
- `PUT /api/v1/alarm-libraries/:id` — 更新
- `DELETE /api/v1/alarm-libraries/:id` — 删除
- `GET /api/v1/alarm-libraries/export` — 导出

- [ ] **Step 3: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/library_handler.go omcgo/internal/alarm/library_service.go
git commit -m "feat(alarm): 实现告警库API Handler和Service层"
```

---

## Task 9: 过滤规则 API Handler

**Files:**
- Create: `omcgo/internal/alarm/filter_handler.go`

- [ ] **Step 1: 实现过滤规则 Handler**

创建 `omcgo/internal/alarm/filter_handler.go`，实现 HTTP 处理器。API 路径：
- `GET /api/v1/alarm-filters` — 列表查询
- `GET /api/v1/alarm-filters/:id` — 获取详情
- `POST /api/v1/alarm-filters` — 创建
- `PUT /api/v1/alarm-filters/:id` — 更新
- `DELETE /api/v1/alarm-filters/:id` — 删除
- `POST /api/v1/alarm-filters/:id/toggle` — 启用/禁用

- [ ] **Step 2: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 3: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/filter_handler.go
git commit -m "feat(alarm): 实现告警过滤规则API Handler"
```

---

## Task 10: 增强 Handler 和集成 FilterEngine

**Files:**
- Modify: `omcgo/internal/alarm/handler.go`
- Modify: `omcgo/internal/alarm/engine.go`

- [ ] **Step 1: 增强 Handler 新增 API**

修改 `handler.go`，新增以下端点：
- `POST /api/v1/alarms/active/batch/acknowledge` — 批量确认
- `POST /api/v1/alarms/active/batch/clear` — 批量清除
- `POST /api/v1/alarms/active/:id/read` — 标记已读
- `GET /api/v1/alarms/active/export` — 导出活动告警
- `GET /api/v1/alarms/history/export` — 导出历史告警
- `GET /api/v1/alarms/active/statistics` — 告警统计（增强版）

- [ ] **Step 2: 集成 FilterEngine 到 Engine**

修改 `engine.go`，在 Process 方法中：
1. 初始化 FilterEngine
2. 调用 FilterEngine.ProcessAlarm 替代原有逻辑
3. 保留现有的运营商严重级映射、事件发布等逻辑

- [ ] **Step 3: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/handler.go omcgo/internal/alarm/engine.go
git commit -m "feat(alarm): 增强Handler新增批量操作/标记已读/导出API，集成FilterEngine到告警引擎"
```

---

## Task 11: 路由注册和依赖注入

**Files:**
- Modify: `omcgo/cmd/app/provider/router.go`
- Modify: `omcgo/cmd/app/provider/deps.go`

- [ ] **Step 1: 新增依赖注入**

修改 `deps.go`，在 AlarmDeps 或新建 AlarmEnhancedDeps 中添加：
- LibraryRepository
- FilterRepository
- FilterEngine (通过构造函数创建)
- DataPermissionChecker

- [ ] **Step 2: 注册新路由**

修改 `router.go`，注册新路由组：
```go
// 告警库路由
alarmLibraryHandler := alarm.NewLibraryHandler(libraryService, c.Logger)
alarmLibraryHandler.RegisterRoutes(permGroup("alarms").Group("/alarm-libraries"))

// 告警过滤规则路由
alarmFilterHandler := alarm.NewFilterHandler(filterRepo, c.Logger)
alarmFilterHandler.RegisterRoutes(permGroup("alarms").Group("/alarm-filters"))
```

同时保留现有的 `/alarms/active`、`/alarms/history` 路由。

- [ ] **Step 3: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 4: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/cmd/app/provider/router.go omcgo/cmd/app/provider/deps.go
git commit -m "feat(alarm): 注册告警库和过滤规则路由，新增依赖注入"
```

---

## Task 12: 清理旧代码和集成测试

**Files:**
- Delete: `omcgo/internal/alarm/rule_model.go`
- Delete: `omcgo/internal/alarm/rule_handler.go`
- Delete: `omcgo/internal/alarm/rule_repository.go`
- Delete: `omcgo/internal/alarm/pg_rule_repository.go`
- Delete: `omcgo/internal/alarm/rule_handler_test.go`
- Modify: `omcgo/internal/alarm/receiver.go`

- [ ] **Step 1: 删除旧规则文件**

```bash
cd /home/baicells/goomc/omcgo && git rm internal/alarm/rule_model.go internal/alarm/rule_handler.go internal/alarm/rule_repository.go internal/alarm/pg_rule_repository.go internal/alarm/rule_handler_test.go
```

- [ ] **Step 2: 更新 receiver.go 适配新模型**

修改 receiver.go，确保 AlarmPayload 到 Alarm 的转换填充新字段（device_name, technology, alarm_source, event_type, alarm_type, probable_cause）。

- [ ] **Step 3: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`

- [ ] **Step 4: 运行全部测试**

Run: `cd /home/baicells/goomc/omcgo && go test ./internal/alarm/ -v`

Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
cd /home/baicells/goomc && git add omcgo/internal/alarm/receiver.go
git commit -m "refactor(alarm): 删除旧告警规则文件，适配receiver到增强模型"
```

---

## Task 13: 构建 Docker 镜像并验证

**Files:** 无源码修改

- [ ] **Step 1: 重建 Docker 镜像**

Run: `cd /home/baicells/goomc/deployments/docker && docker compose build --no-cache app acs worker 2>&1 | tail -20`

- [ ] **Step 2: 重启服务**

Run: `cd /home/baicells/goomc/deployments/docker && docker compose down && docker compose up -d 2>&1 | tail -20`

- [ ] **Step 3: 验证迁移执行**

Run: `docker logs docker-migrate-schema-1 2>&1 | grep 000013`

Expected: OK 000013

- [ ] **Step 4: 验证新 API 端点**

Run:
```bash
# 告警库
curl -s http://172.21.170.33:8081/api/v1/alarm-libraries -H "Authorization: Bearer <token>" | head -50

# 过滤规则
curl -s http://172.21.170.33:8081/api/v1/alarm-filters -H "Authorization: Bearer <token>" | head -50

# 活动告警增强字段
curl -s http://172.21.170.33:8081/api/v1/alarms/active -H "Authorization: Bearer <token>" | head -50
```

Expected: API 正常返回，包含新字段

- [ ] **Step 5: 最终提交**

```bash
cd /home/baicells/goomc && git add -A && git status
git commit -m "chore(alarm): 验证告警管理模块重构，Docker构建和API验证通过"
```

---

## 开发时间线

| 阶段 | 任务 | 依赖 |
|------|------|------|
| Phase 1 | Task 1-2: 数据库迁移 + 模型定义 | 无 |
| Phase 2 | Task 3-4: 告警库 + 过滤规则仓储 | Task 2 |
| Phase 3 | Task 5: Store 增强 | Task 2 |
| Phase 4 | Task 6-7: 过滤引擎 + 数据权限 | Task 3, 4, 5 |
| Phase 5 | Task 8-9: Handler 层 | Task 3, 4, 6, 7 |
| Phase 6 | Task 10-11: 集成 + 路由 | Task 6, 7, 8, 9 |
| Phase 7 | Task 12-13: 清理 + 验证 | Task 11 |
