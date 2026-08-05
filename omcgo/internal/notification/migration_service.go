package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/alarm"
)

var (
	ErrMigrationRuleNotFound         = errors.New("legacy email rule not found")
	ErrMigrationScopeUnsupported     = errors.New("legacy email rule scope cannot be represented safely")
	ErrMigrationOverlapUnconfirmed   = errors.New("legacy rule overlap findings are not confirmed")
	ErrMigrationToleranceUnconfirmed = errors.New("legacy tolerance duration semantics are not confirmed")
	ErrMigrationShadowMismatch       = errors.New("legacy and notification shadow results do not match")
	ErrMigrationCandidateNotEnabled  = errors.New("notification migration candidate is not enabled")
	ErrMigrationCandidateMismatch    = errors.New("notification migration candidate does not match the legacy rule")
)

const migrationReasonPrefix = "legacy alarm filter migration:"

type LegacyEmailMigrationRepository interface {
	ListEnabled(context.Context) ([]alarm.AlarmFilterRule, error)
	ReplaceNotifyEmailWithBarrier(context.Context, uuid.UUID, time.Time, uuid.UUID, uuid.UUID, string) error
}

type MigrationRuleRepository interface {
	Get(context.Context, uuid.UUID) (*NotificationRule, error)
	Create(context.Context, RuleDraftInput, string) (*NotificationRule, error)
}

type MigrationOverlapFinding struct {
	RuleID   uuid.UUID `json:"rule_id"`
	Name     string    `json:"name"`
	Action   string    `json:"action"`
	Priority int       `json:"priority"`
}

type LegacyEmailMigrationPreview struct {
	LegacyRuleID               uuid.UUID                 `json:"legacy_rule_id"`
	LegacyRuleName             string                    `json:"legacy_rule_name"`
	LegacyRuleUpdatedAt        time.Time                 `json:"legacy_rule_updated_at"`
	LegacyPriority             int                       `json:"legacy_priority"`
	CandidatePriority          int                       `json:"candidate_priority"`
	AlarmSources               []string                  `json:"alarm_sources"`
	AlarmIdentifiers           []string                  `json:"alarm_identifiers"`
	DeviceIDs                  []uuid.UUID               `json:"device_ids"`
	DeviceGroupIDs             []uuid.UUID               `json:"device_group_ids"`
	RecipientEmails            []string                  `json:"recipient_emails"`
	CandidateName              string                    `json:"candidate_name"`
	CandidateConditions        RuleMatchConditions       `json:"candidate_conditions"`
	CandidateEnabled           bool                      `json:"candidate_enabled"`
	OverlapFindings            []MigrationOverlapFinding `json:"overlap_findings"`
	UnsupportedScopeFields     []string                  `json:"unsupported_scope_fields"`
	ToleranceDurationConfirmed bool                      `json:"tolerance_duration_confirmed"`
}

type MigrationCandidateBinding struct {
	ChannelConfigID         uuid.UUID
	RaisedTemplateVersionID uuid.UUID
}

type MigrationCutoverInput struct {
	LegacyRuleID               uuid.UUID
	NotificationRuleID         uuid.UUID
	ExpectedLegacyUpdatedAt    time.Time
	ConfirmedOverlapRuleIDs    []uuid.UUID
	ToleranceDurationConfirmed bool
	ShadowComparedEvents       int64
	LegacyShadowMatches        int64
	NotificationShadowMatches  int64
	Actor                      string
}

type MigrationService struct {
	legacy    LegacyEmailMigrationRepository
	rules     MigrationRuleRepository
	protector RecipientProtector
}

func NewMigrationService(
	legacy LegacyEmailMigrationRepository,
	rules MigrationRuleRepository,
	protector RecipientProtector,
) *MigrationService {
	return &MigrationService{legacy: legacy, rules: rules, protector: protector}
}

func (s *MigrationService) Preview(ctx context.Context) ([]LegacyEmailMigrationPreview, error) {
	rules, err := s.orderedEnabledRules(ctx)
	if err != nil {
		return nil, err
	}
	previews := make([]LegacyEmailMigrationPreview, 0)
	for index := range rules {
		if rules[index].Action != alarm.FilterActionNotifyEmail {
			continue
		}
		previews = append(previews, buildLegacyEmailPreview(rules, index))
	}
	return previews, nil
}

