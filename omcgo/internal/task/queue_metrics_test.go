package task

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRedisQueueObserverCollectsBoundedBacklog(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	reg := prometheus.NewRegistry()
	metrics := NewTaskMetrics(reg)
	queuedAt := time.Now().Add(-2 * time.Minute).UTC()
	taskData, err := json.Marshal(&Task{ID: "task-1", CreatedAt: queuedAt})
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(context.Background(), redisx.Keys.ACSTaskQueue("device-1"), redis.Z{
		Score:  1,
		Member: "task-1",
	}).Err())
	require.NoError(t, client.HSet(context.Background(), redisx.Keys.ACSTaskDetail("task-1"), "data", taskData).Err())
	require.NoError(t, client.ZAdd(context.Background(), redisx.Keys.ACSCommandQueue("device-2"), redis.Z{
		Score:  1,
		Member: `{"method":"GetParameterValues"}`,
	}).Err())

	observer := NewRedisQueueObserver(client, metrics, time.Hour, zap.NewNop())
	observer.Collect(context.Background())

	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueLengthTotal.WithLabelValues(redisQueueFamilyTask)))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueActiveDevices.WithLabelValues(redisQueueFamilyTask)))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueMaxLength.WithLabelValues(redisQueueFamilyTask)))
	require.Greater(t, testutil.ToFloat64(metrics.RedisTaskQueueOldestAgeSeconds.WithLabelValues(redisQueueFamilyTask)), float64(100))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueUp.WithLabelValues(redisQueueFamilyTask)))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueLengthTotal.WithLabelValues(redisQueueFamilyCommand)))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueActiveDevices.WithLabelValues(redisQueueFamilyCommand)))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueUp.WithLabelValues(redisQueueFamilyCommand)))
}

func TestRedisQueueObserverFailurePreservesBusinessGauges(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()
	reg := prometheus.NewRegistry()
	metrics := NewTaskMetrics(reg)
	observer := NewRedisQueueObserver(client, metrics, time.Hour, zap.NewNop())

	key := redisx.Keys.ACSTaskQueue("device-1")
	require.NoError(t, client.ZAdd(context.Background(), key, redis.Z{Score: 1, Member: "task-1"}).Err())
	observer.Collect(context.Background())
	before := testutil.ToFloat64(metrics.RedisTaskQueueLengthTotal.WithLabelValues(redisQueueFamilyTask))
	require.Equal(t, float64(1), before)

	mini.Close()
	observer.Collect(context.Background())

	require.Equal(t, before, testutil.ToFloat64(metrics.RedisTaskQueueLengthTotal.WithLabelValues(redisQueueFamilyTask)))
	require.Equal(t, float64(0), testutil.ToFloat64(metrics.RedisTaskQueueUp.WithLabelValues(redisQueueFamilyTask)))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RedisTaskQueueScanFailuresTotal.WithLabelValues(redisQueueFamilyTask)))
}
