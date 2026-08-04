package alarm

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OutboxStatus is the durable publication state of a canonical alarm event.
type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusPublishing OutboxStatus = "publishing"
	OutboxStatusPublished  OutboxStatus = "published"
	OutboxStatusFailed     OutboxStatus = "failed"
	OutboxStatusDead       OutboxStatus = "dead"
)

const OutboxAggregateTypeAlarmOccurrence = "alarm_occurrence"

// IsValid reports whether status is part of the persisted state machine.
func (s OutboxStatus) IsValid() bool {
	switch s {
	case OutboxStatusPending,
		OutboxStatusPublishing,
		OutboxStatusPublished,
		OutboxStatusFailed,
		OutboxStatusDead:
		return true
	default:
		return false
	}
}

// OutboxRecord is a canonical alarm lifecycle event awaiting JetStream PubAck.
// AggregateID and AggregateVersion identify one immutable occurrence transition.
type OutboxRecord struct {
	ID               uuid.UUID       `json:"id"`
	EventID          uuid.UUID       `json:"event_id"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      uuid.UUID       `json:"aggregate_id"`
	AggregateVersion int64           `json:"aggregate_version"`
	Subject          string          `json:"subject"`
	Payload          json.RawMessage `json:"payload"`
	Status           OutboxStatus    `json:"status"`
	AttemptCount     int             `json:"attempt_count"`
	NextAttemptAt    time.Time       `json:"next_attempt_at"`
	LockedBy         *string         `json:"locked_by,omitempty"`
	LockedAt         *time.Time      `json:"locked_at,omitempty"`
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
	LastError        *string         `json:"last_error,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}
