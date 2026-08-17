package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

type runtimeScopeUserReaderStub struct{ user *admin.User }

func (s runtimeScopeUserReaderStub) GetByID(context.Context, uuid.UUID) (*admin.User, error) {
	return s.user, nil
}

type runtimeScopePermissionsStub struct{ groups []uuid.UUID }

func (s runtimeScopePermissionsStub) GetUserVisibleGroupIDs(context.Context, uuid.UUID, bool) ([]uuid.UUID, error) {
	return s.groups, nil
}

type runtimeScopeGroupReaderStub struct{ byDevice map[uuid.UUID][]uuid.UUID }

func (s runtimeScopeGroupReaderStub) GetDeviceGroupIDs(_ context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	return s.byDevice[deviceID], nil
}

type runtimeScopeDeviceLookupStub struct{ bySN map[string]*model.Device }

func (s runtimeScopeDeviceLookupStub) GetBySerialNumber(_ context.Context, serialNumber string) (*model.Device, error) {
	return s.bySN[serialNumber], nil
}

func TestRuntimeDeviceScopeAuthorizerRejectsRevokedAlarmGroup(t *testing.T) {
	t.Parallel()
	userID, selectedGroup, currentGroup := uuid.New(), uuid.New(), uuid.New()
	authorizer := NewRuntimeDeviceScopeAuthorizer(
		runtimeScopeUserReaderStub{user: &admin.User{ID: userID, Source: admin.UserSourceAdmin, Status: admin.UserStatusActive}},
		runtimeScopePermissionsStub{groups: []uuid.UUID{currentGroup}},
		runtimeScopeGroupReaderStub{},
		runtimeScopeDeviceLookupStub{},
	)

	err := authorizer.AuthorizeDeviceScope(context.Background(), userID, nil, []uuid.UUID{selectedGroup})
	require.Error(t, err)
	require.True(t, errors.Is(err, commonerrors.ErrForbidden))
}

func TestRuntimeDeviceScopeAuthorizerRechecksKPIDeviceSerials(t *testing.T) {
	t.Parallel()
	userID, deviceID, visibleGroup := uuid.New(), uuid.New(), uuid.New()
	authorizer := NewRuntimeDeviceScopeAuthorizer(
		runtimeScopeUserReaderStub{user: &admin.User{ID: userID, Source: admin.UserSourceAdmin, Status: admin.UserStatusActive}},
		runtimeScopePermissionsStub{groups: []uuid.UUID{visibleGroup}},
		runtimeScopeGroupReaderStub{byDevice: map[uuid.UUID][]uuid.UUID{deviceID: {visibleGroup}}},
		runtimeScopeDeviceLookupStub{bySN: map[string]*model.Device{"SN-1": {ID: deviceID}}},
	)

	require.NoError(t, authorizer.AuthorizeSerialNumbers(context.Background(), userID, []string{"SN-1"}))
}

func TestRuntimeDeviceScopeAuthorizerAllowsBuiltInCreator(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	authorizer := NewRuntimeDeviceScopeAuthorizer(
		runtimeScopeUserReaderStub{user: &admin.User{ID: userID, Source: admin.UserSourceBuiltIn, Status: admin.UserStatusActive}},
		runtimeScopePermissionsStub{groups: nil},
		runtimeScopeGroupReaderStub{},
		runtimeScopeDeviceLookupStub{},
	)

	require.NoError(t, authorizer.AuthorizeSerialNumbers(context.Background(), userID, nil))
}
