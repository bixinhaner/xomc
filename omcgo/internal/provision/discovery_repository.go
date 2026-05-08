package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// ParameterDiscoveryLogRepository defines the persistence interface for discovery logs.
//
// T-0098 P5-02：data_model_id 列已从 parameter_discovery_log DROP；
// ParameterDiscoveryLog.DataModelID 字段不再持久化（保留 zero value 仅供 in-memory 兼容）。
type ParameterDiscoveryLogRepository interface {
	Create(ctx context.Context, log *ParameterDiscoveryLog) error
	Update(ctx context.Context, log *ParameterDiscoveryLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*ParameterDiscoveryLog, error)
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ParameterDiscoveryLog, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status DiscoveryStatus, errMsg string) error
}

// PgParameterDiscoveryLogRepository implements ParameterDiscoveryLogRepository using PostgreSQL.
type PgParameterDiscoveryLogRepository struct {
	pool *pgxpool.Pool
}

// NewPgParameterDiscoveryLogRepository creates a new PostgreSQL-backed discovery log repository.
func NewPgParameterDiscoveryLogRepository(pool *pgxpool.Pool) *PgParameterDiscoveryLogRepository {
	return &PgParameterDiscoveryLogRepository{
		pool: pool,
	}
}

func (r *PgParameterDiscoveryLogRepository) Create(ctx context.Context, log *ParameterDiscoveryLog) error {
	query, args, err := storage.Psql.Insert("parameter_discovery_log").
		Columns("id", "device_id", "device_sn", "oui", "product_class", "firmware_version",
			"parameter_count", "status", "error_message").
		Values(log.ID, log.DeviceID, log.DeviceSN, log.OUI, log.ProductClass, log.FirmwareVersion,
			log.ParameterCount, log.Status, log.ErrorMessage).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert discovery log: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert discovery log: %w", err)
	}
	return nil
}

func (r *PgParameterDiscoveryLogRepository) Update(ctx context.Context, log *ParameterDiscoveryLog) error {
	query, args, err := storage.Psql.Update("parameter_discovery_log").
		Set("parameter_count", log.ParameterCount).
		Set("status", log.Status).
		Set("error_message", log.ErrorMessage).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": log.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update discovery log: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update discovery log: %w", err)
	}
	return nil
}

func (r *PgParameterDiscoveryLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*ParameterDiscoveryLog, error) {
	query, args, err := storage.Psql.Select(
		"id", "device_id", "device_sn", "oui", "product_class", "firmware_version",
		"parameter_count", "status", "error_message", "created_at", "updated_at",
	).From("parameter_discovery_log").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select discovery log: %w", err)
	}
	return r.scanOne(ctx, query, args...)
}

func (r *PgParameterDiscoveryLogRepository) GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ParameterDiscoveryLog, error) {
	query, args, err := storage.Psql.Select(
		"id", "device_id", "device_sn", "oui", "product_class", "firmware_version",
		"parameter_count", "status", "error_message", "created_at", "updated_at",
	).From("parameter_discovery_log").
		Where(squirrel.Eq{"device_id": deviceID}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select discovery log by device: %w", err)
	}
	return r.scanOne(ctx, query, args...)
}

func (r *PgParameterDiscoveryLogRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status DiscoveryStatus, errMsg string) error {
	query, args, err := storage.Psql.Update("parameter_discovery_log").
		Set("status", status).
		Set("error_message", errMsg).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update discovery log status: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update discovery log status: %w", err)
	}
	return nil
}

func (r *PgParameterDiscoveryLogRepository) scanOne(ctx context.Context, query string, args ...interface{}) (*ParameterDiscoveryLog, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	var log ParameterDiscoveryLog
	if err := row.Scan(
		&log.ID, &log.DeviceID, &log.DeviceSN, &log.OUI, &log.ProductClass, &log.FirmwareVersion,
		&log.ParameterCount, &log.Status, &log.ErrorMessage, &log.CreatedAt, &log.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan discovery log: %w", err)
	}
	return &log, nil
}
