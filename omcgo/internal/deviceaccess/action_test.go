package deviceaccess

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/require"
)

type actionStoreFake struct {
	actions   map[uuid.UUID]Action
	attempts  map[uuid.UUID][]ActionAttempt
	due       []uuid.UUID
	recover   []uuid.UUID
	tasks     map[uuid.UUID]*task.Task
	woken     []uuid.UUID
	plans     []ActionPlan
	owned     *Action
	promoted  *uuid.UUID
	onPromote func(uuid.UUID)
	serial    string
	carrier   string
}

func newActionStoreFake(actions ...Action) *actionStoreFake {
	store := &actionStoreFake{
		actions: make(map[uuid.UUID]Action), attempts: make(map[uuid.UUID][]ActionAttempt),
		tasks:  make(map[uuid.UUID]*task.Task),
		serial: "SN-1", carrier: "cmcc",
	}
	for _, action := range actions {
		store.actions[action.ID] = action
	}
	return store
}

func (s *actionStoreFake) Plan(_ context.Context, plan ActionPlan) (Action, bool, error) {
	for _, action := range s.actions {
		if action.IdempotencyKey == plan.IdempotencyKey && action.IdempotencyKey != "" {
			return action, false, nil
		}
	}
	s.plans = append(s.plans, plan)
	action := Action{
		ID: uuid.New(), DeviceID: plan.DeviceID, CandidateID: plan.CandidateID, DecisionID: plan.DecisionID,
		ActionType: plan.ActionType, Direction: plan.Direction,
		RecoveryOfActionID: plan.RecoveryOfActionID, IdempotencyKey: plan.IdempotencyKey,
		Status: ActionStatusPendingDispatch, SerialNumber: s.serial, Carrier: s.carrier,
		Technology: model.TechLTE, MaxAttempts: defaultActionMaxAttempts,
	}
	if plan.CandidateID != nil {
		action.ProductName = "FAP/LTE"
		action.CandidateOUI = "48BF74"
		action.CandidateRFPaths = []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"}
	}
	s.actions[action.ID] = action
	return action, true, nil
}

func (s *actionStoreFake) Get(_ context.Context, id uuid.UUID) (Action, error) {
	return s.actions[id], nil
}

func (s *actionStoreFake) List(context.Context, ActionListFilter) ([]Action, int64, error) {
	items := make([]Action, 0, len(s.actions))
	for _, action := range s.actions {
		items = append(items, action)
	}
	return items, int64(len(items)), nil
}

func (s *actionStoreFake) FindOwnedIsolation(context.Context, uuid.UUID) (*Action, error) {
	return s.owned, nil
}

