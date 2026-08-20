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

func TestDeviceAccessActionSubscriberRecordsNotConfiguredAssociation(t *testing.T) {
	repo := newMemHistoryRepo()
	subscriber := NewDeviceAccessActionSubscriber(NewHistoryService(repo, zap.NewNop()), zap.NewNop())
	eventID := uuid.New()
	actionID := uuid.New()
	payload, err := json.Marshal(deviceAccessActionEvent{
		EventID: eventID.String(), EventName: "DEVICE_ACCESS_ACTION_FAILED",
		RequestID: "request-1", ActionID: actionID, DeviceID: uuidPointer(uuid.New()),
		Carrier: "cmcc", SerialNumber: "SN-1", ActionStatus: "dead",
		ReasonCode: "cwmp_timeout", OccurredAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	err = subscriber.handle(context.Background(), event.Event{Payload: payload, Timestamp: time.Now().UTC()})

	require.NoError(t, err)
	result, err := repo.List(context.Background(), NotificationHistoryFilter{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	history := result.Items[0]
	require.Equal(t, HistoryStatusNotConfigured, history.Status)
	require.Equal(t, HistoryChannelSystem, history.Channel)
	require.Equal(t, "device_access_action", history.SourceType)
	require.Equal(t, actionID, *history.SourceID)
	require.Equal(t, eventID, *history.EventID)
	require.Equal(t, "request-1", history.CorrelationID)
}

func TestDeviceAccessActionNotificationQueuesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(deviceAccessActionNotificationQueues))
	for subject, queue := range deviceAccessActionNotificationQueues {
		require.NotEmpty(t, subject)
		require.NotEmpty(t, queue)
		_, duplicate := seen[queue]
		require.False(t, duplicate, "queue %q is reused across subjects", queue)
		seen[queue] = struct{}{}
	}
}
