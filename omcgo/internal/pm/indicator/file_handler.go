package indicator

import (
	"bytes"
	"context"
	"encoding/xml"
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

// CacheRefresher 抽象 indicator 缓存刷新(底层 IndicatorManagementService.BumpCacheVersion)。
// 上传流程在 destructive 重载成功后调用它,触发跨实例缓存失效。
// 可为 nil — 此时跳过缓存刷新(测试场景常用)。
type CacheRefresher interface {
	BumpCacheVersion(ctx context.Context)
}

// FileHandler 提供 XML 文件粒度的管理端点:
//   - GET    /api/v1/indicators/summary          — 三制式聚合(平台计数 + groups + platforms)
//   - GET    /api/v1/indicators/files?tech=      — 列出该制式所有 XML 文件 + source/deletable + 计数
//   - PUT    /api/v1/indicators/file-description — 按 (tech, platform) upsert 描述
//   - POST   /api/v1/indicators/upload-xml?tech=&force= — 上传 XML(写 builtin 目录 → destructive 重载 → 刷新缓存)
//   - DELETE /api/v1/indicators/files/{*loadedFrom}    — 删除 XML + 级联 DB 清理
//
// 与 rest_handler.go 的 indicator CRUD(指标行粒度)隔离,因为这里管的是文件粒度。
//
// 2026-06-03 用户决策:取消 builtin/custom 区分,上传直接写 builtin 目录(loader 扫描的同一目录),
// 接受升级丢失;导入 = 写文件 + destructive 重载(删孤儿)+ 刷新缓存三步在上传端点内顺序完成。
type FileHandler struct {
	repo          FileRepository
	reloader      Reloader            // Upload/Delete 后同步触发 Loader.Reload(可为 nil → 跳过)
	cache         CacheRefresher      // Upload 重载成功后刷新 indicator 缓存(可为 nil → 跳过)
	dictRefresher DictSourceRefresher // #241: 导入成功后刷绑定字典(可为 nil → 跳过)
	baseDir       string              // XMLBaseDir,等于 dictloader.XMLBaseDir(e.g. /etc/omcgo/data)
	fileLocks     sync.Map            // map[basename(string)]*sync.Mutex
	logger        *zap.Logger
}

// DictSourceRefresher 在导入 XML 成功后按 source_table 刷新绑定字典(T-0182 / #241)。
// best-effort:刷新失败不阻断导入(daily cron 兜底)。可为 nil(未接入时整段跳过)。
type DictSourceRefresher interface {
	RefreshSourceBoundByTable(ctx context.Context, sourceTable string) (int, error)
}

// NewFileHandler 构造 FileHandler;
// baseDir 必须为 Loader 用的同一 XMLBaseDir,否则 loadedFrom 相对路径无法 join 到正确绝对路径。
// reloader / cache 可为 nil — 此时 Upload 成功后只 audit log,不触发 Reload / 缓存刷新(测试场景常用)。
// dictRefresher 可为 nil(未接入数据字典数据源时);非 nil 时 enb 导入成功后刷 rela_platform_indicator_formula_enb 绑定字典。
func NewFileHandler(repo FileRepository, reloader Reloader, cache CacheRefresher, dictRefresher DictSourceRefresher, baseDir string, logger *zap.Logger) *FileHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileHandler{repo: repo, reloader: reloader, cache: cache, dictRefresher: dictRefresher, baseDir: baseDir, logger: logger.Named("indicator.file")}
}

// RegisterRoutes 挂在 /api/v1 之下;内部使用 /indicators/... 多个子路径。
//
// gin 路由的 *loadedFrom 是 catch-all wildcard,匹配剩余完整路径(含 /),
// 用于承载 "indicator-library-custom/enb/MY.xml" 形式的多段相对路径。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ig := rg.Group("/indicators")
	ig.GET("/summary", h.Summary)
	ig.PUT("/file-description", h.UpdateFileDescription)
	ig.GET("/files", h.ListFiles)
	// 下载用 ?loaded_from= query(而非 wildcard path),避免与 GET /files 冲突且免编码斜杠。
	ig.GET("/file-content", h.DownloadFile)
	ig.POST("/upload-xml", h.UploadXML)
	files := ig.Group("/files")
	files.DELETE("/*loadedFrom", h.DeleteFile)
}

