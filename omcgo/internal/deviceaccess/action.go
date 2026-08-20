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
	ActionStatusVerifying       ActionStatus = "verifying"
	ActionStatusRetryWait       ActionStatus = "retry_wait"
	ActionStatusSucceeded       ActionStatus = "succeeded"
	ActionStatusFailed          ActionStatus = "failed"
	ActionStatusDead            ActionStatus = "dead"
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
	DeviceID           *uuid.UUID       `json:"device_id,omitempty"`
	CandidateID        *uuid.UUID       `json:"candidate_id,omitempty"`
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
	MaxAttempts        int              `json:"max_attempts"`
	NextAttemptAt      *time.Time       `json:"next_attempt_at,omitempty"`
	LastFailureCode    string           `json:"last_failure_code,omitempty"`
	DeadAt             *time.Time       `json:"dead_at,omitempty"`
	ManualRepairNeeded bool             `json:"manual_repair_required"`
	ErrorMessage       string           `json:"error_message,omitempty"`
	DispatchedAt       *time.Time       `json:"dispatched_at,omitempty"`
	CompletedAt        *time.Time       `json:"completed_at,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	SerialNumber       string           `json:"serial_number,omitempty"`
	ProductName        string           `json:"product_name"`
	Carrier            string           `json:"carrier,omitempty"`
	Technology         model.Technology `json:"technology,omitempty"`
	CandidateRFPaths   []string         `json:"-"`
	CandidateOUI       string           `json:"-"`
	BoundSessionID     string           `json:"-"`
	BoundRequestID     string           `json:"-"`
}

// ActionLifecycleEvent is the stable payload emitted after an RF action
// failure or recovery is committed. The outbox event and the action state are
// persisted in the same transaction; downstream notification failures cannot
// change either result.
type ActionLifecycleEvent struct {
	EventID       string       `json:"event_id"`
	EventName     string       `json:"event_name"`
	RequestID     string       `json:"request_id"`
	DecisionID    uuid.UUID    `json:"decision_id"`
	ActionID      uuid.UUID    `json:"action_id"`
	DeviceID      *uuid.UUID   `json:"device_id,omitempty"`
	CandidateID   *uuid.UUID   `json:"candidate_id,omitempty"`
	Carrier       string       `json:"carrier"`
	SerialNumber  string       `json:"serial_number"`
	PolicyVersion *uuid.UUID   `json:"policy_version_id,omitempty"`
	ActionType    ActionType   `json:"action_type"`
	ActionStatus  ActionStatus `json:"action_status"`
	Attempt       int          `json:"attempt"`
	ReasonCode    string       `json:"reason_code,omitempty"`
	Message       string       `json:"message,omitempty"`
	OccurredAt    time.Time    `json:"occurred_at"`
}

type ActionPlan struct {
	DeviceID           *uuid.UUID
	CandidateID        *uuid.UUID
	DecisionID         uuid.UUID
	ActionType         ActionType
	Direction          ActionDirection
	RecoveryOfActionID *uuid.UUID
	IdempotencyKey     string
	RequestedBy        *uuid.UUID
}

type ActionListFilter struct {
	Carrier         string
	SerialNumber    string
	Status          ActionStatus
	Decision        EffectiveAction
	ReasonCode      ReasonCode
	PolicyVersionID *uuid.UUID
	StartedAt       *time.Time
	EndedAt         *time.Time
	Page            int
	PageSize        int
	VisibleGroups   []uuid.UUID
}

type ActionStore interface {
	Plan(ctx context.Context, plan ActionPlan) (Action, bool, error)
	Get(ctx context.Context, id uuid.UUID) (Action, error)
	List(ctx context.Context, filter ActionListFilter) ([]Action, int64, error)
	FindOpenContainment(ctx context.Context, deviceID uuid.UUID) (*Action, error)
	FindOpenCandidateContainment(ctx context.Context, candidateID uuid.UUID) (*Action, error)
	PromoteCandidateTarget(ctx context.Context, candidateID uuid.UUID, carrier, serialNumber string) (uuid.UUID, error)
	FindOwnedIsolation(ctx context.Context, deviceID uuid.UUID) (*Action, error)
	BeginDispatch(ctx context.Context, actionID uuid.UUID, at time.Time) error
	AttachDispatchTask(ctx context.Context, actionID, deviceTaskID uuid.UUID) error
	AdvanceDispatchTask(ctx context.Context, actionID, currentTaskID, nextTaskID uuid.UUID, status ActionStatus, at time.Time) error
	SetRFChangePaths(ctx context.Context, actionID uuid.UUID, paths []string, at time.Time) error
	MarkTerminal(ctx context.Context, actionID uuid.UUID, status ActionStatus, ownedRFChange bool, failureCode, message string, at time.Time) error
	ScheduleRetry(ctx context.Context, actionID uuid.UUID, failureCode, message string, nextAttemptAt time.Time, dead bool) error
	ResetForManualRetry(ctx context.Context, actionID uuid.UUID, reason string, at time.Time) error
	RecordAttemptQueued(ctx context.Context, attempt ActionAttempt) error
	RecordAttemptCompleted(ctx context.Context, attempt ActionAttempt) error
	ListAttempts(ctx context.Context, actionID uuid.UUID) ([]ActionAttempt, error)
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]Action, error)
	ListRecoverable(ctx context.Context, staleBefore time.Time, limit int) ([]Action, error)
	LoadActionTask(ctx context.Context, action Action) (*task.Task, error)
	TouchAction(ctx context.Context, actionID uuid.UUID, at time.Time) error
	WakeRetryForDevice(ctx context.Context, deviceID uuid.UUID, at time.Time) error
	BindCandidateExecution(ctx context.Context, actionID uuid.UUID, sessionID, requestID string, at time.Time) error
}

type RFSwitchDispatcher interface {
	QueueRFSwitch(ctx context.Context, deviceID uuid.UUID, enabled bool, options device.RFSwitchTaskOptions) (*task.Task, error)
	QueueRFReadback(ctx context.Context, deviceID uuid.UUID, options device.RFSwitchTaskOptions) (*task.Task, error)
}

type CandidateRFSwitchDispatcher interface {
	QueueCandidateRFSwitch(ctx context.Context, target device.CandidateRFTarget, enabled bool, options device.RFSwitchTaskOptions) (*task.Task, error)
	QueueCandidateRFReadback(ctx context.Context, target device.CandidateRFTarget, options device.RFSwitchTaskOptions) (*task.Task, error)
}

type ActionService struct {
	store         ActionStore
	dispatcher    RFSwitchDispatcher
	access        Repository
	carriers      *carrier.CarrierRegistry
	now           func() time.Time
	logger        *zap.Logger
	groups        authz.GroupReader
	identityScope IdentityVisibilityChecker
	settings      RuntimeSettingsReader
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

func (s *ActionService) SetIdentityVisibilityChecker(checker IdentityVisibilityChecker) {
	s.identityScope = checker
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

func (s *ActionService) ListAttemptsForActor(
	ctx context.Context,
	actor PolicyActor,
	actionID uuid.UUID,
) ([]ActionAttempt, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("list device access action attempts: %w", ErrAccessGateDependencyMissing)
	}
	action, err := s.store.Get(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("load device access action before listing attempts: %w", err)
	}
	if err := s.authorizeActor(ctx, &actor, action); err != nil {
		return nil, err
	}
	items, err := s.store.ListAttempts(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("list device access action attempts: %w", err)
	}
	return items, nil
}

type decisionActionEvent struct {
	DecisionID      uuid.UUID   `json:"decision_id"`
	DeviceID        *uuid.UUID  `json:"device_id"`
	CandidateID     *uuid.UUID  `json:"candidate_id"`
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
	if payload.DecisionID == uuid.Nil || !exactlyOneActionTarget(payload.DeviceID, payload.CandidateID) {
		return nil
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
	if current.State == nil || !sameActionTarget(current.State.DeviceID, current.State.CandidateID, payload.DeviceID, payload.CandidateID) ||
		current.State.DecisionVersion != payload.DecisionVersion || current.State.State != payload.State {
		return nil // delayed decision event: the latest projection is authoritative
	}

	plan := ActionPlan{DeviceID: payload.DeviceID, CandidateID: payload.CandidateID, DecisionID: payload.DecisionID}
	switch payload.State {
	case AccessStateRejected, AccessStateRevoked:
		var existing *Action
		if payload.DeviceID != nil {
			existing, err = s.store.FindOpenContainment(ctx, *payload.DeviceID)
		} else {
			existing, err = s.store.FindOpenCandidateContainment(ctx, *payload.CandidateID)
		}
		if err != nil {
			return fmt.Errorf("find open RF containment: %w", err)
		}
		if existing != nil {
			return s.resumeDispatch(ctx, *existing)
		}
		plan.ActionType = ActionTypeRFOff
		plan.Direction = ActionDirectionContain
	case AccessStateAccepted:
		if payload.DeviceID == nil {
			promotedDeviceID, promoteErr := s.store.PromoteCandidateTarget(
				ctx, *payload.CandidateID, payload.Carrier, payload.SerialNumber,
			)
			if promoteErr != nil {
				return fmt.Errorf("promote accepted candidate RF ownership: %w", promoteErr)
			}
			payload.DeviceID = &promotedDeviceID
			payload.CandidateID = nil
			plan.DeviceID = payload.DeviceID
			plan.CandidateID = nil
		}
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

func exactlyOneActionTarget(deviceID, candidateID *uuid.UUID) bool {
	deviceValid := deviceID != nil && *deviceID != uuid.Nil
	candidateValid := candidateID != nil && *candidateID != uuid.Nil
	return deviceValid != candidateValid
}

func sameActionTarget(leftDevice, leftCandidate, rightDevice, rightCandidate *uuid.UUID) bool {
	if !exactlyOneActionTarget(leftDevice, leftCandidate) || !exactlyOneActionTarget(rightDevice, rightCandidate) {
		return false
	}
	if leftDevice != nil && rightDevice != nil {
		return *leftDevice == *rightDevice
	}
	return leftCandidate != nil && rightCandidate != nil && *leftCandidate == *rightCandidate
}

func (s *ActionService) resumeDispatch(ctx context.Context, action Action) error {
	if action.Status == ActionStatusFailed || action.Status == ActionStatusRetryWait || isTerminalActionStatus(action.Status) {
		return nil
	}
	if (action.Status == ActionStatusDispatching || action.Status == ActionStatusVerifying) && action.DeviceTaskID != nil {
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
	if current.State == nil || !sameActionTarget(current.State.DeviceID, current.State.CandidateID, action.DeviceID, action.CandidateID) ||
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
	if action.Status != ActionStatusDispatching && action.Status != ActionStatusVerifying {
		return nil
	}
	var targetPaths []string
	if action.ActionType == ActionTypeRFOff && len(action.RFChangePaths) == 0 {
		created, err := s.dispatchRFReadback(ctx, action, device.RFSwitchTaskOptions{
			Source:         task.TaskSourceDeviceAccess,
			SourceID:       action.ID.String(),
			Description:    "device access RF baseline readback",
			CommandKey:     rfBaselineCommandKey(action),
			AdmissionClass: task.AdmissionClassSecurityAction,
		})
		if err != nil {
			code, retryable := classifyRFDispatchFailure(err)
			_ = s.failAction(ctx, action, code, err.Error(), retryable)
			return fmt.Errorf("dispatch automatic RF baseline readback: %w", err)
		}
		if created == nil {
			err = errors.New("RF baseline dispatcher returned no task")
			_ = s.failAction(ctx, action, "dispatcher_empty", err.Error(), true)
			return fmt.Errorf("dispatch automatic RF baseline readback: %w", err)
		}
		return s.attachDispatchTask(ctx, action, created, ActionAttemptBaselineGPV)
	}
	if action.ActionType == ActionTypeRFOn {
		if action.RecoveryOfActionID == nil {
			return s.failAction(ctx, action, "rf_ownership_missing", "RF recovery has no owned isolation", false)
		}
		isolation, loadErr := s.store.Get(ctx, *action.RecoveryOfActionID)
		if loadErr != nil {
			return fmt.Errorf("load owned RF isolation before recovery: %w", loadErr)
		}
		if !isolation.OwnedRFChange || len(isolation.RFChangePaths) == 0 {
			return s.failAction(ctx, action, "rf_ownership_missing", "RF recovery has no verified owned paths", false)
		}
		targetPaths = append([]string(nil), isolation.RFChangePaths...)
		if len(action.RFChangePaths) == 0 {
			if err := s.store.SetRFChangePaths(ctx, action.ID, targetPaths, s.now()); err != nil {
				return fmt.Errorf("persist RF recovery targets: %w", err)
			}
		} else if !sameRFPaths(action.RFChangePaths, targetPaths) {
			return s.failAction(ctx, action, "rf_ownership_mismatch", "RF recovery targets no longer match the owned isolation", false)
		}
		action.RFChangePaths = targetPaths
	} else {
		targetPaths = append([]string(nil), action.RFChangePaths...)
	}
	rfEnabled := action.ActionType == ActionTypeRFOn
	created, err := s.dispatchRFSwitch(ctx, action, rfEnabled, device.RFSwitchTaskOptions{
		Source:         task.TaskSourceDeviceAccess,
		SourceID:       action.ID.String(),
		CreatorID:      "",
		Description:    fmt.Sprintf("automatic device access %s", action.ActionType),
		CommandKey:     rfWriteCommandKey(action),
		AdmissionClass: task.AdmissionClassSecurityAction,
		TargetPaths:    targetPaths,
	})
	if err != nil {
		code, retryable := classifyRFDispatchFailure(err)
		_ = s.failAction(ctx, action, code, err.Error(), retryable)
		return fmt.Errorf("dispatch automatic RF action: %w", err)
	}
	if created == nil {
		err = errors.New("RF dispatcher returned no task")
		_ = s.failAction(ctx, action, "dispatcher_empty", err.Error(), true)
		return fmt.Errorf("dispatch automatic RF action: %w", err)
	}
	return s.attachDispatchTask(ctx, action, created, ActionAttemptSPV)
}

func (s *ActionService) attachDispatchTask(
	ctx context.Context,
	action Action,
	created *task.Task,
	phase ActionAttemptPhase,
) error {
	deviceTaskID, err := uuid.Parse(created.ID)
	if err != nil {
		_ = s.failAction(ctx, action, "invalid_task_id", err.Error(), false)
		return fmt.Errorf("parse automatic RF device task id: %w", err)
	}
	if err := s.recordAttemptQueued(ctx, action, created, deviceTaskID, phase); err != nil {
		return err
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

func (s *ActionService) dispatchRFSwitch(
	ctx context.Context,
	action Action,
	enabled bool,
	options device.RFSwitchTaskOptions,
) (*task.Task, error) {
	if action.DeviceID != nil {
		return s.dispatcher.QueueRFSwitch(ctx, *action.DeviceID, enabled, options)
	}
	if action.CandidateID == nil || enabled {
		return nil, fmt.Errorf("candidate RF target cannot execute action=%s: %w", action.ActionType, ErrActionStateChanged)
	}
	dispatcher, ok := s.dispatcher.(CandidateRFSwitchDispatcher)
	if !ok {
		return nil, fmt.Errorf("candidate RF dispatcher is unavailable: %w", ErrAccessGateDependencyMissing)
	}
	return dispatcher.QueueCandidateRFSwitch(ctx, candidateRFTarget(action), false, options)
}

func (s *ActionService) dispatchRFReadback(
	ctx context.Context,
	action Action,
	options device.RFSwitchTaskOptions,
) (*task.Task, error) {
	if action.DeviceID != nil {
		return s.dispatcher.QueueRFReadback(ctx, *action.DeviceID, options)
	}
	if action.CandidateID == nil {
		return nil, fmt.Errorf("RF action has no target: %w", ErrActionStateChanged)
	}
	dispatcher, ok := s.dispatcher.(CandidateRFSwitchDispatcher)
	if !ok {
		return nil, fmt.Errorf("candidate RF dispatcher is unavailable: %w", ErrAccessGateDependencyMissing)
	}
	return dispatcher.QueueCandidateRFReadback(ctx, candidateRFTarget(action), options)
}

func candidateRFTarget(action Action) device.CandidateRFTarget {
	target := device.CandidateRFTarget{
		Carrier:        model.CarrierCode(action.Carrier),
		SerialNumber:   action.SerialNumber,
		ProductClass:   action.ProductName,
		Technology:     action.Technology,
		RFControlPaths: append([]string(nil), action.CandidateRFPaths...),
	}
	if action.CandidateID != nil {
		target.CandidateID = *action.CandidateID
	}
	return target
}

func (s *ActionService) cancelAction(ctx context.Context, action Action, reason string) error {
	if action.Status != ActionStatusPendingDispatch && action.Status != ActionStatusDispatching {
		return nil
	}
	if err := s.store.MarkTerminal(ctx, action.ID, ActionStatusCancelled, false, "", reason, s.now()); err != nil {
		return fmt.Errorf("cancel stale RF action: %w", err)
	}
	return nil
}

func (s *ActionService) Retry(ctx context.Context, actionID uuid.UUID) error {
	return s.retry(ctx, nil, actionID, "manual retry")
}

func (s *ActionService) RetryForActor(ctx context.Context, actor PolicyActor, actionID uuid.UUID, reason string) error {
	return s.retry(ctx, &actor, actionID, reason)
}

func (s *ActionService) retry(ctx context.Context, actor *PolicyActor, actionID uuid.UUID, reason string) error {
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
	if action.Status != ActionStatusFailed && action.Status != ActionStatusDead {
		return ErrActionNotRetryable
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("retry device access action: repair reason is required")
	}
	if err := s.store.ResetForManualRetry(ctx, actionID, reason, s.now()); err != nil {
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
	var err error
	if action.DeviceID != nil {
		err = authz.AuthorizeDeviceAccess(ctx, s.groups, *action.DeviceID, actor.VisibleGroups)
	} else {
		err = authorizeIdentityScope(ctx, s.identityScope, *actor, action.SerialNumber)
	}
	if err != nil {
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
	if action.Status != ActionStatusDispatching && action.Status != ActionStatusVerifying {
		return nil
	}
	if completed.DeviceSN != "" && completed.DeviceSN != action.SerialNumber {
		return nil
	}
	completedTaskID, parseErr := uuid.Parse(completed.ID)
	if parseErr != nil {
		return s.failAction(ctx, action, "invalid_task_id", fmt.Sprintf("parse completed RF task id: %v", parseErr), false)
	}
	isExpectedReadback := completed.Method == "GetParameterValues" &&
		completed.CommandKey == rfReadbackCommandKey(action)
	isExpectedBaseline := completed.Method == "GetParameterValues" &&
		completed.CommandKey == rfBaselineCommandKey(action)
	isExpectedWrite := completed.Method == "SetParameterValues" &&
		completed.CommandKey == rfWriteCommandKey(action)
	if !isExpectedReadback && !isExpectedBaseline && !isExpectedWrite {
		return nil
	}
	// Command keys identify the action attempt and phase, but concurrent delivery
	// can still leave more than one task carrying the same key. Only the durable
	// device_task_id may advance this action; a completion from a superseded task
	// must be ignored.
	if action.DeviceTaskID == nil || *action.DeviceTaskID != completedTaskID {
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
		failureCode, retryable := classifyRFTaskFailure(completed)
		if err := s.recordAttemptCompleted(ctx, action, completed, actionAttemptFailureStatus(completed), failureCode, message); err != nil {
			return err
		}
		return s.failAction(ctx, action, failureCode, message, retryable)
	}

	switch completed.Method {
	case "SetParameterValues":
		if err := s.recordAttemptCompleted(ctx, action, completed, ActionAttemptSucceeded, "", ""); err != nil {
			return err
		}
		return s.queueRFReadback(ctx, action, completedTaskID, completed.Params)
	case "GetParameterValues":
		if isExpectedBaseline {
			return s.handleRFBaseline(ctx, action, completedTaskID, completed.Params, completed.Result)
		}
		if err := s.verifyRFReadback(action, completed.Params, completed.Result); err != nil {
			if recordErr := s.recordAttemptCompleted(ctx, action, completed, ActionAttemptFailed, "readback_mismatch", err.Error()); recordErr != nil {
				return recordErr
			}
			return s.failAction(ctx, action, "readback_mismatch", err.Error(), true)
		}
		if err := s.recordAttemptCompleted(ctx, action, completed, ActionAttemptSucceeded, "", ""); err != nil {
			return err
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
		completed := completedActionTask(action, baselineTaskID, "GetParameterValues", rfBaselineCommandKey(action), paramsRaw, resultRaw)
		if recordErr := s.recordAttemptCompleted(ctx, action, completed, ActionAttemptFailed, "baseline_invalid", err.Error()); recordErr != nil {
			return recordErr
		}
		return s.failAction(ctx, action, "baseline_invalid", err.Error(), true)
	}
	completed := completedActionTask(action, baselineTaskID, "GetParameterValues", rfBaselineCommandKey(action), paramsRaw, resultRaw)
	if err := s.recordAttemptCompleted(ctx, action, completed, ActionAttemptSucceeded, "", ""); err != nil {
		return err
	}
	if len(paths) == 0 {
		return s.completeAction(ctx, action, true, "")
	}
	if err := s.store.SetRFChangePaths(ctx, action.ID, paths, s.now()); err != nil {
		return fmt.Errorf("persist RF containment baseline: %w", err)
	}
	action.RFChangePaths = paths
	created, err := s.dispatchRFSwitch(ctx, action, false, device.RFSwitchTaskOptions{
		Source:         task.TaskSourceDeviceAccess,
		SourceID:       action.ID.String(),
		Description:    "automatic device access rf_off",
		CommandKey:     rfWriteCommandKey(action),
		AdmissionClass: task.AdmissionClassSecurityAction,
		TargetPaths:    paths,
	})
	if err != nil {
		code, retryable := classifyRFDispatchFailure(err)
		return s.failAction(ctx, action, code, fmt.Sprintf("queue RF containment after baseline: %v", err), retryable)
	}
	if created == nil {
		return s.failAction(ctx, action, "dispatcher_empty", "RF dispatcher returned no task after baseline", true)
	}
	writeTaskID, err := uuid.Parse(created.ID)
	if err != nil {
		return s.failAction(ctx, action, "invalid_task_id", fmt.Sprintf("parse RF containment task id: %v", err), false)
	}
	if err := s.recordAttemptQueued(ctx, action, created, writeTaskID, ActionAttemptSPV); err != nil {
		return err
	}
	if err := s.store.AdvanceDispatchTask(ctx, action.ID, baselineTaskID, writeTaskID, ActionStatusDispatching, s.now()); err != nil {
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
		return s.failAction(ctx, action, "dispatcher_unavailable", "RF readback dispatcher is unavailable", true)
	}
	targetPaths, err := rfWriteTargetPaths(writeParams)
	if err != nil {
		return s.failAction(ctx, action, "invalid_write_contract", err.Error(), false)
	}
	if len(action.RFChangePaths) == 0 || !sameRFPaths(targetPaths, action.RFChangePaths) {
		return s.failAction(ctx, action, "rf_ownership_mismatch", "RF write targets do not match the action-owned paths", false)
	}
	targetPaths = append([]string(nil), action.RFChangePaths...)
	created, err := s.dispatchRFReadback(ctx, action, device.RFSwitchTaskOptions{
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		Description:    "device access RF readback verification",
		CommandKey:     rfReadbackCommandKey(action),
		AdmissionClass: task.AdmissionClassSecurityAction,
		TargetPaths:    targetPaths,
	})
	if err != nil {
		code, retryable := classifyRFDispatchFailure(err)
		return s.failAction(ctx, action, code, fmt.Sprintf("queue RF readback: %v", err), retryable)
	}
	if created == nil {
		return s.failAction(ctx, action, "dispatcher_empty", "RF readback dispatcher returned no task", true)
	}
	readbackTaskID, err := uuid.Parse(created.ID)
	if err != nil {
		return s.failAction(ctx, action, "invalid_task_id", fmt.Sprintf("parse RF readback task id: %v", err), false)
	}
	if err := s.recordAttemptQueued(ctx, action, created, readbackTaskID, ActionAttemptReadbackGPV); err != nil {
		return err
	}
	if writeTaskID != uuid.Nil {
		if err := s.store.AdvanceDispatchTask(ctx, action.ID, writeTaskID, readbackTaskID, ActionStatusVerifying, s.now()); err != nil {
			if errors.Is(err, ErrActionStateChanged) {
				latest, loadErr := s.store.Get(ctx, action.ID)
				if loadErr == nil {
					if isTerminalActionStatus(latest.Status) {
						return nil
					}
					if (latest.Status == ActionStatusDispatching || latest.Status == ActionStatusVerifying) && latest.DeviceTaskID != nil &&
						*latest.DeviceTaskID != writeTaskID {
						return nil // a duplicate completion already correlated readback
					}
				}
			}
			return s.failAction(ctx, action, "task_correlation_failed", fmt.Sprintf("correlate RF readback task: %v", err), true)
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
		path = strings.TrimSpace(path)
		if path == "" {
			return false
		}
		if _, duplicate := want[path]; duplicate {
			return false
		}
		want[path] = struct{}{}
	}
	got := make(map[string]struct{}, len(right))
	for _, path := range right {
		path = strings.TrimSpace(path)
		if path == "" {
			return false
		}
		if _, duplicate := got[path]; duplicate {
			return false
		}
		got[path] = struct{}{}
		if _, ok := want[path]; !ok {
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
	if err := s.store.MarkTerminal(ctx, action.ID, status, owned, "", message, now); err != nil {
		return fmt.Errorf("persist RF action completion: %w", err)
	}
	action.Status = status
	action.OwnedRFChange = owned
	action.ErrorMessage = message
	action.CompletedAt = &now
	return nil
}

func (s *ActionService) failAction(
	ctx context.Context,
	action Action,
	failureCode, message string,
	retryable bool,
) error {
	maxAttempts := action.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = defaultActionMaxAttempts
	}
	dead := !retryable || action.Attempts >= maxAttempts
	nextAttemptAt := s.now().Add(actionRetryDelay(action.Attempts))
	if err := s.store.ScheduleRetry(ctx, action.ID, failureCode, message, nextAttemptAt, dead); err != nil {
		return fmt.Errorf("persist RF action failure: %w", err)
	}
	return nil
}

func actionRetryDelay(attempt int) time.Duration {
	switch {
	case attempt <= 1:
		return 30 * time.Second
	case attempt == 2:
		return 2 * time.Minute
	default:
		return 10 * time.Minute
	}
}

func (s *ActionService) recordAttemptQueued(
	ctx context.Context,
	action Action,
	created *task.Task,
	deviceTaskID uuid.UUID,
	phase ActionAttemptPhase,
) error {
	commandKey := created.CommandKey
	if commandKey == "" {
		switch phase {
		case ActionAttemptBaselineGPV:
			commandKey = rfBaselineCommandKey(action)
		case ActionAttemptSPV:
			commandKey = rfWriteCommandKey(action)
		case ActionAttemptReadbackGPV:
			commandKey = rfReadbackCommandKey(action)
		}
	}
	requestSummary, _ := json.Marshal(map[string]any{
		"phase": phase, "target_path_count": len(action.RFChangePaths),
		"target_paths": append([]string(nil), action.RFChangePaths...),
	})
	if err := s.store.RecordAttemptQueued(ctx, ActionAttempt{
		ActionID: action.ID, AttemptNo: action.Attempts, Phase: phase,
		DeviceTaskID: &deviceTaskID, CommandKey: commandKey, Status: ActionAttemptQueued,
		RequestSummary: requestSummary, StartedAt: s.now(),
	}); err != nil {
		return fmt.Errorf("record queued RF action attempt: %w", err)
	}
	return nil
}

func (s *ActionService) recordAttemptCompleted(
	ctx context.Context,
	action Action,
	completed *task.Task,
	status ActionAttemptStatus,
	failureCode, message string,
) error {
	if completed == nil {
		return nil
	}
	deviceTaskID, err := uuid.Parse(completed.ID)
	if err != nil {
		return fmt.Errorf("parse completed RF attempt task id: %w", err)
	}
	phase := ActionAttemptReadbackGPV
	if completed.Method == "SetParameterValues" {
		phase = ActionAttemptSPV
	} else if completed.CommandKey == rfBaselineCommandKey(action) {
		phase = ActionAttemptBaselineGPV
	}
	faultCode := ""
	if completed.ErrorCode != 0 {
		faultCode = fmt.Sprintf("%d", completed.ErrorCode)
	}
	requestSummary, _ := json.Marshal(map[string]any{
		"method": completed.Method, "command_key": completed.CommandKey,
		"parameters": json.RawMessage(completed.Params),
	})
	responseSummary, _ := json.Marshal(map[string]any{
		"task_status": completed.Status, "error_code": completed.ErrorCode,
		"result": json.RawMessage(completed.Result),
	})
	now := s.now()
	if err := s.store.RecordAttemptCompleted(ctx, ActionAttempt{
		ActionID: action.ID, AttemptNo: action.Attempts, Phase: phase,
		DeviceTaskID: &deviceTaskID, CommandKey: completed.CommandKey, Status: status,
		FaultCode: faultCode, FailureCode: failureCode, ErrorMessage: message,
		RequestSummary: requestSummary, ResponseSummary: responseSummary,
		StartedAt: action.UpdatedAt, CompletedAt: &now,
	}); err != nil {
		return fmt.Errorf("complete RF action attempt: %w", err)
	}
	return nil
}

func completedActionTask(
	action Action,
	taskID uuid.UUID,
	method, commandKey string,
	params, result json.RawMessage,
) *task.Task {
	return &task.Task{
		ID: taskID.String(), DeviceSN: action.SerialNumber, Method: method,
		CommandKey: commandKey, Params: params, Result: result,
		Status: task.TaskStatusCompleted, Source: task.TaskSourceDeviceAccess,
		SourceID: action.ID.String(), AdmissionClass: task.AdmissionClassSecurityAction,
	}
}

func classifyRFTaskFailure(completed *task.Task) (string, bool) {
	if completed == nil {
		return "task_missing", true
	}
	message := strings.ToLower(strings.TrimSpace(completed.ErrorMessage))
	if completed.Status == task.TaskStatusExpired || strings.Contains(message, "timeout") || strings.Contains(message, "超时") {
		return "cwmp_timeout", true
	}
	if strings.Contains(message, "offline") || strings.Contains(message, "离线") {
		return "device_offline", true
	}
	if completed.ErrorCode != 0 {
		code := fmt.Sprintf("spv_fault_%d", completed.ErrorCode)
		return code, completed.ErrorCode == 9002 || completed.ErrorCode == 9004
	}
	if completed.Status == task.TaskStatusCancelled {
		return "task_cancelled", true
	}
	return "cwmp_task_failed", true
}

func actionAttemptFailureStatus(completed *task.Task) ActionAttemptStatus {
	if completed == nil {
		return ActionAttemptFailed
	}
	switch completed.Status {
	case task.TaskStatusExpired:
		return ActionAttemptTimeout
	case task.TaskStatusCancelled:
		return ActionAttemptCancelled
	default:
		return ActionAttemptFailed
	}
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
	return status == ActionStatusSucceeded || status == ActionStatusFailed || status == ActionStatusDead || status == ActionStatusCancelled
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
	if action.Status != ActionStatusDispatching && action.Status != ActionStatusVerifying {
		return false, "security_action_not_dispatching", nil
	}
	if strings.TrimSpace(request.TaskID) != "" {
		taskID, parseErr := uuid.Parse(strings.TrimSpace(request.TaskID))
		if parseErr != nil || action.DeviceTaskID == nil || *action.DeviceTaskID != taskID {
			return false, "security_action_task_superseded", nil
		}
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
	if action.CandidateID != nil && strings.TrimSpace(request.TaskID) != "" {
		if !request.Authenticated || strings.TrimSpace(request.SessionID) == "" || strings.TrimSpace(request.RequestID) == "" {
			return false, "candidate_session_not_authenticated", nil
		}
		if !strings.EqualFold(strings.TrimSpace(action.CandidateOUI), strings.TrimSpace(request.DeviceOUI)) ||
			strings.TrimSpace(action.ProductName) != strings.TrimSpace(request.ProductClass) {
			return false, "candidate_identity_mismatch", nil
		}
		if err := s.store.BindCandidateExecution(
			ctx, action.ID, strings.TrimSpace(request.SessionID), strings.TrimSpace(request.RequestID), s.now(),
		); err != nil {
			return false, "candidate_session_bind_failed", fmt.Errorf("bind candidate security action session: %w", err)
		}
	}
	if request.Method == "SetParameterValues" {
		current, err := s.access.LoadEvaluationContext(ctx, action.Carrier, action.SerialNumber)
		if err != nil {
			return false, "security_action_state_unavailable", fmt.Errorf("load security action access state: %w", err)
		}
		if current.State == nil ||
			!sameActionTarget(current.State.DeviceID, current.State.CandidateID, action.DeviceID, action.CandidateID) ||
			!actionAllowedByState(action.ActionType, current.State.State) {
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
		requestedPaths := make([]string, 0, len(params.Values))
		for _, value := range params.Values {
			if (value.Name != path && !carrier.IsRFControlPath(value.Name) && !device.IsCandidateRFControlPath(value.Name)) ||
				value.Value != expectedValue || value.Type != "xsd:boolean" {
				return false, "invalid_security_action_contract", nil
			}
			requestedPaths = append(requestedPaths, value.Name)
		}
		if len(action.RFChangePaths) > 0 && !sameRFPaths(action.RFChangePaths, requestedPaths) {
			return false, "security_action_target_mismatch", nil
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
		if name != path && !carrier.IsRFControlPath(name) && !device.IsCandidateRFControlPath(name) {
			return false, "invalid_security_action_contract", nil
		}
	}
	if len(action.RFChangePaths) > 0 && !sameRFPaths(action.RFChangePaths, params.Names) {
		return false, "security_action_target_mismatch", nil
	}
	return true, "automatic_security_action_readback", nil
}

func actionAllowedByState(actionType ActionType, state AccessState) bool {
	if actionType == ActionTypeRFOff {
		return state == AccessStateRejected || state == AccessStateRevoked
	}
	return actionType == ActionTypeRFOn && state == AccessStateAccepted
}

func classifyRFDispatchFailure(err error) (string, bool) {
	if errors.Is(err, device.ErrCandidateContainmentUnsupported) {
		return "containment_not_supported", false
	}
	return "dispatch_error", true
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
		handler event.EventHandler
	}{
		{subject: event.SubjectDeviceAccessRejected, queue: "device-access-actions-rejected", handler: c.svc.PlanFromDecision},
		{subject: event.SubjectDeviceAccessRevoked, queue: "device-access-actions-revoked", handler: c.svc.PlanFromDecision},
		{subject: event.SubjectDeviceAccessAccepted, queue: "device-access-actions-accepted", handler: c.svc.PlanFromDecision},
		{subject: event.SubjectDeviceOnline, queue: "device-access-actions-online", handler: c.svc.WakeFromDeviceOnline},
		{subject: event.SubjectDeviceFirmwareChanged, queue: "device-access-actions-firmware-changed", handler: c.svc.WakeFromDeviceOnline},
	} {
		sub, err := c.bus.QueueSubscribe(binding.subject, binding.queue, binding.handler)
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
