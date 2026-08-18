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
	"strconv"
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
//   - POST   /api/v1/alarm-definitions/upload-xml — 名称取自 XML neType 属性 + 双唯一性硬拒(无 force) + 写 sidecar
//     → destructive 全量重载(删孤儿)→ RefreshCache,顺序完成
//   - DELETE /api/v1/alarm-definitions/files/{*loadedFrom}  — 删除 XML 文件 + sidecar + DB 行 + RefreshCache
//
// 单目录 + sidecar(2026-06-04 D3/D5/D6):上传写进 alarm-definitions/(Loader 扫描的同一目录),
// 同时写 sidecar(<name>.xml.custom)标记为用户上传;仅有 sidecar 的文件可在线删。
// 与 handler.go 的 CRUD(单条告警定义粒度)隔离。
type FileHandler struct {
	repo          FileRepository
	service       *Service            // RefreshCache + DeleteOrphansSince:上传/删除后刷新内存 Registry / 删孤儿
	reloader      Reloader            // ReloadOne:上传后全量重扫目录 UPSERT 入库(可为 nil → 跳过)
	dictRefresher DictSourceRefresher // #241: 导入成功后刷绑定字典(可为 nil → 跳过)
	baseDir       string              // XMLBaseDir(= dictloader.XMLBaseDir,如 /etc/omcgo/data)
	fileLocks     sync.Map            // map[basename(string)]*sync.Mutex
	logger        *zap.Logger
}

// DictSourceRefresher 在导入 XML 成功后按 source_table 刷新绑定字典(T-0182 / #241)。
// best-effort:刷新失败不阻断导入(daily cron 兜底)。可为 nil(未接入时整段跳过)。
type DictSourceRefresher interface {
	RefreshSourceBoundByTable(ctx context.Context, sourceTable string) (int, error)
}

// NewFileHandler 构造 FileHandler;baseDir 必须为 Loader 用的同一 XMLBaseDir。
// dictRefresher 可为 nil(未接入数据字典数据源时);非 nil 时导入成功后刷 alarm_definitions 绑定字典。
func NewFileHandler(repo FileRepository, service *Service, reloader Reloader, dictRefresher DictSourceRefresher, baseDir string, logger *zap.Logger) *FileHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileHandler{repo: repo, service: service, reloader: reloader, dictRefresher: dictRefresher, baseDir: baseDir, logger: logger.Named("alarmdef.file")}
}

// RegisterRoutes 挂在 /api/v1 之下;内部使用 /alarm-definitions/... 子路径。
// gin 的 *loadedFrom 是 catch-all wildcard,承载 "alarm-definitions/MY.xml" 多段相对路径。
// 下载用 ?loaded_from= query(而非 wildcard path),避免与既有 GET 路由冲突且免编码斜杠。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/alarm-definitions")
	g.POST("/upload-xml", h.UploadXML)
	g.GET("/file-content", h.DownloadFile)
	files := g.Group("/files")
	files.DELETE("/*loadedFrom", h.DeleteFile)
}

// DownloadFile GET /api/v1/alarm-definitions/file-content
//
// 两种模式(#268):
//   - ?loaded_from=alarm-definitions/<file>.xml — 下载磁盘原文件(builtin 与 custom
//     均可,只读无守门);路径校验与 DeleteFile 同口径:validAlarmFilePath +
//     pathContainedIn 双重防遍历。
//   - ?ne_type=ENB(不带 loaded_from) — 手工新增行没有落盘文件,从 DB 取该
//     ne_type 下 loaded_from 为空的全量定义,动态生成 alarmModel XML 返回。
//     字段与导入格式对齐(severity 用名称 / isShow Y|N / eventType 数字),
//     导出文件可直接回传 upload-xml。
func (h *FileHandler) DownloadFile(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("loaded_from"))
	if raw == "" {
		h.downloadGenerated(c)
		return
	}
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if !validAlarmFilePath(loadedFrom) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid loaded_from %q (expected %s<file>.xml)", raw, BuiltinDirPrefix))
		return
	}
	absPath := filepath.Join(h.baseDir, loadedFrom)
	if !pathContainedIn(filepath.Join(h.baseDir, BuiltinDirSubdir), absPath) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("path %q escapes alarm dir", loadedFrom))
		return
	}
	if _, err := os.Stat(absPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("stat xml: %w", err))
		return
	}
	c.FileAttachment(absPath, filepath.Base(absPath))
}

