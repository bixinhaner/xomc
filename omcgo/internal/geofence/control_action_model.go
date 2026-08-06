package geofence

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ControlActionType string

const (
	ControlActionDeactivate ControlActionType = "deactivate"
	ControlActionActivate   ControlActionType = "activate"
)

type ControlActionStatus string

const (
	ControlActionPending       ControlActionStatus = "pending"
	ControlActionExecuting     ControlActionStatus = "executing"
	ControlActionVerifying     ControlActionStatus = "verifying"
	ControlActionVerified      ControlActionStatus = "verified"
	ControlActionPartialFailed ControlActionStatus = "partial_failed"
	ControlActionFailed        ControlActionStatus = "failed"
)

// ControlParameterState is the immutable parameter evidence attached to one
// geofence control action. Paths are stored in the standard model so the
// record remains stable when a product-private mapping changes.
type ControlParameterState struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

// ControlAction is the durable ownership boundary between a geofence state
// edge and its CWMP tasks. Device tasks remain the transport mechanism; this
// record preserves the pre-control state and verified outcome that a generic
// task row cannot represent.
type ControlAction struct {
	ID                    uuid.UUID               `json:"id"`
	ActionKey             string                  `json:"action_key"`
	ParentActionID        *uuid.UUID              `json:"parent_action_id,omitempty"`
	DeviceID              uuid.UUID               `json:"device_id"`
	DeviceSN              string                  `json:"device_sn"`
	GeofenceID            *uuid.UUID              `json:"geofence_id,omitempty"`
	BindingID             *uuid.UUID              `json:"binding_id,omitempty"`
	EffectiveStateVersion int64                   `json:"effective_state_version"`
	ActionType            ControlActionType       `json:"action_type"`
	Status                ControlActionStatus     `json:"status"`
	BeforeState           []ControlParameterState `json:"before_state"`
	RequestedState        []ControlParameterState `json:"requested_state"`
	VerifiedState         []ControlParameterState `json:"verified_state"`
	LastError             string                  `json:"last_error,omitempty"`
	CreatedAt             time.Time               `json:"created_at"`
	UpdatedAt             time.Time               `json:"updated_at"`
	CompletedAt           *time.Time              `json:"completed_at,omitempty"`
}

func marshalControlState(value any) (json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}
