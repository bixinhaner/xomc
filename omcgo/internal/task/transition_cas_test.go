package task

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type compensationRepairer struct {
	task     *Task
	failures int
	getCalls int
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
	r.getCalls++
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
	_, changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.LessOrEqual(t, m.TTL(q.taskKey(sent.ID)), time.Duration(0),
		"winner must persist until PostgreSQL and cross-slot cleanup are confirmed")

	retry := cloneTransitionTask(sent)
	retry.ResetForRetry()
	_, changed, err = q.prepareTransition(ctx, sent, retry, false)
	require.NoError(t, err)
	require.False(t, changed, "retry must not revive a terminal winner")
	require.Error(t, q.Update(ctx, retry), "Update must surface transition CAS conflicts")

	cancelled := cloneTransitionTask(sent)
	now := time.Now()
	cancelled.Status = TaskStatusCancelled
	cancelled.CompletedAt = &now
	_, changed, err = q.prepareTransition(ctx, sent, cancelled, false)
	require.NoError(t, err)
	require.False(t, changed, "cancel must not overwrite a terminal winner")

	expired := cloneTransitionTask(sent)
	expired.MarkExpired()
	_, changed, err = q.prepareTransition(ctx, sent, expired, false)
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

	token, changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.LessOrEqual(t, m.TTL(q.taskKey(sent.ID)), time.Duration(0))
	require.Equal(t, []string{sent.ID}, mustPendingTransitionIDs(t, q, ctx))

	m.FastForward(24 * time.Hour)
	require.True(t, m.Exists(q.taskKey(sent.ID)),
		"PG outage longer than the former 4h TTL must not lose the only terminal state")

	require.NoError(t, q.acknowledgeTransition(ctx, sent.ID, token))
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

	token, changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)

	q.cleanupTransition = func(context.Context, *Task, string) error {
		return errors.New("redis cluster slot unavailable")
	}
	require.Error(t, q.acknowledgeTransition(ctx, sent.ID, token))
	require.LessOrEqual(t, m.TTL(q.taskKey(sent.ID)), time.Duration(0))
	require.Equal(t, []string{sent.ID}, mustPendingTransitionIDs(t, q, ctx))

	q.cleanupTransition = nil
	require.NoError(t, q.acknowledgeTransition(ctx, sent.ID, token))
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
		_, changed, err := q.prepareTransition(ctx, sent, completed, false)
		require.NoError(t, err)
		require.True(t, changed)
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, []string{"first", "second", "third"}, mustPendingTransitionIDs(t, q, ctx))

	refs, err := q.listPendingTransitions(ctx, 100)
	require.NoError(t, err)
	require.NoError(t, q.deferPendingTransition(ctx, refs[0].TaskID, refs[0].Token))
	require.Equal(t, []string{"second", "third", "first"}, mustPendingTransitionIDs(t, q, ctx))
}

func TestReconciler_PGOutageBeyondTTLRetainsAndRepairsTransition(t *testing.T) {
	q, m := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "pg-outage-retained")
	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(json.RawMessage(`{"ok":true}`))
	_, changed, err := q.prepareTransition(ctx, sent, completed, false)
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

func TestRedisTaskQueue_OldAckCannotClearNewTransition(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "old-ack-new-transition")
	failed := cloneTransitionTask(sent)
	failed.MarkFailed(9002, "retryable")
	oldToken, changed, err := q.prepareTransition(ctx, sent, failed, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, q.acknowledgeTransition(ctx, sent.ID, oldToken))

	pending := cloneTransitionTask(failed)
	pending.MaxRetries = 2
	pending.ResetForRetry()
	newToken, changed, err := q.prepareTransition(ctx, failed, pending, false)
	require.NoError(t, err)
	require.True(t, changed)

	require.Error(t, q.acknowledgeTransition(ctx, sent.ID, oldToken))
	values, err := q.client.HMGet(
		ctx, q.taskKey(sent.ID), "transition_token", "pg_sync_pending", "cleanup_pending",
	).Result()
	require.NoError(t, err)
	require.Equal(t, newToken, values[0])
	require.Equal(t, "1", values[1])
	require.Equal(t, "1", values[2])

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			require.Error(t, q.acknowledgeTransition(ctx, sent.ID, oldToken))
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		require.NoError(t, q.acknowledgeTransition(ctx, sent.ID, newToken))
	}()
	close(start)
	wg.Wait()

	got, err := q.GetByID(ctx, sent.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStatusPending, got.Status)
	require.Empty(t, mustPendingTransitionIDs(t, q, ctx))
}

