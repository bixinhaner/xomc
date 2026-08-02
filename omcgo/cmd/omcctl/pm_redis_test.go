package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type fakePMRedisValue struct {
	payload []byte
	ttl     time.Duration
}

type fakePMRedisStore struct {
	values          map[string]fakePMRedisValue
	disappearOnRead map[string]bool
	readBatchSizes  []int
	restores        int
}

func newFakePMRedisStore(values map[string]fakePMRedisValue) *fakePMRedisStore {
	return &fakePMRedisStore{values: values}
}

func (s *fakePMRedisStore) Scan(_ context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	if cursor != 0 {
		return nil, 0, nil
	}
	prefix := strings.TrimSuffix(pattern, "*")
	keys := make([]string, 0, len(s.values))
	for key := range s.values {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, 0, nil
}

func (s *fakePMRedisStore) Read(_ context.Context, keys []string) ([]pmRedisDump, error) {
	s.readBatchSizes = append(s.readBatchSizes, len(keys))
	result := make([]pmRedisDump, len(keys))
	for i, key := range keys {
		result[i].Key = key
		if s.disappearOnRead[key] {
			delete(s.values, key)
			delete(s.disappearOnRead, key)
		}
		value, ok := s.values[key]
		if !ok {
			result[i].Missing = true
			continue
		}
		result[i].Payload = append([]byte(nil), value.payload...)
		result[i].TTL = value.ttl
	}
	return result, nil
}

func (s *fakePMRedisStore) Restore(_ context.Context, value pmRedisDump, replace bool) error {
	if _, exists := s.values[value.Key]; exists && !replace {
		return fmt.Errorf("BUSYKEY")
	}
	s.restores++
	s.values[value.Key] = fakePMRedisValue{payload: append([]byte(nil), value.Payload...), ttl: value.TTL}
	return nil
}

func TestPMRedisMigratorCopiesOnlyPMAggregationKeysAndPreservesTTL(t *testing.T) {
	source := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:hour:a": {payload: []byte{0, 1, 2, 255}, ttl: 20 * time.Minute},
		"pmagg:week:b": {payload: []byte("persistent"), ttl: -1},
		"kpi-route:x":  {payload: []byte("rebuildable"), ttl: time.Hour},
	})
	target := newFakePMRedisStore(map[string]fakePMRedisValue{})

	result, err := (&pmRedisMigrator{source: source, target: target}).Migrate(context.Background(), pmRedisMigrationOptions{
		Pattern: "pmagg:*", ScanCount: 500, PipelineSize: 1, TTLEpsilon: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.Scanned)
	require.Equal(t, int64(2), result.Copied)
	require.Equal(t, int64(2), result.Verified)
	require.Equal(t, []byte{0, 1, 2, 255}, target.values["pmagg:hour:a"].payload)
	require.Equal(t, 20*time.Minute, target.values["pmagg:hour:a"].ttl)
	require.Equal(t, time.Duration(-1), target.values["pmagg:week:b"].ttl)
	require.NotContains(t, target.values, "kpi-route:x")
	for _, size := range source.readBatchSizes {
		require.LessOrEqual(t, size, 1)
	}
}

func TestPMRedisMigratorConflictIsSafeAndReplaceIsExplicit(t *testing.T) {
	source := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:hour:a": {payload: []byte("authoritative"), ttl: time.Hour},
	})
	target := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:hour:a": {payload: []byte("different"), ttl: time.Hour},
	})
	migrator := &pmRedisMigrator{source: source, target: target}

	result, err := migrator.Migrate(context.Background(), pmRedisMigrationOptions{Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 10, TTLEpsilon: time.Second})
	require.Error(t, err)
	require.Equal(t, int64(1), result.Conflicts)
	require.Equal(t, int64(1), result.Failed)
	require.Equal(t, []byte("different"), target.values["pmagg:hour:a"].payload)

	result, err = migrator.Migrate(context.Background(), pmRedisMigrationOptions{Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 10, Replace: true, TTLEpsilon: time.Second})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Copied)
	require.Equal(t, []byte("authoritative"), target.values["pmagg:hour:a"].payload)
}

func TestPMRedisRollbackReverseCopyReplacesStaleCoreState(t *testing.T) {
	pm := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:hour:a": {payload: []byte("new-pm-state"), ttl: 45 * time.Minute},
	})
	core := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:hour:a": {payload: []byte("stale-pre-cutover-state"), ttl: 20 * time.Minute},
	})

	result, err := (&pmRedisMigrator{source: pm, target: core}).Migrate(
		context.Background(),
		pmRedisMigrationOptions{
			Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 10,
			Replace: true, TTLEpsilon: time.Second,
		},
	)

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Copied)
	require.Equal(t, []byte("new-pm-state"), core.values["pmagg:hour:a"].payload)
	require.Equal(t, 45*time.Minute, core.values["pmagg:hour:a"].ttl)
}

