-- +goose Up
-- device_tasks: convert to HASH partitioned table on device_sn
-- Rationale: device_sn is in every query except PurgeOldTasks (background maintenance)
-- No FK constraints block this migration (deliberately omitted per CLAUDE.md §5.5.3)

-- 1. Rename existing table
ALTER TABLE device_tasks RENAME TO device_tasks_legacy;

-- 2. Create partitioned table with composite PK (id, device_sn)
CREATE TABLE device_tasks (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    method          VARCHAR(64) NOT NULL,
    params          JSONB,
    priority        INTEGER DEFAULT 10,
    command_key     VARCHAR(128),
    cwmp_id         VARCHAR(256),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    result          JSONB,
    error_code      INTEGER,
    error_message   TEXT,
    source          VARCHAR(32) DEFAULT 'api',
    creator_id      VARCHAR(64),
    description     TEXT,
    parent_task_id  UUID,
    command_index   INTEGER DEFAULT 0,
    device_index    INTEGER DEFAULT 0,
    PRIMARY KEY (id, device_sn)
) PARTITION BY HASH (device_sn);

-- 3. Create 16 hash partitions
CREATE TABLE device_tasks_p00 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 0);
CREATE TABLE device_tasks_p01 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 1);
CREATE TABLE device_tasks_p02 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 2);
CREATE TABLE device_tasks_p03 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 3);
CREATE TABLE device_tasks_p04 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 4);
CREATE TABLE device_tasks_p05 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 5);
CREATE TABLE device_tasks_p06 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 6);
CREATE TABLE device_tasks_p07 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 7);
CREATE TABLE device_tasks_p08 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 8);
CREATE TABLE device_tasks_p09 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 9);
CREATE TABLE device_tasks_p10 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 10);
CREATE TABLE device_tasks_p11 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 11);
CREATE TABLE device_tasks_p12 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 12);
CREATE TABLE device_tasks_p13 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 13);
CREATE TABLE device_tasks_p14 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 14);
CREATE TABLE device_tasks_p15 PARTITION OF device_tasks FOR VALUES WITH (MODULUS 16, REMAINDER 15);

-- 4. Migrate data from legacy table
INSERT INTO device_tasks SELECT * FROM device_tasks_legacy;

-- 5. Drop legacy table (and its indexes) before creating new indexes
DROP TABLE device_tasks_legacy;

-- 6. Create indexes (partition-local)
CREATE INDEX idx_device_tasks_status ON device_tasks (status);
CREATE INDEX idx_device_tasks_created_at ON device_tasks (created_at);
CREATE INDEX idx_device_tasks_cwmp_id ON device_tasks (cwmp_id) WHERE cwmp_id IS NOT NULL;
CREATE INDEX idx_device_tasks_pending ON device_tasks (device_sn, status, priority, created_at) WHERE status = 'pending';
CREATE INDEX idx_device_tasks_parent ON device_tasks (parent_task_id) WHERE parent_task_id IS NOT NULL;
CREATE INDEX idx_device_tasks_params_gin ON device_tasks USING GIN (params jsonb_path_ops);
CREATE INDEX idx_device_tasks_result_gin ON device_tasks USING GIN (result jsonb_path_ops);

-- +goose Down
-- Reverse: recreate as non-partitioned table (data loss acceptable in down migration for dev)

-- Rebuild as plain table
CREATE TABLE device_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    method          VARCHAR(64) NOT NULL,
    params          JSONB,
    priority        INTEGER DEFAULT 10,
    command_key     VARCHAR(128),
    cwmp_id         VARCHAR(256),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    result          JSONB,
    error_code      INTEGER,
    error_message   TEXT,
    source          VARCHAR(32) DEFAULT 'api',
    creator_id      VARCHAR(64),
    description     TEXT,
    parent_task_id  UUID,
    command_index   INTEGER DEFAULT 0,
    device_index    INTEGER DEFAULT 0
);

CREATE INDEX idx_device_tasks_device_sn ON device_tasks (device_sn);
CREATE INDEX idx_device_tasks_status ON device_tasks (status);
CREATE INDEX idx_device_tasks_created_at ON device_tasks (created_at);
CREATE INDEX idx_device_tasks_cwmp_id ON device_tasks (cwmp_id) WHERE cwmp_id IS NOT NULL;
CREATE INDEX idx_device_tasks_pending ON device_tasks (device_sn, status, priority, created_at) WHERE status = 'pending';
CREATE INDEX idx_device_tasks_parent ON device_tasks (parent_task_id) WHERE parent_task_id IS NOT NULL;
CREATE INDEX idx_device_tasks_params_gin ON device_tasks USING GIN (params jsonb_path_ops);
CREATE INDEX idx_device_tasks_result_gin ON device_tasks USING GIN (result jsonb_path_ops);
