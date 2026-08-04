package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
)

type alarmOutboxRepoStub struct {
	mu sync.Mutex

	records        []OutboxRecord
	claimRequests  []OutboxClaimRequest
	published      []uuid.UUID
	failed         []OutboxFailure
	replayed       []uuid.UUID
	cleanupBefore  []time.Time
	stats          OutboxStats
	claimErr       error
	markPublishErr error
}

func (s *alarmOutboxRepoStub) Claim(_ context.Context, request OutboxClaimRequest) ([]OutboxRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claimRequests = append(s.claimRequests, request)
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	records := append([]OutboxRecord(nil), s.records...)
	s.records = nil
	return records, nil
}

func (s *alarmOutboxRepoStub) MarkPublished(_ context.Context, eventID uuid.UUID, _ string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published = append(s.published, eventID)
	return s.markPublishErr
}

func (s *alarmOutboxRepoStub) MarkFailed(_ context.Context, failure OutboxFailure) (OutboxStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed = append(s.failed, failure)
	if failure.AttemptCount >= failure.MaxAttempts {
		return OutboxStatusDead, nil
	}
	return OutboxStatusFailed, nil
}

func (s *alarmOutboxRepoStub) Replay(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replayed = append(s.replayed, eventID)
	return nil
}

func (s *alarmOutboxRepoStub) DeletePublishedBefore(_ context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupBefore = append(s.cleanupBefore, before)
	return 0, nil
}

func (s *alarmOutboxRepoStub) Stats(context.Context, time.Time) (OutboxStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats, nil
}

type alarmOutboxBusStub struct {
	mu       sync.Mutex
	subjects []string
	events   []event.Event
	err      error
}

func (s *alarmOutboxBusStub) Publish(_ context.Context, subject string, envelope event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subjects = append(s.subjects, subject)
	s.events = append(s.events, envelope)
	return s.err
}

func (*alarmOutboxBusStub) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (*alarmOutboxBusStub) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (*alarmOutboxBusStub) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (*alarmOutboxBusStub) Close() error { return nil }

func TestAlarmOutboxClaimSQLUsesDueRowsAndSkipLockedLease(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 3, 4, 0, time.UTC)
	query, args, err := buildAlarmOutboxClaimSelect(now, now.Add(-time.Minute), 25)
	require.NoError(t, err)

	normalized := strings.Join(strings.Fields(query), " ")
	require.Contains(t, normalized, "FOR UPDATE SKIP LOCKED")
	require.Contains(t, normalized, "status IN ($1,$2)")
	require.Contains(t, normalized, "next_attempt_at <= $3")
	require.Contains(t, normalized, "status = $4")
	require.Contains(t, normalized, "locked_at < $5")
	require.Contains(t, normalized, "LIMIT 25")
	require.Equal(t, []any{
		OutboxStatusPending, OutboxStatusFailed, now,
		OutboxStatusPublishing, now.Add(-time.Minute),
	}, args)
}

func TestAlarmOutboxTerminalSQLRequiresLeaseAndPublishedState(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 3, 4, 0, time.UTC)
	eventID := uuid.New()

	publishSQL, publishArgs, err := buildMarkAlarmOutboxPublished(eventID, "worker-a", now)
	require.NoError(t, err)
	publishSQL = strings.Join(strings.Fields(publishSQL), " ")
	require.Contains(t, publishSQL, "published_at =")
	require.Contains(t, publishSQL, "locked_by =")
	require.Contains(t, publishSQL, "status =")
	require.Contains(t, publishSQL, "event_id =")
	require.Contains(t, publishArgs, OutboxStatusPublished)
	require.Contains(t, publishArgs, OutboxStatusPublishing)
	require.Contains(t, publishArgs, "worker-a")

	replaySQL, replayArgs, err := buildReplayAlarmOutboxEvent(eventID, now)
	require.NoError(t, err)
	replaySQL = strings.Join(strings.Fields(replaySQL), " ")
	require.Contains(t, replaySQL, "published_at =")
	require.Contains(t, replaySQL, "event_id =")
	require.Contains(t, replaySQL, "status =")
	require.Contains(t, replayArgs, OutboxStatusPending)
	require.Contains(t, replayArgs, OutboxStatusPublished)

	cleanupSQL, cleanupArgs, err := buildDeletePublishedAlarmOutboxBefore(now.Add(-30 * 24 * time.Hour))
	require.NoError(t, err)
	cleanupSQL = strings.Join(strings.Fields(cleanupSQL), " ")
	require.Equal(t, "DELETE FROM alarm_event_outbox WHERE status = $1 AND published_at < $2", cleanupSQL)
	require.Equal(t, []any{OutboxStatusPublished, now.Add(-30 * 24 * time.Hour)}, cleanupArgs)
}

