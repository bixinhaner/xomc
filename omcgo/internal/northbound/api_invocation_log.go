package northbound

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/northbound/pageconfig"
)

const maxAPIInvocationLogBodyBytes = 64 * 1024

type apiInvocationLogReadCloser struct {
	io.ReadCloser
	buf *bytes.Buffer
}

func (r *apiInvocationLogReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 && r.buf.Len() < maxAPIInvocationLogBodyBytes {
		remaining := maxAPIInvocationLogBodyBytes - r.buf.Len()
		if n < remaining {
			remaining = n
		}
		_, _ = r.buf.Write(p[:remaining])
	}
	return n, err
}

type apiInvocationLogResponseWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *apiInvocationLogResponseWriter) Write(data []byte) (int, error) {
	if len(data) > 0 && w.buf.Len() < maxAPIInvocationLogBodyBytes {
		remaining := maxAPIInvocationLogBodyBytes - w.buf.Len()
		if len(data) < remaining {
			remaining = len(data)
		}
		_, _ = w.buf.Write(data[:remaining])
	}
	return w.ResponseWriter.Write(data)
}

func (w *apiInvocationLogResponseWriter) WriteString(data string) (int, error) {
	if data != "" && w.buf.Len() < maxAPIInvocationLogBodyBytes {
		remaining := maxAPIInvocationLogBodyBytes - w.buf.Len()
		if len(data) < remaining {
			remaining = len(data)
		}
		_, _ = w.buf.WriteString(data[:remaining])
	}
	return w.ResponseWriter.WriteString(data)
}

func (r *Router) northboundAPILogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if r.pageConfigService == nil || shouldSkipNorthboundAPILog(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestBody := &bytes.Buffer{}
		if c.Request != nil && c.Request.Body != nil {
			c.Request.Body = &apiInvocationLogReadCloser{ReadCloser: c.Request.Body, buf: requestBody}
		}
		responseBody := &bytes.Buffer{}
		c.Writer = &apiInvocationLogResponseWriter{ResponseWriter: c.Writer, buf: responseBody}

		c.Next()

		apiKey := stringFromContext(c, "northbound_api_key")
		name := r.northboundAPIConfigName(c, apiKey)
		statusCode := c.Writer.Status()
		status := "success"
		if statusCode >= 400 {
			status = "failed"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := r.pageConfigService.CreateAPIInvocationLog(ctx, pageconfig.APIInvocationLog{
			APIKey:        apiKey,
			Name:          name,
			Method:        c.Request.Method,
			Path:          firstNonEmptyString(c.FullPath(), c.Request.URL.Path),
			RequestParams: buildAPIInvocationRequestParams(c, requestBody.Bytes()),
			ResponseBody:  sanitizeAPIInvocationPayload(responseBody.Bytes()),
			StatusCode:    statusCode,
			Status:        status,
			CreateUser:    firstNonEmptyString(stringFromContext(c, "northbound_api_user"), apiInvocationUsernameFromBody(requestBody.Bytes())),
			IPAddress:     c.ClientIP(),
			DurationMs:    time.Since(start).Milliseconds(),
		}); err != nil && r.svc != nil && r.svc.logger != nil {
			r.svc.logger.Warn("record northbound API invocation log failed", zap.Error(err))
		}
	}
}

func shouldSkipNorthboundAPILog(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	return strings.HasSuffix(path, "/log/page") ||
		strings.HasSuffix(path, "/log/exportlogtocsvfile")
}

func (r *Router) northboundAPIConfigName(c *gin.Context, apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" || r.pageConfigService == nil {
		return apiKey
	}
	configs, err := r.pageConfigService.ListAPIConfigs(c.Request.Context())
	if err != nil {
		return apiKey
	}
	for _, config := range configs {
		if strings.EqualFold(config.Key, apiKey) {
			return config.Name
		}
	}
	return apiKey
}

func buildAPIInvocationRequestParams(c *gin.Context, body []byte) string {
	payload := map[string]any{
		"method": c.Request.Method,
		"path":   firstNonEmptyString(c.FullPath(), c.Request.URL.Path),
		"query":  c.Request.URL.RawQuery,
		"body":   sanitizedAPIInvocationValue(body),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return sanitizeAPIInvocationPayload(body)
	}
	return string(raw)
}

func sanitizeAPIInvocationPayload(raw []byte) string {
	value := sanitizedAPIInvocationValue(raw)
	switch v := value.(type) {
	case string:
		return v
	default:
		out, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(out)
	}
}

func sanitizedAPIInvocationValue(raw []byte) any {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return truncateAPIInvocationString(text)
	}
	return redactAPIInvocationValue(value)
}

func redactAPIInvocationValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			if isAPIInvocationSensitiveKey(key) {
				out[key] = "<redacted>"
				continue
			}
			out[key] = redactAPIInvocationValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, redactAPIInvocationValue(item))
		}
		return out
	default:
		return v
	}
}

func isAPIInvocationSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(key, "password") ||
		strings.Contains(key, "passwd") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "credential") ||
		strings.Contains(key, "authorization") ||
		strings.EqualFold(key, "userPwd")
}

func truncateAPIInvocationString(value string) string {
	if len(value) <= maxAPIInvocationLogBodyBytes {
		return value
	}
	return value[:maxAPIInvocationLogBodyBytes] + "...<truncated>"
}

func apiInvocationUsernameFromBody(raw []byte) string {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	return firstStringFromMap(body, "username", "userName", "user", "client_key", "clientKey")
}

func stringFromContext(c *gin.Context, key string) string {
	value, ok := c.Get(key)
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