// DownloadFile GET /api/v1/indicators/file-content?loaded_from=indicator-library/[enb/]<file>.xml
//
// 下载 XML 原文件(2026-06-05 操作列下载功能)。builtin 与 custom 均可下载(只读无守门);
// 路径校验与 DeleteFile 同口径:parseFileTechs(前缀/深度/防遍历)+ pathContainedIn 双重防御。
func (h *FileHandler) DownloadFile(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("loaded_from"))
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if raw == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("loaded_from query parameter is required"))
		return
	}
	if _, err := parseFileTechs(loadedFrom); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	absPath := filepath.Join(h.baseDir, loadedFrom)
	if !pathContainedIn(filepath.Join(h.baseDir, BuiltinDirSubdir), absPath) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("path %q escapes indicator dir", loadedFrom))
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

// parseFileTechs 从 loaded_from 路径推断候选 tech 集合(enb/gsm/gnb)。
//
// 目录调整(2026-06-05):仅 ENB 分子目录,GSM/GNB 不分目录直接落根级:
//   - indicator-library/<tech>/<file>.xml → [<tech>](子目录;enb 为现行落地,
//     gsm/gnb 子目录保留兼容历史数据)
//   - indicator-library/GSM.xml           → ["gsm"](根级出厂单文件,大小写不敏感)
//   - indicator-library/GNB.xml           → ["gnb"](根级出厂单文件)
//   - indicator-library/<other>.xml       → ["gsm","gnb"](根级自定义上传,路径无法
//     区分制式,由调用方逐表查/删 —— 单文件只会落一张表,无副作用)
//
// 校验:必须以 BuiltinDirSubdir/ 开头 + 防多级嵌套绕过 + tech 白名单 +
// 文件名不含 ..(防路径遍历)。
func parseFileTechs(loadedFrom string) ([]string, error) {
	parts := strings.Split(loadedFrom, "/")
	for _, p := range parts {
		if p == "" || strings.Contains(p, "..") {
			return nil, fmt.Errorf("invalid path segment in %q", loadedFrom)
		}
	}
	prefix := BuiltinDirSubdir + "/"
	if !strings.HasPrefix(loadedFrom, prefix) {
		return nil, fmt.Errorf("not an indicator XML path: %q (expected %s prefix)", loadedFrom, prefix)
	}
	switch len(parts) {
	case 3:
		// indicator-library/<tech>/<file>.xml(制式子目录)
		tech := parts[1]
		if _, ok := allowedFileTechs[tech]; !ok {
			return nil, fmt.Errorf("invalid tech segment %q in %q", tech, loadedFrom)
		}
		return []string{tech}, nil
	case 2:
		// 根级:出厂单文件按文件名定 tech;其余为 GSM/GNB 自定义上传(候选双表)
		if !strings.EqualFold(filepath.Ext(parts[1]), ".xml") {
			return nil, fmt.Errorf("unrecognized root-level file %q (expected *.xml)", loadedFrom)
		}
		switch strings.ToLower(parts[1]) {
		case "gsm.xml":
			return []string{"gsm"}, nil
		case "gnb.xml":
			return []string{"gnb"}, nil
		default:
			return []string{"gsm", "gnb"}, nil
		}
	default:
		return nil, fmt.Errorf("invalid path depth (%d segments): %q", len(parts), loadedFrom)
	}
}

