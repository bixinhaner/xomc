package alarm

import (
	"context"
	"fmt"
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

func NewPgAlarmFilterRuleRepository(db *pgxpool.Pool) *PgAlarmFilterRuleRepository {
	return &PgAlarmFilterRuleRepository{db: db}
}

func (r *PgAlarmFilterRuleRepository) Create(ctx context.Context, rule *AlarmFilterRule) error {
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
			"device_ids", "device_group_ids", "action", "acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "created_by", "created_at", "updated_by", "updated_at",
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
		Where(squirrel.Eq{"id": rule.ID})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	cmd, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmFilterRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := storage.Psql.
		Delete("alarm_filters").
		Where(squirrel.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	cmd, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmFilterRuleRepository) List(ctx context.Context, filter AlarmFilterRuleFilter) (*model.ListResponse[AlarmFilterRule], error) {
	query := storage.Psql.
		Select(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "created_by", "created_at", "updated_by", "updated_at",
		).
		From("alarm_filters").
		OrderBy("priority ASC")

	if filter.FilterType != nil && *filter.FilterType != "" {
		query = query.Where(squirrel.Eq{"filter_type": *filter.FilterType})
	}
	if filter.Action != nil && *filter.Action != "" {
		query = query.Where(squirrel.Eq{"action": *filter.Action})
	}
	if filter.Enabled != nil {
		query = query.Where(squirrel.Eq{"enabled": *filter.Enabled})
	}

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
		countQuery := storage.Psql.Select("COUNT(*)").From("alarm_filters")
		if filter.FilterType != nil && *filter.FilterType != "" {
			countQuery = countQuery.Where(squirrel.Eq{"filter_type": *filter.FilterType})
		}
		if filter.Action != nil && *filter.Action != "" {
			countQuery = countQuery.Where(squirrel.Eq{"action": *filter.Action})
		}
		if filter.Enabled != nil {
			countQuery = countQuery.Where(squirrel.Eq{"enabled": *filter.Enabled})
		}

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
		Where(squirrel.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query failed: %w", err)
	}

	cmd, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("toggle failed: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmFilterRuleRepository) ListEnabled(ctx context.Context) ([]AlarmFilterRule, error) {
	query := storage.Psql.
		Select(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "created_by", "created_at", "updated_by", "updated_at",
		).
		From("alarm_filters").
		Where(squirrel.Eq{"enabled": true}).
		OrderBy("priority ASC")

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

var _ AlarmFilterRuleRepository = (*PgAlarmFilterRuleRepository)(nil)