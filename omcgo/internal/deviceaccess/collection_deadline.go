package deviceaccess

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// CollectionDeadlineScheduler converts expired collecting projections into
// durable reevaluation requests. The request key includes the decision version,
// so repeated scans and multiple workers remain idempotent.
type CollectionDeadlineScheduler struct {
	db    storage.DB
	queue ReevaluationQueue
	now   func() time.Time
}

func NewCollectionDeadlineScheduler(pool *pgxpool.Pool, queue ReevaluationQueue) *CollectionDeadlineScheduler {
	return newCollectionDeadlineSchedulerWithDB(storage.NewPoolDB(pool), queue)
}

func newCollectionDeadlineSchedulerWithDB(db storage.DB, queue ReevaluationQueue) *CollectionDeadlineScheduler {
	return &CollectionDeadlineScheduler{db: db, queue: queue, now: func() time.Time { return time.Now().UTC() }}
}

func (s *CollectionDeadlineScheduler) RunOnce(ctx context.Context, limit int) (int, error) {
	if s == nil || s.db == nil || s.queue == nil {
		return 0, fmt.Errorf("schedule expired device access collections: %w", ErrAccessGateDependencyMissing)
	}
	if limit <= 0 {
		limit = 100
	}
	now := s.now().UTC()
	query, args, err := storage.Psql.
		Select("id", "carrier", "serial_number", "decision_version").
		From("device_access_states").
		Where(sq.Eq{"state": AccessStateCollecting}).
		Where(sq.LtOrEq{"decision_expires_at": now}).
		OrderBy("decision_expires_at ASC", "id ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build expired device access collection query: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("query expired device access collections: %w", err)
	}
	defer rows.Close()
	type expiredCollection struct {
		stateID uuid.UUID
		carrier string
		serial  string
		version int64
	}
	items := make([]expiredCollection, 0, limit)
	for rows.Next() {
		var item expiredCollection
		if err := rows.Scan(&item.stateID, &item.carrier, &item.serial, &item.version); err != nil {
			return 0, fmt.Errorf("scan expired device access collection: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate expired device access collections: %w", err)
	}
	queued := 0
	for _, item := range items {
		triggerID := fmt.Sprintf("collection-deadline:%s:v%d", item.stateID, item.version)
		if err := s.queue.Enqueue(ctx, ReevaluationRequest{
			Carrier: item.carrier, SerialNumber: item.serial,
			TriggerType: TriggerCollectionDeadline, TriggerEventID: triggerID,
		}); err != nil {
			return queued, fmt.Errorf("enqueue expired collection %s/%s: %w", item.carrier, item.serial, err)
		}
		queued++
	}
	return queued, nil
}
