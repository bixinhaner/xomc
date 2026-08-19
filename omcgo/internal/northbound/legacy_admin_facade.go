package northbound

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/northbound/pageconfig"
)

func (r *Router) requirePageConfigFacade(c *gin.Context) bool {
	if r.pageConfigService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "northbound page-config facade is not configured")
		return false
	}
	return true
}

func (r *Router) legacyListAPIUsers(c *gin.Context) {
	if !r.requirePageConfigFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	items, err := r.pageConfigService.ListAPIUsers(c.Request.Context())
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	operatorCode := firstNonEmptyString(
		firstStringFromMap(body, "operatorCode", "operator_code"),
		c.Query("operatorCode"),
		c.Query("operator_code"),
	)
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, legacyAPIUserPayload(item, operatorCode))
	}
	response.OK(c, out)
}

func (r *Router) legacyCreateAPIUser(c *gin.Context) {
	if !r.requirePageConfigFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	user := pageconfig.APIUser{
		Username: firstStringFromMap(body, "username", "userName", "client_key", "clientKey"),
		Password: firstStringFromMap(body, "password", "userPwd", "passwd", "token_secret", "tokenSecret"),
		Enabled:  true,
	}
	if enabled, exists := legacyBoolFromMap(body, "enabled", "userEnable", "enable"); exists {
		user.Enabled = enabled
	}
	created, err := r.pageConfigService.CreateAPIUser(c.Request.Context(), user)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, legacyAPIUserPayload(*created, firstStringFromMap(body, "operatorCode", "operator_code")))
}

func (r *Router) legacyUpdateAPIUser(c *gin.Context) {
	if !r.requirePageConfigFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	idOrUsername := firstStringFromMap(body, "id", "userId")
	if idOrUsername == "" {
		idOrUsername = firstStringFromMap(body, "username", "userName", "client_key", "clientKey")
	}
	req := pageconfig.UpdateAPIUserRequest{
		Username: firstStringFromMap(body, "username", "client_key", "clientKey"),
		Password: firstStringFromMap(body, "password", "userPwd", "passwd", "token_secret", "tokenSecret"),
	}
	if firstStringFromMap(body, "id", "userId") != "" && req.Username == "" {
		req.Username = firstStringFromMap(body, "userName")
	}
	if enabled, exists := legacyBoolFromMap(body, "enabled", "userEnable", "enable"); exists {
		req.Enabled = &enabled
	}
	updated, err := r.pageConfigService.UpdateAPIUser(c.Request.Context(), idOrUsername, req)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, legacyAPIUserPayload(*updated, firstStringFromMap(body, "operatorCode", "operator_code")))
}

func (r *Router) legacyDeleteAPIUser(c *gin.Context) {
	if !r.requirePageConfigFacade(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if err := r.pageConfigService.DeleteAPIUser(c.Request.Context(), id); err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithMsg(c, gin.H{"id": id, "userId": id}, "delete northbound API user success")
}

func (r *Router) legacyListAPILogs(c *gin.Context) {
	if !r.requirePageConfigFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	filter, page, pageSize := legacyAPILogFilter(c, body)
	result, err := r.pageConfigService.ListAPIInvocationLogs(c.Request.Context(), filter)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	rows := make([]gin.H, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, legacyAPILogPayload(item))
	}
	response.OK(c, gin.H{
		"rows":        rows,
		"items":       rows,
		"total":       result.Total,
		"page":        page,
		"pageNo":      page,
		"pageSize":    pageSize,
		"page_size":   pageSize,
		"totalRows":   result.Total,
		"total_pages": totalPages(result.Total, pageSize),
	})
}

