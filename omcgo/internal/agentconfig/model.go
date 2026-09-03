package agentconfig

import "time"

const (
	Category = "agent"

	KeyEnabled                 = "enabled"
	KeyAgentStudioBaseURL      = "agent_studio_base_url"
	KeyAgentStudioServiceToken = "agent_studio_service_token"
	KeyConnectorSlug           = "connector_slug"
	KeyConnectorID             = "connector_id"
	KeyRuntimeStreamURL        = "runtime_stream_url"
	KeyStatus                  = "status"
	KeyLastValidatedAt         = "last_validated_at"
	KeyLastError               = "last_error"
	KeyAllowedMethods          = "allowed_methods"
	KeyBlockedPathPrefixes     = "blocked_path_prefixes"
	KeyToolTimeoutSeconds      = "tool_timeout_seconds"
	KeyMaxResponseBytes        = "max_response_bytes"

	StatusNotConfigured = "not_configured"
	StatusDisabled      = "disabled"
	StatusConnected     = "connected"
	StatusError         = "error"

	DefaultAgentStudioBaseURL = "https://bailey.baicells.com"
	DefaultConnectorSlug      = "local-omc-agent"
	DefaultRuntimeStreamPath  = "/api/v1/agent/chat/stream"

	DefaultAllowedMethods      = "GET"
	DefaultBlockedPathPrefixes = "/api/v1/auth/*\n/api/v1/agent/*\n/api/v1/admin/agent-config*\n/api/v1/admin/sysConfig*"
	DefaultToolTimeoutSeconds  = 30
	DefaultMaxResponseBytes    = 262144

	BasicCategory  = "basic"
	OMCNameKey     = "mrOMCName"
	DefaultOMCName = "OMC 统一网管系统"
)

type RuntimePolicy struct {
	AllowedMethods      []string `json:"allowedMethods"`
	BlockedPathPrefixes []string `json:"blockedPathPrefixes"`
	ToolTimeoutSeconds  int      `json:"toolTimeoutSeconds"`
	MaxResponseBytes    int      `json:"maxResponseBytes"`
}

type AdminConfig struct {
	Enabled                bool          `json:"enabled"`
	AgentStudioBaseURL     string        `json:"agentStudioBaseUrl"`
	ServiceTokenConfigured bool          `json:"serviceTokenConfigured"`
	ConnectorSlug          string        `json:"connectorSlug"`
	ConnectorID            string        `json:"connectorId"`
	RuntimeStreamURL       string        `json:"runtimeStreamUrl"`
	Status                 string        `json:"status"`
	LastValidatedAt        string        `json:"lastValidatedAt"`
	LastError              string        `json:"lastError"`
	Policy                 RuntimePolicy `json:"policy"`
}

type RuntimeConfig struct {
	Visible          bool          `json:"visible"`
	Enabled          bool          `json:"enabled"`
	Endpoint         string        `json:"endpoint"`
	ConnectorID      string        `json:"connectorId"`
	Status           string        `json:"status"`
	LastValidatedAt  string        `json:"lastValidatedAt"`
	LastError        string        `json:"lastError"`
	ConfiguredSource string        `json:"configuredSource"`
	Policy           RuntimePolicy `json:"policy"`
}

type VisibilityConfig struct {
	Visible         bool   `json:"visible"`
	Enabled         bool   `json:"enabled"`
	Status          string `json:"status"`
	LastValidatedAt string `json:"lastValidatedAt"`
	LastError       string `json:"lastError"`
}

type RuntimeTarget struct {
	Enabled                 bool
	AgentStudioBaseURL      string
	AgentStudioServiceToken string
	ConnectorSlug           string
	ConnectorID             string
	Status                  string
	LastError               string
	InstanceName            string
	InstanceNameIsDefault   bool
	Policy                  RuntimePolicy
}

type UpdateRequest struct {
	Enabled                 *bool     `json:"enabled"`
	AgentStudioBaseURL      *string   `json:"agentStudioBaseUrl"`
	AgentStudioServiceToken *string   `json:"agentStudioServiceToken"`
	AllowedMethods          *[]string `json:"allowedMethods"`
	BlockedPathPrefixes     *[]string `json:"blockedPathPrefixes"`
	ToolTimeoutSeconds      *int      `json:"toolTimeoutSeconds"`
	MaxResponseBytes        *int      `json:"maxResponseBytes"`
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
