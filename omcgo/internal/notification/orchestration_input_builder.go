package notification

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type orchestrationRuleSource struct {
	RuleID     uuid.UUID
	VersionID  uuid.UUID
	Priority   int
	Conditions RuleMatchConditions
	Policy     NotificationPolicy
	Targets    []RecipientTarget
	Channels   []OrchestrationChannel
}

type PgOrchestrationInputBuilder struct {
	db           storage.DB
	resolver     *RecipientResolver
	deviceGroups RecipientDeviceGroups
	now          func() time.Time
}

func NewPgOrchestrationInputBuilder(
	pool *pgxpool.Pool,
	resolver *RecipientResolver,
	deviceGroups RecipientDeviceGroups,
) *PgOrchestrationInputBuilder {
	return &PgOrchestrationInputBuilder{
		db: storage.NewPoolDB(pool), resolver: resolver, deviceGroups: deviceGroups,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (b *PgOrchestrationInputBuilder) BuildOrchestrationInput(
	ctx context.Context,
	payload event.AlarmLifecyclePayload,
) (OrchestrationInput, error) {
	if b == nil || b.db == nil || b.resolver == nil || b.deviceGroups == nil {
		return OrchestrationInput{}, fmt.Errorf("build notification orchestration input: dependencies are required")
	}
	generation, err := b.eventGeneration(ctx, payload)
	if err != nil {
		return OrchestrationInput{}, err
	}
	input := OrchestrationInput{
		Now: b.now(), Event: payload,
		Occurrence:          occurrenceFromOrchestrationPayload(payload, generation),
		RecipientSendCounts: make(map[string]int),
	}
	if payload.LifecycleType == event.AlarmLifecycleAcknowledged || payload.LifecycleType == event.AlarmLifecycleUnacknowledged {
		return input, nil
	}
	if payload.LifecycleType == event.AlarmLifecycleUpdated &&
		(payload.PreviousSeverity == nil || !containsChange(payload.ChangeMask, event.AlarmChangeSeverity)) {
		return input, nil
	}
	if payload.LifecycleType == event.AlarmLifecycleCleared {
		if err := b.loadPriorDeliveries(ctx, &input); err != nil {
			return input, err
		}
		if err := b.loadHistoricalRecoveryBindings(ctx, &input); err != nil {
			return input, err
		}
		if err := b.loadPendingInitials(ctx, &input); err != nil {
			return input, err
		}
		return input, nil
	}

	sources, err := b.loadEnabledRuleSources(ctx)
	if err != nil {
		return input, err
	}
	candidates := make([]RuleCandidate, 0, len(sources))
	byRuleID := make(map[uuid.UUID]orchestrationRuleSource, len(sources))
	for _, source := range sources {
		candidates = append(candidates, RuleCandidate{ID: source.RuleID, Priority: source.Priority, Conditions: source.Conditions})
		byRuleID[source.RuleID] = source
	}
	deviceGroupIDs, err := b.deviceGroups.GetDeviceGroupIDs(ctx, payload.Snapshot.DeviceID)
	if err != nil {
		return input, fmt.Errorf("resolve notification rule device groups: %w", err)
	}
	for _, match := range MatchRulesForDeviceGroups(payload.Snapshot, deviceGroupIDs, candidates) {
		source := byRuleID[match.RuleID]
		resolution, err := b.resolver.Resolve(ctx, source.Targets, payload.Snapshot)
		if err != nil {
			return input, fmt.Errorf("resolve recipients for notification rule %s: %w", source.RuleID, err)
		}
		recipients := make([]OrchestrationRecipient, 0, len(resolution.Recipients))
		for _, recipient := range resolution.Recipients {
			recipientType := RecipientTargetFixedContact
			if recipient.UserID != nil {
				recipientType = RecipientTargetUser
			}
			recipients = append(recipients, OrchestrationRecipient{RecipientType: recipientType, ResolvedRecipient: recipient})
		}
		input.Rules = append(input.Rules, OrchestrationRule{
			RuleID: source.RuleID, VersionID: source.VersionID, Priority: source.Priority,
			Specificity: match.Specificity, Policy: source.Policy, Channels: source.Channels, Recipients: recipients,
		})
		outcome := "matched"
		if len(recipients) == 0 {
			outcome = "excluded"
		}
		input.Explanations = append(input.Explanations, OrchestrationExplanation{RuleVersionID: source.VersionID, Outcome: outcome})
		for _, exclusion := range resolution.Excluded {
			input.Explanations = append(input.Explanations, OrchestrationExplanation{
				RuleVersionID: source.VersionID, Outcome: "recipient_excluded", Reason: exclusion.Reason,
			})
		}
		for _, channel := range source.Channels {
			templateVersionID := channel.RaisedTemplateVersionID
			if channel.ClearedTemplateVersionID != nil {
				templateVersionID = *channel.ClearedTemplateVersionID
			}
			input.RecoveryBindings = append(input.RecoveryBindings, RecoveryBinding{
				RuleVersionID: source.VersionID, Channel: channel.Channel,
				ChannelConfigID: channel.ChannelConfigID, TemplateVersionID: templateVersionID,
			})
		}
	}

	if err := b.enrichLifecycleContext(ctx, &input); err != nil {
		return input, err
	}
	if err := b.enrichRateUsage(ctx, &input); err != nil {
		return input, err
	}
	return input, nil
}

func (b *PgOrchestrationInputBuilder) eventGeneration(ctx context.Context, payload event.AlarmLifecyclePayload) (int64, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").From("notification_events").
		Where(sq.Eq{
			"occurrence_id": payload.OccurrenceID, "processing_state": "applied",
			"event_type": []string{
				string(event.AlarmLifecycleAcknowledged), string(event.AlarmLifecycleUnacknowledged), string(event.AlarmLifecycleCleared),
			},
		}).Where(sq.LtOrEq{"alarm_version": payload.AlarmVersion}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build notification event generation lookup: %w", err)
	}
	var generationChanges int64
	if err := b.db.QueryRow(ctx, query, args...).Scan(&generationChanges); err != nil {
		return 0, fmt.Errorf("read notification event generation: %w", err)
	}
	return 1 + generationChanges, nil
}

func occurrenceFromOrchestrationPayload(payload event.AlarmLifecyclePayload, generation int64) DomainOccurrence {
	technology := ""
	if payload.Snapshot.Technology != nil {
		technology = *payload.Snapshot.Technology
	}
	return DomainOccurrence{
		OccurrenceID: payload.OccurrenceID, LastAppliedVersion: payload.AlarmVersion,
		ScheduleGeneration: generation, Status: string(payload.Snapshot.Status), Severity: int16(payload.Snapshot.Severity),
		DeviceID: payload.Snapshot.DeviceID, DeviceSN: payload.Snapshot.DeviceSN,
		Carrier: string(payload.Snapshot.Carrier), Technology: technology,
		AlarmIdentifier: payload.Snapshot.AlarmIdentifier, RaisedAt: payload.Snapshot.RaisedAt,
		AcknowledgedAt: payload.Snapshot.AcknowledgedAt, ClearedAt: payload.Snapshot.ClearedAt,
	}
}

func (b *PgOrchestrationInputBuilder) loadEnabledRuleSources(ctx context.Context) ([]orchestrationRuleSource, error) {
	query, args, err := storage.Psql.Select(
		"r.id", "r.current_enabled_version_id", "r.priority", "v.match_conditions", "v.policy",
	).From("notification_rules r").Join("notification_rule_versions v ON v.id = r.current_enabled_version_id").
		Where(sq.Eq{"r.archived": false}).OrderBy("r.priority", "r.id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build enabled notification rule source list: %w", err)
	}
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query enabled notification rule sources: %w", err)
	}
	sources := make([]orchestrationRuleSource, 0)
	for rows.Next() {
		var source orchestrationRuleSource
		var conditions, policy []byte
		if err := rows.Scan(&source.RuleID, &source.VersionID, &source.Priority, &conditions, &policy); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan enabled notification rule source: %w", err)
		}
		if err := json.Unmarshal(conditions, &source.Conditions); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode enabled notification rule conditions: %w", err)
		}
		if len(policy) > 0 {
			if err := json.Unmarshal(policy, &source.Policy); err != nil {
				rows.Close()
				return nil, fmt.Errorf("decode enabled notification rule policy: %w", err)
			}
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate enabled notification rule sources: %w", err)
	}
	rows.Close()
	for index := range sources {
		if err := b.loadRuleSourceBindings(ctx, &sources[index]); err != nil {
			return nil, err
		}
	}
	return sources, nil
}