func TestPMRedisMigratorIsIdempotentAndSkipsDisappearedKeys(t *testing.T) {
	value := fakePMRedisValue{payload: []byte("same"), ttl: -1}
	source := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:a": value,
		"pmagg:b": {payload: []byte("gone"), ttl: time.Minute},
	})
	target := newFakePMRedisStore(map[string]fakePMRedisValue{"pmagg:a": value})
	source.disappearOnRead = map[string]bool{"pmagg:b": true}

	result, err := (&pmRedisMigrator{source: source, target: target}).Migrate(context.Background(), pmRedisMigrationOptions{
		Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 10, TTLEpsilon: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.Skipped)
	require.Zero(t, target.restores)
}

func TestPMRedisMigratorDryRunDoesNotWrite(t *testing.T) {
	source := newFakePMRedisStore(map[string]fakePMRedisValue{
		"pmagg:a": {payload: []byte("value"), ttl: time.Minute},
	})
	target := newFakePMRedisStore(map[string]fakePMRedisValue{})

	result, err := (&pmRedisMigrator{source: source, target: target}).Migrate(context.Background(), pmRedisMigrationOptions{
		Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 10, DryRun: true, TTLEpsilon: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Planned)
	require.Zero(t, result.Copied)
	require.Empty(t, target.values)
}

func TestPMRedisMigratorRejectsUnsafePatterns(t *testing.T) {
	migrator := &pmRedisMigrator{source: newFakePMRedisStore(nil), target: newFakePMRedisStore(nil)}
	for _, pattern := range []string{"*", "kpi-route:*", "pm*"} {
		_, err := migrator.Migrate(context.Background(), pmRedisMigrationOptions{Pattern: pattern, ScanCount: 10, PipelineSize: 10})
		require.Error(t, err, pattern)
	}
}

func TestPMRedisMigratorMiniredisStringEndToEnd(t *testing.T) {
	sourceServer := miniredis.RunT(t)
	targetServer := miniredis.RunT(t)
	sourceClient := redis.NewClient(&redis.Options{Addr: sourceServer.Addr()})
	targetClient := redis.NewClient(&redis.Options{Addr: targetServer.Addr()})
	t.Cleanup(func() { _ = sourceClient.Close(); _ = targetClient.Close() })
	ctx := context.Background()
	require.NoError(t, sourceClient.Set(ctx, "pmagg:string:ttl", "value", 30*time.Minute).Err())
	require.NoError(t, sourceClient.Set(ctx, "pmagg:string:persistent", "forever", 0).Err())
	require.NoError(t, sourceClient.Set(ctx, "kpi-route:skip", "skip", 0).Err())

	result, err := (&pmRedisMigrator{
		source: newGoRedisPMStore(sourceClient),
		target: newGoRedisPMStore(targetClient),
	}).Migrate(ctx, pmRedisMigrationOptions{Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 2, TTLEpsilon: time.Second})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.Copied)
	valueWithTTL, err := targetServer.Get("pmagg:string:ttl")
	require.NoError(t, err)
	require.Equal(t, "value", valueWithTTL)
	persistentValue, err := targetServer.Get("pmagg:string:persistent")
	require.NoError(t, err)
	require.Equal(t, "forever", persistentValue)
	require.False(t, targetServer.Exists("kpi-route:skip"))
	targetTTL, err := targetClient.PTTL(ctx, "pmagg:string:ttl").Result()
	require.NoError(t, err)
	require.InDelta(t, float64(30*time.Minute), float64(targetTTL), float64(time.Second))
	persistentTTL, err := targetClient.PTTL(ctx, "pmagg:string:persistent").Result()
	require.NoError(t, err)
	require.Equal(t, time.Duration(-1), persistentTTL)

	result, err = (&pmRedisMigrator{
		source: newGoRedisPMStore(sourceClient),
		target: newGoRedisPMStore(targetClient),
	}).Migrate(ctx, pmRedisMigrationOptions{Pattern: "pmagg:*", ScanCount: 10, PipelineSize: 2, TTLEpsilon: time.Second})
	require.NoError(t, err)
	require.Zero(t, result.Copied)
	require.Equal(t, int64(2), result.Skipped)
}

func TestPMRedisCommandExists(t *testing.T) {
	cmd := newPMRedisCmd()
	require.Equal(t, "pm-redis", cmd.Use)
	require.NotNil(t, cmd.Commands()[0])
}
