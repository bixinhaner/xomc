// Package provision — service.go provides a thin facade over the existing
// ProvisioningEngine, orchestrator helpers, state machine, and task repository.
//
// Rationale: the provision module already owns rich logic split across
// engine.go (bootstrap workflow + 4-path branching), orchestrator.go (step
// generation + enqueue), state_machine.go (transition rules), and
// pg_repository.go (persistence). Callers outside the package historically
// had to know which collaborator to call for which action. The Service type
// here unifies the high-level operations behind a single ProvisionService
// interface so callers (router DI, future RPC bridges, tests) can depend on
// one façade rather than three concrete types.
//
// The facade does NOT reimplement provisioning logic. It only:
//   - looks up a device via DeviceService and forwards bootstrap-style
//     triggers into the engine;
//   - exposes task lifecycle queries (get / list / state) by delegating to
//     the existing ProvisioningTaskRepository;
//   - exposes a manual retry path that creates a fresh task for a previously
//     failed device, mirroring the HTTP handler behaviour;
//   - exposes the state machine helpers (`ValidateTransition`,
//     `IsTerminal`, `NextState`) by re-exporting them as methods, so
//     consumers do not have to import package-level helpers separately.
//
// All real provisioning work continues to flow through ProvisioningEngine.
package provision

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/device"
)

// ErrDeviceNotFound is returned when the requested device serial number or ID
// cannot be located by the underlying device service.
var ErrDeviceNotFound = errors.New("provision: device not found")

// ErrTaskNotFound is returned when a task lookup yields no row.
var ErrTaskNotFound = errors.New("provision: task not found")

// ErrTaskNotRetryable is returned when a caller attempts to retry a task that
// is not in StateFailed.
var ErrTaskNotRetryable = errors.New("provision: only failed tasks can be retried")

// engineHandle is the minimum slice of ProvisioningEngine that the facade
// invokes. Defining it as a small interface keeps the public Service
// constructible with a mock engine in tests.
type engineHandle interface {
	HandleBootstrap(ctx context.Context, evt bootstrapEvent) error
	HandleRPCResult(ctx context.Context, deviceSN string, method string, success bool, errMsg string) error
}

// deviceLookup is the slice of device.DeviceService used by the facade.
// Defined locally so unit tests can stub without spinning up the full
// DeviceService graph.
type deviceLookup interface {
	GetDevice(ctx context.Context, id uuid.UUID) (*deviceForProvision, error)
	GetBySerialNumber(ctx context.Context, sn string) (*deviceForProvision, error)
}

// deviceForProvision captures only the fields the facade needs from
// model.Device. Pulling them into a local type keeps the lookup interface
// independent of the model package's surface area.
type deviceForProvision struct {
	ID           uuid.UUID
	SerialNumber string
	OUI          string
	ProductClass string
	Carrier      string
	Technology   string
}

// realDeviceLookup adapts *device.DeviceService to the local deviceLookup
// interface, projecting the small handful of fields the facade needs.
type realDeviceLookup struct {
	svc *device.DeviceService
}

func (r *realDeviceLookup) GetDevice(ctx context.Context, id uuid.UUID) (*deviceForProvision, error) {
	dev, err := r.svc.GetDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, ErrDeviceNotFound
	}
	return &deviceForProvision{
		ID:           dev.ID,
		SerialNumber: dev.SerialNumber,
		OUI:          dev.OUI,
		ProductClass: dev.ProductClass,
		Carrier:      string(dev.Carrier),
		Technology:   string(dev.Technology),
	}, nil
}

func (r *realDeviceLookup) GetBySerialNumber(ctx context.Context, sn string) (*deviceForProvision, error) {
	dev, err := r.svc.GetBySerialNumber(ctx, sn)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, ErrDeviceNotFound
	}
	return &deviceForProvision{
		ID:           dev.ID,
		SerialNumber: dev.SerialNumber,
		OUI:          dev.OUI,
		ProductClass: dev.ProductClass,
		Carrier:      string(dev.Carrier),
		Technology:   string(dev.Technology),
	}, nil
}

// ProvisionService is the high-level façade exposed to other modules.
type ProvisionService interface {
	// DiscoverDevice resolves a device by serial number. It is a thin
	// pass-through to the device service that returns ErrDeviceNotFound
	// instead of nil so callers can distinguish a miss from an empty row.
	DiscoverDevice(ctx context.Context, serialNumber string) (*deviceForProvision, error)

	// TriggerProvisioning starts a fresh provisioning workflow for the
	// given device by synthesising a bootstrap-style event and forwarding
	// it to the engine. This is the manual / on-demand variant of the
	// automatic engine.Subscribe flow.
	TriggerProvisioning(ctx context.Context, deviceID uuid.UUID) error

	// GetTaskState returns the current ProvisioningState for the most
	// recent task associated with a device. Returns ErrTaskNotFound if
	// the device has never been provisioned.
	GetTaskState(ctx context.Context, deviceID uuid.UUID) (ProvisioningState, error)

	// GetTask returns a single ProvisioningTask by its ID.
	GetTask(ctx context.Context, taskID uuid.UUID) (*ProvisioningTask, error)

	// ListTasks proxies the repository ListProvisioningTasks call.
	ListTasks(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error)

	// RetryFailedTask creates a new provisioning task for the device of
	// the given failed task. The new task starts in StateDiscovered, just
	// like NewProvisioningTask. Returns ErrTaskNotRetryable when the
	// referenced task is not in StateFailed.
	RetryFailedTask(ctx context.Context, taskID uuid.UUID) (*ProvisioningTask, error)

	// HandleRPCResult is a façade pass-through used by upstream
	// task-completion routers. Kept on the interface so consumers can
	// depend solely on ProvisionService.
	HandleRPCResult(ctx context.Context, deviceSN string, method string, success bool, errMsg string) error

	// IsTerminalState exposes the state machine helper through the
	// façade so callers do not have to import the package-level helper.
	IsTerminalState(state ProvisioningState) bool

	// ValidateStateTransition exposes the state machine helper.
	ValidateStateTransition(current, target ProvisioningState) error
}