func (s *actionStoreFake) FindOpenContainment(context.Context, uuid.UUID) (*Action, error) {
	for _, action := range s.actions {
		if action.ActionType == ActionTypeRFOff &&
			(action.Status == ActionStatusPendingDispatch || action.Status == ActionStatusDispatching ||
				action.Status == ActionStatusVerifying || action.Status == ActionStatusRetryWait) {
			copy := action
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *actionStoreFake) FindOpenCandidateContainment(_ context.Context, candidateID uuid.UUID) (*Action, error) {
	for _, action := range s.actions {
		if action.CandidateID != nil && *action.CandidateID == candidateID &&
			action.ActionType == ActionTypeRFOff &&
			(action.Status == ActionStatusPendingDispatch || action.Status == ActionStatusDispatching ||
				action.Status == ActionStatusVerifying || action.Status == ActionStatusRetryWait) {
			copy := action
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *actionStoreFake) PromoteCandidateTarget(_ context.Context, candidateID uuid.UUID, _, _ string) (uuid.UUID, error) {
	if s.promoted == nil {
		return uuid.Nil, ErrActionStateChanged
	}
	for id, action := range s.actions {
		if action.CandidateID != nil && *action.CandidateID == candidateID {
			action.DeviceID = s.promoted
			action.CandidateID = nil
			s.actions[id] = action
		}
	}
	if s.owned != nil && s.owned.CandidateID != nil && *s.owned.CandidateID == candidateID {
		s.owned.DeviceID = s.promoted
		s.owned.CandidateID = nil
	}
	if s.onPromote != nil {
		s.onPromote(*s.promoted)
	}
	return *s.promoted, nil
}

func uuidPointer(value uuid.UUID) *uuid.UUID { return &value }

func (s *actionStoreFake) BeginDispatch(_ context.Context, id uuid.UUID, at time.Time) error {
	action := s.actions[id]
	if action.Status != ActionStatusPendingDispatch {
		return ErrActionStateChanged
	}
	action.Status = ActionStatusDispatching
	action.Attempts++
	action.DispatchedAt = &at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) AttachDispatchTask(_ context.Context, id, taskID uuid.UUID) error {
	action := s.actions[id]
	if action.Status != ActionStatusDispatching || action.DeviceTaskID != nil {
		return ErrActionStateChanged
	}
	action.DeviceTaskID = &taskID
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) SetRFChangePaths(_ context.Context, id uuid.UUID, paths []string, at time.Time) error {
	action := s.actions[id]
	if action.Status != ActionStatusDispatching || len(action.RFChangePaths) != 0 {
		return ErrActionStateChanged
	}
	action.RFChangePaths = append([]string(nil), paths...)
	action.UpdatedAt = at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) AdvanceDispatchTask(
	_ context.Context,
	id, writeTaskID, readbackTaskID uuid.UUID,
	status ActionStatus,
	at time.Time,
) error {
	action := s.actions[id]
	if (action.Status != ActionStatusDispatching && action.Status != ActionStatusVerifying) ||
		(action.DeviceTaskID != nil && *action.DeviceTaskID != writeTaskID) {
		return ErrActionStateChanged
	}
	action.Status = status
	action.DeviceTaskID = &readbackTaskID
	action.UpdatedAt = at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) MarkTerminal(_ context.Context, id uuid.UUID, status ActionStatus, owned bool, failureCode, message string, at time.Time) error {
	action := s.actions[id]
	action.Status = status
	action.OwnedRFChange = owned
	action.ErrorMessage = message
	action.LastFailureCode = failureCode
	action.CompletedAt = &at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) ScheduleRetry(
	_ context.Context,
	id uuid.UUID,
	failureCode, message string,
	next time.Time,
	dead bool,
) error {
	action := s.actions[id]
	action.Status = ActionStatusRetryWait
	if dead {
		action.Status = ActionStatusDead
		action.DeadAt = &next
		action.ManualRepairNeeded = true
	}
	action.DeviceTaskID = nil
	action.BoundSessionID = ""
	action.BoundRequestID = ""
	action.LastFailureCode = failureCode
	action.ErrorMessage = message
	action.NextAttemptAt = &next
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) ResetForManualRetry(_ context.Context, id uuid.UUID, reason string, at time.Time) error {
	if reason == "" {
		return ErrActionStateChanged
	}
	action := s.actions[id]
	if action.Status != ActionStatusFailed && action.Status != ActionStatusDead {
		return ErrActionStateChanged
	}
	action.Status = ActionStatusPendingDispatch
	action.DeviceTaskID = nil
	action.ErrorMessage = ""
	action.MaxAttempts = action.Attempts + defaultActionMaxAttempts
	action.NextAttemptAt = &at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) RecordAttemptQueued(_ context.Context, attempt ActionAttempt) error {
	s.attempts[attempt.ActionID] = append(s.attempts[attempt.ActionID], attempt)
	return nil
}

func (s *actionStoreFake) RecordAttemptCompleted(_ context.Context, attempt ActionAttempt) error {
	s.attempts[attempt.ActionID] = append(s.attempts[attempt.ActionID], attempt)
	return nil
}

func (s *actionStoreFake) ListAttempts(_ context.Context, actionID uuid.UUID) ([]ActionAttempt, error) {
	return append([]ActionAttempt(nil), s.attempts[actionID]...), nil
}

func (s *actionStoreFake) ClaimDue(_ context.Context, now time.Time, _ int) ([]Action, error) {
	items := make([]Action, 0, len(s.due))
	for _, id := range s.due {
		action := s.actions[id]
		action.Status = ActionStatusDispatching
		action.Attempts++
		action.DeviceTaskID = nil
		action.UpdatedAt = now
		s.actions[id] = action
		items = append(items, action)
	}
	s.due = nil
	return items, nil
}

func (s *actionStoreFake) ListRecoverable(context.Context, time.Time, int) ([]Action, error) {
	items := make([]Action, 0, len(s.recover))
	for _, id := range s.recover {
		items = append(items, s.actions[id])
	}
	return items, nil
}

func (s *actionStoreFake) LoadActionTask(_ context.Context, action Action) (*task.Task, error) {
	return s.tasks[action.ID], nil
}

func (s *actionStoreFake) TouchAction(context.Context, uuid.UUID, time.Time) error { return nil }

func (s *actionStoreFake) WakeRetryForDevice(_ context.Context, deviceID uuid.UUID, _ time.Time) error {
	s.woken = append(s.woken, deviceID)
	return nil
}

func (s *actionStoreFake) BindCandidateExecution(_ context.Context, actionID uuid.UUID, sessionID, requestID string, _ time.Time) error {
	action := s.actions[actionID]
	if action.CandidateID == nil {
		return ErrActionStateChanged
	}
	action.BoundSessionID = sessionID
	action.BoundRequestID = requestID
	s.actions[actionID] = action
	return nil
}

type actionRepositoryFake struct {
	Repository
	evaluation EvaluationContext
}

func (f *actionRepositoryFake) LoadEvaluationContext(context.Context, string, string) (EvaluationContext, error) {
	return f.evaluation, nil
}

type rfDispatcherFake struct {
	enabled         bool
	options         device.RFSwitchTaskOptions
	writeTaskID     uuid.UUID
	writeCalls      int
	onWriteQueued   func(uuid.UUID)
	readbackQueued  bool
	readbackOptions device.RFSwitchTaskOptions
	readbackTaskID  uuid.UUID
	readbackCalls   int
	candidateTarget device.CandidateRFTarget
	candidateWrites int
	candidateReads  int
	readbackErr     error
}

func (f *rfDispatcherFake) QueueCandidateRFSwitch(_ context.Context, target device.CandidateRFTarget, enabled bool, options device.RFSwitchTaskOptions) (*task.Task, error) {
	f.candidateTarget = target
	f.candidateWrites++
	return f.QueueRFSwitch(context.Background(), uuid.Nil, enabled, options)
}

func (f *rfDispatcherFake) QueueCandidateRFReadback(_ context.Context, target device.CandidateRFTarget, options device.RFSwitchTaskOptions) (*task.Task, error) {
	f.candidateTarget = target
	f.candidateReads++
	return f.QueueRFReadback(context.Background(), uuid.Nil, options)
}

func (f *rfDispatcherFake) QueueRFSwitch(_ context.Context, _ uuid.UUID, enabled bool, options device.RFSwitchTaskOptions) (*task.Task, error) {
	f.writeCalls++
	f.enabled = enabled
	f.options = options
	if f.writeTaskID == uuid.Nil {
		f.writeTaskID = uuid.New()
	}
	if f.onWriteQueued != nil {
		f.onWriteQueued(f.writeTaskID)
	}
	return &task.Task{ID: f.writeTaskID.String()}, nil
}

func (f *rfDispatcherFake) QueueRFReadback(_ context.Context, _ uuid.UUID, options device.RFSwitchTaskOptions) (*task.Task, error) {
	f.readbackCalls++
	if f.readbackErr != nil {
		return nil, f.readbackErr
	}
	f.readbackQueued = true
	f.readbackOptions = options
	f.readbackTaskID = uuid.New()
	return &task.Task{ID: f.readbackTaskID.String()}, nil
}

func TestRejectedCandidateUnsupportedRFContainmentIsTerminal(t *testing.T) {
	candidateID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateRejected, DecisionVersion: 1,
	}}}
	dispatcher := &rfDispatcherFake{readbackErr: device.ErrCandidateContainmentUnsupported}
	service := NewActionService(store, dispatcher, repo, nil, nil)

	err := service.PlanFromDecision(context.Background(), rejectedCandidateEvent(t, candidateID, uuid.New(), 1))
	require.ErrorIs(t, err, device.ErrCandidateContainmentUnsupported)
	require.Len(t, store.actions, 1)
	for _, action := range store.actions {
		require.Equal(t, ActionStatusDead, action.Status)
		require.Equal(t, "containment_not_supported", action.LastFailureCode)
		require.False(t, action.OwnedRFChange)
	}
}

func TestCandidateRFInternalBindingIsNotExposedByActionAPI(t *testing.T) {
	candidateID := uuid.New()
	payload, err := json.Marshal(Action{
		ID: uuid.New(), CandidateID: &candidateID,
		CandidateRFPaths: []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"},
		CandidateOUI:     "48BF74", BoundSessionID: "session-secret", BoundRequestID: "request-secret",
	})
	require.NoError(t, err)
	require.NotContains(t, string(payload), "candidate_rf_paths")
	require.NotContains(t, string(payload), "candidate_oui")
	require.NotContains(t, string(payload), "session-secret")
	require.NotContains(t, string(payload), "request-secret")
}

func rejectedEvent(t *testing.T, deviceID, decisionID uuid.UUID, version int64) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectDeviceAccessRejected, decisionActionEvent{
		DecisionID: decisionID, DeviceID: &deviceID, Carrier: "cmcc", SerialNumber: "SN-1",
		State: AccessStateRejected, DecisionVersion: version,
	})
	require.NoError(t, err)
	return evt
}

