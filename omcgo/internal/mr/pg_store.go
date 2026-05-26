package mr

import (
	"context"
	"encoding/json"
	"fmt"
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
	_, err := s.pool.Exec(ctx,
		`INSERT INTO mr_files (id, device_id, device_sn, carrier, mr_type, file_name, file_size, collect_time, minio_path, parsed, record_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		file.ID, file.DeviceID, file.DeviceSN, file.Carrier, file.MRType,
		file.FileName, file.FileSize, file.CollectTime, file.MinioPath,
		file.Parsed, file.RecordCount, file.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert mr_files: %w", err)
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

// ListFileDeviceAggregates 按 device_sn 聚合查询。每设备 1 行，含起止 collect_time + 文件数。
func (s *PgMRStore) ListFileDeviceAggregates(ctx context.Context, filter MRFileDeviceFilter) (*model.ListResponse[MRFileDeviceAggregate], error) {
	// 先 count distinct device_sn 给分页用
	countSQL := `SELECT COUNT(*) FROM (SELECT device_sn FROM mr_files`
	args := make([]interface{}, 0, 2)
	wherePieces := make([]string, 0, 1)
	if filter.Keyword != nil && *filter.Keyword != "" {
		wherePieces = append(wherePieces, `device_sn ILIKE $1`)
		args = append(args, "%"+*filter.Keyword+"%")
	}
	whereClause := ""
	if len(wherePieces) > 0 {
		whereClause = " WHERE " + wherePieces[0]
	}
	countSQL += whereClause + ` GROUP BY device_sn) AS sub`

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

	listSQL := `SELECT device_sn,
	                  MIN(collect_time) AS first_collect_time,
	                  MAX(collect_time) AS last_collect_time,
	                  COUNT(*)::bigint  AS file_count
	            FROM mr_files` + whereClause + `
	            GROUP BY device_sn
	            ORDER BY MAX(collect_time) DESC
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
		if err := rows.Scan(&a.DeviceSN, &a.FirstCollectTime, &a.LastCollectTime, &a.FileCount); err != nil {
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
