package definition

import (
	"context"
	"encoding/xml"
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
//   - POST   /api/v1/alarm-definitions/upload-xml — name + 双唯一性硬拒(无 force) + 写 sidecar
//     → destructive 全量重载(删孤儿)→ RefreshCache,顺序完成
//   - DELETE /api/v1/alarm-definitions/files/{*loadedFrom}  — 删除 XML 文件 + sidecar + DB 行 + RefreshCache
//
// 单目录 + sidecar(2026-06-04 D3/D5/D6):上传写进 alarm-definitions/(Loader 扫描的同一目录),
// 同时写 sidecar(<name>.xml.custom)标记为用户上传;仅有 sidecar 的文件可在线删。
// 与 handler.go 的 CRUD(单条告警定义粒度)隔离。
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

// validAlarmFilePath 校验 loaded_from 恰好 = alarm-definitions/<file>.xml(两段,防遍历)。
//
// 三库 XML 导入重构后单目录:所有 XML 同住 alarm-definitions/,故唯一合法前缀是 BuiltinDirPrefix。
func validAlarmFilePath(loadedFrom string) bool {
	if !strings.HasPrefix(loadedFrom, BuiltinDirPrefix) {
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
// 单目录 + sidecar:仅 custom(有 sidecar)可删,builtin / unknown 一律 403。守门链:
//  1. 解码 + normalize loaded_from
//  2. validAlarmFilePath 二次防遍历(唯一合法前缀 alarm-definitions/)
//  3. IsDeletable 守门:builtin / unknown → 403 + ErrCodeAlarmBuiltinNotDeletable
//  4. CountByLoadedFrom=0 且 文件不存在 → 404
//  5. acquireFileLock 持锁
//  6. 物理 rename .deleted.<ts> 备份;失败保守回滚返 500 + ErrCodeAlarmBackupFailed
//  7. DeleteByLoadedFrom;失败 rename 回滚
//  8. 移除 sidecar 标记 + RefreshCache + 审计日志 + 200
func (h *FileHandler) DeleteFile(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("loadedFrom"), "/")
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if loadedFrom == "" || loadedFrom == "." {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("loaded_from path is required"))
		return
	}

	if !validAlarmFilePath(loadedFrom) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid alarm xml path %q (expected %s<file>.xml)",
				loadedFrom, BuiltinDirPrefix))
		return
	}

	// 内置数据守门:builtin(当前目录 XML 加载)+ unknown 不可删(2026-06-04 用户决策)
	if !IsDeletable(h.baseDir, loadedFrom) {
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("内置数据不允许删除(loaded_from=%q)[code=%d]", loadedFrom, global.ErrCodeAlarmBuiltinNotDeletable))
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

	// DB 行已删,移除 sidecar 标记(<name>.xml.custom)。XML 本体已 rename 为 .deleted.<ts> 备份,
	// sidecar 不再有判定意义;残留会让孤儿 sidecar 累积。ENOENT 容忍(builtin 文件从未写过 sidecar)。
	if rmErr := os.Remove(absPath + CustomMarkerSuffix); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		h.logger.Warn("remove sidecar marker failed (non-fatal)",
			zap.String("loaded_from", loadedFrom),
			zap.String("sidecar", filepath.Base(absPath)+CustomMarkerSuffix),
			zap.Error(rmErr))
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

