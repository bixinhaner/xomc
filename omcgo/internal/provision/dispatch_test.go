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

// ---------------------------------------------------------------------------
// T-0120-b: HandleBootstrap selector OR semantics
// （AutoDispatch=true 单独足以触发 Path A；AutoConfigure 既有语义不变）
// ---------------------------------------------------------------------------

// bootstrapHarnessWithTemplate seeds device + template into a full engine
// harness, captures pushed RPC methods, and lets each test set AutoConfigure
// + tmpl.AutoDispatch independently before running HandleBootstrap.
func bootstrapHarnessWithTemplate(t *testing.T, autoConfigure, autoDispatch bool) (*fullEngineHarness, *[]string) {
	t.Helper()
	h := newFullEngineHarness()
	h.engine.config.AutoConfigure = autoConfigure

	deviceID := uuid.New()
	tmplID := uuid.New()

	h.devRepo.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-T0120b",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}
	h.tmplRepo.FindBestMatchFn = func(_ context.Context, _ model.CarrierCode, _ model.Technology, _ string, _ template.TemplateType) (*template.ConfigTemplate, error) {
		return &template.ConfigTemplate{
			ID:           tmplID,
			Name:         "tmpl",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "SmallCell-LTE",
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{"Device.WiFi.SSID":"OMC"}`),
			Active:       true,
			AutoDispatch: autoDispatch,
		}, nil
	}
	pushedMethods := make([]string, 0)
	h.cmdQueue.CreateFn = func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		pushedMethods = append(pushedMethods, req.Method)
		return task.NewTask(req), nil
	}
	return h, &pushedMethods
}

func runBootstrap(t *testing.T, h *fullEngineHarness) {
	t.Helper()
	evt := bootstrapEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "SN-T0120b",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
		Carrier:      string(model.CarrierCMCC),
		Technology:   string(model.TechLTE),
	}
	// 借用 device GetByIDFn mock 的固定返回，evt.DeviceID 不影响后续 lookup。
	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
}

func TestHandleBootstrap_AutoDispatchTrue_TriggersPathA(t *testing.T) {
	// AutoConfigure=false（全局关闭）+ template.AutoDispatch=true（单条 opt-in）→ Path A
	h, pushed := bootstrapHarnessWithTemplate(t, false, true)
	runBootstrap(t, h)
	// Path A enqueues GPV + SPV + Reboot
	assert.Equal(t, []string{MethodGetParameterValues, MethodSetParameterValues, MethodReboot}, *pushed)
}

func TestHandleBootstrap_AutoConfigureTrue_PreservedSemantics(t *testing.T) {
	// AutoConfigure=true（既有全局开关）+ template.AutoDispatch=false → Path A（向后兼容）
	h, pushed := bootstrapHarnessWithTemplate(t, true, false)
	runBootstrap(t, h)
	assert.Equal(t, []string{MethodGetParameterValues, MethodSetParameterValues, MethodReboot}, *pushed)
}

func TestHandleBootstrap_BothFlagsTrue_StillPathA(t *testing.T) {
	// 两 flag 都 true → 仍走 Path A（OR 语义，幂等）
	h, pushed := bootstrapHarnessWithTemplate(t, true, true)
	runBootstrap(t, h)
	assert.Equal(t, []string{MethodGetParameterValues, MethodSetParameterValues, MethodReboot}, *pushed)
}

func TestHandleBootstrap_BothFlagsFalse_SkipsPathA(t *testing.T) {
	// 两 flag 都 false + 有匹配模板 → 跳过 Path A，落入 Path B/C；
	// 当前 harness 未注入 syncService / modelUploadService → 走 Path D fail。
	h, pushed := bootstrapHarnessWithTemplate(t, false, false)
	evt := bootstrapEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "SN-T0120b",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
		Carrier:      string(model.CarrierCMCC),
		Technology:   string(model.TechLTE),
	}
	_ = h.engine.HandleBootstrap(context.Background(), evt)
	// 关键断言：Path A 未触发（未入队 SPV）
	assert.Empty(t, *pushed, "two flags false + matching tmpl → Path A 必须跳过")
}

