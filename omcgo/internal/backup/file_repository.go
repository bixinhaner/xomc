// Package backup — backup_restore_file 元数据表 (M1 of backup-restore-alignment-plan).
//
// 规范要求每次基站配置文件成功落地后记录元数据 (SN/file_name/md5/size/operator_code/update_time)
// 以支撑后续运维查询 (queryCellInfos / exportFile)。该表由 FilePathRecorder 在
// SubjectBackupFileReceived 触发时 upsert，自然键 (serial_number, file_name)。
package backup

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// BackupRestoreFile 是 backup_restore_file 表的行模型。
type BackupRestoreFile struct {
	ID           int64      `json:"id"`
	SerialNumber string     `json:"serial_number"`
	FileName     string     `json:"file_name"`
	ObjectPath   string     `json:"object_path"`
	MD5          *string    `json:"md5,omitempty"`
	FileSize     int64      `json:"file_size"`
	OperatorCode *string    `json:"operator_code,omitempty"`
	IsDeleted    bool       `json:"is_deleted"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	// TaskID 是 UFTE 主任务的 UUID（upgrade_tasks.id），用于隔离同 SN 下不同
	// 任务上传的同名文件（厂商如 baicells 每次都用 "mib-home-fap.nv" 名）。
	// 旧数据或非任务路径上传的文件可能为空。
	TaskID     *string   `json:"task_id,omitempty"`
	UpdateTime time.Time `json:"update_time"`
	CreatedAt  time.Time `json:"created_at"`
}

// FileRepository 是 backup_restore_file 表的最小契约。
type FileRepository interface {
	// Upsert inserts a new metadata row or updates an existing row on the
	// natural key (serial_number, file_name). update_time / object_path /
	// md5 / file_size / operator_code 都会被覆盖。
	Upsert(ctx context.Context, f *BackupRestoreFile) error

	// ListBySerial 返回某 SN 下的全部备份文件元数据，按 update_time DESC 排序。
	// 用于 queryCellInfos / single/exportFile（M4 里程碑使用）。
	ListBySerial(ctx context.Context, sn string) ([]BackupRestoreFile, error)

	// ListByTaskID 返回 UFTE 主任务 UUID 关联的所有备份/日志文件元数据。
	// 用于任务删除时回收 MinIO 对象。空 taskID 返回空切片。
	ListByTaskID(ctx context.Context, taskID string) ([]BackupRestoreFile, error)

	// DeleteByTaskID 删除属于该 taskID 的所有元数据行。MinIO 对象清理由调用方
	// 在删除元数据**之前**完成（先 List → 删 MinIO → 删元数据）。
	DeleteByTaskID(ctx context.Context, taskID string) error
}

// LogFileQuotaRepository 是 backup_restore_file 上故障日志文件数限额需要的最小契约。
// 它只统计 logs/fault 目录下的 UFTE 故障日志采集文件，不影响运行日志和配置备份。
type LogFileQuotaRepository interface {
	CountActiveLogFiles(ctx context.Context) (int64, error)
	CountActiveLogFilesBySerial(ctx context.Context, sn string) (int64, error)
	ListActiveLogFileSerialCounts(ctx context.Context) ([]LogFileSerialCount, error)
	ListOldestActiveLogFiles(ctx context.Context, limit int) ([]BackupRestoreFile, error)
	ListOldestActiveLogFilesBySerial(ctx context.Context, sn string, limit int) ([]BackupRestoreFile, error)
	MarkFileDeleted(ctx context.Context, id int64) error
}

// LogFileSerialCount 是按设备 SN 聚合的活跃故障日志文件数量。
type LogFileSerialCount struct {
	SerialNumber string
	Count        int64
}

// LogFileRetentionRepository 是 backup_restore_file 上基站日志时间保留需要的最小契约。
// 它统计 logs/running + logs/fault 两类 UFTE 日志采集文件。
type LogFileRetentionRepository interface {
	ListExpiredStationLogFiles(ctx context.Context, cutoff time.Time, limit int) ([]BackupRestoreFile, error)
	MarkFileDeleted(ctx context.Context, id int64) error
}

// PgFileRepository 是 PostgreSQL 实现。
type PgFileRepository struct {
	pool *pgxpool.Pool
}

// NewPgFileRepository 构造一个 PgFileRepository。
func NewPgFileRepository(pool *pgxpool.Pool) *PgFileRepository {
	return &PgFileRepository{pool: pool}
}

var _ FileRepository = (*PgFileRepository)(nil)
var _ LogFileQuotaRepository = (*PgFileRepository)(nil)
var _ LogFileRetentionRepository = (*PgFileRepository)(nil)

const logFileQuotaWhere = "(object_path LIKE '%/fault/%' OR object_path LIKE 'fault/%')"
const logFileRetentionWhere = "(" +
	"object_path LIKE '%/running/%' OR object_path LIKE 'running/%' OR " +
	"object_path LIKE '%/fault/%' OR object_path LIKE 'fault/%')"

// Upsert: ON CONFLICT (serial_number, file_name) DO UPDATE
// —— 自然键冲突时覆盖元数据并刷新 update_time。
func (r *PgFileRepository) Upsert(ctx context.Context, f *BackupRestoreFile) error {
	if f == nil {
		return fmt.Errorf("nil BackupRestoreFile")
	}
	if f.SerialNumber == "" || f.FileName == "" {
		return fmt.Errorf("serial_number and file_name are required")
	}
	now := time.Now()
	f.UpdateTime = now

	// 新唯一键 (serial_number, task_id, file_name) — 见 migrations/000133。
	// 同 sn+task_id+file_name 冲突时更新元数据。不同 task_id 视作不同备份记录。
	query, args, err := storage.Psql.Insert("backup_restore_file").
		Columns("serial_number", "file_name", "object_path",
			"md5", "file_size", "operator_code", "task_id", "update_time", "is_deleted", "deleted_at").
		Values(f.SerialNumber, f.FileName, f.ObjectPath,
			f.MD5, f.FileSize, f.OperatorCode, f.TaskID, f.UpdateTime, false, nil).
		Suffix(`ON CONFLICT (serial_number, task_id, file_name) DO UPDATE SET
			object_path   = EXCLUDED.object_path,
			md5           = EXCLUDED.md5,
			file_size     = EXCLUDED.file_size,
			operator_code = COALESCE(EXCLUDED.operator_code, backup_restore_file.operator_code),
			update_time   = EXCLUDED.update_time,
			is_deleted    = false,
			deleted_at    = NULL
			RETURNING id, is_deleted, deleted_at, created_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build backup_restore_file upsert SQL: %w", err)
	}

	if scanErr := r.pool.QueryRow(ctx, query, args...).Scan(&f.ID, &f.IsDeleted, &f.DeletedAt, &f.CreatedAt); scanErr != nil {
		return fmt.Errorf("upsert backup_restore_file: %w", scanErr)
	}
	return nil
}

