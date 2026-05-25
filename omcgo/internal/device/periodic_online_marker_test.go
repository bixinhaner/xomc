package device

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func newPeriodicEvent(t *testing.T, sn string) event.Event {
	t.Helper()
	payload := map[string]interface{}{
		"device_id": map[string]interface{}{
			"SerialNumber": sn,
		},
	}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)
	return evt
}

func newValueChangeEvent(t *testing.T, sn string) event.Event {
	t.Helper()
	payload := map[string]interface{}{
		"device_id": map[string]interface{}{
			"SerialNumber": sn,
		},
	}
	evt, err := event.NewEvent(event.SubjectDeviceValueChange, payload)
	require.NoError(t, err)
	return evt
}

func Test_PeriodicOnlineMarker_WritesTrueOnPeriodic(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-P-001": {ID: deviceID, SerialNumber: "BLQ-P-001", IsOnline: false},
		},
	}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	require.NoError(t, m.handle(context.Background(), newPeriodicEvent(t, "BLQ-P-001")))

	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, deviceID, repo.updateCalls[0].ID)
	assert.True(t, repo.updateCalls[0].IsOnline, "UpdateOnlineStatus must be called with true on PERIODIC")
}

func Test_PeriodicOnlineMarker_WritesTrueOnValueChange(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-VC-002": {ID: deviceID, SerialNumber: "BLQ-VC-002", IsOnline: false},
		},
	}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	require.NoError(t, m.handle(context.Background(), newValueChangeEvent(t, "BLQ-VC-002")))

	require.Len(t, repo.updateCalls, 1)
	assert.True(t, repo.updateCalls[0].IsOnline)
}

func Test_PeriodicOnlineMarker_EmptySNSkipped(t *testing.T) {
	repo := &stubRebootOfflineRepo{}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	require.NoError(t, m.handle(context.Background(), newPeriodicEvent(t, "")))

	assert.Zero(t, repo.getCallCount, "empty SN should be skipped without repo lookup")
	assert.Empty(t, repo.updateCalls)
}

func Test_PeriodicOnlineMarker_DeviceNotFoundIsSilentNoop(t *testing.T) {
	repo := &stubRebootOfflineRepo{devicesBySN: map[string]*model.Device{}}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	require.NoError(t, m.handle(context.Background(), newPeriodicEvent(t, "UNKNOWN-SN")))

	assert.Equal(t, 1, repo.getCallCount)
	assert.Empty(t, repo.updateCalls, "device not found → no UpdateOnlineStatus call")
}

func Test_PeriodicOnlineMarker_LookupFailureIsLoggedNotPropagated(t *testing.T) {
	repo := &stubRebootOfflineRepo{getErr: errors.New("db dead")}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	err := m.handle(context.Background(), newPeriodicEvent(t, "BLQ-X"))

	assert.NoError(t, err, "lookup failure must not propagate to EventBus")
	assert.Empty(t, repo.updateCalls)
}

func Test_PeriodicOnlineMarker_UpdateFailureIsLoggedNotPropagated(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-U-FAIL": {ID: deviceID, SerialNumber: "BLQ-U-FAIL", IsOnline: false},
		},
		updateErr: errors.New("update dead"),
	}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	err := m.handle(context.Background(), newPeriodicEvent(t, "BLQ-U-FAIL"))

	assert.NoError(t, err, "update failure must not propagate")
	require.Len(t, repo.updateCalls, 1, "update should have been attempted once")
}

func Test_PeriodicOnlineMarker_AlreadyOnlineStillCalls(t *testing.T) {
	// 已经在线时也调用 UpdateOnlineStatus(true)，幂等。简化为不在 handle 内
	// 做"先读 device.IsOnline 决定是否写"的优化（避免 DB 视角与 cache 视角不一致问题）。
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-AO": {ID: deviceID, SerialNumber: "BLQ-AO", IsOnline: true},
		},
	}
	m := NewPeriodicOnlineMarker(repo, NewSequencer(), nil, nil)

	require.NoError(t, m.handle(context.Background(), newPeriodicEvent(t, "BLQ-AO")))

	require.Len(t, repo.updateCalls, 1)
	assert.True(t, repo.updateCalls[0].IsOnline)
}
