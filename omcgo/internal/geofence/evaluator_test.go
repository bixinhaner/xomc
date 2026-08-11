package geofence

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatePolygonGeometryReturnsSignedInsideDistance(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	result, err := Evaluate(EvaluationInput{
		RuleType: RuleTypePolygonAllowZone,
		Geometry: json.RawMessage(`{
			"type":"Polygon",
			"coordinates":[[[121.0,31.0],[121.01,31.0],[121.01,31.01],[121.0,31.01]]]
		}`),
		Position: PositionSnapshot{
			Longitude: 121.005, Latitude: 31.005,
			ObservationVersion: 1, ObservedAt: now,
		},
		Policy:      immediateEvaluationPolicy(),
		EvaluatedAt: now,
	})

	require.NoError(t, err)
	assert.Equal(t, RawPositionInside, result.RawPosition)
	require.NotNil(t, result.SignedDistanceMeters)
	assert.Less(t, *result.SignedDistanceMeters, 0.0)
	assert.InDelta(t, -476, *result.SignedDistanceMeters, 15)
}

func TestEvaluatePolygonGeometryReturnsSignedOutsideDistance(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	result, err := Evaluate(EvaluationInput{
		RuleType: RuleTypePolygonAllowZone,
		Geometry: json.RawMessage(`{
			"type":"Polygon",
			"coordinates":[[[121.0,31.0],[121.01,31.0],[121.01,31.01],[121.0,31.01]]]
		}`),
		Position: PositionSnapshot{
			Longitude: 121.02, Latitude: 31.005,
			ObservationVersion: 1, ObservedAt: now,
		},
		Policy:      immediateEvaluationPolicy(),
		EvaluatedAt: now,
	})

	require.NoError(t, err)
	assert.Equal(t, RawPositionOutside, result.RawPosition)
	require.NotNil(t, result.SignedDistanceMeters)
	assert.Greater(t, *result.SignedDistanceMeters, 0.0)
	assert.InDelta(t, 952, *result.SignedDistanceMeters, 20)
}

func TestEvaluateCircleGeometryReturnsDistanceMinusRadius(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	result, err := Evaluate(EvaluationInput{
		RuleType: RuleTypeBaselineRadius,
		Geometry: json.RawMessage(`{
			"type":"Circle",
			"center":[121.0,31.0],
			"radiusMeters":100,
			"source":"manual"
		}`),
		Position: PositionSnapshot{
			Longitude: 121.0, Latitude: 31.001,
			ObservationVersion: 1, ObservedAt: now,
		},
		Policy:      immediateEvaluationPolicy(),
		EvaluatedAt: now,
	})

	require.NoError(t, err)
	assert.Equal(t, RawPositionOutside, result.RawPosition)
	require.NotNil(t, result.SignedDistanceMeters)
	assert.InDelta(t, 11.2, *result.SignedDistanceMeters, 1)
}

func immediateEvaluationPolicy() EvaluationPolicy {
	return EvaluationPolicy{
		ExitConsecutiveSamples:    1,
		ReentryConsecutiveSamples: 1,
		SampleMaxAgeSeconds:       300,
	}
}

func TestEvaluateGPSQualityRejectsZeroCoordinate(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	result, err := Evaluate(validCircleInput(now, PositionSnapshot{
		Longitude: 0, Latitude: 0, ObservationVersion: 2, ObservedAt: now,
	}))
	require.NoError(t, err)
	assert.Equal(t, RawPositionUnknown, result.RawPosition)
	assert.Equal(t, "invalid_coordinates", result.Reason)
	assert.False(t, result.StateEdge)
}

func TestEvaluateGPSQualityRejectsStaleSample(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	result, err := Evaluate(validCircleInput(now, PositionSnapshot{
		Longitude: 121, Latitude: 31, ObservationVersion: 2,
		ObservedAt: now.Add(-301 * time.Second),
	}))
	require.NoError(t, err)
	assert.Equal(t, RawPositionUnknown, result.RawPosition)
	assert.Equal(t, "stale_sample", result.Reason)
}

func TestEvaluateGPSQualityRejectsFutureDeviceClock(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	future := now.Add(301 * time.Second)
	result, err := Evaluate(validCircleInput(now, PositionSnapshot{
		Longitude: 121, Latitude: 31, ObservationVersion: 2,
		ObservedAt: future, ReceivedAt: now, DeviceReportedAt: &future,
	}))
	require.NoError(t, err)
	assert.Equal(t, RawPositionUnknown, result.RawPosition)
	assert.Equal(t, "device_clock_invalid", result.Reason)
}

func TestEvaluateGPSQualityRejectsImplausibleJump(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	jump := 5001.0
	result, err := Evaluate(validCircleInput(now, PositionSnapshot{
		Longitude: 121, Latitude: 31, ObservationVersion: 2,
		ObservedAt: now, MovementDistanceMeters: &jump,
	}))
	require.NoError(t, err)
	assert.Equal(t, RawPositionUnknown, result.RawPosition)
	assert.Equal(t, "implausible_jump", result.Reason)
}

func TestEvaluateDuplicateObservationPreservesPreviousState(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	input := validCircleInput(now, PositionSnapshot{
		Longitude: 122, Latitude: 32, ObservationVersion: 7, ObservedAt: now,
	})
	input.Previous = EvaluationState{
		ConfirmedState:         ConfirmedStateInside,
		CandidateState:         CandidateStateExit,
		CandidateCount:         2,
		LastObservationVersion: 7,
	}

	result, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, "duplicate_observation", result.Reason)
	assert.Equal(t, ConfirmedStateInside, result.ConfirmedState)
	assert.Equal(t, CandidateStateExit, result.CandidateState)
	assert.Equal(t, 2, result.CandidateCount)
	assert.False(t, result.StateEdge)
}

