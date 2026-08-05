package notification

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestBuildOrchestrationDecision_RaisedPlansGateAndBoundedRepeats(t *testing.T) {
	now := time.Date(2026, 8, 5, 4, 0, 0, 0, time.UTC)
	input := orchestrationFixture(now, event.AlarmLifecycleRaised, model.AlarmCritical)
	input.Rules[0].Policy = NotificationPolicy{
		MinimumActiveSeconds: 60, RepeatIntervalSeconds: 300, MaxRepeatCount: 2,
	}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Schedules, 3)
	require.Equal(t, ScheduleKindInitialGate, decision.Schedules[0].ScheduleKind)
	require.Equal(t, now.Add(time.Minute), decision.Schedules[0].DueAt)
	require.Equal(t, ScheduleKindRepeat, decision.Schedules[1].ScheduleKind)
	require.Equal(t, 1, decision.Schedules[1].SequenceNo)
	require.Equal(t, now.Add(6*time.Minute), decision.Schedules[1].DueAt)
	require.Equal(t, 2, decision.Schedules[2].SequenceNo)
	require.Empty(t, decision.Deliveries, "scheduled work must not pretend an external delivery already exists")
}

func TestBuildOrchestrationDecision_DelayedRaiseDoesNotRestartRepeatDuration(t *testing.T) {
	raisedAt := time.Date(2026, 8, 5, 4, 0, 0, 0, time.UTC)
	now := raisedAt.Add(20 * time.Minute)
	input := orchestrationFixture(now, event.AlarmLifecycleRaised, model.AlarmCritical)
	input.Event.Snapshot.RaisedAt = raisedAt
	input.Rules[0].Policy = NotificationPolicy{
		MinimumActiveSeconds: 60, RepeatIntervalSeconds: 300, MaxRepeatCount: 10,
		MaxRepeatDurationSeconds: 30 * 60,
	}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Equal(t, now, decision.Schedules[0].DueAt, "initial gate may run now after a delayed event")
	require.Equal(t, []int{4, 5, 6}, []int{
		decision.Schedules[1].SequenceNo, decision.Schedules[2].SequenceNo, decision.Schedules[3].SequenceNo,
	})
	require.Equal(t, raisedAt.Add(31*time.Minute), decision.Schedules[3].DueAt)
	require.Len(t, decision.Schedules, 4, "missed repeats and work beyond the original duration must not be recreated")
}

func TestBuildOrchestrationDecision_OrdinaryUpdateDoesNotResend(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleUpdated, model.AlarmMajor)
	input.Event.ChangeMask = []event.AlarmChangeField{event.AlarmChangeCount}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Empty(t, decision.Schedules)
	require.Empty(t, decision.Deliveries)
}

func TestBuildOrchestrationDecision_SeverityUpgradeCreatesImmediateDelivery(t *testing.T) {
	now := time.Now().UTC()
	input := orchestrationFixture(now, event.AlarmLifecycleUpdated, model.AlarmCritical)
	previous := model.AlarmMajor
	input.Event.PreviousSeverity = &previous
	input.Event.ChangeMask = []event.AlarmChangeField{event.AlarmChangeSeverity}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Deliveries, 1)
	require.Equal(t, DispatchKindEscalation, decision.Deliveries[0].DispatchKind)
	require.Equal(t, input.Rules[0].Channels[0].EscalatedTemplateVersionID, &decision.Deliveries[0].TemplateVersionID)
}

