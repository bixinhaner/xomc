package notification

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPgDomainRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_DOMAIN_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_DOMAIN_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	requireDedicatedNotificationDatabase(t, databaseName)

	now := time.Now().UTC().Truncate(time.Millisecond)
	event := &DomainEvent{
		EventID: uuid.New(), EventType: "domain.alarm.lifecycle.raised", OccurrenceID: uuid.New(),
		AlarmVersion: 1, SchemaVersion: 1, OccurredAt: now, Payload: []byte(`{"schema_version":1}`),
	}
	inbox := NewPgInboxRepository(pool)
	inserted, err := inbox.InsertEvent(ctx, event)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = inbox.InsertEvent(ctx, event)
	require.NoError(t, err)
	require.False(t, inserted)

	_, err = pool.Exec(ctx, `INSERT INTO notification_occurrences
        (occurrence_id,last_applied_version,schedule_generation,status,severity,device_id,device_sn,carrier,technology,alarm_identifier,current_snapshot,raised_at,last_event_id)
        VALUES ($1,1,1,'active',31002,$2,'LAB-INTEGRATION','cmcc','nr','10001','{}',$3,$4)`,
		event.OccurrenceID, uuid.New(), now, event.EventID)
	require.NoError(t, err)
	occurrence, err := inbox.GetOccurrence(ctx, event.OccurrenceID)
	require.NoError(t, err)
	require.Equal(t, int64(1), occurrence.LastAppliedVersion)

	ruleID, ruleVersionID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_rules (id,name,created_by) VALUES ($1,$2,'integration')`,
		ruleID, "integration-"+ruleID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_rule_versions (id,rule_id,version_no,created_by,published_at) VALUES ($1,$2,1,'integration',$3)`,
		ruleVersionID, ruleID, now)
	require.NoError(t, err)
	ruleVersion, err := NewPgRuleVersionRepository(pool).GetRuleVersion(ctx, ruleVersionID)
	require.NoError(t, err)
	require.Equal(t, ruleID, ruleVersion.RuleID)

	templateID, templateVersionID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_templates (id,name,channel,subject,body,created_by) VALUES ($1,$2,'email','subject','body','integration')`,
		templateID, "integration-"+templateID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_template_versions (id,template_id,version_no,channel,subject,text_body,created_by,published_at) VALUES ($1,$2,1,'email','subject','body','integration',$3)`,
		templateVersionID, templateID, now)
	require.NoError(t, err)
	templateVersion, err := NewPgTemplateVersionRepository(pool).GetTemplateVersion(ctx, templateVersionID)
	require.NoError(t, err)
	require.Equal(t, templateID, templateVersion.TemplateID)

	channelConfigID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_channel_configs (id,channel,name,created_by) VALUES ($1,'email',$2,'integration')`,
		channelConfigID, "integration-"+channelConfigID.String())
	require.NoError(t, err)
	fingerprint := []byte("integration-fingerprint")
	schedule := &DomainSchedule{
		EventID: event.EventID, OccurrenceID: event.OccurrenceID, RuleVersionID: ruleVersionID,
		TemplateVersionID: templateVersionID, ChannelConfigID: channelConfigID, Channel: "email",
		DispatchKind: DispatchKindInitial, RecipientType: RecipientTargetUser,
		AddressCiphertext: []byte("ciphertext"), AddressKeyVersion: 1,
		RecipientFingerprint: fingerprint, ScheduleKind: "initial_gate", Generation: 1,
		DueAt: now, CreatedEventVersion: 1,
	}
	inserted, err = NewPgScheduleRepository(pool).InsertSchedule(ctx, schedule)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = NewPgScheduleRepository(pool).InsertSchedule(ctx, schedule)
	require.NoError(t, err)
	require.False(t, inserted)

	delivery := &DomainDelivery{
		EventID: event.EventID, OccurrenceID: event.OccurrenceID, RuleVersionID: ruleVersionID,
		TemplateVersionID: templateVersionID, ChannelConfigID: channelConfigID, Channel: "email",
		DispatchKind: "initial", RecipientType: "user", AddressCiphertext: []byte("ciphertext"),
		AddressKeyVersion: 1, RecipientFingerprint: fingerprint, OccurrenceVersion: 1, ScheduleGeneration: 1,
	}
	deliveries := NewPgDeliveryRepository(pool)
	inserted, err = deliveries.InsertDelivery(ctx, delivery)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = deliveries.InsertDelivery(ctx, delivery)
	require.NoError(t, err)
	require.False(t, inserted)
	attempt := &DomainDeliveryAttempt{DeliveryID: delivery.ID, AttemptNo: 1, StartedAt: now, Result: "accepted"}
	inserted, err = deliveries.InsertAttempt(ctx, attempt)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = deliveries.InsertAttempt(ctx, attempt)
	require.NoError(t, err)
	require.False(t, inserted)
}
