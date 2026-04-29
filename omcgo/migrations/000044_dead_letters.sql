-- +goose Up
-- T-0012 / R-106: worker 进程级死信队列（inbound EventBus 失败耗尽 retry 后的持久化）。
-- 与 alarm_webhook_dead_letters（outbound webhook 失败 DLQ）语义不同：
--   - alarm_webhook_dead_letters — 北向告警 webhook 派发失败
--   - dead_letters               — worker 订阅 EventBus 处理失败
-- 两表并存，不合并。

CREATE TABLE IF NOT EXISTS dead_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_module VARCHAR(64) NOT NULL,
    source_subject VARCHAR(128) NOT NULL,
    payload BYTEA NOT NULL,
    error TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0
        CHECK (retry_count >= 0 AND retry_count <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dead_letters_module_created
    ON dead_letters(source_module, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_dead_letters_subject
    ON dead_letters(source_subject);

-- +goose Down
DROP INDEX IF EXISTS idx_dead_letters_subject;
DROP INDEX IF EXISTS idx_dead_letters_module_created;
DROP TABLE IF EXISTS dead_letters;
