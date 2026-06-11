package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
)

const testJWTSecret = "test-secret-minimum-32-characters!!"

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestJWTService(t *testing.T) *admin.JWTService {
	t.Helper()
	svc, err := admin.NewJWTService(testJWTSecret)
	require.NoError(t, err)
	return svc
}

func mintAccessToken(t *testing.T, svc *admin.JWTService, username string) string {
	t.Helper()
	pair, err := svc.GenerateTokenPair(&admin.Claims{
		UserID:   uuid.New(),
		Username: username,
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)
	return pair.AccessToken
}

func newHandlerForTest(t *testing.T) (*SSEHandler, *MessageHub, *fakeStore, *admin.JWTService) {
	t.Helper()
	store := newFakeStore()
	hub := NewMessageHub(store, zap.NewNop())
	jwt := newTestJWTService(t)
	h := NewSSEHandler(hub, jwt, zap.NewNop())
	return h, hub, store, jwt
}

func TestSSEHandler_RegisterRoutes(t *testing.T) {
	h, _, _, _ := newHandlerForTest(t)
	r := gin.New()
	g := r.Group("/api")
	h.RegisterRoutes(g)

	// Verify route registered.
	routes := r.Routes()
	found := false
	for _, ri := range routes {
		if ri.Method == http.MethodGet && ri.Path == "/api/events/stream" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected /api/events/stream to be registered")
}

func TestSSEHandler_Stream_NoToken_Returns401(t *testing.T) {
	h, _, _, _ := newHandlerForTest(t)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"))

	req := httptest.NewRequest(http.MethodGet, "/events/stream", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSSEHandler_Stream_InvalidToken_Returns401(t *testing.T) {
	h, _, _, _ := newHandlerForTest(t)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"))

	req := httptest.NewRequest(http.MethodGet, "/events/stream?token=garbage", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSSEHandler_Stream_AuthorizationHeader_Path(t *testing.T) {
	h, _, _, jwt := newHandlerForTest(t)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"))

	token := mintAccessToken(t, jwt, "alice")

	// Use cancellable context so the streaming loop exits quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events/stream", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Either 200 (stream opened then ctx done) is acceptable.
	assert.Equal(t, http.StatusOK, w.Code)
	// SSE headers must have been set.
	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

func TestSSEHandler_Stream_FlushesConnectedFrameImmediately(t *testing.T) {
	h, _, _, jwt := newHandlerForTest(t)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"))

	token := mintAccessToken(t, jwt, "frank")

	// Cancel well before the 30s keepalive and without publishing any message:
	// the `:connected` comment frame must already be in the body, proving the
	// header + first byte are flushed immediately on subscribe (issue #126.9).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events/stream?token="+token, nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
	assert.Contains(t, w.Body.String(), ":connected\n\n", "expected immediate :connected SSE comment frame")
	assert.True(t, w.Flushed, "expected response writer to be flushed before first event")
}

func TestSSEHandler_Stream_DeliversPublishedMessage(t *testing.T) {
	h, hub, _, jwt := newHandlerForTest(t)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"))

	token := mintAccessToken(t, jwt, "bob")

	// Publish a message slightly after handler subscribes.
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = hub.Publish("bob", &SSEMessage{
			ID:    "evt-1",
			Event: "alarm",
			Data:  json.RawMessage(`{"x":1}`),
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events/stream?token="+token, nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, "id: evt-1")
	assert.Contains(t, body, "event: alarm")
	assert.Contains(t, body, `data: {"x":1}`)
}

func TestSSEHandler_Stream_LastEventID_TriggersReplay(t *testing.T) {
	h, _, store, jwt := newHandlerForTest(t)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"))

	// Pre-seed history so replay path is exercised.
	require.NoError(t, store.Store(context.Background(), "carol", &SSEMessage{ID: "old-1", Event: "x", Data: json.RawMessage(`{}`)}))
	require.NoError(t, store.Store(context.Background(), "carol", &SSEMessage{ID: "old-2", Event: "x", Data: json.RawMessage(`{}`)}))

	token := mintAccessToken(t, jwt, "carol")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events/stream?token="+token, nil).WithContext(ctx)
	req.Header.Set("Last-Event-ID", "old-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, "id: old-2", "expected replay of message after Last-Event-ID")
}

func TestParseTokenSimple_Valid(t *testing.T) {
	secret := []byte("simple-secret")
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "dave",
	})
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)

	user, err := parseTokenSimple(signed, secret)
	require.NoError(t, err)
	assert.Equal(t, "dave", user)
}

func TestParseTokenSimple_WrongSigningMethod(t *testing.T) {
	// Build a token with "none" alg → unexpected method.
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"username":"e"}`
	bogus := encodeSegment(header) + "." + encodeSegment(payload) + "."
	_, err := parseTokenSimple(bogus, []byte("k"))
	require.Error(t, err)
}

func TestParseTokenSimple_Garbage(t *testing.T) {
	_, err := parseTokenSimple("not-a-jwt", []byte("k"))
	require.Error(t, err)
}

// encodeSegment provides minimal base64-url encoding for test fixtures.
func encodeSegment(s string) string {
	const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	out := strings.Builder{}
	bytes := []byte(s)
	for i := 0; i < len(bytes); i += 3 {
		var n int
		end := i + 3
		if end > len(bytes) {
			end = len(bytes)
		}
		switch end - i {
		case 3:
			n = int(bytes[i])<<16 | int(bytes[i+1])<<8 | int(bytes[i+2])
			out.WriteByte(enc[(n>>18)&0x3F])
			out.WriteByte(enc[(n>>12)&0x3F])
			out.WriteByte(enc[(n>>6)&0x3F])
			out.WriteByte(enc[n&0x3F])
		case 2:
			n = int(bytes[i])<<16 | int(bytes[i+1])<<8
			out.WriteByte(enc[(n>>18)&0x3F])
			out.WriteByte(enc[(n>>12)&0x3F])
			out.WriteByte(enc[(n>>6)&0x3F])
		case 1:
			n = int(bytes[i]) << 16
			out.WriteByte(enc[(n>>18)&0x3F])
			out.WriteByte(enc[(n>>12)&0x3F])
		}
	}
	return out.String()
}
