package filemanager

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for the file manager REST API.
type Handler struct {
	service *FileService
	logger  *zap.Logger
}

// NewHandler creates a new file manager Handler.
func NewHandler(service *FileService, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("filemanager-handler"),
	}
}

// RegisterRoutes registers file manager routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	files := rg.Group("/files")
	files.GET("", h.List)
	files.POST("", h.Upload)
	files.GET("/:id", h.GetByID)
	files.DELETE("/:id", h.Delete)
	files.GET("/:id/download", h.Download)
	files.POST("/:id/distribute", h.Distribute)
}

// List handles GET /api/v1/files.
func (h *Handler) List(c *gin.Context) {
	filter := FileFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if ft := c.Query("file_type"); ft != "" {
		t := FileType(ft)
		filter.FileType = &t
	}
	if dsn := c.Query("device_sn"); dsn != "" {
		filter.DeviceSN = &dsn
	}
	if s := c.Query("status"); s != "" {
		st := FileStatus(s)
		filter.Status = &st
	}
	if q := c.Query("search"); q != "" {
		filter.Search = &q
	}

	result, err := h.service.ListFiles(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Upload handles POST /api/v1/files (multipart/form-data).
func (h *Handler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("missing file: %w", err))
		return
	}
	defer file.Close()

	fileType := c.PostForm("file_type")
	if fileType == "" {
		fileType = "other"
	}
	description := c.PostForm("description")
	deviceSN := c.PostForm("device_sn")
	uploader := c.PostForm("uploader")

	contentType := header.Header.Get("Content-Type")

	mf, err := h.service.UploadFile(c.Request.Context(), file, header.Size, header.Filename, contentType, FileType(fileType), description, uploader, deviceSN)
	if err != nil {
		logger.L(c.Request.Context()).Error("upload file", zap.Error(err))
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, mf)
}

// GetByID handles GET /api/v1/files/:id.
func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	mf, err := h.service.GetFileByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, mf)
}

// Delete handles DELETE /api/v1/files/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteFile(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Download handles GET /api/v1/files/:id/download (blob download).
func (h *Handler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	obj, mf, err := h.service.DownloadFile(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	defer obj.Close()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, mf.FileName))
	c.Header("Content-Type", mf.ContentType)
	if mf.FileSize > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", mf.FileSize))
	}

	if _, err := io.Copy(c.Writer, obj); err != nil {
		logger.L(c.Request.Context()).Error("stream file to client", zap.String("file_id", id.String()), zap.Error(err))
		// Response headers already sent, cannot abort with JSON error
	}
}

// DistributeRequest defines the request body for distributing a file to devices.
type DistributeRequest struct {
	DeviceSNs []string `json:"device_sns" binding:"required"`
}

// Distribute handles POST /api/v1/files/:id/distribute.
func (h *Handler) Distribute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req DistributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	succeeded, failed, err := h.service.DistributeFile(c.Request.Context(), id, req.DeviceSNs)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":      id.String(),
		"device_count": len(req.DeviceSNs),
		"succeeded":    succeeded,
		"failed":       failed,
		"status":       "queued",
	})
}
