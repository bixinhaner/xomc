package acs

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// AdmissionController 限制全局并发 TR069 会话数。
//
// issue #65（Option B）：从进程内计数器迁移到 Redis 共享 sorted set，使并发上限
// 在多实例无亲和部署下成为真正的「全局」上限而非「每实例 N×max」。槽位以 sessionID
// 为成员、过期 unix 秒为分数；丢失的 Release（实例崩溃 / Redis 抖动）由 TTL 过期分
// 自愈回收，不会把全局计数永久占满。
//
// 接口优先：保留 AdmissionController 抽象（仅 Acquire/Release 增加 ctx + sessionID
// 参数），单实例 / 测试可用 localAdmissionController，生产用 redisAdmissionController。
type AdmissionController interface {
	// Acquire 尝试为给定 sessionID 申请一个会话槽位。返回 true 表示准入。
	// sessionID 用作槽位成员，使 Acquire/Release 跨实例严格配对，杜绝下溢。
	Acquire(ctx context.Context, sessionID string) bool
	// Release 释放给定 sessionID 的会话槽位（幂等：重复 Release 同一 sessionID 无副作用）。
	Release(ctx context.Context, sessionID string)
	// Current 返回当前活跃会话数（已驱逐过期槽位后的精确值）。
	Current(ctx context.Context) int64
}

// localAdmissionController 是进程内计数器实现，用于单实例部署与单元测试。
// 与改造前的原子计数器行为一致；不依赖 Redis。
type localAdmissionController struct {
	maxSessions     int64
	currentSessions atomic.Int64
}

// NewAdmissionController 创建进程内准入控制器（单实例 / 测试）。
func NewAdmissionController(maxSessions int64) AdmissionController {
	if maxSessions <= 0 {
		maxSessions = 10000
	}
	return &localAdmissionController{maxSessions: maxSessions}
}

func (ac *localAdmissionController) Acquire(_ context.Context, _ string) bool {
	for {
		current := ac.currentSessions.Load()
		if current >= ac.maxSessions {
			return false
		}
		if ac.currentSessions.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (ac *localAdmissionController) Release(_ context.Context, _ string) {
	// 防御性下界：不允许计数下溢为负（理论上 Acquire/Release 配对，但保持鲁棒）。
	for {
		current := ac.currentSessions.Load()
		if current <= 0 {
			return
		}
		if ac.currentSessions.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func (ac *localAdmissionController) Current(_ context.Context) int64 {
	return ac.currentSessions.Load()
}

// defaultAdmissionSlotTTL 是单个准入槽位的存活上限。必须 ≥ 会话 TTL（5min）+ 余量，
// 这样正常完成 / 被 reaper 回收的会话在到期前一定会被显式 Release；只有实例崩溃这类
// 异常路径才依赖 TTL 兜底回收，避免误删仍在进行的长会话槽位。
const defaultAdmissionSlotTTL = 10 * time.Minute

// admitScript 是 Acquire 的原子 Lua：
//  1. ZREMRANGEBYSCORE 驱逐所有过期（score < now）槽位 —— 自愈丢失的 Release；
//  2. ZCARD 取当前活跃数；
//  3. 若 < max → ZADD sessionID(score=now+ttl) 占位并返回 1（准入）；否则返回 0（满）。
//
// KEYS[1]=slots zset；ARGV[1]=now 秒；ARGV[2]=max；ARGV[3]=expiry 秒(now+ttl)；ARGV[4]=sessionID。
var admitScript = redis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', '(' .. ARGV[1])
local n = redis.call('ZCARD', KEYS[1])
if tonumber(n) < tonumber(ARGV[2]) then
  redis.call('ZADD', KEYS[1], ARGV[3], ARGV[4])
  return 1
end
return 0
`)

// redisAdmissionController 是 Redis 共享 sorted set 实现，用于多实例横扩部署。
type redisAdmissionController struct {
	rdb         redis.UniversalClient
	maxSessions int64
	slotTTL     time.Duration
	logger      *zap.Logger
	// failOpen=false（默认）：Redis 不可用时 Acquire 返回 false（fail-closed，503）。
	// ACS 已硬依赖 Redis（会话/任务/STUN 都在 Redis），保守拒绝优于放任并发击穿。
	failOpen bool
	// now 可注入时钟（测试用，确定性推进过期）；nil 时用 time.Now。
	now func() time.Time
}

// NewRedisAdmissionController 创建 Redis 后端的全局准入控制器。
func NewRedisAdmissionController(rdb redis.UniversalClient, maxSessions int64, logger *zap.Logger) AdmissionController {
	if maxSessions <= 0 {
		maxSessions = 10000
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &redisAdmissionController{
		rdb:         rdb,
		maxSessions: maxSessions,
		slotTTL:     defaultAdmissionSlotTTL,
		logger:      logger,
	}
}

// nowFn 返回当前时间（测试可覆盖 ac.now）。
func (ac *redisAdmissionController) nowFn() time.Time {
	if ac.now != nil {
		return ac.now()
	}
	return time.Now()
}

// SetSlotTTL 覆盖默认槽位 TTL（测试用）。
func (ac *redisAdmissionController) SetSlotTTL(ttl time.Duration) {
	if ttl > 0 {
		ac.slotTTL = ttl
	}
}

func (ac *redisAdmissionController) Acquire(ctx context.Context, sessionID string) bool {
	if sessionID == "" {
		// 没有 sessionID 无法构造可配对槽位 —— 退化为准入但不占位，避免泄漏不可释放的槽位。
		// 实际调用方（handleInform）始终传非空 sessionID，这里仅防御。
		ac.logger.Warn("admission acquire with empty sessionID; admitting without slot")
		return true
	}
	now := ac.nowFn()
	expiry := now.Add(ac.slotTTL).Unix()
	key := redisx.Keys.ACSAdmissionSlots()
	res, err := admitScript.Run(ctx, ac.rdb, []string{key},
		now.Unix(), ac.maxSessions, expiry, sessionID).Int64()
	if err != nil {
		ac.logger.Error("admission acquire redis error; fail-closed",
			zap.String("session_id", sessionID),
			zap.Bool("fail_open", ac.failOpen),
			zap.Error(err))
		return ac.failOpen
	}
	return res == 1
}

func (ac *redisAdmissionController) Release(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	if err := ac.rdb.ZRem(ctx, redisx.Keys.ACSAdmissionSlots(), sessionID).Err(); err != nil {
		// 释放失败不致命：该槽位的 expiry 分数到期后会被 admitScript 的
		// ZREMRANGEBYSCORE 自动回收，全局计数最终自洽。
		ac.logger.Warn("admission release redis error (slot will TTL-reclaim)",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}
}

func (ac *redisAdmissionController) Current(ctx context.Context) int64 {
	key := redisx.Keys.ACSAdmissionSlots()
	// 先驱逐过期槽位再计数，返回精确活跃数（与 Acquire 看到的一致）。
	if err := ac.rdb.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("(%d", ac.nowFn().Unix())).Err(); err != nil {
		ac.logger.Warn("admission current: evict expired failed", zap.Error(err))
	}
	n, err := ac.rdb.ZCard(ctx, key).Result()
	if err != nil {
		ac.logger.Warn("admission current: zcard failed", zap.Error(err))
		return 0
	}
	return n
}
