package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
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
				"claim_token = $2",
				"claim_expires_at = $3",
				"next_attempt_at = '-infinity'::timestamptz",
				"last_error = $4",
				"published_at IS NOT NULL",
				"consumed_at IS NULL",
				"barrier_eligible = $5",
				"published_at < $6",
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

func TestOutboxClaimDueRequiresNoActiveLease(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	query, _, err := pendingOutboxSelect(
		"pm_aggregation_outbox", "event_id", "payload",
	).Where(outboxClaimDue(now, "")).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	for _, clause := range []string{
		"next_attempt_at <= ",
		"claim_token IS NULL",
		"claim_expires_at IS NULL",
		"claim_expires_at <= ",
	} {
		if !strings.Contains(query, clause) {
			t.Fatalf("claim SQL lacks %q: %s", clause, query)
		}
	}
}

func TestOutboxFinalizeRequiresCurrentUnexpiredClaim(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	for _, builder := range []struct {
		name  string
		query func() (string, []any, error)
	}{
		{
			name: "published",
			query: func() (string, []any, error) {
				return markOutboxPublishedUpdate(
					"pm_aggregation_outbox", uuid.New(), uuid.New(), now,
				).ToSql()
			},
		},
		{
			name: "failed",
			query: func() (string, []any, error) {
				return markOutboxFailedUpdate(
					"pm_aggregation_outbox", uuid.New(), uuid.New(),
					errors.New("nats unavailable"), now, time.Second,
				).ToSql()
			},
		},
	} {
		t.Run(builder.name, func(t *testing.T) {
			query, _, err := builder.query()
			if err != nil {
				t.Fatal(err)
			}
			for _, clause := range []string{
				"claim_token =",
				"claim_expires_at >",
				"published_at IS NULL",
				"consumed_at IS NULL",
			} {
				if !strings.Contains(query, clause) {
					t.Fatalf("finalize SQL lacks %q: %s", clause, query)
				}
			}
		})
	}
}

type outboxRelayOrderRepo struct {
	calls      []string
	claimToken uuid.UUID
	eventID    uuid.UUID
}

func (r *outboxRelayOrderRepo) ClaimBatch(
	context.Context,
	uint64,
	time.Time,
	time.Duration,
) ([]*OutboxRecord, error) {
	r.calls = append(r.calls, "claim")
	return []*OutboxRecord{{
		EventID:      r.eventID,
		SourceFileID: uuid.New(),
		ClaimToken:   r.claimToken,
	}}, nil
}

func (r *outboxRelayOrderRepo) MarkPublished(
	_ context.Context,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	_ time.Time,
) (bool, error) {
	r.calls = append(r.calls, "mark-published")
	return eventID == r.eventID && claimToken == r.claimToken, nil
}

func (r *outboxRelayOrderRepo) MarkFailed(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	error,
	time.Time,
	time.Duration,
) (bool, error) {
	r.calls = append(r.calls, "mark-failed")
	return true, nil
}

func (r *outboxRelayOrderRepo) RequeueStaleUnconsumed(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type rollupRelayOrderRepo struct {
	calls      []string
	claimToken uuid.UUID
	eventID    uuid.UUID
}

func (r *rollupRelayOrderRepo) ClaimBatch(
	context.Context,
	uint64,
	time.Time,
	time.Duration,
) ([]RollupOutboxRecord, error) {
	r.calls = append(r.calls, "claim")
	return []RollupOutboxRecord{{
		EventID:    r.eventID,
		Subject:    event.SubjectPMAggregationHourlyRollup,
		ClaimToken: r.claimToken,
		Payload: RollupPayload{
			SchemaVersion:     SchemaVersion,
			EventID:           r.eventID,
			TaskID:            uuid.New(),
			TaskVersionID:     uuid.New(),
			SourceGranularity: GranularityHourly,
			EntityKey:         "Network",
			WindowStart:       time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC),
			WindowEnd:         time.Date(2026, 8, 21, 11, 0, 0, 0, time.UTC),
		},
	}}, nil
}

func (r *rollupRelayOrderRepo) MarkPublished(
	_ context.Context,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	_ time.Time,
) (bool, error) {
	r.calls = append(r.calls, "mark-published")
	return eventID == r.eventID && claimToken == r.claimToken, nil
}

func (r *rollupRelayOrderRepo) MarkFailed(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	error,
	time.Time,
	time.Duration,
) (bool, error) {
	r.calls = append(r.calls, "mark-failed")
	return true, nil
}

func (r *rollupRelayOrderRepo) RequeueStaleUnconsumed(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type relayOrderBus struct {
	calls *[]string
}

func (b relayOrderBus) Publish(_ context.Context, _ string, _ event.Event) error {
	*b.calls = append(*b.calls, "publish")
	return nil
}

func (relayOrderBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (relayOrderBus) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (relayOrderBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (relayOrderBus) Close() error { return nil }

func TestOutboxRelayPublishesAfterClaimAndFinalizesByToken(t *testing.T) {
	repo := &outboxRelayOrderRepo{claimToken: uuid.New(), eventID: uuid.New()}
	relay := &OutboxRelay{
		repo:       repo,
		bus:        relayOrderBus{calls: &repo.calls},
		batch:      10,
		claimLease: time.Minute,
		retryAfter: time.Second,
	}
	published, err := relay.publishBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("relay did not publish claimed PM outbox row")
	}
	if got, want := strings.Join(repo.calls, ","), "claim,publish,mark-published"; got != want {
		t.Fatalf("relay order = %s, want %s", got, want)
	}
}

func TestRollupOutboxRelayPublishesAfterClaimAndFinalizesByToken(t *testing.T) {
	repo := &rollupRelayOrderRepo{claimToken: uuid.New(), eventID: uuid.New()}
	relay := &RollupOutboxRelay{
		repo:       repo,
		bus:        relayOrderBus{calls: &repo.calls},
		batch:      10,
		claimLease: time.Minute,
		retryAfter: time.Second,
	}
	published, err := relay.publishBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("relay did not publish claimed PM rollup outbox row")
	}
	if got, want := strings.Join(repo.calls, ","), "claim,publish,mark-published"; got != want {
		t.Fatalf("relay order = %s, want %s", got, want)
	}
}
