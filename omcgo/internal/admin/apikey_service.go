package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const apiKeyPrefix = "omk_"

// APIKeyService manages API key lifecycle.
type APIKeyService struct {
	repo     *PgAPIKeyRepository
	userRepo UserRepository
	logger   *zap.Logger
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(repo *PgAPIKeyRepository, userRepo UserRepository, logger *zap.Logger) *APIKeyService {
	return &APIKeyService{
		repo:     repo,
		userRepo: userRepo,
		logger:   logger.Named("apikey"),
	}
}

// Create generates a new API key for the given user. The plaintext key is returned only once.
func (s *APIKeyService) Create(ctx context.Context, userID uuid.UUID, req CreateAPIKeyRequest) (*APIKeyCreatedResponse, error) {
	// Verify user exists
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("validate user: %w", err)
	}

	// Generate random key: omk_ + 32 hex chars = 36 chars total
	rawBytes := make([]byte, 16)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	plainKey := apiKeyPrefix + hex.EncodeToString(rawBytes)
	keyPrefixStr := plainKey[:8] // "omk_" + first 4 hex chars

	hash, err := hashSecret([]byte(plainKey))
	if err != nil {
		return nil, fmt.Errorf("hash key: %w", err)
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse expires_at: %w", err)
		}
		expiresAt = &t
	}

	scopes := req.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	key := &APIKey{
		UserID:    userID,
		Name:      req.Name,
		KeyPrefix: keyPrefixStr,
		KeyHash:   string(hash),
		Scopes:    scopes,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("store api key: %w", err)
	}

	s.logger.Info("api key created",
		zap.String("key_id", key.ID.String()),
		zap.String("user_id", userID.String()),
		zap.String("name", req.Name),
	)

	return &APIKeyCreatedResponse{
		ID:        key.ID,
		Name:      key.Name,
		Key:       plainKey,
		KeyPrefix: keyPrefixStr,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: key.CreatedAt,
	}, nil
}

// Validate checks a raw API key string and returns the associated APIKey if valid.
//
// 安全（issue #6）：候选遍历不在首个匹配处短路 return，而是把所有同前缀候选都跑完
// bcrypt 比较，并用 crypto/subtle 常量时间方式累积命中结果。这样响应耗时不随
// "命中位置 / 是否命中" 变化（消除时序侧信道，无法据此推断命中的是哪一条或共有几条）。
func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (*APIKey, error) {
	if len(rawKey) < 8 {
		return nil, fmt.Errorf("invalid api key format")
	}

	prefix := rawKey[:8]
	candidates, err := s.repo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("lookup api key: %w", err)
	}

	matched := selectMatchingAPIKey(candidates, rawKey)
	if matched == nil {
		return nil, fmt.Errorf("invalid api key")
	}

	// 命中后再做过期判断与 last_used 更新（耗时与是否命中无关，这两步本就只在命中时发生）。
	if matched.ExpiresAt != nil && time.Now().After(*matched.ExpiresAt) {
		return nil, fmt.Errorf("api key expired")
	}
	// Update last used (fire and forget)
	go func(id uuid.UUID) {
		_ = s.repo.UpdateLastUsed(context.Background(), id)
	}(matched.ID)
	return matched, nil
}

// selectMatchingAPIKey 从同前缀候选里挑出与 rawKey 匹配的那一条，无匹配返回 nil。
//
// 安全（issue #6）：遍历不在首个命中处短路，对每个候选都执行一次 bcrypt 比较，
// 并用 subtle.ConstantTimeCompare 把 0/1 命中标志折叠为不分支的选择。这样总耗时
// 只与候选数量有关，不随"命中位置 / 是否命中"变化，消除时序侧信道。
// 抽成纯函数便于在无 DB 环境下直接单测匹配行为。
func selectMatchingAPIKey(candidates []*APIKey, rawKey string) *APIKey {
	var matched *APIKey
	for _, k := range candidates {
		hit := 0
		if bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(rawKey)) == nil {
			hit = 1
		}
		if subtle.ConstantTimeCompare([]byte{byte(hit)}, []byte{1}) == 1 {
			matched = k
		}
	}
	return matched
}

// List returns all API keys for a user.
func (s *APIKeyService) List(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Revoke revokes an API key.
func (s *APIKeyService) Revoke(ctx context.Context, keyID uuid.UUID) error {
	return s.repo.Revoke(ctx, keyID)
}