func rejectedCandidateEvent(t *testing.T, candidateID, decisionID uuid.UUID, version int64) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectDeviceAccessRejected, decisionActionEvent{
		DecisionID: decisionID, CandidateID: &candidateID, Carrier: "cmcc", SerialNumber: "SN-1",
		State: AccessStateRejected, DecisionVersion: version,
	})
	require.NoError(t, err)
	return evt
}

func TestActionServiceRejectedCandidateDispatchesRestrictedRFContainment(t *testing.T) {
	candidateID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateRejected, DecisionVersion: 1,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)

	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedCandidateEvent(t, candidateID, uuid.New(), 1)))
	require.Len(t, store.plans, 1)
	require.Nil(t, store.plans[0].DeviceID)
	require.Equal(t, candidateID, *store.plans[0].CandidateID)
	require.Equal(t, 1, dispatcher.candidateReads)
	require.Zero(t, dispatcher.candidateWrites)
	require.Equal(t, candidateID, dispatcher.candidateTarget.CandidateID)
	require.Equal(t, "SN-1", dispatcher.candidateTarget.SerialNumber)
}

func TestRejectedCandidateBaselineAdvancesToGuardedRFOff(t *testing.T) {
	candidateID := uuid.New()
	path := "Device.Services.FAPService.1.FAPControl.LTE.AdminState"
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateRejected, DecisionVersion: 1,
	}}}
	dispatcher := &rfDispatcherFake{}
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(store, dispatcher, repo, registry, nil)
	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedCandidateEvent(t, candidateID, uuid.New(), 1)))

	var action Action
	for _, item := range store.actions {
		action = item
	}
	params, _ := json.Marshal(map[string]any{"names": []string{path}})
	result, _ := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{{
		"name": path, "value": "true",
	}}})
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: dispatcher.readbackTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "GetParameterValues", DeviceSN: action.SerialNumber,
		CommandKey: rfBaselineCommandKey(action), Params: params, Result: result,
	}))
	require.Equal(t, 1, dispatcher.candidateWrites)
	require.False(t, dispatcher.enabled)
	require.Equal(t, []string{path}, dispatcher.options.TargetPaths)

	writeParams, _ := json.Marshal(map[string]any{"values": []map[string]string{{
		"name": path, "value": "0", "type": "xsd:boolean",
	}}})
	allowed, reason, err := service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		TaskID: dispatcher.writeTaskID.String(), DeviceSN: action.SerialNumber,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		Method: "SetParameterValues", Params: writeParams,
		SessionID: "session-1", RequestID: "request-1", DeviceOUI: "48BF74",
		ProductClass: "FAP/LTE", Authenticated: true,
	})
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "automatic_security_action", reason)
	bound := store.actions[action.ID]
	require.Equal(t, "session-1", bound.BoundSessionID)
	require.Equal(t, "request-1", bound.BoundRequestID)
}

