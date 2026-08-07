package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestAlarmEmailSettingsService_BuildDraftUsesOnlyOldOMCBusinessFields(t *testing.T) {
	protector, err := NewAESGCMRecipientProtector(make([]byte, 32))
	require.NoError(t, err)
	service := NewAlarmEmailSettingsService(nil, nil, protector, nil)
	deviceID, groupID := uuid.New(), uuid.New()

	draft, err := service.buildDraft(AlarmEmailSettingInput{
		Name: "Core network alarms", Enabled: true,
		AlarmIdentifiers: []string{"11184"}, Severities: []model.AlarmSeverity{model.AlarmCritical},
		DeviceIDs: []uuid.UUID{deviceID}, DeviceGroupIDs: []uuid.UUID{groupID},
		Technologies:    []model.Technology{model.TechNR},
		IntervalMinutes: 30, ToleranceDurationMinutes: 10,
		Recipients: []string{"NOC@EXAMPLE.COM; noc@example.com"}, IncludeDefaultRecipients: true,
	}, "test")
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{groupID}, draft.MatchConditions.DeviceGroupIDs)
	require.Len(t, draft.Recipients, 2, "one encrypted address plus the default recipient group")
	require.Equal(t, RecipientTargetFixedContact, draft.Recipients[0].TargetType)
	require.Equal(t, RecipientTargetContactGroup, draft.Recipients[1].TargetType)
	require.Equal(t, defaultAlarmRecipientGroupID.String(), draft.Recipients[1].TargetID)
	require.Len(t, draft.Channels, 1)
	require.Equal(t, defaultEmailChannelID, draft.Channels[0].ChannelConfigID)
	require.Equal(t, builtinAlarmEmailTemplateID, draft.Channels[0].RaisedTemplateVersionID)
	require.Equal(t, &builtinAlarmEmailTemplateID, draft.Channels[0].ClearedTemplateVersionID)

	var policy NotificationPolicy
	require.NoError(t, json.Unmarshal(draft.Policy, &policy))
	require.Equal(t, DeliveryModeDigest, policy.DeliveryMode)
	require.Equal(t, 30*60, policy.AggregationWindowSeconds)
	require.Equal(t, 10*60, policy.MinimumActiveSeconds)
	require.Zero(t, policy.RepeatIntervalSeconds)
	require.Zero(t, policy.MaxRepeatCount)
}

func TestAlarmEmailSettingsService_RejectsUnsupportedIntervalsAndMissingRecipients(t *testing.T) {
	protector, err := NewAESGCMRecipientProtector(make([]byte, 32))
	require.NoError(t, err)
	service := NewAlarmEmailSettingsService(nil, nil, protector, nil)

	_, err = service.buildDraft(AlarmEmailSettingInput{Name: "invalid", IntervalMinutes: 5, Recipients: []string{"noc@example.com"}}, "test")
	require.Error(t, err)
	_, err = service.buildDraft(AlarmEmailSettingInput{Name: "invalid", IntervalMinutes: 10}, "test")
	require.Error(t, err)
	_, err = service.buildDraft(AlarmEmailSettingInput{Name: "invalid", IntervalMinutes: 10, Recipients: []string{"not-an-email"}}, "test")
	require.Error(t, err)
}

func TestAlarmEmailSettingsService_ListHydratesRuleVersions(t *testing.T) {
	protector, err := NewAESGCMRecipientProtector(make([]byte, 32))
	require.NoError(t, err)

	ruleID, versionID := uuid.New(), uuid.New()
	rules := &alarmEmailListRuleStub{
		listed: []NotificationRule{{ID: ruleID, Name: "shallow header"}},
		items: map[uuid.UUID]*NotificationRule{
			ruleID: {
				ID: ruleID, Name: "hydrated alarm email", Revision: 1,
				Published: &NotificationRuleVersion{
					ID: versionID, RuleID: ruleID, Policy: json.RawMessage(`{}`),
					Channels: []RuleChannel{{
						Channel: TemplateChannelEmail, ChannelConfigID: defaultEmailChannelID,
						RaisedTemplateVersionID: builtinAlarmEmailTemplateID,
					}},
				},
			},
		},
	}
	service := NewAlarmEmailSettingsService(rules, emptyAlarmEmailRecipientReader{}, protector, nil)

	items, err := service.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "hydrated alarm email", items[0].Name)
	require.Equal(t, []uuid.UUID{ruleID}, rules.got)
}

type alarmEmailListRuleStub struct {
	alarmEmailRuleService
	listed []NotificationRule
	items  map[uuid.UUID]*NotificationRule
	got    []uuid.UUID
}

func (s *alarmEmailListRuleStub) List(context.Context) ([]NotificationRule, error) {
	return s.listed, nil
}

func (s *alarmEmailListRuleStub) Get(_ context.Context, id uuid.UUID) (*NotificationRule, error) {
	s.got = append(s.got, id)
	return s.items[id], nil
}

type emptyAlarmEmailRecipientReader struct{}

func (emptyAlarmEmailRecipientReader) ListRuleRecipientTargets(context.Context, uuid.UUID) ([]RecipientTarget, error) {
	return nil, nil
}

func TestAlarmEmailSettingsHandler_AuthorizesExistingSettingScope(t *testing.T) {
	userID, allowedGroup, deniedGroup := uuid.New(), uuid.New(), uuid.New()
	authorizer := NewRuleScopeAuthorizer(&recipientPermissionStub{grants: map[uuid.UUID][]model.DeviceVisibilityGrant{
		userID: {{GroupIDs: []uuid.UUID{allowedGroup}, Technologies: []model.Technology{model.TechLTE}}},
	}}, recipientGroupsStub{groups: []uuid.UUID{allowedGroup}})
	handler := NewAlarmEmailSettingsHandler(nil, authorizer)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/alarm-email-settings", nil).WithContext(context.Background())
	ctx.Set(admin.CtxKeyUserID, userID)
	ctx.Set(admin.CtxKeyIsSuperAdmin, false)

	err := handler.authorizeSetting(ctx, &AlarmEmailSetting{
		DeviceGroupIDs: []uuid.UUID{allowedGroup},
		Technologies:   []model.Technology{model.TechLTE},
	})
	require.NoError(t, err)

	err = handler.authorizeSetting(ctx, &AlarmEmailSetting{
		DeviceGroupIDs: []uuid.UUID{deniedGroup},
		Technologies:   []model.Technology{model.TechLTE},
	})
	require.ErrorIs(t, err, ErrRuleScopeDenied)
}

func TestAlarmEmailSettingsHandler_DefaultRecipientsRequireSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nonAdminRecorder := httptest.NewRecorder()
	nonAdminContext, _ := gin.CreateTestContext(nonAdminRecorder)
	nonAdminContext.Set(admin.CtxKeyIsSuperAdmin, false)
	require.False(t, requireAlarmEmailDefaultsAdmin(nonAdminContext))
	require.Equal(t, http.StatusForbidden, nonAdminRecorder.Code)

	adminContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	adminContext.Set(admin.CtxKeyIsSuperAdmin, true)
	require.True(t, requireAlarmEmailDefaultsAdmin(adminContext))
}
