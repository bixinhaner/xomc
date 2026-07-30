package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type compensationRepairer struct {
	task     *Task
	failures int
}

func (r *compensationRepairer) Update(context.Context, *Task) error { return nil }

func (r *compensationRepairer) TransitionIfStatus(
	_ context.Context, task *Task, from TaskStatus,
) (bool, error) {
	if r.failures > 0 {
		r.failures--
		return false, errors.New("postgres unavailable")
	}
	if r.task == nil || r.task.Status != from {
		return false, nil
	}
	r.task = cloneTransitionTask(task)
	return true, nil
}

func (r *compensationRepairer) GetByID(context.Context, string) (*Task, error) {
	return cloneTransitionTask(r.task), nil
}

func sentTaskForTransition(t *testing.T, q *RedisTaskQueue, id string) *Task {
	t.Helper()
	ctx := context.Background()
	task := newTaskForQueue(id, "SN-"+id, "GetParameterValues")
	require.NoError(t, q.Push(ctx, task))
	require.NoError(t, q.MarkTaskSent(ctx, task.ID, "cwmp-"+id))
	sent, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, sent)
	return sent
}

func cloneTransitionTask(task *Task) *Task {
	clone := *task
	clone.Params = append(json.RawMessage(nil), task.Params...)
	clone.Result = append(json.RawMessage(nil), task.Result...)
	return &clone
}

func TestRedisTaskQueue_FirstTransitionWinnerFencesRetryCancelAndExpire(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "cas-first-winner")

	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(json.RawMessage(`{"winner":"completed"}`))
	changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.LessOrEqual(t, m.TTL(q.taskKey(sent.ID)), time.Duration(0),
		"winner must persist until PostgreSQL and cross-slot cleanup are confirmed")

	retry := cloneTransitionTask(sent)
	retry.ResetForRetry()
	changed, err = q.prepareTransition(ctx, sent, retry, false)
	require.NoError(t, err)
	require.False(t, changed, "retry must not revive a terminal winner")

	cancelled := cloneTransitionTask(sent)
	now := time.Now()
	cancelled.Status = TaskStatusCancelled
	cancelled.CompletedAt = &now
	changed, err = q.prepareTransition(ctx, sent, cancelled, false)
	require.NoError(t, err)
	require.False(t, changed, "cancel must not overwrite a terminal winner")

	expired := cloneTransitionTask(sent)
	expired.MarkExpired()
	changed, err = q.prepareTransition(ctx, sent, expired, false)
	require.NoError(t, err)
	require.False(t, changed, "expire must not overwrite a terminal winner")

	got, err := q.GetByID(ctx, sent.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStatusCompleted, got.Status)
	require.JSONEq(t, `{"winner":"completed"}`, string(got.Result))
}

func TestRedisTaskQueue_TerminalTTLStartsOnlyAfterPGAndCleanupAck(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "ack-gated-ttl")
	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(json.RawMessage(`{"ok":true}`))

	changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.LessOrEqual(t, m.TTL(q.taskKey(sent.ID)), time.Duration(0))
	require.Equal(t, []string{sent.ID}, mustPendingTransitionIDs(t, q, ctx))

	m.FastForward(24 * time.Hour)
	require.True(t, m.Exists(q.taskKey(sent.ID)),
		"PG outage longer than the former 4h TTL must not lose the only terminal state")

	require.NoError(t, q.acknowledgeTransition(ctx, sent.ID))
	require.Equal(t, 15*time.Minute, m.TTL(q.taskKey(sent.ID)))
	require.Empty(t, mustPendingTransitionIDs(t, q, ctx))
	require.False(t, m.Exists(q.cwmpKey(sent.CWMPID)))
	require.Zero(t, mustQueueLen(t, q, ctx, sent.DeviceSN))
}

func TestRedisTaskQueue_CleanupFailureRemainsDurableAndRetryable(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "cleanup-retry")
	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(nil)

	changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)

	q.cleanupTransition = func(context.Context, *Task, string) error {
		return errors.New("redis cluster slot unavailable")
	}
	require.Error(t, q.acknowledgeTransition(ctx, sent.ID))
	require.LessOrEqual(t, m.TTL(q.taskKey(sent.ID)), time.Duration(0))
	require.Equal(t, []string{sent.ID}, mustPendingTransitionIDs(t, q, ctx))

	q.cleanupTransition = nil
	require.NoError(t, q.acknowledgeTransition(ctx, sent.ID))
	require.Equal(t, 15*time.Minute, m.TTL(q.taskKey(sent.ID)))
	require.Empty(t, mustPendingTransitionIDs(t, q, ctx))
}

func TestRedisTaskQueue_DeferPendingTransitionMovesFailureToTail(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	for _, id := range []string{"first", "second", "third"} {
		sent := sentTaskForTransition(t, q, id)
		completed := cloneTransitionTask(sent)
		completed.MarkCompleted(nil)
		changed, err := q.prepareTransition(ctx, sent, completed, false)
		require.NoError(t, err)
		require.True(t, changed)
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, []string{"first", "second", "third"}, mustPendingTransitionIDs(t, q, ctx))

	require.NoError(t, q.deferPendingTransition(ctx, "first"))
	require.Equal(t, []string{"second", "third", "first"}, mustPendingTransitionIDs(t, q, ctx))
}

func TestReconciler_PGOutageBeyondTTLRetainsAndRepairsTransition(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "pg-outage-retained")
	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(json.RawMessage(`{"ok":true}`))
	changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)

	repairer := &compensationRepairer{
		task:     cloneTransitionTask(sent),
		failures: 1,
	}
	reconciler := NewReconciler(
		&fakeActiveLister{}, q, repairer, nil,
		time.Second, time.Minute, 10, nil,
	)
	stats, err := reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.RepairFailed)

	m.FastForward(24 * time.Hour)
	require.True(t, m.Exists(q.taskKey(sent.ID)))
	require.Equal(t, []string{sent.ID}, mustPendingTransitionIDs(t, q, ctx))

	stats, err = reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Repaired)
	require.Equal(t, TaskStatusCompleted, repairer.task.Status)
	require.Equal(t, 15*time.Minute, m.TTL(q.taskKey(sent.ID)))
	require.Empty(t, mustPendingTransitionIDs(t, q, ctx))
}

func mustPendingTransitionIDs(
	t *testing.T,
	q *RedisTaskQueue,
	ctx context.Context,
) []string {
	t.Helper()
	ids, err := q.listPendingTransitionIDs(ctx, 100)
	require.NoError(t, err)
	return ids
}
