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
	StateDiscovering ProvisioningState = "discovering" // 自动发现参数树
	StateSyncing     ProvisioningState = "syncing"     // 同步设备参数值
	StateCompleted   ProvisioningState = "completed"
	StateFailed      ProvisioningState = "failed"
)

// ProvisioningTask represents an auto-provisioning workflow for a device.
type ProvisioningTask struct {
	ID              uuid.UUID         `json:"id"`
	DeviceID        uuid.UUID         `json:"device_id"`
	SerialNumber    string            `json:"serial_number,omitempty"`
	ProductName     string            `json:"product_name,omitempty"`
	PolicyName      string            `json:"policy_name,omitempty"`
	ExecuteType     string            `json:"execute_type,omitempty"`
	Module          string            `json:"module,omitempty"`
	Technology      string            `json:"technology,omitempty"`
	TemplateID      *uuid.UUID        `json:"template_id,omitempty"`
	PolicyID        *uuid.UUID        `json:"policy_id,omitempty"`
	XMLFileID       *uuid.UUID        `json:"xml_file_id,omitempty"`
	DeviceTaskID    *uuid.UUID        `json:"device_task_id,omitempty"`
	Status          ProvisioningState `json:"status"`
	CurrentStep     int               `json:"current_step"`
	CurrentStepName string            `json:"current_step_name,omitempty"`
	TotalSteps      int               `json:"total_steps"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	RetryCount      int               `json:"retry_count"`
	MaxRetries      int               `json:"max_retries"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
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
	DeviceID      *uuid.UUID        `json:"device_id,omitempty"`
	PolicyID      *uuid.UUID        `json:"policy_id,omitempty"`
	Status        ProvisioningState `json:"status,omitempty"`
	RunningOnly   bool              `json:"running_only,omitempty"`
	Search        string            `json:"search,omitempty"`
	ProductName   string            `json:"product_name,omitempty"`
	Module        string            `json:"module,omitempty"`
	StartedAfter  *time.Time        `json:"started_after,omitempty"`
	StartedBefore *time.Time        `json:"started_before,omitempty"`
	// PolicyOnly excludes bootstrap discovery/model-upload tasks and returns
	// only tasks created by plug-and-play policy execution.
	PolicyOnly bool `json:"policy_only,omitempty"`
	Page       int  `form:"page"`
	PageSize   int  `form:"page_size"`
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

// DiscoveryStatus represents the status of a parameter discovery log entry.
type DiscoveryStatus string

const (
	DiscoveryPending     DiscoveryStatus = "pending"
	DiscoveryDiscovering DiscoveryStatus = "discovering"
	DiscoverySyncing     DiscoveryStatus = "syncing"
	DiscoveryCompleted   DiscoveryStatus = "completed"
	DiscoveryFailed      DiscoveryStatus = "failed"
)

// ParameterDiscoveryLog records the parameter discovery process for a device.
//
// T-0098 P5-02：DataModelID 字段已删除（data_model_id 列从 parameter_discovery_log DROP）。
// 设备发现路径改写 discovered_param_mappings（IntersectService），不再生成 datamodel.DataModel。
type ParameterDiscoveryLog struct {
	ID              uuid.UUID       `json:"id"`
	DeviceID        uuid.UUID       `json:"device_id"`
	DeviceSN        string          `json:"device_sn"`
	OUI             string          `json:"oui"`
	ProductClass    string          `json:"product_class"`
	FirmwareVersion string          `json:"firmware_version"`
	ParameterCount  int             `json:"parameter_count"`
	Status          DiscoveryStatus `json:"status"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewParameterDiscoveryLog creates a new discovery log in pending state.
func NewParameterDiscoveryLog(deviceID uuid.UUID, deviceSN, oui, productClass, firmwareVersion string) *ParameterDiscoveryLog {
	return &ParameterDiscoveryLog{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		DeviceSN:        deviceSN,
		OUI:             oui,
		ProductClass:    productClass,
		FirmwareVersion: firmwareVersion,
		Status:          DiscoveryPending,
	}
}