func TestRedisTaskQueue_WinnerTTLIsUsedAcrossProcesses(t *testing.T) {
	qWinner, m := newRedisQueueWithTerminalTTL(t, 20*time.Minute)
	qReconciler := NewRedisTaskQueueWithTerminalTTL(qWinner.client, 10*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, qWinner, "cross-process-ttl")
	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(nil)
	token, changed, err := qWinner.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)

	require.NoError(t, qReconciler.acknowledgeTransition(ctx, sent.ID, token))
	require.Equal(t, 20*time.Minute, m.TTL(qWinner.taskKey(sent.ID)))
}

func TestReconciler_PGRecoveryPublishesStableEventOnce(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	sent := sentTaskForTransition(t, q, "event-outbox")
	sent.SourceID = "source-event-outbox"
	require.NoError(t, q.Update(ctx, sent))
	completed := cloneTransitionTask(sent)
	completed.MarkCompleted(nil)
	token, changed, err := q.prepareTransition(ctx, sent, completed, false)
	require.NoError(t, err)
	require.True(t, changed)

	repairer := &compensationRepairer{task: cloneTransitionTask(sent), failures: 1}
	var attempts, published int
	var eventIDs []string
	reconciler := NewReconciler(
		&fakeActiveLister{}, q, repairer, nil,
		time.Second, time.Minute, 10, nil,
	).WithTransitionPublisher(func(ctx context.Context, task *Task, gotToken string) error {
		attempts++
		eventIDs = append(eventIDs, "task-transition-"+gotToken)
		if attempts == 1 {
			return errors.New("nats unavailable")
		}
		published++
		return q.acknowledgeTransitionEvent(ctx, task.ID, gotToken)
	})

	_, err = reconciler.ReconcileOnce(ctx) // PG fails: event must not publish.
	require.NoError(t, err)
	require.Zero(t, attempts)
	_, err = reconciler.ReconcileOnce(ctx) // PG succeeds, first publish fails.
	require.NoError(t, err)
	require.Equal(t, 1, attempts)
	require.Equal(t, []string{sent.ID}, mustPendingTransitionIDs(t, q, ctx))
	require.True(t, q.client.HExists(ctx, q.taskKey(sent.ID), "data").Val(),
		"recovery payload must remain intact until the terminal event is acknowledged")
	_, err = reconciler.ReconcileOnce(ctx) // Same stable event id is retried.
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
	require.Equal(t, 1, published)
	require.Equal(t, []string{
		"task-transition-" + token,
		"task-transition-" + token,
	}, eventIDs)
	require.Empty(t, mustPendingTransitionIDs(t, q, ctx))
	require.False(t, q.client.HExists(ctx, q.taskKey(sent.ID), "data").Val(),
		"the final event acknowledgement must atomically compact the terminal payload")
	tombstone, err := q.GetByID(ctx, sent.ID)
	require.NoError(t, err)
	require.Equal(t, &Task{ID: sent.ID, Status: TaskStatusCompleted}, tombstone)
	_, err = reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
}

