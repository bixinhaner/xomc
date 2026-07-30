package alarm

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestActiveAlarmMatchKey_CorrelatesCurrentAndExpeditedChannels(t *testing.T) {
	current := &model.Alarm{
		AlarmIdentifier: "11109",
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.FaultMgmt.CurrentAlarm.3.",
			"additional_text":         "LTE0",
			"additional_information":  "LTE0(73828545);S1setup fail. The MME ID: MME1",
		},
	}
	expedited := &model.Alarm{
		AlarmIdentifier: "11109",
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.FaultMgmt.ExpeditedEvent.",
			"additional_text":         "LTE0",
			"additional_information":  "LTE0(73828545);S1setup fail. The MME ID: MME1",
		},
	}

	assert.Equal(t, "11109|LTE0(73828545)", activeAlarmMatchKey(current))
	assert.Equal(t, activeAlarmMatchKey(current), activeAlarmMatchKey(expedited))
}

func TestActiveAlarmMatchKey_IgnoresCurrentAlarmTableIndex(t *testing.T) {
	makeAlarm := func(moi string) *model.Alarm {
		return &model.Alarm{
			AlarmIdentifier: "11112",
			AdditionalInfo: map[string]string{
				"managed_object_instance": moi,
				"additional_text":         "LTE0",
				"additional_information":  "LTE0(73828545);SCTP LINK FAIL",
			},
		}
	}

	assert.Equal(
		t,
		activeAlarmMatchKey(makeAlarm("Device.FaultMgmt.CurrentAlarm.3.")),
		activeAlarmMatchKey(makeAlarm("Device.FaultMgmt.CurrentAlarm.4.")),
	)
}

func TestActiveAlarmMatchKey_PreservesRealManagedObjectInstance(t *testing.T) {
	makeAlarm := func(moi string) *model.Alarm {
		return &model.Alarm{
			AlarmIdentifier: "11184",
			AdditionalInfo: map[string]string{
				"managed_object_instance": moi,
				"additional_information":  "Cell unavailable",
			},
		}
	}

	assert.Equal(t, "11184|Device.Radio.1", activeAlarmMatchKey(makeAlarm("Device.Radio.1")))
	assert.Equal(t, "11184|Device.Radio.2", activeAlarmMatchKey(makeAlarm("Device.Radio.2")))
	assert.NotEqual(
		t,
		activeAlarmMatchKey(makeAlarm("Device.Radio.1")),
		activeAlarmMatchKey(makeAlarm("Device.Radio.2")),
	)
}

func TestActiveAlarmMatchKey_SeparatesParallelBusinessObjects(t *testing.T) {
	makeAlarm := func(text, info string) *model.Alarm {
		return &model.Alarm{
			AlarmIdentifier: "11112",
			AdditionalInfo: map[string]string{
				"managed_object_instance": "Device.FaultMgmt.ExpeditedEvent.",
				"additional_text":         text,
				"additional_information":  info,
			},
		}
	}

	assert.Equal(t, "11112|LTE0(73828545)", activeAlarmMatchKey(makeAlarm("LTE0", "LTE0(73828545);SCTP LINK FAIL")))
	assert.Equal(t, "11112|LTE1(73828546)", activeAlarmMatchKey(makeAlarm("LTE1", "LTE1(73828546);SCTP LINK FAIL")))
	assert.NotEqual(
		t,
		activeAlarmMatchKey(makeAlarm("LTE0", "LTE0(73828545);SCTP LINK FAIL")),
		activeAlarmMatchKey(makeAlarm("LTE1", "LTE1(73828546);SCTP LINK FAIL")),
	)
}

func TestActiveAlarmQualifier_NormalizesAdditionalInformationWhitespace(t *testing.T) {
	makeAlarm := func(info string) *model.Alarm {
		return &model.Alarm{
			AlarmIdentifier: "11190",
			AdditionalInfo: map[string]string{
				"managed_object_instance": "Device.FaultMgmt.HistoryEvent.1.",
				"additional_information":  info,
			},
		}
	}

	assert.Equal(t, "Gps unavailable, gps antenna open.", activeAlarmQualifier(makeAlarm("  Gps  unavailable,\n gps antenna open.  ")))
	assert.Equal(
		t,
		activeAlarmMatchKey(makeAlarm("Gps unavailable, gps antenna open.")),
		activeAlarmMatchKey(makeAlarm("  Gps  unavailable,\n gps antenna open.  ")),
	)
}

func TestFindActiveAlarmsByIdentifier_ReturnsEverySameIdentifierCandidate(t *testing.T) {
	candidate := &model.Alarm{AlarmIdentifier: " 11184 "}
	alarms := []*model.Alarm{
		{AlarmIdentifier: "11184", AdditionalInfo: map[string]string{"additional_information": "cell=1"}},
		{AlarmIdentifier: "11184", AdditionalInfo: map[string]string{"additional_information": "cell=2"}},
		{AlarmIdentifier: "11190"},
		nil,
	}

	matches := findActiveAlarmsByIdentifier(alarms, candidate)

	assert.Len(t, matches, 2)
	assert.Equal(t, "cell=1", matches[0].AdditionalInfo["additional_information"])
	assert.Equal(t, "cell=2", matches[1].AdditionalInfo["additional_information"])
}
