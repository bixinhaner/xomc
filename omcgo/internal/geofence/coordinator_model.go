package geofence

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
)

type ActionLevel string

const (
	ActionLevelNone         ActionLevel = "none"
	ActionLevelNotifyOnly   ActionLevel = "notify_only"
	ActionLevelManualReview ActionLevel = "manual_review"
	ActionLevelDeactivate   ActionLevel = "deactivate"
)

type EffectiveState string

const (
	EffectiveStateUnmanaged EffectiveState = "unmanaged"
	EffectiveStateUnknown   EffectiveState = "unknown"
	EffectiveStateInside    EffectiveState = "inside"
	EffectiveStateOutside   EffectiveState = "outside"
)

type BindingEvaluation struct {
	BindingID      uuid.UUID
	ConfirmedState ConfirmedState
	ActionLevel    ActionLevel
}

type EffectiveSnapshot struct {
	State               EffectiveState
	RequiredActionLevel ActionLevel
	TriggerBindingID    *uuid.UUID
}

type evaluationPolicyJSON struct {
	ExitAction                  string   `json:"exit_action"`
	ExitToleranceMeters         *float64 `json:"exit_tolerance_meters"`
	ReentryToleranceMeters      *float64 `json:"reentry_tolerance_meters"`
	ExitConsecutiveSamples      *int     `json:"exit_consecutive_samples"`
	ReentryConsecutiveSamples   *int     `json:"reentry_consecutive_samples"`
	SampleMaxAgeSeconds         *int     `json:"sample_max_age_seconds"`
	MaxJumpDistanceMeters       *float64 `json:"max_jump_distance_meters"`
	MaxImpliedSpeedMPS          *float64 `json:"max_implied_speed_mps"`
	MaxDeviceClockSkewSeconds   *int     `json:"max_device_clock_skew_seconds"`
	MinimumStateDurationSeconds *int     `json:"minimum_state_duration_seconds"`
}

func decodeEvaluationPolicy(
	raw json.RawMessage,
) (EvaluationPolicy, ActionLevel, error) {
	defaults := EvaluationPolicy{
		ExitToleranceMeters:         20,
		ReentryToleranceMeters:      20,
		ExitConsecutiveSamples:      3,
		ReentryConsecutiveSamples:   3,
		SampleMaxAgeSeconds:         300,
		MaxJumpDistanceMeters:       5000,
		MaxImpliedSpeedMPS:          55,
		MaxDeviceClockSkewSeconds:   300,
		MinimumStateDurationSeconds: 60,
	}
	var wire evaluationPolicyJSON
	if len(raw) == 0 || json.Unmarshal(raw, &wire) != nil {
		return EvaluationPolicy{}, "", fmt.Errorf("decode geofence evaluation policy: invalid JSON object")
	}
	action, ok := actionLevelFromPolicy(wire.ExitAction)
	if !ok {
		return EvaluationPolicy{}, "", fmt.Errorf(
			"decode geofence evaluation policy: unsupported exit_action %q",
			wire.ExitAction,
		)
	}
	if wire.ExitToleranceMeters != nil {
		defaults.ExitToleranceMeters = *wire.ExitToleranceMeters
	}
	if wire.ReentryToleranceMeters != nil {
		defaults.ReentryToleranceMeters = *wire.ReentryToleranceMeters
	}
	if wire.ExitConsecutiveSamples != nil {
		defaults.ExitConsecutiveSamples = *wire.ExitConsecutiveSamples
	}
	if wire.ReentryConsecutiveSamples != nil {
		defaults.ReentryConsecutiveSamples = *wire.ReentryConsecutiveSamples
	}
	if wire.SampleMaxAgeSeconds != nil {
		defaults.SampleMaxAgeSeconds = *wire.SampleMaxAgeSeconds
	}
	if wire.MaxJumpDistanceMeters != nil {
		defaults.MaxJumpDistanceMeters = *wire.MaxJumpDistanceMeters
	}
	if wire.MaxImpliedSpeedMPS != nil {
		defaults.MaxImpliedSpeedMPS = *wire.MaxImpliedSpeedMPS
	}
	if wire.MaxDeviceClockSkewSeconds != nil {
		defaults.MaxDeviceClockSkewSeconds = *wire.MaxDeviceClockSkewSeconds
	}
	if wire.MinimumStateDurationSeconds != nil {
		defaults.MinimumStateDurationSeconds = *wire.MinimumStateDurationSeconds
	}
	if err := validateEvaluationPolicy(defaults); err != nil {
		return EvaluationPolicy{}, "", err
	}
	return defaults, action, nil
}

