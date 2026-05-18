package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler provides HTTP handlers for license management REST API.
type Handler struct {
	service     *Service
	logger      *zap.Logger
	logWriter   LogWriter            // T-0100-P0: 审计日志写入；nil 时退化为 NoopLogWriter
	logRepo     LicenseLogRepository // T-0100-P1: GET /licenses/logs 端点读侧；nil 时返 503
	sigVerifier *SignatureVerifier   // T-0100-P4-C: OEM 公钥验签；nil 时退化为 P3 stub 行为
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

// SetSignatureVerifier 注入 SignatureVerifier（T-0100-P4-C）。nil 等价于不开启
// 强校验（保留 P3 stub "unverified" 行为，让 dev / 单元测试不挂依赖）。
func (h *Handler) SetSignatureVerifier(v *SignatureVerifier) {
	h.sigVerifier = v
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
	licenses.GET("/export", h.ExportAll) // T-0100-P4-A：全量 active license CSV
	licenses.POST("/activate", h.Activate)
	licenses.POST("/import", h.Import)

	licenses.GET("", h.List)
	licenses.GET("/:id", h.GetByID)
	licenses.GET("/:id/logs", h.GetLicenseLogs)     // T-0100-P1：单 license 审计日志（详情抽屉）
	licenses.GET("/:id/export", h.ExportByID)       // T-0100-P4-A：单条 PDF 导出
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
			commonerrors.NewBusinessError(global.ErrCodeLicenseQuotaLoad, "failed to load license quota", err))
		return
	}
	response.OK(c, q)
}

// ---- Request types ----

// ActivateRequest defines the request body for activating a license.
//
// Force=true 让 Activate 在同 (device_type, region) 维度冲突时跳过 409，
// 自动 revoke 旧 active 后再激活新 license（前端 Modal 二次确认通过后调用）。
type ActivateRequest struct {
	LicenseCode string `json:"license_code" binding:"required"`
	Force       bool   `json:"force"`
}

// ActivateConflictResponse 是 Activate 返回 409（同维度冲突）时的响应体。
//
// 前端据此弹 Modal 列出旧 active license（license_code + license_name + max_devices），
// 用户确认后再次 POST /licenses/activate { force: true }。
type ActivateConflictResponse struct {
	Conflict          string     `json:"conflict"` // 固定 "same_dimension_active"
	DeviceType        *string    `json:"device_type"`
	Region            *string    `json:"region"`
	ConflictingActive []*License `json:"conflicting_active"`
	Hint              string     `json:"hint"`
}

// ImportResponse 是 POST /licenses/import 的响应体。
//
// SignatureStatus（T-0100-P3 / Q4=B）：MVP 暂未接入 OEM 公钥强校验，所有
// import 默认 signature_status='unverified' + 后端写一条 zap.Warn；前端用此
// 字段在导入成功 Modal 上显示警告 Tag，提示用户后续 P4-C 阶段会强校验。
type ImportResponse struct {
	License         *License `json:"license"`
	SignatureStatus string   `json:"signature_status"` // "verified" | "unverified" | "invalid"
	SignatureNote   string   `json:"signature_note,omitempty"`
}

// ImportRequest defines the request body for importing a new license.
//
// SignedLicenseJSON（T-0100-P4-C）：可选，原始签名 license JSON 文件内容。
// 当存在时：
//   - 用 SignatureVerifier 做 RSA-PSS 验证（strict 模式下未通过直接拒绝）
//   - mergeFromSignedJSON 把 JSON 里的 license 字段填到 req 中为空的位置
//     （操作员显式传入的字段不被覆盖；该设计支持"上传签名文件后用户在 UI
//     再编辑某些字段"的场景）
//
// 必填字段（LicenseName/LicenseCode/ProductName/IssueDate）走**后置 validate**，
// 不用 binding:"required"——因为前端 importSignedLicense 可能只传 SignedLicenseJSON
// 而不传结构化字段，required 标签会让 bind 阶段直接 400。validate 在 merge 后
// 调用，能命中 signed_license_json 里解出的字段。
type ImportRequest struct {
	LicenseName       string          `json:"license_name"`
	LicenseCode       string          `json:"license_code"`
	ProductName       string          `json:"product_name"`
	LicenseType       LicenseType     `json:"license_type"`
	Status            LicenseStatus   `json:"status"`
	MaxDevices        int             `json:"max_devices"`
	UsedDevices       int             `json:"used_devices"`
	Features          json.RawMessage `json:"features"`
	IssueDate         time.Time       `json:"issue_date"`
	ExpiryDate        *time.Time      `json:"expiry_date"`
	Licensor          *string         `json:"licensor"`
	DeviceType        *string         `json:"device_type"`
	Region            *string         `json:"region"`
	Notes             *string         `json:"notes"`
	SignedLicenseJSON string          `json:"signed_license_json,omitempty"`
}

