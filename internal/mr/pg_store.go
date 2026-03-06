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
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/mr/parser"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

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

func (s *PgMRStore) ListFiles(ctx context.Context, filter MRFileFilter) (*model.ListResponse[MRFileInfo], error) {
	qb := psql.Select("id", "device_id", "device_sn", "carrier", "mr_type", "file_name",
		"file_size", "collect_time", "minio_path", "parsed", "parsed_at", "record_count", "created_at").
		From("mr_files")
	countQb := psql.Select("COUNT(*)").From("mr_files")

	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
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
	qb := psql.Select("time", "file_id", "device_id", "cell_id", "mr_type", "measurement_data").
		From("mr_records")
	countQb := psql.Select("COUNT(*)").From("mr_records")

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
