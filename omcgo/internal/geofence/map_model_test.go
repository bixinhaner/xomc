package geofence

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

func TestParseMapBoundsUsesExistingGISOrder(t *testing.T) {
	got, err := ParseMapBounds("120,121,30,31")

	require.NoError(t, err)
	require.Equal(t, &MapBounds{
		MinLongitude: 120,
		MaxLongitude: 121,
		MinLatitude:  30,
		MaxLatitude:  31,
	}, got)
}

func TestParseMapBoundsRejectsInvalidRanges(t *testing.T) {
	for _, raw := range []string{
		"121,120,30,31",
		"120,121,31,30",
		"-181,121,30,31",
		"120,181,30,31",
		"120,121,-91,31",
		"120,121,30,91",
		"120,121,30",
		"120,NaN,30,31",
		"120,+Inf,30,31",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := ParseMapBounds(raw)
			require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
		})
	}
}

func TestParseMapBoundsAllowsEmptyFilter(t *testing.T) {
	got, err := ParseMapBounds("  ")

	require.NoError(t, err)
	require.Nil(t, got)
}

func TestMapDefinitionJSONUsesStableSnakeCaseBounds(t *testing.T) {
	versionID := uuid.New()
	raw, err := json.Marshal(MapDefinition{
		Definition: Definition{
			ID:               uuid.New(),
			CurrentVersionID: &versionID,
		},
		CurrentVersion: &Version{
			ID: versionID,
			BoundingBox: BoundingBox{
				MinLongitude: 120,
				MinLatitude:  30,
				MaxLongitude: 121,
				MaxLatitude:  31,
			},
		},
	})

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	currentVersion := decoded["current_version"].(map[string]any)
	bounds := currentVersion["bounding_box"].(map[string]any)
	require.Equal(t, float64(120), bounds["min_longitude"])
	require.Equal(t, float64(30), bounds["min_latitude"])
	require.Equal(t, float64(121), bounds["max_longitude"])
	require.Equal(t, float64(31), bounds["max_latitude"])
	require.NotContains(t, bounds, "MinLongitude")
}
