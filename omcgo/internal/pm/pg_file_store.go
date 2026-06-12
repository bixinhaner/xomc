package pm

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
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

	// ON CONFLICT (device_sn, file_name) DO NOTHING：CPE 重传同名文件静默忽略。
	// 重复上传无业务危害（minio 文件覆盖语义；parser 已 first-arrived 语义），
	// 不返回错误也不打 warn 日志（每个采样窗口的常规重传，避免刷屏）。
	_, err := s.pool.Exec(ctx,
		`INSERT INTO pm_files (id, device_id, device_sn, carrier, technology, file_name, file_size,
		                       collect_time, minio_path, parsed, counter_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (device_sn, file_name) DO NOTHING`,
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
	qb := storage.Psql.Select("id", "device_id", "device_sn", "carrier", "technology", "file_name",
		"file_size", "collect_time", "minio_path", "parsed", "parsed_at", "counter_count", "created_at").
		From("pm_files")
	countQb := storage.Psql.Select("COUNT(*)").From("pm_files")

	if filter.DeviceID != nil {
		qb = qb.Where(sq.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(sq.Eq{"device_id": *filter.DeviceID})
	}
	if filter.DeviceSN != nil && *filter.DeviceSN != "" {
		qb = qb.Where(sq.Eq{"device_sn": *filter.DeviceSN})
		countQb = countQb.Where(sq.Eq{"device_sn": *filter.DeviceSN})
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

// ListFileDeviceAggregates 按 device_sn 聚合 pm_files，每设备 1 行。
// 给 File Management → PM Tab 主列表用：起止 collect_time + 文件数 + 是否最近活跃
// + 站名 + 产品类（LEFT JOIN device_dim 影子表，pm_files 在时序库）。
//
// 性能：device_dim.serial_number 有索引，10 万级影子表 + 每页 20 个 SN 的 JOIN
// 命中索引 sub-ms（影子表由 worker 同步任务从主库 devices 刷入）。
func (s *PgPMFileStore) ListFileDeviceAggregates(ctx context.Context, filter PMFileDeviceFilter) (*model.ListResponse[PMFileDeviceAggregate], error) {
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

	// JOIN 在 count 与 list 两段同时存在，保证基于 devices 的过滤一致。
	// device_dim.deleted_at 过滤掉软删行（同 device repository 通用做法）。
	// pm_files 现在落在时序库（s.pool 注入 TsPool），devices 改读本库影子表 device_dim
	// 替代跨库 JOIN。
	joinClause := " LEFT JOIN device_dim d ON d.serial_number = m.device_sn AND d.deleted_at IS NULL"

	countSQL := `SELECT COUNT(*) FROM (SELECT m.device_sn FROM pm_files m` + joinClause + whereClause + ` GROUP BY m.device_sn) AS sub`
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count pm_files devices: %w", err)
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

	// site_name / product_class 用 MAX() 包：device_sn 与 devices 是 1:N 实际上 1:1
	// （devices.serial_number 唯一），MAX 取那唯一一行的值，等价于 ANY_VALUE；
	// 这样保留 GROUP BY device_sn 不增加分组维度。
	listSQL := `SELECT m.device_sn,
	                  COALESCE(MAX(d.site_name), '')     AS site_name,
	                  COALESCE(MAX(d.product_class), '') AS product_class,
	                  MIN(m.collect_time)                AS first_collect_time,
	                  MAX(m.collect_time)                AS last_collect_time,
	                  COUNT(*)::bigint                   AS file_count,
	                  (MAX(m.collect_time) > NOW() - INTERVAL '2 hours') AS reporting
	             FROM pm_files m` + joinClause + whereClause + `
	             GROUP BY m.device_sn
	             ORDER BY MAX(m.collect_time) DESC
	             LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list pm_files devices: %w", err)
	}
	defer rows.Close()
	items := make([]PMFileDeviceAggregate, 0)
	for rows.Next() {
		var a PMFileDeviceAggregate
		if err := rows.Scan(&a.DeviceSN, &a.SiteName, &a.ProductClass,
			&a.FirstCollectTime, &a.LastCollectTime, &a.FileCount, &a.Reporting); err != nil {
			return nil, fmt.Errorf("scan pm_files devices row: %w", err)
		}
		items = append(items, a)
	}
	return model.NewListResponse(items, total, page, pageSize), nil
}

// ListFilesBySN 取某个 SN 名下全部 pm_files 元数据（无分页）。
func (s *PgPMFileStore) ListFilesBySN(ctx context.Context, sn string) ([]PMFileInfo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, device_id, device_sn, carrier, technology, file_name, file_size,
		        collect_time, minio_path, parsed, parsed_at, counter_count, created_at
		 FROM pm_files WHERE device_sn = $1`, sn)
	if err != nil {
		return nil, fmt.Errorf("query pm_files by sn: %w", err)
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
	return items, nil
}

// DeleteFilesBySN 删除该 SN 下所有 pm_files 元数据行。返回删除行数。
func (s *PgPMFileStore) DeleteFilesBySN(ctx context.Context, sn string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pm_files WHERE device_sn = $1`, sn)
	if err != nil {
		return 0, fmt.Errorf("delete pm_files by sn %s: %w", sn, err)
	}
	return tag.RowsAffected(), nil
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
