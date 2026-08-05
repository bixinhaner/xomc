package notification

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type recipientDirectoryStub struct {
	users     map[uuid.UUID]RecipientDirectoryUser
	roleUsers map[uuid.UUID][]uuid.UUID
}

func (s *recipientDirectoryStub) GetUser(_ context.Context, id uuid.UUID) (RecipientDirectoryUser, error) {
	user, ok := s.users[id]
	if !ok {
		return RecipientDirectoryUser{}, fmt.Errorf("user not found")
	}
	return user, nil
}
func (s *recipientDirectoryStub) ListUserIDsByRole(_ context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	return s.roleUsers[id], nil
}

type recipientPermissionStub struct {
	grants map[uuid.UUID][]model.DeviceVisibilityGrant
}

func (s *recipientPermissionStub) GetUserVisibleDeviceGrants(_ context.Context, id uuid.UUID, super bool) ([]model.DeviceVisibilityGrant, error) {
	if super {
		return nil, nil
	}
	return s.grants[id], nil
}

type recipientGroupsStub struct{ groups []uuid.UUID }

func (s recipientGroupsStub) GetDeviceGroupIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return s.groups, nil
}

type contactGroupsStub struct {
	members map[uuid.UUID][]RecipientTarget
}

func (s contactGroupsStub) ListMembers(_ context.Context, id uuid.UUID) ([]RecipientTarget, error) {
	return s.members[id], nil
}

type recipientProtectorStub struct{}

func (recipientProtectorStub) Protect(channel, address string) ([]byte, int, []byte, error) {
	normalized := strings.ToLower(strings.TrimSpace(address))
	fingerprint := sha256.Sum256([]byte(channel + ":" + normalized))
	ciphertext := sha256.Sum256([]byte("cipher:" + normalized))
	return ciphertext[:], 1, fingerprint[:], nil
}

func TestRecipientResolver_RequiresGroupAndTechnologyOnSameGrant(t *testing.T) {
	userID, groupA, groupB := uuid.New(), uuid.New(), uuid.New()
	technology := "nr"
	resolver := NewRecipientResolver(
		&recipientDirectoryStub{users: map[uuid.UUID]RecipientDirectoryUser{
			userID: {ID: userID, Enabled: true, Email: "noc@example.com"},
		}},
		&recipientPermissionStub{grants: map[uuid.UUID][]model.DeviceVisibilityGrant{
			userID: {
				{GroupIDs: []uuid.UUID{groupA}, Technologies: []model.Technology{model.TechLTE}},
				{GroupIDs: []uuid.UUID{groupB}, Technologies: []model.Technology{model.TechNR}},
			},
		}},
		recipientGroupsStub{groups: []uuid.UUID{groupA}}, contactGroupsStub{}, recipientProtectorStub{},
	)
	resolution, err := resolver.Resolve(context.Background(), []RecipientTarget{{
		TargetType: RecipientTargetUser, TargetID: userID.String(), ChannelLimit: []string{"email"},
	}}, event.AlarmLifecycleSnapshot{DeviceID: uuid.New(), Technology: &technology})
	require.NoError(t, err)
	require.Empty(t, resolution.Recipients)
	require.Equal(t, "device_scope_denied", resolution.Excluded[0].Reason)
}

func TestRecipientResolver_ExpandsRoleFiltersDisabledAndDeduplicatesAddress(t *testing.T) {
	roleID, firstUser, secondUser, disabledUser, groupID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	technology := "nr"
	directory := &recipientDirectoryStub{
		users: map[uuid.UUID]RecipientDirectoryUser{
			firstUser:    {ID: firstUser, Enabled: true, Email: "NOC@example.com", Locale: "en-US"},
			secondUser:   {ID: secondUser, Enabled: true, Email: "noc@example.com"},
			disabledUser: {ID: disabledUser, Enabled: false, Email: "disabled@example.com"},
		},
		roleUsers: map[uuid.UUID][]uuid.UUID{roleID: {firstUser, secondUser, disabledUser}},
	}
	permissions := &recipientPermissionStub{grants: map[uuid.UUID][]model.DeviceVisibilityGrant{
		firstUser:  {{GroupIDs: []uuid.UUID{groupID}, Technologies: []model.Technology{model.TechNR}}},
		secondUser: {{GroupIDs: []uuid.UUID{groupID}, Technologies: []model.Technology{model.TechNR}}},
	}}
	resolver := NewRecipientResolver(directory, permissions, recipientGroupsStub{groups: []uuid.UUID{groupID}}, contactGroupsStub{}, recipientProtectorStub{})

	resolution, err := resolver.Resolve(context.Background(), []RecipientTarget{{
		TargetType: RecipientTargetRole, TargetID: roleID.String(), ChannelLimit: []string{"email"},
	}}, event.AlarmLifecycleSnapshot{DeviceID: uuid.New(), Technology: &technology})
	require.NoError(t, err)
	require.Len(t, resolution.Recipients, 1)
	require.Equal(t, firstUser, *resolution.Recipients[0].UserID)
	require.Equal(t, "en-US", resolution.Recipients[0].Locale)
	require.NotContains(t, string(resolution.Recipients[0].Ciphertext), "noc@example.com")
	require.Len(t, resolution.Excluded, 1)
	require.Equal(t, "user_disabled", resolution.Excluded[0].Reason)
}

func TestRecipientResolver_ContactGroupDoesNotBypassUserPermission(t *testing.T) {
	groupID, userID, deviceGroup := uuid.New(), uuid.New(), uuid.New()
	technology := "lte"
	resolver := NewRecipientResolver(
		&recipientDirectoryStub{users: map[uuid.UUID]RecipientDirectoryUser{
			userID: {ID: userID, Enabled: true, Email: "user@example.com"},
		}},
		&recipientPermissionStub{grants: map[uuid.UUID][]model.DeviceVisibilityGrant{userID: {}}},
		recipientGroupsStub{groups: []uuid.UUID{deviceGroup}},
		contactGroupsStub{members: map[uuid.UUID][]RecipientTarget{
			groupID: {{TargetType: RecipientTargetUser, TargetID: userID.String(), ChannelLimit: []string{"email"}}},
		}},
		recipientProtectorStub{},
	)

	resolution, err := resolver.Resolve(context.Background(), []RecipientTarget{{
		TargetType: RecipientTargetContactGroup, TargetID: groupID.String(), ChannelLimit: []string{"email"},
	}}, event.AlarmLifecycleSnapshot{DeviceID: uuid.New(), Technology: &technology})
	require.NoError(t, err)
	require.Empty(t, resolution.Recipients)
	require.Equal(t, "device_scope_denied", resolution.Excluded[0].Reason)
}
