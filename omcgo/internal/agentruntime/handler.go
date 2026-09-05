package agentruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type ConfigProvider interface {
	GetRuntimeTarget(ctx context.Context) (*agentconfig.RuntimeTarget, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ConversationManager interface {
	Active(ctx context.Context, connectorID string, claims *admin.Claims) (string, error)
	Rotate(ctx context.Context, connectorID string, claims *admin.Claims) (string, error)
	InstanceID(ctx context.Context) (string, error)
}

type Handler struct {
	config        ConfigProvider
	jwt           *admin.JWTService
	http          HTTPDoer
	logger        *zap.Logger
	tools         *ToolExecutor
	conversations ConversationManager
}

func NewHandler(config ConfigProvider, jwtService *admin.JWTService, httpClient HTTPDoer, logger *zap.Logger, localHandler http.Handler, routes RouteProvider, conversations ...ConversationManager) *Handler {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	var conversationManager ConversationManager
	if len(conversations) > 0 {
		conversationManager = conversations[0]
	}
	return &Handler{
		config:        config,
		jwt:           jwtService,
		http:          httpClient,
		logger:        logger.Named("agentruntime"),
		tools:         NewToolExecutor(localHandler, routes, jwtService),
		conversations: conversationManager,
	}
}

func (h *Handler) ToolExecutor() *ToolExecutor {
	if h == nil {
		return nil
	}
	return h.tools
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/agent/conversation", h.GetConversation)
	rg.POST("/agent/conversation", h.NewConversation)
	rg.GET("/agent/conversation/messages", h.GetConversationMessages)
	rg.POST("/agent/attachments", h.UploadAttachment)
	rg.DELETE("/agent/attachments/:attachmentID", h.DeleteAttachment)
	rg.POST("/agent/runs/:runID/cancel", h.CancelRun)
	rg.GET("/agent/artifacts/:artifactID/content", h.GetArtifactContent)
	rg.GET("/agent/handbook/routes", h.GetHandbookRoutes)
	rg.GET("/agent/handbook/manifest", h.GetHandbookManifest)
	rg.GET("/agent/handbook/chunks/:index", h.GetHandbookChunk)
	rg.POST("/agent/chat/stream", h.ChatStream)
}

const maxAgentAttachmentBytes = 25 << 20

type runtimeProxyContext struct {
	target         *agentconfig.RuntimeTarget
	claims         *admin.Claims
	conversationID string
}

func (h *Handler) resolveRuntimeProxyContext(c *gin.Context) (*runtimeProxyContext, bool) {
	claims, ok := currentClaims(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return nil, false
	}
	target, err := h.config.GetRuntimeTarget(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return nil, false
	}
	if target == nil || !target.Enabled || target.ConnectorID == "" || strings.TrimSpace(target.AgentStudioServiceToken) == "" || h.conversations == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
		return nil, false
	}
	conversationID, err := h.conversations.Active(c.Request.Context(), target.ConnectorID, claims)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return nil, false
	}
	return &runtimeProxyContext{target: target, claims: claims, conversationID: conversationID}, true
}

func actionConnectorServiceURL(target *agentconfig.RuntimeTarget, suffix string) string {
	return strings.TrimRight(target.AgentStudioBaseURL, "/") + "/api/integrations/action-connectors/" + url.PathEscape(target.ConnectorID) + suffix
}

func (h *Handler) newServiceRequest(c *gin.Context, proxy *runtimeProxyContext, method, suffix string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(c.Request.Context(), method, actionConnectorServiceURL(proxy.target, suffix), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+proxy.target.AgentStudioServiceToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-External-User-ID", proxy.claims.UserID.String())
	return req, nil
}

func (h *Handler) proxyJSON(c *gin.Context, req *http.Request) {
	resp, err := h.http.Do(req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("agent runtime unreachable: %w", err))
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("read agent runtime response: %w", err))
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			detail = fmt.Sprintf("agent runtime failed with HTTP %d", resp.StatusCode)
		}
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("%s", detail))
		return
	}
	if len(body) == 0 {
		response.OK(c, nil)
		return
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("decode agent runtime response: %w", err))
		return
	}
	response.OK(c, payload)
}

func (h *Handler) GetConversationMessages(c *gin.Context) {
	proxy, ok := h.resolveRuntimeProxyContext(c)
	if !ok {
		return
	}
	suffix := "/conversations/" + url.PathEscape(proxy.conversationID) + "/messages"
	req, err := h.newServiceRequest(c, proxy, http.MethodGet, suffix, nil)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.proxyJSON(c, req)
}

