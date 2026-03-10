package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestValidateTransition_Valid — all 12 valid transitions from the map
// ---------------------------------------------------------------------------

func TestValidateTransition_Valid(t *testing.T) {
	tests := []struct {
		name   string
		from   model.DeviceStatus
		to     model.DeviceStatus
	}{
		// discovered ->
		{"discovered to registered", model.DeviceDiscovered, model.DeviceRegistered},
		{"discovered to active", model.DeviceDiscovered, model.DeviceActive},

		// registered ->
		{"registered to provisioning", model.DeviceRegistered, model.DeviceProvisioning},
		{"registered to active", model.DeviceRegistered, model.DeviceActive},

		// provisioning ->
		{"provisioning to active", model.DeviceProvisioning, model.DeviceActive},
		{"provisioning to registered", model.DeviceProvisioning, model.DeviceRegistered},

		// active ->
		{"active to maintenance", model.DeviceActive, model.DeviceMaintenance},
		{"active to offline", model.DeviceActive, model.DeviceOffline},
		{"active to decommissioned", model.DeviceActive, model.DeviceDecommissioned},

		// maintenance ->
		{"maintenance to active", model.DeviceMaintenance, model.DeviceActive},
		{"maintenance to decommissioned", model.DeviceMaintenance, model.DeviceDecommissioned},

		// offline ->
		{"offline to active", model.DeviceOffline, model.DeviceActive},
		{"offline to decommissioned", model.DeviceOffline, model.DeviceDecommissioned},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTransition(tc.from, tc.to)
			require.NoError(t, err)
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateTransition_Invalid — transitions that should be rejected
// ---------------------------------------------------------------------------

func TestValidateTransition_Invalid(t *testing.T) {
	tests := []struct {
		name string
		from model.DeviceStatus
		to   model.DeviceStatus
	}{
		// discovered cannot jump to decommissioned
		{"discovered to decommissioned", model.DeviceDiscovered, model.DeviceDecommissioned},
		// discovered cannot go to maintenance
		{"discovered to maintenance", model.DeviceDiscovered, model.DeviceMaintenance},
		// active cannot go back to discovered
		{"active to discovered", model.DeviceActive, model.DeviceDiscovered},
		// active cannot go to registered
		{"active to registered", model.DeviceActive, model.DeviceRegistered},
		// registered cannot go to offline
		{"registered to offline", model.DeviceRegistered, model.DeviceOffline},
		// offline cannot go to registered
		{"offline to registered", model.DeviceOffline, model.DeviceRegistered},
		// maintenance cannot go to provisioning
		{"maintenance to provisioning", model.DeviceMaintenance, model.DeviceProvisioning},
		// provisioning cannot go to decommissioned
		{"provisioning to decommissioned", model.DeviceProvisioning, model.DeviceDecommissioned},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTransition(tc.from, tc.to)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid transition")
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateTransition_TerminalState — decommissioned has no outgoing edges
// ---------------------------------------------------------------------------

func TestValidateTransition_TerminalState(t *testing.T) {
	targets := []model.DeviceStatus{
		model.DeviceDiscovered,
		model.DeviceRegistered,
		model.DeviceProvisioning,
		model.DeviceActive,
		model.DeviceMaintenance,
		model.DeviceOffline,
	}

	for _, target := range targets {
		t.Run("decommissioned to "+string(target), func(t *testing.T) {
			err := ValidateTransition(model.DeviceDecommissioned, target)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no transitions defined")
		})
	}
}
