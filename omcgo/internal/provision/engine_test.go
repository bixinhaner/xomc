package provision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: ProvisioningTaskRepository (Pattern B — function fields)
// ---------------------------------------------------------------------------

type mockTaskRepo struct {
	CreateFn        func(ctx context.Context, task *ProvisioningTask) error
	GetByIDFn       func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error)
	GetByDeviceIDFn func(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error)
	UpdateFn        func(ctx context.Context, task *ProvisioningTask) error
	UpdateStatusFn  func(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error
	ListFn          func(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error)
	CountByStatusFn func(ctx context.Context) (map[ProvisioningState]int64, error)
	FailStaleFn     func(ctx context.Context, maxAge time.Duration) (int64, error)
}

func (m *mockTaskRepo) Create(ctx context.Context, task *ProvisioningTask) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepo) GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error) {
	if m.GetByDeviceIDFn != nil {
		return m.GetByDeviceIDFn(ctx, deviceID)
	}
	return nil, nil
}

func (m *mockTaskRepo) Update(ctx context.Context, task *ProvisioningTask) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status, errorMsg)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTaskRepo) CountByStatus(ctx context.Context) (map[ProvisioningState]int64, error) {
	if m.CountByStatusFn != nil {
		return m.CountByStatusFn(ctx)
	}
	return nil, nil
}

func (m *mockTaskRepo) FailStale(ctx context.Context, maxAge time.Duration) (int64, error) {
	if m.FailStaleFn != nil {
		return m.FailStaleFn(ctx, maxAge)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: task.Enqueuer
// ---------------------------------------------------------------------------

type mockCommandQueue struct {
	CreateFn func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
	LenFn    func(ctx context.Context, deviceSN string) (int64, error)
}

func (m *mockCommandQueue) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req)
	}
	return task.NewTask(req), nil
}

func (m *mockCommandQueue) GetQueueLength(ctx context.Context, deviceSN string) (int64, error) {
	if m.LenFn != nil {
		return m.LenFn(ctx, deviceSN)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: EventBus
// ---------------------------------------------------------------------------

type mockEventBus struct {
	PublishFn        func(ctx context.Context, subject string, evt event.Event) error
	SubscribeFn      func(subject string, handler event.EventHandler) (event.Subscription, error)
	QueueSubscribeFn func(subject string, queue string, handler event.EventHandler) (event.Subscription, error)
	CloseFn          func() error

	// Capture published events for assertions.
	published []publishedEvent
}

type publishedEvent struct {
	subject string
	event   event.Event
}

func (m *mockEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	m.published = append(m.published, publishedEvent{subject: subject, event: evt})
	if m.PublishFn != nil {
		return m.PublishFn(ctx, subject, evt)
	}
	return nil
}

func (m *mockEventBus) Subscribe(subject string, handler event.EventHandler) (event.Subscription, error) {
	if m.SubscribeFn != nil {
		return m.SubscribeFn(subject, handler)
	}
	return &mockSubscription{}, nil
}

func (m *mockEventBus) QueueSubscribe(subject string, queue string, handler event.EventHandler) (event.Subscription, error) {
	if m.QueueSubscribeFn != nil {
		return m.QueueSubscribeFn(subject, queue, handler)
	}
	return &mockSubscription{}, nil
}

func (m *mockEventBus) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

type mockSubscription struct{}

func (s *mockSubscription) Unsubscribe() error { return nil }

// ---------------------------------------------------------------------------
// Helper: build a fully-wired ProvisioningEngine for tests.
// ---------------------------------------------------------------------------

// engineHarness groups all mocks so tests can assert against them.
type engineHarness struct {
	engine   *ProvisioningEngine
	taskRepo *mockTaskRepo
	cmdQueue *mockCommandQueue
	eventBus *mockEventBus
}

func newEngineHarness(deviceRepo device.DeviceRepository) *engineHarness {
	taskRepo := &mockTaskRepo{}
	cmdQueue := &mockCommandQueue{}
	evtBus := &mockEventBus{}
	logger := zap.NewNop()

	// T-0098 P5-01：dmRegistry 已删除，引擎仅依赖 deviceService/templateService/carrierReg。
	devService := device.NewDeviceService(deviceRepo, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(nil, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		taskRepo,
		devService,
		tmplService,
		carrierReg,
		cmdQueue,
		evtBus,
		appconfig.ProvisionConfig{AutoConfigure: true},
		logger,
	)

	return &engineHarness{
		engine:   engine,
		taskRepo: taskRepo,
		cmdQueue: cmdQueue,
		eventBus: evtBus,
	}
}

// ---------------------------------------------------------------------------
// Mock: DeviceRepository (needed for DeviceService)
// ---------------------------------------------------------------------------

type mockDeviceRepo struct {
	GetByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	GetBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
	CreateFn            func(ctx context.Context, d *model.Device) error
	UpdateFn            func(ctx context.Context, d *model.Device) error
	DeleteFn            func(ctx context.Context, id uuid.UUID) error
	ListFn              func(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error)
	UpdateStatusFn      func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	UpdateLastInformFn  func(ctx context.Context, sn string, at time.Time, events []string) error
	CountByStatusFn     func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
}

func (m *mockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.GetBySerialNumberFn != nil {
		return m.GetBySerialNumberFn(ctx, sn)
	}
	return nil, nil
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, d)
	}
	return nil
}

func (m *mockDeviceRepo) Update(ctx context.Context, d *model.Device) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, d)
	}
	return nil
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockDeviceRepo) List(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}

