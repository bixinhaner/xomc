package agentruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
	"github.com/omcgo/omcgo/internal/agentruntime/handbookgen"
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
	RunID      string           `json:"runId"`
	ToolCallID string           `json:"toolCallId"`
	Status     string           `json:"status"`
	Output     any              `json:"output,omitempty"`
	Files      []ToolResultFile `json:"files,omitempty"`
	Error      *ToolError       `json:"error,omitempty"`
	file       *toolFilePayload
}

type ToolResultFile struct {
	AttachmentID string `json:"attachmentId"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mimeType"`
	SizeBytes    int    `json:"sizeBytes"`
	SHA256       string `json:"sha256"`
	CreatedAt    string `json:"createdAt,omitempty"`
}

type toolFilePayload struct {
	filename string
	mimeType string
	content  []byte
	sha256   string
}

type ToolError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

type toolExecutionError struct {
	code    string
	message string
	details map[string]any
}

func (e *toolExecutionError) Error() string {
	return e.message
}

type ToolExecutor struct {
	handler      http.Handler
	routes       RouteProvider
	jwt          *admin.JWTService
	handbook     *handbookPackage
	handbookErr  error
	handbookOnce sync.Once
}

func NewToolExecutor(handler http.Handler, routes RouteProvider, jwt *admin.JWTService) *ToolExecutor {
	return &ToolExecutor{handler: handler, routes: routes, jwt: jwt}
}

// HandbookDigest returns the digest of the exact runtime API handbook used by
// ToolExecutor. Background Connector events pin this value so a Run never
// silently switches API contracts halfway through an investigation.
func (e *ToolExecutor) HandbookDigest() (string, error) {
	manifest, err := e.handbookManifest()
	if err != nil {
		return "", fmt.Errorf("load runtime handbook digest: %w", err)
	}
	return manifest.HandbookDigest, nil
}

// HandbookMetadata exposes the immutable package coordinates required by a
// remote background agent to download the exact API contract pinned to an
// event. It intentionally contains no user or connector credentials.
func (e *ToolExecutor) HandbookMetadata() (map[string]any, error) {
	manifest, err := e.handbookManifest()
	if err != nil {
		return nil, fmt.Errorf("load runtime handbook metadata: %w", err)
	}
	return map[string]any{
		"schemaVersion": manifest.SchemaVersion, "catalogVersion": manifest.CatalogVersion,
		"handbookDigest": manifest.HandbookDigest, "totalOperations": manifest.TotalOperations,
		"manifestPath": manifest.ManifestPath, "chunkPathTemplate": manifest.ChunkPath,
		"archiveFormat": manifest.ArchiveFormat, "archiveBytes": manifest.ArchiveBytes,
		"chunkBytes": manifest.ChunkBytes, "totalChunks": manifest.TotalChunks,
		"contentRoot": manifest.ContentRoot, "packageAvailable": true,
	}, nil
}

func (e *ToolExecutor) Execute(ctx context.Context, claims *admin.Claims, request ToolRequest, policy agentconfig.RuntimePolicy) ToolResult {
	result := ToolResult{RunID: request.RunID, ToolCallID: request.ToolCallID}
	output, err := e.execute(ctx, claims, request.Input, policy)
	if err != nil {
		result.Status = "error"
		result.Error = &ToolError{Code: "TOOL_EXECUTION_FAILED", Message: err.Error(), Retryable: false}
		var executionError *toolExecutionError
		if errors.As(err, &executionError) {
			result.Error.Code = executionError.code
			result.Error.Details = executionError.details
		}
		return result
	}
	result.Status = "ok"
	if file, ok := output.(*toolFilePayload); ok {
		result.file = file
		result.Output = map[string]any{
			"type":      "file",
			"filename":  file.filename,
			"mimeType":  file.mimeType,
			"sizeBytes": len(file.content),
			"sha256":    file.sha256,
		}
	} else {
		result.Output = output
	}
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
	case "/api/v1/agent/catalog/categories":
		return e.catalogCategories(policy), nil
	case "/api/v1/agent/catalog/describe":
		return e.describe(input.Query, policy), nil
	case "/api/v1/agent/handbook/manifest":
		if method != http.MethodGet {
			return nil, fmt.Errorf("handbook package manifest only supports GET")
		}
		return e.handbookManifest()
	}
	if strings.HasPrefix(requestPath, "/api/v1/agent/handbook/chunks/") {
		if method != http.MethodGet {
			return nil, fmt.Errorf("handbook package chunks only support GET")
		}
		indexText := strings.TrimPrefix(requestPath, "/api/v1/agent/handbook/chunks/")
		index, err := strconv.Atoi(indexText)
		if err != nil || index < 0 {
			return nil, fmt.Errorf("invalid handbook chunk index %q", indexText)
		}
		return e.handbookChunk(index)
	}
	if pathBlocked(policy, requestPath) {
		return nil, fmt.Errorf("path %s is blocked by agent policy", requestPath)
	}
	if err := e.validateToolRequest(input, method, requestPath); err != nil {
		return nil, err
	}

	return e.callLocalAPI(ctx, claims, method, requestPath, input.Query, input.Body, policy)
}

