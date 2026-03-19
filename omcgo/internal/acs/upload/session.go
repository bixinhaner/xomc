package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Session represents an upload session stored in Redis.
type Session struct {
	DeviceSN       string    `json:"device_sn"`
	CommandKey     string    `json:"command_key"`
	FileType       string    `json:"file_type"`
	Bucket         string    `json:"bucket"`
	ObjectPath     string    `json:"object_path"`
	FileSize       int64     `json:"file_size"`
	UploadedAt     time.Time `json:"uploaded_at"`
	TargetFileName string    `json:"target_filename"`
}

// SessionStore manages upload sessions in Redis.
type SessionStore struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewSessionStore creates a new SessionStore.
func NewSessionStore(redis *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{
		redis: redis,
		ttl:   ttl,
	}
}

// Save stores an upload session.
func (s *SessionStore) Save(ctx context.Context, session *Session) error {
	key := s.key(session.DeviceSN, session.CommandKey)
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return s.redis.Set(ctx, key, data, s.ttl).Err()
}

// Get retrieves an upload session.
func (s *SessionStore) Get(ctx context.Context, deviceSN, commandKey string) (*Session, error) {
	key := s.key(deviceSN, commandKey)
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

// Delete removes an upload session.
func (s *SessionStore) Delete(ctx context.Context, deviceSN, commandKey string) error {
	key := s.key(deviceSN, commandKey)
	return s.redis.Del(ctx, key).Err()
}

func (s *SessionStore) key(deviceSN, commandKey string) string {
	return fmt.Sprintf("upload:session:%s:%s", deviceSN, commandKey)
}
