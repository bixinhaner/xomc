package stream

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/event"
)

func TestDevicePeriodReplayQueryFiltersBothDurableSourcesBeforeUnion(t *testing.T) {
	deviceID := uuid.MustParse("30f3d50f-e1c4-4c3a-89b1-ac9da9a51350")
	start := time.Date(2026, 8, 4, 21, 0, 0, 0, time.UTC)

	query, args, err := devicePeriodReplaySelect(
		deviceID, start, start.Add(time.Hour),
	).ToSql()
	if err != nil {
		t.Fatalf("build device-period replay query: %v", err)
	}
	normalized := strings.Join(strings.Fields(query), " ")
	if got := strings.Count(normalized, "device_id = $"); got != 2 {
		t.Fatalf("device-period replay query device predicates = %d, want 2: %s", got, query)
	}
	if got := strings.Count(normalized, "event_window_start >= $"); got != 2 {
		t.Fatalf("device-period replay query lower time predicates = %d, want 2: %s", got, query)
	}
	if got := strings.Count(normalized, "event_window_start < $"); got != 2 {
		t.Fatalf("device-period replay query upper time predicates = %d, want 2: %s", got, query)
	}
	if !strings.Contains(normalized, "ORDER BY event_window_start, event_id") {
		t.Fatalf("device-period replay query lacks stable ordering: %s", query)
	}
	for _, want := range []any{deviceID, start, start.Add(time.Hour)} {
		if !containsSQLArg(args, want) {
			t.Fatalf("device-period replay args %v missing %v", args, want)
		}
	}
}

type orderedReplayRows struct {
	rows            []orderedReplayRow
	index           int
	waitingVisit    bool
	contractBroke   bool
	scanDestCounts  []int
	scannedStarts   []time.Time
	scannedEventIDs []uuid.UUID
}

type orderedReplayRow struct {
	windowStart time.Time
	eventID     uuid.UUID
	payload     []byte
}

func (r *orderedReplayRows) Next() bool {
	if r.waitingVisit {
		r.contractBroke = true
		return false
	}
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *orderedReplayRows) Scan(dest ...any) error {
	r.scanDestCounts = append(r.scanDestCounts, len(dest))
	if len(dest) != 3 {
		return fmt.Errorf("replay row scan destination count = %d, want 3", len(dest))
	}
	row := r.rows[r.index-1]
	*(dest[0].(*time.Time)) = row.windowStart
	*(dest[1].(*uuid.UUID)) = row.eventID
	*(dest[2].(*[]byte)) = row.payload
	r.scannedStarts = append(r.scannedStarts, row.windowStart)
	r.scannedEventIDs = append(r.scannedEventIDs, row.eventID)
	r.waitingVisit = true
	return nil
}

func (r *orderedReplayRows) Close()                                       {}
func (r *orderedReplayRows) Err() error                                   { return nil }
func (r *orderedReplayRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *orderedReplayRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *orderedReplayRows) Values() ([]any, error)                       { return nil, nil }
func (r *orderedReplayRows) RawValues() [][]byte                          { return nil }
func (r *orderedReplayRows) Conn() *pgx.Conn                              { return nil }

func TestVisitReplayPayloadRowsDispatchesBeforeReadingNextRow(t *testing.T) {
	first := event.PMAggregationNormalizedPayload{
		EventID: uuid.New(), WindowStart: time.Now().UTC(),
	}
	second := first
	second.EventID = uuid.New()
	firstRaw, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	rows := &orderedReplayRows{rows: []orderedReplayRow{
		{windowStart: first.WindowStart, eventID: first.EventID, payload: firstRaw},
		{windowStart: second.WindowStart, eventID: second.EventID, payload: secondRaw},
	}}
	var visited []uuid.UUID

	err = visitReplayPayloadRows(rows, func(payload event.PMAggregationNormalizedPayload) error {
		visited = append(visited, payload.EventID)
		rows.waitingVisit = false
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if rows.contractBroke {
		t.Fatal("replay reader advanced before releasing the current decoded payload")
	}
	if len(visited) != 2 || visited[0] != first.EventID || visited[1] != second.EventID {
		t.Fatalf("visited payloads = %v, want [%s %s]", visited, first.EventID, second.EventID)
	}
	if len(rows.scanDestCounts) != 2 || rows.scanDestCounts[0] != 3 || rows.scanDestCounts[1] != 3 {
		t.Fatalf("scan destination counts = %v, want [3 3]", rows.scanDestCounts)
	}
	if len(rows.scannedEventIDs) != 2 || rows.scannedEventIDs[0] != first.EventID || rows.scannedEventIDs[1] != second.EventID {
		t.Fatalf("scanned event ids = %v, want [%s %s]", rows.scannedEventIDs, first.EventID, second.EventID)
	}
	if len(rows.scannedStarts) != 2 || !rows.scannedStarts[0].Equal(first.WindowStart) || !rows.scannedStarts[1].Equal(second.WindowStart) {
		t.Fatalf("scanned starts = %v, want [%s %s]", rows.scannedStarts, first.WindowStart, second.WindowStart)
	}
}
