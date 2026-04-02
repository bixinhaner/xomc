package device

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceRegistration represents a pre-registered device entry.
type DeviceRegistration struct {
	ID           uuid.UUID                  `json:"id"`
	SerialNumber string                     `json:"serial_number"`
	GroupID      *uuid.UUID                 `json:"group_id,omitempty"`
	Carrier      model.CarrierCode          `json:"carrier"`
	SiteName     string                     `json:"site_name,omitempty"`
	Longitude    *float64                   `json:"longitude,omitempty"`
	Latitude     *float64                   `json:"latitude,omitempty"`
	Status       global.RegistrationStatus  `json:"status"`
	Remark       string                     `json:"remark,omitempty"`
	CreatedBy    string                     `json:"created_by,omitempty"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

// CreateRegistrationRequest is the input for pre-registering a device.
type CreateRegistrationRequest struct {
	SerialNumber string  `json:"serial_number" binding:"required"`
	GroupID      string  `json:"group_id"`
	Carrier      string  `json:"carrier" binding:"required"`
	SiteName     string  `json:"site_name"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
	Remark       string  `json:"remark"`
}

// RegistrationFilter provides filtering options for listing registrations.
type RegistrationFilter struct {
	Status       *global.RegistrationStatus
	SerialNumber *string
	Carrier      *model.CarrierCode
	model.ListRequest
}

// ImportResult describes the outcome of a batch import.
type ImportResult struct {
	Total    int      `json:"total"`
	Success  int      `json:"success"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}
