package alarm

import (
	"context"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	additionalInfoManagedObjectInstance = "managed_object_instance"
	additionalInfoAdditionalInformation = "additional_information"
)

func activeAlarmQualifier(alarm *model.Alarm) string {
	if alarm == nil || alarm.AdditionalInfo == nil {
		return ""
	}

	if value := strings.TrimSpace(alarm.AdditionalInfo[additionalInfoManagedObjectInstance]); value != "" {
		return value
	}

	return strings.TrimSpace(alarm.AdditionalInfo[additionalInfoAdditionalInformation])
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