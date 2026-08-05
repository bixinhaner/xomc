package notification

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestBuildOrchestrationDecision_MinorUsesDigestBucket(t *testing.T) {
	now := time.Date(2026, 8, 5, 4, 7, 0, 0, time.UTC)
	input := orchestrationFixture(now, event.AlarmLifecycleRaised, model.AlarmMinor)
	input.Rules[0].Policy = NotificationPolicy{DeliveryMode: DeliveryModeDigest, AggregationWindowSeconds: 900}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Aggregations, 1)
	require.Len(t, decision.Schedules, 1)
	require.Equal(t, ScheduleKindDigestFlush, decision.Schedules[0].ScheduleKind)
	require.Equal(t, time.Date(2026, 8, 5, 4, 15, 0, 0, time.UTC), decision.Schedules[0].DueAt)
	require.Equal(t, digestScheduleSequence(decision.Schedules[0].DueAt), decision.Schedules[0].SequenceNo)
}

func TestAggregationWindowRejectsSubMinuteStormConfigurationAndDeduplicatesPerWindow(t *testing.T) {
	now := time.Date(2026, 8, 5, 4, 7, 0, 0, time.UTC)
	start, end := aggregationWindow(now, 1)
	require.Equal(t, time.Date(2026, 8, 5, 4, 0, 0, 0, time.UTC), start)
	require.Equal(t, start.Add(defaultAggregationWindow), end)

	_, nextEnd := aggregationWindow(end, int(defaultAggregationWindow/time.Second))
	require.NotEqual(t, digestScheduleSequence(end), digestScheduleSequence(nextEnd))
}

func TestBuildOrchestrationDecision_MinorDefaultsToDigestBucket(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleRaised, model.AlarmMinor)

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Len(t, decision.Aggregations, 1)
	require.Len(t, decision.Schedules, 1)
	require.Equal(t, ScheduleKindDigestFlush, decision.Schedules[0].ScheduleKind)
}

func TestMaintenanceSuppressionRequiresActiveApprovedScopeMatch(t *testing.T) {
	base := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleRaised, model.AlarmMajor)
	base.Maintenance = &MaintenanceSuppression{
		WindowID: base.Event.EventID, Status: "active", SuppressAlarms: true, ScopeMatched: true,
	}
	decision, err := BuildOrchestrationDecision(base)
	require.NoError(t, err)
	require.Empty(t, decision.Schedules)
	require.Len(t, decision.Deliveries, 1)
	require.Equal(t, "suppressed", decision.Deliveries[0].FlowState)
	require.Equal(t, base.Maintenance.WindowID, *decision.Deliveries[0].MaintenanceWindowID)
	require.Len(t, decision.Suppressions, 1)
	require.Equal(t, SuppressionMaintenanceWindow, decision.Suppressions[0].Reason)

	for _, mutate := range []func(*MaintenanceSuppression){
		func(window *MaintenanceSuppression) { window.Status = "planned" },
		func(window *MaintenanceSuppression) { window.SuppressAlarms = false },
		func(window *MaintenanceSuppression) { window.ScopeMatched = false },
	} {
		input := base
		window := *base.Maintenance
		mutate(&window)
		input.Maintenance = &window
		decision, err := BuildOrchestrationDecision(input)
		require.NoError(t, err)
		require.NotEmpty(t, decision.Schedules)
	}
}

func TestBuildOrchestrationDecision_CriticalBypassesOrdinaryDigest(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleRaised, model.AlarmCritical)
	input.Rules[0].Policy = NotificationPolicy{DeliveryMode: DeliveryModeDigest, AggregationWindowSeconds: 900}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Empty(t, decision.Aggregations)
	require.Len(t, decision.Schedules, 1)
	require.Equal(t, ScheduleKindInitialGate, decision.Schedules[0].ScheduleKind)
}

func TestBuildOrchestrationDecision_RateLimitOverflowIsAuditable(t *testing.T) {
	input := orchestrationFixture(time.Now().UTC(), event.AlarmLifecycleRaised, model.AlarmCritical)
	input.Rules[0].Policy.RecipientRateLimit = RateLimitPolicy{MaxCount: 1}
	key := input.Rules[0].Channels[0].Channel + ":" + string(input.Rules[0].Recipients[0].Fingerprint)
	input.RecipientSendCounts = map[string]int{key: 1}

	decision, err := BuildOrchestrationDecision(input)
	require.NoError(t, err)
	require.Empty(t, decision.Schedules)
	require.Len(t, decision.Deliveries, 1)
	require.Equal(t, "suppressed", decision.Deliveries[0].FlowState)
	require.Equal(t, SuppressionRecipientRateLimit, *decision.Deliveries[0].SuppressionReason)
}
