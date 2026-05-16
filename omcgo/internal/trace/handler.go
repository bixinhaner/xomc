package trace

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler trace REST handler。
//
// 异步下载（M2-08）依赖一个 MinIO 客户端生成预签名 URL；未注入时端点仍工作但返回的 job
// 不会带 download_url，前端需走代理路径或后续轮询。
//
// L-8 修复：presignClient 应当用 PublicEndpoint 配置（minio.NewPresignClient 构造），
// 否则签出来的 URL host 是 docker 内部名（如 "minio:9000"），浏览器无法解析。
type Handler struct {
	service       *Service
	presignClient *minio.Client // 可 nil（M1 兼容 / 无 MinIO 部署）
	presignTTL    time.Duration
	logger        *zap.Logger
}

// NewHandler 构造函数。
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		service:    service,
		presignTTL: 30 * time.Minute,
		logger:     logger.Named("trace-handler"),
	}
}

// SetMinIO 注入 MinIO 预签名客户端（M2-08 异步下载预签名）。
// 期望传入 minio.NewPresignClient(cfg) 的返回值 — 用 PublicEndpoint 签 URL。
func (h *Handler) SetMinIO(client *minio.Client) {
	h.presignClient = client
}

// RegisterRoutes 在已含 /api/v1 与权限中间件的 RouterGroup 上挂载。
//
//	POST /trace/tasks
//	GET  /trace/tasks
//	GET  /trace/tasks/:id
//	POST /trace/tasks/:id/stop
//	GET  /trace/tasks/:id/messages
//	POST /trace/tasks/:id/export        — M1 同步小批量 XML 直接流式返回
//	GET  /trace/devices/:sn/active-task
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/trace")
	g.POST("/tasks", h.CreateTask)
	g.GET("/tasks", h.ListTasks)
	g.GET("/tasks/:id", h.GetTask)
	g.POST("/tasks/:id/stop", h.StopTask)
	g.GET("/tasks/:id/messages", h.ListMessages)
	g.GET("/tasks/:id/messages/:msgId/payload", h.GetMessagePayload)
	g.POST("/tasks/:id/export", h.ExportXML)
	g.GET("/exports/:jobId", h.GetExportJob)
	g.GET("/devices/:sn/active-task", h.GetActiveTaskBySN)
}

// CreateTask POST /trace/tasks
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	operatorCode, createdBy := h.identity(c)
	task, err := h.service.CreateTask(c.Request.Context(), req, operatorCode, createdBy)
	h.audit(c, "trace_start", task, req.DeviceSN, err)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

// ListTasks GET /trace/tasks
func (h *Handler) ListTasks(c *gin.Context) {
	filter := TaskFilter{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter.DeviceSN = c.Query("device_sn")
	if s := c.Query("status"); s != "" {
		filter.Status = TaskStatus(s)
	}
	filter.OperatorCode = c.Query("operator_code")

	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// GetTask GET /trace/tasks/:id
func (h *Handler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	t, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, t)
}

// StopTask POST /trace/tasks/:id/stop
func (h *Handler) StopTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var req StopTaskRequest
	// body 可选
	_ = c.ShouldBindJSON(&req)

	t, err := h.service.StopTask(c.Request.Context(), id, req.Purge)
	action := "trace_stop"
	if req.Purge {
		action = "trace_purge"
	}
	deviceSN := ""
	if t != nil {
		deviceSN = t.DeviceSN
	}
	h.audit(c, action, t, deviceSN, err)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, t)
}

