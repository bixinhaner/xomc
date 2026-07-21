-- +goose Up
-- 系统配置保存的运行态应用状态。sys_configs 与 batch/target 由应用层放在同一事务提交。

-- IF NOT EXISTS is intentional: development databases may contain the
-- intermediate pre-merge tables without a Goose version record. The ALTER /
-- backfill section below converges those tables to the released schema.
CREATE TABLE IF NOT EXISTS config_apply_versions (
    category varchar(64) PRIMARY KEY,
    config_version bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS config_apply_batches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category varchar(64) NOT NULL,
    config_version bigint NOT NULL,
    status varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_config_apply_batch_status CHECK (status IN ('pending', 'applying', 'applied', 'failed')),
    CONSTRAINT uq_config_apply_batch_version UNIQUE (category, config_version)
);

ALTER TABLE config_apply_batches
    ALTER COLUMN id SET DEFAULT gen_random_uuid();

CREATE TABLE IF NOT EXISTS config_apply_targets (
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

ALTER TABLE config_apply_targets
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ADD COLUMN IF NOT EXISTS category varchar(64),
    ADD COLUMN IF NOT EXISTS lease_token uuid,
    ADD COLUMN IF NOT EXISTS lease_expires_at timestamptz;

-- Preserve intermediate-schema rows and derive the target category from the
-- owning batch before enforcing the final NOT NULL invariant.
UPDATE config_apply_targets AS target
SET category = batch.category
FROM config_apply_batches AS batch
WHERE target.batch_id = batch.id
  AND target.category IS NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM config_apply_targets WHERE category IS NULL) THEN
        RAISE EXCEPTION 'config_apply_targets contains rows without an owning batch category';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE config_apply_targets
    ALTER COLUMN category SET NOT NULL;

-- Continue version allocation after preserved intermediate batches. Without
-- this backfill the next save would restart at version 1 and collide with the
-- existing (category, config_version) unique constraint.
INSERT INTO config_apply_versions (category, config_version, updated_at)
SELECT category, MAX(config_version), now()
FROM config_apply_batches
GROUP BY category
ON CONFLICT (category) DO UPDATE
SET config_version = GREATEST(config_apply_versions.config_version, EXCLUDED.config_version),
    updated_at = now();

-- Old applying rows did not have a lease expiry. Make them immediately
-- reclaimable, and collapse duplicate in-flight category/target work to one
-- row before creating the database-level serialization fence. Failed siblings
-- remain durable and will be retried by the normal apply worker.
UPDATE config_apply_targets
SET lease_expires_at = COALESCE(lease_expires_at, now())
WHERE status = 'applying';

WITH ranked_applying AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY category, target
               ORDER BY updated_at DESC, id DESC
           ) AS position
    FROM config_apply_targets
    WHERE status = 'applying'
)
UPDATE config_apply_targets AS target
SET status = 'failed',
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error = CASE
        WHEN target.last_error = '' THEN 'migration recovered duplicate applying target'
        ELSE target.last_error || '; migration recovered duplicate applying target'
    END,
    updated_at = now()
FROM ranked_applying
WHERE target.id = ranked_applying.id
  AND ranked_applying.position > 1;

-- A category/target pair is a serial side-effect stream. This database-level
-- fence prevents two app replicas from applying older/newer versions at once.
DROP INDEX IF EXISTS uq_config_apply_target_running;
CREATE UNIQUE INDEX uq_config_apply_target_running
    ON config_apply_targets (category, target)
    WHERE status = 'applying';

CREATE INDEX IF NOT EXISTS idx_config_apply_targets_pending
    ON config_apply_targets (status, updated_at)
    WHERE status IN ('pending', 'failed');

DROP INDEX IF EXISTS idx_config_apply_targets_recovering;
CREATE INDEX idx_config_apply_targets_recovering
    ON config_apply_targets (lease_expires_at)
    WHERE status = 'applying';

-- +goose Down
DROP TABLE IF EXISTS config_apply_targets;
DROP TABLE IF EXISTS config_apply_batches;
DROP TABLE IF EXISTS config_apply_versions;
