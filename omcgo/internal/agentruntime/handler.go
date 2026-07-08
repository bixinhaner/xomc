package agentruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type ConfigProvider interface {
	GetRuntimeTarget(ctx context.Context) (*agentconfig.RuntimeTarget, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Handler struct {
	config ConfigProvider
	jwt    *admin.JWTService
	http   HTTPDoer
	logger *zap.Logger
	tools  *ToolExecutor
}

func NewHandler(config ConfigProvider, jwtService *admin.JWTService, httpClient HTTPDoer, logger *zap.Logger, localHandler http.Handler, routes RouteProvider) *Handler {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		config: config,
		jwt:    jwtService,
		http:   httpClient,
		logger: logger.Named("agentruntime"),
		tools:  NewToolExecutor(localHandler, routes, jwtService),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/agent/chat/stream", h.ChatStream)
}

func (h *Handler) ChatStream(c *gin.Context) {
	if h.jwt == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
		return
	}
	claims, ok := currentClaims(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}
	target, err := h.config.GetRuntimeTarget(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if target == nil || !target.Enabled {
		detail := "agent runtime is not configured"
		if target != nil && target.LastError != "" {
			detail = target.LastError
		}
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, fmt.Errorf("%s", detail))
		return
	}

	rawBody, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("read agent chat body: %w", err))
		return
	}
	rawBody = injectExternalIdentity(rawBody, claims)
	delegationToken, _, err := h.jwt.GenerateAgentDelegationToken(claims)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	upstreamURL := buildActionConnectorStreamURL(target.AgentStudioBaseURL, target.ConnectorID)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(rawBody))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("build agent runtime request: %w", err))
		return
	}
	req.Header.Set("Authorization", "Bearer "+delegationToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-OMC-User", claims.Username)

	resp, err := h.http.Do(req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("agent runtime unreachable: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		message := strings.TrimSpace(string(detail))
		if message == "" {
			message = fmt.Sprintf("agent runtime failed with HTTP %d", resp.StatusCode)
		}
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("%s", message))
		return
	}

	copyHeader(c.Writer.Header(), resp.Header, "Content-Type")
	if c.Writer.Header().Get("Content-Type") == "" {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
	}
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	h.streamAgentStudioResponse(c.Request.Context(), c.Writer, resp.Body, claims, delegationToken, target)
}

func currentClaims(c *gin.Context) (*admin.Claims, bool) {
	value, exists := c.Get(admin.CtxKeyClaims)
	claims, ok := value.(*admin.Claims)
	return claims, exists && ok && claims != nil
}

func buildActionConnectorStreamURL(baseURL, connectorID string) string {
	return strings.TrimRight(baseURL, "/") + "/api/action-connectors/" + url.PathEscape(connectorID) + "/chat/stream"
}

func buildToolResultURL(baseURL, connectorID string) string {
	return strings.TrimRight(baseURL, "/") + "/api/action-connectors/" + url.PathEscape(connectorID) + "/tool-results"
}

func copyHeader(dst http.Header, src http.Header, key string) {
	value := src.Get(key)
	if value == "" {
		return
	}
	dst.Set(key, value)
}

type sseFrame struct {
	raw       string
	eventName string
	data      string
}

func injectExternalIdentity(raw []byte, claims *admin.Claims) []byte {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw
	}
	contextValue, _ := payload["context"].(map[string]any)
	if contextValue == nil {
		contextValue = map[string]any{}
	}
	contextValue["externalIdentity"] = map[string]any{
		"externalUserId":   claims.UserID.String(),
		"externalUserName": claims.Username,
		"username":         claims.Username,
		"roles":            claims.Roles,
		"scopes":           claims.Scopes,
	}
	payload["context"] = contextValue
	next, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return next
}

func parseSSEFrame(raw string) sseFrame {
	frame := sseFrame{raw: raw}
	lines := strings.Split(raw, "\n")
	dataLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, "event:") {
			frame.eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	frame.data = strings.Join(dataLines, "\n")
	return frame
}

func toolRequestFromFrame(frame sseFrame) (ToolRequest, bool) {
	if frame.data == "" || frame.data == "[DONE]" {
		return ToolRequest{}, false
	}
	var envelope struct {
		Type       string          `json:"type"`
		RunID      string          `json:"runId"`
		ToolCallID string          `json:"toolCallId"`
		Tool       string          `json:"tool"`
		Title      string          `json:"title"`
		Input      ToolRequestBody `json:"input"`
	}
	if err := json.Unmarshal([]byte(frame.data), &envelope); err != nil {
		return ToolRequest{}, false
	}
	if envelope.Type != "tool_request" {
		return ToolRequest{}, false
	}
	return ToolRequest{
		RunID:      envelope.RunID,
		ToolCallID: envelope.ToolCallID,
		Tool:       envelope.Tool,
		Title:      envelope.Title,
		Input:      envelope.Input,
	}, true
}

func (h *Handler) streamAgentStudioResponse(ctx context.Context, w gin.ResponseWriter, body io.Reader, claims *admin.Claims, delegationToken string, target *agentconfig.RuntimeTarget) {
	flusher, _ := w.(http.Flusher)
	reader := bufio.NewReader(body)
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if strings.TrimRight(line, "\r\n") == "" {
				rawFrame := frame.String()
				frame.Reset()
				if !h.handleSSEFrame(ctx, w, rawFrame, claims, delegationToken, target, flusher) {
					return
				}
				if flusher != nil {
					flusher.Flush()
				}
			} else {
				frame.WriteString(line)
			}
		}
		if err == io.EOF {
			if frame.Len() > 0 {
				_ = h.handleSSEFrame(ctx, w, frame.String(), claims, delegationToken, target, flusher)
			}
			return
		}
		if err != nil {
			h.logger.Warn("agent stream upstream read failed", zap.Error(err))
			return
		}
	}
}

func (h *Handler) handleSSEFrame(ctx context.Context, w gin.ResponseWriter, raw string, claims *admin.Claims, delegationToken string, target *agentconfig.RuntimeTarget, flusher http.Flusher) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	if _, err := w.Write([]byte(raw + "\n\n")); err != nil {
		h.logger.Debug("agent stream client write failed", zap.Error(err))
		return false
	}
	if flusher != nil {
		flusher.Flush()
	}
	frame := parseSSEFrame(raw)
	request, ok := toolRequestFromFrame(frame)
	if !ok {
		return true
	}
	result := h.tools.Execute(ctx, claims, request, target.Policy)
	if err := h.postToolResult(ctx, target, delegationToken, result); err != nil {
		h.logger.Warn("agent tool result callback failed", zap.Error(err))
	}
	return true
}

func (h *Handler) postToolResult(ctx context.Context, target *agentconfig.RuntimeTarget, delegationToken string, result ToolResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal tool result: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildToolResultURL(target.AgentStudioBaseURL, target.ConnectorID), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build tool result request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+delegationToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := h.http.Do(req)
	if err != nil {
		return fmt.Errorf("post tool result: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("tool result callback HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}
