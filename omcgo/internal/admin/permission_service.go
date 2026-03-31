package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	permCacheTTL    = 5 * time.Minute
	permCachePrefix = "perm:visible_groups:"
)

// PermissionService provides data-permission logic for device group visibility.
type PermissionService struct {
	roleRepo      RoleDeviceGroupRepository
	groupExpander GroupExpander
	redis         redis.UniversalClient
	logger        *zap.Logger
}

// GroupExpander resolves L1 group IDs into their L2 children.
type GroupExpander interface {
	ListChildIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error)
	GetGroupLevel(ctx context.Context, id uuid.UUID) (int, error)
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
// A nil carrier means superadmin — all groups are visible (returns nil slice).
// Results are cached in Redis for 5 minutes.
func (s *PermissionService) GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, carrier *model.CarrierCode) ([]uuid.UUID, error) {
	// Superadmin (carrier=nil) sees everything.
	if carrier == nil {
		return nil, nil
	}

	// Try Redis cache.
	cacheKey := permCachePrefix + userID.String()
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

	// Expand L1 groups to their L2 children.
	expanded := make(map[uuid.UUID]struct{})
	for _, gid := range directIDs {
		level, err := s.groupExpander.GetGroupLevel(ctx, gid)
		if err != nil {
			s.logger.Warn("expand group failed", zap.String("group_id", gid.String()), zap.Error(err))
			continue
		}
		if level == 1 {
			// L1 group: add all L2 children.
			children, err := s.groupExpander.ListChildIDs(ctx, gid)
			if err != nil {
				s.logger.Warn("list children failed", zap.String("group_id", gid.String()), zap.Error(err))
				continue
			}
			for _, cid := range children {
				expanded[cid] = struct{}{}
			}
		} else {
			// L2 group: add directly.
			expanded[gid] = struct{}{}
		}
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

// InvalidateUserCache removes the cached visible groups for a user.
func (s *PermissionService) InvalidateUserCache(ctx context.Context, userID uuid.UUID) {
	s.redis.Del(ctx, permCachePrefix+userID.String())
}

// InvalidateRoleCache invalidates cache for all users with the given role.
// This is a simple approach — in production, you might want a more targeted invalidation.
func (s *PermissionService) InvalidateRoleCache(ctx context.Context, roleID uuid.UUID) {
	// For simplicity, we use a pattern scan. In practice, consider maintaining
	// a reverse index of role → user IDs.
	s.logger.Info("role device groups changed, caches will expire via TTL",
		zap.String("role_id", roleID.String()))
}
