package agentruntime

import (
	"bytes"
	"context"
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
}

func NewHandler(config ConfigProvider, jwtService *admin.JWTService, httpClient HTTPDoer, logger *zap.Logger) *Handler {
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
	streamBody(c.Writer, resp.Body, h.logger)
}

func currentClaims(c *gin.Context) (*admin.Claims, bool) {
	value, exists := c.Get(admin.CtxKeyClaims)
	claims, ok := value.(*admin.Claims)
	return claims, exists && ok && claims != nil
}

func buildActionConnectorStreamURL(baseURL, connectorID string) string {
	return strings.TrimRight(baseURL, "/") + "/api/action-connectors/" + url.PathEscape(connectorID) + "/chat/stream"
}

func copyHeader(dst http.Header, src http.Header, key string) {
	value := src.Get(key)
	if value == "" {
		return
	}
	dst.Set(key, value)
}

func streamBody(w gin.ResponseWriter, body io.Reader, logger *zap.Logger) {
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				logger.Debug("agent stream client write failed", zap.Error(writeErr))
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			logger.Warn("agent stream upstream read failed", zap.Error(err))
			return
		}
	}
}
