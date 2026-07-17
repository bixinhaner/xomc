-- +goose Up
-- Issue #100：登录页在认证前通过公开接口读取以下安全配置。
-- consolidated seed baseline 曾遗漏历史 000101 的公开标记，导致接口返回空数组，
-- LoginPage 因而回退到允许浏览器自动填充密码的 autocomplete 属性。
INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES ('security', 'isBrowserAutoRecordPass', 'false', 'bool', '禁止浏览器自动记录密码', TRUE)
ON CONFLICT (category, key) DO UPDATE
SET is_public = TRUE,
    updated_at = NOW();

-- +goose Down
UPDATE sys_configs
SET is_public = FALSE,
    updated_at = NOW()
WHERE category = 'security'
  AND key = 'isBrowserAutoRecordPass';
