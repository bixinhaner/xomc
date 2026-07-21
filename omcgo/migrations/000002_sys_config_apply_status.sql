-- +goose Up
-- 系统配置保存的运行态应用状态。sys_configs 与 batch/target 由应用层放在同一事务提交。

CREATE TABLE config_apply_versions (
    category varchar(64) PRIMARY KEY,
    config_version bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE config_apply_batches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category varchar(64) NOT NULL,
    config_version bigint NOT NULL,
    status varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_config_apply_batch_status CHECK (status IN ('pending', 'applying', 'applied', 'failed')),
    CONSTRAINT uq_config_apply_batch_version UNIQUE (category, config_version)
);

CREATE TABLE config_apply_targets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id uuid NOT NULL REFERENCES config_apply_batches(id) ON DELETE CASCADE,
	category varchar(64) NOT NULL,
    target varchar(128) NOT NULL,
    status varchar(16) NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    applied_at timestamptz,
    last_error text NOT NULL DEFAULT '',
    expected_value jsonb NOT NULL DEFAULT '{}'::jsonb,
    actual_value jsonb NOT NULL DEFAULT '{}'::jsonb,
    lease_token uuid,
    lease_expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_config_apply_target_status CHECK (status IN ('pending', 'applying', 'applied', 'failed')),
    CONSTRAINT uq_config_apply_target UNIQUE (batch_id, target)
);

-- A category/target pair is a serial side-effect stream. This database-level
-- fence prevents two app replicas from applying older/newer versions at once.
CREATE UNIQUE INDEX uq_config_apply_target_running
    ON config_apply_targets (category, target)
    WHERE status = 'applying';

CREATE INDEX idx_config_apply_targets_pending
    ON config_apply_targets (status, updated_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX idx_config_apply_targets_recovering
    ON config_apply_targets (lease_expires_at)
    WHERE status = 'applying';

-- +goose Down
DROP TABLE IF EXISTS config_apply_targets;
DROP TABLE IF EXISTS config_apply_batches;
DROP TABLE IF EXISTS config_apply_versions;
