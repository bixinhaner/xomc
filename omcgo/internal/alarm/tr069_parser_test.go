package alarm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
)

func makeParam(name, value string) tr069.ParameterValueStruct {
	return tr069.ParameterValueStruct{Name: name, Value: value}
}

func TestParseCurrentAlarmParams_FourAlarms(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		// Alarm 1
		makeParam("Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", "ALM-001"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime", "2026-04-20T10:30:00Z"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.AlarmChangedTime", "2026-04-20T10:30:00Z"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.EventType", "communicationsAlarm"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.ProbableCause", "temperatureTooHigh"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.SpecificProblem", "Board temperature exceeds 85°C"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.AdditionalInformation", "slot=1"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.AdditionalText", "Critical temperature alert"),
		makeParam("Device.FaultMgmt.CurrentAlarm.1.ManagedObjectInstance", "Device.Radio.1"),
		// Alarm 2
		makeParam("Device.FaultMgmt.CurrentAlarm.2.AlarmIdentifier", "ALM-002"),
		makeParam("Device.FaultMgmt.CurrentAlarm.2.PerceivedSeverity", "Major"),
		makeParam("Device.FaultMgmt.CurrentAlarm.2.AlarmRaisedTime", "2026-04-20T11:00:00Z"),
		makeParam("Device.FaultMgmt.CurrentAlarm.2.EventType", "qualityOfServiceAlarm"),
		makeParam("Device.FaultMgmt.CurrentAlarm.2.SpecificProblem", "S1 link failure"),
		// Alarm 3
		makeParam("Device.FaultMgmt.CurrentAlarm.3.AlarmIdentifier", "ALM-003"),
		makeParam("Device.FaultMgmt.CurrentAlarm.3.PerceivedSeverity", "Minor"),
		makeParam("Device.FaultMgmt.CurrentAlarm.3.AlarmRaisedTime", "2026-04-20T12:00:00Z"),
		makeParam("Device.FaultMgmt.CurrentAlarm.3.SpecificProblem", "CPU usage high"),
		// Alarm 4
		makeParam("Device.FaultMgmt.CurrentAlarm.5.AlarmIdentifier", "ALM-004"),
		makeParam("Device.FaultMgmt.CurrentAlarm.5.PerceivedSeverity", "Warning"),
		makeParam("Device.FaultMgmt.CurrentAlarm.5.AlarmRaisedTime", "2026-04-20T13:00:00Z"),
		makeParam("Device.FaultMgmt.CurrentAlarm.5.SpecificProblem", "Power supply voltage low"),
		// Non-alarm parameter — should be skipped
		makeParam("Device.DeviceInfo.Manufacturer", "Baicells"),
	}

	alarms, err := ParseCurrentAlarmParams(params)
	assert.NoError(t, err)
	assert.Len(t, alarms, 4)

	// Check alarm 1 (full fields)
	assert.Equal(t, "ALM-001", alarms[0].AlarmIdentifier)
	assert.Equal(t, "Critical", alarms[0].PerceivedSeverity)
	assert.Equal(t, "communicationsAlarm", alarms[0].EventType)
	assert.Equal(t, "temperatureTooHigh", alarms[0].ProbableCause)
	assert.Equal(t, "Board temperature exceeds 85°C", alarms[0].SpecificProblem)
	assert.Equal(t, "slot=1", alarms[0].AdditionalInformation)
	assert.Equal(t, "Critical temperature alert", alarms[0].AdditionalText)
	assert.Equal(t, "Device.Radio.1", alarms[0].ManagedObjectInstance)
	assert.Equal(t, 1, alarms[0].Index)

	// Check alarm 2 (partial fields)
	assert.Equal(t, "ALM-002", alarms[1].AlarmIdentifier)
	assert.Equal(t, "Major", alarms[1].PerceivedSeverity)
	assert.Equal(t, "qualityOfServiceAlarm", alarms[1].EventType)
	assert.Equal(t, "S1 link failure", alarms[1].SpecificProblem)

	// Check sort order by index
	assert.Equal(t, 1, alarms[0].Index)
	assert.Equal(t, 2, alarms[1].Index)
	assert.Equal(t, 3, alarms[2].Index)
	assert.Equal(t, 5, alarms[3].Index)
}

