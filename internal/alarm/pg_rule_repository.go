package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

// psql is defined in pg_store.go — reused here.

var ruleColumns = []string{
	"id", "name", "description", "alarm_code", "severity",
	"condition_type", "condition_config", "action_type", "action_config",
	"carrier", "technology", "enabled", "created_at", "updated_at",
}

var _ AlarmRuleRepository = (*PgAlarmRuleRepository)(nil)

// PgAlarmRuleRepository implements AlarmRuleRepository using PostgreSQL.
type PgAlarmRuleRepository struct {
	pool *pgxpool.Pool
}

// NewPgAlarmRuleRepository creates a new PostgreSQL-backed alarm rule repository.
func NewPgAlarmRuleRepository(pool *pgxpool.Pool) *PgAlarmRuleRepository {
	return &PgAlarmRuleRepository{pool: pool}
}

func (r *PgAlarmRuleRepository) Create(ctx context.Context, rule *AlarmRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	condCfg := ensureJSON(rule.ConditionConfig)
	actCfg := ensureJSON(rule.ActionConfig)

	query, args, err := psql.Insert("alarm_rules").
		Columns("id", "name", "description", "alarm_code", "severity",
			"condition_type", "condition_config", "action_type", "action_config",
			"carrier", "technology", "enabled", "created_at", "updated_at").
		Values(rule.ID, rule.Name, rule.Description, rule.AlarmCode, rule.Severity,
			rule.ConditionType, condCfg, rule.ActionType, actCfg,
			rule.Carrier, rule.Technology, rule.Enabled, rule.CreatedAt, rule.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert alarm_rules SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert alarm_rules: %w", err)
	}
	return nil
}

func (r *PgAlarmRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*AlarmRule, error) {
	query, args, err := psql.Select(ruleColumns...).
		From("alarm_rules").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get alarm_rules SQL: %w", err)
	}

	rule, err := scanAlarmRule(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get alarm_rules: %w", err)
	}
	return rule, nil
}

func (r *PgAlarmRuleRepository) Update(ctx context.Context, rule *AlarmRule) error {
	rule.UpdatedAt = time.Now()

	condCfg := ensureJSON(rule.ConditionConfig)
	actCfg := ensureJSON(rule.ActionConfig)

	query, args, err := psql.Update("alarm_rules").
		Set("name", rule.Name).
		Set("description", rule.Description).
		Set("alarm_code", rule.AlarmCode).
		Set("severity", rule.Severity).
		Set("condition_type", rule.ConditionType).
		Set("condition_config", condCfg).
		Set("action_type", rule.ActionType).
		Set("action_config", actCfg).
		Set("carrier", rule.Carrier).
		Set("technology", rule.Technology).
		Set("enabled", rule.Enabled).
		Set("updated_at", rule.UpdatedAt).
		Where(squirrel.Eq{"id": rule.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update alarm_rules SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update alarm_rules: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgAlarmRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("alarm_rules").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete alarm_rules SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete alarm_rules: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgAlarmRuleRepository) List(ctx context.Context, filter AlarmRuleFilter) (*model.ListResponse[AlarmRule], error) {
	builder := psql.Select(ruleColumns...).From("alarm_rules")
	countBuilder := psql.Select("COUNT(*)").From("alarm_rules")

	builder = applyRuleFilters(builder, filter)
	countBuilder = applyRuleFilters(countBuilder, filter)

	// Count total
	countSQL, countArgs, _ := countBuilder.ToSql()
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count alarm_rules: %w", err)
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
		return nil, fmt.Errorf("build list alarm_rules SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alarm_rules: %w", err)
	}
	defer rows.Close()

	var items []AlarmRule
	for rows.Next() {
		rule, err := scanAlarmRuleRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *rule)
	}

	if items == nil {
		items = []AlarmRule{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func applyRuleFilters(qb squirrel.SelectBuilder, f AlarmRuleFilter) squirrel.SelectBuilder {
	if f.Carrier != nil && *f.Carrier != "" {
		qb = qb.Where(squirrel.Eq{"carrier": *f.Carrier})
	}
	if f.Technology != nil && *f.Technology != "" {
		qb = qb.Where(squirrel.Eq{"technology": *f.Technology})
	}
	if f.Enabled != nil {
		qb = qb.Where(squirrel.Eq{"enabled": *f.Enabled})
	}
	if f.AlarmCode != nil && *f.AlarmCode != "" {
		qb = qb.Where(squirrel.Eq{"alarm_code": *f.AlarmCode})
	}
	if f.ConditionType != nil && *f.ConditionType != "" {
		qb = qb.Where(squirrel.Eq{"condition_type": *f.ConditionType})
	}
	return qb
}

func scanAlarmRule(row pgx.Row) (*AlarmRule, error) {
	var rule AlarmRule
	var condCfg, actCfg []byte
	err := row.Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.AlarmCode, &rule.Severity,
		&rule.ConditionType, &condCfg, &rule.ActionType, &actCfg,
		&rule.Carrier, &rule.Technology, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rule.ConditionConfig = json.RawMessage(condCfg)
	rule.ActionConfig = json.RawMessage(actCfg)
	return &rule, nil
}

func scanAlarmRuleRow(rows pgx.Rows) (*AlarmRule, error) {
	var rule AlarmRule
	var condCfg, actCfg []byte
	err := rows.Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.AlarmCode, &rule.Severity,
		&rule.ConditionType, &condCfg, &rule.ActionType, &actCfg,
		&rule.Carrier, &rule.Technology, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan alarm_rules row: %w", err)
	}
	rule.ConditionConfig = json.RawMessage(condCfg)
	rule.ActionConfig = json.RawMessage(actCfg)
	return &rule, nil
}

// ensureJSON returns the raw message as-is or "{}" if nil/empty.
func ensureJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return []byte(raw)
}
