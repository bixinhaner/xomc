package alarm

import (
	"context"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	additionalInfoManagedObjectInstance = "managed_object_instance"
	additionalInfoAdditionalInformation = "additional_information"
	additionalInfoAdditionalText        = "additional_text"
)

func activeAlarmQualifier(alarm *model.Alarm) string {
	if alarm == nil || alarm.AdditionalInfo == nil {
		return ""
	}

	if value := stableManagedObjectInstance(alarm.AdditionalInfo[additionalInfoManagedObjectInstance]); value != "" {
		return value
	}

	additionalText := normalizeAlarmIdentityValue(alarm.AdditionalInfo[additionalInfoAdditionalText])
	additionalInformation := normalizeAlarmIdentityValue(alarm.AdditionalInfo[additionalInfoAdditionalInformation])
	if value := alarmObjectScopePrefix(additionalText, additionalInformation); value != "" {
		return value
	}
	if additionalText != "" {
		return additionalText
	}
	return additionalInformation
}

func stableManagedObjectInstance(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{
		"Device.FaultMgmt.CurrentAlarm.",
		"Device.FaultMgmt.ExpeditedEvent.",
		"Device.FaultMgmt.HistoryEvent.",
		"InternetGatewayDevice.FaultMgmt.CurrentAlarm.",
		"InternetGatewayDevice.FaultMgmt.ExpeditedEvent.",
		"InternetGatewayDevice.FaultMgmt.HistoryEvent.",
	} {
		if strings.HasPrefix(value, prefix) {
			return ""
		}
	}
	return value
}

func normalizeAlarmIdentityValue(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func alarmObjectScopePrefix(additionalText, additionalInformation string) string {
	if additionalText == "" || additionalInformation == "" {
		return ""
	}
	prefix := additionalInformation
	if index := strings.IndexByte(prefix, ';'); index >= 0 {
		prefix = prefix[:index]
	}
	if strings.HasPrefix(prefix, additionalText+"(") && strings.HasSuffix(prefix, ")") {
		return prefix
	}
	return ""
}

func activeAlarmMatchKey(alarm *model.Alarm) string {
	if alarm == nil {
		return ""
	}

	identifier := strings.TrimSpace(alarm.AlarmIdentifier)
	qualifier := activeAlarmQualifier(alarm)
	if qualifier == "" {
		return identifier
	}

	return identifier + "|" + qualifier
}

func activeAlarmMatches(existing, candidate *model.Alarm) bool {
	if existing == nil || candidate == nil {
		return false
	}
	if strings.TrimSpace(existing.AlarmIdentifier) != strings.TrimSpace(candidate.AlarmIdentifier) {
		return false
	}

	existingQualifier := activeAlarmQualifier(existing)
	candidateQualifier := activeAlarmQualifier(candidate)
	if candidateQualifier == "" {
		return existingQualifier == ""
	}

	return existingQualifier == candidateQualifier
}

func findMatchingActiveAlarm(alarms []*model.Alarm, candidate *model.Alarm) *model.Alarm {
	for _, alarm := range alarms {
		if activeAlarmMatches(alarm, candidate) {
			return alarm
		}
	}

	return nil
}

func findActiveAlarmsByIdentifier(alarms []*model.Alarm, candidate *model.Alarm) []*model.Alarm {
	if candidate == nil {
		return nil
	}

	identifier := strings.TrimSpace(candidate.AlarmIdentifier)
	matches := make([]*model.Alarm, 0)
	for _, alarm := range alarms {
		if alarm == nil {
			continue
		}
		if strings.TrimSpace(alarm.AlarmIdentifier) == identifier {
			matches = append(matches, alarm)
		}
	}
	return matches
}

func loadMatchingActiveAlarm(ctx context.Context, store AlarmStore, candidate *model.Alarm) (*model.Alarm, error) {
	if candidate == nil || store == nil {
		return nil, nil
	}

	alarms, err := store.GetActiveByDeviceSN(ctx, candidate.DeviceSN)
	if err != nil {
		return nil, err
	}

	return findMatchingActiveAlarm(alarms, candidate), nil
}

func indexActiveAlarms(alarms []*model.Alarm) map[string]*model.Alarm {
	indexed := make(map[string]*model.Alarm, len(alarms))
	for _, alarm := range alarms {
		if alarm == nil {
			continue
		}
		indexed[activeAlarmMatchKey(alarm)] = alarm
	}

	return indexed
}