// ListMessages GET /trace/tasks/:id/messages
func (h *Handler) ListMessages(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	filter := MessageFilter{
		TaskID:      id,
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if d := c.Query("direction"); d != "" {
		filter.Direction = Direction(d)
	}
	filter.RPCMethod = c.Query("rpc_method")
	if s := c.Query("start_time"); s != "" {
		if ts, err := time.Parse(time.RFC3339, s); err == nil {
			filter.StartTime = &ts
		}
	}
	if e := c.Query("end_time"); e != "" {
		if ts, err := time.Parse(time.RFC3339, e); err == nil {
			filter.EndTime = &ts
		}
	}

	result, err := h.service.ListMessages(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// GetMessagePayload GET /trace/tasks/:id/messages/:msgId/payload
// 返回单条报文完整 payload。
//   - inline 报文：直接返回 PayloadInline
//   - external（M2-06 大报文外置）：从 MinIO 拉回 GZIP 解压后返回
//
// 响应：application/xml; charset=utf-8
func (h *Handler) GetMessagePayload(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	msgID, err := uuid.Parse(c.Param("msgId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	msg, err := h.service.GetMessage(c.Request.Context(), taskID, msgID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	payload, err := h.service.LoadMessagePayload(c.Request.Context(), msg)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Content-Length", strconv.Itoa(len(payload)))
	_, _ = c.Writer.WriteString(payload)
}

// GetActiveTaskBySN GET /trace/devices/:sn/active-task
// 给设备详情页用；无 running 任务返回 200 + data=null。
func (h *Handler) GetActiveTaskBySN(c *gin.Context) {
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	t, err := h.service.GetActiveTaskBySN(c.Request.Context(), sn)
	if err != nil {
		// 没有 running 任务时返回 nil，避免前端按 404 处理
		if commonerrors.HTTPStatusFromError(err) == http.StatusNotFound {
			response.OK(c, nil)
			return
		}
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, t)
}

// ExportXML POST /trace/tasks/:id/export
//
// M2 异步路径：创建 export_job 并发 trace.export.requested 事件，立即返 202 + job_id。
// 前端轮询 GET /exports/{job_id}；done 时响应携带预签名 URL。
func (h *Handler) ExportXML(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	_, username := h.identity(c)
	if username == "" {
		username = "system"
	}
	job, err := h.service.RequestExport(c.Request.Context(), id, username)
	if err == nil {
		// 拿 task 信息让审计带 SN（job 本身只有 task_id）
		task, _ := h.service.GetTask(c.Request.Context(), id)
		sn := ""
		if task != nil {
			sn = task.DeviceSN
		}
		h.auditExport(c, job, sn, nil)
	} else {
		h.auditExport(c, nil, "", err)
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, job)
}

// GetExportJob GET /trace/exports/:jobId
// 状态查询；done 时附带预签名下载 URL（30min 有效）。
func (h *Handler) GetExportJob(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	job, err := h.service.GetExportJob(c.Request.Context(), jobID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	// done + 有对象 + MinIO 注入 → 生成预签名
	if job.Status == ExportJobDone && job.ObjectKey != "" && job.ObjectBucket != "" && h.presignClient != nil {
		task, _ := h.service.GetTask(c.Request.Context(), job.TaskID)
		fname := exportFilename(task, job)
		reqParams := url.Values{}
		reqParams.Set("response-content-disposition", `attachment; filename="`+fname+`"`)
		// Content-Type 在 Exporter 上传时已经写成 application/octet-stream（见
		// internal/trace/exporter.go），不要在预签名 URL 上覆盖 — minio-go 客户端
		// 对 reqParams 的 SigV4 处理与 MinIO server 不完全一致，覆盖后会得到
		// SignatureDoesNotMatch 403。
		signed, sErr := h.presignClient.PresignedGetObject(c.Request.Context(),
			job.ObjectBucket, job.ObjectKey, h.presignTTL, reqParams)
		if sErr != nil {
			h.logger.Warn("trace: presign URL failed",
				zap.String("job_id", job.ID.String()), zap.Error(sErr))
		} else {
			job.DownloadURL = signed.String()
		}
	}
	response.OK(c, job)
}

// exportFilename 文件名 "trace_{sn}_{startTime YYYYMMDD-HHmm}.xml"，
// 含设备 SN + 抓包开始时间，对运维归档/检索友好。无 task 上下文时回退到
// "trace_{jobID-short}.xml"。
func exportFilename(task *Task, job *ExportJob) string {
	if task != nil && task.DeviceSN != "" {
		ts := task.StartTime.Format("20060102-1504")
		return "trace_" + task.DeviceSN + "_" + ts + ".xml"
	}
	jobShort := job.ID.String()
	if len(jobShort) >= 8 {
		jobShort = jobShort[:8]
	}
	return "trace_" + jobShort + ".xml"
}

// identity 从 gin context 提取 username / operator_code（与 notification 模块一致）。
func (h *Handler) identity(c *gin.Context) (operatorCode, username string) {
	if u, ok := c.Get("username"); ok {
		username, _ = u.(string)
	}
	if op, ok := c.Get("operator_code"); ok {
		operatorCode, _ = op.(string)
	}
	return
}

// audit 写一条业务审计（M3-02）。
// 不阻塞主流程：audit.Log 内部已 swallow error 到 fallback logger。
func (h *Handler) audit(c *gin.Context, action string, task *Task, deviceSN string, err error) {
	if action == "" {
		return
	}
	_, username := h.identity(c)
	details := map[string]interface{}{"device_sn": deviceSN}
	resourceID := ""
	if task != nil {
		resourceID = task.ID.String()
		details["expires_at"] = task.ExpiresAt
		details["status"] = task.Status
	}
	entry := audit.Entry{
		Username:     username,
		Action:       action,
		ResourceType: "trace_task",
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Success:      err == nil,
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.Log(c.Request.Context(), entry)
}

// auditExport 异步导出操作审计。
func (h *Handler) auditExport(c *gin.Context, job *ExportJob, deviceSN string, err error) {
	_, username := h.identity(c)
	details := map[string]interface{}{"device_sn": deviceSN}
	resourceID := ""
	if job != nil {
		resourceID = job.ID.String()
		details["task_id"] = job.TaskID.String()
	}
	entry := audit.Entry{
		Username:     username,
		Action:       "trace_export",
		ResourceType: "trace_export_job",
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Success:      err == nil,
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.Log(c.Request.Context(), entry)
}

