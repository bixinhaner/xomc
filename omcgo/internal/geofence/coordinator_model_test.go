package geofence

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeEvaluationPolicyAppliesSafeDefaults(t *testing.T) {
	policy, action, err := decodeEvaluationPolicy(
		json.RawMessage(`{"exit_action":"notify_only"}`),
	)

	require.NoError(t, err)
	assert.Equal(t, ActionLevelNotifyOnly, action)
	assert.Equal(t, 20.0, policy.ExitToleranceMeters)
	assert.Equal(t, 20.0, policy.ReentryToleranceMeters)
	assert.Equal(t, 3, policy.ExitConsecutiveSamples)
	assert.Equal(t, 3, policy.ReentryConsecutiveSamples)
	assert.Equal(t, 300, policy.SampleMaxAgeSeconds)
	assert.Equal(t, 5000.0, policy.MaxJumpDistanceMeters)
	assert.Equal(t, 55.0, policy.MaxImpliedSpeedMPS)
	assert.Equal(t, 300, policy.MaxDeviceClockSkewSeconds)
	assert.Equal(t, 60, policy.MinimumStateDurationSeconds)
}

func TestDecodeEvaluationPolicyUsesExplicitDocumentedValues(t *testing.T) {
	policy, action, err := decodeEvaluationPolicy(json.RawMessage(`{
		"exit_action":"manual_review",
		"exit_tolerance_meters":5,
		"reentry_tolerance_meters":6,
		"exit_consecutive_samples":2,
		"reentry_consecutive_samples":4,
		"sample_max_age_seconds":120,
		"max_jump_distance_meters":1000,
		"max_implied_speed_mps":30,
		"max_device_clock_skew_seconds":90,
		"minimum_state_duration_seconds":10
	}`))

	require.NoError(t, err)
	assert.Equal(t, ActionLevelManualReview, action)
	assert.Equal(t, 5.0, policy.ExitToleranceMeters)
	assert.Equal(t, 6.0, policy.ReentryToleranceMeters)
	assert.Equal(t, 2, policy.ExitConsecutiveSamples)
	assert.Equal(t, 4, policy.ReentryConsecutiveSamples)
	assert.Equal(t, 120, policy.SampleMaxAgeSeconds)
	assert.Equal(t, 1000.0, policy.MaxJumpDistanceMeters)
	assert.Equal(t, 30.0, policy.MaxImpliedSpeedMPS)
	assert.Equal(t, 90, policy.MaxDeviceClockSkewSeconds)
	assert.Equal(t, 10, policy.MinimumStateDurationSeconds)
}

func TestDecodeEvaluationPolicyRejectsMalformedUnsupportedOrUnsafeValues(t *testing.T) {
	tests := []json.RawMessage{
		json.RawMessage(`not-json`),
		json.RawMessage(`{"exit_action":"shutdown"}`),
		json.RawMessage(`{"exit_action":"notify_only","exit_consecutive_samples":0}`),
		json.RawMessage(`{"exit_action":"notify_only","sample_max_age_seconds":-1}`),
		json.RawMessage(`{"exit_action":"notify_only","max_implied_speed_mps":-1}`),
	}

	for _, raw := range tests {
		_, _, err := decodeEvaluationPolicy(raw)
		require.Error(t, err)
	}
}

func TestAggregateEffectiveStateNoBindingsIsUnmanaged(t *testing.T) {
	result := aggregateEffectiveState(nil)

	assert.Equal(t, EffectiveStateUnmanaged, result.State)
	assert.Equal(t, ActionLevelNone, result.RequiredActionLevel)
	assert.Nil(t, result.TriggerBindingID)
}

func TestAggregateEffectiveStateRequiresAllOutsideAndUsesHighestAction(t *testing.T) {
	insideID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	notifyID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	deactivateID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	result := aggregateEffectiveState([]BindingEvaluation{
		{BindingID: insideID, ConfirmedState: ConfirmedStateInside, ActionLevel: ActionLevelNone},
		{BindingID: notifyID, ConfirmedState: ConfirmedStateOutside, ActionLevel: ActionLevelNotifyOnly},
		{BindingID: deactivateID, ConfirmedState: ConfirmedStateOutside, ActionLevel: ActionLevelDeactivate},
		{BindingID: uuid.New(), ConfirmedState: ConfirmedStateUnknown, ActionLevel: ActionLevelNone},
	})

	assert.Equal(t, EffectiveStateInside, result.State)
	assert.Equal(t, ActionLevelNone, result.RequiredActionLevel)
	assert.Nil(t, result.TriggerBindingID)

	result = aggregateEffectiveState([]BindingEvaluation{
		{BindingID: notifyID, ConfirmedState: ConfirmedStateOutside, ActionLevel: ActionLevelNotifyOnly},
		{BindingID: deactivateID, ConfirmedState: ConfirmedStateOutside, ActionLevel: ActionLevelDeactivate},
	})
	assert.Equal(t, EffectiveStateOutside, result.State)
	assert.Equal(t, ActionLevelDeactivate, result.RequiredActionLevel)
	require.NotNil(t, result.TriggerBindingID)
	assert.Equal(t, deactivateID, *result.TriggerBindingID)
}

func TestAggregateEffectiveStateUnknownWinsOnlyWithoutOutside(t *testing.T) {
	unknownID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	result := aggregateEffectiveState([]BindingEvaluation{
		{BindingID: uuid.New(), ConfirmedState: ConfirmedStateInside, ActionLevel: ActionLevelNone},
		{BindingID: unknownID, ConfirmedState: ConfirmedStateUnknown, ActionLevel: ActionLevelNotifyOnly},
	})

	assert.Equal(t, EffectiveStateInside, result.State)
	assert.Equal(t, ActionLevelNone, result.RequiredActionLevel)
	assert.Nil(t, result.TriggerBindingID)
}

func TestAggregateEffectiveStateAllInside(t *testing.T) {
	result := aggregateEffectiveState([]BindingEvaluation{
		{BindingID: uuid.New(), ConfirmedState: ConfirmedStateInside},
		{BindingID: uuid.New(), ConfirmedState: ConfirmedStateInside},
	})

	assert.Equal(t, EffectiveStateInside, result.State)
	assert.Equal(t, ActionLevelNone, result.RequiredActionLevel)
	assert.Nil(t, result.TriggerBindingID)
}

func TestAggregateEffectiveStateTieUsesStableLowestBindingID(t *testing.T) {
	lower := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	higher := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	result := aggregateEffectiveState([]BindingEvaluation{
		{BindingID: higher, ConfirmedState: ConfirmedStateOutside, ActionLevel: ActionLevelManualReview},
		{BindingID: lower, ConfirmedState: ConfirmedStateOutside, ActionLevel: ActionLevelManualReview},
	})

	require.NotNil(t, result.TriggerBindingID)
	assert.Equal(t, lower, *result.TriggerBindingID)
}
