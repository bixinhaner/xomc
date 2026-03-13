package device

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
)

// PgDeviceParameterRepository implements DeviceParameterRepository using PostgreSQL.
type PgDeviceParameterRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeviceParameterRepository creates a new PostgreSQL device parameter repository.
func NewPgDeviceParameterRepository(pool *pgxpool.Pool) *PgDeviceParameterRepository {
	return &PgDeviceParameterRepository{pool: pool}
}

func (r *PgDeviceParameterRepository) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	if len(params) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	now := time.Now()

	for _, p := range params {
		query, args, err := psql.Insert("device_parameters").
			Columns("device_id", "parameter_path", "parameter_value", "parameter_type", "writable", "last_updated_at").
			Values(deviceID, p.ParameterPath, p.ParameterValue, p.ParameterType, p.Writable, now).
			Suffix("ON CONFLICT (device_id, parameter_path) DO UPDATE SET parameter_value = EXCLUDED.parameter_value, parameter_type = EXCLUDED.parameter_type, writable = EXCLUDED.writable, last_updated_at = EXCLUDED.last_updated_at").
			ToSql()
		if err != nil {
			return fmt.Errorf("build upsert query: %w", err)
		}
		batch.Queue(query, args...)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(params); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("exec batch upsert item %d: %w", i, err)
		}
	}
	return nil
}

func (r *PgDeviceParameterRepository) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	query, args, err := psql.Select("device_id", "parameter_path", "parameter_value", "parameter_type", "writable", "last_updated_at").
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device parameters: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		var p model.DeviceParameter
		if err := rows.Scan(&p.DeviceID, &p.ParameterPath, &p.ParameterValue, &p.ParameterType, &p.Writable, &p.LastUpdatedAt); err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}

	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}

func (r *PgDeviceParameterRepository) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	query, args, err := psql.Select("device_id", "parameter_path", "parameter_value", "parameter_type", "writable", "last_updated_at").
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID, "parameter_path": path}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var p model.DeviceParameter
	err = r.pool.QueryRow(ctx, query, args...).Scan(&p.DeviceID, &p.ParameterPath, &p.ParameterValue, &p.ParameterType, &p.Writable, &p.LastUpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query parameter: %w", err)
	}
	return &p, nil
}

func (r *PgDeviceParameterRepository) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error {
	query, args, _ := psql.Delete("device_parameters").Where(sq.Eq{"device_id": deviceID}).ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}
