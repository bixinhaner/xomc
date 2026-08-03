package slothealth

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type Store interface {
	ExpectedGroups(context.Context, time.Time) ([]ExpectedGroup, error)
	ReceivedGroups(context.Context, time.Time) ([]ReceivedGroup, error)
	UpsertSnapshots(context.Context, []Snapshot) ([]Snapshot, error)
}

type Publisher interface {
	PublishSlotHealth([]Snapshot)
}

type Observer struct {
	store     Store
	publisher Publisher
	startupAt time.Time
	grace     time.Duration
	interval  time.Duration
	logger    *zap.Logger
	now       func() time.Time
}

func NewObserver(
	store Store,
	publisher Publisher,
	startupAt time.Time,
	grace time.Duration,
	interval time.Duration,
	logger *zap.Logger,
) *Observer {
	if grace <= 0 {
		grace = 12 * time.Minute
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Observer{
		store: store, publisher: publisher, startupAt: startupAt.UTC(),
		grace: grace, interval: interval, logger: logger, now: time.Now,
	}
}

func (o *Observer) Observe(ctx context.Context) error {
	now := o.now().UTC()
	slotEnd := LatestEligibleSlotEnd(now, o.grace)
	slotStart := slotEnd.Add(-SlotDuration)
	expected, err := o.store.ExpectedGroups(ctx, slotStart)
	if err != nil {
		return fmt.Errorf("observe PM slot expected snapshot: %w", err)
	}
	received, err := o.store.ReceivedGroups(ctx, slotEnd)
	if err != nil {
		return fmt.Errorf("observe PM slot receipts: %w", err)
	}
	snapshots := BuildSnapshots(slotEnd, now, o.startupAt, expected, received)
	persisted, err := o.store.UpsertSnapshots(ctx, snapshots)
	if err != nil {
		return fmt.Errorf("persist PM slot health: %w", err)
	}
	if o.publisher != nil {
		o.publisher.PublishSlotHealth(persisted)
	}
	return nil
}

func (o *Observer) Run(ctx context.Context) {
	o.observeAndLog(ctx)
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.observeAndLog(ctx)
		}
	}
}

func (o *Observer) observeAndLog(ctx context.Context) {
	if err := o.Observe(ctx); err != nil {
		o.logger.Warn("observe PM slot health", zap.Error(err))
	}
}
