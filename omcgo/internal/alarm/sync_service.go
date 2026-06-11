package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

const (
	syncLockTTL = 10 * time.Minute
	syncLockKey = "alarm:sync:lock:%s"
)

// AlarmSyncService triggers alarm synchronization by creating GPV tasks.
// It also subscribes to alarm.sync.requested events from receivers.
type AlarmSyncService struct {
	taskService *task.TaskService
	redisClient redis.UniversalClient
	eventBus    event.EventBus
	logger      *zap.Logger
}

// NewAlarmSyncService creates a new AlarmSyncService.
func NewAlarmSyncService(
	taskService *task.TaskService,
	redisClient redis.UniversalClient,
	eventBus event.EventBus,
	logger *zap.Logger,
) *AlarmSyncService {
	return &AlarmSyncService{
		taskService: taskService,
		redisClient: redisClient,
		eventBus:    eventBus,
		logger:      logger,
	}
}

// TriggerSync triggers an alarm sync for a device by creating a GPV task.
// Returns error if a sync is already in progress for the device (dedup via Redis SETNX).
func (s *AlarmSyncService) TriggerSync(ctx context.Context, deviceSN string) error {
	// Redis SETNX dedup: prevent concurrent syncs for the same device
	lockKey := fmt.Sprintf(syncLockKey, deviceSN)
	acquired, err := s.redisClient.SetNX(ctx, lockKey, "1", syncLockTTL).Result()
	if err != nil {
		s.logger.Warn("redis setnx failed, proceeding anyway",
			zap.String("device_sn", deviceSN), zap.Error(err))
	} else if !acquired {
		s.logger.Debug("alarm sync already in progress, skipping",
			zap.String("device_sn", deviceSN))
		return nil
	}

	// Create GPV task to query CurrentAlarm parameters
	params := json.RawMessage(`{"names":["Device.FaultMgmt.CurrentAlarm."]}`)
	maxRetries := 2 // CreateTaskRequest.MaxRetries 为 *int（区分未设置/显式 0）
	_, err = s.taskService.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    deviceSN,
		Method:      "GetParameterValues",
		Params:      params,
		Source:      task.TaskSourceSystem,
		CreatorID:   uuid.Nil.String(),
		Description: "alarm sync: query device current alarms",
		Priority:    5,
		MaxRetries:  &maxRetries,
		ExpiresIn:   600, // 10 minutes
	})
	if err != nil {
		// Release lock on failure
		s.redisClient.Del(ctx, lockKey)
		return fmt.Errorf("create alarm sync task: %w", err)
	}

	s.logger.Info("alarm sync task created",
		zap.String("device_sn", deviceSN))
	return nil
}

// Subscribe registers the service for alarm.sync.requested events.
// The receiver publishes this event after processing an Inform alarm event.
func (s *AlarmSyncService) Subscribe() error {
	if s.eventBus == nil {
		return nil
	}

	_, err := s.eventBus.QueueSubscribe(
		event.SubjectAlarmSyncRequested,
		"alarm-sync-trigger",
		func(ctx context.Context, evt event.Event) error {
			var payload struct {
				DeviceSN string `json:"device_sn"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				s.logger.Error("decode alarm sync request", zap.Error(err))
				return nil
			}
			if payload.DeviceSN == "" {
				s.logger.Warn("empty device_sn in alarm sync request")
				return nil
			}
			return s.TriggerSync(ctx, payload.DeviceSN)
		},
	)
	if err != nil {
		return fmt.Errorf("subscribe alarm sync requests: %w", err)
	}

	s.logger.Info("alarm sync service subscribed to sync request events")
	return nil
}

// ReleaseLock releases the sync lock for a device. Called after sync completes or fails.
func (s *AlarmSyncService) ReleaseLock(ctx context.Context, deviceSN string) {
	lockKey := fmt.Sprintf(syncLockKey, deviceSN)
	if err := s.redisClient.Del(ctx, lockKey).Err(); err != nil {
		s.logger.Warn("release sync lock", zap.Error(err), zap.String("device_sn", deviceSN))
	}
}
