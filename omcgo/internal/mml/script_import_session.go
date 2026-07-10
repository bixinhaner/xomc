package mml

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const defaultImportSessionTTL = 15 * time.Minute

var (
	// ErrImportTokenExpired means the validation token was never found or its
	// Redis TTL elapsed. Callers must validate the TXT file again.
	ErrImportTokenExpired = errors.New("script import validation token expired")
	// ErrImportTokenOwnerMismatch means a different authenticated user supplied
	// a still-active validation token.
	ErrImportTokenOwnerMismatch = errors.New("script import validation token belongs to another user")
	// ErrImportTokenClaimed means another save request currently owns the token.
	ErrImportTokenClaimed = errors.New("script import validation token is claimed")
	// ErrImportTokenConsumed means a successful save already finalized the token.
	ErrImportTokenConsumed = errors.New("script import validation token is consumed")
)

// ImportSession is the server-authoritative validation snapshot retained between
// TXT validation and script creation. The client only receives the random token;
// it cannot supply or alter these persisted fields at save time.
type ImportSession struct {
	ID                uuid.UUID              `json:"id"`
	OriginalFilename  string                 `json:"original_filename"`
	NormalizedContent string                 `json:"normalized_content"`
	ContentSHA256     string                 `json:"content_sha256"`
	ValidationVersion string                 `json:"validation_version"`
	Validation        ScriptValidationResult `json:"validation"`
}

// ImportSessionStore manages one-time validation snapshots.
type ImportSessionStore interface {
	Put(ctx context.Context, username string, session *ImportSession) (string, error)
	Get(ctx context.Context, token, username string) (*ImportSession, error)
	Claim(ctx context.Context, token, username, requestID string) (*ImportSession, error)
	Release(ctx context.Context, token, username, requestID string) error
	Finalize(ctx context.Context, token, username, requestID string) error
}

type storedImportSession struct {
	Username       string        `json:"username"`
	ClaimRequestID string        `json:"claim_request_id"`
	Session        ImportSession `json:"session"`
}

// RedisImportSessionStore uses token hashes as Redis keys and Lua compare-and-
// swap operations so parallel app instances cannot consume the same session.
type RedisImportSessionStore struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewRedisImportSessionStore creates a Redis-backed one-time validation-session
// store. A non-positive TTL falls back to the required 15-minute lifetime.
func NewRedisImportSessionStore(rdb redis.UniversalClient, ttl time.Duration) *RedisImportSessionStore {
	if ttl <= 0 {
		ttl = defaultImportSessionTTL
	}
	return &RedisImportSessionStore{rdb: rdb, ttl: ttl}
}

// Put creates a fresh UUID and a 32-byte cryptographically random token. Redis
// stores only SHA-256(token) in its key, never the raw bearer token.
func (s *RedisImportSessionStore) Put(ctx context.Context, username string, session *ImportSession) (string, error) {
	if s == nil || s.rdb == nil {
		return "", fmt.Errorf("put import session: redis client is nil")
	}
	if username == "" {
		return "", fmt.Errorf("put import session: username is empty")
	}
	if session == nil {
		return "", fmt.Errorf("put import session: session is nil")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate import session token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	sessionCopy := *session
	sessionCopy.ID = uuid.New()
	record := storedImportSession{Username: username, Session: sessionCopy}
	payload, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("encode import session: %w", err)
	}
	if err := s.rdb.Set(ctx, s.sessionKey(token), payload, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("store import session: %w", err)
	}
	return token, nil
}

// Get reads a still-active session and checks its authenticated owner.
func (s *RedisImportSessionStore) Get(ctx context.Context, token, username string) (*ImportSession, error) {
	if err := s.ensureClient("get"); err != nil {
		return nil, err
	}
	if consumed, err := s.isConsumed(ctx, token); err != nil {
		return nil, err
	} else if consumed {
		return nil, ErrImportTokenConsumed
	}
	record, err := s.getRecord(ctx, token)
	if err != nil {
		return nil, err
	}
	if record.Username != username {
		return nil, ErrImportTokenOwnerMismatch
	}
	return &record.Session, nil
}

// claimImportSessionScript atomically binds a previously-unclaimed session to a
// save request. Return values: 1=claimed, 2=consumed, 3=expired, 4=wrong owner,
// 5=claimed by another request. The original TTL is preserved on mutation.
var claimImportSessionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then return 2 end
local raw = redis.call('GET', KEYS[1])
if not raw then return 3 end
local record = cjson.decode(raw)
if record.username ~= ARGV[1] then return 4 end
if record.claim_request_id and record.claim_request_id ~= '' then
  if record.claim_request_id == ARGV[2] then return 1 end
  return 5
