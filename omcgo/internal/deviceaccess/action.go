package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

type ActionType string

const (
	ActionTypeRFOff ActionType = "rf_off"
	ActionTypeRFOn  ActionType = "rf_on"
)

type ActionDirection string

const (
	ActionDirectionContain ActionDirection = "contain"
	ActionDirectionRelease ActionDirection = "release"
)

type ActionStatus string

const (
	ActionStatusPendingDispatch ActionStatus = "pending_dispatch"
	ActionStatusDispatching     ActionStatus = "dispatching"
	ActionStatusSucceeded       ActionStatus = "succeeded"
	ActionStatusFailed          ActionStatus = "failed"
	ActionStatusCancelled       ActionStatus = "cancelled"
)

// defaultActionMaxAttempts bounds southbound RF execution. Each retry still
// revalidates the access decision, while the hard limit prevents an operator
// from cycling a failing containment/recovery forever.
const defaultActionMaxAttempts = 3

var (
	ErrActionNotRetryable    = errors.New("device access action is not retryable")
	ErrActionRetryExhausted  = errors.New("device access action retry limit exhausted")
	ErrActionStateChanged    = errors.New("device access state no longer permits this action")
	ErrAccessControlDisabled = errors.New("device access control is disabled for this operator")
)

type Action struct {
	ID                 uuid.UUID        `json:"id"`
	DeviceID           uuid.UUID        `json:"device_id"`
	DecisionID         uuid.UUID        `json:"decision_id"`
	ActionType         ActionType       `json:"action_type"`
	Direction          ActionDirection  `json:"direction"`
	Status             ActionStatus     `json:"status"`
	DeviceTaskID       *uuid.UUID       `json:"device_task_id,omitempty"`
	RecoveryOfActionID *uuid.UUID       `json:"recovery_of_action_id,omitempty"`
	OwnedRFChange      bool             `json:"owned_rf_change"`
	RFChangePaths      []string         `json:"rf_change_paths,omitempty"`
	IdempotencyKey     string           `json:"idempotency_key"`
	RequestedBy        *uuid.UUID       `json:"requested_by,omitempty"`
	Attempts           int              `json:"attempts"`
	ErrorMessage       string           `json:"error_message,omitempty"`
	DispatchedAt       *time.Time       `json:"dispatched_at,omitempty"`
	CompletedAt        *time.Time       `json:"completed_at,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	SerialNumber       string           `json:"serial_number,omitempty"`
	ProductName        string           `json:"product_name"`
	Carrier            string           `json:"carrier,omitempty"`
	Technology         model.Technology `json:"technology,omitempty"`
}

type ActionPlan struct {
	DeviceID           uuid.UUID
	DecisionID         uuid.UUID
	ActionType         ActionType
	Direction          ActionDirection
	RecoveryOfActionID *uuid.UUID
	IdempotencyKey     string
	RequestedBy        *uuid.UUID
}

type ActionListFilter struct {
	Carrier       string
	SerialNumber  string
	Status        ActionStatus
	Page          int
	PageSize      int
	VisibleGroups []uuid.UUID
}

type ActionStore interface {
	Plan(ctx context.Context, plan ActionPlan) (Action, bool, error)
	Get(ctx context.Context, id uuid.UUID) (Action, error)
	List(ctx context.Context, filter ActionListFilter) ([]Action, int64, error)
	FindOpenContainment(ctx context.Context, deviceID uuid.UUID) (*Action, error)
	FindOwnedIsolation(ctx context.Context, deviceID uuid.UUID) (*Action, error)
	BeginDispatch(ctx context.Context, actionID uuid.UUID, at time.Time) error
	AttachDispatchTask(ctx context.Context, actionID, deviceTaskID uuid.UUID) error
	MarkVerifying(ctx context.Context, actionID, writeTaskID, readbackTaskID uuid.UUID, at time.Time) error
	SetRFChangePaths(ctx context.Context, actionID uuid.UUID, paths []string, at time.Time) error
	MarkTerminal(ctx context.Context, actionID uuid.UUID, status ActionStatus, ownedRFChange bool, message string, at time.Time) error
	ResetForRetry(ctx context.Context, actionID uuid.UUID) error
}

type RFSwitchDispatcher interface {
	QueueRFSwitch(ctx context.Context, deviceID uuid.UUID, enabled bool, options device.RFSwitchTaskOptions) (*task.Task, error)
	QueueRFReadback(ctx context.Context, deviceID uuid.UUID, options device.RFSwitchTaskOptions) (*task.Task, error)
}

type ActionService struct {
	store      ActionStore
	dispatcher RFSwitchDispatcher
	access     Repository
	carriers   *carrier.CarrierRegistry
	now        func() time.Time
	logger     *zap.Logger
	groups     authz.GroupReader
	settings   RuntimeSettingsReader
}

func NewActionService(
	store ActionStore,
	dispatcher RFSwitchDispatcher,
	access Repository,
	carriers *carrier.CarrierRegistry,
	logger *zap.Logger,
) *ActionService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ActionService{
		store: store, dispatcher: dispatcher, access: access,
		carriers: carriers, now: func() time.Time { return time.Now().UTC() },
		logger: logger.Named("device-access-action"),
	}
}

func (s *ActionService) SetGroupReader(reader authz.GroupReader) {
	s.groups = reader
}

func (s *ActionService) SetRuntimeSettingsReader(reader RuntimeSettingsReader) {
	s.settings = reader
}

func (s *ActionService) List(ctx context.Context, filter ActionListFilter) ([]Action, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, fmt.Errorf("list device access actions: %w", ErrAccessGateDependencyMissing)
	}
	items, total, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list device access actions: %w", err)
	}
	return items, total, nil
}

type decisionActionEvent struct {
	DecisionID      uuid.UUID   `json:"decision_id"`
	DeviceID        *uuid.UUID  `json:"device_id"`
	Carrier         string      `json:"carrier"`
	SerialNumber    string      `json:"serial_number"`
	State           AccessState `json:"state"`
	DecisionVersion int64       `json:"decision_version"`
}

// PlanFromDecision persists an auditable RF action and immediately dispatches
// it when the current access decision still requires containment or recovery.
func (s *ActionService) PlanFromDecision(ctx context.Context, evt event.Event) error {
	if s == nil || s.store == nil || s.access == nil || s.dispatcher == nil {
		return fmt.Errorf("plan device access action: %w", ErrAccessGateDependencyMissing)
	}
	var payload decisionActionEvent
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device access decision action event: %w", err)
	}
	if payload.DeviceID == nil || *payload.DeviceID == uuid.Nil || payload.DecisionID == uuid.Nil {
		return nil // candidates without a formal device cannot receive RF commands
	}
	enabled, err := runtimeAccessEnabled(ctx, s.settings, payload.Carrier)
	if err != nil {
		return fmt.Errorf("load device access business switch before planning RF action: %w", err)
	}
	if !enabled {
		return nil
	}
	current, err := s.access.LoadEvaluationContext(ctx, payload.Carrier, payload.SerialNumber)
	if err != nil {
		return fmt.Errorf("load current access state before planning RF action: %w", err)
	}
	if current.State == nil || current.State.DeviceID == nil || *current.State.DeviceID != *payload.DeviceID ||
		current.State.DecisionVersion != payload.DecisionVersion || current.State.State != payload.State {
		return nil // delayed decision event: the latest projection is authoritative
	}

	plan := ActionPlan{DeviceID: *payload.DeviceID, DecisionID: payload.DecisionID}
	switch payload.State {
	case AccessStateRejected, AccessStateRevoked:
		existing, err := s.store.FindOpenContainment(ctx, *payload.DeviceID)
		if err != nil {
			return fmt.Errorf("find open RF containment: %w", err)
		}
		if existing != nil {
			return s.resumeDispatch(ctx, *existing)
		}
		plan.ActionType = ActionTypeRFOff
		plan.Direction = ActionDirectionContain
	case AccessStateAccepted:
		owned, err := s.store.FindOwnedIsolation(ctx, *payload.DeviceID)
		if err != nil {
			return fmt.Errorf("find owned RF isolation: %w", err)
		}
		if owned == nil {
			return nil
		}
		plan.ActionType = ActionTypeRFOn
		plan.Direction = ActionDirectionRelease
		plan.RecoveryOfActionID = &owned.ID
	default:
		return nil
	}
	plan.IdempotencyKey = fmt.Sprintf("%s:%s", payload.DecisionID, plan.ActionType)
	action, _, err := s.store.Plan(ctx, plan)
	if err != nil {
		return fmt.Errorf("persist device access action plan: %w", err)
	}
	return s.resumeDispatch(ctx, action)
}

func (s *ActionService) resumeDispatch(ctx context.Context, action Action) error {
	if action.Status == ActionStatusFailed || isTerminalActionStatus(action.Status) {
		return nil
	}
	if action.Status == ActionStatusDispatching && action.DeviceTaskID != nil {
		return nil
	}
	enabled, err := runtimeAccessEnabled(ctx, s.settings, action.Carrier)
	if err != nil {
		return fmt.Errorf("load device access business switch before RF dispatch: %w", err)
	}
	if !enabled {
		return s.cancelAction(ctx, action, "access control disabled before RF dispatch")
	}
	current, err := s.access.LoadEvaluationContext(ctx, action.Carrier, action.SerialNumber)
	if err != nil {
		return fmt.Errorf("load current access state before RF dispatch: %w", err)
	}
	if current.State == nil || current.State.DeviceID == nil || *current.State.DeviceID != action.DeviceID ||
		!actionAllowedByState(action.ActionType, current.State.State) {
		return s.cancelAction(ctx, action, "access state changed before RF dispatch")
	}
	if action.Status == ActionStatusPendingDispatch {
		if err := s.store.BeginDispatch(ctx, action.ID, s.now()); err != nil {
			return fmt.Errorf("begin automatic RF action dispatch: %w", err)
		}
		action, err = s.store.Get(ctx, action.ID)
		if err != nil {
			return fmt.Errorf("reload automatic RF action: %w", err)
		}
	}
	if action.Status != ActionStatusDispatching {
		return nil
	}
	var targetPaths []string
	if action.ActionType == ActionTypeRFOff && len(action.RFChangePaths) == 0 {
		created, err := s.dispatcher.QueueRFReadback(ctx, action.DeviceID, device.RFSwitchTaskOptions{
			Source:         task.TaskSourceDeviceAccess,
			SourceID:       action.ID.String(),
			Description:    "device access RF baseline readback",
			CommandKey:     rfBaselineCommandKey(action),
			AdmissionClass: task.AdmissionClassSecurityAction,
		})
		if err != nil {
			_ = s.store.MarkTerminal(ctx, action.ID, ActionStatusFailed, false, err.Error(), s.now())
			return fmt.Errorf("dispatch automatic RF baseline readback: %w", err)
		}
		if created == nil {
			err = errors.New("RF baseline dispatcher returned no task")
			_ = s.store.MarkTerminal(ctx, action.ID, ActionStatusFailed, false, err.Error(), s.now())
			return fmt.Errorf("dispatch automatic RF baseline readback: %w", err)
		}
		return s.attachDispatchTask(ctx, action, created)
	}
	if action.ActionType == ActionTypeRFOn {
		if action.RecoveryOfActionID == nil {
			return s.completeAction(ctx, action, false, "RF recovery has no owned isolation")
		}
		isolation, loadErr := s.store.Get(ctx, *action.RecoveryOfActionID)
		if loadErr != nil {
			return fmt.Errorf("load owned RF isolation before recovery: %w", loadErr)
		}
		if !isolation.OwnedRFChange || len(isolation.RFChangePaths) == 0 {
			return s.completeAction(ctx, action, false, "RF recovery has no verified owned paths")
		}
		targetPaths = append([]string(nil), isolation.RFChangePaths...)
		if len(action.RFChangePaths) == 0 {
			if err := s.store.SetRFChangePaths(ctx, action.ID, targetPaths, s.now()); err != nil {
				return fmt.Errorf("persist RF recovery targets: %w", err)
			}
		} else if !sameRFPaths(action.RFChangePaths, targetPaths) {
			return s.completeAction(ctx, action, false, "RF recovery targets no longer match the owned isolation")
		}
		action.RFChangePaths = targetPaths
	} else {
		targetPaths = append([]string(nil), action.RFChangePaths...)
	}
	rfEnabled := action.ActionType == ActionTypeRFOn
	created, err := s.dispatcher.QueueRFSwitch(ctx, action.DeviceID, rfEnabled, device.RFSwitchTaskOptions{
		Source:         task.TaskSourceDeviceAccess,
		SourceID:       action.ID.String(),
		CreatorID:      "",
		Description:    fmt.Sprintf("automatic device access %s", action.ActionType),
		CommandKey:     rfWriteCommandKey(action),
		AdmissionClass: task.AdmissionClassSecurityAction,
		TargetPaths:    targetPaths,
	})
	if err != nil {
		_ = s.store.MarkTerminal(ctx, action.ID, ActionStatusFailed, false, err.Error(), s.now())
		return fmt.Errorf("dispatch automatic RF action: %w", err)
	}
	if created == nil {
		err = errors.New("RF dispatcher returned no task")
		_ = s.store.MarkTerminal(ctx, action.ID, ActionStatusFailed, false, err.Error(), s.now())
		return fmt.Errorf("dispatch automatic RF action: %w", err)
	}
	return s.attachDispatchTask(ctx, action, created)
}

func (s *ActionService) attachDispatchTask(ctx context.Context, action Action, created *task.Task) error {
	deviceTaskID, err := uuid.Parse(created.ID)
	if err != nil {
		_ = s.store.MarkTerminal(ctx, action.ID, ActionStatusFailed, false, err.Error(), s.now())
		return fmt.Errorf("parse automatic RF device task id: %w", err)
	}
	if err := s.store.AttachDispatchTask(ctx, action.ID, deviceTaskID); err != nil {
		latest, loadErr := s.store.Get(ctx, action.ID)
		if loadErr == nil && latest.Status == ActionStatusDispatching && latest.DeviceTaskID != nil {
			return nil
		}
		// The device task is durable and uses a deterministic command key. Keep
		// dispatching so event redelivery can safely correlate or resume it.
		return fmt.Errorf("attach automatic RF action task: %w", err)
	}
	return nil
}

func (s *ActionService) cancelAction(ctx context.Context, action Action, reason string) error {
	if action.Status != ActionStatusPendingDispatch && action.Status != ActionStatusDispatching {
		return nil
	}
	if err := s.store.MarkTerminal(ctx, action.ID, ActionStatusCancelled, false, reason, s.now()); err != nil {
		return fmt.Errorf("cancel stale RF action: %w", err)
	}
	return nil
}

func (s *ActionService) Retry(ctx context.Context, actionID uuid.UUID) error {
	return s.retry(ctx, nil, actionID)
}

func (s *ActionService) RetryForActor(ctx context.Context, actor PolicyActor, actionID uuid.UUID) error {
	return s.retry(ctx, &actor, actionID)
}

func (s *ActionService) retry(ctx context.Context, actor *PolicyActor, actionID uuid.UUID) error {
	if s == nil || s.store == nil || s.dispatcher == nil || s.access == nil {
		return fmt.Errorf("retry device access action: %w", ErrAccessGateDependencyMissing)
	}
	action, err := s.store.Get(ctx, actionID)
	if err != nil {
		return fmt.Errorf("load failed device access action: %w", err)
	}
	if err := s.authorizeActor(ctx, actor, action); err != nil {
		return err
	}
	enabled, err := runtimeAccessEnabled(ctx, s.settings, action.Carrier)
	if err != nil {
		return fmt.Errorf("load device access business switch before retry: %w", err)
	}
	if !enabled {
		return ErrAccessControlDisabled
	}
	if action.Status != ActionStatusFailed {
		return ErrActionNotRetryable
	}
	if action.Attempts >= defaultActionMaxAttempts {
		return ErrActionRetryExhausted
	}
	if err := s.store.ResetForRetry(ctx, actionID); err != nil {
		return fmt.Errorf("reset device access action for dispatch: %w", err)
	}
	action, err = s.store.Get(ctx, actionID)
	if err != nil {
		return fmt.Errorf("reload retried device access action: %w", err)
	}
	return s.resumeDispatch(ctx, action)
}

func (s *ActionService) authorizeActor(ctx context.Context, actor *PolicyActor, action Action) error {
	if actor == nil {
		return nil
	}
	if strings.TrimSpace(actor.Carrier) != strings.TrimSpace(action.Carrier) {
		return ErrPolicyCarrierScope
	}
	if err := authz.AuthorizeDeviceAccess(ctx, s.groups, action.DeviceID, actor.VisibleGroups); err != nil {
		if errors.Is(err, commonerrors.ErrForbidden) {
			return commonerrors.ErrForbidden
		}
		return fmt.Errorf("authorize RF action device scope: %w", err)
	}
	return nil
}

func (s *ActionService) HandleCompleted(ctx context.Context, completed *task.Task) error {
	if completed == nil || completed.Source != task.TaskSourceDeviceAccess ||
		completed.AdmissionClass != task.AdmissionClassSecurityAction || completed.SourceID == "" {
		return nil
	}
	actionID, err := uuid.Parse(completed.SourceID)
	if err != nil {
		return fmt.Errorf("parse completed security action id: %w", err)
	}
	if s.store == nil {
		return fmt.Errorf("complete device access action: %w", ErrAccessGateDependencyMissing)
	}
	action, err := s.store.Get(ctx, actionID)
	if err != nil {
		return fmt.Errorf("load completed security action: %w", err)
	}
	if isTerminalActionStatus(action.Status) {
		return nil
	}
	if action.Status != ActionStatusDispatching {
		return nil
	}
	if completed.DeviceSN != "" && completed.DeviceSN != action.SerialNumber {
		return nil
	}
	completedTaskID, parseErr := uuid.Parse(completed.ID)
	if parseErr != nil {
		return s.completeAction(ctx, action, false, fmt.Sprintf("parse completed RF task id: %v", parseErr))
	}
	isExpectedReadback := completed.Method == "GetParameterValues" &&
		completed.CommandKey == rfReadbackCommandKey(action)
	isExpectedBaseline := completed.Method == "GetParameterValues" &&
		completed.CommandKey == rfBaselineCommandKey(action)
	isExpectedWrite := completed.Method == "SetParameterValues" &&
		completed.CommandKey == rfWriteCommandKey(action)
	if action.DeviceTaskID != nil && *action.DeviceTaskID != completedTaskID &&
		!isExpectedReadback && !isExpectedBaseline && !isExpectedWrite {
		return nil
	}
	if completed.Status != task.TaskStatusCompleted {
		if completed.Status != task.TaskStatusFailed && completed.Status != task.TaskStatusExpired && completed.Status != task.TaskStatusCancelled {
			return nil
		}
		message := strings.TrimSpace(completed.ErrorMessage)
		if message == "" {
			message = fmt.Sprintf("RF %s task ended with status %s", completed.Method, completed.Status)
		}
		return s.completeAction(ctx, action, false, message)
	}

	switch completed.Method {
	case "SetParameterValues":
		return s.queueRFReadback(ctx, action, completedTaskID, completed.Params)
	case "GetParameterValues":
		if completed.CommandKey == rfBaselineCommandKey(action) {
			return s.handleRFBaseline(ctx, action, completedTaskID, completed.Params, completed.Result)
		}
		if err := s.verifyRFReadback(action, completed.Params, completed.Result); err != nil {
			return s.completeAction(ctx, action, false, err.Error())
		}
		return s.completeAction(ctx, action, true, "")
	default:
		return nil
	}
}

func (s *ActionService) handleRFBaseline(
	ctx context.Context,
	action Action,
	baselineTaskID uuid.UUID,
	paramsRaw, resultRaw json.RawMessage,
) error {
	paths, err := enabledRFPaths(paramsRaw, resultRaw)
	if err != nil {
		return s.completeAction(ctx, action, false, err.Error())
	}
	if len(paths) == 0 {
		return s.completeAction(ctx, action, true, "")
	}
	if err := s.store.SetRFChangePaths(ctx, action.ID, paths, s.now()); err != nil {
		return fmt.Errorf("persist RF containment baseline: %w", err)
	}
	action.RFChangePaths = paths
	created, err := s.dispatcher.QueueRFSwitch(ctx, action.DeviceID, false, device.RFSwitchTaskOptions{
		Source:         task.TaskSourceDeviceAccess,
		SourceID:       action.ID.String(),
		Description:    "automatic device access rf_off",
		CommandKey:     rfWriteCommandKey(action),
		AdmissionClass: task.AdmissionClassSecurityAction,
		TargetPaths:    paths,
	})
	if err != nil {
		return s.completeAction(ctx, action, false, fmt.Sprintf("queue RF containment after baseline: %v", err))
	}
	if created == nil {
		return s.completeAction(ctx, action, false, "RF dispatcher returned no task after baseline")
	}
	writeTaskID, err := uuid.Parse(created.ID)
	if err != nil {
		return s.completeAction(ctx, action, false, fmt.Sprintf("parse RF containment task id: %v", err))
	}
	if err := s.store.MarkVerifying(ctx, action.ID, baselineTaskID, writeTaskID, s.now()); err != nil {
		return fmt.Errorf("correlate RF baseline with containment task: %w", err)
	}
	return nil
}

func (s *ActionService) queueRFReadback(
	ctx context.Context,
	action Action,
	writeTaskID uuid.UUID,
	writeParams json.RawMessage,
) error {
	if s.dispatcher == nil {
		return s.completeAction(ctx, action, false, "RF readback dispatcher is unavailable")
	}
	targetPaths, err := rfWriteTargetPaths(writeParams)
	if err != nil {
		return s.completeAction(ctx, action, false, err.Error())
	}
	if len(action.RFChangePaths) == 0 || !sameRFPaths(targetPaths, action.RFChangePaths) {
		return s.completeAction(ctx, action, false, "RF write targets do not match the action-owned paths")
	}
	targetPaths = append([]string(nil), action.RFChangePaths...)
	created, err := s.dispatcher.QueueRFReadback(ctx, action.DeviceID, device.RFSwitchTaskOptions{
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		Description:    "device access RF readback verification",
		CommandKey:     rfReadbackCommandKey(action),
		AdmissionClass: task.AdmissionClassSecurityAction,
		TargetPaths:    targetPaths,
	})
	if err != nil {
		return s.completeAction(ctx, action, false, fmt.Sprintf("queue RF readback: %v", err))
	}
	if created == nil {
		return s.completeAction(ctx, action, false, "RF readback dispatcher returned no task")
	}
	readbackTaskID, err := uuid.Parse(created.ID)
	if err != nil {
		return s.completeAction(ctx, action, false, fmt.Sprintf("parse RF readback task id: %v", err))
	}
	if writeTaskID != uuid.Nil {
		if err := s.store.MarkVerifying(ctx, action.ID, writeTaskID, readbackTaskID, s.now()); err != nil {
			if errors.Is(err, ErrActionStateChanged) {
				latest, loadErr := s.store.Get(ctx, action.ID)
				if loadErr == nil {
					if isTerminalActionStatus(latest.Status) {
						return nil
					}
					if latest.Status == ActionStatusDispatching && latest.DeviceTaskID != nil &&
						*latest.DeviceTaskID != writeTaskID {
						return nil // a duplicate completion already correlated readback
					}
				}
			}
			return s.completeAction(ctx, action, false, fmt.Sprintf("correlate RF readback task: %v", err))
		}
	}
	return nil
}

func rfWriteTargetPaths(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("RF write parameters are missing")
	}
	var params struct {
		Values []struct {
			Name string `json:"name"`
		} `json:"values"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || len(params.Values) == 0 {
		return nil, fmt.Errorf("decode RF write parameters")
	}
	paths := make([]string, 0, len(params.Values))
	seen := make(map[string]struct{}, len(params.Values))
	for _, value := range params.Values {
		path := strings.TrimSpace(value.Name)
		if path == "" {
			return nil, fmt.Errorf("RF write path is empty")
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths, nil
}

func sameRFPaths(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	want := make(map[string]struct{}, len(left))
	for _, path := range left {
		want[strings.TrimSpace(path)] = struct{}{}
	}
	for _, path := range right {
		if _, ok := want[strings.TrimSpace(path)]; !ok {
			return false
		}
	}
	return true
}

func enabledRFPaths(paramsRaw, resultRaw json.RawMessage) ([]string, error) {
	var params struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(paramsRaw, &params); err != nil || len(params.Names) == 0 {
		return nil, fmt.Errorf("decode RF baseline parameters")
	}
	var result struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"standard_parameter_values"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		return nil, fmt.Errorf("decode RF baseline result: %w", err)
	}
	values := make(map[string]string, len(result.Values))
	for _, value := range result.Values {
		values[strings.TrimSpace(value.Name)] = value.Value
	}
	enabled := make([]string, 0, len(params.Names))
	seen := make(map[string]struct{}, len(params.Names))
	for _, rawPath := range params.Names {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			return nil, fmt.Errorf("RF baseline path is empty")
		}
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		value, ok := values[path]
		if !ok {
			return nil, fmt.Errorf("RF baseline path %s is missing", path)
		}
		isEnabled, valid := normalizeRFBool(value)
		if !valid {
			return nil, fmt.Errorf("RF baseline value %q is invalid", value)
		}
		if isEnabled {
			enabled = append(enabled, path)
		}
	}
	sort.Strings(enabled)
	return enabled, nil
}

