package license

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler provides HTTP handlers for license management REST API.
type Handler struct {
	service   *Service
	logger    *zap.Logger
	logWriter LogWriter            // T-0100-P0: 审计日志写入；nil 时退化为 NoopLogWriter
	logRepo   LicenseLogRepository // T-0100-P1: GET /licenses/logs 端点读侧；nil 时返 503
}

// NewHandler creates a new license Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service:   service,
		logger:    logger.Named("license-handler"),
		logWriter: NoopLogWriter{}, // 默认 noop；DI 通过 SetLogWriter 注入真实实现
	}
}

// SetLogWriter 注入真实的 LogWriter（T-0100-P0）。
//
// 在 cmd/app/provider/modules.go 内 license 模块装配时调用。bootstrap 早期 / 测试
// 不注入时保留 NoopLogWriter，handler 写日志路径无 nil 风险。
func (h *Handler) SetLogWriter(w LogWriter) {
	if w == nil {
		w = NoopLogWriter{}
	}
	h.logWriter = w
}

// SetLogRepo 注入 LicenseLogRepository（T-0100-P1），供 GET /licenses/logs +
// GET /licenses/:id/logs 端点读取审计日志数据。
func (h *Handler) SetLogRepo(repo LicenseLogRepository) {
	h.logRepo = repo
}

// actorIDFromContext 从 gin.Context 取当前用户 UUID（admin middleware 设置）。
// 找不到 / 类型错误返回 nil（后续作为 system 操作处理）。
func actorIDFromContext(c *gin.Context) *uuid.UUID {
	v, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}

// RegisterRoutes registers license routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	licenses := rg.Group("/licenses")

	// Register specific routes BEFORE the :id route to avoid conflicts.
	licenses.GET("/summary", h.GetSummary)
	licenses.GET("/quota", h.GetQuota)
	licenses.GET("/logs", h.ListLogs) // T-0100-P1：全量审计日志（分页 + 过滤）
	licenses.POST("/activate", h.Activate)
	licenses.POST("/import", h.Import)

	licenses.GET("", h.List)
	licenses.GET("/:id", h.GetByID)
	licenses.GET("/:id/logs", h.GetLicenseLogs) // T-0100-P1：单 license 审计日志（详情抽屉）
	licenses.POST("/:id/revoke", h.Revoke)
}

// GetQuota returns the current license enforcement state.
//
//	GET /api/v1/licenses/quota
//
// Response: 200 + Quota JSON; HasActiveLicense=false when no active license.
func (h *Handler) GetQuota(c *gin.Context) {
	q, err := h.service.Quota(c.Request.Context())
	if err != nil {
		h.logger.Error("get license quota failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			commonerrors.NewBusinessError(9103, "failed to load license quota", err))
		return
	}
	response.OK(c, q)
}

// ---- Request types ----

// ActivateRequest defines the request body for activating a license.
type ActivateRequest struct {
	LicenseCode string `json:"license_code" binding:"required"`
}

// ImportRequest defines the request body for importing a new license.
type ImportRequest struct {
	LicenseName string          `json:"license_name" binding:"required"`
	LicenseCode string          `json:"license_code" binding:"required"`
	ProductName string          `json:"product_name" binding:"required"`
	LicenseType LicenseType     `json:"license_type"`
	Status      LicenseStatus   `json:"status"`
	MaxDevices  int             `json:"max_devices"`
	UsedDevices int             `json:"used_devices"`
	Features    json.RawMessage `json:"features"`
	IssueDate   time.Time       `json:"issue_date" binding:"required"`
	ExpiryDate  *time.Time      `json:"expiry_date"`
	Licensor    *string         `json:"licensor"`
	DeviceType  *string         `json:"device_type"`
	Region      *string         `json:"region"`
	Notes       *string         `json:"notes"`
}

// ---- Handlers ----

// List handles GET /api/v1/licenses.
func (h *Handler) List(c *gin.Context) {
	filter := LicenseFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := LicenseStatus(status)
		filter.Status = &s
	}
	if licenseType := c.Query("license_type"); licenseType != "" {
		t := LicenseType(licenseType)
		filter.LicenseType = &t
	}
	if deviceType := c.Query("device_type"); deviceType != "" {
		filter.DeviceType = &deviceType
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
}

// ListLogs handles GET /api/v1/licenses/logs（T-0100-P1）。
//
// 支持的 query 参数（任选）：
//   - page / page_size       分页（默认 page=1, page_size=20，上限 200）
//   - license_id             单 license UUID
//   - log_type               重复参数支持多值，如 ?log_type=import&log_type=activate
//   - result                 重复参数支持多值
//   - actor_user_id          单用户 UUID
//   - start_time / end_time  RFC3339 时间窗
//   - search                 details JSONB::text ILIKE 关键字（模糊搜索）
//
// 响应：标准 envelope + ListResponse[LicenseLog]，按 created_at DESC 排序。
func (h *Handler) ListLogs(c *gin.Context) {
	if h.logRepo == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			commonerrors.NewBusinessError(9104, "license logs service not configured", nil))
		return
	}

	filter := LicenseLogFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if v := c.Query("license_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.LicenseID = &id
	}
	if v := c.Query("actor_user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.ActorUserID = &id
	}
	if vs := c.QueryArray("log_type"); len(vs) > 0 {
		filter.LogTypes = make([]LogType, 0, len(vs))
		for _, v := range vs {
			if v != "" {
				filter.LogTypes = append(filter.LogTypes, LogType(v))
			}
		}
	}
	if vs := c.QueryArray("result"); len(vs) > 0 {
		filter.Results = make([]LogResult, 0, len(vs))
		for _, v := range vs {
			if v != "" {
				filter.Results = append(filter.Results, LogResult(v))
			}
		}
	}
	if v := c.Query("start_time"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.StartTime = &t
	}
	if v := c.Query("end_time"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.EndTime = &t
	}
	if v := c.Query("search"); v != "" {
		filter.Search = &v
	}

	result, err := h.logRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list license logs failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			commonerrors.NewBusinessError(9105, "failed to list license logs", err))
		return
	}
	response.OK(c, result)
}

