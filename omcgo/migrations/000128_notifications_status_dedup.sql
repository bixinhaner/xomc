-- +goose Up
-- T-0157 C3: notifications 表扩展 status + dedup_key，支撑消息中心 V1。
--
-- status：与 §4.4 五态文案分层对齐的内部状态，前端图标 / 颜色 / 过滤直接消费。
--   queued / sent / completed / failed / expired / cancelled
--   旧行（type='alarm' / 'task_complete' 等）默认 'completed'，保留向后兼容（业务上看作终态消息）。
--
-- dedup_key：task → notification 映射的去重键（值 = task.id uuid）。
--   订阅器在 task 状态变更时按 (user_id, dedup_key) upsert 同一条消息，状态升级不重复插入。
--   NULL 表示该消息非 task 关联（如告警 / 系统公告），不参与去重。
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'completed';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS dedup_key VARCHAR(64);

-- 部分唯一索引：仅在 dedup_key 非空时强制 (user_id, dedup_key) 唯一。
-- WHERE 子句让旧消息（dedup_key=NULL）与未来其他 dedup_key=NULL 的消息互不干扰。
CREATE UNIQUE INDEX IF NOT EXISTS uniq_notifications_user_dedup
    ON notifications(user_id, dedup_key)
    WHERE dedup_key IS NOT NULL;

-- 普通索引：subscriber 按 dedup_key 直接查询（替代 user_id + 列扫描）
CREATE INDEX IF NOT EXISTS idx_notifications_dedup_key ON notifications(dedup_key) WHERE dedup_key IS NOT NULL;

-- 状态过滤索引：前端列表常按 user_id + status 过滤（如只看 failed/expired）
CREATE INDEX IF NOT EXISTS idx_notifications_user_status ON notifications(user_id, status, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_user_status;
DROP INDEX IF EXISTS idx_notifications_dedup_key;
DROP INDEX IF EXISTS uniq_notifications_user_dedup;
ALTER TABLE notifications DROP COLUMN IF EXISTS dedup_key;
ALTER TABLE notifications DROP COLUMN IF EXISTS status;
