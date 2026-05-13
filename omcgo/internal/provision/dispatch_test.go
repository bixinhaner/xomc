package provision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// activeTmpl 构造一份默认 active provisioning 模板。
func activeTmpl(t *testing.T, params string) *template.ConfigTemplate {
	t.Helper()
	return &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "tmpl",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(params),
		Active:       true,
	}
}

func TestDispatchTemplate_NilTemplate(t *testing.T) {
	h := newFullEngineHarness()
	_, err := h.engine.DispatchTemplate(context.Background(), nil, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template is nil")
}

func TestDispatchTemplate_InactiveTemplate(t *testing.T) {
	h := newFullEngineHarness()
	tmpl := activeTmpl(t, `{"Device.WiFi.SSID":"x"}`)
	tmpl.Active = false
	_, err := h.engine.DispatchTemplate(context.Background(), tmpl, uuid.New())
	require.ErrorIs(t, err, ErrDispatchTemplateInactive)
}

func TestDispatchTemplate_DeviceNotFound(t *testing.T) {
	h := newFullEngineHarness()
	h.devRepo.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return nil, nil
	}
	tmpl := activeTmpl(t, `{"Device.WiFi.SSID":"x"}`)
	_, err := h.engine.DispatchTemplate(context.Background(), tmpl, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDispatchTemplate_HappyPath_EnqueuesSPV(t *testing.T) {
	h := newFullEngineHarness()
	deviceID := uuid.New()
	h.devRepo.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-DISP-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "BLQ",
		}, nil
	}

	var enqueued []*task.CreateTaskRequest
	h.cmdQueue.CreateFn = func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		enqueued = append(enqueued, req)
		return task.NewTask(req), nil
	}

	var lastStatus ProvisioningState
	var lastTaskTemplateID *uuid.UUID
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, status ProvisioningState, _ string) error {
		lastStatus = status
		return nil
	}
	h.taskRepo.UpdateFn = func(_ context.Context, tk *ProvisioningTask) error {
		if tk.TemplateID != nil {
			id := *tk.TemplateID
			lastTaskTemplateID = &id
		}
		return nil
	}

	tmpl := activeTmpl(t, `{"Device.WiFi.SSID":"OMC-Test","Device.WiFi.Channel":6}`)
	taskID, err := h.engine.DispatchTemplate(context.Background(), tmpl, deviceID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, taskID)
	// 至少包含一条 SPV 步骤（GPV 验证步骤可选）。
	var hasSPV bool
	for _, req := range enqueued {
		if req.Method == "SetParameterValues" {
			hasSPV = true
		}
	}
	assert.True(t, hasSPV, "应有 SetParameterValues 入队")
	// 任务最终 transition 到 verifying（SPV 步骤已入队等待 CPE 响应）。
	assert.Equal(t, StateVerifying, lastStatus)
	require.NotNil(t, lastTaskTemplateID, "task.TemplateID 应被写入")
	assert.Equal(t, tmpl.ID, *lastTaskTemplateID)
}

func TestDispatchTemplate_BypassesAutoConfigureGate(t *testing.T) {
	// HandleBootstrap 在 AutoConfigure=false 时会跳过 Path A（engine.go:253）；
	// 显式 DispatchTemplate 不查 AutoConfigure，应正常入队。
	h := newFullEngineHarness()
	h.engine.config.AutoConfigure = false // 显式禁用，验证 dispatch 不受影响

	deviceID := uuid.New()
	h.devRepo.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID: deviceID, SerialNumber: "SN-DISP-002",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
			OUI: "001122", ProductClass: "BLQ",
		}, nil
	}
	var enqueued []*task.CreateTaskRequest
	h.cmdQueue.CreateFn = func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		enqueued = append(enqueued, req)
		return task.NewTask(req), nil
	}

	tmpl := activeTmpl(t, `{"Device.WiFi.SSID":"x"}`)
	_, err := h.engine.DispatchTemplate(context.Background(), tmpl, deviceID)
	require.NoError(t, err)
	require.NotEmpty(t, enqueued, "AutoConfigure=false 时显式 dispatch 仍应入队")
}

func TestDispatchTemplate_DeviceLookupError(t *testing.T) {
	h := newFullEngineHarness()
	h.devRepo.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return nil, errors.New("db down")
	}
	tmpl := activeTmpl(t, `{"x":"y"}`)
	_, err := h.engine.DispatchTemplate(context.Background(), tmpl, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}
