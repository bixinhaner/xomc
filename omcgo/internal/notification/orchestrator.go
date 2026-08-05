package notification

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	DispatchKindInitial    = "initial"
	DispatchKindEscalation = "escalation"
	DispatchKindRepeat     = "repeat"
	DispatchKindRecovery   = "recovery"
	DispatchKindDigest     = "digest"

	DeliveryModeRealtime = "realtime"
	DeliveryModeDigest   = "digest"

	SuppressionTransientBeforeGate = "transient_before_gate"
	SuppressionMaintenanceWindow   = "maintenance_window"
)

type NotificationPolicy struct {
	MinimumActiveSeconds     int             `json:"minimum_active_seconds"`
	DeliveryMode             string          `json:"delivery_mode"`
	AggregationWindowSeconds int             `json:"aggregation_window_seconds"`
	RepeatIntervalSeconds    int             `json:"repeat_interval_seconds"`
	MaxRepeatCount           int             `json:"max_repeat_count"`
	MaxRepeatDurationSeconds int             `json:"max_repeat_duration_seconds"`
	RecipientRateLimit       RateLimitPolicy `json:"recipient_rate_limit"`
}

type OrchestrationRecipient struct {
	RecipientType string
	ResolvedRecipient
}

type OrchestrationChannel struct {
	Channel                    string
	ChannelConfigID            uuid.UUID
	RaisedTemplateVersionID    uuid.UUID
	EscalatedTemplateVersionID *uuid.UUID
	ClearedTemplateVersionID   *uuid.UUID
	Policy                     *NotificationPolicy
}

type OrchestrationRule struct {
	RuleID      uuid.UUID
	VersionID   uuid.UUID
	Priority    int
	Specificity int
	Policy      NotificationPolicy
	Channels    []OrchestrationChannel
	Recipients  []OrchestrationRecipient
}

type RecoveryBinding struct {
	RuleVersionID     uuid.UUID
	Channel           string
	ChannelConfigID   uuid.UUID
	TemplateVersionID uuid.UUID
}

type PendingInitial struct {
	RuleVersionID        uuid.UUID
	Channel              string
	RecipientFingerprint []byte
}

type MaintenanceSuppression struct {
	WindowID       uuid.UUID
	Status         string
	SuppressAlarms bool
	ScopeMatched   bool
}

type SuppressionFact struct {
	RuleVersionID        uuid.UUID
	Channel              string
	RecipientFingerprint []byte
	Reason               string
	MaintenanceWindowID  *uuid.UUID
}

type OrchestrationExplanation struct {
	RuleVersionID uuid.UUID `json:"rule_version_id"`
	Outcome       string    `json:"outcome"`
	Reason        string    `json:"reason,omitempty"`
}

type OrchestrationInput struct {
	Now                 time.Time
	Event               event.AlarmLifecyclePayload
	Occurrence          DomainOccurrence
	Rules               []OrchestrationRule
	PriorDeliveries     []DomainDelivery
	RecoveryBindings    []RecoveryBinding
	PendingInitials     []PendingInitial
	Maintenance         *MaintenanceSuppression
	RecipientSendCounts map[string]int
	Explanations        []OrchestrationExplanation
}

type OrchestrationDecision struct {
	Schedules    []DomainSchedule
	Deliveries   []DomainDelivery
	Aggregations []AggregationFact
	Suppressions []SuppressionFact
	Explanations []OrchestrationExplanation
	Fence        *OrchestrationFence
}

type OrchestrationFence struct {
	OccurrenceID        uuid.UUID
	Generation          int64
	EventVersion        int64
	OccurredAt          time.Time
	ResumeFutureRepeats bool
}

