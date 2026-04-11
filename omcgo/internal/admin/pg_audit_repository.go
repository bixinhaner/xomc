package admin

import (
	"context"
	"fmt"
	"math"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgAuditRepository implements AuditRepository using PostgreSQL.
type PgAuditRepository struct {
	pool *pgxpool.Pool
}

var _ AuditRepository = (*PgAuditRepository)(nil)

// NewPgAuditRepository creates a new PgAuditRepository.
func NewPgAuditRepository(pool *pgxpool.Pool) *PgAuditRepository {
	return &PgAuditRepository{pool: pool}
}

func (r *PgAuditRepository) Create(ctx context.Context, log *AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	query, args, err := storage.Psql.Insert("audit_logs").
		Columns("id", "user_id", "username", "action", "resource", "resource_id", "details", "ip_address", "user_agent").
		Values(
			log.ID, log.UserID, log.Username, log.Action,
			nullableString(log.Resource), nullableString(log.ResourceID),
			log.Details, nullableString(log.IPAddress), nullableString(log.UserAgent),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert audit log SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *PgAuditRepository) List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error) {
	base := storage.Psql.Select("id", "user_id", "username", "action", "resource", "resource_id", "details", "ip_address::text", "user_agent", "created_at").
		From("audit_logs")
	countBase := storage.Psql.Select("COUNT(*)").From("audit_logs")

	if filter.UserID != nil {
		base = base.Where(sq.Eq{"user_id": *filter.UserID})
		countBase = countBase.Where(sq.Eq{"user_id": *filter.UserID})
	}
	if filter.Action != nil && *filter.Action != "" {
		base = base.Where(sq.Eq{"action": *filter.Action})
		countBase = countBase.Where(sq.Eq{"action": *filter.Action})
	}
	if filter.Resource != nil && *filter.Resource != "" {
		base = base.Where(sq.Eq{"resource": *filter.Resource})
		countBase = countBase.Where(sq.Eq{"resource": *filter.Resource})
	}
	if filter.StartTime != nil && *filter.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, *filter.StartTime); err == nil {
			base = base.Where(sq.GtOrEq{"created_at": t})
			countBase = countBase.Where(sq.GtOrEq{"created_at": t})
		}
	}
	if filter.EndTime != nil && *filter.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, *filter.EndTime); err == nil {
			base = base.Where(sq.LtOrEq{"created_at": t})
			countBase = countBase.Where(sq.LtOrEq{"created_at": t})
		}
	}

	// Count
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count audit logs SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count audit logs: %w", err)
	}

	// Pagination
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query, args, err := base.
		OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list audit logs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var ipAddr, userAgent, resource, resourceID *string
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Action,
			&resource, &resourceID, &log.Details,
			&ipAddr, &userAgent, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if ipAddr != nil {
			log.IPAddress = *ipAddr
		}
		if userAgent != nil {
			log.UserAgent = *userAgent
		}
		if resource != nil {
			log.Resource = *resource
		}
		if resourceID != nil {
			log.ResourceID = *resourceID
		}
		logs = append(logs, log)
	}

	return &model.ListResponse[AuditLog]{
		Items:      logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}
