package notification

import (
	"context"
	"net"
	"os"
	"strconv"
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
	_, err = pool.Exec(ctx, `UPDATE notification_channel_configs SET enabled=false WHERE channel='email'`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_channel_configs (id,channel,name,created_by) VALUES ($1,'email',$2,'integration')`,
		channelConfigID, "integration-"+channelConfigID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE notification_channel_configs SET enabled=true WHERE id=$1`, channelConfigID)
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

	schedules := NewPgScheduleRepository(pool)
	claimNow := now.Add(time.Second)
	claimed, err := schedules.ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: "integration-scheduler-a", Now: claimNow, LeaseDuration: 30 * time.Second, Limit: 1000,
	})
	require.NoError(t, err)
	require.Contains(t, scheduleIDs(claimed), schedule.ID)
	deliveries := NewPgDeliveryRepository(pool)
	materializedID, err := deliveries.CreateScheduledDelivery(ctx, *schedule, "integration-scheduler-a", claimNow)
	require.NoError(t, err)
	require.Equal(t, schedule.ID, materializedID)
	var linkedID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT delivery_id FROM notification_schedules WHERE id=$1`, schedule.ID).Scan(&linkedID))
	require.Equal(t, materializedID, linkedID)

	_, err = pool.Exec(ctx, `UPDATE notification_schedules SET lease_expires_at=$2 WHERE id=$1`, schedule.ID, claimNow)
	require.NoError(t, err)
	reclaimedAt := claimNow.Add(time.Second)
	claimed, err = schedules.ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: "integration-scheduler-b", Now: reclaimedAt, LeaseDuration: 30 * time.Second, Limit: 1000,
	})
	require.NoError(t, err)
	require.Contains(t, scheduleIDs(claimed), schedule.ID)
	replayedID, err := deliveries.CreateScheduledDelivery(ctx, *schedule, "integration-scheduler-b", reclaimedAt)
	require.NoError(t, err)
	require.Equal(t, materializedID, replayedID)
	var materializedCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_deliveries
WHERE event_id=$1 AND dispatch_kind=$2 AND sequence_no=$3 AND channel=$4 AND recipient_fingerprint=$5`,
		schedule.EventID, schedule.DispatchKind, schedule.SequenceNo, schedule.Channel, schedule.RecipientFingerprint,
	).Scan(&materializedCount))
	require.Equal(t, 1, materializedCount)
	require.NoError(t, schedules.MarkCompleted(ctx, schedule.ID, "integration-scheduler-b", reclaimedAt))

	emailClaimedAt := reclaimedAt.Add(time.Second)
	emailClaims, err := deliveries.ClaimEmailDeliveries(ctx, EmailDeliveryClaimRequest{
		WorkerID: "integration-email-a", Now: emailClaimedAt, LeaseDuration: time.Minute, Limit: 1000,
	})
	require.NoError(t, err)
	require.Contains(t, deliveryIDs(emailClaims), materializedID)
	authorized, err := deliveries.AuthorizeSend(ctx, materializedID, emailClaimedAt)
	require.NoError(t, err)
	require.Equal(t, materializedID, authorized.ID)
	emailAttempt, err := deliveries.StartEmailAttempt(ctx, materializedID, "integration-email-a", emailClaimedAt)
	require.NoError(t, err)
	require.Equal(t, 1, emailAttempt.AttemptNo)
	content, err := deliveries.LoadEmailDeliveryContent(ctx, authorized)
	require.NoError(t, err)
	require.Equal(t, "subject", content.Template.Subject)
	require.NoError(t, deliveries.FinishEmailAttempt(ctx, EmailAttemptCompletion{
		DeliveryID: materializedID, AttemptID: emailAttempt.ID, WorkerID: "integration-email-a",
		FinishedAt: emailClaimedAt, AttemptResult: "accepted", FlowState: "completed", DeliveryResult: "accepted",
	}))
	var flowState, deliveryResult, circuitState string
	require.NoError(t, pool.QueryRow(ctx, `SELECT flow_state,delivery_result FROM notification_deliveries WHERE id=$1`, materializedID).Scan(&flowState, &deliveryResult))
	require.Equal(t, "completed", flowState)
	require.Equal(t, "accepted", deliveryResult)
	require.NoError(t, pool.QueryRow(ctx, `SELECT circuit_state FROM notification_channel_health WHERE channel_config_id=$1`, channelConfigID).Scan(&circuitState))
	require.Equal(t, "closed", circuitState)

	workerNow := emailClaimedAt.Add(time.Second)
	workerDelivery := &DomainDelivery{
		EventID: event.EventID, OccurrenceID: event.OccurrenceID, RuleVersionID: ruleVersionID,
		TemplateVersionID: templateVersionID, ChannelConfigID: channelConfigID, Channel: "email",
		DispatchKind: DispatchKindRepeat, SequenceNo: 1, RecipientType: RecipientTargetUser,
		AddressCiphertext: []byte("integration-worker-ciphertext"), AddressKeyVersion: 1,
		RecipientFingerprint: []byte("integration-worker-fingerprint"), OccurrenceVersion: 1,
		ScheduleGeneration: 1, AvailableAt: workerNow.Add(-time.Second), NextAttemptAt: workerNow.Add(-time.Second),
	}
	inserted, err = deliveries.InsertDelivery(ctx, workerDelivery)
	require.NoError(t, err)
	require.True(t, inserted)
	smtpAddr, getSMTPData := startFakeSMTP(t, fakeSMTPBehavior{connections: 1, finalDataReply: true, quitReply: true})
	smtpHost, smtpPortText, err := net.SplitHostPort(smtpAddr)
	require.NoError(t, err)
	smtpPort, err := strconv.Atoi(smtpPortText)
	require.NoError(t, err)
	workerCalls := make([]string, 0, 1)
	emailWorker := NewEmailWorker(
		deliveries,
		emailUnprotectorStub{address: "noc-acceptance@example.invalid", calls: &workerCalls},
		NewEmailSender(SMTPOptions{Enabled: true, Host: smtpHost, Port: smtpPort, From: "omc@example.invalid"}, nil),
		"integration-email-worker",
		nil,
	)
	emailWorker.now = func() time.Time { return workerNow }
	processed, err := emailWorker.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	var attemptResult string
	require.NoError(t, pool.QueryRow(ctx, `SELECT flow_state,delivery_result FROM notification_deliveries WHERE id=$1`, workerDelivery.ID).Scan(&flowState, &deliveryResult))
	require.Equal(t, "completed", flowState)
	require.Equal(t, "accepted", deliveryResult)
	require.NoError(t, pool.QueryRow(ctx, `SELECT result FROM notification_delivery_attempts WHERE delivery_id=$1`, workerDelivery.ID).Scan(&attemptResult))
	require.Equal(t, "accepted", attemptResult)
	smtpMessages := getSMTPData()
	require.Len(t, smtpMessages, 1)
	require.Contains(t, smtpMessages[0], "To: noc-acceptance@example.invalid\r\n")
	require.Contains(t, smtpMessages[0], "Subject: subject\r\n")

	circuitNow := workerNow.Add(time.Second)
	circuitDeliveries := make([]DomainDelivery, 3)
	for index := range circuitDeliveries {
		circuitDeliveries[index] = *workerDelivery
		circuitDeliveries[index].ID = uuid.Nil
		circuitDeliveries[index].SequenceNo = index + 2
		circuitDeliveries[index].RecipientFingerprint = []byte(uuid.New().String())
		circuitDeliveries[index].AvailableAt = circuitNow.Add(-time.Second)
		circuitDeliveries[index].NextAttemptAt = circuitNow.Add(-time.Second)
		inserted, err = deliveries.InsertDelivery(ctx, &circuitDeliveries[index])
		require.NoError(t, err)
		require.True(t, inserted)
	}
	failingSMTPAddr, _ := startFakeSMTP(t, fakeSMTPBehavior{
		connections: 3, quitReply: true, recipientReply: "451 temporary recipient failure\r\n",
	})
	failingSMTPHost, failingSMTPPortText, err := net.SplitHostPort(failingSMTPAddr)
	require.NoError(t, err)
	failingSMTPPort, err := strconv.Atoi(failingSMTPPortText)
	require.NoError(t, err)
	circuitWorkerCalls := make([]string, 0, 3)
	circuitWorker := NewEmailWorker(
		deliveries,
		emailUnprotectorStub{address: "noc-acceptance@example.invalid", calls: &circuitWorkerCalls},
		NewEmailSender(SMTPOptions{
			Enabled: true, Host: failingSMTPHost, Port: failingSMTPPort, From: "omc@example.invalid",
		}, nil),
		"integration-email-circuit",
		nil,
	)
	circuitWorker.now = func() time.Time { return circuitNow }
	processed, err = circuitWorker.RunOnce(ctx)
	require.Error(t, err)
	require.Equal(t, 3, processed)
	for index := range circuitDeliveries {
		var errorCategory string
		require.NoError(t, pool.QueryRow(ctx, `SELECT d.flow_state,d.delivery_result,a.result,a.error_category
FROM notification_deliveries d JOIN notification_delivery_attempts a ON a.delivery_id=d.id
WHERE d.id=$1`, circuitDeliveries[index].ID).Scan(&flowState, &deliveryResult, &attemptResult, &errorCategory))
		require.Equal(t, "retry_wait", flowState)
		require.Equal(t, "failed", deliveryResult)
		require.Equal(t, "failed", attemptResult)
		require.Equal(t, EmailErrorTemporary, errorCategory)
	}
	require.NoError(t, pool.QueryRow(ctx, `SELECT circuit_state FROM notification_channel_health WHERE channel_config_id=$1`, channelConfigID).Scan(&circuitState))
	require.Equal(t, "open", circuitState)
	channelRepository := NewPgChannelConfigRepository(pool)
	verifiedAt := circuitNow.Add(time.Minute)
	require.NoError(t, channelRepository.RecordVerification(ctx, channelConfigID, verifiedAt, nil, nil))
	var lastVerifiedAt time.Time
	require.NoError(t, pool.QueryRow(ctx, `SELECT circuit_state,last_verified_at FROM notification_channel_health WHERE channel_config_id=$1`, channelConfigID).Scan(&circuitState, &lastVerifiedAt))
	require.Equal(t, "closed", circuitState)
	require.WithinDuration(t, verifiedAt, lastVerifiedAt, time.Millisecond)

	digestEnd := emailClaimedAt.Add(15 * time.Minute)
	digestBucketID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_aggregation_buckets
