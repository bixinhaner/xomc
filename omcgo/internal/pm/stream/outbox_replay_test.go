package stream

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/event"
)

type orderedReplayRows struct {
	rows          [][]byte
	index         int
	waitingVisit  bool
	contractBroke bool
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
	*(dest[0].(*[]byte)) = r.rows[r.index-1]
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
	rows := &orderedReplayRows{rows: [][]byte{firstRaw, secondRaw}}
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
}
