-- 内置角色规范化：仅 admin 保留 is_system=TRUE（系统受保护，不可删除/不可改名）；
-- operator / viewer 历史 seed 误标 is_system=TRUE，本迁移将其下调为 FALSE，
-- 从而允许管理员在 UI 上对这两个默认角色进行修改 / 删除 / 重建。
--
-- 行为依据：internal/admin/pg_role_repository.go Delete 仅在 is_system=TRUE 时返回 403。
--
-- 兼容性：
--   - 全新部署：seed/000001_seed_data.sql 已直接 seed 为 FALSE，本迁移为 no-op。
--   - 已部署环境：本迁移把现有 operator/viewer 行 is_system 由 TRUE 改为 FALSE。
--   - 已被运维手动改过名（不再叫 operator/viewer）：本迁移按名称匹配，匹配不到即跳过，安全。

-- +goose Up
UPDATE roles
SET is_system = FALSE
WHERE name IN ('operator', 'viewer')
  AND is_system = TRUE;

-- +goose Down
-- 回退：把 operator / viewer 重新标记为系统角色，恢复 v0.1 历史 seed 状态。
UPDATE roles
SET is_system = TRUE
WHERE name IN ('operator', 'viewer')
  AND is_system = FALSE;
