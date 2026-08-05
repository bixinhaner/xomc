package notification

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type schedulerRepositoryStub struct {
	mu         sync.Mutex
	schedules  map[uuid.UUID]DomainSchedule
	afterClaim func()
}

func (s *schedulerRepositoryStub) InsertSchedule(_ context.Context, schedule *DomainSchedule) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.schedules[schedule.ID]; exists {
		return false, nil
	}
	s.schedules[schedule.ID] = *schedule
	return true, nil
}

func (s *schedulerRepositoryStub) ClaimDue(_ context.Context, request ScheduleClaimRequest) ([]DomainSchedule, error) {
	s.mu.Lock()
	claimed := make([]DomainSchedule, 0, request.Limit)
	for id, schedule := range s.schedules {
		if len(request.Kinds) > 0 && !containsScheduleKind(request.Kinds, schedule.ScheduleKind) {
			continue
		}
		due := schedule.State == "pending" && !schedule.DueAt.After(request.Now)
		expired := schedule.State == "claimed" && schedule.LeaseExpiresAt != nil && !schedule.LeaseExpiresAt.After(request.Now)
		if (!due && !expired) || len(claimed) >= request.Limit {
			continue
		}
		lockedAt, expiresAt, workerID := request.Now, request.Now.Add(request.LeaseDuration), request.WorkerID
		schedule.State, schedule.LockedBy = "claimed", &workerID
		schedule.LockedAt, schedule.LeaseExpiresAt = &lockedAt, &expiresAt
		s.schedules[id] = schedule
		claimed = append(claimed, schedule)
	}
	s.mu.Unlock()
	if s.afterClaim != nil {
		s.afterClaim()
	}
	return claimed, nil
}

func containsScheduleKind(kinds []string, want string) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}

func (s *schedulerRepositoryStub) MarkCompleted(_ context.Context, id uuid.UUID, workerID string, now time.Time) error {
	return s.transition(id, workerID, "completed", now)
}

func (s *schedulerRepositoryStub) MarkCancelled(_ context.Context, id uuid.UUID, workerID string, now time.Time) error {
	return s.transition(id, workerID, "cancelled", now)
}

func (s *schedulerRepositoryStub) Release(_ context.Context, id uuid.UUID, workerID string, dueAt, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	schedule := s.schedules[id]
	if schedule.State != "claimed" || schedule.LockedBy == nil || *schedule.LockedBy != workerID {
		return ErrScheduleLeaseLost
	}
	schedule.State, schedule.DueAt, schedule.UpdatedAt = "pending", dueAt, now
	schedule.LockedBy, schedule.LockedAt, schedule.LeaseExpiresAt = nil, nil, nil
	s.schedules[id] = schedule
	return nil
}

func (s *schedulerRepositoryStub) transition(id uuid.UUID, workerID, state string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	schedule := s.schedules[id]
	if schedule.State != "claimed" || schedule.LockedBy == nil || *schedule.LockedBy != workerID {
		return ErrScheduleLeaseLost
	}
	schedule.State, schedule.UpdatedAt = state, now
	if state == "completed" {
		schedule.CompletedAt = &now
	} else {
		schedule.CancelledAt = &now
	}
	schedule.LockedBy, schedule.LockedAt, schedule.LeaseExpiresAt = nil, nil, nil
	s.schedules[id] = schedule
	return nil
}

type schedulerOccurrenceStub struct {
	occurrences map[uuid.UUID]*DomainOccurrence
	err         error
}

func (*schedulerOccurrenceStub) InsertEvent(context.Context, *DomainEvent) (bool, error) {
	return false, nil
}
func (s *schedulerOccurrenceStub) GetOccurrence(_ context.Context, id uuid.UUID) (*DomainOccurrence, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.occurrences[id], nil
}