func (b *PgOrchestrationInputBuilder) loadRuleSourceBindings(ctx context.Context, source *orchestrationRuleSource) error {
	query, args, err := storage.Psql.Select(
		"target_type", "target_id", "address_ciphertext", "address_key_version", "recipient_fingerprint", "channel_limit",
	).From("notification_rule_recipients").Where(sq.Eq{"rule_version_id": source.VersionID}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return fmt.Errorf("build orchestration rule recipient lookup: %w", err)
	}
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query orchestration rule recipients: %w", err)
	}
	for rows.Next() {
		var target RecipientTarget
		var targetID *string
		var keyVersion *int
		if err := rows.Scan(
			&target.TargetType, &targetID, &target.AddressCiphertext, &keyVersion,
			&target.RecipientFingerprint, &target.ChannelLimit,
		); err != nil {
			rows.Close()
			return fmt.Errorf("scan orchestration rule recipient: %w", err)
		}
		if targetID != nil {
			target.TargetID = *targetID
		}
		if keyVersion != nil {
			target.AddressKeyVersion = *keyVersion
		}
		source.Targets = append(source.Targets, target)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate orchestration rule recipients: %w", err)
	}
	rows.Close()

	query, args, err = storage.Psql.Select(
		"channel", "channel_config_id", "raised_template_version_id", "escalated_template_version_id",
		"cleared_template_version_id", "policy",
	).From("notification_rule_channels").Where(sq.Eq{"rule_version_id": source.VersionID}).OrderBy("channel", "id").ToSql()
	if err != nil {
		return fmt.Errorf("build orchestration rule channel lookup: %w", err)
	}
	rows, err = b.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query orchestration rule channels: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var channel OrchestrationChannel
		var policy []byte
		if err := rows.Scan(
			&channel.Channel, &channel.ChannelConfigID, &channel.RaisedTemplateVersionID,
			&channel.EscalatedTemplateVersionID, &channel.ClearedTemplateVersionID, &policy,
		); err != nil {
			return fmt.Errorf("scan orchestration rule channel: %w", err)
		}
		merged := source.Policy
		if len(policy) > 0 {
			if err := json.Unmarshal(policy, &merged); err != nil {
				return fmt.Errorf("decode orchestration rule channel policy: %w", err)
			}
		}
		channel.Policy = &merged
		source.Channels = append(source.Channels, channel)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate orchestration rule channels: %w", err)
	}
	return nil
}

