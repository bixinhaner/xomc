package task

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisTaskQueue_TerminalTombstoneRealRedisMemoryBound(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR is not configured")
	}
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Ping(ctx).Err())

	q := NewRedisTaskQueueWithTerminalTTL(client, 15*time.Minute)
	task := newTaskForQueue(
		"terminal-memory-"+strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-"),
		"SN-TERMINAL-MEMORY",
		"GetParameterValues",
	)
	params, err := json.Marshal(map[string]string{"payload": strings.Repeat("p", 32*1024)})
	require.NoError(t, err)
	task.Params = params
	require.NoError(t, q.Push(ctx, task))
	t.Cleanup(func() {
		_ = client.Del(ctx, q.taskKey(task.ID), q.queueKey(task.DeviceSN)).Err()
	})

	require.NoError(t, q.MarkTaskCompleted(
		ctx, task.ID,
		json.RawMessage(`{"payload":"`+strings.Repeat("r", 32*1024)+`"}`),
	))
	require.Equal(t, []string{"status"}, client.HKeys(ctx, q.taskKey(task.ID)).Val())
	require.Equal(t, string(TaskStatusCompleted), client.HGet(ctx, q.taskKey(task.ID), "status").Val())

	usage, err := client.MemoryUsage(ctx, q.taskKey(task.ID), 0).Result()
	require.NoError(t, err)
	t.Logf("real Redis terminal tombstone memory usage: %d bytes", usage)
	require.LessOrEqual(t, usage, int64(512),
		"a production Redis terminal fence must remain within the capacity model")
}
