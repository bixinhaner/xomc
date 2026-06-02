package definition

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// FileHandler 提供告警库 XML 文件粒度的上传/删除端点(严格对标 T-0180 indicator FileHandler;
// 告警侧 custom 目录扁平,无 tech 段):
//   - POST   /api/v1/alarm-definitions/upload-xml[?force=] — 上传自定义 XML,自动 Reload + RefreshCache
//   - DELETE /api/v1/alarm-definitions/files/{*loadedFrom}  — 删除自定义 XML + DB 行 + RefreshCache
//
// 与 handler.go 的 CRUD(单条告警定义粒度)隔离,本 handler 管文件粒度。
type FileHandler struct {
	repo      FileRepository
	service   *Service // RefreshCache:上传/删除后刷新内存 Registry
	reloader  Reloader // ReloadOne:上传后重扫目录 UPSERT 入库(可为 nil → 跳过)
	baseDir   string   // XMLBaseDir(= dictloader.XMLBaseDir,如 /etc/omcgo/data)
	fileLocks sync.Map // map[basename(string)]*sync.Mutex
	logger    *zap.Logger
}

// NewFileHandler 构造 FileHandler;baseDir 必须为 Loader 用的同一 XMLBaseDir。
func NewFileHandler(repo FileRepository, service *Service, reloader Reloader, baseDir string, logger *zap.Logger) *FileHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileHandler{repo: repo, service: service, reloader: reloader, baseDir: baseDir, logger: logger.Named("alarmdef.file")}
}

// RegisterRoutes 挂在 /api/v1 之下;内部使用 /alarm-definitions/... 子路径。
// gin 的 *loadedFrom 是 catch-all wildcard,承载 "alarm-definitions-custom/MY.xml" 多段相对路径。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/alarm-definitions")
	g.POST("/upload-xml", h.UploadXML)
	files := g.Group("/files")
	files.DELETE("/*loadedFrom", h.DeleteFile)
}

