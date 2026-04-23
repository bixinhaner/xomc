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
	assert.JSONEq(t, `{"parameter_names":["Device.FaultMgmt.CurrentAlarm."]}`, string(result))
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
