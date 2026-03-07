CREATE TABLE mr_indicators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    indicator_name VARCHAR(200) NOT NULL,
    indicator_code VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    unit VARCHAR(50),
    category VARCHAR(50),
    value_range_min DOUBLE PRECISION,
    value_range_max DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mr_indicators_code ON mr_indicators(indicator_code);
CREATE INDEX idx_mr_indicators_category ON mr_indicators(category);

CREATE TABLE mr_device_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn VARCHAR(64) NOT NULL,
    device_name VARCHAR(200),
    cell_id VARCHAR(64) NOT NULL,
    cell_name VARCHAR(200),
    enabled BOOLEAN NOT NULL DEFAULT true,
    sampling_interval INTEGER NOT NULL DEFAULT 15,
    last_collect_time TIMESTAMPTZ,
    total_records BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mr_mappings_device ON mr_device_mappings(device_sn);
CREATE INDEX idx_mr_mappings_enabled ON mr_device_mappings(enabled);
CREATE TRIGGER trigger_mr_device_mappings_updated_at
    BEFORE UPDATE ON mr_device_mappings FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed data
INSERT INTO mr_indicators (indicator_name, indicator_code, description, unit, category, value_range_min, value_range_max) VALUES
    ('RSRP', 'RSRP', '参考信号接收功率', 'dBm', 'coverage', -140, -44),
    ('RSRQ', 'RSRQ', '参考信号接收质量', 'dB', 'coverage', -20, -3),
    ('SINR', 'SINR', '信干噪比', 'dB', 'quality', -10, 30),
    ('TA', 'TA', '时间提前量', '', 'timing', 0, 1282),
    ('PHR', 'PHR', '功率余量', 'dB', 'power', -23, 40);