// DeleteFile DELETE /api/v1/indicators/files/{loadedFrom}
//
// 2026-06-03 用户决策:取消 builtin/custom 区分,所有文件可删(无来源守门)。
//
// 流程:
//  1. URL 解码 + 去前导 / 得到 loadedFrom 相对路径
//  2. parseFileTech → 推断 tech (enb/gsm/gnb) + 路径深度/防遍历校验
//  3. CountByLoadedFrom → 0 行 且 文件也不存在 → 404
//  4. acquireFileLock(basename) 持锁直到 return
//  5. 物理 rename "<file>.deleted.<14位ts>" 备份;失败保守回滚返 500 + ErrCodeIndicatorBackupFailed
//  6. DB 单事务级联 DELETE formula/enabled/main(三表),失败 rename 回滚
//  7. 审计日志(audit_action) + 返回 200 + {deleted, loaded_from, backup, rows_affected}
func (h *FileHandler) DeleteFile(c *gin.Context) {
	// 1. gin *loadedFrom 会带前导 /,去掉再 normalize 一次防双 //
	raw := strings.TrimPrefix(c.Param("loadedFrom"), "/")
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	if loadedFrom == "" || loadedFrom == "." {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("loaded_from path is required"))
		return
	}

	// 2. 推断候选 tech 集合(校验路径深度 + 前缀 + 防遍历)。
	// 根级自定义文件路径无法区分 gsm/gnb → 候选双表逐一查/删(单文件只落一张表)。
	techs, err := parseFileTechs(loadedFrom)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 3. 内置数据守门:builtin(当前目录 XML 加载)+ unknown 不可删(2026-06-04 用户决策)
	if !IsDeletable(h.baseDir, loadedFrom) {
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("内置数据不允许删除(loaded_from=%q)[code=%d]", loadedFrom, global.ErrCodeIndicatorBuiltinNotDeletable))
		return
	}

	// 4. 存在性校验:DB 0 行 + 文件不在 → 404;DB 0 行但文件在 → 仍允许删(残留清理)
	rowCount := 0
	tech := techs[0] // 命中行的 tech;无行时回退首候选(仅用于日志/响应)
	for _, t := range techs {
		n, cntErr := h.repo.CountByLoadedFrom(c.Request.Context(), t, loadedFrom)
		if cntErr != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, cntErr)
			return
		}
		if n > 0 {
			tech = t
		}
		rowCount += n
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

	// 7. DB 级联删除(候选 tech 逐表;单文件只会落一张表),失败回滚文件备份
	rowsAffected := 0
	var dbErr error
	for _, t := range techs {
		n, delErr := h.repo.DeleteByLoadedFrom(c.Request.Context(), t, loadedFrom)
		if delErr != nil {
			dbErr = delErr
			break
		}
		if n > 0 {
			tech = t
		}
		rowsAffected += n
	}
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

	// DB 行已删,移除 sidecar 标记(X.xml.custom)。XML 本体已 rename 为 .deleted.<ts> 备份,
	// sidecar 不再有判定意义;残留会让孤儿 sidecar 累积。ENOENT 容忍(可能从未写过)。
	if rmErr := os.Remove(absPath + CustomMarkerSuffix); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		h.logger.Warn("remove sidecar marker failed (non-fatal)",
			zap.String("loaded_from", loadedFrom),
			zap.String("sidecar", filepath.Base(absPath)+CustomMarkerSuffix),
			zap.Error(rmErr))
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
	// source / deletable 据 sidecar 回填(repo 不做文件 IO);
	// deletable 供一级列表删除按钮渲染(内置置灰,2026-06-05)。
	for i := range rows {
		rows[i].Source = string(ClassifySource(h.baseDir, rows[i].LoadedFrom))
		rows[i].Deletable = IsDeletable(h.baseDir, rows[i].LoadedFrom)
	}
	response.OK(c, gin.H{"items": rows})
}

