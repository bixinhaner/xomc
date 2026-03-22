package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	captchaKeyPrefix = "auth:captcha:"
	captchaTTL       = 5 * time.Minute
)

// CaptchaService generates and validates CAPTCHA challenges using Redis.
type CaptchaService struct {
	redis redis.UniversalClient
}

// NewCaptchaService creates a new CaptchaService.
func NewCaptchaService(client redis.UniversalClient) *CaptchaService {
	return &CaptchaService{redis: client}
}

// CaptchaChallenge is the response containing a CAPTCHA challenge for the client.
type CaptchaChallenge struct {
	CaptchaID string `json:"captcha_id"`
	Question  string `json:"question"`
}

// Generate creates a new math CAPTCHA challenge and stores the answer in Redis.
func (s *CaptchaService) Generate(ctx context.Context) (*CaptchaChallenge, error) {
	id, err := generateCaptchaID()
	if err != nil {
		return nil, fmt.Errorf("generate captcha id: %w", err)
	}

	a, _ := rand.Int(rand.Reader, big.NewInt(50))
	b, _ := rand.Int(rand.Reader, big.NewInt(50))
	answer := fmt.Sprintf("%d", a.Int64()+b.Int64())
	question := fmt.Sprintf("%d + %d = ?", a.Int64(), b.Int64())

	key := captchaKeyPrefix + id
	if err := s.redis.Set(ctx, key, answer, captchaTTL).Err(); err != nil {
		return nil, fmt.Errorf("store captcha answer: %w", err)
	}

	return &CaptchaChallenge{
		CaptchaID: id,
		Question:  question,
	}, nil
}

// Verify checks the provided answer against the stored answer and deletes it (one-time use).
func (s *CaptchaService) Verify(ctx context.Context, captchaID, answer string) bool {
	if captchaID == "" || answer == "" {
		return false
	}

	key := captchaKeyPrefix + captchaID
	stored, err := s.redis.GetDel(ctx, key).Result()
	if err != nil {
		return false
	}

	return stored == answer
}

func generateCaptchaID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
