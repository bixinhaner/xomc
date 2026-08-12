package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

const (
	GeofenceControlQueue                = "geofence-control"
	geofenceTerminalVerificationDelay   = 5 * time.Second
	geofenceTerminalVerificationTimeout = 2 * time.Minute
	geofenceTerminalVerificationLimit   = 24
)

// GeofenceControlMonitor executes a confirmed state edge through the existing
// task/CWMP path and restores only values changed by a verified owned action.
type GeofenceControlMonitor struct {
	devices    GeofenceControlDeviceReader
	parameters GeofenceControlParameterReader
	mappings   GeofenceControlMappingReader
	carriers   *carrier.CarrierRegistry
	tasks      task.Enqueuer
	history    GeofenceControlTaskReader
	actions    ControlActionRepository
	logger     *zap.Logger
}

type GeofenceControlDeviceReader interface {
	GetBySerialNumber(context.Context, string) (*model.Device, error)
}

type GeofenceControlTaskReader interface {
	GetByCommandKey(context.Context, string) (*task.Task, error)
	AcquireCommandKeyLock(context.Context, string) (func(), error)
}

type GeofenceControlParameterReader interface {
	GetByDevice(context.Context, uuid.UUID) ([]model.DeviceParameter, error)
}

type GeofenceControlMappingReader interface {
	GetByProductClass(context.Context, string, string) ([]carrier.GeofenceControlMapping, error)
}

func NewGeofenceControlMonitor(
	devices GeofenceControlDeviceReader,
	carriers *carrier.CarrierRegistry,
	tasks task.Enqueuer,
	logger *zap.Logger,
) *GeofenceControlMonitor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GeofenceControlMonitor{
		devices:  devices,
		carriers: carriers,
		tasks:    tasks,
		logger:   logger,
	}
}

func (m *GeofenceControlMonitor) SetParameterReader(reader GeofenceControlParameterReader) {
	m.parameters = reader
}

func (m *GeofenceControlMonitor) SetMappingReader(reader GeofenceControlMappingReader) {
	m.mappings = reader
}

func (m *GeofenceControlMonitor) SetTaskHistoryReader(
	history GeofenceControlTaskReader,
) {
	m.history = history
}

func (m *GeofenceControlMonitor) SetActionRepository(actions ControlActionRepository) {
	m.actions = actions
}

