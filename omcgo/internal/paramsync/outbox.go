package paramsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/task"
)

type OutboxDispatcher struct {
	pool        *pgxpool.Pool
	lifecycle   PlannedTaskLifecycle
	bus         event.EventBus
	maxAttempts int
	now         func() time.Time
}

const parameterSyncOutboxDeliveringIndexSQL = `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_parameter_sync_outbox_delivering_updated
ON parameter_sync_outbox (updated_at) WHERE status = 'delivering'`

const parameterSyncOutboxStatusIndexSQL = `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_parameter_sync_outbox_terminal_status
ON parameter_sync_outbox (status) WHERE status IN ('delivered', 'dead')`

const parameterSyncActiveRunConvergenceIndexSQL = `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_parameter_sync_runs_active_convergence
ON parameter_sync_runs (started_at, id)
WHERE status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling')`

const parameterSyncOutboxIndexLockSQL = `SELECT pg_advisory_lock(hashtextextended('omc.paramsync.outbox.performance_indexes', 0))`
const parameterSyncOutboxIndexUnlockSQL = `SELECT pg_advisory_unlock(hashtextextended('omc.paramsync.outbox.performance_indexes', 0))`

type outboxRuntimeIndex struct {
	qualifiedName string
	createSQL     string
	dropSQL       string
}

var parameterSyncOutboxRuntimeIndexes = []outboxRuntimeIndex{
	{
		qualifiedName: "public.idx_parameter_sync_outbox_delivering_updated",
		createSQL:     parameterSyncOutboxDeliveringIndexSQL,
		dropSQL:       "DROP INDEX CONCURRENTLY IF EXISTS public.idx_parameter_sync_outbox_delivering_updated",
	},
	{
		qualifiedName: "public.idx_parameter_sync_outbox_terminal_status",
		createSQL:     parameterSyncOutboxStatusIndexSQL,
		dropSQL:       "DROP INDEX CONCURRENTLY IF EXISTS public.idx_parameter_sync_outbox_terminal_status",
	},
	{
		qualifiedName: "public.idx_parameter_sync_runs_active_convergence",
		createSQL:     parameterSyncActiveRunConvergenceIndexSQL,
		dropSQL:       "DROP INDEX CONCURRENTLY IF EXISTS public.idx_parameter_sync_runs_active_convergence",
	},
}

type PlannedTaskLifecycle interface {
	ReleasePlannedTask(ctx context.Context, planned *task.Task) (bool, error)
	EvictPlannedTask(ctx context.Context, planned *task.Task) error
}

type coalescedWakePlannedTaskLifecycle interface {
	ReleasePlannedTaskWithoutWake(ctx context.Context, planned *task.Task) (bool, error)
	WakePlannedDevice(deviceSN string)
}

type plannedTaskReleaseBatch struct {
	lifecycle   PlannedTaskLifecycle
	coalesced   coalescedWakePlannedTaskLifecycle
	mu          sync.Mutex
	wakeDevices map[string]struct{}
}

func newPlannedTaskReleaseBatch(lifecycle PlannedTaskLifecycle) *plannedTaskReleaseBatch {
	coalesced, _ := lifecycle.(coalescedWakePlannedTaskLifecycle)
	return &plannedTaskReleaseBatch{
		lifecycle:   lifecycle,
		coalesced:   coalesced,
		wakeDevices: make(map[string]struct{}),
	}
}

func (b *plannedTaskReleaseBatch) Release(ctx context.Context, planned *task.Task) (bool, error) {
	if b.coalesced == nil {
		return b.lifecycle.ReleasePlannedTask(ctx, planned)
	}
	released, err := b.coalesced.ReleasePlannedTaskWithoutWake(ctx, planned)
	if err == nil && released && planned != nil && planned.DeviceSN != "" {
		b.mu.Lock()
		b.wakeDevices[planned.DeviceSN] = struct{}{}
		b.mu.Unlock()
	}
	return released, err
}

func (b *plannedTaskReleaseBatch) WakeDevices() {
	if b.coalesced == nil {
		return
	}
	b.mu.Lock()
	devices := make([]string, 0, len(b.wakeDevices))
	for deviceSN := range b.wakeDevices {
		devices = append(devices, deviceSN)
	}
	b.mu.Unlock()
	for _, deviceSN := range devices {
		b.coalesced.WakePlannedDevice(deviceSN)
	}
}

