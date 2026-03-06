package pm

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// KPIThreshold represents a configurable KPI threshold for alarm generation.
type KPIThreshold struct {
	ID                uuid.UUID `json:"id"`
	KPIName           string    `json:"kpi_name"`
	Carrier           string    `json:"carrier,omitempty"`
	Technology        string    `json:"technology,omitempty"`
	WarningThreshold  *float64  `json:"warning_threshold,omitempty"`
	MinorThreshold    *float64  `json:"minor_threshold,omitempty"`
	MajorThreshold    *float64  `json:"major_threshold,omitempty"`
	CriticalThreshold *float64  `json:"critical_threshold,omitempty"`
	Comparison        string    `json:"comparison"`
	Enabled           bool      `json:"enabled"`
	Description       string    `json:"description,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// KPIThresholdFilter specifies criteria for listing KPI thresholds.
type KPIThresholdFilter struct {
	KPIName    *string
	Carrier    *string
	Technology *string
	Enabled    *bool
	model.ListRequest
}

// CreateKPIThresholdRequest is the JSON body for creating a KPI threshold.
type CreateKPIThresholdRequest struct {
	KPIName           string   `json:"kpi_name" binding:"required"`
	Carrier           string   `json:"carrier"`
	Technology        string   `json:"technology"`
	WarningThreshold  *float64 `json:"warning_threshold"`
	MinorThreshold    *float64 `json:"minor_threshold"`
	MajorThreshold    *float64 `json:"major_threshold"`
	CriticalThreshold *float64 `json:"critical_threshold"`
	Comparison        string   `json:"comparison"`
	Enabled           *bool    `json:"enabled"`
	Description       string   `json:"description"`
}

// UpdateKPIThresholdRequest is the JSON body for updating a KPI threshold.
type UpdateKPIThresholdRequest struct {
	KPIName           *string  `json:"kpi_name"`
	Carrier           *string  `json:"carrier"`
	Technology        *string  `json:"technology"`
	WarningThreshold  *float64 `json:"warning_threshold"`
	MinorThreshold    *float64 `json:"minor_threshold"`
	MajorThreshold    *float64 `json:"major_threshold"`
	CriticalThreshold *float64 `json:"critical_threshold"`
	Comparison        *string  `json:"comparison"`
	Enabled           *bool    `json:"enabled"`
	Description       *string  `json:"description"`
}
