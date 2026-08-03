// Package paramsync owns the durable parameter-synchronisation request/run/result lifecycle.
package paramsync

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
)

type RequestStatus string

const (
	RequestStatusAccepted     RequestStatus = "accepted"
	RequestStatusQueued       RequestStatus = "queued"
	RequestStatusRunning      RequestStatus = "running"
	RequestStatusSucceeded    RequestStatus = "succeeded"
	RequestStatusFailed       RequestStatus = "failed"
	RequestStatusTimedOut     RequestStatus = "timed_out"
	RequestStatusCancelled    RequestStatus = "cancelled"
	RequestStatusDeduplicated RequestStatus = "deduplicated"
	RequestStatusRejected     RequestStatus = "rejected"
)

func (s RequestStatus) Terminal() bool {
	switch s {
	case RequestStatusSucceeded, RequestStatusFailed, RequestStatusTimedOut,
		RequestStatusCancelled, RequestStatusDeduplicated, RequestStatusRejected:
		return true
	default:
		return false
	}
}

type RunStatus string

const (
	RunStatusPlanning      RunStatus = "planning"
	RunStatusEnqueuing     RunStatus = "enqueuing"
	RunStatusWaitingDevice RunStatus = "waiting_device"
	RunStatusExecuting     RunStatus = "executing"
	RunStatusProcessing    RunStatus = "processing"
	RunStatusCancelling    RunStatus = "cancelling"
	RunStatusSucceeded     RunStatus = "succeeded"
	RunStatusFailed        RunStatus = "failed"
	RunStatusCancelled     RunStatus = "cancelled"
)

func (s RunStatus) Terminal() bool {
	return s == RunStatusSucceeded || s == RunStatusFailed || s == RunStatusCancelled
}

func (s RunStatus) AcceptsTaskResults() bool {
	return !s.Terminal()
}

// AcceptsRecovery is deliberately stricter than AcceptsTaskResults. A
// cancelling run still consumes terminal results to converge, but must never
// extend its task plan after the cancellation sweep has started.
func (s RunStatus) AcceptsRecovery() bool {
	return s == RunStatusWaitingDevice || s == RunStatusExecuting
}

type SyncScope string

const (
	SyncScopeFull        SyncScope = "full"
	SyncScopePartial     SyncScope = "partial"
	SyncScopeReadback    SyncScope = "readback"
	SyncScopePolicyProbe SyncScope = "policy_probe"
)

func (s SyncScope) IsFull() bool { return s == SyncScopeFull }

type TriggerReason string

const (
	TriggerBootstrap         TriggerReason = "bootstrap"
	TriggerModelUpload       TriggerReason = "model_upload"
	TriggerDeviceOnline      TriggerReason = "device_online"
	TriggerFirmwareChanged   TriggerReason = "firmware_changed"
	TriggerPeriodic          TriggerReason = "periodic"
	TriggerManual            TriggerReason = "manual"
	TriggerConfigPull        TriggerReason = "config_pull"
	TriggerLicense           TriggerReason = "license"
	TriggerSPVReadback       TriggerReason = "spv_readback"
	TriggerAddObjectReadback TriggerReason = "add_object_readback"
	TriggerInformPeriodProbe TriggerReason = "inform_period_probe"
	TriggerDeviceRegistered  TriggerReason = "device_registered"
	TriggerOMCUpgrade        TriggerReason = "omc_upgrade"
)

func (r TriggerReason) Valid() bool {
	switch r {
	case TriggerBootstrap, TriggerModelUpload, TriggerDeviceOnline, TriggerFirmwareChanged,
		TriggerPeriodic, TriggerManual, TriggerConfigPull, TriggerLicense,
		TriggerSPVReadback, TriggerAddObjectReadback, TriggerInformPeriodProbe,
		TriggerDeviceRegistered, TriggerOMCUpgrade:
		return true
	default:
		return false
	}
}

type ResultCode string

