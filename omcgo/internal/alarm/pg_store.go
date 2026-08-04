package alarm

import (
	"context"
	"encoding/json"
	gerr "errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/authz"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
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
	ensureAlarmVersion(alarm)
	additionalJSON, _ := json.Marshal(alarm.AdditionalInfo)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save active tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx,
		`INSERT INTO alarms_active (id, device_id, device_sn, carrier, severity, alarm_type, alarm_identifier, description, status, raised_at, acknowledged_at, acknowledged_by, ack_note, additional_info, created_at, updated_at,
			 device_name, technology, alarm_source, event_type, network_location, explicit_cause, is_read, ack_count, first_raised_at, last_updated_at, probable_cause, is_unknown, alarm_version)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29)`,
		alarm.ID, alarm.DeviceID, alarm.DeviceSN, alarm.Carrier, alarm.Severity,
		alarm.AlarmType, alarm.AlarmIdentifier, alarm.Description, alarm.Status,
		alarm.RaisedAt, alarm.AcknowledgedAt, alarm.AcknowledgedBy, alarm.AckNote, additionalJSON, alarm.CreatedAt, alarm.UpdatedAt,
		alarm.DeviceName, alarm.Technology, alarm.AlarmSource, alarm.EventType,
		alarm.NetworkLocation, alarm.ExplicitCause, alarm.IsRead, alarm.AckCount,
		alarm.FirstRaisedAt, alarm.LastUpdatedAt, alarm.ProbableCause, alarm.IsUnknown, alarm.Version,
	)
	if err != nil {
		return fmt.Errorf("insert alarms_active: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save active tx: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return s.scanActiveAlarm(ctx, activeAlarmSelect().Where(squirrel.Eq{"alarms_active.id": id}))
}

func (s *PgAlarmStore) GetHistoryByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	qb := historyAlarmRawSelect().
		Where(squirrel.Eq{"alarms_history.alarm_id": id}).
		OrderBy("alarms_history.time DESC").
		Limit(1)
	return s.scanHistoryAlarm(ctx, qb)
}

// GetActiveByIDLocalized / GetHistoryByIDLocalized 仅供用户可见详情查询使用。
// 告警引擎继续调用上面的 raw 方法，避免确认/清除时把展示语言回写到持久化记录。
func (s *PgAlarmStore) GetActiveByIDLocalized(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return s.scanActiveAlarm(ctx, activeAlarmLocalizedSelect(appcontext.GetLocale(ctx)).Where(squirrel.Eq{"alarms_active.id": id}))
}

func (s *PgAlarmStore) GetHistoryByIDLocalized(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	qb := historyAlarmSelect(appcontext.GetLocale(ctx)).
		Where(squirrel.Eq{"alarms_history.alarm_id": id}).
		OrderBy("alarms_history.time DESC").
		Limit(1)
	return s.scanHistoryAlarm(ctx, qb)
}

func (s *PgAlarmStore) GetActiveByDeviceAndIdentifier(ctx context.Context, deviceSN string, alarmIdentifier string) (*model.Alarm, error) {
	return s.scanActiveAlarm(ctx, activeAlarmSelect().
		Where(squirrel.Eq{"alarms_active.device_sn": deviceSN, "alarms_active.alarm_identifier": alarmIdentifier}).
		Where(squirrel.NotEq{"alarms_active.status": "cleared"}))
}

func (s *PgAlarmStore) GetActiveByDeviceSN(ctx context.Context, deviceSN string) ([]*model.Alarm, error) {
	q := activeAlarmSelect().
		Where(squirrel.Eq{"alarms_active.device_sn": deviceSN}).
		Where(squirrel.NotEq{"alarms_active.status": "cleared"})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build GetActiveByDeviceSN query: %w", err)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query GetActiveByDeviceSN: %w", err)
	}
	defer rows.Close()

	var alarms []*model.Alarm
	for rows.Next() {
		a, err := scanAlarmRow(rows)
		if err != nil {
			return nil, err
		}
		alarms = append(alarms, a)
	}

	return alarms, nil
}