func (e *ToolExecutor) validateToolRequest(input ToolRequestBody, method, requestPath string) error {
	e.ensureRuntimeHandbook()
	if e.handbookErr != nil {
		return &toolExecutionError{
			code:    "HANDBOOK_UNAVAILABLE",
			message: "the running API contract is unavailable",
			details: map[string]any{"cause": e.handbookErr.Error()},
		}
	}
	expected, matched := e.handbook.matchOperation(method, requestPath)
	if !matched {
		return &toolExecutionError{
			code:    "OPERATION_NOT_FOUND",
			message: fmt.Sprintf("no running API operation matches %s %s", method, requestPath),
			details: map[string]any{"method": method, "path": requestPath},
		}
	}
	if operationID := strings.TrimSpace(input.OperationID); operationID != "" {
		document, exists := e.handbook.operation(operationID)
		if !exists {
			return &toolExecutionError{
				code:    "OPERATION_NOT_FOUND",
				message: fmt.Sprintf("operationId %s is not published by the running OMC", operationID),
				details: map[string]any{"expectedOperationId": expected.OperationID},
			}
		}
		if document.Method != method || !handbookPathMatches(document.Path, requestPath) {
			return &toolExecutionError{
				code:    "OPERATION_CONTRACT_MISMATCH",
				message: fmt.Sprintf("operationId %s does not match %s %s", operationID, method, requestPath),
				details: map[string]any{
					"expectedOperationId": expected.OperationID,
					"expectedMethod":      expected.Method,
					"expectedPath":        expected.Path,
				},
			}
		}
		expected = document
	}
	if expected.ContractCoverage["request"] == "path-only" {
		return nil
	}
	return validateQueryContract(expected, input.Query)
}

func validateQueryContract(document handbookgen.OperationDocument, query map[string]any) error {
	allowed := make(map[string]handbookgen.Parameter, len(document.QueryParams))
	allowedNames := make([]string, 0, len(document.QueryParams))
	for _, parameter := range document.QueryParams {
		allowed[parameter.Name] = parameter
		allowedNames = append(allowedNames, parameter.Name)
		if parameter.Required {
			if value, ok := query[parameter.Name]; !ok || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
				return &toolExecutionError{
					code:    "MISSING_QUERY_PARAMETER",
					message: fmt.Sprintf("required query parameter %s is missing for %s", parameter.Name, document.OperationID),
					details: map[string]any{"parameter": parameter.Name, "allowedParameters": allowedNames},
				}
			}
		}
	}
	sort.Strings(allowedNames)
	for name, value := range query {
		parameter, ok := allowed[name]
		if !ok {
			return &toolExecutionError{
				code:    "INVALID_QUERY_PARAMETER",
				message: fmt.Sprintf("query parameter %s is not accepted by %s", name, document.OperationID),
				details: map[string]any{"parameter": name, "allowedParameters": allowedNames},
			}
		}
		if len(parameter.Enum) > 0 && !enumContains(parameter.Enum, value) {
			return &toolExecutionError{
				code:    "INVALID_QUERY_VALUE",
				message: fmt.Sprintf("query parameter %s has an unsupported value", name),
				details: map[string]any{"parameter": name, "allowedValues": parameter.Enum},
			}
		}
	}
	return nil
}

func enumContains(allowed []any, value any) bool {
	actual := strings.TrimSpace(fmt.Sprint(value))
	for _, candidate := range allowed {
		if strings.EqualFold(actual, strings.TrimSpace(fmt.Sprint(candidate))) {
			return true
		}
	}
	return false
}

