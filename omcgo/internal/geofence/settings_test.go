package geofence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveRuntimeModeUsesMoreConservativeLevel(t *testing.T) {
	tests := []struct {
		name        string
		systemMode  RuntimeMode
		carrierMode RuntimeMode
		want        RuntimeMode
	}{
		{name: "both off", systemMode: RuntimeModeOff, carrierMode: RuntimeModeOff, want: RuntimeModeOff},
		{name: "system off", systemMode: RuntimeModeOff, carrierMode: RuntimeModeObserve, want: RuntimeModeOff},
		{name: "carrier off", systemMode: RuntimeModeObserve, carrierMode: RuntimeModeOff, want: RuntimeModeOff},
		{name: "both observe", systemMode: RuntimeModeObserve, carrierMode: RuntimeModeObserve, want: RuntimeModeObserve},
		{name: "system observe limits enforce", systemMode: RuntimeModeObserve, carrierMode: RuntimeModeEnforce, want: RuntimeModeObserve},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EffectiveRuntimeMode(tt.systemMode, tt.carrierMode))
		})
	}
}

func TestEffectiveRuntimeModeFailsSafeForMalformedMode(t *testing.T) {
	assert.Equal(t, RuntimeModeOff, EffectiveRuntimeMode("invalid", RuntimeModeObserve))
	assert.Equal(t, RuntimeModeOff, EffectiveRuntimeMode(RuntimeModeObserve, "invalid"))
}

func TestGetAvailabilityUsesOnlySystemMode(t *testing.T) {
	repository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeOff,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc", Mode: RuntimeModeEnforce,
			DefaultBaselineRadiusMeters: 100,
		}},
	}}
	service := NewService(nil, repository)

	availability, err := service.GetAvailability(context.Background())
	require.NoError(t, err)
	assert.False(t, availability.Enabled)

	repository.settings.SystemMode = RuntimeModeObserve
	availability, err = service.GetAvailability(context.Background())
	require.NoError(t, err)
	assert.True(t, availability.Enabled)
}

func TestGetSettingsForVisibleGroupsFiltersCarriers(t *testing.T) {
	repository := &fakeRepository{
		deniedCarriers: map[string]bool{"ctcc": true},
	}
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeEnforce,
		Carriers: []CarrierSetting{
			{Carrier: "cmcc", Mode: RuntimeModeEnforce},
			{Carrier: "ctcc", Mode: RuntimeModeEnforce},
		},
	}}
	service := NewService(repository, settingsRepository)

	settings, err := service.GetSettingsForVisibleGroups(
		context.Background(),
		[]uuid.UUID{uuid.New()},
	)

	require.NoError(t, err)
	require.Len(t, settings.Carriers, 1)
	assert.Equal(t, "cmcc", settings.Carriers[0].Carrier)
}

func TestValidateSettingsAcceptsAllRuntimeModes(t *testing.T) {
	err := ValidateSettings(Settings{
		SystemMode: RuntimeModeEnforce,
		Carriers: []CarrierSetting{
			{
				Carrier:                     "cmcc",
				Mode:                        RuntimeModeEnforce,
				DefaultBaselineRadiusMeters: 100,
			},
			{
				Carrier:                     "ctcc",
				Mode:                        RuntimeModeOff,
				DefaultBaselineRadiusMeters: 200,
			},
		},
	})

	require.NoError(t, err)
}

func TestValidateSettingsRejectsInvalidCarrierRows(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
	}{
		{
			name: "blank carrier",
			settings: Settings{SystemMode: RuntimeModeOff, Carriers: []CarrierSetting{{
				Mode: RuntimeModeOff, DefaultBaselineRadiusMeters: 100,
			}}},
		},
		{
			name: "duplicate carrier",
			settings: Settings{
				SystemMode: RuntimeModeOff,
				Carriers: []CarrierSetting{
					{Carrier: "cmcc", Mode: RuntimeModeOff, DefaultBaselineRadiusMeters: 100},
					{Carrier: "cmcc", Mode: RuntimeModeObserve, DefaultBaselineRadiusMeters: 100},
				},
			},
		},
		{
			name: "zero radius",
			settings: Settings{SystemMode: RuntimeModeOff, Carriers: []CarrierSetting{{
				Carrier: "cmcc", Mode: RuntimeModeOff,
			}}},
		},
		{
			name: "radius too large",
			settings: Settings{SystemMode: RuntimeModeOff, Carriers: []CarrierSetting{{
				Carrier: "cmcc", Mode: RuntimeModeOff, DefaultBaselineRadiusMeters: 50001,
			}}},
		},
		{
			name: "invalid system mode",
			settings: Settings{SystemMode: "invalid", Carriers: []CarrierSetting{{
				Carrier: "cmcc", Mode: RuntimeModeOff, DefaultBaselineRadiusMeters: 100,
			}}},
		},
		{
			name: "invalid carrier mode",
			settings: Settings{SystemMode: RuntimeModeOff, Carriers: []CarrierSetting{{
				Carrier: "cmcc", Mode: "invalid", DefaultBaselineRadiusMeters: 100,
			}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSettings(tt.settings)
			require.Error(t, err)
		})
	}
}