func TestRedisTaskQueue_SentCleanupFailureRollbackRestoresPending(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("sent-cleanup-rollback", "SN-ROLLBACK", "Reboot")
	require.NoError(t, q.Push(ctx, task))
	q.cleanupTransition = func(context.Context, *Task, string) error {
		return errors.New("cluster slot unavailable")
	}

	err := q.MarkTaskSent(ctx, task.ID, "cwmp-rollback")
	var transitionErr *taskTransitionError
	require.ErrorAs(t, err, &transitionErr)
	stuck, getErr := q.GetByID(ctx, task.ID)
	require.NoError(t, getErr)
	require.Equal(t, TaskStatusSent, stuck.Status,
		"regression: PG rollback used to leave Redis permanently sent")

	pending := cloneTransitionTask(stuck)
	pending.Status = TaskStatusPending
	pending.CWMPID = ""
	pending.SentAt = nil
	q.cleanupTransition = nil
	require.NoError(t, q.rollbackSentTransition(
		ctx, pending, "cwmp-rollback", transitionErr.token,
	))
	got, getErr := q.GetByID(ctx, task.ID)
	require.NoError(t, getErr)
	require.Equal(t, TaskStatusPending, got.Status)
	require.Equal(t, int64(1), mustQueueLen(t, q, ctx, task.DeviceSN))
}

func TestReconciler_RetryCleanupAfterSentRollback(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("sent-rollback-reconcile", "SN-ROLLBACK-REC", "Reboot")
	require.NoError(t, q.Push(ctx, task))
	q.cleanupTransition = func(context.Context, *Task, string) error {
		return errors.New("cluster slot unavailable")
	}
	err := q.MarkTaskSent(ctx, task.ID, "cwmp-rollback-rec")
	var transitionErr *taskTransitionError
	require.ErrorAs(t, err, &transitionErr)

	sent, getErr := q.GetByID(ctx, task.ID)
	require.NoError(t, getErr)
	pending := cloneTransitionTask(sent)
	pending.Status = TaskStatusPending
	pending.CWMPID = ""
	pending.SentAt = nil
	require.Error(t, q.rollbackSentTransition(
		ctx, pending, "cwmp-rollback-rec", transitionErr.token,
	))

	repairer := &compensationRepairer{task: cloneTransitionTask(pending)}
	reconciler := NewReconciler(
		&fakeActiveLister{}, q, repairer, nil,
		time.Second, time.Minute, 10, nil,
	)
	stats, err := reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.RepairFailed)
	q.cleanupTransition = nil
	stats, err = reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Repaired)
	got, getErr := q.GetByID(ctx, task.ID)
	require.NoError(t, getErr)
	require.Equal(t, TaskStatusPending, got.Status)
	require.Equal(t, int64(1), mustQueueLen(t, q, ctx, task.DeviceSN))
}

func TestReconciler_PGRollbackWinsWhenSentTransitionStillNeedsPGSync(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	pending := newTaskForQueue("sent-pg-rollback", "SN-PG-ROLLBACK", "Reboot")
	require.NoError(t, q.Push(ctx, pending))
	sent := cloneTransitionTask(pending)
	sent.MarkSent("cwmp-pg-rollback")
	_, changed, err := q.prepareTransition(ctx, pending, sent, false)
	require.NoError(t, err)
	require.True(t, changed)

	repairer := &compensationRepairer{task: cloneTransitionTask(pending)}
	reconciler := NewReconciler(
		&fakeActiveLister{}, q, repairer, nil,
		time.Second, time.Minute, 10, nil,
	)
	stats, err := reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Repaired)
	require.Equal(t, TaskStatusPending, repairer.task.Status,
		"reconciler must not revive PG pending back to sent after rollback")
	got, err := q.GetByID(ctx, pending.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStatusPending, got.Status)
	require.Equal(t, int64(1), mustQueueLen(t, q, ctx, pending.DeviceSN))
}

