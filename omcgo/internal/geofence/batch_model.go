package geofence

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	ManualBindJobType           = "geofence_manual_bind"
	ManualBindPayloadVersion    = 1
	MaxBatchBindingInputs       = 1000
	MaxDeviceSerialNumberLength = 64
	DefaultBatchItemPageSize    = 50
	MaxBatchItemPageSize        = 200
	ManualBindJobMaxAttempts    = 3
)

type BindingInputRequest struct {
	DeviceIDs []uuid.UUID `json:"device_ids"`
	DeviceSNs []string    `json:"device_sns"`
}

type ManualBindingPreviewRequest struct {
	GeofenceID    uuid.UUID
	Inputs        BindingInputRequest
	VisibleGroups []uuid.UUID
}

type CreateManualBindJobRequest struct {
	GeofenceID         uuid.UUID
	ActorID            uuid.UUID
	VisibleGroups      []uuid.UUID
	Inputs             BindingInputRequest
	PreviewFingerprint string
	Reason             string
	ScheduledAt        time.Time
}

type CreateManualBindJobParams struct {
	GeofenceID         uuid.UUID
	ActorID            uuid.UUID
	VisibleGroups      []uuid.UUID
	Inputs             []BindingInput
	PreviewFingerprint string
	Reason             string
	ScheduledAt        time.Time
}

type ManualBindJobPayload struct {
	SchemaVersion      int       `json:"schema_version"`
	GeofenceID         uuid.UUID `json:"geofence_id"`
	GeofenceVersionID  uuid.UUID `json:"geofence_version_id"`
	RequestedBy        uuid.UUID `json:"requested_by"`
	PreviewFingerprint string    `json:"preview_fingerprint"`
	Reason             string    `json:"reason"`
}

type BindingInputKind string

const (
	BindingInputDeviceID BindingInputKind = "device_id"
	BindingInputDeviceSN BindingInputKind = "device_sn"
)

type BindingInput struct {
	Key      string
	Kind     BindingInputKind
	Value    string
	DeviceID *uuid.UUID
}

type BindingDecision string

const (
	BindingDecisionEligible BindingDecision = "eligible"
	BindingDecisionMove     BindingDecision = "move"
	BindingDecisionSkipped  BindingDecision = "skipped"
)

type BindingPreviewItem struct {
	InputKey         string          `json:"input_key"`
	Input            string          `json:"input"`
	DeviceID         *uuid.UUID      `json:"device_id,omitempty"`
	DeviceSN         string          `json:"device_sn,omitempty"`
	Decision         BindingDecision `json:"decision"`
	ReasonCode       string          `json:"reason_code,omitempty"`
	SourceBindingID  *uuid.UUID      `json:"source_binding_id,omitempty"`
	SourceGeofenceID *uuid.UUID      `json:"source_geofence_id,omitempty"`
}

type ManualBindingPreview struct {
	GeofenceID         uuid.UUID            `json:"geofence_id"`
	GeofenceVersionID  uuid.UUID            `json:"geofence_version_id"`
	RuleType           RuleType             `json:"rule_type"`
	InputCount         int                  `json:"input_count"`
	EligibleCount      int                  `json:"eligible_count"`
	MoveCount          int                  `json:"move_count"`
	SkippedCount       int                  `json:"skipped_count"`
	Items              []BindingPreviewItem `json:"items"`
	PreviewFingerprint string               `json:"preview_fingerprint"`
}

type ManualBindSnapshot struct {
	Definition       Definition
	Inputs           []BindingCandidateFact
	VisibilityDigest string
}

type BindingCandidateFact struct {
	Input                 BindingInput
	Device                *DeviceIdentity
	SameGeofenceBindingID *uuid.UUID
	SameGeofenceStatus    *BindingStatus
	ActiveRuleBindingID   *uuid.UUID
	ActiveRuleGeofenceID  *uuid.UUID
}

type BatchItemStatus string