end
record.claim_request_id = ARGV[2]
local ttl = redis.call('PTTL', KEYS[1])
redis.call('SET', KEYS[1], cjson.encode(record), 'PX', ttl)
return 1
`)

// Claim atomically reserves the token for a save request. Repeating the same
// request ID is idempotent; a distinct request receives ErrImportTokenClaimed.
func (s *RedisImportSessionStore) Claim(ctx context.Context, token, username, requestID string) (*ImportSession, error) {
	if err := s.ensureRequest("claim", username, requestID); err != nil {
		return nil, err
	}
	result, err := claimImportSessionScript.Run(ctx, s.rdb, []string{s.sessionKey(token), s.consumedKey(token)}, username, requestID).Int64()
	if err != nil {
		return nil, fmt.Errorf("claim import session: %w", err)
	}
	if err := importSessionResultError(result); err != nil {
		return nil, err
	}
	record, err := s.getRecord(ctx, token)
	if err != nil {
		return nil, err
	}
	return &record.Session, nil
}

// releaseImportSessionScript clears a claim only if it belongs to the caller.
var releaseImportSessionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then return 2 end
local raw = redis.call('GET', KEYS[1])
if not raw then return 3 end
local record = cjson.decode(raw)
if record.username ~= ARGV[1] then return 4 end
if record.claim_request_id ~= ARGV[2] then return 5 end
record.claim_request_id = ''
local ttl = redis.call('PTTL', KEYS[1])
redis.call('SET', KEYS[1], cjson.encode(record), 'PX', ttl)
return 1
`)

// Release gives an unsuccessfully saved session back to its owning request.
func (s *RedisImportSessionStore) Release(ctx context.Context, token, username, requestID string) error {
	if err := s.ensureRequest("release", username, requestID); err != nil {
		return err
	}
	result, err := releaseImportSessionScript.Run(ctx, s.rdb, []string{s.sessionKey(token), s.consumedKey(token)}, username, requestID).Int64()
	if err != nil {
		return fmt.Errorf("release import session: %w", err)
	}
	return importSessionResultError(result)
}

// finalizeImportSessionScript deletes the active session and creates a consumed
// tombstone in one Redis operation. The tombstone TTL is a fresh session TTL.
var finalizeImportSessionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then return 2 end
local raw = redis.call('GET', KEYS[1])
if not raw then return 3 end
local record = cjson.decode(raw)
if record.username ~= ARGV[1] then return 4 end
if record.claim_request_id ~= ARGV[2] then return 5 end
redis.call('DEL', KEYS[1])
redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
return 1
`)

// Finalize permanently consumes a session after its database transaction has
// succeeded, preserving a short-lived tombstone for replay detection.
func (s *RedisImportSessionStore) Finalize(ctx context.Context, token, username, requestID string) error {
	if err := s.ensureRequest("finalize", username, requestID); err != nil {
		return err
	}
	result, err := finalizeImportSessionScript.Run(ctx, s.rdb, []string{s.sessionKey(token), s.consumedKey(token)}, username, requestID, int64(s.ttl.Seconds())).Int64()
	if err != nil {
		return fmt.Errorf("finalize import session: %w", err)
	}
	return importSessionResultError(result)
}

func (s *RedisImportSessionStore) ensureClient(operation string) error {
	if s == nil || s.rdb == nil {
		return fmt.Errorf("%s import session: redis client is nil", operation)
	}
	return nil
}

func (s *RedisImportSessionStore) ensureRequest(operation, username, requestID string) error {
	if err := s.ensureClient(operation); err != nil {
		return err
	}
	if username == "" {
		return fmt.Errorf("%s import session: username is empty", operation)
	}
	if requestID == "" {
		return fmt.Errorf("%s import session: request ID is empty", operation)
	}
	return nil
}

func (s *RedisImportSessionStore) getRecord(ctx context.Context, token string) (*storedImportSession, error) {
	payload, err := s.rdb.Get(ctx, s.sessionKey(token)).Bytes()
	if err == redis.Nil {
		return nil, ErrImportTokenExpired
	}
	if err != nil {
		return nil, fmt.Errorf("get import session: %w", err)
	}
	var record storedImportSession
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, fmt.Errorf("decode import session: %w", err)
	}
	return &record, nil
}

func (s *RedisImportSessionStore) isConsumed(ctx context.Context, token string) (bool, error) {
	result, err := s.rdb.Exists(ctx, s.consumedKey(token)).Result()
	if err != nil {
		return false, fmt.Errorf("check consumed import session: %w", err)
	}
	return result == 1, nil
}

func (s *RedisImportSessionStore) sessionKey(token string) string {
	return redisx.Keys.MMLScriptImportSession(importTokenSHA256(token))
}

func (s *RedisImportSessionStore) consumedKey(token string) string {
	return redisx.Keys.MMLScriptImportConsumed(importTokenSHA256(token))
}

func importTokenSHA256(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func importSessionResultError(result int64) error {
	switch result {
	case 1:
		return nil
	case 2:
		return ErrImportTokenConsumed
	case 3:
		return ErrImportTokenExpired
	case 4:
		return ErrImportTokenOwnerMismatch
	case 5:
		return ErrImportTokenClaimed
	default:
		return fmt.Errorf("import session atomic operation returned unexpected result %d", result)
	}
}
