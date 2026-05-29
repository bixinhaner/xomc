package indicator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

// FileHandler 提供 T-0180 P1.3+ XML 文件粒度的管理端点(对标 T-0178 parammodel handler):
//   - GET    /api/v1/indicators/summary          — 三制式聚合(builtin/custom 计数 + groups + platforms)
//   - GET    /api/v1/indicators/files?tech=      — 列出该制式所有 XML 文件 + source/deletable + 计数
//   - POST   /api/v1/indicators/upload-xml?tech=&force= — 上传新自定义 XML,自动 Reload
//   - DELETE /api/v1/indicators/files/{*loadedFrom}    — 删除自定义 XML + 级联 DB 清理
//
// 与 rest_handler.go 的 indicator CRUD(指标行粒度)隔离,因为这里管的是文件粒度。
type FileHandler struct {
	repo      FileRepository
	reloader  Reloader      // T-0180 P1.4: Upload/Delete 后同步触发 Loader.Reload(可为 nil → 跳过)
	baseDir   string        // XMLBaseDir,等于 dictloader.XMLBaseDir(e.g. /etc/omcgo/data)
	fileLocks sync.Map      // map[basename(string)]*sync.Mutex
	logger    *zap.Logger
}

// NewFileHandler 构造 FileHandler;
// baseDir 必须为 Loader 用的同一 XMLBaseDir,否则 loadedFrom 相对路径无法 join 到正确绝对路径。
// reloader 可为 nil — 此时 Upload 成功后只 audit log,不触发 Reload(测试场景常用)。
func NewFileHandler(repo FileRepository, reloader Reloader, baseDir string, logger *zap.Logger) *FileHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileHandler{repo: repo, reloader: reloader, baseDir: baseDir, logger: logger.Named("indicator.file")}
}

// RegisterRoutes 挂在 /api/v1 之下;内部使用 /indicators/... 多个子路径。
//
// gin 路由的 *loadedFrom 是 catch-all wildcard,匹配剩余完整路径(含 /),
// 用于承载 "indicator-library-custom/enb/MY.xml" 形式的多段相对路径。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ig := rg.Group("/indicators")
	ig.GET("/summary", h.Summary)
	ig.GET("/files", h.ListFiles)
	ig.POST("/upload-xml", h.UploadXML)
	files := ig.Group("/files")
	files.DELETE("/*loadedFrom", h.DeleteFile)
}

