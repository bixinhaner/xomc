package alarm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// AlarmPayload is the event payload for device alarm events.
type AlarmPayload struct {
	DeviceID        string            `json:"device_id"`
	DeviceSN        string            `json:"device_sn"`
	DeviceName      string            `json:"device_name,omitempty"`
	Carrier         string            `json:"carrier"`
	Technology      string            `json:"technology,omitempty"`
	AlarmIdentifier string            `json:"alarm_identifier"`
	AlarmType       string            `json:"alarm_type"`
	AlarmSource     string            `json:"alarm_source,omitempty"`
	EventType       string            `json:"event_type,omitempty"`
	Description     string            `json:"description"`
	Severity        int               `json:"severity"`
	RaisedAt        time.Time         `json:"raised_at"`
	Additional      map[string]string `json:"additional,omitempty"`
}

// AlarmReceiver subscribes to device alarm events and processes them.
type AlarmReceiver struct {
	engine   *AlarmEngine
	eventBus event.EventBus
	logger   *zap.Logger
}

// NewAlarmReceiver creates a new AlarmReceiver.
func NewAlarmReceiver(engine *AlarmEngine, eventBus event.EventBus, logger *zap.Logger) *AlarmReceiver {
	return &AlarmReceiver{engine: engine, eventBus: eventBus, logger: logger}
}

// Subscribe registers the receiver for device alarm events.
func (r *AlarmReceiver) Subscribe(eventBus event.EventBus) error {
	_, err := eventBus.QueueSubscribe(
		event.SubjectDeviceAlarm,
		"alarm-workers",
		r.handleAlarmEvent,
	)
	if err != nil {
		return fmt.Errorf("subscribe alarm events: %w", err)
	}
	r.logger.Info("alarm receiver subscribed", zap.String("subject", event.SubjectDeviceAlarm))
	return nil
}

func (r *AlarmReceiver) handleAlarmEvent(ctx context.Context, evt event.Event) error {
	var payload AlarmPayload
	if err := evt.DecodePayload(&payload); err != nil {
		r.logger.Error("decode alarm payload", zap.Error(err))
		return fmt.Errorf("decode alarm payload: %w", err)
	}

	deviceID, err := uuid.Parse(payload.DeviceID)
	if err != nil {
		r.logger.Error("parse device_id", zap.Error(err), zap.String("device_id", payload.DeviceID))
		return fmt.Errorf("parse device_id: %w", err)
	}

	alarm := &model.Alarm{
		DeviceID:       deviceID,
		DeviceSN:       payload.DeviceSN,
		DeviceName:     strPtr(payload.DeviceName),
		Carrier:        model.CarrierCode(payload.Carrier),
		Technology:     strPtr(payload.Technology),
		AlarmIdentifier: payload.AlarmIdentifier,
		AlarmType:      payload.AlarmType,
		AlarmSource:    strPtr(payload.AlarmSource),
		EventType:      strPtr(payload.EventType),
		Description:    payload.Description,
		Severity:       model.AlarmSeverity(payload.Severity),
		RaisedAt:       payload.RaisedAt,
		AdditionalInfo: payload.Additional,
	}

	if err := r.engine.Process(ctx, alarm); err != nil {
		r.logger.Error("process alarm",
			zap.Error(err),
			zap.String("device_sn", payload.DeviceSN),
			zap.String("alarm_identifier", payload.AlarmIdentifier))
		return fmt.Errorf("process alarm: %w", err)
	}

	// Publish alarm.sync.requested to trigger a full alarm sync via GPV
	if r.eventBus != nil {
		syncPayload := map[string]string{"device_sn": payload.DeviceSN}
		if syncEvt, err := event.NewEvent(event.SubjectAlarmSyncRequested, syncPayload); err == nil {
			if pubErr := r.eventBus.Publish(ctx, event.SubjectAlarmSyncRequested, syncEvt); pubErr != nil {
				r.logger.Warn("publish alarm sync request", zap.Error(pubErr))
			}
		}
	}

	return nil
}
