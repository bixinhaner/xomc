package alarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// AlarmSyncProcessor subscribes to GPV response events, parses TR-069 alarm parameters,
// computes a three-way diff against local active alarms, and applies changes.
type AlarmSyncProcessor struct {
	engine           *AlarmEngine
	store            AlarmStore
	syncService      *AlarmSyncService
	eventBus         event.EventBus
	logger           *zap.Logger
	alarmDefRegistry *definition.Registry
	productResolver  definition.ProductResolver
	defMetrics       fallbackMetrics
	deviceReader     deviceReader
}

// NewAlarmSyncProcessor creates a new AlarmSyncProcessor.
func NewAlarmSyncProcessor(
	engine *AlarmEngine,
	store AlarmStore,
	syncService *AlarmSyncService,
	eventBus event.EventBus,
	logger *zap.Logger,
) *AlarmSyncProcessor {
	return &AlarmSyncProcessor{
		engine:      engine,
		store:       store,
		syncService: syncService,
		eventBus:    eventBus,
		logger:      logger,
	}
}

// WithDeviceReader enables device field backfill for alarms created by sync.
func (p *AlarmSyncProcessor) WithDeviceReader(reader deviceReader) *AlarmSyncProcessor {
	p.deviceReader = reader
	return p
}

// WithAlarmDefRegistry enables severity override from alarm definitions during sync.
func (p *AlarmSyncProcessor) WithAlarmDefRegistry(alarmDefRegistry *definition.Registry) *AlarmSyncProcessor {
	p.alarmDefRegistry = alarmDefRegistry
	if alarmDefRegistry != nil {
		p.defMetrics = alarmDefRegistry.Metrics()
	}
	return p
}

// WithProductResolver enables unknown-alarm fallback decisions during sync.
func (p *AlarmSyncProcessor) WithProductResolver(productResolver definition.ProductResolver) *AlarmSyncProcessor {
	p.productResolver = productResolver
	return p
}

// Start subscribes to GPV response events and processes alarm sync results.
func (p *AlarmSyncProcessor) Start(ctx context.Context) error {
	if p.eventBus == nil {
		p.logger.Warn("no event bus, alarm sync processor not starting")
		return nil
	}

	_, err := p.eventBus.QueueSubscribe(
		event.SubjectCommandGetParamsResponse,
		"alarm-sync-gpv",
		func(ctx context.Context, evt event.Event) error {
			return p.handleGPVResponse(ctx, evt)
		},
	)
	if err != nil {
		return fmt.Errorf("subscribe GPV response for alarm sync: %w", err)
	}

	p.logger.Info("alarm sync processor subscribed to GPV response events")
	return nil
}

// gpvResponsePayload mirrors the ACS-published GPV response structure.
type gpvResponsePayload struct {
	DeviceSN        string                       `json:"device_sn"`
	TaskID          string                       `json:"task_id"`
	Method          string                       `json:"method"`
	Path            string                       `json:"path"`
	ParameterValues []tr069.ParameterValueStruct `json:"parameter_values"`
}

// handleGPVResponse processes a GPV response event.
// It only handles responses that contain FaultMgmt.CurrentAlarm parameters.
func (p *AlarmSyncProcessor) handleGPVResponse(ctx context.Context, evt event.Event) error {
	if !alarmSyncOwnsGPVResponse(evt) {
		return nil
	}
	var payload gpvResponsePayload
	if err := evt.DecodePayload(&payload); err != nil {
		p.logger.Error("decode GPV response event", zap.Error(err))
		return nil
	}

	// Only process GPV responses for alarm parameters
	if !isAlarmGPVResponse(payload.Path, payload.ParameterValues) {
		return nil
	}

	deviceSN := payload.DeviceSN
	p.logger.Info("processing alarm GPV response",
		zap.String("device_sn", deviceSN),
		zap.Int("param_count", len(payload.ParameterValues)))

	result := p.processSync(ctx, deviceSN, payload.ParameterValues)

	// Release the sync lock
	if p.syncService != nil {
		p.syncService.ReleaseLock(ctx, deviceSN, payload.TaskID)
	}

	// Publish sync completed event
	if p.eventBus != nil {
		syncEvt, err := event.NewEvent(event.SubjectAlarmSyncCompleted, result)
		if err == nil {
			if pubErr := p.eventBus.Publish(ctx, event.SubjectAlarmSyncCompleted, syncEvt); pubErr != nil {
				p.logger.Warn("publish alarm.sync.completed", zap.Error(pubErr))
			}
		}
	}

	p.logger.Info("alarm sync completed",
		zap.String("device_sn", deviceSN),
		zap.Int("added", result.Added),
		zap.Int("updated", result.Updated),
		zap.Int("cleared", result.Cleared),
		zap.Int("failed_add", result.FailedAdd),
		zap.Int("failed_update", result.FailedUpdate),
		zap.Int("failed_clear", result.FailedClear))

	return nil
}

