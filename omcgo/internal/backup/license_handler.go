// Package backup — DeviceLicense HTTP handlers (T-0165).
//
// 路由挂载在 RegisterRoutes 内：前缀 /api/v1/backup/device-licenses。
// 接口与 ConfigSnapshot 端点对齐，前端可复用同款 list/import/delete/batchGet 形态。
package backup

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// SetLicenseService wires the LicenseService post-construction.
func (h *Handler) SetLicenseService(s *LicenseService) {
	h.licenseService = s
}

const (
	licenseImportMaxSize  = 10 * 1024 * 1024 // 10 MiB
	licenseImportMaxFiles = 200
)

// ListLicenses GET /api/v1/backup/device-licenses
func (h *Handler) ListLicenses(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
		return
	}
	filter := LicenseFilter{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter.SerialNumber = strings.TrimSpace(c.Query("serial_number"))
	filter.EnbName = strings.TrimSpace(c.Query("enb_name"))
	filter.ProductType = strings.TrimSpace(c.Query("product_type"))
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
	items, total, err := h.licenseService.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, model.NewListResponse(items, total, filter.Page, filter.PageSize))
}

// GetLicense GET /api/v1/backup/device-licenses/:sn
func (h *Handler) GetLicense(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
		return
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("serial_number 不能为空: %w", commonerrors.ErrInvalidInput))
		return
	}
	lic, err := h.licenseService.GetBySerialNumber(c.Request.Context(), sn)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if lic == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, lic)
}

// BatchGetLicensesRequest POST 体。
type BatchGetLicensesRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// BatchGetLicensesResponse 返回已找到 + 缺失列表。
type BatchGetLicensesResponse struct {
	Found   map[string]*DeviceLicense `json:"found"`
	Missing []string                  `json:"missing"`
}

// BatchGetLicenses POST /api/v1/backup/device-licenses/batch-get
func (h *Handler) BatchGetLicenses(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
		return
	}
	var req BatchGetLicensesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	found, err := h.licenseService.repo.BatchGetBySerialNumbers(c.Request.Context(), req.SerialNumbers)
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
	response.OK(c, BatchGetLicensesResponse{Found: found, Missing: missing})
}

// ValidateLicenseSNs POST /api/v1/backup/device-licenses/validate-sns
func (h *Handler) ValidateLicenseSNs(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
		return
	}
	var req ValidateSNsRequest // 复用 snapshot_handler 同款 body
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	existing, missing := h.licenseService.ValidateDeviceSNs(c.Request.Context(), req.SerialNumbers)
	response.OK(c, ValidateSNsResponse{Existing: existing, Missing: missing})
}

// ImportLicenses POST /api/v1/backup/device-licenses/import
func (h *Handler) ImportLicenses(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
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
	if len(files) > licenseImportMaxFiles {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("too many files (%d), max %d: %w",
				len(files), licenseImportMaxFiles, commonerrors.ErrInvalidInput))
		return
	}
	// 可选 description（form field）
	description := ""
	if vals, ok := form.Value["description"]; ok && len(vals) > 0 {
		description = vals[0]
	}

	items := make([]LicenseImportItem, 0, len(files))
	preFailed := make([]LicenseImportFailure, 0)
	for _, fh := range files {
		if fh.Size > licenseImportMaxSize {
			preFailed = append(preFailed, LicenseImportFailure{
				FileName:  fh.Filename,
				ErrorCode: ImportErrEmptyBody,
				Message:   fmt.Sprintf("文件大小超出 %d 字节上限：%d", licenseImportMaxSize, fh.Size),
			})
			continue
		}
		src, openErr := fh.Open()
		if openErr != nil {
			preFailed = append(preFailed, LicenseImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody, Message: openErr.Error(),
			})
			continue
		}
		body, readErr := io.ReadAll(src)
		_ = src.Close()
		if readErr != nil {
			preFailed = append(preFailed, LicenseImportFailure{
				FileName: fh.Filename, ErrorCode: ImportErrEmptyBody, Message: readErr.Error(),
			})
			continue
		}
		items = append(items, LicenseImportItem{
			FileName: fh.Filename, Content: body, Description: description,
		})
	}

	uploadBy := admin.UserIDStringFromCtx(c)
	result, svcErr := h.licenseService.ImportFromUpload(c.Request.Context(), items, uploadBy)
	if svcErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, svcErr)
		return
	}
	result.Failed = append(preFailed, result.Failed...)
	response.OK(c, result)
}

// LicenseDownloadResponse 同 SnapshotDownloadResponse 形态。
type LicenseDownloadResponse struct {
	SerialNumber string `json:"serial_number"`
	FileName     string `json:"file_name"`
	DownloadURL  string `json:"download_url"`
	ExpiresIn    int    `json:"expires_in_seconds"`
}

// DownloadLicense GET /api/v1/backup/device-licenses/:sn/download
func (h *Handler) DownloadLicense(c *gin.Context) {
	if h.licenseService == nil || h.minioClient == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("license download not configured"))
		return
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("serial_number 不能为空: %w", commonerrors.ErrInvalidInput))
		return
	}
	lic, err := h.licenseService.GetBySerialNumber(c.Request.Context(), sn)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if lic == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	presigned, presignErr := h.presignClient().PresignedGetObject(
		c.Request.Context(), lic.ObjectBucket, lic.ObjectPath, time.Hour, nil,
	)
	if presignErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, presignErr)
		return
	}
	response.OK(c, LicenseDownloadResponse{
		SerialNumber: sn,
		FileName:     lic.FileName,
		DownloadURL:  presigned.String(),
		ExpiresIn:    3600,
	})
}

// DeleteLicense DELETE /api/v1/backup/device-licenses/:sn
func (h *Handler) DeleteLicense(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
		return
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("serial_number 不能为空: %w", commonerrors.ErrInvalidInput))
		return
	}
	if err := h.licenseService.Delete(c.Request.Context(), sn); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"deleted": sn})
}

// BatchDeleteLicensesRequest body.
type BatchDeleteLicensesRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// BatchDeleteLicensesResponse 报告每条结果。
type BatchDeleteLicensesResponse struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
}

// BatchDeleteLicenses POST /api/v1/backup/device-licenses/batch-delete
func (h *Handler) BatchDeleteLicenses(c *gin.Context) {
	if h.licenseService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("device license service not configured"))
		return
	}
	var req BatchDeleteLicensesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	succeeded, failed, err := h.licenseService.BatchDelete(c.Request.Context(), req.SerialNumbers)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, BatchDeleteLicensesResponse{Succeeded: succeeded, Failed: failed})
}