func (h *Handler) UploadAttachment(c *gin.Context) {
	proxy, ok := h.resolveRuntimeProxyContext(c)
	if !ok {
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader.Size <= 0 || fileHeader.Size > maxAgentAttachmentBytes {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("open attachment: %w", err))
		return
	}
	defer file.Close()
	suffix := "/conversations/" + url.PathEscape(proxy.conversationID) + "/attachments"
	req, err := h.newServiceRequest(c, proxy, http.MethodPost, suffix, io.LimitReader(file, maxAgentAttachmentBytes+1))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	req.ContentLength = fileHeader.Size
	req.Header.Set("Content-Type", attachmentContentType(fileHeader))
	req.Header.Set("X-File-Name", url.PathEscape(fileHeader.Filename))
	h.proxyJSON(c, req)
}

func attachmentContentType(file *multipart.FileHeader) string {
	if value := strings.TrimSpace(file.Header.Get("Content-Type")); value != "" {
		return value
	}
	return "application/octet-stream"
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	proxy, ok := h.resolveRuntimeProxyContext(c)
	if !ok {
		return
	}
	suffix := "/conversations/" + url.PathEscape(proxy.conversationID) + "/attachments/" + url.PathEscape(c.Param("attachmentID"))
	req, err := h.newServiceRequest(c, proxy, http.MethodDelete, suffix, nil)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.proxyJSON(c, req)
}

func (h *Handler) CancelRun(c *gin.Context) {
	proxy, ok := h.resolveRuntimeProxyContext(c)
	if !ok {
		return
	}
	suffix := "/runs/" + url.PathEscape(c.Param("runID")) + "/cancel"
	req, err := h.newServiceRequest(c, proxy, http.MethodPost, suffix, nil)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.proxyJSON(c, req)
}

func (h *Handler) GetArtifactContent(c *gin.Context) {
	proxy, ok := h.resolveRuntimeProxyContext(c)
	if !ok {
		return
	}
	disposition := "inline"
	if c.Query("disposition") == "attachment" {
		disposition = "attachment"
	}
	suffix := "/conversations/" + url.PathEscape(proxy.conversationID) + "/artifacts/" + url.PathEscape(c.Param("artifactID")) + "/content?disposition=" + disposition
	req, err := h.newServiceRequest(c, proxy, http.MethodGet, suffix, nil)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	req.Header.Set("Accept", "*/*")
	resp, err := h.http.Do(req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("agent runtime unreachable: %w", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("%s", strings.TrimSpace(string(detail))))
		return
	}
	for _, key := range []string{"Content-Type", "Content-Length", "Content-Disposition", "Cache-Control", "X-Content-Type-Options"} {
		copyHeader(c.Writer.Header(), resp.Header, key)
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, resp.Body)
}

// PrepareHandbook builds and caches the immutable handbook for the complete
// route set registered by this OMC instance.
func (h *Handler) PrepareHandbook() error {
	if h == nil || h.tools == nil {
		return fmt.Errorf("agent handbook is unavailable")
	}
	h.tools.ensureRuntimeHandbook()
	if h.tools.handbookErr != nil {
		return h.tools.handbookErr
	}
	manifest, err := h.tools.handbookManifest()
	if err != nil {
		return err
	}
	h.logger.Info("agent handbook prepared",
		zap.String("catalog_version", manifest.CatalogVersion),
		zap.String("handbook_digest", manifest.HandbookDigest),
		zap.Int("total_operations", manifest.TotalOperations),
		zap.Int("archive_bytes", manifest.ArchiveBytes),
	)
	return nil
}

func (h *Handler) GetHandbookRoutes(c *gin.Context) {
	response.OK(c, h.tools.HandbookRouteExport())
}

// GetHandbookManifest returns the immutable API handbook artifact identity and transfer layout.
func (h *Handler) GetHandbookManifest(c *gin.Context) {
	manifest, err := h.tools.handbookManifest()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, err)
		return
	}
	response.OK(c, manifest)
}

// GetHandbookChunk returns one bounded base64 chunk of the immutable API handbook artifact.
func (h *Handler) GetHandbookChunk(c *gin.Context) {
	index, err := strconv.Atoi(strings.TrimSpace(c.Param("index")))
	if err != nil || index < 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	manifest, err := h.tools.handbookManifest()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, err)
		return
	}
	if index >= manifest.TotalChunks {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	chunk, err := h.tools.handbookChunk(index)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, err)
		return
	}
	response.OK(c, chunk)
}

