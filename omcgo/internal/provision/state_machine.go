package provision

import "fmt"

// validTransitions defines the allowed state transitions for provisioning tasks.
var validTransitions = map[ProvisioningState][]ProvisioningState{
	StateDiscovered:  {StateIdentifying, StateFailed},
	StateIdentifying: {StateMatching, StateDiscovering, StateSyncing, StateFailed},
	StateMatching:    {StateConfiguring, StateFailed},
	StateConfiguring: {StateVerifying, StateFailed},
	StateVerifying:   {StateCompleted, StateFailed},
	StateDiscovering: {StateSyncing, StateCompleted, StateFailed},
	StateSyncing:     {StateCompleted, StateFailed},
	// Terminal states — no transitions allowed.
	StateCompleted: {},
	StateFailed:    {},
}

// ValidateTransition checks whether a state transition is allowed.
func ValidateTransition(current, target ProvisioningState) error {
	allowed, ok := validTransitions[current]
	if !ok {
		return fmt.Errorf("unknown provisioning state: %s", current)
	}

	for _, s := range allowed {
		if s == target {
			return nil
		}
	}

	return fmt.Errorf("invalid provisioning state transition: %s -> %s", current, target)
}

// IsTerminal returns true if the state is a terminal state (completed or failed).
func IsTerminal(state ProvisioningState) bool {
	return state == StateCompleted || state == StateFailed
}

// NextState returns the expected next state in the happy path.
func NextState(current ProvisioningState) ProvisioningState {
	switch current {
	case StateDiscovered:
		return StateIdentifying
	case StateIdentifying:
		return StateMatching
	case StateMatching:
		return StateConfiguring
	case StateConfiguring:
		return StateVerifying
	case StateVerifying:
		return StateCompleted
	default:
		return current
	}
}
