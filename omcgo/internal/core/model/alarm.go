package model

import (
	"time"

	"github.com/google/uuid"
)

// Alarm represents an alarm event from a managed device.
type Alarm struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	DeviceID       uuid.UUID         `json:"device_id" db:"device_id"`
	DeviceSN       string            `json:"device_sn" db:"device_sn"`
	Carrier        CarrierCode       `json:"carrier" db:"carrier"`
	Severity       AlarmSeverity     `json:"severity" db:"severity"`
	AlarmType      string            `json:"alarm_type" db:"alarm_type"`
	AlarmCode      string            `json:"alarm_code" db:"alarm_code"`
	Description    string            `json:"description" db:"description"`
	Status         AlarmStatus       `json:"status" db:"status"`
	RaisedAt       time.Time         `json:"raised_at" db:"raised_at"`
	AcknowledgedAt *time.Time        `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	ClearedAt      *time.Time        `json:"cleared_at,omitempty" db:"cleared_at"`
	AcknowledgedBy string            `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	AdditionalInfo map[string]string `json:"additional_info,omitempty" db:"additional_info"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}
