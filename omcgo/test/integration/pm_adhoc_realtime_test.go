package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	gonats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/realtime"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
)

func TestPMAdhocRealtime_CoreNATSBroadcastsToEveryAPPInstance(t *testing.T) {
	natsURL := requireTestNATSURL(t)
	publisherConn := connectTestNATS(t, natsURL, "issue155-publisher")
	appAConn := connectTestNATS(t, natsURL, "issue155-app-a")
	appBConn := connectTestNATS(t, natsURL, "issue155-app-b")

	appAHub := adhoc.NewProgressHub()
	appABridge := adhoc.NewProgressBridge(realtime.NewCoreNATS(appAConn), appAHub, zap.NewNop())
	stopA, err := appABridge.Subscribe()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopA()) })

	appBHub := adhoc.NewProgressHub()
	appBBridge := adhoc.NewProgressBridge(realtime.NewCoreNATS(appBConn), appBHub, zap.NewNop())
	stopB, err := appBBridge.Subscribe()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopB()) })

	taskID := uuid.NewString()
	otherTaskID := uuid.NewString()
	appAEvents, unsubscribeA := appAHub.Subscribe(taskID)
	t.Cleanup(unsubscribeA)
	appBEvents, unsubscribeB := appBHub.Subscribe(taskID)
	t.Cleanup(unsubscribeB)
	appAOtherEvents, unsubscribeAOther := appAHub.Subscribe(otherTaskID)
	t.Cleanup(unsubscribeAOther)
	appBOtherEvents, unsubscribeBOther := appBHub.Subscribe(otherTaskID)
	t.Cleanup(unsubscribeBOther)

	publisher := realtime.NewCoreNATS(publisherConn)
	progressPayload := map[string]any{
		"task_id": taskID, "progress": 35, "granularity": "hourly", "rows": 7,
	}
	require.NoError(t, publisher.Publish(context.Background(), adhoc.SubjectRealtimeProgress, progressPayload))
	require.NoError(t, publisherConn.FlushTimeout(2*time.Second))
	assertRealtimePayload(t, receiveRealtimeEvent(t, appAEvents, "progress"), "progress", progressPayload)
	assertRealtimePayload(t, receiveRealtimeEvent(t, appBEvents, "progress"), "progress", progressPayload)

	otherTaskPayload := map[string]any{
		"task_id": otherTaskID, "progress": 90,
	}
	require.NoError(t, publisher.Publish(context.Background(), adhoc.SubjectRealtimeProgress, otherTaskPayload))
	require.NoError(t, publisherConn.FlushTimeout(2*time.Second))
	assertRealtimePayload(t, receiveRealtimeEvent(t, appAOtherEvents, "progress"), "progress", otherTaskPayload)
	assertRealtimePayload(t, receiveRealtimeEvent(t, appBOtherEvents, "progress"), "progress", otherTaskPayload)
	assertNoBufferedRealtimeEvent(t, appAEvents)
	assertNoBufferedRealtimeEvent(t, appBEvents)

	completedPayload := map[string]any{
		"task_id": taskID, "status": "succeeded", "rows_total": 42,
	}
	require.NoError(t, publisher.Publish(context.Background(), adhoc.SubjectRealtimeCompleted, completedPayload))
	require.NoError(t, publisherConn.FlushTimeout(2*time.Second))
	assertRealtimePayload(t, receiveRealtimeEvent(t, appAEvents, "completed"), "completed", completedPayload)
	assertRealtimePayload(t, receiveRealtimeEvent(t, appBEvents, "completed"), "completed", completedPayload)
}

func TestPMAdhocRealtime_BypassesPMWorkQueueWithoutChangingLegacyQueue(t *testing.T) {
	natsURL := requireTestNATSURL(t)
	conn := connectTestNATS(t, natsURL, "issue155-workqueue-probe")
	js, err := conn.JetStream()
	require.NoError(t, err)

	token := strings.ReplaceAll(uuid.NewString(), "-", "")
	streamName := "ISSUE155_PM_" + strings.ToUpper(token)
	_, err = js.AddStream(&gonats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"pm.>"},
		Retention: gonats.WorkQueuePolicy,
		Storage:   gonats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, js.DeleteStream(streamName)) })

	publisher := realtime.NewCoreNATS(conn)
	require.NoError(t, publisher.Publish(context.Background(), adhoc.SubjectRealtimeProgress, map[string]any{
		"task_id": uuid.NewString(), "progress": 10,
	}))
	require.NoError(t, conn.FlushTimeout(2*time.Second))
	streamInfo, err := js.StreamInfo(streamName)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), streamInfo.State.Msgs, "realtime subject must not enter pm.> WorkQueue")

	probeSubject := "pm.issue155.probe." + token
	_, err = js.Publish(probeSubject, []byte(`{"probe":true}`))
	require.NoError(t, err)
	durable := "issue155-probe-" + token
	sub, err := js.PullSubscribe(probeSubject, durable,
		gonats.BindStream(streamName), gonats.ManualAck(), gonats.AckExplicit())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sub.Unsubscribe()) })
	messages, err := sub.Fetch(1, gonats.MaxWait(2*time.Second))
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.JSONEq(t, `{"probe":true}`, string(messages[0].Data))
	require.NoError(t, messages[0].AckSync())
	require.Eventually(t, func() bool {
		info, infoErr := js.StreamInfo(streamName)
		return infoErr == nil && info.State.Msgs == 0
	}, 2*time.Second, 20*time.Millisecond, "acked WorkQueue message should be removed")
}

func requireTestNATSURL(t *testing.T) string {
	t.Helper()
	natsURL := os.Getenv("OMCGO_TEST_NATS_URL")
	if natsURL == "" {
		t.Skip("OMCGO_TEST_NATS_URL not set, skipping NATS integration test")
	}
	return natsURL
}

func connectTestNATS(t *testing.T, natsURL, name string) *gonats.Conn {
	t.Helper()
	conn, err := gonats.Connect(natsURL, gonats.Name(name+"-"+uuid.NewString()))
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	return conn
}

func receiveRealtimeEvent(t *testing.T, events <-chan adhoc.ProgressEvent, wantName string) adhoc.ProgressEvent {
	t.Helper()
	select {
	case event := <-events:
		require.Equal(t, wantName, event.Name)
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s event", wantName)
		return adhoc.ProgressEvent{}
	}
}

func assertNoBufferedRealtimeEvent(t *testing.T, events <-chan adhoc.ProgressEvent) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected cross-task event: %+v", event)
	default:
	}
}

func assertRealtimePayload(t *testing.T, event adhoc.ProgressEvent, wantName string, want map[string]any) {
	t.Helper()
	require.Equal(t, wantName, event.Name)
	var got map[string]any
	require.NoError(t, json.Unmarshal(event.Data, &got))
	assert.Equal(t, want["task_id"], got["task_id"])
	assert.NotContains(t, got, "payload", "realtime payload must not use the durable EventBus envelope")
	assert.NotContains(t, got, "subject", "realtime payload must not use the durable EventBus envelope")
	assert.NotContains(t, got, "id", "realtime payload must not use the durable EventBus envelope")
	wantJSON, err := json.Marshal(want)
	require.NoError(t, err)
	assert.JSONEq(t, string(wantJSON), string(event.Data), fmt.Sprintf("unexpected %s payload", wantName))
}