func TestRejectedCandidateSecurityActionRejectsDifferentInformIdentity(t *testing.T) {
	candidateID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateRejected, DecisionVersion: 1,
	}}}
	dispatcher := &rfDispatcherFake{}
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(store, dispatcher, repo, registry, nil)
	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedCandidateEvent(t, candidateID, uuid.New(), 1)))

	var action Action
	for _, item := range store.actions {
		action = item
	}
	params, _ := json.Marshal(map[string]any{"names": []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"}})
	allowed, reason, err := service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		TaskID: dispatcher.readbackTaskID.String(), DeviceSN: action.SerialNumber,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(), Method: "GetParameterValues", Params: params,
		SessionID: "session-2", RequestID: "request-2", DeviceOUI: "FFFF00",
		ProductClass: "FAP/LTE", Authenticated: true,
	})
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "candidate_identity_mismatch", reason)
}

func TestActionServiceAcceptedCandidateWaitsForFormalRegistration(t *testing.T) {
	candidateID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateAccepted, DecisionVersion: 2,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)
	evt, err := event.NewEvent(event.SubjectDeviceAccessAccepted, decisionActionEvent{
		DecisionID: uuid.New(), CandidateID: &candidateID, Carrier: "cmcc", SerialNumber: "SN-1",
		State: AccessStateAccepted, DecisionVersion: 2,
	})
	require.NoError(t, err)

	require.Error(t, service.PlanFromDecision(context.Background(), evt))
	require.Empty(t, store.plans)
	require.Zero(t, dispatcher.candidateReads)
	require.Zero(t, dispatcher.candidateWrites)
}

func TestAcceptedRegisteredCandidateRecoversOnlyOwnedIsolation(t *testing.T) {
	candidateID, deviceID := uuid.New(), uuid.New()
	owned := Action{
		ID: uuid.New(), CandidateID: &candidateID, ActionType: ActionTypeRFOff,
		Status: ActionStatusSucceeded, OwnedRFChange: true,
		RFChangePaths: []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"},
		SerialNumber:  "SN-1", Carrier: "cmcc", Technology: model.TechLTE,
	}
	store := newActionStoreFake(owned)
	store.owned = &owned
	store.promoted = &deviceID
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateAccepted, DecisionVersion: 3,
	}}}
	store.onPromote = func(promoted uuid.UUID) {
		repo.evaluation.State.DeviceID = &promoted
		repo.evaluation.State.CandidateID = nil
	}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)
	evt, err := event.NewEvent(event.SubjectDeviceAccessAccepted, decisionActionEvent{
		DecisionID: uuid.New(), CandidateID: &candidateID, Carrier: "cmcc", SerialNumber: "SN-1",
		State: AccessStateAccepted, DecisionVersion: 3,
	})
	require.NoError(t, err)

	require.NoError(t, service.PlanFromDecision(context.Background(), evt))
	require.Len(t, store.plans, 1)
	require.Equal(t, ActionTypeRFOn, store.plans[0].ActionType)
	require.Equal(t, deviceID, *store.plans[0].DeviceID)
	require.True(t, dispatcher.enabled)
	require.Equal(t, owned.RFChangePaths, dispatcher.options.TargetPaths)
}

func TestActionServiceRejectedDecisionAutomaticallyDispatchesRFOff(t *testing.T) {
	deviceID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected, DecisionVersion: 1,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)

	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedEvent(t, deviceID, uuid.New(), 1)))
	require.Len(t, store.plans, 1)
	require.Zero(t, dispatcher.writeCalls)
	require.Equal(t, 1, dispatcher.readbackCalls)
	require.Equal(t, task.TaskSourceDeviceAccess, dispatcher.readbackOptions.Source)
	require.Equal(t, task.AdmissionClassSecurityAction, dispatcher.readbackOptions.AdmissionClass)
	require.Contains(t, dispatcher.readbackOptions.CommandKey, ":baseline")
	for _, action := range store.actions {
		require.Equal(t, ActionStatusDispatching, action.Status)
		require.Equal(t, 1, action.Attempts)
		require.NotNil(t, action.DeviceTaskID)
	}
}

