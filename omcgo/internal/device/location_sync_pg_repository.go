package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/outbox"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgLocationObservationRepository struct {
	pool *pgxpool.Pool
}

func NewPgLocationObservationRepository(pool *pgxpool.Pool) *PgLocationObservationRepository {
	return &PgLocationObservationRepository{pool: pool}
}

// UpsertLatest retains source compatibility for concrete callers while routing
// every write through the transactional observation/outbox contract.
func (r *PgLocationObservationRepository) UpsertLatest(
	ctx context.Context,
	deviceID uuid.UUID,
	observation ReportedLocation,
) error {
	_, err := r.SaveLatestWithOutbox(ctx, deviceID, observation)
	return err
}

func (r *PgLocationObservationRepository) SaveLatestWithOutbox(
	ctx context.Context,
	deviceID uuid.UUID,
	observation ReportedLocation,
) (LocationObservationWriteResult, error) {
	observation = normalizeReportedLocation(observation, time.Now().UTC())
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("begin location observation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	deviceSQL, deviceArgs, err := storage.Psql.
		Select("serial_number", "carrier", "location_source_mode").
		From("devices").
		Where(sq.Eq{"id": deviceID, "deleted_at": nil}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("build lock device for location observation: %w", err)
	}
	var serialNumber, carrier string
	var sourceMode model.LocationSourceMode
	if err := tx.QueryRow(ctx, deviceSQL, deviceArgs...).Scan(
		&serialNumber,
		&carrier,
		&sourceMode,
	); err != nil {
		if err == pgx.ErrNoRows {
			return LocationObservationWriteResult{}, commonerrors.ErrNotFound
		}
		return LocationObservationWriteResult{}, fmt.Errorf("lock device for location observation: %w", err)
	}
	if !locationSourceAllowed(sourceMode, observation.SourcePath) {
		return LocationObservationWriteResult{}, fmt.Errorf(
			"source %q is not allowed in %s mode: %w",
			observation.SourcePath,
			sourceMode,
			ErrLocationSourceNotAllowed,
		)
	}

	previous, err := getLatestObservation(ctx, tx, deviceID, true)
	if err != nil {
		return LocationObservationWriteResult{}, err
	}
	if previous != nil && observation.ObservedAt.Before(previous.ObservedAt) {
		return LocationObservationWriteResult{}, fmt.Errorf(
			"location observation is older than stored observation: %w",
			ErrStaleLocationObservation,
		)
	}
	if previous != nil && observation.ObservedAt.Equal(previous.ObservedAt) {
		return LocationObservationWriteResult{
			Current:  *previous,
			Previous: previous,
		}, nil
	}

	saveSQL, saveArgs, err := storage.Psql.
		Insert("device_location_observations").
		Columns(
			"device_id",
			"latitude",
			"longitude",
			"gps_height",
			"observed_at",
			"received_at",
			"device_reported_at",
			"gps_lock_status",
			"satellite_count",
			"accuracy_meters",
			"version",
			"source_path",
		).
		Values(
			deviceID,
			observation.Latitude,
			observation.Longitude,
			observation.GPSHeight,
			observation.ObservedAt,
			observation.ReceivedAt,
			observation.DeviceReportedAt,
			observation.GPSLockStatus,
			observation.SatelliteCount,
			observation.AccuracyMeters,
			1,
			observation.SourcePath,
		).
		Suffix(`
ON CONFLICT (device_id) DO UPDATE SET
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    gps_height = EXCLUDED.gps_height,
    observed_at = EXCLUDED.observed_at,
    received_at = EXCLUDED.received_at,
    device_reported_at = EXCLUDED.device_reported_at,
    gps_lock_status = EXCLUDED.gps_lock_status,
    satellite_count = EXCLUDED.satellite_count,
    accuracy_meters = EXCLUDED.accuracy_meters,
    version = device_location_observations.version + 1,
    source_path = EXCLUDED.source_path
RETURNING latitude, longitude, gps_height, observed_at, received_at,
          device_reported_at, gps_lock_status, satellite_count,
          accuracy_meters, version, source_path`).
		ToSql()
	if err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("build save location observation: %w", err)
	}

	var current ReportedLocation
	if err := tx.QueryRow(ctx, saveSQL, saveArgs...).Scan(
		&current.Latitude,
		&current.Longitude,
		&current.GPSHeight,
		&current.ObservedAt,
		&current.ReceivedAt,
		&current.DeviceReportedAt,
		&current.GPSLockStatus,
		&current.SatelliteCount,
		&current.AccuracyMeters,
		&current.Version,
		&current.SourcePath,
	); err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("save location observation: %w", err)
	}

	distance, elapsed, speed := movementEvidence(previous, current)
	result := LocationObservationWriteResult{
		Current:           current,
		Previous:          previous,
		MovementDistanceM: distance,
		ElapsedSeconds:    elapsed,
		ImpliedSpeedMPS:   speed,
	}
	payload := buildLocationObservedPayload(deviceID, serialNumber, carrier, result)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("marshal location observed event: %w", err)
	}
	dedupeKey := fmt.Sprintf(
		"%s:%s:%d",
		event.SubjectDeviceLocationObserved,
		deviceID,
		current.Version,
	)
	if err := outbox.NewRepository().InsertTx(ctx, tx, outbox.Record{
		AggregateType: "device",
		AggregateID:   deviceID.String(),
		Subject:       event.SubjectDeviceLocationObserved,
		Payload:       payloadJSON,
		DedupeKey:     dedupeKey,
	}); err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("enqueue location observed event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LocationObservationWriteResult{}, fmt.Errorf("commit location observation: %w", err)
	}
	return result, nil
}

