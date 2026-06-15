package redisx

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// OnlineIndex 是「在线设备索引」的 Redis Sorted Set 封装（key=acs:online，
// member=设备 SN，score=最近 Inform 的 unix 秒）。issue #397。
//
// 设计动机：原离线/在线信号挂在 ACS→NATS→APP 的异步事件链上，PM 上报洪峰压垮
// NATS 时该链饿死 → 设备被误判离线、在线总数不准。OnlineIndex 由 ACS 在收到
// Inform 当场（发 NATS 之前）同步写入，是「免 NATS」的存活信源：
//   - Mark  ：ZADD，O(log N)，幂等覆盖 score
//   - Count ：ZCOUNT [cutoff, +inf]，O(log N) 取「最近 N 秒内有上报」的在线总数
//   - Prune ：ZREMRANGEBYSCORE [-inf, cutoff)，周期清理过期成员，O(log N + M)
//
// 本期（加法优先）只提供索引与计数接口；离线判定仍由 reconciler 读 PG
// last_inform_at（后续 PR 再切到本索引）。
type OnlineIndex struct {
	client redis.Cmdable
	key    string
}

// NewOnlineIndex 构造索引。client 为 nil 时所有方法降级为安全 no-op（dev/test）。
func NewOnlineIndex(client redis.Cmdable) *OnlineIndex {
	return &OnlineIndex{client: client, key: Keys.ACSOnlineSet()}
}

// Mark 记录设备 sn 在 tsUnix（unix 秒）有 Inform。幂等：重复 Mark 覆盖 score。
// sn 为空或 client 不可用时 no-op，绝不让存活信号写入阻断 Inform 主流程。
func (o *OnlineIndex) Mark(ctx context.Context, sn string, tsUnix int64) error {
	if o == nil || o.client == nil || sn == "" {
		return nil
	}
	return o.client.ZAdd(ctx, o.key, redis.Z{Score: float64(tsUnix), Member: sn}).Err()
}

// Count 返回 score ≥ cutoffUnix 的成员数（即「最近一次上报不早于 cutoff」的在线设备数）。
// 用法：cutoffUnix = now - 在线窗口秒。ZCOUNT 的 score 区间天然忽略过期成员，无需先 Prune。
func (o *OnlineIndex) Count(ctx context.Context, cutoffUnix int64) (int64, error) {
	if o == nil || o.client == nil {
		return 0, nil
	}
	return o.client.ZCount(ctx, o.key, strconv.FormatInt(cutoffUnix, 10), "+inf").Result()
}

// Prune 删除 score < cutoffUnix 的成员（清理过期在线记录，控制集合规模），返回删除数。
// 用法：cutoffUnix = now - 保留窗口秒（保留窗口应 ≥ 最大离线阈值，避免误删仍可能在线的设备）。
func (o *OnlineIndex) Prune(ctx context.Context, cutoffUnix int64) (int64, error) {
	if o == nil || o.client == nil {
		return 0, nil
	}
	// "(" 前缀表示开区间（exclusive），即严格小于 cutoff。
	return o.client.ZRemRangeByScore(ctx, o.key, "-inf", "("+strconv.FormatInt(cutoffUnix, 10)).Result()
}
