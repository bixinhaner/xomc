package alarm

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
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

type informAlarmEventPayload struct {
	DeviceID      tr069.DeviceId               `json:"device_id"`
	Events        []string                     `json:"events"`
	ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
	CurrentTime   time.Time                    `json:"current_time"`
}

type alarmInfoEvent struct {
	Index                 int
	NotificationType      string
	AlarmIdentifier       string
	PerceivedSeverity     string
	EventType             string
	ProbableCause         string
	SpecificProblem       string
	AdditionalInformation string
	AdditionalText        string
	EventTime             time.Time
	ManagedObjectInstance string
}

var alarmInfoFieldRE = regexp.MustCompile(`^(?:Device|InternetGatewayDevice)\.Services\.FAPService\.\d+\.FAPControl\.[^.]+\.AlarmInfo\.(\d+)\.(.+)$`)

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

// WithAlarmDefRegistry 注入告警定义 registry。
//
// alarmDefReg 为 nil → 等价于不调用本方法。
// productResolver 可为 nil：此时仅启用已知告警的 severity 覆盖，不启用 unknown fallback。
func (r *AlarmReceiver) WithAlarmDefRegistry(alarmDefReg *definition.Registry, productResolver definition.ProductResolver) *AlarmReceiver {
	if alarmDefReg == nil {
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
	if err := evt.DecodePayload(&payload); err == nil && payload.AlarmIdentifier != "" {
		if err := r.processAlarmPayload(ctx, payload); err != nil {
			return err
		}
		publishAlarmSyncRequest(ctx, r.eventBus, r.logger, payload.DeviceSN)
		return nil
	}

	var informPayload informAlarmEventPayload
	if err := evt.DecodePayload(&informPayload); err != nil {
		r.logger.Error("decode alarm payload", zap.Error(err))
		return fmt.Errorf("decode alarm payload: %w", err)
	}

	payloads, err := r.buildAlarmPayloadsFromInform(ctx, informPayload)
	if err != nil {
		r.logger.Error("build alarm payloads from inform",
			zap.Error(err),
			zap.String("device_sn", informPayload.DeviceID.SerialNumber))
		return err
	}

	for _, normalized := range payloads {
		if err := r.processAlarmPayload(ctx, normalized); err != nil {
			return err
		}
	}
	publishAlarmSyncRequest(ctx, r.eventBus, r.logger, informPayload.DeviceID.SerialNumber)
	return nil
}

func (r *AlarmReceiver) processAlarmPayload(ctx context.Context, payload AlarmPayload) error {
	notificationType := payload.Additional["notification_type"]
	if notificationType == NotificationClearedAlarm {
		clearAlarm := &model.Alarm{
			DeviceSN:        payload.DeviceSN,
			AlarmIdentifier: payload.AlarmIdentifier,
			AdditionalInfo:  payload.Additional,
		}
		if err := r.engine.AutoClear(ctx, clearAlarm); err != nil {
			r.logger.Error("clear alarm",
				zap.Error(err),
				zap.String("device_sn", payload.DeviceSN),
				zap.String("alarm_identifier", payload.AlarmIdentifier))
			return fmt.Errorf("clear alarm: %w", err)
		}
		return nil
	}

	deviceID, err := uuid.Parse(payload.DeviceID)
	if err != nil {
		r.logger.Error("parse device_id", zap.Error(err), zap.String("device_id", payload.DeviceID))
		return fmt.Errorf("parse device_id: %w", err)
	}

	alarm := &model.Alarm{
		DeviceID:        deviceID,
		DeviceSN:        payload.DeviceSN,
		DeviceName:      strPtr(payload.DeviceName),
		Carrier:         model.CarrierCode(payload.Carrier),
		Technology:      strPtr(payload.Technology),
		AlarmIdentifier: payload.AlarmIdentifier,
		AlarmType:       payload.AlarmType,
		AlarmSource:     strPtr(payload.AlarmSource),
		EventType:       strPtr(payload.EventType),
		Description:     payload.Description,
		Severity:        model.AlarmSeverity(payload.Severity),
		RaisedAt:        payload.RaisedAt,
		AdditionalInfo:  payload.Additional,
	}
	probableCause := firstNonEmpty(payload.Additional["probable_cause"], payload.Description, payload.AlarmIdentifier)
	alarm.ProbableCause = strPtr(probableCause)
	r.backfillDeviceFields(ctx, alarm)

	if notificationType == NotificationChangedAlarm {
		if err := applyAlarmDefinitionSeverity(ctx, r.alarmDefRegistry, alarm); err != nil {
			r.logger.Warn("resolve alarm definition severity failed (proceed with source severity)",
				zap.Error(err),
				zap.String("alarm_identifier", payload.AlarmIdentifier))
		}
		if err := r.engine.UpdateByEvent(ctx, alarm); err != nil {
			r.logger.Error("update alarm",
				zap.Error(err),
				zap.String("device_sn", payload.DeviceSN),
				zap.String("alarm_identifier", payload.AlarmIdentifier))
			return fmt.Errorf("update alarm: %w", err)
		}
		return nil
	}

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

	return nil
}

func (r *AlarmReceiver) buildAlarmPayloadsFromInform(ctx context.Context, payload informAlarmEventPayload) ([]AlarmPayload, error) {
	currentAlarms, err := ParseCurrentAlarmParams(payload.ParameterList)
	if err != nil {
		return nil, fmt.Errorf("parse current alarm params: %w", err)
	}
	if len(currentAlarms) > 0 {
		device, err := r.lookupDeviceBySN(ctx, payload.DeviceID.SerialNumber)
		if err != nil {
			return nil, err
		}
		return currentAlarmPayloads(device, currentAlarms, payload.CurrentTime), nil
	}

	alarmInfos, err := parseAlarmInfoParams(payload.ParameterList)
	if err != nil {
		return nil, fmt.Errorf("parse alarm info params: %w", err)
	}
	if len(alarmInfos) == 0 {
		expeditedParams := FilterExpeditedEventParams(payload.ParameterList)
		expeditedEvents, err := ParseExpeditedEventParams(expeditedParams)
		if err != nil {
			return nil, fmt.Errorf("parse expedited event params: %w", err)
		}
		if len(expeditedEvents) == 0 {
			return nil, nil
		}

		device, err := r.lookupDeviceBySN(ctx, payload.DeviceID.SerialNumber)
		if err != nil {
			return nil, err
		}
		return expeditedEventPayloads(device, expeditedEvents, payload.CurrentTime), nil
	}

	device, err := r.lookupDeviceBySN(ctx, payload.DeviceID.SerialNumber)
	if err != nil {
		return nil, err
	}
	return alarmInfoPayloads(device, alarmInfos, payload.CurrentTime), nil
}

func (r *AlarmReceiver) lookupDeviceBySN(ctx context.Context, deviceSN string) (*model.Device, error) {
	if deviceSN == "" {
		return nil, fmt.Errorf("alarm inform payload missing device serial number")
	}
	if r.deviceReader == nil {
		return nil, fmt.Errorf("alarm inform payload requires device reader for %s", deviceSN)
	}
	device, err := r.deviceReader.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device by serial number %s: %w", deviceSN, err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found for serial number %s", deviceSN)
	}
	return device, nil
}

