package geofence

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type RawPosition string

const (
	RawPositionUnknown  RawPosition = "unknown"
	RawPositionInside   RawPosition = "inside"
	RawPositionOutside  RawPosition = "outside"
	RawPositionBoundary RawPosition = "boundary"
)

type ConfirmedState string

const (
	ConfirmedStateUnknown ConfirmedState = "unknown"
	ConfirmedStateInside  ConfirmedState = "inside"
	ConfirmedStateOutside ConfirmedState = "outside"
)

type CandidateState string

const (
	CandidateStateNone    CandidateState = ""
	CandidateStateExit    CandidateState = "exit"
	CandidateStateReentry CandidateState = "reentry"
)

type PositionSnapshot struct {
	Longitude              float64
	Latitude               float64
	ObservationVersion     int64
	ObservedAt             time.Time
	ReceivedAt             time.Time
	DeviceReportedAt       *time.Time
	MovementDistanceMeters *float64
	ImpliedSpeedMPS        *float64
}

type EvaluationPolicy struct {
	ExitToleranceMeters         float64
	ReentryToleranceMeters      float64
	ExitConsecutiveSamples      int
	ReentryConsecutiveSamples   int
	SampleMaxAgeSeconds         int
	MaxJumpDistanceMeters       float64
	MaxImpliedSpeedMPS          float64
	MaxDeviceClockSkewSeconds   int
	MinimumStateDurationSeconds int
}

type EvaluationState struct {
	ConfirmedState         ConfirmedState
	CandidateState         CandidateState
	CandidateCount         int
	CandidateSince         *time.Time
	LastObservationVersion int64
}

type EvaluationInput struct {
	RuleType    RuleType
	Geometry    json.RawMessage
	Position    PositionSnapshot
	Previous    EvaluationState
	Policy      EvaluationPolicy
	EvaluatedAt time.Time
}

type EvaluationResult struct {
	RawPosition          RawPosition
	SignedDistanceMeters *float64
	ConfirmedState       ConfirmedState
	CandidateState       CandidateState
	CandidateCount       int
	CandidateSince       *time.Time
	StateEdge            bool
	Reason               string
	ObservationVersion   int64
}

func Evaluate(input EvaluationInput) (EvaluationResult, error) {
	if input.Position.ObservationVersion > 0 &&
		input.Previous.LastObservationVersion > 0 &&
		input.Position.ObservationVersion <= input.Previous.LastObservationVersion {
		reason := "out_of_order_observation"
		if input.Position.ObservationVersion == input.Previous.LastObservationVersion {
			reason = "duplicate_observation"
		}
		return replayProtectedEvaluation(input, reason), nil
	}
	if reason := positionQualityReason(input); reason != "" {
		return unknownEvaluation(input, reason), nil
	}

	signedDistance, err := signedBoundaryDistance(
		input.RuleType,
		input.Geometry,
		input.Position.Longitude,
		input.Position.Latitude,
	)
	if err != nil {
		return EvaluationResult{}, err
	}
	raw := RawPositionInside
	if signedDistance > input.Policy.ExitToleranceMeters {
		raw = RawPositionOutside
	} else if signedDistance >= -input.Policy.ReentryToleranceMeters {
		raw = RawPositionBoundary
	}
	result := EvaluationResult{
		RawPosition:          raw,
		SignedDistanceMeters: &signedDistance,
		ConfirmedState:       normalizedConfirmedState(input.Previous.ConfirmedState),
		ObservationVersion:   input.Position.ObservationVersion,
		Reason:               "evaluated",
	}
	if raw == RawPositionBoundary {
		result.Reason = "within_hysteresis_boundary"
		return result, nil
	}
	advanceCandidate(&result, input)
	return result, nil
}

func replayProtectedEvaluation(input EvaluationInput, reason string) EvaluationResult {
	return EvaluationResult{
		RawPosition:        RawPositionUnknown,
		ConfirmedState:     normalizedConfirmedState(input.Previous.ConfirmedState),
		CandidateState:     input.Previous.CandidateState,
		CandidateCount:     input.Previous.CandidateCount,
		CandidateSince:     input.Previous.CandidateSince,
		Reason:             reason,
		ObservationVersion: input.Position.ObservationVersion,
	}
}

