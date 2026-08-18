package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	deviceAccessOutboxMaxAttempts = 20
	deviceAccessOutboxBatchSize   = 100
)

type PgReevaluationQueue struct {
	db  storage.DB
	now func() time.Time
}

type PgGPSProbeQueue struct {
	db  storage.DB
	now func() time.Time
}

func NewPgGPSProbeQueue(pool *pgxpool.Pool) *PgGPSProbeQueue {
	return newPgGPSProbeQueueWithDB(storage.NewPoolDB(pool))
}

func newPgGPSProbeQueueWithDB(db storage.DB) *PgGPSProbeQueue {
	return &PgGPSProbeQueue{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (q *PgGPSProbeQueue) EnsureGPSProbe(ctx context.Context, request GPSProbeRequest) error {
	if err := validateIdentity(request.Carrier, request.SerialNumber); err != nil {
		return err
	}
	if q == nil || q.db == nil {
		return fmt.Errorf("enqueue device access probe: %w", ErrAccessGateDependencyMissing)
	}
	if request.EvidenceVersion <= 0 {
		return errors.New("device access probe evidence version must be positive")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode device access probe request: %w", err)
	}
	now := q.now()
	id := uuid.New()
	eventKey := fmt.Sprintf(
		"device-access-probe:%s:%s:v%d:%t:%t:%t",
		request.Carrier, request.SerialNumber, request.EvidenceVersion,
		request.NeedGPS, request.NeedTAC, request.NeedECGI,
	)
	query, args, err := storage.Psql.Insert("device_access_outbox").
		Columns(
			"aggregate_type", "aggregate_id", "event_type", "event_key", "payload",
			"next_attempt_at", "created_at", "updated_at",
		).
		Values(
			"device_access_probe", id, event.SubjectDeviceAccessProbeRequested,
			eventKey, payload, now, now, now,
		).
		Suffix("ON CONFLICT (event_key) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build device access probe outbox insert: %w", err)
	}
	if _, err := q.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert device access probe outbox: %w", err)
	}
	return nil
}

func NewPgReevaluationQueue(pool *pgxpool.Pool) *PgReevaluationQueue {
	return newPgReevaluationQueueWithDB(storage.NewPoolDB(pool))
}

