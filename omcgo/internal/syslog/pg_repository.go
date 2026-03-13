package syslog

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var systemLogColumns = []string{
	"id", "level", "source", "message", "details", "created_at",
}

var neMessageLogColumns = []string{
	"id", "device_sn", "device_id", "message_type", "direction", "content", "created_at",
}

var _ SyslogRepository = (*PgSyslogRepository)(nil)

// PgSyslogRepository implements SyslogRepository using PostgreSQL.
type PgSyslogRepository struct {
	pool *pgxpool.Pool
}

// NewPgSyslogRepository creates a new PostgreSQL-backed syslog repository.
func NewPgSyslogRepository(pool *pgxpool.Pool) *PgSyslogRepository {
	return &PgSyslogRepository{pool: pool}
}

func (r *PgSyslogRepository) ListSystemLogs(ctx context.Context, filter SystemLogFilter) (*model.ListResponse[SystemLog], error) {
	builder := psql.Select(systemLogColumns...).From("system_logs")
	countBuilder := psql.Select("COUNT(*)").From("system_logs")

	builder = applySystemLogFilters(builder, filter)
	countBuilder = applySystemLogFilters(countBuilder, filter)

	// Count total
	countSQL, countArgs, _ := countBuilder.ToSql()
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count system_logs: %w", err)
	}

	// Apply sorting and pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	builder = builder.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list system_logs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list system_logs: %w", err)
	}
	defer rows.Close()

	var items []SystemLog
	for rows.Next() {
		log, err := scanSystemLogRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *log)
	}

	if items == nil {
		items = []SystemLog{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgSyslogRepository) ListNEMessageLogs(ctx context.Context, filter NEMessageLogFilter) (*model.ListResponse[NEMessageLog], error) {
	builder := psql.Select(neMessageLogColumns...).From("ne_message_logs")
	countBuilder := psql.Select("COUNT(*)").From("ne_message_logs")

	builder = applyNEMessageLogFilters(builder, filter)
	countBuilder = applyNEMessageLogFilters(countBuilder, filter)

	// Count total
	countSQL, countArgs, _ := countBuilder.ToSql()
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count ne_message_logs: %w", err)
	}

	// Apply sorting and pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	builder = builder.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list ne_message_logs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ne_message_logs: %w", err)
	}
	defer rows.Close()

	var items []NEMessageLog
	for rows.Next() {
		log, err := scanNEMessageLogRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *log)
	}

	if items == nil {
		items = []NEMessageLog{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func applySystemLogFilters(qb sq.SelectBuilder, f SystemLogFilter) sq.SelectBuilder {
	if f.Level != nil && *f.Level != "" {
		qb = qb.Where(sq.Eq{"level": *f.Level})
	}
	if f.Source != nil && *f.Source != "" {
		qb = qb.Where(sq.Eq{"source": *f.Source})
	}
	if f.StartTime != nil {
		qb = qb.Where(sq.GtOrEq{"created_at": *f.StartTime})
	}
	if f.EndTime != nil {
		qb = qb.Where(sq.LtOrEq{"created_at": *f.EndTime})
	}
	return qb
}

func applyNEMessageLogFilters(qb sq.SelectBuilder, f NEMessageLogFilter) sq.SelectBuilder {
	if f.DeviceSN != nil && *f.DeviceSN != "" {
		qb = qb.Where(sq.Eq{"device_sn": *f.DeviceSN})
	}
	if f.DeviceID != nil {
		qb = qb.Where(sq.Eq{"device_id": *f.DeviceID})
	}
	if f.MessageType != nil && *f.MessageType != "" {
		qb = qb.Where(sq.Eq{"message_type": *f.MessageType})
	}
	if f.Direction != nil && *f.Direction != "" {
		qb = qb.Where(sq.Eq{"direction": *f.Direction})
	}
	if f.StartTime != nil {
		qb = qb.Where(sq.GtOrEq{"created_at": *f.StartTime})
	}
	if f.EndTime != nil {
		qb = qb.Where(sq.LtOrEq{"created_at": *f.EndTime})
	}
	return qb
}

func scanSystemLogRow(rows pgx.Rows) (*SystemLog, error) {
	var log SystemLog
	var details []byte
	err := rows.Scan(
		&log.ID, &log.Level, &log.Source, &log.Message, &details, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan system_logs row: %w", err)
	}
	if details != nil {
		log.Details = json.RawMessage(details)
	}
	return &log, nil
}

func scanNEMessageLogRow(rows pgx.Rows) (*NEMessageLog, error) {
	var log NEMessageLog
	err := rows.Scan(
		&log.ID, &log.DeviceSN, &log.DeviceID, &log.MessageType, &log.Direction, &log.Content, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan ne_message_logs row: %w", err)
	}
	return &log, nil
}
