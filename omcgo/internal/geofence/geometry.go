package geofence

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

const (
	MaxPolygonVertices     = 500
	MaxCircleRadiusMeters  = 50000
	earthRadiusMeters      = 6371008.8
	geometryCompareEpsilon = 1e-12
)

var ErrInvalidGeometry = errors.New("invalid geofence geometry")

// BoundingBox is derived by the server from validated WGS84 geometry.
type BoundingBox struct {
	MinLongitude float64 `json:"min_longitude"`
	MinLatitude  float64 `json:"min_latitude"`
	MaxLongitude float64 `json:"max_longitude"`
	MaxLatitude  float64 `json:"max_latitude"`
}

// GeometrySnapshot is the normalized immutable geometry stored in a version.
type GeometrySnapshot struct {
	JSON        json.RawMessage
	BoundingBox BoundingBox
}

type polygonJSON struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

type circleJSON struct {
	Type                     string    `json:"type"`
	Center                   []float64 `json:"center"`
	RadiusMeters             float64   `json:"radiusMeters"`
	Source                   string    `json:"source,omitempty"`
	SourceObservationVersion *int64    `json:"sourceObservationVersion,omitempty"`
}

// ParseAndValidateGeometry validates a rule-specific geometry, normalizes it,
// and derives its bounding box without trusting client-provided metadata.
func ParseAndValidateGeometry(ruleType RuleType, raw json.RawMessage) (GeometrySnapshot, error) {
	switch ruleType {
	case RuleTypePolygonAllowZone:
		return parsePolygon(raw)
	case RuleTypeBaselineRadius:
		return parseCircle(raw)
	default:
		return GeometrySnapshot{}, fmt.Errorf("unsupported rule type %q: %w", ruleType, ErrInvalidGeometry)
	}
}

func parsePolygon(raw json.RawMessage) (GeometrySnapshot, error) {
	var polygon polygonJSON
	if err := json.Unmarshal(raw, &polygon); err != nil {
		return GeometrySnapshot{}, fmt.Errorf("decode polygon: %w", ErrInvalidGeometry)
	}
	if polygon.Type != "Polygon" {
		return GeometrySnapshot{}, fmt.Errorf("polygon_allow_zone requires Polygon geometry: %w", ErrInvalidGeometry)
	}
	if len(polygon.Coordinates) != 1 {
		return GeometrySnapshot{}, fmt.Errorf("polygon must contain exactly one exterior ring: %w", ErrInvalidGeometry)
	}

	points := polygon.Coordinates[0]
	if len(points) > 1 && samePoint(points[0], points[len(points)-1]) {
		points = points[:len(points)-1]
	}
	if len(points) < 3 {
		return GeometrySnapshot{}, fmt.Errorf("polygon requires at least three vertices: %w", ErrInvalidGeometry)
	}
	if len(points) > MaxPolygonVertices {
		return GeometrySnapshot{}, fmt.Errorf("polygon exceeds %d vertices: %w", MaxPolygonVertices, ErrInvalidGeometry)
	}

	bbox, err := validatePolygonPoints(points)
	if err != nil {
		return GeometrySnapshot{}, err
	}
	if polygonSelfIntersects(points) {
		return GeometrySnapshot{}, fmt.Errorf("polygon self-intersects: %w", ErrInvalidGeometry)
	}

	closed := make([][]float64, 0, len(points)+1)
	for _, point := range points {
		closed = append(closed, []float64{point[0], point[1]})
	}
	closed = append(closed, []float64{points[0][0], points[0][1]})
	polygon.Coordinates = [][][]float64{closed}
	normalized, err := json.Marshal(polygon)
	if err != nil {
		return GeometrySnapshot{}, fmt.Errorf("marshal normalized polygon: %w", err)
	}
	return GeometrySnapshot{JSON: normalized, BoundingBox: bbox}, nil
}

func validatePolygonPoints(points [][]float64) (BoundingBox, error) {
	bbox := BoundingBox{
		MinLongitude: math.Inf(1),
		MinLatitude:  math.Inf(1),
		MaxLongitude: math.Inf(-1),
		MaxLatitude:  math.Inf(-1),
	}
	distinct := make(map[[2]float64]struct{}, len(points))
	for index, point := range points {
		if err := validateCoordinate(point); err != nil {
			return BoundingBox{}, fmt.Errorf("polygon vertex %d: %w", index, err)
		}
		if index > 0 && samePoint(points[index-1], point) {
			return BoundingBox{}, fmt.Errorf("polygon has adjacent duplicate vertices: %w", ErrInvalidGeometry)
		}
		distinct[[2]float64{point[0], point[1]}] = struct{}{}
		bbox.MinLongitude = math.Min(bbox.MinLongitude, point[0])
		bbox.MinLatitude = math.Min(bbox.MinLatitude, point[1])
		bbox.MaxLongitude = math.Max(bbox.MaxLongitude, point[0])
		bbox.MaxLatitude = math.Max(bbox.MaxLatitude, point[1])
	}
	if len(distinct) < 3 {
		return BoundingBox{}, fmt.Errorf("polygon requires three distinct vertices: %w", ErrInvalidGeometry)
	}
	if bbox.MaxLongitude-bbox.MinLongitude > 180 {
		return BoundingBox{}, fmt.Errorf("polygon crosses the international date line: %w", ErrInvalidGeometry)
	}
	return bbox, nil
}

