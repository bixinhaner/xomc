package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	GPVHandoffStatusPending    = "pending"
	GPVHandoffStatusProcessing = "processing"
	GPVHandoffStatusDelivered  = "delivered"
	GPVHandoffStatusFailed     = "failed"
	GPVHandoffStatusDead       = "dead"
)

type GPVHandoffEnqueue struct {
	Durable  string
	DeviceSN string
	Event    Event
}

type GPVHandoffClaimOptions struct {
	Durable     string
	Now         time.Time
	Lease       time.Duration
	Limit       int
	MaxAttempts int
}

type GPVHandoffEntry struct {
	ID           uuid.UUID
	Durable      string
	EventID      string
	DeviceSN     string
	Subject      string
	Payload      json.RawMessage
	Metadata     map[string]string
	Timestamp    time.Time
	AttemptCount int
	LeaseToken   uuid.UUID
	LeaseUntil   time.Time
	CreatedAt    time.Time
}

func (e GPVHandoffEntry) Event() Event {
	return Event{
		ID:        e.EventID,
		Subject:   e.Subject,
		Payload:   e.Payload,
		Metadata:  e.Metadata,
		Timestamp: e.Timestamp,
	}
}

type GPVHandoffRepository interface {
	Enqueue(context.Context, GPVHandoffEnqueue) error
	ClaimDue(context.Context, GPVHandoffClaimOptions) ([]GPVHandoffEntry, error)
	MarkDelivered(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error)
	MarkDead(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error)
}

type PGGPVHandoffRepository struct {
	db storage.DB
}

func NewPGGPVHandoffRepository(db storage.DB) *PGGPVHandoffRepository {
	return &PGGPVHandoffRepository{db: db}
}

var _ GPVHandoffRepository = (*PGGPVHandoffRepository)(nil)

func (r *PGGPVHandoffRepository) Enqueue(ctx context.Context, req GPVHandoffEnqueue) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("enqueue GPV handoff: database is required")
	}
	if err := validateGPVHandoffEnqueue(req); err != nil {
		return err
	}
	metadata, err := json.Marshal(req.Event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal GPV handoff metadata: %w", err)
	}
	query, args, err := storage.Psql.Insert("command_gpv_response_handoffs").
		Columns("durable", "event_id", "device_sn", "subject", "payload", "metadata", "event_timestamp").
		Values(req.Durable, req.Event.ID, req.DeviceSN, req.Event.Subject, req.Event.Payload, metadata, req.Event.Timestamp).
		Suffix("ON CONFLICT (durable, event_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build GPV handoff enqueue: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("enqueue GPV handoff: %w", err)
	}
	return nil
}

func validateGPVHandoffEnqueue(req GPVHandoffEnqueue) error {
	switch {
	case strings.TrimSpace(req.Durable) == "":
		return fmt.Errorf("enqueue GPV handoff: durable is required")
	case strings.TrimSpace(req.DeviceSN) == "":
		return fmt.Errorf("enqueue GPV handoff: device SN is required")
	case strings.TrimSpace(req.Event.ID) == "":
		return fmt.Errorf("enqueue GPV handoff: event ID is required")
	case strings.TrimSpace(req.Event.Subject) == "":
		return fmt.Errorf("enqueue GPV handoff: subject is required")
	case len(req.Event.Payload) == 0:
		return fmt.Errorf("enqueue GPV handoff: payload is required")
	default:
		return nil
	}
}

