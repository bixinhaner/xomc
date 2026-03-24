-- Accelerate prefix queries (LIKE 'prefix%' can utilize btree index with varchar_pattern_ops)
CREATE INDEX IF NOT EXISTS idx_device_params_path_prefix
ON device_parameters (device_id, parameter_path varchar_pattern_ops);
