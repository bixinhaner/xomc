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

type controlMappingReaderStub struct {
	mappings []carrier.GeofenceControlMapping
	err      error
}

type rfActivationAdmissionStub struct {
	allowed bool
	reason  string
	err     error
}

func (s rfActivationAdmissionStub) AllowRFActivation(
	context.Context,
	string,
	string,
	uuid.UUID,
) (bool, string, error) {
	return s.allowed, s.reason, s.err
}

func (s controlParameterReaderStub) GetByDevice(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
	return s.parameters, nil
}

func (s controlMappingReaderStub) GetByProductClass(context.Context, string, string) ([]carrier.GeofenceControlMapping, error) {
	return s.mappings, s.err
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
	due           []ControlAction
	scheduled     bool
	nextAttempt   int
	beginDeadline *time.Time
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

func (s *controlActionRepositoryStub) ListDueVerifications(
	context.Context,
	time.Time,
	int,
) ([]ControlAction, error) {
	return s.due, nil
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

func (s *controlActionRepositoryStub) BeginVerification(
	_ context.Context,
	id uuid.UUID,
	deadline time.Time,
	nextAt time.Time,
) error {
	s.beginDeadline = &deadline
	for _, action := range s.actions {
		if action.ID == id {
			action.Status = ControlActionVerifying
			action.VerificationDeadline = &deadline
			action.NextVerificationAt = &nextAt
		}
	}
	return nil
}

func (s *controlActionRepositoryStub) ScheduleVerification(
	_ context.Context,
	id uuid.UUID,
	verified []ControlParameterState,
	lastError string,
	attempt int,
	_ time.Time,
) error {
	s.scheduled = true
	s.verifiedState = verified
	s.lastError = lastError
	s.nextAttempt = attempt
	for _, action := range s.actions {
		if action.ID == id {
			action.VerificationAttempt = attempt
		}
	}
	return nil
}

func (s *controlActionRepositoryStub) CompleteVerification(
	_ context.Context,
	id uuid.UUID,
	verified []ControlParameterState,
	status ControlActionStatus,
	lastError string,
	_ time.Time,
) error {
	s.verifiedState = verified
	s.completed = status
	s.lastError = lastError
	for _, action := range s.actions {
		if action.ID == id {
			action.Status = status
		}
	}
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
		parameters = append(parameters,
			model.DeviceParameter{
				ParameterPath:  fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", instance),
				ParameterValue: "1", Writable: true, FAPInstance: instance,
			},
			model.DeviceParameter{
				ParameterPath:  fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.AdminState", instance),
				ParameterValue: "1", Writable: true, FAPInstance: instance,
			},
			model.DeviceParameter{
				ParameterPath:  fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", instance),
				ParameterValue: "1", Writable: false, FAPInstance: instance,
			},
		)
	}
	return controlParameterReaderStub{parameters: parameters}
}

func controlMappings() controlMappingReaderStub {
	return controlMappingReaderStub{mappings: []carrier.GeofenceControlMapping{
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			EntryType:    "parameter", Access: "READ_ONLY", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
			PrivatePath:  "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
	}}
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
	monitor.SetMappingReader(controlMappings())
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())

	err := monitor.handleExited(context.Background(), controlExitEvent(t, string(ActionLevelDeactivate)))

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.Equal(t, "SetParameterValues", tasks.request.Method)
	require.Equal(t, task.TaskSourceGeofence, tasks.request.Source)
	require.Contains(t, tasks.request.CommandKey, ":8:deactivate")
	require.Contains(t, string(tasks.request.Params), "IPSEC_ENABLE")
	require.Contains(t, string(tasks.request.Params), "FAPService.2.FAPControl.LTE.RFTxStatus")
	require.Contains(t, string(tasks.request.Params), "FAPService.5.FAPControl.LTE.RFTxStatus")
	require.Contains(t, string(tasks.request.Params), "FAPService.2.FAPControl.LTE.AdminState")
	require.Contains(t, string(tasks.request.Params), "FAPService.5.FAPControl.LTE.AdminState")
	require.Contains(t, string(tasks.request.Params), `"value":"0"`)
	require.Contains(t, string(tasks.request.Params), `"type":"xsd:boolean"`)
	require.NotContains(t, string(tasks.request.Params), "ParameterList")
}

