package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

var (
	ErrCarrierRequired           = errors.New("carrier is required")
	ErrSerialNumberRequired      = errors.New("serial number is required")
	ErrDecisionVersionConflict   = errors.New("device access decision version conflict")
	ErrCarrierNotFound           = errors.New("carrier not found for serial number")
	ErrCarrierAmbiguous          = errors.New("multiple carriers found for serial number")
	ErrAccessEvidenceUnavailable = errors.New("required device access evidence unavailable")
)

const (
	TriggerInformFirstSeen     = "inform_first_seen"
	TriggerInformBoot          = "inform_boot"
	TriggerInformReconnected   = "inform_reconnected"
	TriggerInformPeriodic      = "inform_periodic"
	TriggerAccessListChanged   = "access_list_changed"
	TriggerPolicyPublished     = "policy_published"
	TriggerCandidateReviewed   = "candidate_reviewed"
	TriggerIdentityChanged     = "identity_changed"
	TriggerManualReevaluation  = "manual_reevaluation"
	TriggerAccessProbeEvidence = "access_probe_evidence"
	TriggerCollectionDeadline  = "collection_deadline_expired"
)

type Observation struct {
	Carrier         string
	SerialNumber    string
	DeviceID        *uuid.UUID
	OUI             string
	ProductClass    string
	SoftwareVersion string
	Technology      model.Technology
	RFControlPaths  []string
	RemoteIP        netip.Addr
	ObservedAt      time.Time
	ExpiresAt       time.Time
}

type PolicyPublishedEvent struct {
	Carrier         string `json:"carrier"`
	PolicyVersionID string `json:"policy_version_id"`
	TriggerEventID  string `json:"trigger_event_id"`
}

type Candidate struct {
	ID              uuid.UUID
	Carrier         string
	SerialNumber    string
	OUI             string
	ProductClass    string
	SoftwareVersion string
	Technology      model.Technology
	RFControlPaths  []string
	RemoteIP        netip.Addr
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
	InformCount     int64
	ReviewStatus    string
	ReviewedBy      *uuid.UUID
	ReviewedAt      *time.Time
	ExpiresAt       time.Time
}

type EvidenceRecord struct {
	Type            ConditionType
	Status          EvidenceStatus
	NormalizedValue json.RawMessage
	ValueHash       string
	Source          string
	TaskID          *uuid.UUID
	ObservedAt      time.Time
	ExpiresAt       *time.Time
}

type EvidenceBatch struct {
	Carrier      string
	SerialNumber string
	Version      int64
	Records      []EvidenceRecord
}

type AccessStateProjection struct {
	ID                uuid.UUID
	Carrier           string
	SerialNumber      string
	DeviceID          *uuid.UUID
	CandidateID       *uuid.UUID
	State             AccessState
	EffectiveDecision EffectiveAction
	ReasonCode        ReasonCode
	PolicyVersionID   *uuid.UUID
	EvidenceVersion   int64
	DecisionVersion   int64
	NormalTasksFrozen bool
	DecisionExpiresAt *time.Time
}

type EvaluationContext struct {
	State    *AccessStateProjection
	Evidence EvidenceBatch
}

type OutboxEvent struct {
	EventType string
	EventKey  string
	Payload   json.RawMessage
}

type DecisionChange struct {
	Carrier                 string
	SerialNumber            string
	DeviceID                *uuid.UUID
	CandidateID             *uuid.UUID
	TriggerType             string
	TriggerEventID          string
	ExpectedDecisionVersion int64
	PolicyVersionID         *uuid.UUID
	EvidenceVersion         int64
	OccurredAt              time.Time
	DecisionExpiresAt       *time.Time
	Decision                Decision
	Evidence                *EvidenceBatch
	IdentitySnapshot        *IdentitySnapshot
	Outbox                  OutboxEvent
}

type SavedDecision struct {
	ID              uuid.UUID
	State           AccessState
	DecisionVersion int64
	OccurredAt      time.Time
}

type Repository interface {
	LoadEvaluationContext(ctx context.Context, carrier, serialNumber string) (EvaluationContext, error)
	SaveDecision(ctx context.Context, change DecisionChange) (SavedDecision, error)
	UpsertCandidateObservation(ctx context.Context, observation Observation) (Candidate, error)
	AppendEvidence(ctx context.Context, evidence EvidenceBatch) (int64, error)
	SaveIdentitySnapshot(ctx context.Context, snapshot IdentitySnapshot) error
}
