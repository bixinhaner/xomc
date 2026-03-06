package mr

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/common/model"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for MR data.
type Handler struct {
	store       MRStore
	minioClient *minio.Client
	bucket      string
	logger      *zap.Logger
}

// NewHandler creates a new MR handler.
func NewHandler(store MRStore, minioClient *minio.Client, bucket string, logger *zap.Logger) *Handler {
	return &Handler{store: store, minioClient: minioClient, bucket: bucket, logger: logger}
}

// RegisterRoutes registers MR API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mrGroup := rg.Group("/mr")
	{
		mrGroup.GET("/files", h.ListFiles)
		mrGroup.GET("/files/:id/download", h.DownloadFile)
		mrGroup.GET("/data", h.QueryData)
	}
}

type fileQuery struct {
	DeviceID  string `form:"device_id"`
	MRType    string `form:"mr_type"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) ListFiles(c *gin.Context) {
	var q fileQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := MRFileFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
			return
		}
		filter.DeviceID = &id
	}
	if q.MRType != "" {
		filter.MRType = &q.MRType
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.store.ListFiles(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	// Look up file by ID directly
	fileInfo, err := h.store.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}
	if fileInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	obj, err := h.minioClient.GetObject(c.Request.Context(), h.bucket, fileInfo.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "download failed"})
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get file info failed"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+fileInfo.FileName)
	c.Header("Content-Type", "application/xml")
	c.DataFromReader(http.StatusOK, stat.Size, "application/xml", obj, nil)
}

type dataQuery struct {
	DeviceID string `form:"device_id"`
	FileID   string `form:"file_id"`
	MRType   string `form:"mr_type"`
	CellID   string `form:"cell_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) QueryData(c *gin.Context) {
	var q dataQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := MRRecordFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
			return
		}
		filter.DeviceID = &id
	}
	if q.FileID != "" {
		id, err := uuid.Parse(q.FileID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_id"})
			return
		}
		filter.FileID = &id
	}
	if q.MRType != "" {
		filter.MRType = &q.MRType
	}
	if q.CellID != "" {
		filter.CellID = &q.CellID
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.store.QueryRecords(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