func TestAlarmOutboxRelayPublishesPersistedEnvelopeAndMarksOnlyAfterAck(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 3, 4, 0, time.UTC)
	eventID := uuid.New()
	payload := json.RawMessage(`{"schema_version":1,"event_type":"alarm.raised"}`)
	repo := &alarmOutboxRepoStub{records: []OutboxRecord{{
		EventID: eventID, Subject: event.SubjectAlarmRaised, Payload: payload,
		CreatedAt: now.Add(-time.Second),
	}}}
	bus := &alarmOutboxBusStub{}
	relay := NewAlarmOutboxRelay(repo, bus, nil).WithOptions(AlarmOutboxRelayOptions{
		WorkerID: "worker-a", BatchSize: 10, LeaseDuration: time.Minute, Now: func() time.Time { return now },
	})

	published, err := relay.RelayOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, published)
	require.Equal(t, []uuid.UUID{eventID}, repo.published)
	require.Empty(t, repo.failed)
	require.Equal(t, []string{event.SubjectAlarmRaised}, bus.subjects)
	require.Len(t, bus.events, 1)
	require.Equal(t, eventID.String(), bus.events[0].ID, "NATS envelope ID must reuse the durable outbox event_id")
	require.Equal(t, event.SubjectAlarmRaised, bus.events[0].Subject)
	require.JSONEq(t, string(payload), string(bus.events[0].Payload))
	require.Equal(t, now, bus.events[0].Timestamp)
}

func TestAlarmOutboxRelayPubAckFailureSchedulesRedactedExponentialRetry(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 3, 4, 0, time.UTC)
	eventID := uuid.New()
	repo := &alarmOutboxRepoStub{records: []OutboxRecord{{
		EventID: eventID, Subject: event.SubjectAlarmRaised, Payload: json.RawMessage(`{}`), AttemptCount: 2,
	}}}
	bus := &alarmOutboxBusStub{err: errors.New("nats failed: password=super-secret\nAuthorization: Bearer token")}
	relay := NewAlarmOutboxRelay(repo, bus, nil).WithOptions(AlarmOutboxRelayOptions{
		WorkerID: "worker-a", BatchSize: 10, MaxAttempts: 10,
		RetryBase: time.Second, RetryMax: time.Minute, Now: func() time.Time { return now },
	})

	published, err := relay.RelayOnce(context.Background())
	require.Error(t, err)
	require.Equal(t, 0, published)
	require.Empty(t, repo.published, "a failed PubAck must never be marked published")
	require.Len(t, repo.failed, 1)
	failure := repo.failed[0]
	require.Equal(t, eventID, failure.EventID)
	require.Equal(t, 3, failure.AttemptCount)
	require.Equal(t, now.Add(4*time.Second), failure.NextAttemptAt)
	require.Equal(t, "publish_failed", failure.LastError)
	require.NotContains(t, failure.LastError, "super-secret")
}

func TestAlarmOutboxRelayDeadLettersAtMaxAttempts(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 3, 4, 0, time.UTC)
	repo := &alarmOutboxRepoStub{records: []OutboxRecord{{
		EventID: uuid.New(), Subject: event.SubjectAlarmRaised, Payload: json.RawMessage(`{}`), AttemptCount: 19,
	}}}
	bus := &alarmOutboxBusStub{err: errors.New("unavailable")}
	relay := NewAlarmOutboxRelay(repo, bus, nil).WithOptions(AlarmOutboxRelayOptions{
		WorkerID: "worker-a", MaxAttempts: 20, Now: func() time.Time { return now },
	})

	_, err := relay.RelayOnce(context.Background())
	require.Error(t, err)
	require.Len(t, repo.failed, 1)
	require.Equal(t, 20, repo.failed[0].AttemptCount)
}

func TestAlarmOutboxRelayClaimCarriesLeaseAndReplayIsScoped(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 3, 4, 0, time.UTC)
	repo := &alarmOutboxRepoStub{}
	relay := NewAlarmOutboxRelay(repo, &alarmOutboxBusStub{}, nil).WithOptions(AlarmOutboxRelayOptions{
		WorkerID: "worker-a", BatchSize: 7, LeaseDuration: 90 * time.Second, Now: func() time.Time { return now },
	})

	_, err := relay.RelayOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, []OutboxClaimRequest{{
		WorkerID: "worker-a", Now: now, LeaseDuration: 90 * time.Second, Limit: 7,
	}}, repo.claimRequests)

	eventID := uuid.New()
	require.NoError(t, relay.Replay(context.Background(), eventID))
	require.Equal(t, []uuid.UUID{eventID}, repo.replayed)
}
