package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

var deviceAccessActionNotificationQueues = map[string]string{
	event.SubjectDeviceAccessActionFailed:    "notification-device-access-action-failed",
	event.SubjectDeviceAccessActionRecovered: "notification-device-access-action-recovered",
}

type deviceAccessActionEvent struct {
	EventID      string     `json:"event_id"`
	EventName    string     `json:"event_name"`
	RequestID    string     `json:"request_id"`
	ActionID     uuid.UUID  `json:"action_id"`
	DeviceID     *uuid.UUID `json:"device_id"`
	CandidateID  *uuid.UUID `json:"candidate_id"`
	Carrier      string     `json:"carrier"`
	SerialNumber string     `json:"serial_number"`
	ActionStatus string     `json:"action_status"`
	ReasonCode   string     `json:"reason_code"`
	Message      string     `json:"message"`
	OccurredAt   time.Time  `json:"occurred_at"`
}

// DeviceAccessActionSubscriber records a durable unified-notification history
// association for action failures and recoveries. Until a delivery rule is
// configured, the row explicitly remains not_configured rather than claiming
// a successful email/SMS/webhook delivery.
type DeviceAccessActionSubscriber struct {
	history *HistoryService
	logger  *zap.Logger
}

func NewDeviceAccessActionSubscriber(history *HistoryService, logger *zap.Logger) *DeviceAccessActionSubscriber {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DeviceAccessActionSubscriber{history: history, logger: logger.Named("notification-device-access-action")}
}

func (s *DeviceAccessActionSubscriber) Subscribe(bus event.EventBus) error {
	if s == nil || s.history == nil || bus == nil {
		return fmt.Errorf("device access action notification dependencies are required")
	}
	for subject, queue := range deviceAccessActionNotificationQueues {
		if _, err := bus.QueueSubscribe(subject, queue, s.handle); err != nil {
			return fmt.Errorf("subscribe device access action notification %s: %w", subject, err)
		}
	}
	return nil
}

func (s *DeviceAccessActionSubscriber) handle(ctx context.Context, evt event.Event) error {
	var payload deviceAccessActionEvent
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device access action notification: %w", err)
	}
	eventID, err := uuid.Parse(payload.EventID)
	if err != nil || payload.ActionID == uuid.Nil || payload.EventName == "" {
		return fmt.Errorf("decode device access action notification: invalid event identity")
	}
	body, err := json.Marshal(map[string]any{
		"carrier": payload.Carrier, "serial_number": payload.SerialNumber,
		"device_id": payload.DeviceID, "candidate_id": payload.CandidateID, "action_status": payload.ActionStatus,
		"reason_code": payload.ReasonCode, "message": payload.Message,
	})
	if err != nil {
		return fmt.Errorf("encode device access action notification body: %w", err)
	}
	history := &NotificationHistory{
		Channel: HistoryChannelSystem, Recipients: []string{"device-access-operators"},
		Subject: payload.EventName, Body: string(body), Status: HistoryStatusNotConfigured,
		SourceType: "device_access_action", SourceID: &payload.ActionID, EventID: &eventID,
		CorrelationID: payload.RequestID, CreatedAt: payload.OccurredAt,
	}
	if history.CreatedAt.IsZero() {
		history.CreatedAt = evt.Timestamp.UTC()
	}
	if err := s.history.Insert(ctx, history); err != nil {
		s.logger.Warn("record device access action notification history", zap.String("event_id", payload.EventID), zap.Error(err))
		return err
	}
	return nil
}
