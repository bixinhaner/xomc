package alarm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm/definition"
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

func TestProcessSync_DoesNotClearOMCOwnedAlarmsMissingFromDeviceTable(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	omcAlarm := &model.Alarm{
		ID: uuid.New(), DeviceID: deviceID, DeviceSN: "SN-SYNC-OMC",
		Carrier: model.CarrierCMCC, AlarmIdentifier: AlarmCodeGeofenceLocationOutside,
		AlarmType: "geofence", AlarmSource: strPtr("omc"), Status: model.AlarmActive,
		RaisedAt: time.Now(), AdditionalInfo: map[string]string{},
	}
	deviceAlarm := &model.Alarm{
		ID: uuid.New(), DeviceID: deviceID, DeviceSN: "SN-SYNC-OMC",
		Carrier: model.CarrierCMCC, AlarmIdentifier: "70011",
		AlarmType: "equipment", AlarmSource: strPtr("TR069"), Status: model.AlarmActive,
		RaisedAt: time.Now(), AdditionalInfo: map[string]string{},
	}
	store.active[omcAlarm.ID] = omcAlarm
	store.active[deviceAlarm.ID] = deviceAlarm
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop())

	result := processor.processSync(context.Background(), "SN-SYNC-OMC", nil)

	require.Equal(t, 1, result.Cleared)
	require.Contains(t, store.active, omcAlarm.ID)
	require.NotContains(t, store.active, deviceAlarm.ID)
}

func TestProcessSync_AddsDistinctAlarmsForSameIdentifierWithDifferentAdditionalInformation(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceBySN: map[string]*model.Device{
			"SN-SYNC-003": {
				ID:           deviceID,
				SerialNumber: "SN-SYNC-003",
				Carrier:      model.CarrierCode("cmcc"),
				Technology:   model.TechLTE,
			},
		},
	})

	params := []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "11184"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime", Value: "2026-05-19T06:04:11Z"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", Value: "Major"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.SpecificProblem", Value: "Cell unavailable"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AdditionalInformation", Value: "cell=1"},
		{Name: "Device.FaultMgmt.CurrentAlarm.2.AlarmIdentifier", Value: "11184"},
		{Name: "Device.FaultMgmt.CurrentAlarm.2.AlarmRaisedTime", Value: "2026-05-19T06:05:11Z"},
		{Name: "Device.FaultMgmt.CurrentAlarm.2.PerceivedSeverity", Value: "Major"},
		{Name: "Device.FaultMgmt.CurrentAlarm.2.SpecificProblem", Value: "Cell unavailable"},
		{Name: "Device.FaultMgmt.CurrentAlarm.2.AdditionalInformation", Value: "cell=2"},
	}

	result := processor.processSync(context.Background(), "SN-SYNC-003", params)
	require.Equal(t, 2, result.Added)
	require.Zero(t, result.FailedAdd)
	assert.Len(t, store.active, 2)

	seen := map[string]bool{}
	for _, alarm := range store.active {
		seen[alarm.AdditionalInfo["additional_information"]] = true
	}
	assert.True(t, seen["cell=1"])
	assert.True(t, seen["cell=2"])
}

func TestProcessSync_OverridesSeverityFromDefinitionRegistry(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	registry := newTestAlarmDefRegistry(t, definition.ResolvedDefinition{
		AlarmDefinition: definition.AlarmDefinition{Identifier: "70011"},
		SeverityCode:    31004,
		SeverityName:    "Warning",
	})
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop()).
		WithDeviceReader(&rcvMockDeviceReader{
			deviceBySN: map[string]*model.Device{
				"SN-SYNC-DEF": {
					ID:           deviceID,
					SerialNumber: "SN-SYNC-DEF",
					Carrier:      model.CarrierCode("cmcc"),
					Technology:   model.TechLTE,
				},
			},
		}).
		WithAlarmDefRegistry(registry)

	params := []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "70011"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime", Value: "2026-06-05T09:01:00Z"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.EventType", Value: "Equipment Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.ProbableCause", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.SpecificProblem", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", Value: "Critical"},
	}

	result := processor.processSync(context.Background(), "SN-SYNC-DEF", params)
	require.Equal(t, 1, result.Added)

	alarm, err := store.GetActiveByDeviceAndIdentifier(context.Background(), "SN-SYNC-DEF", "70011")
	require.NoError(t, err)
	require.NotNil(t, alarm)
	assert.Equal(t, model.AlarmSeverity(31004), alarm.Severity)
}

