package authz

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

type RuntimeScopeUserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*admin.User, error)
}

type RuntimeScopeDeviceLookup interface {
	GetBySerialNumber(ctx context.Context, serialNumber string) (*model.Device, error)
}

// RuntimeDeviceScopeAuthorizer rechecks the creator's current data permission
// immediately before a background email job reads device data. Saving a
// subscription or report is not a durable permission grant: later role/group
// revocation must take effect for queued and scheduled jobs as well.
type RuntimeDeviceScopeAuthorizer struct {
	users       RuntimeScopeUserReader
	permissions VisibleGroupsResolver
	groups      GroupReader
	devices     RuntimeScopeDeviceLookup
}

func NewRuntimeDeviceScopeAuthorizer(
	users RuntimeScopeUserReader,
	permissions VisibleGroupsResolver,
	groups GroupReader,
	devices RuntimeScopeDeviceLookup,
) *RuntimeDeviceScopeAuthorizer {
	return &RuntimeDeviceScopeAuthorizer{
		users: users, permissions: permissions, groups: groups, devices: devices,
	}
}

func (a *RuntimeDeviceScopeAuthorizer) AuthorizeDeviceScope(
	ctx context.Context,
	userID uuid.UUID,
	deviceIDs, deviceGroupIDs []uuid.UUID,
) error {
	visibleGroups, err := a.visibleGroups(ctx, userID)
	if err != nil {
		return err
	}
	if visibleGroups == nil {
		return nil
	}
	if len(deviceIDs) == 0 && len(deviceGroupIDs) == 0 {
		return fmt.Errorf("background device scope is empty: %w", commonerrors.ErrForbidden)
	}
	for _, groupID := range deviceGroupIDs {
		if !slices.Contains(visibleGroups, groupID) {
			return fmt.Errorf("device group %s is no longer visible: %w", groupID, commonerrors.ErrForbidden)
		}
	}
	for _, deviceID := range deviceIDs {
		if err := AuthorizeDeviceAccess(ctx, a.groups, deviceID, visibleGroups); err != nil {
			return fmt.Errorf("device %s is no longer visible: %w", deviceID, err)
		}
	}
	return nil
}

func (a *RuntimeDeviceScopeAuthorizer) AuthorizeSerialNumbers(
	ctx context.Context,
	userID uuid.UUID,
	serialNumbers []string,
) error {
	visibleGroups, err := a.visibleGroups(ctx, userID)
	if err != nil {
		return err
	}
	if visibleGroups == nil {
		return nil
	}
	if len(serialNumbers) == 0 {
		return fmt.Errorf("background device scope is empty: %w", commonerrors.ErrForbidden)
	}
	if a.devices == nil {
		return fmt.Errorf("runtime device lookup is not configured")
	}
	seen := make(map[string]struct{}, len(serialNumbers))
	for _, raw := range serialNumbers {
		serialNumber := strings.TrimSpace(raw)
		if serialNumber == "" {
			return fmt.Errorf("background device scope contains an empty serial number: %w", commonerrors.ErrForbidden)
		}
		if _, ok := seen[serialNumber]; ok {
			continue
		}
		seen[serialNumber] = struct{}{}
		device, err := a.devices.GetBySerialNumber(ctx, serialNumber)
		if err != nil {
			return fmt.Errorf("load device %q for runtime authorization: %w", serialNumber, err)
		}
		if device == nil {
			return fmt.Errorf("device %q is no longer available: %w", serialNumber, commonerrors.ErrForbidden)
		}
		if err := AuthorizeDeviceAccess(ctx, a.groups, device.ID, visibleGroups); err != nil {
			return fmt.Errorf("device %q is no longer visible: %w", serialNumber, err)
		}
	}
	return nil
}

func (a *RuntimeDeviceScopeAuthorizer) visibleGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if a == nil || a.users == nil || a.permissions == nil || a.groups == nil {
		return nil, fmt.Errorf("runtime device scope authorizer is not configured")
	}
	if userID == uuid.Nil {
		return nil, fmt.Errorf("background job creator is missing: %w", commonerrors.ErrForbidden)
	}
	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load background job creator: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("background job creator no longer exists: %w", commonerrors.ErrForbidden)
	}
	if user.Status != admin.UserStatusActive {
		return nil, fmt.Errorf("background job creator is not active: %w", commonerrors.ErrForbidden)
	}
	visibleGroups, err := a.permissions.GetUserVisibleGroupIDs(ctx, userID, user.IsSuperAdmin())
	if err != nil {
		return nil, fmt.Errorf("resolve current background job device scope: %w", err)
	}
	return visibleGroups, nil
}
