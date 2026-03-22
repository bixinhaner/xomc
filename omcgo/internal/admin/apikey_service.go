package admin

import (
	"context"
	"crypto/rand"
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

	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
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
func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (*APIKey, error) {
	if len(rawKey) < 8 {
		return nil, fmt.Errorf("invalid api key format")
	}

	prefix := rawKey[:8]
	candidates, err := s.repo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("lookup api key: %w", err)
	}

	for _, k := range candidates {
		if err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(rawKey)); err == nil {
			// Key matched — check expiration
			if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
				return nil, fmt.Errorf("api key expired")
			}
			// Update last used (fire and forget)
			go func() {
				_ = s.repo.UpdateLastUsed(context.Background(), k.ID)
			}()
			return k, nil
		}
	}

	return nil, fmt.Errorf("invalid api key")
}

// List returns all API keys for a user.
func (s *APIKeyService) List(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Revoke revokes an API key.
func (s *APIKeyService) Revoke(ctx context.Context, keyID uuid.UUID) error {
	return s.repo.Revoke(ctx, keyID)
}
