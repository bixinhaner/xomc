package template

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// TemplateType represents the type of configuration template.
type TemplateType string

const (
	TemplateProvisioning  TemplateType = "provisioning"
	TemplateBatchConfig   TemplateType = "batch_config"
	TemplateFirmware      TemplateType = "firmware_upgrade"
)

// ConfigTemplate represents a configuration template stored in the database.
type ConfigTemplate struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Carrier      model.CarrierCode `json:"carrier"`
	Technology   model.Technology  `json:"technology"`
	ProductClass string            `json:"product_class,omitempty"`
	TemplateType TemplateType      `json:"template_type"`
	Parameters   json.RawMessage   `json:"parameters"`
	Priority     int               `json:"priority"`
	Version      int               `json:"version"`
	Active       bool              `json:"active"`
	Description  string            `json:"description,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ConfigTemplateFilter provides filtering options for listing templates.
type ConfigTemplateFilter struct {
	Carrier      model.CarrierCode `json:"carrier,omitempty"`
	Technology   model.Technology  `json:"technology,omitempty"`
	ProductClass string            `json:"product_class,omitempty"`
	TemplateType TemplateType      `json:"template_type,omitempty"`
	Active       *bool             `json:"active,omitempty"`
	model.ListRequest
}