// acquireFileLock 取/建一个 per-basename mutex,返回 unlock 函数。
// 对标 T-0178 parammodel/handler.go::acquireFileLock — Upload(P1.4) /
// Delete(本文件)/ 单文件 Reload 共享同一把锁,防同名文件并发写半截。
func (h *FileHandler) acquireFileLock(basename string) func() {
	mu, _ := h.fileLocks.LoadOrStore(basename, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

// allowedFileTechs 是 file path 中 tech 段的合法值。
// 与 file_repository.go::allowedTech 一致,但额外暴露集合用于路径解析层校验。
var allowedFileTechs = map[string]struct{}{
	"enb": {},
	"gsm": {},
	"gnb": {},
}

// parseCustomTech 从 loaded_from 路径推断 tech 子目录。
//
// 期望格式:CustomDirSubdir/<tech>/<file>.xml(三段)
// 例如:indicator-library-custom/enb/MY.xml → ("enb", nil)
//
// 校验:
//   - 必须以 CustomDirPrefix 开头(防 builtin 误入)
//   - 必须恰好三段(防多级嵌套绕过)
//   - tech 必须在 enb/gsm/gnb 集合
//   - 第三段不能含路径分隔符(防路径遍历)
func parseCustomTech(loadedFrom string) (string, error) {
	if !strings.HasPrefix(loadedFrom, CustomDirPrefix) {
		return "", fmt.Errorf("not a custom indicator path: %q (expected %s prefix)", loadedFrom, CustomDirPrefix)
	}
	parts := strings.Split(loadedFrom, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid custom path depth (expected 3 segments, got %d): %q", len(parts), loadedFrom)
	}
	tech := parts[1]
	if _, ok := allowedFileTechs[tech]; !ok {
		return "", fmt.Errorf("invalid tech segment %q in %q", tech, loadedFrom)
	}
	// 第三段必须是单一 xml 文件名,不含 / 与 ..(parts[2] 已经是单段,但再防一遍)
	base := parts[2]
	if base == "" || strings.Contains(base, "..") {
		return "", fmt.Errorf("invalid filename %q in %q", base, loadedFrom)
	}
	return tech, nil
}

// DeleteFile DELETE /api/v1/indicators/files/{loadedFrom}
//
// 守门链(对标 T-0178 parammodel DeleteModel):
//  1. URL 解码 + 去前导 / 得到 loadedFrom 相对路径
//  2. IsDeletable(loadedFrom) → false 返 403 + ErrCodeIndicatorBuiltinNotDeletable
//  3. parseCustomTech → 推断 tech (enb/gsm/gnb)
//  4. CountByLoadedFrom → 0 行 且 文件也不存在 → 404
//  5. acquireFileLock(basename) 持锁直到 return
//  6. 物理 rename "<file>.deleted.<14位ts>" 备份;失败保守回滚返 500 + ErrCodeIndicatorBackupFailed
//  7. DB 单事务级联 DELETE formula/enabled/main(三表),失败 rename 回滚
//  8. 审计日志(audit_action) + 返回 200 + {deleted, loaded_from, backup, rows_affected}
func (h *FileHandler) DeleteFile(c *gin.Context) {
	// 1. gin *loadedFrom 会带前导 /,去掉再 normalize 一次防双 //
	raw := strings.TrimPrefix(c.Param("loadedFrom"), "/")
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if loadedFrom == "" || loadedFrom == "." {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("loaded_from path is required"))
		return
	}

	// 2. 守门:仅 SourceCustom 可删
	if !IsDeletable(loadedFrom) {
		h.logger.Info("audit: indicator file delete rejected (builtin or unknown)",
			zap.String("audit_action", "indicator.delete.rejected_builtin"),
			zap.String("loaded_from", loadedFrom),
			zap.String("source", string(ClassifySource(loadedFrom))))
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("indicator XML %q (source=%s) is not deletable; "+
				"to remove a builtin file, delete it from data/indicator-library/ in the release image "+
				"and re-deploy [code=%d]",
				loadedFrom, ClassifySource(loadedFrom), global.ErrCodeIndicatorBuiltinNotDeletable))
		return
	}

	// 3. 推断 tech(再校验一次路径深度 + 防遍历)
	tech, err := parseCustomTech(loadedFrom)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 4. 存在性校验:DB 0 行 + 文件不在 → 404;DB 0 行但文件在 → 仍允许删(残留清理)
	rowCount, err := h.repo.CountByLoadedFrom(c.Request.Context(), tech, loadedFrom)
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

	// 5. per-filename 互斥锁(Upload P1.4 共用)
	unlock := h.acquireFileLock(filepath.Base(loadedFrom))
	defer unlock()

	// 6. 物理备份 — rename 原子操作
	backupPath := absPath + ".deleted." + time.Now().Format("20060102150405")
	if renameErr := os.Rename(absPath, backupPath); renameErr != nil {
		switch {
		case errors.Is(renameErr, fs.ErrNotExist):
			// 文件已被外部 rm / 上次中断的 Delete → 视为无需备份,继续清 DB 残留
			h.logger.Warn("audit: custom xml already gone before delete; proceeding to remove DB rows",
				zap.String("audit_action", "indicator.delete.custom.file_already_gone"),
				zap.String("loaded_from", loadedFrom),
				zap.String("path", absPath))
			backupPath = "" // 标记无备份
		default:
			// EACCES / ENOSPC / EROFS / 其他 IO → 保守回滚,绝不删 DB
			h.logger.Error("audit: indicator file delete aborted (backup failed)",
				zap.String("audit_action", "indicator.delete.aborted_backup_failed"),
				zap.String("loaded_from", loadedFrom),
				zap.String("path", absPath),
				zap.Error(renameErr))
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("backup custom xml failed; refusing to delete DB row to avoid data loss; "+
					"please fix the filesystem (check disk space / permissions / mount RO state) and retry: %w [code=%d]",
					renameErr, global.ErrCodeIndicatorBackupFailed))
			return
		}
	}

	// 7. DB 级联删除,失败回滚文件备份
	rowsAffected, dbErr := h.repo.DeleteByLoadedFrom(c.Request.Context(), tech, loadedFrom)
	if dbErr != nil {
		// 反向 rename(忽略 rename 失败 — 此时只能 log,DB 与 host 已分叉)
		if backupPath != "" {
			if rbErr := os.Rename(backupPath, absPath); rbErr != nil {
				h.logger.Error("rollback rename failed; DB and host state diverged",
					zap.String("backup_path", backupPath),
					zap.String("orig_path", absPath),
					zap.Error(rbErr))
			}
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, dbErr)
		return
	}

	// 8. 审计日志 + 响应
	backupName := ""
	if backupPath != "" {
		backupName = filepath.Base(backupPath)
	}
	h.logger.Info("audit: indicator file deleted (custom)",
		zap.String("audit_action", "indicator.delete.custom"),
		zap.String("loaded_from", loadedFrom),
		zap.String("tech", tech),
		zap.Int("rows_affected", rowsAffected),
		zap.String("backup", backupName))

	response.OK(c, gin.H{
		"deleted":       true,
		"loaded_from":   loadedFrom,
		"tech":          tech,
		"rows_affected": rowsAffected,
		"backup":        backupName,
	})
}

