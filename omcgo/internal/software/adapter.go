package software

import "github.com/omcgo/omcgo/internal/core/model"

// UpgradeAdapter isolates 4G/5G upgrade and rollback parameter differences
// from the executor logic. Each technology combination may have
// different rollback parameter paths and download file types.
//
// This is a separate interface from carrier.Carrier to avoid modifying
// the core carrier package and all its adapters for this feature.
// In the future these methods could be migrated into carrier.Carrier.
type UpgradeAdapter interface {
	// RollbackParameterPath returns the TR-069 parameter path to trigger a rollback.
	// For 4G/LTE: typically "ROLLBACK_CONTROL".
	// For 5G/NR: typically "Device.SoftwareCtrl.ActivateEnable".
	RollbackParameterPath(tech model.Technology) string

	// RollbackParameterValue returns the value to set for the rollback parameter.
	RollbackParameterValue(tech model.Technology) string

	// RollbackNeedsEnableCheck returns true if the device must first be queried
	// to verify rollback is available before setting the rollback parameter.
	// 4G/LTE typically requires this check; 5G/NR does not.
	RollbackNeedsEnableCheck(tech model.Technology) bool
}

// DefaultUpgradeAdapter provides default 4G/5G upgrade behavior.
// Technology-specific differences can be implemented by creating custom adapters.
type DefaultUpgradeAdapter struct{}

// NewDefaultUpgradeAdapter creates a new DefaultUpgradeAdapter.
func NewDefaultUpgradeAdapter() *DefaultUpgradeAdapter {
	return &DefaultUpgradeAdapter{}
}

func (a *DefaultUpgradeAdapter) RollbackParameterPath(tech model.Technology) string {
	switch tech {
	case model.TechNR:
		return "Device.SoftwareCtrl.ActivateEnable"
	default: // LTE
		return "ROLLBACK_CONTROL"
	}
}

func (a *DefaultUpgradeAdapter) RollbackParameterValue(tech model.Technology) string {
	return "1"
}

func (a *DefaultUpgradeAdapter) RollbackNeedsEnableCheck(tech model.Technology) bool {
	return tech == model.TechLTE
}
