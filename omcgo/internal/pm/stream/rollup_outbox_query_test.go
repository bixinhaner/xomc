package stream

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

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
	joinedArgs := fmt.Sprint(args)
	if !strings.Contains(joinedArgs, first.String()) ||
		!strings.Contains(joinedArgs, second.String()) {
		t.Fatalf("period replay args lack source versions: %v", args)
	}
}