func TestActionServiceNewRejectedDecisionRevalidatesSucceededContainment(t *testing.T) {
	deviceID := uuid.New()
	previous := Action{
		ID: uuid.New(), DeviceID: &deviceID, DecisionID: uuid.New(), ActionType: ActionTypeRFOff,
		Direction: ActionDirectionContain, Status: ActionStatusSucceeded, OwnedRFChange: true,
		RFChangePaths: []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"},
		SerialNumber:  "SN-1", Carrier: "cmcc", Technology: model.TechLTE,
	}
	store := newActionStoreFake(previous)
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected, DecisionVersion: 2,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)

	newDecisionID := uuid.New()
	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedEvent(t, deviceID, newDecisionID, 2)))
	require.Len(t, store.plans, 1)
	require.Equal(t, newDecisionID, store.plans[0].DecisionID)
	require.Equal(t, 1, dispatcher.readbackCalls)
	require.Zero(t, dispatcher.writeCalls)
}

func TestActionServiceResumesDispatchingActionWithoutDuplicateAttempt(t *testing.T) {
	deviceID := uuid.New()
	action := Action{ID: uuid.New(), DeviceID: &deviceID, DecisionID: uuid.New(), ActionType: ActionTypeRFOff,
		Direction: ActionDirectionContain, Status: ActionStatusDispatching, Attempts: 1,
		SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE}
	store := newActionStoreFake(action)
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected, DecisionVersion: 2,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)

	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedEvent(t, deviceID, uuid.New(), 2)))
	require.Equal(t, 0, dispatcher.writeCalls)
	require.Equal(t, 1, dispatcher.readbackCalls)
	require.Equal(t, 1, store.actions[action.ID].Attempts)
	require.Contains(t, dispatcher.readbackOptions.CommandKey, ":1:baseline")
}

func TestActionServiceDoesNotPlanRFWhenSwitchDisabledOrEventStale(t *testing.T) {
	deviceID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected, DecisionVersion: 2,
	}}}
	service := NewActionService(store, &rfDispatcherFake{}, repo, nil, nil)
	service.SetRuntimeSettingsReader(&runtimeSettingsStub{enabled: false})
	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedEvent(t, deviceID, uuid.New(), 2)))
	require.Empty(t, store.plans)

	service.SetRuntimeSettingsReader(nil)
	require.NoError(t, service.PlanFromDecision(context.Background(), rejectedEvent(t, deviceID, uuid.New(), 1)))
	require.Empty(t, store.plans)
}

func TestActionServiceAcceptedDecisionRestoresOnlyOwnedIsolation(t *testing.T) {
	deviceID := uuid.New()
	store := newActionStoreFake()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateAccepted, DecisionVersion: 3,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)
	evt, err := event.NewEvent(event.SubjectDeviceAccessAccepted, decisionActionEvent{
		DecisionID: uuid.New(), DeviceID: &deviceID, Carrier: "cmcc", SerialNumber: "SN-1",
		State: AccessStateAccepted, DecisionVersion: 3,
	})
	require.NoError(t, err)
	require.NoError(t, service.PlanFromDecision(context.Background(), evt))
	require.Empty(t, store.plans)

	owned := Action{ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusSucceeded, OwnedRFChange: true,
		RFChangePaths: []string{"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable"}}
	store.owned = &owned
	store.actions[owned.ID] = owned
	require.NoError(t, service.PlanFromDecision(context.Background(), evt))
	require.Len(t, store.plans, 1)
	require.Equal(t, ActionTypeRFOn, store.plans[0].ActionType)
	require.Equal(t, &owned.ID, store.plans[0].RecoveryOfActionID)
	require.True(t, dispatcher.enabled)
	require.Equal(t, owned.RFChangePaths, dispatcher.options.TargetPaths)
}

func TestActionManualRetryReopensFailedOrDeadActionWithFreshBudget(t *testing.T) {
	deviceID := uuid.New()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected,
	}}}
	exhausted := Action{ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusFailed, Attempts: defaultActionMaxAttempts, SerialNumber: "SN-1", Carrier: "cmcc"}
	store := newActionStoreFake(exhausted)
	service := NewActionService(store, &rfDispatcherFake{}, repo, nil, nil)
	require.NoError(t, service.Retry(context.Background(), exhausted.ID))
	require.Equal(t, ActionStatusDispatching, store.actions[exhausted.ID].Status)
	require.Equal(t, 4, store.actions[exhausted.ID].Attempts)

	retryable := Action{ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusFailed, Attempts: 1, SerialNumber: "SN-1", Carrier: "cmcc"}
	store.actions[retryable.ID] = retryable
	require.NoError(t, service.Retry(context.Background(), retryable.ID))
	require.Equal(t, ActionStatusDispatching, store.actions[retryable.ID].Status)
	require.Equal(t, 2, store.actions[retryable.ID].Attempts)
}