func (s *ActionService) verifyRFReadback(action Action, paramsRaw, resultRaw json.RawMessage) error {
	if s.carriers == nil {
		return ErrAccessGateDependencyMissing
	}
	adapter, err := s.carriers.Get(model.CarrierCode(action.Carrier))
	if err != nil {
		return fmt.Errorf("resolve RF readback carrier: %w", err)
	}
	fallbackPath := adapter.RFControlPath(action.Technology)
	if fallbackPath == "" {
		return fmt.Errorf("RF readback path is unavailable")
	}
	paths := []string{fallbackPath}
	if len(paramsRaw) > 0 {
		var params struct {
			Names []string `json:"names"`
		}
		if err := json.Unmarshal(paramsRaw, &params); err != nil || len(params.Names) == 0 {
			return fmt.Errorf("decode RF readback parameters")
		}
		paths = params.Names
	}
	var result struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"standard_parameter_values"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		return fmt.Errorf("decode RF readback result: %w", err)
	}
	expected := action.ActionType == ActionTypeRFOn
	values := make(map[string]string, len(result.Values))
	for _, value := range result.Values {
		values[value.Name] = value.Value
	}
	for _, path := range paths {
		value, ok := values[path]
		if !ok {
			return fmt.Errorf("RF readback path %s is missing", path)
		}
		actual, valid := normalizeRFBool(value)
		if !valid {
			return fmt.Errorf("RF readback value %q is invalid", value)
		}
		if actual != expected {
			return fmt.Errorf("RF readback mismatch for %s: expected %t got %t", path, expected, actual)
		}
	}
	return nil
}