// RunVerificationLoop durably resumes OpState polling after worker restarts.
// Each GPV is still an ordinary asynchronous device task; the action row owns
// the next attempt and deadline.
func (m *GeofenceControlMonitor) RunVerificationLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = geofenceTerminalVerificationDelay
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := m.reconcileDueVerifications(ctx); err != nil && ctx.Err() == nil {
			m.logger.Error("reconcile geofence terminal verification", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *GeofenceControlMonitor) Subscribe(bus event.EventBus) error {
	if m == nil || m.devices == nil || m.carriers == nil || m.tasks == nil ||
		m.parameters == nil || m.history == nil || m.actions == nil {
		return fmt.Errorf("geofence control monitor dependencies are required")
	}
	if bus == nil {
		return fmt.Errorf("geofence control monitor event bus is required")
	}
	for _, subject := range []string{
		event.SubjectGeofenceDeviceExited,
		event.SubjectGeofenceDeviceEscalated,
	} {
		if _, err := bus.QueueSubscribe(
			subject,
			geofenceControlQueue(subject),
			m.handleExited,
		); err != nil {
			return fmt.Errorf("subscribe geofence control outside action %s: %w", subject, err)
		}
	}
	for _, subject := range []string{
		event.SubjectTaskCompleted,
		event.SubjectTaskFailed,
	} {
		if _, err := bus.QueueSubscribe(
			subject,
			geofenceControlQueue(subject),
			m.handleTaskTerminal,
		); err != nil {
			return fmt.Errorf("subscribe geofence control task terminal: %w", err)
		}
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectGeofenceDeviceEntered,
		geofenceControlQueue(event.SubjectGeofenceDeviceEntered),
		m.handleEntered,
	); err != nil {
		return fmt.Errorf("subscribe geofence control enter: %w", err)
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectGeofenceLifecycleDeactivationRequired,
		geofenceControlQueue(event.SubjectGeofenceLifecycleDeactivationRequired),
		m.handleLifecycleDeactivation,
	); err != nil {
		return fmt.Errorf("subscribe geofence lifecycle deactivation: %w", err)
	}
	return nil
}

func geofenceControlQueue(subject string) string {
	if subject == event.SubjectGeofenceDeviceExited {
		// Preserve the already deployed durable and its acknowledgement position.
		return GeofenceControlQueue
	}
	suffix := strings.NewReplacer(".", "-", "_", "-").Replace(subject)
	return GeofenceControlQueue + "-" + suffix
}

func (m *GeofenceControlMonitor) handleExited(
	ctx context.Context,
	evt event.Event,
) error {
	var payload event.GeofenceDeviceStatePayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode geofence control event: %w", err)
	}
	if payload.EffectiveState != string(EffectiveStateOutside) ||
		payload.RequiredActionLevel != string(ActionLevelDeactivate) {
		return nil
	}
	deviceRecord, err := m.devices.GetBySerialNumber(ctx, payload.SerialNumber)
	if err != nil {
		return fmt.Errorf("lookup geofence control device: %w", err)
	}
	if deviceRecord == nil {
		return fmt.Errorf("geofence control device %q not found", payload.SerialNumber)
	}
	actionID := fmt.Sprintf(
		"geofence:%s:%d:deactivate",
		payload.DeviceID,
		payload.EffectiveStateVersion,
	)
	now := time.Now().UTC()
	if err := m.queueDeviceControl(
		ctx,
		deviceRecord,
		false,
		nil,
		&ControlAction{
			ID: uuid.New(), ActionKey: actionID, DeviceID: deviceRecord.ID,
			DeviceSN: deviceRecord.SerialNumber, BindingID: payload.TriggerBindingID,
			EffectiveStateVersion: payload.EffectiveStateVersion,
			ActionType:            ControlActionDeactivate, Status: ControlActionPending,
			CreatedAt: now, UpdatedAt: now,
		},
		fmt.Sprintf(
			"geofence deactivation for effective state version %d",
			payload.EffectiveStateVersion,
		),
	); err != nil {
		return err
	}
	m.logger.Info("geofence deactivation queued",
		zap.String("device_id", payload.DeviceID.String()),
		zap.String("serial_number", payload.SerialNumber),
		zap.Int64("effective_state_version", payload.EffectiveStateVersion),
		zap.String("action_id", actionID),
	)
	return nil
}

func (m *GeofenceControlMonitor) handleLifecycleDeactivation(
	ctx context.Context,
	evt event.Event,
) error {
	var payload event.GeofenceLifecycleDeactivationPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode geofence lifecycle deactivation: %w", err)
	}
	if payload.TargetStatus != string(DefinitionStatusDisabled) &&
		payload.TargetStatus != string(DefinitionStatusArchived) {
		return nil
	}
	if strings.TrimSpace(evt.ID) == "" {
		return fmt.Errorf("geofence lifecycle deactivation event ID is required")
	}
	deviceRecord, err := m.devices.GetBySerialNumber(ctx, payload.SerialNumber)
	if err != nil {
		return fmt.Errorf("lookup lifecycle deactivation device: %w", err)
	}
	if deviceRecord == nil {
		return fmt.Errorf("lifecycle deactivation device %q not found", payload.SerialNumber)
	}
	actionID := fmt.Sprintf(
		"geofence:%s:lifecycle:%s:deactivate",
		payload.DeviceID,
		evt.ID,
	)
	now := time.Now().UTC()
	if err := m.queueDeviceControl(
		ctx,
		deviceRecord,
		false,
		nil,
		&ControlAction{
			ID: uuid.New(), ActionKey: actionID, DeviceID: deviceRecord.ID,
			DeviceSN: deviceRecord.SerialNumber, GeofenceID: &payload.GeofenceID,
			ActionType: ControlActionDeactivate, Status: ControlActionPending,
			CreatedAt: now, UpdatedAt: now,
		},
		fmt.Sprintf(
			"geofence lifecycle %s deactivation for fence %s",
			payload.TargetStatus,
			payload.GeofenceID,
		),
	); err != nil {
		return err
	}
	m.logger.Info("geofence lifecycle deactivation queued",
		zap.String("serial_number", payload.SerialNumber),
		zap.String("geofence_id", payload.GeofenceID.String()),
		zap.String("target_status", payload.TargetStatus),
		zap.String("action_id", actionID))
	return nil
}