func TestGeofenceControlMonitorNoOpControlsStillVerifyOpState(t *testing.T) {
	tasks := &controlTaskStub{}
	deviceID := uuid.New()
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: deviceID, SerialNumber: "SN-NOOP-ACTIVE", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReaderStub{parameters: []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "0", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", ParameterValue: "0", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", ParameterValue: "0", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1", Writable: false},
	}})
	monitor.SetMappingReader(controlMappings())
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())

	err := monitor.handleExited(context.Background(), controlExitEvent(t, string(ActionLevelDeactivate)))

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.Equal(t, "GetParameterValues", tasks.request.Method)
	require.Contains(t, string(tasks.request.Params), "FAPControl.LTE.OpState")
	require.NotContains(t, string(tasks.request.Params), "X_COM_RadioEnable")
}

func TestGeofenceControlMonitorPersistsCapabilityFailure(t *testing.T) {
	tasks := &controlTaskStub{}
	actions := newControlActionRepositoryStub()
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "FAP/BAIBLQ/SC",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(1))
	monitor.SetMappingReader(controlMappingReaderStub{})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)

	err := monitor.handleExited(
		context.Background(), controlExitEvent(t, string(ActionLevelDeactivate)),
	)

	require.NoError(t, err)
	require.Nil(t, tasks.request)
	require.Len(t, actions.actions, 1)
	for _, action := range actions.actions {
		require.Equal(t, ControlActionFailed, action.Status)
		require.Contains(t, action.LastError, "has no ParamModel mappings")
		require.NotNil(t, action.CompletedAt)
	}
}

func TestGeofenceControlMonitorQueuesMBS31001IPSecDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID:           uuid.New(),
			SerialNumber: "SN-MBS31001-1",
			ProductClass: "FAP/mBS31001/DC",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
		}},
		controlCarrierRegistry(),
		tasks,
		zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReaderStub{parameters: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "false", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "true", Writable: false},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", ParameterValue: "true", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "true", Writable: false},
		{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
		{ParameterPath: "Device.FAP.Ipsec.2.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "true", Writable: true},
	}})
	monitor.SetMappingReader(controlMappingReaderStub{mappings: []carrier.GeofenceControlMapping{
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			EntryType:    "parameter", Access: "READ_ONLY", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
			PrivatePath:  "Device.FAP.Ipsec.{i}.TUNNEL_CONFIG_TUNNELENABLE",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
	}})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())

	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID:              uuid.New(),
		SerialNumber:          "SN-MBS31001-1",
		EffectiveState:        string(EffectiveStateOutside),
		RequiredActionLevel:   string(ActionLevelDeactivate),
		EffectiveStateVersion: 9,
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleExited(context.Background(), event.Event{
		Subject: event.SubjectGeofenceDeviceExited,
		Payload: payload,
	}))
	require.NotNil(t, tasks.request)
	require.Contains(t, string(tasks.request.Params), "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus")
	require.NotContains(t, string(tasks.request.Params), "Device.DeviceInfo.SAS.RadioEnable")
	require.Contains(t, string(tasks.request.Params), "Device.FAP.Ipsec.1.TUNNEL_ENABLE")
	require.Contains(t, string(tasks.request.Params), "Device.FAP.Ipsec.2.TUNNEL_ENABLE")
}

func TestGeofenceControlMonitorUsesParamModelRFMapping(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID:              uuid.New(),
			SerialNumber:    "SN-MAPPED-RF-1",
			ProductClass:    "FAP/MLN/DC",
			FirmwareVersion: "MLN_5.1.12.2",
			Carrier:         model.CarrierCMCC,
			Technology:      model.TechLTE,
		}},
		controlCarrierRegistry(),
		tasks,
		zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReaderStub{parameters: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", ParameterValue: "false", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "true", Writable: false},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "true", Writable: false},
		{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_ENABLE", ParameterValue: "true", Writable: true},
	}})
	monitor.SetMappingReader(controlMappingReaderStub{mappings: []carrier.GeofenceControlMapping{
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
			EntryType:    "parameter",
			Access:       "READ_WRITE",
			IsActive:     true,
			IsSupported:  true,
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			EntryType:    "parameter", Access: "READ_ONLY", IsActive: true, IsSupported: true,
		},
		{
			StandardPath: "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
			PrivatePath:  "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
			EntryType:    "parameter", Access: "READ_WRITE", IsActive: true, IsSupported: true,
		},
	}})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())

	err := monitor.handleExited(context.Background(), controlExitEvent(t, string(ActionLevelDeactivate)))

	require.NoError(t, err)
	require.NotNil(t, tasks.request)
	require.Contains(t, string(tasks.request.Params), "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus")
	require.NotContains(t, string(tasks.request.Params), "Device.DeviceInfo.SAS.RadioEnable")
}

