package task

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newRedisQueueWithTerminalTTL(
	t *testing.T,
	ttl time.Duration,
) (*RedisTaskQueue, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	return NewRedisTaskQueueWithTerminalTTL(client, ttl), m
}

func TestRedisTaskQueue_TerminalTransitionsUseShortTTLAndCleanIndexes(t *testing.T) {
	const configuredTTL = 15 * time.Minute
	tests := []struct {
		name       string
		transition func(context.Context, *RedisTaskQueue, *Task) error
		wantStatus TaskStatus
	}{
		{
			name: "completed",
			transition: func(ctx context.Context, q *RedisTaskQueue, task *Task) error {
				return q.MarkTaskCompleted(ctx, task.ID, json.RawMessage(`{"ok":true}`))
			},
			wantStatus: TaskStatusCompleted,
		},
		{
			name: "final_failed",
			transition: func(ctx context.Context, q *RedisTaskQueue, task *Task) error {
				return q.MarkTaskFailed(ctx, task.ID, 9002, "internal error")
			},
			wantStatus: TaskStatusFailed,
		},
		{
			name: "cancelled",
			transition: func(ctx context.Context, q *RedisTaskQueue, task *Task) error {
				now := time.Now()
				task.Status = TaskStatusCancelled
				task.CompletedAt = &now
				return q.Update(ctx, task)
			},
			wantStatus: TaskStatusCancelled,
		},
		{
			name: "expired",
			transition: func(ctx context.Context, q *RedisTaskQueue, task *Task) error {
				task.MarkExpired()
				return q.Update(ctx, task)
			},
			wantStatus: TaskStatusExpired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, m := newRedisQueueWithTerminalTTL(t, configuredTTL)
			ctx := context.Background()
			task := newTaskForQueue("task-"+tc.name, "SN-"+tc.name, "GetParameterValues")
			require.NoError(t, q.Push(ctx, task))
			require.NoError(t, q.MarkTaskSent(ctx, task.ID, "cwmp-"+tc.name))
			task, err := q.GetByID(ctx, task.ID)
			require.NoError(t, err)
			require.NotNil(t, task)

			// Recreate a stale executable member to prove every terminal path
			// removes queue residue, including cancellation and recovery updates.
			require.NoError(t, q.client.ZAdd(ctx, q.queueKey(task.DeviceSN), redis.Z{
				Score:  queueScore(task),
				Member: task.ID,
			}).Err())

			require.NoError(t, tc.transition(ctx, q, task))

			got, err := q.GetByID(ctx, task.ID)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, task.ID, got.ID)
			require.Equal(t, tc.wantStatus, got.Status)
			require.Empty(t, got.Params)
			require.Empty(t, got.Result)
			require.False(t, q.client.HExists(ctx, q.taskKey(task.ID), "data").Val(),
				"fully acknowledged terminal tasks must retain only a small fence tombstone")
			require.Equal(t, []string{"status"}, q.client.HKeys(ctx, q.taskKey(task.ID)).Val())
			require.Equal(t, configuredTTL, m.TTL(q.taskKey(task.ID)))
			require.Zero(t, mustQueueLen(t, q, ctx, task.DeviceSN))
			require.False(t, m.Exists(q.cwmpKey("cwmp-"+tc.name)))
		})
	}
}

func TestRedisTaskQueue_NonTerminalAndRetryKeepLongTTL(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("task-retry", "SN-RETRY", "GetParameterValues")
	task.Source = TaskSourceMML
	task.MaxRetries = 2

	require.NoError(t, q.Push(ctx, task))
	require.Equal(t, taskDetailTTL, m.TTL(q.taskKey(task.ID)))

	require.NoError(t, q.MarkTaskSent(ctx, task.ID, "cwmp-retry-1"))
	require.Equal(t, taskDetailTTL, m.TTL(q.taskKey(task.ID)))

	stored, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	stored.ErrorCode = 9002
	stored.ErrorMessage = "retryable failure"
	stored.ResetForRetryAfter(time.Minute)
	require.NoError(t, q.Update(ctx, stored))

	retried, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, retried)
	require.Equal(t, TaskStatusPending, retried.Status)
	require.Equal(t, 1, retried.RetryCount)
	require.Equal(t, taskDetailTTL, m.TTL(q.taskKey(task.ID)))
	require.True(t, m.Exists(q.queueKey(task.DeviceSN)))
}

