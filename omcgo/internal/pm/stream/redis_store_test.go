package stream

import (
	"context"
	"strings"
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
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	contribution := Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 1,
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network", DimensionName: "Network",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationAvg, Value: 12.5,
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

	keys := redisKeys(key, 1)
	fields, err := server.HKeys(keys.acc[0])
	require.NoError(t, err)
	require.Len(t, fields, 1)
	require.Less(t, len(fields[0]), 40, "accumulator fields must use compact IDs")
	require.False(t, server.Exists(keys.defs[0]))
}

func TestRedisWindowStoreUsesOneCompactHashFieldPerMetric(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 29, 1, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	contribution := Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 1,
		Values: []ContributionValue{
			{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 12.5,
			},
			{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C002", MetricType: "counter", Operation: AggregationMax, Value: 7,
			},
		},
	}

	_, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)

	keys := redisKeys(key, 1)
	fields, err := server.HKeys(keys.acc[0])
	require.NoError(t, err)
	require.Len(t, fields, len(contribution.Values))
	for _, field := range fields {
		require.NotContains(t, field, "|sum")
		require.NotContains(t, field, "|count")
		require.NotContains(t, field, "|min")
		require.NotContains(t, field, "|max")
		raw := server.HGet(keys.acc[0], field)
		require.True(t, strings.HasPrefix(raw, compactAccumulatorVersion+"|"))
	}
	require.False(t, server.Exists(keys.defs[0]),
		"new compact windows must not allocate a separate definitions hash")
}

func TestRedisWindowStoreMigratesLegacyAccumulatorOnFirstWrite(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 29, 2, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	value := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C001", MetricType: "counter", Operation: AggregationSum,
	}
	id, err := accumulatorDefinitionID(value)
	require.NoError(t, err)
	encoded, err := encodeDefinition(value)
	require.NoError(t, err)
	keys := redisKeys(key, 1)
	require.NoError(t, client.HSet(context.Background(), keys.meta, map[string]any{
		"expected_slots": 1, "received_slots": 0, "source_expected_slots": 1,
		"source_received_slots": 0, "source_incomplete_slots": 0, "shard_count": 1,
	}).Err())
	require.NoError(t, client.HSet(context.Background(), keys.defs[0], id, encoded).Err())
	require.NoError(t, client.HSet(context.Background(), keys.acc[0], map[string]any{
		id + "|sum": 10, id + "|count": 2, id + "|min": 3, id + "|max": 7,
	}).Err())

	before, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, before.Accumulators, 1)
	require.Equal(t, 10.0, before.Accumulators[0].Sum)
	require.EqualValues(t, 2, before.Accumulators[0].Count)

	value.Value = 2
	_, err = store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 1, Values: []ContributionValue{value},
	})
	require.NoError(t, err)

	after, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, after.Accumulators, 1)
	require.Equal(t, 12.0, after.Accumulators[0].Sum)
	require.EqualValues(t, 3, after.Accumulators[0].Count)
	require.Equal(t, 2.0, after.Accumulators[0].Min)
	require.Equal(t, 7.0, after.Accumulators[0].Max)

	fields, err := server.HKeys(keys.acc[0])
	require.NoError(t, err)
	require.Equal(t, []string{id}, fields)
	raw := server.HGet(keys.acc[0], id)
	require.True(t, strings.HasPrefix(raw, compactAccumulatorVersion+"|"))
	require.False(t, server.Exists(keys.defs[0]))
}

func TestRedisWindowStoreCountsRollupSlotAfterAllChunks(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityDaily,
		EntityKey: "network", Start: start, End: start.AddDate(0, 0, 1),
	}
	contribution := Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: key.TaskVersionID.String(),
		SlotStart: start, ExpectedSlots: 2,
		SourceExpectedSlots: 4, SourceReceivedSlots: 4,
		Rollup: true, RollupChunkIndex: 0, RollupChunkCount: 2,
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum,
			Sum: 10, Count: 1, Min: 10, Max: 10, Composed: true,
		}},
	}
	first, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)
	require.False(t, first.Complete)
	require.Zero(t, first.ReceivedSlots)

	contribution.SourceFileID = uuid.NewString()
	contribution.RollupChunkIndex = 1
	contribution.Values[0].Sum = 20
	contribution.Values[0].Min = 20
	contribution.Values[0].Max = 20
	second, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)
	require.False(t, second.Complete)
	require.EqualValues(t, 1, second.ReceivedSlots)

	contribution.SlotStart = start.Add(time.Hour)
	contribution.SourceFileID = uuid.NewString()
	contribution.RollupChunkIndex = 0
	contribution.Values[0].Sum = 30
	contribution.Values[0].Min = 30
	contribution.Values[0].Max = 30
	third, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)
	require.False(t, third.Complete)
	contribution.SourceFileID = uuid.NewString()
	contribution.RollupChunkIndex = 1
	contribution.Values[0].Sum = 40
	contribution.Values[0].Min = 40
	contribution.Values[0].Max = 40
	fourth, err := store.Accumulate(context.Background(), contribution)
	require.NoError(t, err)
	require.True(t, fourth.Complete)
	require.EqualValues(t, 2, fourth.ReceivedSlots)

	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.EqualValues(t, 8, state.SourceExpectedSlots)
	require.EqualValues(t, 8, state.SourceReceivedSlots,
		"source completeness metadata must be counted once per child, not once per chunk")
	require.Len(t, state.Accumulators, 1)
	require.EqualValues(t, 100, state.Accumulators[0].Sum)
	require.EqualValues(t, 4, state.Accumulators[0].Count)
}

func TestRedisWindowStoreTracksMissingRawSlotsPerEntity(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC)
	deviceID := uuid.NewString()
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), EntityKey: deviceID,
		Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
	}
	for slot := 0; slot < 3; slot++ {
		_, err := store.Accumulate(context.Background(), Contribution{
			Key: key, SourceFileID: uuid.NewString(), DeviceID: deviceID,
			SlotStart: start.Add(time.Duration(slot) * slotDuration), ExpectedSlots: 4,
			Values: []ContributionValue{{
				Dimension: DimensionDevice, DimensionKey: deviceID,
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum, Value: 1,
			}},
		})
		require.NoError(t, err)
	}

	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.EqualValues(t, 4, state.Entities[deviceID].SourceExpectedSlots)
	require.EqualValues(t, 3, state.Entities[deviceID].SourceReceivedSlots)
	require.EqualValues(t, 3, state.Entities[deviceID].ReceivedSlots)
}
