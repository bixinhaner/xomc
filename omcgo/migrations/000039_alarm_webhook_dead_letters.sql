-- +goose Up
-- 迁移 000039: alarm_filters 增加 webhook_secret 列（HMAC 密钥），并新增 alarm_webhook_dead_letters 死信表（W2 T-0011）
-- 范围：webhook 派发耗尽重试后落入死信表，便于后续排查或重投递（重投递留 T-0044）。

ALTER TABLE alarm_filters ADD COLUMN IF NOT EXISTS webhook_secret TEXT;

CREATE TABLE IF NOT EXISTS alarm_webhook_dead_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filter_id UUID NOT NULL REFERENCES alarm_filters(id) ON DELETE CASCADE,
    alarm_id UUID NOT NULL,
    payload JSONB NOT NULL,
    last_error TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_webhook_dead_letters_filter_id ON alarm_webhook_dead_letters(filter_id);
CREATE INDEX IF NOT EXISTS idx_alarm_webhook_dead_letters_failed_at ON alarm_webhook_dead_letters(failed_at DESC);

-- +goose Down
-- 迁移 000039 回滚

DROP INDEX IF EXISTS idx_alarm_webhook_dead_letters_failed_at;
DROP INDEX IF EXISTS idx_alarm_webhook_dead_letters_filter_id;
DROP TABLE IF EXISTS alarm_webhook_dead_letters;
ALTER TABLE alarm_filters DROP COLUMN IF EXISTS webhook_secret;
