-- ============================================================
-- 000135_mml_custom_command_private_name_unique.sql
-- 用户私有模板 command_name 用户级唯一性约束（每用户私有命名空间不允许重名）。
--
-- 业务来源：docs/design/mml-user-private-template-crud-20260520.md §3 D1-D4
--
-- 设计要点：
--   · 作用域：仅 `command_scope='private' AND owner_user_id IS NOT NULL` 行
--   · 公有模板跨用户允许同名（不同用户的"重启脚本"互不影响）
--   · 历史 NULL owner_user_id 脏数据豁免，由 000134 backfill 渐进改进
--   · 大小写敏感（与 PG 默认 collation 一致；不引入业务侧大小写转换分歧）
--
-- 上线前置：
--   · 000134 已加 owner_user_id 列并 backfill；本迁移依赖该列存在
--   · 现网若有 (owner_user_id, command_name, private) 三联同值 → Step 1 自动 rename
--     冲突行的 command_name 为 "原名 (重名-N)"，按 created_at 升序保留第 1 条原名
-- ============================================================

-- +goose Up

-- Step 1: 预 dedup — 自动处理任何现存的 (owner, name) 私有重复
-- +goose StatementBegin
DO $$
DECLARE
    dup RECORD;
BEGIN
    FOR dup IN
        SELECT id,
               command_name,
               ROW_NUMBER() OVER (
                   PARTITION BY owner_user_id, command_name
                   ORDER BY created_at, id
               ) AS rn
        FROM mml_custom_command
        WHERE command_scope = 'private'
          AND owner_user_id IS NOT NULL
    LOOP
        IF dup.rn > 1 THEN
            UPDATE mml_custom_command
               SET command_name = dup.command_name || ' (重名-' || dup.rn || ')'
             WHERE id = dup.id;
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- Step 2: 部分唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS uq_mml_custom_command_private_name_per_owner
    ON mml_custom_command(owner_user_id, command_name)
    WHERE command_scope = 'private' AND owner_user_id IS NOT NULL;

COMMENT ON INDEX uq_mml_custom_command_private_name_per_owner IS
    '用户级唯一性：每个用户的私有模板 command_name 不重复。'
    'public 跨用户允许同名；owner_user_id NULL 的历史脏数据豁免（待 000134 backfill 覆盖）。';


-- +goose Down

-- 仅 DROP 索引；Step 1 自动 rename 过的 command_name 不自动还原（用户可手动恢复）
DROP INDEX IF EXISTS uq_mml_custom_command_private_name_per_owner;
