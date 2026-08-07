package notification

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

var (
	ErrRevisionMismatch   = errors.New("notification resource revision does not match")
	ErrDraftUnavailable   = errors.New("notification resource has no draft version")
	ErrVersionUnpublished = errors.New("notification version is not published")
	ErrRuleIncomplete     = errors.New("notification rule has incomplete recipients or channels")
	ErrChannelDisabled    = errors.New("notification rule channel is disabled")
)

type RuleMatchConditions struct {
	AlarmIdentifiers []string              `json:"alarm_identifiers,omitempty"`
	Severities       []model.AlarmSeverity `json:"severities,omitempty"`
	DeviceIDs        []uuid.UUID           `json:"device_ids,omitempty"`
	DeviceGroupIDs   []uuid.UUID           `json:"device_group_ids,omitempty"`
	Carriers         []model.CarrierCode   `json:"carriers,omitempty"`
	Technologies     []model.Technology    `json:"technologies,omitempty"`
}

type NotificationRule struct {
	ID                        uuid.UUID                `json:"id"`
	Name                      string                   `json:"name"`
	Revision                  int64                    `json:"revision"`
	CurrentDraftVersionID     *uuid.UUID               `json:"current_draft_version_id,omitempty"`
	CurrentPublishedVersionID *uuid.UUID               `json:"current_published_version_id,omitempty"`
	CurrentEnabledVersionID   *uuid.UUID               `json:"current_enabled_version_id,omitempty"`
	Priority                  int                      `json:"priority"`
	Archived                  bool                     `json:"archived"`
	CreatedBy                 string                   `json:"created_by"`
	CreatedAt                 time.Time                `json:"created_at"`
	UpdatedAt                 time.Time                `json:"updated_at"`
	Draft                     *NotificationRuleVersion `json:"draft,omitempty"`
	Published                 *NotificationRuleVersion `json:"published,omitempty"`
	Enabled                   *NotificationRuleVersion `json:"enabled,omitempty"`
}

type NotificationRuleVersion struct {
	ID              uuid.UUID           `json:"id"`
	RuleID          uuid.UUID           `json:"rule_id"`
	VersionNo       int64               `json:"version_no"`
	MatchConditions RuleMatchConditions `json:"match_conditions"`
	Policy          json.RawMessage     `json:"policy"`
	CreatedBy       string              `json:"created_by"`
	ChangeReason    string              `json:"change_reason"`
	CreatedAt       time.Time           `json:"created_at"`
	PublishedAt     *time.Time          `json:"published_at,omitempty"`
	Recipients      []RuleRecipient     `json:"recipients"`
	Channels        []RuleChannel       `json:"channels"`
}

type RuleRecipient struct {
	ID                uuid.UUID `json:"id"`
	TargetType        string    `json:"target_type"`
	TargetID          string    `json:"target_id,omitempty"`
	AddressConfigured bool      `json:"address_configured"`
	ChannelLimit      []string  `json:"channel_limit"`
}

type RuleRecipientInput struct {
	TargetType           string   `json:"target_type"`
	TargetID             string   `json:"target_id,omitempty"`
	AddressCiphertext    []byte   `json:"address_ciphertext,omitempty"`
	AddressKeyVersion    int      `json:"address_key_version,omitempty"`
	RecipientFingerprint []byte   `json:"recipient_fingerprint,omitempty"`
	ChannelLimit         []string `json:"channel_limit"`
}

type RuleChannel struct {
	ID                         uuid.UUID       `json:"id"`
	Channel                    string          `json:"channel"`
	ChannelConfigID            uuid.UUID       `json:"channel_config_id"`
	RaisedTemplateVersionID    uuid.UUID       `json:"raised_template_version_id"`
	EscalatedTemplateVersionID *uuid.UUID      `json:"escalated_template_version_id,omitempty"`
	ClearedTemplateVersionID   *uuid.UUID      `json:"cleared_template_version_id,omitempty"`
	Policy                     json.RawMessage `json:"policy"`
}

type RuleChannelInput struct {
	Channel                    string          `json:"channel"`
	ChannelConfigID            uuid.UUID       `json:"channel_config_id"`
	RaisedTemplateVersionID    uuid.UUID       `json:"raised_template_version_id"`
	EscalatedTemplateVersionID *uuid.UUID      `json:"escalated_template_version_id,omitempty"`
	ClearedTemplateVersionID   *uuid.UUID      `json:"cleared_template_version_id,omitempty"`
	Policy                     json.RawMessage `json:"policy"`
}

type RuleDraftInput struct {
	Name            string               `json:"name"`
	Priority        int                  `json:"priority"`
	MatchConditions RuleMatchConditions  `json:"match_conditions"`
	Policy          json.RawMessage      `json:"policy"`
	ChangeReason    string               `json:"change_reason"`
	Recipients      []RuleRecipientInput `json:"recipients"`
	Channels        []RuleChannelInput   `json:"channels"`
}

type RuleCandidate struct {
	ID         uuid.UUID
	Priority   int
	Conditions RuleMatchConditions
}

type RuleMatch struct {
	RuleID      uuid.UUID `json:"rule_id"`
	Priority    int       `json:"priority"`
	Specificity int       `json:"specificity"`
}

type ResolvedRecipient struct {
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Channel     string     `json:"channel"`
	Ciphertext  []byte     `json:"-"`
	KeyVersion  int        `json:"-"`
	Fingerprint []byte     `json:"-"`
	Locale      string     `json:"locale"`
}

type RuleRecipientCandidate struct {
	Match     RuleMatch
	Recipient ResolvedRecipient
}

type RulePreview struct {
	Matched      bool        `json:"matched"`
	OrderedRules []RuleMatch `json:"ordered_rules"`
}

func matchableSnapshot(payload event.AlarmLifecyclePayload) event.AlarmLifecycleSnapshot {
	return payload.Snapshot
}
