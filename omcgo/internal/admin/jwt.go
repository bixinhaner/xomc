package admin

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService handles JWT token generation and validation.
type JWTService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// jwtClaims is the wire format. v1.0：移除 Carrier，新增 IsSuperAdmin。
// 旧 token 反序列化时 IsSuperAdmin 缺省 false，老的 carrier 字段被 jwt 库忽略，向后兼容。
type jwtClaims struct {
	UserID         uuid.UUID  `json:"user_id"`
	Username       string     `json:"username"`
	IsSuperAdmin   bool       `json:"is_super_admin,omitempty"`
	Roles          []string   `json:"roles"`
	CurrentRoleID  *uuid.UUID `json:"current_role_id,omitempty"`
	IssuedAtMicros int64      `json:"iat_us,omitempty"`
	Scopes         []string   `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

const minJWTSecretLength = 32
const agentDelegationSubject = "agent_delegation"
const agentRuntimeScope = "agent-runtime"

var agentDelegationTTL = 5 * time.Minute
var backgroundAgentTTL = 60 * time.Second

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
	issuedAtMicros := now.UnixMicro()

	accessClaims := &jwtClaims{
		UserID:         claims.UserID,
		Username:       claims.Username,
		IsSuperAdmin:   claims.IsSuperAdmin,
		Roles:          claims.Roles,
		CurrentRoleID:  claims.CurrentRoleID,
		IssuedAtMicros: issuedAtMicros,
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
		UserID:         claims.UserID,
		Username:       claims.Username,
		IsSuperAdmin:   claims.IsSuperAdmin,
		Roles:          claims.Roles,
		IssuedAtMicros: issuedAtMicros,
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

// GenerateAgentDelegationToken creates a short-lived token for agent runtime callbacks.
// It intentionally uses a separate subject and scope from normal web access tokens.
func (s *JWTService) GenerateAgentDelegationToken(claims *Claims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(agentDelegationTTL)
	delegationClaims := &jwtClaims{
		UserID:         claims.UserID,
		Username:       claims.Username,
		IsSuperAdmin:   claims.IsSuperAdmin,
		Roles:          claims.Roles,
		CurrentRoleID:  claims.CurrentRoleID,
		IssuedAtMicros: now.UnixMicro(),
		Scopes:         []string{agentRuntimeScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   agentDelegationSubject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, delegationClaims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign agent delegation token: %w", err)
	}
	return tokenString, expiresAt, nil
}

// GenerateBackgroundAgentToken signs a one-minute access token for a single
// proactive run. The caller must still enforce the scenario operation allowlist
// and resource scope before minting it; scopes make that boundary auditable in
// downstream auth and operation logs.
func (s *JWTService) GenerateBackgroundAgentToken(claims *Claims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(backgroundAgentTTL)
	accessClaims := &jwtClaims{
		UserID:         claims.UserID,
		Username:       claims.Username,
		IsSuperAdmin:   claims.IsSuperAdmin,
		Roles:          claims.Roles,
		CurrentRoleID:  claims.CurrentRoleID,
		IssuedAtMicros: now.UnixMicro(),
		Scopes:         append([]string(nil), claims.Scopes...),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign background agent token: %w", err)
	}
	return tokenString, expiresAt, nil
}

// ValidateAgentDelegationToken validates the short-lived agent runtime token.
func (s *JWTService) ValidateAgentDelegationToken(tokenString string) (*Claims, error) {
	claims, err := s.validateToken(tokenString, agentDelegationSubject)
	if err != nil {
		return nil, err
	}
	for _, scope := range claims.Scopes {
		if scope == agentRuntimeScope {
			return claims, nil
		}
	}
	return nil, fmt.Errorf("missing %s scope", agentRuntimeScope)
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

	var issuedAt int64
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Unix()
	}
	return &Claims{
		UserID:         claims.UserID,
		Username:       claims.Username,
		IsSuperAdmin:   claims.IsSuperAdmin,
		Roles:          claims.Roles,
		CurrentRoleID:  claims.CurrentRoleID,
		IssuedAt:       issuedAt,
		IssuedAtMicros: claims.IssuedAtMicros,
		Scopes:         claims.Scopes,
	}, nil
}
