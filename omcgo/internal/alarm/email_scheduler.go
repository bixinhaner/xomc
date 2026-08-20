package alarm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AlarmEmailSchedulerRepository interface {
	GetGlobalSetting(ctx context.Context) (*AlarmEmailGlobalSetting, error)
	ListEnabledSubscriptions(ctx context.Context, intervalMinutes int) ([]AlarmEmailSubscription, error)
	LatestPeriodicWindowEnd(ctx context.Context, subscriptionID uuid.UUID) (*time.Time, error)
	EnqueueRun(ctx context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string, window AlarmEmailWindow, scheduledAt time.Time) (*AlarmEmailRun, bool, error)
}

const alarmEmailCatchupLimit = 6

type AlarmEmailScheduler struct {
	repository AlarmEmailSchedulerRepository
	logger     *zap.Logger

	mu            sync.Mutex
	lastWindowEnd map[uuid.UUID]time.Time
}

func NewAlarmEmailScheduler(repository AlarmEmailSchedulerRepository, logger *zap.Logger) *AlarmEmailScheduler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AlarmEmailScheduler{
		repository:    repository,
		logger:        logger.Named("alarm-email-scheduler"),
		lastWindowEnd: make(map[uuid.UUID]time.Time),
	}
}

// RunOnce enqueues a bounded number of deterministic windows per periodic
// subscription. The database unique key remains the cross-process/restart
// deduplication guard; lastWindowEnd only avoids needless duplicate insert
// attempts within a worker.
func (s *AlarmEmailScheduler) RunOnce(ctx context.Context, now time.Time) (int, error) {
	if s.repository == nil {
		return 0, fmt.Errorf("alarm email scheduler repository is required")
	}
	setting, err := s.repository.GetGlobalSetting(ctx)
	if err != nil {
		return 0, fmt.Errorf("load alarm email scheduler setting: %w", err)
	}
	if !setting.Enabled {
		return 0, nil
	}

	enqueued := 0
	for _, interval := range []int{AlarmEmailInterval10Min, AlarmEmailInterval30Min, AlarmEmailInterval60Min} {
		subscriptions, err := s.repository.ListEnabledSubscriptions(ctx, interval)
		if err != nil {
			return enqueued, fmt.Errorf("list %d-minute alarm email subscriptions: %w", interval, err)
		}
		for i := range subscriptions {
			subscription := &subscriptions[i]
			window, err := BuildAlarmEmailWindow(now, subscription.IntervalMinutes, subscription.ToleranceMinutes)
			if err != nil {
				s.logger.Warn("skip invalid alarm email subscription window",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				continue
			}
			if s.windowAlreadyObserved(subscription.ID, window.End) {
				continue
			}
			activationCutoff := setting.UpdatedAt
			if subscription.UpdatedAt.After(activationCutoff) {
				activationCutoff = subscription.UpdatedAt
			}
			if !activationCutoff.IsZero() {
				if !activationCutoff.Before(window.End) {
					continue
				}
				if window.Start.Before(activationCutoff) {
					// Preserve alarms raised after activation in the first partial
					// bucket without replaying alarms from before the subscription
					// or global mail setting became active.
					window.Start = activationCutoff
				}
			}
			windows, err := s.windowsToSchedule(ctx, subscription, window, activationCutoff)
			if err != nil {
				return enqueued, fmt.Errorf("build alarm email catch-up windows for subscription %s: %w", subscription.ID, err)
			}
			for _, candidate := range windows {
				_, created, enqueueErr := s.repository.EnqueueRun(ctx, subscription, setting.DefaultRecipients, candidate, now)
				if enqueueErr != nil {
					return enqueued, fmt.Errorf("enqueue alarm email subscription %s: %w", subscription.ID, enqueueErr)
				}
				s.observeWindow(subscription.ID, candidate.End)
				if created {
					enqueued++
				}
			}
		}
	}
	return enqueued, nil
}

func (s *AlarmEmailScheduler) windowsToSchedule(
	ctx context.Context,
	subscription *AlarmEmailSubscription,
	target AlarmEmailWindow,
	activationCutoff time.Time,
) ([]AlarmEmailWindow, error) {
	latestEnd, err := s.repository.LatestPeriodicWindowEnd(ctx, subscription.ID)
	if err != nil {
		return nil, err
	}
	if latestEnd == nil || (!activationCutoff.IsZero() && latestEnd.Before(activationCutoff)) {
		return []AlarmEmailWindow{target}, nil
	}
	if !latestEnd.Before(target.End) {
		return nil, nil
	}
	interval := time.Duration(subscription.IntervalMinutes) * time.Minute
	gap := target.End.Sub(*latestEnd)
	if gap <= 0 || gap%interval != 0 {
		// The subscription interval changed. Start from the current deterministic
		// bucket instead of creating misaligned historical windows.
		return []AlarmEmailWindow{target}, nil
	}
	missing := int(gap / interval)
	if missing > alarmEmailCatchupLimit {
		// Do not replay an unbounded historical backlog after a long maintenance
		// window or an intentional disable/enable cycle.
		return []AlarmEmailWindow{target}, nil
	}
	windows := make([]AlarmEmailWindow, 0, missing)
	for start := *latestEnd; start.Before(target.End) && len(windows) < alarmEmailCatchupLimit; start = start.Add(interval) {
		windows = append(windows, AlarmEmailWindow{Start: start, End: start.Add(interval)})
	}
	return windows, nil
}

func (s *AlarmEmailScheduler) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	if count, err := s.RunOnce(ctx, time.Now()); err != nil {
		s.logger.Warn("initial alarm email schedule failed", zap.Error(err))
	} else if count > 0 {
		s.logger.Info("initial alarm email runs enqueued", zap.Int("count", count))
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			count, err := s.RunOnce(ctx, now)
			if err != nil {
				s.logger.Warn("alarm email schedule failed", zap.Error(err))
			} else if count > 0 {
				s.logger.Info("alarm email runs enqueued", zap.Int("count", count))
			}
		}
	}
}

func (s *AlarmEmailScheduler) windowAlreadyObserved(subscriptionID uuid.UUID, end time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastWindowEnd[subscriptionID].Equal(end)
}

func (s *AlarmEmailScheduler) observeWindow(subscriptionID uuid.UUID, end time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastWindowEnd[subscriptionID] = end
}
