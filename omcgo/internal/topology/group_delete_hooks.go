package topology

import (
	"context"

	"github.com/google/uuid"
)

// AffectedRole 设备分组被删除时受影响的角色摘要（仅 ID + Name，便于前端/审计展示）。
type AffectedRole struct {
	ID   uuid.UUID
	Name string
}

// RoleAffectedQuery 查询绑定了即将被删除分组的角色集合。
// 由 admin 模块的 PgRoleRepository.ListRolesByGroupIDs 适配实现，
// 通过接口注入避免 topology → admin 包反向依赖（admin 已依赖 topology.GroupExpander）。
type RoleAffectedQuery interface {
	ListRolesByGroupIDs(ctx context.Context, groupIDs []uuid.UUID) ([]AffectedRole, error)
}

// RoleCachePurger 失效角色下所有用户的可见域缓存。
// 由 admin 模块的 AdminService.InvalidatePermCacheByRole 适配实现。
type RoleCachePurger interface {
	InvalidatePermCacheByRole(ctx context.Context, roleID uuid.UUID)
}

// SetGroupDeleteHooks 注入"删除设备分组联动"的钩子（PRD users.md §11.7 决议③ /
// roles.md §11.4）。两个 hook 都为 nil 时 DeleteGroup 退化为旧行为（仅删库），
// 用于 unit test 不依赖 admin 模块的场景。
func (s *DeviceGroupService) SetGroupDeleteHooks(query RoleAffectedQuery, purger RoleCachePurger) {
	s.roleQuery = query
	s.roleCacheBuster = purger
}
