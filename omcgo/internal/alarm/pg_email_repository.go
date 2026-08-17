package alarm

import (
	"context"
	"encoding/json"
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

var alarmEmailSubscriptionColumns = []string{
	"id", "name", "description", "enabled", "interval_minutes", "tolerance_minutes",
	"recipients", "include_default_recipients", "alarm_identifiers", "severities",
	"alarm_sources", "event_types", "device_ids", "device_group_ids",
	"created_by", "updated_by", "created_at", "updated_at",
}

type PgAlarmEmailRepository struct {
	pool *pgxpool.Pool
}

var _ AlarmEmailRunRepository = (*PgAlarmEmailRepository)(nil)
var _ AlarmEmailSchedulerRepository = (*PgAlarmEmailRepository)(nil)

func NewPgAlarmEmailRepository(pool *pgxpool.Pool) *PgAlarmEmailRepository {
	return &PgAlarmEmailRepository{pool: pool}
}

func (r *PgAlarmEmailRepository) GetGlobalSetting(ctx context.Context) (*AlarmEmailGlobalSetting, error) {
	query, args, err := storage.Psql.
		Select("enabled", "default_recipients", "updated_by", "updated_at").
		From("alarm_email_global_settings").
		Where(sq.Eq{"id": 1}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build alarm email global setting query: %w", err)
	}
	var setting AlarmEmailGlobalSetting
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&setting.Enabled,
		&setting.DefaultRecipients,
		&setting.UpdatedBy,
		&setting.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get alarm email global setting: %w", err)
	}
	return &setting, nil
}

func (r *PgAlarmEmailRepository) UpdateGlobalSetting(ctx context.Context, setting *AlarmEmailGlobalSetting) error {
	recipients, err := NormalizeEmailRecipients(setting.DefaultRecipients)
	if err != nil {
		return err
	}
	query, args, err := storage.Psql.Update("alarm_email_global_settings").
		Set("enabled", setting.Enabled).
		Set("default_recipients", recipients).
		Set("updated_by", setting.UpdatedBy).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": 1}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build alarm email global setting update: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update alarm email global setting: %w", err)
	}
	return nil
}

func (r *PgAlarmEmailRepository) GetSubscription(ctx context.Context, id uuid.UUID) (*AlarmEmailSubscription, error) {
	query, args, err := storage.Psql.Select(alarmEmailSubscriptionColumns...).
		From("alarm_email_subscriptions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build alarm email subscription query: %w", err)
	}
	subscription, err := scanAlarmEmailSubscription(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get alarm email subscription: %w", err)
	}
	return subscription, nil
}

func (r *PgAlarmEmailRepository) ListEnabledSubscriptions(ctx context.Context, intervalMinutes int) ([]AlarmEmailSubscription, error) {
	query, args, err := storage.Psql.Select(alarmEmailSubscriptionColumns...).
		From("alarm_email_subscriptions").
		Where(sq.Eq{"enabled": true, "interval_minutes": intervalMinutes, "deleted_at": nil}).
		OrderBy("created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build enabled alarm email subscriptions query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list enabled alarm email subscriptions: %w", err)
	}
	defer rows.Close()
	result := make([]AlarmEmailSubscription, 0)
	for rows.Next() {
		subscription, scanErr := scanAlarmEmailSubscription(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan alarm email subscription: %w", scanErr)
		}
		result = append(result, *subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alarm email subscriptions: %w", err)
	}
	return result, nil
}

func (r *PgAlarmEmailRepository) LatestPeriodicWindowEnd(ctx context.Context, subscriptionID uuid.UUID) (*time.Time, error) {
	query, args, err := storage.Psql.Select("MAX(window_end)").
		From("alarm_email_runs").
		Where(sq.Eq{"subscription_id": subscriptionID}).
		Where(sq.Expr("window_end - window_start >= INTERVAL '1 minute'")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest periodic alarm email window query: %w", err)
	}
	var latest *time.Time
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&latest); err != nil {
		return nil, fmt.Errorf("get latest periodic alarm email window: %w", err)
	}
	return latest, nil
}

func (r *PgAlarmEmailRepository) ListSubscriptions(ctx context.Context) ([]AlarmEmailSubscription, error) {
	query, args, err := storage.Psql.Select(alarmEmailSubscriptionColumns...).
		From("alarm_email_subscriptions").
		Where(sq.Eq{"deleted_at": nil}).
		OrderBy("created_at DESC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build alarm email subscriptions query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alarm email subscriptions: %w", err)
	}
	defer rows.Close()
	result := make([]AlarmEmailSubscription, 0)
	for rows.Next() {
		subscription, scanErr := scanAlarmEmailSubscription(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan alarm email subscription: %w", scanErr)
		}
		result = append(result, *subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alarm email subscriptions: %w", err)
	}
	return result, nil
}

