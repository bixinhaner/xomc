package alarm

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type syncSvcMockSubscription struct{}

func (s *syncSvcMockSubscription) Unsubscribe() error { return nil }

type syncSvcMockEventBus struct {
	queueSubCalls []string
}

func (b *syncSvcMockEventBus) Publish(_ context.Context, _ string, _ event.Event) error { return nil }
func (b *syncSvcMockEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &syncSvcMockSubscription{}, nil
}
func (b *syncSvcMockEventBus) QueueSubscribe(subject, queue string, _ event.EventHandler) (event.Subscription, error) {
	b.queueSubCalls = append(b.queueSubCalls, fmt.Sprintf("%s|%s", subject, queue))
	return &syncSvcMockSubscription{}, nil
}
func (b *syncSvcMockEventBus) PullSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &syncSvcMockSubscription{}, nil
}
func (b *syncSvcMockEventBus) Close() error { return nil }

func TestWaitForAlarmSyncTaskVisibilityReturnsTaskAfterRetry(t *testing.T) {
	t.Parallel()

	want := &task.Task{ID: "task-1"}
	lookupCalls := 0
	sleepCalls := 0

	got, err := waitForAlarmSyncTaskVisibility(
		context.Background(),
		3,
		time.Millisecond,
		func() (*task.Task, error) {
			lookupCalls++
			if lookupCalls < 2 {
				return nil, nil
			}
			return want, nil
		},
		func(context.Context, time.Duration) error {
			sleepCalls++
			return nil
		},
	)

	require.NoError(t, err)
	require.Same(t, want, got)
	assert.Equal(t, 2, lookupCalls)
	assert.Equal(t, 1, sleepCalls)
}

func TestWaitForAlarmSyncTaskVisibilityReturnsNilAfterAttempts(t *testing.T) {
	t.Parallel()

	lookupCalls := 0
	sleepCalls := 0

	got, err := waitForAlarmSyncTaskVisibility(
		context.Background(),
		3,
		time.Millisecond,
		func() (*task.Task, error) {
			lookupCalls++
			return nil, nil
		},
		func(context.Context, time.Duration) error {
			sleepCalls++
			return nil
		},
	)

	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Equal(t, 3, lookupCalls)
	assert.Equal(t, 2, sleepCalls)
}

func TestWaitForAlarmSyncTaskVisibilityStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	lookupCalls := 0
	sleepCalls := 0

	got, err := waitForAlarmSyncTaskVisibility(
		ctx,
		3,
		time.Millisecond,
		func() (*task.Task, error) {
			lookupCalls++
			return nil, nil
		},
		func(ctx context.Context, _ time.Duration) error {
			sleepCalls++
			return ctx.Err()
		},
	)

	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 1, lookupCalls)
	assert.Equal(t, 1, sleepCalls)
}

func TestWaitForAlarmSyncTaskVisibilityChecksFullWindowBeforeTreatingTaskAsMissing(t *testing.T) {
	t.Parallel()

	lookupCalls := 0
	sleepCalls := 0
	want := &task.Task{ID: "task-visible-late"}

	got, err := waitForAlarmSyncTaskVisibility(
		context.Background(),
		concurrentTaskVisibilityAttempts,
		concurrentTaskVisibilityDelay,
		func() (*task.Task, error) {
			lookupCalls++
			if lookupCalls == concurrentTaskVisibilityAttempts {
				return want, nil
			}
			return nil, nil
		},
		func(context.Context, time.Duration) error {
			sleepCalls++
			return nil
		},
	)

	require.NoError(t, err)
	require.Same(t, want, got)
	assert.Equal(t, concurrentTaskVisibilityAttempts, lookupCalls)
	assert.Equal(t, concurrentTaskVisibilityAttempts-1, sleepCalls)
}

