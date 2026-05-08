-- B3-Phase2-B: DROP TABLE permissions
--
-- 历史 RBAC 模型抽象 (resource, action) 三元组，与 B3 端点级 (path, method) 模型重复。
-- 端点级策略由 role_api_permissions JOIN api_endpoints 承载，详见
-- migrations/000056_roles_v1_extras.sql + seed/000064_seed_role_api_permissions_viewer.sql。
--
-- 风险：DROP CASCADE 会清掉 permissions 表中存量数据；Casbin LoadPolicy 已切到
-- role_api_permissions 单源（参 internal/admin/casbin.go），rollback 路径见 Down 段。
--
-- 关联 PRD：docs/prd/system/menu-dynamic-loading.md §4.2.4 (B3-Phase2-B)

-- +goose Up
DROP INDEX IF EXISTS idx_permissions_role;
DROP TABLE IF EXISTS permissions CASCADE;

-- +goose Down
CREATE TABLE IF NOT EXISTS permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource VARCHAR(64) NOT NULL,
    action   VARCHAR(16) NOT NULL,
    UNIQUE(role_id, resource, action)
);
CREATE INDEX IF NOT EXISTS idx_permissions_role ON permissions (role_id);