func TestSecurityActionIsRecheckedAndVerifiedByReadback(t *testing.T) {
	deviceID, writeTaskID := uuid.New(), uuid.New()
	action := Action{ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, Attempts: 1, SerialNumber: "SN-2", Carrier: "cmcc",
		Technology: model.TechLTE, DeviceTaskID: &writeTaskID,
		RFChangePaths: []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"}}
	store := newActionStoreFake(action)
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected,
	}}}
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, registry, nil)
	params, err := json.Marshal(map[string]any{"values": []map[string]string{{
		"name": "Device.Services.FAPService.1.FAPControl.LTE.AdminState", "value": "0", "type": "xsd:boolean",
	}}})
	require.NoError(t, err)
	allowed, reason, err := service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-2", Method: "SetParameterValues", Params: params,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
	})
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "automatic_security_action", reason)

	wrongPathParams, err := json.Marshal(map[string]any{"values": []map[string]string{{
		"name": "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", "value": "0", "type": "xsd:boolean",
	}}})
	require.NoError(t, err)
	allowed, reason, err = service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-2", Method: "SetParameterValues", Params: wrongPathParams,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
	})
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "security_action_target_mismatch", reason)

	repo.evaluation.State.State = AccessStateAccepted
	allowed, reason, err = service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-2", Method: "SetParameterValues", Params: params,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
	})
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "security_action_state_changed", reason)
	repo.evaluation.State.State = AccessStateRejected

	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: writeTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "SetParameterValues", CommandKey: rfWriteCommandKey(action), DeviceSN: "SN-2", Params: params,
	}))
	require.True(t, dispatcher.readbackQueued)
	readbackParams, err := json.Marshal(map[string]any{"names": []string{
		"Device.Services.FAPService.1.FAPControl.LTE.AdminState",
	}})
	require.NoError(t, err)
	service.SetRuntimeSettingsReader(&runtimeSettingsStub{enabled: false})
	repo.evaluation.State.State = AccessStateAccepted
	allowed, reason, err = service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-2", Method: "GetParameterValues", Params: readbackParams,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
	})
	require.NoError(t, err)
	require.True(t, allowed, "in-flight GPV must remain readable after the switch or decision changes")
	require.Equal(t, "automatic_security_action_readback", reason)
	result, err := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{{
		"name": "Device.Services.FAPService.1.FAPControl.LTE.AdminState", "value": "false",
	}}})
	require.NoError(t, err)
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: dispatcher.readbackTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "GetParameterValues", CommandKey: rfReadbackCommandKey(action), DeviceSN: "SN-2", Result: result,
	}))
	require.Equal(t, ActionStatusSucceeded, store.actions[action.ID].Status)
	require.True(t, store.actions[action.ID].OwnedRFChange)
}

func TestSameRFPathsRequiresAnExactUniqueSet(t *testing.T) {
	require.True(t, sameRFPaths([]string{"rf-a", "rf-b"}, []string{"rf-b", "rf-a"}))
	require.False(t, sameRFPaths([]string{"rf-a", "rf-b"}, []string{"rf-a", "rf-a"}))
	require.False(t, sameRFPaths([]string{"rf-a", "rf-a"}, []string{"rf-a", "rf-a"}))
	require.False(t, sameRFPaths([]string{"rf-a", ""}, []string{"rf-a", ""}))
}

func TestActionServiceBaselineDoesNotOwnAlreadyDisabledRF(t *testing.T) {
	deviceID, baselineTaskID := uuid.New(), uuid.New()
	paths := []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
	}
	action := Action{ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, Attempts: 1, SerialNumber: "QRTB", Carrier: "cmcc",
		Technology: model.TechLTE, DeviceTaskID: &baselineTaskID}
	store := newActionStoreFake(action)
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, nil, nil, nil)
	params, _ := json.Marshal(map[string]any{"names": paths})
	result, _ := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{
		{"name": paths[0], "value": "false"}, {"name": paths[1], "value": "0"},
	}})
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: baselineTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "GetParameterValues", DeviceSN: action.SerialNumber,
		CommandKey: rfBaselineCommandKey(action), Params: params, Result: result,
	}))
	got := store.actions[action.ID]
	require.Equal(t, ActionStatusSucceeded, got.Status)
	require.False(t, got.OwnedRFChange)
	require.Empty(t, got.RFChangePaths)
	require.Zero(t, dispatcher.writeCalls)
}

func TestActionServiceBaselinePinsMixedMultiInstanceWriteAndReadback(t *testing.T) {
	deviceID, baselineTaskID := uuid.New(), uuid.New()
	paths := []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
	}
	action := Action{ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, Attempts: 1, SerialNumber: "QRTB", Carrier: "cmcc",
		Technology: model.TechLTE, DeviceTaskID: &baselineTaskID}
	store := newActionStoreFake(action)
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, nil, nil, nil)
	params, _ := json.Marshal(map[string]any{"names": paths})
	result, _ := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{
		{"name": paths[0], "value": "true"}, {"name": paths[1], "value": "false"},
	}})
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: baselineTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "GetParameterValues", DeviceSN: action.SerialNumber,
		CommandKey: rfBaselineCommandKey(action), Params: params, Result: result,
	}))
	require.Equal(t, []string{paths[0]}, store.actions[action.ID].RFChangePaths)
	require.Equal(t, []string{paths[0]}, dispatcher.options.TargetPaths)
	writeParams, _ := json.Marshal(map[string]any{"values": []map[string]string{{"name": paths[0], "value": "0"}}})
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: dispatcher.writeTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "SetParameterValues", DeviceSN: action.SerialNumber,
		CommandKey: rfWriteCommandKey(action), Params: writeParams,
	}))
	require.Equal(t, []string{paths[0]}, dispatcher.readbackOptions.TargetPaths)
}

