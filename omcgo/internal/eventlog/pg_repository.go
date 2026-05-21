package eventlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository event_logs 表 DB 操作。
type Repository interface {
	Create(ctx context.Context, e *EventLog) error
	List(ctx context.Context, filter Filter) ([]*EventLog, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*EventLog, error)
}

var cols = []string{
	"id", "device_id", "device_sn", "device_name", "device_type", "is_gnb",
	"operate_ip", "software_version", "event_type", "event_reason", "event_level",
	"event_data", "occurred_at", "created_at",
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) Create(ctx context.Context, e *EventLog) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = now
	}
	if e.EventLevel == "" {
		e.EventLevel = EventLevelInfo
	}

	// nullable 列空字符串/空切片→ NULL，便于查询区分
	nilIfEmpty := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}
	var eventData interface{}
	if len(e.EventData) > 0 {
		eventData = e.EventData
	}

	query, args, err := storage.Psql.Insert("event_logs").
		Columns(cols...).
		Values(
			e.ID, e.DeviceID, e.DeviceSN, nilIfEmpty(e.DeviceName), nilIfEmpty(e.DeviceType), e.IsGNB,
			nilIfEmpty(e.OperateIP), nilIfEmpty(e.SoftwareVersion), e.EventType,
			nilIfEmpty(e.EventReason), e.EventLevel,
			eventData, e.OccurredAt, e.CreatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert event_logs: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgRepository) List(ctx context.Context, filter Filter) ([]*EventLog, int64, error) {
	base := storage.Psql.Select(cols...).From("event_logs")
	countBase := storage.Psql.Select("COUNT(*)").From("event_logs")

	if filter.DeviceSN != "" {
		// 与异常重启列表对齐，device_sn 走 ILIKE 模糊匹配（部分 SN 也能命中）
		pattern := "%" + filter.DeviceSN + "%"
		base = base.Where(sq.ILike{"device_sn": pattern})
		countBase = countBase.Where(sq.ILike{"device_sn": pattern})
	}
	if filter.EventType != "" {
		base = base.Where(sq.Eq{"event_type": filter.EventType})
		countBase = countBase.Where(sq.Eq{"event_type": filter.EventType})
	}
	if filter.StartTime != nil {
		base = base.Where(sq.GtOrEq{"occurred_at": *filter.StartTime})
		countBase = countBase.Where(sq.GtOrEq{"occurred_at": *filter.StartTime})
	}
	if filter.EndTime != nil {
		base = base.Where(sq.LtOrEq{"occurred_at": *filter.EndTime})
		countBase = countBase.Where(sq.LtOrEq{"occurred_at": *filter.EndTime})
	}

	countQuery, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count event_logs: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count event_logs: %w", err)
	}

	query, args, err := base.
		OrderBy("occurred_at DESC").
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset())).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list event_logs: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query event_logs: %w", err)
	}
	defer rows.Close()

	var items []*EventLog
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	return items, total, rows.Err()
}

func (r *PgRepository) GetByID(ctx context.Context, id uuid.UUID) (*EventLog, error) {
	query, args, err := storage.Psql.Select(cols...).From("event_logs").
		Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select event_logs by id: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	e, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func scan(row pgx.Row) (*EventLog, error) {
	var (
		e               EventLog
		deviceName      *string
		deviceType      *string
		operateIP       *string
		softwareVersion *string
		eventReason     *string
		eventData       []byte
	)
	if err := row.Scan(
		&e.ID, &e.DeviceID, &e.DeviceSN, &deviceName, &deviceType, &e.IsGNB,
		&operateIP, &softwareVersion, &e.EventType, &eventReason, &e.EventLevel,
		&eventData, &e.OccurredAt, &e.CreatedAt,
	); err != nil {
		return nil, err
	}
	derefStr := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	e.DeviceName = derefStr(deviceName)
	e.DeviceType = derefStr(deviceType)
	e.OperateIP = derefStr(operateIP)
	e.SoftwareVersion = derefStr(softwareVersion)
	e.EventReason = derefStr(eventReason)
	e.EventData = eventData
	return &e, nil
}
