package paramsync

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
)

type terminalBridgeSubscriptionBus struct {
	event.EventBus
	queueSubjects []string
	pullSubjects  []string
}

func (b *terminalBridgeSubscriptionBus) QueueSubscribe(subject string, _ string, _ event.EventHandler) (event.Subscription, error) {
	b.queueSubjects = append(b.queueSubjects, subject)
	return noopSubscription{}, nil
}

func (b *terminalBridgeSubscriptionBus) PullSubscribe(subject string, _ string, _ event.EventHandler) (event.Subscription, error) {
	b.pullSubjects = append(b.pullSubjects, subject)
	return noopSubscription{}, nil
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error { return nil }

func TestTaskTerminalBridgeUsesPullSubscriptions(t *testing.T) {
	bus := &terminalBridgeSubscriptionBus{}
	bridge := NewTaskTerminalBridge(bus)

	require.NoError(t, bridge.Start())

	assert.Empty(t, bus.queueSubjects)
	assert.ElementsMatch(t, []string{
		event.SubjectTaskCompleted,
		event.SubjectTaskFailed,
		event.SubjectTaskCancelled,
	}, bus.pullSubjects)
}

func TestTaskTerminalBridgePublishesExpiredResultWithoutRawPayload(t *testing.T) {
	bus := event.NewChannelEventBus(8, zap.NewNop())
	bridge := NewTaskTerminalBridge(bus)
	received := make(chan event.ParamSyncTaskResultPayload, 1)
	rawCanonical := make(chan []byte, 1)
	_, err := bus.Subscribe(event.SubjectParamSyncTaskResult, func(_ context.Context, evt event.Event) error {
		var payload event.ParamSyncTaskResultPayload
		_ = evt.DecodePayload(&payload)
		received <- payload
		rawCanonical <- evt.Payload
		return nil
	})
	require.NoError(t, err)

	terminal := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: "SN-1", Method: "GetParameterValues", Source: task.TaskSourceParamSync,
		SourceID: uuid.NewString(), CreatorID: uuid.NewString(),
	})
	terminal.Status = task.TaskStatusExpired
	terminal.Result = []byte(`{"raw_response":"must-not-leak"}`)
	evt, err := event.NewEvent(event.SubjectTaskFailed, terminal)
	require.NoError(t, err)
	require.NoError(t, bridge.Handle(context.Background(), evt))

	got := <-received
	assert.False(t, got.Success)
	assert.Equal(t, "device_tasks:"+terminal.ID, got.ResultRef)
	assert.Empty(t, got.ErrorCode)
	assert.NotContains(t, string(<-rawCanonical), "must-not-leak")
}
