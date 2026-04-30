-- +goose Up
-- T-0027 拓扑自动分组规则引擎激活：device_group_members 加 source_type / source_rule_id 区分手工 vs 规则
-- 引用 PRD §12.3
-- D5.B（auto-migrate from default）：rule 触发的 INSERT/UPDATE 显式置 source_type='rule' + source_rule_id
-- A4 守护：DEFAULT 'manual' 让所有现存行视为 manual override，cron 重评不触动

ALTER TABLE device_group_members
    ADD COLUMN IF NOT EXISTS source_type VARCHAR(16) NOT NULL DEFAULT 'manual';

ALTER TABLE device_group_members
    ADD COLUMN IF NOT EXISTS source_rule_id UUID;

-- CHECK 约束：仅允许 manual / rule 两值
ALTER TABLE device_group_members
    DROP CONSTRAINT IF EXISTS chk_device_group_members_source_type;
ALTER TABLE device_group_members
    ADD CONSTRAINT chk_device_group_members_source_type
    CHECK (source_type IN ('manual', 'rule'));

-- 索引：cron 重评 + ApplyRule 时按 source_type 过滤所需
CREATE INDEX IF NOT EXISTS idx_device_group_members_source
    ON device_group_members(source_type, source_rule_id);


-- +goose Down
DROP INDEX IF EXISTS idx_device_group_members_source;

ALTER TABLE device_group_members
    DROP CONSTRAINT IF EXISTS chk_device_group_members_source_type;

ALTER TABLE device_group_members
    DROP COLUMN IF EXISTS source_rule_id;

ALTER TABLE device_group_members
    DROP COLUMN IF EXISTS source_type;
