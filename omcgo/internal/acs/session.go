package acs

import (
	"encoding/json"
	"fmt"
	"time"
)

// SessionState represents the state of a TR069/CWMP session.
// The session follows a state machine pattern that tracks the lifecycle
// of a TR069 conversation between ACS and CPE.
//
// State Machine Diagram:
//
//	                            Inform
//	                              │
//	                              ▼
//	                    ┌─────────────────┐
//	                    │ INFORM_RECEIVED │←─────────────────┐
//	                    └────────┬────────┘                  │
//	                             │                           │
//	                      Empty POST                        │
//	                             │                           │
//	                             ▼                           │
//	                    ┌─────────────────┐                  │
//	     ┌─────────────│   PROCESSING    │←────────┐        │
//	     │             └────────┬────────┘         │        │
//	     │                      │                  │        │
//	     │            ┌─────────┴─────────┐        │        │
//	     │            │                   │        │        │
//	     │       有待发命令            无待发命令    │        │
//	     │            │                   │        │        │
//	     │            ▼                   ▼        │        │
//	     │   ┌─────────────────┐   ┌──────────┐   │        │
//	     │   │   RPC_PENDING   │   │ COMPLETE │   │        │
//	     │   └────────┬────────┘   └──────────┘   │        │
//	     │            │                          │        │
//	     │     CPE Response                      │        │
//	     │            │                          │        │
//	     │            ▼                          │        │
//	     │   ┌─────────────────┐                  │        │
//	     │   │   RPC_RESPONSE  │──────────────────┘        │
//	     │   └────────┬────────┘   有更多命令              │
//	     │            │                                    │
//	     │     ┌──────┴──────┐                             │
//	     │     │             │                             │
//	     │  有更多命令    无更多命令                         │
//	     │     │             │                             │
//	     │     │             ▼                             │
//	     │     │      ┌──────────┐                         │
//	     └─────┴─────→│ COMPLETE │                         │
//	                  └──────────┘                         │
//	                       │                                │
//	                       │ 会话结束                        │
//	                       ▼                                │
//	                  ┌──────────┐                          │
//	                  │   IDLE   │──────────────────────────┘
//	                  └──────────┘     新 Inform 到达
type SessionState string

const (
	// StateIdle indicates no active session. This is the initial state
	// before a CPE sends an Inform, and the final state after session completion.
	// Next: INFORM_RECEIVED (when new Inform arrives)
	StateIdle SessionState = "IDLE"

	// StateInformReceived indicates ACS has received and parsed an Inform from CPE.
	// The session is created, Cookie is set, and device events are published.
	// ACS will send InformResponse and wait for CPE's empty POST.
	// Next: PROCESSING (on empty POST), COMPLETE (on error/session close)
	StateInformReceived SessionState = "INFORM_RECEIVED"

	// StateProcessing indicates CPE has sent an empty POST after InformResponse.
	// ACS checks command queue for pending commands to send to CPE.
	// Next: RPC_PENDING (if commands exist), COMPLETE (if no commands)
	StateProcessing SessionState = "PROCESSING"

	// StateRPCPending indicates ACS has sent an RPC request to CPE and is waiting for response.
	// The LastRPC field is set to the method name of the pending request.
	// Next: RPC_RESPONSE (when CPE responds)
	StateRPCPending SessionState = "RPC_PENDING"

	// StateRPCResponse indicates ACS has received an RPC response from CPE.
	// ACS processes the response and checks for more commands in queue.
	// Next: PROCESSING (if more commands), RPC_PENDING (for chained commands), COMPLETE (if done)
	StateRPCResponse SessionState = "RPC_RESPONSE"

	// StateComplete indicates the TR069 session has ended normally.
	// Resources are released: admission slot freed, metrics recorded, session deleted.
	// ACS sends empty HTTP 204 response to signal CPE that session is closed.
	// Next: IDLE (ready for new session)
	StateComplete SessionState = "COMPLETE"
)

// Session represents an active TR069/CWMP session with a CPE device.
// Sessions are created on Inform and deleted on completion or timeout.
// They are stored in Redis with TTL for automatic cleanup.
type Session struct {
	// ID is the unique session identifier (UUID v4 format), used as Cookie value.
	// Generated on Inform, returned to CPE via Set-Cookie header.
	// CPE must include this in subsequent requests via Cookie header.
	ID string `json:"id"`

	// DeviceSN is the device serial number, extracted from Inform.DeviceId.SerialNumber.
	// Used as the primary key for device lookup and command queue access.
	DeviceSN string `json:"device_sn"`

	// State is the current session state in the state machine.
	State SessionState `json:"state"`

	// LastRPC is the method name of the most recent RPC sent to CPE.
	// Set when entering RPC_PENDING state. Used for logging and debugging.
	// Example: "GetParameterValues", "SetParameterValues", "Reboot"
	LastRPC string `json:"last_rpc"`

	// InstanceID is typically the CPE's RemoteAddr (IP:Port), used for
	// connection-level tracking and logging. May be used for connSessions fallback.
	InstanceID string `json:"instance_id"`

	// StartedAt is when the session was created (Inform received).
	StartedAt time.Time `json:"started_at"`

	// UpdatedAt is when the session state was last modified.
	UpdatedAt time.Time `json:"updated_at"`

	// InformEvents contains the event codes from the Inform message.
	// Example: "0 BOOTSTRAP", "2 PERIODIC", "4 VALUE CHANGE"
	InformEvents []string `json:"inform_events"`

	// CWMPId is the CWMP ID from the Inform's soap:Header, used to correlate
	// requests and responses in the SOAP conversation.
	CWMPId string `json:"cwmp_id"`

	// SessionTimeout is the CPE's suggested session timeout in seconds,
	// typically from Device.ManagementServer.SessionTimeout parameter.
	// ACS may use this to set session TTL in Redis.
	SessionTimeout int `json:"session_timeout"`
}

// validTransitions defines the allowed state transitions.
// This enforces the TR069 session protocol flow and prevents invalid state changes.
var validTransitions = map[SessionState][]SessionState{
	// IDLE can only transition to INFORM_RECEIVED (new session starts)
	StateIdle: {StateInformReceived},

	// INFORM_RECEIVED can transition to:
	// - PROCESSING: CPE sent empty POST, checking for commands
	// - COMPLETE: Session closed prematurely (error, timeout, or no response)
	StateInformReceived: {StateProcessing, StateComplete},

	// PROCESSING can transition to:
	// - RPC_PENDING: Commands found in queue, sending to CPE
	// - COMPLETE: No commands, session ends normally
	StateProcessing: {StateRPCPending, StateComplete},

	// RPC_PENDING can only transition to RPC_RESPONSE (CPE responded)
	StateRPCPending: {StateRPCResponse},

	// RPC_RESPONSE can transition to:
	// - PROCESSING: Check for more commands after processing response
	// - RPC_PENDING: Chain to next command immediately
	// - COMPLETE: No more commands, session ends
	StateRPCResponse: {StateProcessing, StateRPCPending, StateComplete},

	// COMPLETE transitions back to IDLE (ready for new session)
	StateComplete: {StateIdle},
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
