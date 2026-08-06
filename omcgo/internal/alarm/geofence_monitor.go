package alarm

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

const (
	AlarmCodeGeofenceLocationOutside = "GEOFENCE_LOCATION_OUTSIDE"
	GeofenceAlarmExitedQueue         = "alarm-geofence-monitor-exited"
	GeofenceAlarmEnteredQueue        = "alarm-geofence-monitor-entered"
)

// GeofenceAlarmMonitor projects device-level geofence state edges into the
// existing alarm lifecycle. Rule-level events remain audit facts and do not
// create duplicate active alarms.
type GeofenceAlarmMonitor struct {
	engine *AlarmEngine
	logger *zap.Logger
}

func NewGeofenceAlarmMonitor(
	engine *AlarmEngine,
	logger *zap.Logger,
) *GeofenceAlarmMonitor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GeofenceAlarmMonitor{engine: engine, logger: logger}
}

func (m *GeofenceAlarmMonitor) Subscribe(bus event.EventBus) error {
	for _, spec := range []struct {
		subject string
		queue   string
	}{
		{
			subject: event.SubjectGeofenceDeviceExited,
			queue:   GeofenceAlarmExitedQueue,
		},
		{
			subject: event.SubjectGeofenceDeviceEntered,
			queue:   GeofenceAlarmEnteredQueue,
		},
	} {
		if _, err := bus.QueueSubscribe(
			spec.subject,
			spec.queue,
			m.handle,
		); err != nil {
			return fmt.Errorf("subscribe %s: %w", spec.subject, err)
		}
	}
	return nil
}

func (m *GeofenceAlarmMonitor) handle(
	ctx context.Context,
	evt event.Event,
) error {
	var payload event.GeofenceDeviceStatePayload
	if err := evt.DecodePayload(&payload); err != nil {
		m.logger.Warn("decode geofence device state event", zap.Error(err))
		return nil
	}
	if !validGeofenceAlarmEdge(evt.Subject, payload) {
		m.logger.Warn(
			"ignore invalid geofence device state edge",
			zap.String("subject", evt.Subject),
			zap.String("serial_number", payload.SerialNumber),
			zap.String("effective_state", payload.EffectiveState),
		)
		return nil
	}

	alarm := geofenceLocationAlarm(payload)
	switch evt.Subject {
	case event.SubjectGeofenceDeviceExited:
		if err := m.engine.Process(ctx, alarm); err != nil {
			return fmt.Errorf("raise geofence location alarm: %w", err)
		}
	case event.SubjectGeofenceDeviceEntered:
		if err := m.engine.AutoClear(ctx, alarm); err != nil {
			return fmt.Errorf("clear geofence location alarm: %w", err)
		}
	}
	return nil
}

func validGeofenceAlarmEdge(
	subject string,
	payload event.GeofenceDeviceStatePayload,
) bool {
	if payload.DeviceID == uuid.Nil || payload.SerialNumber == "" {
		return false
	}
	switch subject {
	case event.SubjectGeofenceDeviceExited:
		return payload.EffectiveState == "outside"
	case event.SubjectGeofenceDeviceEntered:
		return payload.EffectiveState == "inside"
	default:
		return false
	}
}

func geofenceLocationAlarm(
	payload event.GeofenceDeviceStatePayload,
) *model.Alarm {
	additionalInfo := map[string]string{
		"observation_version":     strconv.FormatInt(payload.ObservationVersion, 10),
		"effective_state_version": strconv.FormatInt(payload.EffectiveStateVersion, 10),
		"required_action_level":   payload.RequiredActionLevel,
	}
	if payload.TriggerBindingID != nil {
		additionalInfo["geofence_binding_id"] =
			payload.TriggerBindingID.String()
	}
	return &model.Alarm{
		DeviceID:        payload.DeviceID,
		DeviceSN:        payload.SerialNumber,
		Carrier:         model.CarrierCode(payload.Carrier),
		Severity:        geofenceAlarmSeverity(payload.RequiredActionLevel),
		AlarmType:       "geofence",
		AlarmIdentifier: AlarmCodeGeofenceLocationOutside,
		Description:     "Device is outside the configured geofence",
		RaisedAt:        payload.OccurredAt,
		AlarmSource:     strPtr("omc"),
		EventType:       strPtr("location_outside"),
		AdditionalInfo:  additionalInfo,
	}
}

func geofenceAlarmSeverity(actionLevel string) model.AlarmSeverity {
	switch actionLevel {
	case "deactivate":
		return model.AlarmCritical
	case "manual_review":
		return model.AlarmMajor
	default:
		return model.AlarmWarning
	}
}