func (m *GeofenceControlMonitor) handleTaskTerminal(
	ctx context.Context,
	evt event.Event,
) error {
	var completed task.Task
	if err := evt.DecodePayload(&completed); err != nil {
		return fmt.Errorf("decode geofence control task result: %w", err)
	}
	if !strings.HasPrefix(completed.CommandKey, "geofence:") {
		return nil
	}
	if completed.Source != task.TaskSourceGeofence || completed.SourceID == "" {
		return nil
	}
	actionID, err := uuid.Parse(completed.SourceID)
	if err != nil {
		return fmt.Errorf("parse geofence control action ID: %w", err)
	}
	if m.actions == nil {
		return fmt.Errorf("geofence control action repository is not configured")
	}
	action, err := m.actions.GetByID(ctx, actionID)
	if err != nil {
		return fmt.Errorf("load geofence control action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("geofence control action %s not found", actionID)
	}
	if action.Status.IsTerminal() {
		m.logger.Debug("ignore duplicate geofence task event after terminal action",
			zap.String("action_id", action.ActionKey),
			zap.String("command_key", completed.CommandKey),
			zap.String("status", string(action.Status)))
		return nil
	}
	if !taskBelongsToCurrentControlStep(action, &completed) {
		m.logger.Warn("ignore stale or out-of-order geofence task event",
			zap.String("action_id", action.ActionKey),
			zap.String("command_key", completed.CommandKey),
			zap.String("method", completed.Method),
			zap.Int("verification_attempt", action.VerificationAttempt))
		return nil
	}
	if evt.Subject == event.SubjectTaskFailed {
		message := strings.TrimSpace(completed.ErrorMessage)
		if message == "" {
			message = "device task failed without an error message"
		}
		if err := m.actions.CompleteVerification(
			ctx, action.ID, nil, ControlActionFailed, message, time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("fail geofence control action: %w", err)
		}
		m.logger.Warn("geofence control failed",
			zap.String("task_id", completed.ID),
			zap.String("action_id", action.ActionKey),
			zap.String("error_message", message),
		)
		return nil
	}
	switch completed.Method {
	case "SetParameterValues":
		return m.queueControlVerification(ctx, action)
	case "GetParameterValues":
		if action.ContractVersion >= GeofenceControlContractVersion && len(action.TerminalState) == 0 {
			message := "contract v2 action is missing required OpState terminal evidence"
			if err := m.actions.CompleteVerification(
				ctx, action.ID, nil, ControlActionPartialFailed, message, time.Now().UTC(),
			); err != nil {
				return fmt.Errorf("reject geofence verification without OpState: %w", err)
			}
			return nil
		}
		verified, status, message, terminalPending := verifyControlAndTerminalTaskResult(
			completed.Result, action.RequestedState, action.TerminalState,
		)
		if terminalPending {
			now := time.Now().UTC()
			if action.VerificationDeadline == nil || !now.Before(*action.VerificationDeadline) ||
				action.VerificationAttempt >= geofenceTerminalVerificationLimit {
				message = "OpState terminal verification timed out: " + message
				if err := m.actions.CompleteVerification(
					ctx, action.ID, verified, ControlActionPartialFailed, message, now,
				); err != nil {
					return fmt.Errorf("timeout geofence terminal verification: %w", err)
				}
				return nil
			}
			nextAttempt := action.VerificationAttempt + 1
			if err := m.actions.ScheduleVerification(
				ctx, action.ID, verified, message, nextAttempt,
				now.Add(geofenceTerminalVerificationDelay),
			); err != nil {
				return fmt.Errorf("schedule geofence terminal verification: %w", err)
			}
			return nil
		}
		if err := m.actions.CompleteVerification(
			ctx, action.ID, verified, status, message, time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("complete geofence control verification: %w", err)
		}
		m.logger.Info("geofence control verification completed",
			zap.String("task_id", completed.ID),
			zap.String("action_id", action.ActionKey),
			zap.String("status", string(status)))
	}
	return nil
}

func (m *GeofenceControlMonitor) queueControlVerification(
	ctx context.Context,
	action *ControlAction,
) error {
	if len(action.RequestedState) == 0 && len(action.TerminalState) == 0 {
		return nil
	}
	if action.VerificationDeadline == nil {
		now := time.Now().UTC()
		deadline := now.Add(geofenceTerminalVerificationTimeout)
		if err := m.actions.BeginVerification(ctx, action.ID, deadline, now); err != nil {
			return fmt.Errorf("begin geofence terminal verification window: %w", err)
		}
		action.Status = ControlActionVerifying
		action.VerificationDeadline = &deadline
		action.NextVerificationAt = &now
	}
	commandKey := fmt.Sprintf("%s:verify:%d", action.ActionKey, action.VerificationAttempt)
	release, err := m.history.AcquireCommandKeyLock(ctx, commandKey)
	if err != nil {
		return fmt.Errorf("lock geofence verification command key: %w", err)
	}
	defer release()
	existing, err := m.history.GetByCommandKey(ctx, commandKey)
	if err != nil {
		return fmt.Errorf("check geofence verification duplicate: %w", err)
	}
	if existing != nil {
		return m.actions.UpdateStatus(ctx, action.ID, ControlActionVerifying)
	}
	names := make([]string, 0, len(action.RequestedState)+len(action.TerminalState))
	for _, state := range action.RequestedState {
		names = append(names, state.Path)
	}
	for _, state := range action.TerminalState {
		names = append(names, state.Path)
	}
	params, err := json.Marshal(map[string]any{"names": names})
	if err != nil {
		return fmt.Errorf("marshal geofence verification parameters: %w", err)
	}
	_, err = m.tasks.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN: action.DeviceSN, Method: "GetParameterValues", Params: params,
		Priority: 5, CommandKey: commandKey, Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(), MaxRetries: intPtr(3), ExpiresIn: 300,
		Description: "geofence control readback verification",
	})
	if err != nil {
		return fmt.Errorf("queue geofence control verification: %w", err)
	}
	if err := m.actions.UpdateStatus(ctx, action.ID, ControlActionVerifying); err != nil {
		return fmt.Errorf("mark geofence action verifying: %w", err)
	}
	return nil
}