func (h *Handler) GetConversation(c *gin.Context) {
	h.handleConversation(c, false)
}

func (h *Handler) NewConversation(c *gin.Context) {
	h.handleConversation(c, true)
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
	rawBody, conversationID, err := h.applyActiveConversation(c.Request.Context(), rawBody, target.ConnectorID, claims)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	rawBody = h.injectExternalIdentity(c.Request.Context(), rawBody, claims, target)
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
	if err := clearAgentStreamWriteDeadline(c.Writer); err != nil {
		h.logger.Warn("clear agent stream write deadline failed", zap.Error(err))
	}
	c.Status(http.StatusOK)
	h.streamAgentStudioResponse(c.Request.Context(), c.Writer, resp.Body, claims, conversationID, delegationToken, target)
}

func clearAgentStreamWriteDeadline(w http.ResponseWriter) error {
	return http.NewResponseController(w).SetWriteDeadline(time.Time{})
}

func (h *Handler) handleConversation(c *gin.Context, rotate bool) {
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
	if target == nil || !target.Enabled || target.ConnectorID == "" {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
		return
	}
	if h.conversations == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
		return
	}
	var conversationID string
	if rotate {
		conversationID, err = h.conversations.Rotate(c.Request.Context(), target.ConnectorID, claims)
	} else {
		conversationID, err = h.conversations.Active(c.Request.Context(), target.ConnectorID, claims)
	}
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, ConversationResponse{ConversationID: conversationID})
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

func (h *Handler) applyActiveConversation(ctx context.Context, raw []byte, connectorID string, claims *admin.Claims) ([]byte, string, error) {
	if h.conversations == nil {
		return raw, "", nil
	}
	conversationID, err := h.conversations.Active(ctx, connectorID, claims)
	if err != nil {
		return nil, "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, "", fmt.Errorf("%w: decode agent chat body: %v", commonerrors.ErrInvalidInput, err)
	}
	payload["conversationId"] = conversationID
	next, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("encode agent chat body: %w", err)
	}
	return next, conversationID, nil
}

func (h *Handler) injectExternalIdentity(ctx context.Context, raw []byte, claims *admin.Claims, target *agentconfig.RuntimeTarget) []byte {
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
		"metadata":         h.externalIdentityMetadata(ctx, claims, target),
	}
	payload["context"] = contextValue
	next, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return next
}

func (h *Handler) externalIdentityMetadata(ctx context.Context, claims *admin.Claims, target *agentconfig.RuntimeTarget) map[string]any {
	connectorID := ""
	connectorSlug := ""
	instanceNameIsDefault := true
	if target != nil {
		connectorID = target.ConnectorID
		connectorSlug = target.ConnectorSlug
		instanceNameIsDefault = target.InstanceNameIsDefault || strings.TrimSpace(target.InstanceName) == ""
	}
	metadata := map[string]any{
		"sourceSystem":    "omc",
		"userDisplayName": strings.TrimSpace(claims.Username),
		"connectorId":     strings.TrimSpace(connectorID),
		"connectorSlug":   strings.TrimSpace(connectorSlug),
		"instanceName":    agentRuntimeInstanceName(target),
		"localIPs":        localIPStrings(),
	}
	if h.tools != nil {
		handbook := h.tools.HandbookRouteExport()
		handbookMetadata := map[string]any{
			"schemaVersion":    handbook.SchemaVersion,
			"catalogVersion":   handbook.CatalogVersion,
			"totalOperations":  handbook.TotalRoutes,
			"packageAvailable": false,
		}
		if manifest, err := h.tools.handbookManifest(); err != nil {
			h.logger.Warn("embedded agent handbook package is unavailable", zap.Error(err))
		} else {
			handbookMetadata = map[string]any{
				"schemaVersion":     manifest.SchemaVersion,
				"catalogVersion":    manifest.CatalogVersion,
				"handbookDigest":    manifest.HandbookDigest,
				"totalOperations":   manifest.TotalOperations,
				"manifestPath":      manifest.ManifestPath,
				"chunkPathTemplate": manifest.ChunkPath,
				"archiveFormat":     manifest.ArchiveFormat,
				"archiveBytes":      manifest.ArchiveBytes,
				"chunkBytes":        manifest.ChunkBytes,
				"totalChunks":       manifest.TotalChunks,
				"contentRoot":       manifest.ContentRoot,
				"packageAvailable":  true,
			}
		}
		metadata["apiHandbook"] = handbookMetadata
	}
	if instanceNameIsDefault {
		metadata["instanceNameIsDefault"] = true
	}
	if h.conversations != nil {
		instanceID, err := h.conversations.InstanceID(ctx)
		if err != nil {
			h.logger.Warn("read agent runtime instance id for external identity", zap.Error(err))
		} else if instanceID != "" {
			metadata["instanceId"] = instanceID
			metadata["instanceShortId"] = shortAgentInstanceID(instanceID)
		}
	}
	return metadata
}

