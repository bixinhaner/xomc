package stream

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisWindowStoreAccumulateIsIdempotentAndComplete(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	}
	contribution := Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 1,
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network", DimensionName: "Network",
			MetricPath: "K001", MetricType: "kpi", Operation: AggregationAvg, Value: 12.5,
		}},
	}

	first, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)
	require.False(t, first.Duplicate)
	require.True(t, first.Complete)

	second, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)
	require.True(t, second.Duplicate)

	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.EqualValues(t, 1, state.ReceivedSlots)
	require.Len(t, state.Accumulators, 1)
	require.EqualValues(t, 1, state.Accumulators[0].Count)
	require.Equal(t, 12.5, accumulatorValue(state.Accumulators[0]))
}
