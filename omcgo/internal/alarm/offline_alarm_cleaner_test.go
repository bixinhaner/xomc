package alarm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubOfflineAlarmDeviceRepo struct {
	devices []*model.Device
	err     error
	cutoff  time.Time
	limit   int
}

func (s *stubOfflineAlarmDeviceRepo) FindOfflineDevicesBefore(_ context.Context, cutoff time.Time, limit int) ([]*model.Device, error) {
	s.cutoff = cutoff
	s.limit = limit
	if s.err != nil {
		return nil, s.err
	}
	return s.devices, nil
}

type stubOfflineAlarmStore struct {
	alarmsBySN map[string][]*model.Alarm
	errBySN    map[string]error
}

func (s *stubOfflineAlarmStore) GetActiveByDeviceSN(_ context.Context, deviceSN string) ([]*model.Alarm, error) {
	if err := s.errBySN[deviceSN]; err != nil {
		return nil, err
	}
	return s.alarmsBySN[deviceSN], nil
}

type capturingOfflineAlarmClearer struct {
	cleared []*model.Alarm
	errByID map[uuid.UUID]error
}

func (c *capturingOfflineAlarmClearer) ClearBySync(_ context.Context, alarm *model.Alarm) error {
	if err := c.errByID[alarm.ID]; err != nil {
		return err
	}
	c.cleared = append(c.cleared, alarm)
	return nil
}

func TestOfflineAlarmCleanerSweep_ClearsTimedOutOfflineDeviceAlarms(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubOfflineAlarmDeviceRepo{
		devices: []*model.Device{{ID: deviceID, SerialNumber: "SN-OFF-001"}},
	}
	store := &stubOfflineAlarmStore{
		alarmsBySN: map[string][]*model.Alarm{
			"SN-OFF-001": {
				{ID: uuid.New(), DeviceID: deviceID, DeviceSN: "SN-OFF-001", AlarmIdentifier: "A1"},
				{ID: uuid.New(), DeviceID: deviceID, DeviceSN: "SN-OFF-001", AlarmIdentifier: "A2"},
			},
		},
	}
	clearer := &capturingOfflineAlarmClearer{}
	cleaner := NewOfflineAlarmCleaner(repo, store, clearer, zap.NewNop())

	cleaner.sweep(context.Background())

	require.Len(t, clearer.cleared, 2)
	for _, alarm := range clearer.cleared {
		require.NotNil(t, alarm.ClearedBy)
		assert.Equal(t, offlineAlarmCleanupClearedBy, *alarm.ClearedBy)
		require.NotNil(t, alarm.ClearNote)
		assert.Contains(t, *alarm.ClearNote, "device offline for over")
	}
	assert.Equal(t, DefaultOfflineAlarmCleanupBatchSize, repo.limit)
	assert.WithinDuration(t, time.Now().Add(-DefaultOfflineAlarmCleanupThreshold), repo.cutoff, 2*time.Second)
}

func TestOfflineAlarmCleanerSweep_ContinuesWhenOneDeviceLookupFails(t *testing.T) {
	repo := &stubOfflineAlarmDeviceRepo{
		devices: []*model.Device{
			{ID: uuid.New(), SerialNumber: "SN-BAD"},
			{ID: uuid.New(), SerialNumber: "SN-GOOD"},
		},
	}
	goodAlarm := &model.Alarm{ID: uuid.New(), DeviceSN: "SN-GOOD", AlarmIdentifier: "GOOD-1"}
	store := &stubOfflineAlarmStore{
		alarmsBySN: map[string][]*model.Alarm{"SN-GOOD": {goodAlarm}},
		errBySN:    map[string]error{"SN-BAD": errors.New("query failed")},
	}
	clearer := &capturingOfflineAlarmClearer{}
	cleaner := NewOfflineAlarmCleaner(repo, store, clearer, zap.NewNop())

	cleaner.sweep(context.Background())

	require.Len(t, clearer.cleared, 1)
	assert.Equal(t, goodAlarm.ID, clearer.cleared[0].ID)
}

func TestOfflineAlarmCleanerSweep_StopsCurrentDeviceOnClearFailure(t *testing.T) {
	first := &model.Alarm{ID: uuid.New(), DeviceSN: "SN-OFF-002", AlarmIdentifier: "A1"}
	second := &model.Alarm{ID: uuid.New(), DeviceSN: "SN-OFF-002", AlarmIdentifier: "A2"}
	repo := &stubOfflineAlarmDeviceRepo{
		devices: []*model.Device{{ID: uuid.New(), SerialNumber: "SN-OFF-002"}},
	}
	store := &stubOfflineAlarmStore{
		alarmsBySN: map[string][]*model.Alarm{"SN-OFF-002": {first, second}},
	}
	clearer := &capturingOfflineAlarmClearer{errByID: map[uuid.UUID]error{second.ID: errors.New("archive failed")}}
	cleaner := NewOfflineAlarmCleaner(repo, store, clearer, zap.NewNop())

	cleared, err := cleaner.clearDeviceAlarms(context.Background(), repo.devices[0])

	require.Error(t, err)
	assert.Equal(t, 1, cleared)
	require.Len(t, clearer.cleared, 1)
	assert.Equal(t, first.ID, clearer.cleared[0].ID)
}
