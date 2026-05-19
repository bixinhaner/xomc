package stationlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository 日志文件 DB 操作接口。
// 由 PgRunningRepository 和 PgFaultRepository 分别实现，每个实例只操作自己对应的表，
// 无需传入 logType 参数（类型隐含于表名中，扫描时自动填充到 LogFile.LogType）。
type Repository interface {
	Create(ctx context.Context, f *LogFile) error
	GetByID(ctx context.Context, id uuid.UUID) (*LogFile, error)
	List(ctx context.Context, filter LogFileFilter) ([]*LogFile, int64, error)
	MarkDeleted(ctx context.Context, id uuid.UUID) error
	// Count 统计本表未删除的记录总数（用于配额管理）
	Count(ctx context.Context) (int64, error)
	// ListOldest 按采集时间升序返回未删除记录（用于配额超额时清理最旧的文件）
	ListOldest(ctx context.Context, limit int) ([]*LogFile, error)
	// LatestByDevice 获取指定设备最近一条未删除记录
	LatestByDevice(ctx context.Context, deviceID uuid.UUID) (*LogFile, error)
}

// runningLogCols 运行日志表列（station_running_logs，无 fault_reason/fault_detail）
var runningLogCols = []string{
	"id", "device_id", "device_sn", "file_name",
	"object_path", "bucket", "file_size",
	"task_id", "is_deleted", "collected_at", "created_at", "updated_at",
}

// faultLogCols 故障日志表列（station_fault_logs，含 fault_reason/fault_detail）
var faultLogCols = []string{
	"id", "device_id", "device_sn", "file_name",
	"object_path", "bucket", "file_size",
	"fault_reason", "fault_detail",
	"task_id", "is_deleted", "collected_at", "created_at", "updated_at",
}

// PgRepository 参数化的 PostgreSQL 仓库实现，运行日志和故障日志各创建一个实例。
type PgRepository struct {
	pool            *pgxpool.Pool
	tableName       string
	logType         LogType // 扫描后自动填充到 LogFile.LogType
	withFaultFields bool    // true: 含 fault_reason/fault_detail 列
}

// NewPgRunningRepository 创建运行日志仓库，操作 station_running_logs 表。
func NewPgRunningRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{
		pool:            pool,
		tableName:       "station_running_logs",
		logType:         LogTypeRunning,
		withFaultFields: false,
	}
}

// NewPgFaultRepository 创建故障日志仓库，操作 station_fault_logs 表。
func NewPgFaultRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{
		pool:            pool,
		tableName:       "station_fault_logs",
		logType:         LogTypeFault,
		withFaultFields: true,
	}
}

func (r *PgRepository) cols() []string {
	if r.withFaultFields {
		return faultLogCols
	}
	return runningLogCols
}

func (r *PgRepository) Create(ctx context.Context, f *LogFile) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	if f.CollectedAt.IsZero() {
		f.CollectedAt = now
	}

	var qb sq.InsertBuilder
	if r.withFaultFields {
		qb = storage.Psql.Insert(r.tableName).
			Columns(faultLogCols...).
			Values(
				f.ID, f.DeviceID, f.DeviceSN, f.FileName,
				f.ObjectPath, f.Bucket, f.FileSize,
				f.FaultReason, f.FaultDetail,
				f.TaskID, f.IsDeleted, f.CollectedAt, f.CreatedAt, f.UpdatedAt,
			)
	} else {
		qb = storage.Psql.Insert(r.tableName).
			Columns(runningLogCols...).
			Values(
				f.ID, f.DeviceID, f.DeviceSN, f.FileName,
				f.ObjectPath, f.Bucket, f.FileSize,
				f.TaskID, f.IsDeleted, f.CollectedAt, f.CreatedAt, f.UpdatedAt,
			)
	}
	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("build insert %s: %w", r.tableName, err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgRepository) GetByID(ctx context.Context, id uuid.UUID) (*LogFile, error) {
	query, args, err := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select %s by id: %w", r.tableName, err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	f, err := r.scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return f, err
}

func (r *PgRepository) List(ctx context.Context, filter LogFileFilter) ([]*LogFile, int64, error) {
	base := storage.Psql.Select(r.cols()...).From(r.tableName).Where(sq.Eq{"is_deleted": false})
	countBase := storage.Psql.Select("COUNT(*)").From(r.tableName).Where(sq.Eq{"is_deleted": false})

	if filter.DeviceID != nil {
		base = base.Where(sq.Eq{"device_id": *filter.DeviceID})
		countBase = countBase.Where(sq.Eq{"device_id": *filter.DeviceID})
	}
	if filter.DeviceSN != "" {
		base = base.Where(sq.Eq{"device_sn": filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}

	countQuery, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count %s: %w", r.tableName, err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", r.tableName, err)
	}

	query, args, err := base.
		OrderBy("collected_at DESC").
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset())).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list %s: %w", r.tableName, err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query %s: %w", r.tableName, err)
	}
	defer rows.Close()

	var items []*LogFile
	for rows.Next() {
		f, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, f)
	}
	return items, total, rows.Err()
}

func (r *PgRepository) MarkDeleted(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Update(r.tableName).
		Set("is_deleted", true).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark deleted %s: %w", r.tableName, err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgRepository) Count(ctx context.Context) (int64, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").
		From(r.tableName).
		Where(sq.Eq{"is_deleted": false}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count %s: %w", r.tableName, err)
	}
	var count int64
	err = r.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *PgRepository) ListOldest(ctx context.Context, limit int) ([]*LogFile, error) {
	query, args, err := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{"is_deleted": false}).
		OrderBy("collected_at ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list oldest %s: %w", r.tableName, err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query oldest %s: %w", r.tableName, err)
	}
	defer rows.Close()

	var items []*LogFile
	for rows.Next() {
		f, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, f)
	}
	return items, rows.Err()
}

func (r *PgRepository) LatestByDevice(ctx context.Context, deviceID uuid.UUID) (*LogFile, error) {
	query, args, err := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{"device_id": deviceID, "is_deleted": false}).
		OrderBy("collected_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest by device %s: %w", r.tableName, err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	f, err := r.scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return f, err
}

// scan 从 pgx.Row 扫描一条记录，并将 r.logType 注入 LogFile.LogType。
func (r *PgRepository) scan(row pgx.Row) (*LogFile, error) {
	var f LogFile
	f.LogType = r.logType
	var err error
	if r.withFaultFields {
		err = row.Scan(
			&f.ID, &f.DeviceID, &f.DeviceSN, &f.FileName,
			&f.ObjectPath, &f.Bucket, &f.FileSize,
			&f.FaultReason, &f.FaultDetail,
			&f.TaskID, &f.IsDeleted, &f.CollectedAt, &f.CreatedAt, &f.UpdatedAt,
		)
	} else {
		err = row.Scan(
			&f.ID, &f.DeviceID, &f.DeviceSN, &f.FileName,
			&f.ObjectPath, &f.Bucket, &f.FileSize,
			&f.TaskID, &f.IsDeleted, &f.CollectedAt, &f.CreatedAt, &f.UpdatedAt,
		)
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
