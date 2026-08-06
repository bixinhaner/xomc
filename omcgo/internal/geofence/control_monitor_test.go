package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type controlDeviceReaderStub struct {
	device *model.Device
}

type controlParameterReaderStub struct {
	parameters []model.DeviceParameter
}

func (s controlParameterReaderStub) GetByDevice(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
	return s.parameters, nil
}

func (s controlDeviceReaderStub) GetBySerialNumber(context.Context, string) (*model.Device, error) {
	return s.device, nil
}

type controlTaskStub struct {
	request *task.CreateTaskRequest
	err     error
}

type controlTaskHistoryStub struct {
	activation *task.Task
}

type controlActionRepositoryStub struct {
	actions       map[string]*ControlAction
	recoverable   *ControlAction
	updatedStatus ControlActionStatus
	verifiedState []ControlParameterState
	completed     ControlActionStatus
	lastError     string
}

func newControlActionRepositoryStub() *controlActionRepositoryStub {
	return &controlActionRepositoryStub{actions: make(map[string]*ControlAction)}
}

func (s *controlActionRepositoryStub) Create(
	_ context.Context,
	action *ControlAction,
) (*ControlAction, bool, error) {
	if existing := s.actions[action.ActionKey]; existing != nil {
		return existing, false, nil
	}
	copyAction := *action
	s.actions[action.ActionKey] = &copyAction
	return &copyAction, true, nil
}

func (s *controlActionRepositoryStub) GetByID(_ context.Context, id uuid.UUID) (*ControlAction, error) {
	for _, action := range s.actions {
		if action.ID == id {
			return action, nil
		}
	}
	if s.recoverable != nil && s.recoverable.ID == id {
		return s.recoverable, nil
	}
	return nil, nil
}

func (s *controlActionRepositoryStub) GetByActionKey(_ context.Context, key string) (*ControlAction, error) {
	return s.actions[key], nil
}

func (s *controlActionRepositoryStub) ListByGeofence(
	context.Context,
	uuid.UUID,
	int,
) ([]ControlAction, error) {
	return nil, nil
}

func (s *controlActionRepositoryStub) FindRecoverableDeactivation(
	context.Context,
	uuid.UUID,
) (*ControlAction, error) {
	return s.recoverable, nil
}

func (s *controlActionRepositoryStub) UpdateStatus(
	_ context.Context,
	id uuid.UUID,
	status ControlActionStatus,
) error {
	s.updatedStatus = status
	for _, action := range s.actions {
		if action.ID == id {
			action.Status = status
		}
	}
	return nil
}

func (s *controlActionRepositoryStub) CompleteVerification(
	_ context.Context,
	_ uuid.UUID,
	verified []ControlParameterState,
	status ControlActionStatus,
	lastError string,
	_ time.Time,
) error {
	s.verifiedState = verified
	s.completed = status
	s.lastError = lastError
	return nil
}

func (s controlTaskHistoryStub) AcquireCommandKeyLock(context.Context, string) (func(), error) {
	return func() {}, nil
}

func (s controlTaskHistoryStub) GetByCommandKey(context.Context, string) (*task.Task, error) {
	return s.activation, nil
}

func (s *controlTaskStub) CreateTask(_ context.Context, request *task.CreateTaskRequest) (*task.Task, error) {
	s.request = request
	if s.err != nil {
		return nil, s.err
	}
	return &task.Task{ID: uuid.NewString(), DeviceSN: request.DeviceSN}, nil
}

func (s *controlTaskStub) GetQueueLength(context.Context, string) (int64, error) {
	return 0, nil
}

func controlCarrierRegistry() *carrier.CarrierRegistry {
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	return registry
}

func controlParameterReader(instances ...int) controlParameterReaderStub {
	parameters := []model.DeviceParameter{{
		ParameterPath:  "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
		ParameterValue: "1", Writable: true,
	}}
	for _, instance := range instances {
		parameters = append(parameters, model.DeviceParameter{
			ParameterPath:  fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", instance),
			ParameterValue: "1", Writable: true,
			FAPInstance: instance,
		})
	}
	return controlParameterReaderStub{parameters: parameters}
}

func controlExitEvent(t *testing.T, actionLevel string) event.Event {
	t.Helper()
	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID:              uuid.New(),
		SerialNumber:          "SN-CONTROL-1",
		EffectiveState:        string(EffectiveStateOutside),
		RequiredActionLevel:   actionLevel,
		EffectiveStateVersion: 8,
	})
	require.NoError(t, err)
	return event.Event{Subject: event.SubjectGeofenceDeviceExited, Payload: payload}
}

func TestGeofenceControlMonitorQueuesRFDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID:           uuid.New(),
			SerialNumber: "SN-CONTROL-1",
			ProductClass: "BLQ",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
		}},
		controlCarrierRegistry(),
		tasks,
		zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(2, 5))
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())

	err := monitor.handleExited(context.Background(), controlExitEvent(t, string(ActionLevelDeactivate)))

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.Equal(t, "SetParameterValues", tasks.request.Method)
	require.Equal(t, task.TaskSourceGeofence, tasks.request.Source)
	require.Contains(t, tasks.request.CommandKey, ":8:deactivate")
	require.Contains(t, string(tasks.request.Params), "IPSEC_ENABLE")
	require.Contains(t, string(tasks.request.Params), "FAPControl.LTE.RFTxStatus")
	require.Contains(t, string(tasks.request.Params), "FAPService.2.FAPControl")
	require.Contains(t, string(tasks.request.Params), "FAPService.5.FAPControl")
	require.Contains(t, string(tasks.request.Params), `"value":"0"`)
	require.Contains(t, string(tasks.request.Params), `"type":"xsd:boolean"`)
	require.NotContains(t, string(tasks.request.Params), "ParameterList")
}

func TestGeofenceControlMonitorSkipsExistingDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ", Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReaderStub{parameters: []model.DeviceParameter{
		{
			ParameterPath:  "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
			ParameterValue: "0", Writable: true,
		},
		{
			ParameterPath:  "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
			ParameterValue: "0", Writable: true, FAPInstance: 2,
		},
	}})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{
		activation: &task.Task{CommandKey: "geofence:existing:deactivate"},
	})
	monitor.SetActionRepository(newControlActionRepositoryStub())

	err := monitor.handleExited(context.Background(), controlExitEvent(t, string(ActionLevelDeactivate)))

	require.NoError(t, err)
	require.Nil(t, tasks.request)
}

type orderedControlTaskHistoryStub struct {
	locked   bool
	released bool
}

func (s *orderedControlTaskHistoryStub) AcquireCommandKeyLock(
	context.Context,
	string,
) (func(), error) {
	s.locked = true
	return func() { s.released = true }, nil
}

func (s *orderedControlTaskHistoryStub) GetByCommandKey(
	context.Context,
	string,
) (*task.Task, error) {
	if !s.locked || s.released {
		return nil, fmt.Errorf("command history checked outside lock")
	}
	return nil, nil
}

func TestGeofenceControlMonitorLocksCommandKeyDuringCreate(t *testing.T) {
	tasks := &controlTaskStub{}
	history := &orderedControlTaskHistoryStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(2))
	monitor.SetTaskHistoryReader(history)
	monitor.SetActionRepository(newControlActionRepositoryStub())

	err := monitor.handleExited(
		context.Background(),
		controlExitEvent(t, string(ActionLevelDeactivate)),
	)

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.True(t, history.locked)
	require.True(t, history.released)
}

func TestGeofenceControlMonitorRetriesPendingActionAfterQueueFailure(t *testing.T) {
	tasks := &controlTaskStub{err: fmt.Errorf("queue unavailable")}
	actions := newControlActionRepositoryStub()
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(2))
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)
	evt := controlExitEvent(t, string(ActionLevelDeactivate))

	require.ErrorContains(t, monitor.handleExited(context.Background(), evt), "queue unavailable")
	require.Len(t, actions.actions, 1)
	tasks.err = nil
	tasks.request = nil

	require.NoError(t, monitor.handleExited(context.Background(), evt))
	require.NotNil(t, tasks.request)
	require.Equal(t, ControlActionExecuting, actions.updatedStatus)
}

func TestGeofenceControlMonitorQueuesLifecycleDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	deviceID := uuid.MustParse("00000000-0000-0000-0000-000000000008")
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: deviceID, SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(2))
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())
	payload, err := json.Marshal(event.GeofenceLifecycleDeactivationPayload{
		DeviceID: deviceID, SerialNumber: "SN-CONTROL-1", GeofenceID: uuid.New(),
		TargetStatus: string(DefinitionStatusDisabled), Reason: "operator disabled",
	})
	require.NoError(t, err)

	err = monitor.handleLifecycleDeactivation(context.Background(), event.Event{
		ID: "outbox-event-1", Subject: event.SubjectGeofenceLifecycleDeactivationRequired,
		Payload: payload,
	})

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.Equal(t,
		"geofence:"+deviceID.String()+":lifecycle:outbox-event-1:deactivate",
		tasks.request.CommandKey,
	)
	require.Contains(t, string(tasks.request.Params), `"value":"0"`)
}

func TestGeofenceControlMonitorIgnoresNonDeactivationEdge(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{},
		controlCarrierRegistry(),
		tasks,
		zap.NewNop(),
	)

	err := monitor.handleExited(context.Background(), controlExitEvent(t, string(ActionLevelNotifyOnly)))

	require.NoError(t, err)
	require.Nil(t, tasks.request)
}

func TestGeofenceControlMonitorRejectsControlWithoutTaskHistory(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(2))

	err := monitor.handleExited(
		context.Background(),
		controlExitEvent(t, string(ActionLevelDeactivate)),
	)

	require.ErrorContains(t, err, "task history is not configured")
	require.Nil(t, tasks.request)
}