func TestActionServiceReadbackMismatchFailsWithoutOwningRF(t *testing.T) {
	deviceTaskID := uuid.New()
	action := Action{ID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusVerifying, DeviceTaskID: &deviceTaskID, SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE}
	store := newActionStoreFake(action)
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(store, &rfDispatcherFake{}, nil, registry, nil)
	result, err := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{{
		"name": "Device.Services.FAPService.1.FAPControl.LTE.AdminState", "value": "true",
	}}})
	require.NoError(t, err)
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: deviceTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "GetParameterValues", CommandKey: rfReadbackCommandKey(action), DeviceSN: action.SerialNumber, Result: result,
	}))
	require.Equal(t, ActionStatusRetryWait, store.actions[action.ID].Status)
	require.Equal(t, "readback_mismatch", store.actions[action.ID].LastFailureCode)
	require.False(t, store.actions[action.ID].OwnedRFChange)
}

func TestActionServiceIgnoresCompletionFromSupersededTaskWithSameCommandKey(t *testing.T) {
	currentTaskID, staleTaskID := uuid.New(), uuid.New()
	action := Action{
		ID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, Attempts: 2, MaxAttempts: 3,
		DeviceTaskID: &currentTaskID, SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE,
	}
	store := newActionStoreFake(action)
	service := NewActionService(store, &rfDispatcherFake{}, nil, nil, nil)

	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: staleTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusExpired,
		Method: "SetParameterValues", CommandKey: rfWriteCommandKey(action), DeviceSN: action.SerialNumber,
		ErrorMessage: "late timeout from superseded task",
	}))

	got := store.actions[action.ID]
	require.Equal(t, ActionStatusDispatching, got.Status)
	require.Equal(t, currentTaskID, *got.DeviceTaskID)
	require.Empty(t, got.LastFailureCode)
}

func TestSecurityActionAdmissionRejectsSupersededDurableTask(t *testing.T) {
	currentTaskID := uuid.New()
	action := Action{
		ID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, DeviceTaskID: &currentTaskID,
		SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE,
	}
	store := newActionStoreFake(action)
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: action.DeviceID, State: AccessStateRejected,
	}}}
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(store, &rfDispatcherFake{}, repo, registry, nil)
	params, err := json.Marshal(map[string]any{"values": []map[string]string{{
		"name": "Device.Services.FAPService.1.FAPControl.LTE.AdminState", "value": "0", "type": "xsd:boolean",
	}}})
	require.NoError(t, err)

	allowed, reason, err := service.AuthorizeSecurityAction(context.Background(), task.TaskAdmissionRequest{
		TaskID: uuid.NewString(), DeviceSN: action.SerialNumber, AdmissionClass: task.AdmissionClassSecurityAction,
		Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(), Method: "SetParameterValues", Params: params,
	})

	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "security_action_task_superseded", reason)
}

func TestActionFailureUsesBackoffAndMovesToDeadAtLimit(t *testing.T) {
	deviceTaskID := uuid.New()
	action := Action{
		ID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, Attempts: 1, MaxAttempts: 3,
		DeviceTaskID: &deviceTaskID, SerialNumber: "SN-1", Carrier: "cmcc",
	}
	store := newActionStoreFake(action)
	service := NewActionService(store, &rfDispatcherFake{}, nil, nil, nil)
	now := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: deviceTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusExpired,
		Method: "SetParameterValues", CommandKey: rfWriteCommandKey(action), DeviceSN: action.SerialNumber, ErrorMessage: "CWMP timeout",
	}))
	got := store.actions[action.ID]
	require.Equal(t, ActionStatusRetryWait, got.Status)
	require.Equal(t, "cwmp_timeout", got.LastFailureCode)
	require.Equal(t, now.Add(30*time.Second), *got.NextAttemptAt)

	secondTaskID := uuid.New()
	got.Status = ActionStatusDispatching
	got.Attempts = 3
	got.DeviceTaskID = &secondTaskID
	store.actions[action.ID] = got
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: secondTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusFailed,
		Method: "SetParameterValues", CommandKey: rfWriteCommandKey(got), DeviceSN: action.SerialNumber,
		ErrorCode: 9002, ErrorMessage: "internal error",
	}))
	got = store.actions[action.ID]
	require.Equal(t, ActionStatusDead, got.Status)
	require.True(t, got.ManualRepairNeeded)
	require.Equal(t, "spv_fault_9002", got.LastFailureCode)
}

func TestActionNonRetryableSPVFaultMovesDirectlyToDead(t *testing.T) {
	deviceTaskID := uuid.New()
	action := Action{
		ID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, Attempts: 1, MaxAttempts: 3,
		DeviceTaskID: &deviceTaskID, SerialNumber: "SN-1", Carrier: "cmcc",
	}
	store := newActionStoreFake(action)
	service := NewActionService(store, &rfDispatcherFake{}, nil, nil, nil)
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: deviceTaskID.String(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusFailed,
		Method: "SetParameterValues", CommandKey: rfWriteCommandKey(action), DeviceSN: action.SerialNumber,
		ErrorCode: 9005, ErrorMessage: "invalid parameter name",
	}))
	require.Equal(t, ActionStatusDead, store.actions[action.ID].Status)
	require.Equal(t, "spv_fault_9005", store.actions[action.ID].LastFailureCode)
}

