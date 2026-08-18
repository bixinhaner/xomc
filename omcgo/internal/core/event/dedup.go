package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Deduper 基于 Redis SETNX 在分布式环境下保证 EventBus handler 的幂等。
//
// 设计目标：在 NATS JetStream InterestPolicy 下，consumer 失活窗口内
// 已 publish 但未 ack 的消息会重投；handler 中的非幂等副作用（创建任务、
// 下载文件、推送 OSS）会被重复执行。Deduper 用 (namespace, evt.ID) 作 key
// 在 Redis 上 SETNX 抢锁；抢到才放行 handler，没抢到当作"已处理"直接 ack。
//
// Fail-open 语义：Redis 故障时退化为放行 handler，避免单点故障导致整个
// 事件链路停摆。代价是 redis 故障期间存在重复处理风险。
type Deduper struct {
	rdb    redis.UniversalClient
	ttl    time.Duration
	logger *zap.Logger
}

// NewDeduper 构造 Deduper。
// ttl 推荐 24h，覆盖滚动部署 + NATS NAK 重试窗口（最多 1+2+4+8+16=31s）。
func NewDeduper(rdb redis.UniversalClient, ttl time.Duration, logger *zap.Logger) *Deduper {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Deduper{rdb: rdb, ttl: ttl, logger: logger}
}

// ErrDedupRedis 表示 Redis 通信失败。调用方据此决定降级策略。
var ErrDedupRedis = errors.New("dedup redis error")

// ErrDedupInProgress keeps a concurrent delivery unacknowledged until the
// delivery that owns the processing lease either succeeds or releases it.
var ErrDedupInProgress = errors.New("event deduplication is in progress")

const dedupProcessingLease = 5 * time.Minute

var releaseDedupLeaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)

var completeDedupLeaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  redis.call("SET", KEYS[2], "1", "PX", ARGV[2])
  redis.call("DEL", KEYS[1])
  return 1
end
return 0`)

// FirstTime 检查 (namespace, eventID) 是否首次见。
//   - 首次：返回 (true, nil)
//   - 重复：返回 (false, nil)
//   - Redis 出错：返回 (true, ErrDedupRedis-wrapped)，**fail-open**
//   - eventID 为空：返回 (true, nil)，无法去重时放行
func (d *Deduper) FirstTime(ctx context.Context, namespace, eventID string) (bool, error) {
	if eventID == "" {
		return true, nil
	}
	key := fmt.Sprintf("dedup:%s:%s", namespace, eventID)
	ok, err := d.rdb.SetNX(ctx, key, "1", d.ttl).Result()
	if err != nil {
		return true, fmt.Errorf("%w: setnx %s: %v", ErrDedupRedis, key, err)
	}
	return ok, nil
}

// Wrap 把 EventHandler 包装为带去重的 EventHandler。
//   - 首次事件：调用底层 handler，返回其结果
//   - 重复事件：返回 nil（让 NATS ack 掉重投消息）
//   - Redis 错误：fail-open，调用底层 handler
//
// handler 失败由 NATS 层 NAK 重试，dedup key 保留 → 重投时仍跳过 handler，
// 这与"NAK 重试"语义冲突。**因此 Wrap 仅适用于无需 NAK 重试的场景**：
// 业务侧需要在 handler 内部完成所有重试（DB 事务原子化、外部 API 自带 retry）。
// 对必须依赖 NAK 重试的事件（如临时网络错误后的指数退避），不应使用 Wrap。
func (d *Deduper) Wrap(namespace string, handler EventHandler) EventHandler {
	return func(ctx context.Context, evt Event) error {
		first, err := d.FirstTime(ctx, namespace, evt.ID)
		if err != nil {
			d.logger.Warn("dedup check failed, falling open",
				zap.String("namespace", namespace),
				zap.String("event_id", evt.ID),
				zap.String("subject", evt.Subject),
				zap.Error(err))
			return handler(ctx, evt)
		}
		if !first {
			d.logger.Debug("event already processed, skipping",
				zap.String("namespace", namespace),
				zap.String("event_id", evt.ID),
				zap.String("subject", evt.Subject))
			return nil
		}
		return handler(ctx, evt)
	}
}

// WrapAfterSuccess suppresses redelivery only after the wrapped handler has
// completed successfully. It is intended for handlers that rely on the event
// bus NAK/redelivery contract: a failed attempt is left unmarked and can run
// again, while a successfully projected event remains idempotent.
//
// The Redis check is deliberately fail-open. A Redis outage may cause a
// duplicate projection, but must never acknowledge an event whose durable
// projection failed.
func (d *Deduper) WrapAfterSuccess(namespace string, handler EventHandler) EventHandler {
	return func(ctx context.Context, evt Event) error {
		if evt.ID == "" || d == nil || d.rdb == nil {
			return handler(ctx, evt)
		}
		doneKey := fmt.Sprintf("dedup:%s:%s", namespace, evt.ID)
		leaseKey := doneKey + ":processing"
		seen, err := d.rdb.Exists(ctx, doneKey).Result()
		if err != nil {
			d.logger.Warn("dedup check failed, falling open",
				zap.String("namespace", namespace),
				zap.String("event_id", evt.ID),
				zap.String("subject", evt.Subject),
				zap.Error(err))
		} else if seen > 0 {
			return nil
		}
		token := uuid.NewString()
		acquired, err := d.rdb.SetNX(ctx, leaseKey, token, dedupProcessingLease).Result()
		if err != nil {
			d.logger.Warn("dedup processing lease failed, falling open",
				zap.String("namespace", namespace), zap.String("event_id", evt.ID), zap.Error(err))
			return handler(ctx, evt)
		}
		if !acquired {
			return ErrDedupInProgress
		}

		if err := handler(ctx, evt); err != nil {
			if releaseErr := releaseDedupLeaseScript.Run(ctx, d.rdb, []string{leaseKey}, token).Err(); releaseErr != nil {
				d.logger.Warn("release failed event dedup lease",
					zap.String("namespace", namespace), zap.String("event_id", evt.ID), zap.Error(releaseErr))
			}
			return err
		}
		if err := completeDedupLeaseScript.Run(
			ctx, d.rdb, []string{leaseKey, doneKey}, token, d.ttl.Milliseconds(),
		).Err(); err != nil {
			d.logger.Warn("record completed event dedup marker",
				zap.String("namespace", namespace),
				zap.String("event_id", evt.ID),
				zap.String("subject", evt.Subject),
				zap.Error(err))
		}
		return nil
	}
}
