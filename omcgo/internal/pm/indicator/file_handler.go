package indicator

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

// FileHandler 提供 T-0180 P1.3+ XML 文件粒度的管理端点(对标 T-0178 parammodel handler):
//   - DELETE /api/v1/indicators/files/{*loadedFrom} — 删除自定义 XML + 级联 DB 清理
//   - (P1.4) GET    /api/v1/indicators/files?tech=  — 列出该制式所有 XML 文件
//   - (P1.4) POST   /api/v1/indicators/upload-xml?tech= — 上传新自定义 XML
//
// 与 rest_handler.go 的 indicator CRUD(指标行粒度)隔离,因为这里管的是文件粒度。
type FileHandler struct {
	repo      FileRepository
	baseDir   string         // XMLBaseDir,等于 dictloader.XMLBaseDir(e.g. /etc/omcgo/data)
	fileLocks sync.Map       // map[basename(string)]*sync.Mutex
	logger    *zap.Logger
}

// NewFileHandler 构造 FileHandler;
// baseDir 必须为 Loader 用的同一 XMLBaseDir,否则 loadedFrom 相对路径无法 join 到正确绝对路径。
func NewFileHandler(repo FileRepository, baseDir string, logger *zap.Logger) *FileHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileHandler{repo: repo, baseDir: baseDir, logger: logger.Named("indicator.file")}
}

// RegisterRoutes 挂在 /api/v1 之下;内部使用 /indicators/files/... 子路径。
//
// gin 路由的 *loadedFrom 是 catch-all wildcard,匹配剩余完整路径(含 /),
// 用于承载 "indicator-library-custom/enb/MY.xml" 形式的多段相对路径。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/indicators/files")
	g.DELETE("/*loadedFrom", h.DeleteFile)
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
