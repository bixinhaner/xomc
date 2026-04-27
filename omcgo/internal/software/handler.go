package software

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// Handler provides REST API endpoints for software/firmware management.
type Handler struct {
	service      *SoftwareService
	firmwareRepo FirmwareRepository
	taskRepo     TaskRepository
	subTaskRepo  SubTaskRepository
	logger       *zap.Logger
}

// NewHandler creates a new software Handler.
func NewHandler(service *SoftwareService, firmwareRepo FirmwareRepository, taskRepo TaskRepository, subTaskRepo SubTaskRepository, logger *zap.Logger) *Handler {
	return &Handler{
		service:      service,
		firmwareRepo: firmwareRepo,
		taskRepo:     taskRepo,
		subTaskRepo:  subTaskRepo,
		logger:       logger.Named("software-handler"),
	}
}

// RegisterRoutes registers firmware and upgrade task routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	fw := rg.Group("/firmware")
	fw.GET("", h.ListFirmware)
	fw.POST("", h.UploadFirmware)
	fw.GET("/:id", h.GetFirmware)
	fw.DELETE("/:id", h.DeleteFirmware)
	fw.PUT("/:id/recommend", h.ToggleFirmwareRecommend)
	fw.GET("/:id/download", h.DownloadFirmware)
	fw.PUT("/:id", h.UpdateFirmware)

	upgrade := rg.Group("/upgrade-tasks")
	upgrade.GET("", h.ListUpgradeTasks)
	upgrade.GET("/:id", h.GetUpgradeTask)
	upgrade.POST("", h.CreateUpgradeTask)
	upgrade.PUT("/:id/suspend", h.SuspendUpgradeTask)
	upgrade.PUT("/:id/resume", h.ResumeUpgradeTask)
	upgrade.PUT("/:id/terminate", h.TerminateUpgradeTask)
	upgrade.DELETE("/:id", h.DeleteUpgradeTask)
	upgrade.POST("/:id/retry", h.RetryUpgradeTask)
	upgrade.POST("/rollback", h.CreateRollback)
	upgrade.GET("/:id/tasks", h.ListSubTasks)

	subTasks := rg.Group("/upgrade-sub-tasks")
	subTasks.GET("/:id", h.GetSubTask)
	subTasks.GET("", h.ListAllSubTasks)
}

func (h *Handler) ListFirmware(c *gin.Context) {
	var filter FirmwareFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.firmwareRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) UploadFirmware(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	fw := &FirmwareVersion{
		ProductClass: c.PostForm("product_class"),
		Version:      c.PostForm("version"),
		FileName:     header.Filename,
		ReleaseNotes: c.PostForm("release_notes"),
		Uploader:     c.PostForm("uploader"),
		Manufacturer: c.PostForm("manufacturer"),
		Description:  c.PostForm("description"),
	}
	if c.PostForm("recommend") == "true" {
		fw.Recommend = true
	}
	if ft := c.PostForm("file_type"); ft != "" {
		if n, err := strconv.Atoi(ft); err == nil {
			fw.FileType = FileType(n)
		}
	}

	if fw.Version == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(8002, "version is required", commonerrors.ErrInvalidInput))
		return
	}

	if err := h.service.UploadFirmware(c.Request.Context(), fw, file, header.Size); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, fw)
}

func (h *Handler) GetFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	fw, err := h.firmwareRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, fw)
}

func (h *Handler) DeleteFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteFirmware(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ToggleFirmwareRecommend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	fw, err := h.firmwareRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	fw.Recommend = !fw.Recommend
	if err := h.firmwareRepo.Update(c.Request.Context(), fw); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, fw)
}

func (h *Handler) DownloadFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Fetch firmware metadata for filename
	fw, err := h.firmwareRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	obj, err := h.service.DownloadFirmware(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	defer obj.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+fw.FileName+"\"")
	c.Header("Content-Type", "application/octet-stream")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj)
}

func (h *Handler) UpdateFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	fw, err := h.firmwareRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	var req struct {
		ProductClass string `json:"product_class"`
		Version      string `json:"version"`
		Recommend    *bool  `json:"recommend"`
		Description  string `json:"description"`
		ReleaseNotes string `json:"release_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if req.ProductClass != "" {
		fw.ProductClass = req.ProductClass
	}
	if req.Version != "" {
		fw.Version = req.Version
	}
	if req.Recommend != nil {
		fw.Recommend = *req.Recommend
	}
	fw.Description = req.Description
	fw.ReleaseNotes = req.ReleaseNotes

	if err := h.service.UpdateFirmwareMetadata(c.Request.Context(), fw); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, fw)
}

func (h *Handler) ListUpgradeTasks(c *gin.Context) {
	var filter UpgradeTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.taskRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) CreateUpgradeTask(c *gin.Context) {
	var req BatchUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.service.BatchUpgrade(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) SuspendUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.SuspendUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "upgrade task suspended"})
}

func (h *Handler) ResumeUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.ResumeUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "upgrade task resumed"})
}

func (h *Handler) TerminateUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.TerminateUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "upgrade task terminated"})
}

func (h *Handler) RetryUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.RetryUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "retry initiated"})
}

func (h *Handler) DeleteUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "upgrade task deleted"})
}

func (h *Handler) CreateRollback(c *gin.Context) {
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.service.RollbackDevices(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) ListSubTasks(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	var filter SubTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter.TaskID = taskID

	result, err := h.subTaskRepo.ListByTaskID(c.Request.Context(), taskID, filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListAllSubTasks(c *gin.Context) {
	var filter AllSubTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.subTaskRepo.ListAll(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetSubTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.subTaskRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, task)
}
