package deviceaccess

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ownedIsolationReader interface {
	FindOwnedIsolation(ctx context.Context, deviceID uuid.UUID) (*Action, error)
}

// RFActivationAdmissionReader is the single database-backed gate used by
// downstream location/geofence code before it plans any RF activation.
type RFActivationAdmissionReader struct {
	repository Repository
	actions    ownedIsolationReader
	settings   RuntimeSettingsReader
}

func NewRFActivationAdmissionReader(
	repository Repository,
	actions ownedIsolationReader,
	settings RuntimeSettingsReader,
) *RFActivationAdmissionReader {
	return &RFActivationAdmissionReader{repository: repository, actions: actions, settings: settings}
}

func (r *RFActivationAdmissionReader) AllowRFActivation(
	ctx context.Context,
	carrier, serialNumber string,
	deviceID uuid.UUID,
) (bool, string, error) {
	if r == nil || r.repository == nil || r.actions == nil {
		return false, "device_access_admission_dependency_missing", ErrAccessGateDependencyMissing
	}
	enabled, err := runtimeAccessEnabled(ctx, r.settings, carrier)
	if err != nil {
		return false, "device_access_settings_unavailable", fmt.Errorf("load access settings before RF activation: %w", err)
	}
	if !enabled {
		return true, "device_access_disabled_bypass", nil
	}
	current, err := r.repository.LoadEvaluationContext(ctx, carrier, serialNumber)
	if err != nil {
		return false, "device_access_state_unavailable", fmt.Errorf("load access state before RF activation: %w", err)
	}
	if current.State == nil || current.State.DeviceID == nil || *current.State.DeviceID != deviceID {
		return false, "device_access_identity_mismatch", nil
	}
	if current.State.State != AccessStateAccepted || current.State.NormalTasksFrozen {
		return false, "device_access_not_accepted", nil
	}
	owned, err := r.actions.FindOwnedIsolation(ctx, deviceID)
	if err != nil {
		return false, "device_access_isolation_unavailable", fmt.Errorf("load access-owned RF isolation: %w", err)
	}
	if owned != nil {
		return false, "device_access_owned_isolation", nil
	}
	return true, "device_access_accepted", nil
}
