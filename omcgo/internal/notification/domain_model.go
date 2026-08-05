package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the idempotent notification Inbox copy of one canonical
// alarm lifecycle event. Payload remains the immutable event-time snapshot.
type DomainEvent struct {
	ID              uuid.UUID
	EventID         uuid.UUID
	EventType       string
	OccurrenceID    uuid.UUID
	AlarmVersion    int64
	SchemaVersion   int
	OccurredAt      time.Time
	Payload         json.RawMessage
	ProcessingState string
	ReceivedAt      time.Time
	ProcessedAt     *time.Time
	LastError       *string
}

// DomainOccurrence is a notification-only ordered projection. It is a send
// fence and never replaces the alarm domain as the source of truth.
type DomainOccurrence struct {
	ID                 uuid.UUID
	OccurrenceID       uuid.UUID
	LastAppliedVersion int64
	ScheduleGeneration int64
	Status             string
	Severity           int16
	DeviceID           uuid.UUID
	DeviceSN           string
	Carrier            string
	Technology         string
	AlarmIdentifier    string
	CurrentSnapshot    json.RawMessage
	RaisedAt           time.Time
	AcknowledgedAt     *time.Time
	ClearedAt          *time.Time
	LastEventID        uuid.UUID
	VersionGap         bool
	GapFirstSeenAt     *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type DomainRuleVersion struct {
	ID              uuid.UUID
	RuleID          uuid.UUID
	VersionNo       int64
	MatchConditions json.RawMessage
	Policy          json.RawMessage
	CreatedBy       string
	ChangeReason    string
	CreatedAt       time.Time
	PublishedAt     *time.Time
}

type DomainTemplateVersion struct {
	ID           uuid.UUID
	TemplateID   uuid.UUID
	VersionNo    int64
	Channel      string
	Language     string
	Subject      string
	TextBody     string
	HTMLBody     *string
	Variables    []string
	CreatedBy    string
	ChangeReason string
	CreatedAt    time.Time
	PublishedAt  *time.Time
}

type DomainSchedule struct {
	ID                   uuid.UUID
	OccurrenceID         uuid.UUID
	RuleVersionID        uuid.UUID
	Channel              string
	RecipientFingerprint []byte
	ScheduleKind         string
	SequenceNo           int
	Generation           int64
	DueAt                time.Time
	State                string
	CreatedEventVersion  int64
	CreatedAt            time.Time
}

type DomainDelivery struct {
	ID                   uuid.UUID
	EventID              uuid.UUID
	OccurrenceID         uuid.UUID
	RuleVersionID        uuid.UUID
	TemplateVersionID    uuid.UUID
	ChannelConfigID      uuid.UUID
	Channel              string
	DispatchKind         string
	SequenceNo           int
	RecipientType        string
	AddressCiphertext    []byte
	AddressKeyVersion    int
	RecipientFingerprint []byte
	FlowState            string
	DeliveryResult       string
	AvailableAt          time.Time
	NextAttemptAt        time.Time
	OccurrenceVersion    int64
	ScheduleGeneration   int64
	OriginDeliveryID     *uuid.UUID
	CreatedAt            time.Time
}

type DomainDeliveryAttempt struct {
	ID                uuid.UUID
	DeliveryID        uuid.UUID
	AttemptNo         int
	StartedAt         time.Time
	FinishedAt        *time.Time
	Result            string
	ErrorCategory     *string
	StatusSummary     *string
	ProviderRequestID *string
	NextRetryAt       *time.Time
	CreatedAt         time.Time
}
