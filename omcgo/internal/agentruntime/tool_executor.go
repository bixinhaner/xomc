package agentruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
)

type RouteProvider interface {
	Routes() gin.RoutesInfo
}

type ToolRequest struct {
	RunID      string          `json:"runId"`
	ToolCallID string          `json:"toolCallId"`
	Tool       string          `json:"tool"`
	Title      string          `json:"title"`
	Input      ToolRequestBody `json:"input"`
}

type ToolRequestBody struct {
	OperationID string         `json:"operationId"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	Query       map[string]any `json:"query"`
	Body        any            `json:"body"`
	Reason      string         `json:"reason"`
}

type ToolResult struct {
	RunID      string     `json:"runId"`
	ToolCallID string     `json:"toolCallId"`
	Status     string     `json:"status"`
	Output     any        `json:"output,omitempty"`
	Error      *ToolError `json:"error,omitempty"`
}

type ToolError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

type ToolExecutor struct {
	handler http.Handler
	routes  RouteProvider
	jwt     *admin.JWTService
}

func NewToolExecutor(handler http.Handler, routes RouteProvider, jwt *admin.JWTService) *ToolExecutor {
	return &ToolExecutor{handler: handler, routes: routes, jwt: jwt}
}

func (e *ToolExecutor) Execute(ctx context.Context, claims *admin.Claims, request ToolRequest, policy agentconfig.RuntimePolicy) ToolResult {
	result := ToolResult{RunID: request.RunID, ToolCallID: request.ToolCallID}
	output, err := e.execute(ctx, claims, request.Input, policy)
	if err != nil {
		result.Status = "error"
		result.Error = &ToolError{Code: "TOOL_EXECUTION_FAILED", Message: err.Error(), Retryable: false}
		return result
	}
	result.Status = "ok"
	result.Output = output
	return result
}

func (e *ToolExecutor) execute(ctx context.Context, claims *admin.Claims, input ToolRequestBody, policy agentconfig.RuntimePolicy) (any, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	requestPath := cleanAPIPath(input.Path)
	if method == "" {
		method = http.MethodGet
	}
	if requestPath == "" || !strings.HasPrefix(requestPath, "/api/v1/") {
		return nil, fmt.Errorf("path must be under /api/v1")
	}
	if !methodAllowed(policy, method) {
		return nil, fmt.Errorf("method %s is not enabled for agent tools", method)
	}
	switch requestPath {
	case "/api/v1/agent/catalog":
		return e.catalog(input.Query, policy), nil
	case "/api/v1/agent/catalog/describe":
		return e.describe(input.Query, policy), nil
	}
	if pathBlocked(policy, requestPath) {
		return nil, fmt.Errorf("path %s is blocked by agent policy", requestPath)
	}

	return e.callLocalAPI(ctx, claims, method, requestPath, input.Query, input.Body, policy)
}

func (e *ToolExecutor) callLocalAPI(
	ctx context.Context,
	claims *admin.Claims,
	method string,
	requestPath string,
	query map[string]any,
	body any,
	policy agentconfig.RuntimePolicy,
) (any, error) {
	if e.handler == nil || e.jwt == nil {
		return nil, fmt.Errorf("local API executor is not available")
	}
	timeout := time.Duration(policy.ToolTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(agentconfig.DefaultToolTimeoutSeconds) * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, buildTarget(requestPath, query), bodyReader).WithContext(callCtx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Source", "agent")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	tokenPair, err := e.jwt.GenerateTokenPair(claims)
	if err != nil {
		return nil, fmt.Errorf("create local access token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		return nil, fmt.Errorf("local API returned HTTP %d: %s", rec.Code, strings.TrimSpace(limitString(rec.Body.String(), 4096)))
	}
	return decodeLimitedResponse(rec.Body.Bytes(), policy.MaxResponseBytes)
}

func buildTarget(requestPath string, query map[string]any) string {
	values := url.Values{}
	for key, value := range query {
		if key == "" || value == nil {
			continue
		}
		switch v := value.(type) {
		case []any:
			for _, item := range v {
				values.Add(key, fmt.Sprint(item))
			}
		case []string:
			for _, item := range v {
				values.Add(key, item)
			}
		default:
			values.Set(key, fmt.Sprint(value))
		}
	}
	if encoded := values.Encode(); encoded != "" {
		return requestPath + "?" + encoded
	}
	return requestPath
}

func decodeLimitedResponse(raw []byte, maxBytes int) (any, error) {
	if maxBytes <= 0 {
		maxBytes = agentconfig.DefaultMaxResponseBytes
	}
	truncated := len(raw) > maxBytes
	if truncated {
		raw = raw[:maxBytes]
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{
			"text":      string(raw),
			"truncated": truncated,
		}, nil
	}
	if truncated {
		return map[string]any{"data": decoded, "truncated": true}, nil
	}
	return decoded, nil
}

func limitString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func (e *ToolExecutor) catalog(query map[string]any, policy agentconfig.RuntimePolicy) any {
	q := strings.ToLower(queryText(query, "q"))
	limit := queryInt(query, "limit", 50, 50)
	routes := e.filteredRoutes(policy)
	items := make([]map[string]any, 0, len(routes))
	totalMatched := 0
	for _, route := range routes {
		doc := apiCatalogDoc(route)
		text := strings.ToLower(strings.Join([]string{
			doc.OperationID,
			doc.Method,
			doc.Path,
			doc.Title,
			doc.Summary,
			doc.Description,
			doc.Group,
			strings.Join(doc.Tags, " "),
		}, " "))
		if q != "" && !strings.Contains(text, q) {
			continue
		}
		totalMatched++
		if len(items) >= limit {
			continue
		}
		items = append(items, map[string]any{
			"operationId": doc.OperationID,
			"method":      doc.Method,
			"path":        doc.Path,
			"title":       doc.Title,
			"summary":     doc.Summary,
			"description": doc.Description,
			"group":       doc.Group,
			"risk":        doc.Risk,
			"tags":        doc.Tags,
		})
	}
	return map[string]any{
		"items":        items,
		"total":        totalMatched,
		"returned":     len(items),
		"totalRoutes":  len(routes),
		"catalogUsage": `Use describe with operationId before request. Then call rest.request with {"method","path","query","body","reason"}.`,
	}
}

func (e *ToolExecutor) describe(query map[string]any, policy agentconfig.RuntimePolicy) any {
	target := queryText(query, "operationId")
	method := strings.ToUpper(queryText(query, "method"))
	for _, route := range e.filteredRoutes(policy) {
		if method != "" && strings.ToUpper(route.Method) != method {
			continue
		}
		doc := apiCatalogDoc(route)
		if doc.OperationID != target && route.Path != target {
			continue
		}
		return doc
	}
	return map[string]any{"error": "operation not found"}
}

func queryText(query map[string]any, key string) string {
	if query == nil {
		return ""
	}
	value, ok := query[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func queryInt(query map[string]any, key string, fallback int, max int) int {
	if query == nil {
		return fallback
	}
	value, ok := query[key]
	if !ok || value == nil {
		return fallback
	}
	var parsed int
	switch v := value.(type) {
	case int:
		parsed = v
	case int64:
		parsed = int(v)
	case float64:
		parsed = int(v)
	case string:
		next, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fallback
		}
		parsed = next
	default:
		return fallback
	}
	if parsed <= 0 {
		return fallback
	}
	if max > 0 && parsed > max {
		return max
	}
	return parsed
}

func (e *ToolExecutor) filteredRoutes(policy agentconfig.RuntimePolicy) gin.RoutesInfo {
	if e.routes == nil {
		return nil
	}
	routes := e.routes.Routes()
	filtered := make(gin.RoutesInfo, 0, len(routes))
	for _, route := range routes {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		if pathBlocked(policy, route.Path) || !methodAllowed(policy, route.Method) {
			continue
		}
		filtered = append(filtered, route)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Path == filtered[j].Path {
			return filtered[i].Method < filtered[j].Method
		}
		return filtered[i].Path < filtered[j].Path
	})
	return filtered
}

func operationID(method, requestPath string) string {
	cleaned := strings.Trim(cleanAPIPath(requestPath), "/")
	cleaned = strings.ReplaceAll(cleaned, "/:", "/by-")
	cleaned = strings.ReplaceAll(cleaned, "/", ".")
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	return strings.ToLower(method) + "." + strings.TrimPrefix(cleaned, "api.v1.")
}

func riskForMethod(method string) string {
	if isReadMethod(method) {
		return "read"
	}
	if strings.EqualFold(method, http.MethodDelete) {
		return "high"
	}
	return "low"
}