func validateEvaluationPolicy(policy EvaluationPolicy) error {
	finiteNonNegative := func(value float64) bool {
		return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
	}
	switch {
	case !finiteNonNegative(policy.ExitToleranceMeters):
		return fmt.Errorf("decode geofence evaluation policy: invalid exit tolerance")
	case !finiteNonNegative(policy.ReentryToleranceMeters):
		return fmt.Errorf("decode geofence evaluation policy: invalid reentry tolerance")
	case policy.ExitConsecutiveSamples < 1:
		return fmt.Errorf("decode geofence evaluation policy: exit consecutive samples must be positive")
	case policy.ReentryConsecutiveSamples < 1:
		return fmt.Errorf("decode geofence evaluation policy: reentry consecutive samples must be positive")
	case policy.SampleMaxAgeSeconds <= 0:
		return fmt.Errorf("decode geofence evaluation policy: sample maximum age must be positive")
	case !finiteNonNegative(policy.MaxJumpDistanceMeters):
		return fmt.Errorf("decode geofence evaluation policy: invalid maximum jump distance")
	case !finiteNonNegative(policy.MaxImpliedSpeedMPS):
		return fmt.Errorf("decode geofence evaluation policy: invalid maximum implied speed")
	case policy.MaxDeviceClockSkewSeconds < 0:
		return fmt.Errorf("decode geofence evaluation policy: invalid device clock skew")
	case policy.MinimumStateDurationSeconds < 0:
		return fmt.Errorf("decode geofence evaluation policy: invalid minimum state duration")
	default:
		return nil
	}
}

func actionLevelFromPolicy(exitAction string) (ActionLevel, bool) {
	switch ActionLevel(exitAction) {
	case ActionLevelNotifyOnly:
		return ActionLevelNotifyOnly, true
	case ActionLevelManualReview:
		return ActionLevelManualReview, true
	case ActionLevelDeactivate:
		return ActionLevelDeactivate, true
	default:
		return "", false
	}
}

func aggregateEffectiveState(evaluations []BindingEvaluation) EffectiveSnapshot {
	if len(evaluations) == 0 {
		return EffectiveSnapshot{
			State:               EffectiveStateUnmanaged,
			RequiredActionLevel: ActionLevelNone,
		}
	}

	result := EffectiveSnapshot{State: EffectiveStateUnknown}
	var unknownBindingID *uuid.UUID
	var outsideCount int
	var hasInside bool
	var outsideAction ActionLevel
	var outsideBindingID *uuid.UUID
	for _, evaluation := range evaluations {
		switch normalizedConfirmedState(evaluation.ConfirmedState) {
		case ConfirmedStateOutside:
			outsideCount++
			if actionLevelRank(evaluation.ActionLevel) > actionLevelRank(outsideAction) ||
				(actionLevelRank(evaluation.ActionLevel) == actionLevelRank(outsideAction) &&
					bindingIDBefore(evaluation.BindingID, outsideBindingID)) {
				id := evaluation.BindingID
				outsideBindingID = &id
				outsideAction = evaluation.ActionLevel
			}
		case ConfirmedStateInside:
			hasInside = true
		case ConfirmedStateUnknown:
			if unknownBindingID == nil ||
				evaluation.BindingID.String() < unknownBindingID.String() {
				id := evaluation.BindingID
				unknownBindingID = &id
			}
		}
	}
	if outsideCount == len(evaluations) {
		result.State = EffectiveStateOutside
		result.RequiredActionLevel = outsideAction
		result.TriggerBindingID = outsideBindingID
		return result
	}
	if hasInside {
		result.State = EffectiveStateInside
		result.RequiredActionLevel = ActionLevelNone
		return result
	}
	if unknownBindingID != nil {
		result.State = EffectiveStateUnknown
		result.RequiredActionLevel = ActionLevelNone
		result.TriggerBindingID = unknownBindingID
	}
	return result
}

func actionLevelRank(level ActionLevel) int {
	switch level {
	case ActionLevelNone:
		return 0
	case ActionLevelNotifyOnly:
		return 1
	case ActionLevelManualReview:
		return 2
	case ActionLevelDeactivate:
		return 3
	default:
		return -1
	}
}

func bindingIDBefore(candidate uuid.UUID, current *uuid.UUID) bool {
	return current == nil || candidate.String() < current.String()
}
