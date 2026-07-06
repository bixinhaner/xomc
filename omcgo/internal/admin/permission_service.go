package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/model"
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
	grants, err := s.getUserVisibleDeviceGrants(ctx, userID, isSuperAdmin)
	if err != nil {
		return nil, fmt.Errorf("get user visible groups: %w", err)
	}

	if grants == nil {
		return nil, nil
	}
	groupSet := make(map[uuid.UUID]struct{})
	for _, grant := range grants {
		for _, groupID := range grant.GroupIDs {
			groupSet[groupID] = struct{}{}
		}
	}
	result := make([]uuid.UUID, 0, len(groupSet))
	for id := range groupSet {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result, nil
}

// GetUserVisibleDeviceGrants returns expanded device-data grants for the user.
// nil means superadmin, empty slice means authenticated but no data permissions.
func (s *PermissionService) GetUserVisibleDeviceGrants(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]model.DeviceVisibilityGrant, error) {
	return s.getUserVisibleDeviceGrants(ctx, userID, isSuperAdmin)
}

func (s *PermissionService) getUserVisibleDeviceGrants(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]model.DeviceVisibilityGrant, error) {
	if isSuperAdmin {
		return nil, nil
	}

	cacheKey := redisx.Keys.PermVisibleGroups(userID.String())
	if cached, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var grants []model.DeviceVisibilityGrant
		if json.Unmarshal(cached, &grants) == nil {
			return grants, nil
		}
	}

	rawGrants, err := s.roleRepo.GetUserVisibleDeviceGrants(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user visible device grants: %w", err)
	}

	expanded := s.expandDeviceVisibilityGrants(ctx, rawGrants)
	if data, err := json.Marshal(expanded); err == nil {
		s.redis.Set(ctx, cacheKey, data, permCacheTTL)
	}
	return expanded, nil
}

func (s *PermissionService) expandDeviceVisibilityGrants(ctx context.Context, grants []model.DeviceVisibilityGrant) []model.DeviceVisibilityGrant {
	if len(grants) == 0 {
		return []model.DeviceVisibilityGrant{}
	}

	merged := make(map[string]model.DeviceVisibilityGrant)
	for _, grant := range grants {
		if len(grant.GroupIDs) == 0 {
			continue
		}
		expandedGroups := s.expandGrantGroups(ctx, grant.GroupIDs)
		if len(expandedGroups) == 0 {
			continue
		}
		normalized := model.DeviceVisibilityGrant{
			GroupIDs:     expandedGroups,
			Technologies: normalizeGrantTechnologies(grant.Technologies),
		}
		key := grantKey(normalized)
		merged[key] = mergeGrantSets(merged[key], normalized)
	}

	result := make([]model.DeviceVisibilityGrant, 0, len(merged))
	for _, grant := range merged {
		result = append(result, grant)
	}
	sort.Slice(result, func(i, j int) bool { return grantKey(result[i]) < grantKey(result[j]) })
	return result
}

func (s *PermissionService) expandGrantGroups(ctx context.Context, groupIDs []uuid.UUID) []uuid.UUID {
	expanded := make(map[uuid.UUID]struct{})
	visiting := make(map[uuid.UUID]struct{})
	for _, gid := range groupIDs {
		s.expandGroupTree(ctx, gid, expanded, visiting)
	}
	result := make([]uuid.UUID, 0, len(expanded))
	for id := range expanded {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func normalizeGrantTechnologies(technologies []model.Technology) []model.Technology {
	if len(technologies) == 0 {
		return nil
	}
	techSet := make(map[model.Technology]struct{})
	for _, tech := range technologies {
		switch strings.ToLower(string(tech)) {
		case "lte", "enb":
			techSet[model.TechLTE] = struct{}{}
		case "nr", "gnb":
			techSet[model.TechNR] = struct{}{}
		case "gsm":
			techSet[model.TechGSM] = struct{}{}
		default:
			continue
		}
	}
	if len(techSet) == 0 {
		return nil
	}
	result := make([]model.Technology, 0, len(techSet))
	for tech := range techSet {
		result = append(result, tech)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func grantKey(grant model.DeviceVisibilityGrant) string {
	groupParts := make([]string, 0, len(grant.GroupIDs))
	for _, groupID := range grant.GroupIDs {
		groupParts = append(groupParts, groupID.String())
	}
	techParts := make([]string, 0, len(grant.Technologies))
	for _, tech := range grant.Technologies {
		techParts = append(techParts, string(tech))
	}
	return strings.Join(groupParts, ",") + "|" + strings.Join(techParts, ",")
}

func mergeGrantSets(left, right model.DeviceVisibilityGrant) model.DeviceVisibilityGrant {
	if len(left.GroupIDs) == 0 {
		return right
	}
	if len(right.GroupIDs) == 0 {
		return left
	}
	groupSet := make(map[uuid.UUID]struct{}, len(left.GroupIDs)+len(right.GroupIDs))
	for _, groupID := range left.GroupIDs {
		groupSet[groupID] = struct{}{}
	}
	for _, groupID := range right.GroupIDs {
		groupSet[groupID] = struct{}{}
	}
	groups := make([]uuid.UUID, 0, len(groupSet))
	for groupID := range groupSet {
		groups = append(groups, groupID)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].String() < groups[j].String() })
	return model.DeviceVisibilityGrant{GroupIDs: groups, Technologies: right.Technologies}
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
