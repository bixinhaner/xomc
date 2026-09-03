package agentbridge

import (
	"encoding/json"
	"time"
)

const ContractVersion = "1.0"

type ResourceRef struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Role  string `json:"role"`
	Label string `json:"label,omitempty"`
}

type PackRef struct {
	Key     string `json:"key"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type ConnectorEventEnvelope struct {
	ContractVersion string         `json:"contractVersion"`
	EventID         string         `json:"eventId"`
	EventType       string         `json:"eventType"`
	Source          string         `json:"source"`
	OccurredAt      time.Time      `json:"occurredAt"`
	TraceID         string         `json:"traceId"`
	IntegrationPack PackRef        `json:"integrationPack"`
	HandbookDigest  string         `json:"handbookDigest"`
	TenantRef       string         `json:"tenantRef,omitempty"`
	Resources       []ResourceRef  `json:"resources"`
	Data            map[string]any `json:"data"`
}

type ToolInvocation struct {
	ContractVersion string        `json:"contractVersion"`
	InvocationID    string        `json:"invocationId"`
	RunID           string        `json:"runId"`
	ScenarioKey     string        `json:"scenarioKey"`
	PackageDigest   string        `json:"packageDigest"`
	HandbookDigest  string        `json:"handbookDigest"`
	OperationID     string        `json:"operationId"`
	Method          string        `json:"method"`
	Path            string        `json:"path"`
	Arguments       ToolArguments `json:"arguments"`
	ResourceScope   []ResourceRef `json:"resourceScope"`
	LeaseToken      string        `json:"leaseToken"`
	LeaseExpiresAt  time.Time     `json:"leaseExpiresAt"`
	DeadlineAt      time.Time     `json:"deadlineAt"`
	TraceID         string        `json:"traceId"`
}

type ToolArguments struct {
	Path  map[string]any `json:"path"`
	Query map[string]any `json:"query"`
	Body  any            `json:"body"`
}

type ToolResult struct {
	ContractVersion string     `json:"contractVersion"`
	LeaseToken      string     `json:"leaseToken"`
	WorkerID        string     `json:"workerId"`
	Status          string     `json:"status"`
	HTTPStatus      int        `json:"httpStatus,omitempty"`
	Output          any        `json:"output,omitempty"`
	Truncated       bool       `json:"truncated"`
	ResultBytes     int        `json:"resultBytes"`
	Error           *ToolError `json:"error"`
	CompletedAt     time.Time  `json:"completedAt"`
	TraceID         string     `json:"traceId"`
}

type ToolError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type AgentFinding struct {
	SchemaVersion    string           `json:"schemaVersion"`
	ScenarioKey      string           `json:"scenarioKey"`
	ScenarioVersion  int              `json:"scenarioVersion"`
	Title            string           `json:"title"`
	Summary          string           `json:"summary"`
	Severity         string           `json:"severity"`
	Confidence       float64          `json:"confidence"`
	Facts            []map[string]any `json:"facts"`
	Hypotheses       []map[string]any `json:"hypotheses"`
	ResourceRefs     []ResourceRef    `json:"resourceRefs"`
	Details          map[string]any   `json:"details"`
	SuggestedActions []map[string]any `json:"suggestedActions"`
	Presentation     map[string]any   `json:"presentation"`
	ExpiresAt        *time.Time       `json:"expiresAt,omitempty"`
}

type FindingDelivery struct {
	ContractVersion string       `json:"contractVersion"`
	DeliveryID      string       `json:"deliveryId"`
	FindingID       string       `json:"findingId"`
	RunID           string       `json:"runId"`
	ScenarioKey     string       `json:"scenarioKey"`
	PackageDigest   string       `json:"packageDigest"`
	HandbookDigest  string       `json:"handbookDigest"`
	Finding         AgentFinding `json:"finding"`
	LeaseToken      string       `json:"leaseToken"`
	LeaseExpiresAt  time.Time    `json:"leaseExpiresAt"`
	TraceID         string       `json:"traceId"`
}

type FindingDeliveryAck struct {
	ContractVersion string     `json:"contractVersion"`
	LeaseToken      string     `json:"leaseToken"`
	WorkerID        string     `json:"workerId"`
	Status          string     `json:"status"`
	LocalFindingID  string     `json:"localFindingId,omitempty"`
	Error           *ToolError `json:"error"`
	CommittedAt     time.Time  `json:"committedAt"`
	TraceID         string     `json:"traceId"`
}

type StoredFinding struct {
	ID               string           `json:"id"`
	DeliveryID       string           `json:"deliveryId"`
	RemoteFindingID  string           `json:"remoteFindingId"`
	RunID            string           `json:"runId"`
	ScenarioKey      string           `json:"scenarioKey"`
	Title            string           `json:"title"`
	Summary          string           `json:"summary"`
	Severity         string           `json:"severity"`
	Confidence       float64          `json:"confidence"`
	Facts            []map[string]any `json:"facts"`
	Hypotheses       []map[string]any `json:"hypotheses"`
	Details          map[string]any   `json:"details"`
	SuggestedActions []map[string]any `json:"suggestedActions"`
	Presentation     map[string]any   `json:"presentation"`
	Resources        []ResourceRef    `json:"resources"`
	Read             bool             `json:"read"`
	Dismissed        bool             `json:"dismissed"`
	CreatedAt        time.Time        `json:"createdAt"`
	ExpiresAt        *time.Time       `json:"expiresAt,omitempty"`
}

type OutboxItem struct {
	ID      string
	EventID string
	Payload json.RawMessage
	Attempt int
}

type ProactiveScenario struct {
	Key                 string         `json:"key"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	Status              string         `json:"status"`
	RolloutMode         string         `json:"rolloutMode"`
	RolloutPercentage   int            `json:"rolloutPercentage"`
	Version             int            `json:"version"`
	EventTypes          []string       `json:"eventTypes"`
	AllowedOperations   []string       `json:"allowedOperations"`
	DeliverySurfaces    []string       `json:"deliverySurfaces"`
	DedupeWindowSeconds int            `json:"dedupeWindowSeconds"`
	RateLimitPerHour    int            `json:"rateLimitPerHour"`
	TimeoutSeconds      int            `json:"timeoutSeconds"`
	LastRunAt           *time.Time     `json:"lastRunAt,omitempty"`
	Stats               map[string]any `json:"stats,omitempty"`
}