func (r *PgAlarmEmailRepository) CreateSubscription(ctx context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string) error {
	if err := subscription.NormalizeAndValidate(defaultRecipients); err != nil {
		return err
	}
	if subscription.ID == uuid.Nil {
		subscription.ID = uuid.New()
	}
	now := time.Now()
	if subscription.CreatedAt.IsZero() {
		subscription.CreatedAt = now
	}
	subscription.UpdatedAt = now
	query, args, err := storage.Psql.Insert("alarm_email_subscriptions").
		Columns(alarmEmailSubscriptionColumns...).
		Values(
			subscription.ID, subscription.Name, subscription.Description, subscription.Enabled,
			subscription.IntervalMinutes, subscription.ToleranceMinutes, subscription.Recipients,
			subscription.IncludeDefaultRecipients, subscription.AlarmIdentifiers, subscription.Severities,
			subscription.AlarmSources, subscription.EventTypes, subscription.DeviceIDs,
			subscription.DeviceGroupIDs, subscription.CreatedBy, subscription.UpdatedBy,
			subscription.CreatedAt, subscription.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build alarm email subscription insert: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert alarm email subscription: %w", err)
	}
	return nil
}

func (r *PgAlarmEmailRepository) UpdateSubscription(ctx context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string) error {
	if subscription == nil || subscription.ID == uuid.Nil {
		return commonerrors.ErrInvalidInput
	}
	if err := subscription.NormalizeAndValidate(defaultRecipients); err != nil {
		return err
	}
	query, args, err := storage.Psql.Update("alarm_email_subscriptions").
		Set("name", subscription.Name).
		Set("description", subscription.Description).
		Set("enabled", subscription.Enabled).
		Set("interval_minutes", subscription.IntervalMinutes).
		Set("tolerance_minutes", subscription.ToleranceMinutes).
		Set("recipients", subscription.Recipients).
		Set("include_default_recipients", subscription.IncludeDefaultRecipients).
		Set("alarm_identifiers", subscription.AlarmIdentifiers).
		Set("severities", subscription.Severities).
		Set("alarm_sources", subscription.AlarmSources).
		Set("event_types", subscription.EventTypes).
		Set("device_ids", subscription.DeviceIDs).
		Set("device_group_ids", subscription.DeviceGroupIDs).
		Set("updated_by", subscription.UpdatedBy).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": subscription.ID, "deleted_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build alarm email subscription update: %w", err)
	}
	command, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update alarm email subscription: %w", err)
	}
	if command.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgAlarmEmailRepository) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Update("alarm_email_subscriptions").
		Set("enabled", false).
		Set("deleted_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id, "deleted_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build alarm email subscription soft delete: %w", err)
	}
	command, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("soft delete alarm email subscription: %w", err)
	}
	if command.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// EnqueueRun atomically creates an async job and its subscription-scoped run.
