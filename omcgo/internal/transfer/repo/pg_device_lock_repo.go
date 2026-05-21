package repo

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// ErrDeviceBusy 是 DeviceLockRepo.Acquire 在设备已被占用时的 sentinel error。
// 用 commonerrors.ErrAlreadyExists 是为了让 HTTP 中间件直接映射到 409 Conflict。
var ErrDeviceBusy = commonerrors.ErrAlreadyExists

var _ DeviceLockRepo = (*PgDeviceLockRepo)(nil)

// PgDeviceLockRepo 是 DeviceLockRepo 的 PostgreSQL 实现，操作 device_active_tasks 表。
type PgDeviceLockRepo struct {
	pool *pgxpool.Pool
}

// NewPgDeviceLockRepo 创建跨业务设备锁仓储。
func NewPgDeviceLockRepo(pool *pgxpool.Pool) *PgDeviceLockRepo {
	return &PgDeviceLockRepo{pool: pool}
}

func (r *PgDeviceLockRepo) Acquire(ctx context.Context, lock DeviceLock) error {
	query, args, err := storage.Psql.Insert("device_active_tasks").
		Columns("device_id", "sub_task_id", "business_type", "sub_task_table").
		Values(lock.DeviceID, lock.SubTaskID, lock.BusinessType, lock.SubTaskTable).
		ToSql()
	if err != nil {
		return fmt.Errorf("build acquire device lock SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		// 23505 = unique_violation；device_id 是 PK，表示设备已有活跃任务
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDeviceBusy
		}
		return fmt.Errorf("acquire device lock: %w", err)
	}
	return nil
}

func (r *PgDeviceLockRepo) Release(ctx context.Context, deviceID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("device_active_tasks").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build release device lock SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("release device lock: %w", err)
	}
	return nil
}

func (r *PgDeviceLockRepo) ReleaseBySubTask(ctx context.Context, subTaskID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("device_active_tasks").
		Where(sq.Eq{"sub_task_id": subTaskID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build release device lock by sub_task SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("release device lock by sub_task: %w", err)
	}
	return nil
}

func (r *PgDeviceLockRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) (*DeviceLock, error) {
	query, args, err := storage.Psql.
		Select("device_id", "sub_task_id", "business_type", "sub_task_table").
		From("device_active_tasks").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get device lock SQL: %w", err)
	}
	var lock DeviceLock
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&lock.DeviceID, &lock.SubTaskID, &lock.BusinessType, &lock.SubTaskTable,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get device lock: %w", err)
	}
	return &lock, nil
}
