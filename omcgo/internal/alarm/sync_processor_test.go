package alarm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestProcessSync_BackfillsDeviceFieldsForAddedAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceBySN: map[string]*model.Device{
			"SN-SYNC-001": {
				ID:           deviceID,
				SerialNumber: "SN-SYNC-001",
				Carrier:      model.CarrierCode("cmcc"),
				Technology:   model.TechLTE,
			},
		},
	})

	params := []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "70011"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime", Value: "2026-05-19T06:04:11Z"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.EventType", Value: "Equipment Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.ProbableCause", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.SpecificProblem", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", Value: "Major"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AdditionalText", Value: "LTE Cell 1"},
	}

	result := processor.processSync(context.Background(), "SN-SYNC-001", params)
	require.Equal(t, 1, result.Added)
	require.Zero(t, result.FailedAdd)

	alarm, err := store.GetActiveByDeviceAndIdentifier(context.Background(), "SN-SYNC-001", "70011")
	require.NoError(t, err)
	require.NotNil(t, alarm)
	assert.Equal(t, deviceID, alarm.DeviceID)
	assert.Equal(t, model.AlarmMajor, alarm.Severity)
	require.NotNil(t, alarm.Technology)
	assert.Equal(t, string(model.TechLTE), *alarm.Technology)
	assert.Equal(t, model.CarrierCode("cmcc"), alarm.Carrier)
}

func TestProcessSync_UsesExistingAlarmDeviceFieldsWhenLookupMissing(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	existingTech := string(model.TechNR)
	store.active[uuid.New()] = &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		DeviceSN:        "SN-SYNC-002",
		Carrier:         model.CarrierCode("cucc"),
		AlarmIdentifier: "11184",
		Description:     "Cell unavailable",
		Severity:        model.AlarmSeverity(1),
		Status:          model.AlarmActive,
		RaisedAt:        time.Now(),
		Technology:      &existingTech,
		AdditionalInfo:  map[string]string{},
	}
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop())

	params := []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "70011"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime", Value: "2026-05-19T06:04:11Z"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.EventType", Value: "Equipment Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.ProbableCause", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.SpecificProblem", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", Value: "Major"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AdditionalText", Value: "LTE Cell 1"},
	}

	result := processor.processSync(context.Background(), "SN-SYNC-002", params)
	require.Equal(t, 1, result.Added)

	alarm, err := store.GetActiveByDeviceAndIdentifier(context.Background(), "SN-SYNC-002", "70011")
	require.NoError(t, err)
	require.NotNil(t, alarm)
	assert.Equal(t, deviceID, alarm.DeviceID)
	require.NotNil(t, alarm.Technology)
	assert.Equal(t, existingTech, *alarm.Technology)
	assert.Equal(t, model.CarrierCode("cucc"), alarm.Carrier)
}