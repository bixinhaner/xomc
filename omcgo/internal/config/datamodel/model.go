package datamodel

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// DataModelStatus represents the lifecycle state of a data model definition.
type DataModelStatus string

const (
	StatusDraft      DataModelStatus = "draft"
	StatusActive     DataModelStatus = "active"
	StatusDeprecated DataModelStatus = "deprecated"
)

// DataModel represents a TR069 data model definition.
type DataModel struct {
	ID              uuid.UUID            `json:"id"`
	Carrier         model.CarrierCode    `json:"carrier"`
	Technology      model.Technology     `json:"technology"`
	Version         string               `json:"version"`
	OUI             string               `json:"oui,omitempty"`
	ProductClass    string               `json:"product_class,omitempty"`
	Scope           model.DataModelScope `json:"scope"`
	Status          DataModelStatus      `json:"status"`
	IsActive        bool                 `json:"is_active"`
	RootObject      string               `json:"root_object"`
	ParameterTree   json.RawMessage      `json:"parameter_tree"`
	Source          string               `json:"source,omitempty"`
	ImportedBy      string               `json:"imported_by,omitempty"`
	SpecDocumentRef string               `json:"spec_document_ref,omitempty"`
	Description     string               `json:"description,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// Parameter describes a single parameter within a data model.
type Parameter struct {
	Path        string       `json:"path"`
	UnifiedName string       `json:"unified_name,omitempty"`
	Type        string       `json:"type"`
	Writable    bool         `json:"writable"`
	Description string       `json:"description,omitempty"`
	Constraints *Constraints `json:"constraints,omitempty"`
	Category    string       `json:"category,omitempty"`
}

// Constraints defines validation constraints for a parameter.
type Constraints struct {
	MinValue   *int64   `json:"min_value,omitempty"`
	MaxValue   *int64   `json:"max_value,omitempty"`
	EnumValues []string `json:"enum_values,omitempty"`
	Pattern    string   `json:"pattern,omitempty"`
	MaxLength  int      `json:"max_length,omitempty"`
}

// DataModelFilter provides filtering options for listing data models.
type DataModelFilter struct {
	Carrier      model.CarrierCode    `json:"carrier,omitempty"`
	Technology   model.Technology     `json:"technology,omitempty"`
	OUI          string               `json:"oui,omitempty"`
	ProductClass string               `json:"product_class,omitempty"`
	Scope        model.DataModelScope `json:"scope,omitempty"`
	Status       DataModelStatus      `json:"status,omitempty"`
	model.ListRequest
}

// DataModelStats holds aggregate statistics.
type DataModelStats struct {
	Total      int64            `json:"total"`
	Active     int64            `json:"active"`
	Draft      int64            `json:"draft"`
	Deprecated int64            `json:"deprecated"`
	ByCarrier  map[string]int64 `json:"by_carrier"`
	ByScope    map[string]int64 `json:"by_scope"`
}

// ImportLogEntry records a data model lifecycle action.
type ImportLogEntry struct {
	ID            uuid.UUID       `json:"id"`
	DataModelID   uuid.UUID       `json:"data_model_id"`
	Action        string          `json:"action"`
	PerformedBy   string          `json:"performed_by"`
	ChangesSummary json.RawMessage `json:"changes_summary,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// OUIEntry represents an OUI registry record.
type OUIEntry struct {
	OUI          string    `json:"oui"`
	Manufacturer string    `json:"manufacturer"`
	ShortName    string    `json:"short_name"`
	Country      string    `json:"country,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
