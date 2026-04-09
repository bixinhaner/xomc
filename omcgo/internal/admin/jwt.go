package admin

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// JWTService handles JWT token generation and validation.
type JWTService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type jwtClaims struct {
	UserID        uuid.UUID          `json:"user_id"`
	Username      string             `json:"username"`
	Carrier       *model.CarrierCode `json:"carrier,omitempty"`
	Roles         []string           `json:"roles"`
	CurrentRoleID *uuid.UUID         `json:"current_role_id,omitempty"`
	jwt.RegisteredClaims
}

const minJWTSecretLength = 32

// NewJWTService creates a new JWTService with the given secret and default TTLs.
// Returns an error if the secret is empty or shorter than 32 characters.
func NewJWTService(secret string) (*JWTService, error) {
	if len(secret) < minJWTSecretLength {
		return nil, fmt.Errorf("JWT secret must be at least %d characters, got %d", minJWTSecretLength, len(secret))
	}
	return &JWTService{
		secret:          []byte(secret),
		accessTokenTTL:  30 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}, nil
}

// NewJWTServiceWithTTL creates a JWTService with custom TTLs.
func NewJWTServiceWithTTL(secret string, accessTTL, refreshTTL time.Duration) (*JWTService, error) {
	s, err := NewJWTService(secret)
	if err != nil {
		return nil, err
	}
	if accessTTL > 0 {
		s.accessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		s.refreshTokenTTL = refreshTTL
	}
	return s, nil
}

// GenerateTokenPair creates a new access/refresh token pair.
func (s *JWTService) GenerateTokenPair(claims *Claims) (*TokenPair, error) {
	now := time.Now()

	accessClaims := &jwtClaims{
		UserID:        claims.UserID,
		Username:      claims.Username,
		Carrier:       claims.Carrier,
		Roles:         claims.Roles,
		CurrentRoleID: claims.CurrentRoleID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			ID:        uuid.New().String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := &jwtClaims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Carrier:  claims.Carrier,
		Roles:    claims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "refresh",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenTTL)),
			ID:        uuid.New().String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    now.Add(s.accessTokenTTL),
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken validates an access token and returns the claims.
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, "access")
}

// ValidateRefreshToken validates a refresh token and returns the claims.
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, "refresh")
}

func (s *JWTService) validateToken(tokenString, expectedSubject string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	subject, _ := claims.GetSubject()
	if subject != expectedSubject {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedSubject, subject)
	}

	return &Claims{
		UserID:        claims.UserID,
		Username:      claims.Username,
		Carrier:       claims.Carrier,
		Roles:         claims.Roles,
		CurrentRoleID: claims.CurrentRoleID,
	}, nil
}
