package pageconfig

import "time"

// Domain is the northbound page-config data domain.
type Domain string

const (
	DomainCM        Domain = "CM"
	DomainPM        Domain = "PM"
	DomainMR        Domain = "MR"
	DomainLOG       Domain = "LOG"
	DomainInventory Domain = "INVENTORY"
)

// OutputFormat is the generated northbound artifact format.
type OutputFormat string

const (
	FormatCSV OutputFormat = "CSV"
	FormatXML OutputFormat = "XML"
	FormatTXT OutputFormat = "TXT"
)

// Period is a user-facing schedule option. Cron is intentionally not exposed
// by the page-config API.
type Period string

const (
	Period15M Period = "15M"
	Period60M Period = "60M"
	Period24H Period = "24H"
	Period7D  Period = "7D"
	Period1MO Period = "1MO"
)

type CompressionFormat string

const (
	CompressionZip CompressionFormat = "zip"
	CompressionGz  CompressionFormat = "gz"
)

type ProfileStatus string

const (
	StatusNormal     ProfileStatus = "normal"
	StatusTerminated ProfileStatus = "terminated"
)

type ProfileKind string

const (
	ProfileKindFile      ProfileKind = "file"
	ProfileKindInventory ProfileKind = "inventory"
)

type RunStatus string

const (
	RunStatusRunning    RunStatus = "running"
	RunStatusSuccess    RunStatus = "success"
	RunStatusFailed     RunStatus = "failed"
	RunStatusTerminated RunStatus = "terminated"
)

type SupportStatus string

const (
	SupportSupported   SupportStatus = "supported"
	SupportPartial     SupportStatus = "partial"
	SupportUnsupported SupportStatus = "unsupported"
)

type ScenarioObject struct {
	Code    string `json:"code"`
	Tech    string `json:"tech,omitempty"`
	Profile string `json:"profile,omitempty"`
}

type FileGroup struct {
	ID                 string            `json:"id"`
	Domain             Domain            `json:"domain"`
	Format             OutputFormat      `json:"format"`
	Period             Period            `json:"period"`
	StartMinute        int               `json:"start_minute"`
	StartTime          string            `json:"start_time,omitempty"`
	PathTemplate       string            `json:"path_template"`
	FileNameTemplate   string            `json:"file_name_template"`
	CSVSeparator       string            `json:"csv_separator,omitempty"`
	CompressionEnabled bool              `json:"compression_enabled"`
	CompressionFormat  CompressionFormat `json:"compression_format,omitempty"`
	Objects            []ScenarioObject  `json:"objects"`
	// SelectedFields narrows which catalog field keys this group exports.
	// Empty/nil = export all catalog fields for each object (default, backward compatible);
	// non-empty = export only fields whose Key is listed. Keys are domain+object scoped
	// (see FieldDefinition.Key), so a group-level union list partitions correctly per object.
	SelectedFields []string `json:"selected_fields,omitempty"`
}