func advanceCandidate(result *EvaluationResult, input EvaluationInput) {
	target := ConfirmedStateInside
	candidate := CandidateStateReentry
	requiredCount := input.Policy.ReentryConsecutiveSamples
	if result.RawPosition == RawPositionOutside {
		target = ConfirmedStateOutside
		candidate = CandidateStateExit
		requiredCount = input.Policy.ExitConsecutiveSamples
	}
	if requiredCount < 1 {
		requiredCount = 1
	}
	if result.ConfirmedState == target {
		result.Reason = "confirmed_state_unchanged"
		return
	}

	sampleTime := input.Position.ObservedAt
	if sampleTime.IsZero() {
		sampleTime = input.EvaluatedAt
	}
	candidateSince := sampleTime
	candidateCount := 1
	if input.Previous.CandidateState == candidate &&
		input.Previous.CandidateSince != nil {
		candidateSince = *input.Previous.CandidateSince
		candidateCount = input.Previous.CandidateCount + 1
	}
	result.CandidateState = candidate
	result.CandidateCount = candidateCount
	result.CandidateSince = &candidateSince
	result.Reason = "candidate_" + string(candidate)

	minimumDuration := time.Duration(input.Policy.MinimumStateDurationSeconds) * time.Second
	if candidateCount < requiredCount || sampleTime.Sub(candidateSince) < minimumDuration {
		return
	}
	previousConfirmed := result.ConfirmedState
	result.ConfirmedState = target
	result.CandidateState = CandidateStateNone
	result.CandidateCount = 0
	result.CandidateSince = nil
	result.StateEdge = previousConfirmed != target
	result.Reason = "confirmed_" + string(target)
}

func positionQualityReason(input EvaluationInput) string {
	position := input.Position
	if !isFinite(position.Longitude) || !isFinite(position.Latitude) ||
		position.Longitude < -180 || position.Longitude > 180 ||
		position.Latitude < -90 || position.Latitude > 90 ||
		(position.Longitude == 0 && position.Latitude == 0) {
		return "invalid_coordinates"
	}
	if position.ObservedAt.IsZero() {
		return "invalid_observed_at"
	}
	if input.Policy.SampleMaxAgeSeconds > 0 &&
		input.EvaluatedAt.Sub(position.ObservedAt) >
			time.Duration(input.Policy.SampleMaxAgeSeconds)*time.Second {
		return "stale_sample"
	}
	if position.DeviceReportedAt != nil && input.Policy.MaxDeviceClockSkewSeconds > 0 {
		reference := position.ReceivedAt
		if reference.IsZero() {
			reference = input.EvaluatedAt
		}
		if position.DeviceReportedAt.Sub(reference) >
			time.Duration(input.Policy.MaxDeviceClockSkewSeconds)*time.Second {
			return "device_clock_invalid"
		}
	}
	if input.Policy.MaxJumpDistanceMeters > 0 &&
		position.MovementDistanceMeters != nil &&
		*position.MovementDistanceMeters > input.Policy.MaxJumpDistanceMeters {
		return "implausible_jump"
	}
	if input.Policy.MaxImpliedSpeedMPS > 0 &&
		position.ImpliedSpeedMPS != nil &&
		*position.ImpliedSpeedMPS > input.Policy.MaxImpliedSpeedMPS {
		return "implausible_jump"
	}
	return ""
}

func unknownEvaluation(input EvaluationInput, reason string) EvaluationResult {
	return EvaluationResult{
		RawPosition:        RawPositionUnknown,
		ConfirmedState:     normalizedConfirmedState(input.Previous.ConfirmedState),
		CandidateState:     CandidateStateNone,
		CandidateCount:     0,
		CandidateSince:     nil,
		StateEdge:          false,
		Reason:             reason,
		ObservationVersion: input.Position.ObservationVersion,
	}
}

