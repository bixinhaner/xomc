package task

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	redisQueueFamilyCommand = "cmdq"
	redisQueueFamilyTask    = "taskq"
	defaultQueueScanCount   = int64(100)
	queueAgeSampleSize      = int64(32)
)

// redisQueueObserverClient is intentionally narrower than redis.UniversalClient.
// Keeping the observer on SCAN/TYPE/cardinality primitives makes it difficult to
// accidentally add a blocking KEYS call while retaining a small test seam.
type redisQueueObserverClient interface {
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	Type(ctx context.Context, key string) *redis.StatusCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
	LIndex(ctx context.Context, key string, index int64) *redis.StringCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
}

type redisQueueFamilySnapshot struct {
	length      int64
	active      int64
	maxLength   int64
	oldestAge   float64
	oldestKnown bool
}

// RedisQueueObserver exports bounded aggregate metrics for the legacy command
// queues and the unified device task queues. It never emits a device identity as
// a Prometheus label. A failed family observation leaves the last business gauge
// values untouched and only changes its health/failure indicators.
type RedisQueueObserver struct {
	client    redisQueueObserverClient
	metrics   *TaskMetrics
	interval  time.Duration
	scanCount int64
	logger    *zap.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewRedisQueueObserver creates a Redis queue observer with conservative scan
// bounds. The observer is inert when client or metrics is nil.
func NewRedisQueueObserver(client redis.UniversalClient, metrics *TaskMetrics, interval time.Duration, logger *zap.Logger) *RedisQueueObserver {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RedisQueueObserver{
		client:    client,
		metrics:   metrics,
		interval:  interval,
		scanCount: defaultQueueScanCount,
		logger:    logger.Named("redis-task-queue-observer"),
	}
}

// Start begins periodic observation. Start is idempotent.
func (o *RedisQueueObserver) Start(ctx context.Context) {
	if o == nil || o.client == nil || o.metrics == nil {
		return
	}
	o.mu.Lock()
	if o.cancel != nil {
		o.mu.Unlock()
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	o.cancel = cancel
	o.done = make(chan struct{})
	done := o.done
	o.mu.Unlock()

	go func() {
		defer close(done)
		o.Collect(workerCtx)
		ticker := time.NewTicker(o.interval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				o.Collect(workerCtx)
			}
		}
	}()
}

// Stop stops periodic observation and waits for the current collection to exit.
func (o *RedisQueueObserver) Stop() {
	if o == nil {
		return
	}
	o.mu.Lock()
	cancel, done := o.cancel, o.done
	o.cancel, o.done = nil, nil
	o.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

// Collect performs one bounded observation round. It is exported for tests and
// for operational probes that want an immediate sample after startup.
func (o *RedisQueueObserver) Collect(ctx context.Context) {
	if o == nil || o.client == nil || o.metrics == nil {
		return
	}
	o.collectFamily(ctx, redisQueueFamilyCommand, redisx.Keys.ACSCommandQueuePattern())
	o.collectFamily(ctx, redisQueueFamilyTask, redisx.Keys.ACSTaskQueuePattern())
}

func (o *RedisQueueObserver) collectFamily(ctx context.Context, family, pattern string) {
	started := time.Now()
	snapshot, err := o.scanFamily(ctx, family, pattern)
	duration := time.Since(started).Seconds()
	o.metrics.RedisTaskQueueScanDuration.WithLabelValues(family).Set(duration)
	if err != nil {
		o.metrics.RedisTaskQueueScanFailuresTotal.WithLabelValues(family).Inc()
		o.metrics.RedisTaskQueueUp.WithLabelValues(family).Set(0)
		o.logger.Warn("redis queue observation failed", zap.String("queue_family", family), zap.Error(err))
		return
	}

	queue := o.metrics.RedisTaskQueueLengthTotal.WithLabelValues(family)
	active := o.metrics.RedisTaskQueueActiveDevices.WithLabelValues(family)
	maxLength := o.metrics.RedisTaskQueueMaxLength.WithLabelValues(family)
	oldestAge := o.metrics.RedisTaskQueueOldestAgeSeconds.WithLabelValues(family)
	queue.Set(float64(snapshot.length))
	active.Set(float64(snapshot.active))
	maxLength.Set(float64(snapshot.maxLength))
	if snapshot.length == 0 || snapshot.oldestKnown {
		oldestAge.Set(snapshot.oldestAge)
	}
	o.metrics.RedisTaskQueueUp.WithLabelValues(family).Set(1)
	o.metrics.RedisTaskQueueSampleTimestamp.WithLabelValues(family).Set(float64(time.Now().Unix()))
}

func (o *RedisQueueObserver) scanFamily(ctx context.Context, family, pattern string) (redisQueueFamilySnapshot, error) {
	var snapshot redisQueueFamilySnapshot
	var cursor uint64
	now := time.Now()
	for {
		keys, next, err := o.client.Scan(ctx, cursor, pattern, o.scanCount).Result()
		if err != nil {
			return redisQueueFamilySnapshot{}, fmt.Errorf("scan %s: %w", family, err)
		}
		for _, key := range keys {
			length, oldest, known, err := o.inspectQueue(ctx, family, key, now)
			if err != nil {
				return redisQueueFamilySnapshot{}, fmt.Errorf("inspect %s: %w", key, err)
			}
			snapshot.length += length
			if length > 0 {
				snapshot.active++
			}
			if length > snapshot.maxLength {
				snapshot.maxLength = length
			}
			if known && (!snapshot.oldestKnown || oldest > snapshot.oldestAge) {
				snapshot.oldestAge = oldest
				snapshot.oldestKnown = true
			}
		}
		if next == 0 {
			return snapshot, nil
		}
		cursor = next
	}
}

func (o *RedisQueueObserver) inspectQueue(ctx context.Context, family, key string, now time.Time) (int64, float64, bool, error) {
	kind, err := o.client.Type(ctx, key).Result()
	if err != nil {
		return 0, 0, false, fmt.Errorf("type: %w", err)
	}
	switch kind {
	case "none":
		// SCAN does not provide a snapshot. A queue can legitimately become
		// empty and disappear before TYPE runs; treat that race as an empty
		// queue while keeping all other foreign types visible as corruption.
		return 0, 0, false, nil
	case "zset":
		length, err := o.client.ZCard(ctx, key).Result()
		if err != nil {
			return 0, 0, false, fmt.Errorf("zcard: %w", err)
		}
		oldest, known, err := o.oldestFromZSet(ctx, family, key, now)
		return length, oldest, known, err
	case "list":
		length, err := o.client.LLen(ctx, key).Result()
		if err != nil {
			return 0, 0, false, fmt.Errorf("llen: %w", err)
		}
		if length == 0 {
			return 0, 0, false, nil
		}
		member, err := o.client.LIndex(ctx, key, 0).Result()
		if err != nil && err != redis.Nil {
			return 0, 0, false, fmt.Errorf("lindex: %w", err)
		}
		age, known := queueMemberAge(member, now)
		return length, age, known, nil
	default:
		return 0, 0, false, fmt.Errorf("unsupported redis type %q", kind)
	}
}

func (o *RedisQueueObserver) oldestFromZSet(ctx context.Context, family, key string, now time.Time) (float64, bool, error) {
	entries, err := o.client.ZRangeWithScores(ctx, key, 0, queueAgeSampleSize-1).Result()
	if err != nil {
		return 0, false, fmt.Errorf("zrange: %w", err)
	}
	var oldest float64
	known := false
	for _, entry := range entries {
		member, ok := entry.Member.(string)
		if !ok {
			continue
		}
		var age float64
		var memberKnown bool
		if family == redisQueueFamilyTask {
			data, getErr := o.client.HGet(ctx, redisx.Keys.ACSTaskDetail(member), "data").Result()
			if getErr == nil {
				var queued Task
				if json.Unmarshal([]byte(data), &queued) == nil && !queued.QueueTime().IsZero() {
					age, memberKnown = ageSeconds(queued.QueueTime(), now), true
				}
			} else if getErr != redis.Nil {
				return 0, false, fmt.Errorf("task detail: %w", getErr)
			}
			// Task detail hashes intentionally have a short TTL. A queue entry can
			// outlive its hash during a stale backlog, but its sorted-set score is
			// still available. The score includes a priority offset, so this is a
			// conservative lower-bound age; fresh entries use the exact detail time
			// above whenever it is readable.
			if !memberKnown {
				age, memberKnown = queueScoreAge(entry.Score, now)
			}
		} else {
			age, memberKnown = queueMemberAge(member, now)
		}
		if memberKnown && (!known || age > oldest) {
			oldest, known = age, true
		}
	}
	return oldest, known, nil
}

// queueScoreAge returns an age from the timestamp-shaped part of a task queue
// score when its detail hash is unavailable. queueScore stores Unix nanoseconds
// plus a priority offset; treating the complete score as a timestamp therefore
// underestimates the true age by that offset, but preserves a useful positive
// backlog signal without inventing a precise value.
func queueScoreAge(score float64, now time.Time) (float64, bool) {
	// Keep the float-to-int64 conversion bounded for malformed or foreign
	// sorted-set scores; normal Unix-nanosecond scores are ~1e18.
	if math.IsNaN(score) || math.IsInf(score, 0) || score <= 0 || score >= 9.223372036854776e18 {
		return 0, false
	}
	var queuedAt time.Time
	switch {
	case score >= 1e15:
		// Current task queues use Unix nanoseconds (plus priority offset).
		queuedAt = time.Unix(0, int64(score))
	case score >= 1e9:
		// Be tolerant of legacy queues that used Unix seconds.
		queuedAt = time.Unix(int64(score), 0)
	default:
		return 0, false
	}
	if queuedAt.After(now) {
		return 0, false
	}
	return ageSeconds(queuedAt, now), true
}

func ageSeconds(queuedAt, now time.Time) float64 {
	age := now.Sub(queuedAt).Seconds()
	if age < 0 {
		return 0
	}
	return age
}

// queueMemberAge handles the legacy command payload only when it contains an
// explicit timestamp. Most legacy commands do not, so the metric remains at its
// initialized zero while count/active/max still provide reliable backlog data.
func queueMemberAge(member string, now time.Time) (float64, bool) {
	var payload map[string]any
	if json.Unmarshal([]byte(member), &payload) != nil {
		return 0, false
	}
	for _, field := range []string{"created_at", "createdAt", "queued_at", "queue_time", "queueTime"} {
		if value, ok := payload[field]; ok {
			if timestamp, ok := parseQueueTimestamp(value); ok {
				return ageSeconds(timestamp, now), true
			}
		}
	}
	return 0, false
}

func parseQueueTimestamp(value any) (time.Time, bool) {
	switch v := value.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(v))
		return parsed, err == nil
	case float64:
		switch {
		case v > 1e15:
			return time.Unix(0, int64(v)), true
		case v > 1e9:
			return time.Unix(int64(v), 0), true
		}
	}
	return time.Time{}, false
}
