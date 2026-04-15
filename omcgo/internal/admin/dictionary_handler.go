package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "创建成功"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil, "msg": "删除成功"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "更新成功"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "查询成功"})
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  result,
			"total": len(result),
		},
		"msg": "查询成功",
	})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "创建成功"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil, "msg": "删除成功"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "更新成功"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": "查询成功"})
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  result.Items,
			"total": result.Total,
		},
		"msg": "查询成功",
	})
}
