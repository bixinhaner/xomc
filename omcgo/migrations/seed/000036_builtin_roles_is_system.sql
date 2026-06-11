-- +goose Up
-- 修正内置角色 operator / viewer 的 is_system 标记（issue #136）。
--
-- 背景：seed/000001_init_seed.sql 把三个内置种子角色中的 admin 标 is_system=true，
--       但 operator(10000000-...-0002) / viewer(10000000-...-0003) 误标 is_system=false。
--       DELETE /admin/roles/:id 的删除保护仅以 is_system 判定 → 这两个内置角色可被删除，
--       与 admin 受 403 保护口径不一致（安全缺陷）。
-- 修复：把这两个内置角色的 is_system 收紧为 true，使删除保护对三个内置角色口径一致。
-- 幂等：按固定 UUID 更新；重跑命中相同行、结果不变。仅当前为 false 时实际产生行变更。
UPDATE roles
SET is_system = true,
    updated_at = now()
WHERE id IN (
    '10000000-0000-0000-0000-000000000002',  -- operator
    '10000000-0000-0000-0000-000000000003'   -- viewer
)
  AND is_system IS DISTINCT FROM true;

-- +goose Down
-- 还原为 seed 原始口径（operator/viewer is_system=false）。
UPDATE roles
SET is_system = false,
    updated_at = now()
WHERE id IN (
    '10000000-0000-0000-0000-000000000002',  -- operator
    '10000000-0000-0000-0000-000000000003'   -- viewer
)
  AND is_system IS DISTINCT FROM false;
