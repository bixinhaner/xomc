package notification

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ InboxRepository             = (*PgInboxRepository)(nil)
	_ RuleVersionRepository       = (*PgRuleVersionRepository)(nil)
	_ TemplateVersionRepository   = (*PgTemplateVersionRepository)(nil)
	_ ScheduleRepository          = (*PgScheduleRepository)(nil)
	_ DeliveryRepository          = (*PgDeliveryRepository)(nil)
	_ ScheduledDeliveryRepository = (*PgDeliveryRepository)(nil)
	_ EmailWorkerRepository       = (*PgDeliveryRepository)(nil)
)

type domainFakeDB struct {
	execFn     func(string, ...any) (pgconn.CommandTag, error)
	queryRowFn func(string, ...any) pgx.Row
}

func (f *domainFakeDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return f.execFn(query, args...)
}

func (f *domainFakeDB) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (f *domainFakeDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	return f.queryRowFn(query, args...)
}

func (f *domainFakeDB) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected begin")
}

type domainFakeRow struct{ err error }

func (r domainFakeRow) Scan(...any) error { return r.err }

func TestPgInboxRepository_InsertEvent_IdempotentSQL(t *testing.T) {
	var query string
	db := &domainFakeDB{execFn: func(q string, _ ...any) (pgconn.CommandTag, error) {
		query = q
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}}
	event := &DomainEvent{
		EventID: uuid.New(), EventType: "domain.alarm.lifecycle.raised", OccurrenceID: uuid.New(),
		AlarmVersion: 1, SchemaVersion: 1, OccurredAt: time.Now(), Payload: []byte(`{"schema_version":1}`),
	}
	inserted, err := newPgInboxRepository(db).InsertEvent(context.Background(), event)
	require.NoError(t, err)
	require.True(t, inserted)
	assert.NotEqual(t, uuid.Nil, event.ID)
	normalized := strings.ToUpper(query)
	assert.Contains(t, normalized, "INSERT INTO NOTIFICATION_EVENTS")
	assert.Contains(t, normalized, "ON CONFLICT (EVENT_ID) DO NOTHING")
}

func TestPgInboxRepository_InsertEvent_DuplicateAndError(t *testing.T) {
	event := &DomainEvent{ID: uuid.New(), EventID: uuid.New()}
	duplicateDB := &domainFakeDB{execFn: func(string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("INSERT 0 0"), nil
	}}
	inserted, err := newPgInboxRepository(duplicateDB).InsertEvent(context.Background(), event)
	require.NoError(t, err)
	assert.False(t, inserted)

	wantErr := errors.New("database unavailable")
	errorDB := &domainFakeDB{execFn: func(string, ...any) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, wantErr
	}}
	_, err = newPgInboxRepository(errorDB).InsertEvent(context.Background(), event)
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "insert notification event")
}

func TestPgDomainRepositories_LookupErrorsAreWrapped(t *testing.T) {
	wantErr := errors.New("scan failed")
	db := &domainFakeDB{
		execFn:     func(string, ...any) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil },
		queryRowFn: func(string, ...any) pgx.Row { return domainFakeRow{err: wantErr} },
	}

	_, err := newPgInboxRepository(db).GetOccurrence(context.Background(), uuid.New())
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "get notification occurrence")
	_, err = newPgRuleVersionRepository(db).GetRuleVersion(context.Background(), uuid.New())
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "get notification rule version")
	_, err = newPgTemplateVersionRepository(db).GetTemplateVersion(context.Background(), uuid.New())
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "get notification template version")
}

func TestPgScheduleRepository_InsertSchedule_DedupSQL(t *testing.T) {
	var query string
	db := &domainFakeDB{execFn: func(q string, _ ...any) (pgconn.CommandTag, error) {
		query = q
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}}
	schedule := &DomainSchedule{
		EventID: uuid.New(), OccurrenceID: uuid.New(), RuleVersionID: uuid.New(),
		TemplateVersionID: uuid.New(), ChannelConfigID: uuid.New(), Channel: "email",
		DispatchKind: DispatchKindInitial, RecipientType: RecipientTargetUser,
		AddressCiphertext: []byte("ciphertext"), AddressKeyVersion: 1,
		RecipientFingerprint: []byte("fingerprint"), ScheduleKind: "initial_gate",
		Generation: 1, DueAt: time.Now(), CreatedEventVersion: 1,
	}
	inserted, err := newPgScheduleRepository(db).InsertSchedule(context.Background(), schedule)
	require.NoError(t, err)
	assert.True(t, inserted)
	assert.Contains(t, strings.ToUpper(query), "INSERT INTO NOTIFICATION_SCHEDULES")
	assert.Contains(t, strings.ToUpper(query), "ON CONFLICT (OCCURRENCE_ID, RULE_VERSION_ID, CHANNEL, RECIPIENT_FINGERPRINT, SCHEDULE_KIND, SEQUENCE_NO, GENERATION) DO NOTHING")
}

