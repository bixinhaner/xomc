package alarm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
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
//
// T-0098 P2-10：在调用 engine.Process 前对 alarm.AlarmIdentifier 做"识别 → fallback
// 决策"。已知 identifier 直接放过；未知 → 查 device.product.enable_unknown_alarm，
//
//	true  → severity=Warning + IsUnknown=true 写入活动告警（治理闭环用）
//	false → 丢弃 + INFO + alarm_unknown_total{action=dropped}
//
// alarmDefRegistry / productResolver 任一为 nil 时退化到 P2-10 之前的行为（不做识别检查）。
type AlarmReceiver struct {
	engine           *AlarmEngine
	eventBus         event.EventBus
	logger           *zap.Logger
	deviceReader     deviceReader
	alarmDefRegistry *definition.Registry
	productResolver  definition.ProductResolver
	defMetrics       fallbackMetrics // 取自 alarmDefRegistry.Metrics() 的最小子集
}

type deviceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error)
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// fallbackMetrics 是 receiver 需要的 alarm_unknown_total 计数接口。
type fallbackMetrics interface {
	UnknownDropped()
	UnknownKept()
}

// NewAlarmReceiver creates a new AlarmReceiver.
func NewAlarmReceiver(engine *AlarmEngine, eventBus event.EventBus, logger *zap.Logger) *AlarmReceiver {
	return &AlarmReceiver{engine: engine, eventBus: eventBus, logger: logger}
}

// WithDeviceReader enables alarm field backfill from the device table when the
// incoming alarm payload omits device-derived fields such as technology.
func (r *AlarmReceiver) WithDeviceReader(reader deviceReader) *AlarmReceiver {
	r.deviceReader = reader
	return r
}

// WithAlarmDefRegistry 启用 T-0098 P2-10 fallback 决策（设计 §3.5）。
//
// alarmDefReg / productResolver 任一为 nil → 等价于不调用本方法。
func (r *AlarmReceiver) WithAlarmDefRegistry(alarmDefReg *definition.Registry, productResolver definition.ProductResolver) *AlarmReceiver {
	if alarmDefReg == nil || productResolver == nil {
		return r
	}
	r.alarmDefRegistry = alarmDefReg
	r.productResolver = productResolver
	r.defMetrics = alarmDefReg.Metrics()
	return r
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
	r.backfillDeviceFields(ctx, alarm)

	// T-0098 P2-10 fallback 决策：identifier 不在 alarm_definitions → 查 product.enable_unknown_alarm
	if drop, err := r.applyFallback(ctx, alarm, payload); err != nil {
		r.logger.Warn("apply alarm fallback failed (proceed without fallback)",
			zap.Error(err),
			zap.String("alarm_identifier", payload.AlarmIdentifier))
	} else if drop {
		// product.enable_unknown_alarm = false → 静默丢弃
		return nil
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

func (r *AlarmReceiver) backfillDeviceFields(ctx context.Context, alarm *model.Alarm) {
	if r.deviceReader == nil || alarm.Technology != nil || alarm.DeviceSN == "" {
		return
	}

	var device *model.Device
	var err error
	device, err = r.deviceReader.GetByID(ctx, alarm.DeviceID)
	if err != nil || device == nil {
		device, err = r.deviceReader.GetBySerialNumber(ctx, alarm.DeviceSN)
	}
	if err != nil || device == nil || device.Technology == "" {
		return
	}

	technology := string(device.Technology)
	alarm.Technology = &technology
}

// applyFallback 执行 §3.5 fallback 决策。
//
// 返回 (drop=true, nil) 表示告警被丢弃（调用方应直接 return nil）。
// 返回 (drop=false, nil) 表示告警继续走 engine.Process（已知或被保留为 unknown）。
// 返回 (false, err) 表示 fallback 流程出错；调用方按既有路径继续即可。
//
// alarmDefRegistry 或 productResolver 未注入时直接返回 (false, nil)，行为与 P2-10 之前一致。
func (r *AlarmReceiver) applyFallback(ctx context.Context, alarm *model.Alarm, payload AlarmPayload) (bool, error) {
	if r.alarmDefRegistry == nil || r.productResolver == nil {
		return false, nil
	}
	if _, err := r.alarmDefRegistry.Lookup(ctx, alarm.AlarmIdentifier); err == nil {
		return false, nil
	} else if !errors.Is(err, definition.ErrUnknownIdentifier) {
		return false, err
	}

	// 未命中：查 product.enable_unknown_alarm
	productClass := payload.AlarmSource // 约定 alarm_source 透传 device.ProductClass
	if productClass == "" {
		r.dropUnknown(alarm, "no product_class in payload")
		return true, nil
	}
	prod, err := r.productResolver.ResolveByProductClass(ctx, productClass)
	if err != nil {
		return false, fmt.Errorf("resolve product %s: %w", productClass, err)
	}
	if prod == nil || !prod.EnableUnknownAlarm {
		r.dropUnknown(alarm, "enable_unknown_alarm=false")
		return true, nil
	}

	// 保留为 Warning（severity_code=31004）+ IsUnknown=true
	alarm.Severity = model.AlarmSeverity(31004)
	alarm.IsUnknown = true
	if r.defMetrics != nil {
		r.defMetrics.UnknownKept()
	}
	r.logger.Info("unknown alarm kept as fallback",
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.String("device_sn", alarm.DeviceSN),
		zap.String("product_class", productClass),
	)
	return false, nil
}

func (r *AlarmReceiver) dropUnknown(alarm *model.Alarm, reason string) {
	if r.defMetrics != nil {
		r.defMetrics.UnknownDropped()
	}
	r.logger.Info("unknown alarm dropped",
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.String("device_sn", alarm.DeviceSN),
		zap.String("reason", reason),
	)
}
