package mr

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/common/model"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for MR data.
type Handler struct {
	store       MRStore
	indRepo     IndicatorRepository
	mapRepo     MappingRepository
	minioClient *minio.Client
	bucket      string
	logger      *zap.Logger
}

// NewHandler creates a new MR handler.
func NewHandler(store MRStore, indRepo IndicatorRepository, mapRepo MappingRepository, minioClient *minio.Client, bucket string, logger *zap.Logger) *Handler {
	return &Handler{store: store, indRepo: indRepo, mapRepo: mapRepo, minioClient: minioClient, bucket: bucket, logger: logger}
}

// RegisterRoutes registers MR API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mrGroup := rg.Group("/mr")
	{
		mrGroup.GET("/files", h.ListFiles)
		mrGroup.GET("/files/:id/download", h.DownloadFile)
		mrGroup.GET("/data", h.QueryData)

		mrGroup.GET("/indicators", h.ListIndicators)
		mrGroup.GET("/indicators/all", h.ListAllIndicators)
		mrGroup.GET("/indicators/:code/stats", h.GetIndicatorStats)
		mrGroup.GET("/mappings", h.ListMappings)
		mrGroup.PUT("/mappings/:id", h.UpdateMapping)
		mrGroup.PUT("/mappings/:id/toggle", h.ToggleMapping)
		mrGroup.POST("/export", h.ExportMRData)
	}
}

type fileQuery struct {
	DeviceID  string `form:"device_id"`
	MRType    string `form:"mr_type"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) ListFiles(c *gin.Context) {
	var q fileQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := MRFileFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
			return
		}
		filter.DeviceID = &id
	}
	if q.MRType != "" {
		filter.MRType = &q.MRType
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.store.ListFiles(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	// Look up file by ID directly
	fileInfo, err := h.store.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}
	if fileInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	obj, err := h.minioClient.GetObject(c.Request.Context(), h.bucket, fileInfo.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "download failed"})
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get file info failed"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+fileInfo.FileName)
	c.Header("Content-Type", "application/xml")
	c.DataFromReader(http.StatusOK, stat.Size, "application/xml", obj, nil)
}

type dataQuery struct {
	DeviceID string `form:"device_id"`
	FileID   string `form:"file_id"`
	MRType   string `form:"mr_type"`
	CellID   string `form:"cell_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) QueryData(c *gin.Context) {
	var q dataQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := MRRecordFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
			return
		}
		filter.DeviceID = &id
	}
	if q.FileID != "" {
		id, err := uuid.Parse(q.FileID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_id"})
			return
		}
		filter.FileID = &id
	}
	if q.MRType != "" {
		filter.MRType = &q.MRType
	}
	if q.CellID != "" {
		filter.CellID = &q.CellID
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.store.QueryRecords(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ---- Indicator handlers ----

// ListIndicators handles GET /api/v1/mr/indicators.
func (h *Handler) ListIndicators(c *gin.Context) {
	filter := IndicatorFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if category := c.Query("category"); category != "" {
		filter.Category = &category
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filter.Keyword = &keyword
	}

	result, err := h.indRepo.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListAllIndicators handles GET /api/v1/mr/indicators/all.
func (h *Handler) ListAllIndicators(c *gin.Context) {
	items, err := h.indRepo.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// GetIndicatorStats handles GET /api/v1/mr/indicators/:code/stats.
func (h *Handler) GetIndicatorStats(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indicator code is required"})
		return
	}

	// Verify indicator exists
	indicator, err := h.indRepo.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "indicator not found"})
		return
	}

	// Return placeholder stats based on indicator's range
	var minVal, maxVal float64
	if indicator.ValueRangeMin != nil {
		minVal = *indicator.ValueRangeMin
	}
	if indicator.ValueRangeMax != nil {
		maxVal = *indicator.ValueRangeMax
	}
	avg := (minVal + maxVal) / 2

	c.JSON(http.StatusOK, gin.H{
		"indicator_code": code,
		"avg":            avg,
		"min":            minVal,
		"max":            maxVal,
		"p50":            avg,
		"p95":            maxVal * 0.9,
		"sample_count":   0,
	})
}

// ---- Mapping handlers ----

type updateMappingRequest struct {
	DeviceSN         string  `json:"device_sn"`
	DeviceName       *string `json:"device_name"`
	CellID           string  `json:"cell_id"`
	CellName         *string `json:"cell_name"`
	Enabled          bool    `json:"enabled"`
	SamplingInterval int     `json:"sampling_interval"`
}

type toggleMappingRequest struct {
	Enabled bool `json:"enabled"`
}

// ListMappings handles GET /api/v1/mr/mappings.
func (h *Handler) ListMappings(c *gin.Context) {
	filter := MappingFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if deviceSN := c.Query("device_sn"); deviceSN != "" {
		filter.DeviceSN = &deviceSN
	}
	if enabled := c.Query("enabled"); enabled != "" {
		b := enabled == "true"
		filter.Enabled = &b
	}

	result, err := h.mapRepo.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// UpdateMapping handles PUT /api/v1/mr/mappings/:id.
func (h *Handler) UpdateMapping(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req updateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mapping := &MRDeviceMapping{
		ID:               id,
		DeviceSN:         req.DeviceSN,
		DeviceName:       req.DeviceName,
		CellID:           req.CellID,
		CellName:         req.CellName,
		Enabled:          req.Enabled,
		SamplingInterval: req.SamplingInterval,
	}
	if mapping.SamplingInterval == 0 {
		mapping.SamplingInterval = 15
	}

	if err := h.mapRepo.Update(c.Request.Context(), mapping); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mapping)
}

// ToggleMapping handles PUT /api/v1/mr/mappings/:id/toggle.
func (h *Handler) ToggleMapping(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req toggleMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.mapRepo.ToggleEnabled(c.Request.Context(), id, req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ExportMRData handles POST /api/v1/mr/export.
func (h *Handler) ExportMRData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"task_id": "export-placeholder",
		"status":  "pending",
	})
}