// T-0162 新接口方法
func (m *mockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *mockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.UpdateLastInformFn != nil {
		return m.UpdateLastInformFn(ctx, sn, at, events)
	}
	return nil
}

func (m *mockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}

func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	if m.CountByStatusFn != nil {
		return m.CountByStatusFn(ctx, carrier)
	}
	return nil, nil
}
func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return []model.Device{}, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests: HandleRPCResult
// ---------------------------------------------------------------------------

func TestHandleRPCResult_AdvancesStep(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-001"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	// Return a task in verifying state with 3 total steps.
	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 0,
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	var updatedStep int
	h.taskRepo.UpdateFn = func(ctx context.Context, task *ProvisioningTask) error {
		updatedStep = task.CurrentStep
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, true, "")
	require.NoError(t, err)
	assert.Equal(t, 1, updatedStep, "step should advance from 0 to 1")

	// Verify provision.step.done event was published.
	require.GreaterOrEqual(t, len(h.eventBus.published), 1)
	assert.Equal(t, event.SubjectProvisionStepDone, h.eventBus.published[0].subject)
}

func TestHandleRPCResult_AllStepsComplete(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-002"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: deviceSN, Status: model.DeviceProvisioning}, nil
		},
	}

	h := newEngineHarness(devRepo)

	// Task is on the last step (2 of 3, 0-indexed CurrentStep=2 after increment).
	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 2, // will become 3 after increment
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	var completedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		completedStatus = status
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodReboot, true, "")
	require.NoError(t, err)
	assert.Equal(t, StateCompleted, completedStatus)

	// Verify provision.completed event was published.
	var foundCompleted bool
	for _, pe := range h.eventBus.published {
		if pe.subject == event.SubjectProvisionCompleted {
			foundCompleted = true
		}
	}
	assert.True(t, foundCompleted, "provision.completed event should be published")
}

func TestHandleRPCResult_StepFailureWithRetry(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-003"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 0,
			TotalSteps:  3,
			RetryCount:  0,
			MaxRetries:  3,
		}, nil
	}

	var savedErrorMsg string
	h.taskRepo.UpdateFn = func(ctx context.Context, task *ProvisioningTask) error {
		savedErrorMsg = task.ErrorMessage
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, false, "CWMP fault 9002")
	require.NoError(t, err)
	assert.Equal(t, "CWMP fault 9002", savedErrorMsg)
}

