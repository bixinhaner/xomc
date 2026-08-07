package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
)

var ErrRuleScopeDenied = errors.New("notification rule device scope is not manageable by current user")

type RuleScopeAuthorizer struct {
	permissions  RecipientPermissionResolver
	deviceGroups RecipientDeviceGroups
}

func NewRuleScopeAuthorizer(permissions RecipientPermissionResolver, deviceGroups RecipientDeviceGroups) *RuleScopeAuthorizer {
	return &RuleScopeAuthorizer{permissions: permissions, deviceGroups: deviceGroups}
}

func (a *RuleScopeAuthorizer) Authorize(ctx context.Context, userID uuid.UUID, superAdmin bool, input RuleDraftInput) error {
	if superAdmin {
		return nil
	}
	if a == nil || a.permissions == nil || a.deviceGroups == nil || userID == uuid.Nil {
		return ErrRuleScopeDenied
	}
	// 非超管必须显式限定设备或设备组，不能把空范围解释为全网。
	if len(input.MatchConditions.DeviceIDs) == 0 && len(input.MatchConditions.DeviceGroupIDs) == 0 {
		return ErrRuleScopeDenied
	}
	grants, err := a.permissions.GetUserVisibleDeviceGrants(ctx, userID, false)
	if err != nil {
		return fmt.Errorf("authorize notification rule device grants: %w", err)
	}
	for _, deviceID := range input.MatchConditions.DeviceIDs {
		groups, err := a.deviceGroups.GetDeviceGroupIDs(ctx, deviceID)
		if err != nil {
			return fmt.Errorf("authorize notification rule device groups: %w", err)
		}
		if !grantsCoverRuleDevice(grants, groups, input.MatchConditions.Technologies) {
			return ErrRuleScopeDenied
		}
	}
	for _, groupID := range input.MatchConditions.DeviceGroupIDs {
		if !grantsCoverRuleDevice(grants, []uuid.UUID{groupID}, input.MatchConditions.Technologies) {
			return ErrRuleScopeDenied
		}
	}
	return nil
}

func grantsCoverRuleDevice(grants []model.DeviceVisibilityGrant, groups []uuid.UUID, technologies []model.Technology) bool {
	if len(technologies) == 0 {
		for _, grant := range grants {
			if len(grant.Technologies) == 0 && grantCoversDeviceGroups(grant, groups) {
				return true
			}
		}
		return false
	}
	for _, technology := range technologies {
		covered := false
		for _, grant := range grants {
			if grantTechnologyMatches(grant.Technologies, technology) && grantCoversDeviceGroups(grant, groups) {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func grantCoversDeviceGroups(grant model.DeviceVisibilityGrant, groups []uuid.UUID) bool {
	for _, grantedGroup := range grant.GroupIDs {
		if len(groups) == 0 && grantedGroup.String() == global.DefaultLevel2GroupID {
			return true
		}
		for _, group := range groups {
			if grantedGroup == group {
				return true
			}
		}
	}
	return false
}
