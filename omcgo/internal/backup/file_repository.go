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
	ID           int64     `json:"id"`
	SerialNumber string    `json:"serial_number"`
	FileName     string    `json:"file_name"`
	ObjectPath   string    `json:"object_path"`
	MD5          *string   `json:"md5,omitempty"`
	FileSize     int64     `json:"file_size"`
	OperatorCode *string   `json:"operator_code,omitempty"`
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
			"md5", "file_size", "operator_code", "task_id", "update_time").
		Values(f.SerialNumber, f.FileName, f.ObjectPath,
			f.MD5, f.FileSize, f.OperatorCode, f.TaskID, f.UpdateTime).
		Suffix(`ON CONFLICT (serial_number, task_id, file_name) DO UPDATE SET
			object_path   = EXCLUDED.object_path,
			md5           = EXCLUDED.md5,
			file_size     = EXCLUDED.file_size,
			operator_code = COALESCE(EXCLUDED.operator_code, backup_restore_file.operator_code),
			update_time   = EXCLUDED.update_time
			RETURNING id, created_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build backup_restore_file upsert SQL: %w", err)
	}

	if scanErr := r.pool.QueryRow(ctx, query, args...).Scan(&f.ID, &f.CreatedAt); scanErr != nil {
		return fmt.Errorf("upsert backup_restore_file: %w", scanErr)
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
			"md5", "file_size", "operator_code", "task_id", "update_time", "created_at").
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
			&f.MD5, &f.FileSize, &f.OperatorCode, &f.TaskID, &f.UpdateTime, &f.CreatedAt,
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