(id,rule_version_id,channel,recipient_fingerprint,scope_fingerprint,severity,window_started_at,window_ends_at,event_count)
VALUES ($1,$2,'email',$3,$4,3,$5,$6,4)`, digestBucketID, ruleVersionID, fingerprint, []byte("scope"), emailClaimedAt, digestEnd)
	require.NoError(t, err)
	digestSchedule := *schedule
	digestSchedule.ID, digestSchedule.ScheduleKind = uuid.New(), ScheduleKindDigestFlush
	digestSchedule.DispatchKind, digestSchedule.SequenceNo = DispatchKindDigest, 99
	digestSchedule.DueAt, digestSchedule.State, digestSchedule.DeliveryID = digestEnd, "pending", nil
	digestSchedule.LockedBy, digestSchedule.LockedAt, digestSchedule.LeaseExpiresAt = nil, nil, nil
	inserted, err = schedules.InsertSchedule(ctx, &digestSchedule)
	require.NoError(t, err)
	require.True(t, inserted)
	digestClaimedAt := digestEnd.Add(time.Second)
	claimed, err = schedules.ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: "integration-digest", Now: digestClaimedAt, LeaseDuration: time.Minute,
		Limit: 1000, Kinds: []string{ScheduleKindDigestFlush},
	})
	require.NoError(t, err)
	require.Contains(t, scheduleIDs(claimed), digestSchedule.ID)
	digestDeliveryID, err := deliveries.CreateScheduledDelivery(ctx, digestSchedule, "integration-digest", digestClaimedAt)
	require.NoError(t, err)
	var digestBucketState string
	var linkedDigestDeliveryID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT state,delivery_id FROM notification_aggregation_buckets WHERE id=$1`, digestBucketID).Scan(&digestBucketState, &linkedDigestDeliveryID))
	require.Equal(t, "closed", digestBucketState)
	require.Equal(t, digestDeliveryID, linkedDigestDeliveryID)
	digestContent, err := deliveries.LoadEmailDeliveryContent(ctx, AuthorizedDelivery{DomainDelivery: deliveryFromSchedule(digestSchedule, digestClaimedAt)})
	require.NoError(t, err)
	require.Equal(t, 4, digestContent.DigestEventCount)
	require.WithinDuration(t, emailClaimedAt, *digestContent.DigestWindowStartedAt, time.Millisecond)
	require.WithinDuration(t, digestEnd, *digestContent.DigestWindowEndsAt, time.Millisecond)
	require.NoError(t, schedules.MarkCompleted(ctx, digestSchedule.ID, "integration-digest", digestClaimedAt))

	delivery := &DomainDelivery{
		EventID: event.EventID, OccurrenceID: event.OccurrenceID, RuleVersionID: ruleVersionID,
		TemplateVersionID: templateVersionID, ChannelConfigID: channelConfigID, Channel: "email",
		DispatchKind: "initial", RecipientType: "user", AddressCiphertext: []byte("ciphertext"),
		AddressKeyVersion: 1, RecipientFingerprint: []byte("integration-direct-fingerprint"), OccurrenceVersion: 1, ScheduleGeneration: 1,
	}
	inserted, err = deliveries.InsertDelivery(ctx, delivery)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = deliveries.InsertDelivery(ctx, delivery)
	require.NoError(t, err)
	require.False(t, inserted)
	attemptFinishedAt := now
	attempt := &DomainDeliveryAttempt{DeliveryID: delivery.ID, AttemptNo: 1, StartedAt: now, FinishedAt: &attemptFinishedAt, Result: "accepted"}
	inserted, err = deliveries.InsertAttempt(ctx, attempt)
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = deliveries.InsertAttempt(ctx, attempt)
	require.NoError(t, err)
	require.False(t, inserted)

	unknownDelivery := *delivery
	unknownDelivery.ID = uuid.Nil
	unknownDelivery.RecipientFingerprint = []byte("integration-unknown-fingerprint")
	inserted, err = deliveries.InsertDelivery(ctx, &unknownDelivery)
	require.NoError(t, err)
	require.True(t, inserted)
	unknownClaimedAt := emailClaimedAt.Add(time.Minute)
	emailClaims, err = deliveries.ClaimEmailDeliveries(ctx, EmailDeliveryClaimRequest{
		WorkerID: "integration-email-crashed", Now: unknownClaimedAt, LeaseDuration: time.Minute, Limit: 1000,
	})
	require.NoError(t, err)
	require.Contains(t, deliveryIDs(emailClaims), unknownDelivery.ID)
	unknownAttempt, err := deliveries.StartEmailAttempt(ctx, unknownDelivery.ID, "integration-email-crashed", unknownClaimedAt)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE notification_deliveries SET lease_expires_at=$2 WHERE id=$1`, unknownDelivery.ID, unknownClaimedAt)
	require.NoError(t, err)
	afterCrash := unknownClaimedAt.Add(time.Second)
	_, err = deliveries.ClaimEmailDeliveries(ctx, EmailDeliveryClaimRequest{
		WorkerID: "integration-email-recovery", Now: afterCrash, LeaseDuration: time.Minute, Limit: 1000,
	})
	require.NoError(t, err)
	var unknownAttemptResult string
	require.NoError(t, pool.QueryRow(ctx, `SELECT flow_state,delivery_result FROM notification_deliveries WHERE id=$1`, unknownDelivery.ID).Scan(&flowState, &deliveryResult))
	require.Equal(t, "completed", flowState)
	require.Equal(t, "unknown", deliveryResult)
	require.NoError(t, pool.QueryRow(ctx, `SELECT result FROM notification_delivery_attempts WHERE id=$1`, unknownAttempt.ID).Scan(&unknownAttemptResult))
	require.Equal(t, "unknown", unknownAttemptResult)

	authCategory, verificationSummary := EmailErrorAuthentication, "smtp verification failed"
	require.NoError(t, channelRepository.RecordVerification(ctx, channelConfigID, verifiedAt.Add(time.Minute), &authCategory, &verificationSummary))
	require.NoError(t, pool.QueryRow(ctx, `SELECT circuit_state FROM notification_channel_health WHERE channel_config_id=$1`, channelConfigID).Scan(&circuitState))
	require.Equal(t, "open", circuitState)
}

func scheduleIDs(schedules []DomainSchedule) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(schedules))
	for _, schedule := range schedules {
		ids = append(ids, schedule.ID)
	}
	return ids
}

func deliveryIDs(deliveries []DomainDelivery) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(deliveries))
	for _, delivery := range deliveries {
		ids = append(ids, delivery.ID)
	}
	return ids
}