// mergeFromSignedJSON 解析 SignedLicenseJSON 字段并把解析结果填入 req 中
// 为空的字段（操作员显式传入的字段保留不被覆盖）。
//
// SignedLicenseJSON 为空时直接返 nil（无操作）；解析失败返 error，调用方
// 应返 400 让前端看清根因。
func (req *ImportRequest) mergeFromSignedJSON() error {
	if req.SignedLicenseJSON == "" {
		return nil
	}
	var p ImportRequest
	if err := json.Unmarshal([]byte(req.SignedLicenseJSON), &p); err != nil {
		return fmt.Errorf("parse signed_license_json: %w", err)
	}
	if req.LicenseName == "" {
		req.LicenseName = p.LicenseName
	}
	if req.LicenseCode == "" {
		req.LicenseCode = p.LicenseCode
	}
	if req.ProductName == "" {
		req.ProductName = p.ProductName
	}
	if req.LicenseType == "" {
		req.LicenseType = p.LicenseType
	}
	if req.Status == "" {
		req.Status = p.Status
	}
	if req.MaxDevices == 0 {
		req.MaxDevices = p.MaxDevices
	}
	if req.UsedDevices == 0 {
		req.UsedDevices = p.UsedDevices
	}
	if len(req.Features) == 0 {
		req.Features = p.Features
	}
	if req.IssueDate.IsZero() {
		req.IssueDate = p.IssueDate
	}
	if req.ExpiryDate == nil {
		req.ExpiryDate = p.ExpiryDate
	}
	if req.Licensor == nil {
		req.Licensor = p.Licensor
	}
	if req.DeviceType == nil {
		req.DeviceType = p.DeviceType
	}
	if req.Region == nil {
		req.Region = p.Region
	}
	if req.Notes == nil {
		req.Notes = p.Notes
	}
	return nil
}

