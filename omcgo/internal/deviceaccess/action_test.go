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
	actions map[uuid.UUID]Action
	plans   []ActionPlan
	owned   *Action
	serial  string
	carrier string
}

func newActionStoreFake(actions ...Action) *actionStoreFake {
	store := &actionStoreFake{actions: make(map[uuid.UUID]Action), serial: "SN-1", carrier: "cmcc"}
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
		ID: uuid.New(), DeviceID: plan.DeviceID, DecisionID: plan.DecisionID,
		ActionType: plan.ActionType, Direction: plan.Direction,
		RecoveryOfActionID: plan.RecoveryOfActionID, IdempotencyKey: plan.IdempotencyKey,
		Status: ActionStatusPendingDispatch, SerialNumber: s.serial, Carrier: s.carrier,
		Technology: model.TechLTE,
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
		if action.ActionType == ActionTypeRFOff && action.Status != ActionStatusCancelled &&
			(action.Status != ActionStatusFailed || action.Attempts < defaultActionMaxAttempts) {
			copy := action
			return &copy, nil
		}
	}
	return nil, nil
}

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

func (s *actionStoreFake) MarkVerifying(_ context.Context, id, writeTaskID, readbackTaskID uuid.UUID, at time.Time) error {
	action := s.actions[id]
	if action.Status != ActionStatusDispatching ||
		(action.DeviceTaskID != nil && *action.DeviceTaskID != writeTaskID) {
		return ErrActionStateChanged
	}
	action.DeviceTaskID = &readbackTaskID
	action.UpdatedAt = at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) MarkTerminal(_ context.Context, id uuid.UUID, status ActionStatus, owned bool, message string, at time.Time) error {
	action := s.actions[id]
	action.Status = status
	action.OwnedRFChange = owned
	action.ErrorMessage = message
	action.CompletedAt = &at
	s.actions[id] = action
	return nil
}

func (s *actionStoreFake) ResetForRetry(_ context.Context, id uuid.UUID) error {
	action := s.actions[id]
	if action.Status != ActionStatusFailed {
		return ErrActionStateChanged
	}
	action.Status = ActionStatusPendingDispatch
	action.ErrorMessage = ""
	action.DeviceTaskID = nil
	s.actions[id] = action
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
	f.readbackQueued = true
	f.readbackOptions = options
	f.readbackTaskID = uuid.New()
	return &task.Task{ID: f.readbackTaskID.String()}, nil
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

func TestActionServiceResumesDispatchingActionWithoutDuplicateAttempt(t *testing.T) {
	deviceID := uuid.New()
	action := Action{ID: uuid.New(), DeviceID: deviceID, DecisionID: uuid.New(), ActionType: ActionTypeRFOff,
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

	owned := Action{ID: uuid.New(), DeviceID: deviceID, ActionType: ActionTypeRFOff,
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

func TestActionRetryDispatchesAndEnforcesAttemptLimit(t *testing.T) {
	deviceID := uuid.New()
	repo := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected,
	}}}
	exhausted := Action{ID: uuid.New(), DeviceID: deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusFailed, Attempts: defaultActionMaxAttempts, SerialNumber: "SN-1", Carrier: "cmcc"}
	store := newActionStoreFake(exhausted)
	service := NewActionService(store, &rfDispatcherFake{}, repo, nil, nil)
	require.ErrorIs(t, service.Retry(context.Background(), exhausted.ID), ErrActionRetryExhausted)

	retryable := Action{ID: uuid.New(), DeviceID: deviceID, ActionType: ActionTypeRFOff,
		Status: ActionStatusFailed, Attempts: 1, SerialNumber: "SN-1", Carrier: "cmcc"}
	store.actions[retryable.ID] = retryable
	require.NoError(t, service.Retry(context.Background(), retryable.ID))
	require.Equal(t, ActionStatusDispatching, store.actions[retryable.ID].Status)
	require.Equal(t, 2, store.actions[retryable.ID].Attempts)
}

func TestSecurityActionIsRecheckedAndVerifiedByReadback(t *testing.T) {
	deviceID, writeTaskID := uuid.New(), uuid.New()
	action := Action{ID: uuid.New(), DeviceID: deviceID, ActionType: ActionTypeRFOff,
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
		Method: "SetParameterValues", DeviceSN: "SN-2", Params: params,
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
		Method: "GetParameterValues", DeviceSN: "SN-2", Result: result,
	}))
	require.Equal(t, ActionStatusSucceeded, store.actions[action.ID].Status)
	require.True(t, store.actions[action.ID].OwnedRFChange)
}

func TestActionServiceBaselineDoesNotOwnAlreadyDisabledRF(t *testing.T) {
	deviceID, baselineTaskID := uuid.New(), uuid.New()
	paths := []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
	}
	action := Action{ID: uuid.New(), DeviceID: deviceID, ActionType: ActionTypeRFOff,
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
	action := Action{ID: uuid.New(), DeviceID: deviceID, ActionType: ActionTypeRFOff,
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
	action := Action{ID: uuid.New(), DeviceID: uuid.New(), ActionType: ActionTypeRFOff,
		Status: ActionStatusDispatching, SerialNumber: "SN-1", Carrier: "cmcc", Technology: model.TechLTE}
	store := newActionStoreFake(action)
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	service := NewActionService(store, &rfDispatcherFake{}, nil, registry, nil)
	result, err := json.Marshal(map[string]any{"standard_parameter_values": []map[string]string{{
		"name": "Device.Services.FAPService.1.FAPControl.LTE.AdminState", "value": "true",
	}}})
	require.NoError(t, err)
	require.NoError(t, service.HandleCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), Source: task.TaskSourceDeviceAccess, SourceID: action.ID.String(),
		AdmissionClass: task.AdmissionClassSecurityAction, Status: task.TaskStatusCompleted,
		Method: "GetParameterValues", DeviceSN: action.SerialNumber, Result: result,
	}))
	require.Equal(t, ActionStatusFailed, store.actions[action.ID].Status)
	require.False(t, store.actions[action.ID].OwnedRFChange)
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

func TestActionConsumerUsesOneDurablePerDecisionSubject(t *testing.T) {
	bus := &actionConsumerBusFake{}
	consumer := NewActionConsumer(bus, &ActionService{})
	require.NoError(t, consumer.Start())
	require.Equal(t, []string{
		event.SubjectDeviceAccessRejected, event.SubjectDeviceAccessRevoked, event.SubjectDeviceAccessAccepted,
	}, bus.subjects)
	require.NoError(t, consumer.Stop())
	for _, sub := range bus.subs {
		require.True(t, sub.unsubscribed)
	}
}
