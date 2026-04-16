// Package events provides SSE (Server-Sent Events) infrastructure for
// real-time push notifications to authenticated users.
package events

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SSEMessage represents a single Server-Sent Event payload.
type SSEMessage struct {
	ID     string          `json:"id"`
	Event  string          `json:"event"`
	Data   json.RawMessage `json:"data"`
	UserID string          `json:"-"` // not serialized; used for routing
}

// UserChannel holds a per-user SSE subscription channel.
type UserChannel struct {
	userID    string
	ch        chan *SSEMessage
	lastAckID string
	created   time.Time
}

// MessageHub manages SSE channels for all connected users.
// It is safe for concurrent use via sync.RWMutex.
type MessageHub struct {
	mu       sync.RWMutex
	channels map[string]*UserChannel
	store    MessageStore
	logger   *zap.Logger
}

// NewMessageHub creates a new MessageHub with the given persistent store.
func NewMessageHub(store MessageStore, logger *zap.Logger) *MessageHub {
	return &MessageHub{
		channels: make(map[string]*UserChannel),
		store:    store,
		logger:   logger.Named("sse-hub"),
	}
}

// Publish sends a message to a specific user's channel.
// If the channel is full, the oldest message is dropped to make room.
// The message is also persisted to the store for reconnection replay.
func (h *MessageHub) Publish(userID string, msg *SSEMessage) error {
	msg.UserID = userID

	// Persist for reconnection replay
	if h.store != nil {
		if err := h.store.Store(context.Background(), userID, msg); err != nil {
			h.logger.Warn("failed to persist SSE message",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	h.mu.RLock()
	uc, ok := h.channels[userID]
	h.mu.RUnlock()

	if !ok {
		h.logger.Debug("no active SSE channel for user, message persisted only",
			zap.String("user_id", userID),
		)
		return nil
	}

	select {
	case uc.ch <- msg:
	default:
		// Channel full: drop oldest to make room
		select {
		case <-uc.ch:
		default:
		}
		uc.ch <- msg
		h.logger.Warn("SSE channel full, dropped oldest message",
			zap.String("user_id", userID),
		)
	}
	return nil
}

// PublishSimple is a convenience method that constructs an SSEMessage and publishes it.
func (h *MessageHub) PublishSimple(userID, eventType string, data []byte) {
	msg := &SSEMessage{
		ID:     uuid.New().String(),
		Event:  eventType,
		Data:   data,
		UserID: userID,
	}
	if err := h.Publish(userID, msg); err != nil {
		h.logger.Warn("publish simple SSE event failed",
			zap.String("user_id", userID),
			zap.String("event", eventType),
			zap.Error(err),
		)
	}
}

// PublishGlobal sends a message to ALL connected users' channels.
func (h *MessageHub) PublishGlobal(msg *SSEMessage) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for userID := range h.channels {
		uc := h.channels[userID]
		select {
		case uc.ch <- msg:
		default:
			select {
			case <-uc.ch:
			default:
			}
			uc.ch <- msg
		}
	}
	return nil
}

// Subscribe creates a new SSE channel for the given user.
// If the user already has an active channel (e.g. from another tab),
// the existing channel is closed and replaced.
func (h *MessageHub) Subscribe(userID string) (<-chan *SSEMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Kick existing connection for same user
	if existing, ok := h.channels[userID]; ok {
		close(existing.ch)
		delete(h.channels, userID)
		h.logger.Info("kicked existing SSE connection",
			zap.String("user_id", userID),
		)
	}

	uc := &UserChannel{
		userID:  userID,
		ch:      make(chan *SSEMessage, 256),
		created: time.Now(),
	}
	h.channels[userID] = uc

	h.logger.Info("SSE channel subscribed",
		zap.String("user_id", userID),
		zap.Int("total_channels", len(h.channels)),
	)
	return uc.ch, nil
}

// Unsubscribe removes the SSE channel for the given user.
func (h *MessageHub) Unsubscribe(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	uc, ok := h.channels[userID]
	if !ok {
		return
	}
	close(uc.ch)
	delete(h.channels, userID)

	h.logger.Info("SSE channel unsubscribed",
		zap.String("user_id", userID),
		zap.Int("total_channels", len(h.channels)),
	)
}
