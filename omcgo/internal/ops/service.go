package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

// Service provides business logic for ops tools management.
type Service struct {
	templateRepo TemplateRepository
	taskRepo     TaskRepository
	cmdRepo      CommandRecordRepository
	logger       *zap.Logger
}

// NewService creates a new ops Service.
func NewService(
	templateRepo TemplateRepository,
	taskRepo TaskRepository,
	cmdRepo CommandRecordRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		templateRepo: templateRepo,
		taskRepo:     taskRepo,
		cmdRepo:      cmdRepo,
		logger:       logger.Named("ops"),
	}
}

// ---- Template operations ----

// CreateTemplate creates a new ops template.
func (s *Service) CreateTemplate(ctx context.Context, tmpl *OpsTemplate) (*OpsTemplate, error) {
	tmpl.UseCount = 0
	if tmpl.TargetDeviceTypes == nil {
		tmpl.TargetDeviceTypes = []byte("[]")
	}
	if tmpl.Steps == nil {
		tmpl.Steps = []byte("[]")
	}
	if tmpl.Tags == nil {
		tmpl.Tags = []byte("[]")
	}

	if err := s.templateRepo.Create(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("create ops template: %w", err)
	}

	s.logger.Info("ops template created",
		zap.String("template_id", tmpl.ID.String()),
		zap.String("template_name", tmpl.TemplateName),
	)

	return tmpl, nil
}

// GetTemplate retrieves an ops template by ID.
func (s *Service) GetTemplate(ctx context.Context, id uuid.UUID) (*OpsTemplate, error) {
	return s.templateRepo.GetByID(ctx, id)
}

// UpdateTemplate updates an existing ops template.
func (s *Service) UpdateTemplate(ctx context.Context, id uuid.UUID, tmpl *OpsTemplate) (*OpsTemplate, error) {
	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ops template: %w", err)
	}

	existing.TemplateName = tmpl.TemplateName
	existing.Description = tmpl.Description
	existing.Category = tmpl.Category
	existing.TargetDeviceTypes = tmpl.TargetDeviceTypes
	existing.Steps = tmpl.Steps
	existing.EstimatedDuration = tmpl.EstimatedDuration
	existing.Creator = tmpl.Creator
	existing.Tags = tmpl.Tags

	if err := s.templateRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update ops template: %w", err)
	}

	s.logger.Info("ops template updated",
		zap.String("template_id", id.String()),
	)

	return existing, nil
}

// DeleteTemplate deletes an ops template by ID.
func (s *Service) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return s.templateRepo.Delete(ctx, id)
}

// ListTemplates returns a paginated list of ops templates.
func (s *Service) ListTemplates(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error) {
	return s.templateRepo.List(ctx, filter)
}

// ---- Task operations ----

// CreateTask creates a new ops task with status=pending.
func (s *Service) CreateTask(ctx context.Context, task *OpsTask) (*OpsTask, error) {
	task.Status = OpsTaskPending
	task.CurrentStep = 0
	task.Progress = 0
	task.SuccessCount = 0
	task.FailCount = 0

	if task.DeviceSNs == nil {
		task.DeviceSNs = []byte("[]")
	}

	// Count total devices
	var deviceSNs []string
	if err := json.Unmarshal(task.DeviceSNs, &deviceSNs); err == nil {
		task.TotalCount = len(deviceSNs)
	}

	// If template is specified, get total steps from template
	if task.TemplateID != nil {
		tmpl, err := s.templateRepo.GetByID(ctx, *task.TemplateID)
		if err == nil {
			var steps []json.RawMessage
			if err := json.Unmarshal(tmpl.Steps, &steps); err == nil {
				task.TotalSteps = len(steps)
			}
			// Increment use count
			_ = s.templateRepo.IncrementUseCount(ctx, *task.TemplateID)
		}
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create ops task: %w", err)
	}

	s.logger.Info("ops task created",
		zap.String("task_id", task.ID.String()),
		zap.String("task_name", task.TaskName),
	)

	return task, nil
}

// GetTask retrieves an ops task by ID.
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*OpsTask, error) {
	return s.taskRepo.GetByID(ctx, id)
}

// ListTasks returns a paginated list of ops tasks.
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error) {
	return s.taskRepo.List(ctx, filter)
}

// CancelTask cancels an ops task. Only pending or running tasks can be cancelled.
func (s *Service) CancelTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get ops task: %w", err)
	}

	if task.Status != OpsTaskPending && task.Status != OpsTaskRunning {
		return commonerrors.NewBusinessError(8100, "only pending or running tasks can be cancelled", commonerrors.ErrInvalidInput)
	}

	task.Status = OpsTaskCancelled
	now := time.Now()
	task.CompletedAt = &now

	if err := s.taskRepo.UpdateStatus(ctx, task); err != nil {
		return fmt.Errorf("cancel ops task: %w", err)
	}

	s.logger.Info("ops task cancelled",
		zap.String("task_id", id.String()),
	)

	return nil
}

// PauseTask pauses an ops task. Only running tasks can be paused.
func (s *Service) PauseTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get ops task: %w", err)
	}

	if task.Status != OpsTaskRunning {
		return commonerrors.NewBusinessError(8101, "only running tasks can be paused", commonerrors.ErrInvalidInput)
	}

	task.Status = OpsTaskPaused

	if err := s.taskRepo.UpdateStatus(ctx, task); err != nil {
		return fmt.Errorf("pause ops task: %w", err)
	}

	s.logger.Info("ops task paused",
		zap.String("task_id", id.String()),
	)

	return nil
}

// ResumeTask resumes a paused ops task. Only paused tasks can be resumed.
func (s *Service) ResumeTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get ops task: %w", err)
	}

	if task.Status != OpsTaskPaused {
		return commonerrors.NewBusinessError(8102, "only paused tasks can be resumed", commonerrors.ErrInvalidInput)
	}

	task.Status = OpsTaskRunning
	now := time.Now()
	if task.StartedAt == nil {
		task.StartedAt = &now
	}

	if err := s.taskRepo.UpdateStatus(ctx, task); err != nil {
		return fmt.Errorf("resume ops task: %w", err)
	}

	s.logger.Info("ops task resumed",
		zap.String("task_id", id.String()),
	)

	return nil
}

// ---- Command Record operations ----

// CreateCommandRecord creates a new command record.
func (s *Service) CreateCommandRecord(ctx context.Context, record *OpsCommandRecord) (*OpsCommandRecord, error) {
	if record.ExecuteTime.IsZero() {
		record.ExecuteTime = time.Now()
	}

	if err := s.cmdRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create ops command record: %w", err)
	}

	return record, nil
}

// ListCommandRecords returns a paginated list of command records.
func (s *Service) ListCommandRecords(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error) {
	return s.cmdRepo.List(ctx, filter)
}
