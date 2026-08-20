package stream

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestCompactAccumulatorV2StoresOnlyNumericState(t *testing.T) {
	value := ContributionValue{
		Dimension: DimensionDevice, DimensionKey: uuid.NewString(),
		DimensionName: "ENB-000001", ObjectLDN: "Device.Services.FAPService.1.CellConfig.LTE.RAN",
		DeviceOUI: "001A2B", DeviceSN: "SN-000001", Technology: "LTE",
		MetricPath: "C000010070", MetricType: "counter", Operation: AggregationAvg,
	}
	legacyDefinition, err := encodeDefinition(value)
	require.NoError(t, err)
	legacy := strings.Join([]string{
		"v1", legacyDefinition, "12345.6789", "96", "-120.5", "38.75",
	}, "|")

	encoded := encodeCompactAccumulatorV2(12345.6789, 96, -120.5, 38.75)
	require.LessOrEqual(t, len(encoded), len(legacy)*45/100)
	require.NotContains(t, encoded, value.MetricPath)
	require.NotContains(t, encoded, base64.RawURLEncoding.EncodeToString([]byte(value.MetricPath)))

	sum, count, minValue, maxValue, err := decodeCompactAccumulatorV2(encoded)
	require.NoError(t, err)
	require.Equal(t, 12345.6789, sum)
	require.EqualValues(t, 96, count)
	require.Equal(t, -120.5, minValue)
	require.Equal(t, 38.75, maxValue)
}

func TestRedisMemoryModelPlanningEstimateV2FitsTwentyThousandDevices(t *testing.T) {
	const (
		devices       = int64(20_000)
		metrics       = int64(146)
		granularities = int64(4)
		// Redis hash entry overhead varies by allocator/encoding. Use a
		// deliberately conservative 96 bytes on top of the actual v2 field
		// and value lengths so this is a capacity guard, not an optimistic
		// serialization-only estimate.
		hashEntryOverhead = int64(96)
		windowOverhead    = int64(1536)
	)
	fieldBytes := int64(len("0123456789abcdef01234567|0123456789abcdef"))
	valueBytes := int64(len(encodeCompactAccumulatorV2(12345.6789, 96, -120.5, 38.75)))
	total := devices * granularities *
		(windowOverhead + metrics*(fieldBytes+valueBytes+hashEntryOverhead))

	require.Less(t, total, int64(5*1024*1024*1024/2),
		"analytical planning estimate must stay below 2.5 GiB; Task 8 verifies real Redis")
}

func TestRedisWindowStoreV2StoresDefinitionOnceAndIdentityOncePerWindow(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	hook := &redisCommandRecordingHook{}
	client.AddHook(hook)
	store := NewRedisWindowStore(client, time.Hour).SetV2WriteEnabled(true)
	start := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	versionID := uuid.New()
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: versionID, Granularity: GranularityHourly,
		EntityKey: "device-1", Start: start, End: start.Add(time.Hour),
	}
	values := []ContributionValue{
		{
			Dimension: DimensionDevice, DimensionKey: "device-1", DimensionName: "ENB-1",
			DeviceOUI: "001A2B", DeviceSN: "SN-1", Technology: "LTE",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 12,
		},
		{
			Dimension: DimensionDevice, DimensionKey: "device-1", DimensionName: "ENB-1",
			DeviceOUI: "001A2B", DeviceSN: "SN-1", Technology: "LTE",
			MetricPath: "C002", MetricType: "counter", Operation: AggregationMax, Value: 7,
		},
	}
	for slot := 0; slot < 2; slot++ {
		_, err := store.Accumulate(context.Background(), Contribution{
			Key: key, SourceFileID: uuid.NewString(), DeviceID: "device-1",
			SlotStart: start.Add(time.Duration(slot) * slotDuration), ExpectedSlots: 4,
			Values: values,
		})
		require.NoError(t, err)
	}
	hook.mu.Lock()
	require.NotContains(t, hook.commands, "hsetnx",
		"definitions must not add one Redis round trip per metric")
	require.LessOrEqual(t, countStrings(hook.commands, "hset"), 1,
		"task-version definitions must be written in one bounded batch")
	hook.mu.Unlock()

	keys := redisKeys(key, 1)
	identityCount, err := client.HLen(context.Background(), keys.identities).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, identityCount,
		"device/entity identity must be stored once per window")
	identityValues, err := client.HVals(context.Background(), keys.identities).Result()
	require.NoError(t, err)
	require.Len(t, identityValues, 1)
	identityJSON, err := base64.RawURLEncoding.DecodeString(identityValues[0])
	require.NoError(t, err)
	require.NotContains(t, string(identityJSON), `"oui"`,
		"device identity must not be repeated in every result identity")
	require.NotContains(t, string(identityJSON), `"sn"`,
		"device identity must not be repeated in every result identity")
	require.Equal(t, "001A2B", server.HGet(keys.meta, "device_oui"))
	require.Equal(t, "SN-1", server.HGet(keys.meta, "device_sn"))
	definitionCount, err := client.HLen(
		context.Background(), redisVersionDefinitionsKey(versionID),
	).Result()
	require.NoError(t, err)
	require.EqualValues(t, 2, definitionCount,
		"metric definitions must be stored once per immutable task version")
	definitionTTL, err := client.TTL(
		context.Background(), redisVersionDefinitionsKey(versionID),
	).Result()
	require.NoError(t, err)
	require.GreaterOrEqual(t, definitionTTL, 44*24*time.Hour,
		"shared definitions must outlive every supported active window")
	fields, err := server.HKeys(keys.acc[0])
	require.NoError(t, err)
	require.Len(t, fields, 2)
	for _, field := range fields {
		require.True(t, strings.HasPrefix(server.HGet(keys.acc[0], field), "v2|"))
	}

	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, state.Accumulators, 2)
	hook.mu.Lock()
	require.Contains(t, hook.commands, "hscan")
	require.LessOrEqual(t, countStrings(hook.commands, "hgetall"), 1,
		"window identities must use bounded HSCAN; only version definitions may use HGETALL")
	hook.mu.Unlock()
}