// downloadGenerated 是 DownloadFile 的生成模式:按 ne_type 从 DB 生成手工新增
// 告警的 alarmModel XML。description 是 DB 侧备注字段、XML 格式没有,导出即丢;
// deviceType 未入库,省略(回传导入只校验根元素与 neType 属性,不受影响)。
func (h *FileHandler) downloadGenerated(c *gin.Context) {
	neType := strings.TrimSpace(c.Query("ne_type"))
	if neType == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("either loaded_from or ne_type query is required"))
		return
	}
	defs, err := h.repo.ListManualByNeType(c.Request.Context(), neType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("load manual definitions: %w", err))
		return
	}
	if len(defs) == 0 {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	model := xmlAlarmModel{NeType: neType, TotalCount: len(defs), Alarms: make([]xmlAlarm, 0, len(defs))}
	for i := range defs {
		d := &defs[i]
		eventType := ""
		if d.EventType != nil {
			eventType = strconv.Itoa(*d.EventType)
		}
		model.Alarms = append(model.Alarms, xmlAlarm{
			Identifier:      d.Identifier,
			CnName:          d.CnName,
			EnName:          d.EnName,
			Severity:        d.SeverityName,
			EventType:       eventType,
			CnProbableCause: d.CnProbableCause,
			EnProbableCause: d.EnProbableCause,
			IsShow:          map[bool]string{true: "Y", false: "N"}[d.IsShow],
		})
	}
	body, err := xml.MarshalIndent(model, "", "    ")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("marshal alarmModel: %w", err))
		return
	}
	out := append([]byte(xml.Header), body...)

	c.Header("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", sanitizeDownloadName(neType)))
	c.Data(http.StatusOK, "application/xml; charset=utf-8", out)
}

