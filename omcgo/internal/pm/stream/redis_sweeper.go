package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

const redisSweepLockTTL = 2 * time.Minute
const redisSweepOperationTimeout = 30 * time.Second

type PublishedStateVerifier interface {
	CanSweepRedisState(
		ctx context.Context,
		key WindowKey,
		publishedBefore time.Time,
	) (bool, error)
}

type RedisStateSweeper struct {
	store           *RedisWindowStore
	verifier        PublishedStateVerifier
	safetyThreshold time.Duration
	metrics         *Metrics
	logger          *zap.Logger

	mu      sync.Mutex
	cursor  uint64
	pending []string

	sampleCursor  uint64
	samplePending []string
}

type redisStateSample struct {
	windows int64
	keys    int64
	bytes   int64
}

func NewRedisStateSweeper(
	store *RedisWindowStore,
	verifier PublishedStateVerifier,
	safetyThreshold time.Duration,
	metrics *Metrics,
	logger *zap.Logger,
) *RedisStateSweeper {
	if safetyThreshold <= 0 {
		safetyThreshold = 30 * time.Minute
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RedisStateSweeper{
		store: store, verifier: verifier, safetyThreshold: safetyThreshold,
		metrics: metrics, logger: logger,
	}
}

// SweepPublishedState removes only Redis windows that can be tied to an exact
// published DB row. It claims the same fencing lock used by finalize/rebuild,
// so the state cannot become active between verification and UNLINK.
func (s *RedisStateSweeper) SweepPublishedState(
	ctx context.Context,
	scanLimit int,
	unlinkBatch int,
) (int64, error) {
	if s == nil || s.store == nil || s.verifier == nil {
		return 0, errors.New("PM aggregation Redis sweeper dependencies are missing")
	}
	if scanLimit <= 0 || unlinkBatch <= 0 {
		return 0, errors.New("PM aggregation Redis sweeper limits must be positive")
	}
	metaKeys, err := s.nextMetaKeys(ctx, scanLimit)
	if err != nil {
		return 0, err
	}
	var deleted int64
	publishedBefore := time.Now().UTC().Add(-s.safetyThreshold)
	for _, metaKey := range metaKeys {
		key, ok, decodeErr := s.windowKeyFromMeta(ctx, metaKey)
		if decodeErr != nil {
			return deleted, decodeErr
		}
		if !ok {
			continue
		}
		operationCtx, cancel := context.WithTimeout(ctx, redisSweepOperationTimeout)
		lock, lockErr := s.store.TryFinalizeLock(operationCtx, key, redisSweepLockTTL)
		if lockErr != nil {
			cancel()
			return deleted, lockErr
		}
		if lock == nil {
			cancel()
			continue
		}
		canSweep, verifyErr := s.verifier.CanSweepRedisState(
			operationCtx, key, publishedBefore,
		)
		if verifyErr == nil && canSweep {
			verifyErr = s.store.unlinkFenced(
				operationCtx, key, false, unlinkBatch,
				func(batchCtx context.Context) error {
					return lock.Extend(batchCtx, redisSweepLockTTL)
				},
			)
			if verifyErr == nil {
				deleted++
				if s.metrics != nil {
					s.metrics.RedisSweeperDeletedTotal.Inc()
				}
			}
		}
		cancel()
		releaseErr := lock.Release(context.Background())
		if verifyErr != nil || releaseErr != nil {
			return deleted, errors.Join(verifyErr, releaseErr)
		}
	}
	return deleted, nil
}

func (s *RedisStateSweeper) nextMetaKeys(
	ctx context.Context,
	limit int,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextBoundedMetaKeys(ctx, limit, &s.cursor, &s.pending)
}

func (s *RedisStateSweeper) nextSampleMetaKeys(
	ctx context.Context,
	limit int,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextBoundedMetaKeys(ctx, limit, &s.sampleCursor, &s.samplePending)
}

func (s *RedisStateSweeper) nextBoundedMetaKeys(
	ctx context.Context,
	limit int,
	cursor *uint64,
	pending *[]string,
) ([]string, error) {
	for len(*pending) < limit && (len(*pending) == 0 || *cursor != 0) {
		keys, next, err := s.store.client.Scan(
			ctx, *cursor, "pmagg:*:meta", int64(limit),
		).Result()
		if err != nil {
			return nil, fmt.Errorf("scan PM aggregation Redis metadata: %w", err)
		}
		*pending = append(*pending, keys...)
		*cursor = next
		if next == 0 || len(keys) == 0 {
			break
		}
	}
	count := min(limit, len(*pending))
	result := append([]string(nil), (*pending)[:count]...)
	*pending = (*pending)[count:]
	return result, nil
}

func (s *RedisStateSweeper) windowKeyFromMeta(
	ctx context.Context,
	metaKey string,
) (WindowKey, bool, error) {
	values, err := s.store.client.HMGet(
		ctx, metaKey,
		"task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end",
	).Result()
	if err != nil {
		return WindowKey{}, false, fmt.Errorf(
			"read PM aggregation Redis sweep metadata: %w", err,
		)
	}
	for _, value := range values {
		if value == nil || fmt.Sprint(value) == "" {
			return WindowKey{}, false, nil
		}
	}
	taskID, err := uuid.Parse(fmt.Sprint(values[0]))
	if err != nil {
		return WindowKey{}, false, nil
	}
	versionID, err := uuid.Parse(fmt.Sprint(values[1]))
	if err != nil {
		return WindowKey{}, false, nil
	}
	granularity := Granularity(fmt.Sprint(values[3]))
	if granularity != GranularityHourly && granularity != GranularityDaily &&
		granularity != GranularityWeekly && granularity != GranularityMonthly {
		return WindowKey{}, false, nil
	}
	start, err := time.Parse(time.RFC3339Nano, fmt.Sprint(values[4]))
	if err != nil {
		return WindowKey{}, false, nil
	}
	end, err := time.Parse(time.RFC3339Nano, fmt.Sprint(values[5]))
	if err != nil || !end.After(start) {
		return WindowKey{}, false, nil
	}
	key := WindowKey{
		TaskID: taskID, TaskVersionID: versionID,
		EntityKey: fmt.Sprint(values[2]), Granularity: granularity,
		Start: start, End: end,
	}
	if redisKeys(key, 1).meta != metaKey {
		return WindowKey{}, false, nil
	}
	return key, true, nil
}

func (s *RedisStateSweeper) Run(
	ctx context.Context,
	interval time.Duration,
	scanLimit int,
	unlinkBatch int,
) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SweepPublishedState(ctx, scanLimit, unlinkBatch); err != nil {
				s.logger.Warn("sweep published PM aggregation Redis state", zap.Error(err))
			}
			if err := s.sampleRedisState(ctx, scanLimit, scanLimit*8); err != nil {
				s.logger.Warn("sample PM aggregation Redis state", zap.Error(err))
			}
		}
	}
}