func (s *ActionService) completeAction(ctx context.Context, action Action, succeeded bool, message string) error {
	status := ActionStatusFailed
	owned := false
	if succeeded {
		status = ActionStatusSucceeded
		owned = action.ActionType == ActionTypeRFOff && len(action.RFChangePaths) > 0
		message = ""
	}
	now := s.now()
	if err := s.store.MarkTerminal(ctx, action.ID, status, owned, message, now); err != nil {
		return fmt.Errorf("persist RF action completion: %w", err)
	}
	action.Status = status
	action.OwnedRFChange = owned
	action.ErrorMessage = message
	action.CompletedAt = &now
	return nil
}

func rfReadbackCommandKey(action Action) string {
	return fmt.Sprintf("device-access-action:%s:%d:verify", action.ID, action.Attempts)
}

func rfBaselineCommandKey(action Action) string {
	return fmt.Sprintf("device-access-action:%s:%d:baseline", action.ID, action.Attempts)
}

func rfWriteCommandKey(action Action) string {
	return fmt.Sprintf("device-access-action:%s:%d", action.ID, action.Attempts)
}

func isTerminalActionStatus(status ActionStatus) bool {
	return status == ActionStatusSucceeded || status == ActionStatusFailed || status == ActionStatusCancelled
}

func normalizeRFBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true":
		return true, true
	case "0", "false":
		return false, true
	default:
		return false, false
	}
}

