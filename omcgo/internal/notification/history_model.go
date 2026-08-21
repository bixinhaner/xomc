package notification

import (
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// History status constants.
const (
	HistoryStatusPending       = "pending"
	HistoryStatusSent          = "sent"
	HistoryStatusFailed        = "failed"
	HistoryStatusDeadLetter    = "dead_letter"
	HistoryStatusNotConfigured = "not_configured"
	HistoryChannelSystem       = "system"
)

// NotificationHistory records a single send attempt by the notification
// dispatcher. Created either by the alarm pipeline (template-driven) or by
// ad-hoc internal callers (alarm_id / template_id may be nil).
type NotificationHistory struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TemplateID    *uuid.UUID `json:"template_id,omitempty" db:"template_id"`
	Channel       string     `json:"channel" db:"channel"`
	Recipients    []string   `json:"recipients" db:"recipients"`
	Subject       string     `json:"subject" db:"subject"`
	Body          string     `json:"body" db:"body"`
	Status        string     `json:"status" db:"status"`
	ErrorMessage  *string    `json:"error_message,omitempty" db:"error_message"`
	AlarmID       *uuid.UUID `json:"alarm_id,omitempty" db:"alarm_id"`
	SourceType    string     `json:"source_type,omitempty" db:"source_type"`
	SourceID      *uuid.UUID `json:"source_id,omitempty" db:"source_id"`
	EventID       *uuid.UUID `json:"event_id,omitempty" db:"event_id"`
	CorrelationID string     `json:"correlation_id,omitempty" db:"correlation_id"`
	RetryCount    int        `json:"retry_count" db:"retry_count"`
	SentAt        *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// NotificationHistoryFilter describes list-query parameters for history.
type NotificationHistoryFilter struct {
	Channel       *string
	Status        *string
	TemplateID    *uuid.UUID
	AlarmID       *uuid.UUID
	SourceType    *string
	SourceID      *uuid.UUID
	EventID       *uuid.UUID
	CorrelationID *string
	model.ListRequest
}
