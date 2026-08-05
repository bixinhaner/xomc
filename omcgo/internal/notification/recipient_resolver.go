package notification

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	RecipientTargetFixedContact = "fixed_contact"
	RecipientTargetUser         = "user"
	RecipientTargetRole         = "role"
	RecipientTargetContactGroup = "contact_group"
)

type RecipientTarget struct {
	TargetType           string
	TargetID             string
	AddressCiphertext    []byte
	AddressKeyVersion    int
	RecipientFingerprint []byte
	ChannelLimit         []string
}

type RecipientDirectoryUser struct {
	ID         uuid.UUID
	Enabled    bool
	SuperAdmin bool
	Email      string
	Phone      string
	Locale     string
}

type RecipientDirectory interface {
	GetUser(context.Context, uuid.UUID) (RecipientDirectoryUser, error)
	ListUserIDsByRole(context.Context, uuid.UUID) ([]uuid.UUID, error)
}

type RecipientPermissionResolver interface {
	GetUserVisibleDeviceGrants(context.Context, uuid.UUID, bool) ([]model.DeviceVisibilityGrant, error)
}

type RecipientDeviceGroups interface {
	GetDeviceGroupIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
}

type RecipientContactGroups interface {
	ListMembers(context.Context, uuid.UUID) ([]RecipientTarget, error)
}

type RecipientProtector interface {
	Protect(channel, address string) (ciphertext []byte, keyVersion int, fingerprint []byte, err error)
}