func parseCircle(raw json.RawMessage) (GeometrySnapshot, error) {
	var circle circleJSON
	if err := json.Unmarshal(raw, &circle); err != nil {
		return GeometrySnapshot{}, fmt.Errorf("decode circle: %w", ErrInvalidGeometry)
	}
	if circle.Type != "Circle" {
		return GeometrySnapshot{}, fmt.Errorf("baseline_radius requires Circle geometry: %w", ErrInvalidGeometry)
	}
	if err := validateCoordinate(circle.Center); err != nil {
		return GeometrySnapshot{}, fmt.Errorf("circle center: %w", err)
	}
	if circle.Center[0] == 0 && circle.Center[1] == 0 {
		return GeometrySnapshot{}, fmt.Errorf("circle center (0,0) is not a valid location: %w", ErrInvalidGeometry)
	}
	if !isFinite(circle.RadiusMeters) || circle.RadiusMeters <= 0 ||
		circle.RadiusMeters > MaxCircleRadiusMeters {
		return GeometrySnapshot{}, fmt.Errorf(
			"circle radius must be within (0,%d] meters: %w",
			MaxCircleRadiusMeters,
			ErrInvalidGeometry,
		)
	}

	normalized, err := json.Marshal(circle)
	if err != nil {
		return GeometrySnapshot{}, fmt.Errorf("marshal normalized circle: %w", err)
	}
	latitudeDelta := circle.RadiusMeters / earthRadiusMeters * 180 / math.Pi
	cosLatitude := math.Cos(circle.Center[1] * math.Pi / 180)
	longitudeDelta := 180.0
	if math.Abs(cosLatitude) > geometryCompareEpsilon {
		longitudeDelta = latitudeDelta / math.Abs(cosLatitude)
	}
	return GeometrySnapshot{
		JSON: normalized,
		BoundingBox: BoundingBox{
			MinLongitude: math.Max(-180, circle.Center[0]-longitudeDelta),
			MinLatitude:  math.Max(-90, circle.Center[1]-latitudeDelta),
			MaxLongitude: math.Min(180, circle.Center[0]+longitudeDelta),
			MaxLatitude:  math.Min(90, circle.Center[1]+latitudeDelta),
		},
	}, nil
}

func validateCoordinate(point []float64) error {
	if len(point) != 2 {
		return fmt.Errorf("coordinate must be [longitude,latitude]: %w", ErrInvalidGeometry)
	}
	if !isFinite(point[0]) || !isFinite(point[1]) {
		return fmt.Errorf("coordinate must be finite: %w", ErrInvalidGeometry)
	}
	if point[0] < -180 || point[0] > 180 || point[1] < -90 || point[1] > 90 {
		return fmt.Errorf("coordinate is outside WGS84 range: %w", ErrInvalidGeometry)
	}
	return nil
}

func polygonSelfIntersects(points [][]float64) bool {
	count := len(points)
	for first := 0; first < count; first++ {
		firstNext := (first + 1) % count
		for second := first + 1; second < count; second++ {
			secondNext := (second + 1) % count
			if first == second || firstNext == second || secondNext == first {
				continue
			}
			if segmentsIntersect(points[first], points[firstNext], points[second], points[secondNext]) {
				return true
			}
		}
	}
	return false
}

func segmentsIntersect(a, b, c, d []float64) bool {
	o1 := orientation(a, b, c)
	o2 := orientation(a, b, d)
	o3 := orientation(c, d, a)
	o4 := orientation(c, d, b)

	if o1 != o2 && o3 != o4 {
		return true
	}
	return o1 == 0 && pointOnSegment(a, c, b) ||
		o2 == 0 && pointOnSegment(a, d, b) ||
		o3 == 0 && pointOnSegment(c, a, d) ||
		o4 == 0 && pointOnSegment(c, b, d)
}

func orientation(a, b, c []float64) int {
	value := (b[1]-a[1])*(c[0]-b[0]) - (b[0]-a[0])*(c[1]-b[1])
	if math.Abs(value) <= geometryCompareEpsilon {
		return 0
	}
	if value > 0 {
		return 1
	}
	return -1
}

func pointOnSegment(a, point, b []float64) bool {
	return point[0] <= math.Max(a[0], b[0])+geometryCompareEpsilon &&
		point[0] >= math.Min(a[0], b[0])-geometryCompareEpsilon &&
		point[1] <= math.Max(a[1], b[1])+geometryCompareEpsilon &&
		point[1] >= math.Min(a[1], b[1])-geometryCompareEpsilon
}

func samePoint(a, b []float64) bool {
	return len(a) == 2 && len(b) == 2 &&
		math.Abs(a[0]-b[0]) <= geometryCompareEpsilon &&
		math.Abs(a[1]-b[1]) <= geometryCompareEpsilon
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
