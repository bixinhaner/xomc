package model

import (
	"time"

	"github.com/google/uuid"
)

// PMCounter represents a performance management counter sample.
type PMCounter struct {
	Time         time.Time `json:"time" db:"time"`
	DeviceID     uuid.UUID `json:"device_id" db:"device_id"`
	CellID       string    `json:"cell_id" db:"cell_id"`
	CounterGroup string    `json:"counter_group" db:"counter_group"`
	CounterName  string    `json:"counter_name" db:"counter_name"`
	CounterValue float64   `json:"counter_value" db:"counter_value"`
	Granularity  int       `json:"granularity" db:"granularity"` // minutes
}

// KPIValue represents a computed KPI value at a point in time.
type KPIValue struct {
	Time       time.Time   `json:"time" db:"time"`
	DeviceID   uuid.UUID   `json:"device_id" db:"device_id"`
	CellID     string      `json:"cell_id" db:"cell_id"`
	KPIName    string      `json:"kpi_name" db:"kpi_name"`
	KPIValue   float64     `json:"kpi_value" db:"kpi_value"`
	Carrier    CarrierCode `json:"carrier" db:"carrier"`
	Technology Technology  `json:"technology" db:"technology"`
}

// KPIDefinition describes how a KPI is computed from counters.
type KPIDefinition struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Formula     string      `json:"formula"`
	Unit        string      `json:"unit"`
	Carrier     CarrierCode `json:"carrier"`
	Technology  Technology  `json:"technology"`
	Counters    []string    `json:"counters"` // dependent counter names
}
