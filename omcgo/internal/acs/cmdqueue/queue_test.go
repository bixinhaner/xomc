package cmdqueue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestQueue creates a RedisCommandQueue backed by a real Redis (miniredis not used;
// tests use localhost:6379). If Redis is unavailable, the test is skipped.
func newTestQueue(t *testing.T) (*RedisCommandQueue, func()) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable, skipping: %v", err)
	}
	q := NewRedisCommandQueue(client)
	cleanup := func() {
		// Clean up all keys we may have created in DB 15
		keys, _ := client.Keys(ctx, cmdQueueKeyPrefix+"*").Result()
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		client.Close()
	}
	return q, cleanup
}

func TestNewRedisCommandQueue(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	q := NewRedisCommandQueue(client)
	assert.NotNil(t, q)
	assert.Equal(t, client, q.client)
	client.Close()
}

func TestQueueKey(t *testing.T) {
	tests := []struct {
		name     string
		deviceSN string
		want     string
	}{
		{name: "normal serial", deviceSN: "ABC123", want: "acs:cmdq:ABC123"},
		{name: "empty serial", deviceSN: "", want: "acs:cmdq:"},
		{name: "serial with dots", deviceSN: "00:11:22:33", want: "acs:cmdq:00:11:22:33"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, queueKey(tt.deviceSN))
		})
	}
}

func TestRedisCommandQueue_Push_AutofillsDefaults(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	cmd := &Command{
		Method: "GetParameterValues",
		Params: json.RawMessage(`{"names":["Device."]}`),
	}

	err := q.Push(ctx, "test-dev-001", cmd)
	require.NoError(t, err)

	// ID, CreatedAt, CommandKey should be auto-filled
	assert.NotEmpty(t, cmd.ID, "ID should be auto-generated")
	assert.False(t, cmd.CreatedAt.IsZero(), "CreatedAt should be auto-set")
	assert.Equal(t, cmd.ID, cmd.CommandKey, "CommandKey should default to ID")
}

func TestRedisCommandQueue_Push_PreservesExplicitValues(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	cmd := &Command{
		ID:         "explicit-id",
		Method:     "Reboot",
		CreatedAt:  now,
		CommandKey: "explicit-key",
	}

	err := q.Push(ctx, "test-dev-002", cmd)
	require.NoError(t, err)

	assert.Equal(t, "explicit-id", cmd.ID)
	assert.Equal(t, now, cmd.CreatedAt)
	assert.Equal(t, "explicit-key", cmd.CommandKey)
}

func TestRedisCommandQueue_Pop_EmptyQueue(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	cmd, err := q.Pop(context.Background(), "nonexistent-dev")
	require.NoError(t, err)
	assert.Nil(t, cmd, "Pop on empty queue should return nil")
}

func TestRedisCommandQueue_Push_Pop_FIFO(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	deviceSN := "test-fifo-dev"

	// Push 3 commands with same priority but sequential timestamps
	for i, method := range []string{"GetParameterValues", "SetParameterValues", "Reboot"} {
		cmd := &Command{
			Method:    method,
			Priority:  1,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
		require.NoError(t, q.Push(ctx, deviceSN, cmd))
	}

	// Pop should return in FIFO order within same priority
	cmd1, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, "GetParameterValues", cmd1.Method)

	cmd2, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, "SetParameterValues", cmd2.Method)

	cmd3, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, "Reboot", cmd3.Method)

	// Queue should now be empty
	cmd4, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Nil(t, cmd4)
}

func TestRedisCommandQueue_PriorityOrdering(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	deviceSN := "test-priority-dev"
	baseTime := time.Now()

	// Push commands with different priorities (lower = higher priority)
	cmds := []struct {
		method   string
		priority int
	}{
		{"LowPriority", 10},
		{"HighPriority", 1},
		{"MediumPriority", 5},
	}

	for i, c := range cmds {
		cmd := &Command{
			Method:    c.method,
			Priority:  c.priority,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Millisecond),
		}
		require.NoError(t, q.Push(ctx, deviceSN, cmd))
	}

	// Should pop in priority order: 1, 5, 10
	first, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, "HighPriority", first.Method)

	second, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, "MediumPriority", second.Method)

	third, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, "LowPriority", third.Method)
}

func TestRedisCommandQueue_Pop_SkipsExpired(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	deviceSN := "test-expire-dev"

	pastTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	// Push an expired command
	expired := &Command{
		Method:    "Expired",
		Priority:  1,
		CreatedAt: time.Now(),
		ExpiresAt: &pastTime,
	}
	require.NoError(t, q.Push(ctx, deviceSN, expired))

	// Push a valid command
	valid := &Command{
		Method:    "Valid",
		Priority:  2, // lower priority but not expired
		CreatedAt: time.Now(),
		ExpiresAt: &futureTime,
	}
	require.NoError(t, q.Push(ctx, deviceSN, valid))

	// Pop should skip expired and return the valid command
	cmd, err := q.Pop(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "Valid", cmd.Method)
}

func TestRedisCommandQueue_Peek(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	deviceSN := "test-peek-dev"

	// Peek on empty queue
	cmd, err := q.Peek(ctx, deviceSN)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	// Push a command and peek
	require.NoError(t, q.Push(ctx, deviceSN, &Command{
		Method:   "GetParameterValues",
		Priority: 1,
	}))

	cmd, err = q.Peek(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "GetParameterValues", cmd.Method)

	// Peek should not remove the command
	cmd2, err := q.Peek(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, cmd2)
	assert.Equal(t, cmd.ID, cmd2.ID)
}

func TestRedisCommandQueue_Len(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	deviceSN := "test-len-dev"

	n, err := q.Len(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	require.NoError(t, q.Push(ctx, deviceSN, &Command{Method: "A", Priority: 1}))
	require.NoError(t, q.Push(ctx, deviceSN, &Command{Method: "B", Priority: 1}))

	n, err = q.Len(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestRedisCommandQueue_Clear(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	deviceSN := "test-clear-dev"

	require.NoError(t, q.Push(ctx, deviceSN, &Command{Method: "A", Priority: 1}))
	require.NoError(t, q.Push(ctx, deviceSN, &Command{Method: "B", Priority: 1}))

	err := q.Clear(ctx, deviceSN)
	require.NoError(t, err)

	n, err := q.Len(ctx, deviceSN)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestCommand_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	expires := now.Add(10 * time.Minute)
	original := &Command{
		ID:         "cmd-001",
		Method:     "SetParameterValues",
		Params:     json.RawMessage(`{"key":"value"}`),
		Priority:   5,
		CreatedAt:  now,
		ExpiresAt:  &expires,
		CommandKey: "ck-001",
		CWMPID:     "cwmp-001",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored Command
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Method, restored.Method)
	assert.Equal(t, original.Priority, restored.Priority)
	assert.Equal(t, original.CommandKey, restored.CommandKey)
	assert.Equal(t, original.CWMPID, restored.CWMPID)
	assert.JSONEq(t, string(original.Params), string(restored.Params))
}
