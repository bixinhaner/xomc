-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'normal',
    title VARCHAR(200) NOT NULL,
    content TEXT,
    link VARCHAR(500),
    sender VARCHAR(100) DEFAULT 'system',
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