func TestGeofenceControlMonitorSubscribesOutsideAndEscalationEdges(t *testing.T) {
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{},
		controlCarrierRegistry(),
		&controlTaskStub{},
		zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader())
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())
	bus := &coordinatorEventBusStub{}

	require.NoError(t, monitor.Subscribe(bus))
	require.Contains(t, bus.subjects, event.SubjectGeofenceDeviceExited)
	require.Contains(t, bus.subjects, event.SubjectGeofenceDeviceEscalated)
	require.Equal(
		t,
		geofenceControlQueue(event.SubjectGeofenceDeviceExited),
		bus.queues[event.SubjectGeofenceDeviceExited],
	)
	require.Equal(
		t,
		geofenceControlQueue(event.SubjectGeofenceDeviceEscalated),
		bus.queues[event.SubjectGeofenceDeviceEscalated],
	)
	require.NotEqual(
		t,
		bus.queues[event.SubjectGeofenceDeviceExited],
		bus.queues[event.SubjectGeofenceDeviceEscalated],
	)
	require.Len(t, bus.queues, 6)
}

func TestGeofenceControlMonitorQueuesEscalatedDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-CONTROL-1", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(1))
	monitor.SetMappingReader(controlMappings())
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(newControlActionRepositoryStub())
	evt := controlExitEvent(t, string(ActionLevelDeactivate))
	evt.Subject = event.SubjectGeofenceDeviceEscalated

	require.NoError(t, monitor.handleExited(context.Background(), evt))
	require.NotNil(t, tasks.request)
	require.Contains(t, tasks.request.CommandKey, ":8:deactivate")
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
	locked              bool
	released            bool
	partitionedDeviceSN string
	partitionedCommand  string
	unpartitionedCalls  int
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
	s.unpartitionedCalls++
	if !s.locked || s.released {
		return nil, fmt.Errorf("command history checked outside lock")
	}
	return nil, nil
}

func (s *orderedControlTaskHistoryStub) GetByDeviceAndCommandKey(
	_ context.Context,
	deviceSN string,
	commandKey string,
) (*task.Task, error) {
	if !s.locked || s.released {
		return nil, fmt.Errorf("command history checked outside lock")
	}
	s.partitionedDeviceSN = deviceSN
	s.partitionedCommand = commandKey
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
	monitor.SetMappingReader(controlMappings())
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
	require.Equal(t, "SN-CONTROL-1", history.partitionedDeviceSN)
	require.Contains(t, history.partitionedCommand, ":deactivate")
	require.Zero(t, history.unpartitionedCalls)
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
	monitor.SetMappingReader(controlMappings())
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

func TestGeofenceControlMonitorFailsClosedWhenAdmissionReaderIsMissing(t *testing.T) {
	deviceID := uuid.New()
	actions := newControlActionRepositoryStub()
	actions.recoverable = &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:missing-admission:deactivate",
		DeviceID: deviceID, DeviceSN: "SN-NO-ADMISSION", ActionType: ControlActionDeactivate,
		Status: ControlActionVerified, ContractVersion: GeofenceControlContractVersion,
	}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: deviceID, SerialNumber: "SN-NO-ADMISSION", Carrier: model.CarrierCMCC,
		}}, controlCarrierRegistry(), &controlTaskStub{}, zap.NewNop(),
	)
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID: deviceID, SerialNumber: "SN-NO-ADMISSION",
		EffectiveState: string(EffectiveStateInside), EffectiveStateVersion: 1,
	})
	require.NoError(t, err)

	err = monitor.handleEntered(context.Background(), event.Event{Payload: payload})

	require.ErrorContains(t, err, "admission reader is required")
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
	monitor.SetMappingReader(controlMappings())
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
	monitor.SetMappingReader(controlMappings())

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
		Status: ControlActionExecuting,
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
	require.Equal(t, action.ActionKey+":verify:0", tasks.request.CommandKey)
	require.Contains(t, string(tasks.request.Params), action.RequestedState[0].Path)
	require.Equal(t, ControlActionVerifying, actions.updatedStatus)
	require.NotNil(t, actions.beginDeadline)
	require.WithinDuration(t, time.Now().UTC().Add(geofenceTerminalVerificationTimeout), *actions.beginDeadline, time.Second)
}

