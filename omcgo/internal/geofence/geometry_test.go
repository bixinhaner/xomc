package geofence

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidateGeometry_PolygonNormalizesClosure(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Polygon",
		"coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2]]]
	}`)

	snapshot, err := ParseAndValidateGeometry(RuleTypePolygonAllowZone, raw)
	require.NoError(t, err)
	assert.Equal(t, BoundingBox{
		MinLongitude: 121.1,
		MinLatitude:  31.1,
		MaxLongitude: 121.2,
		MaxLatitude:  31.2,
	}, snapshot.BoundingBox)

	var normalized polygonJSON
	require.NoError(t, json.Unmarshal(snapshot.JSON, &normalized))
	require.Len(t, normalized.Coordinates, 1)
	require.Len(t, normalized.Coordinates[0], 4)
	assert.Equal(t, normalized.Coordinates[0][0], normalized.Coordinates[0][3])
}

func TestParseAndValidateGeometry_RejectsSelfIntersection(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Polygon",
		"coordinates":[[[121.1,31.1],[121.2,31.2],[121.1,31.2],[121.2,31.1]]]
	}`)

	_, err := ParseAndValidateGeometry(RuleTypePolygonAllowZone, raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "self-intersect")
}

func TestParseAndValidateGeometry_NormalizesDrawingControlDuplicates(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Polygon",
		"coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.1],[121.2,31.2],[121.1,31.1],[121.1,31.1]]]
	}`)

	snapshot, err := ParseAndValidateGeometry(RuleTypePolygonAllowZone, raw)
	require.NoError(t, err)

	var normalized polygonJSON
	require.NoError(t, json.Unmarshal(snapshot.JSON, &normalized))
	require.Len(t, normalized.Coordinates[0], 4)
	assert.Equal(t, normalized.Coordinates[0][0], normalized.Coordinates[0][3])
}

func TestParseAndValidateGeometry_RejectsFewerThanThreeDistinctVerticesAfterNormalization(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Polygon",
		"coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.1],[121.1,31.1]]]
	}`)

	_, err := ParseAndValidateGeometry(RuleTypePolygonAllowZone, raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least three vertices")
}

func TestParseAndValidateGeometry_RejectsDatelineCrossing(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Polygon",
		"coordinates":[[[179.8,31.1],[-179.8,31.1],[-179.8,31.2]]]
	}`)

	_, err := ParseAndValidateGeometry(RuleTypePolygonAllowZone, raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "date line")
}

func TestParseAndValidateGeometry_CircleRequiresPositiveRadius(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Circle",
		"center":[121.1,31.1],
		"radiusMeters":0,
		"source":"manual"
	}`)

	_, err := ParseAndValidateGeometry(RuleTypeBaselineRadius, raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "radius")
}

func TestParseAndValidateGeometry_RejectsRuleGeometryMismatch(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"Circle",
		"center":[121.1,31.1],
		"radiusMeters":100,
		"source":"manual"
	}`)

	_, err := ParseAndValidateGeometry(RuleTypePolygonAllowZone, raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires Polygon")
}
