-- +goose Up
-- ============================================================
-- 000159_add_devices_other_partition.up.sql
-- 新增 devices_other 分区（支持非中国三大运营商设备）
-- 功能域: F06 拓扑管理
--
-- 说明：
--   - 新增分区支持非 cmcc/ctcc/cucc 运营商的设备
--   - 不修改现有分区结构，不影响现有查询逻辑
--   - 使用 carrier='intl' 隔离数据（international 简写，4 字符与 cmcc/ctcc/
--     cucc 风格一致，VARCHAR(4) 兼容）
--
-- 修复历史（commit e1db9920 → 2026-05-22 三轮修正）：
--   1. devices.carrier 原 VARCHAR(4)，初版用 carrier='other'（5 字符）
--      想配合 ALTER COLUMN TYPE 拓宽，但 PG 16 完全禁止 ALTER 分区键列 type
--      (cannot alter column "carrier" because it is part of the partition key)。
--      fresh deploy 时 000003 改 VARCHAR(16) 可绕过，但 prod 已 applied
--      000003 (VARCHAR(4))，goose 跳过，prod 跑到本 159 时 'other' 5 字符
--      仍塞不进 VARCHAR(4) 分区父表 → SQLSTATE 22001 value too long。
--      → 最终修正：把 carrier code 'other' 短化为 'intl'（4 字符），与现
--        VARCHAR(4) 兼容，prod / fresh 都能跑通。业务代码（Go/前端）零引用
--        'other'，仅 main/000159 + seed/000160 内部使用，改动局部、零业务影响。
--      → 000003 源头 carrier 保持 VARCHAR(16)（fresh 部署上限更友好；'intl'
--        4 字符在 VARCHAR(4) / VARCHAR(16) 都能存，功能一致）。
--   2. 索引不再引用 devices.status —— 该列已在 migration 000137 (D1 硬切)
--      DROP，替换为 lifecycle_state + is_online 双字段
-- ============================================================

-- 1. 创建国际运营商分区
CREATE TABLE devices_other PARTITION OF devices
FOR VALUES IN ('intl');

COMMENT ON TABLE devices_other IS '设备表 - 国际运营商分区（非 cmcc/ctcc/cucc，如赞比亚 ZED 等；carrier=''intl''）';

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
