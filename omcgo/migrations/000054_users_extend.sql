-- +goose Up
-- PRD docs/prd/system/users.md §7 P1 / §10 DoD：用户管理 UI 已暴露的字段补全后端落地。
ALTER TABLE users ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS expire_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_by UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_by UUID;

-- 外键：created_by / updated_by 引用 users(id)；删除引用人时置空，避免连锁删除。
-- 用 DO 块在已存在时跳过，保持迁移幂等。
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_created_by_fk'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_created_by_fk
            FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_updated_by_fk'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_updated_by_fk
            FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;
-- +goose StatementEnd

-- 索引：expire_at 用于定期清理过期账号（计划任务扫描）。
CREATE INDEX IF NOT EXISTS idx_users_expire_at ON users(expire_at) WHERE expire_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_expire_at;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_updated_by_fk;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_created_by_fk;
ALTER TABLE users DROP COLUMN IF EXISTS updated_by;
ALTER TABLE users DROP COLUMN IF EXISTS created_by;
ALTER TABLE users DROP COLUMN IF EXISTS expire_at;
ALTER TABLE users DROP COLUMN IF EXISTS description;