func (s *RedisStateSweeper) sampleRedisState(
	ctx context.Context,
	windowLimit int,
	keyLimit int,
) error {
	if s.metrics == nil || windowLimit <= 0 || keyLimit <= 0 {
		return nil
	}
	metaKeys, err := s.nextSampleMetaKeys(ctx, windowLimit)
	if err != nil {
		return fmt.Errorf("scan sampled PM aggregation Redis metadata: %w", err)
	}
	samples := make(map[Granularity]*redisStateSample)
	remaining := keyLimit
	for _, metaKey := range metaKeys {
		windowKey, ok, keyErr := s.windowKeyFromMeta(ctx, metaKey)
		if keyErr != nil {
			return keyErr
		}
		if !ok {
			continue
		}
		shardCount, shardErr := s.store.client.HGet(ctx, metaKey, "shard_count").Int()
		if shardErr != nil || shardCount <= 0 || shardCount > maxWindowShards {
			continue
		}
		granularity := windowKey.Granularity
		sample := samples[granularity]
		if sample == nil {
			sample = &redisStateSample{}
			samples[granularity] = sample
		}
		sample.windows++
		if remaining == 0 {
			continue
		}
		keys := redisKeys(windowKey, shardCount)
		candidates := []string{
			keys.seen, keys.slots, keys.meta, keys.entityMeta,
			keys.chunks, keys.identities, keys.lock,
		}
		candidates = append(candidates, keys.acc...)
		candidates = append(candidates, keys.defs...)
		for _, key := range candidates {
			if remaining == 0 {
				break
			}
			bytes, usageErr := s.store.client.MemoryUsage(ctx, key, 5).Result()
			if usageErr != nil {
				continue
			}
			sample.keys++
			sample.bytes += bytes
			remaining--
		}
	}
	for _, granularity := range []Granularity{
		GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
	} {
		sample := samples[granularity]
		if sample == nil {
			sample = &redisStateSample{}
		}
		label := string(granularity)
		s.metrics.RedisSampledActiveWindows.WithLabelValues(label).Set(float64(sample.windows))
		s.metrics.RedisSampledKeys.WithLabelValues(label).Set(float64(sample.keys))
		s.metrics.RedisSampledEstimatedBytes.WithLabelValues(label).Set(float64(sample.bytes))
	}
	return nil
}

func (r *WindowRepository) CanSweepRedisState(
	ctx context.Context,
	key WindowKey,
	publishedBefore time.Time,
) (bool, error) {
	query, args, err := redisSweepEligibilityQuery(key, publishedBefore).ToSql()
	if err != nil {
		return false, fmt.Errorf("build PM Redis sweep eligibility query: %w", err)
	}
	var marker int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&marker); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query PM Redis sweep eligibility: %w", err)
	}
	return marker == 1, nil
}

func redisSweepEligibilityQuery(
	key WindowKey,
	publishedBefore time.Time,
) sq.SelectBuilder {
	return storage.Psql.Select("1").
		From("pm_aggregation_windows").
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"task_id": key.TaskID, "window_end": key.End}).
		Where(sq.Eq{"status": "published"}).
		Where(sq.LtOrEq{"published_at": publishedBefore}).
		Where(sq.Expr(
			"(finalize_lease_until IS NULL OR finalize_lease_until <= CURRENT_TIMESTAMP)",
		)).
		Limit(1)
}
