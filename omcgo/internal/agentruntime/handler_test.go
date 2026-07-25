package agentruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
	"github.com/omcgo/omcgo/internal/agentruntime/handbookgen"
)

type fakeConfigProvider struct {
	target *agentconfig.RuntimeTarget
}

func (p fakeConfigProvider) GetRuntimeTarget(context.Context) (*agentconfig.RuntimeTarget, error) {
	return p.target, nil
}

type fakeConversationManager struct {
	active   string
	rotate   string
	instance string
}

type deadlineTrackingWriter struct {
	header   http.Header
	deadline time.Time
}

func (w *deadlineTrackingWriter) Header() http.Header {
	return w.header
}

func (w *deadlineTrackingWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func (w *deadlineTrackingWriter) WriteHeader(int) {}

func (w *deadlineTrackingWriter) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	return nil
}

func (m fakeConversationManager) Active(context.Context, string, *admin.Claims) (string, error) {
	return m.active, nil
}

func (m fakeConversationManager) Rotate(context.Context, string, *admin.Claims) (string, error) {
	return m.rotate, nil
}

func (m fakeConversationManager) InstanceID(context.Context) (string, error) {
	return m.instance, nil
}

func TestClearAgentStreamWriteDeadline(t *testing.T) {
	writer := &deadlineTrackingWriter{
		header:   make(http.Header),
		deadline: time.Now(),
	}

	require.NoError(t, clearAgentStreamWriteDeadline(writer))
	require.True(t, writer.deadline.IsZero())
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
	handler := NewHandler(fakeConfigProvider{target: &agentconfig.RuntimeTarget{
		Enabled:            true,
		AgentStudioBaseURL: upstream.URL,
		ConnectorSlug:      "external-agent-connector",
		ConnectorID:        "connector-1",
		Status:             agentconfig.StatusConnected,
		InstanceName:       "OMC 陕西",
		Policy:             agentconfig.RuntimePolicy{AllowedMethods: []string{http.MethodGet}},
	}}, jwtSvc, upstream.Client(), nil, r, r, fakeConversationManager{
		active:   "server-conversation",
		instance: "omcinst_1234567890abcdef",
	})
	handler.RegisterRoutes(group)
	handler.tools.handbook = testHandbookPackage(t, handler.tools.HandbookRouteExport())
	handler.tools.handbookErr = nil

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
	var capturedPayload map[string]any
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &capturedPayload))
	contextValue, ok := capturedPayload["context"].(map[string]any)
	require.True(t, ok)
	identity, ok := contextValue["externalIdentity"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "operator", identity["externalUserName"])
	metadata, ok := identity["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "omc", metadata["sourceSystem"])
	require.Equal(t, "OMC 陕西", metadata["instanceName"])
	require.Equal(t, "omcinst_1234567890abcdef", metadata["instanceId"])
	require.Equal(t, "12345678", metadata["instanceShortId"])
	require.Equal(t, "external-agent-connector", metadata["connectorSlug"])
	require.Equal(t, "connector-1", metadata["connectorId"])
	require.Equal(t, "operator", metadata["userDisplayName"])
	handbook, ok := metadata["apiHandbook"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, handbookSchemaVersion, handbook["schemaVersion"])
	require.NotEmpty(t, handbook["catalogVersion"])
	require.EqualValues(t, 11, handbook["totalOperations"])
	require.Equal(t, true, handbook["packageAvailable"])
	require.NotEmpty(t, handbook["handbookDigest"])
	require.Equal(t, "/api/v1/agent/handbook/manifest", handbook["manifestPath"])
	require.Equal(t, "/api/v1/agent/handbook/chunks/{index}", handbook["chunkPathTemplate"])
	require.True(t, strings.HasPrefix(capturedAuth, "Bearer "))
	token := strings.TrimPrefix(capturedAuth, "Bearer ")
	claims, err := jwtSvc.ValidateAgentDelegationToken(token)
	require.NoError(t, err)
	require.Equal(t, "operator", claims.Username)
}

