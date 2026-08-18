package device

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEnqueuer captures CreateTask invocations so tests can assert the
// SetRFSwitch handler is queueing the right TR-181 path per (carrier, tech).
type fakeEnqueuer struct {
	calls    []task.CreateTaskRequest
	createFn func(*task.CreateTaskRequest) (*task.Task, error)
}

func (f *fakeEnqueuer) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	f.calls = append(f.calls, *req)
	if f.createFn != nil {
		return f.createFn(req)
	}
	return &task.Task{ID: "fake-task"}, nil
}

func (f *fakeEnqueuer) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// rfSwitchEmittedPath extracts values[0].name from a captured
// CreateTaskRequest so tests can assert path correctness without parsing
// the full SetParameterValues body shape.
func rfSwitchEmittedPath(t *testing.T, req task.CreateTaskRequest) string {
	t.Helper()
	var payload struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(req.Params, &payload))
	require.Len(t, payload.Values, 1)
	require.Equal(t, "xsd:boolean", payload.Values[0].Type)
	return payload.Values[0].Name
}

func rfSwitchEmittedPaths(t *testing.T, req task.CreateTaskRequest) []string {
	t.Helper()
	var payload struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(req.Params, &payload))
	paths := make([]string, 0, len(payload.Values))
	for _, value := range payload.Values {
		require.Equal(t, "0", value.Value)
		require.Equal(t, "xsd:boolean", value.Type)
		paths = append(paths, value.Name)
	}
	return paths
}

// realCarrierRegistry returns a registry with the production cmcc/ctcc/cucc
// adapters wired — the RF control path tests want the real per-carrier
// behaviour, not a mock.
func realCarrierRegistry(t *testing.T) *carrier.CarrierRegistry {
	t.Helper()
	r := carrier.NewRegistry()
	r.Register(cmcc.New())
	r.Register(ctcc.New())
	r.Register(cucc.New())
	return r
}

// V1 — CMCC LTE → standard FAPControl.LTE.AdminState
func TestSetRFSwitch_T0029_CMCC_LTE(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-CMCC-LTE",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
			}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	require.NoError(t, svc.SetRFSwitch(context.Background(), deviceID, true))
	require.Len(t, enq.calls, 1)
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.LTE.AdminState", rfSwitchEmittedPath(t, enq.calls[0]))
}

// V2 — CMCC NR → FAPControl.NR.AdminState
func TestSetRFSwitch_T0029_CMCC_NR(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-CMCC-NR",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechNR,
			}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	require.NoError(t, svc.SetRFSwitch(context.Background(), deviceID, false))
	require.Len(t, enq.calls, 1)
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.NR.AdminState", rfSwitchEmittedPath(t, enq.calls[0]))
}

// V3 — CTCC LTE follows TR-181 standard
func TestSetRFSwitch_T0029_CTCC_LTE(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-CTCC", Carrier: model.CarrierCTCC, Technology: model.TechLTE}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	require.NoError(t, svc.SetRFSwitch(context.Background(), deviceID, true))
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.LTE.AdminState", rfSwitchEmittedPath(t, enq.calls[0]))
}

// V4 — CUCC NR-only — LTE call must fail rather than queue an unkeyed command
func TestSetRFSwitch_T0029_CUCC_LTE_Rejected(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-CUCC", Carrier: model.CarrierCUCC, Technology: model.TechLTE}, nil
		},
	}
	enq := &fakeEnqueuer{}
	paramRepo := &mockParamRepo{getByDeviceFn: func(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
		return []model.DeviceParameter{{
			ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Writable: true,
		}}, nil
	}}
	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	err := svc.SetRFSwitch(context.Background(), deviceID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support RF control")
	assert.Empty(t, enq.calls, "no command must be queued when path is empty")
}

// V5 — CUCC NR works
func TestSetRFSwitch_T0029_CUCC_NR(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-CUCC", Carrier: model.CarrierCUCC, Technology: model.TechNR}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	require.NoError(t, svc.SetRFSwitch(context.Background(), deviceID, true))
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.NR.AdminState", rfSwitchEmittedPath(t, enq.calls[0]))
}

// V6 — carrier registry not wired → fail-fast error, no command queued
func TestSetRFSwitch_T0029_NoRegistry(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-X", Carrier: model.CarrierCMCC, Technology: model.TechLTE}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	// Deliberately omit SetCarrierRegistry.

	err := svc.SetRFSwitch(context.Background(), deviceID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carrier registry not configured")
	assert.Empty(t, enq.calls)
}

// V7 — unknown carrier code → error from registry.Get propagates
func TestSetRFSwitch_T0029_UnknownCarrier(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-X", Carrier: model.CarrierCode("unknown"), Technology: model.TechLTE}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	err := svc.SetRFSwitch(context.Background(), deviceID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve carrier")
	assert.Empty(t, enq.calls)
}

