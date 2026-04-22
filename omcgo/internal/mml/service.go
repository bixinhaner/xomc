package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Service provides business logic for the MML console module.
type Service struct {
	cmdRepo          CommandRepository
	scriptRepo       ScriptRepository
	taskRepo         TaskRepository
	customCommandRepo CustomCommandRepository
	auditRepo        AuditRepository
	cmdParamRepo     CommandParamRepository
	fanouter         *Fanouter
	hub              SSEPublisher
	logger           *zap.Logger
}

// SSEPublisher defines the interface for publishing SSE events.
// Implemented by *events.MessageHub; nil means SSE is disabled.
type SSEPublisher interface {
	PublishSimple(userID, eventType string, data []byte)
}

// NewService creates a new MML Service.
// hub may be nil if SSE is not configured.
// auditRepo may be nil if audit logging is not configured.
func NewService(
	cmdRepo CommandRepository,
	scriptRepo ScriptRepository,
	taskRepo TaskRepository,
	customCommandRepo CustomCommandRepository,
	hub SSEPublisher,
	logger *zap.Logger,
) *Service {
	return &Service{
		cmdRepo:          cmdRepo,
		scriptRepo:       scriptRepo,
		taskRepo:         taskRepo,
		customCommandRepo: customCommandRepo,
		hub:              hub,
		logger:           logger.Named("mml"),
	}
}

// SetAuditRepo sets the audit repository for command execution logging.
func (s *Service) SetAuditRepo(repo AuditRepository) {
	s.auditRepo = repo
}

// SetFanouter sets the fan-out engine for creating device_tasks from MML tasks.
func (s *Service) SetFanouter(f *Fanouter) {
	s.fanouter = f
}

// SetCmdParamRepo sets the command-param relationship repository.
func (s *Service) SetCmdParamRepo(repo CommandParamRepository) {
	s.cmdParamRepo = repo
}

// publishTaskStatus pushes a task status change event via SSE.
func (s *Service) publishTaskStatus(executor, taskID, oldStatus, newStatus string) {
	data, _ := json.Marshal(map[string]string{
		"task_id":    taskID,
		"old_status": oldStatus,
		"new_status": newStatus,
		"executor":   executor,
	})
	s.hub.PublishSimple(executor, "mml_task_status", data)
}

// ---- Command operations (read-only) ----

// ListCommands returns a paginated list of predefined MML commands, enriched with params.
func (s *Service) ListCommands(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	resp, err := s.cmdRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if s.cmdParamRepo != nil && len(resp.Items) > 0 {
		ids := make([]uuid.UUID, len(resp.Items))
		for i, cmd := range resp.Items {
			ids[i] = cmd.ID
		}
		paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, ids)
		if err != nil {
			s.logger.Warn("failed to load command params, skipping enrichment", zap.Error(err))
		} else {
			for i := range resp.Items {
				if params, ok := paramMap[resp.Items[i].ID]; ok {
					resp.Items[i].Params = params
				}
			}
		}
	}

	return resp, nil
}

// GetCommand retrieves a predefined MML command by ID, enriched with params.
func (s *Service) GetCommand(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
	cmd, err := s.cmdRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.cmdParamRepo != nil {
		paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, []uuid.UUID{id})
		if err != nil {
			s.logger.Warn("failed to load command params, skipping enrichment", zap.Error(err))
		} else if params, ok := paramMap[id]; ok {
			cmd.Params = params
		}
	}

	return cmd, nil
}

// GetCommandByCode retrieves a predefined MML command by its command code, enriched with params.
func (s *Service) GetCommandByCode(ctx context.Context, code string) (*MMLCommand, error) {
	cmd, err := s.cmdRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if s.cmdParamRepo != nil {
		paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, []uuid.UUID{cmd.ID})
		if err != nil {
			s.logger.Warn("failed to load command params, skipping enrichment", zap.Error(err))
		} else if params, ok := paramMap[cmd.ID]; ok {
			cmd.Params = params
		}
	}

	return cmd, nil
}

// CommandParamPath represents a TR-069 parameter path bound to an MML command.
type CommandParamPath struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	Writable bool   `json:"writable"`
}

// CommandParamPathsResponse defines the response payload for command parameter paths.
type CommandParamPathsResponse struct {
	CommandCode         string             `json:"command_code"`
	OperationType       string             `json:"operation_type"`
	SupportedOperations []string           `json:"supported_operations"`
	ParamPaths          []CommandParamPath `json:"param_paths"`
}