const (
	BatchItemPending      BatchItemStatus = "pending"
	BatchItemSucceeded    BatchItemStatus = "succeeded"
	BatchItemSkipped      BatchItemStatus = "skipped"
	BatchItemFailed       BatchItemStatus = "failed"
	ReasonProcessingError                 = "processing_error"
)

type BatchProgress struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Succeeded int64 `json:"succeeded"`
	Skipped   int64 `json:"skipped"`
	Failed    int64 `json:"failed"`
}

type BatchJobAccepted struct {
	JobID uuid.UUID `json:"job_id"`
}

type BatchJob struct {
	ID           uuid.UUID       `json:"id"`
	JobType      string          `json:"job_type"`
	Status       asyncjob.Status `json:"status"`
	GeofenceID   uuid.UUID       `json:"geofence_id"`
	RequestedBy  uuid.UUID       `json:"requested_by"`
	Reason       string          `json:"reason"`
	Attempt      int             `json:"attempt"`
	MaxAttempts  int             `json:"max_attempts"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	Progress     BatchProgress   `json:"progress"`
}

type BatchItem struct {
	ID               uuid.UUID        `json:"id"`
	JobID            uuid.UUID        `json:"job_id"`
	GeofenceID       uuid.UUID        `json:"geofence_id"`
	InputKey         string           `json:"input_key"`
	InputKind        BindingInputKind `json:"input_kind"`
	InputValue       string           `json:"input_value"`
	DeviceID         *uuid.UUID       `json:"device_id,omitempty"`
	DeviceSNSnapshot string           `json:"device_sn,omitempty"`
	Status           BatchItemStatus  `json:"status"`
	ReasonCode       string           `json:"reason_code,omitempty"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	BindingID        *uuid.UUID       `json:"binding_id,omitempty"`
	Attempt          int              `json:"attempt"`
	StartedAt        *time.Time       `json:"started_at,omitempty"`
	FinishedAt       *time.Time       `json:"finished_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type BatchItemPage struct {
	Items    []BatchItem `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type BatchItemFilter struct {
	JobID         uuid.UUID
	Status        BatchItemStatus
	Page          int
	PageSize      int
	VisibleGroups []uuid.UUID
}

var (
	ErrStaleBindingPreview = fmt.Errorf(
		"manual binding preview is stale: %w",
		commonerrors.ErrAlreadyExists,
	)
	ErrNoEligibleBindingInputs = errors.New(
		"manual binding preview has no eligible inputs",
	)
)

func normalizeBindingInputs(req BindingInputRequest) ([]BindingInput, error) {
	rawCount := len(req.DeviceIDs) + len(req.DeviceSNs)
	if rawCount == 0 || rawCount > MaxBatchBindingInputs {
		return nil, fmt.Errorf(
			"batch binding input count must be between 1 and %d: %w",
			MaxBatchBindingInputs,
			commonerrors.ErrInvalidInput,
		)
	}

	seen := make(map[string]struct{}, rawCount)
	inputs := make([]BindingInput, 0, rawCount)
	appendInput := func(input BindingInput) {
		if _, ok := seen[input.Key]; ok {
			return
		}
		seen[input.Key] = struct{}{}
		inputs = append(inputs, input)
	}
	for _, id := range req.DeviceIDs {
		if id == uuid.Nil {
			continue
		}
		value := id.String()
		idCopy := id
		appendInput(BindingInput{
			Key: "id:" + value, Kind: BindingInputDeviceID,
			Value: value, DeviceID: &idCopy,
		})
	}
	for _, raw := range req.DeviceSNs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if utf8.RuneCountInString(value) > MaxDeviceSerialNumberLength {
			return nil, fmt.Errorf(
				"device serial number must not exceed %d characters: %w",
				MaxDeviceSerialNumberLength,
				commonerrors.ErrInvalidInput,
			)
		}
		appendInput(BindingInput{
			Key: "sn:" + value, Kind: BindingInputDeviceSN, Value: value,
		})
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf(
			"batch binding inputs are empty after normalization: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	return inputs, nil
}
