package device

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgLocationObservationRepository struct {
	pool *pgxpool.Pool
}

func NewPgLocationObservationRepository(pool *pgxpool.Pool) *PgLocationObservationRepository {
	return &PgLocationObservationRepository{pool: pool}
}

func (r *PgLocationObservationRepository) UpsertLatest(ctx context.Context, deviceID uuid.UUID, observation ReportedLocation) error {
	observedAt := observation.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	query, args, err := storage.Psql.Insert("device_location_observations").
		Columns("device_id", "latitude", "longitude", "gps_height", "observed_at", "version", "source_path").
		Values(deviceID, observation.Latitude, observation.Longitude, observation.GPSHeight, observedAt, 1, observation.SourcePath).
		Suffix("ON CONFLICT (device_id) DO UPDATE SET " +
			"latitude = EXCLUDED.latitude, " +
			"longitude = EXCLUDED.longitude, " +
			"gps_height = EXCLUDED.gps_height, " +
			"observed_at = EXCLUDED.observed_at, " +
			"version = device_location_observations.version + 1, " +
			"source_path = EXCLUDED.source_path").
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert location observation: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert location observation: %w", err)
	}
	return nil
}

func (r *PgLocationObservationRepository) GetLatest(ctx context.Context, deviceID uuid.UUID) (*ReportedLocation, error) {
	query, args, err := storage.Psql.Select(
		"latitude", "longitude", "gps_height", "observed_at", "version", "source_path",
	).
		From("device_location_observations").
		Where(sq.Eq{"device_id": deviceID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get location observation: %w", err)
	}

	var observation ReportedLocation
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&observation.Latitude,
		&observation.Longitude,
		&observation.GPSHeight,
		&observation.ObservedAt,
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

func (r *PgLocationObservationRepository) Accept(ctx context.Context, deviceID uuid.UUID, reportedVersion int64) (*LocationSync, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin location sync transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var acceptedLatitude, acceptedLongitude *float64
	if err := tx.QueryRow(ctx,
		"SELECT d.latitude, d.longitude FROM devices d WHERE d.id = $1 AND d.deleted_at IS NULL FOR UPDATE",
		deviceID,
	).Scan(&acceptedLatitude, &acceptedLongitude); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("lock device for location sync: %w", err)
	}

	var reported ReportedLocation
	if err := tx.QueryRow(ctx,
		"SELECT latitude, longitude, gps_height, observed_at, version, source_path FROM device_location_observations WHERE device_id = $1 FOR UPDATE",
		deviceID,
	).Scan(&reported.Latitude, &reported.Longitude, &reported.GPSHeight, &reported.ObservedAt, &reported.Version, &reported.SourcePath); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("reported location not found: %w", commonerrors.ErrNotFound)
		}
		return nil, fmt.Errorf("read reported location: %w", err)
	}
	if reported.Version != reportedVersion {
		return nil, fmt.Errorf("reported location version conflict: expected %d, got %d: %w", reportedVersion, reported.Version, commonerrors.ErrAlreadyExists)
	}

	query, args, err := storage.Psql.Update("devices").
		Set("latitude", reported.Latitude).
		Set("longitude", reported.Longitude).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": deviceID}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build accept location update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("accept reported location: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit location sync: %w", err)
	}

	acceptedBefore := locationFromDeviceCoordinates(acceptedLatitude, acceptedLongitude, nil)
	accepted := &Location{Latitude: reported.Latitude, Longitude: reported.Longitude}
	result := CompareLocations(accepted, &reported)
	result.AcceptedBefore = acceptedBefore
	return &result, nil
}