func newPgReevaluationQueueWithDB(db storage.DB) *PgReevaluationQueue {
	return &PgReevaluationQueue{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (q *PgReevaluationQueue) Enqueue(ctx context.Context, request ReevaluationRequest) error {
	request.Carrier = strings.TrimSpace(request.Carrier)
	request.SerialNumber = strings.TrimSpace(request.SerialNumber)
	request.TriggerType = strings.TrimSpace(request.TriggerType)
	if err := validateIdentity(request.Carrier, request.SerialNumber); err != nil {
		return err
	}
	if request.TriggerType == "" {
		return errors.New("reevaluation trigger type is required")
	}
	if q == nil || q.db == nil {
		return fmt.Errorf("enqueue device access reevaluation: %w", ErrAccessGateDependencyMissing)
	}
	id := uuid.New()
	if strings.TrimSpace(request.TriggerEventID) == "" {
		request.TriggerEventID = id.String()
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode device access reevaluation request: %w", err)
	}
	now := q.now()
	eventKey := fmt.Sprintf("device-access-reevaluation:%s:%s", request.SerialNumber, id)
	query, args, err := storage.Psql.Insert("device_access_outbox").
		Columns(
			"aggregate_type", "aggregate_id", "event_type", "event_key", "payload",
			"next_attempt_at", "created_at", "updated_at",
		).
		Values(
			"device_access_reevaluation", id, event.SubjectDeviceAccessReevaluationRequested,
			eventKey, payload, now, now, now,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build device access reevaluation outbox insert: %w", err)
	}
	if _, err := q.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert device access reevaluation outbox: %w", err)
	}
	return nil
}

type accessOutboxItem struct {
	ID        uuid.UUID
	EventType string
	EventKey  string
	Payload   json.RawMessage
	Attempts  int
}

type OutboxDispatcher struct {
	db          storage.DB
	bus         event.EventBus
	maxAttempts int
	now         func() time.Time
}

func NewOutboxDispatcher(pool *pgxpool.Pool, bus event.EventBus) *OutboxDispatcher {
	return newOutboxDispatcherWithDB(storage.NewPoolDB(pool), bus)
}

func newOutboxDispatcherWithDB(db storage.DB, bus event.EventBus) *OutboxDispatcher {
	return &OutboxDispatcher{
		db: db, bus: bus, maxAttempts: deviceAccessOutboxMaxAttempts,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (d *OutboxDispatcher) DispatchPending(ctx context.Context, limit int) (int, error) {
	if d == nil || d.db == nil || d.bus == nil {
		return 0, fmt.Errorf("dispatch device access outbox: %w", ErrAccessGateDependencyMissing)
	}
	if limit <= 0 {
		limit = deviceAccessOutboxBatchSize
	}
	items, err := d.claim(ctx, limit)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, item := range items {
		evt := event.Event{
			ID: item.EventKey, Subject: item.EventType, Payload: item.Payload, Timestamp: d.now(),
		}
		if err := d.bus.Publish(ctx, item.EventType, evt); err != nil {
			if markErr := d.markFailure(ctx, item, err); markErr != nil {
				return delivered, markErr
			}
			continue
		}
		if err := d.markDelivered(ctx, item.ID); err != nil {
			return delivered, err
		}
		delivered++
	}
	return delivered, nil
}

func (d *OutboxDispatcher) claim(ctx context.Context, limit int) ([]accessOutboxItem, error) {
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin device access outbox claim: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	query, args, err := storage.Psql.
		Select("id", "event_type", "event_key", "payload", "attempts").
		From("device_access_outbox").
		Where("status IN ('pending', 'failed')").
		Where(sq.LtOrEq{"next_attempt_at": d.now()}).
		OrderBy("created_at ASC", "id ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device access outbox claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device access outbox claim: %w", err)
	}
	items := make([]accessOutboxItem, 0, limit)
	for rows.Next() {
		var item accessOutboxItem
		if err := rows.Scan(&item.ID, &item.EventType, &item.EventKey, &item.Payload, &item.Attempts); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan device access outbox item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate device access outbox items: %w", err)
	}
	rows.Close()
	if len(items) > 0 {
		ids := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		update, updateArgs, err := storage.Psql.Update("device_access_outbox").
			Set("status", "delivering").Set("updated_at", d.now()).
			Where(sq.Eq{"id": ids}).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build device access outbox delivering update: %w", err)
		}
		if _, err := tx.Exec(ctx, update, updateArgs...); err != nil {
			return nil, fmt.Errorf("mark device access outbox delivering: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit device access outbox claim: %w", err)
	}
	return items, nil
}

func (d *OutboxDispatcher) markDelivered(ctx context.Context, id uuid.UUID) error {
	now := d.now()
	query, args, err := storage.Psql.Update("device_access_outbox").
		Set("status", "delivered").Set("published_at", now).Set("updated_at", now).
		Set("last_error", nil).
		Where(sq.Eq{"id": id, "status": "delivering"}).ToSql()
	if err != nil {
		return fmt.Errorf("build device access outbox delivered update: %w", err)
	}
	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark device access outbox delivered: %w", err)
	}
	return nil
}

func (d *OutboxDispatcher) markFailure(ctx context.Context, item accessOutboxItem, cause error) error {
	attempts := item.Attempts + 1
	status := "failed"
	if attempts >= d.maxAttempts {
		status = "dead"
	}
	now := d.now()
	query, args, err := storage.Psql.Update("device_access_outbox").
		Set("status", status).Set("attempts", attempts).
		Set("next_attempt_at", now.Add(deviceAccessOutboxBackoff(attempts))).
		Set("last_error", cause.Error()).Set("updated_at", now).
		Where(sq.Eq{"id": item.ID, "status": "delivering"}).ToSql()
	if err != nil {
		return fmt.Errorf("build device access outbox failure update: %w", err)
	}
	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark device access outbox failed: %w", err)
	}
	return nil
}