// UpdateFileDescription PUT /api/v1/indicators/file-description
//
// body: { "tech": "enb", "platform": "ALL", "description": "..." }
// 按 (tech, platform) 维度 upsert 一条描述(2026-06-02 用户决策:一个平台一条)。
// 描述属运维注记,不动 XML 文件本身。
func (h *FileHandler) UpdateFileDescription(c *gin.Context) {
	var req struct {
		Tech        string `json:"tech" binding:"required"`
		Platform    string `json:"platform" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := validateUploadTech(req.Tech); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.repo.UpsertFileDescription(c.Request.Context(), req.Tech, req.Platform, req.Description); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"tech": req.Tech, "platform": req.Platform, "description": req.Description})
}

// fileEntry 是 ListFiles 端点单行(DB 计数 + 物理扫描 + source/deletable 派生)。
type fileEntry struct {
	LoadedFrom string `json:"loaded_from"`
	Source     string `json:"source"`    // builtin / custom / unknown
	Deletable  bool   `json:"deletable"` // 前端零代码渲染,直接绑定 Tag/Button.disabled
	Count      int    `json:"count"`     // perf_indicators_<tech>.WHERE loaded_from=? 行数
	OnDisk     bool   `json:"on_disk"`   // 物理文件是否在 baseDir 下存在(uploaded-but-not-loaded 场景=true 但 count=0)
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
			Source:     string(ClassifySource(h.baseDir, fg.LoadedFrom)),
			Deletable:  IsDeletable(h.baseDir, fg.LoadedFrom),
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

	// 合并物理扫描(补 uploaded-but-not-loaded 文件)。目录调整(2026-06-05):
	//   - ENB:indicator-library/enb/ 子目录
	//   - GSM/GNB:indicator-library/ 根级(出厂单文件按文件名、自定义上传按 XML
	//     deviceType 属性分类制式)
	// sidecar 判来源/可删,前端零代码同步。
	for _, sf := range h.scanTechFiles(tech) {
		if existing, ok := merged[sf.LoadedFrom]; ok {
			existing.OnDisk = true
			continue
		}
		// uploaded-but-not-loaded:DB 0 行 + 物理文件在;source/deletable 由 sidecar 判定
		merged[sf.LoadedFrom] = &fileEntry{
			LoadedFrom: sf.LoadedFrom,
			Source:     string(ClassifySource(h.baseDir, sf.LoadedFrom)),
			Deletable:  IsDeletable(h.baseDir, sf.LoadedFrom),
			Count:      0,
			OnDisk:     true,
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

// scanTechFiles 物理扫描指定制式的 XML 文件清单(ListFiles 的盘面合并源)。
//
// 目录调整(2026-06-05):ENB 扫 enb/ 子目录;GSM/GNB 扫根级并按
// classifyRootIndicatorTech(出厂文件名 / XML deviceType 属性)过滤本制式。
// 扫描失败不致命 — 调用方仍返 DB 行,这里仅 log。
func (h *FileHandler) scanTechFiles(tech string) []fileSource {
	if tech == ENBSubdirName {
		subdir := filepath.Join(BuiltinDirSubdir, ENBSubdirName)
		names, err := scanXMLBasenamesOptional(filepath.Join(h.baseDir, subdir))
		if err != nil {
			h.logger.Warn("scan enb subdir failed; returning DB rows only",
				zap.String("subdir", subdir), zap.Error(err))
			return nil
		}
		out := make([]fileSource, 0, len(names))
		for _, name := range names {
			out = append(out, fileSource{
				AbsPath:    filepath.Join(h.baseDir, subdir, name),
				LoadedFrom: filepath.ToSlash(filepath.Join(subdir, name)),
			})
		}
		return out
	}
	srcs, err := resolveRootTechSources(h.baseDir, BuiltinDirSubdir, tech)
	if err != nil {
		h.logger.Warn("scan root indicator dir failed; returning DB rows only",
			zap.String("tech", tech), zap.Error(err))
		return nil
	}
	return srcs
}

// resolveUploadTarget 计算上传目标的目录、目标绝对路径与 loaded_from 相对路径。
//
// 目录调整(2026-06-05):仅 ENB 分子目录,GSM/GNB 不分目录直接落根级:
//   - ENB → indicator-library/enb/<name>.xml
//   - GSM → indicator-library/<name>.xml(根级;Loader 按 XML deviceType 分类)
//   - GNB → indicator-library/<name>.xml(根级)
//
// 返回:targetDir(MkdirAll 目标)、targetPath(最终文件)、loadedFrom(slash 相对路径)。
func (h *FileHandler) resolveUploadTarget(tech, base string) (targetDir, targetPath, loadedFrom string) {
	targetDir = filepath.Join(h.baseDir, BuiltinDirSubdir)
	if tech == ENBSubdirName {
		targetDir = filepath.Join(targetDir, ENBSubdirName)
	}
	targetPath = filepath.Join(targetDir, base)
	rel, _ := filepath.Rel(h.baseDir, targetPath)
	loadedFrom = filepath.ToSlash(rel)
	return targetDir, targetPath, loadedFrom
}

// UploadXML POST /api/v1/indicators/upload-xml?tech=enb|gsm|gnb[&force=true]
//
// multipart/form-data;字段 file = XML 字节流。
//
// 名称取消手填(2026-06-05 导入 XML 调整):唯一名称取自 XML
// <indicatorModel platform="..."> 属性,文件名 = <platform>.xml;落地目录仅 ENB
// 分子目录(enb/),GSM/GNB 直接落 indicator-library/ 根级。上传文件自身的 filename 被忽略。
//
// 重复允许覆盖(2026-06-05 调整,需前端二次确认):
//   - 重复判定:platform 已在该 tech 的 formula 表(LoadedFromsByPlatform 定位归属文件,
//     如 GSM 出厂 platform=BSC 归属 GSM.xml)或目标路径已在盘上
//   - 无 &force=true → 409,响应 data.overwritable=true,前端弹确认框
//   - &force=true → 覆盖归属文件(原文件先备份 .bak.<ts>);覆盖内置保持 builtin 身份
//     (不写 sidecar,仍不可删),覆盖自定义保持 custom
//
// 守门链(顺序敏感,任一失败即 400/409,审计明确拒绝原因):
//  1. ?tech= 校验在 enb/gsm/gnb
//  2. file part 必填 + file.Size <= MaxUploadXMLSize
//  3. XML 内容 validateUploadXML(body, tech):根 = <indicatorModel> +
//     platform 必填 + deviceType 与 tech 一致(GSM/GNB 必填,ENB 缺省容忍)
//  4. 名称:validateUploadFilename(<platform>+".xml") 白名单正则
//  5. 目标路径经 pathContainedIn 二次校验不逃逸目标目录
//  6/7. 重复 + force 判定(见上)
//
// 写入流程(全程 per-filename 锁):
//  1. MkdirAll targetDir(首次上传场景)
//  2. [覆盖时先备份旧文件 .bak.<ts> →] 写 tmp → 原子 rename 上线(失败回滚备份)
//  3. sidecar(<platform>.xml.custom 空标记)仅新建写入;失败回滚 XML → 500
//  4. destructive 重载(PerformReloadWithOrphans:全量 UPSERT + 删孤儿)+ BumpCacheVersion
//     重载/缓存失败仅 Warn,Upload 不回滚(用户可重试),响应体反映 reloaded 状态
//  5. audit log + 201 响应 + filename / loaded_from / platform / overwritten / reloaded / orphans
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
			zap.String("filename", fh.Filename),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w [code=%d]", err, global.ErrCodeIndicatorUploadInvalidRoot))
		return
	}

	// 名称提取 —— 唯一来源 XML <indicatorModel platform="..."> 属性(取消手填 name);
	// 内容主键与目标文件名同源,与 Loader xmlIndicatorModel 同口径。
	platform := parseUploadPlatform(body)
	if platform == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("indicatorModel XML 缺少 platform 属性,无法确定名称 [code=%d]",
				global.ErrCodeIndicatorUploadInvalidRoot))
		return
	}
	base := platform + ".xml"
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: indicator upload rejected (invalid platform-derived name)",
			zap.String("audit_action", "indicator.upload.rejected_invalid_name"),
			zap.String("platform", platform))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("platform「%s」不能作为文件名:%w [code=%d]",
				platform, err, global.ErrCodeIndicatorUploadInvalidName))
		return
	}

	force := c.Query("force") == "true"

	// 校验 6/7: 定位目标文件 + 重复判定。
	// 内容主键(platform)已在该 tech 的 formula 表 → 归属文件即覆盖目标
	// (如 GSM 出厂 platform=BSC 归属 indicator-library/GSM.xml);
	// 否则按制式落派生路径,盘上已存在(uploaded-but-not-loaded)同样视为重复。
	libDir := filepath.Join(h.baseDir, BuiltinDirSubdir)
	targetDir, targetPath, loadedFrom := h.resolveUploadTarget(tech, base)
	conflict := false

	owners, err := h.repo.LoadedFromsByPlatform(c.Request.Context(), tech, platform)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if len(owners) > 0 {
		// 归属路径异常(历史脏数据)时回退派生路径,destructive 重载会清孤儿行
		conflict = true
		if p := filepath.Join(h.baseDir, owners[0]); pathContainedIn(libDir, p) {
			loadedFrom = filepath.ToSlash(owners[0])
			targetPath = p
			targetDir = filepath.Dir(p)
		}
	}
	if !pathContainedIn(libDir, targetPath) {
		h.logger.Error("audit: indicator upload rejected (path traversal)",
			zap.String("audit_action", "indicator.upload.rejected_path_traversal"),
			zap.String("filename", base),
			zap.String("target", targetPath))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("computed target path %q escapes dir %q", targetPath, libDir))
		return
	}
	if _, statErr := os.Stat(targetPath); statErr == nil {
		conflict = true
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("stat target: %w", statErr))
		return
	}

	// 重复且未带 force → 409 + overwritable 标记,前端弹二次确认后带 &force=true 重试。
	if conflict && !force {
		h.logger.Info("audit: indicator upload conflict (needs confirm)",
			zap.String("audit_action", "indicator.upload.conflict_needs_confirm"),
			zap.String("platform", platform),
			zap.String("tech", tech),
			zap.String("loaded_from", loadedFrom))
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"ret": 0,
			"msg": fmt.Sprintf("platform「%s」在制式「%s」下已存在(%s),确认后可覆盖", platform, tech, loadedFrom),
			"data": gin.H{
				"overwritable": true,
				"platform":     platform,
				"tech":         tech,
				"loaded_from":  loadedFrom,
			},
		})
		return
	}

	// per-filename 互斥锁(与 DeleteFile 共用)
	unlock := h.acquireFileLock(filepath.Base(targetPath))
	defer unlock()

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("mkdir target dir: %w", err))
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

	// 原子写:tmp.<uuid> → rename;失败回滚备份
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("write target: %w", writeErr))
		return
	}

	// sidecar 标记(<name>.xml.custom 空文件):仅"新建"写入(= custom 可删);
	// 覆盖保持原身份(见上)。失败 → 回滚已写 XML → 500。
	if !overwriting {
		if err := os.WriteFile(targetPath+CustomMarkerSuffix, nil, 0o640); err != nil {
			if rmErr := os.Remove(targetPath); rmErr != nil {
				h.logger.Error("rollback uploaded xml after sidecar write failed; host state inconsistent",
					zap.String("target_path", targetPath),
					zap.Error(rmErr))
			}
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("write sidecar marker: %w", err))
			return
		}
	}

	// destructive 重载(全量 UPSERT + 删孤儿)→ 刷新缓存。
	// 重载/缓存失败仅 Warn 不致命(文件已写,用户可重试)。
	reloadOK := true
	orphans := map[string]int{}
	if h.reloader != nil {
		result, err := PerformReloadWithOrphans(c.Request.Context(), h.repo, h.reloader, h.logger)
		if err != nil {
			reloadOK = false
			h.logger.Warn("audit: indicator upload reload failed (file written, DB not refreshed)",
				zap.String("audit_action", "indicator.upload.reload_failed"),
				zap.String("loaded_from", loadedFrom),
				zap.Error(err))
		} else {
			orphans = result.Orphans
			if h.cache != nil {
				h.cache.BumpCacheVersion(c.Request.Context())
			}
			// #241:导入成功落库后,按本次上传的 deviceType 刷新对应平台公式表绑定字典。
			// 平台名权威来源是 rela_platform_indicator_formula_<tech>.platform_name(perf_indicators_*
			// 不存平台维度),KPI XML 导入会写该公式表(见 file_repository 的 orphan 删除/冲突检测)。
			// 本期仅 enb→kpi_platform_enb;gnb/gsm 无绑定字典则 0 命中,安全 no-op。best-effort,失败只 Warn。
			h.refreshBoundDict(c.Request.Context(), "rela_platform_indicator_formula_"+tech)
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
		zap.String("platform", platform),
		zap.Int("body_size", len(body)),
		zap.Bool("overwritten", overwriting),
		zap.String("backup", backupName),
		zap.Bool("reload_ok", reloadOK))

	response.OKWithStatus(c, http.StatusCreated, gin.H{
		"uploaded":    true,
		"filename":    filepath.Base(targetPath),
		"loaded_from": loadedFrom,
		"tech":        tech,
		"platform":    platform,
		"overwritten": overwriting,
		"backup":      backupName,
		"reloaded":    reloadOK,
		"orphans":     orphans,
	})
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

// parseUploadPlatform 从上传字节流抽 <indicatorModel platform="..."> 的 platform 属性。
// 与 validateUploadXML 同款 encoding/xml.Decoder Strict 解析(无 XXE);未找到返空。
func parseUploadPlatform(raw []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = true
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "indicatorModel" {
			return ""
		}
		return strings.TrimSpace(attrValue(start.Attr, "platform"))
	}
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
// 确保上传落地用的目录树存在(0755):
//   - baseDir/indicator-library/      (GSM/GNB 上传落地处 = 根级,也是出厂单文件所在)
//   - baseDir/indicator-library/enb/  (ENB 多文件落地处,也是 loader 扫描的 ENB 子目录)
//
// 目录调整(2026-06-05):仅 ENB 分子目录,GSM/GNB 不再建 gsm/、gnb/ 子目录。
// 若镜像层已 COPY builtin XML 则为 no-op。
//
// 不在 RegisterRoutes 路径上调用 — cmd/app 启动期由 provider 显式调用。
func EnsureBaseDir(ctx context.Context, baseDir string) error {
	_ = ctx // 预留,目前无 IO 阻塞
	dir := filepath.Join(baseDir, BuiltinDirSubdir, ENBSubdirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure indicator enb dir %s: %w", dir, err)
	}
	return nil
}
