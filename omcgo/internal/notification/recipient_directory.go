package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
)

type recipientUserReader interface {
	GetByID(context.Context, uuid.UUID) (*admin.User, error)
}

type recipientRoleReader interface {
	GetByID(context.Context, uuid.UUID) (*admin.Role, error)
	ListUserIDsByRole(context.Context, uuid.UUID) ([]uuid.UUID, error)
}

type AdminRecipientDirectory struct {
	users recipientUserReader
	roles recipientRoleReader
}

func NewAdminRecipientDirectory(users recipientUserReader, roles recipientRoleReader) *AdminRecipientDirectory {
	return &AdminRecipientDirectory{users: users, roles: roles}
}

func (d *AdminRecipientDirectory) GetUser(ctx context.Context, id uuid.UUID) (RecipientDirectoryUser, error) {
	if d == nil || d.users == nil {
		return RecipientDirectoryUser{}, fmt.Errorf("get notification directory user: repository is required")
	}
	user, err := d.users.GetByID(ctx, id)
	if err != nil {
		return RecipientDirectoryUser{}, fmt.Errorf("get notification directory user: %w", err)
	}
	return RecipientDirectoryUser{
		ID: user.ID, Enabled: user.Status == admin.UserStatusActive, SuperAdmin: user.IsSuperAdmin(),
		Email: user.Email, Phone: user.Phone,
	}, nil
}

func (d *AdminRecipientDirectory) ListUserIDsByRole(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if d == nil || d.roles == nil {
		return nil, fmt.Errorf("list notification directory role users: repository is required")
	}
	if _, err := d.roles.GetByID(ctx, id); err != nil {
		return nil, fmt.Errorf("get notification directory role: %w", err)
	}
	ids, err := d.roles.ListUserIDsByRole(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list notification directory role users: %w", err)
	}
	return ids, nil
}
