// Package admin — UI 定制化资产上传 / 公开下载。
//
// 端点：
//
//	POST /api/v1/admin/uploads/ui-asset       —— 上传 Logo / 登录背景图（鉴权）
//	GET  /api/v1/admin/public/ui-assets/:name —— 公开读取（登录页消费）
//
// 体积上限按 kind 区分：login_bg < 1 MiB；logo_small / logo_large < 400 KiB。
// 仅接收 image/png / image/jpeg。资产持久化到 MinIO 的 BucketConfig.UIAssets。
// 详见 docs/prd/system/ui-customization.md §6.1。
package admin

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/storageprotection"
)

// UI 资产上传体积上限（bytes）。
const (
	uiAssetMaxLoginBg int64 = 1 * 1024 * 1024 // 1 MiB
	uiAssetMaxLogo    int64 = 400 * 1024      // 400 KiB
	uiAssetPublicBase       = "/api/v1/admin/public/ui-assets/"
)

// uploadKindLimits 把 kind → 体积上限映射，未知 kind 走最严格的 logo 上限。
var uploadKindLimits = map[string]int64{
	"login_bg":   uiAssetMaxLoginBg,
	"logo_small": uiAssetMaxLogo,
	"logo_large": uiAssetMaxLogo,
}

// UIAssetHandler 处理 UI 定制化资产的上传与公开读取。
type UIAssetHandler struct {
	minio            *minio.Client
	bucket           string
	storageAdmission storageprotection.WriteAdmission
}

// NewUIAssetHandler creates a new UIAssetHandler bound to the given MinIO bucket.
func NewUIAssetHandler(client *minio.Client, bucket string) *UIAssetHandler {
	return &UIAssetHandler{minio: client, bucket: bucket}
}

func (h *UIAssetHandler) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	h.storageAdmission = admission
}

// RegisterRoutes 注册受鉴权保护的上传端点（挂在 admin 子组下）。
func (h *UIAssetHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/uploads/ui-asset", h.Upload)
}

// RegisterPublicRoutes 注册无需鉴权的资产读取端点（登录页消费）。
func (h *UIAssetHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/admin/public/ui-assets/:name", h.Serve)
}

// Upload 接收 multipart 文件 + kind 字段，写入 MinIO 后返回相对 URL。
func (h *UIAssetHandler) Upload(c *gin.Context) {
	if h.minio == nil || h.bucket == "" {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			commonerrors.NewBusinessError(8, "ui-asset storage not configured", nil))
		return
	}

	kind := c.PostForm("kind")
	limit, ok := uploadKindLimits[kind]
	if !ok {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(7, "invalid or missing kind: "+kind, nil))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(7, "missing file", err))
		return
	}
	if fileHeader.Size > limit {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(7,
				fmt.Sprintf("file too large: %d bytes (kind=%s, limit=%d)", fileHeader.Size, kind, limit),
				nil))
		return
	}

	contentType, ext, ok := detectImageContentType(fileHeader)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(7, "unsupported content type (png/jpeg only)", nil))
		return
	}

	objectName := uuid.NewString() + ext
	if putErr := h.putObject(c.Request.Context(), fileHeader, objectName, contentType); putErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, putErr)
		return
	}

	response.OKWithMsg(c, gin.H{
		"url":  uiAssetPublicBase + objectName,
		"name": objectName,
		"size": fileHeader.Size,
	}, "上传成功")
}

// Serve 把 :name 对应的 MinIO 对象作为静态资源吐回。
// 仅接受形如 <uuid>.png / <uuid>.jpg / <uuid>.jpeg 的文件名（防路径遍历）。
func (h *UIAssetHandler) Serve(c *gin.Context) {
	if h.minio == nil || h.bucket == "" {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			commonerrors.NewBusinessError(8, "ui-asset storage not configured", nil))
		return
	}

	name := c.Param("name")
	if !isSafeAssetName(name) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(7, "invalid asset name", nil))
		return
	}

	obj, err := h.minio.GetObject(c.Request.Context(), h.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, err)
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		// minio-go returns NoSuchKey here when object missing.
		commonerrors.AbortWithError(c, http.StatusNotFound, err)
		return
	}

	contentType := stat.ContentType
	if contentType == "" {
		contentType = contentTypeFromExt(name)
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=300")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, obj); err != nil {
		// 写入响应失败时不能再调 AbortWithError（headers 已发送）。
		_ = c.Error(err)
	}
}

// putObject 流式把 multipart 文件写入 MinIO。
func (h *UIAssetHandler) putObject(ctx context.Context, fh *multipart.FileHeader, objectName, contentType string) error {
	if h.storageAdmission != nil {
		decision, err := h.storageAdmission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, storageprotection.WriteScopeUpload)
		if err != nil {
			return fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
	src, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open uploaded file: %w", err)
	}
	defer src.Close()

	if _, err := h.minio.PutObject(ctx, h.bucket, objectName, src, fh.Size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return fmt.Errorf("put ui-asset to MinIO: %w", err)
	}
	return nil
}

// detectImageContentType 严格只放行 image/png 与 image/jpeg。
// 优先以 multipart 头部的 Content-Type 判定，缺失或不可信时退回到扩展名。
// 目前不做 magic-number 校验（本端点鉴权后，后续 P1 可加）。
func detectImageContentType(fh *multipart.FileHeader) (contentType, ext string, ok bool) {
	ct := strings.ToLower(strings.TrimSpace(fh.Header.Get("Content-Type")))
	switch ct {
	case "image/png":
		return "image/png", ".png", true
	case "image/jpeg", "image/jpg":
		return "image/jpeg", ".jpg", true
	}
	// fallback by extension
	switch strings.ToLower(path.Ext(fh.Filename)) {
	case ".png":
		return "image/png", ".png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", ".jpg", true
	}
	return "", "", false
}

// isSafeAssetName 仅接受 <uuid>.<ext> 形式（无路径分隔符、无 ..）。
func isSafeAssetName(name string) bool {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	ext := strings.ToLower(path.Ext(name))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return false
	}
	stem := strings.TrimSuffix(name, path.Ext(name))
	if _, err := uuid.Parse(stem); err != nil {
		return false
	}
	return true
}

func contentTypeFromExt(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