func (b *PgOrchestrationInputBuilder) enrichLifecycleContext(ctx context.Context, input *OrchestrationInput) error {
	maintenance, err := b.findMaintenanceSuppression(ctx, input.Event.Snapshot)
	if err != nil {
		return err
	}
	input.Maintenance = maintenance
	return nil
}

func (b *PgOrchestrationInputBuilder) loadHistoricalRecoveryBindings(ctx context.Context, input *OrchestrationInput) error {
	if len(input.PriorDeliveries) == 0 {
		return nil
	}
	ruleVersions := make([]uuid.UUID, 0, len(input.PriorDeliveries))
	seen := make(map[uuid.UUID]struct{})
	for _, delivery := range input.PriorDeliveries {
		if _, exists := seen[delivery.RuleVersionID]; !exists {
			seen[delivery.RuleVersionID] = struct{}{}
			ruleVersions = append(ruleVersions, delivery.RuleVersionID)
		}
	}
	query, args, err := storage.Psql.Select(
		"rule_version_id", "channel", "channel_config_id", "COALESCE(cleared_template_version_id, raised_template_version_id)",
	).From("notification_rule_channels").Where(sq.Eq{"rule_version_id": ruleVersions}).ToSql()
	if err != nil {
		return fmt.Errorf("build historical recovery binding lookup: %w", err)
	}
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query historical recovery bindings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var binding RecoveryBinding
		if err := rows.Scan(&binding.RuleVersionID, &binding.Channel, &binding.ChannelConfigID, &binding.TemplateVersionID); err != nil {
			return fmt.Errorf("scan historical recovery binding: %w", err)
		}
		input.RecoveryBindings = append(input.RecoveryBindings, binding)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate historical recovery bindings: %w", err)
	}
	return nil
}

