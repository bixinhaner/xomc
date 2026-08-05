package notification

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

const (
	ScheduleKindInitialGate       = "initial_gate"
	ScheduleKindRepeat            = "repeat"
	ScheduleKindDigestFlush       = "digest_flush"
	ScheduleKindQuietHoursRelease = "quiet_hours_release"
)

type ScheduleHandler func(context.Context, DomainSchedule) error

// Scheduler executes one bounded database-backed scheduling pass. The caller
// owns the process loop so this component never sleeps or creates hidden work.
type Scheduler struct {
	repository  ScheduleRepository
	occurrences InboxRepository
	handlers    map[string]ScheduleHandler
	workerID    string
	now         func() time.Time
	lease       time.Duration
	retryDelay  time.Duration
	batchSize   int
}

func NewScheduler(
	repository ScheduleRepository,
	occurrences InboxRepository,
	workerID string,
	handlers map[string]ScheduleHandler,
) *Scheduler {
	return &Scheduler{
		repository: repository, occurrences: occurrences, workerID: workerID,
		handlers: handlers, now: func() time.Time { return time.Now().UTC() },
		lease: 30 * time.Second, retryDelay: 5 * time.Second, batchSize: 100,
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.repository == nil || s.occurrences == nil {
		return 0, fmt.Errorf("run notification scheduler: repositories are required")
	}
	if s.workerID == "" {
		return 0, fmt.Errorf("run notification scheduler: worker ID is required")
	}
	now := s.now()
	kinds := make([]string, 0, len(s.handlers))
	for kind, handler := range s.handlers {
		if handler != nil {
			kinds = append(kinds, kind)
		}
	}
	sort.Strings(kinds)
	if len(kinds) == 0 {
		return 0, fmt.Errorf("run notification scheduler: at least one schedule handler is required")
	}
	schedules, err := s.repository.ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: s.workerID, Now: now, LeaseDuration: s.lease, Limit: s.batchSize, Kinds: kinds,
	})
	if err != nil {
		return 0, fmt.Errorf("claim notification schedules: %w", err)
	}

	var runErrors []error
	for _, schedule := range schedules {
		if err := s.runClaimed(ctx, schedule); err != nil {
			runErrors = append(runErrors, err)
		}
	}
	return len(schedules), errors.Join(runErrors...)
}

func (s *Scheduler) runClaimed(ctx context.Context, schedule DomainSchedule) error {
	occurrence, err := s.occurrences.GetOccurrence(ctx, schedule.OccurrenceID)
	if err != nil || occurrence == nil {
		if err == nil {
			err = fmt.Errorf("occurrence does not exist")
		}
		return errors.Join(
			fmt.Errorf("read occurrence for notification schedule %s: %w", schedule.ID, err),
			s.release(ctx, schedule.ID),
		)
	}
	if (scheduleUsesLifecycleFence(schedule.ScheduleKind) && occurrence.ScheduleGeneration != schedule.Generation) ||
		((schedule.ScheduleKind == ScheduleKindInitialGate || schedule.ScheduleKind == ScheduleKindRepeat) && occurrence.Status != "active") {
		if err := s.repository.MarkCancelled(ctx, schedule.ID, s.workerID, s.now()); err != nil {
			return fmt.Errorf("cancel fenced notification schedule %s: %w", schedule.ID, err)
		}
		return nil
	}
	handler, ok := s.handlers[schedule.ScheduleKind]
	if !ok || handler == nil {
		return errors.Join(
			fmt.Errorf("run notification schedule %s: unsupported schedule kind %q", schedule.ID, schedule.ScheduleKind),
			s.release(ctx, schedule.ID),
		)
	}
	if err := handler(ctx, schedule); err != nil {
		return errors.Join(
			fmt.Errorf("handle notification schedule %s: %w", schedule.ID, err),
			s.release(ctx, schedule.ID),
		)
	}
	if err := s.repository.MarkCompleted(ctx, schedule.ID, s.workerID, s.now()); err != nil {
		return fmt.Errorf("complete notification schedule %s: %w", schedule.ID, err)
	}
	return nil
}

func scheduleUsesLifecycleFence(kind string) bool {
	return kind != ScheduleKindDigestFlush
}

func (s *Scheduler) release(ctx context.Context, scheduleID uuid.UUID) error {
	now := s.now()
	if err := s.repository.Release(ctx, scheduleID, s.workerID, now.Add(s.retryDelay), now); err != nil {
		return fmt.Errorf("release notification schedule %s: %w", scheduleID, err)
	}
	return nil
}