func (s *MigrationService) CreateDisabledCandidate(
	ctx context.Context,
	legacyRuleID uuid.UUID,
	binding MigrationCandidateBinding,
	actor string,
) (*NotificationRule, error) {
	preview, err := s.previewRule(ctx, legacyRuleID)
	if err != nil {
		return nil, err
	}
	if len(preview.UnsupportedScopeFields) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrMigrationScopeUnsupported, strings.Join(preview.UnsupportedScopeFields, ","))
	}
	if s.protector == nil || binding.ChannelConfigID == uuid.Nil || binding.RaisedTemplateVersionID == uuid.Nil {
		return nil, fmt.Errorf("create disabled notification migration candidate: %w", ErrMigrationCandidateMismatch)
	}
	recipients := make([]RuleRecipientInput, 0, len(preview.RecipientEmails))
	for _, address := range preview.RecipientEmails {
		ciphertext, keyVersion, fingerprint, protectErr := s.protector.Protect(TemplateChannelEmail, address)
		if protectErr != nil {
			return nil, fmt.Errorf("protect migrated email recipient: %w", protectErr)
		}
		recipients = append(recipients, RuleRecipientInput{
			TargetType: RecipientTargetFixedContact, AddressCiphertext: ciphertext,
			AddressKeyVersion: keyVersion, RecipientFingerprint: fingerprint,
			ChannelLimit: []string{TemplateChannelEmail},
		})
	}
	policy, err := json.Marshal(NotificationPolicy{DeliveryMode: DeliveryModeRealtime})
	if err != nil {
		return nil, fmt.Errorf("marshal migrated notification policy: %w", err)
	}
	input, err := normalizeRuleDraft(RuleDraftInput{
		Name: preview.CandidateName, Priority: preview.CandidatePriority,
		MatchConditions: preview.CandidateConditions, Policy: policy,
		ChangeReason: migrationReason(preview.LegacyRuleID), Recipients: recipients,
		Channels: []RuleChannelInput{{
			Channel: TemplateChannelEmail, ChannelConfigID: binding.ChannelConfigID,
			RaisedTemplateVersionID: binding.RaisedTemplateVersionID,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("validate disabled notification migration candidate: %w", err)
	}
	rule, err := s.rules.Create(ctx, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("create disabled notification migration candidate: %w", err)
	}
	if rule.CurrentEnabledVersionID != nil || rule.Enabled != nil {
		return nil, fmt.Errorf("%w: newly migrated rule was unexpectedly enabled", ErrMigrationCandidateMismatch)
	}
	return rule, nil
}

func (s *MigrationService) Cutover(ctx context.Context, input MigrationCutoverInput) error {
	preview, err := s.previewRule(ctx, input.LegacyRuleID)
	if err != nil {
		return err
	}
	if len(preview.UnsupportedScopeFields) > 0 {
		return fmt.Errorf("%w: %s", ErrMigrationScopeUnsupported, strings.Join(preview.UnsupportedScopeFields, ","))
	}
	if !sameUUIDSet(overlapRuleIDs(preview.OverlapFindings), input.ConfirmedOverlapRuleIDs) {
		return ErrMigrationOverlapUnconfirmed
	}
	if !input.ToleranceDurationConfirmed {
		return ErrMigrationToleranceUnconfirmed
	}
	if input.ShadowComparedEvents <= 0 || input.LegacyShadowMatches < 0 ||
		input.NotificationShadowMatches < 0 || input.LegacyShadowMatches != input.NotificationShadowMatches {
		return ErrMigrationShadowMismatch
	}
	candidate, err := s.rules.Get(ctx, input.NotificationRuleID)
	if err != nil {
		return fmt.Errorf("get notification migration candidate: %w", err)
	}
	if candidate.Archived || candidate.CurrentEnabledVersionID == nil || candidate.Enabled == nil ||
		*candidate.CurrentEnabledVersionID != candidate.Enabled.ID {
		return ErrMigrationCandidateNotEnabled
	}
	if !candidateMatchesPreview(candidate, preview) {
		return ErrMigrationCandidateMismatch
	}
	if !input.ExpectedLegacyUpdatedAt.Equal(preview.LegacyRuleUpdatedAt) {
		return alarm.ErrAlarmFilterMigrationPrecondition
	}
	if err := s.legacy.ReplaceNotifyEmailWithBarrier(
		ctx, input.LegacyRuleID, input.ExpectedLegacyUpdatedAt,
		candidate.ID, candidate.Enabled.ID, normalizeActor(input.Actor),
	); err != nil {
		return fmt.Errorf("cut over legacy email rule to compatibility barrier: %w", err)
	}
	return nil
}

func (s *MigrationService) previewRule(ctx context.Context, id uuid.UUID) (LegacyEmailMigrationPreview, error) {
	previews, err := s.Preview(ctx)
	if err != nil {
		return LegacyEmailMigrationPreview{}, err
	}
	for _, preview := range previews {
		if preview.LegacyRuleID == id {
			return preview, nil
		}
	}
	return LegacyEmailMigrationPreview{}, ErrMigrationRuleNotFound
}

func (s *MigrationService) orderedEnabledRules(ctx context.Context) ([]alarm.AlarmFilterRule, error) {
	rules, err := s.legacy.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled legacy alarm filter rules: %w", err)
	}
	rules = append([]alarm.AlarmFilterRule(nil), rules...)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority < rules[j].Priority
		}
		if !rules[i].CreatedAt.Equal(rules[j].CreatedAt) {
			return rules[i].CreatedAt.Before(rules[j].CreatedAt)
		}
		return bytes.Compare(rules[i].ID[:], rules[j].ID[:]) < 0
	})
	return rules, nil
}

