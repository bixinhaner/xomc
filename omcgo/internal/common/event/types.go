package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event is the standard event envelope.
type Event struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// EventHandler processes a received event.
type EventHandler func(ctx context.Context, event Event) error

// NewEvent creates a new Event with a generated ID and current timestamp.
func NewEvent(subject string, payload interface{}) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:        uuid.New().String(),
		Subject:   subject,
		Payload:   data,
		Timestamp: time.Now(),
	}, nil
}

// DecodePayload unmarshals the event payload into the target.
func (e *Event) DecodePayload(target interface{}) error {
	return json.Unmarshal(e.Payload, target)
}