func TestReleaseSyncLockOwnerRequiresMatchingOwner(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	svc := &AlarmSyncService{redisClient: client, logger: zap.NewNop()}
	ctx := context.Background()
	lockKey := "alarm:sync:lock:SN-1"
	require.NoError(t, client.Set(ctx, lockKey, taskLockOwner("task-123"), syncLockTTL).Err())

	deleted, err := svc.releaseSyncLockOwner(ctx, lockKey, taskLockOwner("task-456"))
	require.NoError(t, err)
	assert.False(t, deleted)
	assert.True(t, mr.Exists(lockKey))

	deleted, err = svc.releaseSyncLockOwner(ctx, lockKey, taskLockOwner("task-123"))
	require.NoError(t, err)
	assert.True(t, deleted)
	assert.False(t, mr.Exists(lockKey))
}

func TestPromoteSyncLockOwnerRequiresMatchingOwner(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	svc := &AlarmSyncService{redisClient: client, logger: zap.NewNop()}
	ctx := context.Background()
	lockKey := "alarm:sync:lock:SN-2"
	require.NoError(t, client.Set(ctx, lockKey, newPendingLockOwner(), syncLockTTL).Err())
	currentOwner, getErr := client.Get(ctx, lockKey).Result()
	require.NoError(t, getErr)

	updated, err := svc.promoteSyncLockOwner(ctx, lockKey, newPendingLockOwner(), taskLockOwner("task-1"))
	require.NoError(t, err)
	assert.False(t, updated)
	owner, getErr := client.Get(ctx, lockKey).Result()
	require.NoError(t, getErr)
	assert.Equal(t, currentOwner, owner)

	updated, err = svc.promoteSyncLockOwner(ctx, lockKey, currentOwner, taskLockOwner("task-1"))
	require.NoError(t, err)
	assert.True(t, updated)
	owner, getErr = client.Get(ctx, lockKey).Result()
	require.NoError(t, getErr)
	assert.Equal(t, taskLockOwner("task-1"), owner)
}

func TestTaskIDFromLockOwner(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", taskIDFromLockOwner(""))
	assert.Equal(t, "", taskIDFromLockOwner(newPendingLockOwner()))
	assert.Equal(t, "task-1", taskIDFromLockOwner(taskLockOwner("task-1")))
	assert.Equal(t, "legacy-task", taskIDFromLockOwner("legacy-task"))
}

func TestErrAlarmSyncInProgressMapsToConflict(t *testing.T) {
	t.Parallel()

	assert.True(t, errors.Is(ErrAlarmSyncInProgress, coreerrors.ErrAlreadyExists))
}

func TestTriggerSync_UPSDeviceSkipsGPV(t *testing.T) {
	t.Parallel()

	svc := (&AlarmSyncService{logger: zap.NewNop()}).
		WithDeviceReader(&rcvMockDeviceReader{
			deviceBySN: map[string]*model.Device{
				"UPS-SN-001": {
					SerialNumber: "UPS-SN-001",
					ProductClass: "UPS_M3_BMU",
				},
			},
		})

	syncTask, err := svc.TriggerSync(context.Background(), " UPS-SN-001 ")

	require.NoError(t, err)
	assert.Nil(t, syncTask)
}

func TestSubscribeRegistersTaskTerminalSubjects(t *testing.T) {
	t.Parallel()

	b := &syncSvcMockEventBus{}
	svc := &AlarmSyncService{eventBus: b, logger: zap.NewNop()}

	err := svc.Subscribe()
	require.NoError(t, err)

	assert.Contains(t, b.queueSubCalls, event.SubjectAlarmSyncRequested+"|alarm-sync-trigger")
	assert.Contains(t, b.queueSubCalls, event.SubjectTaskFailed+"|alarm-sync-task-failed")
	assert.Contains(t, b.queueSubCalls, event.SubjectTaskCompleted+"|alarm-sync-task-completed")
	assert.Contains(t, b.queueSubCalls, event.SubjectTaskCancelled+"|alarm-sync-task-cancelled")
}
