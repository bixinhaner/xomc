-- T-0098-P1-04 — 告警字典 schema（设计 §3.2 / 实施计划 §1.3）
--
-- 目标：建立 alarm_severity_levels（4 行）+ alarm_definitions（442 行）+ alarms_active.is_unknown 列。
-- 设计依据：docs/design/参数-KPI-告警-整合设计方案.md §3.2.1 / §3.2.2 / §3.3
--
-- 决策 D2=B（实施计划 §6 决策表）：
--   旧 alarm_libraries / alarm_library_i18n 在 Phase 5 P5-06 时 DROP（本迁移**不动**它们）；
--   原因：P1 与 P2-10（fallback 接收路径）尚未上线时若过早删旧表，receiver 还在依赖旧表会立即崩溃。
--   D2=B 路径在 P5 完成时执行 DROP + CREATE（XML 真相源重载即恢复数据）。
--
-- alarm_definitions 字段对齐设计 §3.2.2：
--   identifier VARCHAR(32) UNIQUE   告警全局唯一标识（如 "10001"）
--   ne_type VARCHAR(16)             ENB / GNB / OMC / EPC / EGW / CPE / UPS（7 种）
--   severity_id FK → alarm_severity_levels(id)   ON DELETE RESTRICT（不允许误删致定义悬挂）
--   cn_name / en_name / cn_probable_cause / en_probable_cause / cn_suggestion / en_suggestion
--   event_type INT                  事件类型（设计示例 30003）
--   is_show BOOLEAN DEFAULT TRUE    前台是否显示
--
-- alarms_active.is_unknown ADD COLUMN：
--   设计 §3.3 注："alarms 表需要预留 is_unknown BOOLEAN NOT NULL DEFAULT FALSE 字段"
--   alarms_active 非分区表（迁移 000006 验证），ALTER ADD COLUMN 安全
--   未来 P2-10 接收路径未命中 + product.enable_unknown_alarm=true 时写 is_unknown=true 行

-- +goose Up

-- 1. alarm_severity_levels — 严重级参考表
CREATE TABLE IF NOT EXISTS alarm_severity_levels (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code          INT NOT NULL UNIQUE,
    name          VARCHAR(16) NOT NULL UNIQUE,
    display_order INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  alarm_severity_levels IS 'T-0098 告警严重级参考表（设计 §3.2.1）；4 行种子（Critical/Major/Minor/Warning）';
COMMENT ON COLUMN alarm_severity_levels.code IS '运营商规范固定码 31001-31004，跨系统稳定';

-- 4 行种子（运营商规范固定 code）
INSERT INTO alarm_severity_levels (code, name, display_order) VALUES
    (31001, 'Critical', 1),
    (31002, 'Major',    2),
    (31003, 'Minor',    3),
    (31004, 'Warning',  4)
ON CONFLICT (code) DO NOTHING;

-- 2. alarm_definitions — 告警定义主表
CREATE TABLE IF NOT EXISTS alarm_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier          VARCHAR(32) NOT NULL UNIQUE,
    ne_type             VARCHAR(16) NOT NULL,
    cn_name             VARCHAR(256),
    en_name             VARCHAR(256),
    severity_id         UUID NOT NULL REFERENCES alarm_severity_levels(id) ON DELETE RESTRICT,
    event_type          INT,
    cn_probable_cause   TEXT,
    en_probable_cause   TEXT,
    cn_suggestion       TEXT,
    en_suggestion       TEXT,
    is_show             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_definitions_ne_type        ON alarm_definitions(ne_type);
CREATE INDEX IF NOT EXISTS idx_alarm_definitions_severity_show  ON alarm_definitions(severity_id, is_show);

CREATE TRIGGER trigger_alarm_definitions_updated_at
    BEFORE UPDATE ON alarm_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  alarm_definitions IS 'T-0098 告警定义主表（设计 §3.2.2）；442 行典型规模，由 7 个 ne_type XML 文件载入';
COMMENT ON COLUMN alarm_definitions.identifier IS '告警全局唯一标识，跨 ne_type 唯一（设计 §3.4 加载流程校验）';
COMMENT ON COLUMN alarm_definitions.ne_type IS '网元类型：ENB / GNB / OMC / EPC / EGW / CPE / UPS';
COMMENT ON COLUMN alarm_definitions.severity_id IS 'ON DELETE RESTRICT：不允许误删严重级致告警定义悬挂';

-- 3. alarms_active 新增 is_unknown 列（fallback 路径标记，P2-10 写入）
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS is_unknown BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_alarms_active_is_unknown ON alarms_active(is_unknown) WHERE is_unknown;
COMMENT ON COLUMN alarms_active.is_unknown IS 'T-0098 fallback 标记：identifier 不在告警库时 product.enable_unknown_alarm=true 路径写入；治理闭环过滤依据';

-- +goose Down

-- 反向：先 drop alarms_active.is_unknown（与新表无 FK 关系，先后无强约束）
DROP INDEX IF EXISTS idx_alarms_active_is_unknown;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS is_unknown;

DROP TRIGGER IF EXISTS trigger_alarm_definitions_updated_at ON alarm_definitions;
DROP TABLE IF EXISTS alarm_definitions;

DROP TABLE IF EXISTS alarm_severity_levels;
