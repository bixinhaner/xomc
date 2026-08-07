package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var (
	defaultEmailChannelID        = uuid.MustParse("aaaa000c-1000-0000-0000-000000000001")
	builtinAlarmEmailTemplateID  = uuid.MustParse("aaaa000c-2100-0000-0000-000000000001")
	defaultAlarmRecipientGroupID = uuid.MustParse("aaaa000c-3000-0000-0000-000000000001")
)

var allowedAlarmEmailMinutes = map[int]struct{}{0: {}, 10: {}, 30: {}, 60: {}}

// AlarmEmailSetting is the business-facing projection of an internal immutable
// notification rule. Template, channel and retry internals are intentionally
// absent because the old OMC workflow does not expose them to operators.
type AlarmEmailSetting struct {
	ID                       uuid.UUID             `json:"id"`
	Name                     string                `json:"name"`
	Revision                 int64                 `json:"revision"`
	Enabled                  bool                  `json:"enabled"`
	AlarmIdentifiers         []string              `json:"alarm_identifiers"`
	Severities               []model.AlarmSeverity `json:"severities"`
	DeviceIDs                []uuid.UUID           `json:"device_ids"`
	DeviceGroupIDs           []uuid.UUID           `json:"device_group_ids"`
	Technologies             []model.Technology    `json:"technologies"`
	IntervalMinutes          int                   `json:"interval_minutes"`
	ToleranceDurationMinutes int                   `json:"tolerance_duration_minutes"`
	Recipients               []string              `json:"recipients"`
	IncludeDefaultRecipients bool                  `json:"include_default_recipients"`
	UpdatedAt                time.Time             `json:"updated_at"`
}

type AlarmEmailSettingInput struct {
	Name                     string                `json:"name"`
	Enabled                  bool                  `json:"enabled"`
	AlarmIdentifiers         []string              `json:"alarm_identifiers"`
	Severities               []model.AlarmSeverity `json:"severities"`
	DeviceIDs                []uuid.UUID           `json:"device_ids"`
	DeviceGroupIDs           []uuid.UUID           `json:"device_group_ids"`
	Technologies             []model.Technology    `json:"technologies"`
	IntervalMinutes          int                   `json:"interval_minutes"`
	ToleranceDurationMinutes int                   `json:"tolerance_duration_minutes"`
	Recipients               []string              `json:"recipients"`
	IncludeDefaultRecipients bool                  `json:"include_default_recipients"`
}