// UploadXML POST /api/v1/alarm-definitions/upload-xml
//
// multipart/form-data;字段 name(必填,用户指定的唯一名称,不含扩展名) + file(XML 字节流)。
// 目标文件名 = <name>.xml,写进单目录 alarm-definitions/(Loader 扫描的同一目录),
// 同时写 sidecar(<name>.xml.custom)标记为用户上传。上传文件自身的 filename 被忽略。
//
// 设计(2026-06-04 锁定决策 D3/D5/D6):单目录 + sidecar;上传 = name + 双唯一性硬拒(无 force)。
// 守门链(顺序敏感,任一失败即 400/409):
//  1. name:validateUploadFilename(name+".xml") 白名单正则
//  2. 大小:file.Size <= MaxUploadXMLSize (1 MiB)
//  3. 内容:validateUploadXML 根元素 = <alarmModel>
//  4. 路径:filepath.Join(builtinDir, <name>.xml) 经 pathContainedIn 二次验证不逃逸
//     5a. 文件名唯一性:<name>.xml 已存在 → 409(请改名),不覆盖、不备份
//     5b. 内容主键唯一性(§7.1):XML 推断的 neType 已在 DB(来自其它文件)→ 409
//
// neType 推断与 Loader 一致:优先 <alarmModel neType="..."> 属性,缺省回退文件名(去 .xml 大写)。
// 写入:tmp → rename → 写 sidecar(失败回滚 XML → 500) → destructive 全量重载(删孤儿)→ RefreshCache。
// 后三步失败只 Warn 不致命(文件已落盘)。
func (h *FileHandler) UploadXML(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("form field \"file\" is required: %w", err))
		return
	}

	// 校验 1: name 表单字段(目标文件名 = <name>.xml);上传文件自身的 filename 被忽略。
	name := strings.TrimSpace(c.PostForm("name"))
	base := name + ".xml"
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: alarm upload rejected (invalid name)",
			zap.String("audit_action", "alarm.upload.rejected_invalid_name"),
			zap.String("name", name), zap.String("basename", base))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeAlarmUploadInvalidName))
		return
	}

	// 校验 2: 大小
	if fh.Size > MaxUploadXMLSize {
		h.logger.Info("audit: alarm upload rejected (too large)",
			zap.String("audit_action", "alarm.upload.rejected_too_large"),
			zap.Int64("size", fh.Size), zap.Int64("max", MaxUploadXMLSize))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("upload size %d exceeds max %d [code=%d]",
				fh.Size, MaxUploadXMLSize, global.ErrCodeAlarmUploadTooLarge))
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

	// 校验 3: XML 根元素 = <alarmModel>
	if err := validateUploadXML(body); err != nil {
		h.logger.Info("audit: alarm upload rejected (invalid xml)",
			zap.String("audit_action", "alarm.upload.rejected_invalid_xml"),
			zap.String("filename", base), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeAlarmUploadInvalidRoot))
		return
	}

	// 内容主键(neType)推断 —— 与 Loader.loadAlarmFile 同口径:优先 XML neType 属性,
	// 缺省回退文件名(去 .xml 大写)。
	neType := parseUploadedNeType(body, base)

	// 校验 4: 单目录写入 + 路径不逃逸
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

	// 校验 5a: 文件名唯一性 —— <name>.xml 已存在 → 409,不覆盖、不备份(请改名)。
	if _, statErr := os.Stat(targetPath); statErr == nil {
		h.logger.Info("audit: alarm upload rejected (filename exists)",
			zap.String("audit_action", "alarm.upload.rejected_name_exists"),
			zap.String("filename", base))
		commonerrors.AbortWithError(c, http.StatusConflict,
			fmt.Errorf("filename %q already exists; please rename", base))
		return
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("stat target: %w", statErr))
		return
	}

	// 校验 5b: 内容主键唯一性 —— 推断的 neType 已在 DB(来自其它文件)→ 409。
	exists, err := h.repo.NeTypeExists(c.Request.Context(), neType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if exists {
		h.logger.Info("audit: alarm upload rejected (ne_type exists)",
			zap.String("audit_action", "alarm.upload.rejected_ne_type_exists"),
			zap.String("filename", base), zap.String("ne_type", neType))
		commonerrors.AbortWithError(c, http.StatusConflict,
			fmt.Errorf("content key (ne_type %q) already exists; please use a different alarmModel", neType))
		return
	}

	// 1. tmp 写入 → 原子 rename 上线(文件名唯一性已保证 target 不存在)。
	tmpPath := targetPath + ".tmp." + uniqueSuffix()
	defer os.Remove(tmpPath) // 兜底:rename 成功后 tmp 已不存在,Remove 返 ENOENT 无害
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("write tmp: %w", err))
		return
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("rename tmp to target: %w", err))
		return
	}

	// 2. 写 sidecar 标记(<name>.xml.custom 空文件)。失败 → 回滚已写 XML → 500。
	if err := os.WriteFile(targetPath+CustomMarkerSuffix, nil, 0o644); err != nil {
		if rmErr := os.Remove(targetPath); rmErr != nil {
			h.logger.Error("rollback uploaded xml after sidecar write failed; host state inconsistent",
				zap.String("target_path", targetPath), zap.Error(rmErr))
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("write sidecar marker: %w", err))
		return
	}

	loadedFrom := filepath.ToSlash(filepath.Join(BuiltinDirSubdir, base))

	// 3. 导入 = 写文件 → destructive 全量重载(删孤儿) → RefreshCache,顺序完成。
	// 任一后续步骤失败只 Warn 不致命(文件已落盘,DB 最多落后一拍)。
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

	h.logger.Info("audit: alarm file uploaded",
		zap.String("audit_action", "alarm.upload.success"),
		zap.String("loaded_from", loadedFrom),
		zap.String("ne_type", neType),
		zap.Int("body_size", len(body)),
		zap.Bool("reload_ok", reloadOK),
		zap.Int64("orphans_deleted", orphansDeleted))

	response.OKWithStatus(c, http.StatusCreated, gin.H{
		"uploaded":        true,
		"filename":        base,
		"loaded_from":     loadedFrom,
		"ne_type":         neType,
		"reloaded":        reloadOK,
		"orphans_deleted": orphansDeleted,
	})
}

// parseUploadedNeType 推断上传 XML 的 ne_type 内容主键,与 Loader.loadAlarmFile 同口径:
// 优先 <alarmModel neType="..."> 属性;缺省回退文件名(去 .xml,大写)。
// 解析失败(已被 validateUploadXML 过 well-formedness,极少触发)同样回退文件名。
func parseUploadedNeType(raw []byte, base string) string {
	var doc xmlAlarmModel
	if err := xml.Unmarshal(raw, &doc); err == nil {
		if nt := strings.TrimSpace(doc.NeType); nt != "" {
			return nt
		}
	}
	return strings.ToUpper(strings.TrimSuffix(base, filepath.Ext(base)))
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
