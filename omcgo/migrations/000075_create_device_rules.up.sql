-- 设备规则表
-- 用于定义设备自动归属规则，规则可以绑定到设备分组

CREATE TABLE device_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    priority        INTEGER NOT NULL DEFAULT 0,
    target_group_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    -- 匹配规则
    matching_mode   VARCHAR(16),
    name_rule_list  JSONB,
    lac_list        INTEGER[],
    tac_list        INTEGER[],
    -- 审计字段
    description     TEXT,
    operators       TEXT,  -- 生成的规则描述
    created_by      VARCHAR(64),
    updated_by      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 唯一约束：同一优先级只能有一个启用的规则
CREATE UNIQUE INDEX idx_device_rules_priority_unique ON device_rules (priority) WHERE enabled = true;

-- 索引
CREATE INDEX idx_device_rules_target_group ON device_rules (target_group_id);
CREATE INDEX idx_device_rules_enabled ON device_rules (enabled);
CREATE INDEX idx_device_rules_priority ON device_rules (priority);

-- 更新触发器
CREATE TRIGGER trigger_device_rules_updated_at
    BEFORE UPDATE ON device_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE device_rules IS '设备归属规则';
COMMENT ON COLUMN device_rules.id IS '规则唯一标识';
COMMENT ON COLUMN device_rules.name IS '规则名称';
COMMENT ON COLUMN device_rules.priority IS '优先级，数字越小优先级越高';
COMMENT ON COLUMN device_rules.target_group_id IS '目标设备分组，设备匹配后归属到此分组';
COMMENT ON COLUMN device_rules.enabled IS '启用状态，未启用的规则无法应用';
COMMENT ON COLUMN device_rules.matching_mode IS '匹配模式: deviceName(设备名称), lac(位置区码), tac(跟踪区码)';
COMMENT ON COLUMN device_rules.name_rule_list IS '设备名称匹配规则，JSON数组格式';
COMMENT ON COLUMN device_rules.lac_list IS 'LAC 位置区码列表';
COMMENT ON COLUMN device_rules.tac_list IS 'TAC 跟踪区码列表';
COMMENT ON COLUMN device_rules.operators IS '生成的规则描述，用于前端显示';

-- 设备分组表新增字段：绑定的规则ID
ALTER TABLE device_groups ADD COLUMN bound_rule_id UUID REFERENCES device_rules(id) ON DELETE SET NULL;

COMMENT ON COLUMN device_groups.bound_rule_id IS '绑定的设备规则ID，存在时使用规则的匹配条件而非分组自身的规则';
