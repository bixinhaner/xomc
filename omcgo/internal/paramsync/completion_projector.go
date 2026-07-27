package paramsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
)

// FullRunProjection refreshes derived device projections after the official
// parameter merge has committed. Implementations must be idempotent because
// the terminal event is delivered at least once.
type FullRunProjection interface {
	Refresh(ctx context.Context, deviceID uuid.UUID) error
}

type CompletionProjector struct {
	pool       *pgxpool.Pool
	bus        event.EventBus
	projection FullRunProjection
	sub        event.Subscription
	now        func() time.Time
}

func NewCompletionProjector(pool *pgxpool.Pool, bus event.EventBus, projection FullRunProjection) *CompletionProjector {
	return &CompletionProjector{pool: pool, bus: bus, projection: projection, now: func() time.Time { return time.Now().UTC() }}
}

func (p *CompletionProjector) Start() error {
	if p.pool == nil || p.bus == nil || p.projection == nil {
		return fmt.Errorf("parameter sync completion projector dependencies are required")
	}
	sub, err := p.bus.QueueSubscribe(event.SubjectParamSyncRunCompleted, "param-sync-device-projection", p.handleCompleted)
	if err != nil {
		return fmt.Errorf("subscribe parameter sync completion projection: %w", err)
	}
	p.sub = sub
	return nil
}

func (p *CompletionProjector) handleCompleted(ctx context.Context, evt event.Event) error {
	var payload struct {
		RunID uuid.UUID `json:"run_id"`
	}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil || payload.RunID == uuid.Nil {
		return fmt.Errorf("decode parameter sync completion projection: %w", err)
	}
	return p.projectRun(ctx, payload.RunID)
}

func (p *CompletionProjector) projectRun(ctx context.Context, runID uuid.UUID) error {
	_, err := p.projectRunOnce(ctx, runID)
	return err
}

const parameterSyncProjectionLease = 5 * time.Minute

func (p *CompletionProjector) projectRunOnce(ctx context.Context, runID uuid.UUID) (bool, error) {
	now := p.now()
	leaseToken := uuid.New()
	var deviceID uuid.UUID
	var attempts int
	claim := `UPDATE parameter_sync_runs SET projection_status='processing',
projection_attempts=projection_attempts+1, projection_error=NULL,
projection_lease_token=$2, projection_lease_until=$3
WHERE id=$1 AND status='succeeded' AND sync_scope='full' AND (
  (projection_status IN ('pending','failed') AND projection_next_attempt_at <= $4)
  OR (projection_status='processing' AND (projection_lease_until IS NULL OR projection_lease_until <= $4))
)
RETURNING device_id, projection_attempts`
	if err := p.pool.QueryRow(ctx, claim, runID, leaseToken, now.Add(parameterSyncProjectionLease), now).
		Scan(&deviceID, &attempts); err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("claim completed parameter sync projection: %w", err)
	}
	if err := p.projection.Refresh(ctx, deviceID); err != nil {
		nextAttempt := p.now().Add(projectionRetryBackoff(attempts))
		_, updateErr := p.pool.Exec(ctx, `UPDATE parameter_sync_runs SET projection_status='failed',
projection_error=$3, projection_lease_token=NULL, projection_lease_until=NULL,
projection_next_attempt_at=$4 WHERE id=$1 AND projection_status='processing' AND projection_lease_token=$2`,
			runID, leaseToken, err.Error(), nextAttempt)
		return false, errors.Join(
			fmt.Errorf("refresh completed parameter sync projections: %w", err),
			wrapProjectionUpdateError("record failed parameter sync projection", updateErr),
		)
	}
	tag, err := p.pool.Exec(ctx, `UPDATE parameter_sync_runs SET projection_status='completed',
projection_error=NULL, projection_completed_at=$3, projection_lease_token=NULL,
projection_lease_until=NULL WHERE id=$1 AND projection_status='processing' AND projection_lease_token=$2`,
		runID, leaseToken, p.now())
	if err != nil {
		return false, fmt.Errorf("complete parameter sync projection feedback: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return false, fmt.Errorf("complete parameter sync projection feedback: lease lost")
	}
	return true, nil
}

func wrapProjectionUpdateError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func projectionRetryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 7 {
		attempt = 7
	}
	backoff := 30 * time.Second * time.Duration(1<<uint(attempt-1))
	if backoff > 30*time.Minute {
		return 30 * time.Minute
	}
	return backoff
}

// ReconcilePending closes the database feedback loop when the terminal event
// exhausts its delivery attempts or APP stops after the full merge commits.
func (p *CompletionProjector) ReconcilePending(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := p.pool.Query(ctx, `SELECT id FROM parameter_sync_runs
WHERE status='succeeded' AND sync_scope='full' AND (
  (projection_status IN ('pending','failed') AND projection_next_attempt_at <= $2)
  OR (projection_status='processing' AND (projection_lease_until IS NULL OR projection_lease_until <= $2))
)
ORDER BY projection_next_attempt_at, completed_at NULLS LAST LIMIT $1`, limit, p.now())
	if err != nil {
		return 0, fmt.Errorf("list pending parameter sync projections: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan pending parameter sync projection: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate pending parameter sync projections: %w", err)
	}
	rows.Close()
	completed := 0
	var errs []error
	for _, id := range ids {
		projected, err := p.projectRunOnce(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("project parameter sync run %s: %w", id, err))
			continue
		}
		if projected {
			completed++
		}
	}
	return completed, errors.Join(errs...)
}

func (p *CompletionProjector) Stop() error {
	if p.sub == nil {
		return nil
	}
	return p.sub.Unsubscribe()
}