func TestGeofenceControlMonitorAcceptsOnlyGeofenceTaskResults(t *testing.T) {
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	payload, err := json.Marshal(task.Task{ID: "task-1", CommandKey: "geofence:device:8:deactivate"})
	require.NoError(t, err)

	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted,
		Payload: payload,
	}))
	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskFailed,
		Payload: payload,
	}))

	otherPayload, err := json.Marshal(task.Task{ID: "task-2", CommandKey: "mml:task-2"})
	require.NoError(t, err)
	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted,
		Payload: otherPayload,
	}))
}

func TestGeofenceControlMonitorQueuesCorrelatedReadbackAfterSPV(t *testing.T) {
	tasks := &controlTaskStub{}
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		RequestedState: []ControlParameterState{{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0",
		}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, tasks, zap.NewNop())
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "spv-1", DeviceSN: action.DeviceSN, Method: "SetParameterValues",
		CommandKey: action.ActionKey, Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(),
	})
	require.NoError(t, err)

	err = monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted, Payload: payload,
	})

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.Equal(t, "GetParameterValues", tasks.request.Method)
	require.Equal(t, action.ID.String(), tasks.request.SourceID)
	require.Equal(t, action.ActionKey+":verify", tasks.request.CommandKey)
	require.Contains(t, string(tasks.request.Params), action.RequestedState[0].Path)
	require.Equal(t, ControlActionVerifying, actions.updatedStatus)
}

func TestGeofenceControlMonitorMarksVerifiedOnlyAfterMatchingReadback(t *testing.T) {
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		RequestedState: []ControlParameterState{{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0",
		}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	result := json.RawMessage(`{"standard_parameter_values":[{"name":"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus","value":"false"}]}`)
	payload, err := json.Marshal(task.Task{
		ID: "gpv-1", DeviceSN: action.DeviceSN, Method: "GetParameterValues",
		CommandKey: action.ActionKey + ":verify", Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(), Result: result,
	})
	require.NoError(t, err)

	err = monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted, Payload: payload,
	})

	require.NoError(t, err)
	require.Equal(t, ControlActionVerified, actions.completed)
	require.Equal(t, []ControlParameterState{{Path: action.RequestedState[0].Path, Value: "0"}}, actions.verifiedState)
	require.Empty(t, actions.lastError)
}

func TestGeofenceControlMonitorDoesNotClaimVerificationOnReadbackMismatch(t *testing.T) {
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		RequestedState: []ControlParameterState{{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0"}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "gpv-1", Method: "GetParameterValues", CommandKey: action.ActionKey + ":verify",
		Source: task.TaskSourceGeofence, SourceID: action.ID.String(),
		Result: json.RawMessage(`{"standard_parameter_values":[{"name":"Device.Services.FAPService.Ipsec.IPSEC_ENABLE","value":"1"}]}`),
	})
	require.NoError(t, err)

	err = monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted, Payload: payload,
	})

	require.NoError(t, err)
	require.Equal(t, ControlActionPartialFailed, actions.completed)
	require.Contains(t, actions.lastError, "expected 0 got 1")
}

func TestGeofenceControlMonitorQueuesActivationOnlyAfterCompletedDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	actions := newControlActionRepositoryStub()
	deactivationID := uuid.New()
	actions.recoverable = &ControlAction{
		ID: deactivationID, ActionKey: "geofence:device:8:deactivate",
		ActionType: ControlActionDeactivate, Status: ControlActionVerified,
		BeforeState: []ControlParameterState{
			{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0"},
			{Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Value: "1"},
		},
		RequestedState: []ControlParameterState{{
			Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Value: "0",
		}},
	}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ", Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReaderStub{parameters: []model.DeviceParameter{
		{
			ParameterPath:  "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
			ParameterValue: "0", Writable: true,
		},
		{
			ParameterPath:  "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
			ParameterValue: "0", Writable: true, FAPInstance: 2,
		},
	}})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID: uuid.MustParse("00000000-0000-0000-0000-000000000008"), SerialNumber: "SN-CONTROL-1",
		EffectiveState: string(EffectiveStateInside), EffectiveStateVersion: 9,
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleEntered(context.Background(), event.Event{Payload: payload}))
	require.NotNil(t, tasks.request)
	require.Equal(t, "geofence:device:8:activate", tasks.request.CommandKey)
	require.Equal(t, actions.actions[tasks.request.CommandKey].ID.String(), tasks.request.SourceID)
	require.Contains(t, string(tasks.request.Params), `"value":"1"`)
	require.NotContains(t, string(tasks.request.Params), "IPSEC_ENABLE")
}

func TestGeofenceControlMonitorDoesNotActivateWithoutCompletedDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(controlDeviceReaderStub{}, controlCarrierRegistry(), tasks, zap.NewNop())
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())
	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID: uuid.New(), SerialNumber: "SN-CONTROL-1", EffectiveState: string(EffectiveStateInside),
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleEntered(context.Background(), event.Event{Payload: payload}))
	require.Nil(t, tasks.request)
}