// AlarmEmailDefaults is the global recipient list used when a setting enables
// "include default recipients". It is intentionally part of the alarm email
// workflow instead of exposing the generic contact-group model to operators.
type AlarmEmailDefaults struct {
	Recipients []string  `json:"recipients"`
	Revision   int64     `json:"revision"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AlarmEmailDefaultsInput struct {
	Recipients []string `json:"recipients"`
}

type alarmEmailRuleService interface {
	List(context.Context) ([]NotificationRule, error)
	Get(context.Context, uuid.UUID) (*NotificationRule, error)
	Create(context.Context, RuleDraftInput, string) (*NotificationRule, error)
	UpdateDraft(context.Context, uuid.UUID, int64, RuleDraftInput, string) (*NotificationRule, error)
	Publish(context.Context, uuid.UUID, int64, string) (*NotificationRule, error)
	Enable(context.Context, uuid.UUID, int64, uuid.UUID) (*NotificationRule, error)
	Disable(context.Context, uuid.UUID, int64) (*NotificationRule, error)
	Archive(context.Context, uuid.UUID, int64) (*NotificationRule, error)
	SavePublished(context.Context, uuid.UUID, int64, RuleDraftInput, bool, string) (*NotificationRule, error)
}

type alarmEmailRecipientMaterialReader interface {
	ListRuleRecipientTargets(context.Context, uuid.UUID) ([]RecipientTarget, error)
}

type alarmEmailRecipientProtector interface {
	RecipientProtector
	Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error)
}

type alarmEmailDefaultRecipientRepository interface {
	Get(context.Context, uuid.UUID) (*ContactGroup, error)
	Update(context.Context, uuid.UUID, int64, ContactGroupInput, string) (*ContactGroup, error)
	ListMembers(context.Context, uuid.UUID) ([]RecipientTarget, error)
}

type AlarmEmailSettingsService struct {
	rules      alarmEmailRuleService
	recipients alarmEmailRecipientMaterialReader
	protector  alarmEmailRecipientProtector
	defaults   alarmEmailDefaultRecipientRepository
}

func NewAlarmEmailSettingsService(
	rules alarmEmailRuleService,
	recipients alarmEmailRecipientMaterialReader,
	protector alarmEmailRecipientProtector,
	defaults alarmEmailDefaultRecipientRepository,
) *AlarmEmailSettingsService {
	return &AlarmEmailSettingsService{rules: rules, recipients: recipients, protector: protector, defaults: defaults}
}

func (s *AlarmEmailSettingsService) List(ctx context.Context) ([]AlarmEmailSetting, error) {
	if s == nil || s.rules == nil || s.recipients == nil || s.protector == nil {
		return nil, fmt.Errorf("list alarm email settings: dependencies are required")
	}
	rules, err := s.rules.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list alarm email settings: %w", err)
	}
	settings := make([]AlarmEmailSetting, 0, len(rules))
	for i := range rules {
		if rules[i].Archived {
			continue
		}
		// RuleRepository.List intentionally returns only rule headers. Load the
		// complete immutable version before checking the built-in alarm-email
		// channel/template binding; otherwise every persisted setting is filtered
		// out because the shallow list item has no Published/Draft projection.
		rule, err := s.rules.Get(ctx, rules[i].ID)
		if err != nil {
			return nil, fmt.Errorf("get alarm email setting %s for list: %w", rules[i].ID, err)
		}
		if !isBuiltInAlarmEmailRule(rule) {
			continue
		}
		setting, err := s.toSetting(ctx, rule)
		if err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}
	return settings, nil
}

func (s *AlarmEmailSettingsService) Get(ctx context.Context, id uuid.UUID) (*AlarmEmailSetting, error) {
	rule, err := s.rules.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get alarm email setting: %w", err)
	}
	if !isBuiltInAlarmEmailRule(rule) {
		return nil, commonerrors.ErrNotFound
	}
	setting, err := s.toSetting(ctx, rule)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *AlarmEmailSettingsService) GetDefaults(ctx context.Context) (*AlarmEmailDefaults, error) {
	if s == nil || s.defaults == nil || s.protector == nil {
		return nil, fmt.Errorf("get alarm email default recipients: dependencies are required")
	}
	group, err := s.defaults.Get(ctx, defaultAlarmRecipientGroupID)
	if err != nil {
		return nil, fmt.Errorf("get alarm email default recipient group: %w", err)
	}
	targets, err := s.defaults.ListMembers(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("list alarm email default recipients: %w", err)
	}
	addresses, err := s.unprotectEmailTargets(targets)
	if err != nil {
		return nil, err
	}
	return &AlarmEmailDefaults{Recipients: addresses, Revision: group.Revision, UpdatedAt: group.UpdatedAt}, nil
}

func (s *AlarmEmailSettingsService) UpdateDefaults(
	ctx context.Context,
	revision int64,
	input AlarmEmailDefaultsInput,
	actor string,
) (*AlarmEmailDefaults, error) {
	if s == nil || s.defaults == nil || s.protector == nil {
		return nil, fmt.Errorf("update alarm email default recipients: dependencies are required")
	}
	addresses, err := normalizeAlarmEmailAddresses(input.Recipients)
	if err != nil {
		return nil, err
	}
	members := make([]ContactGroupMemberInput, 0, len(addresses))
	for _, address := range addresses {
		ciphertext, keyVersion, fingerprint, err := s.protector.Protect(TemplateChannelEmail, address)
		if err != nil {
			return nil, fmt.Errorf("protect alarm email default recipient: %w", err)
		}
		members = append(members, ContactGroupMemberInput{
			TargetType: RecipientTargetFixedContact, AddressCiphertext: ciphertext,
			AddressKeyVersion: keyVersion, RecipientFingerprint: fingerprint,
			ChannelLimit: []string{TemplateChannelEmail},
		})
	}
	group, err := s.defaults.Update(ctx, defaultAlarmRecipientGroupID, revision, ContactGroupInput{
		Name: "告警邮件默认收件人", Description: "告警邮件设置勾选默认收件人时使用",
		IsDefault: true, Members: members,
	}, actor)
	if err != nil {
		return nil, fmt.Errorf("update alarm email default recipients: %w", err)
	}
	return &AlarmEmailDefaults{Recipients: addresses, Revision: group.Revision, UpdatedAt: group.UpdatedAt}, nil
}

func (s *AlarmEmailSettingsService) Create(ctx context.Context, input AlarmEmailSettingInput, actor string) (*AlarmEmailSetting, error) {
	draft, err := s.buildDraft(input, "创建告警邮件设置")
	if err != nil {
		return nil, err
	}
	rule, err := s.rules.SavePublished(ctx, uuid.Nil, 0, draft, input.Enabled, actor)
	if err != nil {
		return nil, fmt.Errorf("create alarm email setting: %w", err)
	}
	setting, err := s.toSetting(ctx, rule)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *AlarmEmailSettingsService) Update(
	ctx context.Context,
	id uuid.UUID,
	revision int64,
	input AlarmEmailSettingInput,
	actor string,
) (*AlarmEmailSetting, error) {
	existing, err := s.rules.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get alarm email setting for update: %w", err)
	}
	if !isBuiltInAlarmEmailRule(existing) {
		return nil, commonerrors.ErrNotFound
	}
	draft, err := s.buildDraft(input, "更新告警邮件设置")
	if err != nil {
		return nil, err
	}
	rule, err := s.rules.SavePublished(ctx, id, revision, draft, input.Enabled, actor)
	if err != nil {
		return nil, fmt.Errorf("update alarm email setting: %w", err)
	}
	setting, err := s.toSetting(ctx, rule)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *AlarmEmailSettingsService) Archive(ctx context.Context, id uuid.UUID, revision int64) error {
	existing, err := s.rules.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get alarm email setting for archive: %w", err)
	}
	if !isBuiltInAlarmEmailRule(existing) {
		return commonerrors.ErrNotFound
	}
	if _, err := s.rules.Archive(ctx, id, revision); err != nil {
		return fmt.Errorf("archive alarm email setting: %w", err)
	}
	return nil
}

func (s *AlarmEmailSettingsService) buildDraft(input AlarmEmailSettingInput, reason string) (RuleDraftInput, error) {
	input, addresses, err := validateAlarmEmailSettingInput(input)
	if err != nil {
		return RuleDraftInput{}, err
	}
	if len(addresses) == 0 && !input.IncludeDefaultRecipients {
		return RuleDraftInput{}, commonerrors.ErrInvalidInput
	}
	recipients := make([]RuleRecipientInput, 0, len(addresses)+1)
	for _, address := range addresses {
		ciphertext, keyVersion, fingerprint, err := s.protector.Protect(TemplateChannelEmail, address)
		if err != nil {
			return RuleDraftInput{}, fmt.Errorf("protect alarm email recipient: %w", err)
		}
		recipients = append(recipients, RuleRecipientInput{
			TargetType: RecipientTargetFixedContact, AddressCiphertext: ciphertext,
			AddressKeyVersion: keyVersion, RecipientFingerprint: fingerprint,
			ChannelLimit: []string{TemplateChannelEmail},
		})
	}
	if input.IncludeDefaultRecipients {
		recipients = append(recipients, RuleRecipientInput{
			TargetType: RecipientTargetContactGroup, TargetID: defaultAlarmRecipientGroupID.String(),
			ChannelLimit: []string{TemplateChannelEmail},
		})
	}
	policy := NotificationPolicy{
		MinimumActiveSeconds: input.ToleranceDurationMinutes * 60,
		DeliveryMode:         DeliveryModeRealtime,
	}
	if input.IntervalMinutes > 0 {
		policy.DeliveryMode = DeliveryModeDigest
		policy.AggregationWindowSeconds = input.IntervalMinutes * 60
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return RuleDraftInput{}, fmt.Errorf("encode alarm email policy: %w", err)
	}
	return RuleDraftInput{
		Name: input.Name, Priority: 100,
		MatchConditions: RuleMatchConditions{
			AlarmIdentifiers: input.AlarmIdentifiers, Severities: input.Severities,
			DeviceIDs: input.DeviceIDs, DeviceGroupIDs: input.DeviceGroupIDs,
			Technologies: input.Technologies,
		},
		Policy: policyJSON, ChangeReason: reason, Recipients: recipients,
		Channels: []RuleChannelInput{{
			Channel: TemplateChannelEmail, ChannelConfigID: defaultEmailChannelID,
			RaisedTemplateVersionID:  builtinAlarmEmailTemplateID,
			ClearedTemplateVersionID: &builtinAlarmEmailTemplateID,
			Policy:                   policyJSON,
		}},
	}, nil
}

func validateAlarmEmailSettingInput(input AlarmEmailSettingInput) (AlarmEmailSettingInput, []string, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 128 {
		return input, nil, commonerrors.ErrInvalidInput
	}
	if _, ok := allowedAlarmEmailMinutes[input.IntervalMinutes]; !ok {
		return input, nil, commonerrors.ErrInvalidInput
	}
	if _, ok := allowedAlarmEmailMinutes[input.ToleranceDurationMinutes]; !ok {
		return input, nil, commonerrors.ErrInvalidInput
	}
	addresses, err := normalizeAlarmEmailAddresses(input.Recipients)
	if err != nil {
		return input, nil, err
	}
	if len(addresses) == 0 && !input.IncludeDefaultRecipients {
		return input, nil, commonerrors.ErrInvalidInput
	}
	return input, addresses, nil
}

func (s *AlarmEmailSettingsService) toSetting(ctx context.Context, rule *NotificationRule) (AlarmEmailSetting, error) {
	if rule == nil {
		return AlarmEmailSetting{}, fmt.Errorf("map alarm email setting: rule is required")
	}
	version := rule.Published
	if version == nil {
		version = rule.Draft
	}
	if version == nil {
		return AlarmEmailSetting{}, fmt.Errorf("map alarm email setting %s: version is missing", rule.ID)
	}
	var policy NotificationPolicy
	if len(version.Policy) > 0 {
		if err := json.Unmarshal(version.Policy, &policy); err != nil {
			return AlarmEmailSetting{}, fmt.Errorf("decode alarm email policy: %w", err)
		}
	}
	targets, err := s.recipients.ListRuleRecipientTargets(ctx, version.ID)
	if err != nil {
		return AlarmEmailSetting{}, fmt.Errorf("list alarm email recipients: %w", err)
	}
	addresses := make([]string, 0, len(targets))
	includeDefault := false
	for _, target := range targets {
		switch target.TargetType {
		case RecipientTargetFixedContact:
			address, err := s.protector.Unprotect(TemplateChannelEmail, target.AddressCiphertext, target.AddressKeyVersion)
			if err != nil {
				return AlarmEmailSetting{}, fmt.Errorf("decrypt alarm email recipient: %w", err)
			}
			addresses = append(addresses, address)
		case RecipientTargetContactGroup:
			includeDefault = target.TargetID == defaultAlarmRecipientGroupID.String()
		}
	}
	sort.Strings(addresses)
	intervalMinutes := 0
	if policy.DeliveryMode == DeliveryModeDigest {
		intervalMinutes = policy.AggregationWindowSeconds / 60
	}
	return AlarmEmailSetting{
		ID: rule.ID, Name: rule.Name, Revision: rule.Revision,
		Enabled:                  rule.CurrentEnabledVersionID != nil,
		AlarmIdentifiers:         append([]string(nil), version.MatchConditions.AlarmIdentifiers...),
		Severities:               append([]model.AlarmSeverity(nil), version.MatchConditions.Severities...),
		DeviceIDs:                append([]uuid.UUID(nil), version.MatchConditions.DeviceIDs...),
		DeviceGroupIDs:           append([]uuid.UUID(nil), version.MatchConditions.DeviceGroupIDs...),
		Technologies:             append([]model.Technology(nil), version.MatchConditions.Technologies...),
		IntervalMinutes:          intervalMinutes,
		ToleranceDurationMinutes: policy.MinimumActiveSeconds / 60,
		Recipients:               addresses, IncludeDefaultRecipients: includeDefault,
		UpdatedAt: rule.UpdatedAt,
	}, nil
}

func (s *AlarmEmailSettingsService) unprotectEmailTargets(targets []RecipientTarget) ([]string, error) {
	addresses := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.TargetType != RecipientTargetFixedContact {
			continue
		}
		address, err := s.protector.Unprotect(TemplateChannelEmail, target.AddressCiphertext, target.AddressKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decrypt alarm email recipient: %w", err)
		}
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses, nil
}

func isBuiltInAlarmEmailRule(rule *NotificationRule) bool {
	if rule == nil {
		return false
	}
	version := rule.Published
	if version == nil {
		version = rule.Draft
	}
	if version == nil || len(version.Channels) != 1 {
		return false
	}
	channel := version.Channels[0]
	return channel.Channel == TemplateChannelEmail &&
		channel.ChannelConfigID == defaultEmailChannelID &&
		channel.RaisedTemplateVersionID == builtinAlarmEmailTemplateID
}

func normalizeAlarmEmailAddresses(values []string) ([]string, error) {
	unique := make(map[string]struct{}, len(values))
	addresses := make([]string, 0, len(values))
	for _, raw := range values {
		for _, value := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ',' || r == '\n' }) {
			value = strings.ToLower(strings.TrimSpace(value))
			parsed, err := mail.ParseAddress(value)
			if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
				return nil, commonerrors.ErrInvalidInput
			}
			if _, exists := unique[value]; exists {
				continue
			}
			unique[value] = struct{}{}
			addresses = append(addresses, value)
		}
	}
	sort.Strings(addresses)
	return addresses, nil
}
