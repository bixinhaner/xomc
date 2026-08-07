package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type RuleService struct{ repository RuleRepository }

type publishedRuleRepository interface {
	SavePublished(context.Context, uuid.UUID, int64, RuleDraftInput, bool, string) (*NotificationRule, error)
}

func NewRuleService(repository RuleRepository) *RuleService {
	return &RuleService{repository: repository}
}

func (s *RuleService) List(ctx context.Context) ([]NotificationRule, error) {
	rules, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification rules: %w", err)
	}
	return rules, nil
}

func (s *RuleService) Get(ctx context.Context, id uuid.UUID) (*NotificationRule, error) {
	rule, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Create(ctx context.Context, input RuleDraftInput, actor string) (*NotificationRule, error) {
	input, err := normalizeRuleDraft(input)
	if err != nil {
		return nil, err
	}
	rule, err := s.repository.Create(ctx, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("create notification rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) UpdateDraft(ctx context.Context, id uuid.UUID, revision int64, input RuleDraftInput, actor string) (*NotificationRule, error) {
	input, err := normalizeRuleDraft(input)
	if err != nil {
		return nil, err
	}
	rule, err := s.repository.UpdateDraft(ctx, id, revision, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("update notification rule draft: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Publish(ctx context.Context, id uuid.UUID, revision int64, actor string) (*NotificationRule, error) {
	rule, err := s.repository.Publish(ctx, id, revision, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("publish notification rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Enable(ctx context.Context, id uuid.UUID, revision int64, versionID uuid.UUID) (*NotificationRule, error) {
	if versionID == uuid.Nil {
		return nil, commonerrors.ErrInvalidInput
	}
	rule, err := s.repository.Enable(ctx, id, revision, versionID)
	if err != nil {
		return nil, fmt.Errorf("enable notification rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Disable(ctx context.Context, id uuid.UUID, revision int64) (*NotificationRule, error) {
	rule, err := s.repository.Disable(ctx, id, revision)
	if err != nil {
		return nil, fmt.Errorf("disable notification rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Archive(ctx context.Context, id uuid.UUID, revision int64) (*NotificationRule, error) {
	rule, err := s.repository.Archive(ctx, id, revision)
	if err != nil {
		return nil, fmt.Errorf("archive notification rule: %w", err)
	}
	return rule, nil
}

// SavePublished is the narrow atomic write used by the alarm-email business
// adapter. Generic rule draft/publish APIs keep their independent lifecycle,
// while one alarm-email form submission commits its version and enabled state
// together so operators never observe a partially applied setting.
func (s *RuleService) SavePublished(
	ctx context.Context,
	id uuid.UUID,
	revision int64,
	input RuleDraftInput,
	enabled bool,
	actor string,
) (*NotificationRule, error) {
	input, err := normalizeRuleDraft(input)
	if err != nil {
		return nil, err
	}
	repository, ok := s.repository.(publishedRuleRepository)
	if !ok {
		return nil, fmt.Errorf("save published notification rule: repository does not support atomic publication")
	}
	rule, err := repository.SavePublished(ctx, id, revision, input, enabled, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("save published notification rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Preview(payload event.AlarmLifecyclePayload, candidates []RuleCandidate) RulePreview {
	matches := MatchRules(matchableSnapshot(payload), candidates)
	return RulePreview{Matched: len(matches) > 0, OrderedRules: matches}
}

func normalizeRuleDraft(input RuleDraftInput) (RuleDraftInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ChangeReason = strings.TrimSpace(input.ChangeReason)
	if input.Name == "" || len(input.Name) > 128 {
		return input, commonerrors.ErrInvalidInput
	}
	if input.Priority == 0 {
		input.Priority = 100
	}
	if input.Priority < 1 {
		return input, commonerrors.ErrInvalidInput
	}
	if len(input.Policy) == 0 {
		input.Policy = json.RawMessage(`{}`)
	}
	if !json.Valid(input.Policy) {
		return input, commonerrors.ErrInvalidInput
	}
	input.MatchConditions.AlarmIdentifiers = normalizeVariables(input.MatchConditions.AlarmIdentifiers)
	for _, severity := range input.MatchConditions.Severities {
		if severity != model.AlarmCritical && severity != model.AlarmMajor && severity != model.AlarmMinor && severity != model.AlarmWarning {
			return input, commonerrors.ErrInvalidInput
		}
	}
	for _, deviceID := range input.MatchConditions.DeviceIDs {
		if deviceID == uuid.Nil {
			return input, commonerrors.ErrInvalidInput
		}
	}
	for _, groupID := range input.MatchConditions.DeviceGroupIDs {
		if groupID == uuid.Nil {
			return input, commonerrors.ErrInvalidInput
		}
	}
	for _, carrier := range input.MatchConditions.Carriers {
		if !carrier.IsValid() {
			return input, commonerrors.ErrInvalidInput
		}
	}
	for _, technology := range input.MatchConditions.Technologies {
		if !technology.IsValid() {
			return input, commonerrors.ErrInvalidInput
		}
	}
	for i := range input.Recipients {
		recipient := &input.Recipients[i]
		recipient.TargetID = strings.TrimSpace(recipient.TargetID)
		recipient.ChannelLimit = normalizeVariables(recipient.ChannelLimit)
		if len(recipient.ChannelLimit) == 0 || !validNotificationChannels(recipient.ChannelLimit) {
			return input, commonerrors.ErrInvalidInput
		}
		switch recipient.TargetType {
		case RecipientTargetUser, RecipientTargetRole, RecipientTargetContactGroup:
			if _, err := uuid.Parse(recipient.TargetID); err != nil {
				return input, commonerrors.ErrInvalidInput
			}
		case RecipientTargetFixedContact:
			if len(recipient.AddressCiphertext) == 0 || recipient.AddressKeyVersion < 1 || len(recipient.RecipientFingerprint) == 0 {
				return input, commonerrors.ErrInvalidInput
			}
		default:
			return input, commonerrors.ErrInvalidInput
		}
	}
	seenChannels := make(map[string]struct{}, len(input.Channels))
	for i := range input.Channels {
		channel := &input.Channels[i]
		if !validNotificationChannel(channel.Channel) || channel.ChannelConfigID == uuid.Nil || channel.RaisedTemplateVersionID == uuid.Nil {
			return input, commonerrors.ErrInvalidInput
		}
		key := channel.Channel + ":" + channel.ChannelConfigID.String()
		if _, exists := seenChannels[key]; exists {
			return input, commonerrors.ErrInvalidInput
		}
		seenChannels[key] = struct{}{}
		if len(channel.Policy) == 0 {
			channel.Policy = json.RawMessage(`{}`)
		}
		if !json.Valid(channel.Policy) {
			return input, commonerrors.ErrInvalidInput
		}
	}
	return input, nil
}

func validNotificationChannels(channels []string) bool {
	for _, channel := range channels {
		if !validNotificationChannel(channel) {
			return false
		}
	}
	return true
}

func validNotificationChannel(channel string) bool {
	switch channel {
	case TemplateChannelEmail, "sms_kafka", "sms_direct":
		return true
	default:
		return false
	}
}

func normalizeActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}
