package rawcleanup

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCandidateQueriesSplitNewAndRetryReadyPopulations(t *testing.T) {
	queries, err := buildCandidateQueries("pm_files", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 {
		t.Fatalf("queries=%d want 2", len(queries))
	}
	if strings.Contains(queries[0].sql, " OR ") || strings.Contains(queries[1].sql, " OR ") {
		t.Fatalf("retry populations were combined: %#v", queries)
	}
	if !strings.Contains(queries[0].sql, "raw_delete_next_attempt_at IS NULL") ||
		!strings.Contains(queries[0].sql, "ORDER BY collect_time ASC, id ASC LIMIT 100") {
		t.Fatalf("new candidate query does not match partial index: %s", queries[0].sql)
	}
	if !strings.Contains(queries[1].sql, "raw_delete_next_attempt_at <= NOW()") ||
		!strings.Contains(queries[1].sql, "ORDER BY raw_delete_next_attempt_at ASC, collect_time ASC, id ASC LIMIT 100") {
		t.Fatalf("retry query does not match retry index: %s", queries[1].sql)
	}
}

func TestPMOutboxCleanupPreservesRebuildReplayHorizon(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	query, args, err := buildPMOutboxCleanupQuery([]uuid.UUID{uuid.New()}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "created_at <") {
		t.Fatalf("cleanup query has no durable replay retention boundary: %s", query)
	}
	want := now.Add(-45 * 24 * time.Hour)
	found := false
	for _, arg := range args {
		if value, ok := arg.(time.Time); ok && value.Equal(want) {
			found = true
		}
	}
	if !found {
		t.Fatalf("cleanup query args do not contain 45-day replay cutoff: %#v", args)
	}
}

func TestRetryDelayDoublesAndCapsAtOneHour(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, time.Minute},
		{1, 2 * time.Minute},
		{5, 32 * time.Minute},
		{6, time.Hour},
		{20, time.Hour},
	}
	for _, tt := range tests {
		if got := RetryDelay(tt.attempt); got != tt.want {
			t.Fatalf("RetryDelay(%d)=%v want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestTruncateErrorBoundsDatabasePayload(t *testing.T) {
	in := make([]byte, 800)
	for i := range in {
		in[i] = 'x'
	}
	if got := TruncateError(string(in)); len(got) != 512 {
		t.Fatalf("error length=%d want 512", len(got))
	}
}
