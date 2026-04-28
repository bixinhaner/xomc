package notification

import (
	"context"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// TemplateRepository defines the persistence interface for notification templates.
type TemplateRepository interface {
	List(ctx context.Context, filter NotificationTemplateFilter) (*model.ListResponse[NotificationTemplate], error)
	GetByID(ctx context.Context, id uuid.UUID) (*NotificationTemplate, error)
	GetByName(ctx context.Context, name string) (*NotificationTemplate, error)
	Create(ctx context.Context, tpl *NotificationTemplate) error
	Update(ctx context.Context, tpl *NotificationTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
}
