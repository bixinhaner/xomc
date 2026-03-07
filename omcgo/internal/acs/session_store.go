package acs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionStore defines the interface for persisting TR069 session state.
type SessionStore interface {
	Create(ctx context.Context, deviceSN string, session *Session) error
	Get(ctx context.Context, deviceSN string) (*Session, error)
	Update(ctx context.Context, deviceSN string, session *Session) error
	Delete(ctx context.Context, deviceSN string) error
	SetTTL(ctx context.Context, deviceSN string, ttl time.Duration) error
}

const sessionKeyPrefix = "acs:session:"

// RedisSessionStore implements SessionStore using Redis.
type RedisSessionStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

// NewRedisSessionStore creates a new Redis-backed session store.
func NewRedisSessionStore(client redis.UniversalClient, ttl time.Duration) *RedisSessionStore {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &RedisSessionStore{client: client, ttl: ttl}
}

func sessionKey(deviceSN string) string {
	return sessionKeyPrefix + deviceSN
}

func (s *RedisSessionStore) Create(ctx context.Context, deviceSN string, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.client.Set(ctx, sessionKey(deviceSN), data, s.ttl).Err()
}

func (s *RedisSessionStore) Get(ctx context.Context, deviceSN string) (*Session, error) {
	data, err := s.client.Get(ctx, sessionKey(deviceSN)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

func (s *RedisSessionStore) Update(ctx context.Context, deviceSN string, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.client.Set(ctx, sessionKey(deviceSN), data, s.ttl).Err()
}

func (s *RedisSessionStore) Delete(ctx context.Context, deviceSN string) error {
	return s.client.Del(ctx, sessionKey(deviceSN)).Err()
}

func (s *RedisSessionStore) SetTTL(ctx context.Context, deviceSN string, ttl time.Duration) error {
	return s.client.Expire(ctx, sessionKey(deviceSN), ttl).Err()
}