// acquireFileLock 取/建 per-basename mutex,Upload / Delete 共享,防同名文件并发写半截。
func (h *FileHandler) acquireFileLock(basename string) func() {
	mu, _ := h.fileLocks.LoadOrStore(basename, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

// refreshCache 刷新内存 Registry(失败仅 Warn,不阻断主流程)。
func (h *FileHandler) refreshCache(ctx context.Context) {
	if h.service == nil {
		return
	}
	if err := h.service.RefreshCache(ctx); err != nil {
		h.logger.Warn("post-op alarm registry refresh failed", zap.Error(err))
	}
}

// validCustomPath 校验 loaded_from 恰好 = alarm-definitions-custom/<file>.xml(两段,防遍历)。
func validCustomPath(loadedFrom string) bool {
	if !strings.HasPrefix(loadedFrom, CustomDirPrefix) {
		return false
	}
	parts := strings.Split(loadedFrom, "/")
	if len(parts) != 2 {
		return false
	}
	return parts[1] != "" && !strings.Contains(parts[1], "..")
}

// DeleteFile DELETE /api/v1/alarm-definitions/files/{loadedFrom}
//
// 守门链(对标 indicator DeleteFile):
//  1. 解码 + normalize loaded_from
//  2. IsDeletable → false 返 403 + ErrCodeAlarmBuiltinNotDeletable
//  3. validCustomPath 二次防遍历
//  4. CountByLoadedFrom=0 且 文件不存在 → 404
//  5. acquireFileLock 持锁
//  6. 物理 rename .deleted.<ts> 备份;失败保守回滚返 500 + ErrCodeAlarmBackupFailed
//  7. DeleteByLoadedFrom;失败 rename 回滚
//  8. RefreshCache + 审计日志 + 200
func (h *FileHandler) DeleteFile(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("loadedFrom"), "/")
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if loadedFrom == "" || loadedFrom == "." {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("loaded_from path is required"))
		return
	}

	if !IsDeletable(loadedFrom) {
		h.logger.Info("audit: alarm file delete rejected (builtin or unknown)",
			zap.String("audit_action", "alarm.delete.rejected_builtin"),
			zap.String("loaded_from", loadedFrom),
			zap.String("source", string(ClassifySource(loadedFrom))))
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("alarm XML %q (source=%s) is not deletable; "+
				"builtin files are managed by the release image (data/alarm-definitions/) [code=%d]",
				loadedFrom, ClassifySource(loadedFrom), global.ErrCodeAlarmBuiltinNotDeletable))
		return
	}
	if !validCustomPath(loadedFrom) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid custom alarm path %q (expected %s<file>.xml)", loadedFrom, CustomDirPrefix))
		return
	}

	rowCount, err := h.repo.CountByLoadedFrom(c.Request.Context(), loadedFrom)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	absPath := filepath.Join(h.baseDir, loadedFrom)
	_, statErr := os.Stat(absPath)
	if rowCount == 0 && errors.Is(statErr, fs.ErrNotExist) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	unlock := h.acquireFileLock(filepath.Base(loadedFrom))
	defer unlock()

	backupPath := absPath + ".deleted." + time.Now().Format("20060102150405")
	if renameErr := os.Rename(absPath, backupPath); renameErr != nil {
		switch {
		case errors.Is(renameErr, fs.ErrNotExist):
			h.logger.Warn("audit: custom alarm xml already gone before delete; proceeding to remove DB rows",
				zap.String("audit_action", "alarm.delete.custom.file_already_gone"),
				zap.String("loaded_from", loadedFrom))
			backupPath = ""
		default:
			h.logger.Error("audit: alarm file delete aborted (backup failed)",
				zap.String("audit_action", "alarm.delete.aborted_backup_failed"),
				zap.String("loaded_from", loadedFrom), zap.Error(renameErr))
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("backup custom xml failed; refusing to delete DB rows to avoid data loss; "+
					"please fix the filesystem and retry: %w [code=%d]",
					renameErr, global.ErrCodeAlarmBackupFailed))
			return
		}
	}

	rowsAffected, dbErr := h.repo.DeleteByLoadedFrom(c.Request.Context(), loadedFrom)
	if dbErr != nil {
		if backupPath != "" {
			if rbErr := os.Rename(backupPath, absPath); rbErr != nil {
				h.logger.Error("rollback rename failed; DB and host state diverged",
					zap.String("backup_path", backupPath), zap.String("orig_path", absPath), zap.Error(rbErr))
			}
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, dbErr)
		return
	}

	h.refreshCache(c.Request.Context())

	backupName := ""
	if backupPath != "" {
		backupName = filepath.Base(backupPath)
	}
	h.logger.Info("audit: alarm file deleted (custom)",
		zap.String("audit_action", "alarm.delete.custom"),
		zap.String("loaded_from", loadedFrom),
		zap.Int("rows_affected", rowsAffected),
		zap.String("backup", backupName))

	response.OK(c, gin.H{
		"deleted":       true,
		"loaded_from":   loadedFrom,
		"rows_affected": rowsAffected,
		"backup":        backupName,
	})
}

