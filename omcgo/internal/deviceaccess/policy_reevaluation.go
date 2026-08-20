package deviceaccess

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PolicyPublishedExpander turns one durable policy-published event into one
// idempotent reevaluation request per known access-control identity. Expansion
// runs in the worker, outside the HTTP publish transaction.
type PolicyPublishedExpander struct {
	db storage.DB
}

func NewPolicyPublishedExpander(pool *pgxpool.Pool) *PolicyPublishedExpander {
	return newPolicyPublishedExpanderWithDB(storage.NewPoolDB(pool))
}

func newPolicyPublishedExpanderWithDB(db storage.DB) *PolicyPublishedExpander {
	return &PolicyPublishedExpander{db: db}
}

func (e *PolicyPublishedExpander) Handle(ctx context.Context, evt event.Event) error {
	if e == nil || e.db == nil {
		return fmt.Errorf("expand policy published reevaluation: %w", ErrAccessGateDependencyMissing)
	}
	var published PolicyPublishedEvent
	if err := evt.DecodePayload(&published); err != nil {
		return fmt.Errorf("decode policy published event: %w", err)
	}
	published.Carrier = strings.TrimSpace(published.Carrier)
	published.PolicyVersionID = strings.TrimSpace(published.PolicyVersionID)
	published.TriggerEventID = strings.TrimSpace(published.TriggerEventID)
	if published.Carrier == "" || published.PolicyVersionID == "" || published.TriggerEventID == "" {
		return fmt.Errorf("expand policy published reevaluation: %w", ErrInvalidAccessInput)
	}
	now := time.Now().UTC()
	triggerKey := sq.Expr("concat(?, state.serial_number)", "policy:"+published.TriggerEventID+":")
	eventKey := sq.Expr("concat(?, state.serial_number)", "device-access-policy-reevaluation:"+published.PolicyVersionID+":")
	payload := sq.Expr(`jsonb_build_object(
		'carrier', state.carrier,
		'serial_number', state.serial_number,
		'trigger_type', ?,
		'trigger_event_id', ?
	)`, TriggerPolicyPublished, triggerKey)
	selectRows := storage.Psql.Select().
		Column(sq.Expr("?", "device_access_reevaluation")).
		Column("gen_random_uuid()").
		Column(sq.Expr("?", event.SubjectDeviceAccessReevaluationRequested)).
		Column(eventKey).
		Column(payload).
		Column(sq.Expr("?", now)).
		Column(sq.Expr("?", now)).
		Column(sq.Expr("?", now)).
		From("device_access_states state").
		Where(sq.Eq{"state.carrier": published.Carrier}).
		OrderBy("state.serial_number ASC")
	query, args, err := storage.Psql.Insert("device_access_outbox").
		Columns("aggregate_type", "aggregate_id", "event_type", "event_key", "payload", "next_attempt_at", "created_at", "updated_at").
		Select(selectRows).
		Suffix("ON CONFLICT (event_key) DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build policy published reevaluation expansion: %w", err)
	}
	if _, err := e.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("expand policy published reevaluations: %w", err)
	}
	return nil
}

type PolicyPublishedConsumer struct {
	bus      event.EventBus
	expander *PolicyPublishedExpander
	sub      event.Subscription
}

func NewPolicyPublishedConsumer(bus event.EventBus, expander *PolicyPublishedExpander) *PolicyPublishedConsumer {
	return &PolicyPublishedConsumer{bus: bus, expander: expander}
}

func (c *PolicyPublishedConsumer) Start() error {
	if c == nil || c.bus == nil || c.expander == nil {
		return fmt.Errorf("start policy published consumer: %w", ErrAccessGateDependencyMissing)
	}
	sub, err := c.bus.QueueSubscribe(event.SubjectDeviceAccessPolicyPublished, "device-access-policy-published", c.expander.Handle)
	if err != nil {
		return fmt.Errorf("subscribe policy published events: %w", err)
	}
	c.sub = sub
	return nil
}

func (c *PolicyPublishedConsumer) Stop() error {
	if c == nil || c.sub == nil {
		return nil
	}
	return c.sub.Unsubscribe()
}
