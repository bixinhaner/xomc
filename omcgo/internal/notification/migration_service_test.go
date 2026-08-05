package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/alarm"
)

type migrationLegacyRepositoryStub struct {
	rules           []alarm.AlarmFilterRule
	replacedID      uuid.UUID
	replacedAt      time.Time
	replacedRule    uuid.UUID
	replacedVersion uuid.UUID
	replacedBy      string
	replaceCalls    int
	err             error
}

func (s *migrationLegacyRepositoryStub) ListEnabled(context.Context) ([]alarm.AlarmFilterRule, error) {
	return append([]alarm.AlarmFilterRule(nil), s.rules...), s.err
}

func (s *migrationLegacyRepositoryStub) ReplaceNotifyEmailWithBarrier(
	_ context.Context, id uuid.UUID, updatedAt time.Time,
	notificationRuleID uuid.UUID, enabledVersionID uuid.UUID, actor string,
) error {
	s.replaceCalls++
	s.replacedID, s.replacedAt, s.replacedRule, s.replacedVersion, s.replacedBy =
		id, updatedAt, notificationRuleID, enabledVersionID, actor
	return s.err
}

type migrationRuleRepositoryStub struct {
	rule        *NotificationRule
	createInput RuleDraftInput
	createActor string
	err         error
}

func (s *migrationRuleRepositoryStub) Get(context.Context, uuid.UUID) (*NotificationRule, error) {
	return s.rule, s.err
}

func (s *migrationRuleRepositoryStub) Create(
	_ context.Context, input RuleDraftInput, actor string,
) (*NotificationRule, error) {
	s.createInput, s.createActor = input, actor
	return s.rule, s.err
}

type migrationProtectorStub struct{}

func (migrationProtectorStub) Protect(_ string, address string) ([]byte, int, []byte, error) {
	return []byte("cipher:" + address), 1, []byte("fingerprint:" + address), nil
}

func TestMigrationPreviewReportsFirstMatchOverlapAndCompatibility(t *testing.T) {
	legacyID, overlapID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	repository := &migrationLegacyRepositoryStub{rules: []alarm.AlarmFilterRule{
		{
			ID: overlapID, Name: "lower-ignore", Action: alarm.FilterActionIgnore, Priority: 20,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"}, CreatedAt: now.Add(time.Minute),
		},
		{
			ID: legacyID, Name: "legacy-email", Action: alarm.FilterActionNotifyEmail, Priority: 10,
			AlarmSources: []string{"Device"}, AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			EmailRecipients: []string{" NOC@example.com ", "noc@example.com"}, CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: uuid.New(), Name: "disjoint-clear", Action: alarm.FilterActionAutoClear, Priority: 30,
			AlarmIdentifiers: []string{"CPU_HIGH"}, CreatedAt: now.Add(2 * time.Minute),
		},
	}}
	service := NewMigrationService(repository, &migrationRuleRepositoryStub{}, migrationProtectorStub{})

	previews, err := service.Preview(context.Background())

	require.NoError(t, err)
	require.Len(t, previews, 1)
	preview := previews[0]
	require.Equal(t, legacyID, preview.LegacyRuleID)
	require.Equal(t, 10, preview.LegacyPriority)
	require.Equal(t, 1, preview.CandidatePriority)
	require.Equal(t, []string{"noc@example.com"}, preview.RecipientEmails)
	require.Equal(t, []string{"alarm_sources"}, preview.UnsupportedScopeFields)
	require.Equal(t, []MigrationOverlapFinding{{
		RuleID: overlapID, Name: "lower-ignore", Action: alarm.FilterActionIgnore, Priority: 20,
	}}, preview.OverlapFindings)
}

func TestMigrationCreateCandidateIsDraftOnlyAndProtectsEachRecipient(t *testing.T) {
	legacyID := uuid.New()
	legacy := &migrationLegacyRepositoryStub{rules: []alarm.AlarmFilterRule{{
		ID: legacyID, Name: "legacy-email", Action: alarm.FilterActionNotifyEmail, Priority: 0,
		AlarmIdentifiers: []string{"DEVICE_OFFLINE"}, EmailRecipients: []string{"noc@example.com"},
	}}}
	created := &NotificationRule{ID: uuid.New(), CurrentEnabledVersionID: nil, Enabled: nil}
	rules := &migrationRuleRepositoryStub{rule: created}
	service := NewMigrationService(legacy, rules, migrationProtectorStub{})

	result, err := service.CreateDisabledCandidate(context.Background(), legacyID, MigrationCandidateBinding{
		ChannelConfigID: uuid.New(), RaisedTemplateVersionID: uuid.New(),
	}, "operator")

	require.NoError(t, err)
	require.Same(t, created, result)
	require.Equal(t, "operator", rules.createActor)
	require.Equal(t, "legacy-email-"+legacyID.String(), rules.createInput.Name)
	require.Equal(t, 1, rules.createInput.Priority)
	require.Equal(t, migrationReason(legacyID), rules.createInput.ChangeReason)
	require.Equal(t, []string{"DEVICE_OFFLINE"}, rules.createInput.MatchConditions.AlarmIdentifiers)
	require.Len(t, rules.createInput.Recipients, 1)
	require.Equal(t, []byte("cipher:noc@example.com"), rules.createInput.Recipients[0].AddressCiphertext)
	require.Len(t, rules.createInput.Channels, 1)
	require.Nil(t, result.CurrentEnabledVersionID)
}

