package software

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStages(t *testing.T) {
	tests := []struct {
		name    string
		stages  []CanaryStage
		wantErr bool
	}{
		{"default stages valid", DefaultCanaryStages, false},
		{"single 100 stage valid", []CanaryStage{{Percent: 100, FailureThreshold: 30}}, false},
		{"empty stages invalid", []CanaryStage{}, true},
		{
			"non-monotonic invalid",
			[]CanaryStage{{Percent: 10, FailureThreshold: 5}, {Percent: 5, FailureThreshold: 5}, {Percent: 100, FailureThreshold: 30}},
			true,
		},
		{
			"missing 100 invalid",
			[]CanaryStage{{Percent: 1, FailureThreshold: 5}, {Percent: 50, FailureThreshold: 20}},
			true,
		},
		{
			"percent out of range",
			[]CanaryStage{{Percent: 0, FailureThreshold: 5}, {Percent: 100, FailureThreshold: 30}},
			true,
		},
		{
			"threshold out of range",
			[]CanaryStage{{Percent: 100, FailureThreshold: 0}},
			true,
		},
		{
			"percent over 100",
			[]CanaryStage{{Percent: 50, FailureThreshold: 5}, {Percent: 150, FailureThreshold: 30}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStages(tt.stages)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDevicesForStage(t *testing.T) {
	stages := DefaultCanaryStages // 1, 10, 50, 100
	tests := []struct {
		name     string
		stageIdx int
		total    int
		want     int
	}{
		{"stage 0 of 100", 0, 100, 1},
		{"stage 1 of 100", 1, 100, 10},
		{"stage 2 of 100", 2, 100, 50},
		{"stage 3 of 100", 3, 100, 100},
		{"stage 0 of 10 (rounds to 1 minimum)", 0, 10, 1},
		{"stage 0 of 1 (single device)", 0, 1, 1},
		{"stage 0 of 0 (no devices)", 0, 0, 0},
		{"out of range stage", 5, 100, 0},
		{"negative stage", -1, 100, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DevicesForStage(tt.stageIdx, tt.total, stages)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFailureRate(t *testing.T) {
	assert.InDelta(t, 0.0, FailureRate(0, 0), 0.001)
	assert.InDelta(t, 0.0, FailureRate(0, 100), 0.001)
	assert.InDelta(t, 0.5, FailureRate(50, 100), 0.001)
	assert.InDelta(t, 1.0, FailureRate(100, 100), 0.001)
	// edge: total <= 0 → 0
	assert.Equal(t, 0.0, FailureRate(5, 0))
}

func TestMarshalUnmarshalStages_RoundTrip(t *testing.T) {
	raw, err := MarshalStages(DefaultCanaryStages)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	out, err := UnmarshalStages(raw)
	require.NoError(t, err)
	assert.Equal(t, DefaultCanaryStages, out)
}

func TestUnmarshalStages_NilReturnsDefaults(t *testing.T) {
	out, err := UnmarshalStages(nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultCanaryStages, out)
}

func TestUnmarshalStages_EmptyReturnsDefaults(t *testing.T) {
	out, err := UnmarshalStages(json.RawMessage{})
	require.NoError(t, err)
	assert.Equal(t, DefaultCanaryStages, out)
}

func TestUnmarshalStages_InvalidJSON(t *testing.T) {
	_, err := UnmarshalStages(json.RawMessage(`not-json`))
	assert.Error(t, err)
}

func TestUnmarshalStages_InvalidStages(t *testing.T) {
	// missing 100 trailing
	_, err := UnmarshalStages(json.RawMessage(`[{"percent":1,"failure_threshold":5}]`))
	assert.Error(t, err)
}

func TestCanaryFields_IsCanary(t *testing.T) {
	var nilFields *CanaryFields
	assert.False(t, nilFields.IsCanary())

	full := &CanaryFields{Strategy: StrategyFull}
	assert.False(t, full.IsCanary())

	canary := &CanaryFields{Strategy: StrategyCanary}
	assert.True(t, canary.IsCanary())
}

func TestCanaryFields_CurrentStageDescriptor(t *testing.T) {
	c := &CanaryFields{
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 2, // 1-indexed → second stage = 10%
	}
	desc := c.CurrentStageDescriptor()
	require.NotNil(t, desc)
	assert.Equal(t, 10, desc.Percent)
	assert.Equal(t, 10, desc.FailureThreshold)

	// out of range
	c.CurrentStage = 99
	assert.Nil(t, c.CurrentStageDescriptor())

	c.CurrentStage = 0
	assert.Nil(t, c.CurrentStageDescriptor())
}

func TestStageHistoryEntry_JSONRoundTrip(t *testing.T) {
	entries := []StageHistoryEntry{
		{Stage: 1, Percent: 1, DevicesInStage: 1, SuccessCount: 1, FailCount: 0, FailureRate: 0.0, Action: "advanced"},
		{Stage: 2, Percent: 10, DevicesInStage: 10, SuccessCount: 9, FailCount: 1, FailureRate: 0.1, Action: "paused", Reason: "threshold exceeded"},
	}
	b, err := json.Marshal(entries)
	require.NoError(t, err)
	var out []StageHistoryEntry
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, entries, out)
}