func TestGeofenceControlMonitorIgnoresTaskFailureAfterActionVerified(t *testing.T) {
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		Status: ControlActionVerified,
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "late-spv-failure", DeviceSN: action.DeviceSN, Method: "SetParameterValues",
		CommandKey: action.ActionKey, Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(), ErrorMessage: "late duplicate failure",
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskFailed, Payload: payload,
	}))
	require.Equal(t, ControlActionVerified, action.Status)
	require.Empty(t, actions.completed)
	require.Empty(t, actions.lastError)
}

func TestGeofenceControlMonitorIgnoresStaleVerificationAttempt(t *testing.T) {
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		Status: ControlActionVerifying, VerificationAttempt: 1,
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "stale-gpv", DeviceSN: action.DeviceSN, Method: "GetParameterValues",
		CommandKey: action.ActionKey + ":verify:0", Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(), ErrorMessage: "stale attempt failure",
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskFailed, Payload: payload,
	}))
	require.Equal(t, ControlActionVerifying, action.Status)
	require.Empty(t, actions.completed)
	require.False(t, actions.scheduled)
}

func TestGeofenceControlMonitorMarksVerifiedOnlyAfterMatchingReadback(t *testing.T) {
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		Status: ControlActionVerifying,
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
		CommandKey: action.ActionKey + ":verify:0", Source: task.TaskSourceGeofence,
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
		Status:         ControlActionVerifying,
		RequestedState: []ControlParameterState{{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0"}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "gpv-1", Method: "GetParameterValues", CommandKey: action.ActionKey + ":verify:0",
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

func TestGeofenceControlMonitorContractV2CannotVerifyWithoutOpState(t *testing.T) {
	actions := newControlActionRepositoryStub()
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:8:deactivate", DeviceSN: "SN-1",
		Status:          ControlActionVerifying,
		ContractVersion: GeofenceControlContractVersion,
		RequestedState: []ControlParameterState{{
			Path:  "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
			Value: "0", Role: carrier.GeofenceRoleRF,
		}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "gpv-no-terminal", Method: "GetParameterValues",
		CommandKey: action.ActionKey + ":verify:0", Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(),
		Result:   json.RawMessage(`{"standard_parameter_values":[{"name":"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus","value":"0"}]}`),
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted, Payload: payload,
	}))
	require.Equal(t, ControlActionPartialFailed, actions.completed)
	require.Contains(t, actions.lastError, "missing required OpState")
}

func TestGeofenceControlMonitorKeepsVerifyingUntilOpStateInactive(t *testing.T) {
	actions := newControlActionRepositoryStub()
	deadline := time.Now().UTC().Add(time.Minute)
	rfPath := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	opStatePath := "Device.Services.FAPService.1.FAPControl.LTE.OpState"
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:9:deactivate", DeviceSN: "SN-1",
		Status:          ControlActionVerifying,
		ContractVersion: GeofenceControlContractVersion, VerificationDeadline: &deadline,
		RequestedState: []ControlParameterState{{Path: rfPath, Value: "0", Role: carrier.GeofenceRoleRF}},
		TerminalState:  []ControlParameterState{{Path: opStatePath, Value: "0", Role: carrier.GeofenceRoleOpState}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "gpv-terminal-active", Method: "GetParameterValues",
		CommandKey: action.ActionKey + ":verify:0", Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(),
		Result: json.RawMessage(fmt.Sprintf(
			`{"standard_parameter_values":[{"name":%q,"value":"false"},{"name":%q,"value":"true"}]}`,
			rfPath, opStatePath,
		)),
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted, Payload: payload,
	}))
	require.True(t, actions.scheduled)
	require.Equal(t, 1, actions.nextAttempt)
	require.Empty(t, actions.completed)
	require.Contains(t, actions.lastError, "expected 0 got 1")
}

