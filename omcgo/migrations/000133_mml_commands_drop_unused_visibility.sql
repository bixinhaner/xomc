-- ============================================================
-- 000133_mml_commands_drop_unused_visibility.sql
-- 移除 v2.3 catalog 方案预留但实际未启用的 mml_commands 列。
--
-- 背景：方案 mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.1
-- 设想把 Customized（私有/公共模板）合并进 mml_commands，因此在 000132 给
-- mml_commands 加了 visibility / owner_user_id 列。但实施时 Customized 仍
-- 保留在独立表 mml_custom_command，这两列**全代码库零引用**：
--
--   grep -rn 'mml_commands\.visibility\|mml_commands\.owner_user_id' → 0 hit
--   grep -rn '\.OwnerUserID\|\.Visibility' internal/mml/             → 0 hit
--   grep -rn 'visibility' internal/mml/catalogloader/                → 0 hit
--
-- 同步清理：
--   - DROP COLUMN visibility / owner_user_id（自动级联 chk_mml_commands_visibility
--     CHECK 约束与 idx_mml_commands_owner 部分索引）
--   - 保留 deprecated_at（catalogloader 软删机制实际在用）
--   - 保留 mml_command_sub_field_overrides 表（per-user 配置保护，未来启用）
--
-- 何时该恢复：若 Customized 与 catalog 决定真正合表，先恢复列再实施合表
-- 改造；Down 段保留与 000132 完全等价的列定义/约束/索引/COMMENT。
--
-- 关联：
--   - 历史决策记录 → docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.5
--   - 当前 Customized 路径 → internal/mml/{handler.go,service.go,pg_repository.go}
--     使用 mml_custom_command + creator(varchar) + command_scope(public|private)
-- ============================================================

-- +goose Up

-- visibility 与 owner_user_id 同时 DROP；PG 会自动级联：
--   · chk_mml_commands_visibility（CHECK 仅引用 visibility）
--   · idx_mml_commands_owner（部分索引引用 visibility + owner_user_id）
ALTER TABLE mml_commands
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS owner_user_id;


-- +goose Down

-- 恢复列（与 000132 §2 一致：visibility VARCHAR(8) 允许 NULL，owner_user_id UUID）
ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(8),
    ADD COLUMN IF NOT EXISTS owner_user_id UUID;

-- 恢复 CHECK 约束（仅在不存在时创建，与 000132 一致的幂等保护）
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.constraint_column_usage
        WHERE table_name = 'mml_commands' AND constraint_name = 'chk_mml_commands_visibility'
    ) THEN
        ALTER TABLE mml_commands
            ADD CONSTRAINT chk_mml_commands_visibility
            CHECK (visibility IS NULL OR visibility IN ('public', 'private'));
    END IF;
END $$;
-- +goose StatementEnd

-- 恢复列注释
COMMENT ON COLUMN mml_commands.visibility IS
    'R-5: Customized 命令可见性 public/private；source=admin 时必填，standard 时 NULL';
COMMENT ON COLUMN mml_commands.owner_user_id IS
    'R-5: Customized 命令创建者；private 时按 owner 隔离，admin 角色可跨用户访问';

-- 恢复回填（保守默认）
UPDATE mml_commands
    SET visibility = 'private'
    WHERE source = 'admin' AND visibility IS NULL;

-- 恢复部分索引（与 000132 完全一致）
CREATE INDEX IF NOT EXISTS idx_mml_commands_owner
    ON mml_commands(visibility, owner_user_id)
    WHERE owner_user_id IS NOT NULL;
