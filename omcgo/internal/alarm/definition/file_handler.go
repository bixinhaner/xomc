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

// FileHandler 提供告警库 XML 文件粒度的上传/删除端点。告警库 XML 管理重构后,
// "导入 XML / 重载 XML / 刷新缓存"三个功能合并为一个 upload-xml 端点:
//   - POST   /api/v1/alarm-definitions/upload-xml[?force=] — 写文件(同名 force 覆盖 + .bak 备份)
//     → destructive 全量重载(删孤儿)→ RefreshCache,三步顺序完成
//   - DELETE /api/v1/alarm-definitions/files/{*loadedFrom}  — 删除 XML 文件 + DB 行 + RefreshCache
//
// 取消 builtin/custom 区分:上传统一写进 builtin 目录(Loader 扫描的同一目录),
// 接受升级丢失;所有文件均可删。与 handler.go 的 CRUD(单条告警定义粒度)隔离。
type FileHandler struct {
	repo      FileRepository
	service   *Service // RefreshCache + DeleteOrphansSince:上传/删除后刷新内存 Registry / 删孤儿
	reloader  Reloader // ReloadOne:上传后全量重扫目录 UPSERT 入库(可为 nil → 跳过)
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
// gin 的 *loadedFrom 是 catch-all wildcard,承载 "alarm-definitions/MY.xml" 多段相对路径。
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

// validAlarmFilePath 校验 loaded_from 恰好 = alarm-definitions/<file>.xml 或
// alarm-definitions-custom/<file>.xml(两段,防遍历)。
//
// 告警库 XML 管理重构后上传统一写进 builtin 目录,故主路径是 BuiltinDirPrefix;
// 同时兼容历史 custom 行(重构前上传过的文件),让其仍可删除。
func validAlarmFilePath(loadedFrom string) bool {
	if !strings.HasPrefix(loadedFrom, BuiltinDirPrefix) &&
		!strings.HasPrefix(loadedFrom, CustomDirPrefix) {
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
// 告警库 XML 管理重构后取消 builtin/custom 区分,所有 XML 文件一律可删
// (IsDeletable 恒 true,不再有 403 拦截)。守门链:
//  1. 解码 + normalize loaded_from
//  2. validAlarmFilePath 二次防遍历(接受 builtin/custom 两种前缀)
//  3. CountByLoadedFrom=0 且 文件不存在 → 404
//  4. acquireFileLock 持锁
//  5. 物理 rename .deleted.<ts> 备份;失败保守回滚返 500 + ErrCodeAlarmBackupFailed
//  6. DeleteByLoadedFrom;失败 rename 回滚
//  7. RefreshCache + 审计日志 + 200
func (h *FileHandler) DeleteFile(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("loadedFrom"), "/")
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if loadedFrom == "" || loadedFrom == "." {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("loaded_from path is required"))
		return
	}

	if !validAlarmFilePath(loadedFrom) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid alarm xml path %q (expected %s<file>.xml or %s<file>.xml)",
				loadedFrom, BuiltinDirPrefix, CustomDirPrefix))
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
	h.logger.Info("audit: alarm file deleted",
		zap.String("audit_action", "alarm.delete"),
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
// XML 根 <alarmModel> → 路径不逃逸(写进 builtin 目录)。写入:tmp → (overwrite 备份 .bak)
// → rename → destructive 全量重载(UPSERT 入库 + 删孤儿)→ RefreshCache。
// 后三步失败只 Warn 不致命(文件已落盘)。
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

	// 告警库 XML 管理重构:取消 builtin/custom 区分,上传直接写进 builtin 目录
	// (Loader 扫描的同一目录),接受升级丢失。
	builtinDir := filepath.Join(h.baseDir, BuiltinDirSubdir)
	targetPath := filepath.Join(builtinDir, base)
	if !pathContainedIn(builtinDir, targetPath) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("computed target path %q escapes builtin dir %q", targetPath, builtinDir))
		return
	}

	unlock := h.acquireFileLock(base)
	defer unlock()

	if err := os.MkdirAll(builtinDir, 0o755); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("mkdir builtin dir: %w", err))
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

	loadedFrom := filepath.ToSlash(filepath.Join(BuiltinDirSubdir, base))

	// 导入 = 写文件 → destructive 全量重载(删孤儿) → RefreshCache,三步顺序完成。
	// 任一后续步骤失败只 Warn 不致命(文件已落盘,DB 最多落后一拍)。
	//
	// destructive 删孤儿:记录 startedAt,Loader 全量 UPSERT 后用 updated_at < startedAt
	// 判定本次未被触达的旧定义为孤儿并删除(与原"重载 XML"端点 mode=reload 同语义)。
	startedAt := time.Now()
	reloadOK := true
	if h.reloader != nil {
		if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
			reloadOK = false
			h.logger.Warn("audit: alarm upload reload failed (file written, DB not refreshed)",
				zap.String("audit_action", "alarm.upload.reload_failed"),
				zap.String("loaded_from", loadedFrom), zap.Error(err))
		}
	}

	var orphansDeleted int64
	if reloadOK && h.service != nil {
		n, err := h.service.DeleteOrphansSince(c.Request.Context(), startedAt)
		if err != nil {
			h.logger.Warn("audit: alarm upload orphan cleanup failed (reload ok, orphans kept)",
				zap.String("audit_action", "alarm.upload.orphan_cleanup_failed"),
				zap.String("loaded_from", loadedFrom), zap.Error(err))
		} else {
			orphansDeleted = n
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
		zap.Int64("orphans_deleted", orphansDeleted),
		zap.String("backup", backupName))

	status := http.StatusCreated
	if overwrite {
		status = http.StatusOK
	}
	response.OKWithStatus(c, status, gin.H{
		"uploaded":        true,
		"filename":        base,
		"loaded_from":     loadedFrom,
		"overwrite":       overwrite,
		"backup":          backupName,
		"reloaded":        reloadOK,
		"orphans_deleted": orphansDeleted,
	})
}

// EnsureBaseDir 确保 baseDir/<BuiltinDirSubdir>/ 目录存在(0755)。启动期由 provider 调用。
// 告警库 XML 管理重构后上传统一写进 builtin 目录(Loader 扫描的同一目录),故这里确保
// builtin 目录存在(镜像层通常已有,空库 / 本地裸跑时兜底)。
func EnsureBaseDir(baseDir string) error {
	dir := filepath.Join(baseDir, BuiltinDirSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure alarm builtin dir %s: %w", dir, err)
	}
	return nil
}
