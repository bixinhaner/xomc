-- +goose Up
CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL UNIQUE,
    channel VARCHAR(32) NOT NULL CHECK (channel IN ('email', 'sms', 'webhook')),
    language VARCHAR(16) NOT NULL DEFAULT 'zh-CN',
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    variables TEXT[] NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notification_templates_channel ON notification_templates(channel);
CREATE INDEX IF NOT EXISTS idx_notification_templates_enabled ON notification_templates(enabled);

CREATE TABLE IF NOT EXISTS notification_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID REFERENCES notification_templates(id) ON DELETE SET NULL,
    channel VARCHAR(32) NOT NULL CHECK (channel IN ('email', 'sms', 'webhook')),
    recipients TEXT[] NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending','sent','failed','dead_letter')),
    error_message TEXT,
    alarm_id UUID,
    retry_count INT NOT NULL DEFAULT 0,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notification_history_channel_created ON notification_history(channel, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_history_template_id ON notification_history(template_id);
CREATE INDEX IF NOT EXISTS idx_notification_history_alarm_id ON notification_history(alarm_id);
CREATE INDEX IF NOT EXISTS idx_notification_history_status ON notification_history(status);

-- +goose Down
DROP INDEX IF EXISTS idx_notification_history_status;
DROP INDEX IF EXISTS idx_notification_history_alarm_id;
DROP INDEX IF EXISTS idx_notification_history_template_id;
DROP INDEX IF EXISTS idx_notification_history_channel_created;
DROP TABLE IF EXISTS notification_history;
DROP INDEX IF EXISTS idx_notification_templates_enabled;
DROP INDEX IF EXISTS idx_notification_templates_channel;
DROP TABLE IF EXISTS notification_templates;
