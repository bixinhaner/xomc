package main

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartRedisQueueObserverStartsSamplingInWorker(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	reg := prometheus.NewRegistry()
	metrics := task.NewTaskMetrics(reg)
	queueKey := redisx.Keys.ACSTaskQueue("worker-test-device")
	require.NoError(t, client.ZAdd(context.Background(), queueKey, redis.Z{
		Score:  1,
		Member: "task-1",
	}).Err())

	gs := components.NewGracefulShutdown(time.Second, zap.NewNop())
	observer := startRedisQueueObserver(client, metrics, gs, zap.NewNop())
	require.NotNil(t, observer)
	t.Cleanup(func() { _ = gs.Shutdown(context.Background()) })

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(metrics.RedisTaskQueueUp.WithLabelValues("taskq")) == 1 &&
			testutil.ToFloat64(metrics.RedisTaskQueueLengthTotal.WithLabelValues("taskq")) == 1
	}, time.Second, 10*time.Millisecond)
}
