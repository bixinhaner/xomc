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
	"github.com/omcgo/omcgo/internal/core/storage"
)

// UPSDeviceInfoRepository stores UPS-only local device information.
// It intentionally does not use the radio-oriented device_info table.
type UPSDeviceInfoRepository interface {
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error)
	CreateForDevice(ctx context.Context, device *model.Device) error
	UpdateManualFields(ctx context.Context, deviceID uuid.UUID, serialNumber string, req UpdateDeviceInfoRequest, updater string) error
	RecordOnline(ctx context.Context, device *model.Device, at time.Time) error
}

type PgUPSDeviceInfoRepository struct {
	pool *pgxpool.Pool
}

func NewPgUPSDeviceInfoRepository(pool *pgxpool.Pool) *PgUPSDeviceInfoRepository {
	return &PgUPSDeviceInfoRepository{pool: pool}
}

func (r *PgUPSDeviceInfoRepository) GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error) {
	if r == nil || r.pool == nil {
		return nil, nil
	}
	const query = `SELECT
		device_id, device_name, address, remark,
		first_online_time, last_online_time, last_offline_time,
		run_time, cumulative_online_duration,
		creator, updater, created_at, updated_at
	FROM device_ups_info
	WHERE device_id = $1`
	var info DeviceInfo
	var deviceName, address, remark, creator, updater *string
	err := r.pool.QueryRow(ctx, query, deviceID).Scan(
		&info.DeviceID,
		&deviceName, &address, &remark,
		&info.FirstOnlineTime, &info.LastOnlineTime, &info.LastOfflineTime,
		&info.RunTime, &info.CumulativeOnlineDuration,
		&creator, &updater, &info.CreatedAt, &info.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get UPS device info: %w", err)
	}
	if deviceName != nil {
		info.DeviceName = *deviceName
	}
	if address != nil {
		info.Address = *address
	}
	if remark != nil {
		info.Remark = *remark
	}
	if creator != nil {
		info.Creator = *creator
	}
	if updater != nil {
		info.Updater = *updater
	}
	return &info, nil
}

func (r *PgUPSDeviceInfoRepository) CreateForDevice(ctx context.Context, device *model.Device) error {
	if r == nil || r.pool == nil || device == nil {
		return nil
	}
	const query = `INSERT INTO device_ups_info (
		device_id, device_serial_number, device_name, site_id, created_at, updated_at
	) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NOW(), NOW())
	ON CONFLICT (device_id) DO UPDATE SET
		device_serial_number = EXCLUDED.device_serial_number,
		device_name = COALESCE(device_ups_info.device_name, EXCLUDED.device_name),
		site_id = COALESCE(device_ups_info.site_id, EXCLUDED.site_id),
		updated_at = NOW()`
	if _, err := r.pool.Exec(ctx, query, device.ID, device.SerialNumber, device.DeviceName, device.SiteID); err != nil {
		return fmt.Errorf("create UPS device info: %w", err)
	}
	return nil
}

func (r *PgUPSDeviceInfoRepository) UpdateManualFields(ctx context.Context, deviceID uuid.UUID, serialNumber string, req UpdateDeviceInfoRequest, updater string) error {
	if r == nil || r.pool == nil {
		return nil
	}
	if err := r.ensure(ctx, deviceID, serialNumber); err != nil {
		return err
	}
	builder := storage.Psql.Update("device_ups_info").Where(sq.Eq{"device_id": deviceID})
	if req.DeviceName != nil {
		builder = builder.Set("device_name", *req.DeviceName)
	}
	if req.SiteID != nil {
		builder = builder.Set("site_id", *req.SiteID)
	}
	if req.Address != nil {
		builder = builder.Set("address", *req.Address)
	}
	if req.Remark != nil {
		builder = builder.Set("remark", *req.Remark)
	}
	builder = builder.Set("device_serial_number", serialNumber).
		Set("updater", updater).
		Set("updated_at", time.Now())
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update UPS device info: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update UPS device info: %w", err)
	}
	return nil
}

func (r *PgUPSDeviceInfoRepository) RecordOnline(ctx context.Context, device *model.Device, at time.Time) error {
	if r == nil || r.pool == nil || device == nil {
		return nil
	}
	const query = `INSERT INTO device_ups_info (
		device_id, device_serial_number, device_name, site_id,
		first_online_time, last_online_time, created_at, updated_at
	) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $5, NOW(), NOW())
	ON CONFLICT (device_id) DO UPDATE SET
		device_serial_number = EXCLUDED.device_serial_number,
		device_name = COALESCE(device_ups_info.device_name, EXCLUDED.device_name),
		site_id = COALESCE(device_ups_info.site_id, EXCLUDED.site_id),
		first_online_time = COALESCE(device_ups_info.first_online_time, EXCLUDED.first_online_time),
		last_online_time = EXCLUDED.last_online_time,
		updated_at = NOW()`
	if _, err := r.pool.Exec(ctx, query, device.ID, device.SerialNumber, device.DeviceName, device.SiteID, at); err != nil {
		return fmt.Errorf("record UPS online: %w", err)
	}
	return nil
}

func (r *PgUPSDeviceInfoRepository) ensure(ctx context.Context, deviceID uuid.UUID, serialNumber string) error {
	const query = `INSERT INTO device_ups_info (
		device_id, device_serial_number, created_at, updated_at
	) VALUES ($1, NULLIF($2, ''), NOW(), NOW())
	ON CONFLICT (device_id) DO UPDATE SET
		device_serial_number = COALESCE(NULLIF(EXCLUDED.device_serial_number, ''), device_ups_info.device_serial_number),
		updated_at = NOW()`
	if _, err := r.pool.Exec(ctx, query, deviceID, serialNumber); err != nil {
		return fmt.Errorf("ensure UPS device info: %w", err)
	}
	return nil
}