func TestReconciler_AckResponseLostStillHonorsPGRollback(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	pending := newTaskForQueue("sent-ack-response-lost", "SN-ACK-LOST", "Reboot")
	require.NoError(t, q.Push(ctx, pending))
	sent := cloneTransitionTask(pending)
	sent.MarkSent("cwmp-ack-lost")
	token, changed, err := q.prepareTransition(ctx, pending, sent, false)
	require.NoError(t, err)
	require.True(t, changed)

	// Redis executed the PG acknowledgement, but the client lost the response
	// before cleanup. The service then rolled PG back while Redis was unavailable.
	result, err := acknowledgeTaskTransitionFieldScript.Run(
		ctx, q.client, []string{q.taskKey(pending.ID)}, token, "pg_sync_pending",
	).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1), result)

	repairer := &compensationRepairer{task: cloneTransitionTask(pending)}
	reconciler := NewReconciler(
		&fakeActiveLister{}, q, repairer, nil,
		time.Second, time.Minute, 10, nil,
	)
	stats, err := reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Repaired)
	require.Equal(t, 1, repairer.getCalls)
	got, err := q.GetByID(ctx, pending.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStatusPending, got.Status)
	require.Equal(t, int64(1), mustQueueLen(t, q, ctx, pending.DeviceSN))
}

func TestReconciler_AcknowledgedSentWithPGSentIsNotRolledBack(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	pending := newTaskForQueue("sent-pg-still-sent", "SN-PG-SENT", "Reboot")
	require.NoError(t, q.Push(ctx, pending))
	sent := cloneTransitionTask(pending)
	sent.MarkSent("cwmp-pg-sent")
	token, changed, err := q.prepareTransition(ctx, pending, sent, false)
	require.NoError(t, err)
	require.True(t, changed)
	_, err = acknowledgeTaskTransitionFieldScript.Run(
		ctx, q.client, []string{q.taskKey(pending.ID)}, token, "pg_sync_pending",
	).Result()
	require.NoError(t, err)

	repairer := &compensationRepairer{task: cloneTransitionTask(sent)}
	reconciler := NewReconciler(
		&fakeActiveLister{}, q, repairer, nil,
		time.Second, time.Minute, 1, nil,
	)
	stats, err := reconciler.ReconcileOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Repaired)
	require.Equal(t, 1, repairer.getCalls,
		"sent PG verification must remain bounded by the reconciler batch")
	got, err := q.GetByID(ctx, pending.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStatusSent, got.Status)
	require.Zero(t, mustQueueLen(t, q, ctx, pending.DeviceSN))
	byCWMP, err := q.GetByCWMPID(ctx, sent.CWMPID)
	require.NoError(t, err)
	require.NotNil(t, byCWMP)
}

func TestRedisTaskQueue_OldSentRollbackCannotOverwriteNewTransition(t *testing.T) {
	q, _ := newRedisQueueWithTerminalTTL(t, 15*time.Minute)
	ctx := context.Background()
	task := newTaskForQueue("old-sent-rollback", "SN-OLD-ROLLBACK", "Reboot")
	require.NoError(t, q.Push(ctx, task))
	current, err := q.GetByID(ctx, task.ID)
	require.NoError(t, err)
	sent := cloneTransitionTask(current)
	sent.MarkSent("cwmp-old")
	oldToken, changed, err := q.prepareTransition(ctx, current, sent, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, q.acknowledgeTransition(ctx, task.ID, oldToken))

	pending := cloneTransitionTask(sent)
	pending.Status = TaskStatusPending
	pending.CWMPID = ""
	pending.SentAt = nil
	tokenPending, changed, err := q.prepareTransition(ctx, sent, pending, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, q.acknowledgeTransition(ctx, task.ID, tokenPending))

	sentAgain := cloneTransitionTask(pending)
	sentAgain.MarkSent("cwmp-new")
	newToken, changed, err := q.prepareTransition(ctx, pending, sentAgain, false)
	require.NoError(t, err)
	require.True(t, changed)
	require.Error(t, q.rollbackSentTransition(ctx, pending, "cwmp-old", oldToken))
	values, err := q.client.HMGet(
		ctx, q.taskKey(task.ID), "status", "transition_token",
	).Result()
	require.NoError(t, err)
	require.Equal(t, string(TaskStatusSent), values[0])
	require.Equal(t, newToken, values[1])
}

func mustPendingTransitionIDs(
	t *testing.T,
	q *RedisTaskQueue,
	ctx context.Context,
) []string {
	t.Helper()
	refs, err := q.listPendingTransitions(ctx, 100)
	require.NoError(t, err)
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.TaskID)
	}
	return ids
}
