// Package backup — ConfigSnapshot HTTP handlers (T-0164 / B4).
//
// 路由挂载在 RegisterRoutes 现有 backup group 内，前缀 /api/v1/backup/config-snapshots。
// 与 backup_tasks / restore_tasks 共享同一权限组 ("devices")，但写操作（import / delete）
// 由前端层结合 RBAC 角色控制（config:write）。
//
// 所有端点都通过 *Handler.SetSnapshotService 注入；未注入时返回 503，
// 与 policy / restore 的 nil-safe 模式一致。
package backup

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// SetSnapshotService wires the SnapshotService post-construction (DI hook,
// matches T-0071 / T-0072 patterns).
func (h *Handler) SetSnapshotService(s *SnapshotService) {
	h.snapshotService = s
}

// snapshotImportMaxSize 是单文件导入上限（10 MiB）。
// 配置文件实际大小一般 <100 KiB；上限主要防误传日志 / 二进制大对象。
const snapshotImportMaxSize = 10 * 1024 * 1024

// snapshotImportMaxFiles 是单次导入文件数上限。
const snapshotImportMaxFiles = 200

// ListSnapshots handles GET /api/v1/backup/config-snapshots
//
// Query 参数：
//
//	serial_number      string   模糊
//	enb_name           string   模糊
//	product_type       string   精确
//	source             string   "backup" | "manual_upload"
//	updated_after      RFC3339
//	updated_before     RFC3339
//	page / page_size / sort_by / sort_dir  通用分页
func (h *Handler) ListSnapshots(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	filter := SnapshotFilter{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter.SerialNumber = strings.TrimSpace(c.Query("serial_number"))
	filter.EnbName = strings.TrimSpace(c.Query("enb_name"))
	filter.ProductType = strings.TrimSpace(c.Query("product_type"))
	if src := strings.TrimSpace(c.Query("source")); src != "" {
		ss := SnapshotSource(src)
		filter.Source = &ss
	}
	if v := strings.TrimSpace(c.Query("updated_after")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.UpdatedAfter = &t
		}
	}
	if v := strings.TrimSpace(c.Query("updated_before")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.UpdatedBefore = &t
		}
	}

	items, total, err := h.snapshotService.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, model.NewListResponse(items, total, filter.Page, filter.PageSize))
}

// GetSnapshot handles GET /api/v1/backup/config-snapshots/:sn
func (h *Handler) GetSnapshot(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, ErrEmptySerialNumber)
		return
	}
	snap, err := h.snapshotService.GetBySerialNumber(c.Request.Context(), sn)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if snap == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, snap)
}

// BatchGetSnapshotsRequest is the body for POST /backup/config-snapshots/batch-get.
type BatchGetSnapshotsRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// BatchGetSnapshotsResponse 返回找到的 map + 缺失 SN 列表。
// 调用方（前端恢复创建页）据此渲染"哪些设备暂无快照"。
type BatchGetSnapshotsResponse struct {
	Found   map[string]*ConfigSnapshot `json:"found"`
	Missing []string                   `json:"missing"`
}

// ValidateSNsRequest is the body for POST /backup/config-snapshots/validate-sns.
type ValidateSNsRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// ValidateSNsResponse 报告每个 SN 在 devices 表中是否存在，
// 给前端 ImportDrawer 拖入文件后立即标红未知 SN。
type ValidateSNsResponse struct {
	Existing []string `json:"existing"`
	Missing  []string `json:"missing"`
}

// ValidateImportSNs handles POST /api/v1/backup/config-snapshots/validate-sns
func (h *Handler) ValidateImportSNs(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	var req ValidateSNsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	existing, missing := h.snapshotService.ValidateDeviceSNs(c.Request.Context(), req.SerialNumbers)
	response.OK(c, ValidateSNsResponse{Existing: existing, Missing: missing})
}

// BatchGetSnapshots handles POST /api/v1/backup/config-snapshots/batch-get
func (h *Handler) BatchGetSnapshots(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	var req BatchGetSnapshotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	found, err := h.snapshotService.BatchGetBySerialNumbers(c.Request.Context(), req.SerialNumbers)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	missing := make([]string, 0)
	for _, sn := range req.SerialNumbers {
		if _, ok := found[sn]; !ok {
			missing = append(missing, sn)
		}
	}
	response.OK(c, BatchGetSnapshotsResponse{Found: found, Missing: missing})
}

