package storageprotection

import "time"

const stateConfirmations = 2

func validatePolicy(policy *Policy) error {
	if policy == nil {
		return errInvalidPolicy("policy is nil")
	}
	if policy.TargetType == "" || policy.TargetID == "" || policy.WriteScope == "" {
		return errInvalidPolicy("target type, target id and write scope are required")
	}
	if policy.TargetType != TargetFilesystem {
		return errInvalidPolicy("storage protection only supports filesystem targets")
	}
	if policy.WriteScope != WriteScopeAll {
		return errInvalidPolicy("storage protection policies apply to all writes on their target")
	}
	if policy.WarnUsedPercent < 0 || policy.WarnUsedPercent >= policy.RecoverUsedPercent ||
		policy.RecoverUsedPercent >= policy.BlockUsedPercent || policy.BlockUsedPercent > 100 {
		return errInvalidPolicy("thresholds must satisfy 0 <= warn < recover < block <= 100")
	}
	if policy.CheckIntervalSeconds <= 0 {
		return errInvalidPolicy("check interval must be positive")
	}
	if policy.UnknownBehavior != UnknownAllowWithAlarm && policy.UnknownBehavior != UnknownBlockNewWrites {
		return errInvalidPolicy("unknown behavior is invalid")
	}
	return nil
}

func evaluateState(policy *Policy, ratio float64, observedAt time.Time) (State, int, string) {
	previous := policy.CurrentState
	if previous == "" {
		previous = StateNormal
	}
	if ratio >= float64(policy.BlockUsedPercent)/100 {
		if previous == StateBlocked {
			return StateBlocked, 0, "usage remains above block threshold"
		}
		observations := policy.StateObservations + 1
		if observations >= stateConfirmations {
			return StateBlocked, 0, "usage reached block threshold for two checks"
		}
		return StateWarning, observations, "usage reached block threshold; confirmation pending"
	}
	if previous == StateBlocked {
		if ratio <= float64(policy.RecoverUsedPercent)/100 {
			observations := policy.StateObservations + 1
			if observations >= stateConfirmations {
				return StateNormal, 0, "usage recovered below recovery threshold for two checks"
			}
			return StateBlocked, observations, "usage below recovery threshold; recovery confirmation pending"
		}
		return StateBlocked, 0, "usage remains above recovery threshold"
	}
	if previous == StateWarning && ratio >= float64(policy.WarnUsedPercent)/100 {
		// Keep warning latched while usage remains at or above the warning
		// threshold. Re-running the two-check confirmation from StateWarning
		// would alternate warning/normal on every poll and repeatedly toggle
		// the process log admission gate.
		return StateWarning, 0, "usage remains above warning threshold"
	}
	if ratio >= float64(policy.WarnUsedPercent)/100 {
		observations := policy.StateObservations + 1
		if observations >= stateConfirmations {
			return StateWarning, 0, "usage reached warning threshold for two checks"
		}
		return StateNormal, observations, "usage reached warning threshold; confirmation pending"
	}
	return StateNormal, 0, "usage below warning threshold"
}

func transitionReason(previous, next State, fallback string) string {
	if previous == next {
		return fallback
	}
	return string(previous) + " -> " + string(next) + ": " + fallback
}
