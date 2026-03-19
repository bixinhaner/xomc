package upload

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims for upload tokens.
type Claims struct {
	jwt.RegisteredClaims
	DeviceSN       string `json:"device_sn"`
	CommandKey     string `json:"command_key"`
	FileType       string `json:"file_type"`
	TargetFileName string `json:"target_filename"`
}

// TokenManager manages JWT tokens for file uploads.
type TokenManager struct {
	secretKey []byte
	ttl       time.Duration
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager(secretKey string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secretKey: []byte(secretKey),
		ttl:       ttl,
	}
}

// Generate creates a new upload token.
func (m *TokenManager) Generate(deviceSN, commandKey, fileType, filename string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        commandKey,
		},
		DeviceSN:       deviceSN,
		CommandKey:     commandKey,
		FileType:       fileType,
		TargetFileName: filename,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// Validate validates the token and returns the claims.
func (m *TokenManager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
