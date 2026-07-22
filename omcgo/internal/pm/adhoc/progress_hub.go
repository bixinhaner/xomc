package adhoc

import "sync"

const progressSubscriberBuffer = 1

// ProgressEvent is one SSE event ready for a local browser connection.
type ProgressEvent struct {
	Name string
	Data []byte
}

type progressTopic struct {
	subscribers map[*progressSubscriber]struct{}
}

type progressSubscriber struct {
	events    chan ProgressEvent
	completed bool
}

// ProgressHub fans transient task progress out to every local SSE connection.
type ProgressHub struct {
	mu     sync.Mutex
	topics map[string]*progressTopic
	closed bool
}

func NewProgressHub() *ProgressHub {
	return &ProgressHub{topics: make(map[string]*progressTopic)}
}

func (h *ProgressHub) Subscribe(taskID string) (<-chan ProgressEvent, func()) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		events := make(chan ProgressEvent)
		close(events)
		return events, func() {}
	}
	subscriber := &progressSubscriber{events: make(chan ProgressEvent, progressSubscriberBuffer)}
	topic := h.topics[taskID]
	if topic == nil {
		topic = &progressTopic{subscribers: make(map[*progressSubscriber]struct{})}
		h.topics[taskID] = topic
	}
	topic.subscribers[subscriber] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return subscriber.events, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			topic := h.topics[taskID]
			if topic == nil {
				return
			}
			if _, ok := topic.subscribers[subscriber]; !ok {
				return
			}
			delete(topic.subscribers, subscriber)
			close(subscriber.events)
			if len(topic.subscribers) == 0 {
				delete(h.topics, taskID)
			}
		})
	}
}

// Publish never waits for a browser. Progress keeps the newest buffered value;
// completed clears queued progress and prevents later progress from resurfacing.
func (h *ProgressHub) Publish(taskID string, event ProgressEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	topic := h.topics[taskID]
	if topic == nil {
		return
	}
	event.Data = append([]byte(nil), event.Data...)
	if event.Name == "completed" {
		for subscriber := range topic.subscribers {
			if subscriber.completed {
				continue
			}
			subscriber.completed = true
			drainProgressEvents(subscriber.events)
			select {
			case subscriber.events <- event:
			default:
			}
		}
		return
	}

	for subscriber := range topic.subscribers {
		if subscriber.completed {
			continue
		}
		select {
		case subscriber.events <- event:
		default:
			select {
			case <-subscriber.events:
			default:
			}
			select {
			case subscriber.events <- event:
			default:
			}
		}
	}
}

// Close atomically stops publication and closes every local subscriber.
func (h *ProgressHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for taskID, topic := range h.topics {
		for subscriber := range topic.subscribers {
			close(subscriber.events)
			delete(topic.subscribers, subscriber)
		}
		delete(h.topics, taskID)
	}
}

func drainProgressEvents(events chan ProgressEvent) {
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}
