package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgAlarmStore implements AlarmStore using PostgreSQL and TimescaleDB.
type PgAlarmStore struct {
	pool   *pgxpool.Pool
	tsPool *pgxpool.Pool
}

// NewPgAlarmStore creates a new PostgreSQL-backed alarm store.
func NewPgAlarmStore(pool *pgxpool.Pool, tsPool *pgxpool.Pool) *PgAlarmStore {
	return &PgAlarmStore{pool: pool, tsPool: tsPool}
}

func (s *PgAlarmStore) SaveActive(ctx context.Context, alarm *model.Alarm) error {
	additionalJSON, _ := json.Marshal(alarm.AdditionalInfo)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO alarms_active (id, device_id, device_sn, carrier, severity, alarm_type, alarm_code, description, status, raised_at, additional_info, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		alarm.ID, alarm.DeviceID, alarm.DeviceSN, alarm.Carrier, alarm.Severity,
		alarm.AlarmType, alarm.AlarmCode, alarm.Description, alarm.Status,
		alarm.RaisedAt, additionalJSON, alarm.CreatedAt, alarm.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert alarms_active: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return s.scanActiveAlarm(ctx, storage.Psql.Select(activeColumns...).From("alarms_active").Where(squirrel.Eq{"id": id}))
}

func (s *PgAlarmStore) GetActiveByDeviceAndCode(ctx context.Context, deviceSN string, alarmCode string) (*model.Alarm, error) {
	return s.scanActiveAlarm(ctx, storage.Psql.Select(activeColumns...).From("alarms_active").
		Where(squirrel.Eq{"device_sn": deviceSN, "alarm_code": alarmCode}).
		Where(squirrel.NotEq{"status": "cleared"}))
}

func (s *PgAlarmStore) UpdateActive(ctx context.Context, alarm *model.Alarm) error {
	additionalJSON, _ := json.Marshal(alarm.AdditionalInfo)
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET severity=$1, status=$2, raised_at=$3, acknowledged_at=$4,
		 acknowledged_by=$5, additional_info=$6, updated_at=$7 WHERE id=$8`,
		alarm.Severity, alarm.Status, alarm.RaisedAt, alarm.AcknowledgedAt,
		alarm.AcknowledgedBy, additionalJSON, time.Now(), alarm.ID,
	)
	if err != nil {
		return fmt.Errorf("update alarms_active: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) RemoveActive(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM alarms_active WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete alarms_active: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) ListActive(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	qb := storage.Psql.Select(activeColumns...).From("alarms_active")
	countQb := storage.Psql.Select("COUNT(*)").From("alarms_active")

	qb = applyActiveFilters(qb, filter)
	countQb = applyActiveFilters(countQb, filter)

	countSQL, countArgs, _ := countQb.ToSql()
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count alarms_active: %w", err)
	}

	qb = qb.OrderBy("raised_at DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))
	sql, args, _ := qb.ToSql()
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query alarms_active: %w", err)
	}
	defer rows.Close()

	var items []model.Alarm
	for rows.Next() {
		a, err := scanAlarmRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *a)
	}
	if items == nil {
		items = []model.Alarm{}
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (s *PgAlarmStore) Archive(ctx context.Context, alarm *model.Alarm) error {
	_, err := s.tsPool.Exec(ctx,
		`INSERT INTO alarms_history (time, alarm_id, device_id, device_sn, carrier, severity, alarm_type, alarm_code, description, status, raised_at, acknowledged_at, cleared_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		time.Now(), alarm.ID, alarm.DeviceID, alarm.DeviceSN, alarm.Carrier,
		alarm.Severity, alarm.AlarmType, alarm.AlarmCode, alarm.Description,
		alarm.Status, alarm.RaisedAt, alarm.AcknowledgedAt, alarm.ClearedAt,
	)
	if err != nil {
		return fmt.Errorf("insert alarms_history: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) ListHistory(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	qb := storage.Psql.Select("time", "alarm_id", "device_id", "device_sn", "carrier", "severity",
		"alarm_type", "alarm_code", "description", "status", "raised_at", "acknowledged_at", "cleared_at").
		From("alarms_history")
	countQb := storage.Psql.Select("COUNT(*)").From("alarms_history")

	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
	}
	if filter.Severity != nil {
		qb = qb.Where(squirrel.Eq{"severity": *filter.Severity})
		countQb = countQb.Where(squirrel.Eq{"severity": *filter.Severity})
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
		return nil, fmt.Errorf("count alarms_history: %w", err)
	}

	qb = qb.OrderBy("time DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))
	sql, args, _ := qb.ToSql()
	rows, err := s.tsPool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query alarms_history: %w", err)
	}
	defer rows.Close()

	var items []model.Alarm
	for rows.Next() {
		var a model.Alarm
		var eventTime time.Time
		if err := rows.Scan(&eventTime, &a.ID, &a.DeviceID, &a.DeviceSN, &a.Carrier, &a.Severity,
			&a.AlarmType, &a.AlarmCode, &a.Description, &a.Status, &a.RaisedAt, &a.AcknowledgedAt, &a.ClearedAt); err != nil {
			return nil, fmt.Errorf("scan alarms_history: %w", err)
		}
		items = append(items, a)
	}
	if items == nil {
		items = []model.Alarm{}
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (s *PgAlarmStore) Statistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error) {
	stats := &AlarmStatistics{BySeverity: make(map[model.AlarmSeverity]int64), ByType: make(map[string]int64)}

	// Total active
	var total int64
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM alarms_active").Scan(&total); err != nil {
		return nil, fmt.Errorf("count active: %w", err)
	}
	stats.TotalActive = total

	// By severity
	rows, err := s.pool.Query(ctx, "SELECT severity, COUNT(*) FROM alarms_active GROUP BY severity")
	if err != nil {
		return nil, fmt.Errorf("stats by severity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sev model.AlarmSeverity
		var cnt int64
		if err := rows.Scan(&sev, &cnt); err == nil {
			stats.BySeverity[sev] = cnt
		}
	}

	// By type
	rows2, err := s.pool.Query(ctx, "SELECT alarm_type, COUNT(*) FROM alarms_active WHERE alarm_type != '' GROUP BY alarm_type")
	if err != nil {
		return nil, fmt.Errorf("stats by type: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var typ string
		var cnt int64
		if err := rows2.Scan(&typ, &cnt); err == nil {
			stats.ByType[typ] = cnt
		}
	}

	return stats, nil
}

// helpers

var activeColumns = []string{
	"id", "device_id", "device_sn", "carrier", "severity", "alarm_type", "alarm_code",
	"description", "status", "raised_at", "acknowledged_at", "acknowledged_by",
	"additional_info", "created_at", "updated_at",
}

func applyActiveFilters(qb squirrel.SelectBuilder, f AlarmFilter) squirrel.SelectBuilder {
	if f.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *f.DeviceID})
	}
	if f.DeviceSN != nil {
		qb = qb.Where(squirrel.Eq{"device_sn": *f.DeviceSN})
	}
	if f.Carrier != nil {
		qb = qb.Where(squirrel.Eq{"carrier": *f.Carrier})
	}
	if f.Severity != nil {
		qb = qb.Where(squirrel.Eq{"severity": *f.Severity})
	}
	if f.Status != nil {
		qb = qb.Where(squirrel.Eq{"status": *f.Status})
	}
	return qb
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanAlarmRow(row scannable) (*model.Alarm, error) {
	var a model.Alarm
	var additionalJSON []byte
	if err := row.Scan(&a.ID, &a.DeviceID, &a.DeviceSN, &a.Carrier, &a.Severity,
		&a.AlarmType, &a.AlarmCode, &a.Description, &a.Status, &a.RaisedAt,
		&a.AcknowledgedAt, &a.AcknowledgedBy, &additionalJSON, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan alarm: %w", err)
	}
	if len(additionalJSON) > 0 {
		_ = json.Unmarshal(additionalJSON, &a.AdditionalInfo)
	}
	return &a, nil
}

func (s *PgAlarmStore) scanActiveAlarm(ctx context.Context, qb squirrel.SelectBuilder) (*model.Alarm, error) {
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	row := s.pool.QueryRow(ctx, sql, args...)
	return scanAlarmRow(row)
}

var _ AlarmStore = (*PgAlarmStore)(nil)