func TestParseCurrentAlarmParams_InternetGatewayDevicePrefix(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		makeParam("InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", "ALM-IGD-001"),
		makeParam("InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", "Major"),
		makeParam("InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.SpecificProblem", "NTP sync lost"),
	}

	alarms, err := ParseCurrentAlarmParams(params)
	assert.NoError(t, err)
	assert.Len(t, alarms, 1)
	assert.Equal(t, "ALM-IGD-001", alarms[0].AlarmIdentifier)
	assert.Equal(t, "Major", alarms[0].PerceivedSeverity)
	assert.Equal(t, "NTP sync lost", alarms[0].SpecificProblem)
	assert.Equal(t, 1, alarms[0].Index)
}

func TestParseCurrentAlarmParams_Empty(t *testing.T) {
	alarms, err := ParseCurrentAlarmParams(nil)
	assert.NoError(t, err)
	assert.Len(t, alarms, 0)

	alarms, err = ParseCurrentAlarmParams([]tr069.ParameterValueStruct{})
	assert.NoError(t, err)
	assert.Len(t, alarms, 0)
}

func TestParseCurrentAlarmParams_SkipNoIdentifier(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", "Critical"),
		// No AlarmIdentifier — should be skipped
		makeParam("Device.FaultMgmt.CurrentAlarm.2.AlarmIdentifier", "ALM-002"),
		makeParam("Device.FaultMgmt.CurrentAlarm.2.PerceivedSeverity", "Minor"),
	}

	alarms, err := ParseCurrentAlarmParams(params)
	assert.NoError(t, err)
	assert.Len(t, alarms, 1)
	assert.Equal(t, "ALM-002", alarms[0].AlarmIdentifier)
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"critical", 1},
		{"CRITICAL", 1},
		{"major", 2},
		{"MAJOR", 2},
		{"minor", 3},
		{"MINOR", 3},
		{"warning", 4},
		{"WARNING", 4},
		{"unknown", 2}, // default to Major
		{"", 2},       // default to Major
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapSeverity(tt.input)
			assert.Equal(t, tt.expected, int(result))
		})
	}
}

func TestParseTR069Time(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{"valid RFC3339", "2026-04-20T10:30:00Z", false},
		{"no timezone suffix", "2026-04-12T06:43:04", false},
		{"no timezone negative offset", "2026-04-20T10:30:00-07:00", false},
		{"with milliseconds", "2026-04-20T10:30:00.000Z", false},
		{"empty string", "", true},
		{"epoch time", "1970-01-01T00:00:00Z", true},
		{"epoch no tz", "1970-01-01T00:00:00", true},
		{"pre-2000", "1999-12-31T23:59:59Z", true},
		{"invalid format", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTR069Time(tt.input)
			if tt.wantZero {
				assert.True(t, result.IsZero(), "expected zero time for %q", tt.input)
			} else {
				assert.False(t, result.IsZero(), "expected non-zero time for %q", tt.input)
			}
		})
	}
}

func TestTR069Alarm_Validate(t *testing.T) {
	t.Run("valid alarm", func(t *testing.T) {
		a := &TR069Alarm{Index: 1, AlarmIdentifier: "ALM-001"}
		assert.NoError(t, a.Validate())
	})

	t.Run("missing identifier", func(t *testing.T) {
		a := &TR069Alarm{Index: 1}
		assert.Error(t, a.Validate())
		assert.Contains(t, a.Validate().Error(), "no AlarmIdentifier")
	})
}

