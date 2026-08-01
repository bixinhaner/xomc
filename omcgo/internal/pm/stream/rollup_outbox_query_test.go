package stream

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type recordingRollupDeleteTx struct {
	pgx.Tx
	queries []string
}

func (tx *recordingRollupDeleteTx) Exec(
	_ context.Context,
	query string,
	_ ...any,
) (pgconn.CommandTag, error) {
	tx.queries = append(tx.queries, query)
	return pgconn.NewCommandTag("DELETE 1"), nil
}

type countingRebuildSnapshotSource struct {
	calls    int
	payloads []RollupPayload
}

func (source *countingRebuildSnapshotSource) VisitSnapshotsForPeriod(
	_ context.Context,
	_ []uuid.UUID,
	_ Granularity,
	_, _ time.Time,
	visit func(RollupPayload) error,
) error {
	source.calls++
	for _, payload := range source.payloads {
		if err := visit(payload); err != nil {
			return err
		}
	}
	return nil
}

func TestVisitSnapshotsForRebuildBatchScansSharedPeriodOnce(t *testing.T) {
	versionID := uuid.New()
	start := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	source := &countingRebuildSnapshotSource{payloads: []RollupPayload{
		{EventID: uuid.New()},
		{EventID: uuid.New()},
	}}
	dispatched := 0

	rows, err := visitSnapshotsForRebuildBatch(
		context.Background(),
		source,
		[]uuid.UUID{versionID},
		GranularityHourly,
		start,
		start.Add(24*time.Hour),
		func(_ RollupPayload) error {
			dispatched++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 1 {
		t.Fatalf("shared rebuild period scans = %d, want 1", source.calls)
	}
	if rows != 2 {
		t.Fatalf("snapshot rows = %d, want 2", rows)
	}
	if dispatched != 2 {
		t.Fatalf("payload dispatches = %d, want 2", dispatched)
	}
}

func TestCounterRollupPeriodSelectRestrictsSourceVersions(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	query, args, err := counterRollupPeriodSelect(
		[]uuid.UUID{first, second},
		GranularityHourly,
		time.Date(2026, 7, 29, 19, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC),
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "task_version_id IN") {
		t.Fatalf("period replay query lacks source-version predicate: %s", query)
	}
	if !strings.Contains(query, "publication_eligible =") {
		t.Fatalf("period replay query can read prepared snapshots: %s", query)
	}
	if !strings.Contains(query, "published_window.published_revision = rollup.revision") ||
		!strings.Contains(query, "published_window.entity_key = rollup.entity_key") {
		t.Fatalf("period replay query is not pinned to the entity's published revision: %s", query)
	}
	joinedArgs := fmt.Sprint(args)
	if !strings.Contains(joinedArgs, first.String()) ||
		!strings.Contains(joinedArgs, second.String()) {
		t.Fatalf("period replay args lack source versions: %v", args)
	}
}

func TestPeriodRebuildIndexMatchesPeriodQueryPrefix(t *testing.T) {
	want := "task_version_id, granularity, window_start"
	if !strings.Contains(strings.Join(strings.Fields(periodRebuildIndexSQL), " "), want) {
		t.Fatalf("period rebuild index does not lead with %q: %s", want, periodRebuildIndexSQL)
	}
	if !strings.Contains(periodRebuildIndexSQL, "WHERE publication_eligible") {
		t.Fatalf("period rebuild index must match the eligible snapshot predicate: %s", periodRebuildIndexSQL)
	}
}

func TestPendingRollupOutboxRequiresPublishedRevisionEligibility(t *testing.T) {
	query, _, err := pendingRollupOutboxSelect("event_id", "subject", "payload").ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "barrier_eligible =") {
		t.Fatalf("pending rollup query can expose prepared revision: %s", query)
	}
	if !strings.Contains(query, "published_window.published_revision = rollup.revision") ||
		!strings.Contains(query, "published_window.entity_key = rollup.entity_key") {
		t.Fatalf("pending rollup query is not pinned to the entity's published revision: %s", query)
	}
}

func TestDeleteStaleRollupRevisionWithEmptyReplacementDeletesAllRows(t *testing.T) {
	tx := &recordingRollupDeleteTx{}
	key := WindowKey{
		TaskVersionID: uuid.New(), EntityKey: "Network",
		Granularity: GranularityDaily,
		Start:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := deleteStaleRollupRevisionTx(
		context.Background(), tx, key, 2, nil,
	); err != nil {
		t.Fatal(err)
	}
	if len(tx.queries) != 2 {
		t.Fatalf("empty replacement delete statements = %d, want snapshot and outbox", len(tx.queries))
	}
	for _, query := range tx.queries {
		if strings.Contains(strings.ToUpper(query), "EVENT_ID <>") ||
			strings.Contains(strings.ToUpper(query), "EVENT_ID NOT IN") {
			t.Fatalf("empty replacement unexpectedly preserves old rollup rows: %s", query)
		}
	}
}
