package mr

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// MRIndicator represents a measurement report indicator definition.
type MRIndicator struct {
	ID            uuid.UUID  `json:"id"`
	IndicatorName string     `json:"indicator_name"`
	IndicatorCode string     `json:"indicator_code"`
	Description   *string    `json:"description,omitempty"`
	Unit          *string    `json:"unit,omitempty"`
	Category      *string    `json:"category,omitempty"`
	ValueRangeMin *float64   `json:"value_range_min,omitempty"`
	ValueRangeMax *float64   `json:"value_range_max,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// MRDeviceMapping represents a mapping between a device and MR collection configuration.
type MRDeviceMapping struct {
	ID               uuid.UUID  `json:"id"`
	DeviceSN         string     `json:"device_sn"`
	DeviceName       *string    `json:"device_name,omitempty"`
	CellID           string     `json:"cell_id"`
	CellName         *string    `json:"cell_name,omitempty"`
	Enabled          bool       `json:"enabled"`
	SamplingInterval int        `json:"sampling_interval"`
	LastCollectTime  *time.Time `json:"last_collect_time,omitempty"`
	TotalRecords     int64      `json:"total_records"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// IndicatorFilter specifies criteria for listing MR indicators.
type IndicatorFilter struct {
	Category *string
	Keyword  *string
	model.ListRequest
}

// MappingFilter specifies criteria for listing MR device mappings.
type MappingFilter struct {
	DeviceSN *string
	Enabled  *bool
	model.ListRequest
}