func TestToModel(t *testing.T) {
	raisedTime, _ := time.Parse(time.RFC3339, "2026-04-20T10:30:00Z")

	tAlarm := &TR069Alarm{
		Index:                 1,
		AlarmIdentifier:       "ALM-001",
		PerceivedSeverity:     "Critical",
		AlarmRaisedTime:       raisedTime,
		AlarmChangedTime:      time.Time{},
		EventType:             "communicationsAlarm",
		ProbableCause:         "temperatureTooHigh",
		SpecificProblem:       "Board temperature exceeds 85°C",
		AdditionalInformation: "slot=1",
		AdditionalText:        "Critical temperature alert",
		ManagedObjectInstance: "Device.Radio.1",
	}

	deviceID := validUUID(t)
	alarm := tAlarm.ToModel(deviceID, "SN001", "cmcc")

	assert.Equal(t, "ALM-001", alarm.AlarmIdentifier)
	assert.Equal(t, "SN001", alarm.DeviceSN)
	assert.Equal(t, "Board temperature exceeds 85°C", alarm.Description)
	assert.Equal(t, "TR069", *alarm.AlarmSource)
	assert.Equal(t, "communicationsAlarm", *alarm.EventType)
	assert.Equal(t, "temperatureTooHigh", *alarm.ProbableCause)
	assert.Equal(t, "slot=1", alarm.AdditionalInfo["additional_information"])
	assert.Equal(t, "Critical temperature alert", alarm.AdditionalInfo["additional_text"])
	assert.Equal(t, "Device.Radio.1", alarm.AdditionalInfo["managed_object_instance"])
	assert.False(t, alarm.LastUpdatedAt.IsZero())
}

func TestFormatAlarmParamsForGPV(t *testing.T) {
	result := FormatAlarmParamsForGPV()
	assert.JSONEq(t, `{"names":["Device.FaultMgmt.CurrentAlarm."]}`, string(result))
}

func TestComputeDiff(t *testing.T) {
	makeAlarm := func(id, desc string, severity int) *model.Alarm {
		return &model.Alarm{
			AlarmIdentifier: id,
			Description:     desc,
			Severity:        model.AlarmSeverity(severity),
		}
	}

	t.Run("all new alarms", func(t *testing.T) {
		remote := []*model.Alarm{
			makeAlarm("ALM-001", "desc1", 1),
			makeAlarm("ALM-002", "desc2", 2),
		}
		diff := ComputeDiff(remote, nil)
		assert.Len(t, diff.ToAdd, 2)
		assert.Len(t, diff.ToUpdate, 0)
		assert.Len(t, diff.ToClear, 0)
	})

	t.Run("all cleared alarms", func(t *testing.T) {
		local := []*model.Alarm{
			makeAlarm("ALM-001", "desc1", 1),
		}
		diff := ComputeDiff(nil, local)
		assert.Len(t, diff.ToAdd, 0)
		assert.Len(t, diff.ToUpdate, 0)
		assert.Len(t, diff.ToClear, 1)
		assert.Equal(t, "ALM-001", diff.ToClear[0])
	})

	t.Run("updated alarm", func(t *testing.T) {
		remote := []*model.Alarm{makeAlarm("ALM-001", "new desc", 1)}
		local := []*model.Alarm{makeAlarm("ALM-001", "old desc", 2)}
		diff := ComputeDiff(remote, local)
		assert.Len(t, diff.ToAdd, 0)
		assert.Len(t, diff.ToUpdate, 1)
		assert.Len(t, diff.ToClear, 0)
	})

	t.Run("unchanged alarm", func(t *testing.T) {
		remote := []*model.Alarm{makeAlarm("ALM-001", "desc", 1)}
		local := []*model.Alarm{makeAlarm("ALM-001", "desc", 1)}
		diff := ComputeDiff(remote, local)
		assert.Len(t, diff.ToAdd, 0)
		assert.Len(t, diff.ToUpdate, 0)
		assert.Len(t, diff.ToClear, 0)
	})

	t.Run("mixed scenario", func(t *testing.T) {
		remote := []*model.Alarm{
			makeAlarm("ALM-001", "desc", 1),      // unchanged
			makeAlarm("ALM-002", "new desc", 2),   // updated
			makeAlarm("ALM-003", "desc", 3),        // new
		}
		local := []*model.Alarm{
			makeAlarm("ALM-001", "desc", 1),      // unchanged
			makeAlarm("ALM-002", "old desc", 2),   // updated
			makeAlarm("ALM-999", "desc", 1),       // cleared
		}
		diff := ComputeDiff(remote, local)
		assert.Len(t, diff.ToAdd, 1)
		assert.Len(t, diff.ToUpdate, 1)
		assert.Len(t, diff.ToClear, 1)
		assert.Equal(t, "ALM-003", diff.ToAdd[0].AlarmIdentifier)
		assert.Contains(t, diff.ToUpdate, "ALM-002")
		assert.Equal(t, "ALM-999", diff.ToClear[0])
	})
}