func (e *ToolExecutor) handbookManifest() (HandbookPackageManifest, error) {
	if e == nil {
		return HandbookPackageManifest{}, fmt.Errorf("handbook package is unavailable")
	}
	e.ensureRuntimeHandbook()
	if e.handbookErr != nil {
		return HandbookPackageManifest{}, e.handbookErr
	}
	return e.handbook.validatedManifest(e.HandbookRouteExport())
}

func (e *ToolExecutor) handbookChunk(index int) (HandbookPackageChunk, error) {
	if e == nil {
		return HandbookPackageChunk{}, fmt.Errorf("handbook package is unavailable")
	}
	e.ensureRuntimeHandbook()
	if e.handbookErr != nil {
		return HandbookPackageChunk{}, e.handbookErr
	}
	return e.handbook.chunk(e.HandbookRouteExport(), index)
}

func (e *ToolExecutor) ensureRuntimeHandbook() {
	e.handbookOnce.Do(func() {
		if e.handbook != nil || e.handbookErr != nil {
			return
		}
		e.handbook, e.handbookErr = buildRuntimeHandbookPackage(e.HandbookRouteExport())
	})
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
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("X-Agent-Source", "agent")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	accessToken := ""
	var err error
	if hasBackgroundAgentScope(claims.Scopes) {
		accessToken, _, err = e.jwt.GenerateBackgroundAgentToken(claims)
	} else {
		var tokenPair *admin.TokenPair
		tokenPair, err = e.jwt.GenerateTokenPair(claims)
		if tokenPair != nil {
			accessToken = tokenPair.AccessToken
		}
	}
	if err != nil {
		return nil, fmt.Errorf("create local access token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		return nil, fmt.Errorf("local API returned HTTP %d: %s", rec.Code, strings.TrimSpace(limitString(rec.Body.String(), 4096)))
	}
	if filename, mimeType, ok := downloadableResponse(rec.Header(), requestPath); ok {
		content := rec.Body.Bytes()
		if len(content) == 0 {
			return nil, fmt.Errorf("local API returned an empty file")
		}
		if len(content) > maxAgentAttachmentBytes {
			return nil, fmt.Errorf("local API file exceeds %d bytes", maxAgentAttachmentBytes)
		}
		digest := sha256.Sum256(content)
		return &toolFilePayload{
			filename: filename,
			mimeType: mimeType,
			content:  bytes.Clone(content),
			sha256:   fmt.Sprintf("%x", digest),
		}, nil
	}
	return decodeLimitedResponse(rec.Body.Bytes(), policy.MaxResponseBytes)
}

func hasBackgroundAgentScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.HasPrefix(scope, "agent-background:") {
			return true
		}
	}
	return false
}