func TestMigrationCutoverRequiresEveryGate(t *testing.T) {
	legacyID, overlapID, notificationRuleID, versionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	updatedAt := time.Now().UTC()
	legacy := &migrationLegacyRepositoryStub{rules: []alarm.AlarmFilterRule{
		{
			ID: legacyID, Name: "legacy-email", Action: alarm.FilterActionNotifyEmail, Priority: 10,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"}, EmailRecipients: []string{"noc@example.com"},
			UpdatedAt: updatedAt,
		},
		{
			ID: overlapID, Name: "lower-clear", Action: alarm.FilterActionAutoClear, Priority: 20,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
		},
	}}
	enabledVersion := &NotificationRuleVersion{
		ID: versionID, MatchConditions: RuleMatchConditions{AlarmIdentifiers: []string{"DEVICE_OFFLINE"}},
		ChangeReason: migrationReason(legacyID), Recipients: []RuleRecipient{{
			TargetType: RecipientTargetFixedContact, ChannelLimit: []string{TemplateChannelEmail},
		}},
		Channels: []RuleChannel{{Channel: TemplateChannelEmail}},
	}
	candidate := &NotificationRule{
		ID: notificationRuleID, Name: "legacy-email-" + legacyID.String(), Priority: 1,
		CurrentEnabledVersionID: &versionID, Enabled: enabledVersion,
	}
	rules := &migrationRuleRepositoryStub{rule: candidate}
	service := NewMigrationService(legacy, rules, migrationProtectorStub{})
	valid := MigrationCutoverInput{
		LegacyRuleID: legacyID, NotificationRuleID: notificationRuleID,
		ExpectedLegacyUpdatedAt: updatedAt, ConfirmedOverlapRuleIDs: []uuid.UUID{overlapID},
		ShadowComparedEvents: 20,
		LegacyShadowMatches:  7, NotificationShadowMatches: 7, Actor: "operator",
	}

	tests := []struct {
		name string
		edit func(*MigrationCutoverInput)
		err  error
	}{
		{"overlap", func(input *MigrationCutoverInput) { input.ConfirmedOverlapRuleIDs = nil }, ErrMigrationOverlapUnconfirmed},
		{"shadow sample", func(input *MigrationCutoverInput) { input.ShadowComparedEvents = 0 }, ErrMigrationShadowMismatch},
		{"shadow mismatch", func(input *MigrationCutoverInput) { input.NotificationShadowMatches++ }, ErrMigrationShadowMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.edit(&input)
			err := service.Cutover(context.Background(), input)
			require.ErrorIs(t, err, test.err)
		})
	}
	require.Zero(t, legacy.replaceCalls)

	candidate.CurrentEnabledVersionID = nil
	err := service.Cutover(context.Background(), valid)
	require.ErrorIs(t, err, ErrMigrationCandidateNotEnabled)
	candidate.CurrentEnabledVersionID = &versionID

	candidate.Enabled.MatchConditions.AlarmIdentifiers = []string{"CPU_HIGH"}
	err = service.Cutover(context.Background(), valid)
	require.ErrorIs(t, err, ErrMigrationCandidateMismatch)
	require.Zero(t, legacy.replaceCalls)
}

func TestMigrationCutoverReplacesOnlyTheSelectedLegacyEmailRule(t *testing.T) {
	legacyID, notificationRuleID, versionID := uuid.New(), uuid.New(), uuid.New()
	updatedAt := time.Now().UTC()
	legacy := &migrationLegacyRepositoryStub{rules: []alarm.AlarmFilterRule{{
		ID: legacyID, Name: "legacy-email", Action: alarm.FilterActionNotifyEmail, Priority: 10,
		AlarmIdentifiers: []string{"DEVICE_OFFLINE"}, EmailRecipients: []string{"noc@example.com"},
		UpdatedAt: updatedAt,
	}}}
	enabledVersion := &NotificationRuleVersion{
		ID: versionID, MatchConditions: RuleMatchConditions{AlarmIdentifiers: []string{"DEVICE_OFFLINE"}},
		ChangeReason: migrationReason(legacyID), Recipients: []RuleRecipient{{
			TargetType: RecipientTargetFixedContact, ChannelLimit: []string{TemplateChannelEmail},
		}},
		Channels: []RuleChannel{{Channel: TemplateChannelEmail}},
	}
	rules := &migrationRuleRepositoryStub{rule: &NotificationRule{
		ID: notificationRuleID, Name: "legacy-email-" + legacyID.String(), Priority: 1,
		CurrentEnabledVersionID: &versionID, Enabled: enabledVersion,
	}}
	service := NewMigrationService(legacy, rules, migrationProtectorStub{})

	err := service.Cutover(context.Background(), MigrationCutoverInput{
		LegacyRuleID: legacyID, NotificationRuleID: notificationRuleID,
		ExpectedLegacyUpdatedAt: updatedAt,
		ShadowComparedEvents:    10, LegacyShadowMatches: 3, NotificationShadowMatches: 3,
		Actor: "operator",
	})

	require.NoError(t, err)
	require.Equal(t, 1, legacy.replaceCalls)
	require.Equal(t, legacyID, legacy.replacedID)
	require.Equal(t, updatedAt, legacy.replacedAt)
	require.Equal(t, notificationRuleID, legacy.replacedRule)
	require.Equal(t, versionID, legacy.replacedVersion)
	require.Equal(t, "operator", legacy.replacedBy)
}