func (b *PgOrchestrationInputBuilder) loadPriorDeliveries(ctx context.Context, input *OrchestrationInput) error {
	query, args, err := storage.Psql.Select(
		"id", "event_id", "rule_version_id", "template_version_id", "channel_config_id", "channel",
		"dispatch_kind", "recipient_type", "address_ciphertext", "address_key_version", "recipient_fingerprint",
		"flow_state", "delivery_result", "occurrence_version", "schedule_generation",
	).From("notification_deliveries").Where(sq.Eq{"occurrence_id": input.Event.OccurrenceID}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return fmt.Errorf("build prior notification delivery lookup: %w", err)
	}
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query prior notification deliveries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var delivery DomainDelivery
		delivery.OccurrenceID = input.Event.OccurrenceID
		if err := rows.Scan(
			&delivery.ID, &delivery.EventID, &delivery.RuleVersionID, &delivery.TemplateVersionID,
			&delivery.ChannelConfigID, &delivery.Channel, &delivery.DispatchKind, &delivery.RecipientType,
			&delivery.AddressCiphertext, &delivery.AddressKeyVersion, &delivery.RecipientFingerprint,
			&delivery.FlowState, &delivery.DeliveryResult, &delivery.OccurrenceVersion, &delivery.ScheduleGeneration,
		); err != nil {
			return fmt.Errorf("scan prior notification delivery: %w", err)
		}
		input.PriorDeliveries = append(input.PriorDeliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate prior notification deliveries: %w", err)
	}
	return nil
}

func (b *PgOrchestrationInputBuilder) loadPendingInitials(ctx context.Context, input *OrchestrationInput) error {
	query, args, err := storage.Psql.Select(
		"s.rule_version_id", "s.channel", "s.recipient_fingerprint", "rc.channel_config_id", "rc.raised_template_version_id",
	).From("notification_schedules s").
		Join("notification_rule_channels rc ON rc.rule_version_id = s.rule_version_id AND rc.channel = s.channel").
		Where(sq.Eq{
			"s.occurrence_id": input.Event.OccurrenceID, "s.schedule_kind": ScheduleKindInitialGate,
			"s.state": []string{"pending", "cancelled"}, "s.delivery_id": nil,
		}).Where(sq.Lt{"s.generation": input.Occurrence.ScheduleGeneration}).ToSql()
	if err != nil {
		return fmt.Errorf("build pending notification initial lookup: %w", err)
	}
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query pending notification initials: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pending PendingInitial
		var configID, templateID uuid.UUID
		if err := rows.Scan(&pending.RuleVersionID, &pending.Channel, &pending.RecipientFingerprint, &configID, &templateID); err != nil {
			return fmt.Errorf("scan pending notification initial: %w", err)
		}
		input.PendingInitials = append(input.PendingInitials, pending)
		input.Rules = append(input.Rules, OrchestrationRule{
			RuleID: pending.RuleVersionID, VersionID: pending.RuleVersionID,
			Channels: []OrchestrationChannel{{
				Channel: pending.Channel, ChannelConfigID: configID, RaisedTemplateVersionID: templateID,
			}},
			Recipients: []OrchestrationRecipient{{
				RecipientType: RecipientTargetFixedContact,
				ResolvedRecipient: ResolvedRecipient{
					Channel: pending.Channel, Ciphertext: []byte{}, KeyVersion: 1,
					Fingerprint: cloneBytes(pending.RecipientFingerprint),
				},
			}},
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate pending notification initials: %w", err)
	}
	return nil
}

func (b *PgOrchestrationInputBuilder) findMaintenanceSuppression(
	ctx context.Context,
	snapshot event.AlarmLifecycleSnapshot,
) (*MaintenanceSuppression, error) {
	now := b.now()
	query, args, err := storage.Psql.Select("id", "scope_type", "scope_ids").From("ops_maintenance_windows").
		Where(sq.Eq{"status": "active", "suppress_alarms": true}).
		Where(sq.NotEq{"approver_user_id": nil}).Where(sq.LtOrEq{"start_at": now}).Where(sq.GtOrEq{"end_at": now}).
		OrderBy("start_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active alarm maintenance window lookup: %w", err)
	}
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active alarm maintenance windows: %w", err)
	}
	defer rows.Close()
	groups, err := b.deviceGroups.GetDeviceGroupIDs(ctx, snapshot.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("resolve alarm maintenance device groups: %w", err)
	}
	for rows.Next() {
		var id uuid.UUID
		var scopeType string
		var scopeIDs []byte
		if err := rows.Scan(&id, &scopeType, &scopeIDs); err != nil {
			return nil, fmt.Errorf("scan active alarm maintenance window: %w", err)
		}
		if maintenanceScopeMatches(scopeType, scopeIDs, snapshot.DeviceID, groups) {
			return &MaintenanceSuppression{WindowID: id, Status: "active", SuppressAlarms: true, ScopeMatched: true}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active alarm maintenance windows: %w", err)
	}
	return nil, nil
}