func (s *PgAlarmStore) UpdateActive(ctx context.Context, alarm *model.Alarm) error {
	ensureAlarmVersion(alarm)
	additionalJSON, _ := json.Marshal(alarm.AdditionalInfo)
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET severity=$1, status=$2, raised_at=$3, acknowledged_at=$4,
			 acknowledged_by=$5, ack_note=$6, additional_info=$7, updated_at=$8, device_name=$9, technology=$10,
			 alarm_source=$11, event_type=$12, network_location=$13, explicit_cause=$14,
			 is_read=$15, ack_count=$16, last_updated_at=$17, probable_cause=$18, alarm_version=$19 WHERE id=$20`,
		alarm.Severity, alarm.Status, alarm.RaisedAt, alarm.AcknowledgedAt,
		alarm.AcknowledgedBy, alarm.AckNote, additionalJSON, time.Now(),
		alarm.DeviceName, alarm.Technology,
		alarm.AlarmSource, alarm.EventType,
		alarm.NetworkLocation, alarm.ExplicitCause,
		alarm.IsRead, alarm.AckCount,
		alarm.LastUpdatedAt, alarm.ProbableCause, alarm.Version,
		alarm.ID,
	)
	if err != nil {
		return fmt.Errorf("update alarms_active: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) RemoveActive(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin remove active tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 获取 device_id，供删除后同事务维护计数。
	var deviceID uuid.UUID
	if err = tx.QueryRow(ctx, "SELECT device_id FROM alarms_active WHERE id = $1", id).Scan(&deviceID); err != nil {
		if gerr.Is(err, pgx.ErrNoRows) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf("commit remove active tx (no rows): %w", commitErr)
			}
			return nil
		}
		return fmt.Errorf("get device_id for alarm %s: %w", id, err)
	}

	res, err := tx.Exec(ctx, "DELETE FROM alarms_active WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete alarms_active: %w", err)
	}
	if res.RowsAffected() == 0 {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return fmt.Errorf("commit remove active tx (delete no rows): %w", commitErr)
		}
		return nil
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit remove active tx: %w", err)
	}
	return nil
}

func (s *PgAlarmStore) ListActive(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	qb := activeAlarmListSelect(appcontext.GetLocale(ctx))
	countQb := storage.Psql.Select("COUNT(*)").From("alarms_active").LeftJoin("devices d ON d.id = alarms_active.device_id")

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
	ensureAlarmVersion(alarm)
	additionalJSON, _ := json.Marshal(alarm.AdditionalInfo)
	archiveUpdatedAt := historyUpdatedAt(alarm)
	if archiveUpdatedAt.IsZero() {
		archiveUpdatedAt = time.Now()
	}
	// alarms_history 在时序库（TsPool）。
	_, err := s.tsPool.Exec(ctx,
		`INSERT INTO alarms_history (time, alarm_id, device_id, device_sn, carrier, severity, alarm_type, alarm_identifier, description, status, raised_at, acknowledged_at, cleared_at, device_name, technology, alarm_source, event_type, network_location, explicit_cause, ack_count, acknowledged_by, ack_note, additional_info, updated_at, cleared_by, clear_note, probable_cause, alarm_version)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)`,
		time.Now(), alarm.ID, alarm.DeviceID, alarm.DeviceSN, alarm.Carrier,
		alarm.Severity, alarm.AlarmType, alarm.AlarmIdentifier, alarm.Description,
		alarm.Status, alarm.RaisedAt, alarm.AcknowledgedAt, alarm.ClearedAt,
		alarm.DeviceName, alarm.Technology, alarm.AlarmSource, alarm.EventType,
		alarm.NetworkLocation, alarm.ExplicitCause, alarm.AckCount,
		alarm.AcknowledgedBy, alarm.AckNote, additionalJSON, archiveUpdatedAt,
		alarm.ClearedBy, alarm.ClearNote,
		alarm.ProbableCause, alarm.Version,
	)
	if err != nil {
		return fmt.Errorf("insert alarms_history: %w", err)
	}
	return nil
}

func historyUpdatedAt(alarm *model.Alarm) time.Time {
	if !alarm.LastUpdatedAt.IsZero() {
		return alarm.LastUpdatedAt
	}
	return alarm.UpdatedAt
}

func ensureAlarmVersion(alarm *model.Alarm) {
	if alarm.Version < 1 {
		alarm.Version = 1
	}
}

func (s *PgAlarmStore) ListHistory(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	// issue #67：description 列改为按请求 locale 取 alarm_definitions 本地化名，字典未命中
	// 退回设备上报原文（COALESCE 兜底）；其余列与列表 scan 顺序保持不变。
	qb := historyAlarmSelect(appcontext.GetLocale(ctx))
	countQb := storage.Psql.Select("COUNT(*)").From("alarms_history").LeftJoin("device_dim d ON d.id = alarms_history.device_id")

	qb = applyHistoryFilters(qb, filter)
	countQb = applyHistoryFilters(countQb, filter)

	// alarms_history 与其 JOIN 的影子表（device_dim/alarm_definition_dim）都在时序库（TsPool）。
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
		a, err := scanHistoryAlarmRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan alarms_history: %w", err)
		}
		items = append(items, *a)
	}
	if items == nil {
		items = []model.Alarm{}
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// activeStatsBase 返回带 filter 的 alarms_active 统计基底查询。
// 与 ListActive 保持同一 LEFT JOIN，使 applyActiveFilters 内的 d.technology 等
// 跨表条件可用；统计随之按 filter 收窄，而非旧实现的全表扫描。
func activeStatsBase(filter AlarmFilter) squirrel.SelectBuilder {
	qb := storage.Psql.Select().From("alarms_active").LeftJoin("devices d ON d.id = alarms_active.device_id")
	return applyActiveFilters(qb, filter)
}

func (s *PgAlarmStore) Statistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error) {
	stats := &AlarmStatistics{BySeverity: make(map[model.AlarmSeverity]int64), ByType: make(map[string]int64)}

	totalSQL, totalArgs, err := activeStatsBase(filter).Column("COUNT(*)").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count active: %w", err)
	}
	if err := s.pool.QueryRow(ctx, totalSQL, totalArgs...).Scan(&stats.TotalActive); err != nil {
		return nil, fmt.Errorf("count active: %w", err)
	}

	unackSQL, unackArgs, err := activeStatsBase(filter).
		Column("COUNT(*)").
		Where(squirrel.Eq{"alarms_active.status": "active"}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count unacknowledged: %w", err)
	}
	if err := s.pool.QueryRow(ctx, unackSQL, unackArgs...).Scan(&stats.Unacknowledged); err != nil {
		return nil, fmt.Errorf("count unacknowledged: %w", err)
	}

	unreadSQL, unreadArgs, err := activeStatsBase(filter).
		Column("COUNT(*)").
		Where(squirrel.Eq{"alarms_active.is_read": false}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count unread: %w", err)
	}
	if err := s.pool.QueryRow(ctx, unreadSQL, unreadArgs...).Scan(&stats.Unread); err != nil {
		return nil, fmt.Errorf("count unread: %w", err)
	}

	versionSQL, versionArgs, err := activeStatsBase(filter).
		Column("COALESCE(SUM((EXTRACT(EPOCH FROM alarms_active.updated_at))::bigint), 0)::bigint AS state_version").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build state version: %w", err)
	}
	if err := s.pool.QueryRow(ctx, versionSQL, versionArgs...).Scan(&stats.StateVersion); err != nil {
		return nil, fmt.Errorf("count state version: %w", err)
	}

	sevSQL, sevArgs, err := activeStatsBase(filter).
		Columns("alarms_active.severity", "COUNT(*)").
		GroupBy("alarms_active.severity").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build stats by severity: %w", err)
	}
	rows, err := s.pool.Query(ctx, sevSQL, sevArgs...)
	if err != nil {
		return nil, fmt.Errorf("stats by severity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sev model.AlarmSeverity
		var cnt int64
		if err := rows.Scan(&sev, &cnt); err == nil {
			stats.BySeverity[canonicalAlarmSeverity(sev)] += cnt
		}
	}

	typeSQL, typeArgs, err := activeStatsBase(filter).
		Columns("alarms_active.alarm_type", "COUNT(*)").
		Where(squirrel.NotEq{"alarms_active.alarm_type": ""}).
		GroupBy("alarms_active.alarm_type").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build stats by type: %w", err)
	}
	rows2, err := s.pool.Query(ctx, typeSQL, typeArgs...)
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
	"alarms_active.id", "alarms_active.device_id", "alarms_active.device_sn", "alarms_active.carrier", "alarms_active.severity", "alarms_active.alarm_type", "alarms_active.alarm_identifier",
	"alarms_active.description", "alarms_active.status", "alarms_active.raised_at", "alarms_active.acknowledged_at", "alarms_active.acknowledged_by", "alarms_active.ack_note",
	"alarms_active.additional_info", "alarms_active.created_at", "alarms_active.updated_at",
	// 增强字段
	"alarms_active.device_name", "COALESCE(alarms_active.technology, d.technology) AS technology", "alarms_active.alarm_source", "alarms_active.event_type",
	"alarms_active.network_location", "alarms_active.explicit_cause", "alarms_active.is_read", "alarms_active.ack_count",
	"alarms_active.first_raised_at", "alarms_active.last_updated_at", "alarms_active.probable_cause",
	// T-0098 P2-10
	"alarms_active.is_unknown", "alarms_active.alarm_version",
}

func activeAlarmSelect() squirrel.SelectBuilder {
	return storage.Psql.Select(activeColumns...).From("alarms_active").LeftJoin("devices d ON d.id = alarms_active.device_id")
}

// localizedAlarmNameExpr 返回列表查询里 description 列的 locale 化表达式（issue #67）。
//
// 按请求 locale 在 alarm_definitions.en_name / cn_name 间取名：identifier 命中
// 时仅在中英文字典列间回退，未命中时才保留告警记录原文。
//
// 别名固定为 description，使列表 scan 顺序与既有 scanAlarmRow / ListHistory 解码完全一致。
func localizedAlarmNameExpr(loc appcontext.Locale, table string) string {
	descCol := table + ".description"
	primary, fallback := "ad.cn_name", "ad.en_name"
	if loc == appcontext.LocaleEN {
		primary, fallback = fallback, primary
	}
	return fmt.Sprintf(
		"CASE WHEN ad.identifier IS NULL THEN COALESCE(%s, '') "+
			"ELSE COALESCE(NULLIF(%s, ''), NULLIF(%s, ''), '') END AS description",
		descCol, primary, fallback,
	)
}

// localizedProbableCauseExpr 与告警名称使用同一 identifier 命中规则，不区分告警来源。
func localizedProbableCauseExpr(loc appcontext.Locale, table string) string {
	causeCol := table + ".probable_cause"
	primary, fallback := "ad.cn_probable_cause", "ad.en_probable_cause"
	if loc == appcontext.LocaleEN {
		primary, fallback = fallback, primary
	}
	return fmt.Sprintf(
		"CASE WHEN ad.identifier IS NULL THEN COALESCE(%s, '') "+
			"ELSE COALESCE(NULLIF(%s, ''), NULLIF(%s, ''), '') END AS probable_cause",
		causeCol, primary, fallback,
	)
}

func localizedActiveColumns(loc appcontext.Locale) []string {
	cols := make([]string, len(activeColumns))
	copy(cols, activeColumns)
	for i, column := range cols {
		switch column {
		case "alarms_active.description":
			cols[i] = localizedAlarmNameExpr(loc, "alarms_active")
		case "alarms_active.probable_cause":
			cols[i] = localizedProbableCauseExpr(loc, "alarms_active")
		}
	}
	return cols
}

func activeAlarmLocalizedSelect(loc appcontext.Locale) squirrel.SelectBuilder {
	return storage.Psql.Select(localizedActiveColumns(loc)...).
		From("alarms_active").
		LeftJoin("devices d ON d.id = alarms_active.device_id").
		LeftJoin("alarm_definitions ad ON ad.identifier = alarms_active.alarm_identifier")
}

// activeAlarmListSelect 构建活跃告警**列表**查询，description 列按 locale 取字典本地化名
// （issue #67）。与 activeAlarmSelect（详情/GetByID 用，列表外保持原文）区别仅在 description
// 列与多一个 alarm_definitions LEFT JOIN，scan 顺序不变（仍走 scanAlarmRow）。
func activeAlarmListSelect(loc appcontext.Locale) squirrel.SelectBuilder {
	return activeAlarmLocalizedSelect(loc)
}

func historyAlarmSelect(loc appcontext.Locale) squirrel.SelectBuilder {
	return storage.Psql.Select(
		"alarms_history.time", "alarms_history.alarm_id", "alarms_history.device_id", "alarms_history.device_sn", "alarms_history.carrier", "alarms_history.severity",
		"COALESCE(alarms_history.alarm_type, '') AS alarm_type", "alarms_history.alarm_identifier",
		localizedAlarmNameExpr(loc, "alarms_history"),
		"alarms_history.status", "alarms_history.raised_at",
		"alarms_history.acknowledged_at", "alarms_history.cleared_at", "alarms_history.acknowledged_by", "alarms_history.ack_note",
		"alarms_history.additional_info",
		"alarms_history.device_name", "COALESCE(alarms_history.technology, d.technology) AS technology", "alarms_history.alarm_source", "alarms_history.event_type",
		"alarms_history.ack_count", "alarms_history.updated_at",
		"alarms_history.cleared_by", "alarms_history.clear_note",
		localizedProbableCauseExpr(loc, "alarms_history"),
		"alarms_history.alarm_version",
	).From("alarms_history").
		LeftJoin("device_dim d ON d.id = alarms_history.device_id").
		LeftJoin("alarm_definition_dim ad ON ad.identifier = alarms_history.alarm_identifier")
}

func historyAlarmRawSelect() squirrel.SelectBuilder {
	return storage.Psql.Select(
		"alarms_history.time", "alarms_history.alarm_id", "alarms_history.device_id", "alarms_history.device_sn", "alarms_history.carrier", "alarms_history.severity",
		"COALESCE(alarms_history.alarm_type, '') AS alarm_type", "alarms_history.alarm_identifier", "COALESCE(alarms_history.description, '') AS description",
		"alarms_history.status", "alarms_history.raised_at",
		"alarms_history.acknowledged_at", "alarms_history.cleared_at", "alarms_history.acknowledged_by", "alarms_history.ack_note",
		"alarms_history.additional_info",
		"alarms_history.device_name", "COALESCE(alarms_history.technology, d.technology) AS technology", "alarms_history.alarm_source", "alarms_history.event_type",
		"alarms_history.ack_count", "alarms_history.updated_at",
		"alarms_history.cleared_by", "alarms_history.clear_note",
		"alarms_history.probable_cause",
		"alarms_history.alarm_version",
	).From("alarms_history").
		LeftJoin("device_dim d ON d.id = alarms_history.device_id")
}

// historyStatsBase 返回带 filter 的 alarms_history 统计基底查询。
// 与 ListHistory 保持同一 LEFT JOIN，使 applyHistoryFilters 内的 d.technology 等
// 跨表条件可用；统计随之按 filter 收窄，而非旧实现的全表扫描。
func historyStatsBase(filter AlarmFilter) squirrel.SelectBuilder {
	qb := storage.Psql.Select().From("alarms_history").LeftJoin("device_dim d ON d.id = alarms_history.device_id")
	return applyHistoryFilters(qb, filter)
}

func (s *PgAlarmStore) HistoryStatistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error) {
	stats := &AlarmStatistics{BySeverity: make(map[model.AlarmSeverity]int64), ByType: make(map[string]int64)}

	// alarms_history 及其 JOIN 的 device_dim 影子表都在时序库（TsPool）。
	totalSQL, totalArgs, err := historyStatsBase(filter).Column("COUNT(*)").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count history: %w", err)
	}
	if err := s.tsPool.QueryRow(ctx, totalSQL, totalArgs...).Scan(&stats.TotalActive); err != nil {
		return nil, fmt.Errorf("count history: %w", err)
	}

	versionSQL, versionArgs, err := historyStatsBase(filter).
		Column("COALESCE(SUM((EXTRACT(EPOCH FROM alarms_history.updated_at))::bigint), 0)::bigint AS state_version").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build history state version: %w", err)
	}
	if err := s.tsPool.QueryRow(ctx, versionSQL, versionArgs...).Scan(&stats.StateVersion); err != nil {
		return nil, fmt.Errorf("count history state version: %w", err)
	}

	sevSQL, sevArgs, err := historyStatsBase(filter).
		Columns("alarms_history.severity", "COUNT(*)").
		GroupBy("alarms_history.severity").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build stats history by severity: %w", err)
	}
	rows, err := s.tsPool.Query(ctx, sevSQL, sevArgs...)
	if err != nil {
		return nil, fmt.Errorf("stats history by severity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sev model.AlarmSeverity
		var cnt int64
		if err := rows.Scan(&sev, &cnt); err == nil {
			stats.BySeverity[canonicalAlarmSeverity(sev)] += cnt
		}
	}

	typeSQL, typeArgs, err := historyStatsBase(filter).
		Columns("alarms_history.alarm_type", "COUNT(*)").
		Where(squirrel.NotEq{"alarms_history.alarm_type": ""}).
		GroupBy("alarms_history.alarm_type").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build stats history by type: %w", err)
	}
	rows2, err := s.tsPool.Query(ctx, typeSQL, typeArgs...)
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
		qb = qb.Where(squirrel.Eq{"alarms_active.device_id": *f.DeviceID})
	}
	if f.DeviceSN != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.device_sn": *f.DeviceSN})
	}
	if f.Carrier != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.carrier": *f.Carrier})
	}
	if len(f.Severities) > 0 {
		qb = qb.Where(squirrel.Eq{"alarms_active.severity": expandSeverityAliases(f.Severities)})
	} else if f.Severity != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.severity": severityAliases(*f.Severity)})
	}
	if f.Status != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.status": *f.Status})
	}
	if f.StartTime != nil {
		qb = qb.Where(squirrel.GtOrEq{"alarms_active.raised_at": *f.StartTime})
	}
	if f.EndTime != nil {
		qb = qb.Where(squirrel.LtOrEq{"alarms_active.raised_at": *f.EndTime})
	}
	if f.AlarmType != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.alarm_type": *f.AlarmType})
	}
	if f.EventType != nil {
		qb = qb.Where(normalizedEventTypeExpr("alarms_active.event_type", *f.EventType))
	}
	if f.IsRead != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.is_read": *f.IsRead})
	}
	if f.IsUnknown != nil {
		qb = qb.Where(squirrel.Eq{"alarms_active.is_unknown": *f.IsUnknown})
	}
	if f.DeviceName != nil {
		qb = qb.Where(squirrel.Like{"alarms_active.device_name": "%" + *f.DeviceName + "%"})
	}
	if technologyExpr := normalizedTechnologyExpr("COALESCE(alarms_active.technology, d.technology)", f.Technologies); technologyExpr != nil {
		qb = qb.Where(technologyExpr)
	}
	if f.Keyword != nil {
		kw := "%" + *f.Keyword + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.Like{"alarms_active.alarm_identifier": kw},
			squirrel.Like{"alarms_active.device_sn": kw},
			squirrel.Like{"alarms_active.device_name": kw},
			squirrel.Like{"alarms_active.description": kw},
		})
	}
	if len(f.AlarmIdentifiers) > 0 {
		qb = qb.Where(squirrel.Eq{"alarms_active.alarm_identifier": f.AlarmIdentifiers})
	}
	if len(f.AlarmSources) > 0 {
		qb = qb.Where(squirrel.Eq{"alarms_active.alarm_source": f.AlarmSources})
	}
	if len(f.DeviceIDs) > 0 {
		qb = qb.Where(squirrel.Eq{"alarms_active.device_id": f.DeviceIDs})
	}
	// #64 设备组可见性 fail-closed 过滤（统一强制层）。
	qb = authz.ApplyDeviceVisibilityFilter(qb, "alarms_active.device_id", f.VisibleGroups)
	return qb
}

func applyHistoryFilters(qb squirrel.SelectBuilder, f AlarmFilter) squirrel.SelectBuilder {
	if f.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"alarms_history.device_id": *f.DeviceID})
	}
	if f.DeviceSN != nil {
		qb = qb.Where(squirrel.Eq{"alarms_history.device_sn": *f.DeviceSN})
	}
	if f.Carrier != nil {
		qb = qb.Where(squirrel.Eq{"alarms_history.carrier": *f.Carrier})
	}
	if len(f.Severities) > 0 {
		qb = qb.Where(squirrel.Eq{"alarms_history.severity": expandSeverityAliases(f.Severities)})
	} else if f.Severity != nil {
		qb = qb.Where(squirrel.Eq{"alarms_history.severity": severityAliases(*f.Severity)})
	}
	if f.StartTime != nil {
		qb = qb.Where(squirrel.GtOrEq{"alarms_history.time": *f.StartTime})
	}
	if f.EndTime != nil {
		qb = qb.Where(squirrel.LtOrEq{"alarms_history.time": *f.EndTime})
	}
	if f.EventType != nil {
		qb = qb.Where(normalizedEventTypeExpr("alarms_history.event_type", *f.EventType))
	}
	if technologyExpr := normalizedTechnologyExpr("COALESCE(alarms_history.technology, d.technology)", f.Technologies); technologyExpr != nil {
		qb = qb.Where(technologyExpr)
	}
	if f.Keyword != nil {
		kw := "%" + *f.Keyword + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.Like{"alarms_history.alarm_identifier": kw},
			squirrel.Like{"alarms_history.device_sn": kw},
			squirrel.Like{"alarms_history.description": kw},
		})
	}
	// #64 设备组可见性 fail-closed 过滤（统一强制层）。
	qb = authz.ApplyDeviceVisibilityFilter(qb, "alarms_history.device_id", f.VisibleGroups)
	return qb
}

func normalizedEventTypeExpr(column string, raw string) squirrel.Sqlizer {
	aliases := normalizedEventTypeAliases(raw)
	placeholders := make([]string, 0, len(aliases))
	args := make([]interface{}, 0, len(aliases))
	for _, alias := range aliases {
		placeholders = append(placeholders, "?")
		args = append(args, alias)
	}

	expr := fmt.Sprintf(
		"REPLACE(REPLACE(REPLACE(LOWER(COALESCE(%s, '')), ' ', ''), '-', ''), '_', '') IN (%s)",
		column,
		strings.Join(placeholders, ", "),
	)
	return squirrel.Expr(expr, args...)
}

func normalizedTechnologyExpr(column string, values []string) squirrel.Sqlizer {
	var exprs squirrel.Or
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		aliases := normalizedTechnologyAliases(trimmed)
		placeholders := make([]string, 0, len(aliases))
		args := make([]interface{}, 0, len(aliases))
		for _, alias := range aliases {
			placeholders = append(placeholders, "?")
			args = append(args, alias)
		}

		expr := fmt.Sprintf(
			"REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(LOWER(COALESCE(%s, '')), ' ', ''), '-', ''), '_', ''), '(', ''), ')', '') IN (%s)",
			column,
			strings.Join(placeholders, ", "),
		)
		exprs = append(exprs, squirrel.Expr(expr, args...))
	}
	if len(exprs) == 0 {
		return nil
	}
	if len(exprs) == 1 {
		return exprs[0]
	}
	return exprs
}

func canonicalAlarmSeverity(severity model.AlarmSeverity) model.AlarmSeverity {
	switch severity {
	case 31001:
		return model.AlarmCritical
	case 31002:
		return model.AlarmMajor
	case 31003:
		return model.AlarmMinor
	case 31004:
		return model.AlarmWarning
	default:
		return severity
	}
}

func severityAliases(severity model.AlarmSeverity) []model.AlarmSeverity {
	switch canonicalAlarmSeverity(severity) {
	case model.AlarmCritical:
		return []model.AlarmSeverity{model.AlarmCritical, 31001}
	case model.AlarmMajor:
		return []model.AlarmSeverity{model.AlarmMajor, 31002}
	case model.AlarmMinor:
		return []model.AlarmSeverity{model.AlarmMinor, 31003}
	case model.AlarmWarning:
		return []model.AlarmSeverity{model.AlarmWarning, 31004}
	default:
		return []model.AlarmSeverity{severity}
	}
}

func expandSeverityAliases(severities []model.AlarmSeverity) []model.AlarmSeverity {
	expanded := make([]model.AlarmSeverity, 0, len(severities)*2)
	seen := make(map[model.AlarmSeverity]struct{}, len(severities)*2)
	for _, severity := range severities {
		for _, alias := range severityAliases(severity) {
			if _, exists := seen[alias]; exists {
				continue
			}
			seen[alias] = struct{}{}
			expanded = append(expanded, alias)
		}
	}
	return expanded
}

func normalizedTechnologyAliases(raw string) []string {
	key := normalizeTechnologyToken(raw)
	buckets := map[string][]string{
		"enb":    {"enb", "lte", "enodeb"},
		"lte":    {"enb", "lte", "enodeb"},
		"enodeb": {"enb", "lte", "enodeb"},
		"gnb":    {"gnb", "nr", "5gnr", "gnodeb"},
		"nr":     {"gnb", "nr", "5gnr", "gnodeb"},
		"5gnr":   {"gnb", "nr", "5gnr", "gnodeb"},
		"gnodeb": {"gnb", "nr", "5gnr", "gnodeb"},
		"gsm":    {"gsm"},
	}

	if aliases, ok := buckets[key]; ok {
		return aliases
	}
	return []string{key}
}

func normalizedEventTypeAliases(raw string) []string {
	key := normalizeEventTypeToken(raw)
	buckets := map[string][]string{
		"30000":                 {"30000", "communication", "communications", "communicationalarm", "communicationsalarm"},
		"communication":         {"30000", "communication", "communications", "communicationalarm", "communicationsalarm"},
		"communications":        {"30000", "communication", "communications", "communicationalarm", "communicationsalarm"},
		"communicationalarm":    {"30000", "communication", "communications", "communicationalarm", "communicationsalarm"},
		"communicationsalarm":   {"30000", "communication", "communications", "communicationalarm", "communicationsalarm"},
		"30001":                 {"30001", "qualityofservice", "qualityofservicealarm"},
		"qualityofservice":      {"30001", "qualityofservice", "qualityofservicealarm"},
		"qualityofservicealarm": {"30001", "qualityofservice", "qualityofservicealarm"},
		"30002":                 {"30002", "processingerror", "processingerroralarm"},
		"processingerror":       {"30002", "processingerror", "processingerroralarm"},
		"processingerroralarm":  {"30002", "processingerror", "processingerroralarm"},
		"30003":                 {"30003", "device", "equipment", "devicealarm", "equipmentalarm"},
		"device":                {"30003", "device", "equipment", "devicealarm", "equipmentalarm"},
		"equipment":             {"30003", "device", "equipment", "devicealarm", "equipmentalarm"},
		"devicealarm":           {"30003", "device", "equipment", "devicealarm", "equipmentalarm"},
		"equipmentalarm":        {"30003", "device", "equipment", "devicealarm", "equipmentalarm"},
		"30004":                 {"30004", "environment", "environmental", "environmentalarm", "environmentalalarm"},
		"environment":           {"30004", "environment", "environmental", "environmentalarm", "environmentalalarm"},
		"environmental":         {"30004", "environment", "environmental", "environmentalarm", "environmentalalarm"},
		"environmentalarm":      {"30004", "environment", "environmental", "environmentalarm", "environmentalalarm"},
		"environmentalalarm":    {"30004", "environment", "environmental", "environmentalarm", "environmentalalarm"},
		"30006":                 {"30006", "30007", "performance", "service", "performancealarm", "servicealarm"},
		"30007":                 {"30006", "30007", "performance", "service", "performancealarm", "servicealarm"},
		"performance":           {"30006", "30007", "performance", "service", "performancealarm", "servicealarm"},
		"service":               {"30006", "30007", "performance", "service", "performancealarm", "servicealarm"},
		"performancealarm":      {"30006", "30007", "performance", "service", "performancealarm", "servicealarm"},
		"servicealarm":          {"30006", "30007", "performance", "service", "performancealarm", "servicealarm"},
	}

	if aliases, ok := buckets[key]; ok {
		return aliases
	}
	return []string{key}
}

func normalizeEventTypeToken(raw string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

func normalizeTechnologyToken(raw string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", "(", "", ")", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanAlarmRow(row scannable) (*model.Alarm, error) {
	var a model.Alarm
	var additionalJSON []byte
	if err := row.Scan(&a.ID, &a.DeviceID, &a.DeviceSN, &a.Carrier, &a.Severity,
		&a.AlarmType, &a.AlarmIdentifier, &a.Description, &a.Status, &a.RaisedAt,
		&a.AcknowledgedAt, &a.AcknowledgedBy, &a.AckNote, &additionalJSON, &a.CreatedAt, &a.UpdatedAt,
		&a.DeviceName, &a.Technology, &a.AlarmSource, &a.EventType,
		&a.NetworkLocation, &a.ExplicitCause, &a.IsRead, &a.AckCount,
		&a.FirstRaisedAt, &a.LastUpdatedAt, &a.ProbableCause, &a.IsUnknown, &a.Version); err != nil {
		if gerr.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("alarm not found: %w", commonerrors.ErrNotFound)
		}
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

func (s *PgAlarmStore) scanHistoryAlarm(ctx context.Context, qb squirrel.SelectBuilder) (*model.Alarm, error) {
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build history query: %w", err)
	}
	// alarms_history 在时序库（TsPool）；其 JOIN 的 devices→device_dim 影子表同库。
	row := s.tsPool.QueryRow(ctx, sql, args...)
	a, err := scanHistoryAlarmRow(row)
	if gerr.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("history alarm not found: %w", commonerrors.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("scan history alarm: %w", err)
	}
	return a, nil
}

func scanHistoryAlarmRow(row scannable) (*model.Alarm, error) {
	var a model.Alarm
	var timeVal time.Time
	var additionalJSON []byte
	if err := row.Scan(
		&timeVal,
		&a.ID, &a.DeviceID, &a.DeviceSN, &a.Carrier, &a.Severity,
		&a.AlarmType, &a.AlarmIdentifier, &a.Description, &a.Status, &a.RaisedAt,
		&a.AcknowledgedAt, &a.ClearedAt, &a.AcknowledgedBy, &a.AckNote,
		&additionalJSON,
		&a.DeviceName, &a.Technology, &a.AlarmSource, &a.EventType,
		&a.AckCount, &a.UpdatedAt,
		&a.ClearedBy, &a.ClearNote,
		&a.ProbableCause, &a.Version,
	); err != nil {
		return nil, err
	}
	if len(additionalJSON) > 0 {
		_ = json.Unmarshal(additionalJSON, &a.AdditionalInfo)
	}
	return &a, nil
}

func (s *PgAlarmStore) BatchAcknowledge(ctx context.Context, ids []uuid.UUID, by string, note string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET status = 'acknowledged', acknowledged_at = NOW(), acknowledged_by = $1, ack_note = $2, updated_at = NOW() WHERE id = ANY($3) AND status != 'cleared'`,
		by, note, ids)
	return err
}

