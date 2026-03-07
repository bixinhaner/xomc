package provision

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProvisioningState represents the lifecycle state of a provisioning task.
type ProvisioningState string

const (
	StateDiscovered  ProvisioningState = "discovered"
	StateIdentifying ProvisioningState = "identifying"
	StateMatching    ProvisioningState = "matching"
	StateConfiguring ProvisioningState = "configuring"
	StateVerifying   ProvisioningState = "verifying"
	StateCompleted   ProvisioningState = "completed"
	StateFailed      ProvisioningState = "failed"
)

// ProvisioningTask represents an auto-provisioning workflow for a device.
type ProvisioningTask struct {
	ID           uuid.UUID         `json:"id"`
	DeviceID     uuid.UUID         `json:"device_id"`
	TemplateID   *uuid.UUID        `json:"template_id,omitempty"`
	Status       ProvisioningState `json:"status"`
	CurrentStep  int               `json:"current_step"`
	TotalSteps   int               `json:"total_steps"`
	ErrorMessage string            `json:"error_message,omitempty"`
	RetryCount   int               `json:"retry_count"`
	MaxRetries   int               `json:"max_retries"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ProvisioningStep describes a single step in the provisioning sequence.
type ProvisioningStep struct {
	Order    int             `json:"order"`
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params"`
	Timeout  time.Duration   `json:"timeout"`
	Required bool            `json:"required"`
}

// ProvisioningTaskFilter provides filtering options for listing tasks.
type ProvisioningTaskFilter struct {
	DeviceID *uuid.UUID        `json:"device_id,omitempty"`
	Status   ProvisioningState `json:"status,omitempty"`
	Page     int               `form:"page"`
	PageSize int               `form:"page_size"`
}

// NewProvisioningTask creates a new task in discovered state.
func NewProvisioningTask(deviceID uuid.UUID) *ProvisioningTask {
	return &ProvisioningTask{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		Status:     StateDiscovered,
		MaxRetries: 3,
	}
}
