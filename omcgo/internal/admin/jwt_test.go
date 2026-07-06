package admin

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateTokenPair(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	claims := &Claims{
		UserID:       uuid.New(),
		Username:     "testuser",
		IsSuperAdmin: false,
		Roles:        []string{"admin", "operator"},
	}

	pair, err := svc.GenerateTokenPair(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.True(t, pair.ExpiresAt.After(time.Now()))
}

func TestJWTService_ValidateAccessToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	original := &Claims{
		UserID:       userID,
		Username:     "testuser",
		IsSuperAdmin: false,
		Roles:        []string{"viewer"},
	}

	pair, err := svc.GenerateTokenPair(original)
	require.NoError(t, err)

	parsed, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, parsed.UserID)
	assert.Equal(t, "testuser", parsed.Username)
	assert.False(t, parsed.IsSuperAdmin)
	assert.Equal(t, []string{"viewer"}, parsed.Roles)
}

func TestJWTService_ValidateRefreshToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	original := &Claims{
		UserID:   userID,
		Username: "testuser",
		Roles:    []string{"admin"},
	}

	pair, err := svc.GenerateTokenPair(original)
	require.NoError(t, err)

	parsed, err := svc.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, parsed.UserID)
	assert.Equal(t, "testuser", parsed.Username)
}

func TestJWTService_AccessTokenRejectsRefreshToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	pair, err := svc.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "testuser",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(pair.RefreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}

func TestJWTService_RefreshTokenRejectsAccessToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	pair, err := svc.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "testuser",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	_, err = svc.ValidateRefreshToken(pair.AccessToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}

func TestJWTService_InvalidSecret(t *testing.T) {
	svc1, err := NewJWTService("secret-one-minimum-32-characters!!")
	require.NoError(t, err)
	svc2, err := NewJWTService("secret-two-minimum-32-characters!!")
	require.NoError(t, err)

	pair, err := svc1.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "testuser",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	_, err = svc2.ValidateAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTService_ExpiredToken(t *testing.T) {
	// Directly set negative TTL to produce an already-expired token.
	svc := &JWTService{
		secret:          []byte("test-secret-minimum-32-characters!!"),
		accessTokenTTL:  -1 * time.Hour,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}

	pair, err := svc.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "testuser",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTService_InvalidTokenString(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken("invalid-token-string")
	assert.Error(t, err)

	_, err = svc.ValidateRefreshToken("")
	assert.Error(t, err)
}

func TestJWTService_SuperAdminClaim(t *testing.T) {
	// v1.0：替换原 TestJWTService_NilCarrier，验证 IsSuperAdmin claim 签发与解析。
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	pair, err := svc.GenerateTokenPair(&Claims{
		UserID:       uuid.New(),
		Username:     "superadmin",
		IsSuperAdmin: true,
		Roles:        []string{"admin"},
	})
	require.NoError(t, err)

	parsed, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.True(t, parsed.IsSuperAdmin)
}

func TestJWTService_AgentDelegationToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	token, expiresAt, err := svc.GenerateAgentDelegationToken(&Claims{
		UserID:       userID,
		Username:     "operator",
		IsSuperAdmin: false,
		Roles:        []string{"operator"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(6*time.Minute)))

	parsed, err := svc.ValidateAgentDelegationToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, parsed.UserID)
	assert.Equal(t, "operator", parsed.Username)
	assert.Equal(t, []string{"operator"}, parsed.Roles)
	assert.Equal(t, []string{"agent-actions"}, parsed.Scopes)
}

func TestJWTService_AgentDelegationRejectsAccessToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	pair, err := svc.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "operator",
		Roles:    []string{"operator"},
	})
	require.NoError(t, err)

	_, err = svc.ValidateAgentDelegationToken(pair.AccessToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}

func TestJWTService_AccessTokenRejectsAgentDelegationToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	token, _, err := svc.GenerateAgentDelegationToken(&Claims{
		UserID:   uuid.New(),
		Username: "operator",
		Roles:    []string{"operator"},
	})
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}
