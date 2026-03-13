package baseline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Service provides business logic for config baseline management.
type Service struct {
	baselineRepo BaselineRepository
	taskRepo     ConfigTaskRepository
	neighborRepo NeighborRepository
	logger       *zap.Logger
}

// NewService creates a new config baseline Service.
func NewService(
	baselineRepo BaselineRepository,
	taskRepo ConfigTaskRepository,
	neighborRepo NeighborRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		baselineRepo: baselineRepo,
		taskRepo:     taskRepo,
		neighborRepo: neighborRepo,
		logger:       logger.Named("config-baseline"),
	}
}

// ---- Baseline methods ----

// CreateBaseline creates a new baseline config with default status=draft.
func (s *Service) CreateBaseline(ctx context.Context, baseline *BaselineConfig) (*BaselineConfig, error) {
	if baseline.Status == "" {
		baseline.Status = BaselineDraft
	}
	if baseline.Params == nil {
		baseline.Params = json.RawMessage("[]")
	}

	if err := s.baselineRepo.Create(ctx, baseline); err != nil {
		return nil, fmt.Errorf("create baseline config: %w", err)
	}

	s.logger.Info("baseline config created",
		zap.String("baseline_id", baseline.ID.String()),
		zap.String("baseline_name", baseline.BaselineName),
	)

	return baseline, nil
}

// GetBaseline retrieves a baseline config by ID.
func (s *Service) GetBaseline(ctx context.Context, id uuid.UUID) (*BaselineConfig, error) {
	return s.baselineRepo.GetByID(ctx, id)
}

// UpdateBaseline updates an existing baseline config.
func (s *Service) UpdateBaseline(ctx context.Context, id uuid.UUID, baseline *BaselineConfig) (*BaselineConfig, error) {
	existing, err := s.baselineRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get baseline config: %w", err)
	}

	// Apply updates
	existing.BaselineName = baseline.BaselineName
	existing.Description = baseline.Description
	existing.DeviceType = baseline.DeviceType
	existing.Version = baseline.Version
	if baseline.Params != nil {
		existing.Params = baseline.Params
	}
	existing.Creator = baseline.Creator
	if baseline.Status != "" {
		existing.Status = baseline.Status
	}

	if err := s.baselineRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update baseline config: %w", err)
	}

	s.logger.Info("baseline config updated",
		zap.String("baseline_id", id.String()),
	)

	return existing, nil
}

// DeleteBaseline deletes a baseline config by ID.
func (s *Service) DeleteBaseline(ctx context.Context, id uuid.UUID) error {
	return s.baselineRepo.Delete(ctx, id)
}

// ListBaselines returns a paginated list of baseline configs.
func (s *Service) ListBaselines(ctx context.Context, filter BaselineFilter) (*model.ListResponse[BaselineConfig], error) {
	return s.baselineRepo.List(ctx, filter)
}

// ---- Config Task methods ----

// CreateConfigTask creates a new config task with default status=pending.
func (s *Service) CreateConfigTask(ctx context.Context, task *ConfigTask) (*ConfigTask, error) {
	task.Status = ConfigTaskPending
	task.Progress = 0

	if task.DeviceSns == nil {
		task.DeviceSns = json.RawMessage("[]")
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create config task: %w", err)
	}

	s.logger.Info("config task created",
		zap.String("task_id", task.ID.String()),
		zap.String("task_name", task.TaskName),
		zap.String("task_type", string(task.TaskType)),
	)

	return task, nil
}

// ListConfigTasks returns a paginated list of config tasks.
func (s *Service) ListConfigTasks(ctx context.Context, filter ConfigTaskFilter) (*model.ListResponse[ConfigTask], error) {
	return s.taskRepo.List(ctx, filter)
}

// ---- Neighbor methods ----

// ListNeighbors returns a paginated list of neighbor params.
func (s *Service) ListNeighbors(ctx context.Context, filter NeighborFilter) (*model.ListResponse[NeighborParam], error) {
	return s.neighborRepo.List(ctx, filter)
}
