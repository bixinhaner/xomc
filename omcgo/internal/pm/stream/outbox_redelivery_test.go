package stream

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStaleBarrierRedeliveryUpdateRequiresPublishedUnconsumedBarrier(t *testing.T) {
	before := time.Date(2026, 7, 29, 14, 0, 0, 0, time.UTC)

	for _, table := range []string{
		"pm_aggregation_outbox",
		"pm_aggregation_rollup_outbox",
	} {
		t.Run(table, func(t *testing.T) {
			query, args, err := staleBarrierRedeliveryUpdate(table, before).ToSql()
			if err != nil {
				t.Fatal(err)
			}
			for _, clause := range []string{
				"published_at = $1",
				"last_error = $2",
				"published_at IS NOT NULL",
				"consumed_at IS NULL",
				"barrier_eligible = $3",
				"published_at < $4",
			} {
				if !strings.Contains(query, clause) {
					t.Fatalf("redelivery SQL lacks %q: %s", clause, query)
				}
			}
			if !strings.Contains(query, "UPDATE "+table) {
				t.Fatalf("redelivery SQL targets wrong table: %s", query)
			}
			if got := fmt.Sprint(args); !strings.Contains(got, before.String()) {
				t.Fatalf("redelivery SQL lacks cutoff argument: %v", args)
			}
		})
	}
}

func TestRunRedeliveryLoopProgressesIndependentlyOfPublishBacklog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		runRedeliveryLoop(ctx, time.Millisecond, func() {
			if calls.Add(1) >= 2 {
				cancel()
			}
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("redelivery loop was starved while the publish path remained busy")
	}
	if got := calls.Load(); got < 2 {
		t.Fatalf("redelivery calls = %d, want at least 2", got)
	}
}

func TestPendingOutboxSelectExcludesAlreadyConsumedRows(t *testing.T) {
	query, _, err := pendingOutboxSelect(
		"pm_aggregation_outbox", "event_id", "payload",
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	for _, clause := range []string{
		"published_at IS NULL",
		"consumed_at IS NULL",
	} {
		if !strings.Contains(query, clause) {
			t.Fatalf("pending outbox SQL lacks %q: %s", clause, query)
		}
	}
}