// BuildOrchestrationDecision is the deterministic policy core. It only plans
// durable OMC work; it never calls an external mail or SMS provider.
func BuildOrchestrationDecision(input OrchestrationInput) (OrchestrationDecision, error) {
	if input.Now.IsZero() {
		return OrchestrationDecision{}, fmt.Errorf("build notification orchestration decision: current time is required")
	}
	if input.Event.EventID == uuid.Nil || input.Event.OccurrenceID == uuid.Nil {
		return OrchestrationDecision{}, fmt.Errorf("build notification orchestration decision: event and occurrence IDs are required")
	}

	var decision OrchestrationDecision
	switch input.Event.LifecycleType {
	case event.AlarmLifecycleRaised:
		decision = buildRaisedDecision(input)
	case event.AlarmLifecycleUpdated:
		decision = buildUpdatedDecision(input)
	case event.AlarmLifecycleCleared:
		decision = buildClearedDecision(input)
	case event.AlarmLifecycleAcknowledged, event.AlarmLifecycleUnacknowledged:
		decision = OrchestrationDecision{}
	default:
		return OrchestrationDecision{}, fmt.Errorf("build notification orchestration decision: unsupported lifecycle %q", input.Event.LifecycleType)
	}
	if input.Event.LifecycleType == event.AlarmLifecycleAcknowledged ||
		input.Event.LifecycleType == event.AlarmLifecycleUnacknowledged ||
		input.Event.LifecycleType == event.AlarmLifecycleCleared {
		decision.Fence = &OrchestrationFence{
			OccurrenceID: input.Event.OccurrenceID, Generation: input.Occurrence.ScheduleGeneration,
			EventVersion: input.Event.AlarmVersion, OccurredAt: input.Event.OccurredAt,
			ResumeFutureRepeats: input.Event.LifecycleType == event.AlarmLifecycleUnacknowledged,
		}
	}
	decision.Explanations = append([]OrchestrationExplanation(nil), input.Explanations...)
	return decision, nil
}

func buildRaisedDecision(input OrchestrationInput) OrchestrationDecision {
	decision := OrchestrationDecision{}
	rules := orderedOrchestrationRules(input.Rules)
	seen := make(map[string]struct{})

	for _, rule := range rules {
		for _, channel := range rule.Channels {
			policy := orchestrationChannelPolicy(rule, channel)
			for _, recipient := range rule.Recipients {
				if recipient.Channel != channel.Channel {
					continue
				}
				key := channel.Channel + ":" + string(recipient.Fingerprint)
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}

				if maintenanceSuppresses(input.Maintenance) {
					windowID := input.Maintenance.WindowID
					reason := SuppressionMaintenanceWindow
					decision.Suppressions = append(decision.Suppressions, SuppressionFact{
						RuleVersionID: rule.VersionID, Channel: channel.Channel,
						RecipientFingerprint: cloneBytes(recipient.Fingerprint),
						Reason:               SuppressionMaintenanceWindow, MaintenanceWindowID: &windowID,
					})
					decision.Deliveries = append(decision.Deliveries, newSuppressedDelivery(
						input, rule.VersionID, channel, recipient, DispatchKindInitial, reason, &windowID,
					))
					continue
				}

				windowSeconds := policy.RecipientRateLimit.WindowSeconds
				if windowSeconds <= 0 {
					windowSeconds = 60
				}
				currentCount, found := input.RecipientSendCounts[recipientRateUsageKey(channel.Channel, recipient.Fingerprint, windowSeconds)]
				if !found {
					currentCount = input.RecipientSendCounts[key]
				}
				rateDecision := EvaluateRateLimit(
					policy.RecipientRateLimit,
					currentCount,
					input.Event.Snapshot.Severity,
				)
				if !rateDecision.Allowed && !rateDecision.Aggregate {
					reason := rateDecision.Reason
					decision.Suppressions = append(decision.Suppressions, SuppressionFact{
						RuleVersionID: rule.VersionID, Channel: channel.Channel,
						RecipientFingerprint: cloneBytes(recipient.Fingerprint), Reason: reason,
					})
					decision.Deliveries = append(decision.Deliveries, newSuppressedDelivery(
						input, rule.VersionID, channel, recipient, DispatchKindInitial, reason, nil,
					))
					continue
				}

				ordinaryDigest := policy.DeliveryMode == DeliveryModeDigest ||
					(policy.DeliveryMode == "" && (input.Event.Snapshot.Severity == model.AlarmMinor || input.Event.Snapshot.Severity == model.AlarmWarning))
				useDigest := rateDecision.Aggregate ||
					(ordinaryDigest && input.Event.Snapshot.Severity != model.AlarmCritical)
				if useDigest {
					start, end := aggregationWindow(input.Now, policy.AggregationWindowSeconds)
					decision.Aggregations = append(decision.Aggregations, AggregationFact{
						RuleVersionID: rule.VersionID, Channel: channel.Channel,
						RecipientFingerprint: cloneBytes(recipient.Fingerprint),
						ScopeFingerprint:     aggregationScopeFingerprint(input.Event.Snapshot.DeviceID),
						Severity:             int16(input.Event.Snapshot.Severity), WindowStart: start, WindowEnd: end, EventCount: 1,
					})
					decision.Schedules = append(decision.Schedules, newSchedule(
						input, rule.VersionID, channel, channel.RaisedTemplateVersionID, recipient,
						ScheduleKindDigestFlush, DispatchKindDigest, digestScheduleSequence(end), end,
					))
					continue
				}

				originalGateDue := input.Event.Snapshot.RaisedAt.Add(time.Duration(policy.MinimumActiveSeconds) * time.Second)
				gateDue := originalGateDue
				if gateDue.Before(input.Now) {
					gateDue = input.Now
				}
				decision.Schedules = append(decision.Schedules, newSchedule(
					input, rule.VersionID, channel, channel.RaisedTemplateVersionID, recipient,
					ScheduleKindInitialGate, DispatchKindInitial, 0, gateDue,
				))

				interval := time.Duration(policy.RepeatIntervalSeconds) * time.Second
				maxDuration := time.Duration(policy.MaxRepeatDurationSeconds) * time.Second
				for sequence := 1; interval > 0 && sequence <= policy.MaxRepeatCount; sequence++ {
					dueAt := originalGateDue.Add(time.Duration(sequence) * interval)
					if maxDuration > 0 && dueAt.After(originalGateDue.Add(maxDuration)) {
						break
					}
					if !dueAt.After(input.Now) {
						continue
					}
					decision.Schedules = append(decision.Schedules, newSchedule(
						input, rule.VersionID, channel, channel.RaisedTemplateVersionID, recipient,
						ScheduleKindRepeat, DispatchKindRepeat, sequence, dueAt,
					))
				}
			}
		}
	}
	return decision
}

