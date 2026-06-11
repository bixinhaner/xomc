package events

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/response"
)

// SSEHandler provides the HTTP handler for SSE stream connections.
type SSEHandler struct {
	hub        *MessageHub
	jwtService *admin.JWTService
	logger     *zap.Logger
}

// NewSSEHandler creates a new SSEHandler.
func NewSSEHandler(hub *MessageHub, jwtService *admin.JWTService, logger *zap.Logger) *SSEHandler {
	return &SSEHandler{
		hub:        hub,
		jwtService: jwtService,
		logger:     logger.Named("sse-handler"),
	}
}

// RegisterRoutes registers SSE routes under the given router group.
// The stream endpoint is registered at GET /events/stream.
func (h *SSEHandler) RegisterRoutes(rg *gin.RouterGroup) {
	events := rg.Group("/events")
	events.GET("/stream", h.Stream)
}

// Stream handles the SSE connection lifecycle:
//  1. Authenticate via JWT token (query param or Authorization header)
//  2. Subscribe to the user's message channel
//  3. Set SSE headers
//  4. Immediately commit the 200 header + flush a `:connected` comment frame
//     so EventSource onopen fires without waiting for the first event/keepalive
//  5. Replay missed messages if Last-Event-ID is present
//  6. Stream events with 30s keepalive (unsubscribe on disconnect)
func (h *SSEHandler) Stream(c *gin.Context) {
	// Step 1: Authenticate
	username, err := h.authenticate(c)
	if err != nil {
		h.logger.Warn("SSE authentication failed", zap.Error(err))
		response.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Step 2: Subscribe
	ch, err := h.hub.Subscribe(username)
	if err != nil {
		h.logger.Error("SSE subscribe failed",
			zap.String("user_id", username),
			zap.Error(err),
		)
		response.Fail(c, http.StatusInternalServerError, "subscribe failed")
		return
	}
	defer h.hub.Unsubscribe(username)

	// Step 3: Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Assert http.Flusher for streaming
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("streaming not supported")
		response.Fail(c, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// 解除 http.Server.WriteTimeout（在 infra.go::ListenAndServe 设为 30s）对本
	// SSE 长连接的限制。WriteTimeout 是整个 ResponseWriter 写操作的硬截止时间，
	// 任何 streaming 响应超过 30s 就被强制关连接（curl 实测：返 Content-Length:0
	// + Connection:close）。其它 REST API 仍保留 30s 防慢客户端攻击；这里仅
	// 针对 SSE endpoint 通过 ResponseController（Go 1.20+）清掉 deadline。
	if rc := http.NewResponseController(c.Writer); rc != nil {
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			h.logger.Warn("clear SSE write deadline failed",
				zap.String("user_id", username),
				zap.Error(err),
			)
		}
	}

	// Step 4: 立即提交 200 响应头并发出 `:connected` 注释帧后 flush。
	// 否则在首个业务事件 / 30s keepalive 到达前，客户端收不到任何字节：
	//   - 浏览器 EventSource 的 onopen 最长延迟 30s；
	//   - 中间代理 / LB 可能按首字节超时切断这条 SSE 长连接。
	// SSE 注释帧（以 ':' 开头）会被规范的 EventSource 实现忽略，仅用于探活/保活，
	// 既能立刻冲刷响应头，又不会被当作一条事件投递给业务回调。
	c.Status(http.StatusOK)
	fmt.Fprint(c.Writer, ":connected\n\n")
	flusher.Flush()

	// Step 5: Replay missed messages if Last-Event-ID present
	if lastEventID := c.GetHeader("Last-Event-ID"); lastEventID != "" {
		h.replayMessages(c, username, lastEventID, flusher)
	}

	// Step 6: Stream loop with keepalive
	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				// Channel was closed (kicked by new connection)
				return
			}
			h.writeSSEEvent(c, msg, flusher)

		case <-keepalive.C:
			// Write SSE comment as keepalive
			fmt.Fprintf(c.Writer, ":keepalive\n\n")
			flusher.Flush()

		case <-ctx.Done():
			// Client disconnected
			return
		}
	}
}

// authenticate extracts and validates the JWT token from either
// the `token` query parameter or the `Authorization: Bearer` header.
func (h *SSEHandler) authenticate(c *gin.Context) (string, error) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		return "", fmt.Errorf("no token provided")
	}

	claims, err := h.jwtService.ValidateAccessToken(tokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if claims.Username == "" {
		return "", fmt.Errorf("token has no username claim")
	}

	return claims.Username, nil
}

// replayMessages replays missed SSE messages after the given Last-Event-ID.
func (h *SSEHandler) replayMessages(c *gin.Context, userID string, afterID string, flusher http.Flusher) {
	if h.hub.store == nil {
		return
	}

	ctx := c.Request.Context()
	messages, err := h.hub.store.GetSince(ctx, userID, afterID, 50)
	if err != nil {
		h.logger.Warn("failed to replay SSE messages",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return
	}

	for _, msg := range messages {
		h.writeSSEEvent(c, msg, flusher)
	}
}

// writeSSEEvent writes a single SSE event to the response writer.
// Format: `id: <id>\nevent: <event>\ndata: <data>\n\n`
func (h *SSEHandler) writeSSEEvent(c *gin.Context, msg *SSEMessage, flusher http.Flusher) {
	if msg.ID != "" {
		fmt.Fprintf(c.Writer, "id: %s\n", msg.ID)
	}
	if msg.Event != "" {
		fmt.Fprintf(c.Writer, "event: %s\n", msg.Event)
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", msg.Data)
	flusher.Flush()
}

// parseTokenSimple is a fallback JWT parser that extracts the username claim
// without requiring the full admin.JWTService. Used only when jwtService is nil.
func parseTokenSimple(tokenStr string, secret []byte) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	username, _ := claims["username"].(string)
	return username, nil
}