func TestGeofenceControlMonitorVerifiesAfterOpStateInactive(t *testing.T) {
	actions := newControlActionRepositoryStub()
	deadline := time.Now().UTC().Add(time.Minute)
	rfPath := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	opStatePath := "Device.Services.FAPService.1.FAPControl.LTE.OpState"
	action := &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:10:deactivate", DeviceSN: "SN-1",
		Status:          ControlActionVerifying,
		ContractVersion: GeofenceControlContractVersion, VerificationDeadline: &deadline,
		RequestedState: []ControlParameterState{{Path: rfPath, Value: "0", Role: carrier.GeofenceRoleRF}},
		TerminalState:  []ControlParameterState{{Path: opStatePath, Value: "0", Role: carrier.GeofenceRoleOpState}},
	}
	actions.actions[action.ActionKey] = action
	monitor := NewGeofenceControlMonitor(nil, nil, nil, zap.NewNop())
	monitor.SetActionRepository(actions)
	payload, err := json.Marshal(task.Task{
		ID: "gpv-terminal-inactive", Method: "GetParameterValues",
		CommandKey: action.ActionKey + ":verify:0", Source: task.TaskSourceGeofence,
		SourceID: action.ID.String(),
		Result: json.RawMessage(fmt.Sprintf(
			`{"standard_parameter_values":[{"name":%q,"value":"0"},{"name":%q,"value":"false"}]}`,
			rfPath, opStatePath,
		)),
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleTaskTerminal(context.Background(), event.Event{
		Subject: event.SubjectTaskCompleted, Payload: payload,
	}))
	require.Equal(t, ControlActionVerified, actions.completed)
	require.False(t, actions.scheduled)
}

func TestGeofenceControlMonitorQueuesActivationOnlyAfterCompletedDeactivation(t *testing.T) {
	tasks := &controlTaskStub{}
	actions := newControlActionRepositoryStub()
	deactivationID := uuid.New()
	actions.recoverable = &ControlAction{
		ID: deactivationID, ActionKey: "geofence:device:8:deactivate",
		ActionType: ControlActionDeactivate, Status: ControlActionVerified,
		ContractVersion: GeofenceControlContractVersion,
		BeforeState: []ControlParameterState{
			{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0", Role: carrier.GeofenceRoleIPSec},
			{Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Value: "1", Role: carrier.GeofenceRoleRF},
			{Path: "Device.Services.FAPService.2.FAPControl.LTE.OpState", Value: "1", Role: carrier.GeofenceRoleOpState},
		},
		RequestedState: []ControlParameterState{{
			Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Value: "0", Role: carrier.GeofenceRoleRF,
		}},
		TerminalState: []ControlParameterState{{Path: "Device.Services.FAPService.2.FAPControl.LTE.OpState", Value: "0", Role: carrier.GeofenceRoleOpState}},
		VerifiedState: []ControlParameterState{
			{Path: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Value: "0", Role: carrier.GeofenceRoleRF},
			{Path: "Device.Services.FAPService.2.FAPControl.LTE.OpState", Value: "0", Role: carrier.GeofenceRoleOpState},
		},
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
		{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.OpState", ParameterValue: "0", Writable: false, FAPInstance: 2},
		{
			ParameterPath:  "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
			ParameterValue: "0", Writable: true, FAPInstance: 2,
		},
	}})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)
	monitor.SetRFActivationAdmissionReader(rfActivationAdmissionStub{allowed: true})
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
	require.Contains(t, string(tasks.request.Params), "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus")
	require.NotContains(t, string(tasks.request.Params), "IPSEC_ENABLE")
}

