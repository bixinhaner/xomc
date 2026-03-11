package mml

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/model"
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
