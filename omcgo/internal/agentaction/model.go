package agentaction

import (
	"time"

	"github.com/google/uuid"
)

const (
	RiskRead = "read"

	ActionDeviceSearch       = "device.search"
	ActionDeviceSummary      = "device.summary"
	ActionAlarmActiveSummary = "alarm.active_summary"
	ActionSystemHealth       = "system.health"
)

type ActionDescriptor struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Risk        string         `json:"risk"`
	Scopes      []string       `json:"scopes"`
}

type SearchActionsRequest struct {
	Query string `json:"query"`
}

type DescribeActionRequest struct {
	ActionID string `json:"actionId" binding:"required"`
}

type ActionRequest struct {
	ActionID string         `json:"actionId" binding:"required"`
	Input    map[string]any `json:"input"`
	DryRun   bool           `json:"dryRun"`
}

type ActionPreview struct {
	ActionID string `json:"actionId"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Risk     string `json:"risk"`
}

type ActionResponse struct {
	ActionID string `json:"actionId"`
	Status   string `json:"status"`
	Result   any    `json:"result"`
}

type DelegationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type IdentityResponse struct {
	ExternalUserID   string         `json:"externalUserId"`
	ExternalUserName string         `json:"externalUserName"`
	Roles            []string       `json:"roles,omitempty"`
	Scopes           []string       `json:"scopes,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type RequestContext struct {
	UserID        uuid.UUID
	Username      string
	IsSuperAdmin  bool
	VisibleGroups []uuid.UUID
}

type DeviceSummary struct {
	ID              uuid.UUID        `json:"id"`
	SerialNumber    string           `json:"serial_number"`
	DeviceName      string           `json:"device_name"`
	ProductClass    string           `json:"product_class"`
	ModelName       string           `json:"model_name"`
	Manufacturer    string           `json:"manufacturer"`
	Technology      string           `json:"technology"`
	LifecycleState  string           `json:"lifecycle_state"`
	IsOnline        bool             `json:"is_online"`
	IPAddress       string           `json:"ip_address"`
	GroupName       string           `json:"group_name,omitempty"`
	LastInformAt    *time.Time       `json:"last_inform_at,omitempty"`
	ParameterSample []ParameterValue `json:"parameter_sample,omitempty"`
}

type ParameterValue struct {
	Path      string `json:"path"`
	Value     string `json:"value"`
	Writable  bool   `json:"writable"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type AlarmRow struct {
	ID              uuid.UUID `json:"id"`
	DeviceID        uuid.UUID `json:"device_id"`
	DeviceSN        string    `json:"device_sn"`
	Severity        int       `json:"severity"`
	AlarmIdentifier string    `json:"alarm_identifier"`
	Description     string    `json:"description"`
	RaisedAt        string    `json:"raised_at"`
	IsRead          bool      `json:"is_read"`
}
