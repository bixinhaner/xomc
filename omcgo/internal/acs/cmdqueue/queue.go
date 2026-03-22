package cmdqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Command represents a queued RPC command for a device.
type Command struct {
	ID         string          `json:"id"`
	Method     string          `json:"method"`
	Params     json.RawMessage `json:"params"`
	Priority   int             `json:"priority"` // lower = higher priority
	CreatedAt  time.Time       `json:"created_at"`
	ExpiresAt  *time.Time      `json:"expires_at,omitempty"`
	CommandKey string          `json:"command_key"`
	CWMPID     string          `json:"cwmp_id"` // SOAP Header cwmp:ID (distinct from CommandKey)
}

// CommandQueue defines the interface for device command queues.
type CommandQueue interface {
	Push(ctx context.Context, deviceSN string, cmd *Command) error
	Pop(ctx context.Context, deviceSN string) (*Command, error)
	Peek(ctx context.Context, deviceSN string) (*Command, error)
	Len(ctx context.Context, deviceSN string) (int64, error)
	Clear(ctx context.Context, deviceSN string) error
}

const cmdQueueKeyPrefix = "acs:cmdq:"

func queueKey(deviceSN string) string {
	return cmdQueueKeyPrefix + deviceSN
}

// RedisCommandQueue implements CommandQueue using Redis Sorted Sets.
type RedisCommandQueue struct {
	client redis.UniversalClient
}

// NewRedisCommandQueue creates a new Redis-backed command queue.
func NewRedisCommandQueue(client redis.UniversalClient) *RedisCommandQueue {
	return &RedisCommandQueue{client: client}
}

func (q *RedisCommandQueue) Push(ctx context.Context, deviceSN string, cmd *Command) error {
	if cmd.ID == "" {
		cmd.ID = uuid.New().String()
	}
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now()
	}
	if cmd.CommandKey == "" {
		cmd.CommandKey = cmd.ID
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	// Score: priority * 1e12 + timestamp for FIFO within same priority
	score := float64(cmd.Priority)*1e12 + float64(cmd.CreatedAt.UnixNano())/1e9

	return q.client.ZAdd(ctx, queueKey(deviceSN), redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

func (q *RedisCommandQueue) Pop(ctx context.Context, deviceSN string) (*Command, error) {
	key := queueKey(deviceSN)

	// Iterative loop to skip expired commands (max 100 to prevent infinite loops).
	for i := 0; i < 100; i++ {
		results, err := q.client.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err != nil {
			return nil, fmt.Errorf("peek command queue: %w", err)
		}
		if len(results) == 0 {
			return nil, nil
		}

		member := results[0].Member.(string)
		removed, err := q.client.ZRem(ctx, key, member).Result()
		if err != nil {
			return nil, fmt.Errorf("remove from command queue: %w", err)
		}
		if removed == 0 {
			continue // someone else popped it, try next
		}

		var cmd Command
		if err := json.Unmarshal([]byte(member), &cmd); err != nil {
			return nil, fmt.Errorf("unmarshal command: %w", err)
		}

		// Check expiration — skip expired commands, try next
		if cmd.ExpiresAt != nil && time.Now().After(*cmd.ExpiresAt) {
			continue
		}

		return &cmd, nil
	}

	return nil, nil // exhausted or all expired
}

func (q *RedisCommandQueue) Peek(ctx context.Context, deviceSN string) (*Command, error) {
	results, err := q.client.ZRangeWithScores(ctx, queueKey(deviceSN), 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("peek command queue: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	var cmd Command
	if err := json.Unmarshal([]byte(results[0].Member.(string)), &cmd); err != nil {
		return nil, fmt.Errorf("unmarshal command: %w", err)
	}
	return &cmd, nil
}

func (q *RedisCommandQueue) Len(ctx context.Context, deviceSN string) (int64, error) {
	return q.client.ZCard(ctx, queueKey(deviceSN)).Result()
}

func (q *RedisCommandQueue) Clear(ctx context.Context, deviceSN string) error {
	return q.client.Del(ctx, queueKey(deviceSN)).Err()
}
