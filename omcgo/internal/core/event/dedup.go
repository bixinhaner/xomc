package event

import (
	"context"
	"errors"
	"fmt"
	"time"

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
