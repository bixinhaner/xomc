-- +goose Up
-- ============================================================
-- system 内部账户 + admin 角色绑定
-- ------------------------------------------------------------
-- 用途:为 omcctl 与运维脚本提供"零配置"API key 入口。
--   app 启动期 EnsureInternalAPIKey 用此账户签发 omc-internal API key,
--   写入 /var/lib/omcgo/secrets/.api-key (0640) 供容器内工具读取。
--
-- 设计要点:
--   1. UUID 00000000-...001 为预留系统账户,不可与运营人账户冲突
--   2. password_hash='!disabled-no-password-login!' 非 bcrypt,密码登录天然失败
--   3. source='builtIn' 防止 UI 误删/误改
--   4. 绑定到 admin 角色 (10000000-...001),admin 已绑全 445 端点
--      (admin 端点集见 seed/000067* / 000077* 等)
-- ============================================================
INSERT INTO users (id, username, password_hash, display_name, email, status, source, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'system',
    '!disabled-no-password-login!',
    'OMC System Internal',
    'system@omcgo.internal',
    'active',
    'builtIn',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    source = 'builtIn',
    status = 'active',
    updated_at = NOW()
WHERE users.source IS DISTINCT FROM 'builtIn' OR users.status IS DISTINCT FROM 'active';

INSERT INTO user_roles (user_id, role_id, is_default, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    TRUE,
    NOW()
)
ON CONFLICT (user_id, role_id) DO NOTHING;

-- +goose Down
DELETE FROM user_roles
 WHERE user_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM users
 WHERE id = '00000000-0000-0000-0000-000000000001';
