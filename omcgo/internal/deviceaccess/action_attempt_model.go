package deviceaccess

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ActionAttemptPhase string

const (
	ActionAttemptBaselineGPV ActionAttemptPhase = "baseline_gpv"
	ActionAttemptSPV         ActionAttemptPhase = "spv"
	ActionAttemptReadbackGPV ActionAttemptPhase = "readback_gpv"
)

type ActionAttemptStatus string

const (
	ActionAttemptQueued    ActionAttemptStatus = "queued"
	ActionAttemptSent      ActionAttemptStatus = "sent"
	ActionAttemptSucceeded ActionAttemptStatus = "succeeded"
	ActionAttemptFailed    ActionAttemptStatus = "failed"
	ActionAttemptTimeout   ActionAttemptStatus = "timeout"
	ActionAttemptCancelled ActionAttemptStatus = "cancelled"
)

type ActionAttempt struct {
	ID              uuid.UUID           `json:"id"`
	ActionID        uuid.UUID           `json:"action_id"`
	AttemptNo       int                 `json:"attempt_no"`
	Phase           ActionAttemptPhase  `json:"phase"`
	DeviceTaskID    *uuid.UUID          `json:"device_task_id,omitempty"`
	CommandKey      string              `json:"command_key,omitempty"`
	Status          ActionAttemptStatus `json:"status"`
	FaultCode       string              `json:"fault_code,omitempty"`
	FailureCode     string              `json:"failure_code,omitempty"`
	ErrorMessage    string              `json:"error_message,omitempty"`
	RequestSummary  json.RawMessage     `json:"request_summary,omitempty"`
	ResponseSummary json.RawMessage     `json:"response_summary,omitempty"`
	TraceID         string              `json:"trace_id,omitempty"`
	StartedAt       time.Time           `json:"started_at"`
	CompletedAt     *time.Time          `json:"completed_at,omitempty"`
}