type outboxTask struct {
	ID           uuid.UUID
	EventType    string
	Payload      []byte
	AttemptCount int
}

func (d *OutboxDispatcher) WithEventBus(bus event.EventBus) *OutboxDispatcher {
	d.bus = bus
	return d
}

func NewOutboxDispatcher(pool *pgxpool.Pool, lifecycle PlannedTaskLifecycle, maxAttempts int) *OutboxDispatcher {
	if maxAttempts <= 0 {
		maxAttempts = 20
	}
	return &OutboxDispatcher{pool: pool, lifecycle: lifecycle, maxAttempts: maxAttempts, now: func() time.Time { return time.Now().UTC() }}
}

func (d *OutboxDispatcher) DispatchPending(ctx context.Context, limit int) (int, error) {
	releaseBatch := newPlannedTaskReleaseBatch(d.lifecycle)
	defer releaseBatch.WakeDevices()
	return d.dispatchPending(ctx, limit, releaseBatch)
}

func (d *OutboxDispatcher) DispatchPendingConcurrent(
	ctx context.Context,
	workers int,
	batchSize int,
) (int, error) {
	if workers <= 1 {
		return d.DispatchPending(ctx, batchSize)
	}
	releaseBatch := newPlannedTaskReleaseBatch(d.lifecycle)
	if releaseBatch.coalesced == nil {
		return d.DispatchPending(ctx, workers*batchSize)
	}
	defer releaseBatch.WakeDevices()
	return runConcurrentOutboxBatches(workers, func() (int, error) {
		return d.dispatchPending(ctx, batchSize, releaseBatch)
	})
}

func runConcurrentOutboxBatches(workers int, dispatch func() (int, error)) (int, error) {
	if workers < 1 {
		workers = 1
	}
	type result struct {
		delivered int
		err       error
	}
	results := make(chan result, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			delivered, err := dispatch()
			results <- result{delivered: delivered, err: err}
		}()
	}
	wg.Wait()
	close(results)

	var delivered int
	var errs []error
	for item := range results {
		delivered += item.delivered
		if item.err != nil {
			errs = append(errs, item.err)
		}
	}
	return delivered, errors.Join(errs...)
}

func (d *OutboxDispatcher) dispatchPending(
	ctx context.Context,
	limit int,
	releaseBatch *plannedTaskReleaseBatch,
) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	claimed, err := d.claim(ctx, limit)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, item := range claimed {
		if item.EventType != "param_sync.task.enqueue" && item.EventType != "param_sync.task.cancel" {
			if d.bus == nil {
				if markErr := d.markFailure(ctx, item, fmt.Errorf("event bus is not configured")); markErr != nil {
					return delivered, markErr
				}
				continue
			}
			evt := event.Event{ID: uuid.NewString(), Subject: item.EventType, Payload: item.Payload, Timestamp: d.now()}
			if err := d.bus.Publish(ctx, item.EventType, evt); err != nil {
				if markErr := d.markFailure(ctx, item, err); markErr != nil {
					return delivered, markErr
				}
				continue
			}
			if err := d.markDelivered(ctx, item.ID); err != nil {
				return delivered, err
			}
			delivered++
			continue
		}
		var planned task.Task
		if err := json.Unmarshal(item.Payload, &planned); err != nil {
			if markErr := d.markFailure(ctx, item, fmt.Errorf("decode task payload: %w", err)); markErr != nil {
				return delivered, markErr
			}
			continue
		}
		if item.EventType == "param_sync.task.cancel" {
			if err := d.lifecycle.EvictPlannedTask(ctx, &planned); err != nil {
				if markErr := d.markFailure(ctx, item, err); markErr != nil {
					return delivered, markErr
				}
				continue
			}
			if err := d.markDelivered(ctx, item.ID); err != nil {
				return delivered, err
			}
			delivered++
			continue
		}
		allowed, err := d.releaseAllowed(ctx, &planned)
		if err != nil {
			if markErr := d.markFailure(ctx, item, err); markErr != nil {
				return delivered, markErr
			}
			continue
		}
		if !allowed {
			if err := d.markDelivered(ctx, item.ID); err != nil {
				return delivered, err
			}
			delivered++
			continue
		}
		if _, err := releaseBatch.Release(ctx, &planned); err != nil {
			if markErr := d.markFailure(ctx, item, err); markErr != nil {
				return delivered, markErr
			}
			continue
		}
		// Fence the Push-vs-Cancel race. A cancelling transaction may commit
		// between the first status check and Redis Push; re-check and evict the
		// just-released task before acknowledging the enqueue outbox.
		allowed, err = d.releaseAllowed(ctx, &planned)
		if err != nil {
			if markErr := d.markFailure(ctx, item, err); markErr != nil {
				return delivered, markErr
			}
			continue
		}
		if !allowed {
			if err := d.lifecycle.EvictPlannedTask(ctx, &planned); err != nil {
				if markErr := d.markFailure(ctx, item, err); markErr != nil {
					return delivered, markErr
				}
				continue
			}
		}
		if err := d.markDelivered(ctx, item.ID); err != nil {
			return delivered, err
		}
		delivered++
	}
	return delivered, nil
}

