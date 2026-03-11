package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

// Handler provides HTTP handlers for the file manager REST API.
type Handler struct {
	repo        FileRepository
	minioClient *minio.Client
	bucket      string
	logger      *zap.Logger
}

// NewHandler creates a new file manager Handler.
func NewHandler(repo FileRepository, minioClient *minio.Client, bucket string, logger *zap.Logger) *Handler {
	return &Handler{
		repo:        repo,
		minioClient: minioClient,
		bucket:      bucket,
		logger:      logger.Named("filemanager-handler"),
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

	result, err := h.repo.List(c.Request.Context(), filter)
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

	// Build MinIO path: managed-files/{file_type}/{date}/{filename}
	objectPath := fmt.Sprintf("managed-files/%s/%s/%s", fileType, time.Now().Format("2006-01-02"), header.Filename)

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload to MinIO
	_, err = h.minioClient.PutObject(c.Request.Context(), h.bucket, objectPath, file, header.Size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		h.logger.Error("upload file to MinIO", zap.String("path", objectPath), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("upload to storage: %w", err))
		return
	}

	// Save metadata to DB
	mf := &ManagedFile{
		FileName:    header.Filename,
		FileType:    FileType(fileType),
		FileSize:    header.Size,
		MinIOPath:   objectPath,
		ContentType: contentType,
		Status:      FileReady,
	}
	if uploader != "" {
		mf.Uploader = &uploader
	}
	if deviceSN != "" {
		mf.DeviceSN = &deviceSN
	}
	if description != "" {
		mf.Description = &description
	}

	if err := h.repo.Create(c.Request.Context(), mf); err != nil {
		h.logger.Error("create file record", zap.Error(err))
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

	mf, err := h.repo.GetByID(c.Request.Context(), id)
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

	// Fetch file metadata to get MinIO path for cleanup
	mf, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// Delete from MinIO
	if err := h.minioClient.RemoveObject(c.Request.Context(), h.bucket, mf.MinIOPath, minio.RemoveObjectOptions{}); err != nil {
		h.logger.Warn("remove file from MinIO", zap.String("path", mf.MinIOPath), zap.Error(err))
		// Continue with DB deletion even if MinIO removal fails
	}

	// Delete metadata from DB
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
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

	mf, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	obj, err := h.minioClient.GetObject(c.Request.Context(), h.bucket, mf.MinIOPath, minio.GetObjectOptions{})
	if err != nil {
		h.logger.Error("get file from MinIO", zap.String("path", mf.MinIOPath), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("retrieve from storage: %w", err))
		return
	}
	defer obj.Close()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, mf.FileName))
	c.Header("Content-Type", mf.ContentType)
	if mf.FileSize > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", mf.FileSize))
	}

	if _, err := io.Copy(c.Writer, obj); err != nil {
		h.logger.Error("stream file to client", zap.String("file_id", id.String()), zap.Error(err))
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

	mf, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	var req DistributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Queue a distribution task (simplified — full implementation deferred to worker integration)
	taskID := uuid.New().String()

	h.logger.Info("file distribution queued",
		zap.String("task_id", taskID),
		zap.String("file_id", mf.ID.String()),
		zap.Int("device_count", len(req.DeviceSNs)),
	)

	c.JSON(http.StatusOK, gin.H{
		"task_id":      taskID,
		"file_id":      mf.ID.String(),
		"device_count": len(req.DeviceSNs),
		"status":       "queued",
	})
}