func alarmSyncOwnsGPVResponse(evt event.Event) bool {
	if evt.Metadata == nil {
		return true
	}
	owner := evt.Metadata[event.MetadataGPVOwner]
	if owner == "" {
		return true
	}
	return owner == event.GPVOwnerAlarmSync
}

// processSync executes the full sync pipeline: parse → diff → apply.
func (p *AlarmSyncProcessor) processSync(ctx context.Context, deviceSN string, params []tr069.ParameterValueStruct) *SyncResult {
	// issue #20：告警同步 diff-apply 整链补 span。一次 GPV 响应可能触发多条 add/update/
	// clear，整体卡顿（store 慢 / 大批 diff）此前不可观测；逐条 engine.Process/ClearBySync
	// 各自带子 span，在此根 span 下形成调用树，便于定位"告警同步迟迟不收敛"。
	ctx, span := tracing.StartSpan(ctx, tracing.AlarmTracerName, "Alarm processSync",
		attribute.String("alarm.device_sn", deviceSN),
		attribute.Int("alarm.param_count", len(params)),
	)
	defer span.End()

	result := &SyncResult{
		DeviceSN: deviceSN,
		SyncedAt: time.Now(),
	}

	// 1. Parse TR-069 CurrentAlarm parameters
	tr069Alarms, err := ParseCurrentAlarmParams(params)
	if err != nil {
		p.logger.Error("parse current alarm params", zap.Error(err), zap.String("device_sn", deviceSN))
		result.FailedAdd = len(params) // approximate
		return result
	}

	// 2. Get local active alarms
	localAlarms, err := p.store.GetActiveByDeviceSN(ctx, deviceSN)
	if err != nil {
		p.logger.Error("get local active alarms", zap.Error(err), zap.String("device_sn", deviceSN))
		result.FailedAdd = len(tr069Alarms)
		return result
	}
	// 3. Resolve device-derived fields for any newly added alarms.
	deviceID, carrier, technology, productClass := p.resolveDeviceFields(ctx, deviceSN, localAlarms)
	remoteAlarms := make([]*model.Alarm, 0, len(tr069Alarms))
	for i := range tr069Alarms {
		alarm := tr069Alarms[i].ToModel(deviceID, deviceSN, carrier)
		if technology != "" {
			alarm.Technology = &technology
		}
		drop, err := applyUnknownAlarmFallback(ctx, p.alarmDefRegistry, p.productResolver, p.defMetrics, p.logger, alarm, productClass)
		if err != nil {
			p.logger.Warn("apply synced alarm fallback failed (proceed without fallback)",
				zap.Error(err),
				zap.String("device_sn", deviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
		}
		if drop {
			continue
		}
		remoteAlarms = append(remoteAlarms, alarm)
	}

	// 4. Compute diff only against alarms owned by the device alarm table.
	// OMC-generated alarms (for example geofence/offline) have independent
	// lifecycle monitors and must not be cleared merely because they are absent
	// from Device.FaultMgmt.CurrentAlarm.
	diff := ComputeDiff(remoteAlarms, deviceReconciledAlarms(localAlarms))

	// 5. Apply diff: ToAdd
	for _, alarm := range diff.ToAdd {
		if err := p.engine.Process(ctx, alarm); err != nil {
			p.logger.Error("sync add alarm", zap.Error(err),
				zap.String("device_sn", deviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
			result.FailedAdd++
		} else {
			result.Added++
		}
	}

	// 6. Apply diff: ToUpdate — merge remote fields into existing local alarm
	for _, update := range diff.ToUpdate {
		localAlarm := update.Local
		remoteAlarm := update.Remote
		if localAlarm == nil || remoteAlarm == nil {
			result.FailedUpdate++
			continue
		}
		// Merge updatable fields from remote into local (preserves ID, status, ack state, etc.)
		localAlarm.Severity = remoteAlarm.Severity
		localAlarm.Description = remoteAlarm.Description
		localAlarm.AlarmType = remoteAlarm.AlarmType
		localAlarm.AlarmSource = remoteAlarm.AlarmSource
		localAlarm.EventType = remoteAlarm.EventType
		localAlarm.ProbableCause = remoteAlarm.ProbableCause
		if localAlarm.RaisedAt.IsZero() && !remoteAlarm.RaisedAt.IsZero() {
			localAlarm.RaisedAt = remoteAlarm.RaisedAt
		}
		localAlarm.LastUpdatedAt = time.Now()
		// Merge additional info
		for k, v := range remoteAlarm.AdditionalInfo {
			localAlarm.AdditionalInfo[k] = v
		}
		if err := p.engine.UpdateFromSync(ctx, localAlarm); err != nil {
			p.logger.Error("sync update alarm", zap.Error(err),
				zap.String("device_sn", deviceSN),
				zap.String("alarm_identifier", remoteAlarm.AlarmIdentifier))
			result.FailedUpdate++
		} else {
			result.Updated++
		}
	}

	// 7. Apply diff: ToClear
	for _, alarm := range diff.ToClear {
		if alarm == nil {
			result.Cleared++
			continue
		}
		if err := p.engine.ClearBySync(ctx, alarm); err != nil {
			p.logger.Error("sync clear alarm", zap.Error(err),
				zap.String("device_sn", deviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
			result.FailedClear++
		} else {
			result.Cleared++
		}
	}

	// 8. Apply diff: stale duplicate rows for keys that still exist remotely.
	for _, duplicate := range diff.ToClearDuplicates {
		alarm := duplicate.Duplicate
		if alarm == nil {
			result.FailedClear++
			continue
		}
		clearedBy := "system:alarm_sync_duplicate"
		clearNote := "duplicate active alarm reconciled by full sync"
		alarm.ClearedBy = &clearedBy
		alarm.ClearNote = &clearNote
		if err := p.engine.ClearBySync(ctx, alarm); err != nil {
			p.logger.Error("sync clear duplicate alarm", zap.Error(err),
				zap.String("device_sn", deviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier),
				zap.String("alarm_id", alarm.ID.String()))
			result.FailedClear++
			continue
		}
		if duplicate.Keeper != nil && p.engine.redisStore != nil {
			if err := p.engine.redisStore.Set(
				ctx,
				duplicate.Keeper.DeviceSN,
				activeAlarmMatchKey(duplicate.Keeper),
				duplicate.Keeper.ID.String(),
			); err != nil {
				p.logger.Warn("restore keeper redis key after duplicate clear",
					zap.Error(err),
					zap.String("device_sn", deviceSN),
					zap.String("alarm_identifier", duplicate.Keeper.AlarmIdentifier))
			}
		}
		if p.engine.metrics != nil {
			p.engine.metrics.ReconciliationTotal.WithLabelValues("sync_duplicate_cleared").Inc()
		}
		result.Cleared++
	}

	return result
}

func deviceReconciledAlarms(alarms []*model.Alarm) []*model.Alarm {
	result := make([]*model.Alarm, 0, len(alarms))
	for _, alarm := range alarms {
		if alarm == nil {
			continue
		}
		if alarm.AlarmSource != nil &&
			strings.EqualFold(strings.TrimSpace(*alarm.AlarmSource), "omc") {
			continue
		}
		result = append(result, alarm)
	}
	return result
}

func (p *AlarmSyncProcessor) resolveDeviceFields(ctx context.Context, deviceSN string, localAlarms []*model.Alarm) (uuid.UUID, model.CarrierCode, string, string) {
	deviceID := uuid.Nil
	var carrier model.CarrierCode
	technology := ""
	productClass := ""

	for _, alarm := range localAlarms {
		if alarm == nil {
			continue
		}
		if deviceID == uuid.Nil && alarm.DeviceID != uuid.Nil {
			deviceID = alarm.DeviceID
		}
		if carrier == "" && alarm.Carrier != "" {
			carrier = alarm.Carrier
		}
		if technology == "" && alarm.Technology != nil && *alarm.Technology != "" {
			technology = *alarm.Technology
		}
	}

	needProductClass := p.productResolver != nil && productClass == ""
	if p.deviceReader == nil || (deviceID != uuid.Nil && carrier != "" && technology != "" && !needProductClass) {
		return deviceID, carrier, technology, productClass
	}

	device, err := p.deviceReader.GetBySerialNumber(ctx, deviceSN)
	if err != nil || device == nil {
		return deviceID, carrier, technology, productClass
	}
	if deviceID == uuid.Nil {
		deviceID = device.ID
	}
	if carrier == "" {
		carrier = device.Carrier
	}
	if technology == "" && device.Technology != "" {
		technology = string(device.Technology)
	}
	productClass = device.ProductClass

	return deviceID, carrier, technology, productClass
}

// isAlarmGPVResponse checks if the GPV response contains alarm parameters.
func isAlarmGPVResponse(path string, params []tr069.ParameterValueStruct) bool {
	// Check path hint
	if strings.Contains(path, "FaultMgmt.CurrentAlarm") {
		return true
	}
	// Check parameter names
	for _, p := range params {
		if strings.Contains(p.Name, "FaultMgmt.CurrentAlarm") {
			return true
		}
	}
	return false
}