func buildUpdatedDecision(input OrchestrationInput) OrchestrationDecision {
	if input.Event.PreviousSeverity == nil ||
		!containsChange(input.Event.ChangeMask, event.AlarmChangeSeverity) ||
		input.Event.Snapshot.Severity >= *input.Event.PreviousSeverity {
		return OrchestrationDecision{}
	}

	decision := OrchestrationDecision{}
	seen := make(map[string]struct{})
	for _, rule := range orderedOrchestrationRules(input.Rules) {
		for _, channel := range rule.Channels {
			policy := orchestrationChannelPolicy(rule, channel)
			templateID := channel.RaisedTemplateVersionID
			if channel.EscalatedTemplateVersionID != nil {
				templateID = *channel.EscalatedTemplateVersionID
			}
			for _, recipient := range rule.Recipients {
				if recipient.Channel != channel.Channel {
					continue
				}
				key := channel.Channel + ":" + string(recipient.Fingerprint)
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}

				if maintenanceSuppresses(input.Maintenance) {
					windowID := input.Maintenance.WindowID
					decision.Suppressions = append(decision.Suppressions, SuppressionFact{
						RuleVersionID: rule.VersionID, Channel: channel.Channel,
						RecipientFingerprint: cloneBytes(recipient.Fingerprint),
						Reason:               SuppressionMaintenanceWindow, MaintenanceWindowID: &windowID,
					})
					decision.Deliveries = append(decision.Deliveries, newSuppressedDeliveryWithTemplate(
						input, rule.VersionID, channel, templateID, recipient, DispatchKindEscalation,
						SuppressionMaintenanceWindow, &windowID,
					))
					continue
				}

				windowSeconds := policy.RecipientRateLimit.WindowSeconds
				if windowSeconds <= 0 {
					windowSeconds = 60
				}
				currentCount := input.RecipientSendCounts[recipientRateUsageKey(channel.Channel, recipient.Fingerprint, windowSeconds)]
				rateDecision := EvaluateRateLimit(policy.RecipientRateLimit, currentCount, input.Event.Snapshot.Severity)
				if !rateDecision.Allowed && !rateDecision.Aggregate {
					decision.Suppressions = append(decision.Suppressions, SuppressionFact{
						RuleVersionID: rule.VersionID, Channel: channel.Channel,
						RecipientFingerprint: cloneBytes(recipient.Fingerprint), Reason: rateDecision.Reason,
					})
					decision.Deliveries = append(decision.Deliveries, newSuppressedDeliveryWithTemplate(
						input, rule.VersionID, channel, templateID, recipient, DispatchKindEscalation,
						rateDecision.Reason, nil,
					))
					continue
				}
				if rateDecision.Aggregate {
					start, end := aggregationWindow(input.Now, policy.AggregationWindowSeconds)
					decision.Aggregations = append(decision.Aggregations, AggregationFact{
						RuleVersionID: rule.VersionID, Channel: channel.Channel,
						RecipientFingerprint: cloneBytes(recipient.Fingerprint),
						ScopeFingerprint:     aggregationScopeFingerprint(input.Event.Snapshot.DeviceID),
						Severity:             int16(input.Event.Snapshot.Severity), WindowStart: start, WindowEnd: end, EventCount: 1,
					})
					decision.Schedules = append(decision.Schedules, newSchedule(
						input, rule.VersionID, channel, templateID, recipient,
						ScheduleKindDigestFlush, DispatchKindDigest, digestScheduleSequence(end), end,
					))
					continue
				}
				decision.Deliveries = append(decision.Deliveries, newDelivery(
					input, rule.VersionID, channel, templateID, recipient, DispatchKindEscalation, 0, nil,
				))
			}
		}
	}
	return decision
}