func TestBuildOrchestrationDecision_SeverityUpgradeRespectsMaintenanceWindow(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleUpdated, model.AlarmCritical)
	previous := model.AlarmMajor
	input.Event.PreviousSeverity = &previous
	input.Event.ChangeMask = []event.AlarmChangeField{event.AlarmChangeSeverity}
	input.Maintenance = &MaintenanceSuppression{
		WindowID: uuid.New(), Status: "active", SuppressAlarms: true, ScopeMatched: true,
	}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Deliveries, 1)
	require.Equal(t, "suppressed", decision.Deliveries[0].FlowState)
	require.Equal(t, DispatchKindEscalation, decision.Deliveries[0].DispatchKind)
	require.Equal(t, *input.Rules[0].Channels[0].EscalatedTemplateVersionID, decision.Deliveries[0].TemplateVersionID)
	require.Equal(t, SuppressionMaintenanceWindow, *decision.Deliveries[0].SuppressionReason)
	require.Equal(t, input.Maintenance.WindowID, *decision.Deliveries[0].MaintenanceWindowID)
}

func TestBuildOrchestrationDecision_CriticalUpgradeDoesNotBypassRecipientRateCap(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleUpdated, model.AlarmCritical)
	previous := model.AlarmMajor
	input.Event.PreviousSeverity = &previous
	input.Event.ChangeMask = []event.AlarmChangeField{event.AlarmChangeSeverity}
	input.Rules[0].Policy.RecipientRateLimit = RateLimitPolicy{MaxCount: 1, WindowSeconds: 60}
	recipient := input.Rules[0].Recipients[0]
	input.RecipientSendCounts = make(map[string]int)
	input.RecipientSendCounts[recipientRateUsageKey("email", recipient.Fingerprint, 60)] = 1

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Deliveries, 1)
	require.Equal(t, "suppressed", decision.Deliveries[0].FlowState)
	require.Equal(t, DispatchKindEscalation, decision.Deliveries[0].DispatchKind)
	require.Equal(t, SuppressionRecipientRateLimit, *decision.Deliveries[0].SuppressionReason)
}

func TestBuildOrchestrationDecision_AcknowledgeAndUnacknowledgeDoNotCreateNewFirstNotification(t *testing.T) {
	for _, lifecycle := range []event.AlarmLifecycleType{event.AlarmLifecycleAcknowledged, event.AlarmLifecycleUnacknowledged} {
		t.Run(string(lifecycle), func(t *testing.T) {
			decision, err := BuildOrchestrationDecision(orchestrationFixture(time.Now().UTC(), lifecycle, model.AlarmCritical))
			require.NoError(t, err)
			require.Empty(t, decision.Schedules)
			require.Empty(t, decision.Deliveries)
		})
	}
}

func TestBuildOrchestrationDecision_RecoveryPairsOnlyAcceptedOrHandoff(t *testing.T) {
	now := time.Now().UTC()
	input := orchestrationFixture(now, event.AlarmLifecycleCleared, model.AlarmCritical)
	accepted := priorDelivery(input, "accepted", "completed")
	handoff := priorDelivery(input, "handoff_only", "completed")
	handoff.ID, handoff.Channel = uuid.New(), "sms_kafka"
	handoff.RecipientFingerprint = []byte("sms-fingerprint")
	failed := priorDelivery(input, "failed", "dead_letter")
	failed.ID, failed.RecipientFingerprint = uuid.New(), []byte("failed")
	suppressed := priorDelivery(input, "none", "suppressed")
	suppressed.ID, suppressed.RecipientFingerprint = uuid.New(), []byte("suppressed")
	input.PriorDeliveries = []DomainDelivery{accepted, handoff, failed, suppressed}
	input.RecoveryBindings = []RecoveryBinding{
		{RuleVersionID: accepted.RuleVersionID, Channel: "email", ChannelConfigID: accepted.ChannelConfigID, TemplateVersionID: uuid.New()},
		{RuleVersionID: handoff.RuleVersionID, Channel: "sms_kafka", ChannelConfigID: handoff.ChannelConfigID, TemplateVersionID: uuid.New()},
	}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Deliveries, 2)
	for _, delivery := range decision.Deliveries {
		require.Equal(t, DispatchKindRecovery, delivery.DispatchKind)
		require.NotNil(t, delivery.OriginDeliveryID)
	}
}

