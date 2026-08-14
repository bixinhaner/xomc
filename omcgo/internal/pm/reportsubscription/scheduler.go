package reportsubscription

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
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/storage"
)

const JobType = "pm_query_report_email"

type JobPayload struct {
	RunID string `json:"run_id"`
}

type dueSubscription struct {
	Subscription
	TemplateName string
	Payload      json.RawMessage
}

type Scheduler struct {
	pool     *pgxpool.Pool
	location func() *time.Location
	logger   *zap.Logger
}

func NewScheduler(pool *pgxpool.Pool, location func() *time.Location, logger *zap.Logger) *Scheduler {
	if location == nil {
		location = func() *time.Location { return time.UTC }
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Scheduler{pool: pool, location: location, logger: logger.Named("pm.report.scheduler")}
}

func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := s.EnqueueDue(ctx, time.Now(), 50); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Warn("enqueue due report subscriptions", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// EnqueueDue 在一个事务中锁定到期订阅、创建自然窗口 run/job，并推进 next_run_at。
func (s *Scheduler) EnqueueDue(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin report scheduler transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	location := s.location()
	if location == nil {
		location = time.UTC
	}
	query, args, err := storage.Psql.Select(
		"s.id", "s.query_template_id", "s.enabled", "s.period", "s.send_times::text[]",
		"s.recipients", "s.timezone_name", "s.next_run_at", "s.last_run_at", "s.last_status", "s.last_error",
		"s.created_by", "s.created_at", "s.updated_at", "t.name", "t.payload",
	).From("pm_query_report_subscriptions s").
		Join("pm_query_templates t ON t.id = s.query_template_id").
		Where(sq.Eq{"s.enabled": true}).
		Where(sq.Or{
			sq.LtOrEq{"s.next_run_at": now},
			sq.NotEq{"s.timezone_name": location.String()},
		}).
		OrderBy("s.next_run_at ASC").Limit(uint64(limit)).
		Suffix("FOR UPDATE OF s SKIP LOCKED").ToSql()
	if err != nil {
		return 0, fmt.Errorf("build due report subscriptions SQL: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("query due report subscriptions: %w", err)
	}
	items := make([]dueSubscription, 0)
	for rows.Next() {
		var item dueSubscription
		var period string
		if err := rows.Scan(
			&item.ID, &item.QueryTemplateID, &item.Enabled, &period, &item.SendTimes,
			&item.Recipients, &item.TimezoneName, &item.NextRunAt, &item.LastRunAt, &item.LastStatus, &item.LastError,
			&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.TemplateName, &item.Payload,
		); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan due report subscription: %w", err)
		}
		item.Period = Period(period)
		for i := range item.SendTimes {
			item.SendTimes[i] = trimPGTime(item.SendTimes[i])
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate due report subscriptions: %w", err)
	}
	rows.Close()

	created := 0
	for _, item := range items {
		if item.TimezoneName != location.String() {
			nextRun, err := NextRunAt(now, item.SendTimes, location)
			if err != nil {
				return created, err
			}
			updateSubscription, updateArgs, err := storage.Psql.Update("pm_query_report_subscriptions").
				Set("timezone_name", location.String()).
				Set("next_run_at", nextRun).
				Where(sq.Eq{"id": item.ID}).
				ToSql()
			if err != nil {
				return created, fmt.Errorf("build realign report subscription timezone SQL: %w", err)
			}
			if _, err := tx.Exec(ctx, updateSubscription, updateArgs...); err != nil {
				return created, fmt.Errorf("realign report subscription timezone: %w", err)
			}
			continue
		}
		start, end, nextRun, err := scheduledWindow(item, now, location)
		if err != nil {
			return created, err
		}
		runID := uuid.New()
		insertRun, runArgs, err := storage.Psql.Insert("pm_query_report_runs").Columns(
			"id", "subscription_id", "query_template_id", "query_template_name", "query_payload",
			"period", "recipients", "window_start", "window_end", "status",
		).Values(
			runID, item.ID, item.QueryTemplateID, item.TemplateName, item.Payload,
			string(item.Period), item.Recipients, start, end, RunStatusPending,
		).Suffix("ON CONFLICT (subscription_id, window_start, window_end) DO NOTHING").ToSql()
		if err != nil {
			return created, fmt.Errorf("build insert report run SQL: %w", err)
		}
		cmd, err := tx.Exec(ctx, insertRun, runArgs...)
		if err != nil {
			return created, fmt.Errorf("insert report run: %w", err)
		}
		runCreated := cmd.RowsAffected() == 1
		if runCreated {
			payload, err := json.Marshal(JobPayload{RunID: runID.String()})
			if err != nil {
				return created, fmt.Errorf("marshal report job payload: %w", err)
			}
			jobID := uuid.New()
			insertJob, jobArgs, err := storage.Psql.Insert("async_jobs").Columns(
				"id", "job_type", "status", "scheduled_at", "payload", "max_attempts",
			).Values(jobID, JobType, "pending", now, payload, 3).ToSql()
			if err != nil {
				return created, fmt.Errorf("build insert report job SQL: %w", err)
			}
			if _, err := tx.Exec(ctx, insertJob, jobArgs...); err != nil {
				return created, fmt.Errorf("insert report job: %w", err)
			}
			updateRun, updateArgs, _ := storage.Psql.Update("pm_query_report_runs").
				Set("job_id", jobID).Where(sq.Eq{"id": runID}).ToSql()
			if _, err := tx.Exec(ctx, updateRun, updateArgs...); err != nil {
				return created, fmt.Errorf("link report run job: %w", err)
			}
			created++
		}
		updateBuilder := storage.Psql.Update("pm_query_report_subscriptions").Set("next_run_at", nextRun)
		if runCreated {
			updateBuilder = updateBuilder.Set("last_status", RunStatusPending).Set("last_error", nil)
		}
		updateSubscription, updateArgs, _ := updateBuilder.Where(sq.Eq{"id": item.ID}).ToSql()
		if _, err := tx.Exec(ctx, updateSubscription, updateArgs...); err != nil {
			return created, fmt.Errorf("advance report subscription: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit report scheduler transaction: %w", err)
	}
	return created, nil
}

func scheduledWindow(item dueSubscription, now time.Time, location *time.Location) (time.Time, time.Time, time.Time, error) {
	if item.NextRunAt == nil {
		return time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("%w: due subscription has no next_run_at", ErrInvalid)
	}
	// Collapse every missed slot into the latest closed natural window. Advancing
	// directly from now prevents the 30-second scanner from replaying one stale
	// slot per tick after a long outage.
	start, end, err := NaturalWindow(item.Period, now, location)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}
	nextRun, err := NextRunAt(now, item.SendTimes, location)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}
	return start, end, nextRun, nil
}

func trimPGTime(value string) string {
	if len(value) >= 5 {
		return value[:5]
	}
	return value
}