func buildClearedDecision(input OrchestrationInput) OrchestrationDecision {
	decision := OrchestrationDecision{}
	for _, pending := range input.PendingInitials {
		reason := SuppressionTransientBeforeGate
		decision.Suppressions = append(decision.Suppressions, SuppressionFact{
			RuleVersionID: pending.RuleVersionID, Channel: pending.Channel,
			RecipientFingerprint: cloneBytes(pending.RecipientFingerprint), Reason: SuppressionTransientBeforeGate,
		})
		if channel, recipient, ok := findPendingRecipient(input.Rules, pending); ok {
			decision.Deliveries = append(decision.Deliveries, newSuppressedDelivery(
				input, pending.RuleVersionID, channel, recipient, DispatchKindInitial, reason, nil,
			))
		}
	}

	seen := make(map[string]struct{})
	for _, prior := range input.PriorDeliveries {
		if prior.FlowState != "completed" || (prior.DeliveryResult != "accepted" && prior.DeliveryResult != "handoff_only") {
			continue
		}
		if prior.DispatchKind == DispatchKindRecovery {
			continue
		}
		key := prior.Channel + ":" + string(prior.RecipientFingerprint)
		if _, exists := seen[key]; exists {
			continue
		}
		binding, ok := findRecoveryBinding(input.RecoveryBindings, prior)
		if !ok {
			continue
		}
		seen[key] = struct{}{}
		originID := prior.ID
		decision.Deliveries = append(decision.Deliveries, DomainDelivery{
			ID: uuid.New(), EventID: input.Event.EventID, OccurrenceID: input.Event.OccurrenceID,
			RuleVersionID: prior.RuleVersionID, TemplateVersionID: binding.TemplateVersionID,
			ChannelConfigID: prior.ChannelConfigID, Channel: prior.Channel, DispatchKind: DispatchKindRecovery,
			RecipientType: prior.RecipientType, AddressCiphertext: cloneBytes(prior.AddressCiphertext),
			AddressKeyVersion: prior.AddressKeyVersion, RecipientFingerprint: cloneBytes(prior.RecipientFingerprint),
			FlowState: "queued", DeliveryResult: "none", AvailableAt: input.Now, NextAttemptAt: input.Now,
			OccurrenceVersion: input.Event.AlarmVersion, ScheduleGeneration: input.Occurrence.ScheduleGeneration,
			OriginDeliveryID: &originID, CreatedAt: input.Now,
		})
	}
	return decision
}

func newSchedule(
	input OrchestrationInput,
	ruleVersionID uuid.UUID,
	channel OrchestrationChannel,
	templateVersionID uuid.UUID,
	recipient OrchestrationRecipient,
	scheduleKind string,
	dispatchKind string,
	sequence int,
	dueAt time.Time,
) DomainSchedule {
	return DomainSchedule{
		ID: uuid.New(), EventID: input.Event.EventID, OccurrenceID: input.Event.OccurrenceID,
		RuleVersionID: ruleVersionID, TemplateVersionID: templateVersionID,
		ChannelConfigID: channel.ChannelConfigID, Channel: channel.Channel, DispatchKind: dispatchKind,
		RecipientType: recipient.RecipientType, AddressCiphertext: cloneBytes(recipient.Ciphertext),
		AddressKeyVersion: recipient.KeyVersion, RecipientFingerprint: cloneBytes(recipient.Fingerprint),
		ScheduleKind: scheduleKind,
		SequenceNo:   sequence, Generation: input.Occurrence.ScheduleGeneration, DueAt: dueAt,
		State: "pending", CreatedEventVersion: input.Event.AlarmVersion,
		CreatedAt: input.Now, UpdatedAt: input.Now,
	}
}