func locationSourceAllowed(mode model.LocationSourceMode, sourcePath string) bool {
	isExternal := strings.HasPrefix(
		strings.ToLower(strings.TrimSpace(sourcePath)),
		"third_party:",
	)
	if mode == model.LocationSourceExternal {
		return isExternal
	}
	return mode == model.LocationSourceTR069 && !isExternal
}

func (r *PgLocationObservationRepository) GetLatest(
	ctx context.Context,
	deviceID uuid.UUID,
) (*ReportedLocation, error) {
	return getLatestObservation(ctx, r.pool, deviceID, false)
}

func (r *PgLocationObservationRepository) Accept(
	ctx context.Context,
	deviceID uuid.UUID,
	reportedVersion int64,
) (*LocationSync, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin location sync transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	deviceSQL, deviceArgs, err := storage.Psql.
		Select("latitude", "longitude").
		From("devices").
		Where(sq.Eq{"id": deviceID, "deleted_at": nil}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock device for location sync: %w", err)
	}
	var acceptedLatitude, acceptedLongitude *float64
	if err := tx.QueryRow(ctx, deviceSQL, deviceArgs...).Scan(
		&acceptedLatitude,
		&acceptedLongitude,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("lock device for location sync: %w", err)
	}

	reported, err := getLatestObservation(ctx, tx, deviceID, true)
	if err != nil {
		return nil, err
	}
	if reported == nil {
		return nil, fmt.Errorf("reported location not found: %w", commonerrors.ErrNotFound)
	}
	if reported.Version != reportedVersion {
		return nil, fmt.Errorf(
			"reported location version conflict: expected %d, got %d: %w",
			reportedVersion,
			reported.Version,
			commonerrors.ErrAlreadyExists,
		)
	}

	updateSQL, updateArgs, err := storage.Psql.
		Update("devices").
		Set("latitude", reported.Latitude).
		Set("longitude", reported.Longitude).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": deviceID, "deleted_at": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build accept location update: %w", err)
	}
	if _, err := tx.Exec(ctx, updateSQL, updateArgs...); err != nil {
		return nil, fmt.Errorf("accept reported location: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit location sync: %w", err)
	}

	acceptedBefore := locationFromDeviceCoordinates(acceptedLatitude, acceptedLongitude, nil)
	accepted := &Location{Latitude: reported.Latitude, Longitude: reported.Longitude}
	result := CompareLocations(accepted, reported)
	result.AcceptedBefore = acceptedBefore
	return &result, nil
}

type locationObservationQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func locationObservationColumns() []string {
	return []string{
		"latitude",
		"longitude",
		"gps_height",
		"observed_at",
		"received_at",
		"device_reported_at",
		"gps_lock_status",
		"satellite_count",
		"accuracy_meters",
		"version",
		"source_path",
	}
}

func getLatestObservation(
	ctx context.Context,
	querier locationObservationQuerier,
	deviceID uuid.UUID,
	forUpdate bool,
) (*ReportedLocation, error) {
	builder := storage.Psql.
		Select(locationObservationColumns()...).
		From("device_location_observations").
		Where(sq.Eq{"device_id": deviceID}).
		Limit(1)
	if forUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get location observation: %w", err)
	}

	var observation ReportedLocation
	err = querier.QueryRow(ctx, query, args...).Scan(
		&observation.Latitude,
		&observation.Longitude,
		&observation.GPSHeight,
		&observation.ObservedAt,
		&observation.ReceivedAt,
		&observation.DeviceReportedAt,
		&observation.GPSLockStatus,
		&observation.SatelliteCount,
		&observation.AccuracyMeters,
		&observation.Version,
		&observation.SourcePath,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get location observation: %w", err)
	}
	return &observation, nil
}

func buildLocationObservedPayload(
	deviceID uuid.UUID,
	serialNumber string,
	carrier string,
	result LocationObservationWriteResult,
) event.DeviceLocationObservedPayload {
	payload := event.DeviceLocationObservedPayload{
		DeviceID:               deviceID,
		SerialNumber:           serialNumber,
		Carrier:                carrier,
		ObservationVersion:     result.Current.Version,
		Latitude:               result.Current.Latitude,
		Longitude:              result.Current.Longitude,
		GPSHeight:              result.Current.GPSHeight,
		GPSLockStatus:          result.Current.GPSLockStatus,
		SatelliteCount:         result.Current.SatelliteCount,
		AccuracyMeters:         result.Current.AccuracyMeters,
		ObservedAt:             result.Current.ObservedAt,
		ReceivedAt:             result.Current.ReceivedAt,
		DeviceReportedAt:       result.Current.DeviceReportedAt,
		SourcePath:             result.Current.SourcePath,
		MovementDistanceMeters: result.MovementDistanceM,
		ElapsedSeconds:         result.ElapsedSeconds,
		ImpliedSpeedMPS:        result.ImpliedSpeedMPS,
	}
	if result.Previous != nil {
		version := result.Previous.Version
		payload.PreviousObservationVersion = &version
	}
	return payload
}