func TestPtrStrNE(t *testing.T) {
	a := "hello"
	b := "hello"
	c := "world"

	assert.False(t, ptrStrNE(nil, nil))
	assert.True(t, ptrStrNE(nil, &a))
	assert.True(t, ptrStrNE(&a, nil))
	assert.False(t, ptrStrNE(&a, &b))
	assert.True(t, ptrStrNE(&a, &c))
}

// Helper to create a valid UUID for tests
func validUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// --- ExpeditedEvent tests ---

func TestHasExpeditedEventParams(t *testing.T) {
	t.Run("has expedited event params", func(t *testing.T) {
		params := []tr069.ParameterValueStruct{
			makeParam("Device.DeviceInfo.Manufacturer", "Baicells"),
			makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
			makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		}
		assert.True(t, HasExpeditedEventParams(params))
	})

	t.Run("has expedited event params with igd root", func(t *testing.T) {
		params := []tr069.ParameterValueStruct{
			makeParam("InternetGatewayDevice.DeviceInfo.Manufacturer", "Baicells"),
			makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
			makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		}
		assert.True(t, HasExpeditedEventParams(params))
	})

	t.Run("no expedited event params", func(t *testing.T) {
		params := []tr069.ParameterValueStruct{
			makeParam("Device.DeviceInfo.Manufacturer", "Baicells"),
			makeParam("Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", "ALM-001"),
		}
		assert.False(t, HasExpeditedEventParams(params))
	})

	t.Run("empty params", func(t *testing.T) {
		assert.False(t, HasExpeditedEventParams(nil))
	})
}

func TestFilterExpeditedEventParams(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		makeParam("Device.DeviceInfo.Manufacturer", "Baicells"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.Services.FAPService.1.FAPControl.LTE.CellOpState", "1"),
	}

	filtered := FilterExpeditedEventParams(params)
	assert.Len(t, filtered, 3)
	for _, p := range filtered {
		assert.True(t, len(p.Name) > len("Device.FaultMgmt.ExpeditedEvent."))
	}
}

func TestFilterExpeditedEventParams_InternetGatewayDevicePrefix(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		makeParam("InternetGatewayDevice.DeviceInfo.Manufacturer", "Baicells"),
		makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("InternetGatewayDevice.Services.FAPService.1.FAPControl.LTE.CellOpState", "1"),
	}

	filtered := FilterExpeditedEventParams(params)
	assert.Len(t, filtered, 2)
	for _, p := range filtered {
		assert.Contains(t, p.Name, "FaultMgmt.ExpeditedEvent.")
	}
}

