package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	syncLockTTL                      = 10 * time.Minute
	syncLockKey                      = "alarm:sync:lock:%s"
	concurrentTaskVisibilityAttempts = 5
	concurrentTaskVisibilityDelay    = 50 * time.Millisecond
	pendingLockOwnerPrefix           = "pending:"
	taskLockOwnerPrefix              = "task:"
)

var ErrAlarmSyncInProgress = fmt.Errorf("alarm sync already in progress: %w", coreerrors.ErrAlreadyExists)

var compareAndDeleteLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

var compareAndSwapLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	redis.call("PSETEX", KEYS[1], ARGV[3], ARGV[2])
	return 1
end
return 0
`)

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func waitForAlarmSyncTaskVisibility(
	ctx context.Context,
	attempts int,
	delay time.Duration,
	lookup func() (*task.Task, error),
	sleep func(context.Context, time.Duration) error,
) (*task.Task, error) {
	if attempts <= 0 {
		attempts = 1
	}
	if sleep == nil {
		sleep = sleepWithContext
	}

	for attempt := 0; attempt < attempts; attempt++ {
		existingTask, err := lookup()
		if err != nil {
			return nil, err
		}
		if existingTask != nil {
			return existingTask, nil
		}
		if attempt == attempts-1 {
			break
		}
		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func newPendingLockOwner() string {
	return pendingLockOwnerPrefix + uuid.NewString()
}

func taskLockOwner(taskID string) string {
	if taskID == "" {
		return ""
	}
	return taskLockOwnerPrefix + taskID
}

func isPendingLockOwner(owner string) bool {
	return strings.HasPrefix(owner, pendingLockOwnerPrefix)
}

func taskIDFromLockOwner(owner string) string {
	if strings.HasPrefix(owner, taskLockOwnerPrefix) {
		return strings.TrimPrefix(owner, taskLockOwnerPrefix)
	}
	if strings.HasPrefix(owner, pendingLockOwnerPrefix) {
		return ""
	}
	return owner
}

func (s *AlarmSyncService) acquireSyncLock(ctx context.Context, lockKey, owner string) (bool, error) {
	return s.redisClient.SetNX(ctx, lockKey, owner, syncLockTTL).Result()
}

func (s *AlarmSyncService) getSyncLockOwner(ctx context.Context, lockKey string) (string, error) {
	owner, err := s.redisClient.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return owner, nil
}

func (s *AlarmSyncService) releaseSyncLockOwner(ctx context.Context, lockKey, owner string) (bool, error) {
	if owner == "" {
		return false, nil
	}
	deleted, err := compareAndDeleteLockScript.Run(ctx, s.redisClient, []string{lockKey}, owner).Int()
	if err != nil {
		return false, err
	}
	return deleted > 0, nil
}

func (s *AlarmSyncService) promoteSyncLockOwner(ctx context.Context, lockKey, currentOwner, nextOwner string) (bool, error) {
	if currentOwner == "" || nextOwner == "" {
		return false, nil
	}
	updated, err := compareAndSwapLockScript.Run(
		ctx,
		s.redisClient,
		[]string{lockKey},
		currentOwner,
		nextOwner,
		strconv.FormatInt(syncLockTTL.Milliseconds(), 10),
	).Int()
	if err != nil {
		return false, err
	}
	return updated > 0, nil
}

func isAlarmSyncTask(tk *task.Task) bool {
	return tk != nil && tk.Method == "GetParameterValues" && tk.Description == "alarm sync: query device current alarms"
}

func (s *AlarmSyncService) releaseTaskLock(ctx context.Context, deviceSN, taskID string) {
	lockKey := fmt.Sprintf(syncLockKey, deviceSN)
	if taskID == "" {
		s.logger.Warn("release sync lock skipped: empty task id", zap.String("device_sn", deviceSN))
		return
	}
	deleted, err := s.releaseSyncLockOwner(ctx, lockKey, taskLockOwner(taskID))
	if err != nil {
		s.logger.Warn("release sync lock", zap.Error(err), zap.String("device_sn", deviceSN), zap.String("task_id", taskID))
		return
	}
	if deleted {
		return
	}
	deleted, err = s.releaseSyncLockOwner(ctx, lockKey, taskID)
	if err != nil {
		s.logger.Warn("release legacy sync lock", zap.Error(err), zap.String("device_sn", deviceSN), zap.String("task_id", taskID))
		return
	}
	if !deleted {
		s.logger.Debug("release sync lock skipped: owner mismatch or already cleared",
			zap.String("device_sn", deviceSN),
			zap.String("task_id", taskID))
	}
}

func (s *AlarmSyncService) lookupTaskByLockOwner(ctx context.Context, owner string) (*task.Task, error) {
	taskID := taskIDFromLockOwner(owner)
	if taskID == "" {
		return nil, nil
	}
	return s.taskService.GetTask(ctx, taskID)
}

func (s *AlarmSyncService) releaseTerminalLockIfOwnedTaskDone(ctx context.Context, deviceSN, lockKey, owner string) (bool, error) {
	ownedTask, err := s.lookupTaskByLockOwner(ctx, owner)
	if err != nil {
		return false, err
	}
	if ownedTask == nil {
		deleted, delErr := s.releaseSyncLockOwner(ctx, lockKey, owner)
		if delErr != nil {
			return false, delErr
		}
		return deleted, nil
	}
	if !isAlarmSyncTask(ownedTask) {
		return false, nil
	}
	switch ownedTask.Status {
	case task.TaskStatusPending, task.TaskStatusSent:
		return false, nil
	default:
		deleted, delErr := s.releaseSyncLockOwner(ctx, lockKey, owner)
		if delErr != nil {
			return false, delErr
		}
		return deleted, nil
	}
}

// AlarmSyncService triggers alarm synchronization by creating GPV tasks.
// It also subscribes to alarm.sync.requested events from receivers.
type AlarmSyncService struct {
	taskService *task.TaskService
	redisClient redis.UniversalClient
	eventBus    event.EventBus
	logger      *zap.Logger
	deviceRead  deviceReader
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

func (s *AlarmSyncService) WithDeviceReader(reader deviceReader) *AlarmSyncService {
	s.deviceRead = reader
	return s
}

// TriggerSync triggers an alarm sync for a device by creating a GPV task.
// When a sync is already in progress, it returns the latest open task so callers can
// observe the eventual terminal status instead of treating the request as completed.
func (s *AlarmSyncService) TriggerSync(ctx context.Context, deviceSN string) (*task.Task, error) {
	deviceSN = strings.TrimSpace(deviceSN)
	if s.deviceRead != nil && deviceSN != "" {
		dev, err := s.deviceRead.GetBySerialNumber(ctx, deviceSN)
		if err != nil {
			return nil, fmt.Errorf("lookup device for alarm sync: %w", err)
		}
		if dev != nil && isUPSProductClass(dev.ProductClass) {
			s.logger.Info("skip alarm sync GPV for UPS device",
				zap.String("device_sn", deviceSN),
				zap.String("product_class", dev.ProductClass))
			return nil, nil
		}
	}

	// Redis SETNX dedup: prevent concurrent syncs for the same device
	lockKey := fmt.Sprintf(syncLockKey, deviceSN)
	lockOwner := newPendingLockOwner()
	lookupLatestTask := func() (*task.Task, error) {
		return s.taskService.LatestOpenTaskByDeviceAndMethod(
			ctx,
			deviceSN,
			"GetParameterValues",
			"alarm sync: query device current alarms",
		)
	}
	acquired, err := s.acquireSyncLock(ctx, lockKey, lockOwner)
	if err != nil {
		s.logger.Warn("redis setnx failed, proceeding anyway",
			zap.String("device_sn", deviceSN), zap.Error(err))
	} else if !acquired {
		initialOwner, ownerErr := s.getSyncLockOwner(ctx, lockKey)
		if ownerErr != nil {
			return nil, fmt.Errorf("read existing alarm sync lock owner: %w", ownerErr)
		}
		existingTask, lookupErr := waitForAlarmSyncTaskVisibility(
			ctx,
			concurrentTaskVisibilityAttempts,
			concurrentTaskVisibilityDelay,
			lookupLatestTask,
			nil,
		)
		if lookupErr != nil {
			return nil, fmt.Errorf("lookup existing alarm sync task: %w", lookupErr)
		}
		if existingTask != nil {
			s.logger.Debug("alarm sync already in progress, reusing open task",
				zap.String("device_sn", deviceSN),
				zap.String("task_id", existingTask.ID))
			return existingTask, nil
		}

		currentOwner, currentOwnerErr := s.getSyncLockOwner(ctx, lockKey)
		if currentOwnerErr != nil {
			return nil, fmt.Errorf("read alarm sync lock owner after visibility wait: %w", currentOwnerErr)
		}
		if initialOwner != "" && currentOwner != initialOwner {
			retryTask, retryLookupErr := waitForAlarmSyncTaskVisibility(
				ctx,
				concurrentTaskVisibilityAttempts,
				concurrentTaskVisibilityDelay,
				lookupLatestTask,
				nil,
			)
			if retryLookupErr != nil {
				return nil, fmt.Errorf("lookup existing alarm sync task after owner change: %w", retryLookupErr)
			}
			if retryTask != nil {
				s.logger.Debug("alarm sync task found after lock owner changed",
					zap.String("device_sn", deviceSN),
					zap.String("task_id", retryTask.ID))
				return retryTask, nil
			}
		}

		s.logger.Warn("alarm sync lock held but open task missing after visibility wait",
			zap.String("device_sn", deviceSN),
			zap.String("owner", currentOwner))
		if isPendingLockOwner(currentOwner) {
			return nil, ErrAlarmSyncInProgress
		}
		if currentOwner != "" {
			deleted, delErr := s.releaseTerminalLockIfOwnedTaskDone(ctx, deviceSN, lockKey, currentOwner)
			if delErr != nil {
				s.logger.Warn("failed to reconcile stale alarm sync lock",
					zap.String("device_sn", deviceSN),
					zap.String("owner", currentOwner),
					zap.Error(delErr))
				return nil, fmt.Errorf("reconcile stale alarm sync lock: %w", delErr)
			}
			if !deleted {
				ownedTask, getTaskErr := s.lookupTaskByLockOwner(ctx, currentOwner)
				if getTaskErr != nil {
					return nil, fmt.Errorf("lookup task by alarm sync lock owner: %w", getTaskErr)
				}
				if isAlarmSyncTask(ownedTask) {
					return ownedTask, nil
				}
				return nil, ErrAlarmSyncInProgress
			}
		}

		reacquired, reacquireErr := s.acquireSyncLock(ctx, lockKey, lockOwner)
		if reacquireErr != nil {
			s.logger.Warn("redis setnx failed after reconciling alarm sync lock, proceeding anyway",
				zap.String("device_sn", deviceSN), zap.Error(reacquireErr))
		} else if !reacquired {
			retryTask, retryLookupErr := waitForAlarmSyncTaskVisibility(
				ctx,
				concurrentTaskVisibilityAttempts,
				concurrentTaskVisibilityDelay,
				lookupLatestTask,
				nil,
			)
			if retryLookupErr != nil {
				return nil, fmt.Errorf("lookup existing alarm sync task after lock reconcile: %w", retryLookupErr)
			}
			if retryTask != nil {
				s.logger.Debug("alarm sync task found after lock reconcile",
					zap.String("device_sn", deviceSN),
					zap.String("task_id", retryTask.ID))
				return retryTask, nil
			}
			return nil, ErrAlarmSyncInProgress
		}
	}

	// Create GPV task to query CurrentAlarm parameters
	params := json.RawMessage(`{"names":["Device.FaultMgmt.CurrentAlarm."]}`)
	maxRetries := 2 // CreateTaskRequest.MaxRetries 为 *int（区分未设置/显式 0）
	syncTask, err := s.taskService.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:  deviceSN,
		Method:    "GetParameterValues",
		Params:    params,
		Source:    task.TaskSourceSystem,
		CreatorID: "", // 系统任务：空串表示非用户发起，与其它 TaskSourceSystem 任务
		// （见 pm/online_subscriber.go、notification/task_subscriber.go 等）保持一致。
		// 之前误用 uuid.Nil.String()（"00000000-...-000000000000"）——该值能被
		// uuid.Parse 成功解析，导致 taskLogObserver.parseOperatorID 把它当成合法
		// operator_id 写入 sys_task_logs，因 users 表里没有全零 UUID 的占位用户而
		// 触发外键约束违反（sys_task_logs_operator_id_fkey）。
		Description: "alarm sync: query device current alarms",
		Priority:    5,
		MaxRetries:  &maxRetries,
		ExpiresIn:   600, // 10 minutes
	})
	if err != nil {
		// Release lock on failure
		if _, delErr := s.releaseSyncLockOwner(ctx, lockKey, lockOwner); delErr != nil {
			s.logger.Warn("release alarm sync lock after create failure",
				zap.String("device_sn", deviceSN),
				zap.Error(delErr))
		}
		return nil, fmt.Errorf("create alarm sync task: %w", err)
	}

	if promoted, promoteErr := s.promoteSyncLockOwner(ctx, lockKey, lockOwner, taskLockOwner(syncTask.ID)); promoteErr != nil {
		s.logger.Warn("promote alarm sync lock owner to task id failed",
			zap.String("device_sn", deviceSN),
			zap.String("task_id", syncTask.ID),
			zap.Error(promoteErr))
	} else if !promoted {
		s.logger.Warn("alarm sync lock owner changed before task id promotion",
			zap.String("device_sn", deviceSN),
			zap.String("task_id", syncTask.ID))
	}

	s.logger.Info("alarm sync task created",
		zap.String("device_sn", deviceSN))
	return syncTask, nil
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
			_, triggerErr := s.TriggerSync(ctx, payload.DeviceSN)
			return triggerErr
		},
	)
	if err != nil {
		return fmt.Errorf("subscribe alarm sync requests: %w", err)
	}

	terminalHandler := func(ctx context.Context, evt event.Event) error {
		var syncTask task.Task
		if err := evt.DecodePayload(&syncTask); err != nil {
			s.logger.Error("decode alarm sync terminal task", zap.Error(err), zap.String("subject", evt.Subject))
			return nil
		}
		if !isAlarmSyncTask(&syncTask) {
			return nil
		}
		s.releaseTaskLock(ctx, syncTask.DeviceSN, syncTask.ID)
		return nil
	}
	if _, err := s.eventBus.QueueSubscribe(event.SubjectTaskFailed, "alarm-sync-task-failed", terminalHandler); err != nil {
		return fmt.Errorf("subscribe alarm sync task.failed: %w", err)
	}
	if _, err := s.eventBus.QueueSubscribe(event.SubjectTaskCompleted, "alarm-sync-task-completed", terminalHandler); err != nil {
		return fmt.Errorf("subscribe alarm sync task.completed: %w", err)
	}
	if _, err := s.eventBus.QueueSubscribe(event.SubjectTaskCancelled, "alarm-sync-task-cancelled", terminalHandler); err != nil {
		return fmt.Errorf("subscribe alarm sync task.cancelled: %w", err)
	}

	s.logger.Info("alarm sync service subscribed to sync request events")
	return nil
}

// ReleaseLock releases the sync lock for a device when the current holder matches owner.
func (s *AlarmSyncService) ReleaseLock(ctx context.Context, deviceSN, owner string) {
	s.releaseTaskLock(ctx, deviceSN, owner)
}
