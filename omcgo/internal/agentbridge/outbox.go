package agentbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository { return &OutboxRepository{pool: pool} }

func (r *OutboxRepository) Enqueue(ctx context.Context, envelope ConnectorEventEnvelope) error {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal agent event outbox payload: %w", err)
	}
	query, args, err := psql.Insert("agent_bridge_event_outbox").
		Columns("event_id", "event_type", "payload").
		Values(envelope.EventID, envelope.EventType, raw).
		Suffix("ON CONFLICT (event_id) DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build agent event outbox insert: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert agent event outbox: %w", err)
	}
	return nil
}

func (r *OutboxRepository) Claim(ctx context.Context, limit int) ([]OutboxItem, error) {
	if limit < 1 {
		limit = 20
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin agent outbox claim: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	query, args, err := psql.Select("id", "event_id", "payload", "attempt").
		From("agent_bridge_event_outbox").
		Where(sq.Eq{"status": "pending"}).
		Where(sq.LtOrEq{"next_attempt_at": time.Now().UTC()}).
		OrderBy("created_at ASC").Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build agent outbox claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query agent outbox claim: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (OutboxItem, error) {
		var item OutboxItem
		err := row.Scan(&item.ID, &item.EventID, &item.Payload, &item.Attempt)
		return item, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan agent outbox claim: %w", err)
	}
	// Move claimed rows' retry horizon while the lock is held. This is a
	// lightweight recoverable lease: another process cannot redeliver the same
	// row immediately, while a crashed worker makes it eligible again shortly.
	leaseUntil := time.Now().UTC().Add(time.Minute)
	for _, item := range items {
		updateSQL, updateArgs, buildErr := psql.Update("agent_bridge_event_outbox").
			Set("next_attempt_at", leaseUntil).Where(sq.Eq{"id": item.ID, "status": "pending"}).ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build agent outbox claim lease: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, updateSQL, updateArgs...); execErr != nil {
			return nil, fmt.Errorf("lease agent outbox row: %w", execErr)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit agent outbox claim: %w", err)
	}
	return items, nil
}

func (r *OutboxRepository) MarkDelivered(ctx context.Context, id string) error {
	query, args, err := psql.Update("agent_bridge_event_outbox").
		Set("status", "delivered").Set("delivered_at", time.Now().UTC()).Set("last_error", nil).
		Where(sq.Eq{"id": id, "status": "pending"}).ToSql()
	if err != nil {
		return fmt.Errorf("build agent outbox delivered update: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark agent outbox delivered: %w", err)
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id string, attempt int, cause error) error {
	nextAttempt := attempt + 1
	status := "pending"
	if nextAttempt >= 12 {
		status = "dead"
	}
	delay := time.Duration(1<<min(nextAttempt, 8)) * time.Second
	query, args, err := psql.Update("agent_bridge_event_outbox").
		Set("status", status).Set("attempt", nextAttempt).
		Set("next_attempt_at", time.Now().UTC().Add(delay)).Set("last_error", cause.Error()).
		Where(sq.Eq{"id": id, "status": "pending"}).ToSql()
	if err != nil {
		return fmt.Errorf("build agent outbox failure update: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark agent outbox failed: %w", err)
	}
	return nil
}
