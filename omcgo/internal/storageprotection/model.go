package storageprotection

import "time"

type TargetType string

const (
	TargetFilesystem TargetType = "filesystem"
	// The remaining target types are kept for wire compatibility with older
	// clients. New policies must use the single physical filesystem target
	// defined below; logical component targets are not independent disks.
	TargetMinIO       TargetType = "minio"
	TargetDatabase    TargetType = "database"
	TargetRedis       TargetType = "redis"
	TargetNATS        TargetType = "nats"
	TargetMonitoring  TargetType = "monitoring"
	TargetApplication TargetType = "application"
)

const (
	// UnifiedStorageTargetID is the canonical storage-protection target for
	// policy compatibility. Thresholds remain global and are stored on this
	// target, but runtime evaluation expands it to every protected host
	// mountpoint that contains an OMC write path.
	UnifiedStorageTargetID   = "root"
	UnifiedStorageMountpoint = "/"

	DefaultPolicyID = "27800000-0000-4000-8000-000000000001"
)

type WriteScope string

const (
	WriteScopeUpload WriteScope = "upload"
	WriteScopeLog    WriteScope = "log"
	WriteScopeBackup WriteScope = "backup"
	WriteScopeReport WriteScope = "report"
	WriteScopeTrace  WriteScope = "trace"
	WriteScopePM     WriteScope = "pm"
	WriteScopeMR     WriteScope = "mr"
	WriteScopeAll    WriteScope = "all"
)

type UnknownBehavior string

const (
	UnknownAllowWithAlarm UnknownBehavior = "allow_with_alarm"
	UnknownBlockNewWrites UnknownBehavior = "block_new_uploads"
)

type State string

const (
	StateNormal  State = "normal"
	StateWarning State = "warning"
	StateBlocked State = "blocked"
	StateUnknown State = "unknown"
)

type Policy struct {
	ID                   string          `json:"id"`
	TargetType           TargetType      `json:"target_type"`
	TargetID             string          `json:"target_id"`
	WriteScope           WriteScope      `json:"write_scope"`
	Enabled              bool            `json:"enabled"`
	WarnUsedPercent      int             `json:"warn_used_percent"`
	BlockUsedPercent     int             `json:"block_used_percent"`
	RecoverUsedPercent   int             `json:"recover_used_percent"`
	CheckIntervalSeconds int             `json:"check_interval_seconds"`
	UnknownBehavior      UnknownBehavior `json:"unknown_behavior"`
	CurrentState         State           `json:"current_state"`
	StateObservations    int             `json:"state_observations"`
	LastObservedRatio    *float64        `json:"last_observed_ratio,omitempty"`
	LastObservedAt       *time.Time      `json:"last_observed_at,omitempty"`
	LastStateChangedAt   *time.Time      `json:"last_state_changed_at,omitempty"`
	UpdatedBy            string          `json:"updated_by"`
	Version              int64           `json:"version"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

func DefaultPolicy() Policy {
	return Policy{
		ID:                   DefaultPolicyID,
		TargetType:           TargetFilesystem,
		TargetID:             UnifiedStorageTargetID,
		WriteScope:           WriteScopeAll,
		Enabled:              true,
		WarnUsedPercent:      80,
		RecoverUsedPercent:   85,
		BlockUsedPercent:     90,
		CheckIntervalSeconds: 30,
		UnknownBehavior:      UnknownAllowWithAlarm,
		CurrentState:         StateNormal,
		StateObservations:    0,
		UpdatedBy:            "system",
		Version:              1,
	}
}

type UsageSnapshot struct {
	TargetType     TargetType `json:"target_type"`
	TargetID       string     `json:"target_id"`
	Mountpoint     string     `json:"mountpoint,omitempty"`
	ProtectedPaths []string   `json:"protected_paths,omitempty"`
	CapacityBytes  uint64     `json:"capacity_bytes"`
	UsedBytes      uint64     `json:"used_bytes"`
	UsedRatio      float64    `json:"used_ratio"`
	ObservedAt     time.Time  `json:"observed_at"`
	Available      bool       `json:"available"`
	Reason         string     `json:"reason,omitempty"`
}

type TargetSnapshot struct {
	TargetType     TargetType   `json:"target_type"`
	TargetID       string       `json:"target_id"`
	Mountpoint     string       `json:"mountpoint,omitempty"`
	ProtectedPaths []string     `json:"protected_paths,omitempty"`
	CapacityBytes  uint64       `json:"capacity_bytes"`
	UsedBytes      uint64       `json:"used_bytes"`
	UsedRatio      float64      `json:"used_ratio"`
	Available      bool         `json:"available"`
	Reason         string       `json:"reason,omitempty"`
	ObservedAt     time.Time    `json:"observed_at"`
	CurrentState   State        `json:"current_state"`
	WriteScopes    []WriteScope `json:"write_scopes"`
}

type AdmissionDecision struct {
	Allowed    bool          `json:"allowed"`
	State      State         `json:"state"`
	Reason     string        `json:"reason"`
	RetryAfter time.Duration `json:"-"`
	ObservedAt time.Time     `json:"observed_at"`
}

type Event struct {
	PolicyID      string     `json:"policy_id"`
	TargetType    TargetType `json:"target_type"`
	TargetID      string     `json:"target_id"`
	WriteScope    WriteScope `json:"write_scope"`
	PreviousState State      `json:"previous_state,omitempty"`
	NewState      State      `json:"new_state"`
	Reason        string     `json:"reason"`
	ObservedRatio *float64   `json:"observed_ratio,omitempty"`
	PolicyVersion int64      `json:"policy_version"`
	OperatorID    string     `json:"operator_id"`
	CreatedAt     time.Time  `json:"created_at"`
}
