package geofence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type LifecycleImpact struct {
	GeofenceID          uuid.UUID        `json:"geofence_id"`
	CurrentVersionID    *uuid.UUID       `json:"current_version_id,omitempty"`
	CurrentStatus       DefinitionStatus `json:"current_status"`
	TargetStatus        DefinitionStatus `json:"target_status"`
	BindingCount        int64            `json:"binding_count"`
	DeviceCount         int64            `json:"device_count"`
	ActiveBatchJobCount int64            `json:"active_batch_job_count"`
	PreviewFingerprint  string           `json:"preview_fingerprint"`
}

var ErrStaleLifecyclePreview = commonerrors.NewBusinessError(
	ErrCodeStaleLifecyclePreview,
	"geofence lifecycle preview is stale",
	commonerrors.ErrAlreadyExists,
)

var ErrActiveBatchJobs = commonerrors.NewBusinessError(
	ErrCodeActiveBatchJobs,
	"geofence has active batch jobs",
	commonerrors.ErrAlreadyExists,
)

func validateDefinitionTransition(
	current DefinitionStatus,
	target DefinitionStatus,
) error {
	allowed := (current == DefinitionStatusEnabled &&
		target == DefinitionStatusDisabled) ||
		(current == DefinitionStatusDisabled &&
			target == DefinitionStatusEnabled) ||
		(current == DefinitionStatusDisabled &&
			target == DefinitionStatusArchived)
	if !allowed {
		return fmt.Errorf(
			"geofence transition %s -> %s is not allowed: %w",
			current,
			target,
			commonerrors.ErrInvalidInput,
		)
	}
	return nil
}

func lifecyclePreviewFingerprint(impact LifecycleImpact) string {
	versionID := ""
	if impact.CurrentVersionID != nil {
		versionID = impact.CurrentVersionID.String()
	}
	content := fmt.Sprintf(
		"%s|%s|%s|%s|%d|%d|%d",
		impact.GeofenceID,
		versionID,
		impact.CurrentStatus,
		impact.TargetStatus,
		impact.BindingCount,
		impact.DeviceCount,
		impact.ActiveBatchJobCount,
	)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func validateBindingTransition(
	current BindingStatus,
	target BindingStatus,
) error {
	allowed := (current == BindingStatusActive &&
		target == BindingStatusSuspended) ||
		(current == BindingStatusSuspended &&
			target == BindingStatusActive) ||
		((current == BindingStatusActive ||
			current == BindingStatusSuspended) &&
			target == BindingStatusRemoved)
	if !allowed {
		return fmt.Errorf(
			"geofence binding transition %s -> %s is not allowed: %w",
			current,
			target,
			commonerrors.ErrInvalidInput,
		)
	}
	return nil
}

func validateBindingResumeParent(
	status DefinitionStatus,
	currentVersionID *uuid.UUID,
) error {
	if status != DefinitionStatusEnabled || currentVersionID == nil {
		return fmt.Errorf(
			"parent geofence is not published and enabled: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	return nil
}
