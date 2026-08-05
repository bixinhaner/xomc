package notification

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestPgOrchestrationRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_ORCHESTRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_ORCHESTRATION_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	requireDedicatedNotificationDatabase(t, databaseName)

	now := time.Now().UTC().Truncate(time.Millisecond)
	eventID, occurrenceID, deviceID := uuid.New(), uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_events
(event_id,event_type,occurrence_id,alarm_version,schema_version,occurred_at,payload,processing_state)
VALUES ($1,'raised',$2,1,1,$3,'{}','applied')`, eventID, occurrenceID, now)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_occurrences
(occurrence_id,last_applied_version,schedule_generation,status,severity,device_id,device_sn,carrier,technology,alarm_identifier,current_snapshot,raised_at,last_event_id)
VALUES ($1,1,1,'active',1,$2,'TASK10-INTEGRATION','cmcc','nr','POWER_FAIL','{}',$3,$4)`,
		occurrenceID, deviceID, now, eventID)
	require.NoError(t, err)

	ruleID, ruleVersionID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_rules (id,name,created_by) VALUES ($1,$2,'integration')`, ruleID, "task10-"+ruleID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_rule_versions (id,rule_id,version_no,created_by,published_at) VALUES ($1,$2,1,'integration',$3)`, ruleVersionID, ruleID, now)
	require.NoError(t, err)
	templateID, templateVersionID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_templates (id,name,channel,subject,body,created_by) VALUES ($1,$2,'sms','subject','body','integration')`, templateID, "task10-"+templateID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_template_versions (id,template_id,version_no,channel,subject,text_body,created_by,published_at) VALUES ($1,$2,1,'sms','subject','body','integration',$3)`, templateVersionID, templateID, now)
	require.NoError(t, err)
	configID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_channel_configs (id,channel,name,enabled,created_by) VALUES ($1,'sms_kafka',$2,true,'integration')`, configID, "task10-"+configID.String())
	require.NoError(t, err)
	fingerprint := []byte("task10-recipient")
	_, err = pool.Exec(ctx, `INSERT INTO notification_rule_recipients
