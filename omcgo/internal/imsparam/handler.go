// Package imsparam HTTP handler。
//
// 路由（挂在 /api/v1 下，见 cmd/app/provider/router.go）：
//
//	GET    /imsparam/param-types          参数类型注册表（FT_ImsCore_*）
//	GET    /imsparam/param-files          分页列表（param_type / file_name 过滤）
//	POST   /imsparam/param-files/import   multipart 批量导入（files + param_type + description）
//	GET    /imsparam/param-files/:id/download  下载文件
//	DELETE /imsparam/param-files/:id      删除
//	POST   /imsparam/param-files/batch-delete  批量删除（POST body，防代理吞 DELETE+body）
package imsparam

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

const (
	imsParamImportMaxFiles = 50
	imsParamImportMaxSize  = 32 << 20 // 32MB / 文件
)

type Handler struct {
	service      *Service
	objectClient ObjectGetter
}

func NewHandler(service *Service, objectClient ObjectGetter) *Handler {
	return &Handler{service: service, objectClient: objectClient}
}

// RegisterRoutes 注册到 /api/v1 组。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/imsparam/param-types", h.ListParamTypes)
	files := rg.Group("/imsparam/param-files")
	{
		files.GET("", h.ListFiles)
		files.POST("/import", h.ImportFiles)
		files.POST("/batch-delete", h.BatchDeleteFiles)
		files.GET("/:id/download", h.DownloadFile)
		files.DELETE("/:id", h.DeleteFile)
	}
}

// ParamTypeOption 是 GET /imsparam/param-types 的响应条目。
// kind / cwmpFileType 供前端按任务模板（参数/日志/鉴权/备份…）过滤下拉。
type ParamTypeOption = FileTypeDef

func (h *Handler) ListParamTypes(c *gin.Context) {
	response.OK(c, ParamTypes())
}

// BackendParamFile 是列表条目的 wire 格式（snake_case）。
type BackendParamFile struct {
	ID          string  `json:"id"`
	ParamType   string  `json:"param_type"`
	FileName    string  `json:"file_name"`
	ObjectPath  string  `json:"object_path"`
	MD5         *string `json:"md5"`
	FileSize    int64   `json:"file_size"`
	Description *string `json:"description"`
	UploadedBy  *string `json:"uploaded_by"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func (h *Handler) ListFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	filter := ParamFileFilter{
		ParamType:  c.Query("param_type"),
		FileName:   c.Query("file_name"),
		UploadedBy: c.Query("uploaded_by"),
		Page:       page,
		PageSize:   pageSize,
		SortBy:     c.Query("sort_by"),
		SortDir:    c.Query("sort_dir"),
	}
	items, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]BackendParamFile, 0, len(items))
	for _, item := range items {
		out = append(out, BackendParamFile{
			ID:          item.ID.String(),
			ParamType:   item.ParamType,
			FileName:    item.FileName,
			ObjectPath:  item.ObjectPath,
			MD5:         item.MD5,
			FileSize:    item.FileSize,
			Description: item.Description,
			UploadedBy:  item.UploadedBy,
			CreatedAt:   item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	response.OK(c, gin.H{
		"items": out, "total": total, "page": page, "page_size": filter.Limit(),
	})
}

// usernameFromGin 取 JWT 中间件注入的用户名（CtxKeyUsername）；未认证（如内部
// API key 调用）时回退 user_id 字符串，保证 uploaded_by 仍可追溯。
func usernameFromGin(c *gin.Context) string {
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		if name, ok := v.(string); ok && name != "" {
			return name
		}
	}
	return admin.UserIDStringFromCtx(c)
}

func (h *Handler) ImportFiles(c *gin.Context) {
	paramType := c.PostForm("param_type")
	if _, ok := Lookup(paramType); !ok {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("未知参数类型 %q: %w", paramType, commonerrors.ErrInvalidInput))
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("parse multipart: %w", err))
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("no files uploaded: %w", commonerrors.ErrInvalidInput))
		return
	}
	if len(files) > imsParamImportMaxFiles {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("too many files (%d), max %d: %w", len(files), imsParamImportMaxFiles, commonerrors.ErrInvalidInput))
		return
	}
	description := c.PostForm("description")

	items := make([]ImportItem, 0, len(files))
	preFailed := make([]ImportFailure, 0)
	for _, fh := range files {
		if fh.Size > imsParamImportMaxSize {
			preFailed = append(preFailed, ImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody,
				Message: fmt.Sprintf("文件大小超出 %d 字节上限：%d", imsParamImportMaxSize, fh.Size),
			})
			continue
		}
		src, openErr := fh.Open()
		if openErr != nil {
			preFailed = append(preFailed, ImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody, Message: openErr.Error(),
			})
			continue
		}
		body, readErr := io.ReadAll(src)
		_ = src.Close()
		if readErr != nil {
			preFailed = append(preFailed, ImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody, Message: readErr.Error(),
			})
			continue
		}
		items = append(items, ImportItem{
			FileName: fh.Filename, Content: body, Description: description,
		})
	}

	uploadBy := usernameFromGin(c)
	result, svcErr := h.service.ImportFromUpload(c.Request.Context(), paramType, items, uploadBy)
	if svcErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, svcErr)
		return
	}
	result.Failed = append(preFailed, result.Failed...)
	response.OK(c, result)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	if h.objectClient == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("ims param download not configured"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid id: %w", commonerrors.ErrInvalidInput))
		return
	}
	file, svcErr := h.service.GetByID(c.Request.Context(), id)
	if svcErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, svcErr)
		return
	}
	if file == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	obj, getErr := h.objectClient.GetObject(
		c.Request.Context(), file.ObjectBucket, file.ObjectPath, minio.GetObjectOptions{},
	)
	if getErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, getErr)
		return
	}
	defer obj.Close()
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.FileName))
	c.DataFromReader(http.StatusOK, file.FileSize, "application/octet-stream", obj, nil)
}

func (h *Handler) DeleteFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid id: %w", commonerrors.ErrInvalidInput))
		return
	}
	if delErr := h.service.Delete(c.Request.Context(), id); delErr != nil {
		if delErr == commonerrors.ErrNotFound {
			commonerrors.AbortWithError(c, http.StatusNotFound, delErr)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, delErr)
		return
	}
	response.OK(c, gin.H{"deleted": id.String()})
}

func (h *Handler) BatchDeleteFiles(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, raw := range req.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid id %q: %w", raw, commonerrors.ErrInvalidInput))
			return
		}
		ids = append(ids, id)
	}
	succeeded, failed, err := h.service.BatchDelete(c.Request.Context(), ids)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	outSucceeded := make([]string, 0, len(succeeded))
	for _, id := range succeeded {
		outSucceeded = append(outSucceeded, id.String())
	}
	outFailed := make([]string, 0, len(failed))
	for _, id := range failed {
		outFailed = append(outFailed, id.String())
	}
	response.OK(c, gin.H{"succeeded": outSucceeded, "failed": outFailed})
}
