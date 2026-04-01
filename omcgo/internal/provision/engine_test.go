package provision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
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
// Mock: CommandQueue
// ---------------------------------------------------------------------------

type mockCommandQueue struct {
	PushFn  func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error
	PopFn   func(ctx context.Context, deviceSN string) (*cmdqueue.Command, error)
	PeekFn  func(ctx context.Context, deviceSN string) (*cmdqueue.Command, error)
	LenFn   func(ctx context.Context, deviceSN string) (int64, error)
	ClearFn func(ctx context.Context, deviceSN string) error
}

func (m *mockCommandQueue) Push(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
	if m.PushFn != nil {
		return m.PushFn(ctx, deviceSN, cmd)
	}
	return nil
}

func (m *mockCommandQueue) Pop(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	if m.PopFn != nil {
		return m.PopFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockCommandQueue) Peek(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	if m.PeekFn != nil {
		return m.PeekFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockCommandQueue) Len(ctx context.Context, deviceSN string) (int64, error) {
	if m.LenFn != nil {
		return m.LenFn(ctx, deviceSN)
	}
	return 0, nil
}

func (m *mockCommandQueue) Clear(ctx context.Context, deviceSN string) error {
	if m.ClearFn != nil {
		return m.ClearFn(ctx, deviceSN)
	}
	return nil
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

	// Construct real (lightweight) collaborators that the engine expects.
	devService := device.NewDeviceService(deviceRepo, nil, nil, nil, logger)
	dmRegistry := datamodel.NewDataModelRegistry(nil, nil, logger)
	tmplService := template.NewConfigTemplateService(nil, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		taskRepo,
		devService,
		dmRegistry,
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

func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.UpdateLastInformFn != nil {
		return m.UpdateLastInformFn(ctx, sn, at, events)
	}
	return nil
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
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 retries")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get device")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching template")
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

type mockDataModelRepo struct {
	CreateFn     func(ctx context.Context, dm *datamodel.DataModel) error
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*datamodel.DataModel, error)
	UpdateFn     func(ctx context.Context, dm *datamodel.DataModel) error
	DeleteFn     func(ctx context.Context, id uuid.UUID) error
	ListFn       func(ctx context.Context, filter datamodel.DataModelFilter) (*model.ListResponse[datamodel.DataModel], error)
	FindActiveFn func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, oui, productClass string, scope model.DataModelScope) (*datamodel.DataModel, error)
	ActivateFn   func(ctx context.Context, id uuid.UUID) error
	DeprecateFn  func(ctx context.Context, id uuid.UUID) error
	StatisticsFn func(ctx context.Context) (*datamodel.DataModelStats, error)
}

func (m *mockDataModelRepo) Create(ctx context.Context, dm *datamodel.DataModel) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, dm)
	}
	return nil
}

func (m *mockDataModelRepo) GetByID(ctx context.Context, id uuid.UUID) (*datamodel.DataModel, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockDataModelRepo) Update(ctx context.Context, dm *datamodel.DataModel) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, dm)
	}
	return nil
}

func (m *mockDataModelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockDataModelRepo) List(ctx context.Context, filter datamodel.DataModelFilter) (*model.ListResponse[datamodel.DataModel], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockDataModelRepo) FindActive(ctx context.Context, carrier model.CarrierCode, tech model.Technology, oui, productClass string, scope model.DataModelScope) (*datamodel.DataModel, error) {
	if m.FindActiveFn != nil {
		return m.FindActiveFn(ctx, carrier, tech, oui, productClass, scope)
	}
	return nil, nil
}

func (m *mockDataModelRepo) Activate(ctx context.Context, id uuid.UUID) error {
	if m.ActivateFn != nil {
		return m.ActivateFn(ctx, id)
	}
	return nil
}

func (m *mockDataModelRepo) Deprecate(ctx context.Context, id uuid.UUID) error {
	if m.DeprecateFn != nil {
		return m.DeprecateFn(ctx, id)
	}
	return nil
}

func (m *mockDataModelRepo) Statistics(ctx context.Context) (*datamodel.DataModelStats, error) {
	if m.StatisticsFn != nil {
		return m.StatisticsFn(ctx)
	}
	return nil, nil
}

func (m *mockDataModelRepo) FindActiveWithFirmware(ctx context.Context, carrier model.CarrierCode, tech model.Technology, oui, productClass, firmwareVersion string, scope model.DataModelScope) (*datamodel.DataModel, error) {
	return nil, nil
}

func (m *mockDataModelRepo) FindActiveForMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology, oui, productClass, firmwareVersion string) (*datamodel.DataModel, error) {
	return nil, nil
}

func (m *mockDataModelRepo) TouchLastAccessed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockDataModelRepo) DeleteExpired(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error) {
	return 0, nil
}

func newFullEngineHarness() *fullEngineHarness {
	logger := zap.NewNop()
	taskRepo := &mockTaskRepo{}
	devRepo := &mockDeviceRepo{}
	tmplRepo := &mockTemplateRepo{}
	dmRepo := &mockDataModelRepo{}
	cmdQueue := &mockCommandQueue{}
	evtBus := &mockEventBus{}

	devService := device.NewDeviceService(devRepo, nil, nil, nil, logger)
	dmRegistry := datamodel.NewDataModelRegistry(dmRepo, nil, logger)
	tmplService := template.NewConfigTemplateService(tmplRepo, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		taskRepo,
		devService,
		dmRegistry,
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
	h.cmdQueue.PushFn = func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
		pushedMethods = append(pushedMethods, cmd.Method)
		return nil
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

	h.cmdQueue.PushFn = func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
		return errors.New("redis unavailable")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enqueue steps")
	assert.Equal(t, StateFailed, failedStatus)
}