func TestPgScheduleRepository_ClaimUsesSkipLockedAndExpiredLeaseRecovery(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	query, args, err := buildScheduleClaim(ScheduleClaimRequest{
		WorkerID: "worker-a", Now: now, LeaseDuration: 30 * time.Second, Limit: 50,
	})
	require.NoError(t, err)
	normalized := strings.ToUpper(query)
	assert.Contains(t, normalized, "FOR UPDATE SKIP LOCKED")
	assert.Contains(t, normalized, "STATE = $1")
	assert.Contains(t, normalized, "LEASE_EXPIRES_AT <=")
	assert.Contains(t, normalized, "RETURNING ID, EVENT_ID, OCCURRENCE_ID")
	assert.NotEmpty(t, args)
}

func TestPgDeliveryRepository_EmailClaimUsesLeaseAndExcludesUnfinishedAttempts(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 30, 0, 0, time.UTC)
	query, args, err := buildEmailDeliveryClaim(EmailDeliveryClaimRequest{
		WorkerID: "email-a", Now: now, LeaseDuration: time.Minute, Limit: 20,
	})
	require.NoError(t, err)
	normalized := strings.ToUpper(query)
	assert.Contains(t, normalized, "FOR UPDATE SKIP LOCKED")
	assert.Contains(t, normalized, "NOT EXISTS (SELECT 1 FROM NOTIFICATION_DELIVERY_ATTEMPTS")
	assert.Contains(t, normalized, "FLOW_STATE")
	assert.Contains(t, normalized, "LEASE_EXPIRES_AT")
	assert.Contains(t, normalized, "RETURNING ID, EVENT_ID, OCCURRENCE_ID")
	assert.NotEmpty(t, args)
}

func TestPgScheduleRepository_TransitionRequiresClaimOwner(t *testing.T) {
	var queries []string
	db := &domainFakeDB{execFn: func(query string, _ ...any) (pgconn.CommandTag, error) {
		queries = append(queries, query)
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}}
	repository := newPgScheduleRepository(db)

	err := repository.MarkCompleted(context.Background(), uuid.New(), "stale-worker", time.Now())
	require.ErrorIs(t, err, ErrScheduleLeaseLost)
	err = repository.MarkCancelled(context.Background(), uuid.New(), "stale-worker", time.Now())
	require.ErrorIs(t, err, ErrScheduleLeaseLost)
	err = repository.Release(context.Background(), uuid.New(), "stale-worker", time.Now(), time.Now())
	require.ErrorIs(t, err, ErrScheduleLeaseLost)
	require.Len(t, queries, 3)
	for _, query := range queries {
		assert.Contains(t, strings.ToUpper(query), "LEASE_EXPIRES_AT >")
	}
}

func TestPgDeliveryRepository_UsesCiphertextAndStableDedup(t *testing.T) {
	var queries []string
	db := &domainFakeDB{execFn: func(q string, _ ...any) (pgconn.CommandTag, error) {
		queries = append(queries, q)
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}}
	delivery := &DomainDelivery{
		EventID: uuid.New(), OccurrenceID: uuid.New(), RuleVersionID: uuid.New(),
		TemplateVersionID: uuid.New(), ChannelConfigID: uuid.New(), Channel: "email",
		DispatchKind: "initial", RecipientType: "user", AddressCiphertext: []byte("ciphertext"),
		AddressKeyVersion: 1, RecipientFingerprint: []byte("fingerprint"), OccurrenceVersion: 1,
		ScheduleGeneration: 1,
	}
	inserted, err := newPgDeliveryRepository(db).InsertDelivery(context.Background(), delivery)
	require.NoError(t, err)
	assert.True(t, inserted)

	attempt := &DomainDeliveryAttempt{DeliveryID: delivery.ID, AttemptNo: 1, StartedAt: time.Now(), Result: "accepted"}
	inserted, err = newPgDeliveryRepository(db).InsertAttempt(context.Background(), attempt)
	require.NoError(t, err)
	assert.True(t, inserted)
	require.Len(t, queries, 2)
	deliverySQL := strings.ToUpper(queries[0])
	assert.Contains(t, deliverySQL, "ADDRESS_CIPHERTEXT")
	assert.Contains(t, deliverySQL, "RECIPIENT_FINGERPRINT")
	assert.NotContains(t, deliverySQL, "EMAIL_ADDRESS")
	assert.NotContains(t, deliverySQL, "PHONE_NUMBER")
	assert.Contains(t, deliverySQL, "ON CONFLICT (EVENT_ID, DISPATCH_KIND, SEQUENCE_NO, CHANNEL, RECIPIENT_FINGERPRINT) DO NOTHING")
	assert.Contains(t, strings.ToUpper(queries[1]), "ON CONFLICT (DELIVERY_ID, ATTEMPT_NO) DO NOTHING")
}
