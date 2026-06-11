package auth

import (
	"context"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// NonceStore 管理 HTTP Digest 一次性 nonce 的存取。
//
// issue #65（Option B）：digest nonce 原为进程内 map，在多实例无亲和部署下
// Challenge 落在 A、Authenticate 落在 B 时 nonce 命中失败 → 401 死循环。抽象为接口
// 后：单实例 / 测试用 memoryNonceStore（无 Redis 依赖）；多实例用 redisNonceStore
// （SETEX 写 + GETDEL 原子一次性消费），Challenge 与 Authenticate 可落在不同实例。
type NonceStore interface {
	// Store 写入一个新 nonce，TTL 到期自动失效。
	Store(ctx context.Context, nonce string)
	// Consume 原子地取出并删除一个 nonce。返回 true 表示该 nonce 存在且未被用过
	// （一次性语义：第二次 Consume 同一 nonce 返回 false）。
	Consume(ctx context.Context, nonce string) bool
}

// memoryNonceStore 是进程内实现（单实例 / 测试）。带后台清理 goroutine 驱逐过期项。
type memoryNonceStore struct {
	mu     sync.Mutex
	nonces map[string]time.Time
	ttl    time.Duration
	stop   chan struct{}
}

// NewMemoryNonceStore 创建进程内 nonce 存储并启动清理循环。
func NewMemoryNonceStore(ttl time.Duration) NonceStore {
	if ttl <= 0 {
		ttl = nonceTTL
	}
	s := &memoryNonceStore{
		nonces: make(map[string]time.Time),
		ttl:    ttl,
		stop:   make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func (s *memoryNonceStore) Store(_ context.Context, nonce string) {
	s.mu.Lock()
	s.nonces[nonce] = time.Now()
	s.mu.Unlock()
}

func (s *memoryNonceStore) Consume(_ context.Context, nonce string) bool {
	s.mu.Lock()
	created, exists := s.nonces[nonce]
	if exists {
		delete(s.nonces, nonce)
	}
	s.mu.Unlock()
	return exists && time.Since(created) <= s.ttl
}

func (s *memoryNonceStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for nonce, created := range s.nonces {
				if now.Sub(created) > s.ttl {
					delete(s.nonces, nonce)
				}
			}
			s.mu.Unlock()
		}
	}
}

// redisNonceStore 是 Redis 实现（多实例横扩）。无需后台清理：SETEX 的 TTL 由 Redis
// 自动过期，GETDEL 保证一次性消费的原子性。
type redisNonceStore struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewRedisNonceStore 创建 Redis 后端的 nonce 存储。
func NewRedisNonceStore(rdb redis.UniversalClient, ttl time.Duration) NonceStore {
	if ttl <= 0 {
		ttl = nonceTTL
	}
	return &redisNonceStore{rdb: rdb, ttl: ttl}
}

func (s *redisNonceStore) Store(ctx context.Context, nonce string) {
	// 值无意义（仅判存在），用 "1" 占位。SETEX 写入带 TTL。
	s.rdb.Set(ctx, redisx.Keys.ACSAuthNonce(nonce), "1", s.ttl)
}

func (s *redisNonceStore) Consume(ctx context.Context, nonce string) bool {
	// GETDEL：原子取出并删除。命中（非 redis.Nil）即表示该 nonce 存在且本次首用。
	err := s.rdb.GetDel(ctx, redisx.Keys.ACSAuthNonce(nonce)).Err()
	return err == nil
}