// GetCommandParamPaths retrieves TR-069 parameter paths for a predefined MML command.
func (s *Service) GetCommandParamPaths(ctx context.Context, id uuid.UUID) (*CommandParamPathsResponse, error) {
	cmd, err := s.GetCommand(ctx, id)
	if err != nil {
		return nil, err
	}

	supportedOperations := make([]string, 0, len(cmd.SupportedOperations))
	supportedOperations = append(supportedOperations, cmd.SupportedOperations...)

	paramPaths, err := normalizeCommandParamPaths(cmd.ParamPaths, cmd.OperationType, supportedOperations)
	if err != nil {
		return nil, fmt.Errorf("parse command param paths: %w", err)
	}

	return &CommandParamPathsResponse{
		CommandCode:         cmd.CommandCode,
		OperationType:       cmd.OperationType,
		SupportedOperations: supportedOperations,
		ParamPaths:          paramPaths,
	}, nil
}

func normalizeCommandParamPaths(raw json.RawMessage, operationType string, supportedOperations []string) ([]CommandParamPath, error) {
	if len(raw) == 0 {
		return []CommandParamPath{}, nil
	}

	defaultWritable := isWritableOperation(operationType)
	if !defaultWritable {
		for _, op := range supportedOperations {
			if isWritableOperation(op) {
				defaultWritable = true
				break
			}
		}
	}

	type rawCommandParamPath struct {
		Path     string `json:"path"`
		Label    string `json:"label"`
		Writable *bool  `json:"writable"`
	}

	var objectPaths []rawCommandParamPath
	if err := json.Unmarshal(raw, &objectPaths); err == nil {
		paramPaths := make([]CommandParamPath, 0, len(objectPaths))
		for _, item := range objectPaths {
			if item.Path == "" {
				continue
			}

			label := item.Label
			if label == "" {
				label = item.Path
			}

			writable := defaultWritable
			if item.Writable != nil {
				writable = *item.Writable
			}

			paramPaths = append(paramPaths, CommandParamPath{
				Path:     item.Path,
				Label:    label,
				Writable: writable,
			})
		}
		return paramPaths, nil
	}

	var stringPaths []string
	if err := json.Unmarshal(raw, &stringPaths); err != nil {
		return nil, err
	}

	paramPaths := make([]CommandParamPath, 0, len(stringPaths))
	for _, path := range stringPaths {
		if path == "" {
			continue
		}
		paramPaths = append(paramPaths, CommandParamPath{
			Path:     path,
			Label:    path,
			Writable: defaultWritable,
		})
	}
	return paramPaths, nil
}

func isWritableOperation(operation string) bool {
	switch strings.ToUpper(operation) {
	case "MOD", "ADD", "RMV", "ACT", "DEA", "RST", "CLR", "UPG":
		return true
	default:
		return false
	}
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
	Executor    string                   `json:"executor,omitempty"`
	Commands    []map[string]interface{} `json:"commands"`
	ScriptID    *string                  `json:"script_id,omitempty"`

	// Parameter path command support
	ParamPaths    []string `json:"param_paths"`
	OperationType string   `json:"operation_type"`

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

	// If a script_id is provided, resolve its content into commands
	var scriptID *uuid.UUID
	if req.ScriptID != nil && *req.ScriptID != "" {
		sid, err := uuid.Parse(*req.ScriptID)
		if err != nil {
			return nil, fmt.Errorf("parse script_id %q: %w", *req.ScriptID, err)
		}
		script, err := s.scriptRepo.GetByID(ctx, sid)
		if err != nil {
			return nil, fmt.Errorf("resolve script %s: %w", sid, err)
		}
		scriptID = &sid
		// Parse script content into command entries (one command per line)
		for _, line := range splitScriptLines(script.Content) {
			if line == "" {
				continue
			}
			commands = append(commands, map[string]interface{}{
				"command_code": line,
				"source":       "script",
				"script_name":  script.ScriptName,
			})
		}
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
		if len(req.ParamPaths) > 0 {
			entry["param_paths"] = req.ParamPaths
		}
		if req.OperationType != "" {
			entry["operation_type"] = req.OperationType
		}
		commands = append(commands, entry)
	}

	task := &MMLTask{
		TaskName:  req.TaskName,
		ScriptID:  scriptID,
		DeviceSNs: req.DeviceSNs,
		Commands:  commands,
		Status:    TaskPending,
		Results:   []map[string]interface{}{},
		Creator:   req.Creator,
		Executor:  req.Executor,

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

	// Write audit log entries for each command+device combination
	s.writeAuditLogs(ctx, task)

	// Fan-out to device_tasks for immediate execution
	if task.Status == TaskPending && s.fanouter != nil {
		created, err := s.fanouter.Fanout(ctx, task)
		if err != nil {
			s.logger.Error("fanout mml task failed", zap.Error(err))
		} else if created > 0 {
			if err := s.taskRepo.UpdateStatus(ctx, task.ID, TaskRunning); err != nil {
				s.logger.Error("update mml task to running", zap.Error(err))
			}
			task.Status = TaskRunning
		}
	}

	// Push SSE event to executor
	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), "", string(task.Status))
	}

	return task, nil
}

