-- +goose Up
-- ============================================================================
-- P2: 把"登录页消费"的 4 个 security 字段标记为 is_public=true，让未登录的
-- LoginPage 通过 GET /admin/public/configs?category=security 拿到值。
--
-- 这些字段无敏感性，公开无害：
--   - userSessionExpirationMin: 屏幕锁定分钟数（idle timer 用，登录后才生效，
--     但 LoginPage 提前拉一次让登录后立即生效，无需再发请求）
--   - isBrowserAutoRecordPass:  禁止浏览器记密码（LoginPage 切 autocomplete）
--   - enabledFlag:               登录通知开关（LoginPage 决定是否消费 msg）
--   - msg:                       登录通知文案（直接由 Login 响应附带，此处冗余
--                                公开仅供 SecuritySettings 设置页消费）
--
-- 其余 security 字段（attemptTimes / sumTimes / unlockMinu / sigh）保持
-- is_public=false（管理员才能查看 / 修改）。
-- ============================================================================

UPDATE sys_configs
SET is_public = true, updated_at = NOW()
WHERE category = 'security'
  AND key IN ('userSessionExpirationMin', 'isBrowserAutoRecordPass', 'enabledFlag', 'msg');

-- +goose Down
UPDATE sys_configs
SET is_public = false, updated_at = NOW()
WHERE category = 'security'
  AND key IN ('userSessionExpirationMin', 'isBrowserAutoRecordPass', 'enabledFlag', 'msg');