type RecipientExclusion struct {
	TargetType string     `json:"target_type"`
	TargetID   string     `json:"target_id,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Reason     string     `json:"reason"`
}

type RecipientResolution struct {
	Recipients []ResolvedRecipient  `json:"recipients"`
	Excluded   []RecipientExclusion `json:"excluded"`
}

type RecipientResolver struct {
	directory     RecipientDirectory
	permissions   RecipientPermissionResolver
	deviceGroups  RecipientDeviceGroups
	contactGroups RecipientContactGroups
	protector     RecipientProtector
}

func NewRecipientResolver(
	directory RecipientDirectory,
	permissions RecipientPermissionResolver,
	deviceGroups RecipientDeviceGroups,
	contactGroups RecipientContactGroups,
	protector RecipientProtector,
) *RecipientResolver {
	return &RecipientResolver{
		directory: directory, permissions: permissions, deviceGroups: deviceGroups,
		contactGroups: contactGroups, protector: protector,
	}
}

func (r *RecipientResolver) Resolve(
	ctx context.Context,
	targets []RecipientTarget,
	snapshot event.AlarmLifecycleSnapshot,
) (RecipientResolution, error) {
	if r == nil || r.directory == nil || r.permissions == nil || r.deviceGroups == nil || r.protector == nil {
		return RecipientResolution{}, fmt.Errorf("resolve notification recipients: dependencies are required")
	}
	deviceGroups, err := r.deviceGroups.GetDeviceGroupIDs(ctx, snapshot.DeviceID)
	if err != nil {
		return RecipientResolution{}, fmt.Errorf("resolve notification device groups: %w", err)
	}
	result := RecipientResolution{Recipients: make([]ResolvedRecipient, 0), Excluded: make([]RecipientExclusion, 0)}
	seen := make(map[string]struct{})
	for _, target := range targets {
		if err := r.resolveTarget(ctx, target, snapshot, deviceGroups, &result, seen); err != nil {
			return RecipientResolution{}, err
		}
	}
	return result, nil
}

func (r *RecipientResolver) resolveTarget(
	ctx context.Context,
	target RecipientTarget,
	snapshot event.AlarmLifecycleSnapshot,
	deviceGroups []uuid.UUID,
	result *RecipientResolution,
	seen map[string]struct{},
) error {
	switch target.TargetType {
	case RecipientTargetFixedContact:
		if len(target.AddressCiphertext) == 0 || target.AddressKeyVersion < 1 || len(target.RecipientFingerprint) == 0 {
			result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, Reason: "invalid_fixed_contact"})
			return nil
		}
		for _, channel := range target.ChannelLimit {
			r.addResolved(result, seen, ResolvedRecipient{
				Channel: channel, Ciphertext: target.AddressCiphertext, KeyVersion: target.AddressKeyVersion,
				Fingerprint: target.RecipientFingerprint, Locale: TemplateLanguageZhCN,
			})
		}
		return nil
	case RecipientTargetUser:
		userID, err := uuid.Parse(target.TargetID)
		if err != nil {
			result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, Reason: "invalid_user_id"})
			return nil
		}
		return r.resolveUser(ctx, userID, target, snapshot, deviceGroups, result, seen)
	case RecipientTargetRole:
		roleID, err := uuid.Parse(target.TargetID)
		if err != nil {
			result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, Reason: "invalid_role_id"})
			return nil
		}
		userIDs, err := r.directory.ListUserIDsByRole(ctx, roleID)
		if err != nil {
			if errors.Is(err, commonerrors.ErrNotFound) {
				result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, Reason: "role_not_found"})
				return nil
			}
			return fmt.Errorf("list notification role recipients: %w", err)
		}
		for _, userID := range userIDs {
			if err := r.resolveUser(ctx, userID, target, snapshot, deviceGroups, result, seen); err != nil {
				return err
			}
		}
		return nil
	case RecipientTargetContactGroup:
		if r.contactGroups == nil {
			return fmt.Errorf("resolve notification contact group: repository is required")
		}
		groupID, err := uuid.Parse(target.TargetID)
		if err != nil {
			result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, Reason: "invalid_contact_group_id"})
			return nil
		}
		members, err := r.contactGroups.ListMembers(ctx, groupID)
		if err != nil {
			if errors.Is(err, commonerrors.ErrNotFound) {
				result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, Reason: "contact_group_not_found"})
				return nil
			}
			return fmt.Errorf("list notification contact group members: %w", err)
		}
		for _, member := range members {
			if member.TargetType == RecipientTargetContactGroup {
				return fmt.Errorf("resolve notification contact group: nested groups are not supported")
			}
			if len(member.ChannelLimit) == 0 {
				member.ChannelLimit = append([]string(nil), target.ChannelLimit...)
			}
			if err := r.resolveTarget(ctx, member, snapshot, deviceGroups, result, seen); err != nil {
				return err
			}
		}
		return nil
	default:
		result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, Reason: "unsupported_target_type"})
		return nil
	}
}

func (r *RecipientResolver) resolveUser(
	ctx context.Context,
	userID uuid.UUID,
	target RecipientTarget,
	snapshot event.AlarmLifecycleSnapshot,
	deviceGroups []uuid.UUID,
	result *RecipientResolution,
	seen map[string]struct{},
) error {
	user, err := r.directory.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			result.Excluded = append(result.Excluded, RecipientExclusion{
				TargetType: target.TargetType, TargetID: target.TargetID, UserID: &userID, Reason: "user_not_found",
			})
			return nil
		}
		return fmt.Errorf("get notification recipient user: %w", err)
	}
	if !user.Enabled {
		result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, UserID: &userID, Reason: "user_disabled"})
		return nil
	}
	grants, err := r.permissions.GetUserVisibleDeviceGrants(ctx, userID, user.SuperAdmin)
	if err != nil {
		return fmt.Errorf("get notification recipient device grants: %w", err)
	}
	if !recipientCanSeeDevice(grants, deviceGroups, snapshot.Technology) {
		result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, UserID: &userID, Reason: "device_scope_denied"})
		return nil
	}
	locale := strings.TrimSpace(user.Locale)
	if locale == "" {
		locale = TemplateLanguageZhCN
	}
	for _, channel := range target.ChannelLimit {
		address := user.Email
		if strings.HasPrefix(channel, "sms") {
			address = user.Phone
		}
		address = strings.TrimSpace(address)
		if address == "" {
			result.Excluded = append(result.Excluded, RecipientExclusion{TargetType: target.TargetType, TargetID: target.TargetID, UserID: &userID, Reason: "channel_address_missing"})
			continue
		}
		ciphertext, keyVersion, fingerprint, err := r.protector.Protect(channel, address)
		if err != nil {
			return fmt.Errorf("protect notification recipient address: %w", err)
		}
		r.addResolved(result, seen, ResolvedRecipient{
			UserID: &userID, Channel: channel, Ciphertext: ciphertext,
			KeyVersion: keyVersion, Fingerprint: fingerprint, Locale: locale,
		})
	}
	return nil
}

func (r *RecipientResolver) addResolved(result *RecipientResolution, seen map[string]struct{}, recipient ResolvedRecipient) {
	key := recipient.Channel + ":" + hex.EncodeToString(recipient.Fingerprint)
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	result.Recipients = append(result.Recipients, recipient)
}

func recipientCanSeeDevice(
	grants []model.DeviceVisibilityGrant,
	deviceGroups []uuid.UUID,
	technologyValue *string,
) bool {
	if grants == nil {
		return true
	}
	technology := model.Technology("")
	if technologyValue != nil {
		technology = model.Technology(*technologyValue)
	}
	for _, grant := range grants {
		if !grantTechnologyMatches(grant.Technologies, technology) {
			continue
		}
		for _, grantedGroup := range grant.GroupIDs {
			if len(deviceGroups) == 0 && grantedGroup.String() == global.DefaultLevel2GroupID {
				return true
			}
			for _, deviceGroup := range deviceGroups {
				if grantedGroup == deviceGroup {
					return true
				}
			}
		}
	}
	return false
}

func grantTechnologyMatches(allowed []model.Technology, actual model.Technology) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, technology := range allowed {
		if technology == actual {
			return true
		}
	}
	return false
}
