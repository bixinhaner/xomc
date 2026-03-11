package model

import (
	"time"

	"github.com/google/uuid"
)

// DeviceParameter represents a TR069 parameter value on a specific device.
type DeviceParameter struct {
	DeviceID       uuid.UUID     `json:"device_id" db:"device_id"`
	ParameterPath  string        `json:"parameter_path" db:"parameter_path"`
	ParameterValue string        `json:"parameter_value" db:"parameter_value"`
	ParameterType  ParameterType `json:"parameter_type" db:"parameter_type"`
	Writable       bool          `json:"writable" db:"writable"`
	LastUpdatedAt  time.Time     `json:"last_updated_at" db:"last_updated_at"`
}

// ParameterDefinition describes a parameter in a data model definition.
type ParameterDefinition struct {
	Path         string        `json:"path"`
	Type         ParameterType `json:"type"`
	Writable     bool          `json:"writable"`
	DefaultValue string        `json:"default_value,omitempty"`
	Description  string        `json:"description,omitempty"`
	MinValue     *int64        `json:"min_value,omitempty"`
	MaxValue     *int64        `json:"max_value,omitempty"`
	MaxLength    *int          `json:"max_length,omitempty"`
	EnumValues   []string      `json:"enum_values,omitempty"`
}