func (d *OutboxDispatcher) RequeueStaleDeliveries(ctx context.Context, staleBefore time.Time) (int64, error) {
	now := d.now()
	query, args, err := storage.Psql.Update("device_access_outbox").
		Set("status", "failed").Set("next_attempt_at", now).
		Set("last_error", "stale delivery claim recovered").Set("updated_at", now).
		Where(sq.Eq{"status": "delivering"}).Where(sq.Lt{"updated_at": staleBefore}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build stale device access outbox recovery: %w", err)
	}
	tag, err := d.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("recover stale device access outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func deviceAccessOutboxBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 10 {
		attempt = 10
	}
	return time.Second * time.Duration(1<<uint(attempt-1))
}

type ReevaluationConsumer struct {
	bus     event.EventBus
	handler ReevaluationHandler
	sub     event.Subscription
}

type GPSProbeConsumer struct {
	bus     event.EventBus
	planner GPSProbePlanner
	sub     event.Subscription
}

func NewGPSProbeConsumer(bus event.EventBus, planner GPSProbePlanner) *GPSProbeConsumer {
	return &GPSProbeConsumer{bus: bus, planner: planner}
}

func (c *GPSProbeConsumer) Start() error {
	if c == nil || c.bus == nil || c.planner == nil {
		return fmt.Errorf("start device access probe consumer: %w", ErrAccessGateDependencyMissing)
	}
	sub, err := c.bus.QueueSubscribe(
		event.SubjectDeviceAccessProbeRequested,
		"device-access-probe",
		c.Handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe device access probe requests: %w", err)
	}
	c.sub = sub
	return nil
}

func (c *GPSProbeConsumer) Handle(ctx context.Context, evt event.Event) error {
	var request GPSProbeRequest
	if err := evt.DecodePayload(&request); err != nil {
		return fmt.Errorf("decode device access probe request: %w", err)
	}
	if err := c.planner.EnsureGPSProbe(ctx, request); err != nil {
		return fmt.Errorf("handle device access probe request: %w", err)
	}
	return nil
}

func (c *GPSProbeConsumer) Stop() error {
	if c == nil || c.sub == nil {
		return nil
	}
	return c.sub.Unsubscribe()
}

func NewReevaluationConsumer(bus event.EventBus, handler ReevaluationHandler) *ReevaluationConsumer {
	return &ReevaluationConsumer{bus: bus, handler: handler}
}

func (c *ReevaluationConsumer) Start() error {
	if c == nil || c.bus == nil || c.handler == nil {
		return fmt.Errorf("start device access reevaluation consumer: %w", ErrAccessGateDependencyMissing)
	}
	sub, err := c.bus.QueueSubscribe(
		event.SubjectDeviceAccessReevaluationRequested,
		"device-access-reevaluation",
		c.Handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe device access reevaluation requests: %w", err)
	}
	c.sub = sub
	return nil
}

func (c *ReevaluationConsumer) Handle(ctx context.Context, evt event.Event) error {
	var request ReevaluationRequest
	if err := evt.DecodePayload(&request); err != nil {
		return fmt.Errorf("decode device access reevaluation request: %w", err)
	}
	if err := c.handler.Handle(ctx, request); err != nil {
		return fmt.Errorf("handle device access reevaluation request: %w", err)
	}
	return nil
}

func (c *ReevaluationConsumer) Stop() error {
	if c == nil || c.sub == nil {
		return nil
	}
	return c.sub.Unsubscribe()
}

var _ ReevaluationQueue = (*PgReevaluationQueue)(nil)
var _ GPSProbePlanner = (*PgGPSProbeQueue)(nil)
