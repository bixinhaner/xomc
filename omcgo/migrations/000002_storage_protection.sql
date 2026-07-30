-- +goose Up
-- Storage protection policy and audit state. Business writes remain unchanged
-- until an enabled policy is explicitly configured for a target and scope.
CREATE TABLE IF NOT EXISTS storage_protection_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type varchar(32) NOT NULL,
    target_id varchar(128) NOT NULL,
    write_scope varchar(32) NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    warn_used_percent integer NOT NULL DEFAULT 80,
    block_used_percent integer NOT NULL DEFAULT 90,
    recover_used_percent integer NOT NULL DEFAULT 85,
    check_interval_seconds integer NOT NULL DEFAULT 30,
    unknown_behavior varchar(32) NOT NULL DEFAULT 'allow_with_alarm',
    current_state varchar(16) NOT NULL DEFAULT 'normal',
    state_observations integer NOT NULL DEFAULT 0,
    last_observed_ratio double precision,
    last_observed_at timestamptz,
    last_state_changed_at timestamptz,
    updated_by varchar(64) NOT NULL DEFAULT 'system',
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT storage_protection_thresholds_chk CHECK (
        warn_used_percent >= 0 AND
        warn_used_percent < recover_used_percent AND
        recover_used_percent < block_used_percent AND
        block_used_percent <= 100
    ),
    -- The current deployment has one physical storage target. Logical
    -- component names are admission scopes, not independent disks.
    CONSTRAINT storage_protection_target_type_chk CHECK (
        target_type = 'filesystem' AND target_id = 'root' AND write_scope = 'all'
    ),
    CONSTRAINT storage_protection_unknown_behavior_chk CHECK (
        unknown_behavior IN ('allow_with_alarm', 'block_new_uploads')
    ),
    CONSTRAINT storage_protection_state_chk CHECK (
        current_state IN ('normal', 'warning', 'blocked', 'unknown')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS storage_protection_policy_target_scope_uq
    ON storage_protection_policies (target_type, target_id, write_scope);

CREATE TABLE IF NOT EXISTS storage_protection_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id uuid NOT NULL REFERENCES storage_protection_policies(id) ON DELETE CASCADE,
    target_type varchar(32) NOT NULL,
    target_id varchar(128) NOT NULL,
    write_scope varchar(32) NOT NULL,
    previous_state varchar(16),
    new_state varchar(16) NOT NULL,
    reason varchar(256) NOT NULL,
    observed_ratio double precision,
    policy_version bigint NOT NULL,
    operator_id varchar(64) NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS storage_protection_events_target_time_idx
    ON storage_protection_events (target_type, target_id, write_scope, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS storage_protection_events_target_time_idx;
DROP TABLE IF EXISTS storage_protection_events;
DROP INDEX IF EXISTS storage_protection_policy_target_scope_uq;
DROP TABLE IF EXISTS storage_protection_policies;
