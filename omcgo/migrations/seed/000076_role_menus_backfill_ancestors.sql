-- 修复历史 role_menus 数据：当某 role 授予了菜单 / 按钮，但没有授予其祖先目录时，
-- assembleMenuTree 会因父节点缺失把整棵子树丢弃（孤儿节点既不是 root，也不挂在
-- 任何 root 下，导致前端 NavMenu 看不到该子树）。
--
-- 配套代码侧：service.AdminService.SetRoleMenus 已加 expandMenuAncestors，新写入
-- 会自动补全祖先链。本迁移负责把已有数据一次性纠偏：对每一行 (role_id, menu_id)，
-- 沿 menus.parent_id 向上递归，把所有祖先 ID 也写入 role_menus（去重）。
--
-- 幂等：ON CONFLICT (role_id, menu_id) DO NOTHING；多次执行结果一致。
-- UNION（非 UNION ALL）天然去重，且能在祖先链有环或重复时收敛。

-- +goose Up
WITH RECURSIVE ancestors AS (
    -- 起始集：所有已授权的 (role_id, menu_id) 自身
    SELECT rm.role_id, m.id AS menu_id, m.parent_id
    FROM role_menus rm
    JOIN menus m ON m.id = rm.menu_id
    UNION
    -- 递归：每层把 parent_id 当成新的 menu_id 加入集合
    SELECT a.role_id, m.id AS menu_id, m.parent_id
    FROM ancestors a
    JOIN menus m ON m.id = a.parent_id
)
INSERT INTO role_menus (role_id, menu_id, created_by)
SELECT DISTINCT role_id, menu_id, NULL::uuid
FROM ancestors
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
-- 无回滚：纠偏新增的 role_menus 行无法精确区分"用户原本就显式授权"还是"本次补全"，
-- 强行删除会破坏正常授权。需要回滚时由运维按 role 重置 role_menus。
SELECT 1;