// ImportSnapshots handles POST /api/v1/backup/config-snapshots/import
//
// 接收 multipart/form-data，字段名 "files"，可挂多个文件。
// 每个文件按 <SN>_CFG.{xml,nv} 严格校验；不通过的进入 failed 列表，
// 通过的写 MinIO + Upsert。返回 succeeded + failed 两部分。
func (h *Handler) ImportSnapshots(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("parse multipart: %w", err))
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("no files uploaded: %w", commonerrors.ErrInvalidInput))
		return
	}
	if len(files) > snapshotImportMaxFiles {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("too many files (%d), max %d: %w",
				len(files), snapshotImportMaxFiles, commonerrors.ErrInvalidInput))
		return
	}

	items := make([]SnapshotImportItem, 0, len(files))
	tooLarge := make([]SnapshotImportFailure, 0)
	for _, fh := range files {
		if fh.Size > snapshotImportMaxSize {
			tooLarge = append(tooLarge, SnapshotImportFailure{
				FileName:  fh.Filename,
				ErrorCode: ImportErrEmptyBody, // 复用 code；message 含详细原因
				Message:   fmt.Sprintf("文件大小超出 %d 字节上限：%d", snapshotImportMaxSize, fh.Size),
			})
			continue
		}
		src, openErr := fh.Open()
		if openErr != nil {
			tooLarge = append(tooLarge, SnapshotImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody,
				Message: openErr.Error(),
			})
			continue
		}
		body, readErr := io.ReadAll(src)
		_ = src.Close()
		if readErr != nil {
			tooLarge = append(tooLarge, SnapshotImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody,
				Message: readErr.Error(),
			})
			continue
		}
		items = append(items, SnapshotImportItem{FileName: fh.Filename, Content: body})
	}

	uploadBy := admin.UserIDStringFromCtx(c)
	result, svcErr := h.snapshotService.ImportFromUpload(c.Request.Context(), items, uploadBy)
	if svcErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, svcErr)
		return
	}
	// 把 multipart 阶段失败的并入 service 返回的 failed，统一一个出口。
	result.Failed = append(tooLarge, result.Failed...)
	response.OK(c, result)
}

// DownloadSnapshot handles GET /api/v1/backup/config-snapshots/:sn/download.
// It streams the object through the authenticated API origin so browsers do not
// need direct access to the MinIO public endpoint.
func (h *Handler) DownloadSnapshot(c *gin.Context) {
	if h.snapshotService == nil || h.objectClient == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("snapshot download not configured"))
		return
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, ErrEmptySerialNumber)
		return
	}
	snap, err := h.snapshotService.GetBySerialNumber(c.Request.Context(), sn)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if snap == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	obj, getErr := h.objectClient.GetObject(
		c.Request.Context(), snap.ObjectBucket, snap.ObjectPath, minio.GetObjectOptions{},
	)
	if getErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, getErr)
		return
	}
	defer obj.Close()

	stat, statErr := obj.Stat()
	if statErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, statErr)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", snap.FileName))
	c.DataFromReader(http.StatusOK, stat.Size, "application/octet-stream", obj, nil)
}

// DeleteSnapshot handles DELETE /api/v1/backup/config-snapshots/:sn
func (h *Handler) DeleteSnapshot(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, ErrEmptySerialNumber)
		return
	}
	if err := h.snapshotService.Delete(c.Request.Context(), sn); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"deleted": sn})
}

// BatchDeleteSnapshotsRequest body.
type BatchDeleteSnapshotsRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// BatchDeleteSnapshotsResponse 报告每条 SN 的删除结果。
type BatchDeleteSnapshotsResponse struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
}

// BatchDeleteSnapshots handles DELETE /api/v1/backup/config-snapshots
func (h *Handler) BatchDeleteSnapshots(c *gin.Context) {
	if h.snapshotService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("config snapshot service not configured"))
		return
	}
	var req BatchDeleteSnapshotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	succeeded, failed, err := h.snapshotService.BatchDelete(c.Request.Context(), req.SerialNumbers)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, BatchDeleteSnapshotsResponse{Succeeded: succeeded, Failed: failed})
}
