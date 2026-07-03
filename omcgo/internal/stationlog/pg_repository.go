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

	"github.com/omcgo/omcgo/internal/authz"
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
	// CountByDevice 统计指定设备本表未删除记录数（用于每设备配额管理，#798）
	CountByDevice(ctx context.Context, deviceID uuid.UUID) (int64, error)
	// ListOldestByDevice 按采集时间升序返回指定设备未删除记录（用于每设备配额超额清理，#798）
	ListOldestByDevice(ctx context.Context, deviceID uuid.UUID, limit int) ([]*LogFile, error)
	// ListExpired 按入库时间(created_at)升序返回早于 cutoff 的未删除记录（用于按时间保留清理，#320）
	ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]*LogFile, error)
	// LatestByDevice 获取指定设备最近一条未删除记录
	LatestByDevice(ctx context.Context, deviceID uuid.UUID) (*LogFile, error)
}

// FaultExtraRepository 故障日志专用接口：提供 UpdateFile 等运行日志不需要的方法。
//
// detected → file_received 的转换在文件到达时通过 UpdateFile 完成；
// LatestDetectedByDeviceSN 在文件到达时用于找到对应的占位记录。
type FaultExtraRepository interface {
	// UpdateFile 用文件信息更新一条 detected 记录，使其进入 file_received 状态。
	UpdateFile(ctx context.Context, id uuid.UUID, fileName, objectPath, bucket string, fileSize int64) error
	// LatestDetectedByDeviceSN 返回设备最近一条 detected 状态（未文件落地）的故障日志。
	LatestDetectedByDeviceSN(ctx context.Context, sn string) (*LogFile, error)
}

// runningLogCols 运行日志表列（station_running_logs，无 fault_reason/fault_detail）
var runningLogCols = []string{
	"id", "device_id", "device_sn", "file_name",
	"object_path", "bucket", "file_size",
	"task_id", "is_deleted", "collected_at", "created_at", "updated_at",
}

