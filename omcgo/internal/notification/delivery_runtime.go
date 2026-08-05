package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DeliveryRunOnce interface {
	RunOnce(context.Context) (int, error)
}

// DeliveryRuntime owns only the polling lifecycle. Database leases provide
// cross-instance coordination; this loop deliberately adds no in-memory queue.
type DeliveryRuntime struct {
	scheduler DeliveryRunOnce
	worker    DeliveryRunOnce
	interval  time.Duration
	logger    *zap.Logger
	cancel    context.CancelFunc
	done      chan struct{}
	mu        sync.Mutex
}

func NewDeliveryRuntime(scheduler, worker DeliveryRunOnce, logger *zap.Logger) *DeliveryRuntime {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DeliveryRuntime{
		scheduler: scheduler, worker: worker, interval: time.Second,
		logger: logger.Named("notification-delivery-runtime"),
	}
}

func (r *DeliveryRuntime) RunOnce(ctx context.Context) (int, error) {
	if r == nil || r.scheduler == nil || r.worker == nil {
		return 0, fmt.Errorf("run notification delivery runtime: scheduler and worker are required")
	}
	scheduled, scheduleErr := r.scheduler.RunOnce(ctx)
	delivered, deliveryErr := r.worker.RunOnce(ctx)
	return scheduled + delivered, errors.Join(scheduleErr, deliveryErr)
}

func (r *DeliveryRuntime) Start(ctx context.Context) error {
	if r == nil || r.scheduler == nil || r.worker == nil || r.interval <= 0 {
		return fmt.Errorf("start notification delivery runtime: valid dependencies and interval are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return nil
	}
	runtimeCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.done = make(chan struct{})
	go r.loop(runtimeCtx, r.done)
	return nil
}

func (r *DeliveryRuntime) Stop() {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel, r.done = nil, nil
	r.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func (r *DeliveryRuntime) loop(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		if _, err := r.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			r.logger.Error("notification delivery pass failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