// ── T-0180 P1.4: Summary / ListFiles / UploadXML ─────────────────────────────

// Summary GET /api/v1/indicators/summary
//
// 返回三制式聚合行(enb/gsm/gnb),供一级页面 SummaryTab 渲染:
// indicators 总数 / builtin / custom / unknown / groups / platforms[]
func (h *FileHandler) Summary(c *gin.Context) {
	rows, err := h.repo.SummaryByTech(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": rows})
}

// fileEntry 是 ListFiles 端点单行(DB 计数 + 物理扫描 + source/deletable 派生)。
type fileEntry struct {
	LoadedFrom string `json:"loaded_from"`
	Source     string `json:"source"`       // builtin / custom / unknown
	Deletable  bool   `json:"deletable"`    // 前端零代码渲染,直接绑定 Tag/Button.disabled
	Count      int    `json:"count"`        // perf_indicators_<tech>.WHERE loaded_from=? 行数
	OnDisk     bool   `json:"on_disk"`      // 物理文件是否在 baseDir 下存在(uploaded-but-not-loaded 场景=true 但 count=0)
}

// ListFiles GET /api/v1/indicators/files?tech=enb|gsm|gnb
//
// 行为(对标 PRD §2.3 "管理 XML 文件" Modal):
//  1. 从 DB GROUP BY loaded_from 拉所有已入库文件 + indicator 计数
//  2. 物理扫两个目录(builtin + custom)合并补 uploaded-but-not-loaded 的文件(count=0)
//  3. 每行派生 Source(ClassifySource) + Deletable(IsDeletable),前端不重新推导
//  4. 按 loaded_from 字典序排序
func (h *FileHandler) ListFiles(c *gin.Context) {
	tech := c.Query("tech")
	if err := validateUploadTech(tech); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dbRows, err := h.repo.ListFilesByTech(c.Request.Context(), tech)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// 索引便于合并(键 = loadedFrom)
	merged := make(map[string]*fileEntry, len(dbRows)+8)
	for _, fg := range dbRows {
		entry := &fileEntry{
			LoadedFrom: fg.LoadedFrom,
			Source:     string(ClassifySource(fg.LoadedFrom)),
			Deletable:  IsDeletable(fg.LoadedFrom),
			Count:      fg.Count,
		}
		// DB 中 loaded_from 不为空,Stat 一下判 on_disk
		if fg.LoadedFrom != "" {
			if _, err := os.Stat(filepath.Join(h.baseDir, fg.LoadedFrom)); err == nil {
				entry.OnDisk = true
			}
		}
		merged[fg.LoadedFrom] = entry
	}

	// 合并物理扫描(只关心 custom 侧的 uploaded-but-not-loaded;builtin 不会有未入库的)
	customSubdir := filepath.Join(CustomDirSubdir, tech)
	customSrcs, err := scanXMLBasenamesOptional(filepath.Join(h.baseDir, customSubdir))
	if err != nil {
		// 目录扫描失败不阻塞 — DB 行已返,仅 log
		h.logger.Warn("scan custom subdir failed; returning DB rows only",
			zap.String("custom_subdir", customSubdir), zap.Error(err))
	} else {
		for _, name := range customSrcs {
			lf := filepath.ToSlash(filepath.Join(customSubdir, name))
			if existing, ok := merged[lf]; ok {
				existing.OnDisk = true
				continue
			}
			// uploaded-but-not-loaded:DB 0 行 + 物理文件在
			merged[lf] = &fileEntry{
				LoadedFrom: lf,
				Source:     string(SourceCustom),
				Deletable:  true,
				Count:      0,
				OnDisk:     true,
			}
		}
	}

	// 输出按 LoadedFrom 字典序排序
	out := make([]fileEntry, 0, len(merged))
	for _, e := range merged {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LoadedFrom < out[j].LoadedFrom
	})
	response.OK(c, gin.H{"items": out, "tech": tech})
}

