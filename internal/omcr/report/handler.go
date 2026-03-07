package report

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

// Handler provides HTTP handlers for report management REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new report Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("report-handler"),
	}
}

// RegisterRoutes registers report routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	reports := rg.Group("/reports")

	defs := reports.Group("/definitions")
	defs.GET("", h.ListDefinitions)
	defs.POST("", h.CreateDefinition)
	defs.GET("/:id", h.GetDefinition)
	defs.PUT("/:id", h.UpdateDefinition)
	defs.DELETE("/:id", h.DeleteDefinition)

	records := reports.Group("/records")
	records.GET("", h.ListRecords)
	records.GET("/:id/download", h.DownloadRecord)

	reports.POST("/generate", h.GenerateReport)
	reports.GET("/sample-data", h.GetSampleData)
}

// ---- Definition request types ----

// CreateDefinitionRequest defines the request body for creating a report definition.
type CreateDefinitionRequest struct {
	ReportName     string       `json:"report_name" binding:"required"`
	ReportType     ReportType   `json:"report_type" binding:"required"`
	Description    string       `json:"description"`
	Format         []string     `json:"format"`
	Period         ReportPeriod `json:"period"`
	KPICodes       []string     `json:"kpi_codes"`
	DeviceGroups   []string     `json:"device_groups"`
	AutoGenerate   bool         `json:"auto_generate"`
	CronExpression string       `json:"cron_expression"`
	Status         ReportStatus `json:"status"`
	Creator        string       `json:"creator"`
}

// UpdateDefinitionRequest defines the request body for updating a report definition.
type UpdateDefinitionRequest struct {
	ReportName     string       `json:"report_name" binding:"required"`
	ReportType     ReportType   `json:"report_type" binding:"required"`
	Description    string       `json:"description"`
	Format         []string     `json:"format"`
	Period         ReportPeriod `json:"period"`
	KPICodes       []string     `json:"kpi_codes"`
	DeviceGroups   []string     `json:"device_groups"`
	AutoGenerate   bool         `json:"auto_generate"`
	CronExpression string       `json:"cron_expression"`
	Status         ReportStatus `json:"status"`
	Creator        string       `json:"creator"`
}

// GenerateReportRequest defines the request body for triggering report generation.
type GenerateReportRequest struct {
	DefinitionID string `json:"definition_id" binding:"required"`
	Period       string `json:"period"`
}

// ---- Definition handlers ----

// ListDefinitions handles GET /api/v1/reports/definitions.
func (h *Handler) ListDefinitions(c *gin.Context) {
	filter := DefinitionFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if reportType := c.Query("report_type"); reportType != "" {
		t := ReportType(reportType)
		filter.ReportType = &t
	}
	if status := c.Query("status"); status != "" {
		s := ReportStatus(status)
		filter.Status = &s
	}

	result, err := h.service.ListDefinitions(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateDefinition handles POST /api/v1/reports/definitions.
func (h *Handler) CreateDefinition(c *gin.Context) {
	var req CreateDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	def := &ReportDefinition{
		ReportName:     req.ReportName,
		ReportType:     req.ReportType,
		Description:    req.Description,
		Format:         req.Format,
		Period:         req.Period,
		KPICodes:       req.KPICodes,
		DeviceGroups:   req.DeviceGroups,
		AutoGenerate:   req.AutoGenerate,
		CronExpression: req.CronExpression,
		Status:         req.Status,
		Creator:        req.Creator,
	}

	created, err := h.service.CreateDefinition(c.Request.Context(), def)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetDefinition handles GET /api/v1/reports/definitions/:id.
func (h *Handler) GetDefinition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	def, err := h.service.GetDefinition(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, def)
}

// UpdateDefinition handles PUT /api/v1/reports/definitions/:id.
func (h *Handler) UpdateDefinition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	def := &ReportDefinition{
		ReportName:     req.ReportName,
		ReportType:     req.ReportType,
		Description:    req.Description,
		Format:         req.Format,
		Period:         req.Period,
		KPICodes:       req.KPICodes,
		DeviceGroups:   req.DeviceGroups,
		AutoGenerate:   req.AutoGenerate,
		CronExpression: req.CronExpression,
		Status:         req.Status,
		Creator:        req.Creator,
	}

	updated, err := h.service.UpdateDefinition(c.Request.Context(), id, def)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteDefinition handles DELETE /api/v1/reports/definitions/:id.
func (h *Handler) DeleteDefinition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteDefinition(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---- Record handlers ----

// ListRecords handles GET /api/v1/reports/records.
func (h *Handler) ListRecords(c *gin.Context) {
	filter := RecordFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if defID := c.Query("definition_id"); defID != "" {
		id, err := uuid.Parse(defID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DefinitionID = &id
	}
	if format := c.Query("format"); format != "" {
		filter.Format = &format
	}

	result, err := h.service.ListRecords(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateReport handles POST /api/v1/reports/generate.
func (h *Handler) GenerateReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	defID, err := uuid.Parse(req.DefinitionID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	record, err := h.service.Generate(c.Request.Context(), defID, req.Period)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, record)
}

// DownloadRecord handles GET /api/v1/reports/records/:id/download.
func (h *Handler) DownloadRecord(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	record, err := h.service.GetRecord(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if record.DownloadURL != "" {
		c.JSON(http.StatusOK, gin.H{
			"url":       record.DownloadURL,
			"file_name": record.ReportName + "." + record.Format,
		})
		return
	}

	// Placeholder if no download URL yet
	c.JSON(http.StatusOK, gin.H{
		"url":       "",
		"file_name": record.ReportName + "." + record.Format,
		"message":   "report is still generating or no file available",
	})
}

// GetSampleData handles GET /api/v1/reports/sample-data.
func (h *Handler) GetSampleData(c *gin.Context) {
	// Return placeholder sample data for report preview/charting
	sampleData := gin.H{
		"kpi_summary": gin.H{
			"rrc_succ_rate":  gin.H{"value": 97.8, "trend": 0.3, "status": "good"},
			"erab_succ_rate": gin.H{"value": 96.5, "trend": -0.2, "status": "good"},
			"ho_succ_rate":   gin.H{"value": 95.1, "trend": 0.5, "status": "good"},
			"dl_throughput":  gin.H{"value": 89.3, "unit": "Mbps", "trend": 2.1, "status": "good"},
			"ul_throughput":  gin.H{"value": 35.7, "unit": "Mbps", "trend": 0.8, "status": "good"},
			"radio_drop":    gin.H{"value": 0.8, "trend": -0.1, "status": "good"},
		},
		"alarm_summary": gin.H{
			"total_alarms": 650,
			"by_day":       []int{45, 52, 38, 61, 43, 29, 55},
			"by_severity":  gin.H{"critical": 12, "major": 89, "minor": 210, "warning": 339},
			"top5_types": []gin.H{
				{"name": "小区不可用", "count": 45},
				{"name": "S1链路中断", "count": 38},
				{"name": "射频单元故障", "count": 29},
				{"name": "GPS失锁", "count": 25},
				{"name": "温度过高", "count": 22},
			},
			"avg_clear_time": 42,
		},
		"device_summary": gin.H{
			"total_devices": 200,
			"online":        170,
			"offline":       30,
			"by_type":       gin.H{"eNB": 80, "gNB": 60, "CPE": 40, "eGW": 20},
			"by_region":     gin.H{"华北": 45, "华东": 60, "华南": 50, "西南": 25, "西北": 20},
		},
	}

	c.JSON(http.StatusOK, sampleData)
}
