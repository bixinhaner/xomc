-- +goose Up
-- upgrade_tasks.firmware_id: rollback tasks do not reference a firmware file, so allow NULL.
--
-- T-0080 hotfix (2026-04-29): renamed from 000038 → 000049. The version 38
-- collided with 000038_alarm_filter_webhook_url.sql (committed 2026-04-27,
-- earlier); this file landed on 2026-04-28 (commit 8c825790) under the same
-- version. PostgreSQL `ALTER COLUMN ... DROP NOT NULL` on an already-nullable
-- column is a silent no-op, so re-applying under the new version number is
-- safe in environments where this DDL was somehow already executed.
ALTER TABLE upgrade_tasks ALTER COLUMN firmware_id DROP NOT NULL;

-- +goose Down
ALTER TABLE upgrade_tasks ALTER COLUMN firmware_id SET NOT NULL;
