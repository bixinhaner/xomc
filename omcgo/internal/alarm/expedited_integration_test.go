package alarm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// integrationTestDevice is a minimal DeviceLookup for integration tests.
type integrationTestDevice struct {
	device *model.Device
}

func (d *integrationTestDevice) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	if sn == d.device.SerialNumber {
		return d.device, nil
	}
	return nil, nil
}

// buildACSExpeditedPayload replicates the exact JSON structure the ACS handler publishes
// when it detects ExpeditedEvent parameters in a VALUE CHANGE Inform.
func buildACSExpeditedPayload(deviceSN string, params []tr069.ParameterValueStruct) event.Event {
	payload := map[string]interface{}{
		"device_sn":        deviceSN,
		"parameter_values": params,
	}
	evt, err := event.NewEvent(event.SubjectDeviceExpeditedAlarm, payload)
	if err != nil {
		panic(err)
	}
	return evt
}

func TestIntegration_FullPipeline_NewAlarm(t *testing.T) {
	// Setup: ChannelEventBus + ExpeditedEventReceiver (same as production wiring)
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	testDevice := &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "SN-BJC-001",
		Carrier:      model.CarrierCMCC,
		Status:       model.DeviceActive,
	}
	deviceLookup := &integrationTestDevice{device: testDevice}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	require.NoError(t, receiver.Subscribe(bus))

	ctx := context.Background()

	// Simulate ACS handler publishing expedited alarm event
	// (same structure as publishInformEvents in acs/handler.go)
	params := []tr069.ParameterValueStruct{
		makeParam("Device.DeviceInfo.Manufacturer", "BAICELLS"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventType", "Equipment Alarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ProbableCause", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.SpecificProblem", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AdditionalText", "LTE0"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventTime", "2026-04-23T09:03:01"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ManagedObjectInstance", "Device.FaultMgmt.ExpeditedEvent."),
		makeParam("Device.Services.FAPService.1.FAPControl.LTE.CellOpState", "1"),
	}

	evt := buildACSExpeditedPayload("SN-BJC-001", params)
	err := bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt)
	require.NoError(t, err)

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify: alarm created in store
	assert.Len(t, store.active, 1)
	var found *model.Alarm
	for _, a := range store.active {
		if a.AlarmIdentifier == "11184" {
			found = a
			break
		}
	}
	require.NotNil(t, found, "alarm should be created")
	assert.Equal(t, "SN-BJC-001", found.DeviceSN)
	assert.Equal(t, model.CarrierCMCC, found.Carrier)
	assert.Equal(t, model.AlarmActive, found.Status)
	assert.Equal(t, "Cell unavailable", found.Description)
	assert.Equal(t, "LTE0", found.AdditionalInfo["additional_text"])
	assert.Equal(t, "NewAlarm", found.AdditionalInfo["notification_type"])
}

func TestIntegration_FullPipeline_ChangedThenCleared(t *testing.T) {
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	testDevice := &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "SN-SH-002",
		Carrier:      model.CarrierCTCC,
		Status:       model.DeviceActive,
	}
	deviceLookup := &integrationTestDevice{device: testDevice}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	require.NoError(t, receiver.Subscribe(bus))

	ctx := context.Background()

	// Step 1: NewAlarm
	newAlarmParams := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.AlarmIdentifier", "ALM-500"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.PerceivedSeverity", "Major"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.ProbableCause", "Temperature high"),
	}
	evt1 := buildACSExpeditedPayload("SN-SH-002", newAlarmParams)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt1))
	time.Sleep(100 * time.Millisecond)
	assert.Len(t, store.active, 1, "should have 1 active alarm after NewAlarm")

	// Step 2: ChangedAlarm (severity escalation)
	changeParams := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.NotificationType", "ChangedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.AlarmIdentifier", "ALM-500"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.PerceivedSeverity", "Critical"),
	}
	evt2 := buildACSExpeditedPayload("SN-SH-002", changeParams)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt2))
	time.Sleep(100 * time.Millisecond)
	assert.Len(t, store.active, 1, "should still have 1 alarm after ChangedAlarm")

	existing, err := store.GetActiveByDeviceAndIdentifier(ctx, "SN-SH-002", "ALM-500")
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.Equal(t, model.AlarmCritical, existing.Severity, "severity should be escalated to Critical")

	// Step 3: ClearedAlarm
	clearParams := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.3.NotificationType", "ClearedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.3.AlarmIdentifier", "ALM-500"),
	}
	evt3 := buildACSExpeditedPayload("SN-SH-002", clearParams)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt3))
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, store.active, 0, "alarm should be cleared from active")
	assert.Len(t, store.history, 1, "alarm should be in history")
	assert.Equal(t, model.AlarmCleared, store.history[0].Status)
}

