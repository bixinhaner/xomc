// Package backup — restore task model (T-0072).
//
// One RestoreTask row per `POST /backup/restore` call. Per-device fan-out is
// represented in TargetDeviceSNs (jsonb in DB); per-device progress is queried
// from the `device_tasks` table (method=Download) at read time, not stored
// here, to avoid double-write inconsistency.
package backup

import (
	"time"

	"github.com/google/uuid"
)

// RestoreStatus mirrors the DB CHECK constraint on restore_tasks.status.
type RestoreStatus string

const (
	RestorePending   RestoreStatus = "pending"
	RestoreRunning   RestoreStatus = "running"
	RestoreCompleted RestoreStatus = "completed"
	RestoreFailed    RestoreStatus = "failed"
	RestoreCancelled RestoreStatus = "cancelled"
)

// RestoreTask is the persisted row backing a restore operation.
type RestoreTask struct {
	ID                uuid.UUID     `json:"id"`
	SourceBucket      string        `json:"source_bucket"`
	SourceObjectPath  string        `json:"source_object_path"`
	TargetDeviceSNs   []string      `json:"target_device_sns"`
	Status            RestoreStatus `json:"status"`
	Progress          int           `json:"progress"`
	ErrorMessage      *string       `json:"error_message,omitempty"`
	StartedAt         *time.Time    `json:"started_at,omitempty"`
	CompletedAt       *time.Time    `json:"completed_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	CreatedBy         *string       `json:"created_by,omitempty"`
}
