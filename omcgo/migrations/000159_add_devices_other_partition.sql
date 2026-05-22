-- +goose Up
-- ============================================================
-- 000159_add_devices_other_partition.up.sql
-- 新增 devices_other 分区（支持非中国三大运营商设备）
-- 功能域: F06 拓扑管理
--
-- 说明：
--   - 新增 'other' 分区支持非 cmcc/ctcc/cucc 运营商的设备
--   - 不修改现有分区结构，不影响现有查询逻辑
--   - 使用 carrier='other' 隔离数据
--
-- 前置修复历史（commit e1db9920 → 2026-05-22 修正）：
--   1. devices.carrier / device_groups.carrier 原 VARCHAR(4) 装不下 'other'
--      (5 字符)。本文件原先尝试 ALTER COLUMN TYPE 拓宽，但 PG 16 完全禁止
--      ALTER 分区键列 type (cannot alter column "carrier" because it is part
--      of the partition key)，fresh deploy 上 fatal。
--      → 修正路径：直接改 000003_devices.sql 源头 VARCHAR(4) → VARCHAR(16)，
--        本文件不再尝试 ALTER（多余且不可行）。已部署到 158 的 prod DB 由于
--        partition key 限制，carrier 列**留作 VARCHAR(4)**，需 DBA 手动重建
--        devices 表才能扩宽；fresh deploy 走 000003 新 DDL 自然 VARCHAR(16)。
--   2. 索引不再引用 devices.status —— 该列已在 migration 000137 (D1 硬切)
--      DROP，替换为 lifecycle_state + is_online 双字段
-- ============================================================

-- 1. 创建 other 运营商分区
CREATE TABLE devices_other PARTITION OF devices
FOR VALUES IN ('other');

COMMENT ON TABLE devices_other IS '设备表 - 其他运营商分区（非 cmcc/ctcc/cucc，如赞比亚 ZED 等国际运营商）';

-- 2. 为 other 分区创建索引（与 migration 000003 + 000137 后的活分区索引对齐）
CREATE INDEX idx_devices_other_serial_number       ON devices_other (serial_number);
CREATE INDEX idx_devices_other_carrier_lifecycle   ON devices_other (carrier, lifecycle_state);
CREATE INDEX idx_devices_other_oui                 ON devices_other (oui);
CREATE INDEX idx_devices_other_carrier_tech        ON devices_other (carrier, technology);
CREATE INDEX idx_devices_other_lifecycle_state     ON devices_other (lifecycle_state);
CREATE INDEX idx_devices_other_is_online           ON devices_other (is_online) WHERE is_online = TRUE;
CREATE INDEX idx_devices_other_last_inform         ON devices_other (last_inform_at);
CREATE INDEX idx_devices_other_deleted_at          ON devices_other (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
-- ============================================================
-- 回滚迁移
-- ============================================================

DROP INDEX IF EXISTS idx_devices_other_deleted_at;
DROP INDEX IF EXISTS idx_devices_other_last_inform;
DROP INDEX IF EXISTS idx_devices_other_is_online;
DROP INDEX IF EXISTS idx_devices_other_lifecycle_state;
DROP INDEX IF EXISTS idx_devices_other_carrier_tech;
DROP INDEX IF EXISTS idx_devices_other_oui;
DROP INDEX IF EXISTS idx_devices_other_carrier_lifecycle;
DROP INDEX IF EXISTS idx_devices_other_serial_number;

DROP TABLE IF EXISTS devices_other;

-- 注：carrier 列 TYPE 不回滚到 VARCHAR(4) —— 数据可能已含 'other' 字符串
-- 导致 truncate 风险；保持 VARCHAR(16) 是安全的（兼容老短字符串）。
