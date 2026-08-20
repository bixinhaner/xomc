package notification

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDeviceAccessDecisionSubscriberRecordsNotConfiguredAssociation(t *testing.T) {
	repo := newMemHistoryRepo()
	subscriber := NewDeviceAccessDecisionSubscriber(NewHistoryService(repo, zap.NewNop()), zap.NewNop())
	eventID := uuid.New()
	decisionID := uuid.New()
	payload, err := json.Marshal(deviceAccessDecisionEvent{
		EventID: eventID.String(), EventName: "DEVICE_ACCESS_ACCEPTED",
		RequestID: "request-1", DecisionID: decisionID, DeviceID: uuidPointer(uuid.New()),
		Carrier: "cmcc", SerialNumber: "SN-1", State: "accepted",
		ReasonCode: "allowlist_matched", OccurredAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	err = subscriber.handle(context.Background(), event.Event{Payload: payload, Timestamp: time.Now().UTC()})

	require.NoError(t, err)
	result, err := repo.List(context.Background(), NotificationHistoryFilter{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	history := result.Items[0]
	require.Equal(t, HistoryStatusNotConfigured, history.Status)
	require.Equal(t, "device_access_decision", history.SourceType)
	require.Equal(t, decisionID, *history.SourceID)
	require.Equal(t, eventID, *history.EventID)
	require.Equal(t, "request-1", history.CorrelationID)
}

func TestDeviceAccessDecisionNotificationQueuesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(deviceAccessDecisionNotificationQueues))
	for subject, queue := range deviceAccessDecisionNotificationQueues {
		require.NotEmpty(t, subject)
		require.NotEmpty(t, queue)
		_, duplicate := seen[queue]
		require.False(t, duplicate, "queue %q is reused across subjects", queue)
		seen[queue] = struct{}{}
	}
}

func uuidPointer(value uuid.UUID) *uuid.UUID { return &value }