func TestScheduler_TwoInstancesClaimScheduleOnlyOnce(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 0, 0, 0, time.UTC)
	schedule := schedulerFixture(now, ScheduleKindInitialGate, 1)
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{schedule.ID: schedule}}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		schedule.OccurrenceID: {OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 1, Status: "active"},
	}}
	calls := 0
	handlers := map[string]ScheduleHandler{ScheduleKindInitialGate: func(context.Context, DomainSchedule) error { calls++; return nil }}
	first, second := NewScheduler(repository, occurrences, "worker-a", handlers), NewScheduler(repository, occurrences, "worker-b", handlers)
	first.now, second.now = func() time.Time { return now }, func() time.Time { return now }

	processed, err := first.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	processed, err = second.RunOnce(context.Background())
	require.NoError(t, err)
	require.Zero(t, processed)
	require.Equal(t, 1, calls)
	require.Equal(t, "completed", repository.schedules[schedule.ID].State)
}

func TestScheduler_RecoversExpiredLease(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 10, 0, 0, time.UTC)
	schedule := schedulerFixture(now.Add(-time.Minute), ScheduleKindRepeat, 2)
	deadWorker, expired := "dead-worker", now.Add(-time.Second)
	schedule.State, schedule.LockedBy, schedule.LeaseExpiresAt = "claimed", &deadWorker, &expired
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{schedule.ID: schedule}}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		schedule.OccurrenceID: {OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 2, Status: "active"},
	}}
	scheduler := NewScheduler(repository, occurrences, "recovery-worker", map[string]ScheduleHandler{
		ScheduleKindRepeat: func(context.Context, DomainSchedule) error { return nil },
	})
	scheduler.now = func() time.Time { return now }

	processed, err := scheduler.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, "completed", repository.schedules[schedule.ID].State)
}

func TestScheduler_GenerationFenceCancelsBeforeHandler(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 20, 0, 0, time.UTC)
	schedule := schedulerFixture(now, ScheduleKindRepeat, 2)
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{schedule.ID: schedule}}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		schedule.OccurrenceID: {OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 3, Status: "active"},
	}}
	called := false
	scheduler := NewScheduler(repository, occurrences, "worker", map[string]ScheduleHandler{
		ScheduleKindRepeat: func(context.Context, DomainSchedule) error { called = true; return nil },
	})
	scheduler.now = func() time.Time { return now }

	processed, err := scheduler.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.False(t, called)
	require.Equal(t, "cancelled", repository.schedules[schedule.ID].State)
}

func TestScheduler_ClearAfterClaimTripsGenerationFence(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 25, 0, 0, time.UTC)
	schedule := schedulerFixture(now, ScheduleKindRepeat, 1)
	occurrence := &DomainOccurrence{OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 1, Status: "active"}
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{schedule.ID: schedule}}
	repository.afterClaim = func() {
		occurrence.ScheduleGeneration = 2
		occurrence.Status = "cleared"
	}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		schedule.OccurrenceID: occurrence,
	}}
	called := false
	scheduler := NewScheduler(repository, occurrences, "worker", map[string]ScheduleHandler{
		ScheduleKindRepeat: func(context.Context, DomainSchedule) error { called = true; return nil },
	})
	scheduler.now = func() time.Time { return now }

	processed, err := scheduler.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.False(t, called)
	require.Equal(t, "cancelled", repository.schedules[schedule.ID].State)
}

func TestScheduler_HandlesOnlyFourExplicitScheduleKinds(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 30, 0, 0, time.UTC)
	kinds := []string{ScheduleKindInitialGate, ScheduleKindRepeat, ScheduleKindDigestFlush, ScheduleKindQuietHoursRelease}
	repository := &schedulerRepositoryStub{schedules: make(map[uuid.UUID]DomainSchedule)}
	occurrences := &schedulerOccurrenceStub{occurrences: make(map[uuid.UUID]*DomainOccurrence)}
	handlers := make(map[string]ScheduleHandler)
	calls := make(map[string]int)
	for _, kind := range kinds {
		schedule := schedulerFixture(now, kind, 1)
		repository.schedules[schedule.ID] = schedule
		occurrences.occurrences[schedule.OccurrenceID] = &DomainOccurrence{
			OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 1, Status: "active",
		}
		capturedKind := kind
		handlers[kind] = func(context.Context, DomainSchedule) error { calls[capturedKind]++; return nil }
	}
	scheduler := NewScheduler(repository, occurrences, "worker", handlers)
	scheduler.now = func() time.Time { return now }

	processed, err := scheduler.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(kinds), processed)
	for _, kind := range kinds {
		require.Equal(t, 1, calls[kind])
	}
}

