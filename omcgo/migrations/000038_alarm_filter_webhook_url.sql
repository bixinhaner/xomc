-- +goose Up
-- 迁移 000038: alarm_filters 增加 webhook_url 列，支持 notify_webhook action（W1.5 冒烟）
-- 范围：仅 webhook URL 单字段。retry / dead-letter / HMAC / header / template 留 Wave 2 扩列。

ALTER TABLE alarm_filters ADD COLUMN IF NOT EXISTS webhook_url TEXT;

-- CHECK 约束：当 action=notify_webhook 时 webhook_url 必须非空
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'alarm_filters' AND constraint_name = 'chk_alarm_filters_webhook_url_required'
    ) THEN
        ALTER TABLE alarm_filters
            ADD CONSTRAINT chk_alarm_filters_webhook_url_required
            CHECK (action <> 'notify_webhook' OR (webhook_url IS NOT NULL AND webhook_url <> ''));
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- 迁移 000038 回滚：删除 CHECK 约束 + 删除 webhook_url 列

ALTER TABLE alarm_filters DROP CONSTRAINT IF EXISTS chk_alarm_filters_webhook_url_required;
ALTER TABLE alarm_filters DROP COLUMN IF EXISTS webhook_url;
