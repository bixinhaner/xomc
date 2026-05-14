-- +goose Up
-- ============================================================================
-- 000096: users 表加密码策略所需的 2 列（system/config 安全设置 P1）
--
-- ① modifyPWD（首次登录强制改密）依赖 must_change_password 列：
--   - CreateUser 时根据 sys_configs.security.modifyPWD 把该列置为 true
--   - Login 成功后若该列为 true，响应附标记，FE 强制跳改密页
--   - ChangePassword/ResetPassword 成功后置为 false
--
-- ④ expires + validPeriod + promptBeforeDays（密码有效期）依赖 password_changed_at：
--   - 任何 UpdatePassword 都把该列设为 NOW()
--   - Login 时检查 NOW() - password_changed_at > validPeriod * 1 day → 强制改密
--   - 距离过期 ≤ promptBeforeDays 时响应附"即将过期"提示
--   - 老用户首次登录时该列为 NULL（迁移默认）→ 视为"刚改密"，
--     UpdatePassword 一次后自然进入正常生命周期
-- ============================================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS password_changed_at  TIMESTAMPTZ;

COMMENT ON COLUMN users.must_change_password IS
    'P1 ①：true 表示用户下次登录必须先修改密码（FE 跳改密页）。CreateUser/ResetPassword 时由策略决定；ChangePassword 后清零';
COMMENT ON COLUMN users.password_changed_at IS
    'P1 ④：最近一次密码修改时间戳；NULL 视为初始未改。Login 时与 sys_configs.security.validPeriod 比较判断过期';

-- 索引：Login 路径常按 (must_change_password, password_changed_at) 派生策略状态。
-- 单列条件索引（must_change_password=true）足够，避免覆盖全表加成本。
CREATE INDEX IF NOT EXISTS idx_users_must_change_password
    ON users(must_change_password)
    WHERE must_change_password = true;

-- +goose Down
DROP INDEX IF EXISTS idx_users_must_change_password;
ALTER TABLE users
    DROP COLUMN IF EXISTS must_change_password,
    DROP COLUMN IF EXISTS password_changed_at;
