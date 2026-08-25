package alarm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgAlarmEmailReader assembles one email window from the active-alarm main
// database and the cleared-alarm TimescaleDB database.
type PgAlarmEmailReader struct {
	pool   *pgxpool.Pool
	tsPool *pgxpool.Pool
}

func NewPgAlarmEmailReader(pool, tsPool *pgxpool.Pool) *PgAlarmEmailReader {
	return &PgAlarmEmailReader{pool: pool, tsPool: tsPool}
}

var _ AlarmEmailAlarmReader = (*PgAlarmEmailReader)(nil)

type alarmEmailRow struct {
	id   uuid.UUID
	item AlarmEmailItem
}

func (r *PgAlarmEmailReader) ListForEmailWindow(
	ctx context.Context,
	subscription *AlarmEmailSubscription,
	window AlarmEmailWindow,
) ([]AlarmEmailItem, error) {
	if r.pool == nil || r.tsPool == nil {
		return nil, fmt.Errorf("alarm email reader database pools are required")
	}
	if subscription == nil || !window.Start.Before(window.End) {
		return nil, ErrAlarmEmailInvalidWindow
	}

	deviceIDs, deviceScoped, err := r.resolveDeviceScope(ctx, subscription)
	if err != nil {
		return nil, err
	}
	active, err := r.queryRows(ctx, r.pool, buildActiveAlarmEmailQuery(subscription, window, deviceIDs, deviceScoped))
	if err != nil {
		return nil, fmt.Errorf("query active alarms for email: %w", err)
	}
	history, err := r.queryRows(ctx, r.tsPool, buildHistoryAlarmEmailQuery(subscription, window, deviceIDs, deviceScoped))
	if err != nil {
		return nil, fmt.Errorf("query cleared alarms for email: %w", err)
	}

	byID := make(map[uuid.UUID]AlarmEmailItem, len(active)+len(history))
	identifiers := make(map[string]struct{}, len(active)+len(history))
	for _, row := range append(active, history...) {
		byID[row.id] = row.item
		identifiers[row.item.AlarmIdentifier] = struct{}{}
	}
	advice, err := r.loadAdvice(ctx, identifiers)
	if err != nil {
		return nil, err
	}

	items := make([]AlarmEmailItem, 0, len(byID))
	for _, item := range byID {
		item.Advice = advice[item.AlarmIdentifier]
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].RaisedAt.Equal(items[j].RaisedAt) {
			return items[i].AlarmIdentifier < items[j].AlarmIdentifier
		}
		return items[i].RaisedAt.Before(items[j].RaisedAt)
	})
	return items, nil
}

func (r *PgAlarmEmailReader) resolveDeviceScope(
	ctx context.Context,
	subscription *AlarmEmailSubscription,
) ([]uuid.UUID, bool, error) {
	deviceScoped := len(subscription.DeviceIDs) > 0 || len(subscription.DeviceGroupIDs) > 0
	seen := make(map[uuid.UUID]struct{}, len(subscription.DeviceIDs))
	for _, id := range subscription.DeviceIDs {
		if id != uuid.Nil {
			seen[id] = struct{}{}
		}
	}
	if len(subscription.DeviceGroupIDs) > 0 {
		query, args, err := buildAlarmEmailDeviceScopeQuery(subscription.DeviceGroupIDs)
		if err != nil {
			return nil, false, fmt.Errorf("build alarm email device-group query: %w", err)
		}
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			return nil, false, fmt.Errorf("query alarm email device groups: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return nil, false, fmt.Errorf("scan alarm email device group: %w", err)
			}
			seen[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return nil, false, fmt.Errorf("iterate alarm email device groups: %w", err)
		}
	}
	ids := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids, deviceScoped, nil
}

func buildAlarmEmailDeviceScopeQuery(selectedGroupIDs []uuid.UUID) (string, []any, error) {
	// device_groups 受 chk_dg_level_parent 约束为两级树。订阅页允许选择任一级，
	// 因此范围同时包含所选组自身及其直接子组，和页面树的选择语义保持一致。
	groupIDs := sq.Select("id").
		From("device_groups").
		Where(sq.Or{
			sq.Eq{"id": selectedGroupIDs},
			sq.Eq{"parent_id": selectedGroupIDs},
		})
	memberDeviceIDs := sq.Select("device_id").
		From("device_group_members").
		Where(sq.Expr("group_id IN (?)", groupIDs))

	clauses := sq.Or{sq.Expr("d.id IN (?)", memberDeviceIDs)}
	if includesDefaultDeviceGroup(selectedGroupIDs) {
		// 默认二级组是历史无 membership 设备的逻辑归属；选择默认一级组时
		// 也会覆盖该子组，所以两种选择都必须兼容这批历史数据。
		clauses = append(clauses, sq.Expr(
			"NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)",
		))
	}

	return storage.Psql.Select("DISTINCT d.id").
		From("devices d").
		Where(clauses).
		ToSql()
}

func includesDefaultDeviceGroup(groupIDs []uuid.UUID) bool {
	for _, groupID := range groupIDs {
		switch groupID.String() {
		case global.DefaultLevel1GroupID, global.DefaultLevel2GroupID:
			return true
		}
	}
	return false
}