func TestParseExpeditedEventParams_NewAlarm(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		// ExpeditedEvent.10 — NewAlarm
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventType", "Equipment Alarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ProbableCause", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.SpecificProblem", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AdditionalText", "LTE0"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventTime", "2026-04-23T09:03:01"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ManagedObjectInstance", "Device.FaultMgmt.ExpeditedEvent."),
		// Unrelated params should be skipped
		makeParam("Device.DeviceInfo.Manufacturer", "BAICELLS"),
		makeParam("Device.DeviceInfo.SoftwareVersion", "BaiBLN_5.1.11.2"),
	}

	events, err := ParseExpeditedEventParams(params)
	assert.NoError(t, err)
	assert.Len(t, events, 1)

	ev := events[0]
	assert.Equal(t, 10, ev.Index)
	assert.Equal(t, "NewAlarm", ev.NotificationType)
	assert.Equal(t, "11184", ev.AlarmIdentifier)
	assert.Equal(t, "Critical", ev.PerceivedSeverity)
	assert.Equal(t, "Equipment Alarm", ev.EventType)
	assert.Equal(t, "Cell unavailable", ev.ProbableCause)
	assert.Equal(t, "Cell unavailable", ev.SpecificProblem)
	assert.Equal(t, "LTE0", ev.AdditionalText)
	assert.False(t, ev.EventTime.IsZero())
	assert.Equal(t, "Device.FaultMgmt.ExpeditedEvent.", ev.ManagedObjectInstance)
}

func TestParseExpeditedEventParams_InternetGatewayDevicePrefix(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "50003"),
		makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Minor"),
		makeParam("InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.SpecificProblem", "Time synchronization failed"),
	}

	events, err := ParseExpeditedEventParams(params)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, 10, events[0].Index)
	assert.Equal(t, "NewAlarm", events[0].NotificationType)
	assert.Equal(t, "50003", events[0].AlarmIdentifier)
	assert.Equal(t, "Minor", events[0].PerceivedSeverity)
	assert.Equal(t, "Time synchronization failed", events[0].SpecificProblem)
}

func TestParseExpeditedEventParams_MultipleEvents(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		// Event 1: NewAlarm
		makeParam("Device.FaultMgmt.ExpeditedEvent.5.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.5.AlarmIdentifier", "ALM-A"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.5.PerceivedSeverity", "Major"),
		// Event 2: ChangedAlarm
		makeParam("Device.FaultMgmt.ExpeditedEvent.8.NotificationType", "ChangedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.8.AlarmIdentifier", "ALM-B"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.8.PerceivedSeverity", "Warning"),
		// Event 3: ClearedAlarm
		makeParam("Device.FaultMgmt.ExpeditedEvent.12.NotificationType", "ClearedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.12.AlarmIdentifier", "ALM-C"),
	}

	events, err := ParseExpeditedEventParams(params)
	assert.NoError(t, err)
	assert.Len(t, events, 3)

	assert.Equal(t, "NewAlarm", events[0].NotificationType)
	assert.Equal(t, "ALM-A", events[0].AlarmIdentifier)
	assert.Equal(t, 5, events[0].Index)

	assert.Equal(t, "ChangedAlarm", events[1].NotificationType)
	assert.Equal(t, "ALM-B", events[1].AlarmIdentifier)
	assert.Equal(t, 8, events[1].Index)

	assert.Equal(t, "ClearedAlarm", events[2].NotificationType)
	assert.Equal(t, "ALM-C", events[2].AlarmIdentifier)
	assert.Equal(t, 12, events[2].Index)
}

func TestParseExpeditedEventParams_SkipNoIdentifier(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.1.PerceivedSeverity", "Critical"),
		// No AlarmIdentifier — should be skipped
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.AlarmIdentifier", "ALM-002"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.2.NotificationType", "ChangedAlarm"),
	}

	events, err := ParseExpeditedEventParams(params)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "ALM-002", events[0].AlarmIdentifier)
}