func (r *Router) legacyExportAPILogs(c *gin.Context) {
	if !r.requirePageConfigFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	filter, _, _ := legacyAPILogFilter(c, body)
	if filter.Limit <= 0 || filter.Limit < 1000 {
		filter.Limit = 1000
	}
	result, err := r.pageConfigService.ListAPIInvocationLogs(c.Request.Context(), filter)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{
		"id", "nameEn", "nameCn", "type", "status", "createUser",
		"ipAddress", "operatorCode", "reqParams", "resParams1",
		"resParams2", "createTime", "updateTime",
	})
	for _, item := range result.Items {
		payload := legacyAPILogPayload(item)
		_ = writer.Write([]string{
			stringFromGinH(payload, "id"),
			stringFromGinH(payload, "nameEn"),
			stringFromGinH(payload, "nameCn"),
			stringFromGinH(payload, "type"),
			stringFromGinH(payload, "status"),
			stringFromGinH(payload, "createUser"),
			stringFromGinH(payload, "ipAddress"),
			stringFromGinH(payload, "operatorCode"),
			stringFromGinH(payload, "reqParams"),
			stringFromGinH(payload, "resParams1"),
			stringFromGinH(payload, "resParams2"),
			stringFromGinH(payload, "createTime"),
			stringFromGinH(payload, "updateTime"),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		response.Fail(c, http.StatusInternalServerError, "export northbound API logs failed")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="northbound-api-log.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func legacyAPIUserPayload(user pageconfig.APIUser, operatorCode string) gin.H {
	return gin.H{
		"id":           user.ID,
		"userId":       user.ID,
		"username":     user.Username,
		"userName":     user.Username,
		"enabled":      user.Enabled,
		"userEnable":   legacyUserEnable(user.Enabled),
		"password_set": user.PasswordSet,
		"operatorCode": operatorCode,
		"createTime":   user.CreatedAt,
		"created_at":   user.CreatedAt,
		"updateTime":   user.UpdatedAt,
		"updated_at":   user.UpdatedAt,
		"responseTime": time.Now().UTC().Format(time.RFC3339),
	}
}

func legacyUserEnable(enabled bool) string {
	if enabled {
		return "1"
	}
	return "0"
}

func legacyBoolFromMap(body map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true
		case float64:
			return v != 0, true
		case json.Number:
			parsed, err := strconv.Atoi(v.String())
			if err == nil {
				return parsed != 0, true
			}
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "1", "true", "yes", "y", "on", "enable", "enabled", "normal":
				return true, true
			case "0", "false", "no", "n", "off", "disable", "disabled", "terminated":
				return false, true
			}
		}
	}
	return false, false
}

func legacyAPILogFilter(c *gin.Context, body map[string]any) (pageconfig.APIInvocationLogFilter, int, int) {
	page := legacyIntBodyOrQuery(c, body, 1, "page", "pageNo")
	pageSize := legacyIntBodyOrQuery(c, body, 20, "rows", "pageSize", "page_size", "limit")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := legacyIntBodyOrQuery(c, body, (page-1)*pageSize, "offset")
	return pageconfig.APIInvocationLogFilter{
		APIKey:     legacyStringBodyOrQuery(c, body, "api_key", "apiKey", "type"),
		Method:     legacyStringBodyOrQuery(c, body, "method"),
		Path:       legacyStringBodyOrQuery(c, body, "path", "url"),
		Status:     legacyStringBodyOrQuery(c, body, "status"),
		CreateUser: legacyStringBodyOrQuery(c, body, "createUser", "create_user", "userName", "username", "operatorCode", "operator_code"),
		IPAddress:  legacyStringBodyOrQuery(c, body, "ipAddress", "ip_address", "ip"),
		Limit:      pageSize,
		Offset:     offset,
	}, page, pageSize
}

func legacyStringBodyOrQuery(c *gin.Context, body map[string]any, keys ...string) string {
	if value := firstStringFromMap(body, keys...); value != "" {
		return value
	}
	for _, key := range keys {
		if value := strings.TrimSpace(c.Query(key)); value != "" {
			return value
		}
	}
	return ""
}

func legacyIntBodyOrQuery(c *gin.Context, body map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		if value := intFromMap(body, key, -1); value > 0 {
			return value
		}
		if parsed, err := strconv.Atoi(strings.TrimSpace(c.Query(key))); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func legacyAPILogPayload(item pageconfig.APIInvocationLog) gin.H {
	return gin.H{
		"id":             item.ID,
		"nameEn":         item.APIKey,
		"nameCn":         item.Name,
		"type":           item.APIKey,
		"reqParams":      item.RequestParams,
		"resParams1":     item.ResponseBody,
		"resParams2":     "",
		"status":         item.Status,
		"statusCode":     item.StatusCode,
		"createUser":     item.CreateUser,
		"ipAddress":      item.IPAddress,
		"operatorCode":   item.CreateUser,
		"createTime":     item.CreatedAt,
		"updateTime":     item.UpdatedAt,
		"api_key":        item.APIKey,
		"name":           item.Name,
		"method":         item.Method,
		"path":           item.Path,
		"request_params": item.RequestParams,
		"response_body":  item.ResponseBody,
		"status_code":    item.StatusCode,
		"create_user":    item.CreateUser,
		"ip_address":     item.IPAddress,
		"duration_ms":    item.DurationMs,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func stringFromGinH(values gin.H, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		raw, err := json.Marshal(v)
		if err == nil {
			return string(raw)
		}
		return ""
	}
}

func totalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