func currentAlarmPayloads(device *model.Device, alarms []TR069Alarm, _ time.Time) []AlarmPayload {
	result := make([]AlarmPayload, 0, len(alarms))
	for _, alarm := range alarms {
		description := firstNonEmpty(alarm.SpecificProblem, alarm.ProbableCause, alarm.AlarmIdentifier)
		additional := map[string]string{}
		if alarm.AdditionalInformation != "" {
			additional["additional_information"] = alarm.AdditionalInformation
		}
		if alarm.AdditionalText != "" {
			additional["additional_text"] = alarm.AdditionalText
		}
		if alarm.ManagedObjectInstance != "" {
			additional["managed_object_instance"] = alarm.ManagedObjectInstance
		}
		if alarm.SpecificProblem != "" {
			additional["specific_problem"] = alarm.SpecificProblem
		}
		if alarm.ProbableCause != "" {
			additional["probable_cause"] = alarm.ProbableCause
		}
		result = append(result, AlarmPayload{
			DeviceID:        device.ID.String(),
			DeviceSN:        device.SerialNumber,
			DeviceName:      device.DeviceName,
			Carrier:         string(device.Carrier),
			Technology:      string(device.Technology),
			AlarmIdentifier: alarm.AlarmIdentifier,
			AlarmType:       firstNonEmpty(strings.TrimSpace(alarm.EventType), "alarm"),
			AlarmSource:     device.ProductClass,
			EventType:       alarm.EventType,
			Description:     description,
			Severity:        int(mapSeverity(alarm.PerceivedSeverity)),
			Additional:      additional,
		})
	}
	return result
}