func newDelivery(input OrchestrationInput, ruleVersionID uuid.UUID, channel OrchestrationChannel, templateID uuid.UUID, recipient OrchestrationRecipient, kind string, sequence int, originID *uuid.UUID) DomainDelivery {
	return DomainDelivery{
		ID: uuid.New(), EventID: input.Event.EventID, OccurrenceID: input.Event.OccurrenceID,
		RuleVersionID: ruleVersionID, TemplateVersionID: templateID, ChannelConfigID: channel.ChannelConfigID,
		Channel: channel.Channel, DispatchKind: kind, SequenceNo: sequence, RecipientType: recipient.RecipientType,
		AddressCiphertext: cloneBytes(recipient.Ciphertext), AddressKeyVersion: recipient.KeyVersion,
		RecipientFingerprint: cloneBytes(recipient.Fingerprint), FlowState: "queued", DeliveryResult: "none",
		AvailableAt: input.Now, NextAttemptAt: input.Now, OccurrenceVersion: input.Event.AlarmVersion,
		ScheduleGeneration: input.Occurrence.ScheduleGeneration, OriginDeliveryID: originID, CreatedAt: input.Now,
	}
}

func newSuppressedDelivery(
	input OrchestrationInput,
	ruleVersionID uuid.UUID,
	channel OrchestrationChannel,
	recipient OrchestrationRecipient,
	kind string,
	reason string,
	maintenanceWindowID *uuid.UUID,
) DomainDelivery {
	return newSuppressedDeliveryWithTemplate(
		input, ruleVersionID, channel, channel.RaisedTemplateVersionID, recipient, kind, reason, maintenanceWindowID,
	)
}

func newSuppressedDeliveryWithTemplate(
	input OrchestrationInput,
	ruleVersionID uuid.UUID,
	channel OrchestrationChannel,
	templateID uuid.UUID,
	recipient OrchestrationRecipient,
	kind string,
	reason string,
	maintenanceWindowID *uuid.UUID,
) DomainDelivery {
	delivery := newDelivery(input, ruleVersionID, channel, templateID, recipient, kind, 0, nil)
	delivery.FlowState = "suppressed"
	delivery.SuppressionReason = &reason
	delivery.MaintenanceWindowID = maintenanceWindowID
	return delivery
}

func orderedOrchestrationRules(rules []OrchestrationRule) []OrchestrationRule {
	ordered := append([]OrchestrationRule(nil), rules...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority < ordered[j].Priority
		}
		if ordered[i].Specificity != ordered[j].Specificity {
			return ordered[i].Specificity > ordered[j].Specificity
		}
		return bytes.Compare(ordered[i].RuleID[:], ordered[j].RuleID[:]) < 0
	})
	return ordered
}

func orchestrationChannelPolicy(rule OrchestrationRule, channel OrchestrationChannel) NotificationPolicy {
	if channel.Policy != nil {
		return *channel.Policy
	}
	return rule.Policy
}

func findRecoveryBinding(bindings []RecoveryBinding, prior DomainDelivery) (RecoveryBinding, bool) {
	for _, binding := range bindings {
		if binding.RuleVersionID == prior.RuleVersionID && binding.Channel == prior.Channel &&
			binding.ChannelConfigID == prior.ChannelConfigID && binding.TemplateVersionID != uuid.Nil {
			return binding, true
		}
	}
	return RecoveryBinding{}, false
}

func findPendingRecipient(rules []OrchestrationRule, pending PendingInitial) (OrchestrationChannel, OrchestrationRecipient, bool) {
	for _, rule := range rules {
		if rule.VersionID != pending.RuleVersionID {
			continue
		}
		for _, channel := range rule.Channels {
			if channel.Channel != pending.Channel {
				continue
			}
			for _, recipient := range rule.Recipients {
				if recipient.Channel == pending.Channel && bytes.Equal(recipient.Fingerprint, pending.RecipientFingerprint) {
					return channel, recipient, true
				}
			}
		}
	}
	return OrchestrationChannel{}, OrchestrationRecipient{}, false
}

func maintenanceSuppresses(window *MaintenanceSuppression) bool {
	return window != nil && window.Status == "active" && window.SuppressAlarms && window.ScopeMatched
}

func containsChange(changes []event.AlarmChangeField, target event.AlarmChangeField) bool {
	for _, change := range changes {
		if change == target {
			return true
		}
	}
	return false
}

func cloneBytes(value []byte) []byte {
	return append([]byte(nil), value...)
}