// writeAuditLogs creates audit log entries for a newly created task.
// Errors are logged but do not fail the task creation.
func (s *Service) writeAuditLogs(ctx context.Context, task *MMLTask) {
	if s.auditRepo == nil {
		return
	}

	creator := task.Creator
	if creator == "" {
		creator = task.Executor
	}

	var entries []*MMLAuditLog
	for _, cmd := range task.Commands {
		commandCode, _ := cmd["command_code"].(string)
		operationType, _ := cmd["operation_type"].(string)

		var params map[string]interface{}
		if p, ok := cmd["parameters"]; ok {
			if pm, ok := p.(map[string]interface{}); ok {
				params = pm
			}
		}

		var paramPaths []string
		if pp, ok := cmd["param_paths"]; ok {
			if ppSlice, ok := pp.([]string); ok {
				paramPaths = ppSlice
			}
		}

		for _, sn := range task.DeviceSNs {
			entries = append(entries, &MMLAuditLog{
				TaskID:        &task.ID,
				CommandCode:   commandCode,
				OperationType: operationType,
				DeviceSN:      sn,
				Parameters:    params,
				ParamPaths:    paramPaths,
				ResultStatus:  string(task.Status),
				Creator:       creator,
			})
		}
	}

	if len(entries) == 0 {
		return
	}

	if err := s.auditRepo.CreateBatch(ctx, entries); err != nil {
		s.logger.Error("failed to write MML audit logs",
			zap.String("task_id", task.ID.String()),
			zap.Int("entry_count", len(entries)),
			zap.Error(err),
		)
	}
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

	oldStatus := string(task.Status)

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskRunning); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskRunning

	// Fan-out to device_tasks when starting from paused
	if s.fanouter != nil {
		created, err := s.fanouter.Fanout(ctx, task)
		if err != nil {
			s.logger.Error("fanout on start failed", zap.Error(err))
		} else {
			s.logger.Info("mml task fanned out on start",
				zap.String("task_id", id.String()),
				zap.Int("device_tasks", created))
		}
	}

	s.logger.Info("mml task started", zap.String("task_id", id.String()))

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskRunning))
	}

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

	oldStatus := string(task.Status)

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskPaused); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskPaused
	s.logger.Info("mml task paused", zap.String("task_id", id.String()))

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskPaused))
	}

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

	oldStatus := string(task.Status)

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskCancelled); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskCancelled
	s.logger.Info("mml task cancelled", zap.String("task_id", id.String()))

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskCancelled))
	}

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

// splitScriptLines splits script content into individual command lines,
// trimming whitespace and ignoring empty lines and comments.
func splitScriptLines(content string) []string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// Strip trailing semicolons
		line = strings.TrimSuffix(line, ";")
		lines = append(lines, line)
	}
	return lines
}

// ---- Custom Command operations (Phase 3) ----

// ListCustomCommands returns a paginated list of MML custom commands.
// For private commands, only the creator's commands are shown.
func (s *Service) ListCustomCommands(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
	return s.customCommandRepo.List(ctx, filter)
}

// GetCustomCommand retrieves an MML custom command by ID.
func (s *Service) GetCustomCommand(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error) {
	return s.customCommandRepo.GetByID(ctx, id)
}

// CreateCustomCommand creates a new user-defined custom command.
func (s *Service) CreateCustomCommand(ctx context.Context, cmd *MMLCustomCommand) (*MMLCustomCommand, error) {
	if cmd.Parameters == nil {
		cmd.Parameters = map[string]interface{}{}
	}
	if cmd.ParamPaths == nil {
		cmd.ParamPaths = []string{}
	}
	if cmd.ProductTypes == nil {
		cmd.ProductTypes = []string{}
	}

	if err := s.customCommandRepo.Create(ctx, cmd); err != nil {
		return nil, fmt.Errorf("create mml custom command: %w", err)
	}

	s.logger.Info("mml custom command created",
		zap.String("command_id", cmd.ID.String()),
		zap.String("command_name", cmd.CommandName),
	)
	return cmd, nil
}

// UpdateCustomCommand updates an existing user-defined custom command.
func (s *Service) UpdateCustomCommand(ctx context.Context, id uuid.UUID, cmd *MMLCustomCommand) (*MMLCustomCommand, error) {
	existing, err := s.customCommandRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}

	existing.CommandName = cmd.CommandName
	existing.CommandCode = cmd.CommandCode
	existing.OperationType = cmd.OperationType
	existing.CommandScope = cmd.CommandScope
	existing.CategoryGroup = cmd.CategoryGroup
	existing.Description = cmd.Description
	if cmd.Parameters != nil {
		existing.Parameters = cmd.Parameters
	}
	if cmd.ParamPaths != nil {
		existing.ParamPaths = cmd.ParamPaths
	}
	if cmd.ProductTypes != nil {
		existing.ProductTypes = cmd.ProductTypes
	}

	if err := s.customCommandRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update mml custom command: %w", err)
	}

	s.logger.Info("mml custom command updated", zap.String("command_id", id.String()))
	return existing, nil
}

