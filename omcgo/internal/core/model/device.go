package model

import (
	"time"

	"github.com/google/uuid"
)

// Device represents a managed base station/small cell.
type Device struct {
	ID                   uuid.UUID              `json:"id" db:"id"`
	SerialNumber         string                 `json:"serial_number" db:"serial_number"`
	OUI                  string                 `json:"oui" db:"oui"`
	ProductClass         string                 `json:"product_class" db:"product_class"`
	Manufacturer         string                 `json:"manufacturer" db:"manufacturer"`
	ModelName            string                 `json:"model_name" db:"model_name"`
	Carrier              CarrierCode            `json:"carrier" db:"carrier"`
	Technology           Technology             `json:"technology" db:"technology"`
	DataModelID          *uuid.UUID             `json:"data_model_id,omitempty" db:"data_model_id"`
	Status               DeviceStatus           `json:"status" db:"status"`
	FirmwareVersion      string                 `json:"firmware_version" db:"firmware_version"`
	IPAddress            string                 `json:"ip_address" db:"ip_address"`
	ConnectionRequestURL          string         `json:"connection_request_url" db:"connection_request_url"`
	NatDetected                   bool           `json:"nat_detected" db:"nat_detected"`
	UDPConnectionRequestAddress   string         `json:"udp_connection_request_address,omitempty" db:"udp_connection_request_address"`
	LastInformAt                  *time.Time     `json:"last_inform_at,omitempty" db:"last_inform_at"`
	LastInformEvents     []string               `json:"last_inform_events,omitempty" db:"last_inform_events"`
	InformInterval       int                    `json:"inform_interval" db:"inform_interval"`
	SiteName             string                 `json:"site_name" db:"site_name"`
	SiteID               string                 `json:"site_id" db:"site_id"`
	Latitude             float64                `json:"latitude" db:"latitude"`
	Longitude            float64                `json:"longitude" db:"longitude"`
	ExtensionData        map[string]interface{} `json:"extension_data,omitempty" db:"extension_data"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt            *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}