// The run unique key prevents duplicate sends when multiple schedulers race.
func (r *PgAlarmEmailRepository) EnqueueRun(
	ctx context.Context,
	subscription *AlarmEmailSubscription,
	defaultRecipients []string,
	window AlarmEmailWindow,
	scheduledAt time.Time,
) (*AlarmEmailRun, bool, error) {
	if subscription == nil || subscription.ID == uuid.Nil || !window.Start.Before(window.End) {
		return nil, false, ErrAlarmEmailInvalidWindow
	}
	recipients, err := ResolveAlarmEmailRecipients(
		subscription.Recipients,
		defaultRecipients,
		subscription.IncludeDefaultRecipients,
	)
	if err != nil {
		return nil, false, fmt.Errorf("resolve alarm email run recipients: %w", err)
	}
	subscriptionSnapshot, err := json.Marshal(subscription)
	if err != nil {
		return nil, false, fmt.Errorf("encode alarm email subscription snapshot: %w", err)
	}
	runID := uuid.New()
	jobID := uuid.New()
	payload, err := json.Marshal(alarmEmailJobPayload{RunID: runID})
	if err != nil {
		return nil, false, fmt.Errorf("encode alarm email job payload: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin alarm email enqueue: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	jobSQL, jobArgs, err := storage.Psql.Insert("async_jobs").
		Columns("id", "job_type", "status", "scheduled_at", "payload", "max_attempts").
		Values(jobID, AlarmEmailJobType, "pending", scheduledAt, payload, 3).
		ToSql()
	if err != nil {
		return nil, false, fmt.Errorf("build alarm email async job insert: %w", err)
	}
	if _, err := tx.Exec(ctx, jobSQL, jobArgs...); err != nil {
		return nil, false, fmt.Errorf("insert alarm email async job: %w", err)
	}

	runSQL, runArgs, err := storage.Psql.Insert("alarm_email_runs").
		Columns(
			"id", "subscription_id", "async_job_id", "window_start", "window_end", "status",
			"subscription_snapshot", "recipients_snapshot",
		).
		Values(
			runID, subscription.ID, jobID, window.Start, window.End, AlarmEmailRunPending,
			subscriptionSnapshot, recipients,
		).
		Suffix("ON CONFLICT (subscription_id, window_start, window_end) DO NOTHING RETURNING id").
		ToSql()
	if err != nil {
		return nil, false, fmt.Errorf("build alarm email run insert: %w", err)
	}
	var insertedID uuid.UUID
	if err := tx.QueryRow(ctx, runSQL, runArgs...).Scan(&insertedID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, fmt.Errorf("insert alarm email run: %w", err)
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return nil, false, fmt.Errorf("rollback duplicate alarm email run: %w", rollbackErr)
		}
		existing, getErr := r.getRunByWindow(ctx, subscription.ID, window)
		return existing, false, getErr
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit alarm email enqueue: %w", err)
	}
	return &AlarmEmailRun{
		ID: insertedID, SubscriptionID: subscription.ID, AsyncJobID: jobID,
		Window: window, Status: AlarmEmailRunPending,
		SubscriptionSnapshot: *subscription, RecipientsSnapshot: recipients,
	}, true, nil
}

func (r *PgAlarmEmailRepository) GetRun(ctx context.Context, runID uuid.UUID) (*AlarmEmailRun, error) {
	query, args, err := alarmEmailRunSelect().Where(sq.Eq{"id": runID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build alarm email run query: %w", err)
	}
	return scanAlarmEmailRunWithContext(r.pool.QueryRow(ctx, query, args...), "get alarm email run")
}

func (r *PgAlarmEmailRepository) getRunByWindow(ctx context.Context, subscriptionID uuid.UUID, window AlarmEmailWindow) (*AlarmEmailRun, error) {
	query, args, err := alarmEmailRunSelect().Where(sq.Eq{
		"subscription_id": subscriptionID,
		"window_start":    window.Start,
		"window_end":      window.End,
	}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build existing alarm email run query: %w", err)
	}
	return scanAlarmEmailRunWithContext(r.pool.QueryRow(ctx, query, args...), "get existing alarm email run")
}

func alarmEmailRunSelect() sq.SelectBuilder {
	return storage.Psql.Select(
		"id", "subscription_id", "async_job_id", "window_start", "window_end", "status",
		"subscription_snapshot", "recipients_snapshot",
	).From("alarm_email_runs")
}

func scanAlarmEmailRunWithContext(row interface{ Scan(dest ...any) error }, operation string) (*AlarmEmailRun, error) {
	var run AlarmEmailRun
	var subscriptionSnapshot []byte
	if err := row.Scan(
		&run.ID, &run.SubscriptionID, &run.AsyncJobID, &run.Window.Start, &run.Window.End, &run.Status,
		&subscriptionSnapshot, &run.RecipientsSnapshot,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	if len(subscriptionSnapshot) > 0 && string(subscriptionSnapshot) != "{}" {
		if err := json.Unmarshal(subscriptionSnapshot, &run.SubscriptionSnapshot); err != nil {
			return nil, fmt.Errorf("%s subscription snapshot: %w", operation, err)
		}
	}
	return &run, nil
}

func (r *PgAlarmEmailRepository) MarkRunProcessing(ctx context.Context, runID uuid.UUID) error {
	return r.updateRun(ctx, runID, map[string]any{"status": AlarmEmailRunProcessing, "last_error": nil})
}

func (r *PgAlarmEmailRepository) MarkRunResult(ctx context.Context, runID uuid.UUID, status, subject, bodySummary, lastError string) error {
	values := map[string]any{"status": status, "subject": nullIfEmpty(subject), "body_summary": nullIfEmpty(bodySummary), "last_error": nullIfEmpty(lastError)}
	return r.updateRun(ctx, runID, values)
}

func (r *PgAlarmEmailRepository) updateRun(ctx context.Context, runID uuid.UUID, values map[string]any) error {
	queryBuilder := storage.Psql.Update("alarm_email_runs").Set("updated_at", sq.Expr("NOW()"))
	for column, value := range values {
		queryBuilder = queryBuilder.Set(column, value)
	}
	query, args, err := queryBuilder.Where(sq.Eq{"id": runID}).ToSql()
	if err != nil {
		return fmt.Errorf("build alarm email run update: %w", err)
	}
	command, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update alarm email run: %w", err)
	}
	if command.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgAlarmEmailRepository) EnsureDeliveries(ctx context.Context, runID uuid.UUID, recipients []string) error {
	for _, recipient := range recipients {
		query, args, err := storage.Psql.Insert("alarm_email_deliveries").
			Columns("run_id", "recipient", "status").
			Values(runID, recipient, AlarmEmailDeliveryPending).
			Suffix("ON CONFLICT (run_id, lower(recipient)) DO NOTHING").
			ToSql()
		if err != nil {
			return fmt.Errorf("build alarm email delivery insert: %w", err)
		}
		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert alarm email delivery: %w", err)
		}
	}
	return nil
}

func (r *PgAlarmEmailRepository) ListUnsentDeliveries(ctx context.Context, runID uuid.UUID) ([]AlarmEmailDelivery, error) {
	query, args, err := storage.Psql.Select("id", "run_id", "recipient", "status", "attempt").
		From("alarm_email_deliveries").
		Where(sq.Eq{"run_id": runID, "status": []string{
			AlarmEmailDeliveryPending,
			AlarmEmailDeliveryProcessing,
			AlarmEmailDeliveryFailed,
		}}).
		OrderBy("created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build unsent alarm email deliveries query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list unsent alarm email deliveries: %w", err)
	}
	defer rows.Close()
	result := make([]AlarmEmailDelivery, 0)
	for rows.Next() {
		var delivery AlarmEmailDelivery
		if err := rows.Scan(&delivery.ID, &delivery.RunID, &delivery.Recipient, &delivery.Status, &delivery.Attempt); err != nil {
			return nil, fmt.Errorf("scan alarm email delivery: %w", err)
		}
		result = append(result, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alarm email deliveries: %w", err)
	}
	return result, nil
}

func (r *PgAlarmEmailRepository) MarkDeliveryProcessing(ctx context.Context, deliveryID uuid.UUID) error {
	return r.updateDelivery(ctx, deliveryID, map[string]any{
		"status":     AlarmEmailDeliveryProcessing,
		"attempt":    sq.Expr("attempt + 1"),
		"last_error": nil,
	})
}

func (r *PgAlarmEmailRepository) MarkDeliverySent(ctx context.Context, deliveryID uuid.UUID, sentAt time.Time) error {
	return r.updateDelivery(ctx, deliveryID, map[string]any{
		"status":     AlarmEmailDeliverySent,
		"sent_at":    sentAt,
		"last_error": nil,
	})
}

func (r *PgAlarmEmailRepository) MarkDeliveryFailed(ctx context.Context, deliveryID uuid.UUID, sendError string) error {
	return r.updateDelivery(ctx, deliveryID, map[string]any{
		"status":     AlarmEmailDeliveryFailed,
		"last_error": sendError,
	})
}

func (r *PgAlarmEmailRepository) updateDelivery(ctx context.Context, deliveryID uuid.UUID, values map[string]any) error {
	queryBuilder := storage.Psql.Update("alarm_email_deliveries").Set("updated_at", sq.Expr("NOW()"))
	for column, value := range values {
		queryBuilder = queryBuilder.Set(column, value)
	}
	query, args, err := queryBuilder.Where(sq.Eq{"id": deliveryID}).ToSql()
	if err != nil {
		return fmt.Errorf("build alarm email delivery update: %w", err)
	}
	command, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update alarm email delivery: %w", err)
	}
	if command.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func scanAlarmEmailSubscription(row interface{ Scan(dest ...any) error }) (*AlarmEmailSubscription, error) {
	var subscription AlarmEmailSubscription
	err := row.Scan(
		&subscription.ID, &subscription.Name, &subscription.Description, &subscription.Enabled,
		&subscription.IntervalMinutes, &subscription.ToleranceMinutes, &subscription.Recipients,
		&subscription.IncludeDefaultRecipients, &subscription.AlarmIdentifiers, &subscription.Severities,
		&subscription.AlarmSources, &subscription.EventTypes, &subscription.DeviceIDs,
		&subscription.DeviceGroupIDs, &subscription.CreatedBy, &subscription.UpdatedBy,
		&subscription.CreatedAt, &subscription.UpdatedAt,
	)
	return &subscription, err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
