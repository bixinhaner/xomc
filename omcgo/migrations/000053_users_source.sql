-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS source VARCHAR(16) NOT NULL DEFAULT 'admin';

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_source_check'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_source_check
            CHECK (source IN ('builtIn', 'admin', 'LDAP'));
    END IF;
END $$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_users_source ON users(source);

-- 现网内置用户：seed 文件以 username='admin' 注入，迁移时一次性升级为内置来源
UPDATE users SET source = 'builtIn' WHERE username = 'admin' AND source <> 'builtIn';

-- +goose Down
DROP INDEX IF EXISTS idx_users_source;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_source_check;
ALTER TABLE users DROP COLUMN IF EXISTS source;