func taskBelongsToCurrentControlStep(action *ControlAction, completed *task.Task) bool {
	if action == nil || completed == nil {
		return false
	}
	switch completed.Method {
	case "SetParameterValues":
		return (action.Status == ControlActionPending || action.Status == ControlActionExecuting) &&
			completed.CommandKey == action.ActionKey
	case "GetParameterValues":
		expected := fmt.Sprintf("%s:verify:%d", action.ActionKey, action.VerificationAttempt)
		return action.Status == ControlActionVerifying && completed.CommandKey == expected
	default:
		return false
	}
}

func verifyControlTaskResult(
	raw []byte,
	requested []ControlParameterState,
) ([]ControlParameterState, ControlActionStatus, string) {
	verified, status, message, _ := verifyControlAndTerminalTaskResult(raw, requested, nil)
	return verified, status, message
}

func verifyControlAndTerminalTaskResult(
	raw []byte,
	requested []ControlParameterState,
	terminals []ControlParameterState,
) ([]ControlParameterState, ControlActionStatus, string, bool) {
	var result struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"standard_parameter_values"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, ControlActionPartialFailed, "readback result is not valid JSON", false
	}
	expectedRoles := make(map[string]carrier.GeofenceParameterRole, len(requested)+len(terminals))
	for _, expected := range requested {
		expectedRoles[expected.Path] = expected.Role
	}
	for _, expected := range terminals {
		expectedRoles[expected.Path] = carrier.GeofenceRoleOpState
	}
	actual := make(map[string]string, len(result.Values))
	verified := make([]ControlParameterState, 0, len(result.Values))
	for _, value := range result.Values {
		normalized, err := normalizeControlBoolean(value.Value)
		if err != nil {
			continue
		}
		actual[value.Name] = normalized
		verified = append(verified, ControlParameterState{
			Path: value.Name, Value: normalized, Role: expectedRoles[value.Name],
		})
	}
	var mismatches []string
	for _, expected := range requested {
		value, ok := actual[expected.Path]
		if !ok {
			mismatches = append(mismatches, expected.Path+" missing")
			continue
		}
		if value != expected.Value {
			mismatches = append(mismatches, fmt.Sprintf(
				"%s expected %s got %s", expected.Path, expected.Value, value,
			))
		}
	}
	if len(mismatches) > 0 {
		return verified, ControlActionPartialFailed, strings.Join(mismatches, "; "), false
	}
	var terminalMismatches []string
	for _, expected := range terminals {
		value, ok := actual[expected.Path]
		if !ok {
			terminalMismatches = append(terminalMismatches, expected.Path+" missing")
			continue
		}
		if value != expected.Value {
			terminalMismatches = append(terminalMismatches, fmt.Sprintf(
				"%s expected %s got %s", expected.Path, expected.Value, value,
			))
		}
	}
	if len(terminalMismatches) > 0 {
		return verified, ControlActionVerifying, strings.Join(terminalMismatches, "; "), true
	}
	return verified, ControlActionVerified, "", false
}

