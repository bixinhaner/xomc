package adhoc

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressHub_BroadcastsToEverySubscriberForTheTask(t *testing.T) {
	hub := NewProgressHub()
	first, unsubscribeFirst := hub.Subscribe("task-a")
	defer unsubscribeFirst()
	second, unsubscribeSecond := hub.Subscribe("task-a")
	defer unsubscribeSecond()
	other, unsubscribeOther := hub.Subscribe("task-b")
	defer unsubscribeOther()

	want := ProgressEvent{Name: "progress", Data: []byte(`{"task_id":"task-a","progress":35}`)}
	hub.Publish("task-a", want)

	require.Equal(t, want, receiveProgressEvent(t, first))
	require.Equal(t, want, receiveProgressEvent(t, second))
	select {
	case got := <-other:
		t.Fatalf("different task received event: %+v", got)
	default:
	}
}

func TestProgressHub_SlowSubscriberDoesNotBlockAndCompletedSupersedesProgress(t *testing.T) {
	hub := NewProgressHub()
	events, unsubscribe := hub.Subscribe("task-a")
	defer unsubscribe()

	for i := 0; i < 100; i++ {
		hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":10}`)})
	}
	completed := ProgressEvent{Name: "completed", Data: []byte(`{"status":"succeeded"}`)}
	done := make(chan struct{})
	go func() {
		hub.Publish("task-a", completed)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow subscriber blocked publisher")
	}

	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":20}`)})
	assert.Equal(t, completed, receiveProgressEvent(t, events))
	select {
	case got := <-events:
		t.Fatalf("progress was delivered after completed: %+v", got)
	default:
	}

	nextRun, unsubscribeNextRun := hub.Subscribe("task-a")
	defer unsubscribeNextRun()
	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":5}`)})
	assert.Equal(t, "progress", receiveProgressEvent(t, nextRun).Name,
		"a new connection for the next continuous run must receive progress")
}

func TestProgressHub_SlowSubscriberKeepsOnlyNewestProgress(t *testing.T) {
	hub := NewProgressHub()
	events, unsubscribe := hub.Subscribe("task-a")
	defer unsubscribe()

	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":10}`)})
	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":20}`)})
	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":30}`)})

	latest := receiveProgressEvent(t, events)
	assert.JSONEq(t, `{"progress":30}`, string(latest.Data))
	select {
	case got := <-events:
		t.Fatalf("slow subscriber retained stale progress: %+v", got)
	default:
	}
}

func TestProgressHub_CloseClosesSubscribersAndRejectsFurtherEvents(t *testing.T) {
	hub := NewProgressHub()
	events, unsubscribe := hub.Subscribe("task-a")

	hub.Close()
	unsubscribe()
	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":50}`)})

	_, open := <-events
	assert.False(t, open)
	afterClose, unsubscribeAfterClose := hub.Subscribe("task-a")
	defer unsubscribeAfterClose()
	_, open = <-afterClose
	assert.False(t, open)
}

func TestProgressHub_ConcurrentConsumerPublishAndCloseCompleteWithinBound(t *testing.T) {
	hub := NewProgressHub()
	events, unsubscribe := hub.Subscribe("task-a")
	defer unsubscribe()

	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		for range events {
		}
	}()

	const publishers = 8
	const messagesPerPublisher = 10000
	start := make(chan struct{})
	var publishWG sync.WaitGroup
	publishWG.Add(publishers)
	for publisher := 0; publisher < publishers; publisher++ {
		go func(publisher int) {
			defer publishWG.Done()
			<-start
			for message := 0; message < messagesPerPublisher; message++ {
				hub.Publish("task-a", ProgressEvent{
					Name: "progress",
					Data: []byte(fmt.Sprintf(`{"publisher":%d,"message":%d}`, publisher, message)),
				})
			}
		}(publisher)
	}
	close(start)

	published := make(chan struct{})
	go func() {
		publishWG.Wait()
		close(published)
	}()
	select {
	case <-published:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent Publish blocked")
	}

	closed := make(chan struct{})
	go func() {
		hub.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close blocked after concurrent publication")
	}
	select {
	case <-consumerDone:
	case <-time.After(time.Second):
		t.Fatal("consumer did not observe Hub closure")
	}
}

func TestProgressHub_UnsubscribeRemovesOnlyThatConnection(t *testing.T) {
	hub := NewProgressHub()
	first, unsubscribeFirst := hub.Subscribe("task-a")
	second, unsubscribeSecond := hub.Subscribe("task-a")
	defer unsubscribeSecond()

	unsubscribeFirst()
	hub.Publish("task-a", ProgressEvent{Name: "progress", Data: []byte(`{"progress":50}`)})

	_, open := <-first
	assert.False(t, open)
	assert.Equal(t, "progress", receiveProgressEvent(t, second).Name)
}

func receiveProgressEvent(t *testing.T, events <-chan ProgressEvent) ProgressEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for progress event")
		return ProgressEvent{}
	}
}
