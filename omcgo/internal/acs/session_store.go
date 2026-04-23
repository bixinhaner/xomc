package acs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// SessionStore 定义 TR069 会话状态持久化接口。
// 使用 Session ID（Cookie 值）作为主键，支持 Cookie-based 会话管理。
type SessionStore interface {
	// GetByID 通过 Session ID 获取会话
	GetByID(ctx context.Context, sessionID string) (*Session, error)
	// CreateWithID 创建新会话，使用指定的 Session ID
	CreateWithID(ctx context.Context, sessionID string, session *Session) error
	// UpdateByID 更新指定 Session ID 的会话
	UpdateByID(ctx context.Context, sessionID string, session *Session) error
	// DeleteByID 删除指定 Session ID 的会话
	DeleteByID(ctx context.Context, sessionID string) error
}

// RedisSessionStore 使用 Redis 实现 SessionStore。
type RedisSessionStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

// NewRedisSessionStore 创建 Redis 后端的会话存储。
func NewRedisSessionStore(client redis.UniversalClient, ttl time.Duration) *RedisSessionStore {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &RedisSessionStore{client: client, ttl: ttl}
}

func sessionByIDKey(sessionID string) string {
	return redisx.Keys.ACSSession(sessionID)
}

// GetByID 通过 Session ID（Cookie 值）获取会话。
func (s *RedisSessionStore) GetByID(ctx context.Context, sessionID string) (*Session, error) {
	data, err := s.client.Get(ctx, sessionByIDKey(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

// CreateWithID 使用指定的 Session ID 存储会话。
func (s *RedisSessionStore) CreateWithID(ctx context.Context, sessionID string, session *Session) error {
	session.ID = sessionID
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.client.Set(ctx, sessionByIDKey(sessionID), data, s.ttl).Err()
}

// UpdateByID 更新指定 Session ID 的会话。
func (s *RedisSessionStore) UpdateByID(ctx context.Context, sessionID string, session *Session) error {
	session.ID = sessionID
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.client.Set(ctx, sessionByIDKey(sessionID), data, s.ttl).Err()
}

// DeleteByID 删除指定 Session ID 的会话。
func (s *RedisSessionStore) DeleteByID(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, sessionByIDKey(sessionID)).Err()
}
