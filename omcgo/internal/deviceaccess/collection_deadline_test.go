package deviceaccess

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type collectionDeadlineRows struct {
	pgx.Rows
	items []struct {
		id      uuid.UUID
		carrier string
		serial  string
		version int64
	}
	index int
}

func (r *collectionDeadlineRows) Close()     {}
func (r *collectionDeadlineRows) Err() error { return nil }
func (r *collectionDeadlineRows) Next() bool {
	return r.index < len(r.items)
}
func (r *collectionDeadlineRows) Scan(dest ...any) error {
	item := r.items[r.index]
	r.index++
	*dest[0].(*uuid.UUID) = item.id
	*dest[1].(*string) = item.carrier
	*dest[2].(*string) = item.serial
	*dest[3].(*int64) = item.version
	return nil
}

type collectionDeadlineDB struct {
	repositoryTestDB
	rows    pgx.Rows
	lastSQL string
}

func (d *collectionDeadlineDB) Query(_ context.Context, query string, _ ...any) (pgx.Rows, error) {
	d.lastSQL = query
	return d.rows, nil
}

func TestCollectionDeadlineSchedulerQueuesIdempotentTerminalReevaluation(t *testing.T) {
	stateID := uuid.New()
	rows := &collectionDeadlineRows{items: []struct {
		id      uuid.UUID
		carrier string
		serial  string
		version int64
	}{{id: stateID, carrier: "cmcc", serial: "SN-DEADLINE", version: 7}}}
	db := &collectionDeadlineDB{rows: rows}
	queue := &reevaluationQueueStub{}
	scheduler := newCollectionDeadlineSchedulerWithDB(db, queue)
	scheduler.now = func() time.Time { return time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC) }

	queued, err := scheduler.RunOnce(context.Background(), 20)

	require.NoError(t, err)
	require.Equal(t, 1, queued)
	require.Contains(t, db.lastSQL, "decision_expires_at")
	require.Len(t, queue.requests, 1)
	require.Equal(t, TriggerCollectionDeadline, queue.requests[0].TriggerType)
	require.Equal(t, "collection-deadline:"+stateID.String()+":v7", queue.requests[0].TriggerEventID)
}