// DeleteCustomCommand deletes an MML custom command.
// Only the creator or an admin can delete public commands.
func (s *Service) DeleteCustomCommand(ctx context.Context, id uuid.UUID, currentUser string) error {
	cmd, err := s.customCommandRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get mml custom command: %w", err)
	}

	if cmd.CommandScope == "public" && cmd.Creator != currentUser {
		return fmt.Errorf("only creator can delete public commands: %w", ErrForbidden)
	}

	if err := s.customCommandRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete mml custom command: %w", err)
	}

	s.logger.Info("mml custom command deleted", zap.String("command_id", id.String()))
	return nil
}

// CloneCustomCommand clones a public custom command as a private copy for the current user.
func (s *Service) CloneCustomCommand(ctx context.Context, id uuid.UUID, currentUser string) (*MMLCustomCommand, error) {
	source, err := s.customCommandRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}

	clone := &MMLCustomCommand{
		CommandName:   source.CommandName + " (副本)",
		CommandCode:   source.CommandCode,
		OperationType: source.OperationType,
		CommandScope:  "private",
		CategoryGroup: source.CategoryGroup,
		Parameters:    source.Parameters,
		ParamPaths:    source.ParamPaths,
		Description:   source.Description,
		ProductTypes:  source.ProductTypes,
		Creator:       currentUser,
	}

	if err := s.customCommandRepo.Create(ctx, clone); err != nil {
		return nil, fmt.Errorf("clone mml custom command: %w", err)
	}

	s.logger.Info("mml custom command cloned",
		zap.String("source_id", id.String()),
		zap.String("clone_id", clone.ID.String()),
	)
	return clone, nil
}

// ErrForbidden is returned when a user lacks permission for an operation.
var ErrForbidden = errors.New("forbidden")

// ---- Dangerous command detection (Phase 2) ----

// DangerousCommand describes a command pattern that requires confirmation.
type DangerousCommand struct {
	Pattern *regexp.Regexp
	Name    string
	Desc    string
}

// DangerousCommands holds the list of command patterns requiring extra confirmation.
var DangerousCommands = []DangerousCommand{
	{regexp.MustCompile(`(?i)\bRST\b`), "重启", "此操作将重启设备，设备会暂时断开连接"},
	{regexp.MustCompile(`(?i)\bFACTORYRESET\b`), "恢复默认配置", "此操作将恢复设备出厂设置，所有配置将被清除"},
	{regexp.MustCompile(`(?i)\bCELLDEACTIVATE\b`), "小区去激活", "此操作将去激活小区，可能影响网络服务"},
	{regexp.MustCompile(`(?i)\bRFCTXOFF\b`), "关闭小区射频", "此操作将关闭小区射频发射，会影响无线信号"},
	{regexp.MustCompile(`(?i)\bCOLDREBOOT\b`), "冷重启", "此操作将执行设备冷重启，设备会完全断电重启"},
}

// ErrDangerousCommand is returned when a command matches a dangerous pattern.
var ErrDangerousCommand = errors.New("dangerous command requires confirmation")

// CheckDangerousCommand checks if a command code matches any dangerous pattern.
// Returns the matching DangerousCommand if found, or nil if safe.
func CheckDangerousCommand(commandCode string) *DangerousCommand {
	for _, dc := range DangerousCommands {
		if dc.Pattern.MatchString(commandCode) {
			return &dc
		}
	}
	return nil
}

// IsDangerousCommand checks if a command code is dangerous and requires confirmation.
func (s *Service) IsDangerousCommand(ctx context.Context, commandCode string) (*DangerousCommand, error) {
	return CheckDangerousCommand(commandCode), nil
}

// GetTaskResults returns paginated per-device execution results for a task.
func (s *Service) GetTaskResults(ctx context.Context, id uuid.UUID, page, pageSize int) (*model.ListResponse[map[string]interface{}], error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	results := task.Results
	if results == nil {
		results = []map[string]interface{}{}
	}

	total := int64(len(results))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > int(total) {
		start = int(total)
	}
	if end > int(total) {
		end = int(total)
	}

	items := results[start:end]
	if items == nil {
		items = []map[string]interface{}{}
	}

	return model.NewListResponse(items, total, page, pageSize), nil
}