// validate 检查必填字段；缺任一返 error，调用方返 400。
//
// 替代 binding:"required"——后者会在 bind 阶段直接拒，让前端只传
// signed_license_json 的合法场景失败。validate 在 mergeFromSignedJSON 之后
// 调用，能识别 signed JSON 中解出的字段。
func (req *ImportRequest) validate() error {
	if req.LicenseName == "" {
		return fmt.Errorf("license_name is required")
	}
	if req.LicenseCode == "" {
		return fmt.Errorf("license_code is required")
	}
	if req.ProductName == "" {
		return fmt.Errorf("product_name is required")
	}
	if req.IssueDate.IsZero() {
		return fmt.Errorf("issue_date is required")
	}
	return nil
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
			commonerrors.NewBusinessError(global.ErrCodeLicenseLogsServiceUnavail, "license logs service not configured", nil))
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
			commonerrors.NewBusinessError(global.ErrCodeLicenseLogsListFailed, "failed to list license logs", err))
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
			commonerrors.NewBusinessError(global.ErrCodeLicenseLogsServiceUnavail, "license logs service not configured", nil))
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
			commonerrors.NewBusinessError(global.ErrCodeLicenseLogsByIDFailed, "failed to list license logs by id", err))
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
//
// T-0100-P3 / PRD §5.3.2 同维度冲突流程：
//   - force=false（默认）：service 返回 SameDimensionConflictError → 409 +
//     ActivateConflictResponse（含旧 active 列表），前端弹 Modal 二次确认。
//   - force=true：跳过冲突预检，自动 revoke 同维度旧 active 后激活新 license；
//     每条 auto-revoke 写一条 license_log（log_type=auto_revoke_by_activate），
//     再写新 license 的 activate 日志。
func (h *Handler) Activate(c *gin.Context) {
	var req ActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	actor := actorIDFromContext(c)
	result, err := h.service.Activate(c.Request.Context(), req.LicenseCode, req.Force)
	if err != nil {
		// 同维度冲突：返回 409 + 旧 active 列表，前端弹确认 Modal；不写 failed
		// 日志（属正常预检反馈，非真失败）。
		var conflict *SameDimensionConflictError
		if errors.As(err, &conflict) {
			c.AbortWithStatusJSON(http.StatusConflict, ActivateConflictResponse{
				Conflict:          "same_dimension_active",
				DeviceType:        conflict.DeviceType,
				Region:            conflict.Region,
				ConflictingActive: conflict.Conflicting,
				Hint:              "same (device_type, region) already has active license; resubmit with force=true to auto-revoke and activate",
			})
			return
		}

		h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
			LogType:     LogTypeActivate,
			ActorUserID: actor,
			Result:      LogResultFailed,
			Details: map[string]any{
				"summary":      "activate license failed",
				"license_code": req.LicenseCode,
				"force":        req.Force,
				"err":          err.Error(),
			},
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// 自动 revoke 同维度旧 active：每条单独写一条 auto_revoke_by_activate 日志，
	// actor 与 force activate 同一用户。
	for _, old := range result.AutoRevoked {
		oldID := old.ID
		h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
			LicenseID:   &oldID,
			LogType:     LogTypeAutoRevokeByActivate,
			ActorUserID: actor,
			Result:      LogResultSuccess,
			Details: map[string]any{
				"summary":               "auto-revoked by force activate",
				"revoked_license_code":  old.LicenseCode,
				"revoked_license_name":  old.LicenseName,
				"triggered_by_activate": req.LicenseCode,
			},
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
	}

	// 主激活成功日志
	lic := result.Activated
	licID := lic.ID
	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LicenseID:   &licID,
		LogType:     LogTypeActivate,
		ActorUserID: actor,
		Result:      LogResultSuccess,
		Details: map[string]any{
			"summary":            "license activated",
			"license_code":       lic.LicenseCode,
			"license_name":       lic.LicenseName,
			"force":              req.Force,
			"auto_revoked_count": len(result.AutoRevoked),
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

// ExportByID handles GET /api/v1/licenses/:id/export?format=pdf|json.
//
// T-0100-P4-A / PRD §5.3.4：单条 license 导出。format=pdf 返 application/pdf
// 字节流 + Content-Disposition attachment；format=json 返 application/json。
// 成功时按 query_detail 写一条审计日志（PRD §5.4.1 列出 query_detail 是敏感读
// 操作之一，导出比纯查询更敏感，复用同 log_type）。
func (h *Handler) ExportByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	format := c.DefaultQuery("format", "pdf")
	if format != "pdf" && format != "json" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(global.ErrCodeLicenseExportFormatInvalid, "format must be pdf or json", commonerrors.ErrInvalidInput))
		return
	}

	lic, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// 收集近 30 天审计 + 当前 quota（缺一不影响导出主体）。
	var logs []LicenseLog
	if h.logRepo != nil {
		ll, lerr := h.logRepo.ListByLicense(c.Request.Context(), id, 50)
		if lerr != nil {
			h.logger.Warn("list license logs for export degraded",
				zap.String("license_id", id.String()), zap.Error(lerr))
		} else {
			logs = ll
		}
	}
	quota, qerr := h.service.Quota(c.Request.Context())
	if qerr != nil {
		h.logger.Warn("get quota for export degraded", zap.Error(qerr))
		quota = nil
	}

	actor := actorIDFromContext(c)
	username := ""
	if v, ok := c.Get("username"); ok {
		if s, ok2 := v.(string); ok2 {
			username = s
		}
	}

	exportCtx := ExportContext{
		License:     lic,
		Quota:       quota,
		RecentLogs:  logs,
		GeneratedAt: nowFunc(),
		GeneratedBy: username,
	}

	// 审计：query_detail 类型 + details.export_format 用于后续追溯
	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LicenseID:   &id,
		LogType:     LogTypeQueryDetail,
		ActorUserID: actor,
		Result:      LogResultSuccess,
		Details: map[string]any{
			"summary":       "license exported",
			"export_format": format,
			"license_code":  lic.LicenseCode,
		},
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	switch format {
	case "pdf":
		filename := fmt.Sprintf("license-%s.pdf", lic.LicenseCode)
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		if err := WriteSingleLicensePDF(c.Writer, exportCtx); err != nil {
			h.logger.Error("write license PDF failed",
				zap.String("license_id", id.String()), zap.Error(err))
			// 已 flush 部分字节，无法回 5xx；仅记录。
		}
	case "json":
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition",
			fmt.Sprintf(`attachment; filename="license-%s.json"`, lic.LicenseCode))
		c.JSON(http.StatusOK, gin.H{
			"license":      lic,
			"quota":        quota,
			"recent_logs":  logs,
			"generated_at": exportCtx.GeneratedAt,
			"generated_by": valueOrSystem(username),
		})
	}
}

