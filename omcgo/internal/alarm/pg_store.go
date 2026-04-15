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
		`INSERT INTO alarms_active (id, device_id, device_sn, carrier, severity, alarm_type, alarm_code, description, status, raised_at, additional_info, created_at, updated_at,
		 device_name, technology, alarm_source, event_type, network_location, explicit_cause, is_read, ack_count, first_raised_at, last_updated_at, probable_cause)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
		alarm.ID, alarm.DeviceID, alarm.DeviceSN, alarm.Carrier, alarm.Severity,
		alarm.AlarmType, alarm.AlarmCode, alarm.Description, alarm.Status,
		alarm.RaisedAt, additionalJSON, alarm.CreatedAt, alarm.UpdatedAt,
		alarm.DeviceName, alarm.Technology, alarm.AlarmSource, alarm.EventType,
		alarm.NetworkLocation, alarm.ExplicitCause, alarm.IsRead, alarm.AckCount,
		alarm.FirstRaisedAt, alarm.LastUpdatedAt, alarm.ProbableCause,
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
		 acknowledged_by=$5, additional_info=$6, updated_at=$7, device_name=$8, technology=$9,
		 alarm_source=$10, event_type=$11, network_location=$12, explicit_cause=$13,
		 is_read=$14, ack_count=$15, last_updated_at=$16, probable_cause=$17 WHERE id=$18`,
		alarm.Severity, alarm.Status, alarm.RaisedAt, alarm.AcknowledgedAt,
		alarm.AcknowledgedBy, additionalJSON, time.Now(),
		alarm.DeviceName, alarm.Technology,
		alarm.AlarmSource, alarm.EventType,
		alarm.NetworkLocation, alarm.ExplicitCause,
		alarm.IsRead, alarm.AckCount,
		alarm.LastUpdatedAt, alarm.ProbableCause,
		alarm.ID,
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

	qb = applyHistoryFilters(qb, filter)
	countQb = applyHistoryFilters(countQb, filter)

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

	// Unacknowledged
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM alarms_active WHERE status = 'active'").Scan(&stats.Unacknowledged); err != nil {
		return nil, fmt.Errorf("count unacknowledged: %w", err)
	}

	// Unread
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM alarms_active WHERE is_read = false").Scan(&stats.Unread); err != nil {
		return nil, fmt.Errorf("count unread: %w", err)
	}

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
	// 增强字段
	"device_name", "technology", "alarm_source", "event_type",
	"network_location", "explicit_cause", "is_read", "ack_count",
	"first_raised_at", "last_updated_at", "probable_cause",
}

func (s *PgAlarmStore) HistoryStatistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error) {
	stats := &AlarmStatistics{BySeverity: make(map[model.AlarmSeverity]int64), ByType: make(map[string]int64)}

	var total int64
	if err := s.tsPool.QueryRow(ctx, "SELECT COUNT(*) FROM alarms_history").Scan(&total); err != nil {
		return nil, fmt.Errorf("count history: %w", err)
	}
	stats.TotalActive = total

	rows, err := s.tsPool.Query(ctx, "SELECT severity, COUNT(*) FROM alarms_history GROUP BY severity")
	if err != nil {
		return nil, fmt.Errorf("stats history by severity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sev model.AlarmSeverity
		var cnt int64
		if err := rows.Scan(&sev, &cnt); err == nil {
			stats.BySeverity[sev] = cnt
		}
	}

	rows2, err := s.tsPool.Query(ctx, "SELECT alarm_type, COUNT(*) FROM alarms_history WHERE alarm_type != '' GROUP BY alarm_type")
	if err != nil {
		return nil, fmt.Errorf("stats history by type: %w", err)
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
	if f.StartTime != nil {
		qb = qb.Where(squirrel.GtOrEq{"raised_at": *f.StartTime})
	}
	if f.EndTime != nil {
		qb = qb.Where(squirrel.LtOrEq{"raised_at": *f.EndTime})
	}
	// 新增过滤字段
	if f.AlarmType != nil {
		qb = qb.Where(squirrel.Eq{"alarm_type": *f.AlarmType})
	}
	if f.IsRead != nil {
		qb = qb.Where(squirrel.Eq{"is_read": *f.IsRead})
	}
	if f.DeviceName != nil {
		qb = qb.Where(squirrel.Like{"device_name": "%" + *f.DeviceName + "%"})
	}
	if f.Keyword != nil {
		kw := "%" + *f.Keyword + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.Like{"alarm_code": kw},
			squirrel.Like{"device_sn": kw},
			squirrel.Like{"device_name": kw},
			squirrel.Like{"description": kw},
		})
	}
	if len(f.AlarmCodes) > 0 {
		qb = qb.Where(squirrel.Eq{"alarm_code": f.AlarmCodes})
	}
	if len(f.AlarmSources) > 0 {
		qb = qb.Where(squirrel.Eq{"alarm_source": f.AlarmSources})
	}
	if len(f.DeviceIDs) > 0 {
		qb = qb.Where(squirrel.Eq{"device_id": f.DeviceIDs})
	}
	return qb
}