func TestRedisTaskQueue_TerminalTransitionIsIdempotentForLateDuplicate(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("task-duplicate", "SN-DUP", "GetParameterValues")
	require.NoError(t, q.Push(ctx, task))
	require.NoError(t, q.MarkTaskSent(ctx, task.ID, "cwmp-duplicate"))
	require.NoError(t, q.MarkTaskCompleted(ctx, task.ID, json.RawMessage(`{"winner":"first"}`)))

	first, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Equal(t, TaskStatusCompleted, first.Status)
	require.False(t, q.client.HExists(ctx, q.taskKey(task.ID), "data").Val())
	firstTTL := m.TTL(q.taskKey(task.ID))

	m.FastForward(time.Minute)
	require.NoError(t, q.MarkTaskFailed(ctx, task.ID, 9002, "late duplicate fault"))

	afterDuplicate, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, afterDuplicate)
	require.Equal(t, TaskStatusCompleted, afterDuplicate.Status)
	require.Empty(t, afterDuplicate.Result)
	require.Equal(t, firstTTL-time.Minute, m.TTL(q.taskKey(task.ID)),
		"late duplicate must neither overwrite state nor extend retention")
}

func TestRedisTaskQueue_LegacyTerminalHashWithoutStatusFieldIsStillFenced(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	completed := newTaskForQueue("task-legacy-terminal", "SN-LEGACY", "GetParameterValues")
	completed.MarkCompleted(json.RawMessage(`{"winner":"legacy"}`))
	data, err := json.Marshal(completed)
	require.NoError(t, err)

	// Simulate a terminal Hash written by the previous version, which only had
	// the "data" field and retained the original four-hour TTL.
	require.NoError(t, q.client.HSet(ctx, q.taskKey(completed.ID), "data", data).Err())
	require.NoError(t, q.client.Expire(ctx, q.taskKey(completed.ID), taskDetailTTL).Err())

	require.NoError(t, q.MarkTaskFailed(ctx, completed.ID, 9002, "late fault"))
	got, err := q.GetByID(ctx, completed.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, TaskStatusCompleted, got.Status)
	require.JSONEq(t, `{"winner":"legacy"}`, string(got.Result))
	require.Equal(t, taskDetailTTL, m.TTL(q.taskKey(completed.ID)),
		"legacy terminal duplicate must not mutate or extend the inherited TTL")
}

func TestRedisTaskQueue_ConcurrentTerminalTransitionsHaveSingleWinner(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("task-race-terminal", "SN-RACE-TERM", "GetParameterValues")
	require.NoError(t, q.Push(ctx, task))
	require.NoError(t, q.MarkTaskSent(ctx, task.ID, "cwmp-race-terminal"))

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		errs <- q.MarkTaskCompleted(ctx, task.ID, json.RawMessage(`{"winner":"success"}`))
	}()
	go func() {
		defer wg.Done()
		<-start
		errs <- q.MarkTaskFailed(ctx, task.ID, 9002, "failure winner")
	}()
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	got, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Contains(t, []TaskStatus{TaskStatusCompleted, TaskStatusFailed}, got.Status)
	require.Empty(t, got.Params)
	require.Empty(t, got.Result)
	require.Empty(t, got.ErrorMessage)
	require.False(t, q.client.HExists(ctx, q.taskKey(task.ID), "data").Val())
	require.Equal(t, 15*time.Minute, m.TTL(q.taskKey(task.ID)))
	require.False(t, m.Exists(q.cwmpKey("cwmp-race-terminal")))
}

func TestRedisTaskQueue_TerminalTTLHasSafeLowerBound(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, time.Second)
	ctx := context.Background()
	task := newTaskForQueue("task-clamped", "SN-CLAMP", "Reboot")
	require.NoError(t, q.Push(ctx, task))
	require.NoError(t, q.MarkTaskCompleted(ctx, task.ID, nil))
	require.Equal(t, minimumTerminalTaskTTL, m.TTL(q.taskKey(task.ID)))
}

func TestRedisTaskQueue_UpdateCanMaterializeTerminalTombstoneFromDurableTask(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("task-durable-terminal", "SN-DURABLE", "Reboot")
	task.SourceID = generateUUID()
	task.MarkExpired()

	require.NoError(t, q.Update(ctx, task))
	got, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, TaskStatusExpired, got.Status)
	require.False(t, q.client.HExists(ctx, q.taskKey(task.ID), "data").Val())
	require.Equal(t, []string{"status"}, q.client.HKeys(ctx, q.taskKey(task.ID)).Val())
	require.Equal(t, 15*time.Minute, m.TTL(q.taskKey(task.ID)))
}

func TestTerminalTaskCapacityModelAtObservedRate(t *testing.T) {
	const (
		observedTasksPerMinute = int64(40000)
		maxBytesPerTombstone   = int64(512)
		maxSteadyStateBytes    = int64(512 * 1024 * 1024)
	)
	retainedTasks := observedTasksPerMinute * int64(defaultTerminalTaskTTL/time.Minute)
	steadyStateBytes := retainedTasks * maxBytesPerTombstone
	require.Less(t, steadyStateBytes, maxSteadyStateBytes,
		"15-minute terminal tombstones must keep the observed ACS burst below 0.5 GiB")
}