// GetLicenseLogs handles GET /api/v1/licenses/:id/logs（T-0100-P1）。
//
// 取单 license 最近 N 条审计日志（详情抽屉 "最近操作" 用）。
//   - query limit：默认 10，上限 100
func (h *Handler) GetLicenseLogs(c *gin.Context) {
	if h.logRepo == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			commonerrors.NewBusinessError(9104, "license logs service not configured", nil))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	logs, err := h.logRepo.ListByLicense(c.Request.Context(), id, limit)
	if err != nil {
		h.logger.Error("list license logs by id failed",
			zap.String("license_id", id.String()),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			commonerrors.NewBusinessError(9106, "failed to list license logs by id", err))
		return
	}

	response.OK(c, gin.H{"items": logs, "total": len(logs)})
}

// GetByID handles GET /api/v1/licenses/:id.
func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	lic, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, lic)
}

// GetSummary handles GET /api/v1/licenses/summary.
func (h *Handler) GetSummary(c *gin.Context) {
	summary, err := h.service.Summary(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, summary)
}

// Activate handles POST /api/v1/licenses/activate.
func (h *Handler) Activate(c *gin.Context) {
	var req ActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	actor := actorIDFromContext(c)
	lic, err := h.service.Activate(c.Request.Context(), req.LicenseCode)
	if err != nil {
		// failed audit
		h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
			LogType:     LogTypeActivate,
			ActorUserID: actor,
			Result:      LogResultFailed,
			Details: map[string]any{
				"summary":      "activate license failed",
				"license_code": req.LicenseCode,
				"err":          err.Error(),
			},
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// success audit
	licID := lic.ID
	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LicenseID:   &licID,
		LogType:     LogTypeActivate,
		ActorUserID: actor,
		Result:      LogResultSuccess,
		Details: map[string]any{
			"summary":      "license activated",
			"license_code": lic.LicenseCode,
			"license_name": lic.LicenseName,
		},
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	response.OK(c, lic)
}

// Revoke handles POST /api/v1/licenses/:id/revoke.
func (h *Handler) Revoke(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	actor := actorIDFromContext(c)
	if err := h.service.Revoke(c.Request.Context(), id); err != nil {
		h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
			LicenseID:   &id,
			LogType:     LogTypeRevoke,
			ActorUserID: actor,
			Result:      LogResultFailed,
			Details: map[string]any{
				"summary": "revoke license failed",
				"err":     err.Error(),
			},
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LicenseID:   &id,
		LogType:     LogTypeRevoke,
		ActorUserID: actor,
		Result:      LogResultSuccess,
		Details: map[string]any{
			"summary": "license revoked",
		},
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	response.OK(c, gin.H{"status": "revoked"})
}

// Import handles POST /api/v1/licenses/import.
func (h *Handler) Import(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	lic := &License{
		LicenseName: req.LicenseName,
		LicenseCode: req.LicenseCode,
		ProductName: req.ProductName,
		LicenseType: req.LicenseType,
		Status:      req.Status,
		MaxDevices:  req.MaxDevices,
		UsedDevices: req.UsedDevices,
		Features:    req.Features,
		IssueDate:   req.IssueDate,
		ExpiryDate:  req.ExpiryDate,
		Licensor:    req.Licensor,
		DeviceType:  req.DeviceType,
		Region:      req.Region,
		Notes:       req.Notes,
	}

	if lic.LicenseType == "" {
		lic.LicenseType = TypeSubscription
	}

	actor := actorIDFromContext(c)
	created, err := h.service.Import(c.Request.Context(), lic)
	if err != nil {
		h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
			LogType:     LogTypeImport,
			ActorUserID: actor,
			Result:      LogResultFailed,
			Details: map[string]any{
				"summary":      "import license failed",
				"license_code": req.LicenseCode,
				"err":          err.Error(),
			},
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	createdID := created.ID
	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LicenseID:   &createdID,
		LogType:     LogTypeImport,
		ActorUserID: actor,
		Result:      LogResultSuccess,
		Details: map[string]any{
			"summary":      "license imported",
			"license_code": created.LicenseCode,
			"license_name": created.LicenseName,
			"license_type": string(created.LicenseType),
			"max_devices":  created.MaxDevices,
		},
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	response.OKWithStatus(c, http.StatusCreated, created)
}
