package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type ContactGroup struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	IsDefault   bool                 `json:"is_default"`
	Revision    int64                `json:"revision"`
	Archived    bool                 `json:"archived"`
	CreatedBy   string               `json:"created_by"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	Members     []ContactGroupMember `json:"members"`
}

type ContactGroupMember struct {
	ID                uuid.UUID `json:"id"`
	TargetType        string    `json:"target_type"`
	TargetID          string    `json:"target_id,omitempty"`
	AddressConfigured bool      `json:"address_configured"`
	ChannelLimit      []string  `json:"channel_limit"`
}

type ContactGroupMemberInput struct {
	TargetType           string   `json:"target_type"`
	TargetID             string   `json:"target_id,omitempty"`
	AddressCiphertext    []byte   `json:"address_ciphertext,omitempty"`
	AddressKeyVersion    int      `json:"address_key_version,omitempty"`
	RecipientFingerprint []byte   `json:"recipient_fingerprint,omitempty"`
	ChannelLimit         []string `json:"channel_limit"`
}

type ContactGroupInput struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	IsDefault   bool                      `json:"is_default"`
	Members     []ContactGroupMemberInput `json:"members"`
}

type ContactGroupRepository interface {
	List(context.Context) ([]ContactGroup, error)
	Get(context.Context, uuid.UUID) (*ContactGroup, error)
	Create(context.Context, ContactGroupInput, string) (*ContactGroup, error)
	Update(context.Context, uuid.UUID, int64, ContactGroupInput, string) (*ContactGroup, error)
	ListMembers(context.Context, uuid.UUID) ([]RecipientTarget, error)
}

type ContactGroupService struct{ repository ContactGroupRepository }

func NewContactGroupService(repository ContactGroupRepository) *ContactGroupService {
	return &ContactGroupService{repository: repository}
}

func (s *ContactGroupService) List(ctx context.Context) ([]ContactGroup, error) {
	groups, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification contact groups: %w", err)
	}
	return groups, nil
}

func (s *ContactGroupService) Get(ctx context.Context, id uuid.UUID) (*ContactGroup, error) {
	group, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification contact group: %w", err)
	}
	return group, nil
}

func (s *ContactGroupService) Create(ctx context.Context, input ContactGroupInput, actor string) (*ContactGroup, error) {
	input, err := normalizeContactGroupInput(input)
	if err != nil {
		return nil, err
	}
	group, err := s.repository.Create(ctx, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("create notification contact group: %w", err)
	}
	return group, nil
}

func (s *ContactGroupService) Update(ctx context.Context, id uuid.UUID, revision int64, input ContactGroupInput, actor string) (*ContactGroup, error) {
	input, err := normalizeContactGroupInput(input)
	if err != nil {
		return nil, err
	}
	group, err := s.repository.Update(ctx, id, revision, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("update notification contact group: %w", err)
	}
	return group, nil
}

func normalizeContactGroupInput(input ContactGroupInput) (ContactGroupInput, error) {
	input.Name, input.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.Description)
	if input.Name == "" || len(input.Name) > 128 {
		return input, commonerrors.ErrInvalidInput
	}
	for i := range input.Members {
		member := &input.Members[i]
		member.TargetID = strings.TrimSpace(member.TargetID)
		member.ChannelLimit = normalizeVariables(member.ChannelLimit)
		if len(member.ChannelLimit) == 0 || !validNotificationChannels(member.ChannelLimit) {
			return input, commonerrors.ErrInvalidInput
		}
		switch member.TargetType {
		case RecipientTargetUser, RecipientTargetRole:
			if _, err := uuid.Parse(member.TargetID); err != nil {
				return input, commonerrors.ErrInvalidInput
			}
		case RecipientTargetFixedContact:
			if len(member.AddressCiphertext) == 0 || member.AddressKeyVersion < 1 || len(member.RecipientFingerprint) == 0 {
				return input, commonerrors.ErrInvalidInput
			}
		default:
			return input, commonerrors.ErrInvalidInput
		}
	}
	return input, nil
}
