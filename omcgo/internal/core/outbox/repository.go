package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Repository persists generic business events in the caller's transaction.
type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// InsertTx inserts an event using the same transaction as its aggregate
// mutation. A duplicate dedupe key is an idempotent success.
func (r *Repository) InsertTx(ctx context.Context, tx pgx.Tx, record Record) error {
	return r.insert(ctx, tx, record)
}

func (r *Repository) insert(ctx context.Context, exec executor, record Record) error {
	switch {
	case record.AggregateType == "":
		return fmt.Errorf("insert event outbox: aggregate type is required")
	case record.AggregateID == "":
		return fmt.Errorf("insert event outbox: aggregate id is required")
	case record.Subject == "":
		return fmt.Errorf("insert event outbox: subject is required")
	case len(record.Payload) == 0:
		return fmt.Errorf("insert event outbox: payload is required")
	case record.DedupeKey == "":
		return fmt.Errorf("insert event outbox: dedupe key is required")
	}

	query, args, err := storage.Psql.Insert("event_outbox").
		Columns("aggregate_type", "aggregate_id", "subject", "payload", "dedupe_key").
		Values(record.AggregateType, record.AggregateID, record.Subject, record.Payload, record.DedupeKey).
		Suffix("ON CONFLICT (dedupe_key) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build event outbox insert: %w", err)
	}
	if _, err := exec.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert event outbox: %w", err)
	}
	return nil
}
