package mml

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Service provides business logic for the MML console module.
type Service struct {
	cmdRepo    CommandRepository
	scriptRepo ScriptRepository
	taskRepo   TaskRepository
	logger     *zap.Logger
}

// NewService creates a new MML Service.
func NewService(
	cmdRepo CommandRepository,
	scriptRepo ScriptRepository,
	taskRepo TaskRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		cmdRepo:    cmdRepo,
		scriptRepo: scriptRepo,
		taskRepo:   taskRepo,
		logger:     logger.Named("mml"),
	}
}

// ---- Command operations (read-only) ----

// ListCommands returns a paginated list of predefined MML commands.
func (s *Service) ListCommands(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	return s.cmdRepo.List(ctx, filter)
}

// GetCommand retrieves a predefined MML command by ID.
func (s *Service) GetCommand(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
	return s.cmdRepo.GetByID(ctx, id)
}

// GetCommandByCode retrieves a predefined MML command by its command code.
func (s *Service) GetCommandByCode(ctx context.Context, code string) (*MMLCommand, error) {
	return s.cmdRepo.GetByCode(ctx, code)
}

// ---- Script operations (CRUD) ----

// CreateScript creates a new user-defined MML script.
func (s *Service) CreateScript(ctx context.Context, script *MMLScript) (*MMLScript, error) {
	if script.Tags == nil {
		script.Tags = []string{}
	}

	if err := s.scriptRepo.Create(ctx, script); err != nil {
		return nil, fmt.Errorf("create mml script: %w", err)
	}

	s.logger.Info("mml script created",
		zap.String("script_id", script.ID.String()),
		zap.String("script_name", script.ScriptName),
	)

	return script, nil
}

// GetScript retrieves an MML script by ID.
func (s *Service) GetScript(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	return s.scriptRepo.GetByID(ctx, id)
}

// UpdateScript updates an existing MML script.
func (s *Service) UpdateScript(ctx context.Context, id uuid.UUID, script *MMLScript) (*MMLScript, error) {
	existing, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml script: %w", err)
	}

	// Apply updates
	existing.ScriptName = script.ScriptName
	existing.Description = script.Description
	existing.Content = script.Content
	existing.DeviceType = script.DeviceType
	if script.Tags != nil {
		existing.Tags = script.Tags
	}

	if err := s.scriptRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update mml script: %w", err)
	}

	s.logger.Info("mml script updated",
		zap.String("script_id", id.String()),
	)

	return existing, nil
}

// DeleteScript deletes an MML script by ID.
func (s *Service) DeleteScript(ctx context.Context, id uuid.UUID) error {
	return s.scriptRepo.Delete(ctx, id)
}

// ListScripts returns a paginated list of MML scripts.
func (s *Service) ListScripts(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error) {
	return s.scriptRepo.List(ctx, filter)
}

// ---- Task / Execution operations ----

// ExecuteRequest defines the parameters for executing an MML command.
type ExecuteRequest struct {
	CommandCode string                   `json:"command_code"`
	DeviceSNs   []string                 `json:"device_sns"`
	Parameters  map[string]interface{}   `json:"parameters"`
	TaskName    string                   `json:"task_name"`
	Creator     string                   `json:"creator"`
	Commands    []map[string]interface{} `json:"commands"`

	// Scheduling
	ExecuteType ExecuteType `json:"execute_type"`
	ScheduledAt *string     `json:"scheduled_at"`
	PeriodStart *string     `json:"period_start"`
	PeriodEnd   *string     `json:"period_end"`
	PeriodTime  string      `json:"period_time"`

	// Retry strategy
	OfflineRetry        bool `json:"offline_retry"`
	OfflineRetryWait    int  `json:"offline_retry_wait"`
	FailedRetry         bool `json:"failed_retry"`
	FailedRetryCount    int  `json:"failed_retry_count"`
	FailedRetryInterval int  `json:"failed_retry_interval"`
}

