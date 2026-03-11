package event

import "context"

// EventBus defines the interface for publishing and subscribing to events.
type EventBus interface {
	// Publish sends an event to all subscribers of the given subject.
	Publish(ctx context.Context, subject string, event Event) error

	// Subscribe registers a handler for events on the given subject.
	// Each subscriber independently receives all matching events.
	Subscribe(subject string, handler EventHandler) (Subscription, error)

	// QueueSubscribe registers a handler in a named queue group.
	// Events are load-balanced across handlers in the same queue group.
	QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)

	// Close shuts down the event bus and releases resources.
	Close() error
}

// Subscription represents an active event subscription.
type Subscription interface {
	Unsubscribe() error
}
