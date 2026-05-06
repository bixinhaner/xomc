-- v0.6（roles.md）批量补齐：
--   1. role_api_permissions 关联表（§7 P0 #1）—— handler/repo 代码已就位但表缺失
--   2. roles 加 code 字段（§7 P2 #8）—— 程序化引用用，业务名 name 仍主显示
--   3. roles 加 created_by / updated_by（§7 P1 #5）—— 操作者审计
--
-- 约定：
--   - role_api_permissions.endpoint_id REFERENCES api_endpoints(id)（v1.0 已落地）
--   - roles.code 允许 NULL；非 NULL 时全局唯一（部分唯一索引）
--   - created_by / updated_by 引用 users(id) ON DELETE SET NULL；
--     seed 内置角色（admin/operator/viewer）这两个字段为 NULL，对应前端"创建人/更新人"列展示"内置"
--     （与 [users.md §11.9 v0.8](../docs/prd/system/users.md) 落点一致）

-- +goose Up
CREATE TABLE IF NOT EXISTS role_api_permissions (
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES api_endpoints(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, endpoint_id)
);
CREATE INDEX IF NOT EXISTS idx_role_api_permissions_role     ON role_api_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_api_permissions_endpoint ON role_api_permissions(endpoint_id);
COMMENT ON TABLE role_api_permissions IS '角色-API端点关联：v0.6 落定（先前仅 handler/repo 代码就位但 DDL 缺失）';

ALTER TABLE roles ADD COLUMN IF NOT EXISTS code       VARCHAR(64);
ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- code 部分唯一：仅 NOT NULL 行参与唯一性，允许多角色共存"未配 code"状态
CREATE UNIQUE INDEX IF NOT EXISTS uniq_roles_code ON roles(code) WHERE code IS NOT NULL;

COMMENT ON COLUMN roles.code IS '角色编码（程序化引用用，可选）';
COMMENT ON COLUMN roles.created_by IS '创建者用户 ID；NULL 表示由 seed 写入（内置角色）';
COMMENT ON COLUMN roles.updated_by IS '最近一次修改者用户 ID；NULL 表示由 seed 写入或从未被人工修改过';

-- v0.6 决议方向 B：role_device_groups.network_types 维持"每分组独立数组"DDL，
-- 前端 UI 临时简化为"角色级"统一字段（CreateRole/UpdateRole 写入时给每条 role_device_groups 行
-- 写入相同的 network_types 数组）。如未来需要"按分组配置"，再走前端 UI 升级，DDL 不动。
COMMENT ON COLUMN role_device_groups.network_types IS
    'v0.6 决议 B：DB 是每分组独立数组；当前前端 UI 简化为角色级统一字段（同步 SetRoleDeviceGroups 时所有行写入相同值）';

-- +goose Down
DROP INDEX IF EXISTS uniq_roles_code;
ALTER TABLE roles DROP COLUMN IF EXISTS updated_by;
ALTER TABLE roles DROP COLUMN IF EXISTS created_by;
ALTER TABLE roles DROP COLUMN IF EXISTS code;
DROP TABLE IF EXISTS role_api_permissions;