func TestEvaluateOutOfOrderObservationPreservesPreviousState(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	input := validCircleInput(now, PositionSnapshot{
		Longitude: 122, Latitude: 32, ObservationVersion: 6, ObservedAt: now,
	})
	input.Previous = EvaluationState{
		ConfirmedState:         ConfirmedStateOutside,
		LastObservationVersion: 7,
	}

	result, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, "out_of_order_observation", result.Reason)
	assert.Equal(t, ConfirmedStateOutside, result.ConfirmedState)
	assert.False(t, result.StateEdge)
}

func validCircleInput(now time.Time, position PositionSnapshot) EvaluationInput {
	policy := immediateEvaluationPolicy()
	policy.MaxJumpDistanceMeters = 5000
	policy.MaxImpliedSpeedMPS = 55
	policy.MaxDeviceClockSkewSeconds = 300
	return EvaluationInput{
		RuleType: RuleTypeBaselineRadius,
		Geometry: json.RawMessage(`{
			"type":"Circle","center":[121,31],"radiusMeters":100,"source":"manual"
		}`),
		Position: position, Policy: policy, EvaluatedAt: now,
		Previous: EvaluationState{ConfirmedState: ConfirmedStateInside},
	}
}

func TestEvaluateHysteresisConfirmsExitOnlyAfterCountAndDuration(t *testing.T) {
	start := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	policy := immediateEvaluationPolicy()
	policy.ExitConsecutiveSamples = 3
	policy.MinimumStateDurationSeconds = 60
	input := outsideCircleInput(start, 1, policy, EvaluationState{
		ConfirmedState: ConfirmedStateInside,
	})

	first, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, ConfirmedStateInside, first.ConfirmedState)
	assert.Equal(t, CandidateStateExit, first.CandidateState)
	assert.Equal(t, 1, first.CandidateCount)
	assert.False(t, first.StateEdge)

	input = outsideCircleInput(start.Add(30*time.Second), 2, policy, stateFromResult(first))
	second, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, 2, second.CandidateCount)
	assert.False(t, second.StateEdge)

	input = outsideCircleInput(start.Add(61*time.Second), 3, policy, stateFromResult(second))
	third, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, ConfirmedStateOutside, third.ConfirmedState)
	assert.Equal(t, CandidateStateNone, third.CandidateState)
	assert.True(t, third.StateEdge)
}

func TestEvaluateHysteresisConfirmsReentryOnlyAfterConfiguredCount(t *testing.T) {
	start := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	policy := immediateEvaluationPolicy()
	policy.ReentryConsecutiveSamples = 2
	input := insideCircleInput(start, 10, policy, EvaluationState{
		ConfirmedState: ConfirmedStateOutside,
	})

	first, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, ConfirmedStateOutside, first.ConfirmedState)
	assert.Equal(t, CandidateStateReentry, first.CandidateState)
	assert.False(t, first.StateEdge)

	input = insideCircleInput(start.Add(time.Second), 11, policy, stateFromResult(first))
	second, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, ConfirmedStateInside, second.ConfirmedState)
	assert.True(t, second.StateEdge)
}

func TestEvaluateBoundaryResetsPendingCandidate(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	policy := immediateEvaluationPolicy()
	policy.ExitToleranceMeters = 20
	policy.ReentryToleranceMeters = 20
	input := validCircleInput(now, PositionSnapshot{
		Longitude: 121, Latitude: 31.00094,
		ObservationVersion: 3, ObservedAt: now,
	})
	input.Policy = policy
	input.Previous = EvaluationState{
		ConfirmedState:         ConfirmedStateInside,
		CandidateState:         CandidateStateExit,
		CandidateCount:         2,
		LastObservationVersion: 2,
	}

	result, err := Evaluate(input)
	require.NoError(t, err)
	assert.Equal(t, RawPositionBoundary, result.RawPosition)
	assert.Equal(t, ConfirmedStateInside, result.ConfirmedState)
	assert.Equal(t, CandidateStateNone, result.CandidateState)
	assert.Zero(t, result.CandidateCount)
	assert.False(t, result.StateEdge)
}

func outsideCircleInput(
	observedAt time.Time,
	version int64,
	policy EvaluationPolicy,
	previous EvaluationState,
) EvaluationInput {
	input := validCircleInput(observedAt, PositionSnapshot{
		Longitude: 121, Latitude: 31.002,
		ObservationVersion: version, ObservedAt: observedAt,
	})
	input.Policy = policy
	input.Previous = previous
	return input
}

func insideCircleInput(
	observedAt time.Time,
	version int64,
	policy EvaluationPolicy,
	previous EvaluationState,
) EvaluationInput {
	input := validCircleInput(observedAt, PositionSnapshot{
		Longitude: 121, Latitude: 31,
		ObservationVersion: version, ObservedAt: observedAt,
	})
	input.Policy = policy
	input.Previous = previous
	return input
}

func stateFromResult(result EvaluationResult) EvaluationState {
	return EvaluationState{
		ConfirmedState:         result.ConfirmedState,
		CandidateState:         result.CandidateState,
		CandidateCount:         result.CandidateCount,
		CandidateSince:         result.CandidateSince,
		LastObservationVersion: result.ObservationVersion,
	}
}