// V8 — taskSvc not configured (programmer error) reports clearly
func TestSetRFSwitch_T0029_NoTaskService(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-X", Carrier: model.CarrierCMCC, Technology: model.TechLTE}, nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	// taskSvc deliberately omitted

	err := svc.SetRFSwitch(context.Background(), deviceID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task service not configured")
}

// V9 — device not found returns commonerrors.ErrNotFound (status 404 path)
func TestSetRFSwitch_T0029_DeviceNotFound(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return nil, nil // repo returns nil device + nil err for "not found"
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(&fakeEnqueuer{})
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	err := svc.SetRFSwitch(context.Background(), uuid.New(), true)
	require.Error(t, err)
}

// V10 — task service propagates error path (e.g. queue full)
func TestSetRFSwitch_T0029_TaskServiceError(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-X", Carrier: model.CarrierCMCC, Technology: model.TechLTE}, nil
		},
	}
	enq := &fakeEnqueuer{
		createFn: func(_ *task.CreateTaskRequest) (*task.Task, error) {
			return nil, errors.New("queue full")
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	err := svc.SetRFSwitch(context.Background(), deviceID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue RF switch command")
}

func TestQueueRFReadbackUsesSameCarrierPathAndSecurityMetadata(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID: deviceID, SerialNumber: "SN-RF-VERIFY",
				Carrier: model.CarrierCMCC, Technology: model.TechLTE,
			}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	created, err := svc.QueueRFReadback(context.Background(), deviceID, RFSwitchTaskOptions{
		Source: task.TaskSourceDeviceAccess, SourceID: "action-1",
		CommandKey:     "device-access-action:action-1:1:verify",
		AdmissionClass: task.AdmissionClassSecurityAction,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Len(t, enq.calls, 1)
	require.Equal(t, "GetParameterValues", enq.calls[0].Method)
	require.Equal(t, task.TaskSourceDeviceAccess, enq.calls[0].Source)
	require.Equal(t, "action-1", enq.calls[0].SourceID)
	require.Equal(t, task.AdmissionClassSecurityAction, enq.calls[0].AdmissionClass)
	var payload struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(enq.calls[0].Params, &payload))
	require.Equal(t, []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"}, payload.Names)
}

func TestQueueRFReadbackUsesPinnedSPVTargets(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID: deviceID, SerialNumber: "SN-RF-PINNED",
				Carrier: model.CarrierCMCC, Technology: model.TechLTE,
			}, nil
		},
	}
	paramRepo := &mockParamRepo{getByDeviceFn: func(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
		return []model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", Writable: true},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Writable: true},
		}, nil
	}}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))
	want := []string{
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus",
	}

	_, err := svc.QueueRFReadback(context.Background(), deviceID, RFSwitchTaskOptions{TargetPaths: want})
	require.NoError(t, err)
	require.Len(t, enq.calls, 1)
	var payload struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(enq.calls[0].Params, &payload))
	require.Equal(t, want, payload.Names)
}

func TestQueueRFSwitchRejectsPinnedTargetOutsideCurrentSnapshot(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{getByIDFn: func(context.Context, uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, SerialNumber: "SN-RF-STALE", Carrier: model.CarrierCMCC, Technology: model.TechLTE}, nil
	}}
	paramRepo := &mockParamRepo{getByDeviceFn: func(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
		return []model.DeviceParameter{{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Writable: true}}, nil
	}}
	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetTaskService(&fakeEnqueuer{})
	svc.SetCarrierRegistry(realCarrierRegistry(t))
	_, err := svc.QueueRFSwitch(context.Background(), deviceID, false, RFSwitchTaskOptions{TargetPaths: []string{
		"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus",
	}})
	require.ErrorContains(t, err, "not writable for current device snapshot")
}

func TestQueueRFSwitchAndReadbackResolveBLQMultiInstanceSnapshot(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID: deviceID, SerialNumber: "SN-BLQ", ProductClass: "FAP/BAIBLQ/SC",
				Carrier: model.CarrierCMCC, Technology: model.TechLTE,
			}, nil
		},
	}
	paramRepo := &mockParamRepo{
		getByDeviceFn: func(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
			return []model.DeviceParameter{
				{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: true},
				{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: true},
			}, nil
		},
	}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	_, err := svc.QueueRFSwitch(context.Background(), deviceID, false, RFSwitchTaskOptions{})
	require.NoError(t, err)
	_, err = svc.QueueRFReadback(context.Background(), deviceID, RFSwitchTaskOptions{})
	require.NoError(t, err)
	require.Len(t, enq.calls, 2)
	want := []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
	}
	require.Equal(t, want, rfSwitchEmittedPaths(t, enq.calls[0]))
	var readback struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(enq.calls[1].Params, &readback))
	require.Equal(t, want, readback.Names)
}

func TestQueueRFSwitchUsesMBS31001SASControlInsteadOfPerCellStatusAliases(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, SerialNumber: "SN-QRTB", ProductClass: "FAP/mBS31001/DC",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE}, nil
	}}
	paramRepo := &mockParamRepo{getByDeviceFn: func(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
		return []model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: true},
			{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: true},
			// device_parameters.writable is an early CPE snapshot and may be
			// false even though the ParamModel exposes this exact path as
			// READ_WRITE. The product resolver must follow the same T-0148
			// single-source rule as the parameter-tree write path.
			{ParameterPath: "Device.DeviceInfo.SAS.RadioEnable", Writable: false},
		}, nil
	}}
	enq := &fakeEnqueuer{}
	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetTaskService(enq)
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	_, err := svc.QueueRFSwitch(context.Background(), deviceID, true, RFSwitchTaskOptions{})
	require.NoError(t, err)
	require.Len(t, enq.calls, 1)
	require.Equal(t, "Device.DeviceInfo.SAS.RadioEnable", rfSwitchEmittedPath(t, enq.calls[0]))
	require.Contains(t, string(enq.calls[0].Params), `"value":"1"`)
}

func TestQueueRFSwitchRejectsMBS31001WithoutReportedSASControl(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, SerialNumber: "SN-QRTB", ProductClass: "FAP/mBS31001/DC",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE}, nil
	}}
	paramRepo := &mockParamRepo{getByDeviceFn: func(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
		return []model.DeviceParameter{{
			ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", Writable: true,
		}}, nil
	}}
	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetTaskService(&fakeEnqueuer{})
	svc.SetCarrierRegistry(realCarrierRegistry(t))

	_, err := svc.QueueRFSwitch(context.Background(), deviceID, true, RFSwitchTaskOptions{})
	require.ErrorContains(t, err, "requires reported Device.DeviceInfo.SAS.RadioEnable")
}