type ProactivePackage struct {
	Key       string    `json:"key"`
	Version   string    `json:"version"`
	Digest    string    `json:"digest"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type ProactiveRun struct {
	ID          string     `json:"id"`
	ScenarioKey string     `json:"scenarioKey"`
	Status      string     `json:"status"`
	RolloutMode string     `json:"rolloutMode"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	ErrorCode   string     `json:"errorCode,omitempty"`
}

type ConnectorHealth struct {
	Status          string     `json:"status"`
	WorkerID        string     `json:"workerId,omitempty"`
	HandbookDigest  string     `json:"handbookDigest,omitempty"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt,omitempty"`
	QueueDepth      int        `json:"queueDepth"`
	Message         string     `json:"message,omitempty"`
}

type ProactiveOverview struct {
	Scenarios       []ProactiveScenario `json:"scenarios"`
	Packages        []ProactivePackage  `json:"packages"`
	Runs            []ProactiveRun      `json:"runs"`
	ConnectorHealth ConnectorHealth     `json:"connectorHealth"`
	Stats           map[string]any      `json:"stats"`
}

type ScenarioUpdate struct {
	Status            string `json:"status,omitempty"`
	RolloutMode       string `json:"rolloutMode,omitempty"`
	RolloutPercentage *int   `json:"rolloutPercentage,omitempty"`
}

type ConnectorHeartbeat struct {
	WorkerID       string `json:"workerId"`
	HandbookDigest string `json:"handbookDigest"`
	QueueDepth     int    `json:"queueDepth"`
}
