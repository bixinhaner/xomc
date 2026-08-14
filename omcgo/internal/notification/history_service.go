package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// HistoryService is the business-logic layer for notification history. The
// public API only allows reads (List/GetByID); writes are reserved for internal
// dispatchers (alarm webhook/email/sms) via Insert + UpdateStatus.
type HistoryService struct {
	repo   HistoryRepository
	logger *zap.Logger
}

// NewHistoryService creates a HistoryService.
func NewHistoryService(repo HistoryRepository, logger *zap.Logger) *HistoryService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HistoryService{
		repo:   repo,
		logger: logger.Named("notification-history"),
	}
}

// List returns paginated history entries matching the filter.
func (s *HistoryService) List(ctx context.Context, filter NotificationHistoryFilter) (*model.ListResponse[NotificationHistory], error) {
	return s.repo.List(ctx, filter)
}

// GetByID returns a single history entry.
func (s *HistoryService) GetByID(ctx context.Context, id uuid.UUID) (*NotificationHistory, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *HistoryService) GetByDedupKey(ctx context.Context, dedupKey string) (*NotificationHistory, error) {
	return s.repo.GetByDedupKey(ctx, dedupKey)
}

// Insert persists a new history entry. Intended for internal dispatchers, not
// HTTP clients. The dispatcher passes a pre-populated NotificationHistory; this
// method validates required fields and applies defaults.
func (s *HistoryService) Insert(ctx context.Context, h *NotificationHistory) error {
	if err := validateHistory(h); err != nil {
		return err
	}
	return s.repo.Insert(ctx, h)
}

// InsertIfAbsent 原子创建带幂等键的发送记录。
func (s *HistoryService) InsertIfAbsent(ctx context.Context, h *NotificationHistory) (*NotificationHistory, bool, error) {
	if err := validateHistory(h); err != nil {
		return nil, false, err
	}
	return s.repo.InsertIfAbsent(ctx, h)
}

func validateHistory(h *NotificationHistory) error {
	if h == nil {
		return commonerrors.ErrInvalidInput
	}
	if err := validateHistoryChannel(h.Channel); err != nil {
		return err
	}
	if err := validateHistoryStatus(h.Status); err != nil {
		return err
	}
	if len(h.Recipients) == 0 {
		return fmt.Errorf("%w: at least one recipient required", commonerrors.ErrInvalidInput)
	}
	if h.DedupKey != nil {
		key := strings.TrimSpace(*h.DedupKey)
		if key == "" {
			h.DedupKey = nil
		} else {
			h.DedupKey = &key
		}
	}
	return nil
}

// ClaimAttempt 原子抢占一次实际发送尝试。只有失败记录或超时的
// pending 记录可以被抢占，避免多个异步任务重复投递同一封邮件。
func (s *HistoryService) ClaimAttempt(ctx context.Context, id uuid.UUID, attemptedAt, staleBefore time.Time) (bool, error) {
	return s.repo.ClaimAttempt(ctx, id, attemptedAt, staleBefore)
}

// MarkSent updates the entry to "sent" with the supplied timestamp.
func (s *HistoryService) MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	return s.repo.UpdateStatus(ctx, id, HistoryStatusSent, nil, &sentAt)
}

// MarkFailed updates the entry to "failed" with the supplied error message.
func (s *HistoryService) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	msg := strings.TrimSpace(errorMessage)
	var msgPtr *string
	if msg != "" {
		msgPtr = &msg
	}
	return s.repo.UpdateStatus(ctx, id, HistoryStatusFailed, msgPtr, nil)
}

// MarkDeadLetter moves the entry to the dead-letter terminal state with the
// supplied error message.
func (s *HistoryService) MarkDeadLetter(ctx context.Context, id uuid.UUID, errorMessage string) error {
	msg := strings.TrimSpace(errorMessage)
	var msgPtr *string
	if msg != "" {
		msgPtr = &msg
	}
	return s.repo.UpdateStatus(ctx, id, HistoryStatusDeadLetter, msgPtr, nil)
}

// validateHistoryChannel rejects unsupported channel values.
func validateHistoryChannel(channel string) error {
	switch channel {
	case TemplateChannelEmail, TemplateChannelSMS, TemplateChannelWebhook:
		return nil
	default:
		return fmt.Errorf("%w: unsupported history channel %q", commonerrors.ErrInvalidInput, channel)
	}
}

// validateHistoryStatus rejects unsupported status values.
func validateHistoryStatus(status string) error {
	switch status {
	case HistoryStatusPending, HistoryStatusSent, HistoryStatusFailed, HistoryStatusDeadLetter:
		return nil
	default:
		return fmt.Errorf("%w: unsupported history status %q", commonerrors.ErrInvalidInput, status)
	}
}