func alarmInfoPayloads(device *model.Device, alarms []alarmInfoEvent, _ time.Time) []AlarmPayload {
	result := make([]AlarmPayload, 0, len(alarms))
	for _, alarm := range alarms {
		additional := map[string]string{}
		if alarm.AdditionalInformation != "" {
			additional["additional_information"] = alarm.AdditionalInformation
		}
		if alarm.AdditionalText != "" {
			additional["additional_text"] = alarm.AdditionalText
		}
		if alarm.ManagedObjectInstance != "" {
			additional["managed_object_instance"] = alarm.ManagedObjectInstance
		}
		if alarm.NotificationType != "" {
			additional["notification_type"] = alarm.NotificationType
		}
		if alarm.SpecificProblem != "" {
			additional["specific_problem"] = alarm.SpecificProblem
		}
		if alarm.ProbableCause != "" {
			additional["probable_cause"] = alarm.ProbableCause
		}
		result = append(result, AlarmPayload{
			DeviceID:        device.ID.String(),
			DeviceSN:        device.SerialNumber,
			DeviceName:      device.DeviceName,
			Carrier:         string(device.Carrier),
			Technology:      string(device.Technology),
			AlarmIdentifier: alarm.AlarmIdentifier,
			AlarmType:       firstNonEmpty(strings.TrimSpace(alarm.EventType), "alarm"),
			AlarmSource:     device.ProductClass,
			EventType:       alarm.EventType,
			Description:     firstNonEmpty(alarm.SpecificProblem, alarm.ProbableCause, alarm.AlarmIdentifier),
			Severity:        int(mapSeverity(alarm.PerceivedSeverity)),
			Additional:      additional,
		})
	}
	return result
}

func expeditedEventPayloads(device *model.Device, alarms []ExpeditedEvent, _ time.Time) []AlarmPayload {
	result := make([]AlarmPayload, 0, len(alarms))
	for _, alarm := range alarms {
		additional := map[string]string{}
		if alarm.AdditionalInformation != "" {
			additional["additional_information"] = alarm.AdditionalInformation
		}
		if alarm.AdditionalText != "" {
			additional["additional_text"] = alarm.AdditionalText
		}
		if alarm.ManagedObjectInstance != "" {
			additional["managed_object_instance"] = alarm.ManagedObjectInstance
		}
		if alarm.ProbableCause != "" {
			additional["probable_cause"] = alarm.ProbableCause
		}
		if alarm.SpecificProblem != "" {
			additional["specific_problem"] = alarm.SpecificProblem
		}
		if alarm.NotificationType != "" {
			additional["notification_type"] = alarm.NotificationType
		}

		result = append(result, AlarmPayload{
			DeviceID:        device.ID.String(),
			DeviceSN:        device.SerialNumber,
			DeviceName:      device.DeviceName,
			Carrier:         string(device.Carrier),
			Technology:      string(device.Technology),
			AlarmIdentifier: alarm.AlarmIdentifier,
			AlarmType:       firstNonEmpty(strings.TrimSpace(alarm.EventType), "alarm"),
			AlarmSource:     device.ProductClass,
			EventType:       alarm.EventType,
			Description:     firstNonEmpty(alarm.SpecificProblem, alarm.ProbableCause, alarm.AlarmIdentifier),
			Severity:        int(mapSeverity(alarm.PerceivedSeverity)),
			Additional:      additional,
		})
	}
	return result
}

