package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type contactGroupRepositoryStub struct {
	input    ContactGroupInput
	revision int64
	group    *ContactGroup
}

func (*contactGroupRepositoryStub) List(context.Context) ([]ContactGroup, error) { return nil, nil }
func (s *contactGroupRepositoryStub) Get(context.Context, uuid.UUID) (*ContactGroup, error) {
	return s.group, nil
}
func (s *contactGroupRepositoryStub) Create(_ context.Context, input ContactGroupInput, _ string) (*ContactGroup, error) {
	s.input = input
	return s.group, nil
}
func (s *contactGroupRepositoryStub) Update(_ context.Context, _ uuid.UUID, revision int64, input ContactGroupInput, _ string) (*ContactGroup, error) {
	s.input, s.revision = input, revision
	return s.group, nil
}
func (*contactGroupRepositoryStub) ListMembers(context.Context, uuid.UUID) ([]RecipientTarget, error) {
	return nil, nil
}

func TestContactGroupService_RejectsUnprotectedFixedContact(t *testing.T) {
	service := NewContactGroupService(&contactGroupRepositoryStub{})
	_, err := service.Create(context.Background(), ContactGroupInput{
		Name: "NOC", Members: []ContactGroupMemberInput{{
			TargetType: RecipientTargetFixedContact, ChannelLimit: []string{"email"},
		}},
	}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestContactGroupService_AcceptsOnlyUserRoleOrProtectedFixedMember(t *testing.T) {
	userID := uuid.New()
	repository := &contactGroupRepositoryStub{group: &ContactGroup{ID: uuid.New(), Revision: 1}}
	service := NewContactGroupService(repository)
	_, err := service.Create(context.Background(), ContactGroupInput{
		Name: " NOC ", Members: []ContactGroupMemberInput{
			{TargetType: RecipientTargetUser, TargetID: userID.String(), ChannelLimit: []string{" email ", "email"}},
			{TargetType: RecipientTargetFixedContact, AddressCiphertext: []byte("cipher"), AddressKeyVersion: 1, RecipientFingerprint: []byte("fingerprint"), ChannelLimit: []string{"email"}},
		},
	}, "operator")
	require.NoError(t, err)
	require.Equal(t, "NOC", repository.input.Name)
	require.Equal(t, []string{"email"}, repository.input.Members[0].ChannelLimit)
}
