package mr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/mr/parser"
)

// PgMRStore implements MRStore using PostgreSQL and TimescaleDB.
type PgMRStore struct {
	pool   *pgxpool.Pool
	tsPool *pgxpool.Pool
}

// NewPgMRStore creates a new PostgreSQL-backed MR store.
func NewPgMRStore(pool *pgxpool.Pool, tsPool *pgxpool.Pool) *PgMRStore {
	return &PgMRStore{pool: pool, tsPool: tsPool}
}

func (s *PgMRStore) SaveFile(ctx context.Context, file *MRFileInfo) error {
	// ON CONFLICT DO NOTHING：NATS mr.file.received QueueSubscribe 失败时会重投
	// 最多 5 次，同一物理文件可能被 Collector 处理多次。db 端 uq_mr_files_sn_filename
	// 唯一索引（migration 000204）+ ON CONFLICT 让重复插入幂等无副作用。
	_, err := s.pool.Exec(ctx,
		`INSERT INTO mr_files (id, device_id, device_sn, carrier, mr_type, file_name, file_size, collect_time, minio_path, parsed, record_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (device_sn, file_name) DO NOTHING`,
		file.ID, file.DeviceID, file.DeviceSN, file.Carrier, file.MRType,
		file.FileName, file.FileSize, file.CollectTime, file.MinioPath,
		file.Parsed, file.RecordCount, file.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert mr_files: %w", err)
	}
	return nil
}

// ListUncompressed 返回 raw_compressed=false 且 created_at < olderThan 的 mr_files 对应
// MinIO 对象键（按 created_at 升序，至多 limit 条）。供 rawarchive.Sweeper 补偿扫描，命中
// idx_mr_files_uncompressed 部分索引。
func (s *PgMRStore) ListUncompressed(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT minio_path FROM mr_files
		 WHERE raw_compressed = false AND created_at < $1
		 ORDER BY created_at
		 LIMIT $2`, olderThan, limit)
	if err != nil {
		return nil, fmt.Errorf("list uncompressed mr_files: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan uncompressed mr_files: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MarkCompressed 把给定 MinIO 对象键对应的 mr_files 行标记为已压缩回写（raw_compressed=true）。
// 内联压缩成功后单键调用、Sweeper 补压后批量调用，二者共用。
func (s *PgMRStore) MarkCompressed(ctx context.Context, objects []string) error {
	if len(objects) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE mr_files SET raw_compressed = true WHERE minio_path = ANY($1)`, objects)
	if err != nil {
		return fmt.Errorf("mark mr_files raw_compressed: %w", err)
	}
	return nil
}

func (s *PgMRStore) UpdateFileParsed(ctx context.Context, fileID uuid.UUID, recordCount int) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE mr_files SET parsed = true, parsed_at = $1, record_count = $2 WHERE id = $3`,
		now, recordCount, fileID,
	)
	if err != nil {
		return fmt.Errorf("update mr_files parsed: %w", err)
	}
	return nil
}

// ListFileDeviceAggregates 按 device_sn 聚合查询。每设备 1 行，含起止 collect_time + 文件数
// + 站名 + 产品类（LEFT JOIN devices）。
//
// 性能：devices.serial_number 有索引，每页 20 个 SN 的 JOIN 命中索引 sub-ms。
func (s *PgMRStore) ListFileDeviceAggregates(ctx context.Context, filter MRFileDeviceFilter) (*model.ListResponse[MRFileDeviceAggregate], error) {
	args := make([]interface{}, 0, 4)
	wherePieces := make([]string, 0, 3)
	if filter.Keyword != nil && *filter.Keyword != "" {
		args = append(args, "%"+*filter.Keyword+"%")
		wherePieces = append(wherePieces, fmt.Sprintf("m.device_sn ILIKE $%d", len(args)))
	}
	if filter.SiteName != nil && *filter.SiteName != "" {
		args = append(args, "%"+*filter.SiteName+"%")
		wherePieces = append(wherePieces, fmt.Sprintf("d.site_name ILIKE $%d", len(args)))
	}
	if filter.ProductClass != nil && *filter.ProductClass != "" {
		args = append(args, "%"+*filter.ProductClass+"%")
		wherePieces = append(wherePieces, fmt.Sprintf("d.product_class ILIKE $%d", len(args)))
	}
	whereClause := ""
	if len(wherePieces) > 0 {
		whereClause = " WHERE " + strings.Join(wherePieces, " AND ")
	}

	// JOIN devices 提供 site_name / product_class，同时支持基于这两列的过滤。
	joinClause := " LEFT JOIN devices d ON d.serial_number = m.device_sn AND d.deleted_at IS NULL"

	countSQL := `SELECT COUNT(*) FROM (SELECT m.device_sn FROM mr_files m` + joinClause + whereClause + ` GROUP BY m.device_sn) AS sub`
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr_files devices: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	// reporting 列：EXISTS 查 mr_customize_task task_status='on' 且
	// device_sn 在 target_device_sns 数组里，存在即 true。
	// site_name / product_class 用 MAX() 包：devices.serial_number 唯一，
	// 1:1 关系下 MAX 等价 ANY_VALUE，不影响 GROUP BY 分组维度。
	listSQL := `SELECT m.device_sn,
	                  COALESCE(MAX(d.site_name), '')     AS site_name,
	                  COALESCE(MAX(d.product_class), '') AS product_class,
	                  MIN(m.collect_time) AS first_collect_time,
	                  MAX(m.collect_time) AS last_collect_time,
	                  COUNT(*)::bigint    AS file_count,
	                  EXISTS (
	                      SELECT 1 FROM mr_customize_task t
	                      WHERE t.task_status = 'on'
	                        AND m.device_sn = ANY(t.target_device_sns)
	                  ) AS reporting
	            FROM mr_files m` + joinClause + whereClause + `
	            GROUP BY m.device_sn
	            ORDER BY MAX(m.collect_time) DESC
	            LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list mr_files devices: %w", err)
	}
	defer rows.Close()
	items := make([]MRFileDeviceAggregate, 0)
	for rows.Next() {
		var a MRFileDeviceAggregate
		if err := rows.Scan(&a.DeviceSN, &a.SiteName, &a.ProductClass,
			&a.FirstCollectTime, &a.LastCollectTime, &a.FileCount, &a.Reporting); err != nil {
			return nil, fmt.Errorf("scan mr_files devices row: %w", err)
		}
		items = append(items, a)
	}
	return model.NewListResponse(items, total, page, pageSize), nil
}

