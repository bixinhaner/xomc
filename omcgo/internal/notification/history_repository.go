package notification

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// HistoryRepository defines persistence operations for notification history.
// History rows are append-only from the API perspective; status updates flow
// through the dispatcher (UpdateStatus) when delivery completes.
type HistoryRepository interface {
	List(ctx context.Context, filter NotificationHistoryFilter) (*model.ListResponse[NotificationHistory], error)
	GetByID(ctx context.Context, id uuid.UUID) (*NotificationHistory, error)
	GetByDedupKey(ctx context.Context, dedupKey string) (*NotificationHistory, error)
	Insert(ctx context.Context, h *NotificationHistory) error
	InsertIfAbsent(ctx context.Context, h *NotificationHistory) (*NotificationHistory, bool, error)
	ClaimAttempt(ctx context.Context, id uuid.UUID, attemptedAt, staleBefore time.Time) (bool, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, sentAt *time.Time) error
}
