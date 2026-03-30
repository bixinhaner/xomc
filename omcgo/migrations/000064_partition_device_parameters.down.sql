-- ============================================================
-- 000064 down: 回退到单表
-- ============================================================

ALTER TABLE device_parameters RENAME TO device_parameters_partitioned;

CREATE TABLE device_parameters (
    device_id        UUID NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);

CREATE INDEX idx_device_params_device ON device_parameters (device_id);

INSERT INTO device_parameters (
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at
)
SELECT device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at
FROM device_parameters_partitioned;

-- 恢复 varchar_pattern_ops 索引（原 000061 迁移）
CREATE INDEX idx_device_params_path_prefix
ON device_parameters (device_id, parameter_path varchar_pattern_ops);

DROP TABLE device_parameters_partitioned;

ANALYZE device_parameters;
