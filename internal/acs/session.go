package acs

import (
	"encoding/json"
	"fmt"
	"time"
)

// SessionState represents the state of a TR069 session.
type SessionState string

const (
	StateIdle           SessionState = "IDLE"
	StateInformReceived SessionState = "INFORM_RECEIVED"
	StateProcessing     SessionState = "PROCESSING"
	StateRPCPending     SessionState = "RPC_PENDING"
	StateRPCResponse    SessionState = "RPC_RESPONSE"
	StateComplete       SessionState = "COMPLETE"
)

// Session represents an active TR069/CWMP session with a CPE device.
type Session struct {
	DeviceSN     string       `json:"device_sn"`
	State        SessionState `json:"state"`
	LastRPC      string       `json:"last_rpc"`
	InstanceID   string       `json:"instance_id"`
	StartedAt    time.Time    `json:"started_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	InformEvents []string     `json:"inform_events"`
	CWMPId       string       `json:"cwmp_id"`
}

// validTransitions defines the allowed state transitions.
var validTransitions = map[SessionState][]SessionState{
	StateIdle:           {StateInformReceived},
	StateInformReceived: {StateProcessing, StateComplete},
	StateProcessing:     {StateRPCPending, StateComplete},
	StateRPCPending:     {StateRPCResponse},
	StateRPCResponse:    {StateProcessing, StateRPCPending, StateComplete},
	StateComplete:       {StateIdle},
}

// TransitionTo attempts to move the session to a new state.
func (s *Session) TransitionTo(newState SessionState) error {
	allowed, ok := validTransitions[s.State]
	if !ok {
		return fmt.Errorf("no transitions defined for state %s", s.State)
	}

	for _, a := range allowed {
		if a == newState {
			s.State = newState
			s.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("invalid session transition: %s → %s", s.State, newState)
}

// MarshalJSON serializes the session to JSON for Redis storage.
func (s *Session) MarshalJSON() ([]byte, error) {
	type Alias Session
	return json.Marshal((*Alias)(s))
}

// UnmarshalJSON deserializes the session from JSON.
func (s *Session) UnmarshalJSON(data []byte) error {
	type Alias Session
	return json.Unmarshal(data, (*Alias)(s))
}
