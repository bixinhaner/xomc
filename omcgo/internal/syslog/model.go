package syslog

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// SystemLog represents an application-level log entry.
type SystemLog struct {
	ID        uuid.UUID       `json:"id"`
	Level     string          `json:"level"`
	Source    string          `json:"source,omitempty"`
	Message   string          `json:"message"`
	Details   json.RawMessage `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// NEMessageLog represents a network element communication log entry.
type NEMessageLog struct {
	ID          uuid.UUID  `json:"id"`
	DeviceSN    string     `json:"device_sn"`
	DeviceID    *uuid.UUID `json:"device_id,omitempty"`
	MessageType string     `json:"message_type"`
	Direction   string     `json:"direction"`
	Content     string     `json:"content,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SystemLogFilter specifies criteria for listing system logs.
type SystemLogFilter struct {
	Level     *string
	Source    *string
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// NEMessageLogFilter specifies criteria for listing NE message logs.
type NEMessageLogFilter struct {
	DeviceSN    *string
	DeviceID    *uuid.UUID
	MessageType *string
	Direction   *string
	StartTime   *time.Time
	EndTime     *time.Time
	model.ListRequest
}