func TestBuildOrchestrationDecision_ClearBeforeGateRecordsSuppressionFact(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleCleared, model.AlarmMajor)
	input.PendingInitials = []PendingInitial{{
		RuleVersionID: input.Rules[0].VersionID, Channel: "email",
		RecipientFingerprint: input.Rules[0].Recipients[0].Fingerprint,
	}}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Suppressions, 1)
	require.Equal(t, SuppressionTransientBeforeGate, decision.Suppressions[0].Reason)
	require.Len(t, decision.Deliveries, 1)
	require.Equal(t, "suppressed", decision.Deliveries[0].FlowState)
	require.Equal(t, SuppressionTransientBeforeGate, *decision.Deliveries[0].SuppressionReason)
	require.Nil(t, decision.Deliveries[0].OriginDeliveryID, "a transient alarm must not produce orphan recovery")
}

func orchestrationFixture(now time.Time, lifecycle event.AlarmLifecycleType, severity model.AlarmSeverity) OrchestrationInput {
	occurrenceID, ruleID, versionID, configID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	raisedTemplate, escalatedTemplate, clearedTemplate := uuid.New(), uuid.New(), uuid.New()
	fingerprint := sha256.Sum256([]byte("noc@example.com"))
	ciphertext := sha256.Sum256([]byte("cipher:noc@example.com"))
	tech := "nr"
	return OrchestrationInput{
		Now: now,
		Event: event.AlarmLifecyclePayload{
			SchemaVersion: event.AlarmLifecycleSchemaVersion, EventID: uuid.New(), LifecycleType: lifecycle,
			OccurredAt: now, OccurrenceID: occurrenceID, AlarmVersion: 1,
			Snapshot: event.AlarmLifecycleSnapshot{
				AlarmID: occurrenceID, DeviceID: uuid.New(), DeviceSN: "SN001", Technology: &tech,
				Severity: severity, Status: model.AlarmActive, AlarmIdentifier: "POWER_FAIL", RaisedAt: now,
			},
		},
		Occurrence: DomainOccurrence{
			OccurrenceID: occurrenceID, LastAppliedVersion: 1, ScheduleGeneration: 1,
			Status: string(model.AlarmActive), Severity: int16(severity), RaisedAt: now,
		},
		Rules: []OrchestrationRule{{
			RuleID: ruleID, VersionID: versionID, Priority: 10,
			Channels: []OrchestrationChannel{{
				Channel: "email", ChannelConfigID: configID, RaisedTemplateVersionID: raisedTemplate,
				EscalatedTemplateVersionID: &escalatedTemplate, ClearedTemplateVersionID: &clearedTemplate,
			}},
			Recipients: []OrchestrationRecipient{{
				RecipientType:     RecipientTargetUser,
				ResolvedRecipient: ResolvedRecipient{Channel: "email", Ciphertext: ciphertext[:], KeyVersion: 1, Fingerprint: fingerprint[:]},
			}},
		}},
	}
}

func priorDelivery(input OrchestrationInput, result, flow string) DomainDelivery {
	rule, channel := input.Rules[0], input.Rules[0].Channels[0]
	recipient := rule.Recipients[0]
	return DomainDelivery{
		ID: uuid.New(), EventID: uuid.New(), OccurrenceID: input.Event.OccurrenceID,
		RuleVersionID: rule.VersionID, TemplateVersionID: channel.RaisedTemplateVersionID,
		ChannelConfigID: channel.ChannelConfigID, Channel: channel.Channel, DispatchKind: DispatchKindInitial,
		RecipientType: recipient.RecipientType, AddressCiphertext: recipient.Ciphertext,
		AddressKeyVersion: recipient.KeyVersion, RecipientFingerprint: recipient.Fingerprint,
		FlowState: flow, DeliveryResult: result, OccurrenceVersion: 1, ScheduleGeneration: 1,
	}
}
