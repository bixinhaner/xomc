package acs

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// ConnReqURLStore 持久化设备 Inform 上报的 ConnectionRequestURL（HTTP 唤醒回退用）。
//
// issue #65（Option B）：取代进程内 connReqURLCache sync.Map。会话后续唤（postSessionWake）
// 可能在非 Inform 的实例上重新唤醒设备；进程态缓存在那台实例上 miss → httpURL 为空 →
// 纯 HTTP-CR 设备的即时续唤失败。改为共享 Redis 后任意实例都能读到该 URL。
//
// CR-URL 选型（issue #65 第 3 项，选低风险方案）：
//   - 复用设备表 connection_request_url 列：与 App 侧 wakeDevice 同源，但 ACS 进程刻意
//     不强依赖 device repo（buildACSPathTranslator 是 nil-safe，acsConnReqSender 注释明确
//     说「HTTP URL 在 ACS handler 上下文不易取得」），把 device repo 接入 Inform 热路径
//     会引入新耦合，风险更高。
//   - 选 Redis 键 acs:connreq:url:{sn}（TTL）：完全镜像同一个 Inform 循环里已有的
//     stun.Store（acs:stun:{sn}），ACS 已硬依赖 Redis，无新耦合，是更低风险方案。
type ConnReqURLStore interface {
	// Set 写入设备的 ConnectionRequestURL（带 TTL）。
	Set(ctx context.Context, deviceSN, url string) error
	// Get 读取设备的 ConnectionRequestURL（空表示未缓存）。
	Get(ctx context.Context, deviceSN string) (string, error)
}

// defaultConnReqURLTTL 与 stun.Store 一致（30min），覆盖典型 Inform 间隔。
const defaultConnReqURLTTL = 30 * time.Minute

// RedisConnReqURLStore 用 Redis STRING + TTL 实现 ConnReqURLStore，镜像 stun.Store。
type RedisConnReqURLStore struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewRedisConnReqURLStore 创建 Redis 后端的 CR-URL 存储。
func NewRedisConnReqURLStore(rdb redis.UniversalClient, ttl time.Duration) *RedisConnReqURLStore {
	if ttl <= 0 {
		ttl = defaultConnReqURLTTL
	}
	return &RedisConnReqURLStore{rdb: rdb, ttl: ttl}
}

// Set 见接口注释。
func (s *RedisConnReqURLStore) Set(ctx context.Context, deviceSN, url string) error {
	if err := s.rdb.Set(ctx, redisx.Keys.ACSConnReqURL(deviceSN), url, s.ttl).Err(); err != nil {
		return fmt.Errorf("set connreq url: %w", err)
	}
	return nil
}

// Get 见接口注释。
func (s *RedisConnReqURLStore) Get(ctx context.Context, deviceSN string) (string, error) {
	v, err := s.rdb.Get(ctx, redisx.Keys.ACSConnReqURL(deviceSN)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get connreq url: %w", err)
	}
	return v, nil
}
