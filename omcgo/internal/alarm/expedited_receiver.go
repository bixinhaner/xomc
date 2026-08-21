package alarm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// ExpeditedEventPayload is the event payload for device.inform.expedited_alarm.
// Published by the ACS handler when a VALUE CHANGE Inform contains ExpeditedEvent parameters.
type ExpeditedEventPayload struct {
	DeviceSN        string                       `json:"device_sn"`
	ParameterValues []tr069.ParameterValueStruct `json:"parameter_values"`
}

// DeviceLookup resolves device metadata by serial number.
// Implemented by device.Repository; kept as a minimal interface to avoid circular imports.
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// ExpeditedEventReceiver subscribes to expedited alarm events and routes them
// to the appropriate alarm engine method based on NotificationType.
type ExpeditedEventReceiver struct {
	engine           *AlarmEngine
	deviceLookup     DeviceLookup
	eventBus         event.EventBus
	logger           *zap.Logger
	alarmDefRegistry *definition.Registry
	productResolver  definition.ProductResolver
	defMetrics       fallbackMetrics
}

// NewExpeditedEventReceiver creates a new ExpeditedEventReceiver.
func NewExpeditedEventReceiver(engine *AlarmEngine, deviceLookup DeviceLookup, eventBus event.EventBus, logger *zap.Logger) *ExpeditedEventReceiver {
	return &ExpeditedEventReceiver{
		engine:       engine,
		deviceLookup: deviceLookup,
		eventBus:     eventBus,
		logger:       logger,
	}
}

// WithAlarmDefRegistry 注入告警定义 registry。
//
// productResolver 可为 nil：此时仅启用已知告警的 severity 覆盖，不启用 unknown fallback。
func (r *ExpeditedEventReceiver) WithAlarmDefRegistry(alarmDefReg *definition.Registry, productResolver definition.ProductResolver) *ExpeditedEventReceiver {
	if alarmDefReg == nil {
		return r
	}
	r.alarmDefRegistry = alarmDefReg
	r.productResolver = productResolver
	r.defMetrics = alarmDefReg.Metrics()
	return r
}

// Subscribe registers the receiver for device.inform.expedited_alarm events.
func (r *ExpeditedEventReceiver) Subscribe(eventBus event.EventBus) error {
	_, err := eventBus.QueueSubscribe(
		event.SubjectDeviceExpeditedAlarm,
		"expedited-alarm-workers",
		r.handleExpeditedAlarmEvent,
	)
	if err != nil {
		return fmt.Errorf("subscribe expedited alarm events: %w", err)
	}
	r.logger.Info("expedited alarm receiver subscribed", zap.String("subject", event.SubjectDeviceExpeditedAlarm))
	return nil
}

func (r *ExpeditedEventReceiver) handleExpeditedAlarmEvent(ctx context.Context, evt event.Event) error {
	var payload ExpeditedEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		r.logger.Error("decode expedited alarm payload", zap.Error(err))
		return fmt.Errorf("decode expedited alarm payload: %w", err)
	}

	if payload.DeviceSN == "" {
		return fmt.Errorf("expedited alarm payload missing device_sn")
	}

	// Resolve device metadata (ID, carrier)
	dev, err := r.deviceLookup.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		return fmt.Errorf("lookup device %s: %w", payload.DeviceSN, err)
	}
	if dev == nil {
		r.logger.Warn("device not found for expedited alarm, skipping",
			zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	// Filter and parse ExpeditedEvent parameters
	expeditedParams := FilterExpeditedEventParams(payload.ParameterValues)
	events, err := ParseExpeditedEventParams(expeditedParams)
	if err != nil {
		r.logger.Error("parse expedited event params", zap.Error(err))
		return fmt.Errorf("parse expedited event params: %w", err)
	}

	if len(events) == 0 {
		r.logger.Debug("no valid expedited events found",
			zap.String("device_sn", payload.DeviceSN))
		return nil
	}

	hasValidEvent := false
	for _, exp := range events {
		if err := exp.Validate(); err != nil {
			r.logger.Warn("invalid expedited event, skipping",
				zap.Error(err),
				zap.String("device_sn", payload.DeviceSN),
				zap.Int("index", exp.Index))
			continue
		}
		hasValidEvent = true

		switch exp.NotificationType {
		case NotificationNewAlarm:
			alarm := exp.ToModel(dev.ID, payload.DeviceSN, dev.Carrier)
			addUPSEquipmentInfo(dev, alarm.AdditionalInfo)
			drop, err := applyUnknownAlarmFallback(ctx, r.alarmDefRegistry, r.productResolver, r.defMetrics, r.logger, alarm, dev.ProductClass)
			if err != nil {
				r.logger.Warn("apply expedited alarm fallback failed (proceed without fallback)",
					zap.Error(err),
					zap.String("alarm_identifier", exp.AlarmIdentifier))
			} else if drop {
				continue
			}
			if err := r.engine.Process(ctx, alarm); err != nil {
				r.logger.Error("process NewAlarm",
					zap.Error(err),
					zap.String("device_sn", payload.DeviceSN),
					zap.String("alarm_identifier", exp.AlarmIdentifier))
				continue
			}
			r.logger.Info("expedited NewAlarm processed",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("alarm_identifier", exp.AlarmIdentifier))

		case NotificationChangedAlarm:
			alarm := exp.ToModel(dev.ID, payload.DeviceSN, dev.Carrier)
			addUPSEquipmentInfo(dev, alarm.AdditionalInfo)
			if err := applyAlarmDefinitionSeverity(ctx, r.alarmDefRegistry, alarm); err != nil {
				r.logger.Warn("resolve expedited alarm definition severity failed (proceed with source severity)",
					zap.Error(err),
					zap.String("alarm_identifier", exp.AlarmIdentifier))
			}
			if err := r.engine.UpdateByEvent(ctx, alarm); err != nil {
				r.logger.Error("process ChangedAlarm",
					zap.Error(err),
					zap.String("device_sn", payload.DeviceSN),
					zap.String("alarm_identifier", exp.AlarmIdentifier))
				continue
			}
			r.logger.Info("expedited ChangedAlarm processed",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("alarm_identifier", exp.AlarmIdentifier))

		case NotificationClearedAlarm:
			alarm := exp.ToModel(dev.ID, payload.DeviceSN, dev.Carrier)
			addUPSEquipmentInfo(dev, alarm.AdditionalInfo)
			if err := r.engine.AutoClear(ctx, alarm); err != nil {
				r.logger.Error("process ClearedAlarm",
					zap.Error(err),
					zap.String("device_sn", payload.DeviceSN),
					zap.String("alarm_identifier", exp.AlarmIdentifier))
				continue
			}
			r.logger.Info("expedited ClearedAlarm processed",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("alarm_identifier", exp.AlarmIdentifier))

		default:
			r.logger.Warn("unknown notification type, skipping",
				zap.String("notification_type", exp.NotificationType),
				zap.String("device_sn", payload.DeviceSN),
				zap.Int("index", exp.Index))
		}
	}
	if hasValidEvent {
		publishAlarmSyncRequest(ctx, r.eventBus, r.logger, payload.DeviceSN)
	}

	return nil
}

// deviceIDOrZero safely parses a UUID string, returning zero UUID on failure.
func deviceIDOrZero(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