const (
	ResultCodeOK                     ResultCode = "OK"
	ResultCodeActiveSyncExists       ResultCode = "ACTIVE_SYNC_EXISTS"
	ResultCodePathBUnavailable       ResultCode = "PATH_B_UNAVAILABLE"
	ResultCodeNoStorablePath         ResultCode = "NO_STORABLE_PATH"
	ResultCodeTaskFailed             ResultCode = "TASK_FAILED"
	ResultCodeTaskExpired            ResultCode = "TASK_EXPIRED"
	ResultCodeResultProcessingFailed ResultCode = "RESULT_PROCESSING_FAILED"
	ResultCodeDeadlineExceeded       ResultCode = "DEADLINE_EXCEEDED"
	ResultCodeAutomaticBackoff       ResultCode = "AUTOMATIC_BACKOFF"
	ResultCodeAutomaticBackpressure  ResultCode = "AUTOMATIC_BACKPRESSURE"
	ResultCodeSupersededRelease      ResultCode = "SUPERSEDED_RELEASE_CAMPAIGN"
)

func (r TriggerReason) Automatic() bool {
	switch r {
	case TriggerModelUpload, TriggerDeviceOnline, TriggerPeriodic, TriggerFirmwareChanged,
		TriggerDeviceRegistered, TriggerOMCUpgrade:
		return true
	default:
		return false
	}
}

type SyncRequest struct {
	ID                      uuid.UUID       `json:"id"`
	DeviceID                uuid.UUID       `json:"device_id"`
	DeviceSN                string          `json:"device_sn"`
	CallerType              string          `json:"caller_type"`
	TriggerReason           TriggerReason   `json:"trigger_reason"`
	SyncScope               SyncScope       `json:"sync_scope"`
	RequestedPaths          []string        `json:"requested_paths"`
	Status                  RequestStatus   `json:"status"`
	RunID                   *uuid.UUID      `json:"run_id,omitempty"`
	ActiveRunID             *uuid.UUID      `json:"active_run_id,omitempty"`
	Priority                int             `json:"priority"`
	NextAttemptAt           time.Time       `json:"next_attempt_at"`
	DeadlineAt              *time.Time      `json:"deadline_at,omitempty"`
	IdempotencyKey          *string         `json:"idempotency_key,omitempty"`
	ResultCode              ResultCode      `json:"result_code,omitempty"`
	ResultSummary           json.RawMessage `json:"result_summary,omitempty"`
	ErrorMessage            string          `json:"error_message,omitempty"`
	CampaignID              *uuid.UUID      `json:"campaign_id,omitempty"`
	SourceEventID           *string         `json:"source_event_id,omitempty"`
	OriginEventType         *string         `json:"origin_event_type,omitempty"`
	ModelUploadIntentID     *uuid.UUID      `json:"model_upload_intent_id,omitempty"`
	ModelUploadStatus       *string         `json:"model_upload_status,omitempty"`
	AdmissionClass          *string         `json:"admission_class,omitempty"`
	AdmissionReason         string          `json:"admission_reason,omitempty"`
	AdmissionSnapshot       json.RawMessage `json:"admission_snapshot,omitempty"`
	AdmissionQueuedAt       *time.Time      `json:"admission_queued_at,omitempty"`
	DeduplicatedToRequestID *uuid.UUID      `json:"deduplicated_to_request_id,omitempty"`
	CreatedAt               time.Time       `json:"created_at"`
	StartedAt               *time.Time      `json:"started_at,omitempty"`
	CompletedAt             *time.Time      `json:"completed_at,omitempty"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

type SyncRun struct {
	ID                 uuid.UUID       `json:"id"`
	RequestID          uuid.UUID       `json:"request_id"`
	DeviceID           uuid.UUID       `json:"device_id"`
	DeviceSN           string          `json:"device_sn"`
	TriggerReason      TriggerReason   `json:"trigger_reason"`
	SyncScope          SyncScope       `json:"sync_scope"`
	MappingSource      string          `json:"mapping_source,omitempty"`
	MappingVersion     string          `json:"mapping_version,omitempty"`
	Coverage           []CoverageScope `json:"coverage,omitempty"`
	Status             RunStatus       `json:"status"`
	ExpectedTaskCount  int             `json:"expected_task_count"`
	TerminalTaskCount  int             `json:"terminal_task_count"`
	ProcessedTaskCount int             `json:"processed_task_count"`
	FailedTaskCount    int             `json:"failed_task_count"`
	ErrorMessage       string          `json:"error_message,omitempty"`
	StartedAt          time.Time       `json:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
	Version            int64           `json:"version"`
}

