package software

import "fmt"

// validUpgradeTransitions defines the allowed state transitions for upgrade tasks.
var validUpgradeTransitions = map[UpgradeState][]UpgradeState{
	UpgradePending:     {UpgradeDownloading, UpgradeFailed, UpgradeTerminated},
	UpgradeDownloading: {UpgradeRebooting, UpgradeCompleted, UpgradeFailed, UpgradeSuspended, UpgradeTerminated},
	UpgradeRebooting:   {UpgradeVerifying, UpgradeFailed, UpgradeSuspended, UpgradeTerminated},
	UpgradeVerifying:   {UpgradeCompleted, UpgradeFailed, UpgradeTerminated},
	UpgradeSuspended:   {UpgradeDownloading, UpgradeRebooting, UpgradeTerminated},
	// Terminal states — no transitions allowed.
	UpgradeCompleted:  {},
	UpgradeFailed:     {},
	UpgradeTerminated: {},
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
	return state == UpgradeCompleted || state == UpgradeFailed || state == UpgradeTerminated
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

// NextStateAfterTC returns the target state after receiving TransferComplete.
// 4G devices complete immediately; 5G devices enter rebooting (wait for 102 UPGRADE FINISH).
func NextStateAfterTC(is5G bool) UpgradeState {
	if is5G {
		return UpgradeRebooting
	}
	return UpgradeCompleted
}