func TestGeofenceControlMonitorBlocksActivationWhenDeviceAccessRejectsIt(t *testing.T) {
	deviceID := uuid.New()
	tasks := &controlTaskStub{}
	actions := newControlActionRepositoryStub()
	actions.recoverable = &ControlAction{
		ID: uuid.New(), ActionKey: "geofence:device:12:deactivate",
		DeviceID: deviceID, DeviceSN: "SN-BLOCKED", ActionType: ControlActionDeactivate,
		Status: ControlActionVerified, ContractVersion: GeofenceControlContractVersion,
		BeforeState: []ControlParameterState{{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "1", Role: carrier.GeofenceRoleRF,
		}},
		VerifiedState: []ControlParameterState{{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0", Role: carrier.GeofenceRoleRF,
		}},
		TerminalState: []ControlParameterState{{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Value: "0", Role: carrier.GeofenceRoleOpState,
		}},
	}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: deviceID, SerialNumber: "SN-BLOCKED", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReader(1))
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)
	monitor.SetRFActivationAdmissionReader(rfActivationAdmissionStub{
		allowed: false, reason: "device_access_owned_isolation",
	})
	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID: deviceID, SerialNumber: "SN-BLOCKED",
		EffectiveState: string(EffectiveStateInside), EffectiveStateVersion: 13,
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleEntered(context.Background(), event.Event{Payload: payload}))
	require.Nil(t, tasks.request)
	blocked := actions.actions["geofence:device:12:activate"]
	require.NotNil(t, blocked)
	require.Equal(t, ControlActionFailed, blocked.Status)
	require.Contains(t, blocked.LastError, "device_access_owned_isolation")
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

func TestGeofenceControlMonitorRestoresEveryOwnedRFAndIPSecChange(t *testing.T) {
	tasks := &controlTaskStub{}
	actions := newControlActionRepositoryStub()
	deactivationID := uuid.New()
	rfPath := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	ipsecPath := "Device.FAP.Ipsec.1.TUNNEL_ENABLE"
	actions.recoverable = &ControlAction{
		ID: deactivationID, ActionKey: "geofence:device:10:deactivate",
		ActionType: ControlActionDeactivate, Status: ControlActionVerified,
		ContractVersion: GeofenceControlContractVersion,
		BeforeState: []ControlParameterState{
			{Path: rfPath, Value: "1", Role: carrier.GeofenceRoleRF},
			{Path: ipsecPath, Value: "1", Role: carrier.GeofenceRoleIPSec},
			{Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Value: "1", Role: carrier.GeofenceRoleOpState},
		},
		RequestedState: []ControlParameterState{
			{Path: rfPath, Value: "0", Role: carrier.GeofenceRoleRF},
			{Path: ipsecPath, Value: "0", Role: carrier.GeofenceRoleIPSec},
		},
		TerminalState: []ControlParameterState{{Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Value: "0", Role: carrier.GeofenceRoleOpState}},
		VerifiedState: []ControlParameterState{
			{Path: rfPath, Value: "0", Role: carrier.GeofenceRoleRF},
			{Path: ipsecPath, Value: "0", Role: carrier.GeofenceRoleIPSec},
			{Path: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Value: "0", Role: carrier.GeofenceRoleOpState},
		},
	}
	monitor := NewGeofenceControlMonitor(
		controlDeviceReaderStub{device: &model.Device{
			ID: uuid.New(), SerialNumber: "SN-RESTORE-ALL", ProductClass: "BLQ",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		controlCarrierRegistry(), tasks, zap.NewNop(),
	)
	monitor.SetParameterReader(controlParameterReaderStub{parameters: []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", ParameterValue: "0", Writable: true, FAPInstance: 1},
		{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE", ParameterValue: "0", Writable: true},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "0", Writable: false},
	}})
	monitor.SetTaskHistoryReader(controlTaskHistoryStub{})
	monitor.SetActionRepository(actions)
	monitor.SetRFActivationAdmissionReader(rfActivationAdmissionStub{allowed: true})
	payload, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID: uuid.New(), SerialNumber: "SN-RESTORE-ALL",
		EffectiveState:        string(EffectiveStateInside),
		EffectiveStateVersion: 11,
	})
	require.NoError(t, err)

	require.NoError(t, monitor.handleEntered(context.Background(), event.Event{Payload: payload}))
	require.NotNil(t, tasks.request)
	require.Contains(t, string(tasks.request.Params), rfPath)
	require.Contains(t, string(tasks.request.Params), ipsecPath)
}