func TestAccumulatorV2LuaKeepsStableMetricLoopFreeOfIdentityAndLegacyWork(t *testing.T) {
	loop := strings.Index(accumulateLua, "for i = 1, value_count do")
	require.Positive(t, loop)
	identityWrite := strings.Index(accumulateLua, `redis.call("HSETNX", identities`)
	require.Positive(t, identityWrite)
	require.Less(t, identityWrite, loop,
		"identity registration must happen once per unique entity before the metric loop")
	currentLookup := strings.Index(accumulateLua, `local current = redis.call("HGET", acc, base)`)
	require.Positive(t, currentLookup)
	migrationElse := currentLookup + strings.Index(accumulateLua[currentLookup:], "\n    else\n")
	require.Greater(t, migrationElse, currentLookup)
	require.Greater(t,
		strings.Index(accumulateLua, `local legacy_compact = redis.call("HGET", acc, legacy_base)`),
		migrationElse,
		"legacy probes must run only for the one-time v1-to-v2 migration path")
}

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
	require.Less(t, len(fields[0]), 48, "accumulator fields must use compact IDs")
	require.False(t, server.Exists(keys.defs[0]))
}

func TestRedisWindowStoreDefaultsToRollbackSafeV1Writes(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	_, err := store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 4,
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 1,
		}},
	})
	require.NoError(t, err)
	fields, err := server.HKeys(redisKeys(key, 1).acc[0])
	require.NoError(t, err)
	require.Len(t, fields, 1)
	require.True(t, strings.HasPrefix(
		server.HGet(redisKeys(key, 1).acc[0], fields[0]),
		legacyCompactAccumulatorVersion+"|",
	))
	require.False(t, server.Exists(redisVersionDefinitionsKey(key.TaskVersionID)))
	require.False(t, server.Exists(redisKeys(key, 1).identities))
}

func TestRedisWindowStoreFencesNormalWritesDuringRebuild(t *testing.T) {
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
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 1,
		}},
	}

	lock, err := store.TryFinalizeLock(context.Background(), key, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lock)

	_, err = store.Accumulate(context.Background(), contribution)
	require.ErrorContains(t, err, "PM_AGGREGATION_WINDOW_LOCKED")

	_, err = store.accumulateWithLock(context.Background(), contribution, lock)
	require.NoError(t, err)
	require.NoError(t, store.DeleteState(context.Background(), key))
	require.True(t, server.Exists(redisKeys(key, 1).lock), "state deletion must retain fencing lock")

	require.NoError(t, lock.Extend(context.Background(), 2*time.Minute))
	require.NoError(t, lock.Release(context.Background()))
	require.False(t, server.Exists(redisKeys(key, 1).lock))
}

func TestRedisWindowStoreInitializesEmptyRebuildStateUnderOwnedLock(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		Granularity: GranularityDaily, EntityKey: "Network",
		Start: start, End: start.Add(24 * time.Hour),
	}
	lock, err := store.TryFinalizeLock(context.Background(), key, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lock)

	require.NoError(t, store.InitializeEmptyWithLock(context.Background(), key, lock))
	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.Zero(t, state.ExpectedSlots)
	require.Zero(t, state.ReceivedSlots)
	require.Empty(t, state.Accumulators)
	require.True(t, server.TTL(redisKeys(key, 1).meta) > 0)

	require.NoError(t, lock.Release(context.Background()))
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
		require.True(t, strings.HasPrefix(raw, legacyCompactAccumulatorVersion+"|"))
	}
	require.False(t, server.Exists(keys.defs[0]),
		"new compact windows must not allocate a separate definitions hash")
}

func TestRedisWindowStoreMigratesLegacyAccumulatorOnFirstWrite(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour).SetV2WriteEnabled(true)
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
	require.Len(t, fields, 1)
	require.NotEqual(t, id, fields[0], "the v1 full-definition ID must be migrated")
	raw := server.HGet(keys.acc[0], fields[0])
	require.True(t, strings.HasPrefix(raw, compactAccumulatorVersion+"|"))
	require.False(t, server.Exists(keys.defs[0]))
}

