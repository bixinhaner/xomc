package device

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocationSourceAllowedSeparatesTR069AndExternalModes(t *testing.T) {
	assert.True(t, locationSourceAllowed(model.LocationSourceTR069, "Device.FAP.GPS"))
	assert.False(t, locationSourceAllowed(
		model.LocationSourceTR069,
		"third_party:fence/batchUpdateDeviceLocation",
	))
	assert.True(t, locationSourceAllowed(
		model.LocationSourceExternal,
		"third_party:fence/batchUpdateDeviceLocation",
	))
	assert.False(t, locationSourceAllowed(model.LocationSourceExternal, "Device.FAP.GPS"))
}

func TestNormalizeReportedLocationUsesReceiveTimeWithoutDeviceSampleTime(t *testing.T) {
	receivedAt := time.Date(2026, 7, 30, 2, 0, 0, 0, time.UTC)
	observation := ReportedLocation{Latitude: 31.2, Longitude: 121.5}

	got := normalizeReportedLocation(observation, receivedAt)

	assert.Equal(t, receivedAt, got.ReceivedAt)
	assert.Equal(t, receivedAt, got.ObservedAt)
	assert.Nil(t, got.DeviceReportedAt)
}

func TestNormalizeReportedLocationUsesTrustedDeviceSampleTime(t *testing.T) {
	receivedAt := time.Date(2026, 7, 30, 2, 0, 1, 0, time.UTC)
	reportedAt := receivedAt.Add(-time.Second)
	observation := ReportedLocation{
		Latitude:         31.2,
		Longitude:        121.5,
		ObservedAt:       receivedAt.Add(-time.Hour),
		DeviceReportedAt: &reportedAt,
	}

	got := normalizeReportedLocation(observation, receivedAt)

	assert.Equal(t, receivedAt, got.ReceivedAt)
	assert.Equal(t, reportedAt, got.ObservedAt)
}

func TestMovementEvidenceCalculatesDistanceElapsedAndSpeed(t *testing.T) {
	start := time.Date(2026, 7, 30, 2, 0, 0, 0, time.UTC)
	previous := &ReportedLocation{
		Latitude:   31.2,
		Longitude:  121.5,
		ObservedAt: start,
		Version:    17,
	}
	current := ReportedLocation{
		Latitude:   31.201,
		Longitude:  121.5,
		ObservedAt: start.Add(10 * time.Second),
		Version:    18,
	}

	distance, elapsed, speed := movementEvidence(previous, current)

	require.NotNil(t, distance)
	require.NotNil(t, elapsed)
	require.NotNil(t, speed)
	assert.InDelta(t, 111.19, *distance, 0.5)
	assert.Equal(t, 10.0, *elapsed)
	assert.InDelta(t, *distance/10, *speed, 0.001)
}

func TestMovementEvidenceDoesNotInventSpeedForNonPositiveElapsedTime(t *testing.T) {
	observedAt := time.Date(2026, 7, 30, 2, 0, 0, 0, time.UTC)
	previous := &ReportedLocation{Latitude: 31.2, Longitude: 121.5, ObservedAt: observedAt}
	current := ReportedLocation{Latitude: 31.201, Longitude: 121.5, ObservedAt: observedAt}

	distance, elapsed, speed := movementEvidence(previous, current)

	require.NotNil(t, distance)
	assert.Nil(t, elapsed)
	assert.Nil(t, speed)
}

func TestBuildLocationObservedPayloadCarriesPreviousVersionAndEvidence(t *testing.T) {
	deviceID := uuid.New()
	current := ReportedLocation{
		Latitude:       31.2,
		Longitude:      121.5,
		ObservedAt:     time.Date(2026, 7, 30, 2, 0, 0, 0, time.UTC),
		ReceivedAt:     time.Date(2026, 7, 30, 2, 0, 1, 0, time.UTC),
		Version:        18,
		SourcePath:     "Device.FAP.GPS",
		SatelliteCount: intPointer(8),
	}
	previous := &ReportedLocation{Version: 17}
	distance, elapsed, speed := 3.2, 60.0, 3.2/60

	payload := buildLocationObservedPayload(
		deviceID,
		"SN-1",
		"cmcc",
		LocationObservationWriteResult{
			Current:           current,
			Previous:          previous,
			MovementDistanceM: &distance,
			ElapsedSeconds:    &elapsed,
			ImpliedSpeedMPS:   &speed,
		},
	)

	assert.Equal(t, int64(18), payload.ObservationVersion)
	require.NotNil(t, payload.PreviousObservationVersion)
	assert.Equal(t, int64(17), *payload.PreviousObservationVersion)
	assert.Equal(t, 8, *payload.SatelliteCount)
	assert.Equal(t, distance, *payload.MovementDistanceMeters)
}

func intPointer(value int) *int {
	return &value
}
