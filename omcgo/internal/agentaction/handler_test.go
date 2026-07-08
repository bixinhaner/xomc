package agentaction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
)

type fakePerm struct {
	groups []uuid.UUID
}

func (f fakePerm) GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error) {
	return f.groups, nil
}

func TestRequireDelegationSetsAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()
	token, _, err := jwtSvc.GenerateAgentDelegationToken(&admin.Claims{
		UserID:       userID,
		Username:     "agent-user",
		IsSuperAdmin: true,
		Roles:        []string{"operator"},
	})
	require.NoError(t, err)

	r := gin.New()
	r.GET("/agent-actions/actions", RequireDelegation(jwtSvc), func(c *gin.Context) {
		gotUserID, _ := c.Get(admin.CtxKeyUserID)
		gotUsername, _ := c.Get(admin.CtxKeyUsername)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  gotUserID.(uuid.UUID).String(),
			"username": gotUsername,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/agent-actions/actions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), userID.String())
	assert.Contains(t, w.Body.String(), "agent-user")
}

func TestCreateDelegationRequiresWebClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	handler := NewHandler(jwtSvc, NewService(nil, nil, zap.NewNop()), nil, zap.NewNop())

	r := gin.New()
	group := r.Group("/api/v1")
	handler.RegisterDelegationRoutes(group)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/delegation", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateDelegationIssuesScopedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()
	handler := NewHandler(jwtSvc, NewService(nil, nil, zap.NewNop()), nil, zap.NewNop())

	r := gin.New()
	group := r.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyClaims, &admin.Claims{
			UserID:   userID,
			Username: "web-user",
			Roles:    []string{"operator"},
		})
		c.Next()
	})
	handler.RegisterDelegationRoutes(group)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/delegation", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope struct {
		Data DelegationTokenResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	claims, err := jwtSvc.ValidateAgentDelegationToken(envelope.Data.Token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, []string{"agent-actions"}, claims.Scopes)
}

func TestIdentityReturnsDelegatedExternalIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()
	token, _, err := jwtSvc.GenerateAgentDelegationToken(&admin.Claims{
		UserID:       userID,
		Username:     "agent-user",
		IsSuperAdmin: true,
		Roles:        []string{"operator"},
	})
	require.NoError(t, err)
	handler := NewHandler(jwtSvc, NewService(nil, nil, zap.NewNop()), nil, zap.NewNop())

	r := gin.New()
	group := r.Group("/api/v1/agent-actions")
	group.Use(RequireDelegation(jwtSvc))
	handler.RegisterActionRoutes(group)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-actions/identity", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope struct {
		Data IdentityResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Equal(t, userID.String(), envelope.Data.ExternalUserID)
	assert.Equal(t, "agent-user", envelope.Data.ExternalUserName)
	assert.Equal(t, []string{"operator"}, envelope.Data.Roles)
	assert.Contains(t, envelope.Data.Scopes, "agent-actions")
}
