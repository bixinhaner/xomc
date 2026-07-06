package admin

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRoleDeviceGroupRepo struct {
	visible []uuid.UUID
	grants  []model.DeviceVisibilityGrant
}

func (r *fakeRoleDeviceGroupRepo) GetGroupIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *fakeRoleDeviceGroupRepo) SetGroupIDs(context.Context, uuid.UUID, []uuid.UUID) error {
	return nil
}

func (r *fakeRoleDeviceGroupRepo) GetUserVisibleGroupIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return r.visible, nil
}

func (r *fakeRoleDeviceGroupRepo) GetUserVisibleDeviceGrants(context.Context, uuid.UUID) ([]model.DeviceVisibilityGrant, error) {
	if r.grants != nil {
		return r.grants, nil
	}
	grants := make([]model.DeviceVisibilityGrant, 0, len(r.visible))
	for _, gid := range r.visible {
		grants = append(grants, model.DeviceVisibilityGrant{GroupIDs: []uuid.UUID{gid}})
	}
	return grants, nil
}

func (r *fakeRoleDeviceGroupRepo) GetDeviceGroupData(context.Context, uuid.UUID) (*RoleDeviceGroupData, error) {
	return &RoleDeviceGroupData{}, nil
}

func (r *fakeRoleDeviceGroupRepo) SetDeviceGroupData(context.Context, uuid.UUID, RoleDeviceGroupData) error {
	return nil
}

func (r *fakeRoleDeviceGroupRepo) ListRolesByGroupIDs(context.Context, []uuid.UUID) ([]Role, error) {
	return nil, nil
}

type fakeGroupExpander struct {
	children map[uuid.UUID][]uuid.UUID
}

func (e *fakeGroupExpander) ListChildIDs(_ context.Context, parentID uuid.UUID) ([]uuid.UUID, error) {
	return e.children[parentID], nil
}

func TestPermissionService_ExpandsSelectedGroupRecursively(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	userID := uuid.New()
	rootID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()

	svc := NewPermissionService(
		&fakeRoleDeviceGroupRepo{visible: []uuid.UUID{rootID}},
		&fakeGroupExpander{children: map[uuid.UUID][]uuid.UUID{
			rootID:  {childID},
			childID: {grandchildID},
		}},
		redisClient,
		zap.NewNop(),
	)

	visible, err := svc.GetUserVisibleGroupIDs(context.Background(), userID, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{rootID, childID, grandchildID}, visible)
}

func TestPermissionService_SuperadminSeesAllWithoutGroupExpansion(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	svc := NewPermissionService(
		&fakeRoleDeviceGroupRepo{visible: []uuid.UUID{uuid.New()}},
		&fakeGroupExpander{},
		redisClient,
		zap.NewNop(),
	)

	visible, err := svc.GetUserVisibleGroupIDs(context.Background(), uuid.New(), true)
	require.NoError(t, err)
	assert.Nil(t, visible)
}

func TestPermissionService_ExpandsDeviceVisibilityGrants(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	userID := uuid.New()
	rootID := uuid.New()
	childID := uuid.New()

	svc := NewPermissionService(
		&fakeRoleDeviceGroupRepo{grants: []model.DeviceVisibilityGrant{{GroupIDs: []uuid.UUID{rootID}, Technologies: []model.Technology{model.TechLTE}}}},
		&fakeGroupExpander{children: map[uuid.UUID][]uuid.UUID{rootID: {childID}}},
		redisClient,
		zap.NewNop(),
	)

	grants, err := svc.GetUserVisibleDeviceGrants(context.Background(), userID, false)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.ElementsMatch(t, []uuid.UUID{rootID, childID}, grants[0].GroupIDs)
	assert.Equal(t, []model.Technology{model.TechLTE}, grants[0].Technologies)
}

func TestNormalizeDeviceVisibilityTechnologiesTreatsEmptyAndInvalidAsUnrestricted(t *testing.T) {
	assert.Nil(t, normalizeGrantTechnologies(nil))
	assert.Nil(t, normalizeGrantTechnologies([]model.Technology{}))
	assert.Nil(t, normalizeGrantTechnologies([]model.Technology{model.Technology(" "), model.Technology("bogus")}))

	assert.Nil(t, normalizeNetworkTypesForDeviceVisibility(nil))
	assert.Nil(t, normalizeNetworkTypesForDeviceVisibility([]string{}))
	assert.Nil(t, normalizeNetworkTypesForDeviceVisibility([]string{" ", "bogus"}))
}
