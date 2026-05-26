package mr

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
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
		mrGroup.GET("/files/devices", h.ListFileDevices) // 按设备聚合，给 File Management MR Tab 用
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
	DeviceSN  string `form:"device_sn"`
	MRType    string `form:"mr_type"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) ListFiles(c *gin.Context) {
	var q fileQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := MRFileFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}
	if q.DeviceSN != "" {
		filter.DeviceSN = &q.DeviceSN
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
		coreerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// ListFileDevices 给 File Management → MR Tab 主列表用。每行 1 个设备 +
// 该设备 mr_files 起止 collect_time + 文件数。
func (h *Handler) ListFileDevices(c *gin.Context) {
	filter := MRFileDeviceFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if kw := c.Query("keyword"); kw != "" {
		filter.Keyword = &kw
	}
	result, err := h.store.ListFileDeviceAggregates(c.Request.Context(), filter)
	if err != nil {
		coreerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid file id")
		return
	}

	// Look up file by ID directly
	fileInfo, err := h.store.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "lookup failed")
		return
	}
	if fileInfo == nil {
		response.Fail(c, http.StatusNotFound, "file not found")
		return
	}

	obj, err := h.minioClient.GetObject(c.Request.Context(), h.bucket, fileInfo.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "download failed")
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "get file info failed")
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
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := MRRecordFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}
	if q.FileID != "" {
		id, err := uuid.Parse(q.FileID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid file_id")
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
		coreerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// ---- Indicator handlers ----

// ListIndicators handles GET /api/v1/mr/indicators.
func (h *Handler) ListIndicators(c *gin.Context) {
	filter := IndicatorFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
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
		coreerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// ListAllIndicators handles GET /api/v1/mr/indicators/all.
func (h *Handler) ListAllIndicators(c *gin.Context) {
	items, err := h.indRepo.ListAll(c.Request.Context())
	if err != nil {
		coreerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, items)
}

// GetIndicatorStats handles GET /api/v1/mr/indicators/:code/stats.
func (h *Handler) GetIndicatorStats(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.Fail(c, http.StatusBadRequest, "indicator code is required")
		return
	}

	// Verify indicator exists
	indicator, err := h.indRepo.GetByCode(c.Request.Context(), code)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "indicator not found")
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

	response.OK(c, gin.H{
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
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
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
		coreerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// UpdateMapping handles PUT /api/v1/mr/mappings/:id.
func (h *Handler) UpdateMapping(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
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
		coreerrors.AbortWithError(c, coreerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, mapping)
}

// ToggleMapping handles PUT /api/v1/mr/mappings/:id/toggle.
func (h *Handler) ToggleMapping(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req toggleMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.mapRepo.ToggleEnabled(c.Request.Context(), id, req.Enabled)
	if err != nil {
		coreerrors.AbortWithError(c, coreerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// exportRequest defines the request body for MR data export.
type exportRequest struct {
	DeviceID  string `json:"device_id"`
	MRType    string `json:"mr_type"`
	CellID    string `json:"cell_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Format    string `json:"format"` // "json" or "csv", default "json"
}

// ExportMRData handles POST /api/v1/mr/export.
func (h *Handler) ExportMRData(c *gin.Context) {
	var req exportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := MRRecordFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 10000},
	}
	if req.DeviceID != "" {
		id, err := uuid.Parse(req.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}
	if req.MRType != "" {
		filter.MRType = &req.MRType
	}
	if req.CellID != "" {
		filter.CellID = &req.CellID
	}
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.store.QueryRecords(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, fmt.Sprintf("query MR records: %v", err))
		return
	}

	if req.Format == "csv" {
		h.exportCSV(c, result.Items)
		return
	}

	// Default: JSON export
	c.Header("Content-Disposition", "attachment; filename=mr_export.json")
	response.OK(c, gin.H{
		"total":   result.Total,
		"records": result.Items,
	})
}

func (h *Handler) exportCSV(c *gin.Context, records []MRRecordEntry) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=mr_export.csv")
	c.Status(http.StatusOK)

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	// Write header
	_ = w.Write([]string{"time", "file_id", "device_id", "cell_id", "mr_type", "measurement_data"})

	for _, rec := range records {
		dataJSON, _ := json.Marshal(rec.MeasurementData)
		_ = w.Write([]string{
			rec.Time.Format(time.RFC3339),
			rec.FileID.String(),
			rec.DeviceID.String(),
			rec.CellID,
			rec.MRType,
			string(dataJSON),
		})
	}
}
