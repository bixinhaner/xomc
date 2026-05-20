-- ============================================================
-- 000134_mml_custom_command_owner_user_id.sql
-- mml_custom_command: 添加 owner_user_id (UUID FK to users.id)，逐步替代
-- 历史 creator (VARCHAR(100) 用户名字符串) 作为所有权字段。
--
-- 背景：
--   现状 creator 字段是字符串：
--     · users.username 改名后，所有 mml_custom_command 行变孤儿
--     · 可见性过滤 SQL（pg_repository.go:1414-1421）逐行 JOIN users 走
--       username 比对，10^5 行规模 + GIN 缺失 → 全表扫
--     · 删用户时无 FK 级联，残留模板"挂在死人头上"
--
-- 三阶段平滑迁移计划（仅 Phase 1 在本迁移完成）：
--   Phase 1（本迁移 + 配套代码 dual-write）：
--     · 加列 owner_user_id UUID NULL，FK → users(id) ON DELETE SET NULL
--     · 一次性 backfill：creator 字符串 ↔ users.username 匹配
--     · 加索引 idx_mml_custom_command_owner（覆盖未来按 owner 过滤的 query）
--     · Repository.Create() 双写 owner_user_id（creator 仍写，保持 read 路径不变）
--     · 读路径仍按 creator 走（验证 backfill 质量再切换）
--
--   Phase 2（独立迁移 + 代码切换，建议下个 release）：
--     · 验证 SELECT COUNT(*) WHERE owner_user_id IS NULL AND creator <> '' = 0
--     · Repository.List() 可见性过滤改用 owner_user_id 比对 current_user_id
--     · 旧 creator 字段保留以兼容历史 audit 查询
--
--   Phase 3（再下个 release，可选）：
--     · ALTER COLUMN owner_user_id SET NOT NULL
--     · 标 creator deprecated，最终 DROP
--
-- 不破坏现网：
--   · 加列允许 NULL → 老代码读 / 写 creator 行为不变
--   · backfill 用 LEFT JOIN 失败兜底 NULL，不阻塞迁移
--   · ON DELETE SET NULL 保证用户删除时模板自动孤儿化（与现状 creator 行为一致）
--
-- 关联：
--   - 设计来源：上一轮对话深度核查 §6 缺口 — creator(string) → owner_user_id(FK) 升级建议
--   - 代码同步：internal/mml/model.go MMLCustomCommand + pg_repository.go Create()
-- ============================================================

-- +goose Up

-- 1. 添加 owner_user_id 列（NULL 允许）+ FK
ALTER TABLE mml_custom_command
    ADD COLUMN IF NOT EXISTS owner_user_id UUID;

-- FK 约束独立加：先确认列、再加约束，便于回滚单独删约束
-- ON DELETE SET NULL：用户删除后保留模板（与历史 creator 字符串残留语义一致）
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'mml_custom_command'
          AND constraint_name = 'fk_mml_custom_command_owner'
    ) THEN
        ALTER TABLE mml_custom_command
            ADD CONSTRAINT fk_mml_custom_command_owner
            FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;
-- +goose StatementEnd

COMMENT ON COLUMN mml_custom_command.owner_user_id IS
    'UUID FK → users.id；正在替代历史 creator(varchar) 作为所有权字段。'
    'Phase 1 仅 dual-write，read 路径仍按 creator；Phase 2 切读后 creator 进入弃用期。';

-- 2. 一次性 backfill：creator (username) → users.id
--    历史脏数据（creator 在 users 表查不到）保持 NULL，Phase 2 可视化后人工修复
UPDATE mml_custom_command t
SET owner_user_id = u.id
FROM users u
WHERE t.owner_user_id IS NULL
  AND t.creator IS NOT NULL
  AND t.creator <> ''
  AND u.username = t.creator;

-- 3. 索引：为 Phase 2 可见性过滤 by owner_user_id 准备
CREATE INDEX IF NOT EXISTS idx_mml_custom_command_owner
    ON mml_custom_command(owner_user_id)
    WHERE owner_user_id IS NOT NULL;


-- +goose Down

-- 回滚顺序：索引 → FK 约束 → 列
DROP INDEX IF EXISTS idx_mml_custom_command_owner;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'mml_custom_command'
          AND constraint_name = 'fk_mml_custom_command_owner'
    ) THEN
        ALTER TABLE mml_custom_command
            DROP CONSTRAINT fk_mml_custom_command_owner;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE mml_custom_command DROP COLUMN IF EXISTS owner_user_id;