func TestRedisWindowStoreMergesV1WrittenDuringRollingUpgrade(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour).SetV2WriteEnabled(true)
	start := time.Date(2026, 7, 30, 3, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	value := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 12,
	}
	_, err := store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 4, Values: []ContributionValue{value},
	})
	require.NoError(t, err)

	legacyID, err := accumulatorDefinitionID(value)
	require.NoError(t, err)
	legacyDefinition, err := encodeDefinition(value)
	require.NoError(t, err)
	require.NoError(t, client.HSet(
		context.Background(), redisKeys(key, 1).acc[0], legacyID,
		strings.Join([]string{
			legacyCompactAccumulatorVersion, legacyDefinition, "5", "1", "5", "5",
		}, "|"),
	).Err())

	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, state.Accumulators, 1)
	require.Equal(t, 17.0, state.Accumulators[0].Sum)
	require.EqualValues(t, 2, state.Accumulators[0].Count)
}

func TestRedisWindowStoreCanDisableV2AfterDrainWithoutLosingDualReadState(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour).SetV2WriteEnabled(true)
	start := time.Date(2026, 7, 30, 3, 30, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	value := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 12,
	}
	_, err := store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 4, Values: []ContributionValue{value},
	})
	require.NoError(t, err)
	store.SetV2WriteEnabled(false)
	value.Value = 5
	_, err = store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start.Add(slotDuration), ExpectedSlots: 4,
		Values: []ContributionValue{value},
	})
	require.NoError(t, err)

	state, err := store.Read(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, state.Accumulators, 1)
	require.Equal(t, 17.0, state.Accumulators[0].Sum)
	require.EqualValues(t, 2, state.Accumulators[0].Count)
}

type redisCommandRecordingHook struct {
	mu       sync.Mutex
	commands []string
}

func countStrings(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}

func (h *redisCommandRecordingHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *redisCommandRecordingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.mu.Lock()
		h.commands = append(h.commands, cmd.Name())
		h.mu.Unlock()
		return next(ctx, cmd)
	}
}

func (h *redisCommandRecordingHook) ProcessPipelineHook(
	next redis.ProcessPipelineHook,
) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		h.mu.Lock()
		for _, command := range commands {
			h.commands = append(h.commands, command.Name())
		}
		h.mu.Unlock()
		return next(ctx, commands)
	}
}

func TestRedisWindowStoreDeletesLargeStateWithUnlink(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	hook := &redisCommandRecordingHook{}
	client.AddHook(hook)
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 30, 4, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	_, err := store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 4,
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 1,
		}},
	})
	require.NoError(t, err)
	hook.mu.Lock()
	hook.commands = nil
	hook.mu.Unlock()

	require.NoError(t, store.DeleteState(context.Background(), key))
	hook.mu.Lock()
	defer hook.mu.Unlock()
	require.Contains(t, hook.commands, "unlink")
	require.NotContains(t, hook.commands, "del")
}

func TestRedisWindowStoreDeletesVersionDefinitionsInBatches(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	versionIDs := make([]uuid.UUID, 0, 130)
	for index := 0; index < 130; index++ {
		versionID := uuid.New()
		versionIDs = append(versionIDs, versionID)
		require.NoError(t, client.Set(
			context.Background(), redisVersionDefinitionsKey(versionID), "ISSUE353_SENTINEL", 0,
		).Err())
	}

	require.NoError(t, store.DeleteVersionDefinitions(context.Background(), versionIDs))
	for _, versionID := range versionIDs {
		require.False(t, server.Exists(redisVersionDefinitionsKey(versionID)))
	}
}

type failNthUnlinkHook struct {
	nth   int
	count int
}

func (h *failNthUnlinkHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *failNthUnlinkHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "unlink" {
			h.count++
			if h.count == h.nth {
				return errors.New("injected unlink failure")
			}
		}
		return next(ctx, cmd)
	}
}

func (h *failNthUnlinkHook) ProcessPipelineHook(
	next redis.ProcessPipelineHook,
) redis.ProcessPipelineHook {
	return next
}

func TestRedisWindowStoreRetainsMetaUntilAllUnlinkBatchesSucceed(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	hook := &failNthUnlinkHook{nth: 3}
	client.AddHook(hook)
	store := NewRedisWindowStore(client, time.Hour)
	start := time.Date(2026, 7, 30, 5, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), Granularity: GranularityHourly,
		EntityKey: "network", Start: start, End: start.Add(time.Hour),
	}
	_, err := store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: uuid.NewString(),
		SlotStart: start, ExpectedSlots: 4,
		Values: []ContributionValue{{
			Dimension: DimensionNetwork, DimensionKey: "network",
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 1,
		}},
	})
	require.NoError(t, err)

	err = store.unlink(context.Background(), key, false, 2)
	require.ErrorContains(t, err, "injected unlink failure")
	require.True(t, server.Exists(redisKeys(key, 1).meta),
		"meta and shard_count are the retry anchor and must be unlinked last")

	require.NoError(t, store.unlink(context.Background(), key, false, 2))
	require.False(t, server.Exists(redisKeys(key, 1).meta))
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