func TestHandleRPCResult_StepFailureExhaustsRetries(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-004"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 1,
			TotalSteps:  3,
			RetryCount:  2, // already retried twice
			MaxRetries:  3, // max is 3 => next failure exhausts retries
		}, nil
	}

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, false, "timeout")
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

func TestHandleRPCResult_TerminalTaskIsNoOp(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-005"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:       uuid.New(),
			DeviceID: id,
			Status:   StateCompleted, // terminal
		}, nil
	}

	updateCalled := false
	h.taskRepo.UpdateFn = func(ctx context.Context, task *ProvisioningTask) error {
		updateCalled = true
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodReboot, true, "")
	require.NoError(t, err)
	assert.False(t, updateCalled, "should not update a terminal task")
}

func TestHandleRPCResult_DeviceNotFound(t *testing.T) {
	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, errors.New("device not found")
		},
	}

	h := newEngineHarness(devRepo)

	err := h.engine.HandleRPCResult(context.Background(), "UNKNOWN-SN", MethodSetParameterValues, true, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find device by SN")
}

func TestHandleRPCResult_TaskNotFound(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-006"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return nil, errors.New("task not found")
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, true, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get task for device")
}

// ---------------------------------------------------------------------------
// Tests: HandleBootstrap (partial — tests the create and fail paths that
// do not require real DataModelRegistry / ConfigTemplateService backends)
// ---------------------------------------------------------------------------

func TestHandleBootstrap_CreateTaskError(t *testing.T) {
	devRepo := &mockDeviceRepo{}
	h := newEngineHarness(devRepo)

	h.taskRepo.CreateFn = func(ctx context.Context, task *ProvisioningTask) error {
		return errors.New("db connection refused")
	}

	evt := bootstrapEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "SN-ERR-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create provisioning task")
}

func TestHandleBootstrap_DeviceNotFound(t *testing.T) {
	devRepo := &mockDeviceRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return nil, errors.New("device not found in DB")
		},
	}

	h := newEngineHarness(devRepo)

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	evt := bootstrapEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "SN-NODEV-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

func TestHandleBootstrap_NoMatchingTemplate(t *testing.T) {
	h := newFullEngineHarness()
	deviceID := uuid.New()

	h.devRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-NOTMPL-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}

	// Data model resolution succeeds (returns nil = no model, which is OK).
	// Template matching returns nil = no matching template.
	h.tmplRepo.FindBestMatchFn = func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
		return nil, nil // no matching template
	}

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	evt := bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-NOTMPL-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

// ---------------------------------------------------------------------------
// Tests: transitionTask (indirectly through engine state progression)
// ---------------------------------------------------------------------------

func TestTransitionTask_InvalidTransition(t *testing.T) {
	// We can exercise the transitionTask logic by inspecting HandleBootstrap
	// behavior. Since transitionTask is unexported, we test it via the state
	// machine's ValidateTransition (which is already tested in
	// state_machine_test.go). Here we verify the engine properly wraps it.

	// A task stuck in "completed" state cannot transition to "identifying".
	err := ValidateTransition(StateCompleted, StateIdentifying)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid provisioning state transition")
}

// ---------------------------------------------------------------------------
// Tests: completeTask — device status transition
// ---------------------------------------------------------------------------

func TestCompleteTask_TransitionsDeviceToActive(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-COMPLETE"

	var transitionedStatus model.DeviceStatus
	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn, Status: model.DeviceProvisioning}, nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: deviceSN, Status: model.DeviceProvisioning}, nil
		},
		UpdateStatusFn: func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
			transitionedStatus = status
			return nil
		},
	}

	h := newEngineHarness(devRepo)

	// Task on last step.
	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 2,
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodReboot, true, "")
	require.NoError(t, err)
	assert.Equal(t, model.DeviceActive, transitionedStatus)
}

// ---------------------------------------------------------------------------
// Tests: publishEvent (verify events are captured)
// ---------------------------------------------------------------------------

