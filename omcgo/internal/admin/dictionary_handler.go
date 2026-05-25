package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// DictionaryHandler provides HTTP endpoints for dictionary management.
type DictionaryHandler struct {
	service *DictionaryService
}

// NewDictionaryHandler creates a new DictionaryHandler.
func NewDictionaryHandler(service *DictionaryService) *DictionaryHandler {
	return &DictionaryHandler{service: service}
}

// RegisterRoutes registers dictionary routes on the given router group.
func (h *DictionaryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	dict := rg.Group("/sysDictionary")
	{
		dict.POST("/createSysDictionary", h.CreateDictionary)
		dict.DELETE("/deleteSysDictionary", h.DeleteDictionary)
		dict.PUT("/updateSysDictionary", h.UpdateDictionary)
		dict.GET("/findSysDictionary", h.FindDictionary)
		dict.GET("/getSysDictionaryList", h.GetDictionaryList)
		dict.GET("/batch", h.BatchGetDicts)
	}

	detail := rg.Group("/sysDictionaryDetail")
	{
		detail.POST("/createSysDictionaryDetail", h.CreateDictionaryDetail)
		detail.DELETE("/deleteSysDictionaryDetail", h.DeleteDictionaryDetail)
		detail.PUT("/updateSysDictionaryDetail", h.UpdateDictionaryDetail)
		detail.GET("/findSysDictionaryDetail", h.FindDictionaryDetail)
		detail.GET("/getSysDictionaryDetailList", h.GetDictionaryDetailList)
	}
}

// CreateDictionary handles POST /sysDictionary/createSysDictionary
func (h *DictionaryHandler) CreateDictionary(c *gin.Context) {
	var req CreateDictionaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.CreateDictionary(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, result, "创建成功")
}

// DeleteDictionary handles DELETE /sysDictionary/deleteSysDictionary?id=123
func (h *DictionaryHandler) DeleteDictionary(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "id is required", nil))
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}

	if err := h.service.DeleteDictionary(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, nil, "删除成功")
}

// UpdateDictionary handles PUT /sysDictionary/updateSysDictionary
func (h *DictionaryHandler) UpdateDictionary(c *gin.Context) {
	var req UpdateDictionaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.UpdateDictionary(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, result, "更新成功")
}

// FindDictionary handles GET /sysDictionary/findSysDictionary?type=gender
func (h *DictionaryHandler) FindDictionary(c *gin.Context) {
	dictType := c.Query("type")
	if dictType == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "type is required", nil))
		return
	}

	result, err := h.service.GetDictionaryByType(c.Request.Context(), dictType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, result, "查询成功")
}

// GetDictionaryList handles GET /sysDictionary/getSysDictionaryList
func (h *DictionaryHandler) GetDictionaryList(c *gin.Context) {
	result, err := h.service.ListDictionaries(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if result == nil {
		result = []Dictionary{}
	}

	response.OKWithMsg(c, gin.H{
		"list":  result,
		"total": len(result),
	}, "查询成功")
}

// CreateDictionaryDetail handles POST /sysDictionaryDetail/createSysDictionaryDetail
func (h *DictionaryHandler) CreateDictionaryDetail(c *gin.Context) {
	var req CreateDictionaryDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.CreateDictionaryDetail(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, result, "创建成功")
}

// DeleteDictionaryDetail handles DELETE /sysDictionaryDetail/deleteSysDictionaryDetail?id=1
func (h *DictionaryHandler) DeleteDictionaryDetail(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "id is required", nil))
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}

	if err := h.service.DeleteDictionaryDetail(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, nil, "删除成功")
}

// UpdateDictionaryDetail handles PUT /sysDictionaryDetail/updateSysDictionaryDetail
func (h *DictionaryHandler) UpdateDictionaryDetail(c *gin.Context) {
	var req UpdateDictionaryDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.UpdateDictionaryDetail(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, result, "更新成功")
}

// FindDictionaryDetail handles GET /sysDictionaryDetail/findSysDictionaryDetail?id=1
func (h *DictionaryHandler) FindDictionaryDetail(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "id is required", nil))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}

	result, err := h.service.GetDictionaryDetail(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, result, "查询成功")
}

// GetDictionaryDetailList handles GET /sysDictionaryDetail/getSysDictionaryDetailList
func (h *DictionaryHandler) GetDictionaryDetailList(c *gin.Context) {
	var req DictionaryDetailListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListDictionaryDetails(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, gin.H{
		"list":  result.Items,
		"total": result.Total,
	}, "查询成功")
}

// BatchGetDicts handles GET /sysDictionary/batch?codes=is_online,op_state,...
// Returns multiple dictionaries in a single request to reduce HTTP round-trips.
func (h *DictionaryHandler) BatchGetDicts(c *gin.Context) {
	codesStr := c.Query("codes")
	if codesStr == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "codes parameter is required", nil))
		return
	}

	codes := splitAndTrim(codesStr, ",")
	if len(codes) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "at least one valid code is required", nil))
		return
	}

	dicts, err := h.service.BatchGetDicts(c.Request.Context(), codes)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, gin.H{"dicts": dicts}, "查询成功")
}

// splitAndTrim splits a string by sep and trims each part.
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
	 trimmed := trimString(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by sep without importing strings.
func splitString(s, sep string) []string {
	if sep == "" {
		return nil
	}
	result := make([]string, 0)
	start := 0
	for {
		idx := indexOf(s, sep, start)
		if idx == -1 {
			result = append(result, s[start:])
			break
		}
		result = append(result, s[start:idx])
		start = idx + len(sep)
	}
	return result
}

// indexOf returns the index of sep in s starting from start, or -1 if not found.
func indexOf(s, sep string, start int) int {
	if len(sep) == 0 || start > len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(sep); i++ {
		match := true
		for j := 0; j < len(sep); j++ {
			if i+j >= len(s) || s[i+j] != sep[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// trimString removes leading and trailing whitespace.
func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && isSpace(rune(s[start])) {
		start++
	}
	for end > start && isSpace(rune(s[end-1])) {
		end--
	}
	return s[start:end]
}

// isSpace checks if a rune is a whitespace character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
