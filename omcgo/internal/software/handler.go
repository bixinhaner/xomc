package software

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides REST API endpoints for software/firmware management.
type Handler struct {
	service      *SoftwareService
	firmwareRepo FirmwareRepository
	upgradeRepo  UpgradeTaskRepository
	logger       *zap.Logger
}

// NewHandler creates a new software Handler.
func NewHandler(service *SoftwareService, firmwareRepo FirmwareRepository, upgradeRepo UpgradeTaskRepository, logger *zap.Logger) *Handler {
	return &Handler{
		service:      service,
		firmwareRepo: firmwareRepo,
		upgradeRepo:  upgradeRepo,
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

	upgrade := rg.Group("/upgrade-tasks")
	upgrade.GET("", h.ListUpgradeTasks)
	upgrade.GET("/:id", h.GetUpgradeTask)
	upgrade.POST("", h.TriggerUpgrade)
	upgrade.POST("/batch", h.BatchUpgrade)
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
		Carrier:      model.CarrierCode(c.PostForm("carrier")),
		ProductClass: c.PostForm("product_class"),
		Version:      c.PostForm("version"),
		FileName:     header.Filename,
		ReleaseNotes: c.PostForm("release_notes"),
	}

	if string(fw.Carrier) == "" || fw.Version == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(8002, "carrier and version are required", commonerrors.ErrInvalidInput))
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

	if err := h.firmwareRepo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ListUpgradeTasks(c *gin.Context) {
	var filter UpgradeTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.upgradeRepo.List(c.Request.Context(), filter)
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

	task, err := h.upgradeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) TriggerUpgrade(c *gin.Context) {
	var req TriggerUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.service.StartUpgrade(c.Request.Context(), req.DeviceID, req.FirmwareID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) BatchUpgrade(c *gin.Context) {
	var req BatchUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tasks, err := h.service.BatchUpgrade(c.Request.Context(), req.DeviceIDs, req.FirmwareID, req.Concurrency)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}
