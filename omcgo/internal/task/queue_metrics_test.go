package task

import (
	"context"
	"encoding/json"
	"fmt"
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

type scanCountingRedisClient struct {
	redis.UniversalClient
	scans       int
	directTypes int
}

func (c *scanCountingRedisClient) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	c.scans++
	return c.UniversalClient.Scan(ctx, cursor, match, count)
}

func (c *scanCountingRedisClient) Type(ctx context.Context, key string) *redis.StatusCmd {
	c.directTypes++
	return c.UniversalClient.Type(ctx, key)
}

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

func TestRedisQueueObserverScansBothFamiliesInOneKeyspacePass(t *testing.T) {
	mini := miniredis.RunT(t)
	base := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer base.Close()
	client := &scanCountingRedisClient{UniversalClient: base}
	require.NoError(t, base.ZAdd(context.Background(), redisx.Keys.ACSTaskQueue("device-1"), redis.Z{Score: 1, Member: "task-1"}).Err())
	require.NoError(t, base.ZAdd(context.Background(), redisx.Keys.ACSCommandQueue("device-2"), redis.Z{Score: 1, Member: `{}`}).Err())

	observer := NewRedisQueueObserver(client, NewTaskMetrics(prometheus.NewRegistry()), time.Hour, zap.NewNop())
	observer.Collect(context.Background())

	require.Equal(t, 1, client.scans, "a small keyspace should require one combined SCAN call, not one pass per family")
	require.Zero(t, client.directTypes, "queue metadata must be pipelined instead of fetched one key per round trip")
	require.GreaterOrEqual(t, observer.scanCount, int64(10_000))
}

func TestRedisQueueObserverUsesQueueScoreWhenTaskDetailExpired(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	reg := prometheus.NewRegistry()
	metrics := NewTaskMetrics(reg)
	// Keep the task older than the priority component in queueScore. In
	// production a missing detail hash means the task has already outlived its
	// Redis TTL, so this also matches the real stale-backlog shape.
	queuedAt := time.Now().Add(-48 * time.Hour).UTC()
	// The queue score is retained after the task-detail hash expires. This is
	// the production failure mode: backlog remains visible but age must not be
	// reported as zero just because HGET returns redis.Nil.
	score := queueScore(&Task{Priority: 3, CreatedAt: queuedAt})
	require.NoError(t, client.ZAdd(context.Background(), redisx.Keys.ACSTaskQueue("device-expired-detail"), redis.Z{
		Score:  score,
		Member: "task-detail-expired",
	}).Err())

	observer := NewRedisQueueObserver(client, metrics, time.Hour, zap.NewNop())
	observer.Collect(context.Background())

	require.Greater(t,
		testutil.ToFloat64(metrics.RedisTaskQueueOldestAgeSeconds.WithLabelValues(redisQueueFamilyTask)),
		float64(100),
		"queue score should provide oldest age when task detail is unavailable")
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

func TestDisableRedisQueueObservationRemovesPrimedWorkerSeries(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewTaskMetrics(reg)

	metrics.DisableRedisQueueObservation()

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.NotContains(t, family.GetName(), "omc_redis_task_queue_",
			"a process that does not own observation must not export false up=0 series")
	}
}

func TestRedisQueueObserverScansMoreThanLegacyTenThousandKeyLimit(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()
	ctx := context.Background()

	pipe := client.Pipeline()
	for i := range 10_001 {
		pipe.ZAdd(ctx, redisx.Keys.ACSCommandQueue(fmt.Sprintf("device-%05d", i)), redis.Z{
			Score:  1,
			Member: `{"method":"GetParameterValues"}`,
		})
	}
	_, err := pipe.Exec(ctx)
	require.NoError(t, err)

	observer := NewRedisQueueObserver(client, NewTaskMetrics(prometheus.NewRegistry()), time.Hour, zap.NewNop())
	observer.scanCount = 20_000
	snapshot, err := observer.scanFamily(ctx, redisQueueFamilyCommand, redisx.Keys.ACSCommandQueuePattern())

	require.NoError(t, err)
	require.Equal(t, int64(10_001), snapshot.length)
	require.Equal(t, int64(10_001), snapshot.active)
}

func TestRedisQueueObserverIgnoresQueueDeletedAfterScan(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()
	observer := NewRedisQueueObserver(client, NewTaskMetrics(prometheus.NewRegistry()), time.Hour, zap.NewNop())

	length, oldest, known, err := observer.inspectQueue(
		context.Background(),
		redisQueueFamilyTask,
		redisx.Keys.ACSTaskQueue("already-deleted"),
		time.Now(),
	)

	require.NoError(t, err)
	require.Zero(t, length)
	require.Zero(t, oldest)
	require.False(t, known)
}

func TestRedisQueueObserverRejectsUnexpectedQueueType(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()
	key := redisx.Keys.ACSTaskQueue("corrupt")
	require.NoError(t, client.Set(context.Background(), key, "not-a-queue", 0).Err())
	observer := NewRedisQueueObserver(client, NewTaskMetrics(prometheus.NewRegistry()), time.Hour, zap.NewNop())

	_, _, _, err := observer.inspectQueue(
		context.Background(),
		redisQueueFamilyTask,
		key,
		time.Now(),
	)

	require.ErrorContains(t, err, `unsupported redis type "string"`)
}