(rule_version_id,target_type,address_ciphertext,address_key_version,recipient_fingerprint,channel_limit)
VALUES ($1,'fixed_contact',$2,1,$3,ARRAY['sms_kafka'])`, ruleVersionID, []byte("encrypted"), fingerprint)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_rule_channels
(rule_version_id,channel,channel_config_id,raised_template_version_id,policy)
VALUES ($1,'sms_kafka',$2,$3,'{}')`, ruleVersionID, configID, templateVersionID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE notification_rules SET current_enabled_version_id=$2 WHERE id=$1`, ruleID, ruleVersionID)
	require.NoError(t, err)
	windowID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO ops_maintenance_windows (id,name,scope_type,start_at,end_at,status) VALUES ($1,$2,'all',$3,$4,'active')`, windowID, "task10-"+windowID.String(), now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)

	payload := event.AlarmLifecyclePayload{
		SchemaVersion: event.AlarmLifecycleSchemaVersion, EventID: eventID,
		LifecycleType: event.AlarmLifecycleRaised, OccurredAt: now, OccurrenceID: occurrenceID, AlarmVersion: 1,
		Snapshot: event.AlarmLifecycleSnapshot{
			AlarmID: occurrenceID, DeviceID: deviceID, DeviceSN: "TASK10-INTEGRATION",
			Severity: model.AlarmMinor, Status: model.AlarmActive, AlarmIdentifier: "POWER_FAIL", RaisedAt: now,
		},
	}
	groups := recipientGroupsStub{}
	resolver := NewRecipientResolver(
		&recipientDirectoryStub{}, &recipientPermissionStub{}, groups, contactGroupsStub{}, recipientProtectorStub{},
	)
	builder := NewPgOrchestrationInputBuilder(pool, resolver, groups)
	builder.now = func() time.Time { return now }
	input, err := builder.BuildOrchestrationInput(ctx, payload)
	require.NoError(t, err)
	require.Len(t, input.Rules, 1)
	require.Equal(t, []byte("encrypted"), input.Rules[0].Recipients[0].Ciphertext)
	planned, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, planned.Aggregations, 1, "Minor defaults to a digest in the production input path")

	reason := SuppressionMaintenanceWindow
	decision := OrchestrationDecision{
		Schedules: []DomainSchedule{{
			ID: uuid.New(), OccurrenceID: occurrenceID, RuleVersionID: ruleVersionID,
			Channel: "sms_kafka", RecipientFingerprint: fingerprint, ScheduleKind: ScheduleKindDigestFlush,
			Generation: 1, DueAt: now.Add(15 * time.Minute), State: "pending", CreatedEventVersion: 1,
		}},
		Aggregations: []AggregationFact{{
			RuleVersionID: ruleVersionID, Channel: "sms_kafka", RecipientFingerprint: fingerprint,
			ScopeFingerprint: []byte("scope"), Severity: 3, WindowStart: now, WindowEnd: now.Add(15 * time.Minute), EventCount: 1,
		}},
		Deliveries: []DomainDelivery{{
			ID: uuid.New(), EventID: eventID, OccurrenceID: occurrenceID, RuleVersionID: ruleVersionID,
			TemplateVersionID: templateVersionID, ChannelConfigID: configID, Channel: "sms_kafka",
			DispatchKind: DispatchKindInitial, RecipientType: RecipientTargetFixedContact,
			AddressCiphertext: []byte("encrypted"), AddressKeyVersion: 1, RecipientFingerprint: fingerprint,
			FlowState: "suppressed", DeliveryResult: "none", AvailableAt: now, NextAttemptAt: now,
			OccurrenceVersion: 1, ScheduleGeneration: 1, SuppressionReason: &reason, MaintenanceWindowID: &windowID,
		}},
		Explanations: []OrchestrationExplanation{{RuleVersionID: ruleVersionID, Outcome: "matched"}},
	}
	repository := NewPgOrchestrationRepository(pool)
	inserted, err := repository.SaveDecision(ctx, eventID, decision)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = repository.SaveDecision(ctx, eventID, decision)
	require.NoError(t, err)
	require.False(t, inserted)

	var orchestrationState string
	var bucketCount, deliveryCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT orchestration_state FROM notification_events WHERE event_id=$1`, eventID).Scan(&orchestrationState))
	require.Equal(t, "completed", orchestrationState)
	require.NoError(t, pool.QueryRow(ctx, `SELECT event_count FROM notification_aggregation_buckets WHERE rule_version_id=$1`, ruleVersionID).Scan(&bucketCount))
	require.Equal(t, 1, bucketCount, "event replay must not increment the digest bucket twice")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_deliveries WHERE event_id=$1`, eventID).Scan(&deliveryCount))
	require.Equal(t, 1, deliveryCount)

	secondEventID, secondOccurrenceID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_events
(event_id,event_type,occurrence_id,alarm_version,schema_version,occurred_at,payload,processing_state)
VALUES ($1,'raised',$2,1,1,$3,'{}','applied')`, secondEventID, secondOccurrenceID, now)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_occurrences
(occurrence_id,last_applied_version,schedule_generation,status,severity,device_id,device_sn,carrier,technology,alarm_identifier,current_snapshot,raised_at,last_event_id)
VALUES ($1,1,1,'active',1,$2,'TASK10-INTEGRATION-2','cmcc','nr','POWER_FAIL','{}',$3,$4)`,
		secondOccurrenceID, uuid.New(), now, secondEventID)
	require.NoError(t, err)
	secondDecision := OrchestrationDecision{
		Schedules: []DomainSchedule{{
			ID: uuid.New(), OccurrenceID: secondOccurrenceID, RuleVersionID: ruleVersionID,
			Channel: "sms_kafka", RecipientFingerprint: fingerprint, ScheduleKind: ScheduleKindDigestFlush,
			Generation: 1, DueAt: now.Add(15 * time.Minute), State: "pending", CreatedEventVersion: 1,
		}},
		Aggregations: decision.Aggregations,
	}
	inserted, err = repository.SaveDecision(ctx, secondEventID, secondDecision)
	require.NoError(t, err)
	require.True(t, inserted)
	var scheduleCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_schedules
WHERE rule_version_id=$1 AND schedule_kind='digest_flush'`, ruleVersionID).Scan(&scheduleCount))
	require.Equal(t, 1, scheduleCount, "one digest bucket must create only one flush schedule")
	require.NoError(t, pool.QueryRow(ctx, `SELECT event_count FROM notification_aggregation_buckets WHERE rule_version_id=$1`, ruleVersionID).Scan(&bucketCount))
	require.Equal(t, 2, bucketCount)
}