func downloadableResponse(header http.Header, requestPath string) (string, string, bool) {
	contentType, _, _ := mime.ParseMediaType(strings.TrimSpace(header.Get("Content-Type")))
	filename := strings.TrimSpace(header.Get("X-File-Name"))
	disposition, params, _ := mime.ParseMediaType(strings.TrimSpace(header.Get("Content-Disposition")))
	if filename == "" {
		filename = strings.TrimSpace(params["filename"])
	}
	isFile := filename != "" || strings.EqualFold(disposition, "attachment") ||
		contentType == "application/octet-stream"
	if !isFile {
		return "", "", false
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if decoded, err := url.PathUnescape(filename); err == nil {
		filename = decoded
	}
	filename = sanitizeToolFilename(filename)
	if filename == "" {
		filename = sanitizeToolFilename(path.Base(requestPath))
	}
	if filename == "" || filename == "." || filename == "/" {
		filename = "download"
	}
	return filename, contentType, true
}

func sanitizeToolFilename(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = path.Base(value)
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	return strings.TrimSpace(value)
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
	q := queryText(query, "q")
	category := strings.ToLower(queryText(query, "category"))
	limit := queryInt(query, "limit", 20, 100)
	offset := queryInt(query, "offset", 0, 0)
	routes := e.filteredRoutes(policy)
	matched := make([]apiRouteDoc, 0, len(routes))
	for _, route := range routes {
		doc := apiCatalogDoc(route)
		if category != "" && strings.ToLower(doc.Group) != category {
			continue
		}
		if !apiCatalogMatches(doc, q) {
			continue
		}
		matched = append(matched, doc)
	}
	sort.SliceStable(matched, func(i, j int) bool {
		leftScore := apiCatalogMatchScore(matched[i], q)
		rightScore := apiCatalogMatchScore(matched[j], q)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if matched[i].Path == matched[j].Path {
			return matched[i].Method < matched[j].Method
		}
		return matched[i].Path < matched[j].Path
	})
	if offset > len(matched) {
		offset = len(matched)
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	items := make([]map[string]any, 0, end-offset)
	for _, doc := range matched[offset:end] {
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
	hasMore := end < len(matched)
	nextOffset := end
	return map[string]any{
		"items":          items,
		"total":          len(matched),
		"returned":       len(items),
		"offset":         offset,
		"nextOffset":     nextOffset,
		"hasMore":        hasMore,
		"category":       category,
		"totalRoutes":    len(routes),
		"catalogVersion": apiCatalogVersion(routes),
		"catalogUsage":   `Reuse known or previously successful operations directly. For unknown APIs, search with category and concise path tokens; describe only when parameters or write semantics are unclear.`,
	}
}

func (e *ToolExecutor) catalogCategories(policy agentconfig.RuntimePolicy) any {
	routes := e.filteredRoutes(policy)
	type categorySummary struct {
		count   int
		methods map[string]struct{}
	}
	summaries := make(map[string]*categorySummary)
	for _, route := range routes {
		doc := apiCatalogDoc(route)
		category := doc.Group
		if category == "" {
			category = "other"
		}
		summary := summaries[category]
		if summary == nil {
			summary = &categorySummary{methods: make(map[string]struct{})}
			summaries[category] = summary
		}
		summary.count++
		summary.methods[doc.Method] = struct{}{}
	}

	categoryIDs := make([]string, 0, len(summaries))
	for category := range summaries {
		categoryIDs = append(categoryIDs, category)
	}
	sort.Strings(categoryIDs)
	items := make([]map[string]any, 0, len(categoryIDs))
	for _, category := range categoryIDs {
		summary := summaries[category]
		methods := make([]string, 0, len(summary.methods))
		for method := range summary.methods {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		items = append(items, map[string]any{
			"id":      category,
			"title":   apiCategoryTitle(category),
			"count":   summary.count,
			"methods": methods,
		})
	}
	return map[string]any{
		"items":           items,
		"totalCategories": len(items),
		"totalRoutes":     len(routes),
		"catalogVersion":  apiCatalogVersion(routes),
		"categoryUsage":   `Choose one category, then search /api/v1/agent/catalog with category, concise q tokens, and a small limit.`,
	}
}

func apiCatalogMatches(doc apiRouteDoc, query string) bool {
	tokens := apiCatalogSearchTokens(query)
	if len(tokens) == 0 {
		return true
	}
	text := apiCatalogSearchText(doc)
	for _, token := range tokens {
		if !strings.Contains(text, token) {
			return false
		}
	}
	return true
}

func apiCatalogMatchScore(doc apiRouteDoc, query string) int {
	normalizedQuery := strings.Join(apiCatalogSearchTokens(query), " ")
	if normalizedQuery == "" {
		return 0
	}
	operationID := normalizeAPICatalogSearchText(doc.OperationID)
	requestPath := normalizeAPICatalogSearchText(doc.Path)
	title := normalizeAPICatalogSearchText(doc.Title)
	score := 0
	if operationID == normalizedQuery || requestPath == normalizedQuery {
		score += 100
	}
	if strings.Contains(operationID, normalizedQuery) {
		score += 30
	}
	if strings.Contains(requestPath, normalizedQuery) {
		score += 20
	}
	if strings.Contains(title, normalizedQuery) {
		score += 10
	}
	return score
}

func apiCatalogSearchText(doc apiRouteDoc) string {
	return normalizeAPICatalogSearchText(strings.Join([]string{
		doc.OperationID,
		doc.Method,
		doc.Path,
		doc.Title,
		doc.Summary,
		doc.Description,
		doc.Group,
		strings.Join(doc.Tags, " "),
	}, " "))
}

func apiCatalogSearchTokens(query string) []string {
	return strings.Fields(normalizeAPICatalogSearchText(query))
}

func normalizeAPICatalogSearchText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"/", " ",
		".", " ",
		"_", " ",
		"-", " ",
		":", " ",
		"?", " ",
		"&", " ",
		"=", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(normalized)), " ")
}

func apiCatalogVersion(routes gin.RoutesInfo) string {
	hash := sha256.New()
	for _, route := range routes {
		_, _ = fmt.Fprintf(hash, "%s\x00%s\n", strings.ToUpper(route.Method), route.Path)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
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
