package software

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// UpgradeState represents the current state of a firmware upgrade task.
type UpgradeState string

const (
	UpgradePending     UpgradeState = "pending"
	UpgradeDownloading UpgradeState = "downloading"
	UpgradeRebooting   UpgradeState = "rebooting"
	UpgradeVerifying   UpgradeState = "verifying"
	UpgradeCompleted   UpgradeState = "completed"
	UpgradeFailed      UpgradeState = "failed"
)

// FirmwareVersion represents a firmware image stored in MinIO.
type FirmwareVersion struct {
	ID            uuid.UUID         `json:"id"`
	Carrier       model.CarrierCode `json:"carrier"`
	ProductClass  string            `json:"product_class"`
	Version       string            `json:"version"`
	FileName      string            `json:"file_name"`
	FileSize      int64             `json:"file_size"`
	MinIOPath     string            `json:"minio_path"`
	CompatibleOUI []string          `json:"compatible_oui"`
	ReleaseNotes  string            `json:"release_notes"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// UpgradeTask represents a single device firmware upgrade operation.
type UpgradeTask struct {
	ID           uuid.UUID    `json:"id"`
	DeviceID     uuid.UUID    `json:"device_id"`
	FirmwareID   uuid.UUID    `json:"firmware_id"`
	BatchID      *uuid.UUID   `json:"batch_id,omitempty"`
	Status       UpgradeState `json:"status"`
	ErrorMessage string       `json:"error_message,omitempty"`
	RetryCount   int          `json:"retry_count"`
	MaxRetries   int          `json:"max_retries"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// FirmwareFilter specifies criteria for listing firmware versions.
type FirmwareFilter struct {
	Carrier      *model.CarrierCode
	ProductClass *string
	model.ListRequest
}

// UpgradeTaskFilter specifies criteria for listing upgrade tasks.
type UpgradeTaskFilter struct {
	DeviceID   *uuid.UUID
	FirmwareID *uuid.UUID
	BatchID    *uuid.UUID
	Status     *UpgradeState
	model.ListRequest
}

// TriggerUpgradeRequest is the JSON body for triggering a single device upgrade.
type TriggerUpgradeRequest struct {
	DeviceID   uuid.UUID `json:"device_id" binding:"required"`
	FirmwareID uuid.UUID `json:"firmware_id" binding:"required"`
}

// BatchUpgradeRequest is the JSON body for triggering a batch upgrade.
type BatchUpgradeRequest struct {
	DeviceIDs   []uuid.UUID `json:"device_ids" binding:"required,min=1"`
	FirmwareID  uuid.UUID   `json:"firmware_id" binding:"required"`
	Concurrency int         `json:"concurrency"`
}
