package software

import "fmt"

// validUpgradeTransitions defines the allowed state transitions for upgrade tasks.
var validUpgradeTransitions = map[UpgradeState][]UpgradeState{
	UpgradePending:     {UpgradeDownloading, UpgradeFailed},
	UpgradeDownloading: {UpgradeRebooting, UpgradeFailed},
	UpgradeRebooting:   {UpgradeVerifying, UpgradeFailed},
	UpgradeVerifying:   {UpgradeCompleted, UpgradeFailed},
	// Terminal states — no transitions allowed.
	UpgradeCompleted: {},
	UpgradeFailed:    {},
}

// ValidateUpgradeTransition checks whether a state transition is allowed.
func ValidateUpgradeTransition(current, target UpgradeState) error {
	allowed, ok := validUpgradeTransitions[current]
	if !ok {
		return fmt.Errorf("unknown upgrade state: %s", current)
	}

	for _, s := range allowed {
		if s == target {
			return nil
		}
	}

	return fmt.Errorf("invalid upgrade state transition: %s -> %s", current, target)
}

// IsUpgradeTerminal returns true if the state is a terminal state.
func IsUpgradeTerminal(state UpgradeState) bool {
	return state == UpgradeCompleted || state == UpgradeFailed
}

// NextUpgradeState returns the expected next state in the happy path.
func NextUpgradeState(current UpgradeState) UpgradeState {
	switch current {
	case UpgradePending:
		return UpgradeDownloading
	case UpgradeDownloading:
		return UpgradeRebooting
	case UpgradeRebooting:
		return UpgradeVerifying
	case UpgradeVerifying:
		return UpgradeCompleted
	default:
		return current
	}
}
