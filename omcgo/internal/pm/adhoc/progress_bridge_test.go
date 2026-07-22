package adhoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/realtime"
)

type bridgeStubSubscription struct {
	unsubscribed bool
	unsubCalls   int
	err          error
}

func (s *bridgeStubSubscription) Unsubscribe() error {
	s.unsubscribed = true
	s.unsubCalls++
	return s.err
}

type bridgeStubRealtime struct {
	subjects    []string
	handlers    map[string]func([]byte)
	subs        []*bridgeStubSubscription
	failSubject string
	failErr     error
	nextSubErr  error
}

func (s *bridgeStubRealtime) Subscribe(subject string, handler func([]byte)) (realtime.Subscription, error) {
	s.subjects = append(s.subjects, subject)
	if subject == s.failSubject {
		if s.failErr != nil {
			return nil, s.failErr
		}
		return nil, errors.New("subscribe failed")
	}
	if s.handlers == nil {
		s.handlers = make(map[string]func([]byte))
	}
	s.handlers[subject] = handler
	sub := &bridgeStubSubscription{err: s.nextSubErr}
	s.nextSubErr = nil
	s.subs = append(s.subs, sub)
	return sub, nil
}

func TestProgressBridge_SubscribesOncePerRealtimeSubjectAndRoutesByTask(t *testing.T) {
	realtimeBus := &bridgeStubRealtime{}
	hub := NewProgressHub()
	bridge := NewProgressBridge(realtimeBus, hub, nil)

	stop, err := bridge.Subscribe()
	require.NoError(t, err)
	defer func() { require.NoError(t, stop()) }()
	assert.Equal(t, []string{SubjectRealtimeProgress, SubjectRealtimeCompleted}, realtimeBus.subjects)

	taskA, unsubscribeA := hub.Subscribe("task-a")
	defer unsubscribeA()
	taskB, unsubscribeB := hub.Subscribe("task-b")
	defer unsubscribeB()

	realtimeBus.handlers[SubjectRealtimeProgress]([]byte(`{"task_id":"task-a","progress":35}`))
	event := receiveProgressEvent(t, taskA)
	assert.Equal(t, "progress", event.Name)
	assert.JSONEq(t, `{"task_id":"task-a","progress":35}`, string(event.Data))
	select {
	case got := <-taskB:
		t.Fatalf("event leaked to another task: %+v", got)
	default:
	}
}

func TestProgressBridge_StopUnsubscribesBothUpstreamSubscriptions(t *testing.T) {
	realtimeBus := &bridgeStubRealtime{}
	hub := NewProgressHub()
	bridge := NewProgressBridge(realtimeBus, hub, nil)

	stop, err := bridge.Subscribe()
	require.NoError(t, err)
	require.NoError(t, stop())

	require.Len(t, realtimeBus.subs, 2)
	assert.True(t, realtimeBus.subs[0].unsubscribed)
	assert.True(t, realtimeBus.subs[1].unsubscribed)
	events, unsubscribe := hub.Subscribe("task-after-stop")
	defer unsubscribe()
	_, open := <-events
	assert.False(t, open)
}

func TestProgressBridge_SecondSubscriptionFailureCleansUpFirst(t *testing.T) {
	subscribeErr := errors.New("subscribe failed")
	rollbackErr := errors.New("rollback unsubscribe failed")
	realtimeBus := &bridgeStubRealtime{
		failSubject: SubjectRealtimeCompleted,
		failErr:     subscribeErr,
		nextSubErr:  rollbackErr,
	}
	hub := NewProgressHub()
	bridge := NewProgressBridge(realtimeBus, hub, nil)

	stop, err := bridge.Subscribe()
	require.Error(t, err)
	assert.ErrorIs(t, err, subscribeErr)
	assert.ErrorIs(t, err, rollbackErr)
	assert.Contains(t, err.Error(), "subscribe completed events")
	assert.Contains(t, err.Error(), "rollback progress subscription for realtime subject "+SubjectRealtimeProgress)
	assert.Nil(t, stop)
	require.Len(t, realtimeBus.subs, 1)
	assert.True(t, realtimeBus.subs[0].unsubscribed)
	events, unsubscribe := hub.Subscribe("task-after-failure")
	defer unsubscribe()
	_, open := <-events
	assert.False(t, open)
}

func TestProgressBridge_IgnoresMalformedOrMissingTaskIDPayload(t *testing.T) {
	realtimeBus := &bridgeStubRealtime{}
	hub := NewProgressHub()
	bridge := NewProgressBridge(realtimeBus, hub, nil)
	stop, err := bridge.Subscribe()
	require.NoError(t, err)
	defer func() { require.NoError(t, stop()) }()
	events, unsubscribe := hub.Subscribe("task-a")
	defer unsubscribe()

	realtimeBus.handlers[SubjectRealtimeProgress]([]byte(`not-json`))
	realtimeBus.handlers[SubjectRealtimeProgress]([]byte(`{"progress":35}`))

	select {
	case got := <-events:
		t.Fatalf("invalid payload was routed: %+v", got)
	default:
	}
}

func TestProgressBridge_CopiesRawPayloadBeforeReturningToNATSCallback(t *testing.T) {
	realtimeBus := &bridgeStubRealtime{}
	hub := NewProgressHub()
	bridge := NewProgressBridge(realtimeBus, hub, nil)
	stop, err := bridge.Subscribe()
	require.NoError(t, err)
	defer func() { require.NoError(t, stop()) }()
	events, unsubscribe := hub.Subscribe("task-a")
	defer unsubscribe()

	payload := []byte(`{"task_id":"task-a","progress":35}`)
	realtimeBus.handlers[SubjectRealtimeProgress](payload)
	copy(payload, []byte(`xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`))

	event := receiveProgressEvent(t, events)
	assert.JSONEq(t, `{"task_id":"task-a","progress":35}`, string(event.Data))
}

func TestProgressBridge_StopAttemptsBothSubscriptionsClosesHubAndIsIdempotent(t *testing.T) {
	progressErr := errors.New("progress unsubscribe failed")
	completedErr := errors.New("completed unsubscribe failed")
	realtimeBus := &bridgeStubRealtime{}
	hub := NewProgressHub()
	bridge := NewProgressBridge(realtimeBus, hub, nil)
	stop, err := bridge.Subscribe()
	require.NoError(t, err)
	require.Len(t, realtimeBus.subs, 2)
	realtimeBus.subs[0].err = progressErr
	realtimeBus.subs[1].err = completedErr

	err = stop()
	require.ErrorIs(t, err, progressErr)
	require.ErrorIs(t, err, completedErr)
	assert.Contains(t, err.Error(), "stop progress subscription for realtime subject "+SubjectRealtimeProgress)
	assert.Contains(t, err.Error(), "stop completed subscription for realtime subject "+SubjectRealtimeCompleted)
	assert.Equal(t, 1, realtimeBus.subs[0].unsubCalls)
	assert.Equal(t, 1, realtimeBus.subs[1].unsubCalls)
	events, unsubscribe := hub.Subscribe("task-after-stop-error")
	defer unsubscribe()
	_, open := <-events
	assert.False(t, open)

	require.ErrorIs(t, stop(), progressErr)
	require.ErrorIs(t, stop(), completedErr)
	assert.Equal(t, 1, realtimeBus.subs[0].unsubCalls)
	assert.Equal(t, 1, realtimeBus.subs[1].unsubCalls)
}