func (m *GeofenceControlMonitor) handleEntered(
	ctx context.Context,
	evt event.Event,
) error {
	var payload event.GeofenceDeviceStatePayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode geofence control enter event: %w", err)
	}
	if payload.EffectiveState != string(EffectiveStateInside) {
		return nil
	}
	if m.actions == nil {
		m.logger.Warn("skip geofence activation: action repository is not configured",
			zap.String("serial_number", payload.SerialNumber))
		return nil
	}
	deactivation, err := m.actions.FindRecoverableDeactivation(ctx, payload.DeviceID)
	if err != nil {
		return fmt.Errorf("find recoverable geofence deactivation: %w", err)
	}
	if deactivation == nil {
		m.logger.Info("skip geofence activation: no verified owned deactivation",
			zap.String("serial_number", payload.SerialNumber))
		return nil
	}
	activationKey := strings.TrimSuffix(deactivation.ActionKey, ":deactivate") + ":activate"
	deviceRecord, err := m.devices.GetBySerialNumber(ctx, payload.SerialNumber)
	if err != nil {
		return fmt.Errorf("lookup geofence activation device: %w", err)
	}
	if deviceRecord == nil {
		return fmt.Errorf("geofence activation device %q not found", payload.SerialNumber)
	}
	now := time.Now().UTC()
	if err := m.queueDeviceControl(
		ctx,
		deviceRecord,
		true,
		restoreTargets(deactivation.BeforeState, deactivation.VerifiedState),
		&ControlAction{
			ID: uuid.New(), ActionKey: activationKey, ParentActionID: &deactivation.ID,
			DeviceID: deviceRecord.ID, DeviceSN: deviceRecord.SerialNumber,
			EffectiveStateVersion: payload.EffectiveStateVersion,
			ActionType:            ControlActionActivate, Status: ControlActionPending,
			ContractVersion: GeofenceControlContractVersion,
			TerminalState:   originalTerminalTargets(deactivation.BeforeState),
			CreatedAt:       now, UpdatedAt: now,
		},
		fmt.Sprintf(
			"geofence activation after completed deactivation %s",
			deactivation.ActionKey,
		),
	); err != nil {
		return err
	}
	m.logger.Info("geofence activation queued",
		zap.String("serial_number", payload.SerialNumber),
		zap.String("action_id", activationKey))
	return nil
}

