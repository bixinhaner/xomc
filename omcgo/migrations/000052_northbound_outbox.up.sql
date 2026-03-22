-- Northbound push outbox table for reliable event delivery.
-- Events are written here first, then consumed by a background worker.
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
