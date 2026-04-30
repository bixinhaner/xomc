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

// rfSwitchEmittedPath extracts the ParameterList[0].Name from a captured
// CreateTaskRequest so tests can assert path correctness without parsing
// the full SetParameterValues body shape.
func rfSwitchEmittedPath(t *testing.T, req task.CreateTaskRequest) string {
	t.Helper()
	var payload struct {
		ParameterList []struct {
			Name  string `json:"Name"`
			Value string `json:"Value"`
		} `json:"ParameterList"`
	}
	require.NoError(t, json.Unmarshal(req.Params, &payload))
	require.Len(t, payload.ParameterList, 1)
	return payload.ParameterList[0].Name
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
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
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