func (s *ActionService) OnTaskCompleted(ctx context.Context, completed *task.Task) {
	if err := s.OnTaskCompletedReliable(ctx, completed); err != nil {
		s.logger.Warn("project RF security action completion", zap.Error(err))
	}
}

func (s *ActionService) OnTaskCompletedReliable(ctx context.Context, completed *task.Task) error {
	return s.HandleCompleted(ctx, completed)
}

// AuthorizeSecurityAction is called both when the task is created and when ACS
// dequeues it. An intervening access decision can therefore stop stale RF work.
func (s *ActionService) AuthorizeSecurityAction(ctx context.Context, request task.TaskAdmissionRequest) (bool, string, error) {
	if s == nil || s.store == nil || s.access == nil {
		return false, "security_action_dependency_missing", ErrAccessGateDependencyMissing
	}
	if request.Source != task.TaskSourceDeviceAccess ||
		(request.Method != "SetParameterValues" && request.Method != "GetParameterValues") {
		return false, "invalid_security_action_contract", nil
	}
	actionID, err := uuid.Parse(request.SourceID)
	if err != nil {
		return false, "invalid_security_action_id", nil
	}
	action, err := s.store.Get(ctx, actionID)
	if err != nil {
		return false, "security_action_unavailable", fmt.Errorf("load security action: %w", err)
	}
	if action.Status != ActionStatusDispatching {
		return false, "security_action_not_dispatching", nil
	}
	enabled, err := runtimeAccessEnabled(ctx, s.settings, action.Carrier)
	if err != nil {
		return false, "access_control_settings_unavailable", fmt.Errorf("load device access business switch before security action: %w", err)
	}
	if !enabled && request.Method != "GetParameterValues" {
		return false, "access_control_disabled", nil
	}
	if action.SerialNumber != request.DeviceSN {
		return false, "security_action_target_mismatch", nil
	}
	if request.Method == "SetParameterValues" {
		current, err := s.access.LoadEvaluationContext(ctx, action.Carrier, action.SerialNumber)
		if err != nil {
			return false, "security_action_state_unavailable", fmt.Errorf("load security action access state: %w", err)
		}
		if current.State == nil || !actionAllowedByState(action.ActionType, current.State.State) {
			return false, "security_action_state_changed", nil
		}
	}
	if s.carriers == nil {
		return false, "security_action_carrier_registry_missing", ErrAccessGateDependencyMissing
	}
	adapter, err := s.carriers.Get(model.CarrierCode(action.Carrier))
	if err != nil {
		return false, "security_action_carrier_unavailable", fmt.Errorf("resolve security action carrier: %w", err)
	}
	path := adapter.RFControlPath(action.Technology)
	if path == "" {
		return false, "security_action_rf_unsupported", nil
	}
	if request.Method == "SetParameterValues" {
		expectedValue := "0"
		if action.ActionType == ActionTypeRFOn {
			expectedValue = "1"
		}
		var params struct {
			Values []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
				Type  string `json:"type"`
			} `json:"values"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil || len(params.Values) == 0 {
			return false, "invalid_security_action_contract", nil
		}
		for _, value := range params.Values {
			if (value.Name != path && !carrier.IsRFControlPath(value.Name)) ||
				value.Value != expectedValue || value.Type != "xsd:boolean" {
				return false, "invalid_security_action_contract", nil
			}
		}
		return true, "automatic_security_action", nil
	}
	var params struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil || len(params.Names) == 0 {
		return false, "invalid_security_action_contract", nil
	}
	for _, name := range params.Names {
		if name != path && !carrier.IsRFControlPath(name) {
			return false, "invalid_security_action_contract", nil
		}
	}
	return true, "automatic_security_action_readback", nil
}

func actionAllowedByState(actionType ActionType, state AccessState) bool {
	if actionType == ActionTypeRFOff {
		return state == AccessStateRejected || state == AccessStateRevoked
	}
	return actionType == ActionTypeRFOn && state == AccessStateAccepted
}

type ActionConsumer struct {
	bus  event.EventBus
	svc  *ActionService
	subs []event.Subscription
}

func NewActionConsumer(bus event.EventBus, svc *ActionService) *ActionConsumer {
	return &ActionConsumer{bus: bus, svc: svc}
}

func (c *ActionConsumer) Start() error {
	if c == nil || c.bus == nil || c.svc == nil {
		return ErrAccessGateDependencyMissing
	}
	for _, binding := range []struct {
		subject string
		queue   string
	}{
		{subject: event.SubjectDeviceAccessRejected, queue: "device-access-actions-rejected"},
		{subject: event.SubjectDeviceAccessRevoked, queue: "device-access-actions-revoked"},
		{subject: event.SubjectDeviceAccessAccepted, queue: "device-access-actions-accepted"},
	} {
		sub, err := c.bus.QueueSubscribe(binding.subject, binding.queue, c.svc.PlanFromDecision)
		if err != nil {
			_ = c.Stop()
			return fmt.Errorf("subscribe %s action planning: %w", binding.subject, err)
		}
		c.subs = append(c.subs, sub)
	}
	return nil
}

func (c *ActionConsumer) Stop() error {
	var first error
	for _, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil && first == nil {
			first = err
		}
	}
	c.subs = nil
	return first
}

var _ task.TaskCompletionCallback = (*ActionService)(nil)
