package deviceaccess

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

const (
	defaultActionReconcileInterval = 5 * time.Second
	defaultActionStaleAfter        = 30 * time.Second
	defaultActionReconcileBatch    = 20
)

// ProcessDue claims due pending/retry actions in PostgreSQL before dispatching
// them. ClaimDue uses row locks so concurrent workers cannot consume the same
// retry budget or create a second southbound command.
func (s *ActionService) ProcessDue(ctx context.Context, limit int) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("process due device access actions: %w", ErrAccessGateDependencyMissing)
	}
	actions, err := s.store.ClaimDue(ctx, s.now(), limit)
	if err != nil {
		return err
	}
	var firstErr error
	for _, action := range actions {
		if err := s.resumeDispatch(ctx, action); err != nil && !errors.Is(err, ErrActionStateChanged) {
			if firstErr == nil {
				firstErr = err
			}
			s.logger.Warn("resume due RF action", zap.String("action_id", action.ID.String()), zap.Error(err))
		}
	}
	return firstErr
}

// RecoverStale reconciles an in-flight action with its durable device task.
// It never repeats a write while the original task still exists; terminal task
// state is projected through the normal completion handler.
func (s *ActionService) RecoverStale(ctx context.Context, staleBefore time.Time, limit int) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("recover device access actions: %w", ErrAccessGateDependencyMissing)
	}
	actions, err := s.store.ListRecoverable(ctx, staleBefore, limit)
	if err != nil {
		return err
	}
	var firstErr error
	for _, action := range actions {
		if err := s.recoverAction(ctx, action); err != nil && !errors.Is(err, ErrActionStateChanged) {
			if firstErr == nil {
				firstErr = err
			}
			s.logger.Warn("recover stale RF action", zap.String("action_id", action.ID.String()), zap.Error(err))
		}
	}
	return firstErr
}

func (s *ActionService) recoverAction(ctx context.Context, action Action) error {
	deviceTask, err := s.store.LoadActionTask(ctx, action)
	if err != nil {
		return err
	}
	if deviceTask == nil {
		if action.DeviceTaskID == nil {
			return s.resumeDispatch(ctx, action)
		}
		return s.failAction(ctx, action, "device_task_missing", "durable RF device task is missing", true)
	}
	if action.DeviceTaskID == nil {
		deviceTaskID, parseErr := uuid.Parse(deviceTask.ID)
		if parseErr != nil {
			return s.failAction(ctx, action, "invalid_task_id", parseErr.Error(), false)
		}
		if action.Status == ActionStatusVerifying {
			err = s.store.AdvanceDispatchTask(ctx, action.ID, uuid.Nil, deviceTaskID, ActionStatusVerifying, s.now())
		} else {
			err = s.store.AttachDispatchTask(ctx, action.ID, deviceTaskID)
		}
		if err != nil && !errors.Is(err, ErrActionStateChanged) {
			return fmt.Errorf("restore RF action task correlation: %w", err)
		}
	}
	switch deviceTask.Status {
	case task.TaskStatusCompleted, task.TaskStatusFailed, task.TaskStatusExpired, task.TaskStatusCancelled:
		return s.HandleCompleted(ctx, deviceTask)
	case task.TaskStatusPending, task.TaskStatusSent:
		return s.store.TouchAction(ctx, action.ID, s.now())
	default:
		return s.failAction(ctx, action, "invalid_task_status", "RF device task has an invalid status", false)
	}
}

func (s *ActionService) WakeFromDeviceOnline(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceID uuid.UUID `json:"device_id"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device online RF wake-up: %w", err)
	}
	if payload.DeviceID == uuid.Nil {
		return nil
	}
	if err := s.store.WakeRetryForDevice(ctx, payload.DeviceID, s.now()); err != nil {
		return err
	}
	return s.ProcessDue(ctx, defaultActionReconcileBatch)
}

type ActionReconciler struct {
	service    *ActionService
	interval   time.Duration
	staleAfter time.Duration
	batchSize  int
	logger     *zap.Logger
}

func NewActionReconciler(service *ActionService, logger *zap.Logger) *ActionReconciler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ActionReconciler{
		service: service, interval: defaultActionReconcileInterval,
		staleAfter: defaultActionStaleAfter, batchSize: defaultActionReconcileBatch,
		logger: logger.Named("device-access-action-reconciler"),
	}
}

func (r *ActionReconciler) RunOnce(ctx context.Context) error {
	if r == nil || r.service == nil {
		return ErrAccessGateDependencyMissing
	}
	if err := r.service.RecoverStale(ctx, r.service.now().Add(-r.staleAfter), r.batchSize); err != nil {
		return err
	}
	return r.service.ProcessDue(ctx, r.batchSize)
}

func (r *ActionReconciler) Start(ctx context.Context) error {
	if err := r.RunOnce(ctx); err != nil {
		r.logger.Warn("initial RF action recovery failed", zap.Error(err))
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.RunOnce(ctx); err != nil {
				r.logger.Warn("RF action recovery pass failed", zap.Error(err))
			}
		}
	}
}