// UploadXML POST /api/v1/indicators/upload-xml?tech=enb|gsm|gnb[&force=true|false]
//
// multipart/form-data;字段 file = 上传的 XML 字节流。
//
// 守门链(顺序敏感,任一失败即 400/409,审计明确拒绝原因):
//  1. ?tech= 校验在 enb/gsm/gnb
//  2. file part 必填 + file.Size <= MaxUploadXMLSize
//  3. 文件名 filepath.Base + uploadFilenamePattern 白名单
//  4. XML 内容 validateUploadXML(body, tech):根 = <indicatorModel> +
//     platform 必填 + deviceType(若 present)与 tech 一致
//  5. 路径 filepath.Join(baseDir, customSubdir, base) 经 pathContainedIn 二次校验不逃逸
//
// 写入流程(全程 per-filename 锁):
//  1. MkdirAll customDir(首次上传场景)
//  2. 同名检查:存在但无 ?force=true → 409 Conflict
//  3. 写 tmp 文件 = targetPath + .tmp.<uuid>(原子写第一步)
//  4. 若 overwrite:原文件 mv 到 .bak.<14位ts>
//  5. tmp rename → targetPath(原子提交)
//  6. 同步触发 Reloader.ReloadOne — Loader 重扫两目录入库;
//     Reload 失败仅 Warn,Upload 不回滚(用户可手动 Reload 重试)
//  7. audit log + 200 响应 + filename / loaded_from / backup / reloaded
func (h *FileHandler) UploadXML(c *gin.Context) {
	tech := c.Query("tech")
	if err := validateUploadTech(tech); err != nil {
		h.logger.Info("audit: indicator upload rejected (invalid tech)",
			zap.String("audit_action", "indicator.upload.rejected_invalid_tech"),
			zap.String("tech", tech))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeIndicatorUploadInvalidTech))
		return
	}
	force := c.Query("force") == "true"

	fh, err := c.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("form field \"file\" is required: %w", err))
		return
	}
	if fh.Size > MaxUploadXMLSize {
		h.logger.Info("audit: indicator upload rejected (too large)",
			zap.String("audit_action", "indicator.upload.rejected_too_large"),
			zap.Int64("size", fh.Size),
			zap.Int64("max", MaxUploadXMLSize))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("upload size %d exceeds max %d [code=%d]",
				fh.Size, MaxUploadXMLSize, global.ErrCodeIndicatorUploadTooLarge))
		return
	}

	base := filepath.Base(fh.Filename)
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: indicator upload rejected (invalid filename)",
			zap.String("audit_action", "indicator.upload.rejected_invalid_name"),
			zap.String("filename", fh.Filename))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeIndicatorUploadInvalidName))
		return
	}

	// 读 body
	src, err := fh.Open()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("open upload: %w", err))
		return
	}
	body, err := readAllLimited(src, MaxUploadXMLSize)
	_ = src.Close()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("read upload: %w", err))
		return
	}

	if err := validateUploadXML(body, tech); err != nil {
		h.logger.Info("audit: indicator upload rejected (invalid xml)",
			zap.String("audit_action", "indicator.upload.rejected_invalid_xml"),
			zap.String("filename", base),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeIndicatorUploadInvalidRoot))
		return
	}

	// 路径计算 + 二次防遍历
	customDir := filepath.Join(h.baseDir, CustomDirSubdir, tech)
	targetPath := filepath.Join(customDir, base)
	if !pathContainedIn(customDir, targetPath) {
		h.logger.Error("audit: indicator upload rejected (path traversal)",
			zap.String("audit_action", "indicator.upload.rejected_path_traversal"),
			zap.String("filename", base),
			zap.String("target", targetPath))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("computed target path %q escapes custom dir %q", targetPath, customDir))
		return
	}

	// per-filename 互斥锁(与 DeleteFile 共用)
	unlock := h.acquireFileLock(base)
	defer unlock()

	if err := os.MkdirAll(customDir, 0o755); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("mkdir custom dir: %w", err))
		return
	}

	// 同名检查
	_, statErr := os.Stat(targetPath)
	overwrite := false
	if statErr == nil {
		if !force {
			h.logger.Info("audit: indicator upload conflict (same-name exists)",
				zap.String("audit_action", "indicator.upload.conflict"),
				zap.String("filename", base),
				zap.String("tech", tech))
			commonerrors.AbortWithError(c, http.StatusConflict,
				fmt.Errorf("file %q already exists; pass ?force=true to overwrite (will backup to .bak.<ts>)", base))
			return
		}
		overwrite = true
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("stat target: %w", statErr))
		return
	}

	// 原子写:tmp.<uuid> → 验完 → rename
	tmpPath := targetPath + ".tmp." + uniqueSuffix()
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("write tmp: %w", err))
		return
	}
	// tmp 写成功后,若 overwrite 走 .bak.<ts>;否则直接 rename
	var backupPath string
	if overwrite {
		backupPath = targetPath + ".bak." + time.Now().Format("20060102150405")
		if err := os.Rename(targetPath, backupPath); err != nil {
			_ = os.Remove(tmpPath) // 清 tmp
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("backup existing file: %w", err))
			return
		}
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		// rollback:把 .bak 复位
		if backupPath != "" {
			_ = os.Rename(backupPath, targetPath)
		}
		_ = os.Remove(tmpPath)
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("rename tmp to target: %w", err))
		return
	}

	loadedFrom := filepath.ToSlash(filepath.Join(CustomDirSubdir, tech, base))

	// 同步触发 Loader.Reload(让 DB 立即可见新指标)
	reloadOK := true
	if h.reloader != nil {
		if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
			reloadOK = false
			h.logger.Warn("audit: indicator upload reload failed (file written, DB not refreshed)",
				zap.String("audit_action", "indicator.upload.reload_failed"),
				zap.String("loaded_from", loadedFrom),
				zap.Error(err))
		}
	}

	backupName := ""
	if backupPath != "" {
		backupName = filepath.Base(backupPath)
	}
	h.logger.Info("audit: indicator file uploaded",
		zap.String("audit_action", "indicator.upload.success"),
		zap.String("loaded_from", loadedFrom),
		zap.String("tech", tech),
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
		"tech":        tech,
		"overwrite":   overwrite,
		"backup":      backupName,
		"reloaded":    reloadOK,
	})
}

