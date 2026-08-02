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

type rollupIndexStateRow struct {
	exists bool
	valid  bool
}

func (row rollupIndexStateRow) Scan(dest ...any) error {
	*(dest[0].(*bool)) = row.exists
	*(dest[1].(*bool)) = row.valid
	return nil
}

type recordingRollupIndexDB struct {
	exists bool
	valid  bool
	execs  []string
}

func (db *recordingRollupIndexDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return rollupIndexStateRow{exists: db.exists, valid: db.valid}
}

func (db *recordingRollupIndexDB) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	db.execs = append(db.execs, query)
	return pgconn.NewCommandTag("CREATE INDEX"), nil
}

func TestEnsurePeriodRebuildIndexDropsInvalidArtifactBeforeCreate(t *testing.T) {
	db := &recordingRollupIndexDB{exists: true, valid: false}

	err := ensurePeriodRebuildIndex(context.Background(), db)

	if err != nil {
		t.Fatal(err)
	}
	if len(db.execs) != 2 || !strings.HasPrefix(db.execs[0], "DROP INDEX") ||
		!strings.Contains(db.execs[1], "CREATE INDEX") {
		t.Fatalf("index repair statements = %v, want DROP invalid then CREATE", db.execs)
	}
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
	query, args, err := counterRollupPeriodPageSelect(
		[]uuid.UUID{first, second},
		GranularityHourly,
		time.Date(2026, 7, 29, 19, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC),
		nil,
		256,
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "task_version_id IN") {
		t.Fatalf("period replay query lacks source-version predicate: %s", query)
	}
	if strings.Contains(query, "publication_eligible") {
		t.Fatalf("period replay query must retain the captured revision after eligibility flips: %s", query)
	}
	if !strings.Contains(query, "FROM pm_rebuild_source_snapshot rollup") {
		t.Fatalf("period replay query does not page the materialized source rows: %s", query)
	}
	if !strings.Contains(query, "LIMIT 256") {
		t.Fatalf("period replay query is not bounded: %s", query)
	}
	if !strings.Contains(query, "rollup.event_id") {
		t.Fatalf("period replay query lacks a unique keyset tie-breaker: %s", query)
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
	if !strings.Contains(periodRebuildIndexSQL, "CREATE INDEX CONCURRENTLY") ||
		!strings.Contains(periodRebuildIndexDropSQL, "DROP INDEX CONCURRENTLY") {
		t.Fatalf("period rebuild index maintenance must not block hot writers: create=%s drop=%s",
			periodRebuildIndexSQL, periodRebuildIndexDropSQL)
	}
	if !strings.Contains(periodRebuildIndexLockSQL, "pg_advisory_lock") ||
		!strings.Contains(periodRebuildIndexUnlockSQL, "pg_advisory_unlock") {
		t.Fatal("period rebuild index maintenance must serialize worker replicas")
	}
	if !strings.Contains(periodRebuildIndexSQL, "event_id") {
		t.Fatalf("period rebuild index must cover the unique page cursor: %s", periodRebuildIndexSQL)
	}
}

func TestRevisionCleanupIndexesMatchDeletePredicate(t *testing.T) {
	wantColumns := "publication_task_version_id, entity_key, granularity, window_start, revision, event_id"
	for name, query := range map[string]string{
		"counter rollups": revisionCleanupCounterRollupsIndexSQL,
		"rollup outbox":   revisionCleanupOutboxIndexSQL,
	} {
		normalized := strings.Join(strings.Fields(query), " ")
		if !strings.Contains(normalized, wantColumns) {
			t.Fatalf("%s cleanup index does not match DELETE predicate: %s", name, query)
		}
		if !strings.Contains(query, "CREATE INDEX CONCURRENTLY") {
			t.Fatalf("%s cleanup index can block hot writers: %s", name, query)
		}
	}
	if !strings.Contains(revisionCleanupIndexLockSQL, "pg_advisory_lock") ||
		!strings.Contains(revisionCleanupIndexUnlockSQL, "pg_advisory_unlock") {
		t.Fatal("revision cleanup online index maintenance must serialize worker replicas")
	}
}

func TestEnsureRevisionCleanupIndexesRepairsBothInvalidArtifacts(t *testing.T) {
	db := &recordingRollupIndexDB{exists: true, valid: false}

	err := ensureRevisionCleanupIndexes(context.Background(), db)

	if err != nil {
		t.Fatal(err)
	}
	if len(db.execs) != 4 {
		t.Fatalf("revision cleanup index repair statements = %d, want DROP+CREATE for two indexes: %v", len(db.execs), db.execs)
	}
	for i := 0; i < len(db.execs); i += 2 {
		if !strings.HasPrefix(db.execs[i], "DROP INDEX CONCURRENTLY") ||
			!strings.Contains(db.execs[i+1], "CREATE INDEX CONCURRENTLY") {
			t.Fatalf("revision cleanup repair pair %d is not online DROP+CREATE: %v", i/2, db.execs[i:i+2])
		}
	}
}

func TestPeriodRebuildSnapshotMapsSourceVersionToPublicationVersion(t *testing.T) {
	query := strings.Join(strings.Fields(periodRebuildSnapshotSQL), " ")
	if !strings.Contains(query, "rollup.task_version_id = ANY($1)") {
		t.Fatalf("snapshot must filter the source rollup version IDs: %s", query)
	}
	if !strings.Contains(query,
		"published_window.task_version_id = rollup.publication_task_version_id") {
		t.Fatalf("snapshot must map source rollup versions to publication window versions: %s", query)
	}
	if !strings.Contains(query, "published_window.published_revision = rollup.revision") {
		t.Fatalf("snapshot must materialize only the currently published revision: %s", query)
	}
	if strings.Contains(query, "published_window.task_version_id = ANY($1)") {
		t.Fatalf("snapshot incorrectly treats source IDs as publication window IDs: %s", query)
	}
	for _, column := range []string{"rollup.event_id", "rollup.payload", "rollup.revision"} {
		if !strings.Contains(query, column) {
			t.Fatalf("snapshot must materialize %s so retention cannot remove a later page: %s",
				column, query)
		}
	}
}

func TestVisitRollupSnapshotPagesClosesEachBoundedPageBeforeContinuing(t *testing.T) {
	first := rollupSnapshotPageRow{
		cursor:  rollupSnapshotCursor{EventID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		payload: RollupPayload{EventID: uuid.MustParse("10000000-0000-0000-0000-000000000001")},
	}
	second := rollupSnapshotPageRow{
		cursor:  rollupSnapshotCursor{EventID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
		payload: RollupPayload{EventID: uuid.MustParse("10000000-0000-0000-0000-000000000002")},
	}
	third := rollupSnapshotPageRow{
		cursor:  rollupSnapshotCursor{EventID: uuid.MustParse("00000000-0000-0000-0000-000000000003")},
		payload: RollupPayload{EventID: uuid.MustParse("10000000-0000-0000-0000-000000000003")},
	}
	var fetchCursors []*rollupSnapshotCursor
	fetch := func(cursor *rollupSnapshotCursor) ([]rollupSnapshotPageRow, error) {
		if cursor == nil {
			fetchCursors = append(fetchCursors, nil)
			return []rollupSnapshotPageRow{first, second}, nil
		}
		copyOfCursor := *cursor
		fetchCursors = append(fetchCursors, &copyOfCursor)
		if cursor.EventID == second.cursor.EventID {
			return []rollupSnapshotPageRow{third}, nil
		}
		return nil, nil
	}
	var visited []uuid.UUID

	err := visitRollupSnapshotPages(fetch, func(payload RollupPayload) error {
		visited = append(visited, payload.EventID)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(fetchCursors) != 3 || fetchCursors[0] != nil ||
		fetchCursors[1].EventID != second.cursor.EventID ||
		fetchCursors[2].EventID != third.cursor.EventID {
		t.Fatalf("page cursors = %#v, want nil -> second -> third", fetchCursors)
	}
	wantVisited := []uuid.UUID{first.payload.EventID, second.payload.EventID, third.payload.EventID}
	if fmt.Sprint(visited) != fmt.Sprint(wantVisited) {
		t.Fatalf("visited = %v, want %v", visited, wantVisited)
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

func TestDeleteStaleRollupRevisionSkipsInitialPublication(t *testing.T) {
	tx := &recordingRollupDeleteTx{}
	key := WindowKey{
		TaskVersionID: uuid.New(), EntityKey: "Network",
		Granularity: GranularityHourly,
		Start:       time.Date(2026, 8, 2, 14, 0, 0, 0, time.UTC),
	}

	if err := deleteStaleRollupRevisionTx(
		context.Background(), tx, key, 1, []uuid.UUID{uuid.New()},
	); err != nil {
		t.Fatal(err)
	}
	if len(tx.queries) != 0 {
		t.Fatalf("initial publication executed %d stale DELETE statements, want 0: %v", len(tx.queries), tx.queries)
	}
}