func TestScheduler_LeavesUnregisteredScheduleKindsPending(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 35, 0, 0, time.UTC)
	initial := schedulerFixture(now, ScheduleKindInitialGate, 1)
	digest := schedulerFixture(now, ScheduleKindDigestFlush, 1)
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{
		initial.ID: initial, digest.ID: digest,
	}}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		initial.OccurrenceID: {OccurrenceID: initial.OccurrenceID, ScheduleGeneration: 1, Status: "active"},
		digest.OccurrenceID:  {OccurrenceID: digest.OccurrenceID, ScheduleGeneration: 1, Status: "active"},
	}}
	scheduler := NewScheduler(repository, occurrences, "worker", map[string]ScheduleHandler{
		ScheduleKindInitialGate: func(context.Context, DomainSchedule) error { return nil },
	})
	scheduler.now = func() time.Time { return now }

	processed, err := scheduler.RunOnce(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, "completed", repository.schedules[initial.ID].State)
	require.Equal(t, "pending", repository.schedules[digest.ID].State)
}

func TestScheduler_HandlerFailureReleasesForBoundedRetry(t *testing.T) {
	now := time.Date(2026, 8, 5, 11, 40, 0, 0, time.UTC)
	schedule := schedulerFixture(now, ScheduleKindDigestFlush, 1)
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{schedule.ID: schedule}}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		schedule.OccurrenceID: {OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 1, Status: "active"},
	}}
	wantErr := errors.New("handler unavailable")
	scheduler := NewScheduler(repository, occurrences, "worker", map[string]ScheduleHandler{
		ScheduleKindDigestFlush: func(context.Context, DomainSchedule) error { return wantErr },
	})
	scheduler.now = func() time.Time { return now }

	processed, err := scheduler.RunOnce(context.Background())
	require.Equal(t, 1, processed)
	require.ErrorIs(t, err, wantErr)
	released := repository.schedules[schedule.ID]
	require.Equal(t, "pending", released.State)
	require.Equal(t, now.Add(5*time.Second), released.DueAt)
}

func TestScheduler_DigestSurvivesOccurrenceClearGenerationChange(t *testing.T) {
	now := time.Now().UTC()
	schedule := schedulerFixture(now, ScheduleKindDigestFlush, 1)
	repository := &schedulerRepositoryStub{schedules: map[uuid.UUID]DomainSchedule{schedule.ID: schedule}}
	occurrences := &schedulerOccurrenceStub{occurrences: map[uuid.UUID]*DomainOccurrence{
		schedule.OccurrenceID: {OccurrenceID: schedule.OccurrenceID, ScheduleGeneration: 2, Status: "cleared"},
	}}
	called := false
	scheduler := NewScheduler(repository, occurrences, "worker", map[string]ScheduleHandler{
		ScheduleKindDigestFlush: func(context.Context, DomainSchedule) error { called = true; return nil },
	})
	scheduler.now = func() time.Time { return now }

	_, err := scheduler.RunOnce(context.Background())
	require.NoError(t, err)
	require.True(t, called, "digest must retain raised-and-cleared facts in the same window")
}

func schedulerFixture(now time.Time, kind string, generation int64) DomainSchedule {
	return DomainSchedule{
		ID: uuid.New(), OccurrenceID: uuid.New(), RuleVersionID: uuid.New(), Channel: "email",
		RecipientFingerprint: []byte("recipient"), ScheduleKind: kind, Generation: generation,
		DueAt: now, State: "pending", CreatedEventVersion: 1,
	}
}