// readAllLimited 是 io.ReadAll 的限长版本,防 multipart header 声明小但实际 body 大的攻击。
// 实际再读 max+1 字节,如果读到 max+1 字节表示超限。
func readAllLimited(r io.Reader, max int64) ([]byte, error) {
	limited := io.LimitReader(r, max+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > max {
		return nil, fmt.Errorf("upload exceeds max %d bytes", max)
	}
	return buf, nil
}

// uniqueSuffix 生成 tmp 文件唯一后缀(纳秒时间戳;高频并发场景下足以避免同名)。
func uniqueSuffix() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// 编译期接口契约:确保 PgFileRepository 实现 FileRepository。
var _ FileRepository = (*PgFileRepository)(nil)

// EnsureBaseDir 是测试与 cmd/app 启动期共用的 helper,
// 确保 baseDir/<CustomDirSubdir>/{enb,gsm,gnb}/ 目录树存在(0755)。
// 若 host bind mount 已就位则为 no-op。
//
// 不在 RegisterRoutes 路径上调用 — cmd/app 启动期由 provider 显式调用。
func EnsureBaseDir(ctx context.Context, baseDir string) error {
	_ = ctx // 预留,目前无 IO 阻塞
	for _, tech := range []string{"enb", "gsm", "gnb"} {
		dir := filepath.Join(baseDir, CustomDirSubdir, tech)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("ensure custom dir %s: %w", dir, err)
		}
	}
	return nil
}
