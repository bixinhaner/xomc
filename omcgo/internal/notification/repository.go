package notification

import (
	"context"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Repository defines the persistence interface for notifications.
type Repository interface {
	List(ctx context.Context, filter NotificationFilter) (*model.ListResponse[Notification], error)
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	Create(ctx context.Context, notif *Notification) error
	// UpsertByDedup 按 (user_id, dedup_key) upsert notification (T-0157 C3)。
	// 用于 task → notification 订阅器；DedupKey 为空时降级走 Create。
	UpsertByDedup(ctx context.Context, notif *Notification) error
	MarkRead(ctx context.Context, id uuid.UUID, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
	GetUnreadCount(ctx context.Context, userID string) (int64, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
}