// DeleteFilesBefore 删除 collect_time < cutoff 的 mr_files 行。返回删除行数。
// 由 internal/mr/task/cleaner.go 在 MinIO 目录清理后调用，保持 MinIO 与 PG 一致。
//
// 注意：mr_records 是 TimescaleDB hypertable，通常按 time 列分区。本方法仅删
// mr_files 元数据行；记录表的清理由 TimescaleDB retention policy 单独管（详见
// migrations/000006_alarms_mr_firmware.sql 中 mr_records 的 add_retention_policy）。
func (s *PgMRStore) DeleteFilesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM mr_files WHERE collect_time < $1`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("delete mr_files before %s: %w", cutoff.Format(time.RFC3339), err)
	}
	return tag.RowsAffected(), nil
}

// ListFilesBySN 取某 SN 名下全部 mr_files 元数据（无分页）。
func (s *PgMRStore) ListFilesBySN(ctx context.Context, sn string) ([]MRFileInfo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, device_id, device_sn, carrier, mr_type, file_name, file_size,
		        collect_time, minio_path, parsed, parsed_at, record_count, created_at
		 FROM mr_files WHERE device_sn = $1`, sn)
	if err != nil {
		return nil, fmt.Errorf("query mr_files by sn: %w", err)
	}
	defer rows.Close()
	var items []MRFileInfo
	for rows.Next() {
		var f MRFileInfo
		if err := rows.Scan(&f.ID, &f.DeviceID, &f.DeviceSN, &f.Carrier, &f.MRType,
			&f.FileName, &f.FileSize, &f.CollectTime, &f.MinioPath,
			&f.Parsed, &f.ParsedAt, &f.RecordCount, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mr_files: %w", err)
		}
		items = append(items, f)
	}
	return items, nil
}

