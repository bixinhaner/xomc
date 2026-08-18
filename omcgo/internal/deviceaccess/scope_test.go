package deviceaccess

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/stretchr/testify/require"
)

type identityVisibilityStub struct {
	allowed      bool
	carrier      string
	serialNumber string
	groups       []uuid.UUID
}

func (s *identityVisibilityStub) CanAccessIdentity(
	_ context.Context,
	carrier, serialNumber string,
	groups []uuid.UUID,
) (bool, error) {
	s.carrier = carrier
	s.serialNumber = serialNumber
	s.groups = groups
	return s.allowed, nil
}

func TestBuildCarrierVisibilityQueryUsesCanonicalDeviceScope(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildCarrierVisibilityQuery("cmcc", []uuid.UUID{groupID})

	require.NoError(t, err)
	require.Contains(t, query, "FROM devices scope_device")
	require.Contains(t, query, "device_group_members")
	require.Contains(t, args, "cmcc")
	require.Contains(t, args, groupID)
}

func TestBuildFormalDeviceIdentityQueryDoesNotUseHistoricalScope(t *testing.T) {
	query, args, err := buildFormalDeviceIdentityQuery("cmcc", "SN-001")

	require.NoError(t, err)
	require.Contains(t, query, "FROM devices scope_device")
	require.NotContains(t, query, "device_registrations")
	require.NotContains(t, query, "device_group_members")
	require.Contains(t, args, "cmcc")
	require.Contains(t, args, "SN-001")
}

func TestApplyAccessIdentityVisibilityIncludesFormalAndPreregisteredScope(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	builder := sq.Select("c.id").From("device_access_candidates c")

	query, args, err := applyAccessIdentityVisibility(
		builder, "c.device_id", "c.carrier", "c.serial_number", []uuid.UUID{groupID},
	).PlaceholderFormat(sq.Dollar).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "device_group_members")
	require.Contains(t, query, "device_registrations scope_registration")
	require.Contains(t, query, "c.device_id IS NULL AND EXISTS")
	require.Contains(t, query, "scope_registration.carrier = c.carrier")
	require.Contains(t, args, groupID)
}

func TestBuildRegistrationCarrierVisibilityQueryUsesPreRegistrationGroup(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildRegistrationCarrierVisibilityQuery("cmcc", []uuid.UUID{groupID})

	require.NoError(t, err)
	require.Contains(t, query, "FROM device_registrations scope_registration")
	require.Contains(t, query, "scope_registration.group_id")
	require.Contains(t, args, "cmcc")
	require.Contains(t, args, groupID)
}

func TestBuildRegistrationCarrierVisibilityQueryIncludesLegacyUngroupedScope(t *testing.T) {
	defaultGroupID := uuid.MustParse(global.DefaultLevel2GroupID)

	query, _, err := buildRegistrationCarrierVisibilityQuery("cmcc", []uuid.UUID{defaultGroupID})

	require.NoError(t, err)
	require.Contains(t, query, "scope_registration.group_id IS NULL")
}
