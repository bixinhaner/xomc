package notification

import (
	"context"

	"github.com/google/uuid"
)

type RuleRepository interface {
	List(context.Context) ([]NotificationRule, error)
	Get(context.Context, uuid.UUID) (*NotificationRule, error)
	Create(context.Context, RuleDraftInput, string) (*NotificationRule, error)
	UpdateDraft(context.Context, uuid.UUID, int64, RuleDraftInput, string) (*NotificationRule, error)
	Publish(context.Context, uuid.UUID, int64, string) (*NotificationRule, error)
	Enable(context.Context, uuid.UUID, int64, uuid.UUID) (*NotificationRule, error)
	Archive(context.Context, uuid.UUID, int64) (*NotificationRule, error)
}