// Service is the concrete implementation of ProvisionService. It composes
// the existing engine + repository + device lookup without owning their
// dependencies.
type Service struct {
	engine engineHandle
	repo   ProvisioningTaskRepository
	devs   deviceLookup
}

// NewService constructs a Service from the production collaborators. Any of
// the dependencies may be nil-checked at the call site; passing nil here is
// allowed for partially-initialised wiring (e.g. tests that only exercise
// state-machine helpers).
func NewService(engine *ProvisioningEngine, repo ProvisioningTaskRepository, devSvc *device.DeviceService) *Service {
	var devs deviceLookup
	if devSvc != nil {
		devs = &realDeviceLookup{svc: devSvc}
	}
	var eng engineHandle
	if engine != nil {
		eng = engine
	}
	return &Service{
		engine: eng,
		repo:   repo,
		devs:   devs,
	}
}

// newServiceWithMocks is the test-only constructor that takes the smaller
// interfaces directly. Lower-case so it does not leak into the public API.
func newServiceWithMocks(engine engineHandle, repo ProvisioningTaskRepository, devs deviceLookup) *Service {
	return &Service{engine: engine, repo: repo, devs: devs}
}

// DiscoverDevice implements ProvisionService.
func (s *Service) DiscoverDevice(ctx context.Context, serialNumber string) (*deviceForProvision, error) {
	if s.devs == nil {
		return nil, fmt.Errorf("provision service: device lookup not configured")
	}
	if serialNumber == "" {
		return nil, fmt.Errorf("provision service: empty serial number")
	}
	return s.devs.GetBySerialNumber(ctx, serialNumber)
}

// TriggerProvisioning implements ProvisionService.
func (s *Service) TriggerProvisioning(ctx context.Context, deviceID uuid.UUID) error {
	if s.engine == nil {
		return fmt.Errorf("provision service: engine not configured")
	}
	if s.devs == nil {
		return fmt.Errorf("provision service: device lookup not configured")
	}
	dev, err := s.devs.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("trigger provisioning: %w", err)
	}
	evt := bootstrapEvent{
		DeviceID:     dev.ID,
		SerialNumber: dev.SerialNumber,
		OUI:          dev.OUI,
		ProductClass: dev.ProductClass,
		Carrier:      dev.Carrier,
		Technology:   dev.Technology,
	}
	return s.engine.HandleBootstrap(ctx, evt)
}

// GetTaskState implements ProvisionService.
func (s *Service) GetTaskState(ctx context.Context, deviceID uuid.UUID) (ProvisioningState, error) {
	if s.repo == nil {
		return "", fmt.Errorf("provision service: repository not configured")
	}
	task, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return "", fmt.Errorf("get task state: %w", err)
	}
	if task == nil {
		return "", ErrTaskNotFound
	}
	return task.Status, nil
}

// GetTask implements ProvisionService.
func (s *Service) GetTask(ctx context.Context, taskID uuid.UUID) (*ProvisioningTask, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("provision service: repository not configured")
	}
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// ListTasks implements ProvisionService.
func (s *Service) ListTasks(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
	if s.repo == nil {
		return nil, 0, fmt.Errorf("provision service: repository not configured")
	}
	return s.repo.List(ctx, filter)
}

// RetryFailedTask implements ProvisionService.
func (s *Service) RetryFailedTask(ctx context.Context, taskID uuid.UUID) (*ProvisioningTask, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("provision service: repository not configured")
	}
	existing, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("retry task: %w", err)
	}
	if existing == nil {
		return nil, ErrTaskNotFound
	}
	if existing.Status != StateFailed {
		return nil, ErrTaskNotRetryable
	}
	newTask := NewProvisioningTask(existing.DeviceID)
	if err := s.repo.Create(ctx, newTask); err != nil {
		return nil, fmt.Errorf("retry task: create: %w", err)
	}
	return newTask, nil
}

// HandleRPCResult implements ProvisionService.
func (s *Service) HandleRPCResult(ctx context.Context, deviceSN string, method string, success bool, errMsg string) error {
	if s.engine == nil {
		return fmt.Errorf("provision service: engine not configured")
	}
	return s.engine.HandleRPCResult(ctx, deviceSN, method, success, errMsg)
}

// IsTerminalState implements ProvisionService.
func (s *Service) IsTerminalState(state ProvisioningState) bool {
	return IsTerminal(state)
}

// ValidateStateTransition implements ProvisionService.
func (s *Service) ValidateStateTransition(current, target ProvisioningState) error {
	return ValidateTransition(current, target)
}

// Compile-time guarantee that *Service satisfies ProvisionService.
var _ ProvisionService = (*Service)(nil)
