package backup

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Service provides business logic for backup management.
type Service struct {
	taskRepo     TaskRepository
	scheduleRepo ScheduleRepository
	eventBus     event.EventBus
	logger       *zap.Logger
}

// NewService creates a new backup Service.
func NewService(taskRepo TaskRepository, scheduleRepo ScheduleRepository, eventBus event.EventBus, logger *zap.Logger) *Service {
	return &Service{
		taskRepo:     taskRepo,
		scheduleRepo: scheduleRepo,
		eventBus:     eventBus,
		logger:       logger.Named("backup"),
	}
}

// CreateTask creates a new backup task with status=pending and a generated UUID.
func (s *Service) CreateTask(ctx context.Context, task *BackupTask) (*BackupTask, error) {
	task.Status = TaskPending
	task.Progress = 0

	if task.TargetIDs == nil {
		task.TargetIDs = []string{}
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create backup task: %w", err)
	}

	s.logger.Info("backup task created",
		zap.String("task_id", task.ID.String()),
		zap.String("task_type", string(task.TaskType)),
		zap.String("target_type", task.TargetType),
	)

	// Publish event to trigger executor
	if s.eventBus != nil {
		evt, err := event.NewEvent(event.SubjectBackupTaskCreated, map[string]interface{}{
			"task_id": task.ID.String(),
		})
		if err == nil {
			_ = s.eventBus.Publish(ctx, event.SubjectBackupTaskCreated, evt)
		}
	}

	return task, nil
}

// GetTask retrieves a backup task by ID.
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
	return s.taskRepo.GetByID(ctx, id)
}

// CancelTask cancels a backup task. Only pending or running tasks can be cancelled.
func (s *Service) CancelTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get backup task: %w", err)
	}

	if task.Status != TaskPending && task.Status != TaskRunning {
		return commonerrors.NewBusinessError(8100, "only pending or running tasks can be cancelled", commonerrors.ErrInvalidInput)
	}

	task.Status = TaskCancelled
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("cancel backup task: %w", err)
	}

	s.logger.Info("backup task cancelled",
		zap.String("task_id", id.String()),
	)

	return nil
}

// DeleteTask deletes a backup task by ID.
func (s *Service) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return s.taskRepo.Delete(ctx, id)
}

// ListTasks returns a paginated list of backup tasks.
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error) {
	return s.taskRepo.List(ctx, filter)
}

// CreateSchedule creates a new backup schedule.
func (s *Service) CreateSchedule(ctx context.Context, schedule *BackupSchedule) (*BackupSchedule, error) {
	if schedule.TargetIDs == nil {
		schedule.TargetIDs = []string{}
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, fmt.Errorf("create backup schedule: %w", err)
	}

	s.logger.Info("backup schedule created",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("name", schedule.Name),
		zap.String("cron_expr", schedule.CronExpr),
	)

	s.notifyScheduleChanged(ctx, "create", schedule.ID.String())

	return schedule, nil
}

// UpdateSchedule updates an existing backup schedule.
func (s *Service) UpdateSchedule(ctx context.Context, id uuid.UUID, schedule *BackupSchedule) (*BackupSchedule, error) {
	existing, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get backup schedule: %w", err)
	}

	// Apply updates
	existing.Name = schedule.Name
	existing.CronExpr = schedule.CronExpr
	existing.Enabled = schedule.Enabled
	existing.TaskType = schedule.TaskType
	existing.TargetType = schedule.TargetType
	if schedule.TargetIDs != nil {
		existing.TargetIDs = schedule.TargetIDs
	}

	if err := s.scheduleRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update backup schedule: %w", err)
	}

	s.logger.Info("backup schedule updated",
		zap.String("schedule_id", id.String()),
	)

	s.notifyScheduleChanged(ctx, "update", id.String())

	return existing, nil
}

// DeleteSchedule deletes a backup schedule by ID.
func (s *Service) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if err := s.scheduleRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.notifyScheduleChanged(ctx, "delete", id.String())
	return nil
}

// notifyScheduleChanged 发布 backup.schedule.changed 事件，让
// worker 侧的 PeriodScheduler 拉表刷新 cron。eventBus 未注入时 silent。
func (s *Service) notifyScheduleChanged(ctx context.Context, action, scheduleID string) {
	if s.eventBus == nil {
		return
	}
	evt, err := event.NewEvent(event.SubjectBackupScheduleChanged, map[string]interface{}{
		"action":      action,
		"schedule_id": scheduleID,
	})
	if err != nil {
		s.logger.Warn("build schedule.changed event", zap.Error(err))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectBackupScheduleChanged, evt); err != nil {
		s.logger.Warn("publish schedule.changed",
			zap.String("action", action),
			zap.String("schedule_id", scheduleID),
			zap.Error(err))
	}
}

// ListSchedules returns a paginated list of backup schedules.
func (s *Service) ListSchedules(ctx context.Context, filter ScheduleFilter) (*model.ListResponse[BackupSchedule], error) {
	return s.scheduleRepo.List(ctx, filter)
}
