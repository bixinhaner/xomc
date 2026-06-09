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
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// AlarmSyncProcessor subscribes to GPV response events, parses TR-069 alarm parameters,
// computes a three-way diff against local active alarms, and applies changes.
type AlarmSyncProcessor struct {
	engine       *AlarmEngine
	store        AlarmStore
	syncService  *AlarmSyncService
	eventBus     event.EventBus
	logger       *zap.Logger
	alarmDefRegistry *definition.Registry
	deviceReader deviceReader
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
	Method          string                       `json:"method"`
	Path            string                       `json:"path"`
	ParameterValues []tr069.ParameterValueStruct `json:"parameter_values"`
}

// handleGPVResponse processes a GPV response event.
// It only handles responses that contain FaultMgmt.CurrentAlarm parameters.
func (p *AlarmSyncProcessor) handleGPVResponse(ctx context.Context, evt event.Event) error {
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
		p.syncService.ReleaseLock(ctx, deviceSN)
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

// processSync executes the full sync pipeline: parse → diff → apply.
func (p *AlarmSyncProcessor) processSync(ctx context.Context, deviceSN string, params []tr069.ParameterValueStruct) *SyncResult {
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
	localAlarmByKey := indexActiveAlarms(localAlarms)

	// 3. Resolve device-derived fields for any newly added alarms.
	deviceID, carrier, technology := p.resolveDeviceFields(ctx, deviceSN, localAlarms)
	remoteAlarms := make([]*model.Alarm, 0, len(tr069Alarms))
	for i := range tr069Alarms {
		alarm := tr069Alarms[i].ToModel(deviceID, deviceSN, carrier)
		if err := applyAlarmDefinitionSeverity(ctx, p.alarmDefRegistry, alarm); err != nil {
			p.logger.Warn("resolve synced alarm definition severity failed (proceed with source severity)",
				zap.Error(err),
				zap.String("device_sn", deviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
		}
		if technology != "" {
			alarm.Technology = &technology
		}
		remoteAlarms = append(remoteAlarms, alarm)
	}

	// 4. Compute diff
	diff := ComputeDiff(remoteAlarms, localAlarms)

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
	for matchKey, remoteAlarm := range diff.ToUpdate {
		localAlarm := localAlarmByKey[matchKey]
		if localAlarm == nil {
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
		localAlarm.LastUpdatedAt = resolveAlarmBusinessTime(remoteAlarm, time.Now())
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
	for _, matchKey := range diff.ToClear {
		alarm := localAlarmByKey[matchKey]
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

	return result
}

func (p *AlarmSyncProcessor) resolveDeviceFields(ctx context.Context, deviceSN string, localAlarms []*model.Alarm) (uuid.UUID, model.CarrierCode, string) {
	deviceID := uuid.Nil
	var carrier model.CarrierCode
	technology := ""

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

	if p.deviceReader == nil || (deviceID != uuid.Nil && carrier != "" && technology != "") {
		return deviceID, carrier, technology
	}

	device, err := p.deviceReader.GetBySerialNumber(ctx, deviceSN)
	if err != nil || device == nil {
		return deviceID, carrier, technology
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

	return deviceID, carrier, technology
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
