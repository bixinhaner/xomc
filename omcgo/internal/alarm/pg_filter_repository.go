package alarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"

	"github.com/Masterminds/squirrel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type PgAlarmFilterRuleRepository struct {
	db *pgxpool.Pool
}

var alarmFilterRuleOrderBy = []string{
	"priority ASC",
	"created_at ASC",
	"id ASC",
}

func applyAlarmFilterRuleOrdering(query squirrel.SelectBuilder) squirrel.SelectBuilder {
	return query.OrderBy(alarmFilterRuleOrderBy...)
}

func buildAlarmFilterRuleDimensionExpr(filterType string) squirrel.Sqlizer {
	switch filterType {
	case FilterTypeAlarmIdentifier:
		return squirrel.Expr("cardinality(alarm_identifiers) > 0")
	case FilterTypeAlarmSource:
		return squirrel.Expr("cardinality(alarm_sources) > 0")
	case FilterTypeDeviceGroup:
		return squirrel.Expr("cardinality(device_group_ids) > 0")
	case FilterTypeDevice:
		return squirrel.Expr("cardinality(device_ids) > 0")
	default:
		return nil
	}
}

func applyAlarmFilterRuleFilters(query squirrel.SelectBuilder, filter AlarmFilterRuleFilter) squirrel.SelectBuilder {
	if len(filter.FilterTypes) > 0 {
		dimensionFilters := squirrel.Or{}
		seen := make(map[string]struct{}, len(filter.FilterTypes))
		for _, filterType := range filter.FilterTypes {
			trimmed := strings.TrimSpace(filterType)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			if expr := buildAlarmFilterRuleDimensionExpr(trimmed); expr != nil {
				dimensionFilters = append(dimensionFilters, expr)
			}
		}
		if len(dimensionFilters) > 0 {
			query = query.Where(dimensionFilters)
		}
	}
	if filter.Action != nil && *filter.Action != "" {
		query = query.Where(squirrel.Eq{"action": *filter.Action})
	}
	if filter.Enabled != nil {
		query = query.Where(squirrel.Eq{"enabled": *filter.Enabled})
	}
	if filter.Keyword != nil {
		keyword := strings.TrimSpace(*filter.Keyword)
		if keyword != "" {
			like := "%" + keyword + "%"
			query = query.Where(squirrel.Or{
				squirrel.ILike{"name": like},
				squirrel.ILike{"created_by": like},
				squirrel.ILike{"updated_by": like},
				squirrel.ILike{"filter_type": like},
				squirrel.ILike{"action": like},
				squirrel.Expr("array_to_string(alarm_identifiers, ',') ILIKE ?", like),
				squirrel.Expr("array_to_string(alarm_sources, ',') ILIKE ?", like),
				squirrel.Expr("array_to_string(device_ids, ',') ILIKE ?", like),
				squirrel.Expr("array_to_string(device_group_ids, ',') ILIKE ?", like),
				squirrel.Expr("EXISTS (SELECT 1 FROM devices d WHERE d.id = ANY(alarm_filters.device_ids) AND d.serial_number ILIKE ?)", like),
			})
		}
	}

	return query
}

func NewPgAlarmFilterRuleRepository(db *pgxpool.Pool) *PgAlarmFilterRuleRepository {
	return &PgAlarmFilterRuleRepository{db: db}
}

func (r *PgAlarmFilterRuleRepository) Create(ctx context.Context, rule *AlarmFilterRule) error {
	if rule.Action == FilterActionLegacyNotificationBarrier {
		return ErrAlarmFilterBarrierManaged
	}
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}

	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	query := storage.Psql.
		Insert("alarm_filters").
		Columns(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "created_by", "created_at", "updated_by", "updated_at",
		).
		Values(
			rule.ID, rule.Name, rule.FilterType, rule.AlarmSources, rule.AlarmIdentifiers,
			rule.DeviceIDs, rule.DeviceGroupIDs, rule.Action, rule.AcknowledgeDesc, rule.WebhookURL, rule.WebhookSecret, rule.EmailRecipients,
			rule.Priority, rule.Enabled, rule.CreatedBy, rule.CreatedAt, rule.UpdatedBy, rule.UpdatedAt,
		).
		Suffix("RETURNING id")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	return r.db.QueryRow(ctx, sql, args...).Scan(&rule.ID)
}

