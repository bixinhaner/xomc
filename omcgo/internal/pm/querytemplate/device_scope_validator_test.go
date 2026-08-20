package querytemplate

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

type deviceScopePermissionsStub struct{ groups []uuid.UUID }

func (s deviceScopePermissionsStub) GetUserVisibleGroupIDs(context.Context, uuid.UUID, bool) ([]uuid.UUID, error) {
	return s.groups, nil
}

type deviceScopeLookupStub struct{ devices map[string]*model.Device }

func (s deviceScopeLookupStub) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	return s.devices[sn], nil
}

type deviceScopeGroupReaderStub struct{ memberships map[uuid.UUID][]uuid.UUID }

func (s deviceScopeGroupReaderStub) GetDeviceGroupIDs(_ context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	return s.memberships[id], nil
}

func TestDeviceScopeValidatorRejectsRegularReportWithoutDevices(t *testing.T) {
	validator := NewDeviceScopeValidator(deviceScopePermissionsStub{}, deviceScopeLookupStub{}, deviceScopeGroupReaderStub{})
	err := validator.ValidatePayload(context.Background(), []byte(`{"regular_report":{"enabled":true}}`), uuid.New(), false)
	require.ErrorContains(t, err, "requires at least one device")
}

func TestDeviceScopeValidatorRejectsDeviceOutsideVisibleGroups(t *testing.T) {
	visibleGroupID := uuid.New()
	deviceGroupID := uuid.New()
	deviceID := uuid.New()
	validator := NewDeviceScopeValidator(
		deviceScopePermissionsStub{groups: []uuid.UUID{visibleGroupID}},
		deviceScopeLookupStub{devices: map[string]*model.Device{"SN-1": {ID: deviceID}}},
		deviceScopeGroupReaderStub{memberships: map[uuid.UUID][]uuid.UUID{deviceID: {deviceGroupID}}},
	)
	err := validator.ValidatePayload(context.Background(), []byte(`{
		"device_sns":["SN-1"],
		"regular_report":{"enabled":true}
	}`), uuid.New(), false)
	require.Error(t, err)
}

func TestDeviceScopeValidatorAllowsVisibleDevice(t *testing.T) {
	groupID := uuid.New()
	deviceID := uuid.New()
	validator := NewDeviceScopeValidator(
		deviceScopePermissionsStub{groups: []uuid.UUID{groupID}},
		deviceScopeLookupStub{devices: map[string]*model.Device{"SN-1": {ID: deviceID}}},
		deviceScopeGroupReaderStub{memberships: map[uuid.UUID][]uuid.UUID{deviceID: {groupID}}},
	)
	require.NoError(t, validator.ValidatePayload(context.Background(), []byte(`{
		"device_sns":["SN-1"],
		"regular_report":{"enabled":true}
	}`), uuid.New(), false))
}