// ExportAll handles GET /api/v1/licenses/export?format=csv.
//
// T-0100-P4-A / PRD §5.3.4：全量 active license CSV 汇总；UTF-8 BOM 头方便
// Excel 直接打开。仅支持 csv 格式（PDF 全量没意义；JSON 全量去 GET /licenses?status=active）。
func (h *Handler) ExportAll(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	if format != "csv" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(global.ErrCodeLicenseBulkExportFormatInvalid, "format must be csv for bulk export", commonerrors.ErrInvalidInput))
		return
	}

	licenses, err := h.service.ListActiveForExport(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	actor := actorIDFromContext(c)
	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LogType:     LogTypeQueryDetail,
		ActorUserID: actor,
		Result:      LogResultSuccess,
		Details: map[string]any{
			"summary":       "bulk license export",
			"export_format": "csv",
			"row_count":     len(licenses),
		},
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	filename := fmt.Sprintf("licenses-active-%s.csv", nowFunc().Format("20060102"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := WriteAllActiveCSV(c.Writer, licenses); err != nil {
		h.logger.Error("write bulk license CSV failed", zap.Error(err))
	}
}

func valueOrSystem(s string) string {
	if s == "" {
		return "system"
	}
	return s
}

// Import handles POST /api/v1/licenses/import.
func (h *Handler) Import(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 用户决策 2026-05-18：前端 importSignedLicense 只传 signed_license_json
	// 字段（不传 license_name/code/product_name/issue_date 结构化字段）。先合并
	// signed JSON 里的字段到 req 中**为空**的位置（操作员显式传入的不被覆盖），
	// 再做必填字段校验。
	if err := req.mergeFromSignedJSON(); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(global.ErrCodeLicenseSignatureVerifyFailed,
				"parse signed_license_json: "+err.Error(), commonerrors.ErrInvalidInput))
		return
	}
	if err := req.validate(); err != nil {
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

	// T-0100-P4-C：当 sigVerifier 注入时，对 SignedLicenseJSON 字段做 RSA-PSS 验签。
	// strict=true：unverified/invalid 直接 400 拒绝；strict=false：放过但反映状态。
	// 未注入 verifier（nil）或 SignedLicenseJSON 空 → 退化为 P3 stub 行为（unverified）。
	var sigStatus SignatureStatus
	var sigNote string
	if h.sigVerifier != nil && req.SignedLicenseJSON != "" {
		var sigErr error
		sigStatus, sigNote, sigErr = h.sigVerifier.VerifyLicenseJSON([]byte(req.SignedLicenseJSON))
		if sigErr != nil {
			// strict 模式：写 failed 审计 + 400 拒绝
			h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
				LogType:     LogTypeImport,
				ActorUserID: actor,
				Result:      LogResultFailed,
				Details: map[string]any{
					"summary":          "import rejected by strict signature verification",
					"license_code":     req.LicenseCode,
					"signature_status": string(sigStatus),
					"signature_note":   sigNote,
				},
				ClientIP:  c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
			})
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				commonerrors.NewBusinessError(global.ErrCodeLicenseSignatureVerifyFailed,
					"license signature verification failed (strict mode): "+sigNote,
					commonerrors.ErrInvalidInput))
			return
		}
	} else {
		// 退化：保持 P3 stub 行为（让没接 verifier 的部署仍可导入）。
		sigStatus, sigNote = VerifySignature(nil)
	}
	if sigStatus != SignatureVerified {
		h.logger.Warn("license import signature unverified",
			zap.String("license_code", req.LicenseCode),
			zap.String("signature_status", string(sigStatus)),
			zap.String("signature_note", sigNote),
		)
	}

	created, err := h.service.Import(c.Request.Context(), lic)
	if err != nil {
		h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
			LogType:     LogTypeImport,
			ActorUserID: actor,
			Result:      LogResultFailed,
			Details: map[string]any{
				"summary":          "import license failed",
				"license_code":     req.LicenseCode,
				"signature_status": string(sigStatus),
				"err":              err.Error(),
			},
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// 成功导入：日志 result=warning（签名未验证）/ success（已验证），兼容
	// monitor / 合规审计的"敏感事件"过滤。
	logResult := LogResultSuccess
	if sigStatus != SignatureVerified {
		logResult = LogResultWarning
	}
	createdID := created.ID
	h.logWriter.Write(c.Request.Context(), LicenseLogEntry{
		LicenseID:   &createdID,
		LogType:     LogTypeImport,
		ActorUserID: actor,
		Result:      logResult,
		Details: map[string]any{
			"summary":          "license imported",
			"license_code":     created.LicenseCode,
			"license_name":     created.LicenseName,
			"license_type":     string(created.LicenseType),
			"max_devices":      created.MaxDevices,
			"signature_status": string(sigStatus),
			"signature_note":   sigNote,
		},
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	response.OKWithStatus(c, http.StatusCreated, ImportResponse{
		License:         created,
		SignatureStatus: string(sigStatus),
		SignatureNote:   sigNote,
	})
}
