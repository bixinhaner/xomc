package license

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// LicenseLogRepository 提供 license_logs 表的持久化能力。
//
// 关注点正交于 LicenseRepository（后者管 licenses 表）；本 repo 只读写 license_logs。
type LicenseLogRepository interface {
	// Create 写入一条审计日志。
	// 写日志失败不应阻断主业务流程；调用方（LogWriter）已经把 err 降级为 warn 日志。
	Create(ctx context.Context, log *LicenseLog) error

	// List 按过滤条件分页返回审计日志。created_at DESC 默认排序。
	List(ctx context.Context, filter LicenseLogFilter) (*model.ListResponse[LicenseLog], error)

	// ListByLicense 取单 license 最近 N 条日志（详情抽屉 "最近操作" 用）。
	// limit ≤ 0 时使用默认值 10。
	ListByLicense(ctx context.Context, licenseID uuid.UUID, limit int) ([]LicenseLog, error)
}

// licenseLogColumns 全列清单（与 migration 000073 字段一致）。
var licenseLogColumns = []string{
	"id", "license_id", "log_type", "actor_user_id",
	"result", "details", "client_ip", "user_agent", "created_at",
}

// PgLicenseLogRepository 是 LicenseLogRepository 的 PostgreSQL 实现。
type PgLicenseLogRepository struct {
	pool *pgxpool.Pool
}

var _ LicenseLogRepository = (*PgLicenseLogRepository)(nil)

// NewPgLicenseLogRepository 构造一个新的 PgLicenseLogRepository。
func NewPgLicenseLogRepository(pool *pgxpool.Pool) *PgLicenseLogRepository {
	return &PgLicenseLogRepository{pool: pool}
}

// Create 实现 LicenseLogRepository.Create。
func (r *PgLicenseLogRepository) Create(ctx context.Context, log *LicenseLog) error {
	if log == nil {
		return fmt.Errorf("nil log: %w", commonerrors.ErrInvalidInput)
	}
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if len(log.Details) == 0 {
		log.Details = json.RawMessage(`{}`)
	}

	query, args, err := storage.Psql.Insert("license_logs").
		Columns("id", "license_id", "log_type", "actor_user_id",
			"result", "details", "client_ip", "user_agent").
		Values(log.ID, log.LicenseID, string(log.LogType), log.ActorUserID,
			string(log.Result), log.Details, log.ClientIP, log.UserAgent).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert license_log SQL: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert license_log: %w", err)
	}
	return nil
}

// List 实现 LicenseLogRepository.List，按过滤条件分页 + created_at DESC 排序。
func (r *PgLicenseLogRepository) List(ctx context.Context, filter LicenseLogFilter) (*model.ListResponse[LicenseLog], error) {
	page, pageSize := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 公共 WHERE 部分
	where := sq.And{}
	if filter.LicenseID != nil {
		where = append(where, sq.Eq{"license_id": *filter.LicenseID})
	}
	if len(filter.LogTypes) > 0 {
		typeStrs := make([]string, len(filter.LogTypes))
		for i, t := range filter.LogTypes {
			typeStrs[i] = string(t)
		}
		where = append(where, sq.Eq{"log_type": typeStrs})
	}
	if len(filter.Results) > 0 {
		resultStrs := make([]string, len(filter.Results))
		for i, r := range filter.Results {
			resultStrs[i] = string(r)
		}
		where = append(where, sq.Eq{"result": resultStrs})
	}
	if filter.ActorUserID != nil {
		where = append(where, sq.Eq{"actor_user_id": *filter.ActorUserID})
	}
	if filter.StartTime != nil {
		where = append(where, sq.GtOrEq{"created_at": *filter.StartTime})
	}
	if filter.EndTime != nil {
		where = append(where, sq.LtOrEq{"created_at": *filter.EndTime})
	}
	if filter.Search != nil && *filter.Search != "" {
		// 注意：用 details::text ILIKE，避免 jsonb 全表 → 现规模下可接受；
		// 数据量大时需上 GIN 索引（pg_trgm）或专用全文检索字段。
		where = append(where, sq.ILike{"details::text": "%" + *filter.Search + "%"})
	}

	// total
	countSQL, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("license_logs").
		Where(where).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count license_logs SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count license_logs: %w", err)
	}

	// rows
	listSQL, listArgs, err := storage.Psql.Select(licenseLogColumns...).
		From("license_logs").
		Where(where).
		OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list license_logs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("query license_logs: %w", err)
	}
	defer rows.Close()

	items := make([]LicenseLog, 0, pageSize)
	for rows.Next() {
		log, err := scanLicenseLog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate license_logs rows: %w", err)
	}

	return model.NewListResponse(items, total, page, pageSize), nil
}

// ListByLicense 实现 LicenseLogRepository.ListByLicense，取单 license 最近 N 条。
func (r *PgLicenseLogRepository) ListByLicense(ctx context.Context, licenseID uuid.UUID, limit int) ([]LicenseLog, error) {
	if limit <= 0 {
		limit = 10
	}

	query, args, err := storage.Psql.Select(licenseLogColumns...).
		From("license_logs").
		Where(sq.Eq{"license_id": licenseID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list-by-license license_logs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query license_logs by license: %w", err)
	}
	defer rows.Close()

	items := make([]LicenseLog, 0, limit)
	for rows.Next() {
		log, err := scanLicenseLog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate list-by-license rows: %w", err)
	}
	return items, nil
}

// scanLicenseLog 把 pgx.Row 扫成 *LicenseLog。
//
// client_ip 列在 PG 是 inet 类型；pgx v5 默认扫描成 netip.Addr 或 nil。
// 这里把它收敛成 *string（IPv4/v6 字符串），让 JSON 输出与前端期望一致。
func scanLicenseLog(rows pgx.Row) (*LicenseLog, error) {
	var (
		log         LicenseLog
		licenseID   uuid.NullUUID
		actorUserID uuid.NullUUID
		clientIP    *string
		userAgent   *string
		details     []byte
	)
	if err := rows.Scan(
		&log.ID,
		&licenseID,
		&log.LogType,
		&actorUserID,
		&log.Result,
		&details,
		&clientIP,
		&userAgent,
		&log.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w", commonerrors.ErrNotFound)
		}
		return nil, fmt.Errorf("scan license_log: %w", err)
	}
	if licenseID.Valid {
		id := licenseID.UUID
		log.LicenseID = &id
	}
	if actorUserID.Valid {
		id := actorUserID.UUID
		log.ActorUserID = &id
	}
	log.ClientIP = clientIP
	log.UserAgent = userAgent
	if len(details) == 0 {
		log.Details = json.RawMessage(`{}`)
	} else {
		log.Details = json.RawMessage(details)
	}
	return &log, nil
}
