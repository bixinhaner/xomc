package admin

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateTokenPair(t *testing.T) {
	svc := NewJWTService("test-secret-minimum-32-characters!!")
	carrier := model.CarrierCMCC

	claims := &Claims{
		UserID:   uuid.New(),
		Username: "testuser",
		Carrier:  &carrier,
		Roles:    []string{"admin", "operator"},
	}

	pair, err := svc.GenerateTokenPair(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.True(t, pair.ExpiresAt.After(time.Now()))
}

func TestJWTService_ValidateAccessToken(t *testing.T) {
	svc := NewJWTService("test-secret-minimum-32-characters!!")
	userID := uuid.New()
	carrier := model.CarrierCTCC

	original := &Claims{
		UserID:   userID,
		Username: "testuser",
		Carrier:  &carrier,
		Roles:    []string{"viewer"},
	}

	pair, err := svc.GenerateTokenPair(original)
	require.NoError(t, err)

	parsed, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, parsed.UserID)
	assert.Equal(t, "testuser", parsed.Username)
	assert.NotNil(t, parsed.Carrier)
	assert.Equal(t, model.CarrierCTCC, *parsed.Carrier)
	assert.Equal(t, []string{"viewer"}, parsed.Roles)
}

func TestJWTService_ValidateRefreshToken(t *testing.T) {
	svc := NewJWTService("test-secret-minimum-32-characters!!")
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
	assert.Nil(t, parsed.Carrier)
}

func TestJWTService_AccessTokenRejectsRefreshToken(t *testing.T) {
	svc := NewJWTService("test-secret-minimum-32-characters!!")

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
	svc := NewJWTService("test-secret-minimum-32-characters!!")

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
	svc1 := NewJWTService("secret-one-minimum-32-characters!!")
	svc2 := NewJWTService("secret-two-minimum-32-characters!!")

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
	svc := NewJWTService("test-secret-minimum-32-characters!!")

	_, err := svc.ValidateAccessToken("invalid-token-string")
	assert.Error(t, err)

	_, err = svc.ValidateRefreshToken("")
	assert.Error(t, err)
}

func TestJWTService_NilCarrier(t *testing.T) {
	svc := NewJWTService("test-secret-minimum-32-characters!!")

	pair, err := svc.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "superadmin",
		Carrier:  nil,
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	parsed, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Nil(t, parsed.Carrier)
}