func applyHistoryFilters(qb squirrel.SelectBuilder, f AlarmFilter) squirrel.SelectBuilder {
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
	if f.StartTime != nil {
		qb = qb.Where(squirrel.GtOrEq{"time": *f.StartTime})
	}
	if f.EndTime != nil {
		qb = qb.Where(squirrel.LtOrEq{"time": *f.EndTime})
	}
	if f.Keyword != nil {
		kw := "%" + *f.Keyword + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.Like{"alarm_code": kw},
			squirrel.Like{"device_sn": kw},
			squirrel.Like{"description": kw},
		})
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
		&a.AcknowledgedAt, &a.AcknowledgedBy, &additionalJSON, &a.CreatedAt, &a.UpdatedAt,
		&a.DeviceName, &a.Technology, &a.AlarmSource, &a.EventType,
		&a.NetworkLocation, &a.ExplicitCause, &a.IsRead, &a.AckCount,
		&a.FirstRaisedAt, &a.LastUpdatedAt, &a.ProbableCause); err != nil {
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

func (s *PgAlarmStore) BatchAcknowledge(ctx context.Context, ids []uuid.UUID, by string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET status = 'acknowledged', acknowledged_at = NOW(), acknowledged_by = $1, updated_at = NOW() WHERE id = ANY($2) AND status != 'cleared'`,
		by, ids)
	return err
}

func (s *PgAlarmStore) BatchUnacknowledge(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET status = 'active', acknowledged_at = NULL, acknowledged_by = NULL, updated_at = NOW() WHERE id = ANY($1) AND status = 'acknowledged'`,
		ids)
	return err
}

func (s *PgAlarmStore) BatchHistoryAcknowledge(ctx context.Context, ids []uuid.UUID, by string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_history SET acknowledged_at = NOW(), acknowledged_by = $1, updated_at = NOW() WHERE alarm_id = ANY($2) AND acknowledged_at IS NULL`,
		by, ids)
	return err
}

func (s *PgAlarmStore) BatchHistoryUnacknowledge(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_history SET acknowledged_at = NULL, acknowledged_by = NULL, updated_at = NOW() WHERE alarm_id = ANY($1) AND acknowledged_at IS NOT NULL`,
		ids)
	return err
}

func (s *PgAlarmStore) BatchHistoryDelete(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.tsPool.Exec(ctx,
		`DELETE FROM alarms_history WHERE alarm_id = ANY($1)`,
		ids)
	return err
}

func (s *PgAlarmStore) BatchClear(ctx context.Context, ids []uuid.UUID) error {
	// 逐条归档到历史表后从活动表删除
	for _, id := range ids {
		alarm, err := s.GetActiveByID(ctx, id)
		if err != nil {
			continue // 不存在的跳过
		}
		now := time.Now()
		alarm.Status = model.AlarmCleared
		alarm.ClearedAt = &now
		if archiveErr := s.Archive(ctx, alarm); archiveErr != nil {
			return fmt.Errorf("archive alarm %s: %w", id, archiveErr)
		}
		if delErr := s.RemoveActive(ctx, id); delErr != nil {
			return fmt.Errorf("remove active alarm %s: %w", id, delErr)
		}
	}
	return nil
}

func (s *PgAlarmStore) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET is_read = TRUE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

var _ AlarmStore = (*PgAlarmStore)(nil)