type alarmEmailQuery struct {
	sql  string
	args []any
	err  error
}

func buildActiveAlarmEmailQuery(subscription *AlarmEmailSubscription, window AlarmEmailWindow, deviceIDs []uuid.UUID, deviceScoped bool) alarmEmailQuery {
	return buildAlarmEmailQuery("alarms_active", "id", "NULL::timestamptz", []string{"raised_at"}, subscription, window, deviceIDs, deviceScoped)
}

func buildHistoryAlarmEmailQuery(subscription *AlarmEmailSubscription, window AlarmEmailWindow, deviceIDs []uuid.UUID, deviceScoped bool) alarmEmailQuery {
	return buildAlarmEmailQuery("alarms_history", "alarm_id", "cleared_at", []string{"raised_at", "cleared_at"}, subscription, window, deviceIDs, deviceScoped)
}

func buildAlarmEmailQuery(
	table, idColumn, clearedAtExpression string,
	windowColumns []string,
	subscription *AlarmEmailSubscription,
	window AlarmEmailWindow,
	deviceIDs []uuid.UUID,
	deviceScoped bool,
) alarmEmailQuery {
	qb := storage.Psql.Select(
		idColumn,
		"alarm_identifier",
		"severity",
		"COALESCE(probable_cause, '')",
		"COALESCE(NULLIF(device_name, ''), device_sn)",
		"raised_at",
		clearedAtExpression,
		"COALESCE(description, '')",
	).From(table)
	windowPredicates := make(sq.Or, 0, len(windowColumns))
	for _, column := range windowColumns {
		windowPredicates = append(windowPredicates, sq.And{
			sq.GtOrEq{column: window.Start},
			sq.Lt{column: window.End},
		})
	}
	qb = qb.Where(windowPredicates)
	if len(subscription.AlarmIdentifiers) > 0 {
		qb = qb.Where(sq.Eq{"alarm_identifier": subscription.AlarmIdentifiers})
	}
	if len(subscription.Severities) > 0 {
		severities := make([]model.AlarmSeverity, 0, len(subscription.Severities))
		for _, severity := range subscription.Severities {
			severities = append(severities, model.AlarmSeverity(severity))
		}
		qb = qb.Where(sq.Eq{"severity": expandSeverityAliases(severities)})
	}
	if len(subscription.AlarmSources) > 0 {
		qb = qb.Where(sq.Eq{"alarm_source": subscription.AlarmSources})
	}
	if eventTypes := normalizedEventTypesExpr("event_type", subscription.EventTypes); eventTypes != nil {
		qb = qb.Where(eventTypes)
	}
	if deviceScoped {
		if len(deviceIDs) == 0 {
			qb = qb.Where(sq.Expr("FALSE"))
		} else {
			qb = qb.Where(sq.Eq{"device_id": deviceIDs})
		}
	}
	query, args, err := qb.OrderBy("raised_at ASC", idColumn+" ASC").ToSql()
	return alarmEmailQuery{sql: query, args: args, err: err}
}

func normalizedEventTypesExpr(column string, values []string) sq.Sqlizer {
	conditions := make(sq.Or, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		conditions = append(conditions, normalizedEventTypeExpr(column, value))
	}
	if len(conditions) == 0 {
		return nil
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	return conditions
}

func (r *PgAlarmEmailReader) queryRows(ctx context.Context, pool *pgxpool.Pool, query alarmEmailQuery) ([]alarmEmailRow, error) {
	if query.err != nil {
		return nil, fmt.Errorf("build alarm email source query: %w", query.err)
	}
	rows, err := pool.Query(ctx, query.sql, query.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]alarmEmailRow, 0)
	for rows.Next() {
		var row alarmEmailRow
		if err := rows.Scan(
			&row.id,
			&row.item.AlarmIdentifier,
			&row.item.Severity,
			&row.item.ProbableCause,
			&row.item.DeviceLabel,
			&row.item.RaisedAt,
			&row.item.ClearedAt,
			&row.item.Description,
		); err != nil {
			return nil, fmt.Errorf("scan alarm email source row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alarm email source rows: %w", err)
	}
	return result, nil
}

func (r *PgAlarmEmailReader) loadAdvice(ctx context.Context, identifiers map[string]struct{}) (map[string]string, error) {
	result := make(map[string]string, len(identifiers))
	if len(identifiers) == 0 {
		return result, nil
	}
	keys := make([]string, 0, len(identifiers))
	for identifier := range identifiers {
		keys = append(keys, identifier)
	}
	sort.Strings(keys)
	query, args, err := storage.Psql.Select(
		"identifier",
		"COALESCE(NULLIF(cn_suggestion, ''), NULLIF(en_suggestion, ''), '')",
	).From("alarm_definitions").Where(sq.Eq{"identifier": keys}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build alarm email advice query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query alarm email advice: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var identifier, advice string
		if err := rows.Scan(&identifier, &advice); err != nil {
			return nil, fmt.Errorf("scan alarm email advice: %w", err)
		}
		result[strings.TrimSpace(identifier)] = advice
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alarm email advice: %w", err)
	}
	return result, nil
}