func TestDeviceOnlineWakesDueRetryAndStartsFreshAttempt(t *testing.T) {
	deviceID := uuid.New()
	action := Action{
		ID: uuid.New(), DeviceID: &deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusRetryWait, Attempts: 1, MaxAttempts: 3,
		SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE,
	}
	store := newActionStoreFake(action)
	store.due = []uuid.UUID{action.ID}
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected,
	}}}
	dispatcher := &rfDispatcherFake{}
	service := NewActionService(store, dispatcher, repo, nil, nil)
	evt, err := event.NewEvent(event.SubjectDeviceOnline, map[string]any{"device_id": deviceID})
	require.NoError(t, err)
	require.NoError(t, service.WakeFromDeviceOnline(context.Background(), evt))
	require.Equal(t, []uuid.UUID{deviceID}, store.woken)
	require.Equal(t, ActionStatusDispatching, store.actions[action.ID].Status)
	require.Equal(t, 2, store.actions[action.ID].Attempts)
	require.Equal(t, 1, dispatcher.readbackCalls)
}

func TestRecoverStaleVerifyingActionProjectsDurableReadback(t *testing.T) {
	deviceTaskID := uuid.New()
	path := "Device.Services.FAPService.1.FAPControl.LTE.AdminState"
	action := Action{
		ID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusVerifying, Attempts: 1, MaxAttempts: 3,
		DeviceTaskID: &deviceTaskID, RFChangePaths: []string{path},
		SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE,
	}
	params, _ := json.Marshal(map[string]any{"names": []string{path}})
	result, _ := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{{
		"name": path, "value": "false",
	}}})
	store := newActionStoreFake(action)
	store.recover = []uuid.UUID{action.ID}
	store.tasks[action.ID] = &task.Task{
		ID: deviceTaskID.String(), DeviceSN: action.SerialNumber, Method: "GetParameterValues",
		Params: params, Result: result, CommandKey: rfReadbackCommandKey(action),
		Status: task.TaskStatusCompleted, Source: task.TaskSourceDeviceAccess,
		SourceID: action.ID.String(), AdmissionClass: task.AdmissionClassSecurityAction,
	}
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(store, &rfDispatcherFake{}, nil, registry, nil)
	require.NoError(t, service.RecoverStale(context.Background(), time.Now(), 20))
	require.Equal(t, ActionStatusSucceeded, store.actions[action.ID].Status)
	require.True(t, store.actions[action.ID].OwnedRFChange)
}

func TestVerifyRFReadbackRequiresEveryWrittenTarget(t *testing.T) {
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(nil, nil, nil, registry, nil)
	action := Action{ActionType: ActionTypeRFOff, Carrier: "cmcc", Technology: model.TechLTE}
	paths := []string{
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus",
	}
	params, err := json.Marshal(map[string]any{"names": paths})
	require.NoError(t, err)
	partial, err := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{{
		"name": paths[0], "value": "0",
	}}})
	require.NoError(t, err)
	err = service.verifyRFReadback(action, params, partial)
	require.ErrorContains(t, err, paths[1]+" is missing")

	complete, err := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{
		{"name": paths[0], "value": "0"},
		{"name": paths[1], "value": "false"},
	}})
	require.NoError(t, err)
	require.NoError(t, service.verifyRFReadback(action, params, complete))
}

func TestRFWriteTargetPathsPreservesAllUniqueSPVTargets(t *testing.T) {
	raw := json.RawMessage(`{"values":[
		{"name":"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus","value":"0"},
		{"name":"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus","value":"0"},
		{"name":"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus","value":"0"}
	]}`)
	paths, err := rfWriteTargetPaths(raw)
	require.NoError(t, err)
	require.Equal(t, []string{
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus",
	}, paths)
}

type actionConsumerSubscriptionFake struct{ unsubscribed bool }

func (s *actionConsumerSubscriptionFake) Unsubscribe() error { s.unsubscribed = true; return nil }

type actionConsumerBusFake struct {
	subjects []string
	queues   []string
	subs     []*actionConsumerSubscriptionFake
}

func (b *actionConsumerBusFake) Publish(context.Context, string, event.Event) error { return nil }
func (b *actionConsumerBusFake) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}
func (b *actionConsumerBusFake) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}
func (b *actionConsumerBusFake) QueueSubscribe(subject, queue string, _ event.EventHandler) (event.Subscription, error) {
	sub := &actionConsumerSubscriptionFake{}
	b.subjects = append(b.subjects, subject)
	b.queues = append(b.queues, queue)
	b.subs = append(b.subs, sub)
	return sub, nil
}
func (b *actionConsumerBusFake) Close() error { return nil }

func TestActionConsumerSubscribesDecisionsAndOnlineWakeUp(t *testing.T) {
	bus := &actionConsumerBusFake{}
	consumer := NewActionConsumer(bus, &ActionService{})
	require.NoError(t, consumer.Start())
	require.Equal(t, []string{
		event.SubjectDeviceAccessRejected, event.SubjectDeviceAccessRevoked, event.SubjectDeviceAccessAccepted,
		event.SubjectDeviceOnline, event.SubjectDeviceFirmwareChanged,
	}, bus.subjects)
	require.NoError(t, consumer.Stop())
	for _, sub := range bus.subs {
		require.True(t, sub.unsubscribed)
	}
}
