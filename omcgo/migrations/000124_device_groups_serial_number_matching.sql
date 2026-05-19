-- T-DRULE-SN: device_groups / device_rules 增加 'serialNumber' 匹配模式 + serial_number_list 列。
--
-- 背景：现有 matching_mode 仅支持 deviceName/lac/tac 三种模糊或区码匹配；对"精确指
--      定一组 SN 自动分配到某分组"的场景没有原生支持，只能借 deviceName contain
--      规则迂回（每个 SN 一条 OR 条件，规则膨胀）。本迁移加 serialNumber 模式 +
--      TEXT[] 类型 SN 列表，让 matcher 走 list 成员判定，O(N) 一次匹配。
--
-- 兼容性：
--   - 添加列 + 放宽 CHECK 都是非破坏性变更
--   - 旧行 matching_mode IN (deviceName/lac/tac) 保持原状
--   - serial_number_list 默认 NULL，仅 serialNumber 模式行才有值

-- +goose Up

-- 1. device_groups: 加 serial_number_list 列 + 放宽 matching_mode CHECK
ALTER TABLE device_groups
    ADD COLUMN IF NOT EXISTS serial_number_list TEXT[];

ALTER TABLE device_groups
    DROP CONSTRAINT IF EXISTS chk_dg_matching_mode;

ALTER TABLE device_groups
    ADD CONSTRAINT chk_dg_matching_mode
    CHECK (matching_mode IS NULL OR matching_mode IN ('deviceName', 'lac', 'tac', 'serialNumber'));

-- 部分索引：让 SN 匹配 lookup 走 GIN，1000 个 SN 量级也是毫秒级。
CREATE INDEX IF NOT EXISTS idx_dg_serial_number_list
    ON device_groups USING GIN (serial_number_list)
    WHERE serial_number_list IS NOT NULL;

-- 2. device_rules: 同步扩展（保持两表 schema 对齐）
ALTER TABLE device_rules
    ADD COLUMN IF NOT EXISTS serial_number_list TEXT[];

-- device_rules 原本没有 matching_mode CHECK 约束（看 000003），故无需放宽；
-- 应用层 / 校验层负责确保 matching_mode 值合法。

-- +goose Down

DROP INDEX IF EXISTS idx_dg_serial_number_list;

ALTER TABLE device_groups
    DROP CONSTRAINT IF EXISTS chk_dg_matching_mode;

ALTER TABLE device_groups
    ADD CONSTRAINT chk_dg_matching_mode
    CHECK (matching_mode IS NULL OR matching_mode IN ('deviceName', 'lac', 'tac'));

ALTER TABLE device_groups
    DROP COLUMN IF EXISTS serial_number_list;

ALTER TABLE device_rules
    DROP COLUMN IF EXISTS serial_number_list;
