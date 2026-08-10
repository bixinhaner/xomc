package geofence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
)

const CoordinatorQueue = "geofence-coordinator"

// Coordinator is the queue-group boundary for immutable device location
// observations. It delegates all state and event mutations to one repository
// transaction and deliberately has no notification or device-control
// dependencies.
type Coordinator struct {
	repository CoordinatorRepository
	now        func() time.Time

	mu                    sync.Mutex
	subscription          event.Subscription
	lifecycleSubscription event.Subscription
}

func NewCoordinator(repository CoordinatorRepository) *Coordinator {
	return &Coordinator{
		repository: repository,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (c *Coordinator) Start(bus event.EventBus) error {
	if c == nil || c.repository == nil {
		return fmt.Errorf("start geofence coordinator: repository is required")
	}
	if bus == nil {
		return fmt.Errorf("start geofence coordinator: event bus is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscription != nil {
		return nil
	}
	subscription, err := bus.QueueSubscribe(
		event.SubjectDeviceLocationObserved,
		CoordinatorQueue,
		c.Handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe geofence coordinator: %w", err)
	}
	if subscription == nil {
		return fmt.Errorf("subscribe geofence coordinator: nil subscription")
	}
	c.subscription = subscription
	lifecycleSubscription, err := bus.QueueSubscribe(
		event.SubjectGeofenceLifecycleReevaluate,
		CoordinatorQueue,
		c.HandleLifecycleReevaluate,
	)
	if err != nil {
		_ = subscription.Unsubscribe()
		c.subscription = nil
		return fmt.Errorf("subscribe geofence lifecycle reevaluation: %w", err)
	}
	if lifecycleSubscription == nil {
		_ = subscription.Unsubscribe()
		c.subscription = nil
		return fmt.Errorf("subscribe geofence lifecycle reevaluation: nil subscription")
	}
	c.lifecycleSubscription = lifecycleSubscription
	return nil
}

func (c *Coordinator) Stop() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	subscription := c.subscription
	lifecycleSubscription := c.lifecycleSubscription
	c.subscription = nil
	c.lifecycleSubscription = nil
	c.mu.Unlock()
	if subscription == nil && lifecycleSubscription == nil {
		return nil
	}
	if subscription != nil {
		if err := subscription.Unsubscribe(); err != nil {
			return fmt.Errorf("unsubscribe geofence coordinator: %w", err)
		}
	}
	if lifecycleSubscription != nil {
		if err := lifecycleSubscription.Unsubscribe(); err != nil {
			return fmt.Errorf("unsubscribe geofence lifecycle reevaluation: %w", err)
		}
	}
	return nil
}

func (c *Coordinator) HandleLifecycleReevaluate(ctx context.Context, evt event.Event) error {
	if c == nil || c.repository == nil {
		return fmt.Errorf("handle geofence lifecycle reevaluation: repository is required")
	}
	var payload event.GeofenceLifecycleReevaluatePayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode geofence lifecycle reevaluation payload: %w", err)
	}
	now := time.Now().UTC()
	if c.now != nil {
		now = c.now()
	}
	if _, err := c.repository.ReevaluateLatest(ctx, payload.DeviceID, now); err != nil {
		return fmt.Errorf("reevaluate geofence lifecycle state: %w", err)
	}
	return nil
}

func (c *Coordinator) Handle(ctx context.Context, evt event.Event) error {
	if c == nil || c.repository == nil {
		return fmt.Errorf("handle geofence location: repository is required")
	}
	if evt.Subject != "" && evt.Subject != event.SubjectDeviceLocationObserved {
		return fmt.Errorf(
			"handle geofence location: unexpected subject %q",
			evt.Subject,
		)
	}
	var payload event.DeviceLocationObservedPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device location observed payload: %w", err)
	}
	now := time.Now().UTC()
	if c.now != nil {
		now = c.now()
	}
	if _, err := c.repository.EvaluateLocation(ctx, payload, now); err != nil {
		return fmt.Errorf("coordinate device geofence evaluation: %w", err)
	}
	return nil
}
