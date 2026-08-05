package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestRuleScopeAuthorizer_RequiresExplicitScopeAndSameGrantDimensions(t *testing.T) {
	userID, deviceID, groupID := uuid.New(), uuid.New(), uuid.New()
	permissions := &recipientPermissionStub{grants: map[uuid.UUID][]model.DeviceVisibilityGrant{
		userID: {
			{GroupIDs: []uuid.UUID{groupID}, Technologies: []model.Technology{model.TechLTE}},
			{GroupIDs: []uuid.UUID{uuid.New()}, Technologies: []model.Technology{model.TechNR}},
		},
	}}
	authorizer := NewRuleScopeAuthorizer(permissions, recipientGroupsStub{groups: []uuid.UUID{groupID}})

	err := authorizer.Authorize(context.Background(), userID, false, RuleDraftInput{})
	require.ErrorIs(t, err, ErrRuleScopeDenied)
	err = authorizer.Authorize(context.Background(), userID, false, RuleDraftInput{MatchConditions: RuleMatchConditions{
		DeviceIDs: []uuid.UUID{deviceID}, Technologies: []model.Technology{model.TechNR},
	}})
	require.ErrorIs(t, err, ErrRuleScopeDenied,
		"group permission and technology permission from different grants must not be combined")
	err = authorizer.Authorize(context.Background(), userID, false, RuleDraftInput{MatchConditions: RuleMatchConditions{
		DeviceIDs: []uuid.UUID{deviceID}, Technologies: []model.Technology{model.TechLTE},
	}})
	require.NoError(t, err)
}

func TestRuleScopeAuthorizer_SuperAdminCanManageBroadRule(t *testing.T) {
	authorizer := NewRuleScopeAuthorizer(nil, nil)
	require.NoError(t, authorizer.Authorize(context.Background(), uuid.New(), true, RuleDraftInput{}))
}