func parseAlarmInfoParams(params []tr069.ParameterValueStruct) ([]alarmInfoEvent, error) {
	indexMap := make(map[int]*alarmInfoEvent)
	for _, p := range params {
		m := alarmInfoFieldRE.FindStringSubmatch(p.Name)
		if m == nil {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		field := m[2]

		ev, ok := indexMap[idx]
		if !ok {
			ev = &alarmInfoEvent{Index: idx}
			indexMap[idx] = ev
		}

		switch field {
		case "NotificationType":
			ev.NotificationType = p.Value
		case "AlarmIdentifier":
			ev.AlarmIdentifier = p.Value
		case "PerceivedSeverity":
			ev.PerceivedSeverity = p.Value
		case "EventType":
			ev.EventType = p.Value
		case "ProbableCause":
			ev.ProbableCause = p.Value
		case "SpecificProblem":
			ev.SpecificProblem = p.Value
		case "AdditionalInformation":
			ev.AdditionalInformation = p.Value
		case "AdditionalText":
			ev.AdditionalText = p.Value
		case "EventTime":
			ev.EventTime = parseTR069Time(p.Value)
		case "ManagedObjectInstance", "FaultLocation":
			ev.ManagedObjectInstance = p.Value
		}
	}

	result := make([]alarmInfoEvent, 0, len(indexMap))
	for _, ev := range indexMap {
		if ev.AlarmIdentifier == "" {
			continue
		}
		result = append(result, *ev)
	}
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Index < result[i].Index {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
	productClass := payload.AlarmSource // 约定 alarm_source 透传 device.ProductClass
	return applyUnknownAlarmFallback(ctx, r.alarmDefRegistry, r.productResolver, r.defMetrics, r.logger, alarm, productClass)
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

func applyAlarmDefinitionSeverity(ctx context.Context, alarmDefRegistry *definition.Registry, alarm *model.Alarm) error {
	if alarmDefRegistry == nil || alarm == nil || alarm.AlarmIdentifier == "" {
		return nil
	}
	rd, err := alarmDefRegistry.Lookup(ctx, alarm.AlarmIdentifier)
	if err == nil {
		if rd.SeverityCode != 0 {
			alarm.Severity = model.AlarmSeverity(rd.SeverityCode)
		}
		return nil
	}
	if errors.Is(err, definition.ErrUnknownIdentifier) {
		return nil
	}
	return err
}

func applyUnknownAlarmFallback(
	ctx context.Context,
	alarmDefRegistry *definition.Registry,
	productResolver definition.ProductResolver,
	metrics fallbackMetrics,
	logger *zap.Logger,
	alarm *model.Alarm,
	productClass string,
) (bool, error) {
	if alarmDefRegistry == nil {
		return false, nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if rd, err := alarmDefRegistry.Lookup(ctx, alarm.AlarmIdentifier); err == nil {
		if rd.SeverityCode != 0 {
			alarm.Severity = model.AlarmSeverity(rd.SeverityCode)
		}
		return false, nil
	} else if !errors.Is(err, definition.ErrUnknownIdentifier) {
		return false, err
	}
	if productResolver == nil {
		return false, nil
	}

	if productClass == "" {
		if metrics != nil {
			metrics.UnknownDropped()
		}
		logger.Info("unknown alarm dropped",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("device_sn", alarm.DeviceSN),
			zap.String("reason", "no product_class in payload"),
		)
		return true, nil
	}
	prod, err := productResolver.ResolveByProductClass(ctx, productClass)
	if err != nil {
		return false, fmt.Errorf("resolve product %s: %w", productClass, err)
	}
	if prod == nil || !prod.EnableUnknownAlarm {
		if metrics != nil {
			metrics.UnknownDropped()
		}
		logger.Info("unknown alarm dropped",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("device_sn", alarm.DeviceSN),
			zap.String("reason", "enable_unknown_alarm=false"),
		)
		return true, nil
	}

	alarm.Severity = model.AlarmSeverity(31004)
	alarm.IsUnknown = true
	if metrics != nil {
		metrics.UnknownKept()
	}
	logger.Info("unknown alarm kept as fallback",
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.String("device_sn", alarm.DeviceSN),
		zap.String("product_class", productClass),
	)
	return false, nil
}
