package task

import (
	stderrors "errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 任务管理 REST API Handler
type Handler struct {
	service *TaskService
}

// NewHandler 创建任务 Handler
func NewHandler(service *TaskService) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes 注册路由
// 所有接口使用查询参数 device_sn 而非路径参数，避免与 /devices/:id 路由冲突
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	tasks := r.Group("/devices/tasks")
	{
		tasks.POST("", h.CreateTask)               // POST /api/v1/devices/tasks?device_sn=xxx
		tasks.GET("", h.GetTaskHistory)            // GET /api/v1/devices/tasks?device_sn=xxx
		tasks.GET("/pending", h.GetPendingTasks)   // GET /api/v1/devices/tasks/pending?device_sn=xxx
		tasks.GET("/:task_id", h.GetTask)          // GET /api/v1/devices/tasks/:task_id
		tasks.DELETE("/:task_id", h.CancelTask)    // DELETE /api/v1/devices/tasks/:task_id
		tasks.GET("/stats", h.GetTaskStats)        // GET /api/v1/devices/tasks/stats?device_sn=xxx
		tasks.POST("/batch", h.BatchCreateTasks)   // POST /api/v1/devices/tasks/batch?device_sn=xxx
		tasks.POST("/:task_id/retry", h.RetryTask) // POST /api/v1/devices/tasks/:task_id/retry
	}

	// 管理员接口
	r.POST("/tasks/purge", h.PurgeOldTasks)
}

// CreateTask 创建任务
// POST /api/v1/devices/tasks?device_sn=xxx
func (h *Handler) CreateTask(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	// 设置设备 SN
	req.DeviceSN = deviceSN

	// REST callers must not be able to impersonate an internal task source or
	// opt into the privileged access-probe/security-action admission classes.
	req.Source = TaskSourceAPI
	req.AdmissionClass = AdmissionClassNormal
	if req.CreatorID == "" {
		req.CreatorID = admin.UserIDStringFromCtx(c)
	}

	task, err := h.service.CreateTask(c.Request.Context(), &req)
	if err != nil {
		logger.L(c.Request.Context()).Error("create task",
			zap.Error(err),
			zap.String("device_sn", deviceSN),
			zap.String("method", req.Method))
		if stderrors.Is(err, ErrTaskAdmissionDenied) {
			errors.AbortWithError(c, http.StatusConflict, ErrTaskAdmissionDenied)
			return
		}
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, task)
}

// GetTask 获取任务详情
// GET /api/v1/devices/tasks/:task_id
func (h *Handler) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTaskByDevice(c.Request.Context(), c.Query("device_sn"), taskID)
	if err != nil {
		logger.L(c.Request.Context()).Error("get task", zap.Error(err), zap.String("task_id", taskID))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	if task == nil {
		errors.AbortWithError(c, http.StatusNotFound, errors.ErrNotFound)
		return
	}

	response.OK(c, task)
}