// ListByTaskID 按 task_id 列出该任务关联的所有备份/日志文件元数据。
func (r *PgFileRepository) ListByTaskID(ctx context.Context, taskID string) ([]BackupRestoreFile, error) {
	if taskID == "" {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select("id", "serial_number", "file_name", "object_path",
			"md5", "file_size", "operator_code", "is_deleted", "deleted_at", "task_id", "update_time", "created_at").
		From("backup_restore_file").
		Where(sq.Eq{"task_id": taskID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build backup_restore_file list-by-task SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup_restore_file by task_id: %w", err)
	}
	defer rows.Close()
	out := make([]BackupRestoreFile, 0)
	for rows.Next() {
		var f BackupRestoreFile
		if scanErr := rows.Scan(
			&f.ID, &f.SerialNumber, &f.FileName, &f.ObjectPath,
			&f.MD5, &f.FileSize, &f.OperatorCode, &f.IsDeleted, &f.DeletedAt, &f.TaskID, &f.UpdateTime, &f.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan backup_restore_file: %w", scanErr)
		}
		out = append(out, f)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate backup_restore_file: %w", rowsErr)
	}
	return out, nil
}

// DeleteByTaskID 删除 task_id 关联的所有元数据行。幂等：taskID 为空或无匹配行时返回 nil。
func (r *PgFileRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	if taskID == "" {
		return nil
	}
	query, args, err := storage.Psql.
		Delete("backup_restore_file").
		Where(sq.Eq{"task_id": taskID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build backup_restore_file delete-by-task SQL: %w", err)
	}
	if _, execErr := r.pool.Exec(ctx, query, args...); execErr != nil {
		return fmt.Errorf("delete backup_restore_file by task_id: %w", execErr)
	}
	return nil
}

// ListBySerial 按 SN 列出全部备份文件元数据。
func (r *PgFileRepository) ListBySerial(ctx context.Context, sn string) ([]BackupRestoreFile, error) {
	if sn == "" {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select("id", "serial_number", "file_name", "object_path",
			"md5", "file_size", "operator_code", "is_deleted", "deleted_at", "task_id", "update_time", "created_at").
		From("backup_restore_file").
		Where(sq.Eq{"serial_number": sn}).
		OrderBy("update_time DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build backup_restore_file list SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup_restore_file: %w", err)
	}
	defer rows.Close()

	out := make([]BackupRestoreFile, 0)
	for rows.Next() {
		var f BackupRestoreFile
		if scanErr := rows.Scan(
			&f.ID, &f.SerialNumber, &f.FileName, &f.ObjectPath,
			&f.MD5, &f.FileSize, &f.OperatorCode, &f.IsDeleted, &f.DeletedAt, &f.TaskID, &f.UpdateTime, &f.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan backup_restore_file: %w", scanErr)
		}
		out = append(out, f)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate backup_restore_file: %w", rowsErr)
	}
	return out, nil
}

func (r *PgFileRepository) CountActiveLogFiles(ctx context.Context) (int64, error) {
	return r.countActiveLogFiles(ctx, nil)
}

func (r *PgFileRepository) CountActiveLogFilesBySerial(ctx context.Context, sn string) (int64, error) {
	if sn == "" {
		return 0, nil
	}
	return r.countActiveLogFiles(ctx, sq.Eq{"serial_number": sn})
}

func (r *PgFileRepository) ListActiveLogFileSerialCounts(ctx context.Context) ([]LogFileSerialCount, error) {
	query, args, err := storage.Psql.
		Select("serial_number", "COUNT(*)").
		From("backup_restore_file").
		Where("is_deleted = false").
		Where(logFileQuotaWhere).
		Where(sq.NotEq{"serial_number": ""}).
		GroupBy("serial_number").
		OrderBy("serial_number ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build backup_restore_file log serial counts SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup_restore_file log serial counts: %w", err)
	}
	defer rows.Close()

	var out []LogFileSerialCount
	for rows.Next() {
		var item LogFileSerialCount
		if scanErr := rows.Scan(&item.SerialNumber, &item.Count); scanErr != nil {
			return nil, fmt.Errorf("scan backup_restore_file log serial count: %w", scanErr)
		}
		out = append(out, item)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate backup_restore_file log serial counts: %w", rowsErr)
	}
	return out, nil
}

func (r *PgFileRepository) countActiveLogFiles(ctx context.Context, extra sq.Sqlizer) (int64, error) {
	builder := storage.Psql.
		Select("COUNT(*)").
		From("backup_restore_file").
		Where("is_deleted = false").
		Where(logFileQuotaWhere)
	if extra != nil {
		builder = builder.Where(extra)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build backup_restore_file active log count SQL: %w", err)
	}
	var count int64
	if scanErr := r.pool.QueryRow(ctx, query, args...).Scan(&count); scanErr != nil {
		return 0, fmt.Errorf("count backup_restore_file active logs: %w", scanErr)
	}
	return count, nil
}

func (r *PgFileRepository) ListOldestActiveLogFiles(ctx context.Context, limit int) ([]BackupRestoreFile, error) {
	return r.listOldestActiveLogFiles(ctx, nil, limit)
}

func (r *PgFileRepository) ListOldestActiveLogFilesBySerial(ctx context.Context, sn string, limit int) ([]BackupRestoreFile, error) {
	if sn == "" {
		return nil, nil
	}
	return r.listOldestActiveLogFiles(ctx, sq.Eq{"serial_number": sn}, limit)
}

func (r *PgFileRepository) listOldestActiveLogFiles(ctx context.Context, extra sq.Sqlizer, limit int) ([]BackupRestoreFile, error) {
	if limit <= 0 {
		return nil, nil
	}
	builder := storage.Psql.
		Select("id", "serial_number", "file_name", "object_path",
			"md5", "file_size", "operator_code", "is_deleted", "deleted_at", "task_id", "update_time", "created_at").
		From("backup_restore_file").
		Where("is_deleted = false").
		Where(logFileQuotaWhere).
		OrderBy("update_time ASC", "id ASC").
		Limit(uint64(limit))
	if extra != nil {
		builder = builder.Where(extra)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build backup_restore_file oldest log SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup_restore_file oldest logs: %w", err)
	}
	defer rows.Close()

	out := make([]BackupRestoreFile, 0, limit)
	for rows.Next() {
		var f BackupRestoreFile
		if scanErr := rows.Scan(
			&f.ID, &f.SerialNumber, &f.FileName, &f.ObjectPath,
			&f.MD5, &f.FileSize, &f.OperatorCode, &f.IsDeleted, &f.DeletedAt, &f.TaskID, &f.UpdateTime, &f.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan backup_restore_file oldest log: %w", scanErr)
		}
		out = append(out, f)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate backup_restore_file oldest logs: %w", rowsErr)
	}
	return out, nil
}

func (r *PgFileRepository) ListExpiredStationLogFiles(ctx context.Context, cutoff time.Time, limit int) ([]BackupRestoreFile, error) {
	if limit <= 0 {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select("id", "serial_number", "file_name", "object_path",
			"md5", "file_size", "operator_code", "is_deleted", "deleted_at", "task_id", "update_time", "created_at").
		From("backup_restore_file").
		Where("is_deleted = false").
		Where(logFileRetentionWhere).
		Where(sq.Lt{"update_time": cutoff}).
		OrderBy("update_time ASC", "id ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build backup_restore_file expired station log SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup_restore_file expired station logs: %w", err)
	}
	defer rows.Close()

	out := make([]BackupRestoreFile, 0, limit)
	for rows.Next() {
		var f BackupRestoreFile
		if scanErr := rows.Scan(
			&f.ID, &f.SerialNumber, &f.FileName, &f.ObjectPath,
			&f.MD5, &f.FileSize, &f.OperatorCode, &f.IsDeleted, &f.DeletedAt, &f.TaskID, &f.UpdateTime, &f.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan backup_restore_file expired station log: %w", scanErr)
		}
		out = append(out, f)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate backup_restore_file expired station logs: %w", rowsErr)
	}
	return out, nil
}

func (r *PgFileRepository) MarkFileDeleted(ctx context.Context, id int64) error {
	query, args, err := storage.Psql.
		Update("backup_restore_file").
		Set("is_deleted", true).
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build backup_restore_file mark-deleted SQL: %w", err)
	}
	if _, execErr := r.pool.Exec(ctx, query, args...); execErr != nil {
		return fmt.Errorf("mark backup_restore_file deleted: %w", execErr)
	}
	return nil
}