// ExecuteCommand creates an MML task with pending status.
// Real execution through cmdQueue to ACS is a future integration.
func (s *Service) ExecuteCommand(ctx context.Context, req ExecuteRequest) (*MMLTask, error) {
	// Build the commands list from the request
	commands := req.Commands
	if commands == nil {
		commands = []map[string]interface{}{}
	}

	// If a command_code is provided, resolve it and build the command entry
	if req.CommandCode != "" {
		cmd, err := s.cmdRepo.GetByCode(ctx, req.CommandCode)
		if err != nil {
			return nil, fmt.Errorf("resolve command code %q: %w", req.CommandCode, err)
		}

		entry := map[string]interface{}{
			"command_code": cmd.CommandCode,
			"rpc_method":   cmd.RPCMethod,
			"parameters":   req.Parameters,
		}
		commands = append(commands, entry)
	}

	task := &MMLTask{
		TaskName:  req.TaskName,
		DeviceSNs: req.DeviceSNs,
		Commands:  commands,
		Status:    TaskPending,
		Results:   []map[string]interface{}{},
		Creator:   req.Creator,

		ExecuteType:         req.ExecuteType,
		OfflineRetry:        req.OfflineRetry,
		OfflineRetryWait:    req.OfflineRetryWait,
		FailedRetry:         req.FailedRetry,
		FailedRetryCount:    req.FailedRetryCount,
		FailedRetryInterval: req.FailedRetryInterval,
		TotalDevices:        len(req.DeviceSNs),
	}

	// Map execute_type to initial status
	switch req.ExecuteType {
	case ExecuteSuspended:
		task.Status = TaskPaused
	case ExecuteScheduled, ExecutePeriodic:
		task.Status = TaskPending
	default:
		task.Status = TaskPending
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create mml task: %w", err)
	}

	s.logger.Info("mml task created",
		zap.String("task_id", task.ID.String()),
		zap.String("task_name", task.TaskName),
		zap.Int("device_count", len(task.DeviceSNs)),
	)

	return task, nil
}

// GetTask retrieves an MML task by ID.
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	return s.taskRepo.GetByID(ctx, id)
}

// ListTasks returns a paginated list of MML tasks.
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
	return s.taskRepo.List(ctx, filter)
}

// ---- Task control operations (Phase 2) ----

var (
	// ErrInvalidTransition is returned when a task status transition is not allowed.
	ErrInvalidTransition = errors.New("invalid task status transition")
	// ErrCannotDeleteRunning is returned when trying to delete a running task.
	ErrCannotDeleteRunning = errors.New("cannot delete a running task")
)

// StartTask transitions a task from pending/paused to running.
func (s *Service) StartTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	if task.Status != TaskPending && task.Status != TaskPaused {
		return nil, fmt.Errorf("start task: status %s cannot transition to running: %w", task.Status, ErrInvalidTransition)
	}

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskRunning); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskRunning
	s.logger.Info("mml task started", zap.String("task_id", id.String()))
	return task, nil
}

// PauseTask transitions a running task to paused.
func (s *Service) PauseTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	if task.Status != TaskRunning {
		return nil, fmt.Errorf("pause task: status %s cannot transition to paused: %w", task.Status, ErrInvalidTransition)
	}

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskPaused); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskPaused
	s.logger.Info("mml task paused", zap.String("task_id", id.String()))
	return task, nil
}

// CancelTask transitions any task to cancelled.
func (s *Service) CancelTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	if task.Status == TaskCancelled || task.Status == TaskCompleted || task.Status == TaskFailed {
		return nil, fmt.Errorf("cancel task: status %s cannot transition to cancelled: %w", task.Status, ErrInvalidTransition)
	}

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskCancelled); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskCancelled
	s.logger.Info("mml task cancelled", zap.String("task_id", id.String()))
	return task, nil
}

// DeleteTask removes a non-running task.
func (s *Service) DeleteTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get mml task: %w", err)
	}

	if task.Status == TaskRunning {
		return ErrCannotDeleteRunning
	}

	if err := s.taskRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete mml task: %w", err)
	}

	s.logger.Info("mml task deleted", zap.String("task_id", id.String()))
	return nil
}
