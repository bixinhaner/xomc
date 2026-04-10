-- ============================================================================
-- Goose 迁移文件示例: 创建 PM 计数器超表
-- ============================================================================
-- 这是 Goose 迁移文件格式 (与 golang-migrate 兼容)
--  goose:build
-- ============================================================================

-- +goose Up
-- +goose StatementBegin

-- 1. 创建 PM 计数器基础表
CREATE TABLE IF NOT EXISTS pm_counters (
    time TIMESTAMPTZ NOT NULL,
    device_id UUID NOT NULL,
    counter_name VARCHAR(100) NOT NULL,
    counter_value DOUBLE PRECISION,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 2. 添加主键 (必须在 create_hypertable 之前)
ALTER TABLE pm_counters ADD PRIMARY KEY (time, device_id, counter_name);

-- 3. 转换为 TimescaleDB 超表
-- 注意: 这是 DDL 管理语句,必须使用 Goose 执行,sqlc 不处理
SELECT create_hypertable(
    'pm_counters', 
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- 4. 创建索引
CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time 
ON pm_counters (device_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_pm_counters_name 
ON pm_counters (counter_name);

-- 5. 添加表注释
COMMENT ON TABLE pm_counters IS 'PM 计数器时序数据超表';
COMMENT ON COLUMN pm_counters.time IS '时间戳';
COMMENT ON COLUMN pm_counters.device_id IS '设备 ID';
COMMENT ON COLUMN pm_counters.counter_name IS '计数器名称';
COMMENT ON COLUMN pm_counters.counter_value IS '计数器值';
COMMENT ON COLUMN pm_counters.metadata IS '元数据 (JSONB)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 回滚: 删除超表和索引
DROP INDEX IF EXISTS idx_pm_counters_name;
DROP INDEX IF EXISTS idx_pm_counters_device_time;
DROP TABLE IF EXISTS pm_counters CASCADE;

-- +goose StatementEnd