func (r *PGGPVHandoffRepository) ClaimDue(
	ctx context.Context,
	options GPVHandoffClaimOptions,
) ([]GPVHandoffEntry, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("claim GPV handoff: database is required")
	}
	if err := validateGPVHandoffClaimOptions(options); err != nil {
		return nil, err
	}
	leaseToken := uuid.New()
	leaseUntil := options.Now.Add(options.Lease)
	rows, err := r.db.Query(ctx, claimGPVHandoffSQL, options.Durable, options.Now, options.Limit, options.MaxAttempts, leaseToken, leaseUntil)
	if err != nil {
		return nil, fmt.Errorf("claim GPV handoff: %w", err)
	}
	defer rows.Close()

	entries := make([]GPVHandoffEntry, 0, options.Limit)
	for rows.Next() {
		var entry GPVHandoffEntry
		var metadata []byte
		if err := rows.Scan(
			&entry.ID,
			&entry.Durable,
			&entry.EventID,
			&entry.DeviceSN,
			&entry.Subject,
			&entry.Payload,
			&metadata,
			&entry.Timestamp,
			&entry.AttemptCount,
			&entry.LeaseToken,
			&entry.LeaseUntil,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan GPV handoff: %w", err)
		}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
				return nil, fmt.Errorf("decode GPV handoff metadata: %w", err)
			}
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate GPV handoff claims: %w", err)
	}
	return entries, nil
}

func validateGPVHandoffClaimOptions(options GPVHandoffClaimOptions) error {
	switch {
	case strings.TrimSpace(options.Durable) == "":
		return fmt.Errorf("claim GPV handoff: durable is required")
	case options.Now.IsZero():
		return fmt.Errorf("claim GPV handoff: current time is required")
	case options.Lease <= 0:
		return fmt.Errorf("claim GPV handoff: lease must be positive")
	case options.Limit <= 0:
		return fmt.Errorf("claim GPV handoff: limit must be positive")
	case options.MaxAttempts <= 0:
		return fmt.Errorf("claim GPV handoff: max attempts must be positive")
	default:
		return nil
	}
}

const claimGPVHandoffSQL = `
WITH candidate AS (
    SELECT h.id
      FROM command_gpv_response_handoffs h
     WHERE h.durable = $1
       AND (
            (h.status IN ('pending', 'failed') AND h.next_attempt_at <= $2 AND h.attempt_count < $4)
            OR
            (h.status = 'processing' AND h.lease_until <= $2)
       )
       AND NOT EXISTS (
            SELECT 1
              FROM command_gpv_response_handoffs earlier
             WHERE earlier.durable = h.durable
               AND earlier.device_sn = h.device_sn
               AND earlier.status IN ('pending', 'failed', 'processing')
               AND (earlier.created_at, earlier.id) < (h.created_at, h.id)
       )
     ORDER BY h.created_at, h.id
     LIMIT $3
     FOR UPDATE SKIP LOCKED
)
UPDATE command_gpv_response_handoffs h
   SET status = 'processing',
       attempt_count = h.attempt_count + 1,
       lease_token = $5,
       lease_until = $6,
       last_error = NULL,
       updated_at = $2
  FROM candidate
 WHERE h.id = candidate.id
RETURNING h.id, h.durable, h.event_id, h.device_sn, h.subject, h.payload,
          h.metadata, h.event_timestamp, h.attempt_count, h.lease_token,
          h.lease_until, h.created_at`