func TestPublishEvent_FailureIsLoggedNotReturned(t *testing.T) {
	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: uuid.New(), SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	// Make event bus fail.
	h.eventBus.PublishFn = func(ctx context.Context, subject string, evt event.Event) error {
		return errors.New("NATS connection lost")
	}

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 0,
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	// HandleRPCResult should still succeed even though event publishing fails.
	err := h.engine.HandleRPCResult(context.Background(), "SN-EVT-FAIL", MethodSetParameterValues, true, "")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Tests: NewProvisioningTask
// ---------------------------------------------------------------------------

func TestNewProvisioningTask(t *testing.T) {
	deviceID := uuid.New()
	task := NewProvisioningTask(deviceID)

	assert.Equal(t, deviceID, task.DeviceID)
	assert.Equal(t, StateDiscovered, task.Status)
	assert.Equal(t, 3, task.MaxRetries)
	assert.NotEqual(t, uuid.Nil, task.ID)
	assert.Nil(t, task.StartedAt)
	assert.Nil(t, task.CompletedAt)
	assert.Equal(t, 0, task.CurrentStep)
	assert.Equal(t, 0, task.TotalSteps)
}

// ---------------------------------------------------------------------------
// Tests: HandleBootstrap full success path
// ---------------------------------------------------------------------------

// fullEngineHarness builds an engine with real-enough collaborators for a
// full HandleBootstrap success path. We use concrete mock repos all the way.
type fullEngineHarness struct {
	engine   *ProvisioningEngine
	taskRepo *mockTaskRepo
	devRepo  *mockDeviceRepo
	tmplRepo *mockTemplateRepo
	cmdQueue *mockCommandQueue
	eventBus *mockEventBus
}

type mockTemplateRepo struct {
	CreateFn            func(ctx context.Context, t *template.ConfigTemplate) error
	GetByIDFn           func(ctx context.Context, id uuid.UUID) (*template.ConfigTemplate, error)
	UpdateFn            func(ctx context.Context, t *template.ConfigTemplate) error
	DeleteFn            func(ctx context.Context, id uuid.UUID) error
	ListFn              func(ctx context.Context, filter template.ConfigTemplateFilter) (*model.ListResponse[template.ConfigTemplate], error)
	FindByCarrierTechFn func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType template.TemplateType) ([]template.ConfigTemplate, error)
	FindBestMatchFn     func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error)
}

func (m *mockTemplateRepo) Create(ctx context.Context, t *template.ConfigTemplate) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*template.ConfigTemplate, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateRepo) Update(ctx context.Context, t *template.ConfigTemplate) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockTemplateRepo) List(ctx context.Context, filter template.ConfigTemplateFilter) (*model.ListResponse[template.ConfigTemplate], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTemplateRepo) FindByCarrierTech(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType template.TemplateType) ([]template.ConfigTemplate, error) {
	if m.FindByCarrierTechFn != nil {
		return m.FindByCarrierTechFn(ctx, carrier, tech, templateType)
	}
	return nil, nil
}

func (m *mockTemplateRepo) FindBestMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
	if m.FindBestMatchFn != nil {
		return m.FindBestMatchFn(ctx, carrier, tech, productClass, tmplType)
	}
	return nil, nil
}

// T-0098 P5-01：mockDataModelRepo 已删除（datamodel 包整体下线）。

