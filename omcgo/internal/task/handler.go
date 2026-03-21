package task

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/errors"
	"go.uber.org/zap"
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
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	tasks := r.Group("/devices/:device_sn/tasks")
	{
		tasks.POST("", h.CreateTask)
		tasks.GET("", h.GetTaskHistory)
		tasks.GET("/pending", h.GetPendingTasks)
		tasks.GET("/:task_id", h.GetTask)
		tasks.DELETE("/:task_id", h.CancelTask)
		tasks.GET("/stats", h.GetTaskStats)
	}
}

// CreateTask 创建任务
// POST /api/v1/devices/:device_sn/tasks
func (h *Handler) CreateTask(c *gin.Context) {
	deviceSN := c.Param("device_sn")
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

	// 设置默认值
	if req.Source == "" {
		req.Source = TaskSourceAPI
	}
	if req.CreatorID == "" {
		// 从 context 获取用户 ID
		if userID, exists := c.Get("user_id"); exists {
			req.CreatorID = userID.(string)
		}
	}

	task, err := h.service.CreateTask(c.Request.Context(), &req)
	if err != nil {
		logger.L(c.Request.Context()).Error("create task",
			zap.Error(err),
			zap.String("device_sn", deviceSN),
			zap.String("method", req.Method))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": task,
	})
}

// GetTask 获取任务详情
// GET /api/v1/devices/:device_sn/tasks/:task_id
func (h *Handler) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		logger.L(c.Request.Context()).Error("get task", zap.Error(err), zap.String("task_id", taskID))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	if task == nil {
		errors.AbortWithError(c, http.StatusNotFound, errors.ErrNotFound)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": task,
	})
}

// GetPendingTasks 获取待处理任务列表
// GET /api/v1/devices/:device_sn/tasks/pending
func (h *Handler) GetPendingTasks(c *gin.Context) {
	deviceSN := c.Param("device_sn")
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"tasks": tasks,
			"total": len(tasks),
		},
	})
}

// GetTaskHistory 获取任务历史
// GET /api/v1/devices/:device_sn/tasks
func (h *Handler) GetTaskHistory(c *gin.Context) {
	deviceSN := c.Param("device_sn")
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": resp,
	})
}

// CancelTask 取消任务
// DELETE /api/v1/devices/:device_sn/tasks/:task_id
func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	err := h.service.CancelTask(c.Request.Context(), taskID)
	if err != nil {
		logger.L(c.Request.Context()).Error("cancel task",
			zap.Error(err),
			zap.String("task_id", taskID))

		if err.Error() == "task not found" {
			errors.AbortWithError(c, http.StatusNotFound, errors.ErrNotFound)
			return
		}

		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "task cancelled",
	})
}

// GetTaskStats 获取任务统计
// GET /api/v1/devices/:device_sn/tasks/stats
func (h *Handler) GetTaskStats(c *gin.Context) {
	deviceSN := c.Param("device_sn")
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"by_status":   stats,
			"queue_length": queueLen,
		},
	})
}

// BatchCreateTasks 批量创建任务
// POST /api/v1/devices/:device_sn/tasks/batch
func (h *Handler) BatchCreateTasks(c *gin.Context) {
	deviceSN := c.Param("device_sn")
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
	userID, _ := c.Get("user_id")
	for _, req := range reqs {
		req.DeviceSN = deviceSN
		if req.Source == "" {
			req.Source = TaskSourceAPI
		}
		if req.CreatorID == "" {
			if uid, ok := userID.(string); ok {
				req.CreatorID = uid
			}
		}
	}

	tasks, err := h.service.BatchCreateTasks(c.Request.Context(), reqs)
	if err != nil {
		logger.L(c.Request.Context()).Error("batch create tasks",
			zap.Error(err),
			zap.String("device_sn", deviceSN),
			zap.Int("count", len(reqs)))
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": gin.H{
			"created": len(tasks),
			"tasks":   tasks,
		},
	})
}

// RetryTask 重试任务
// POST /api/v1/devices/:device_sn/tasks/:task_id/retry
func (h *Handler) RetryTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		errors.AbortWithError(c, http.StatusBadRequest, errors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		errors.AbortWithError(c, http.StatusInternalServerError, errors.ErrInternal)
		return
	}

	if task == nil {
		errors.AbortWithError(c, http.StatusNotFound, errors.ErrNotFound)
		return
	}

	// 检查是否可以重试
	if !task.CanRetry() {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "task cannot be retried: exceeded max retries",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "task queued for retry",
		"data":    task,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "old tasks purged",
		"data": gin.H{
			"deleted_count": count,
		},
	})
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