func buildLegacyEmailPreview(rules []alarm.AlarmFilterRule, index int) LegacyEmailMigrationPreview {
	rule := rules[index]
	preview := LegacyEmailMigrationPreview{
		LegacyRuleID: rule.ID, LegacyRuleName: rule.Name, LegacyRuleUpdatedAt: rule.UpdatedAt,
		LegacyPriority: rule.Priority, CandidatePriority: index + 1,
		AlarmSources:     cloneStrings(rule.AlarmSources),
		AlarmIdentifiers: cloneStrings(rule.AlarmIdentifiers), DeviceIDs: cloneUUIDs(rule.DeviceIDs),
		DeviceGroupIDs: cloneUUIDs(rule.DeviceGroupIDs), RecipientEmails: normalizedEmails(rule.EmailRecipients),
		CandidateName: "legacy-email-" + rule.ID.String(),
		CandidateConditions: RuleMatchConditions{
			AlarmIdentifiers: cloneStrings(rule.AlarmIdentifiers), DeviceIDs: cloneUUIDs(rule.DeviceIDs),
		},
		CandidateEnabled: false, OverlapFindings: make([]MigrationOverlapFinding, 0),
		UnsupportedScopeFields: make([]string, 0), ToleranceDurationConfirmed: false,
	}
	if len(rule.AlarmSources) > 0 {
		preview.UnsupportedScopeFields = append(preview.UnsupportedScopeFields, "alarm_sources")
	}
	if len(rule.DeviceGroupIDs) > 0 {
		preview.UnsupportedScopeFields = append(preview.UnsupportedScopeFields, "device_group_ids")
	}
	for lowerIndex := index + 1; lowerIndex < len(rules); lowerIndex++ {
		lower := rules[lowerIndex]
		if legacyScopesOverlap(rule, lower) {
			preview.OverlapFindings = append(preview.OverlapFindings, MigrationOverlapFinding{
				RuleID: lower.ID, Name: lower.Name, Action: lower.Action, Priority: lower.Priority,
			})
		}
	}
	return preview
}

func legacyScopesOverlap(left, right alarm.AlarmFilterRule) bool {
	return stringDimensionsOverlap(left.AlarmSources, right.AlarmSources) &&
		stringDimensionsOverlap(left.AlarmIdentifiers, right.AlarmIdentifiers) &&
		uuidDimensionsOverlap(left.DeviceIDs, right.DeviceIDs) &&
		uuidDimensionsOverlap(left.DeviceGroupIDs, right.DeviceGroupIDs)
}

func stringDimensionsOverlap(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	values := make(map[string]struct{}, len(left))
	for _, value := range left {
		values[value] = struct{}{}
	}
	for _, value := range right {
		if _, exists := values[value]; exists {
			return true
		}
	}
	return false
}

func uuidDimensionsOverlap(left, right []uuid.UUID) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	values := make(map[uuid.UUID]struct{}, len(left))
	for _, value := range left {
		values[value] = struct{}{}
	}
	for _, value := range right {
		if _, exists := values[value]; exists {
			return true
		}
	}
	return false
}

func candidateMatchesPreview(candidate *NotificationRule, preview LegacyEmailMigrationPreview) bool {
	if candidate == nil || candidate.Enabled == nil || candidate.Name != preview.CandidateName ||
		candidate.Priority != preview.CandidatePriority || candidate.Enabled.ChangeReason != migrationReason(preview.LegacyRuleID) ||
		!equalMatchConditions(candidate.Enabled.MatchConditions, preview.CandidateConditions) ||
		len(candidate.Enabled.Recipients) != len(preview.RecipientEmails) {
		return false
	}
	for _, channel := range candidate.Enabled.Channels {
		if channel.Channel == TemplateChannelEmail {
			return true
		}
	}
	return false
}

func equalMatchConditions(left, right RuleMatchConditions) bool {
	return sameStringSet(left.AlarmIdentifiers, right.AlarmIdentifiers) &&
		sameUUIDSet(left.DeviceIDs, right.DeviceIDs) && len(left.Severities) == 0 && len(right.Severities) == 0 &&
		len(left.Carriers) == 0 && len(right.Carriers) == 0 &&
		len(left.Technologies) == 0 && len(right.Technologies) == 0
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	values := make(map[string]int, len(left))
	for _, value := range left {
		values[value]++
	}
	for _, value := range right {
		values[value]--
	}
	for _, count := range values {
		if count != 0 {
			return false
		}
	}
	return true
}

func sameUUIDSet(left, right []uuid.UUID) bool {
	if len(left) != len(right) {
		return false
	}
	values := make(map[uuid.UUID]int, len(left))
	for _, value := range left {
		values[value]++
	}
	for _, value := range right {
		values[value]--
	}
	for _, count := range values {
		if count != 0 {
			return false
		}
	}
	return true
}

func overlapRuleIDs(findings []MigrationOverlapFinding) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(findings))
	for _, finding := range findings {
		ids = append(ids, finding.RuleID)
	}
	return ids
}

func migrationReason(id uuid.UUID) string { return migrationReasonPrefix + " " + id.String() }

func normalizedEmails(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func cloneStrings(values []string) []string     { return append([]string(nil), values...) }
func cloneUUIDs(values []uuid.UUID) []uuid.UUID { return append([]uuid.UUID(nil), values...) }