func TestProcessSync_DropsUnknownAlarmWhenProductDisablesFallback(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	registry := newTestAlarmDefRegistry(t)
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop()).
		WithDeviceReader(&rcvMockDeviceReader{
			deviceBySN: map[string]*model.Device{
				"SN-SYNC-BM": {
					ID:           deviceID,
					SerialNumber: "SN-SYNC-BM",
					Carrier:      model.CarrierCode("cmcc"),
					Technology:   model.TechLTE,
					ProductClass: "FAP/BU1810",
				},
			},
		}).
		WithAlarmDefRegistry(registry).
		WithProductResolver(&mockProductResolver{products: map[string]*definition.ProductSnapshot{
			"FAP/BU1810": {Name: "BM Product", EnableUnknownAlarm: false},
		}})

	params := []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.5.AlarmIdentifier", Value: "70011"},
		{Name: "Device.FaultMgmt.CurrentAlarm.5.AlarmRaisedTime", Value: "2026-06-15T08:11:19Z"},
		{Name: "Device.FaultMgmt.CurrentAlarm.5.EventType", Value: "Equipment Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.5.ProbableCause", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.5.SpecificProblem", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.5.PerceivedSeverity", Value: "Major"},
		{Name: "Device.FaultMgmt.CurrentAlarm.5.AdditionalText", Value: "LTE Cell 2"},
	}

	result := processor.processSync(context.Background(), "SN-SYNC-BM", params)
	require.Zero(t, result.Added)
	require.Zero(t, result.FailedAdd)
	assert.Empty(t, store.active)
}

func TestProcessSync_PreservesFirstRaisedAtOnUpdate(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceBySN: map[string]*model.Device{
			"SN-SYNC-004": {
				ID:           deviceID,
				SerialNumber: "SN-SYNC-004",
				Carrier:      model.CarrierCode("cmcc"),
				Technology:   model.TechLTE,
			},
		},
	})

	firstRaisedAt := time.Date(2026, 6, 5, 9, 1, 0, 0, time.UTC)
	updatedRaisedAt := firstRaisedAt.Add(5 * time.Minute)
	initial := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		DeviceSN:        "SN-SYNC-004",
		Carrier:         model.CarrierCode("cmcc"),
		AlarmIdentifier: "70011",
		AlarmType:       "equipment",
		Severity:        model.AlarmMajor,
		Status:          model.AlarmActive,
		RaisedAt:        firstRaisedAt,
		FirstRaisedAt:   firstRaisedAt,
		LastUpdatedAt:   firstRaisedAt,
		AdditionalInfo:  map[string]string{},
	}
	store.active[initial.ID] = initial

	params := []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "70011"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime", Value: updatedRaisedAt.Format(time.RFC3339)},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.EventType", Value: "Equipment Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.ProbableCause", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.SpecificProblem", Value: "RU RF shutdown for cell"},
		{Name: "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", Value: "Critical"},
	}

	beforeSync := time.Now().Add(-1 * time.Second)
	result := processor.processSync(context.Background(), "SN-SYNC-004", params)
	afterSync := time.Now().Add(1 * time.Second)
	require.Equal(t, 1, result.Updated)
	require.Zero(t, result.FailedUpdate)

	alarm, err := store.GetActiveByDeviceAndIdentifier(context.Background(), "SN-SYNC-004", "70011")
	require.NoError(t, err)
	require.NotNil(t, alarm)
	assert.Equal(t, firstRaisedAt, alarm.RaisedAt)
	assert.Equal(t, firstRaisedAt, alarm.FirstRaisedAt)
	assert.Equal(t, 2, alarm.AckCount)
	assert.NotEqual(t, updatedRaisedAt, alarm.LastUpdatedAt)
	assert.False(t, alarm.LastUpdatedAt.Before(beforeSync))
	assert.False(t, alarm.LastUpdatedAt.After(afterSync))
	assert.Equal(t, model.AlarmCritical, alarm.Severity)
}
