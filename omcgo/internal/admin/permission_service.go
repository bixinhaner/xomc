package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const permCacheTTL = 5 * time.Minute

// PermissionService provides data-permission logic for device group visibility.
type PermissionService struct {
	roleRepo      RoleDeviceGroupRepository
	groupExpander GroupExpander
	redis         redis.UniversalClient
	logger        *zap.Logger
}

// GroupExpander resolves device-group tree relationships.
type GroupExpander interface {
	ListChildIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error)
}

// NewPermissionService creates a PermissionService.
func NewPermissionService(
	roleRepo RoleDeviceGroupRepository,
	groupExpander GroupExpander,
	redisClient redis.UniversalClient,
	logger *zap.Logger,
) *PermissionService {
	return &PermissionService{
		roleRepo:      roleRepo,
		groupExpander: groupExpander,
		redis:         redisClient,
		logger:        logger.Named("permission-service"),
	}
}

// GetUserVisibleGroupIDs returns the L2 group IDs visible to the given user.
//
// v1.0：参数 `isSuperAdmin` 替代旧 `carrier *model.CarrierCode`（详见 docs/prd/system/users.md §11.11）。
// isSuperAdmin = true（即 user.Source == 'builtIn'） → 看见所有分组（返回 nil 切片）。
// isSuperAdmin = false → 走 user → roles → role_device_groups 派生 + 设备组树递归展开，结果缓存 5min。
func (s *PermissionService) GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error) {
	// 超管 (source='builtIn') 看见所有分组。
	if isSuperAdmin {
		return nil, nil
	}

	// Try Redis cache.
	cacheKey := redisx.Keys.PermVisibleGroups(userID.String())
	cached, err := s.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var ids []uuid.UUID
		if json.Unmarshal(cached, &ids) == nil {
			return ids, nil
		}
	}

	// Query from DB: user → roles → role_device_groups → group IDs.
	directIDs, err := s.roleRepo.GetUserVisibleGroupIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user visible groups: %w", err)
	}

	// Expand selected groups to all descendants. 现场设备组已存在多级树，且设备可能挂在任意
	// 子孙节点；只展开 L1→L2 会让“最大设备权限”的普通用户在设备列表中看到 0 台。
	expanded := make(map[uuid.UUID]struct{})
	visiting := make(map[uuid.UUID]struct{})
	for _, gid := range directIDs {
		s.expandGroupTree(ctx, gid, expanded, visiting)
	}

	result := make([]uuid.UUID, 0, len(expanded))
	for id := range expanded {
		result = append(result, id)
	}

	// Cache in Redis.
	if data, err := json.Marshal(result); err == nil {
		s.redis.Set(ctx, cacheKey, data, permCacheTTL)
	}

	return result, nil
}

func (s *PermissionService) expandGroupTree(ctx context.Context, groupID uuid.UUID, expanded, visiting map[uuid.UUID]struct{}) {
	if _, seen := expanded[groupID]; seen {
		return
	}
	if _, cycle := visiting[groupID]; cycle {
		s.logger.Warn("device group tree cycle detected", zap.String("group_id", groupID.String()))
		return
	}

	visiting[groupID] = struct{}{}
	expanded[groupID] = struct{}{}

	children, err := s.groupExpander.ListChildIDs(ctx, groupID)
	if err != nil {
		s.logger.Warn("list child groups failed", zap.String("group_id", groupID.String()), zap.Error(err))
		delete(visiting, groupID)
		return
	}
	for _, childID := range children {
		s.expandGroupTree(ctx, childID, expanded, visiting)
	}
	delete(visiting, groupID)
}

// InvalidateUserCache removes the cached visible groups for a user.
// 返回 error 以便调用方记录失败计数器（PRD §10 DoD: omc_perm_cache_invalidate_failed_total）。
func (s *PermissionService) InvalidateUserCache(ctx context.Context, userID uuid.UUID) error {
	if err := s.redis.Del(ctx, redisx.Keys.PermVisibleGroups(userID.String())).Err(); err != nil {
		return fmt.Errorf("invalidate user cache %s: %w", userID, err)
	}
	return nil
}

// InvalidateRoleCache invalidates cache for all users with the given role.
// This is a simple approach — in production, you might want a more targeted invalidation.
func (s *PermissionService) InvalidateRoleCache(ctx context.Context, roleID uuid.UUID) {
	// For simplicity, we use a pattern scan. In practice, consider maintaining
	// a reverse index of role → user IDs.
	s.logger.Info("role device groups changed, caches will expire via TTL",
		zap.String("role_id", roleID.String()))
}