func (d *OutboxDispatcher) releaseAllowed(ctx context.Context, planned *task.Task) (bool, error) {
	if planned == nil {
		return false, nil
	}
	const query = `
SELECT EXISTS (
  SELECT 1 FROM device_tasks t
  JOIN parameter_sync_runs run ON run.id=t.source_id
  WHERE t.id=$1 AND t.source='param_sync' AND t.source_id=$2
    AND t.status='pending'
    AND run.status IN ('waiting_device','executing','processing')
)`
	var allowed bool
	if err := d.pool.QueryRow(ctx, query, planned.ID, planned.SourceID).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check parameter sync task release fence: %w", err)
	}
	return allowed, nil
}

func (d *OutboxDispatcher) claim(ctx context.Context, limit int) ([]outboxTask, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim parameter sync outbox: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	query, args, err := buildClaimOutboxSQL(d.now(), limit)
	if err != nil {
		return nil, fmt.Errorf("build claim parameter sync outbox: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query parameter sync outbox: %w", err)
	}
	defer rows.Close()
	var items []outboxTask
	for rows.Next() {
		var item outboxTask
		if err := rows.Scan(&item.ID, &item.EventType, &item.Payload, &item.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan parameter sync outbox: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate parameter sync outbox: %w", err)
	}
	if len(items) > 0 {
		ids := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		update, updateArgs, err := storage.Psql.Update("parameter_sync_outbox").
			Set("status", "delivering").Set("updated_at", d.now()).Where(sq.Eq{"id": ids}).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build mark parameter sync outbox delivering: %w", err)
		}
		if _, err := tx.Exec(ctx, update, updateArgs...); err != nil {
			return nil, fmt.Errorf("mark parameter sync outbox delivering: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim parameter sync outbox: %w", err)
	}
	return items, nil
}

func buildClaimOutboxSQL(now time.Time, limit int) (string, []interface{}, error) {
	// Keep the immutable status predicate literal. PostgreSQL cannot prove that
	// parameterized status values imply idx_parameter_sync_outbox_ready_created's
	// partial-index predicate when pgx selects a generic prepared plan. Under a
	// cold-start backlog that turns this LIMIT query into a full scan and an
	// external sort of every payload row.
	return storage.Psql.Select("id", "event_type", "payload", "attempt_count").
		From("parameter_sync_outbox").
		Where("status IN ('pending', 'failed')").
		Where(sq.LtOrEq{"next_attempt_at": now}).
		OrderBy("created_at ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
}

// EnsurePerformanceIndexes online-upgrades databases created from an older
// consolidated baseline. CREATE INDEX CONCURRENTLY avoids blocking the hot
// outbox writers while a large existing queue is being indexed.
func (d *OutboxDispatcher) EnsurePerformanceIndexes(ctx context.Context) (returnErr error) {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire parameter sync outbox index connection: %w", err)
	}
	locked := false
	lockStateUncertain := false
	defer func() {
		if !locked {
			if lockStateUncertain {
				// A cancelled lock call may have succeeded server-side before the
				// client observed the error. Close the session so a possible lock
				// cannot leak back into the pool.
				rawConn := conn.Hijack()
				closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = rawConn.Close(closeCtx)
				return
			}
			conn.Release()
			return
		}
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, parameterSyncOutboxIndexUnlockSQL).Scan(&unlocked); err == nil && unlocked {
			conn.Release()
			return
		} else if returnErr == nil {
			if err != nil {
				returnErr = fmt.Errorf("release parameter sync outbox index advisory lock: %w", err)
			} else {
				returnErr = errors.New("release parameter sync outbox index advisory lock: lock was not held")
			}
		}
		// Never return a session with a possibly-held advisory lock to the pool.
		rawConn := conn.Hijack()
		_ = rawConn.Close(unlockCtx)
	}()

	lockStateUncertain = true
	if _, err := conn.Exec(ctx, parameterSyncOutboxIndexLockSQL); err != nil {
		return fmt.Errorf("acquire parameter sync outbox index advisory lock: %w", err)
	}
	lockStateUncertain = false
	locked = true
	for _, index := range parameterSyncOutboxRuntimeIndexes {
		var exists, valid bool
		if err := conn.QueryRow(ctx, `
SELECT to_regclass($1) IS NOT NULL,
       COALESCE((SELECT indisvalid FROM pg_index WHERE indexrelid = to_regclass($1)), false)`,
			index.qualifiedName).Scan(&exists, &valid); err != nil {
			return fmt.Errorf("inspect parameter sync outbox index %s: %w", index.qualifiedName, err)
		}
		if valid {
			continue
		}
		if exists {
			if _, err := conn.Exec(ctx, index.dropSQL); err != nil {
				return fmt.Errorf("drop invalid parameter sync outbox index %s: %w", index.qualifiedName, err)
			}
		}
		if _, err := conn.Exec(ctx, index.createSQL); err != nil {
			return fmt.Errorf("create parameter sync outbox index %s: %w", index.qualifiedName, err)
		}
	}
	return nil
}