// sanitizeDownloadName 把 neType 收敛成合法下载文件名 <neType>-manual.xml:
// 仅保留 [A-Za-z0-9_-],其余字符替换为 '_',防 Content-Disposition 注入。
// 收敛后必匹配 uploadFilenamePattern(neType ≤ 16 字符),下载文件可直接回传
// upload-xml。
func sanitizeDownloadName(neType string) string {
	var b strings.Builder
	b.Grow(len(neType) + len("-manual.xml"))
	for _, r := range neType {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String() + "-manual.xml"
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

// refreshBoundDict 在导入成功后刷新绑定 sourceTable 的数据字典(T-0182 / #241)。
// best-effort:未接入(dictRefresher==nil)或刷新失败均只记日志,不影响导入响应。
func (h *FileHandler) refreshBoundDict(ctx context.Context, sourceTable string) {
	if h.dictRefresher == nil {
		return
	}
	n, err := h.dictRefresher.RefreshSourceBoundByTable(ctx, sourceTable)
	if err != nil {
		h.logger.Warn("post-upload bound dictionary refresh failed",
			zap.String("source_table", sourceTable), zap.Error(err))
		return
	}
	if n > 0 {
		h.logger.Info("post-upload bound dictionary refreshed",
			zap.String("source_table", sourceTable), zap.Int("dicts", n))
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

// UploadXML POST /api/v1/alarm-definitions/upload-xml[?force=true]
//
// multipart/form-data;字段 file(XML 字节流)。
//
// 名称取消手填(2026-06-05 导入 XML 调整):唯一名称取自 XML <alarmModel neType="...">
// 属性,目标文件名 = <neType>.xml,写进单目录 alarm-definitions/(Loader 扫描的同一目录)。
// 上传文件自身的 filename 被忽略。
//
// 重复允许覆盖(2026-06-05 调整,需前端二次确认):
//   - 重复判定:neType 已在 DB(LoadedFromsByNeType 定位归属文件)或 <neType>.xml 已在盘上
//   - 无 ?force=true → 409,响应 data.overwritable=true,前端弹确认框
//   - ?force=true → 覆盖归属文件(原文件先备份 .bak.<ts>,backup_cleanup 周期清理);
//     覆盖内置文件时保持 builtin 身份(不写 sidecar,仍不可删),覆盖自定义保持 custom
//
// 守门链(顺序敏感,任一失败即 400/409):
//  1. 大小:file.Size <= MaxUploadXMLSize (1 MiB)
//  2. 内容:validateUploadXML 根元素 = <alarmModel>
//  3. 名称:neType 属性必填;长度 ≤ MaxNeTypeLen(ne_type 列 varchar(16),#123);
//     validateUploadFilename(neType+".xml") 白名单正则
//  4. 路径:目标路径经 pathContainedIn 二次验证不逃逸
//  5. 重复 + force 判定(见上)
//
// 写入:[备份旧文件 →] tmp → rename → sidecar(仅新建)→ destructive 全量重载(删孤儿)
// → RefreshCache。重载失败 → 回滚落盘文件 + sidecar 并返 500(#123,不留"文件在盘上
// 但 DB 0 行"的半成品);孤儿清理 / RefreshCache 失败只 Warn 不致命。
func (h *FileHandler) UploadXML(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("form field \"file\" is required: %w", err))
		return
	}

	// 校验 1: 大小
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

	// 校验 2: XML 根元素 = <alarmModel>
	if err := validateUploadXML(body); err != nil {
		h.logger.Info("audit: alarm upload rejected (invalid xml)",
			zap.String("audit_action", "alarm.upload.rejected_invalid_xml"),
			zap.String("filename", fh.Filename), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeAlarmUploadInvalidRoot))
		return
	}

	// 校验 3: 名称提取 —— 唯一来源 XML <alarmModel neType="..."> 属性(取消手填 name)。
	// 内容主键与目标文件名同源:neType,与 Loader.loadAlarmFile 同口径。
	neType := parseUploadedNeType(body)
	if neType == "" {
		h.logger.Info("audit: alarm upload rejected (missing neType)",
			zap.String("audit_action", "alarm.upload.rejected_missing_ne_type"),
			zap.String("filename", fh.Filename))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("alarmModel XML 缺少 neType 属性,无法确定名称 [code=%d]",
				global.ErrCodeAlarmUploadInvalidName))
		return
	}
	// 校验 3.1: neType 长度 ≤ MaxNeTypeLen(=ne_type 列 varchar(16))。不前置拒绝的话,
	// Loader 重载该文件事务必失败,形成"201 假成功但 0 行入库"的静默失败(#123)。
	if err := validateUploadNeType(neType); err != nil {
		h.logger.Info("audit: alarm upload rejected (neType too long)",
			zap.String("audit_action", "alarm.upload.rejected_ne_type_too_long"),
			zap.String("ne_type", neType))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("neType「%s」超长:长度不能超过 %d 字符(数据库 ne_type 列为 varchar(%d)):%w [code=%d]",
				neType, MaxNeTypeLen, MaxNeTypeLen, err, global.ErrCodeAlarmUploadInvalidName))
		return
	}
	base := neType + ".xml"
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: alarm upload rejected (invalid neType-derived name)",
			zap.String("audit_action", "alarm.upload.rejected_invalid_name"),
			zap.String("ne_type", neType), zap.String("basename", base))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("neType「%s」不能作为文件名:%w [code=%d]",
				neType, err, global.ErrCodeAlarmUploadInvalidName))
		return
	}

	force := c.Query("force") == "true"

	// 校验 4/5: 定位目标文件 + 重复判定。
	// 内容主键(neType)已在 DB → 归属文件即覆盖目标;否则落 <neType>.xml,
	// 盘上已存在(uploaded-but-not-loaded)同样视为重复。
	builtinDir := filepath.Join(h.baseDir, BuiltinDirSubdir)
	targetPath := filepath.Join(builtinDir, base)
	loadedFrom := filepath.ToSlash(filepath.Join(BuiltinDirSubdir, base))
	conflict := false

	owners, err := h.repo.LoadedFromsByNeType(c.Request.Context(), neType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if len(owners) > 0 {
		// 归属路径异常(历史脏数据)时回退派生路径,destructive 重载会清孤儿行
		conflict = true
		if validAlarmFilePath(owners[0]) {
			loadedFrom = owners[0]
			targetPath = filepath.Join(h.baseDir, loadedFrom)
		}
	}
	if !pathContainedIn(builtinDir, targetPath) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("computed target path %q escapes builtin dir %q", targetPath, builtinDir))
		return
	}
	if _, statErr := os.Stat(targetPath); statErr == nil {
		conflict = true
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("stat target: %w", statErr))
		return
	}

	// 重复且未带 force → 409 + overwritable 标记,前端弹二次确认后带 ?force=true 重试。
	if conflict && !force {
		h.logger.Info("audit: alarm upload conflict (needs confirm)",
			zap.String("audit_action", "alarm.upload.conflict_needs_confirm"),
			zap.String("ne_type", neType), zap.String("loaded_from", loadedFrom))
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"ret": 0,
			"msg": fmt.Sprintf("neType「%s」已存在(%s),确认后可覆盖", neType, loadedFrom),
			"data": gin.H{
				"overwritable": true,
				"ne_type":      neType,
				"loaded_from":  loadedFrom,
			},
		})
		return
	}

	unlock := h.acquireFileLock(filepath.Base(targetPath))
	defer unlock()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("mkdir builtin dir: %w", err))
		return
	}

	// 覆盖场景:先把旧文件备份为 .bak.<ts>(worker backup_cleanup 周期清理)。
	// 覆盖后保持原 builtin/custom 身份:builtin 不写 sidecar(仍不可删),
	// custom 的 sidecar 不随 rename 移动、本就在位。
	overwriting := false
	backupPath := ""
	if _, statErr := os.Stat(targetPath); statErr == nil {
		overwriting = true
		backupPath = targetPath + ".bak." + time.Now().Format("20060102150405")
		if err := os.Rename(targetPath, backupPath); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("backup existing xml before overwrite: %w", err))
			return
		}
	}

	// 1. tmp 写入 → 原子 rename 上线;失败回滚备份。
	tmpPath := targetPath + ".tmp." + uniqueSuffix()
	defer os.Remove(tmpPath) // 兜底:rename 成功后 tmp 已不存在,Remove 返 ENOENT 无害
	writeErr := os.WriteFile(tmpPath, body, 0o644)
	if writeErr == nil {
		writeErr = os.Rename(tmpPath, targetPath)
	}
	if writeErr != nil {
		if backupPath != "" {
			if rbErr := os.Rename(backupPath, targetPath); rbErr != nil {
				h.logger.Error("rollback backup rename failed; host state inconsistent",
					zap.String("backup_path", backupPath), zap.Error(rbErr))
			}
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("write target: %w", writeErr))
		return
	}

	// 2. sidecar 标记(<neType>.xml.custom 空文件):仅"新建"写入(= custom 可删);
	// 覆盖保持原身份(见上)。失败 → 回滚已写 XML → 500。
	if !overwriting {
		if err := os.WriteFile(targetPath+CustomMarkerSuffix, nil, 0o644); err != nil {
			if rmErr := os.Remove(targetPath); rmErr != nil {
				h.logger.Error("rollback uploaded xml after sidecar write failed; host state inconsistent",
					zap.String("target_path", targetPath), zap.Error(rmErr))
			}
			commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("write sidecar marker: %w", err))
			return
		}
	}

	// 3. 导入 = 写文件 → destructive 全量重载(删孤儿) → RefreshCache,顺序完成。
	// 重载失败必须让上传请求失败并回滚落盘文件 + sidecar(不留半成品),否则形成
	// "201 假成功但 0 行入库"的静默失败(#123);仅孤儿清理 / RefreshCache 失败 Warn 不致命。
	startedAt := time.Now()
	if h.reloader != nil {
		if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
			// 回滚落盘状态:覆盖场景还原 .bak 备份(rename 原子覆盖回原位);
			// 新建场景删除已写 XML 与 sidecar 标记。
			if overwriting {
				if rbErr := os.Rename(backupPath, targetPath); rbErr != nil {
					h.logger.Error("rollback backup rename after reload failure failed; host state inconsistent",
						zap.String("backup_path", backupPath), zap.String("target_path", targetPath), zap.Error(rbErr))
				}
			} else {
				if rmErr := os.Remove(targetPath); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
					h.logger.Error("rollback uploaded xml after reload failure failed; host state inconsistent",
						zap.String("target_path", targetPath), zap.Error(rmErr))
				}
				if rmErr := os.Remove(targetPath + CustomMarkerSuffix); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
					h.logger.Error("rollback sidecar marker after reload failure failed",
						zap.String("sidecar", filepath.Base(targetPath)+CustomMarkerSuffix), zap.Error(rmErr))
				}
			}
			h.logger.Error("audit: alarm upload reload failed; uploaded file rolled back",
				zap.String("audit_action", "alarm.upload.reload_failed"),
				zap.String("loaded_from", loadedFrom), zap.Bool("overwritten", overwriting), zap.Error(err))
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("重载告警定义失败,已回滚上传文件(neType=%s): %w", neType, err))
			return
		}
	}

	var orphansDeleted int64
	if h.service != nil {
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

	// #241:导入成功落库后,主动刷新绑定 alarm_definitions 的字典(alarm_ne_type),
	// 使「新增产品 → 告警名称」下拉即时出现新 neType(无需等 daily cron / 手动刷新)。
	// best-effort:刷新失败只 Warn,不影响 upload 响应。
	h.refreshBoundDict(c.Request.Context(), "alarm_definitions")

	backupName := ""
	if backupPath != "" {
		backupName = filepath.Base(backupPath)
	}
	h.logger.Info("audit: alarm file uploaded",
		zap.String("audit_action", "alarm.upload.success"),
		zap.String("loaded_from", loadedFrom),
		zap.String("ne_type", neType),
		zap.Int("body_size", len(body)),
		zap.Bool("overwritten", overwriting),
		zap.String("backup", backupName),
		zap.Int64("orphans_deleted", orphansDeleted))

	response.OKWithStatus(c, http.StatusCreated, gin.H{
		"uploaded":        true,
		"filename":        filepath.Base(targetPath),
		"loaded_from":     loadedFrom,
		"ne_type":         neType,
		"overwritten":     overwriting,
		"backup":          backupName,
		"reloaded":        true, // 重载失败已在上方 500 返回,走到这里必为 true(字段保留兼容前端)
		"orphans_deleted": orphansDeleted,
	})
}

// parseUploadedNeType 提取上传 XML 的 <alarmModel neType="..."> 属性。
// 2026-06-05 导入 XML 调整:neType 是名称与内容主键的唯一来源(取消手填 name +
// 文件名回退),缺失 / 解析失败返空,由调用方拒绝(400)。
func parseUploadedNeType(raw []byte) string {
	var doc xmlAlarmModel
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(doc.NeType)
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