type FileProfile struct {
	ID             string        `json:"id"`
	Code           string        `json:"code"`
	Name           string        `json:"name"`
	Vendor         string        `json:"vendor"`
	ScenarioName   string        `json:"scenario_name"`
	ScenarioNameEn string        `json:"scenario_name_en"`
	Description    string        `json:"description"`
	Flags          []string      `json:"flags"`
	Enabled        bool          `json:"enabled"`
	Status         ProfileStatus `json:"status"`
	Groups         []FileGroup   `json:"groups"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type FileProfilePreview struct {
	ProfileCode string             `json:"profile_code"`
	Items       []FileGroupPreview `json:"items"`
}

type FileGroupPreview struct {
	GroupID              string            `json:"group_id"`
	Domain               Domain            `json:"domain"`
	Format               OutputFormat      `json:"format"`
	Period               Period            `json:"period"`
	PathTemplate         string            `json:"path_template"`
	FileNameTemplate     string            `json:"file_name_template"`
	CSVSeparator         string            `json:"csv_separator,omitempty"`
	PreviewPath          string            `json:"preview_path"`
	PreviewFileName      string            `json:"preview_file_name"`
	CompressionEnabled   bool              `json:"compression_enabled"`
	CompressionFormat    CompressionFormat `json:"compression_format,omitempty"`
	PreviewArtifactName  string            `json:"preview_artifact_name"`
	UnsupportedTokenKeys []string          `json:"unsupported_token_keys,omitempty"`
	Warnings             []string          `json:"warnings,omitempty"`
}

type UpdateFileProfileRequest struct {
	Name           string        `json:"name"`
	Vendor         string        `json:"vendor"`
	ScenarioName   string        `json:"scenario_name"`
	ScenarioNameEn string        `json:"scenario_name_en"`
	Description    string        `json:"description"`
	Flags          []string      `json:"flags"`
	Enabled        *bool         `json:"enabled"`
	Status         ProfileStatus `json:"status"`
	Groups         []FileGroup   `json:"groups"`
}

type InventoryProfile struct {
	ID                 string                 `json:"id"`
	Code               string                 `json:"code"`
	Name               string                 `json:"name"`
	ObjectCode         string                 `json:"object_code"`
	Tech               string                 `json:"tech"`
	Period             Period                 `json:"period"`
	StartMinute        int                    `json:"start_minute"`
	PathTemplate       string                 `json:"path_template"`
	FileNameTemplate   string                 `json:"file_name_template"`
	CompressionEnabled bool                   `json:"compression_enabled"`
	CompressionFormat  CompressionFormat      `json:"compression_format,omitempty"`
	Enabled            bool                   `json:"enabled"`
	Status             ProfileStatus          `json:"status"`
	Fields             []InventoryFieldConfig `json:"fields"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// InventoryFieldConfig is the persisted field contract for one inventory
// profile. Fields are kept with the profile so edits survive reloads and are
// applied by the inventory generator rather than remaining page-only state.
type InventoryFieldConfig struct {
	Key         string `json:"key"`
	OutputAlias string `json:"output_alias"`
	SystemField string `json:"system_field"`
	Source      string `json:"source"`
	DataType    string `json:"data_type"`
	Renderer    string `json:"renderer"`
	Enabled     bool   `json:"enabled"`
}

type UpdateInventoryProfileRequest struct {
	Name               string                 `json:"name"`
	ObjectCode         string                 `json:"object_code"`
	Tech               string                 `json:"tech"`
	Period             Period                 `json:"period"`
	StartMinute        *int                   `json:"start_minute"`
	PathTemplate       string                 `json:"path_template"`
	FileNameTemplate   string                 `json:"file_name_template"`
	CompressionEnabled *bool                  `json:"compression_enabled"`
	CompressionFormat  CompressionFormat      `json:"compression_format,omitempty"`
	Enabled            *bool                  `json:"enabled"`
	Status             ProfileStatus          `json:"status"`
	Fields             []InventoryFieldConfig `json:"fields"`
}

type FieldDefinition struct {
	Key           string        `json:"key"`
	Domain        Domain        `json:"domain"`
	ObjectCode    string        `json:"object_code"`
	Scope         string        `json:"scope,omitempty"`
	Tech          string        `json:"tech,omitempty"`
	OutputAlias   string        `json:"output_alias"`
	SystemField   string        `json:"system_field"`
	Source        string        `json:"source"`
	DataType      string        `json:"data_type"`
	Renderer      string        `json:"renderer"`
	ProductClass  string        `json:"product_class,omitempty"`
	MetricType    string        `json:"metric_type,omitempty"`
	StatisType    string        `json:"statis_type,omitempty"`
	Unit          string        `json:"unit,omitempty"`
	CnName        string        `json:"cn_name,omitempty"`
	SupportStatus SupportStatus `json:"support_status"`
}

type FieldFilter struct {
	Domain     Domain
	ObjectCode string
	Tech       string
	Profile    string
}

type RunProfileRequest struct {
	GroupID       string     `json:"group_id"`
	WindowStart   *time.Time `json:"window_start"`
	WindowEnd     *time.Time `json:"window_end"`
	Limit         int        `json:"limit"`
	TriggerReason string     `json:"trigger_reason,omitempty"`
}

type RunProfileResponse struct {
	ProfileKind ProfileKind `json:"profile_kind"`
	ProfileCode string      `json:"profile_code"`
	Items       []FileRun   `json:"items"`
	Total       int         `json:"total"`
}

type RunFilter struct {
	ProfileKind      ProfileKind
	ProfileCode      string
	Status           RunStatus
	LatestPerProfile bool
	Limit            int
	Offset           int
}

type RunListResult struct {
	Items  []FileRun `json:"items"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type ResultRetentionPolicy struct {
	RunRetentionDays   int `json:"run_retention_days"`
	EventRetentionDays int `json:"event_retention_days"`
}

type ResultCleanupSummary struct {
	RunsDeleted        int64     `json:"runs_deleted"`
	EventsDeleted      int64     `json:"events_deleted"`
	RunRetentionDays   int       `json:"run_retention_days"`
	EventRetentionDays int       `json:"event_retention_days"`
	RunBefore          time.Time `json:"run_before"`
	EventBefore        time.Time `json:"event_before"`
}

type FileRun struct {
	ID                 string            `json:"id"`
	ProfileKind        ProfileKind       `json:"profile_kind"`
	ProfileCode        string            `json:"profile_code"`
	GroupID            string            `json:"group_id"`
	Domain             Domain            `json:"domain"`
	ObjectCode         string            `json:"object_code"`
	Status             RunStatus         `json:"status"`
	WindowStart        *time.Time        `json:"window_start,omitempty"`
	WindowEnd          *time.Time        `json:"window_end,omitempty"`
	ArtifactPath       string            `json:"artifact_path"`
	ArtifactName       string            `json:"artifact_name"`
	ArtifactContent    string            `json:"artifact_content,omitempty"`
	ArtifactSize       int64             `json:"artifact_size"`
	RowCount           int               `json:"row_count"`
	CompressionEnabled bool              `json:"compression_enabled"`
	CompressionFormat  CompressionFormat `json:"compression_format,omitempty"`
	ErrorMessage       string            `json:"error_message,omitempty"`
	Summary            map[string]any    `json:"summary,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type ExportDataRow map[string]string

type PMMetricQuery struct {
	MetricPaths []string
	Tech        string
	WindowStart *time.Time
	WindowEnd   *time.Time
	Limit       int
}

type DeliveryScope string

const (
	DeliveryScopeFile      DeliveryScope = "file"
	DeliveryScopeInventory DeliveryScope = "inventory"
	DeliveryScopeSocket    DeliveryScope = "socket"
)

type DeliveryProtocol string

const (
	DeliveryProtocolFTP  DeliveryProtocol = "FTP"
	DeliveryProtocolSFTP DeliveryProtocol = "SFTP"
)

type DeliveryAuthMode string

const (
	DeliveryAuthPassword   DeliveryAuthMode = "PASSWORD"
	DeliveryAuthPrivateKey DeliveryAuthMode = "PRIVATE_KEY"
)

type DeliveryHostKeyPolicy string

const (
	DeliveryHostKeyInsecure    DeliveryHostKeyPolicy = "INSECURE"
	DeliveryHostKeyFingerprint DeliveryHostKeyPolicy = "FINGERPRINT"
)

type DeliveryTarget struct {
	ID                 string                `json:"id"`
	Scope              DeliveryScope         `json:"scope"`
	OwnerCode          string                `json:"owner_code"`
	Key                string                `json:"key"`
	Name               string                `json:"name"`
	Enabled            bool                  `json:"enabled"`
	Protocol           DeliveryProtocol      `json:"protocol"`
	Host               string                `json:"host"`
	Port               int                   `json:"port"`
	Username           string                `json:"username"`
	Credential         string                `json:"credential,omitempty"`
	CredentialSet      bool                  `json:"credential_set"`
	AuthMode           DeliveryAuthMode      `json:"auth_mode"`
	RemoteRoot         string                `json:"remote_root"`
	RetryTimes         int                   `json:"retry_times"`
	TimeoutSeconds     int                   `json:"timeout_seconds"`
	PassiveMode        bool                  `json:"passive_mode"`
	HostKeyPolicy      DeliveryHostKeyPolicy `json:"host_key_policy"`
	HostKeyFingerprint string                `json:"host_key_fingerprint,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

type DeliveryTargetFilter struct {
	Scope     DeliveryScope
	OwnerCode string
}

type ReplaceDeliveryTargetsRequest struct {
	Scope     DeliveryScope    `json:"scope"`
	OwnerCode string           `json:"owner_code"`
	Items     []DeliveryTarget `json:"items"`
}

type SNMPAlarmField struct {
	Order      int    `json:"order"`
	Field      string `json:"field"`
	OID        string `json:"oid"`
	Type       string `json:"type"`
	Source     string `json:"source"`
	Required   bool   `json:"required"`
	Definition string `json:"definition,omitempty"`
}

type SNMPAlarmTarget struct {
	ID                  string           `json:"id"`
	Key                 string           `json:"key"`
	Name                string           `json:"name"`
	Enabled             bool             `json:"enabled"`
	Version             string           `json:"version"`
	NotificationType    string           `json:"notification_type"`
	ListenIP            string           `json:"listen_ip"`
	ListenPort          int              `json:"listen_port"`
	TargetHost          string           `json:"target_host"`
	TargetPort          int              `json:"target_port"`
	Community           string           `json:"community,omitempty"`
	CommunitySet        bool             `json:"community_set"`
	SecurityName        string           `json:"security_name"`
	AuthProtocol        string           `json:"auth_protocol,omitempty"`
	AuthCredential      string           `json:"auth_credential,omitempty"`
	AuthCredentialSet   bool             `json:"auth_credential_set"`
	PrivProtocol        string           `json:"priv_protocol,omitempty"`
	PrivCredential      string           `json:"priv_credential,omitempty"`
	PrivCredentialSet   bool             `json:"priv_credential_set"`
	ClearSeverityPolicy string           `json:"clear_severity_policy"`
	MIBQueryEnabled     bool             `json:"mib_query_enabled"`
	TimeoutSeconds      int              `json:"timeout_seconds"`
	Retries             int              `json:"retries"`
	MIBFields           []SNMPAlarmField `json:"mib_fields"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

type SocketAccount struct {
	Key           string `json:"key"`
	Enabled       bool   `json:"enabled"`
	Channel       string `json:"channel"`
	Username      string `json:"username"`
	Type          string `json:"type"`
	Credential    string `json:"credential,omitempty"`
	CredentialSet bool   `json:"credential_set"`
	Purpose       string `json:"purpose"`
}

type SocketAlarmConfig struct {
	ID                  string          `json:"id"`
	Key                 string          `json:"key"`
	Name                string          `json:"name"`
	Enabled             bool            `json:"enabled"`
	Profile             string          `json:"profile"`
	Mode                string          `json:"mode"`
	ListenIP            string          `json:"listen_ip"`
	ListenPort          int             `json:"listen_port"`
	MaxClients          int             `json:"max_clients"`
	RealtimePushEnabled bool            `json:"realtime_push_enabled"`
	ClientSyncEnabled   bool            `json:"client_sync_enabled"`
	HeartbeatSeconds    int             `json:"heartbeat_seconds"`
	HeartbeatTimes      int             `json:"heartbeat_times"`
	IdleTimeoutSeconds  int             `json:"idle_timeout_seconds"`
	Accounts            []SocketAccount `json:"accounts"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type APIConfig struct {
	ID                 string         `json:"id"`
	Key                string         `json:"key"`
	Name               string         `json:"name"`
	Method             string         `json:"method"`
	Path               string         `json:"path"`
	Kind               string         `json:"kind"`
	DataType           string         `json:"data_type"`
	Enabled            bool           `json:"enabled"`
	OldSystemSupported bool           `json:"old_system_supported"`
	CurrentSupported   bool           `json:"current_supported"`
	Source             string         `json:"source"`
	ResponseContract   map[string]any `json:"response_contract"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

type APIClient struct {
	ID             string     `json:"id"`
	ClientKey      string     `json:"client_key"`
	Name           string     `json:"name"`
	Enabled        bool       `json:"enabled"`
	AllowedAPIKeys []string   `json:"allowed_api_keys"`
	IPWhitelist    []string   `json:"ip_whitelist"`
	TokenSecret    string     `json:"token_secret,omitempty"`
	TokenSet       bool       `json:"token_set"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ReplaceAPIClientsRequest struct {
	Items []APIClient `json:"items"`
}

type APIUser struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Enabled     bool      `json:"enabled"`
	Password    string    `json:"password,omitempty"`
	PasswordSet bool      `json:"password_set"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReplaceAPIUsersRequest struct {
	Items []APIUser `json:"items"`
}

type UpdateAPIUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

type APIUserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type APIUserToken struct {
	Token       string    `json:"token"`
	AccessToken string    `json:"access_token"`
	Expires     int       `json:"expires"`
	ExpiresAt   time.Time `json:"expires_at"`
	TokenType   string    `json:"token_type"`
}

type EventArtifactType string

const (
	EventArtifactFile    EventArtifactType = "file"
	EventArtifactMessage EventArtifactType = "message"
	EventArtifactJSON    EventArtifactType = "json"
)

type PageConfigEvent struct {
	ID                 string            `json:"id"`
	Capability         string            `json:"capability"`
	OwnerCode          string            `json:"owner_code"`
	TargetKey          string            `json:"target_key"`
	EventType          string            `json:"event_type"`
	Status             RunStatus         `json:"status"`
	ArtifactType       EventArtifactType `json:"artifact_type"`
	ArtifactName       string            `json:"artifact_name"`
	ArtifactPath       string            `json:"artifact_path"`
	Payload            string            `json:"payload,omitempty"`
	PayloadContentType string            `json:"payload_content_type"`
	ErrorMessage       string            `json:"error_message,omitempty"`
	Summary            map[string]any    `json:"summary,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type EventFilter struct {
	Capability string
	OwnerCode  string
	TargetKey  string
	EventType  string
	Status     RunStatus
	Limit      int
	Offset     int
}

type EventListResult struct {
	Items  []PageConfigEvent `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

type APIInvocationLog struct {
	ID            string    `json:"id"`
	APIKey        string    `json:"api_key"`
	Name          string    `json:"name"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	RequestParams string    `json:"request_params"`
	ResponseBody  string    `json:"response_body"`
	StatusCode    int       `json:"status_code"`
	Status        string    `json:"status"`
	CreateUser    string    `json:"create_user"`
	IPAddress     string    `json:"ip_address"`
	DurationMs    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type APIInvocationLogFilter struct {
	APIKey     string
	Name       string
	Method     string
	Path       string
	Status     string
	CreateUser string
	IPAddress  string
	Keyword    string
	StartTime  string
	EndTime    string
	Limit      int
	Offset     int
}

type APIInvocationLogListResult struct {
	Items  []APIInvocationLog `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type DeliveryConnectionResult struct {
	Success            bool             `json:"success"`
	TCPReachable       bool             `json:"tcp_reachable"`
	AuthProbeSupported bool             `json:"auth_probe_supported"`
	AuthProbePassed    *bool            `json:"auth_probe_passed,omitempty"`
	LatencyMs          int64            `json:"latency_ms"`
	Message            string           `json:"message"`
	Protocol           DeliveryProtocol `json:"protocol"`
	Host               string           `json:"host"`
	Port               int              `json:"port"`
}

type DeliveryUploadResult struct {
	Success     bool             `json:"success"`
	Protocol    DeliveryProtocol `json:"protocol"`
	Host        string           `json:"host"`
	Port        int              `json:"port"`
	RemotePath  string           `json:"remote_path"`
	Attempts    int              `json:"attempts"`
	LatencyMs   int64            `json:"latency_ms"`
	Bytes       int64            `json:"bytes"`
	Message     string           `json:"message"`
	Error       string           `json:"error,omitempty"`
	RunID       string           `json:"run_id"`
	ProfileKind ProfileKind      `json:"profile_kind"`
	ProfileCode string           `json:"profile_code"`
}

type SNMPSendResult struct {
	Success          bool   `json:"success"`
	AlarmID          string `json:"alarm_id"`
	TargetKey        string `json:"target_key"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Version          string `json:"version"`
	NotificationType string `json:"notification_type"`
	VarbindCount     int    `json:"varbind_count"`
	LatencyMs        int64  `json:"latency_ms"`
	Message          string `json:"message"`
	Error            string `json:"error,omitempty"`
}

type Overview struct {
	FileProfiles      int      `json:"file_profiles"`
	InventoryProfiles int      `json:"inventory_profiles"`
	FieldDefinitions  int      `json:"field_definitions"`
	SupportedDomains  []Domain `json:"supported_domains"`
	SupportedPeriods  []Period `json:"supported_periods"`
}

type ValidateRequest struct {
	ProfileKind        string            `json:"profile_kind"`
	Domain             Domain            `json:"domain"`
	Format             OutputFormat      `json:"format"`
	Period             Period            `json:"period"`
	CompressionEnabled bool              `json:"compression_enabled"`
	CompressionFormat  CompressionFormat `json:"compression_format"`
	Objects            []ScenarioObject  `json:"objects"`
	Fields             []string          `json:"fields"`
	MetricPaths        []string          `json:"metric_paths"`
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}