func (d *OutboxDispatcher) markDelivered(ctx context.Context, id uuid.UUID) error {
	now := d.now()
	query, args, err := storage.Psql.Update("parameter_sync_outbox").Set("status", "delivered").
		Set("delivered_at", now).Set("updated_at", now).Set("last_error", nil).
		Where(sq.Eq{"id": id, "status": "delivering"}).ToSql()
	if err != nil {
		return fmt.Errorf("build mark parameter sync outbox delivered: %w", err)
	}
	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark parameter sync outbox delivered: %w", err)
	}
	return nil
}

func (d *OutboxDispatcher) markFailure(ctx context.Context, item outboxTask, cause error) error {
	attempts := item.AttemptCount + 1
	status := "failed"
	if attempts >= d.maxAttempts {
		status = "dead"
	}
	now := d.now()
	query, args, err := storage.Psql.Update("parameter_sync_outbox").Set("status", status).
		Set("attempt_count", attempts).Set("next_attempt_at", now.Add(outboxBackoff(attempts))).
		Set("last_error", cause.Error()).Set("updated_at", now).
		Where(sq.Eq{"id": item.ID, "status": "delivering"}).ToSql()
	if err != nil {
		return fmt.Errorf("build mark parameter sync outbox failed: %w", err)
	}
	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark parameter sync outbox failed: %w", err)
	}
	return nil
}

func outboxBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 10 {
		attempt = 10
	}
	return time.Second * time.Duration(1<<uint(attempt-1))
}

func (d *OutboxDispatcher) RequeueStaleDeliveries(ctx context.Context, staleBefore time.Time) (int64, error) {
	query, args, err := buildRequeueStaleOutboxSQL(d.now(), staleBefore)
	if err != nil {
		return 0, fmt.Errorf("build requeue stale parameter sync outbox: %w", err)
	}
	tag, err := d.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("requeue stale parameter sync outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func buildRequeueStaleOutboxSQL(now, staleBefore time.Time) (string, []interface{}, error) {
	// Keep the status literal so PostgreSQL can use the partial
	// idx_parameter_sync_outbox_delivering_updated index with generic plans.
	return `UPDATE parameter_sync_outbox
SET status = 'failed', next_attempt_at = $1,
    last_error = 'stale delivery claim recovered', updated_at = $1
WHERE status = 'delivering' AND updated_at < $2`, []interface{}{now, staleBefore}, nil
}