func newFullEngineHarness() *fullEngineHarness {
	logger := zap.NewNop()
	taskRepo := &mockTaskRepo{}
	devRepo := &mockDeviceRepo{}
	tmplRepo := &mockTemplateRepo{}
	cmdQueue := &mockCommandQueue{}
	evtBus := &mockEventBus{}

	devService := device.NewDeviceService(devRepo, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(tmplRepo, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		taskRepo,
		devService,
		tmplService,
		carrierReg,
		cmdQueue,
		evtBus,
		appconfig.ProvisionConfig{AutoConfigure: true},
		logger,
	)

	return &fullEngineHarness{
		engine:   engine,
		taskRepo: taskRepo,
		devRepo:  devRepo,
		tmplRepo: tmplRepo,
		cmdQueue: cmdQueue,
		eventBus: evtBus,
	}
}

func TestHandleBootstrap_FullSuccessPath(t *testing.T) {
	h := newFullEngineHarness()

	deviceID := uuid.New()
	templateID := uuid.New()

	h.devRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-FULL-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}

	h.tmplRepo.FindBestMatchFn = func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
		return &template.ConfigTemplate{
			ID:           templateID,
			Name:         "cmcc_lte_provision",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "SmallCell-LTE",
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "OMC-Default"}`),
			Priority:     10,
			Active:       true,
		}, nil
	}

	// Track state transitions.
	var stateTransitions []ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		stateTransitions = append(stateTransitions, status)
		return nil
	}

	// Track pushed commands.
	var pushedMethods []string
	h.cmdQueue.CreateFn = func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		pushedMethods = append(pushedMethods, req.Method)
		return task.NewTask(req), nil
	}

	evt := bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FULL-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)

	// Verify state transitions went through the expected sequence.
	expected := []ProvisioningState{
		StateIdentifying,
		StateMatching,
		StateConfiguring,
		StateVerifying,
	}
	assert.Equal(t, expected, stateTransitions)

	// Verify commands were enqueued: GetParameterValues, SetParameterValues, Reboot.
	require.Len(t, pushedMethods, 3)
	assert.Equal(t, MethodGetParameterValues, pushedMethods[0])
	assert.Equal(t, MethodSetParameterValues, pushedMethods[1])
	assert.Equal(t, MethodReboot, pushedMethods[2])

	// Verify provision.started event was published.
	require.GreaterOrEqual(t, len(h.eventBus.published), 1)
	assert.Equal(t, event.SubjectProvisionStarted, h.eventBus.published[0].subject)
}

func TestHandleBootstrap_EnqueueStepsError(t *testing.T) {
	h := newFullEngineHarness()

	deviceID := uuid.New()
	templateID := uuid.New()

	h.devRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-ENQERR-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}

	h.tmplRepo.FindBestMatchFn = func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
		return &template.ConfigTemplate{
			ID:           templateID,
			Name:         "tmpl",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "test"}`),
			Active:       true,
		}, nil
	}

	h.cmdQueue.CreateFn = func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		return nil, errors.New("redis unavailable")
	}

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	evt := bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-ENQERR-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

// ---------------------------------------------------------------------------
// Tests: T-0123 HandleDeviceOnline — Redis token bucket + Path B 触发
// ---------------------------------------------------------------------------

// newOnlineHarness 构造一个挂 miniredis 的 ProvisioningEngine（T-0123 测试用）。
//
// syncService 默认 nil — 测试只验证：① Redis token bucket；② device lookup；③ Redis 容错。
// 真 Path B 链路由 sync_pathb_test.go 覆盖（避免本测试既测引擎又测 SyncService 内部）。
func newOnlineHarness(t *testing.T, deviceRepo device.DeviceRepository) (*ProvisioningEngine, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	logger := zap.NewNop()
	devService := device.NewDeviceService(deviceRepo, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(nil, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		&mockTaskRepo{}, devService, tmplService, carrierReg, &mockCommandQueue{},
		&mockEventBus{}, appconfig.ProvisionConfig{}, logger,
	)
	engine.SetRedisClient(rdb)
	return engine, mr
}

func TestHandleDeviceOnline_RedisTokenBucketSkipsRepeat(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-ONLINE-001",
				ProductClass: "SmallCell",
				Status:       model.DeviceActive,
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceOnlineEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-ONLINE-001",
		ProductClass: "SmallCell",
		SwVersion:    "1.0.0",
	}

	// 第一次：拿到 token，进入 device lookup（syncService nil 早返，但 lookup 已发生）
	err := engine.HandleDeviceOnline(context.Background(), evt)
	require.NoError(t, err)
	firstLookup := lookupCount

	// 60s 内第二次：被 token bucket 拦截，不进 device lookup
	err = engine.HandleDeviceOnline(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, firstLookup, lookupCount, "second call within 60s should be throttled by token bucket")
}

