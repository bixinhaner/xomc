package nedirect

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SessionStatus represents the state of a NE direct connection session.
type SessionStatus string

const (
	SessionActive       SessionStatus = "active"
	SessionDisconnected SessionStatus = "disconnected"
	SessionTimedOut     SessionStatus = "timed_out"
)

// CommandStatus represents the execution state of a NE direct command.
type CommandStatus string

const (
	CommandPending   CommandStatus = "pending"
	CommandRunning   CommandStatus = "running"
	CommandCompleted CommandStatus = "completed"
	CommandFailed    CommandStatus = "failed"
	CommandTimedOut  CommandStatus = "timed_out"
)

// Session represents an active NE direct connection session between
// a user and a network element device.
type Session struct {
	ID           uuid.UUID     `json:"id"`
	DeviceID     uuid.UUID     `json:"device_id"`
	DeviceSN     string        `json:"device_sn"`
	UserID       string        `json:"user_id"`
	Username     string        `json:"username"`
	Status       SessionStatus `json:"status"`
	IPAddress    string        `json:"ip_address"`
	ConnectedAt  time.Time     `json:"connected_at"`
	LastActiveAt time.Time     `json:"last_active_at"`
	DisconnectAt *time.Time    `json:"disconnect_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Command represents a CLI/MML command sent through a NE direct session.
type Command struct {
	ID         uuid.UUID     `json:"id"`
	SessionID  uuid.UUID     `json:"session_id"`
	DeviceSN   string        `json:"device_sn"`
	CommandStr string        `json:"command"`
	Status     CommandStatus `json:"status"`
	Response   string        `json:"response"`
	ErrorMsg   string        `json:"error_msg,omitempty"`
	SentAt     time.Time     `json:"sent_at"`
	RespondAt  *time.Time    `json:"respond_at,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// SessionFilter specifies criteria for listing NE direct sessions.
type SessionFilter struct {
	DeviceSN *string
	UserID   *string
	Status   *SessionStatus
	model.ListRequest
}

// CommandFilter specifies criteria for listing NE direct commands.
type CommandFilter struct {
	SessionID *uuid.UUID
	DeviceSN  *string
	Status    *CommandStatus
	model.ListRequest
}