// DeleteFilesBySN 删除该 SN 下所有 mr_files 元数据行。返回删除行数。
func (s *PgMRStore) DeleteFilesBySN(ctx context.Context, sn string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM mr_files WHERE device_sn = $1`, sn)
	if err != nil {
		return 0, fmt.Errorf("delete mr_files by sn %s: %w", sn, err)
	}
	return tag.RowsAffected(), nil
}

func (s *PgMRStore) BatchInsertRecords(ctx context.Context, fileID, deviceID uuid.UUID, mrType string, records []parser.MRRecord) error {
	if len(records) == 0 {
		return nil
	}

	rows := make([][]interface{}, 0, len(records))
	for _, rec := range records {
		dataJSON, _ := json.Marshal(rec.MeasurementData)
		rows = append(rows, []interface{}{
			rec.Time, fileID, deviceID, rec.CellID, mrType, dataJSON,
		})
	}

	_, err := s.tsPool.CopyFrom(ctx,
		pgx.Identifier{"mr_records"},
		[]string{"time", "file_id", "device_id", "cell_id", "mr_type", "measurement_data"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert mr_records: %w", err)
	}
	return nil
}

func (s *PgMRStore) GetFileByID(ctx context.Context, fileID uuid.UUID) (*MRFileInfo, error) {
	var f MRFileInfo
	err := s.pool.QueryRow(ctx,
		`SELECT id, device_id, device_sn, carrier, mr_type, file_name, file_size,
		        collect_time, minio_path, parsed, parsed_at, record_count, created_at
		 FROM mr_files WHERE id = $1`, fileID,
	).Scan(&f.ID, &f.DeviceID, &f.DeviceSN, &f.Carrier, &f.MRType,
		&f.FileName, &f.FileSize, &f.CollectTime, &f.MinioPath,
		&f.Parsed, &f.ParsedAt, &f.RecordCount, &f.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get mr_file by id: %w", err)
	}
	return &f, nil
}

func (s *PgMRStore) ListFiles(ctx context.Context, filter MRFileFilter) (*model.ListResponse[MRFileInfo], error) {
	qb := storage.Psql.Select("id", "device_id", "device_sn", "carrier", "mr_type", "file_name",
		"file_size", "collect_time", "minio_path", "parsed", "parsed_at", "record_count", "created_at").
		From("mr_files")
	countQb := storage.Psql.Select("COUNT(*)").From("mr_files")

	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
	}
	if filter.DeviceSN != nil {
		qb = qb.Where(squirrel.Eq{"device_sn": *filter.DeviceSN})
		countQb = countQb.Where(squirrel.Eq{"device_sn": *filter.DeviceSN})
	}
	if filter.MRType != nil {
		qb = qb.Where(squirrel.Eq{"mr_type": *filter.MRType})
		countQb = countQb.Where(squirrel.Eq{"mr_type": *filter.MRType})
	}
	if filter.StartTime != nil {
		qb = qb.Where(squirrel.GtOrEq{"collect_time": *filter.StartTime})
		countQb = countQb.Where(squirrel.GtOrEq{"collect_time": *filter.StartTime})
	}
	if filter.EndTime != nil {
		qb = qb.Where(squirrel.LtOrEq{"collect_time": *filter.EndTime})
		countQb = countQb.Where(squirrel.LtOrEq{"collect_time": *filter.EndTime})
	}

	countSQL, countArgs, _ := countQb.ToSql()
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr_files: %w", err)
	}

	qb = qb.OrderBy("collect_time DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))
	sql, args, _ := qb.ToSql()
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query mr_files: %w", err)
	}
	defer rows.Close()

	var items []MRFileInfo
	for rows.Next() {
		var f MRFileInfo
		if err := rows.Scan(&f.ID, &f.DeviceID, &f.DeviceSN, &f.Carrier, &f.MRType,
			&f.FileName, &f.FileSize, &f.CollectTime, &f.MinioPath,
			&f.Parsed, &f.ParsedAt, &f.RecordCount, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mr_files: %w", err)
		}
		items = append(items, f)
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (s *PgMRStore) QueryRecords(ctx context.Context, filter MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
	qb := storage.Psql.Select("time", "file_id", "device_id", "cell_id", "mr_type", "measurement_data").
		From("mr_records")
	countQb := storage.Psql.Select("COUNT(*)").From("mr_records")

	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
	}
	if filter.FileID != nil {
		qb = qb.Where(squirrel.Eq{"file_id": *filter.FileID})
		countQb = countQb.Where(squirrel.Eq{"file_id": *filter.FileID})
	}
	if filter.MRType != nil {
		qb = qb.Where(squirrel.Eq{"mr_type": *filter.MRType})
		countQb = countQb.Where(squirrel.Eq{"mr_type": *filter.MRType})
	}
	if filter.CellID != nil {
		qb = qb.Where(squirrel.Eq{"cell_id": *filter.CellID})
		countQb = countQb.Where(squirrel.Eq{"cell_id": *filter.CellID})
	}
	if filter.StartTime != nil {
		qb = qb.Where(squirrel.GtOrEq{"time": *filter.StartTime})
		countQb = countQb.Where(squirrel.GtOrEq{"time": *filter.StartTime})
	}
	if filter.EndTime != nil {
		qb = qb.Where(squirrel.LtOrEq{"time": *filter.EndTime})
		countQb = countQb.Where(squirrel.LtOrEq{"time": *filter.EndTime})
	}

	countSQL, countArgs, _ := countQb.ToSql()
	var total int64
	if err := s.tsPool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr_records: %w", err)
	}

	qb = qb.OrderBy("time DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))
	sql, args, _ := qb.ToSql()
	rows, err := s.tsPool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query mr_records: %w", err)
	}
	defer rows.Close()

	var items []MRRecordEntry
	for rows.Next() {
		var rec MRRecordEntry
		var dataJSON []byte
		if err := rows.Scan(&rec.Time, &rec.FileID, &rec.DeviceID, &rec.CellID, &rec.MRType, &dataJSON); err != nil {
			return nil, fmt.Errorf("scan mr_records: %w", err)
		}
		if len(dataJSON) > 0 {
			_ = json.Unmarshal(dataJSON, &rec.MeasurementData)
		}
		items = append(items, rec)
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

var _ MRStore = (*PgMRStore)(nil)
