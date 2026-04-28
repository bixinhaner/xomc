-- +goose Up
-- W2.A.1 / T-0007 整合：F04 邮件通道收件人列表持久化。
-- 与 webhook_url + webhook_secret（000038/000039）并列：
--   - action=notify_email 时该列必填非空，由应用层校验（DB 不加 CHECK，避免与既有 webhook 规则混迁）
--   - 类型 TEXT[]：与既有 alarm_sources / alarm_identifiers / device_ids 数组列同风格
--   - 默认 NULL：既有规则保持兼容，不强制回填
ALTER TABLE alarm_filters ADD COLUMN email_recipients TEXT[];

-- +goose Down
ALTER TABLE alarm_filters DROP COLUMN IF EXISTS email_recipients;
