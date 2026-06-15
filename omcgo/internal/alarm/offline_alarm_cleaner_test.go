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

// #358：阈值/周期/批量可配——SetThreshold/SetInterval/SetBatchSize 注入后，
// sweep 的 cutoff 与 FindOfflineDevicesBefore limit 须随配置变化，而非硬编码 Default*。
func TestOfflineAlarmCleaner_ConfiguredThresholdOverridesDefault(t *testing.T) {
	deviceID := uuid.New()
	repo := &stubOfflineAlarmDeviceRepo{
		devices: []*model.Device{{ID: deviceID, SerialNumber: "SN-OFF-CFG"}},
	}
	store := &stubOfflineAlarmStore{
		alarmsBySN: map[string][]*model.Alarm{
			"SN-OFF-CFG": {{ID: uuid.New(), DeviceID: deviceID, DeviceSN: "SN-OFF-CFG", AlarmIdentifier: "A1"}},
		},
	}
	clearer := &capturingOfflineAlarmClearer{}
	cleaner := NewOfflineAlarmCleaner(repo, store, clearer, zap.NewNop())

	// 运营商把收敛阈值降到分钟级（120s），周期 30s，批量 50。
	const cfgThreshold = 120 * time.Second
	const cfgInterval = 30 * time.Second
	const cfgBatch = 50
	cleaner.SetThreshold(cfgThreshold)
	cleaner.SetInterval(cfgInterval)
	cleaner.SetBatchSize(cfgBatch)

	// getter 反映注入后的生效值（启动日志依赖）。
	assert.Equal(t, cfgThreshold, cleaner.Threshold())
	assert.Equal(t, cfgInterval, cleaner.Interval())
	assert.Equal(t, cfgBatch, cleaner.BatchSize())

	cleaner.sweep(context.Background())

	// cutoff 须 = now - 配置阈值（而非 now - DefaultOfflineAlarmCleanupThreshold=1h）。
	assert.Equal(t, cfgBatch, repo.limit)
	assert.WithinDuration(t, time.Now().Add(-cfgThreshold), repo.cutoff, 2*time.Second)
	// 反向断言：cutoff 明显不同于默认 1h 阈值算出的 cutoff（差约 1h-120s）。
	defaultCutoff := time.Now().Add(-DefaultOfflineAlarmCleanupThreshold)
	assert.Greater(t, repo.cutoff.Sub(defaultCutoff), 30*time.Minute,
		"配置阈值生效时 cutoff 应明显晚于默认 1h 阈值的 cutoff")
	require.Len(t, clearer.cleared, 1)
}

// #358：非法（<=0）配置值不应覆盖默认，setter 须忽略，保持向后兼容。
func TestOfflineAlarmCleaner_NonPositiveConfigKeepsDefaults(t *testing.T) {
	cleaner := NewOfflineAlarmCleaner(&stubOfflineAlarmDeviceRepo{}, &stubOfflineAlarmStore{}, &capturingOfflineAlarmClearer{}, zap.NewNop())

	cleaner.SetThreshold(0)
	cleaner.SetThreshold(-5 * time.Second)
	cleaner.SetInterval(0)
	cleaner.SetBatchSize(-1)

	assert.Equal(t, DefaultOfflineAlarmCleanupThreshold, cleaner.Threshold())
	assert.Equal(t, DefaultOfflineAlarmCleanupInterval, cleaner.Interval())
	assert.Equal(t, DefaultOfflineAlarmCleanupBatchSize, cleaner.BatchSize())
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
