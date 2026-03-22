-- NE Direct sessions: tracks active connections between users and network elements
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

-- NE Direct commands: records of CLI/MML commands sent through sessions
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

-- Auto-update updated_at trigger for sessions
CREATE OR REPLACE FUNCTION nedirect_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_nedirect_sessions_updated_at
    BEFORE UPDATE ON nedirect_sessions
    FOR EACH ROW
    EXECUTE FUNCTION nedirect_sessions_updated_at();