// GetPendingTasks 获取待处理任务列表
// GET /api/v1/devices/tasks/pending?device_sn=xxx
func (h *Handler) GetPendingTasks(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	tasks, err := h.service.GetPendingTasks(c.Request.Context(), deviceSN, 0)
	if err != nil {
		logger.L(c.Request.Context()).Error("get pending tasks", zap.Error(err), zap.String("device_sn", deviceSN))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OK(c, gin.H{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// GetTaskHistory 获取任务历史
// GET /api/v1/devices/tasks?device_sn=xxx
func (h *Handler) GetTaskHistory(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	var opts TaskHistoryOptions
	if err := c.ShouldBindQuery(&opts); err != nil {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	// 设置默认分页
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 20
	}
	if opts.PageSize > 100 {
		opts.PageSize = 100
	}

	resp, err := h.service.GetTaskHistory(c.Request.Context(), deviceSN, &opts)
	if err != nil {
		logger.L(c.Request.Context()).Error("get task history",
			zap.Error(err),
			zap.String("device_sn", deviceSN))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OK(c, resp)
}

// CancelTask 取消任务
// DELETE /api/v1/devices/tasks/:task_id
func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	err := h.service.CancelTaskByDevice(c.Request.Context(), c.Query("device_sn"), taskID)
	if err != nil {
		logger.L(c.Request.Context()).Error("cancel task",
			zap.Error(err),
			zap.String("task_id", taskID))

		if stderrors.Is(err, ErrTaskNotFound) {
			errors.AbortWithError(c, http.StatusNotFound, errors.ErrNotFound)
			return
		}

		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OKWithMsg(c, nil, "task cancelled")
}

// GetTaskStats 获取任务统计
// GET /api/v1/devices/tasks/stats?device_sn=xxx
func (h *Handler) GetTaskStats(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	stats, err := h.service.GetTaskStats(c.Request.Context(), deviceSN)
	if err != nil {
		logger.L(c.Request.Context()).Error("get task stats", zap.Error(err), zap.String("device_sn", deviceSN))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	// 获取队列长度
	queueLen, err := h.service.GetQueueLength(c.Request.Context(), deviceSN)
	if err != nil {
		logger.L(c.Request.Context()).Warn("get queue length", zap.Error(err))
	}

	response.OK(c, gin.H{
		"by_status":    stats,
		"queue_length": queueLen,
	})
}

// BatchCreateTasks 批量创建任务
// POST /api/v1/devices/tasks/batch?device_sn=xxx
func (h *Handler) BatchCreateTasks(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	var reqs []*CreateTaskRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	// 设置设备 SN 和默认值
	creatorID := admin.UserIDStringFromCtx(c)
	for _, req := range reqs {
		if req == nil {
			errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
			return
		}
		req.DeviceSN = deviceSN
		req.Source = TaskSourceAPI
		req.AdmissionClass = AdmissionClassNormal
		if req.CreatorID == "" {
			req.CreatorID = creatorID
		}
	}

	tasks, err := h.service.BatchCreateTasks(c.Request.Context(), reqs)
	if err != nil {
		logger.L(c.Request.Context()).Error("batch create tasks",
			zap.Error(err),
			zap.String("device_sn", deviceSN),
			zap.Int("count", len(reqs)))
		if stderrors.Is(err, ErrTaskAdmissionDenied) {
			errors.AbortWithError(c, http.StatusConflict, ErrTaskAdmissionDenied)
			return
		}
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, gin.H{
		"created": len(tasks),
		"tasks":   tasks,
	})
}

// RetryTask 重试任务
// POST /api/v1/devices/tasks/:task_id/retry
func (h *Handler) RetryTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTaskByDevice(c.Request.Context(), c.Query("device_sn"), taskID)
	if err != nil {
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	if task == nil {
		errors.AbortWithError(c, http.StatusNotFound, errors.ErrNotFound)
		return
	}

	// 检查是否可主动重试：仅 failed/expired 终态失败类且重试预算未耗尽可 retry；
	// cancelled/completed/pending/sent 一律拒绝（4xx），不把非失败任务"复活"回 pending。
	if !task.CanManualRetry() {
		response.Fail(c, http.StatusBadRequest,
			"task cannot be retried: only failed/expired tasks within retry budget can retry, current status="+string(task.Status))
		return
	}

	// 重置任务状态
	task.ResetForRetry()

	// 更新到数据库和队列
	if err := h.service.RetryTask(c.Request.Context(), task); err != nil {
		logger.L(c.Request.Context()).Error("retry task", zap.Error(err), zap.String("task_id", taskID))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OKWithMsg(c, task, "task queued for retry")
}

// PurgeOldTasks 清理旧任务（管理员接口）
// POST /api/v1/tasks/purge
func (h *Handler) PurgeOldTasks(c *gin.Context) {
	retentionDays := 30 // 默认保留 30 天
	if d := c.Query("retention_days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			retentionDays = parsed
		}
	}

	count, err := h.service.PurgeOldTasks(c.Request.Context(), retentionDays)
	if err != nil {
		logger.L(c.Request.Context()).Error("purge old tasks", zap.Error(err))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	response.OKWithMsg(c, gin.H{
		"deleted_count": count,
	}, "old tasks purged")
}

// parseTimeParam 解析时间参数
func parseTimeParam(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
