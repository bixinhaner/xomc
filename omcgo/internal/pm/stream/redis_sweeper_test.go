package stream

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisSweeperEligibilityRequiresPublishedAgeAndNoActiveDBLease(t *testing.T) {
	start := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), EntityKey: "device-1",
		Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
	}
	query, args, err := redisSweepEligibilityQuery(
		key, start.Add(2*time.Hour),
	).ToSql()
	require.NoError(t, err)
	normalized := strings.Join(strings.Fields(query), " ")
	require.Contains(t, normalized, "status =")
	require.Contains(t, normalized, "published_at <=")
	require.Contains(t, normalized, "finalize_lease_until IS NULL")
	require.Contains(t, normalized, "finalize_lease_until <= CURRENT_TIMESTAMP")
	require.Contains(t, normalized, "task_id =")
	require.Contains(t, normalized, "window_end =")
	require.Contains(t, args, key.TaskID.String())
	require.Contains(t, args, key.TaskVersionID.String())
}

type fakePublishedStateVerifier struct {
	mu              sync.Mutex
	published       map[WindowKey]bool
	publishedBefore []time.Time
}

func (f *fakePublishedStateVerifier) CanSweepRedisState(
	_ context.Context,
	key WindowKey,
	publishedBefore time.Time,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.publishedBefore = append(f.publishedBefore, publishedBefore)
	return f.published[key], nil
}

func TestRedisSweeperOnlyUnlinksOldPublishedUnlockedWindows(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	verifier := &fakePublishedStateVerifier{published: make(map[WindowKey]bool)}
	sweeper := NewRedisStateSweeper(store, verifier, 30*time.Minute, nil, nil)
	start := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)

	published := sweepTestWindow(t, store, start, "published")
	open := sweepTestWindow(t, store, start, "open")
	locked := sweepTestWindow(t, store, start, "locked")
	verifier.published[published] = true
	verifier.published[locked] = true
	lock, err := store.TryFinalizeLock(context.Background(), locked, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lock)
	defer func() { require.NoError(t, lock.Release(context.Background())) }()

	deleted, err := sweeper.SweepPublishedState(context.Background(), 16, 4)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	require.False(t, server.Exists(redisKeys(published, 1).meta))
	require.True(t, server.Exists(redisKeys(open, 1).meta),
		"an open DB window must never be deleted by prefix or TTL")
	require.True(t, server.Exists(redisKeys(locked, 1).meta),
		"an active finalize/rebuild lock must fence the sweeper")
	require.NotEmpty(t, verifier.publishedBefore)
	require.WithinDuration(t, time.Now().Add(-30*time.Minute),
		verifier.publishedBefore[0], 3*time.Second)
}

func TestRedisSweeperHasHardScanLimitAndSkipsUnverifiableMetadata(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisWindowStore(client, time.Hour)
	verifier := &fakePublishedStateVerifier{published: make(map[WindowKey]bool)}
	sweeper := NewRedisStateSweeper(store, verifier, time.Hour, nil, nil)
	start := time.Date(2026, 7, 30, 2, 0, 0, 0, time.UTC)

	for index := 0; index < 3; index++ {
		key := sweepTestWindow(t, store, start, uuid.NewString())
		verifier.published[key] = true
	}
	require.NoError(t, client.HSet(
		context.Background(), "pmagg:{unverifiable}:meta", "received_slots", 1,
	).Err())

	deleted, err := sweeper.SweepPublishedState(context.Background(), 1, 1)
	require.NoError(t, err)
	require.LessOrEqual(t, deleted, int64(1))
	require.True(t, server.Exists("pmagg:{unverifiable}:meta"),
		"metadata that cannot be tied to an exact DB window must be retained")
}

func sweepTestWindow(
	t *testing.T,
	store *RedisWindowStore,
	start time.Time,
	entity string,
) WindowKey {
	t.Helper()
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(), EntityKey: entity,
		Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
	}
	_, err := store.Accumulate(context.Background(), Contribution{
		Key: key, SourceFileID: uuid.NewString(), DeviceID: entity,
		SlotStart: start, ExpectedSlots: 4,
		Values: []ContributionValue{{
			Dimension: DimensionDevice, DimensionKey: entity,
			MetricPath: "C001", MetricType: "counter", Operation: AggregationSum, Value: 1,
		}},
	})
	require.NoError(t, err)
	return key
}