func TestParseExpeditedEventParams_Empty(t *testing.T) {
	events, err := ParseExpeditedEventParams(nil)
	assert.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestExpeditedEvent_Validate(t *testing.T) {
	t.Run("valid NewAlarm", func(t *testing.T) {
		ev := &ExpeditedEvent{Index: 10, NotificationType: "NewAlarm", AlarmIdentifier: "11184"}
		assert.NoError(t, ev.Validate())
	})

	t.Run("valid ChangedAlarm", func(t *testing.T) {
		ev := &ExpeditedEvent{Index: 10, NotificationType: "ChangedAlarm", AlarmIdentifier: "11184"}
		assert.NoError(t, ev.Validate())
	})

	t.Run("valid ClearedAlarm", func(t *testing.T) {
		ev := &ExpeditedEvent{Index: 10, NotificationType: "ClearedAlarm", AlarmIdentifier: "11184"}
		assert.NoError(t, ev.Validate())
	})

	t.Run("missing identifier", func(t *testing.T) {
		ev := &ExpeditedEvent{Index: 10, NotificationType: "NewAlarm"}
		assert.Error(t, ev.Validate())
		assert.Contains(t, ev.Validate().Error(), "no AlarmIdentifier")
	})

	t.Run("unknown notification type", func(t *testing.T) {
		ev := &ExpeditedEvent{Index: 10, NotificationType: "UnknownType", AlarmIdentifier: "11184"}
		assert.Error(t, ev.Validate())
		assert.Contains(t, ev.Validate().Error(), "unknown NotificationType")
	})
}

func TestExpeditedEvent_ToModel(t *testing.T) {
	eventTime, _ := time.Parse("2006-01-02T15:04:05", "2026-04-23T09:03:01")

	ev := &ExpeditedEvent{
		Index:                 10,
		NotificationType:      "NewAlarm",
		AlarmIdentifier:       "11184",
		PerceivedSeverity:     "Critical",
		EventType:             "Equipment Alarm",
		ProbableCause:         "Cell unavailable",
		SpecificProblem:       "Cell unavailable",
		AdditionalText:        "LTE0",
		EventTime:             eventTime,
		ManagedObjectInstance: "Device.FaultMgmt.ExpeditedEvent.",
	}

	deviceID := validUUID(t)
	alarm := ev.ToModel(deviceID, "SN-TEST", model.CarrierCMCC)

	assert.Equal(t, "11184", alarm.AlarmIdentifier)
	assert.Equal(t, "SN-TEST", alarm.DeviceSN)
	assert.Equal(t, model.CarrierCMCC, alarm.Carrier)
	assert.Equal(t, "Cell unavailable", alarm.Description)
	assert.Equal(t, "TR069", *alarm.AlarmSource)
	assert.Equal(t, "Equipment Alarm", *alarm.EventType)
	assert.Equal(t, "Cell unavailable", *alarm.ProbableCause)
	assert.Equal(t, "LTE0", alarm.AdditionalInfo["additional_text"])
	assert.Equal(t, "Device.FaultMgmt.ExpeditedEvent.", alarm.AdditionalInfo["managed_object_instance"])
	assert.Equal(t, "NewAlarm", alarm.AdditionalInfo["notification_type"])
	assert.Equal(t, model.AlarmActive, alarm.Status)
	assert.False(t, alarm.RaisedAt.IsZero())
	assert.False(t, alarm.LastUpdatedAt.IsZero())
}

func TestExpeditedEvent_ToModel_ProbableCauseFallback(t *testing.T) {
	t.Run("no probable cause, falls back to specific problem", func(t *testing.T) {
		ev := &ExpeditedEvent{
			AlarmIdentifier:  "ALM-001",
			SpecificProblem:  "Board failure",
			PerceivedSeverity: "Major",
		}
		alarm := ev.ToModel(uuid.UUID{}, "SN", model.CarrierCMCC)
		assert.Equal(t, "Board failure", *alarm.ProbableCause)
	})

	t.Run("no probable cause or specific problem, falls back to identifier", func(t *testing.T) {
		ev := &ExpeditedEvent{
			AlarmIdentifier:  "ALM-002",
			PerceivedSeverity: "Minor",
		}
		alarm := ev.ToModel(uuid.UUID{}, "SN", model.CarrierCMCC)
		assert.Equal(t, "ALM-002", *alarm.ProbableCause)
	})
}
