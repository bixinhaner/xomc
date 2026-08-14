package device

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestBuildCurrentDeviceControlSummariesQueryIsBatchAndSourceAware(t *testing.T) {
	deviceIDs := []uuid.UUID{uuid.New(), uuid.New()}

	query, args, err := buildCurrentDeviceControlSummariesQuery(deviceIDs)

	require.NoError(t, err)
	require.Contains(t, query, "DISTINCT ON (deactivation.device_id)")
	require.Contains(t, query, "LEFT JOIN LATERAL")
	require.Contains(t, query, "device_geofence_bindings")
	require.Contains(t, query, "geofence_definitions")
	require.Contains(t, query, "deactivation.device_id IN")
	require.Contains(t, args, deviceIDs[0])
	require.Contains(t, args, deviceIDs[1])
	require.Contains(t, args, "activate")
	require.Contains(t, args, "deactivate")
	requireDollarPlaceholdersMatchArgs(t, query, args)
}

func TestCurrentDeviceControlFilterConditionUsesLatestActionAndRecovery(t *testing.T) {
	source := DeviceControlSourceGeofence
	condition := currentDeviceControlFilterCondition(DeviceFilter{
		ControlSource: &source,
		ControlPhases: []string{DeviceControlPhaseDeactivated, DeviceControlPhaseRecoveryFailed},
	})

	query, args, err := sq.Select("d.id").
		From("devices d").
		Where(sq.Eq{"d.carrier": "CMCC"}).
		Where(condition).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "EXISTS")
	require.Contains(t, query, "LEFT JOIN LATERAL")
	require.Contains(t, query, "latest.device_id = d.id")
	require.Contains(t, query, "recovery.status IN")
	require.Contains(t, args, "activate")
	require.Contains(t, args, "verified")
	require.Contains(t, args, "partial_failed")
	requireDollarPlaceholdersMatchArgs(t, query, args)
}

func TestCurrentDeviceControlFilterConditionRejectsUnknownSource(t *testing.T) {
	source := "alarm"
	condition := currentDeviceControlFilterCondition(DeviceFilter{ControlSource: &source})

	query, _, err := sq.Select("d.id").From("devices d").Where(condition).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "WHERE FALSE")
}

func TestResolveDeviceControlPhase(t *testing.T) {
	tests := []struct {
		name               string
		deactivationStatus string
		recoveryStatus     *string
		wantPhase          string
		wantVisible        bool
	}{
		{name: "queued", deactivationStatus: "pending", wantPhase: DeviceControlPhaseDeactivating, wantVisible: true},
		{name: "readback", deactivationStatus: "verifying", wantPhase: DeviceControlPhaseVerifying, wantVisible: true},
		{name: "owned", deactivationStatus: "verified", wantPhase: DeviceControlPhaseDeactivated, wantVisible: true},
		{name: "partial", deactivationStatus: "partial_failed", wantPhase: DeviceControlPhasePartialFailed, wantVisible: true},
		{name: "failed", deactivationStatus: "failed", wantPhase: DeviceControlPhaseFailed, wantVisible: true},
		{name: "recovering", deactivationStatus: "verified", recoveryStatus: stringPointer("executing"), wantPhase: DeviceControlPhaseRecovering, wantVisible: true},
		{name: "recovery failed", deactivationStatus: "verified", recoveryStatus: stringPointer("failed"), wantPhase: DeviceControlPhaseRecoveryFailed, wantVisible: true},
		{name: "recovered", deactivationStatus: "verified", recoveryStatus: stringPointer("verified"), wantVisible: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			phase, visible := resolveDeviceControlPhase(test.deactivationStatus, test.recoveryStatus)
			require.Equal(t, test.wantPhase, phase)
			require.Equal(t, test.wantVisible, visible)
		})
	}
}

func TestListDevicesWithInfoDegradesWhenControlSummaryFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	deviceID := uuid.New()
	list := model.NewListResponse([]DeviceWithInfo{{
		Device: model.Device{ID: deviceID},
	}}, 1, 1, 20)
	infoRepo.EXPECT().ComputeListStats(gomock.Any(), gomock.Any()).Return(nil, errors.New("stats unavailable"))
	infoRepo.EXPECT().ListDevicesWithInfo(gomock.Any(), gomock.Any()).Return(list, nil)

	service := NewDeviceService(nil, nil, nil, nil, zap.NewNop())
	service.deviceInfoRepo = infoRepo
	service.SetControlSummaryReader(failingControlSummaryReader{})

	result, err := service.ListDevicesWithInfo(context.Background(), DeviceFilter{})

	require.NoError(t, err)
	require.Same(t, list, result)
	require.Len(t, result.Items, 1)
	require.Nil(t, result.Items[0].ControlSummary)
}

type failingControlSummaryReader struct{}

func (failingControlSummaryReader) ListCurrentByDeviceIDs(
	context.Context,
	[]uuid.UUID,
) (map[uuid.UUID]DeviceControlSummary, error) {
	return nil, errors.New("summary unavailable")
}

func (failingControlSummaryReader) ListHistoryByDeviceID(
	context.Context,
	uuid.UUID,
	int,
	int,
) (*DeviceControlActionHistoryList, error) {
	return nil, errors.New("history unavailable")
}

func stringPointer(value string) *string {
	return &value
}

func requireDollarPlaceholdersMatchArgs(t *testing.T, query string, args []any) {
	t.Helper()
	matches := regexp.MustCompile(`\$(\d+)`).FindAllStringSubmatch(query, -1)
	require.NotEmpty(t, matches)
	seen := make(map[int]struct{}, len(matches))
	maxPlaceholder := 0
	for _, match := range matches {
		placeholder, err := strconv.Atoi(match[1])
		require.NoError(t, err)
		seen[placeholder] = struct{}{}
		if placeholder > maxPlaceholder {
			maxPlaceholder = placeholder
		}
	}
	require.Equal(t, len(args), maxPlaceholder, "SQL max placeholder must match argument count")
	require.Len(t, seen, len(args), "SQL placeholders must be contiguous and unique")
}
