package device

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

const (
	resetLMTPasswordMethod         = "X_BAICELLS_COM_PasswordReset"
	resetLMTPasswordFallbackMethod = "X_COMMON_COM_PasswordReset"
	resetLMTPasswordFallbackPrefix = "reset-lmt-pwd-common-"
)

type ResetLMTPasswordResult struct {
	TaskID string
}

// ResetLMTPassword queues the legacy LMT password reset message task.
func (s *DeviceService) ResetLMTPassword(ctx context.Context, id uuid.UUID, creatorID string) (*ResetLMTPasswordResult, error) {
	dev, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get device for password reset: %w", err)
	}
	if dev == nil {
		return nil, commonerrors.ErrNotFound
	}
	if s.taskSvc == nil {
		return nil, fmt.Errorf("task service not configured")
	}

	paramsJSON, err := json.Marshal(map[string]string{
		"message_type": resetLMTPasswordMethod,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal password reset message params: %w", err)
	}

	createdTask, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    dev.SerialNumber,
		Method:      resetLMTPasswordMethod,
		Params:      paramsJSON,
		Priority:    5,
		CommandKey:  fmt.Sprintf("reset-lmt-pwd-%s", uuid.New().String()[:8]),
		Source:      task.TaskSourceAPI,
		CreatorID:   creatorID,
		Description: "reset LMT password",
	})
	if err != nil {
		return nil, fmt.Errorf("queue password reset message: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("password reset message queued",
			zap.String("device_id", id.String()),
			zap.String("serial_number", dev.SerialNumber),
			zap.String("task_id", createdTask.ID),
		)
	}

	return &ResetLMTPasswordResult{TaskID: createdTask.ID}, nil
}