func maintenanceScopeMatches(scopeType string, raw []byte, deviceID uuid.UUID, groups []uuid.UUID) bool {
	if scopeType == "all" {
		return true
	}
	var ids []uuid.UUID
	if err := json.Unmarshal(raw, &ids); err != nil {
		return false
	}
	switch scopeType {
	case "device", "devices":
		return containsUUIDOrEmpty(ids, deviceID) && len(ids) > 0
	case "device_group", "device_groups", "group":
		for _, id := range ids {
			for _, group := range groups {
				if id == group {
					return true
				}
			}
		}
	}
	return false
}

func (b *PgOrchestrationInputBuilder) enrichRateUsage(ctx context.Context, input *OrchestrationInput) error {
	type rateKey struct {
		channel     string
		fingerprint string
		window      int
	}
	keys := make(map[rateKey][]byte)
	for _, rule := range input.Rules {
		for _, channel := range rule.Channels {
			policy := orchestrationChannelPolicy(rule, channel).RecipientRateLimit
			if policy.MaxCount <= 0 {
				continue
			}
			window := policy.WindowSeconds
			if window <= 0 {
				window = 60
			}
			for _, recipient := range rule.Recipients {
				if recipient.Channel == channel.Channel {
					keys[rateKey{channel: channel.Channel, fingerprint: hex.EncodeToString(recipient.Fingerprint), window: window}] = recipient.Fingerprint
				}
			}
		}
	}
	ordered := make([]rateKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].channel != ordered[j].channel {
			return ordered[i].channel < ordered[j].channel
		}
		if ordered[i].fingerprint != ordered[j].fingerprint {
			return ordered[i].fingerprint < ordered[j].fingerprint
		}
		return ordered[i].window < ordered[j].window
	})
	for _, key := range ordered {
		query, args, err := storage.Psql.Select("COUNT(*)").From("notification_deliveries").Where(sq.Eq{
			"channel": key.channel, "recipient_fingerprint": keys[key],
		}).Where(sq.NotEq{"flow_state": []string{"suppressed", "cancelled"}}).
			Where(sq.GtOrEq{"created_at": input.Now.Add(-time.Duration(key.window) * time.Second)}).ToSql()
		if err != nil {
			return fmt.Errorf("build notification recipient rate usage lookup: %w", err)
		}
		var count int
		if err := b.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
			return fmt.Errorf("read notification recipient rate usage: %w", err)
		}
		input.RecipientSendCounts[recipientRateUsageKey(key.channel, keys[key], key.window)] = count
	}
	return nil
}

func recipientRateUsageKey(channel string, fingerprint []byte, windowSeconds int) string {
	return strings.ToLower(channel) + ":" + hex.EncodeToString(fingerprint) + ":" + fmt.Sprintf("%d", windowSeconds)
}
