// pg_system_license_usage.go — 累计使用时长 + 时间回拨检测的持久化（复刻旧项目
// small_cell.check_au_info 的 use_duaration / last_visited_time）。
//
// 单行表（id 固定 1）。enforcer 在过期检查时 compute-on-read：按 now -
// last_visited_time 累加 use_duration_hours；若 now < last_visited_time 判定回拨。
package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemLicenseUsageRepository 持久化累计使用时长状态。
type SystemLicenseUsageRepository interface {
	// LastVisited 返回上次检查时间；行不存在（首次）返回 now 兜底，避免误判回拨。
	LastVisited(ctx context.Context) (time.Time, error)
	// Advance 原子累加 addHours（负值按 0 计）、last_visited_time=now，返回累加后总时长（小时）。
	Advance(ctx context.Context, addHours float64) (float64, error)
	// CurrentUsage 只读返回 (totalUsedHours, lastVisited)；行不存在返回 (0, now, nil)。
	CurrentUsage(ctx context.Context) (totalUsedHours float64, lastVisited time.Time, err error)
}

type PgSystemLicenseUsageRepository struct {
	pool *pgxpool.Pool
}

func NewPgSystemLicenseUsageRepository(pool *pgxpool.Pool) *PgSystemLicenseUsageRepository {
	return &PgSystemLicenseUsageRepository{pool: pool}
}

func (r *PgSystemLicenseUsageRepository) LastVisited(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := r.pool.QueryRow(ctx,
		"SELECT last_visited_time FROM system_license_usage WHERE id = 1").Scan(&t)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nowFunc(), nil
		}
		return time.Time{}, fmt.Errorf("get license usage last_visited: %w", err)
	}
	return t, nil
}

func (r *PgSystemLicenseUsageRepository) Advance(ctx context.Context, addHours float64) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO system_license_usage (id, use_duration_hours, last_visited_time, updated_at)
		VALUES (1, GREATEST($1, 0), now(), now())
		ON CONFLICT (id) DO UPDATE
		  SET use_duration_hours = system_license_usage.use_duration_hours + GREATEST($1, 0),
		      last_visited_time = now(),
		      updated_at = now()
		RETURNING use_duration_hours`, addHours).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("advance license usage: %w", err)
	}
	return total, nil
}

func (r *PgSystemLicenseUsageRepository) CurrentUsage(ctx context.Context) (float64, time.Time, error) {
	var total float64
	var lastVisited time.Time
	err := r.pool.QueryRow(ctx,
		"SELECT use_duration_hours, last_visited_time FROM system_license_usage WHERE id = 1").
		Scan(&total, &lastVisited)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nowFunc(), nil
		}
		return 0, time.Time{}, fmt.Errorf("get license usage: %w", err)
	}
	return total, lastVisited, nil
}
