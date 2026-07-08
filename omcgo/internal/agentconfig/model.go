package agentconfig

import "time"

const (
	Category = "agent"

	KeyEnabled                 = "enabled"
	KeyAgentStudioBaseURL      = "agent_studio_base_url"
	KeyAgentStudioServiceToken = "agent_studio_service_token"
	KeyOMCPublicBaseURL        = "omc_public_base_url"
	KeyConnectorSlug           = "connector_slug"
	KeyConnectorID             = "connector_id"
	KeyRuntimeStreamURL        = "runtime_stream_url"
	KeyStatus                  = "status"
	KeyLastValidatedAt         = "last_validated_at"
	KeyLastError               = "last_error"

	StatusNotConfigured = "not_configured"
	StatusDisabled      = "disabled"
	StatusConnected     = "connected"
	StatusError         = "error"

	DefaultHealthPath         = "/api/v1/agent/health"
	DefaultActionListPath     = "/api/v1/agent-actions/actions"
	DefaultActionSearchPath   = "/api/v1/agent-actions/actions/search"
	DefaultActionDescribePath = "/api/v1/agent-actions/actions/describe"
	DefaultActionPreviewPath  = "/api/v1/agent-actions/actions/preview"
	DefaultActionExecutePath  = "/api/v1/agent-actions/actions/execute"
	DefaultIdentityPath       = "/api/v1/agent-actions/identity"
	DefaultRuntimeStreamPath  = "/api/v1/agent/chat/stream"
)

type AdminConfig struct {
	Enabled                bool   `json:"enabled"`
	AgentStudioBaseURL     string `json:"agentStudioBaseUrl"`
	ServiceTokenConfigured bool   `json:"serviceTokenConfigured"`
	OMCPublicBaseURL       string `json:"omcPublicBaseUrl"`
	ConnectorSlug          string `json:"connectorSlug"`
	ConnectorID            string `json:"connectorId"`
	RuntimeStreamURL       string `json:"runtimeStreamUrl"`
	Status                 string `json:"status"`
	LastValidatedAt        string `json:"lastValidatedAt"`
	LastError              string `json:"lastError"`
	HealthPath             string `json:"healthPath"`
	ActionListPath         string `json:"actionListPath"`
	ActionSearchPath       string `json:"actionSearchPath"`
	ActionDescribePath     string `json:"actionDescribePath"`
	ActionPreviewPath      string `json:"actionPreviewPath"`
	ActionExecutePath      string `json:"actionExecutePath"`
	IdentityPath           string `json:"identityPath"`
}

type RuntimeConfig struct {
	Enabled          bool   `json:"enabled"`
	Endpoint         string `json:"endpoint"`
	ConnectorID      string `json:"connectorId"`
	Status           string `json:"status"`
	LastValidatedAt  string `json:"lastValidatedAt"`
	LastError        string `json:"lastError"`
	ConfiguredSource string `json:"configuredSource"`
}

type RuntimeTarget struct {
	Enabled            bool
	AgentStudioBaseURL string
	ConnectorID        string
	Status             string
	LastError          string
}

type UpdateRequest struct {
	Enabled                 *bool  `json:"enabled"`
	AgentStudioBaseURL      string `json:"agentStudioBaseUrl"`
	AgentStudioServiceToken string `json:"agentStudioServiceToken"`
	OMCPublicBaseURL        string `json:"omcPublicBaseUrl"`
	ConnectorSlug           string `json:"connectorSlug"`
}

type ProvisionResult struct {
	ConnectorID       string `json:"connectorId"`
	Slug              string `json:"slug"`
	Status            string `json:"status"`
	RuntimeStreamPath string `json:"runtimeStreamPath"`
	RuntimeStreamURL  string `json:"runtimeStreamUrl"`
	Detail            string `json:"detail"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checkedAt"`
}