func normalizedConfirmedState(state ConfirmedState) ConfirmedState {
	if state == "" {
		return ConfirmedStateUnknown
	}
	return state
}

func signedBoundaryDistance(
	ruleType RuleType,
	raw json.RawMessage,
	longitude, latitude float64,
) (float64, error) {
	switch ruleType {
	case RuleTypePolygonAllowZone:
		var polygon polygonJSON
		if err := json.Unmarshal(raw, &polygon); err != nil ||
			polygon.Type != "Polygon" ||
			len(polygon.Coordinates) != 1 ||
			len(polygon.Coordinates[0]) < 3 {
			return 0, fmt.Errorf("invalid polygon evaluation snapshot: %w", ErrInvalidGeometry)
		}
		ring := polygon.Coordinates[0]
		distance := distanceToRingMeters(longitude, latitude, ring)
		if pointInRing(longitude, latitude, ring) {
			return -distance, nil
		}
		return distance, nil
	case RuleTypeBaselineRadius:
		var circle circleJSON
		if err := json.Unmarshal(raw, &circle); err != nil ||
			circle.Type != "Circle" ||
			len(circle.Center) != 2 ||
			circle.RadiusMeters <= 0 {
			return 0, fmt.Errorf("invalid circle evaluation snapshot: %w", ErrInvalidGeometry)
		}
		return haversineMeters(
			latitude, longitude,
			circle.Center[1], circle.Center[0],
		) - circle.RadiusMeters, nil
	default:
		return 0, fmt.Errorf("unsupported evaluation rule %q: %w", ruleType, ErrInvalidGeometry)
	}
}

func pointInRing(longitude, latitude float64, ring [][]float64) bool {
	inside := false
	for current, previous := 0, len(ring)-1; current < len(ring); previous, current = current, current+1 {
		a, b := ring[current], ring[previous]
		if len(a) != 2 || len(b) != 2 {
			continue
		}
		intersects := (a[1] > latitude) != (b[1] > latitude) &&
			longitude < (b[0]-a[0])*(latitude-a[1])/(b[1]-a[1])+a[0]
		if intersects {
			inside = !inside
		}
	}
	return inside
}

func distanceToRingMeters(longitude, latitude float64, ring [][]float64) float64 {
	minimum := math.Inf(1)
	for index := range ring {
		next := (index + 1) % len(ring)
		if len(ring[index]) != 2 || len(ring[next]) != 2 {
			continue
		}
		distance := pointToSegmentMeters(
			longitude, latitude,
			ring[index][0], ring[index][1],
			ring[next][0], ring[next][1],
		)
		minimum = math.Min(minimum, distance)
	}
	return minimum
}

func pointToSegmentMeters(px, py, ax, ay, bx, by float64) float64 {
	referenceLatitude := (py + ay + by) / 3 * math.Pi / 180
	toXY := func(longitude, latitude float64) (float64, float64) {
		return longitude * math.Pi / 180 * earthRadiusMeters * math.Cos(referenceLatitude),
			latitude * math.Pi / 180 * earthRadiusMeters
	}
	pX, pY := toXY(px, py)
	aX, aY := toXY(ax, ay)
	bX, bY := toXY(bx, by)
	dX, dY := bX-aX, bY-aY
	if dX == 0 && dY == 0 {
		return math.Hypot(pX-aX, pY-aY)
	}
	fraction := ((pX-aX)*dX + (pY-aY)*dY) / (dX*dX + dY*dY)
	fraction = math.Max(0, math.Min(1, fraction))
	return math.Hypot(pX-(aX+fraction*dX), pY-(aY+fraction*dY))
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Radians := lat1 * math.Pi / 180
	lat2Radians := lat2 * math.Pi / 180
	deltaLatitude := (lat2 - lat1) * math.Pi / 180
	deltaLongitude := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(deltaLatitude/2)*math.Sin(deltaLatitude/2) +
		math.Cos(lat1Radians)*math.Cos(lat2Radians)*
			math.Sin(deltaLongitude/2)*math.Sin(deltaLongitude/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
