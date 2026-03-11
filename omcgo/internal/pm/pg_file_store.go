package pm

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/model"
)

// PgPMFileStore implements PMFileStore using PostgreSQL.
type PgPMFileStore struct {
	pool *pgxpool.Pool
}

// NewPgPMFileStore creates a new PostgreSQL-backed PM file store.
func NewPgPMFileStore(pool *pgxpool.Pool) *PgPMFileStore {
	return &PgPMFileStore{pool: pool}
}

func (s *PgPMFileStore) SaveFile(ctx context.Context, info *PMFileInfo) error {
	if info.ID == uuid.Nil {
		info.ID = uuid.New()
	}
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO pm_files (id, device_id, device_sn, carrier, technology, file_name, file_size,
		                       collect_time, minio_path, parsed, counter_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		info.ID, info.DeviceID, info.DeviceSN, info.Carrier, info.Technology,
		info.FileName, info.FileSize, info.CollectTime, info.MinioPath,
		info.Parsed, info.CounterCount, info.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pm_files: %w", err)
	}
	return nil
}

func (s *PgPMFileStore) GetFileByID(ctx context.Context, id uuid.UUID) (*PMFileInfo, error) {
	var f PMFileInfo
	err := s.pool.QueryRow(ctx,
		`SELECT id, device_id, device_sn, carrier, technology, file_name, file_size,
		        collect_time, minio_path, parsed, parsed_at, counter_count, created_at
		 FROM pm_files WHERE id = $1`, id,
	).Scan(&f.ID, &f.DeviceID, &f.DeviceSN, &f.Carrier, &f.Technology,
		&f.FileName, &f.FileSize, &f.CollectTime, &f.MinioPath,
		&f.Parsed, &f.ParsedAt, &f.CounterCount, &f.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get pm_file by id: %w", err)
	}
	return &f, nil
}

func (s *PgPMFileStore) ListFiles(ctx context.Context, filter PMFileFilter) (*model.ListResponse[PMFileInfo], error) {
	qb := psql.Select("id", "device_id", "device_sn", "carrier", "technology", "file_name",
		"file_size", "collect_time", "minio_path", "parsed", "parsed_at", "counter_count", "created_at").
		From("pm_files")
	countQb := psql.Select("COUNT(*)").From("pm_files")

	if filter.DeviceID != nil {
		qb = qb.Where(sq.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(sq.Eq{"device_id": *filter.DeviceID})
	}
	if filter.StartTime != nil {
		qb = qb.Where(sq.GtOrEq{"collect_time": *filter.StartTime})
		countQb = countQb.Where(sq.GtOrEq{"collect_time": *filter.StartTime})
	}
	if filter.EndTime != nil {
		qb = qb.Where(sq.LtOrEq{"collect_time": *filter.EndTime})
		countQb = countQb.Where(sq.LtOrEq{"collect_time": *filter.EndTime})
	}

	countSQL, countArgs, _ := countQb.ToSql()
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count pm_files: %w", err)
	}

	qb = qb.OrderBy("collect_time DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))
	sql, args, _ := qb.ToSql()
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query pm_files: %w", err)
	}
	defer rows.Close()

	var items []PMFileInfo
	for rows.Next() {
		var f PMFileInfo
		if err := rows.Scan(&f.ID, &f.DeviceID, &f.DeviceSN, &f.Carrier, &f.Technology,
			&f.FileName, &f.FileSize, &f.CollectTime, &f.MinioPath,
			&f.Parsed, &f.ParsedAt, &f.CounterCount, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pm_files: %w", err)
		}
		items = append(items, f)
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (s *PgPMFileStore) UpdateFileParsed(ctx context.Context, id uuid.UUID, counterCount int) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE pm_files SET parsed = true, parsed_at = $1, counter_count = $2 WHERE id = $3`,
		now, counterCount, id,
	)
	if err != nil {
		return fmt.Errorf("update pm_files parsed: %w", err)
	}
	return nil
}

var _ PMFileStore = (*PgPMFileStore)(nil)
