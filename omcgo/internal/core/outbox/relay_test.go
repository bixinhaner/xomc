package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type relayRepositoryStub struct {
	entries     []Entry
	claimErr    error
	claimCalls  int
	lastOptions ClaimOptions

	published      []Entry
	failed         []Entry
	dead           []Entry
	lastError      string
	nextRetry      time.Time
	finalizeResult bool
	finalizeErr    error
}

func (s *relayRepositoryStub) ClaimDue(_ context.Context, options ClaimOptions) ([]Entry, error) {
	s.claimCalls++
	s.lastOptions = options
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	entries := s.entries
	s.entries = nil
	return entries, nil
}

func (s *relayRepositoryStub) MarkPublished(
	_ context.Context,
	id uuid.UUID,
	token uuid.UUID,
	publishedAt time.Time,
) (bool, error) {
	s.published = append(s.published, Entry{
		ID: id, ClaimToken: token, CreatedAt: publishedAt,
	})
	return s.finalize()
}

func (s *relayRepositoryStub) MarkFailed(
	_ context.Context,
	id uuid.UUID,
	token uuid.UUID,
	message string,
	nextRetry time.Time,
) (bool, error) {
	s.failed = append(s.failed, Entry{ID: id, ClaimToken: token})
	s.lastError = message
	s.nextRetry = nextRetry
	return s.finalize()
}

func (s *relayRepositoryStub) MarkDead(
	_ context.Context,
	id uuid.UUID,
	token uuid.UUID,
	message string,
	deadAt time.Time,
) (bool, error) {
	s.dead = append(s.dead, Entry{ID: id, ClaimToken: token, CreatedAt: deadAt})
	s.lastError = message
	return s.finalize()
}

func (s *relayRepositoryStub) finalize() (bool, error) {
	if s.finalizeErr != nil {
		return false, s.finalizeErr
	}
	if !s.finalizeResult {
		return false, nil
	}
	return true, nil
}

type relayPublisherStub struct {
	events   []event.Event
	subjects []string
	err      error
}

func (s *relayPublisherStub) Publish(_ context.Context, subject string, evt event.Event) error {
	s.subjects = append(s.subjects, subject)
	s.events = append(s.events, evt)
	return s.err
}

func relayTestEntry(attempts int) Entry {
	return Entry{
		ID:         uuid.New(),
		Subject:    event.SubjectDeviceLocationObserved,
		Payload:    json.RawMessage(`{"observation_version":18}`),
		Status:     StatusPublishing,
		Attempts:   attempts,
		ClaimToken: uuid.New(),
		CreatedAt:  time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
	}
}

func relayTestConfig() RelayConfig {
	return RelayConfig{
		PollInterval: time.Second,
		ClaimLease:   30 * time.Second,
		BatchSize:    100,
		MaxAttempts:  3,
		BaseBackoff:  time.Second,
		MaxBackoff:   5 * time.Minute,
	}
}

func TestRelayPublishesStableEventEnvelopeAndMarksClaimPublished(t *testing.T) {
	entry := relayTestEntry(1)
	repo := &relayRepositoryStub{entries: []Entry{entry}, finalizeResult: true}
	publisher := &relayPublisherStub{}
	relay, err := NewRelay(repo, publisher, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)
	now := time.Date(2026, 7, 30, 12, 1, 0, 0, time.UTC)
	relay.now = func() time.Time { return now }

	processed, err := relay.ProcessOnce(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, entry.ID.String(), publisher.events[0].ID)
	assert.Equal(t, entry.Subject, publisher.events[0].Subject)
	assert.JSONEq(t, string(entry.Payload), string(publisher.events[0].Payload))
	assert.Equal(t, entry.CreatedAt, publisher.events[0].Timestamp)
	assert.Equal(t, []string{entry.Subject}, publisher.subjects)
	require.Len(t, repo.published, 1)
	assert.Equal(t, entry.ID, repo.published[0].ID)
	assert.Equal(t, entry.ClaimToken, repo.published[0].ClaimToken)
	assert.Equal(t, now, repo.published[0].CreatedAt)
}

func TestRelaySchedulesExponentialRetryAfterPublishFailure(t *testing.T) {
	entry := relayTestEntry(2)
	repo := &relayRepositoryStub{entries: []Entry{entry}, finalizeResult: true}
	publishErr := errors.New("nats unavailable")
	publisher := &relayPublisherStub{err: publishErr}
	relay, err := NewRelay(repo, publisher, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)
	now := time.Date(2026, 7, 30, 12, 1, 0, 0, time.UTC)
	relay.now = func() time.Time { return now }

	processed, err := relay.ProcessOnce(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	require.Len(t, repo.failed, 1)
	assert.Empty(t, repo.dead)
	assert.Equal(t, publishErr.Error(), repo.lastError)
	assert.Equal(t, now.Add(2*time.Second), repo.nextRetry)
}

func TestRelayMarksDeadWhenPublishFailsAtMaximumAttempt(t *testing.T) {
	entry := relayTestEntry(3)
	repo := &relayRepositoryStub{entries: []Entry{entry}, finalizeResult: true}
	publisher := &relayPublisherStub{err: errors.New("nats unavailable")}
	relay, err := NewRelay(repo, publisher, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)

	processed, err := relay.ProcessOnce(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Empty(t, repo.failed)
	require.Len(t, repo.dead, 1)
	assert.Equal(t, entry.ID, repo.dead[0].ID)
	assert.Equal(t, entry.ClaimToken, repo.dead[0].ClaimToken)
}

func TestRelayReturnsFinalizationFailureForLeaseRecovery(t *testing.T) {
	entry := relayTestEntry(1)
	finalizeErr := errors.New("database unavailable")
	repo := &relayRepositoryStub{
		entries:        []Entry{entry},
		finalizeResult: true,
		finalizeErr:    finalizeErr,
	}
	relay, err := NewRelay(repo, &relayPublisherStub{}, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)

	processed, err := relay.ProcessOnce(context.Background())

	assert.Zero(t, processed)
	require.ErrorIs(t, err, finalizeErr)
}

func TestRelayReturnsErrorWhenClaimOwnershipWasLost(t *testing.T) {
	entry := relayTestEntry(1)
	repo := &relayRepositoryStub{entries: []Entry{entry}}
	relay, err := NewRelay(repo, &relayPublisherStub{}, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)

	processed, err := relay.ProcessOnce(context.Background())

	assert.Zero(t, processed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claim ownership lost")
}

func TestRelayEmptyClaimDoesNotPublish(t *testing.T) {
	repo := &relayRepositoryStub{finalizeResult: true}
	publisher := &relayPublisherStub{}
	relay, err := NewRelay(repo, publisher, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)

	processed, err := relay.ProcessOnce(context.Background())

	require.NoError(t, err)
	assert.Zero(t, processed)
	assert.Empty(t, publisher.events)
}

func TestRelayRunProcessesImmediatelyPollsAndStopsOnCancellation(t *testing.T) {
	repo := &relayRepositoryStub{finalizeResult: true}
	relay, err := NewRelay(repo, &relayPublisherStub{}, relayTestConfig(), zap.NewNop())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	waitCalls := 0
	relay.wait = func(context.Context, time.Duration) bool {
		waitCalls++
		if waitCalls == 1 {
			return true
		}
		cancel()
		return false
	}

	err = relay.Run(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, repo.claimCalls)
	assert.Equal(t, 2, waitCalls)
}
