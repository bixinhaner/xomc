package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/events"
)

// Service provides business logic for notification management.
type Service struct {
	repo   Repository
	hub    *events.MessageHub
	logger *zap.Logger
}

// NewService creates a new notification Service.
// The hub parameter is optional; if nil, notifications will not be pushed via SSE.
func NewService(repo Repository, hub *events.MessageHub, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		hub:    hub,
		logger: logger.Named("notification"),
	}
}

// List returns a paginated list of notifications for the given user.
func (s *Service) List(ctx context.Context, filter NotificationFilter) (*model.ListResponse[Notification], error) {
	return s.repo.List(ctx, filter)
}

// GetByID returns a single notification by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// CreateNotification creates a new notification and pushes it via SSE if hub is available.
func (s *Service) CreateNotification(ctx context.Context, notif *Notification) (*Notification, error) {
	if err := s.repo.Create(ctx, notif); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	// Push via SSE if hub is available
	if s.hub != nil {
		s.pushSSEEvent(notif)
	}

	return notif, nil
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID, userID string) error {
	return s.repo.MarkRead(ctx, id, userID)
}

// MarkAllRead marks all notifications for a user as read.
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// GetUnreadCount returns the number of unread notifications for a user.
func (s *Service) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// Delete removes a notification.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

// pushSSEEvent publishes a notification event to the user's SSE channel.
func (s *Service) pushSSEEvent(notif *Notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		s.logger.Error("failed to marshal notification for SSE",
			zap.String("notification_id", notif.ID.String()),
			zap.Error(err),
		)
		return
	}

	msg := &events.SSEMessage{
		ID:     notif.ID.String(),
		Event:  "notification",
		Data:   data,
		UserID: notif.UserID,
	}

	if err := s.hub.Publish(notif.UserID, msg); err != nil {
		s.logger.Error("failed to push notification via SSE",
			zap.String("user_id", notif.UserID),
			zap.Error(err),
		)
	}
}

// SendNotification is a convenience method to create and push a notification
// in a single call. Useful for other modules to trigger notifications.
func (s *Service) SendNotification(ctx context.Context, userID string, notifType NotificationType, priority NotificationPriority, title, content, link string) error {
	notif := &Notification{
		UserID:   userID,
		Type:     notifType,
		Priority: priority,
		Title:    title,
		Content:  content,
		Link:     link,
		Sender:   "system",
	}

	_, err := s.CreateNotification(ctx, notif)
	return err
}

// SendGlobalNotification sends a notification to multiple users.
// For bulk notifications, this avoids repeated create+publish cycles.
func (s *Service) SendGlobalNotification(ctx context.Context, userIDs []string, notifType NotificationType, priority NotificationPriority, title, content, link string) error {
	for _, userID := range userIDs {
		if err := s.SendNotification(ctx, userID, notifType, priority, title, content, link); err != nil {
			s.logger.Warn("failed to send notification to user",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}
	return nil
}

// CreateAndBroadcast creates a notification for a user and also sends a global
// SSE event if needed (e.g., for admin dashboards).
func (s *Service) CreateAndBroadcast(ctx context.Context, notif *Notification) error {
	if err := s.repo.Create(ctx, notif); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	// Push to specific user
	if s.hub != nil {
		s.pushSSEEvent(notif)
	}

	return nil
}