func (r *PgAlarmFilterRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*AlarmFilterRule, error) {
	query := storage.Psql.
		Select(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "COALESCE(acknowledge_desc, '') AS acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "COALESCE(created_by, '') AS created_by", "created_at", "COALESCE(updated_by, '') AS updated_by", "updated_at",
		).
		From("alarm_filters").
		Where(squirrel.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query failed: %w", err)
	}

	rule := &AlarmFilterRule{}
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&rule.ID, &rule.Name, &rule.FilterType, &rule.AlarmSources, &rule.AlarmIdentifiers,
		&rule.DeviceIDs, &rule.DeviceGroupIDs, &rule.Action, &rule.AcknowledgeDesc, &rule.WebhookURL, &rule.WebhookSecret, &rule.EmailRecipients,
		&rule.Priority, &rule.Enabled, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedBy, &rule.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return rule, nil
}

func (r *PgAlarmFilterRuleRepository) Update(ctx context.Context, rule *AlarmFilterRule) error {
	if rule.Action == FilterActionLegacyNotificationBarrier {
		return ErrAlarmFilterBarrierManaged
	}
	now := time.Now()
	rule.UpdatedAt = now

	query := storage.Psql.
		Update("alarm_filters").
		Set("name", rule.Name).
		Set("filter_type", rule.FilterType).
		Set("alarm_sources", rule.AlarmSources).
		Set("alarm_identifiers", rule.AlarmIdentifiers).
		Set("device_ids", rule.DeviceIDs).
		Set("device_group_ids", rule.DeviceGroupIDs).
		Set("action", rule.Action).
		Set("acknowledge_desc", rule.AcknowledgeDesc).
		Set("webhook_url", rule.WebhookURL).
		Set("webhook_secret", rule.WebhookSecret).
		Set("email_recipients", rule.EmailRecipients).
		Set("priority", rule.Priority).
		Set("enabled", rule.Enabled).
		Set("updated_by", rule.UpdatedBy).
		Set("updated_at", rule.UpdatedAt).
		Where(squirrel.Eq{"id": rule.ID}).
		Where(squirrel.NotEq{"action": FilterActionLegacyNotificationBarrier})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	cmd, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return r.alarmFilterMutationMiss(ctx, rule.ID)
	}

	return nil
}

func (r *PgAlarmFilterRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := storage.Psql.
		Delete("alarm_filters").
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.NotEq{"action": FilterActionLegacyNotificationBarrier})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	cmd, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return r.alarmFilterMutationMiss(ctx, id)
	}

	return nil
}

func (r *PgAlarmFilterRuleRepository) List(ctx context.Context, filter AlarmFilterRuleFilter) (*model.ListResponse[AlarmFilterRule], error) {
	query := applyAlarmFilterRuleFilters(applyAlarmFilterRuleOrdering(storage.Psql.
		Select(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "COALESCE(acknowledge_desc, '') AS acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "COALESCE(created_by, '') AS created_by", "created_at", "COALESCE(updated_by, '') AS updated_by", "updated_at",
		).
		From("alarm_filters")), filter)

	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Limit(uint64(filter.PageSize)).Offset(uint64(offset))
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query failed: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var rules []AlarmFilterRule
	for rows.Next() {
		rule := AlarmFilterRule{}
		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.FilterType, &rule.AlarmSources, &rule.AlarmIdentifiers,
			&rule.DeviceIDs, &rule.DeviceGroupIDs, &rule.Action, &rule.AcknowledgeDesc, &rule.WebhookURL, &rule.WebhookSecret, &rule.EmailRecipients,
			&rule.Priority, &rule.Enabled, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedBy, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	var total int64
	if filter.Page > 0 && filter.PageSize > 0 {
		countQuery := applyAlarmFilterRuleFilters(storage.Psql.Select("COUNT(*)").From("alarm_filters"), filter)

		countSql, countArgs, err := countQuery.ToSql()
		if err != nil {
			return nil, fmt.Errorf("build count query failed: %w", err)
		}

		err = r.db.QueryRow(ctx, countSql, countArgs...).Scan(&total)
		if err != nil {
			return nil, fmt.Errorf("count query failed: %w", err)
		}
	} else {
		total = int64(len(rules))
	}

	return &model.ListResponse[AlarmFilterRule]{
		Items: rules,
		Total: total,
	}, nil
}

func (r *PgAlarmFilterRuleRepository) Toggle(ctx context.Context, id uuid.UUID) error {
	query := storage.Psql.
		Update("alarm_filters").
		Set("enabled", squirrel.Expr("NOT enabled")).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.NotEq{"action": FilterActionLegacyNotificationBarrier})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	cmd, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("toggle failed: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return r.alarmFilterMutationMiss(ctx, id)
	}

	return nil
}

