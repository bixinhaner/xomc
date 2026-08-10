// Package outbox provides the generic transactional event-outbox contract.
package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusPublishing Status = "publishing"
	StatusPublished  Status = "published"
	StatusFailed     Status = "failed"
	StatusDead       Status = "dead"
)

// Record is one durable business event waiting to be published.
type Record struct {
	AggregateType string
	AggregateID   string
	Subject       string
	Payload       json.RawMessage
	DedupeKey     string
}

// Entry is a claimed durable event ready for EventBus publication.
type Entry struct {
	ID             uuid.UUID
	AggregateType  string
	AggregateID    string
	Subject        string
	Payload        json.RawMessage
	DedupeKey      string
	Status         Status
	Attempts       int
	ClaimToken     uuid.UUID
	ClaimExpiresAt time.Time
	CreatedAt      time.Time
}

// ClaimOptions controls one short, transactional relay claim.
type ClaimOptions struct {
	Now         time.Time
	Lease       time.Duration
	Limit       int
	MaxAttempts int
}

// DeliveryRepository owns the durable delivery state machine. Each finalizer
// is conditional on the current lease token and returns false when ownership
// was lost.
type DeliveryRepository interface {
	ClaimDue(context.Context, ClaimOptions) ([]Entry, error)
	MarkPublished(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error)
	MarkDead(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error)
}
