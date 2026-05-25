package device

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type stubRebootOfflineRepo struct {
	mu             sync.Mutex
	devicesBySN    map[string]*model.Device
	getErr         error
	updateErr      error
	updateCalls    []updateOnlineCall
	getCallCount   int
}

type updateOnlineCall struct {
	ID       uuid.UUID
	IsOnline bool
}

func (s *stubRebootOfflineRepo) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCallCount++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if d, ok := s.devicesBySN[sn]; ok {
		return d, nil
	}
	return nil, nil
}

func (s *stubRebootOfflineRepo) UpdateOnlineStatus(_ context.Context, id uuid.UUID, isOnline bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCalls = append(s.updateCalls, updateOnlineCall{ID: id, IsOnline: isOnline})
	return s.updateErr
}

func newRebootResponseEvent(t *testing.T, sn string, method string) event.Event {
	t.Helper()
	payload := map[string]interface{}{
		"device_sn": sn,
		"method":    method,
	}
	evt, err := event.NewEvent(event.SubjectCommandRebootResponse, payload)
	require.NoError(t, err)
	return evt
}

func Test_RebootResponseOfflineMarker_MarksDeviceOffline(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-TEST-001": {ID: deviceID, SerialNumber: "BLQ-TEST-001", IsOnline: true},
		},
	}
	m := NewRebootResponseOfflineMarker(repo, NewSequencer(), nil)

	require.NoError(t, m.handle(context.Background(), newRebootResponseEvent(t, "BLQ-TEST-001", "Reboot")))

	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, deviceID, repo.updateCalls[0].ID)
	assert.False(t, repo.updateCalls[0].IsOnline, "UpdateOnlineStatus should be called with false")
}

func Test_RebootResponseOfflineMarker_EmptySNSkipped(t *testing.T) {
	repo := &stubRebootOfflineRepo{}
	m := NewRebootResponseOfflineMarker(repo, NewSequencer(), nil)

	require.NoError(t, m.handle(context.Background(), newRebootResponseEvent(t, "", "Reboot")))

	assert.Zero(t, repo.getCallCount, "empty SN should be skipped without repo lookup")
	assert.Empty(t, repo.updateCalls)
}

func Test_RebootResponseOfflineMarker_DeviceNotFoundIsSilentNoop(t *testing.T) {
	repo := &stubRebootOfflineRepo{devicesBySN: map[string]*model.Device{}}
	m := NewRebootResponseOfflineMarker(repo, NewSequencer(), nil)

	require.NoError(t, m.handle(context.Background(), newRebootResponseEvent(t, "UNKNOWN", "Reboot")))

	assert.Equal(t, 1, repo.getCallCount)
	assert.Empty(t, repo.updateCalls)
}

func Test_RebootResponseOfflineMarker_LookupFailureIsLoggedNotPropagated(t *testing.T) {
	repo := &stubRebootOfflineRepo{getErr: errors.New("db dead")}
	m := NewRebootResponseOfflineMarker(repo, NewSequencer(), nil)

	err := m.handle(context.Background(), newRebootResponseEvent(t, "BLQ-X", "Reboot"))

	assert.NoError(t, err, "lookup failure must not propagate as event handler error")
	assert.Empty(t, repo.updateCalls)
}

func Test_RebootResponseOfflineMarker_UpdateFailureIsLoggedNotPropagated(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-Y": {ID: deviceID, SerialNumber: "BLQ-Y", IsOnline: true},
		},
		updateErr: errors.New("update dead"),
	}
	m := NewRebootResponseOfflineMarker(repo, NewSequencer(), nil)

	err := m.handle(context.Background(), newRebootResponseEvent(t, "BLQ-Y", "Reboot"))

	assert.NoError(t, err, "update failure must not propagate")
	require.Len(t, repo.updateCalls, 1, "update should have been attempted once")
}

func Test_RebootResponseOfflineMarker_AlreadyOfflineStillCalls(t *testing.T) {
	// 已经离线时也调用 UpdateOnlineStatus(false)，避免分支判断；幂等。
	deviceID := uuid.New()
	repo := &stubRebootOfflineRepo{
		devicesBySN: map[string]*model.Device{
			"BLQ-Z": {ID: deviceID, SerialNumber: "BLQ-Z", IsOnline: false},
		},
	}
	m := NewRebootResponseOfflineMarker(repo, NewSequencer(), nil)

	require.NoError(t, m.handle(context.Background(), newRebootResponseEvent(t, "BLQ-Z", "Reboot")))

	require.Len(t, repo.updateCalls, 1)
	assert.False(t, repo.updateCalls[0].IsOnline)
}
