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
	// By Session ID (Cookie-based, primary methods)
	GetByID(ctx context.Context, sessionID string) (*Session, error)
	CreateWithID(ctx context.Context, sessionID string, session *Session) error
	UpdateByID(ctx context.Context, sessionID string, session *Session) error
	DeleteByID(ctx context.Context, sessionID string) error

	// By Device SN (legacy, kept for migration)
	Create(ctx context.Context, deviceSN string, session *Session) error
	Get(ctx context.Context, deviceSN string) (*Session, error)
	Update(ctx context.Context, deviceSN string, session *Session) error
	Delete(ctx context.Context, deviceSN string) error
	SetTTL(ctx context.Context, deviceSN string, ttl time.Duration) error
}

const sessionKeyPrefix = "acs:session:"
const sessionByIDKeyPrefix = "acs:session:id:"

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

func sessionByIDKey(sessionID string) string {
	return sessionByIDKeyPrefix + sessionID
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

// GetByID retrieves a session by its Session ID (Cookie value).
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

// CreateWithID stores a session with the given Session ID as the key.
func (s *RedisSessionStore) CreateWithID(ctx context.Context, sessionID string, session *Session) error {
	session.ID = sessionID
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.client.Set(ctx, sessionByIDKey(sessionID), data, s.ttl).Err()
}

// UpdateByID updates an existing session by its Session ID.
func (s *RedisSessionStore) UpdateByID(ctx context.Context, sessionID string, session *Session) error {
	session.ID = sessionID
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.client.Set(ctx, sessionByIDKey(sessionID), data, s.ttl).Err()
}

// DeleteByID removes a session by its Session ID.
func (s *RedisSessionStore) DeleteByID(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, sessionByIDKey(sessionID)).Err()
}
