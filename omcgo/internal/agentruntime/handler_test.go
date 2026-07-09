package agentruntime

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
)

type fakeConfigProvider struct {
	target *agentconfig.RuntimeTarget
}

func (p fakeConfigProvider) GetRuntimeTarget(context.Context) (*agentconfig.RuntimeTarget, error) {
	return p.target, nil
}

type fakeConversationManager struct {
	active string
	rotate string
}

func (m fakeConversationManager) Active(context.Context, string, *admin.Claims) (string, error) {
	return m.active, nil
}

func (m fakeConversationManager) Rotate(context.Context, string, *admin.Claims) (string, error) {
	return m.rotate, nil
}

func TestChatStreamProxiesToAgentStudioWithDelegation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := strings.Repeat("s", 32)
	jwtSvc, err := admin.NewJWTService(secret)
	require.NoError(t, err)

	var capturedAuth string
	var capturedBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/action-connectors/connector-1/chat/stream", r.URL.Path)
		capturedAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		capturedBody = string(raw)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"start\",\"runId\":\"run-1\",\"conversationId\":\"c1\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"done\"}\n\n"))
	}))
	defer upstream.Close()

	r := gin.New()
	group := r.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyClaims, &admin.Claims{
			UserID:   uuid.New(),
			Username: "operator",
			Roles:    []string{"admin"},
		})
		c.Next()
	})
	NewHandler(fakeConfigProvider{target: &agentconfig.RuntimeTarget{
		Enabled:            true,
		AgentStudioBaseURL: upstream.URL,
		ConnectorID:        "connector-1",
		Status:             agentconfig.StatusConnected,
		Policy:             agentconfig.RuntimePolicy{AllowedMethods: []string{http.MethodGet}},
	}}, jwtSvc, upstream.Client(), nil, nil, nil, fakeConversationManager{active: "server-conversation"}).RegisterRoutes(group)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/chat/stream", strings.NewReader(`{"message":"hello","conversationId":"stale-browser-value"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"type":"start"`)
	require.Contains(t, w.Body.String(), `"type":"done"`)
	require.Contains(t, capturedBody, `"message":"hello"`)
	require.Contains(t, capturedBody, `"conversationId":"server-conversation"`)
	require.NotContains(t, capturedBody, "stale-browser-value")
	require.Contains(t, capturedBody, `"externalIdentity"`)
	require.True(t, strings.HasPrefix(capturedAuth, "Bearer "))
	token := strings.TrimPrefix(capturedAuth, "Bearer ")
	claims, err := jwtSvc.ValidateAgentDelegationToken(token)
	require.NoError(t, err)
	require.Equal(t, "operator", claims.Username)
}

func TestChatStreamRequiresConfiguredRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService(strings.Repeat("s", 32))
	require.NoError(t, err)

	r := gin.New()
	group := r.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyClaims, &admin.Claims{UserID: uuid.New(), Username: "operator"})
		c.Next()
	})
	NewHandler(fakeConfigProvider{target: &agentconfig.RuntimeTarget{
		Enabled: false,
		Status:  agentconfig.StatusDisabled,
	}}, jwtSvc, nil, nil, nil, nil).RegisterRoutes(group)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/chat/stream", strings.NewReader(`{"message":"hello"}`))
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestConversationRoutesReturnActiveAndRotatedConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService(strings.Repeat("s", 32))
	require.NoError(t, err)

	r := gin.New()
	group := r.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyClaims, &admin.Claims{UserID: uuid.New(), Username: "operator"})
		c.Next()
	})
	NewHandler(fakeConfigProvider{target: &agentconfig.RuntimeTarget{
		Enabled:     true,
		ConnectorID: "connector-1",
		Status:      agentconfig.StatusConnected,
	}}, jwtSvc, nil, nil, nil, nil, fakeConversationManager{
		active: "active-conversation",
		rotate: "rotated-conversation",
	}).RegisterRoutes(group)

	activeRecorder := httptest.NewRecorder()
	activeReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/conversation", nil)
	r.ServeHTTP(activeRecorder, activeReq)
	require.Equal(t, http.StatusOK, activeRecorder.Code)
	require.Contains(t, activeRecorder.Body.String(), `"conversationId":"active-conversation"`)

	rotatedRecorder := httptest.NewRecorder()
	rotatedReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/conversation", nil)
	r.ServeHTTP(rotatedRecorder, rotatedReq)
	require.Equal(t, http.StatusOK, rotatedRecorder.Code)
	require.Contains(t, rotatedRecorder.Body.String(), `"conversationId":"rotated-conversation"`)
}
