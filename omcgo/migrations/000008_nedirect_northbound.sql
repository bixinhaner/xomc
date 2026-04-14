-- +goose Up
-- ============================================================
-- 000008_nedirect_northbound.up.sql
-- NE Direct sessions, NE Direct commands, Northbound outbox
-- ============================================================

-- ============================================================
-- 1. nedirect_sessions — active NE direct connections
-- ============================================================
CREATE TABLE IF NOT EXISTS nedirect_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    device_sn       VARCHAR(128) NOT NULL,
    user_id         VARCHAR(128) NOT NULL,
    username        VARCHAR(128) NOT NULL DEFAULT '',
    status          VARCHAR(32) NOT NULL DEFAULT 'active',
    ip_address      VARCHAR(64) NOT NULL DEFAULT '',
    connected_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnect_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nedirect_sessions_device_sn ON nedirect_sessions (device_sn);
CREATE INDEX IF NOT EXISTS idx_nedirect_sessions_user_id ON nedirect_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_nedirect_sessions_status ON nedirect_sessions (status);
CREATE INDEX IF NOT EXISTS idx_nedirect_sessions_device_user_status ON nedirect_sessions (device_sn, user_id, status);

-- ============================================================
-- 2. nedirect_commands — CLI/MML commands sent through sessions
-- ============================================================
CREATE TABLE IF NOT EXISTS nedirect_commands (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES nedirect_sessions(id) ON DELETE CASCADE,
    device_sn   VARCHAR(128) NOT NULL,
    command     TEXT NOT NULL DEFAULT '',
    status      VARCHAR(32) NOT NULL DEFAULT 'pending',
    response    TEXT NOT NULL DEFAULT '',
    error_msg   TEXT NOT NULL DEFAULT '',
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    respond_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nedirect_commands_session_id ON nedirect_commands (session_id);
CREATE INDEX IF NOT EXISTS idx_nedirect_commands_device_sn ON nedirect_commands (device_sn);
CREATE INDEX IF NOT EXISTS idx_nedirect_commands_status ON nedirect_commands (status);

-- ============================================================
-- 3. nedirect_sessions auto-update trigger
-- ============================================================
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION nedirect_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_nedirect_sessions_updated_at
    BEFORE UPDATE ON nedirect_sessions
    FOR EACH ROW
    EXECUTE FUNCTION nedirect_sessions_updated_at();

-- ============================================================
-- 4. northbound_outbox — reliable event delivery outbox
-- ============================================================
CREATE TABLE IF NOT EXISTS northbound_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL,
    subject TEXT NOT NULL,
    payload JSONB NOT NULL,
    target_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, processing, delivered, dead
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for worker polling: fetch pending/processing items ready for retry.
CREATE INDEX idx_northbound_outbox_status_retry
    ON northbound_outbox (status, next_retry_at)
    WHERE status IN ('pending', 'processing');

-- Index for dead letter queue listing.
CREATE INDEX idx_northbound_outbox_dead
    ON northbound_outbox (created_at DESC)
    WHERE status = 'dead';

-- Index for deduplication by event_id + target_id.
CREATE UNIQUE INDEX idx_northbound_outbox_event_target
    ON northbound_outbox (event_id, target_id);

-- +goose Down
DROP TABLE IF EXISTS northbound_outbox CASCADE;
DROP TABLE IF EXISTS nedirect_commands CASCADE;
DROP TABLE IF EXISTS nedirect_sessions CASCADE;
