package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var _ ChannelConfigRepository = (*PgChannelConfigRepository)(nil)

type PgChannelConfigRepository struct{ db storage.DB }

func NewPgChannelConfigRepository(pool *pgxpool.Pool) *PgChannelConfigRepository {
	return &PgChannelConfigRepository{db: storage.NewPoolDB(pool)}
}

var channelConfigColumns = []string{
	"id", "channel", "name", "enabled", "parameters", "secret_ref IS NOT NULL AND secret_ref <> ''",
	"revision", "created_by", "created_at", "updated_at",
}

func (r *PgChannelConfigRepository) List(ctx context.Context) ([]ChannelConfig, error) {
	query, args, err := storage.Psql.Select(channelConfigColumns...).From("notification_channel_configs").OrderBy("channel", "name", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification channel list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()
	items := make([]ChannelConfig, 0)
	for rows.Next() {
		item, scanErr := scanChannelConfig(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan notification channel: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification channels: %w", err)
	}
	return items, nil
}

func (r *PgChannelConfigRepository) Get(ctx context.Context, id uuid.UUID) (*ChannelConfig, error) {
	return getChannelConfig(ctx, r.db, id, false)
}

func (r *PgChannelConfigRepository) Update(ctx context.Context, id uuid.UUID, expectedRevision int64, input ChannelConfigUpdate, actor string) (*ChannelConfig, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification channel update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := getChannelConfig(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if current.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	now := time.Now().UTC()
	if input.Enabled && current.Channel == TemplateChannelEmail {
		query, args, buildErr := storage.Psql.Update("notification_channel_configs").
			Set("enabled", false).Set("revision", sq.Expr("revision + 1")).Set("updated_at", now).
			Where(sq.Eq{"channel": TemplateChannelEmail, "enabled": true}).Where(sq.NotEq{"id": id}).ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build previous email channel disable: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			return nil, fmt.Errorf("disable previous email channel: %w", execErr)
		}
	}
	builder := storage.Psql.Update("notification_channel_configs").
		Set("name", input.Name).Set("enabled", input.Enabled).Set("parameters", input.Parameters).
		Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision})
	if input.SecretRef != nil {
		if *input.SecretRef == "" {
			builder = builder.Set("secret_ref", nil)
		} else {
			builder = builder.Set("secret_ref", *input.SecretRef)
		}
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification channel update: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, query, args...); execErr != nil {
		return nil, mapRuleWriteError("update notification channel", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	_ = actor
	item, err := getChannelConfig(ctx, tx, id, false)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit notification channel update: %w", err)
	}
	return item, nil
}

func (r *PgChannelConfigRepository) GetHealth(ctx context.Context, id uuid.UUID) (*ChannelHealth, error) {
	query, args, err := storage.Psql.Select(
		"channel_config_id", "circuit_state", "consecutive_successes", "consecutive_failures",
		"last_success_at", "last_failure_at", "last_verified_at", "last_error_category",
		"last_error_summary", "circuit_opened_at", "next_probe_at", "updated_at",
	).From("notification_channel_health").Where(sq.Eq{"channel_config_id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification channel health lookup: %w", err)
	}
	var health ChannelHealth
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&health.ChannelConfigID, &health.CircuitState, &health.ConsecutiveSuccesses, &health.ConsecutiveFailures,
		&health.LastSuccessAt, &health.LastFailureAt, &health.LastVerifiedAt, &health.LastErrorCategory,
		&health.LastErrorSummary, &health.CircuitOpenedAt, &health.NextProbeAt, &health.UpdatedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get notification channel health: %w", err)
	}
	return &health, nil
}

func (r *PgChannelConfigRepository) RecordVerification(
	ctx context.Context,
	id uuid.UUID,
	now time.Time,
	errorCategory *string,
	errorSummary *string,
) error {
	state, failures := "closed", 0
	var successAt, failureAt, openedAt, nextProbe any = now, nil, nil, nil
	if errorCategory != nil {
		state, failures = "open", 1
		successAt, failureAt, openedAt, nextProbe = nil, now, now, now.Add(5*time.Minute)
	}
	query, args, err := storage.Psql.Insert("notification_channel_health").Columns(
		"channel_config_id", "circuit_state", "consecutive_successes", "consecutive_failures",
		"last_success_at", "last_failure_at", "last_verified_at", "last_error_category",
		"last_error_summary", "circuit_opened_at", "next_probe_at", "updated_at",
	).Values(
		id, state, boolInt(errorCategory == nil), failures, successAt, failureAt, now,
		errorCategory, errorSummary, openedAt, nextProbe, now,
	).Suffix(`ON CONFLICT (channel_config_id) DO UPDATE SET
        circuit_state=EXCLUDED.circuit_state,
        consecutive_successes=CASE WHEN EXCLUDED.circuit_state='closed' THEN notification_channel_health.consecutive_successes+1 ELSE 0 END,
        consecutive_failures=EXCLUDED.consecutive_failures,
        last_success_at=COALESCE(EXCLUDED.last_success_at, notification_channel_health.last_success_at),
        last_failure_at=COALESCE(EXCLUDED.last_failure_at, notification_channel_health.last_failure_at),
        last_verified_at=EXCLUDED.last_verified_at,
        last_error_category=EXCLUDED.last_error_category,
        last_error_summary=EXCLUDED.last_error_summary,
        circuit_opened_at=EXCLUDED.circuit_opened_at,
        next_probe_at=EXCLUDED.next_probe_at,
        updated_at=EXCLUDED.updated_at`).ToSql()
	if err != nil {
		return fmt.Errorf("build notification channel verification health: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record notification channel verification health: %w", err)
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func getChannelConfig(ctx context.Context, queryer ruleQueryer, id uuid.UUID, lock bool) (*ChannelConfig, error) {
	builder := storage.Psql.Select(channelConfigColumns...).From("notification_channel_configs").Where(sq.Eq{"id": id})
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification channel lookup: %w", err)
	}
	item, err := scanChannelConfig(queryer.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get notification channel: %w", err)
	}
	return &item, nil
}

func scanChannelConfig(row interface{ Scan(...any) error }) (ChannelConfig, error) {
	var item ChannelConfig
	err := row.Scan(
		&item.ID, &item.Channel, &item.Name, &item.Enabled, &item.Parameters, &item.SecretConfigured,
		&item.Revision, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}
