# DD-21: 数据库迁移策略与 Schema 演进

> 关联功能域：全部
> 关联 backend-design.md 章节：第五章（数据存储策略）
> 实施阶段：Phase 1 ~ Phase 4（���穿全部阶段）
> 依赖文档：DD-02

---

## 1. 概述

### 1.1 模块定位

数据库迁移（`migrations/` + `cmd/migrate/`）管理所有 PostgreSQL 和 TimescaleDB 的 Schema 版本演进，使用 golang-migrate 工具。

### 1.2 核心职责

- 迁移文件版本管理（严格递增编号）
- up/down 对称迁移（可回滚）
- 按实施阶段分批创建迁移
- 种子数据加载策略

---

## 2. 详细设计

### 2.1 迁移策略

- 迁移工具：`github.com/golang-migrate/migrate/v4`
- 文件格式：`{sequence}_{description}.{up|down}.sql`
- 序号规则：三位数字，001 起始
- 每个迁移文件必须有 up 和 down 对

### 2.2 cmd/migrate 入口

```go
// 子命令：
// omcgo-migrate up       — 执行所有未应用的迁移
// omcgo-migrate down     — 回滚最后一次迁移
// omcgo-migrate version  — 查看当前版本
// omcgo-migrate force N  — 强制设置版本号（修复错误迁移）
```

### 2.3 迁移文件清单

#### Phase 1 — 基础表

```sql
-- 001_create_devices.up.sql
CREATE TABLE devices (...) PARTITION BY LIST (carrier);
CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');
CREATE INDEX idx_devices_status ON devices (carrier, technology, status);
CREATE INDEX idx_devices_last_inform ON devices (last_inform_at);

-- 002_create_device_parameters.up.sql
CREATE TABLE device_parameters (...);
```

#### Phase 2 — 数据模型与配置表

```sql
-- 003_create_data_model_definitions.up.sql
CREATE TABLE data_model_definitions (...);
-- 含所有 partial unique index

-- 004_create_oui_registry.up.sql
CREATE TABLE oui_registry (...);
INSERT INTO oui_registry ... (种子数据)

-- 005_create_data_model_import_log.up.sql
CREATE TABLE data_model_import_log (...);

-- 006_create_config_templates.up.sql
CREATE TABLE config_templates (...);

-- 007_create_provisioning_tasks.up.sql
CREATE TABLE provisioning_tasks (...);

-- 008_add_device_data_model_ref.up.sql
ALTER TABLE devices ADD COLUMN data_model_id UUID REFERENCES data_model_definitions(id);

-- 009_create_device_groups.up.sql
CREATE TABLE device_groups (...);
CREATE TABLE device_group_members (...);
```

#### Phase 3 — 时序数据表

```sql
-- 010_create_pm_counters.up.sql
CREATE TABLE pm_counters (...);
SELECT create_hypertable('pm_counters', 'time', chunk_time_interval => INTERVAL '1 day');
CREATE INDEX idx_pm_device_time ON pm_counters (device_id, time DESC);
SELECT add_compression_policy('pm_counters', INTERVAL '7 days');
SELECT add_retention_policy('pm_counters', INTERVAL '90 days');

-- 011_create_kpi_values.up.sql
CREATE TABLE kpi_values (...);
SELECT create_hypertable('kpi_values', 'time');

-- 012_create_pm_hourly_aggregate.up.sql
CREATE MATERIALIZED VIEW pm_counters_hourly WITH (timescaledb.continuous) AS ...;

-- 013_create_alarms.up.sql
CREATE TABLE alarms_active (...);
CREATE TABLE alarms_history (...);
SELECT create_hypertable('alarms_history', 'time');
```

#### Phase 4 — 管理与北向表

```sql
-- 014_create_users.up.sql
CREATE TABLE users (...);
CREATE TABLE roles (...);
CREATE TABLE user_roles (...);
CREATE TABLE permissions (...);

-- 015_create_audit_logs.up.sql
CREATE TABLE audit_logs (...);

-- 016_create_firmware_versions.up.sql
CREATE TABLE firmware_versions (...);
CREATE TABLE upgrade_tasks (...);
```

### 2.4 种子数据加载

```
datamodels/seed/carrier_defaults/ → data_model_definitions 表
datamodels/seed/product_models/   → data_model_definitions 表
datamodels/seed/oui_registry.json → oui_registry 表
```

通过 `cmd/migrate seed` 子命令或迁移文件中的 INSERT 加载。

---

## 3. 实施子阶段

### 阶段 21a：基���表（Phase 1）— 001~002
### 阶段 21b：数据模型/配置表（Phase 2）— 003~009
### 阶段 21c：时序数据表（Phase 3）— 010~013
### 阶段 21d：管理表（Phase 4）— 014~016

---

## 4. 文件清单

```
cmd/migrate/main.go
migrations/001_create_devices.up.sql
migrations/001_create_devices.down.sql
migrations/002_create_device_parameters.up.sql
migrations/002_create_device_parameters.down.sql
... (每阶段对应的 up/down SQL 文件)
```

---

## 5. 参考

- backend-design.md 第五章：数据存储策略（全部 Schema）
