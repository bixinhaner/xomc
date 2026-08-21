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

var deviceAccessDecisionNotificationQueues = map[string]string{
	event.SubjectDeviceAccessAccepted:       "notification-device-access-decision-accepted",
	event.SubjectDeviceAccessRejected:       "notification-device-access-decision-rejected",
	event.SubjectDeviceAccessReviewRequired: "notification-device-access-decision-review-required",
	event.SubjectDeviceAccessRevoked:        "notification-device-access-decision-revoked",
}

type deviceAccessDecisionEvent struct {
	EventID         string     `json:"event_id"`
	EventName       string     `json:"event_name"`
	RequestID       string     `json:"request_id"`
	DecisionID      uuid.UUID  `json:"decision_id"`
	DeviceID        *uuid.UUID `json:"device_id"`
	CandidateID     *uuid.UUID `json:"candidate_id"`
	Carrier         string     `json:"carrier"`
	SerialNumber    string     `json:"serial_number"`
	State           string     `json:"state"`
	ReasonCode      string     `json:"reason_code"`
	PolicyVersion   *uuid.UUID `json:"policy_version_id"`
	DecisionVersion int64      `json:"decision_version"`
	OccurredAt      time.Time  `json:"occurred_at"`
}

// DeviceAccessDecisionSubscriber records every terminal access decision in
// notification history even when no external channel is configured.
type DeviceAccessDecisionSubscriber struct {
	history *HistoryService
	logger  *zap.Logger
}

func NewDeviceAccessDecisionSubscriber(history *HistoryService, logger *zap.Logger) *DeviceAccessDecisionSubscriber {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DeviceAccessDecisionSubscriber{history: history, logger: logger.Named("notification-device-access-decision")}
}

func (s *DeviceAccessDecisionSubscriber) Subscribe(bus event.EventBus) error {
	if s == nil || s.history == nil || bus == nil {
		return fmt.Errorf("device access decision notification dependencies are required")
	}
	for subject, queue := range deviceAccessDecisionNotificationQueues {
		if _, err := bus.QueueSubscribe(subject, queue, s.handle); err != nil {
			return fmt.Errorf("subscribe device access decision notification %s: %w", subject, err)
		}
	}
	return nil
}

func (s *DeviceAccessDecisionSubscriber) handle(ctx context.Context, evt event.Event) error {
	var payload deviceAccessDecisionEvent
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device access decision notification: %w", err)
	}
	eventID, err := uuid.Parse(payload.EventID)
	if err != nil || payload.DecisionID == uuid.Nil || payload.EventName == "" {
		return fmt.Errorf("decode device access decision notification: invalid event identity")
	}
	body, err := json.Marshal(map[string]any{
		"carrier": payload.Carrier, "serial_number": payload.SerialNumber,
		"device_id": payload.DeviceID, "candidate_id": payload.CandidateID,
		"state": payload.State, "reason_code": payload.ReasonCode,
		"policy_version_id": payload.PolicyVersion, "decision_version": payload.DecisionVersion,
	})
	if err != nil {
		return fmt.Errorf("encode device access decision notification body: %w", err)
	}
	history := &NotificationHistory{
		Channel: HistoryChannelSystem, Recipients: []string{"device-access-operators"},
		Subject: payload.EventName, Body: string(body), Status: HistoryStatusNotConfigured,
		SourceType: "device_access_decision", SourceID: &payload.DecisionID, EventID: &eventID,
		CorrelationID: payload.RequestID, CreatedAt: payload.OccurredAt,
	}
	if history.CreatedAt.IsZero() {
		history.CreatedAt = evt.Timestamp.UTC()
	}
	if err := s.history.Insert(ctx, history); err != nil {
		s.logger.Warn("record device access decision notification history", zap.String("event_id", payload.EventID), zap.Error(err))
		return err
	}
	return nil
}