func (m *GeofenceControlMonitor) queueDeviceControl(
	ctx context.Context,
	deviceRecord *model.Device,
	enabled bool,
	targets []carrier.GeofenceControlParameter,
	action *ControlAction,
	description string,
) error {
	if m.history == nil {
		return fmt.Errorf("geofence control task history is not configured")
	}
	if m.actions == nil {
		return fmt.Errorf("geofence control action repository is not configured")
	}
	if action == nil || strings.TrimSpace(action.ActionKey) == "" {
		return fmt.Errorf("geofence control action is required")
	}
	release, err := m.history.AcquireCommandKeyLock(ctx, action.ActionKey)
	if err != nil {
		return fmt.Errorf("lock geofence control command key: %w", err)
	}
	defer release()

	existing, err := m.history.GetByCommandKey(ctx, action.ActionKey)
	if err != nil {
		return fmt.Errorf("check geofence control duplicate: %w", err)
	}
	if existing != nil {
		stored, loadErr := m.actions.GetByActionKey(ctx, action.ActionKey)
		if loadErr != nil {
			return fmt.Errorf("load duplicate geofence control action: %w", loadErr)
		}
		if stored != nil && stored.Status == ControlActionPending {
			if updateErr := m.actions.UpdateStatus(
				ctx, stored.ID, ControlActionExecuting,
			); updateErr != nil {
				return fmt.Errorf("mark queued geofence action executing: %w", updateErr)
			}
		}
		m.logger.Info("skip duplicate geofence control",
			zap.String("serial_number", deviceRecord.SerialNumber),
			zap.String("action_id", action.ActionKey),
		)
		return nil
	}
	stored, err := m.actions.GetByActionKey(ctx, action.ActionKey)
	if err != nil {
		return fmt.Errorf("load geofence control action: %w", err)
	}
	var plan geofenceControlPlan
	var params []byte
	if stored != nil {
		if stored.Status != ControlActionPending {
			return nil
		}
		plan = geofenceControlPlan{
			Before: stored.BeforeState, Requested: stored.RequestedState,
		}
		params, err = marshalGeofenceControlPayload(plan.Requested)
		if err != nil {
			return fmt.Errorf("marshal pending geofence control parameters: %w", err)
		}
	} else {
		carrierAdapter, resolveErr := m.carriers.Get(deviceRecord.Carrier)
		if resolveErr != nil {
			return fmt.Errorf("resolve geofence control carrier: %w", resolveErr)
		}
		plan, params, err = m.geofenceControlPlan(
			ctx, carrierAdapter, deviceRecord, enabled, targets, action.TerminalState,
			action.ActionType == ControlActionActivate,
		)
		if err != nil {
			return err
		}
		action.BeforeState = plan.Before
		action.RequestedState = plan.Requested
		action.TerminalState = plan.Terminals
		if action.ContractVersion == 0 {
			action.ContractVersion = GeofenceControlContractVersion
		}
		stored, _, err = m.actions.Create(ctx, action)
		if err != nil {
			return fmt.Errorf("create geofence control action: %w", err)
		}
	}
	if len(plan.Requested) == 0 {
		return m.queueControlVerification(ctx, stored)
	}
	_, err = m.tasks.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    deviceRecord.SerialNumber,
		Method:      "SetParameterValues",
		Params:      params,
		CommandKey:  action.ActionKey,
		Source:      task.TaskSourceGeofence,
		SourceID:    stored.ID.String(),
		MaxRetries:  intPtr(3),
		ExpiresIn:   300,
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("queue geofence control: %w", err)
	}
	if err := m.actions.UpdateStatus(ctx, stored.ID, ControlActionExecuting); err != nil {
		return fmt.Errorf("mark geofence action executing: %w", err)
	}
	return nil
}

func intPtr(value int) *int {
	return &value
}