func (r SyncRun) ReadyToFinalize() bool {
	return r.ExpectedTaskCount >= 0 &&
		r.TerminalTaskCount == r.ExpectedTaskCount &&
		r.ProcessedTaskCount == r.ExpectedTaskCount
}

type TaskResult struct {
	RunID        uuid.UUID  `json:"run_id"`
	TaskID       string     `json:"task_id"`
	EventID      string     `json:"event_id"`
	Success      bool       `json:"success"`
	ResultRef    string     `json:"result_ref,omitempty"`
	Status       string     `json:"status"`
	ErrorCode    string     `json:"error_code,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type StartResult struct {
	Request      *SyncRequest `json:"request"`
	Run          *SyncRun     `json:"run,omitempty"`
	Deduplicated bool         `json:"deduplicated"`
	ActiveRunID  *uuid.UUID   `json:"active_run_id,omitempty"`
	ActiveRun    *SyncRun     `json:"-"`
}

type SubmitCommand struct {
	DeviceID            uuid.UUID
	DeviceSN            string
	CallerType          string
	TriggerReason       TriggerReason
	Scope               SyncScope
	RequestedPaths      []string
	Priority            int
	DeadlineAt          *time.Time
	IdempotencyKey      string
	CampaignID          *uuid.UUID
	SourceEventID       string
	OriginEventType     string
	ModelUploadIntentID *uuid.UUID
	ModelUploadStatus   string
}

type SubmitResult struct {
	RequestID     uuid.UUID     `json:"request_id"`
	RunID         *uuid.UUID    `json:"run_id,omitempty"`
	Status        RequestStatus `json:"status"`
	ResultCode    ResultCode    `json:"result_code,omitempty"`
	Scope         SyncScope     `json:"scope"`
	TriggerReason TriggerReason `json:"trigger_reason"`
	TaskCount     int           `json:"task_count"`
	ActiveRunID   *uuid.UUID    `json:"active_run_id,omitempty"`
}

type FeatureFlags struct {
	RunEnabled            bool `mapstructure:"run_enabled" json:"run_enabled"`
	ResultConsumerEnabled bool `mapstructure:"result_consumer_enabled" json:"result_consumer_enabled"`
	StagingEnabled        bool `mapstructure:"staging_enabled" json:"staging_enabled"`
	CanaryPercent         int  `mapstructure:"canary_percent" json:"canary_percent"`
	LegacyFallbackEnabled bool `mapstructure:"legacy_fallback_enabled" json:"legacy_fallback_enabled"`
}

func (f FeatureFlags) Validate() error {
	if f.CanaryPercent < 0 || f.CanaryPercent > 100 {
		return fmt.Errorf("param_sync.canary_percent must be between 0 and 100, got %d", f.CanaryPercent)
	}
	if !f.RunEnabled {
		if f.ResultConsumerEnabled || f.StagingEnabled || f.CanaryPercent != 0 {
			return fmt.Errorf("param_sync consumer, staging, and canary settings require run_enabled=true")
		}
		return nil
	}
	if !f.ResultConsumerEnabled {
		return fmt.Errorf("param_sync.result_consumer_enabled must be true when run_enabled=true")
	}
	if !f.StagingEnabled {
		return fmt.Errorf("param_sync.staging_enabled must be true when run_enabled=true")
	}
	if f.CanaryPercent <= 0 {
		return fmt.Errorf("param_sync.canary_percent must be greater than 0 when run_enabled=true")
	}
	return nil
}

func (f FeatureFlags) EnabledForDevice(deviceID string) bool {
	if !f.RunEnabled || f.CanaryPercent <= 0 {
		return false
	}
	if f.CanaryPercent >= 100 {
		return true
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(deviceID))
	return int(h.Sum64()%100) < f.CanaryPercent
}
