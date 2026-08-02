package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWorkerRedisRoutingKeepsPMAndKPIRouteKeysOffCore(t *testing.T) {
	coreServer := miniredis.RunT(t)
	pmServer := miniredis.RunT(t)
	coreClient := redis.NewClient(&redis.Options{Addr: coreServer.Addr()})
	pmClient := redis.NewClient(&redis.Options{Addr: pmServer.Addr()})
	t.Cleanup(func() {
		_ = coreClient.Close()
		_ = pmClient.Close()
	})
	w := &workerInfra{Infra: &components.Infra{
		Redis: coreClient, PMRedis: pmClient, Logger: zap.NewNop(),
	}}
	ctx := context.Background()

	l2 := newWorkerKPIRouteL2(w)
	require.NotNil(t, l2)
	productID := uuid.New()
	require.NoError(t, l2.Put(ctx, &router.KPIRoute{ProductID: productID}, 0))
	appBumper := router.NewRedisCache(pmClient)
	_, err := appBumper.BumpVersion(ctx)
	require.NoError(t, err)
	got, err := l2.Get(ctx, productID)
	require.NoError(t, err)
	require.Nil(t, got, "app version bump must invalidate the worker PM Redis entry")

	store := newWorkerPMWindowStore(w, time.Hour, nil)
	start := time.Date(2026, 8, 2, 15, 0, 0, 0, time.UTC)
	_, err = store.Accumulate(ctx, pmstream.Contribution{
		Key: pmstream.WindowKey{
			TaskID: uuid.New(), TaskVersionID: uuid.New(), EntityKey: "network",
			Granularity: pmstream.GranularityHourly, Start: start, End: start.Add(time.Hour),
		},
		SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(), SlotStart: start,
		ExpectedSlots: 4,
		Values: []pmstream.ContributionValue{{
			Dimension: pmstream.DimensionNetwork, DimensionKey: "network",
			MetricPath: "C1", MetricType: "counter", Operation: pmstream.AggregationSum, Value: 1,
		}},
	})
	require.NoError(t, err)

	require.Empty(t, coreServer.Keys(), "core Redis must not receive PM/KPI keys")
	pmKeys := pmServer.Keys()
	require.NotEmpty(t, pmKeys)
	require.True(t, containsKeyPrefix(pmKeys, "kpi-route:"))
	require.True(t, containsKeyPrefix(pmKeys, "pmagg:"))
}

func containsKeyPrefix(keys []string, prefix string) bool {
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}