func (s *PgAlarmStore) BatchUnacknowledge(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alarms_active SET status = 'active', acknowledged_at = NULL, acknowledged_by = NULL, ack_note = '', updated_at = NOW() WHERE id = ANY($1) AND status = 'acknowledged'`,
		ids)
	return err
}

func (s *PgAlarmStore) BatchHistoryAcknowledge(ctx context.Context, ids []uuid.UUID, by string, note string) error {
	// alarms_history 在时序库（TsPool）。
	_, err := s.tsPool.Exec(ctx,
		`UPDATE alarms_history SET acknowledged_at = NOW(), acknowledged_by = $1, ack_note = $2, updated_at = NOW() WHERE alarm_id = ANY($3) AND acknowledged_at IS NULL`,
		by, note, ids)
	return err
}

func (s *PgAlarmStore) BatchHistoryUnacknowledge(ctx context.Context, ids []uuid.UUID) error {
	// alarms_history 在时序库（TsPool）。
	_, err := s.tsPool.Exec(ctx,
		`UPDATE alarms_history SET acknowledged_at = NULL, acknowledged_by = NULL, updated_at = NOW() WHERE alarm_id = ANY($1) AND acknowledged_at IS NOT NULL`,
		ids)
	return err
}

func (s *PgAlarmStore) BatchHistoryDelete(ctx context.Context, ids []uuid.UUID) error {
	// alarms_history 在时序库（TsPool）。
	_, err := s.tsPool.Exec(ctx,
		`DELETE FROM alarms_history WHERE alarm_id = ANY($1)`,
		ids)
	return err
}

func (s *PgAlarmStore) BatchClear(ctx context.Context, ids []uuid.UUID, by string, note string) error {
	for _, id := range ids {
		alarm, err := s.GetActiveByID(ctx, id)
		if err != nil {
			continue
		}
		now := time.Now()
		alarm.Status = model.AlarmCleared
		alarm.ClearedAt = &now
		alarm.ClearedBy = &by
		alarm.ClearNote = &note
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
