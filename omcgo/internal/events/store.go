package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// MessageStore persists SSE messages for reconnection replay.
type MessageStore interface {
	Store(ctx context.Context, userID string, msg *SSEMessage) error
	GetSince(ctx context.Context, userID string, afterID string, limit int) ([]*SSEMessage, error)
}

// RedisMessageStore implements MessageStore using a Redis Sorted Set per user.
// Key format: sse:pending:{userID}
// Score: timestamp nanoseconds; Member: SSEMessage JSON.
type RedisMessageStore struct {
	client redis.UniversalClient
	ttl    time.Duration
	logger *zap.Logger
}

// NewRedisMessageStore creates a new RedisMessageStore with the given TTL.
func NewRedisMessageStore(client redis.UniversalClient, ttl time.Duration) *RedisMessageStore {
	return &RedisMessageStore{
		client: client,
		ttl:    ttl,
		logger: zap.NewNop(),
	}
}

// SetLogger allows injecting a logger after construction.
func (s *RedisMessageStore) SetLogger(logger *zap.Logger) {
	s.logger = logger.Named("sse-store")
}

// Store persists an SSE message to the user's pending queue.
func (s *RedisMessageStore) Store(ctx context.Context, userID string, msg *SSEMessage) error {
	key := redisx.Keys.SSEPending(userID)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal SSE message: %w", err)
	}

	score := float64(time.Now().UnixNano())

	pipe := s.client.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: string(data)})
	pipe.Expire(ctx, key, s.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis ZAdd SSE message: %w", err)
	}
	return nil
}

// GetSince replays messages after the given ID for reconnection.
// It retrieves up to `limit` messages sorted by timestamp.
func (s *RedisMessageStore) GetSince(ctx context.Context, userID string, afterID string, limit int) ([]*SSEMessage, error) {
	key := redisx.Keys.SSEPending(userID)

	// Fetch all recent messages (sorted by score = timestamp)
	results, err := s.client.ZRevRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ZRevRange SSE messages: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Parse messages and filter: only return messages with ID after afterID
	var messages []*SSEMessage
	found := afterID == "" // if no afterID, return all
	for i := len(results) - 1; i >= 0; i-- {
		var msg SSEMessage
		if err := json.Unmarshal([]byte(results[i]), &msg); err != nil {
			s.logger.Warn("failed to unmarshal SSE message from Redis",
				zap.String("user_id", userID),
				zap.Error(err),
			)
			continue
		}
		if !found {
			if msg.ID == afterID {
				found = true
			}
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}