func (m *GeofenceControlMonitor) geofenceControlPlan(
	ctx context.Context,
	adapter carrier.Carrier,
	device *model.Device,
	enabled bool,
	targets []carrier.GeofenceControlParameter,
	terminalTargets []ControlParameterState,
	forceRequested bool,
) (geofenceControlPlan, []byte, error) {
	_, ok := adapter.(carrier.GeofenceControlParameterInstanceResolver)
	if !ok {
		return geofenceControlPlan{}, nil, fmt.Errorf(
			"carrier %s does not expose geofence control parameters",
			device.Carrier,
		)
	}
	if m.parameters == nil {
		return geofenceControlPlan{}, nil, fmt.Errorf("geofence device parameter reader is not configured")
	}
	parameterSnapshot, err := m.parameters.GetByDevice(ctx, device.ID)
	if err != nil {
		return geofenceControlPlan{}, nil, fmt.Errorf("read geofence device parameter snapshot: %w", err)
	}
	if targets == nil {
		var mappings []carrier.GeofenceControlMapping
		if m.mappings != nil {
			mappings, err = m.mappings.GetByProductClass(
				ctx, device.ProductClass, device.FirmwareVersion,
			)
			if err != nil {
				return geofenceControlPlan{}, nil, fmt.Errorf("read geofence ParamModel mappings: %w", err)
			}
		}
		capability, capabilityErr := carrier.BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
			device.ProductClass,
			device.Technology,
			enabled,
			parameterSnapshot,
			mappings,
		)
		if capabilityErr != nil {
			return geofenceControlPlan{}, nil, fmt.Errorf(
				"resolve geofence deactivation capability from ParamModel: %w", capabilityErr,
			)
		}
		targets = capability.Controls
		terminalTargets = make([]ControlParameterState, 0, len(capability.Terminals))
		for _, terminal := range capability.Terminals {
			terminalTargets = append(terminalTargets, ControlParameterState{
				Path: terminal.Path, Value: terminal.Value, Role: carrier.GeofenceRoleOpState,
			})
		}
	}
	terminals := make([]carrier.GeofenceControlParameter, 0, len(terminalTargets))
	for _, terminal := range terminalTargets {
		terminals = append(terminals, carrier.GeofenceControlParameter{
			Path: terminal.Path, Value: terminal.Value, Role: carrier.GeofenceRoleOpState,
			AccessProven: true,
		})
	}
	plan, err := buildGeofenceControlPlanWithTerminal(
		parameterSnapshot, targets, terminals, forceRequested,
	)
	if err != nil {
		return geofenceControlPlan{}, nil, fmt.Errorf("build geofence control plan: %w", err)
	}
	// Keep this payload identical to acs/rpc.SetParameterValuesHandler. Using
	// the legacy ParameterList/Name/Value shape creates a valid task row but an
	// empty CWMP ParameterList at dispatch time, and also bypasses the existing
	// SPV readback hook.
	params, err := marshalGeofenceControlPayload(plan.Requested)
	if err != nil {
		return geofenceControlPlan{}, nil, fmt.Errorf("marshal geofence control parameters: %w", err)
	}
	return plan, params, nil
}

func (m *GeofenceControlMonitor) reconcileDueVerifications(ctx context.Context) error {
	if m.actions == nil || m.history == nil || m.tasks == nil {
		return fmt.Errorf("geofence terminal verification dependencies are required")
	}
	now := time.Now().UTC()
	actions, err := m.actions.ListDueVerifications(ctx, now, 100)
	if err != nil {
		return fmt.Errorf("list due geofence terminal verifications: %w", err)
	}
	for index := range actions {
		action := &actions[index]
		if action.VerificationDeadline == nil || !now.Before(*action.VerificationDeadline) {
			message := strings.TrimSpace(action.LastError)
			if message == "" {
				message = "OpState terminal verification deadline exceeded"
			}
			if err := m.actions.CompleteVerification(
				ctx, action.ID, action.VerifiedState, ControlActionPartialFailed,
				message, now,
			); err != nil {
				return fmt.Errorf("expire geofence terminal verification: %w", err)
			}
			continue
		}
		if err := m.queueControlVerification(ctx, action); err != nil {
			return fmt.Errorf("queue due geofence terminal verification: %w", err)
		}
	}
	return nil
}