func (r *PgAlarmFilterRuleRepository) alarmFilterMutationMiss(ctx context.Context, id uuid.UUID) error {
	rule, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if rule.Action == FilterActionLegacyNotificationBarrier {
		return ErrAlarmFilterBarrierManaged
	}
	return commonerrors.ErrNotFound
}

func (r *PgAlarmFilterRuleRepository) ListEnabled(ctx context.Context) ([]AlarmFilterRule, error) {
	query := applyAlarmFilterRuleOrdering(storage.Psql.
		Select(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "COALESCE(acknowledge_desc, '') AS acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "COALESCE(created_by, '') AS created_by", "created_at", "COALESCE(updated_by, '') AS updated_by", "updated_at",
		).
		From("alarm_filters").
		Where(squirrel.Eq{"enabled": true}))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query failed: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var rules []AlarmFilterRule
	for rows.Next() {
		rule := AlarmFilterRule{}
		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.FilterType, &rule.AlarmSources, &rule.AlarmIdentifiers,
			&rule.DeviceIDs, &rule.DeviceGroupIDs, &rule.Action, &rule.AcknowledgeDesc, &rule.WebhookURL, &rule.WebhookSecret, &rule.EmailRecipients,
			&rule.Priority, &rule.Enabled, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedBy, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return rules, nil
}

// ReplaceNotifyEmailWithBarrier atomically replaces one unchanged legacy email
// rule with the compatibility barrier. It is intentionally not part of the
// ordinary AlarmFilterRuleRepository API and is only consumed by notification
// migration code.
func (r *PgAlarmFilterRuleRepository) ReplaceNotifyEmailWithBarrier(
	ctx context.Context,
	id uuid.UUID,
	expectedUpdatedAt time.Time,
	notificationRuleID uuid.UUID,
	enabledVersionID uuid.UUID,
	actor string,
) error {
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("alarm_filters").
		Set("action", FilterActionLegacyNotificationBarrier).
		Set("updated_by", actor).
		Set("updated_at", now).
		Where(squirrel.Eq{
			"id":         id,
			"action":     FilterActionNotifyEmail,
			"enabled":    true,
			"updated_at": expectedUpdatedAt,
		}).
		Where(squirrel.Expr(
			"EXISTS (SELECT 1 FROM notification_rules WHERE id = ? AND archived = false AND current_enabled_version_id = ?)",
			notificationRuleID, enabledVersionID,
		)).ToSql()
	if err != nil {
		return fmt.Errorf("build legacy email barrier update: %w", err)
	}
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("replace legacy email rule with barrier: %w", err)
	}
	if result.RowsAffected() == 1 {
		return nil
	}
	if _, err := r.GetByID(ctx, id); err != nil {
		return fmt.Errorf("check legacy email rule after barrier conflict: %w", err)
	}
	return ErrAlarmFilterMigrationPrecondition
}

var _ AlarmFilterRuleRepository = (*PgAlarmFilterRuleRepository)(nil)