// faultLogCols 故障日志表列（station_fault_logs）。
//
// 顺序固定，scan / insert 双向使用。包含 000150 引入的新字段：device_name /
// device_type / is_gnb / operate_ip / software_version / runtime_before_reboot /
// record_status / collection_fail_reason / manual_collection_status。
var faultLogCols = []string{
	"id", "device_id", "device_sn", "file_name",
	"object_path", "bucket", "file_size",
	"fault_reason", "fault_detail",
	"task_id", "is_deleted", "collected_at", "created_at", "updated_at",
	"device_name", "device_type", "is_gnb", "operate_ip", "software_version",
	"runtime_before_reboot",
	"record_status", "collection_fail_reason", "manual_collection_status",
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

// nilIfEmpty 把空字符串转成 nil，便于让 file_name/object_path/bucket 等
// nullable 列在 detected 占位记录里写成 NULL 而非 ”。
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nilIfZero 把 0 转成 nil（用于 runtime_before_reboot 等 BIGINT 列）。
func nilIfZero(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
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
		// 默认值兜底：未指定 record_status 视为 file_received（兼容老路径）。
		recordStatus := f.RecordStatus
		if recordStatus == "" {
			if f.FileName == "" {
				recordStatus = FaultRecordStatusDetected
			} else {
				recordStatus = FaultRecordStatusFileReceived
			}
			f.RecordStatus = recordStatus
		}
		manualStatus := f.ManualCollectionStatus
		if manualStatus == "" {
			manualStatus = ManualCollectionIdle
			f.ManualCollectionStatus = manualStatus
		}

		qb = storage.Psql.Insert(r.tableName).
			Columns(faultLogCols...).
			Values(
				f.ID, f.DeviceID, f.DeviceSN, nilIfEmpty(f.FileName),
				nilIfEmpty(f.ObjectPath), nilIfEmpty(f.Bucket), f.FileSize,
				nilIfEmpty(f.FaultReason), nilIfEmpty(f.FaultDetail),
				f.TaskID, f.IsDeleted, f.CollectedAt, f.CreatedAt, f.UpdatedAt,
				nilIfEmpty(f.DeviceName), nilIfEmpty(f.DeviceType), f.IsGNB,
				nilIfEmpty(f.OperateIP), nilIfEmpty(f.SoftwareVersion),
				nilIfZero(f.RuntimeBeforeReboot),
				recordStatus, nilIfEmpty(f.CollectionFailReason), manualStatus,
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
		// 模糊匹配（ILIKE 大小写不敏感）：让前端输入部分 SN 如 "12028" 也能命中完整 SN
		pattern := "%" + filter.DeviceSN + "%"
		base = base.Where(sq.ILike{"device_sn": pattern})
		countBase = countBase.Where(sq.ILike{"device_sn": pattern})
	}
	if r.withFaultFields {
		if filter.RecordStatus != "" {
			base = base.Where(sq.Eq{"record_status": filter.RecordStatus})
			countBase = countBase.Where(sq.Eq{"record_status": filter.RecordStatus})
		}
		if filter.DeviceType != "" {
			base = base.Where(sq.Eq{"device_type": filter.DeviceType})
			countBase = countBase.Where(sq.Eq{"device_type": filter.DeviceType})
		}
	}
	if filter.StartTime != nil {
		base = base.Where(sq.GtOrEq{"collected_at": *filter.StartTime})
		countBase = countBase.Where(sq.GtOrEq{"collected_at": *filter.StartTime})
	}
	if filter.EndTime != nil {
		base = base.Where(sq.LtOrEq{"collected_at": *filter.EndTime})
		countBase = countBase.Where(sq.LtOrEq{"collected_at": *filter.EndTime})
	}
	// #63 设备组可见性 fail-closed 过滤（device_id 为 NULL 的未关联记录对非超管不可见）。
	base = authz.ApplyDeviceVisibilityFilter(base, "device_id", filter.VisibleGroups)
	countBase = authz.ApplyDeviceVisibilityFilter(countBase, "device_id", filter.VisibleGroups)

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
	// 配额清理只针对 file_received 状态：detected 占位记录没文件需要删，
	// 也不应被算入"超额"配额。在故障日志表上额外限定 record_status；
	// 运行日志表沿用旧逻辑（无 record_status 列）。
	q := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{"is_deleted": false})
	if r.withFaultFields {
		q = q.Where(sq.Eq{"record_status": FaultRecordStatusFileReceived})
	}
	query, args, err := q.
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

// CountByDevice 统计指定设备本表未删除记录数（#798 每设备配额管理）。故障日志表额外限定
// record_status=file_received，与 Count/ListOldest 的口径一致：detected 占位记录没文件，
// 不应被算入配额。
func (r *PgRepository) CountByDevice(ctx context.Context, deviceID uuid.UUID) (int64, error) {
	q := storage.Psql.Select("COUNT(*)").
		From(r.tableName).
		Where(sq.Eq{"device_id": deviceID, "is_deleted": false})
	if r.withFaultFields {
		q = q.Where(sq.Eq{"record_status": FaultRecordStatusFileReceived})
	}
	query, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count by device %s: %w", r.tableName, err)
	}
	var count int64
	err = r.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// ListOldestByDevice 按采集时间升序返回指定设备未删除记录（#798 每设备配额超额清理）。
func (r *PgRepository) ListOldestByDevice(ctx context.Context, deviceID uuid.UUID, limit int) ([]*LogFile, error) {
	q := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{"device_id": deviceID, "is_deleted": false})
	if r.withFaultFields {
		q = q.Where(sq.Eq{"record_status": FaultRecordStatusFileReceived})
	}
	query, args, err := q.
		OrderBy("collected_at ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list oldest by device %s: %w", r.tableName, err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query oldest by device %s: %w", r.tableName, err)
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

// ListExpired 按入库时间（created_at）升序返回早于 cutoff 的未删除记录，用于按时间保留清理
// （#320）。不限定 record_status：60 天前仍是 detected 占位（始终没等到文件）的故障记录同样
// 视为过期，由调用方对无 object_path 的记录跳过 MinIO 删除。
func (r *PgRepository) ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]*LogFile, error) {
	query, args, err := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{"is_deleted": false}).
		Where(sq.Lt{"created_at": cutoff}).
		OrderBy("created_at ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expired %s: %w", r.tableName, err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query expired %s: %w", r.tableName, err)
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

// UpdateFile 把 detected 状态的故障日志记录推进到 file_received。
// 仅在 station_fault_logs 上有意义；其他表调用会返回错误。
func (r *PgRepository) UpdateFile(ctx context.Context, id uuid.UUID, fileName, objectPath, bucket string, fileSize int64) error {
	if !r.withFaultFields {
		return fmt.Errorf("UpdateFile only supported on fault log table, not %s", r.tableName)
	}
	query, args, err := storage.Psql.Update(r.tableName).
		Set("file_name", fileName).
		Set("object_path", objectPath).
		Set("bucket", bucket).
		Set("file_size", fileSize).
		Set("record_status", FaultRecordStatusFileReceived).
		Set("collected_at", time.Now()).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update file %s: %w", r.tableName, err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

// LatestDetectedByDeviceSN 取设备最近一条 detected 状态的故障日志，用于
// 文件到达时把占位记录推进到 file_received。
func (r *PgRepository) LatestDetectedByDeviceSN(ctx context.Context, sn string) (*LogFile, error) {
	if !r.withFaultFields {
		return nil, fmt.Errorf("LatestDetectedByDeviceSN only supported on fault log table, not %s", r.tableName)
	}
	query, args, err := storage.Psql.Select(r.cols()...).
		From(r.tableName).
		Where(sq.Eq{
			"device_sn":     sn,
			"is_deleted":    false,
			"record_status": FaultRecordStatusDetected,
		}).
		OrderBy("collected_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest detected %s: %w", r.tableName, err)
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
		// 故障日志：nullable 列用 *string 接住，避免 pgx 在 NULL 上 panic
		var (
			fileName, objectPath, bucket                       *string
			faultReason, faultDetail                           *string
			deviceName, deviceType, operateIP, softwareVersion *string
			runtimeBeforeReboot                                *int64
			collectionFailReason                               *string
		)
		err = row.Scan(
			&f.ID, &f.DeviceID, &f.DeviceSN, &fileName,
			&objectPath, &bucket, &f.FileSize,
			&faultReason, &faultDetail,
			&f.TaskID, &f.IsDeleted, &f.CollectedAt, &f.CreatedAt, &f.UpdatedAt,
			&deviceName, &deviceType, &f.IsGNB, &operateIP, &softwareVersion,
			&runtimeBeforeReboot,
			&f.RecordStatus, &collectionFailReason, &f.ManualCollectionStatus,
		)
		if err != nil {
			return nil, err
		}
		f.FileName = derefStr(fileName)
		f.ObjectPath = derefStr(objectPath)
		f.Bucket = derefStr(bucket)
		f.FaultReason = derefStr(faultReason)
		f.FaultDetail = derefStr(faultDetail)
		f.DeviceName = derefStr(deviceName)
		f.DeviceType = derefStr(deviceType)
		f.OperateIP = derefStr(operateIP)
		f.SoftwareVersion = derefStr(softwareVersion)
		f.CollectionFailReason = derefStr(collectionFailReason)
		if runtimeBeforeReboot != nil {
			f.RuntimeBeforeReboot = *runtimeBeforeReboot
		}
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

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