func (r *PGGPVHandoffRepository) MarkDelivered(
	ctx context.Context,
	id uuid.UUID,
	leaseToken uuid.UUID,
	deliveredAt time.Time,
) (bool, error) {
	query, args, err := storage.Psql.Update("command_gpv_response_handoffs").
		Set("status", GPVHandoffStatusDelivered).
		Set("lease_token", nil).
		Set("lease_until", nil).
		Set("delivered_at", deliveredAt).
		Set("updated_at", deliveredAt).
		Where("id = ?", id).
		Where("lease_token = ?", leaseToken).
		Where("status = ?", GPVHandoffStatusProcessing).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build mark GPV handoff delivered: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("mark GPV handoff delivered: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PGGPVHandoffRepository) MarkFailed(
	ctx context.Context,
	id uuid.UUID,
	leaseToken uuid.UUID,
	errorMessage string,
	nextAttemptAt time.Time,
) (bool, error) {
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("command_gpv_response_handoffs").
		Set("status", GPVHandoffStatusFailed).
		Set("lease_token", nil).
		Set("lease_until", nil).
		Set("last_error", truncateGPVHandoffError(errorMessage)).
		Set("next_attempt_at", nextAttemptAt).
		Set("updated_at", now).
		Where("id = ?", id).
		Where("lease_token = ?", leaseToken).
		Where("status = ?", GPVHandoffStatusProcessing).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build mark GPV handoff failed: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("mark GPV handoff failed: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PGGPVHandoffRepository) MarkDead(
	ctx context.Context,
	id uuid.UUID,
	leaseToken uuid.UUID,
	errorMessage string,
	deadAt time.Time,
) (bool, error) {
	query, args, err := storage.Psql.Update("command_gpv_response_handoffs").
		Set("status", GPVHandoffStatusDead).
		Set("lease_token", nil).
		Set("lease_until", nil).
		Set("last_error", truncateGPVHandoffError(errorMessage)).
		Set("delivered_at", deadAt).
		Set("updated_at", deadAt).
		Where("id = ?", id).
		Where("lease_token = ?", leaseToken).
		Where("status = ?", GPVHandoffStatusProcessing).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build mark GPV handoff dead: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("mark GPV handoff dead: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func truncateGPVHandoffError(message string) string {
	const maxRunes = 2048
	runes := []rune(message)
	if len(runes) <= maxRunes {
		return message
	}
	return string(runes[:maxRunes])
}

func GPVHandoffEnqueueHandler(
	repo GPVHandoffRepository,
	durable string,
	keyFunc EventKeyFunc,
) EventHandler {
	return func(ctx context.Context, evt Event) error {
		if repo == nil {
			return fmt.Errorf("GPV handoff enqueue %s: repository is required", durable)
		}
		if keyFunc == nil {
			return fmt.Errorf("GPV handoff enqueue %s: key function is required", durable)
		}
		deviceSN, err := keyFunc(evt)
		if err != nil {
			return err
		}
		if strings.TrimSpace(deviceSN) == "" {
			return fmt.Errorf("GPV handoff enqueue %s: device SN is required", durable)
		}
		return repo.Enqueue(ctx, GPVHandoffEnqueue{
			Durable:  durable,
			DeviceSN: deviceSN,
			Event:    evt,
		})
	}
}

type GPVHandoffWorkerConfig struct {
	Durable     string
	Concurrency int
	BatchSize   int
	Lease       time.Duration
	MaxAttempts int
	PollIdle    time.Duration
	RetryBase   time.Duration
	RetryMax    time.Duration
}

func (c GPVHandoffWorkerConfig) Defaults() GPVHandoffWorkerConfig {
	if c.Concurrency <= 0 {
		c.Concurrency = 1
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1
	}
	if c.Lease <= 0 {
		c.Lease = 2 * time.Minute
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = maxDeliveries
	}
	if c.PollIdle <= 0 {
		c.PollIdle = 100 * time.Millisecond
	}
	if c.RetryBase <= 0 {
		c.RetryBase = 500 * time.Millisecond
	}
	if c.RetryMax <= 0 {
		c.RetryMax = 30 * time.Second
	}
	return c
}

func RunGPVHandoffWorkers(
	ctx context.Context,
	repo GPVHandoffRepository,
	config GPVHandoffWorkerConfig,
	handler EventHandler,
	logger *zap.Logger,
) error {
	config = config.Defaults()
	if repo == nil {
		return fmt.Errorf("run GPV handoff workers: repository is required")
	}
	if strings.TrimSpace(config.Durable) == "" {
		return fmt.Errorf("run GPV handoff workers: durable is required")
	}
	if handler == nil {
		return fmt.Errorf("run GPV handoff workers: handler is required")
	}
	for i := 0; i < config.Concurrency; i++ {
		go runGPVHandoffWorker(ctx, repo, config, handler, logger)
	}
	return nil
}

func runGPVHandoffWorker(
	ctx context.Context,
	repo GPVHandoffRepository,
	config GPVHandoffWorkerConfig,
	handler EventHandler,
	logger *zap.Logger,
) {
	for {
		if ctx.Err() != nil {
			return
		}
		now := time.Now().UTC()
		entries, err := repo.ClaimDue(ctx, GPVHandoffClaimOptions{
			Durable:     config.Durable,
			Now:         now,
			Lease:       config.Lease,
			Limit:       config.BatchSize,
			MaxAttempts: config.MaxAttempts,
		})
		if err != nil {
			if logger != nil {
				logger.Error("claim GPV handoff failed", zap.Error(err))
			}
			sleepOrDone(ctx, config.PollIdle)
			continue
		}
		if len(entries) == 0 {
			sleepOrDone(ctx, config.PollIdle)
			continue
		}
		processGPVHandoffBatch(ctx, repo, config, handler, logger, entries)
	}
}

func processGPVHandoffBatch(
	ctx context.Context,
	repo GPVHandoffRepository,
	config GPVHandoffWorkerConfig,
	handler EventHandler,
	logger *zap.Logger,
	entries []GPVHandoffEntry,
) {
	var wg sync.WaitGroup
	for _, entry := range entries {
		entry := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			processGPVHandoffEntry(ctx, repo, config, handler, logger, entry)
		}()
	}
	wg.Wait()
}

func processGPVHandoffEntry(
	ctx context.Context,
	repo GPVHandoffRepository,
	config GPVHandoffWorkerConfig,
	handler EventHandler,
	logger *zap.Logger,
	entry GPVHandoffEntry,
) {
	err := handler(ctx, entry.Event())
	now := time.Now().UTC()
	if err == nil {
		if ok, markErr := repo.MarkDelivered(ctx, entry.ID, entry.LeaseToken, now); markErr != nil && logger != nil {
			logger.Error("mark GPV handoff delivered failed",
				zap.String("event_id", entry.EventID),
				zap.String("device_sn", entry.DeviceSN),
				zap.Error(markErr))
		} else if !ok && logger != nil {
			logger.Warn("mark GPV handoff delivered lost lease",
				zap.String("event_id", entry.EventID),
				zap.String("device_sn", entry.DeviceSN))
		}
		return
	}

	if entry.AttemptCount >= config.MaxAttempts {
		if ok, markErr := repo.MarkDead(ctx, entry.ID, entry.LeaseToken, err.Error(), now); markErr != nil && logger != nil {
			logger.Error("mark GPV handoff dead failed",
				zap.String("event_id", entry.EventID),
				zap.String("device_sn", entry.DeviceSN),
				zap.Error(markErr))
		} else if !ok && logger != nil {
			logger.Warn("mark GPV handoff dead lost lease",
				zap.String("event_id", entry.EventID),
				zap.String("device_sn", entry.DeviceSN))
		}
		return
	}
	nextAttemptAt := now.Add(gpvHandoffBackoff(entry.AttemptCount, config.RetryBase, config.RetryMax))
	if ok, markErr := repo.MarkFailed(ctx, entry.ID, entry.LeaseToken, err.Error(), nextAttemptAt); markErr != nil && logger != nil {
		logger.Error("mark GPV handoff failed failed",
			zap.String("event_id", entry.EventID),
			zap.String("device_sn", entry.DeviceSN),
			zap.Error(markErr))
	} else if !ok && logger != nil {
		logger.Warn("mark GPV handoff failed lost lease",
			zap.String("event_id", entry.EventID),
			zap.String("device_sn", entry.DeviceSN))
	}
}

func gpvHandoffBackoff(attempt int, base, maxDelay time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := base
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= maxDelay {
			return maxDelay
		}
	}
	return delay
}

func sleepOrDone(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