// UploadXML POST /api/v1/alarm-definitions/upload-xml[?force=true]
//
// multipart/form-data;字段 file = XML 字节流。守门链:文件大小 → 文件名白名单 →
// XML 根 <alarmModel> → 路径不逃逸。写入:tmp → (overwrite 备份 .bak) → rename →
// Reload(UPSERT 入库)→ RefreshCache。
func (h *FileHandler) UploadXML(c *gin.Context) {
	force := c.Query("force") == "true"

	fh, err := c.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("form field \"file\" is required: %w", err))
		return
	}
	if fh.Size > MaxUploadXMLSize {
		h.logger.Info("audit: alarm upload rejected (too large)",
			zap.String("audit_action", "alarm.upload.rejected_too_large"),
			zap.Int64("size", fh.Size), zap.Int64("max", MaxUploadXMLSize))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("upload size %d exceeds max %d [code=%d]",
				fh.Size, MaxUploadXMLSize, global.ErrCodeAlarmUploadTooLarge))
		return
	}

	base := filepath.Base(fh.Filename)
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: alarm upload rejected (invalid filename)",
			zap.String("audit_action", "alarm.upload.rejected_invalid_name"),
			zap.String("filename", fh.Filename))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeAlarmUploadInvalidName))
		return
	}

	src, err := fh.Open()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("open upload: %w", err))
		return
	}
	body, err := readAllLimited(src, MaxUploadXMLSize)
	_ = src.Close()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("read upload: %w", err))
		return
	}
	if err := validateUploadXML(body); err != nil {
		h.logger.Info("audit: alarm upload rejected (invalid xml)",
			zap.String("audit_action", "alarm.upload.rejected_invalid_xml"),
			zap.String("filename", base), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeAlarmUploadInvalidRoot))
		return
	}

	customDir := filepath.Join(h.baseDir, CustomDirSubdir)
	targetPath := filepath.Join(customDir, base)
	if !pathContainedIn(customDir, targetPath) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("computed target path %q escapes custom dir %q", targetPath, customDir))
		return
	}

	unlock := h.acquireFileLock(base)
	defer unlock()

	if err := os.MkdirAll(customDir, 0o755); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("mkdir custom dir: %w", err))
		return
	}

	_, statErr := os.Stat(targetPath)
	overwrite := false
	if statErr == nil {
		if !force {
			h.logger.Info("audit: alarm upload conflict (same-name exists)",
				zap.String("audit_action", "alarm.upload.conflict"),
				zap.String("filename", base))
			commonerrors.AbortWithError(c, http.StatusConflict,
				fmt.Errorf("file %q already exists; pass ?force=true to overwrite (will backup to .bak.<ts>)", base))
			return
		}
		overwrite = true
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("stat target: %w", statErr))
		return
	}

	tmpPath := targetPath + ".tmp." + uniqueSuffix()
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("write tmp: %w", err))
		return
	}
	var backupPath string
	if overwrite {
		backupPath = targetPath + ".bak." + time.Now().Format("20060102150405")
		if err := os.Rename(targetPath, backupPath); err != nil {
			_ = os.Remove(tmpPath)
			commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("backup existing file: %w", err))
			return
		}
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		if backupPath != "" {
			_ = os.Rename(backupPath, targetPath)
		}
		_ = os.Remove(tmpPath)
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("rename tmp to target: %w", err))
		return
	}

	loadedFrom := filepath.ToSlash(filepath.Join(CustomDirSubdir, base))

	reloadOK := true
	if h.reloader != nil {
		if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
			reloadOK = false
			h.logger.Warn("audit: alarm upload reload failed (file written, DB not refreshed)",
				zap.String("audit_action", "alarm.upload.reload_failed"),
				zap.String("loaded_from", loadedFrom), zap.Error(err))
		}
	}
	h.refreshCache(c.Request.Context())

	backupName := ""
	if backupPath != "" {
		backupName = filepath.Base(backupPath)
	}
	h.logger.Info("audit: alarm file uploaded",
		zap.String("audit_action", "alarm.upload.success"),
		zap.String("loaded_from", loadedFrom),
		zap.Bool("overwrite", overwrite),
		zap.Int("body_size", len(body)),
		zap.Bool("reload_ok", reloadOK),
		zap.String("backup", backupName))

	status := http.StatusCreated
	if overwrite {
		status = http.StatusOK
	}
	response.OKWithStatus(c, status, gin.H{
		"uploaded":    true,
		"filename":    base,
		"loaded_from": loadedFrom,
		"overwrite":   overwrite,
		"backup":      backupName,
		"reloaded":    reloadOK,
	})
}

// EnsureBaseDir 确保 baseDir/<CustomDirSubdir>/ 目录存在(0755)。启动期由 provider 调用。
func EnsureBaseDir(baseDir string) error {
	dir := filepath.Join(baseDir, CustomDirSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure alarm custom dir %s: %w", dir, err)
	}
	return nil
}