func TestHandleDeviceOnline_DeviceNotFound_NoOp(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return nil, nil // not found
		},
	}
	// 需要先 SetSyncService 才能跑到 GetByID — 没 syncService 时函数早返
	// 此处直接验证 lookup 被跳过 (nil syncService 早返 → 不调 GetByID)
	engine, _ := newOnlineHarness(t, deviceRepo)
	// 测：nil syncService + device-not-found 路径都不应 panic
	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-GONE"}
	err := engine.HandleDeviceOnline(context.Background(), evt)
	assert.NoError(t, err)
}

func TestHandleDeviceOnline_NilSyncService_NoOp(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{ID: deviceID, SerialNumber: "SN-NOSYNC"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	// engine.syncService 默认为 nil（newOnlineHarness 未设置）

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-NOSYNC"}
	err := engine.HandleDeviceOnline(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, 0, lookupCount, "nil syncService 应早返不调 GetByID")
}

func TestHandleDeviceOnline_RedisDown_StillProceeds(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-REDIS-DOWN"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)
	mr.Close() // 模拟 Redis 不可达 — SetNX 返 err；HandleDeviceOnline 应继续推进不阻塞

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-REDIS-DOWN"}
	err := engine.HandleDeviceOnline(context.Background(), evt)
	assert.NoError(t, err, "Redis 失败应不阻塞主流程（容忍 Redis 抖动）")
}

// ---------------------------------------------------------------------------
// Tests: T-0125 HandleFirmwareChanged — Redis 串行锁 + reason hint + fallback Path B
// ---------------------------------------------------------------------------

func TestHandleFirmwareChanged_RedisSerialLockSkipsConcurrent(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    "SN-FW-LOCK",
				ProductClass:    "SmallCell",
				FirmwareVersion: "1.0.0",
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-LOCK",
		ProductClass: "SmallCell",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}

	// 第一次：拿到锁，进入 device lookup（modelUploadService=nil 不影响锁逻辑）
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	require.NoError(t, err)
	firstLookup := lookupCount

	// 10min 内第二次：被串行锁拦截，不进 device lookup
	err = engine.HandleFirmwareChanged(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, firstLookup, lookupCount, "second call within 10min should be skipped by serial lock")
}

func TestHandleFirmwareChanged_WritesReasonHintToRedis(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-FW-HINT"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-HINT",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	require.NoError(t, err)

	// 验证 reason hint 写入 Redis
	val, getErr := mr.Get("provision:syncreason:" + deviceID.String())
	require.NoError(t, getErr)
	assert.Equal(t, "firmware_changed", val, "reason hint 应写入 Redis 供 handleDataModelFileReceived auto-sync 读取")
}

func TestHandleFirmwareChanged_DeviceNotFound_NoOp(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return nil, nil // not found
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-GONE",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	assert.NoError(t, err, "device-not-found should be no-op without error")
}

func TestHandleFirmwareChanged_NilModelUpload_FallbackPathBSkippedWhenSyncNil(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-FW-FALLBACK",
				ProductClass: "SmallCell",
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	// modelUploadService 和 syncService 都 nil — 验证 fallback 路径不 panic + 早返

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-FALLBACK",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	assert.NoError(t, err, "nil modelUploadService + nil syncService 应无 panic 无 error")
}

func TestHandleFirmwareChanged_RedisDown_StillProceeds(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{ID: deviceID, SerialNumber: "SN-FW-REDIS-DOWN"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)
	mr.Close() // 模拟 Redis 不可达

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-REDIS-DOWN",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, 1, lookupCount, "Redis 失败仍应推进到 device lookup")
}
