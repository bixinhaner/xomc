-- Phase 2: Provisioning Tasks table
-- Tracks auto-provisioning workflow state for each device

CREATE TABLE provisioning_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id      UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    template_id    UUID REFERENCES config_templates(id),
    status         VARCHAR(20) NOT NULL DEFAULT 'discovered'
                   CHECK (status IN ('discovered', 'identifying', 'matching', 'configuring', 'verifying', 'completed', 'failed')),
    current_step   INTEGER NOT NULL DEFAULT 0,
    total_steps    INTEGER NOT NULL DEFAULT 0,
    error_message  TEXT,
    retry_count    INTEGER NOT NULL DEFAULT 0,
    max_retries    INTEGER NOT NULL DEFAULT 3,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pt_device ON provisioning_tasks (device_id);
CREATE INDEX idx_pt_status ON provisioning_tasks (status);
CREATE INDEX idx_pt_device_status ON provisioning_tasks (device_id, status);
CREATE INDEX idx_pt_created ON provisioning_tasks (created_at DESC);

CREATE TRIGGER trigger_pt_updated_at
    BEFORE UPDATE ON provisioning_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
