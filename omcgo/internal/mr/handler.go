package mr

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/compress"
	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for MR data.
type Handler struct {
	store           MRStore
	indRepo         IndicatorRepository
	mapRepo         MappingRepository
	minioClient     *minio.Client
	bucket          string
	productResolver ProductPatternResolver // #602; nil-safe (product_id 过滤参数被忽略)
	logger          *zap.Logger
}

// ProductPatternResolver 把 product_id 解析为该产品的 product_class 模式字面量集合。
// 由 cmd/app/provider 将 *product.Registry 以接口注入，避免 mr 包直接依赖 product 包。
type ProductPatternResolver interface {
	GetPatternsByProductID(ctx context.Context, productID uuid.UUID) ([]string, error)
}

// NewHandler creates a new MR handler.
func NewHandler(store MRStore, indRepo IndicatorRepository, mapRepo MappingRepository, minioClient *minio.Client, bucket string, logger *zap.Logger) *Handler {
	return &Handler{store: store, indRepo: indRepo, mapRepo: mapRepo, minioClient: minioClient, bucket: bucket, logger: logger}
}

// SetProductPatternResolver 装配「产品名称下拉」过滤能力。未装配时 product_id 查询参数被忽略。
func (h *Handler) SetProductPatternResolver(r ProductPatternResolver) {
	h.productResolver = r
}

// RegisterRoutes registers MR API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mrGroup := rg.Group("/mr")
	{
		mrGroup.GET("/files", h.ListFiles)
		mrGroup.GET("/files/devices", h.ListFileDevices) // 按设备聚合，给 File Management MR Tab 用
		mrGroup.GET("/files/:id/download", h.DownloadFile)
		mrGroup.POST("/files/batch-delete", h.BatchDeleteFiles) // 按 SN 批量删除（PG + MinIO）
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
// 该设备 mr_files 起止 collect_time + 文件数 + 站名/产品类。
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
	if sn := c.Query("site_name"); sn != "" {
		filter.SiteName = &sn
	}
	if pc := c.Query("product_class"); pc != "" {
		filter.ProductClass = &pc
	}
	if raw := strings.TrimSpace(c.Query("product_id")); raw != "" && h.productResolver != nil {
		pid, perr := uuid.Parse(raw)
		if perr != nil {
			coreerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid product_id: %w", perr))
			return
		}
		patterns, rerr := h.productResolver.GetPatternsByProductID(c.Request.Context(), pid)
		if rerr != nil {
			coreerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("resolve product patterns: %w", rerr))
			return
		}
		if len(patterns) == 0 {
			response.OK(c, model.NewListResponse[MRFileDeviceAggregate](nil, 0, filter.Page, filter.PageSize))
			return
		}
		filter.ProductClasses = patterns
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

	// issue #321：入库后原始 MR XML 被 gzip 压缩回写 MinIO，但对象键 / file_name 仍是 .xml。
	// 嗅探 gzip 魔数：压缩内容补 .gz 下载名（拿到可正常解压的 .xml.gz），明文原样透传。
	// 不设 Content-Encoding: gzip，避免浏览器自动解压与 .gz 文件名矛盾。
	filename := fileInfo.FileName
	contentType := "application/xml"
	br := bufio.NewReader(obj)
	if head, _ := br.Peek(2); compress.IsGzip(head) {
		if !strings.HasSuffix(strings.ToLower(filename), ".gz") {
			filename += ".gz"
		}
		contentType = "application/gzip"
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", contentType)
	c.DataFromReader(http.StatusOK, stat.Size, contentType, br, nil)
}

// BatchDeleteFilesRequest body.
type BatchDeleteFilesRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// BatchDeleteFilesResponse 报告每条结果（与 license/pm batch-delete 对齐）。
type BatchDeleteFilesResponse struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
}

// BatchDeleteFiles handles POST /mr/files/batch-delete.
// 每个 SN → 列文件 → 删 MinIO 对象 → 删 PG 行。MinIO 删失败也继续删 PG
// （MinIO 残留靠 retention policy 兜底，比保留孤儿 PG 行更干净）。
func (h *Handler) BatchDeleteFiles(c *gin.Context) {
	var req BatchDeleteFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	succeeded := make([]string, 0, len(req.SerialNumbers))
	failed := make([]string, 0)
	for _, sn := range req.SerialNumbers {
		files, err := h.store.ListFilesBySN(c.Request.Context(), sn)
		if err != nil {
			h.logger.Warn("mr batch delete: list files failed",
				zap.String("sn", sn), zap.Error(err))
			failed = append(failed, sn)
			continue
		}
		for _, f := range files {
			if rmErr := h.minioClient.RemoveObject(c.Request.Context(), h.bucket, f.MinioPath, minio.RemoveObjectOptions{}); rmErr != nil {
				h.logger.Warn("mr batch delete: minio remove failed",
					zap.String("sn", sn), zap.String("path", f.MinioPath), zap.Error(rmErr))
			}
		}
		if _, err := h.store.DeleteFilesBySN(c.Request.Context(), sn); err != nil {
			h.logger.Warn("mr batch delete: pg delete failed",
				zap.String("sn", sn), zap.Error(err))
			failed = append(failed, sn)
			continue
		}
		succeeded = append(succeeded, sn)
	}
	response.OK(c, BatchDeleteFilesResponse{Succeeded: succeeded, Failed: failed})
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