func agentRuntimeInstanceName(target *agentconfig.RuntimeTarget) string {
	if target == nil {
		return agentconfig.DefaultOMCName
	}
	if value := strings.TrimSpace(target.InstanceName); value != "" {
		return value
	}
	return agentconfig.DefaultOMCName
}

func shortAgentInstanceID(instanceID string) string {
	value := strings.TrimPrefix(strings.TrimSpace(instanceID), instanceIDPrefix)
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func localIPStrings() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]string, 0, 4)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				out = append(out, ip4.String())
			}
			if len(out) >= 8 {
				return out
			}
		}
	}
	return out
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

func (h *Handler) streamAgentStudioResponse(ctx context.Context, w gin.ResponseWriter, body io.Reader, claims *admin.Claims, conversationID, delegationToken string, target *agentconfig.RuntimeTarget) {
	flusher, _ := w.(http.Flusher)
	reader := bufio.NewReader(body)
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if strings.TrimRight(line, "\r\n") == "" {
				rawFrame := frame.String()
				frame.Reset()
				if !h.handleSSEFrame(ctx, w, rawFrame, claims, conversationID, delegationToken, target, flusher) {
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
				_ = h.handleSSEFrame(ctx, w, frame.String(), claims, conversationID, delegationToken, target, flusher)
			}
			return
		}
		if err != nil {
			h.logger.Warn("agent stream upstream read failed", zap.Error(err))
			return
		}
	}
}

func (h *Handler) handleSSEFrame(ctx context.Context, w gin.ResponseWriter, raw string, claims *admin.Claims, conversationID, delegationToken string, target *agentconfig.RuntimeTarget, flusher http.Flusher) bool {
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
	if result.file != nil {
		file, err := h.uploadToolResultFile(ctx, target, claims, conversationID, result.file)
		if err != nil {
			result.Status = "error"
			result.Output = nil
			result.Error = &ToolError{
				Code:      "FILE_TRANSFER_FAILED",
				Message:   fmt.Sprintf("transfer tool result file: %v", err),
				Retryable: true,
			}
		} else {
			result.Files = []ToolResultFile{file}
		}
		result.file = nil
	}
	if err := h.postToolResult(ctx, target, delegationToken, result); err != nil {
		h.logger.Warn("agent tool result callback failed", zap.Error(err))
	}
	return true
}

func (h *Handler) uploadToolResultFile(
	ctx context.Context,
	target *agentconfig.RuntimeTarget,
	claims *admin.Claims,
	conversationID string,
	file *toolFilePayload,
) (ToolResultFile, error) {
	if target == nil || claims == nil || file == nil || strings.TrimSpace(conversationID) == "" {
		return ToolResultFile{}, fmt.Errorf("tool result file context is incomplete")
	}
	if strings.TrimSpace(target.AgentStudioServiceToken) == "" {
		return ToolResultFile{}, fmt.Errorf("Agent Studio service token is not configured")
	}
	suffix := "/conversations/" + url.PathEscape(conversationID) + "/attachments"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, actionConnectorServiceURL(target, suffix), bytes.NewReader(file.content))
	if err != nil {
		return ToolResultFile{}, fmt.Errorf("build file upload request: %w", err)
	}
	req.ContentLength = int64(len(file.content))
	req.Header.Set("Authorization", "Bearer "+target.AgentStudioServiceToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", file.mimeType)
	req.Header.Set("X-External-User-ID", claims.UserID.String())
	req.Header.Set("X-File-Name", url.PathEscape(file.filename))
	resp, err := h.http.Do(req)
	if err != nil {
		return ToolResultFile{}, fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ToolResultFile{}, fmt.Errorf("read file upload response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ToolResultFile{}, fmt.Errorf("file upload HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Attachment ToolResultFile `json:"attachment"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ToolResultFile{}, fmt.Errorf("decode file upload response: %w", err)
	}
	if payload.Attachment.AttachmentID == "" ||
		payload.Attachment.SizeBytes != len(file.content) ||
		!strings.EqualFold(payload.Attachment.SHA256, file.sha256) {
		return ToolResultFile{}, fmt.Errorf("file upload integrity validation failed")
	}
	return payload.Attachment, nil
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
