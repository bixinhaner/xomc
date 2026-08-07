package event

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestAlarmLifecyclePayload_RoundTrip(t *testing.T) {
	previous := model.Alarm{Severity: model.AlarmMajor}
	alarmEventType := "equipment_alarm"
	alarm := model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-1",
		Carrier:         model.CarrierCMCC,
		Severity:        model.AlarmCritical,
		AlarmType:       "communications",
		AlarmIdentifier: "CELL_UNAVAILABLE",
		Description:     "cell unavailable",
		Status:          model.AlarmActive,
		RaisedAt:        time.Date(2026, 8, 4, 9, 59, 0, 0, time.UTC),
		EventType:       &alarmEventType,
		AckCount:        3,
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.Radio.1",
			"ne_type":                 "ENB",
			"specific_problem":        "S1 link unavailable",
			"handling_suggestion":     "Check the transport link",
			"internal_debug_token":    "must-not-leak",
		},
	}
	occurredAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	changeMask := []AlarmChangeField{AlarmChangeSeverity, AlarmChangeCount}

	payload, err := NewAlarmLifecyclePayload(
		AlarmLifecycleUpdated,
		alarm,
		&previous,
		2,
		occurredAt,
		changeMask,
	)
	require.NoError(t, err)

	data, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NotContains(t, string(data), "internal_debug_token")
	require.NotContains(t, string(data), `"alarm":`)

	var decoded AlarmLifecyclePayload
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, AlarmLifecycleSchemaVersion, decoded.SchemaVersion)
	require.NotEqual(t, uuid.Nil, decoded.EventID)
	require.Equal(t, AlarmLifecycleUpdated, decoded.LifecycleType)
	require.Equal(t, occurredAt, decoded.OccurredAt)
	require.Equal(t, alarm.ID, decoded.OccurrenceID)
	require.Equal(t, int64(2), decoded.AlarmVersion)
	require.Equal(t, changeMask, decoded.ChangeMask)
	require.NotNil(t, decoded.PreviousSeverity)
	require.Equal(t, model.AlarmMajor, *decoded.PreviousSeverity)
	require.Equal(t, alarm.ID, decoded.Snapshot.AlarmID)
	require.Equal(t, alarm.DeviceID, decoded.Snapshot.DeviceID)
	require.Equal(t, alarmEventType, *decoded.Snapshot.AlarmEventType)
	require.Equal(t, "cell unavailable", *decoded.Snapshot.AlarmName)
	require.Equal(t, "ENB", *decoded.Snapshot.NEType)
	require.Equal(t, "S1 link unavailable", *decoded.Snapshot.SpecificProblem)
	require.Equal(t, "Check the transport link", *decoded.Snapshot.HandlingSuggestion)
	require.Equal(t, 3, decoded.Snapshot.AlarmCount)
	require.Equal(t, "Device.Radio.1", decoded.Snapshot.Extensions["managed_object_instance"])
}

func TestNewAlarmLifecyclePayload_CopiesAndBoundsMutableFields(t *testing.T) {
	technology := "NR"
	alarm := model.Alarm{
		ID:         uuid.New(),
		DeviceID:   uuid.New(),
		Technology: &technology,
		AdditionalInfo: map[string]string{
			"additional_text": strings.Repeat("界", alarmLifecycleExtensionValueMaxBytes),
			"secret":          "must-not-leak",
		},
	}
	changeMask := []AlarmChangeField{AlarmChangeExtensions}

	payload, err := NewAlarmLifecyclePayload(
		AlarmLifecycleUpdated,
		alarm,
		nil,
		2,
		time.Now(),
		changeMask,
	)
	require.NoError(t, err)
	alarm.AdditionalInfo["additional_text"] = "changed"
	technology = "LTE"
	changeMask[0] = AlarmChangeStatus

	require.Equal(t, "NR", *payload.Snapshot.Technology)
	require.NotEqual(t, "changed", payload.Snapshot.Extensions["additional_text"])
	require.LessOrEqual(t, len(payload.Snapshot.Extensions["additional_text"]), alarmLifecycleExtensionValueMaxBytes)
	require.True(t, utf8.ValidString(payload.Snapshot.Extensions["additional_text"]))
	require.NotContains(t, payload.Snapshot.Extensions, "secret")
	require.Equal(t, []AlarmChangeField{AlarmChangeExtensions}, payload.ChangeMask)
	require.Nil(t, payload.PreviousSeverity)
}

func TestNewAlarmLifecyclePayload_RejectsInvalidCanonicalFacts(t *testing.T) {
	validAlarm := model.Alarm{ID: uuid.New(), DeviceID: uuid.New()}
	validTime := time.Now()

	tests := []struct {
		name          string
		lifecycleType AlarmLifecycleType
		alarm         model.Alarm
		previous      *model.Alarm
		version       int64
		occurredAt    time.Time
		changeMask    []AlarmChangeField
	}{
		{name: "unknown lifecycle", lifecycleType: "unexpected", alarm: validAlarm, version: 1, occurredAt: validTime},
		{name: "missing occurrence", lifecycleType: AlarmLifecycleRaised, alarm: model.Alarm{DeviceID: uuid.New()}, version: 1, occurredAt: validTime},
		{name: "missing managed element", lifecycleType: AlarmLifecycleRaised, alarm: model.Alarm{ID: uuid.New()}, version: 1, occurredAt: validTime},
		{name: "non-positive version", lifecycleType: AlarmLifecycleRaised, alarm: validAlarm, version: 0, occurredAt: validTime},
		{name: "missing occurred time", lifecycleType: AlarmLifecycleRaised, alarm: validAlarm, version: 1},
		{
			name:          "severity change without previous snapshot",
			lifecycleType: AlarmLifecycleUpdated,
			alarm:         validAlarm,
			version:       2,
			occurredAt:    validTime,
			changeMask:    []AlarmChangeField{AlarmChangeSeverity},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAlarmLifecyclePayload(
				tt.lifecycleType,
				tt.alarm,
				tt.previous,
				tt.version,
				tt.occurredAt,
				tt.changeMask,
			)
			require.Error(t, err)
		})
	}
}

func TestAlarmLifecycleType_Subject(t *testing.T) {
	tests := []struct {
		lifecycleType AlarmLifecycleType
		subject       string
	}{
		{AlarmLifecycleRaised, SubjectDomainAlarmLifecycleRaised},
		{AlarmLifecycleUpdated, SubjectDomainAlarmLifecycleUpdated},
		{AlarmLifecycleAcknowledged, SubjectDomainAlarmLifecycleAcknowledged},
		{AlarmLifecycleUnacknowledged, SubjectDomainAlarmLifecycleUnacknowledged},
		{AlarmLifecycleCleared, SubjectDomainAlarmLifecycleCleared},
	}

	for _, tt := range tests {
		t.Run(string(tt.lifecycleType), func(t *testing.T) {
			subject, err := tt.lifecycleType.Subject()
			require.NoError(t, err)
			require.Equal(t, tt.subject, subject)
		})
	}

	_, err := AlarmLifecycleType("unexpected").Subject()
	require.Error(t, err)
}
