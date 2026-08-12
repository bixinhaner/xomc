package device

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

var ErrStaleLocationObservation = errors.New("stale location observation")
var ErrLocationSourceNotAllowed = errors.New("location source is not allowed for device mode")

const (
	LocationSyncNoReport    LocationSyncStatus = "no_report"
	LocationSyncInitialized LocationSyncStatus = "initialized"
	LocationSyncInSync      LocationSyncStatus = "in_sync"
	LocationSyncPending     LocationSyncStatus = "pending"

	locationSyncDistanceThresholdMeters = 10.0
	locationSyncHeightThresholdMeters   = 10.0
)

type LocationSyncStatus string

// Location is the coordinate pair currently accepted by OMC.
type Location struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	GPSHeight *float64 `json:"gps_height,omitempty"`
}

// ReportedLocation is the latest valid coordinate pair observed from a device.
type ReportedLocation struct {
	Latitude         float64    `json:"latitude"`
	Longitude        float64    `json:"longitude"`
	GPSHeight        *float64   `json:"gps_height,omitempty"`
	GPSLockStatus    *string    `json:"gps_lock_status,omitempty"`
	SatelliteCount   *int       `json:"satellite_count,omitempty"`
	AccuracyMeters   *float64   `json:"accuracy_meters,omitempty"`
	ObservedAt       time.Time  `json:"observed_at"`
	ReceivedAt       time.Time  `json:"received_at"`
	DeviceReportedAt *time.Time `json:"device_reported_at,omitempty"`
	Version          int64      `json:"version"`
	SourcePath       string     `json:"source_path"`
}

type LocationObservationWriteResult struct {
	Current           ReportedLocation
	Previous          *ReportedLocation
	MovementDistanceM *float64
	ElapsedSeconds    *float64
	ImpliedSpeedMPS   *float64
}

type LocationSync struct {
	Status   LocationSyncStatus `json:"status"`
	Accepted *Location          `json:"accepted,omitempty"`
	// AcceptedBefore is retained for audit logging after a successful promotion;
	// it is intentionally excluded from the public response payload.
	AcceptedBefore   *Location         `json:"-"`
	Reported         *ReportedLocation `json:"reported,omitempty"`
	DistanceMeters   *float64          `json:"distance_meters,omitempty"`
	HeightDiffMeters *float64          `json:"height_diff_meters,omitempty"`
}

func locationFromDeviceCoordinates(latitude, longitude *float64, gpsHeight *float64) *Location {
	if latitude == nil || longitude == nil {
		return nil
	}
	return &Location{Latitude: *latitude, Longitude: *longitude, GPSHeight: gpsHeight}
}

type gpsCoordinatePathPair struct {
	latitude  string
	longitude string
	source    string
}

var gpsCoordinatePathPairs = []gpsCoordinatePathPair{
	{
		latitude:  "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude",
		longitude: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude",
		source:    "Device.DeviceInfo.SAS.FAP.GPS",
	},
	{
		latitude:  "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude2",
		longitude: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude2",
		source:    "Device.DeviceInfo.SAS.FAP.GPS.2",
	},
	{
		latitude:  "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude3",
		longitude: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude3",
		source:    "Device.DeviceInfo.SAS.FAP.GPS.3",
	},
	{
		latitude:  "Device.FAP.GPS.LockedLatitude",
		longitude: "Device.FAP.GPS.LockedLongitude",
		source:    "Device.FAP.GPS",
	},
	{
		latitude:  "Device.FAP.GPS.LockedLatitude2",
		longitude: "Device.FAP.GPS.LockedLongitude2",
		source:    "Device.FAP.GPS.2",
	},
	{
		latitude:  "Device.FAP.GPS.LockedLatitude3",
		longitude: "Device.FAP.GPS.LockedLongitude3",
		source:    "Device.FAP.GPS.3",
	},
}

// LookupGPSCoordinates accepts both the vendor path and the canonical SAS path.
// A complete, valid pair is required; (0,0) is treated as no successful scan.
func LookupGPSCoordinates(paramValues map[string]string) (float64, float64, string, bool) {
	for _, pair := range gpsCoordinatePathPairs {
		latitude, latOK := ParseGPSCoordinate(paramValues[pair.latitude], 90)
		longitude, lngOK := ParseGPSCoordinate(paramValues[pair.longitude], 180)
		if !latOK || !lngOK || (latitude == 0 && longitude == 0) {
			continue
		}
		return latitude, longitude, pair.source, true
	}
	return 0, 0, "", false
}

// ParseGPSCoordinate accepts decimal degrees and TR-181 millionths of a degree.
func ParseGPSCoordinate(raw string, maxAbs float64) (float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}

	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	if math.Abs(value) <= maxAbs {
		return value, true
	}
	// Values outside the decimal-degree range are accepted as millionths only
	// when they are integer-form values with at least six significant digits.
	// This prevents an invalid decimal such as "91" from being reinterpreted
	// as 0.000091 degrees while retaining the TR-181 integer representation.
	if isMillionthsCoordinate(trimmed) && math.Abs(value) <= maxAbs*gpsCoordinateScale {
		return value / gpsCoordinateScale, true
	}
	return 0, false
}

func isMillionthsCoordinate(raw string) bool {
	if strings.ContainsAny(raw, ".eE") {
		return false
	}
	digits := strings.TrimLeft(raw, "+-")
	digits = strings.TrimLeft(digits, "0")
	return len(digits) >= 6
}

// CompareLocations computes the server-side state used by both list and detail APIs.
func CompareLocations(accepted *Location, reported *ReportedLocation) LocationSync {
	result := LocationSync{Accepted: accepted, Reported: reported}
	if reported == nil {
		result.Status = LocationSyncNoReport
		return result
	}
	if accepted == nil {
		result.Status = LocationSyncInitialized
		return result
	}

	distance := haversineMeters(accepted.Latitude, accepted.Longitude, reported.Latitude, reported.Longitude)
	result.DistanceMeters = &distance
	if accepted.GPSHeight != nil && reported.GPSHeight != nil {
		heightDiff := math.Abs(*accepted.GPSHeight - *reported.GPSHeight)
		result.HeightDiffMeters = &heightDiff
		if heightDiff > locationSyncHeightThresholdMeters {
			result.Status = LocationSyncPending
			return result
		}
	}
	if distance > locationSyncDistanceThresholdMeters {
		result.Status = LocationSyncPending
		return result
	}
	result.Status = LocationSyncInSync
	return result
}

func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMeters = 6_371_000.0
	toRadians := func(value float64) float64 { return value * math.Pi / 180 }
	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)
	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func normalizeReportedLocation(observation ReportedLocation, receivedFallback time.Time) ReportedLocation {
	if observation.ReceivedAt.IsZero() {
		observation.ReceivedAt = receivedFallback
	}
	if observation.DeviceReportedAt != nil {
		observation.ObservedAt = *observation.DeviceReportedAt
	} else if observation.ObservedAt.IsZero() {
		observation.ObservedAt = observation.ReceivedAt
	}
	return observation
}

func movementEvidence(previous *ReportedLocation, current ReportedLocation) (*float64, *float64, *float64) {
	if previous == nil {
		return nil, nil, nil
	}
	distance := haversineMeters(
		previous.Latitude,
		previous.Longitude,
		current.Latitude,
		current.Longitude,
	)
	elapsed := current.ObservedAt.Sub(previous.ObservedAt).Seconds()
	if elapsed <= 0 {
		return &distance, nil, nil
	}
	speed := distance / elapsed
	return &distance, &elapsed, &speed
}