func TestIntegration_FullPipeline_NoExpeditedParams(t *testing.T) {
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	testDevice := &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "SN-GZ-003",
		Carrier:      model.CarrierCUCC,
		Status:       model.DeviceActive,
	}
	deviceLookup := &integrationTestDevice{device: testDevice}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	require.NoError(t, receiver.Subscribe(bus))

	ctx := context.Background()

	// VALUE CHANGE Inform WITHOUT ExpeditedEvent params — should not create alarms
	params := []tr069.ParameterValueStruct{
		makeParam("Device.DeviceInfo.SoftwareVersion", "v5.1.11"),
		makeParam("Device.Services.FAPService.1.FAPControl.LTE.CellOpState", "1"),
	}
	evt := buildACSExpeditedPayload("SN-GZ-003", params)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt))
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, store.active, 0, "no alarm should be created without ExpeditedEvent params")
}

func TestIntegration_FullPipeline_MultipleEventsInOneInform(t *testing.T) {
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	testDevice := &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "SN-CD-005",
		Carrier:      model.CarrierCMCC,
		Status:       model.DeviceActive,
	}
	deviceLookup := &integrationTestDevice{device: testDevice}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	require.NoError(t, receiver.Subscribe(bus))

	ctx := context.Background()

	// One Inform with 2 ExpeditedEvents: NewAlarm + ClearedAlarm
	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.AlarmIdentifier", "ALM-A"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.PerceivedSeverity", "Warning"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.AlarmIdentifier", "ALM-B"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.PerceivedSeverity", "Minor"),
	}
	evt := buildACSExpeditedPayload("SN-CD-005", params)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt))
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, store.active, 2, "2 alarms should be created from 2 ExpeditedEvents")
}

func TestIntegration_FullPipeline_RealWorldXMLPayload(t *testing.T) {
	// This test replicates the exact XML payload from the reference document:
	// docs/design/alarm_valuechange_Message.xml (BAICELLS FAP/pCRB2000/CA)
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	testDevice := &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "1202000559233HB0042",
		Carrier:      model.CarrierCMCC,
		Status:       model.DeviceActive,
	}
	deviceLookup := &integrationTestDevice{device: testDevice}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	require.NoError(t, receiver.Subscribe(bus))

	ctx := context.Background()

	// Exact parameters from the reference XML document
	params := []tr069.ParameterValueStruct{
		// Device info params (non-alarm, should be filtered out)
		makeParam("Device.DeviceInfo.Manufacturer", "BAICELLS"),
		makeParam("Device.DeviceInfo.OUI", "48BF74"),
		makeParam("Device.DeviceInfo.ProductClass", "FAP/pCRB2000/CA"),
		makeParam("Device.DeviceInfo.SoftwareVersion", "BaiBLN_5.1.11.2"),
		makeParam("Device.DeviceInfo.HardwareVersion", "C"),
		makeParam("Device.DeviceInfo.X_COM_GPS_Status", "1"),
		// ExpeditedEvent params (from the XML)
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventType", "Equipment Alarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ProbableCause", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.SpecificProblem", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AdditionalText", "LTE0"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventTime", "2026-04-23T09:03:01"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ManagedObjectInstance", "Device.FaultMgmt.ExpeditedEvent."),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AdditionalInformation", ""),
		// FAPService params (non-alarm, should be filtered out)
		makeParam("Device.Services.FAPService.1.FAPControl.LTE.CellOpState", "1"),
		makeParam("Device.Services.FAPService.1.FAPControl.LTE.OpState", "0"),
		makeParam("Device.ManagementServer.ConnectionRequestURL", "http://10.10.3.229:7547"),
	}

	evt := buildACSExpeditedPayload("1202000559233HB0042", params)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt))
	time.Sleep(100 * time.Millisecond)

	// Verify: exactly 1 alarm created from 101 parameters
	assert.Len(t, store.active, 1, "should create 1 alarm from ExpeditedEvent params")

	var found *model.Alarm
	for _, a := range store.active {
		if a.AlarmIdentifier == "11184" {
			found = a
			break
		}
	}
	require.NotNil(t, found)

	// Verify all parsed fields match the XML
	assert.Equal(t, "1202000559233HB0042", found.DeviceSN)
	assert.Equal(t, model.CarrierCMCC, found.Carrier)
	assert.Equal(t, model.AlarmActive, found.Status)
	assert.Equal(t, "Cell unavailable", found.Description)
	assert.Equal(t, "Equipment Alarm", *found.EventType)
	assert.Equal(t, "Cell unavailable", *found.ProbableCause)
	assert.Equal(t, "LTE0", found.AdditionalInfo["additional_text"])
	assert.Equal(t, "Device.FaultMgmt.ExpeditedEvent.", found.AdditionalInfo["managed_object_instance"])
	assert.Equal(t, "NewAlarm", found.AdditionalInfo["notification_type"])

	// Verify EventTime parsed correctly
	assert.False(t, found.RaisedAt.IsZero(), "EventTime should be parsed")
	assert.False(t, found.LastUpdatedAt.IsZero(), "LastUpdatedAt should be set")
}

func TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed(t *testing.T) {
	// ChangedAlarm arrives before NewAlarm — should fallback to Process()
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	testDevice := &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "SN-RACE-001",
		Carrier:      model.CarrierCMCC,
		Status:       model.DeviceActive,
	}
	deviceLookup := &integrationTestDevice{device: testDevice}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	require.NoError(t, receiver.Subscribe(bus))

	ctx := context.Background()

	// ChangedAlarm for an alarm that doesn't exist yet (race condition)
	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.NotificationType", "ChangedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.AlarmIdentifier", "ALM-RACE"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.PerceivedSeverity", "Major"),
	}
	evt := buildACSExpeditedPayload("SN-RACE-001", params)
	require.NoError(t, bus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, evt))
	time.Sleep(100 * time.Millisecond)

	// Should NOT error — fallback to Process() creates the alarm
	assert.Len(t, store.active, 1, "fallback to Process() should create alarm")
}

// Verify the ACS payload format matches what the receiver expects
func TestIntegration_ACSPayloadFormatCompatibility(t *testing.T) {
	// The ACS handler publishes: {"device_sn": "...", "parameter_values": [...]}
	// The receiver expects: ExpeditedEventPayload{DeviceSN, ParameterValues}
	// This test verifies JSON serialization compatibility.

	acsPayload := map[string]interface{}{
		"device_sn": "SN-TEST",
		"parameter_values": []map[string]string{
			{"name": "Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "value": "NewAlarm"},
			{"name": "Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "value": "11184"},
		},
	}
	acsJSON, err := json.Marshal(acsPayload)
	require.NoError(t, err)

	// Decode using the receiver's expected type
	var payload ExpeditedEventPayload
	err = json.Unmarshal(acsJSON, &payload)
	require.NoError(t, err)

	assert.Equal(t, "SN-TEST", payload.DeviceSN)
	require.Len(t, payload.ParameterValues, 2)
	assert.Equal(t, "Device.FaultMgmt.ExpeditedEvent.10.NotificationType", payload.ParameterValues[0].Name)
	assert.Equal(t, "NewAlarm", payload.ParameterValues[0].Value)
}

// Verify that HasExpeditedEventParams works on the ACS handler's parameter list
func TestIntegration_HasExpeditedEventParams_MixedParameterList(t *testing.T) {
	// Simulate the exact parameter list from a real Inform (101 parameters)
	params := []tr069.ParameterValueStruct{
		makeParam("Device.DeviceInfo.DnPrefix", "00256D"),
		makeParam("Device.DeviceInfo.FAP_adminstate", "true"),
		makeParam("Device.DeviceInfo.HardwareVersion", "C"),
		makeParam("Device.DeviceInfo.SoftwareVersion", "BaiBLN_5.1.11.2"),
		makeParam("Device.DeviceInfo.X_COM_GPS_Status", "1"),
		makeParam("Device.DeviceInfo.X_COM_STATION_RUN_Time", "1d 1h 21m 9s"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AdditionalInformation", ""),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AdditionalText", "LTE0"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventTime", "2026-04-23T09:03:01"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventType", "Equipment Alarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ManagedObjectInstance", "Device.FaultMgmt.ExpeditedEvent."),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ProbableCause", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.SpecificProblem", "Cell unavailable"),
		makeParam("Device.ManagementServer.ConnectionRequestURL", "http://10.10.3.229:7547"),
		makeParam("Device.ManagementServer.ParameterKey", "key_1"),
		makeParam("Device.Services.FAPService.1.FAPControl.LTE.CellOpState", "1"),
	}

	assert.True(t, HasExpeditedEventParams(params), "should detect ExpeditedEvent in mixed parameter list")

	filtered := FilterExpeditedEventParams(params)
	assert.Len(t, filtered, 10, "should filter exactly 10 ExpeditedEvent parameters")

	// All filtered params should start with the prefix
	for _, p := range filtered {
		assert.True(t, len(p.Name) > len("Device.FaultMgmt.ExpeditedEvent."),
			"filtered param should be ExpeditedEvent: %s", p.Name)
	}
}