func TestUploadAttachmentUsesServiceTokenAndActiveConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	var received []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/integrations/action-connectors/connector-1/conversations/conversation-1/attachments", r.URL.Path)
		require.Equal(t, "Bearer service-secret", r.Header.Get("Authorization"))
		require.Equal(t, userID.String(), r.Header.Get("X-External-User-ID"))
		require.Equal(t, "report%20%E6%8A%A5%E5%91%8A.txt", r.Header.Get("X-File-Name"))
		received, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"attachment":{"attachmentId":"a1","filename":"report 报告.txt","mimeType":"text/plain","sizeBytes":5}}`))
	}))
	defer upstream.Close()

	r := gin.New()
	group := r.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyClaims, &admin.Claims{UserID: userID, Username: "operator"})
		c.Next()
	})
	NewHandler(fakeConfigProvider{target: &agentconfig.RuntimeTarget{
		Enabled:                 true,
		AgentStudioBaseURL:      upstream.URL,
		AgentStudioServiceToken: "service-secret",
		ConnectorID:             "connector-1",
		Status:                  agentconfig.StatusConnected,
	}}, nil, upstream.Client(), nil, nil, nil, fakeConversationManager{active: "conversation-1"}).RegisterRoutes(group)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "report 报告.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/attachments", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, []byte("hello"), received)
	require.Contains(t, recorder.Body.String(), `"attachmentId":"a1"`)
}

func TestChatStreamTransfersToolDownloadAndPostsFileReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := strings.Repeat("s", 32)
	jwtSvc, err := admin.NewJWTService(secret)
	require.NoError(t, err)
	userID := uuid.New()
	fileContent := []byte("site,status\nsite-1,online\n")
	fileDigest := fmt.Sprintf("%x", sha256.Sum256(fileContent))
	uploaded := make(chan []byte, 1)
	results := make(chan ToolResult, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/action-connectors/connector-1/chat/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(`data: {"type":"tool_request","runId":"run-1","toolCallId":"tool-1","tool":"rest.request","input":{"method":"GET","path":"/api/v1/exports/export-1"}}` + "\n\n"))
			_, _ = w.Write([]byte(`data: {"type":"done"}` + "\n\n"))
		case "/api/integrations/action-connectors/connector-1/conversations/conversation-1/attachments":
			require.Equal(t, "Bearer service-secret", r.Header.Get("Authorization"))
			require.Equal(t, userID.String(), r.Header.Get("X-External-User-ID"))
			require.Equal(t, "status%20report.csv", r.Header.Get("X-File-Name"))
			body, readErr := io.ReadAll(r.Body)
			require.NoError(t, readErr)
			uploaded <- body
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"attachment":{"attachmentId":"0123456789abcdef0123456789abcdef","filename":"status report.csv","mimeType":"text/csv","sizeBytes":%d,"sha256":"%s","createdAt":"2026-07-23T00:00:00Z"}}`, len(fileContent), fileDigest)
		case "/api/action-connectors/connector-1/tool-results":
			require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))
			var result ToolResult
			require.NoError(t, json.NewDecoder(r.Body).Decode(&result))
			results <- result
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	r := gin.New()
	r.GET("/api/v1/exports/:id", func(c *gin.Context) {
		c.Header("Content-Disposition", `attachment; filename="status report.csv"`)
		c.Data(http.StatusOK, "text/csv", fileContent)
	})
	group := r.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyClaims, &admin.Claims{UserID: userID, Username: "operator"})
		c.Next()
	})
	handler := NewHandler(fakeConfigProvider{target: &agentconfig.RuntimeTarget{
		Enabled:                 true,
		AgentStudioBaseURL:      upstream.URL,
		AgentStudioServiceToken: "service-secret",
		ConnectorID:             "connector-1",
		Status:                  agentconfig.StatusConnected,
		Policy: agentconfig.RuntimePolicy{
			AllowedMethods: []string{http.MethodGet}, ToolTimeoutSeconds: 30, MaxResponseBytes: 262144,
		},
	}}, jwtSvc, upstream.Client(), nil, r, r, fakeConversationManager{active: "conversation-1"})
	handler.RegisterRoutes(group)
	handler.tools.handbook = &handbookPackage{operations: map[string]handbookgen.OperationDocument{
		"get.exports.by.id": {
			OperationID:      "get.exports.by.id",
			Method:           http.MethodGet,
			Path:             "/api/v1/exports/:id",
			ContractCoverage: map[string]string{"request": "path-only"},
		},
	}}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/chat/stream", strings.NewReader(`{"message":"download the report"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	var result ToolResult
	select {
	case result = <-results:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tool result callback")
	}
	require.Equal(t, "ok", result.Status, "tool result error: %#v", result.Error)
	var uploadedContent []byte
	select {
	case uploadedContent = <-uploaded:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tool result file upload")
	}
	require.Equal(t, fileContent, uploadedContent)
	require.Len(t, result.Files, 1)
	require.Equal(t, "0123456789abcdef0123456789abcdef", result.Files[0].AttachmentID)
	require.Equal(t, "status report.csv", result.Files[0].Filename)
	require.Equal(t, fileDigest, result.Files[0].SHA256)
	require.Equal(t, "file", result.Output.(map[string]any)["type"])
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
