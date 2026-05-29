package parammodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// optInt64 在 JSON 解码时同时接受 number、numeric string、null、""。
//   - 字段缺省                → *optInt64 为 nil(语义:不提供)
//   - JSON `null` / `""`      → 非 nil 指针但 Valid=false(语义:显式清空)
//   - JSON number / 数字字符串 → 非 nil 指针且 Valid=true,Value=对应 int64
//
// 用于 min_value / max_value 这类 BIGINT 列:前端 Antd <Input> 总是吐字符串,
// 不能直接绑 *int64,否则 JSON 解码报 "cannot unmarshal string into ... int64"。
type optInt64 struct {
	Value int64
	Valid bool
}

func (o *optInt64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		o.Value, o.Valid = n, true
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected number or numeric string, got %s", string(data))
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("parse int64 from %q: %w", s, err)
	}
	o.Value, o.Valid = parsed, true
	return nil
}

// Ptr 把 *optInt64 转 *int64:nil 或 Valid=false 都返回 nil,否则返回拷贝指针。
func (o *optInt64) Ptr() *int64 {
	if o == nil || !o.Valid {
		return nil
	}
	v := o.Value
	return &v
}

// Handler 暴露 /api/v1/param-models/* 与 standard_params CRUD（设计 §1.13）。
//
// 写路径会自动调用 Registry.Refresh 同步内存映射缓存；XML 重载通过
// Reloader 接口注入（dictloader.Registry.ReloadOne 适配器）。
//
// T-0178: baseDir 与 Loader 共用,DELETE/Upload 物理路径解析基础;
// customDir 是 Upload 落地的绝对路径(NewHandler 一次性算出);
// fileLocks 提供 per-filename 进程内互斥(Upload + Delete + ReloadOne 三方共用,
// 避免同名文件并发写入竞态)。
type Handler struct {
	repo      *PgRepository
	registry  *Registry
	reloader  Reloader
	logger    *zap.Logger
	baseDir   string
	customDir string   // absolute path = filepath.Join(baseDir, CustomDirSubdir)
	fileLocks sync.Map // map[basename]*sync.Mutex
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不强依赖 dictloader 包。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// NewHandler 构造 Handler；reloader 可为 nil（import-directory 端点会返回 503）。
// baseDir 来自 DictLoaderConfig.XMLBaseDir,用于 T-0178 Custom XML 物理删除/上传定位。
func NewHandler(repo *PgRepository, registry *Registry, reloader Reloader, baseDir string, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		repo:      repo,
		registry:  registry,
		reloader:  reloader,
		baseDir:   baseDir,
		customDir: filepath.Join(baseDir, CustomDirSubdir),
		logger:    logger.Named("parammodel.handler"),
	}
}

// acquireFileLock 取/建一个 per-basename mutex,返回 unlock 函数。
// T-0178: Upload / Delete / 单文件 Reload 三方共享同一把锁,避免:
//   - 同时上传同名文件的 lost update
//   - Delete 与 Upload 并发产生半截状态
//   - Reload 期间 Upload 写入被 Loader 读半截
func (h *Handler) acquireFileLock(basename string) func() {
	mu, _ := h.fileLocks.LoadOrStore(basename, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

// RegisterRoutes 挂在 /api/v1 下；内部使用 /param-models 子路径。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/param-models")
	// 集中操作（无 :name）
	g.GET("", h.ListModels)
	g.POST("/import-directory", h.ImportDirectory)
	g.POST("/upload-xml", h.UploadXML) // T-0178: 上传自定义 paramModel XML
	g.POST("/cache/refresh", h.CacheRefresh)
	g.POST("/translate", h.Translate)
	// 标准参数树（位于 /param-models/standard 子路径）
	g.GET("/standard", h.ListStandard)
	g.GET("/standard/:path", h.GetStandard)
	g.POST("/standard", h.UpsertStandard)
	g.PUT("/standard/:path", h.UpdateStandard)
	g.DELETE("/standard/:path", h.DeleteStandard)
	// 单 paramModel
	g.GET("/:name", h.GetModel)
	g.PUT("/:name", h.UpdateModel)
	g.DELETE("/:name", h.DeleteModel)
	// mappings 子资源
	g.GET("/:name/mappings", h.ListMappings)
	g.POST("/:name/mappings", h.CreateMapping)
	g.PUT("/:name/mappings/:id", h.UpdateMapping)
	g.DELETE("/:name/mappings/:id", h.DeleteMapping)

	// discovered 视图（按 product 隔离，挂在 products 命名空间下）
	prod := rg.Group("/products")
	prod.GET("/:id/discovered", h.ListDiscovered)
	prod.GET("/:id/discovered/versions", h.ListDiscoveredVersions)
	prod.DELETE("/:id/discovered/versions/:swVersion", h.DeleteDiscoveredVersion)
}

// ── ParamModel ──────────────────────────────────────────────────────

type modelView struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	TotalEntries int       `json:"total_entries"`
	TotalObjects int       `json:"total_objects"`
	TotalParams  int       `json:"total_params"`
	Description  string    `json:"description"`
	IsActive     bool      `json:"is_active"`
	LoadedFrom   string    `json:"loaded_from"`

	// T-0178: source 与 deletable 是后端唯一真值源,前端直接渲染:
	//   - source ∈ {"builtin","custom","unknown"} — UI 来源列 Tag
	//   - deletable: 操作列删除按钮可见性(builtin/unknown 一律置灰 + Tooltip)
	// 改判定规则只动 source.go::ClassifySource / IsDeletable,无需重新发版前端
	Source    Source `json:"source"`
	Deletable bool   `json:"deletable"`
}

func toModelView(m *ParamModel) modelView {
	return modelView{
		ID: m.ID, Name: m.Name,
		TotalEntries: m.TotalEntries, TotalObjects: m.TotalObjects, TotalParams: m.TotalParams,
		Description: m.Description, IsActive: m.IsActive, LoadedFrom: m.LoadedFrom,
		Source:    ClassifySource(m.LoadedFrom),
		Deletable: IsDeletable(m.LoadedFrom),
	}
}

func (h *Handler) ListModels(c *gin.Context) {
	models, err := h.repo.ListParamModels(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]modelView, 0, len(models))
	for i := range models {
		views = append(views, toModelView(&models[i]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views)})
}

func (h *Handler) GetModel(c *gin.Context) {
	name := c.Param("name")
	m, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, toModelView(m))
}

type updateModelReq struct {
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

func (h *Handler) UpdateModel(c *gin.Context) {
	name := c.Param("name")
	var req updateModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	m, err := h.repo.UpdateParamModelMeta(c.Request.Context(), name, req.Description, req.IsActive)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "update-model")
	response.OK(c, toModelView(m))
}

// DeleteModel 删除 paramModel(T-0178 §9.5 守门顺序)。
//
// 守门规则:
//  1. 仅 Source=custom(loaded_from 以 "param-mappings-custom/" 开头)的模型可删
//     →内置 / 历史无前缀数据返 403 ErrCodeParamModelBuiltinNotDeletable
//  2. per-filename mutex 与 Upload / 单文件 Reload 互斥
//  3. 物理 rename → "<file>.deleted.<14位ts>" 备份(可回滚)
//     · ENOENT 容忍:文件已被外部 rm,视为"已备份",继续删 DB
//     · 其他 errno(EACCES/ENOSPC/EROFS):保守回滚(用户决策 4)→ 500
//     ErrCodeParamModelBackupFailed,不删 DB
//  4. DELETE param_models 行
//     · CASCADE 删 param_mappings、SET NULL 写 products.param_model_id
//     · DB 失败 → 反向 rename(backup → original)+ 500
//  5. registry.Refresh + 结构化审计日志(audit_action)
func (h *Handler) DeleteModel(c *gin.Context) {
	name := c.Param("name")

	// 1. 先取出 loaded_from,判定 Source
	pm, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if !IsDeletable(pm.LoadedFrom) {
		h.logger.Info("audit: param-model delete rejected (builtin)",
			zap.String("audit_action", "parammodel.delete.rejected_builtin"),
			zap.String("name", name),
			zap.String("loaded_from", pm.LoadedFrom),
			zap.String("source", string(ClassifySource(pm.LoadedFrom))))
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("builtin param model %q (loaded_from=%s) is not deletable; "+
				"to remove, delete the XML in data/param-mappings/ in the release image "+
				"and re-deploy [code=%d]",
				name, pm.LoadedFrom, global.ErrCodeParamModelBuiltinNotDeletable))
		return
	}

	// 2. per-filename 互斥锁(Upload / Reload 共用)
	unlock := h.acquireFileLock(filepath.Base(pm.LoadedFrom))
	defer unlock()

	// 3. 物理备份 — rename 是原子操作,无需中间状态
	absPath := filepath.Join(h.baseDir, pm.LoadedFrom)
	backupPath := absPath + ".deleted." + time.Now().Format("20060102150405")
	if renameErr := os.Rename(absPath, backupPath); renameErr != nil {
		switch {
		case errors.Is(renameErr, fs.ErrNotExist):
			// 文件已不在(外部 rm / 上次中断的 Delete)→ 视为"无需备份",继续删 DB 清理残留行
			h.logger.Warn("audit: custom xml already gone before delete; proceeding to remove DB row",
				zap.String("audit_action", "parammodel.delete.custom.file_already_gone"),
				zap.String("name", name),
				zap.String("loaded_from", pm.LoadedFrom),
				zap.String("path", absPath))
		default:
			// EACCES / ENOSPC / EROFS / 其他 IO → 保守回滚,绝不删 DB
			h.logger.Error("audit: param-model delete aborted (backup failed)",
				zap.String("audit_action", "parammodel.delete.aborted_backup_failed"),
				zap.String("name", name),
				zap.String("loaded_from", pm.LoadedFrom),
				zap.String("path", absPath),
				zap.Error(renameErr))
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("backup custom xml failed; refusing to delete DB row to avoid data loss; "+
					"please fix the filesystem (check disk space / permissions / mount RO state) and retry: %w [code=%d]",
					renameErr, global.ErrCodeParamModelBackupFailed))
			return
		}
	}

	// 4. DELETE DB 行,失败回滚文件
	ok, err := h.repo.DeleteParamModel(c.Request.Context(), name)
	if err != nil {
		// 反向 rename(忽略 rename 失败 — 此时只能 log 告警,DB 与 host 已分叉)
		if rbErr := os.Rename(backupPath, absPath); rbErr != nil {
			h.logger.Error("rollback rename failed; DB and host state diverged",
				zap.String("backup_path", backupPath),
				zap.String("orig_path", absPath),
				zap.Error(rbErr))
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		// DB 已无该行(竞态:并发 Delete),与 ENOENT 同款处理 — 文件备份保留,行不存在不报错
		h.logger.Warn("param-model already deleted by concurrent request",
			zap.String("name", name))
	}

	h.logger.Info("audit: param-model deleted (custom)",
		zap.String("audit_action", "parammodel.delete.custom"),
		zap.String("name", name),
		zap.String("loaded_from", pm.LoadedFrom),
		zap.String("backup", filepath.Base(backupPath)))

	h.refreshAsync(c.Request.Context(), "delete-model")
	response.OK(c, gin.H{
		"deleted": true,
		"name":    name,
		"backup":  filepath.Base(backupPath),
	})
}

// UploadXML 接收用户上传的自定义 paramModel XML(T-0178 §9.4)。
//
// POST /api/v1/param-models/upload-xml[?force=true]
// Content-Type: multipart/form-data
// Field: file
//
// 校验链(顺序敏感,任一失败即 400/409,审计明确拒绝原因):
//  1. 文件名:filepath.Base + uploadFilenamePattern 白名单正则 +
//     reservedUploadFilenames 保留名拦截
//  2. 大小:file.Size <= MaxUploadXMLSize (1 MiB)
//  3. 内容:validateUploadXML 根元素 = <paramModel>
//  4. 路径:filepath.Join(customDir, base) 经 pathContainedIn 二次验证不逃逸 customDir
//
// 写入流程(全程 per-filename 锁):
//  1. 确保 customDir 存在(MkdirAll,首次上传场景)
//  2. 检查同名:存在但无 ?force=true → 409 Conflict
//  3. 写 tmp 文件:targetPath + .tmp.<uuid>(原子写第一步)
//     defer os.Remove(tmp) — 任何路径退出都清掉残留
//  4. 若同名存在 + force=true:os.Rename(target, target+.bak.<ts>) 备份
//  5. os.Rename(tmp, target) 原子上线;失败 → 反向 rename 还原 .bak
//  6. 触发全量 ReloadOne("param-model") 更新 DB(reload 失败不算 upload 失败,
//     文件已落地用户可手动 reload 重试)
//  7. registry.Refresh
//
// 安全:
//   - 文件名正则拒绝路径分隔符 / 点开头 / 空白 / 多扩展名
//   - pathContainedIn 二次防御 filepath.Clean 解释差异
//   - encoding/xml Strict + 不处理外部实体(Go 标准库默认安全,无 XXE)
//   - 单文件大小硬上限 1 MiB(Gin engine MaxMultipartMemory 应配 4 MiB)
//   - per-filename 锁保证同名 Upload/Delete/Reload 串行
func (h *Handler) UploadXML(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("missing or invalid 'file' multipart field: %w", err))
		return
	}

	// 校验 1: 文件名
	base := filepath.Base(file.Filename)
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: upload rejected (invalid filename)",
			zap.String("audit_action", "parammodel.upload.rejected_invalid_name"),
			zap.String("raw_filename", file.Filename),
			zap.String("basename", base),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 校验 2: 大小
	if file.Size <= 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("file size must be positive, got %d", file.Size))
		return
	}
	if file.Size > MaxUploadXMLSize {
		h.logger.Info("audit: upload rejected (size limit)",
			zap.String("audit_action", "parammodel.upload.rejected_too_large"),
			zap.String("filename", base),
			zap.Int64("size", file.Size),
			zap.Int64("limit", MaxUploadXMLSize))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("file size %d exceeds limit %d bytes (%.2f MiB)",
				file.Size, MaxUploadXMLSize, float64(MaxUploadXMLSize)/float64(1<<20)))
		return
	}

	// 校验 3: 读全文 + XML 内容校验
	src, err := file.Open()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("open uploaded file: %w", err))
		return
	}
	// LimitReader: 防 Content-Length 与 Size 撒谎导致 OOM
	raw, err := io.ReadAll(io.LimitReader(src, MaxUploadXMLSize+1))
	_ = src.Close()
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("read uploaded file: %w", err))
		return
	}
	if int64(len(raw)) > MaxUploadXMLSize {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("file size exceeds limit during stream read"))
		return
	}
	if err := validateUploadXML(raw); err != nil {
		h.logger.Info("audit: upload rejected (invalid xml)",
			zap.String("audit_action", "parammodel.upload.rejected_invalid_xml"),
			zap.String("filename", base),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 校验 4: 路径包含性二次防御
	targetPath := filepath.Join(h.customDir, base)
	if !pathContainedIn(h.customDir, targetPath) {
		h.logger.Error("audit: upload rejected (path traversal detected)",
			zap.String("audit_action", "parammodel.upload.rejected_path_traversal"),
			zap.String("filename", base),
			zap.String("computed_path", targetPath),
			zap.String("custom_dir", h.customDir))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("path traversal detected for filename %q", base))
		return
	}

	// per-filename 锁(与 Delete / 单文件 Reload 共用)
	unlock := h.acquireFileLock(base)
	defer unlock()

	// 确保 custom 目录存在(首次部署 + 0750 = owner rwx,group rx,others -)
	if err := os.MkdirAll(h.customDir, 0o750); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("ensure custom dir: %w", err))
		return
	}

	// 同名冲突检测
	force := strings.EqualFold(c.Query("force"), "true")
	_, statErr := os.Stat(targetPath)
	exists := statErr == nil
	if exists && !force {
		commonerrors.AbortWithError(c, http.StatusConflict,
			fmt.Errorf("custom xml %q already exists; use ?force=true to overwrite "+
				"(existing file will be backed up to .bak.<ts>)", base))
		return
	}

	// 1. tmp 写入(失败不影响现有文件)
	tmpPath := targetPath + ".tmp." + uuid.New().String()
	defer os.Remove(tmpPath) // 兜底:rename 成功后 tmp 已不存在,Remove 返 ENOENT 无害
	if err := os.WriteFile(tmpPath, raw, 0o640); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("write tmp file: %w", err))
		return
	}

	// 2. 备份现有(若 overwrite)
	var backupPath string
	if exists {
		backupPath = targetPath + ".bak." + time.Now().Format("20060102150405")
		if err := os.Rename(targetPath, backupPath); err != nil {
			h.logger.Error("audit: upload aborted (backup existing failed)",
				zap.String("audit_action", "parammodel.upload.aborted_backup_failed"),
				zap.String("filename", base),
				zap.Error(err))
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("backup existing file failed: %w", err))
			return
		}
	}

	// 3. tmp → target 原子上线
	if err := os.Rename(tmpPath, targetPath); err != nil {
		// 还原备份(若有)
		if backupPath != "" {
			if rbErr := os.Rename(backupPath, targetPath); rbErr != nil {
				h.logger.Error("rollback rename failed; host state inconsistent",
					zap.String("backup_path", backupPath),
					zap.String("target_path", targetPath),
					zap.Error(rbErr))
			}
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("rename tmp → target: %w", err))
		return
	}

	// 4. 触发全量 ReloadOne(文件级 reload 当前未支持;全量 reload 幂等且 <1s)
	if h.reloader != nil {
		if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
			// 文件已落地,reload 失败不阻塞 upload 响应,用户可手动 reload 重试
			h.logger.Warn("post-upload reload failed",
				zap.String("filename", base),
				zap.Error(err))
		}
	}
	if h.registry != nil {
		if err := h.registry.Refresh(c.Request.Context()); err != nil {
			h.logger.Warn("post-upload registry refresh failed",
				zap.String("filename", base),
				zap.Error(err))
		}
	}

	action := "parammodel.upload.success"
	if exists {
		action = "parammodel.upload.overwrite"
	}
	h.logger.Info("audit: param-model uploaded",
		zap.String("audit_action", action),
		zap.String("filename", base),
		zap.Int64("size", file.Size),
		zap.Bool("overwrite", exists),
		zap.String("backup", filepath.Base(backupPath)))

	out := gin.H{
		"filename":  base,
		"size":      file.Size,
		"overwrite": exists,
	}
	if backupPath != "" {
		out["backup"] = filepath.Base(backupPath)
	}
	response.OK(c, out)
}

// ── Mappings ────────────────────────────────────────────────────────

type mappingView struct {
	ID            uuid.UUID `json:"id"`
	ParamModelID  uuid.UUID `json:"param_model_id"`
	StandardPath  string    `json:"standard_path"`
	PrivatePath   string    `json:"private_path"`
	EntryType     string    `json:"entry_type"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *int64    `json:"min_value,omitempty"`
	MaxValue      *int64    `json:"max_value,omitempty"`
	IsStorable    bool      `json:"is_storable"`
	IsActive      bool      `json:"is_active"`
	IsSupported   bool      `json:"is_supported"`
	SoftwareVer   *string   `json:"software_version,omitempty"`
}

func toMappingView(m *ParamMapping) mappingView {
	return mappingView{
		ID: m.ID, ParamModelID: m.ParamModelID,
		StandardPath: m.StandardPath, PrivatePath: m.PrivatePath, EntryType: m.EntryType,
		Access: m.Access, DataType: m.DataType, ChangeApplies: m.ChangeApplies,
		MinValue: m.MinValue, MaxValue: m.MaxValue,
		IsStorable: m.IsStorable, IsActive: m.IsActive, IsSupported: m.IsSupported, SoftwareVer: m.SoftwareVersion,
	}
}

func (h *Handler) ListMappings(c *gin.Context) {
	name := c.Param("name")
	m, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	mappings, err := h.repo.ListMappingsByParamModel(c.Request.Context(), m.ID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]mappingView, 0, len(mappings))
	for i := range mappings {
		views = append(views, toMappingView(&mappings[i]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views), "param_model": toModelView(m)})
}

type createMappingReq struct {
	StandardPath  string    `json:"standard_path" binding:"required"`
	PrivatePath   string    `json:"private_path" binding:"required"`
	EntryType     string    `json:"entry_type" binding:"required"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *optInt64 `json:"min_value"`
	MaxValue      *optInt64 `json:"max_value"`
	IsStorable    *bool     `json:"is_storable"`
}

func (h *Handler) CreateMapping(c *gin.Context) {
	name := c.Param("name")
	m, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	var req createMappingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	in := CreateMappingInput{
		StandardPath:  strings.TrimSpace(req.StandardPath),
		PrivatePath:   strings.TrimSpace(req.PrivatePath),
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
		IsStorable:    true,
	}
	if req.IsStorable != nil {
		in.IsStorable = *req.IsStorable
	}
	created, err := h.repo.CreateMapping(c.Request.Context(), m.ID, in)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.invalidateModel(c.Request.Context(), m.ID, "create-mapping")
	response.OKWithStatus(c, http.StatusCreated, toMappingView(created))
}

type updateMappingReq struct {
	PrivatePath   *string   `json:"private_path"`
	Access        *string   `json:"access"`
	DataType      *string   `json:"data_type"`
	ChangeApplies *string   `json:"change_applies"`
	MinValue      *optInt64 `json:"min_value"`
	MaxValue      *optInt64 `json:"max_value"`
	IsStorable    *bool     `json:"is_storable"`
	IsActive      *bool     `json:"is_active"`
}

func (h *Handler) UpdateMapping(c *gin.Context) {
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("mapping id not a uuid"))
		return
	}
	var req updateMappingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	updated, err := h.repo.UpdateMapping(c.Request.Context(), mappingID, UpdateMappingInput{
		PrivatePath:   req.PrivatePath,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
		IsStorable:    req.IsStorable,
		IsActive:      req.IsActive,
	})
	if errors.Is(err, ErrNoMapping) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.invalidateModel(c.Request.Context(), updated.ParamModelID, "update-mapping")
	response.OK(c, toMappingView(updated))
}

func (h *Handler) DeleteMapping(c *gin.Context) {
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("mapping id not a uuid"))
		return
	}
	// 删前查 paramModelID 用于失效
	mapping, getErr := h.repo.getMappingByID(c.Request.Context(), mappingID)
	ok, err := h.repo.DeleteMapping(c.Request.Context(), mappingID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if getErr == nil && mapping != nil {
		h.invalidateModel(c.Request.Context(), mapping.ParamModelID, "delete-mapping")
	}
	response.OK(c, gin.H{"deleted": true, "id": mappingID})
}

// ── Discovered ──────────────────────────────────────────────────────

func (h *Handler) ListDiscovered(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product id not a uuid"))
		return
	}
	swVersion := strings.TrimSpace(c.Query("swVersion"))
	if swVersion == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("swVersion query parameter required"))
		return
	}
	mappings, err := h.repo.ListDiscoveredMappings(c.Request.Context(), productID, swVersion)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]mappingView, 0, len(mappings))
	for i := range mappings {
		views = append(views, toMappingView(&mappings[i]))
	}
	response.OK(c, gin.H{
		"items":            views,
		"total":            len(views),
		"product_id":       productID,
		"software_version": swVersion,
	})
}

func (h *Handler) ListDiscoveredVersions(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product id not a uuid"))
		return
	}
	versions, err := h.repo.ListDiscoveredVersions(c.Request.Context(), productID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": versions, "total": len(versions), "product_id": productID})
}

func (h *Handler) DeleteDiscoveredVersion(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product id not a uuid"))
		return
	}
	swVersion := strings.TrimSpace(c.Param("swVersion"))
	if swVersion == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("swVersion path parameter required"))
		return
	}
	deleted, err := h.repo.DeleteDiscoveredVersion(c.Request.Context(), productID, swVersion)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if h.registry != nil {
		_ = h.registry.InvalidateProduct(c.Request.Context(), productID, swVersion)
	}
	response.OK(c, gin.H{
		"deleted":          deleted,
		"product_id":       productID,
		"software_version": swVersion,
	})
}

// ── Standard params ─────────────────────────────────────────────────

type standardView struct {
	StandardPath  string `json:"standard_path"`
	EntryType     string `json:"entry_type"`
	Access        string `json:"access"`
	DataType      string `json:"data_type"`
	ChangeApplies string `json:"change_applies"`
	MinValue      *int64 `json:"min_value,omitempty"`
	MaxValue      *int64 `json:"max_value,omitempty"`
}

func toStandardView(sp *StandardParam) standardView {
	return standardView{
		StandardPath: sp.StandardPath, EntryType: sp.EntryType,
		Access: sp.Access, DataType: sp.DataType, ChangeApplies: sp.ChangeApplies,
		MinValue: sp.MinValue, MaxValue: sp.MaxValue,
	}
}

func (h *Handler) ListStandard(c *gin.Context) {
	keyword := c.Query("keyword")
	entryType := c.Query("entry_type")
	items, err := h.repo.ListStandardParams(c.Request.Context(), keyword, entryType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]standardView, 0, len(items))
	for i := range items {
		views = append(views, toStandardView(&items[i]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views)})
}

func (h *Handler) GetStandard(c *gin.Context) {
	standardPath := c.Param("path")
	sp, err := h.repo.GetStandardParam(c.Request.Context(), standardPath)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, toStandardView(sp))
}

type upsertStandardReq struct {
	StandardPath  string    `json:"standard_path" binding:"required"`
	EntryType     string    `json:"entry_type" binding:"required"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *optInt64 `json:"min_value"`
	MaxValue      *optInt64 `json:"max_value"`
}

func (h *Handler) UpsertStandard(c *gin.Context) {
	var req upsertStandardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	sp, err := h.repo.UpsertStandardParam(c.Request.Context(), UpsertStandardParamInput{
		StandardPath:  req.StandardPath,
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, toStandardView(sp))
}

func (h *Handler) UpdateStandard(c *gin.Context) {
	standardPath := c.Param("path")
	var req upsertStandardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// 路径覆盖：URL 上的 path 优先；body 没填或不一致都以 URL 为准
	if req.StandardPath == "" {
		req.StandardPath = standardPath
	} else if req.StandardPath != standardPath {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("standard_path in body (%q) differs from URL (%q)", req.StandardPath, standardPath))
		return
	}
	sp, err := h.repo.UpsertStandardParam(c.Request.Context(), UpsertStandardParamInput{
		StandardPath:  req.StandardPath,
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, toStandardView(sp))
}

func (h *Handler) DeleteStandard(c *gin.Context) {
	standardPath := c.Param("path")
	ok, err := h.repo.DeleteStandardParam(c.Request.Context(), standardPath)
	if err != nil {
		// 引用拒绝错误用 409 Conflict
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, gin.H{"deleted": true, "standard_path": standardPath})
}

// ── Translate ───────────────────────────────────────────────────────

type translateReq struct {
	ProductID       string   `json:"productId" binding:"required"`
	SoftwareVersion string   `json:"softwareVersion"`
	Direction       string   `json:"direction" binding:"required"` // "to_private" | "to_standard"
	Paths           []string `json:"paths" binding:"required"`
}

type translateItem struct {
	Original   string `json:"original"`
	Translated string `json:"translated"`
	Found      bool   `json:"found"`
	Source     string `json:"source"`
}

func (h *Handler) Translate(c *gin.Context) {
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, fmt.Errorf("param registry not wired"))
		return
	}
	var req translateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("productId not a uuid"))
		return
	}
	if req.Direction != "to_private" && req.Direction != "to_standard" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("direction must be to_private or to_standard"))
		return
	}

	tr, err := h.registry.Translator(c.Request.Context(), productID, req.SoftwareVersion)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	src := string(tr.Source())
	out := make([]translateItem, 0, len(req.Paths))
	for _, p := range req.Paths {
		var tres TranslationResult
		if req.Direction == "to_private" {
			tres = tr.ToPrivate(p)
		} else {
			tres = tr.ToStandard(p)
		}
		out = append(out, translateItem{
			Original:   tres.Original,
			Translated: tres.Translated,
			Found:      tres.Found,
			Source:     src,
		})
	}
	response.OK(c, gin.H{
		"results":          out,
		"product_id":       productID,
		"software_version": req.SoftwareVersion,
		"direction":        req.Direction,
		"source":           src,
	})
}

// ── Cache + Import ──────────────────────────────────────────────────

// ImportDirectory 从 datamodels/ 目录加载 XML。
//
// Query params:
//   - mode=import (默认): 加法 UPSERT —— 仅写入/更新现有 XML 中的模型，
//     不删除 DB 中不在 XML 文件里的孤儿模型（手工 UI 添加项保留）。
//   - mode=reload: destructive 全量重载 —— 完成 UPSERT 后，
//     删除 DB 中所有未被本次加载触达的 param_models（孤儿模型）；
//     param_mappings CASCADE 删除；products.param_model_id SET NULL。
func (h *Handler) ImportDirectory(c *gin.Context) {
	if h.reloader == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("dictloader registry not wired"))
		return
	}
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	if mode == "" {
		mode = "import"
	}
	if mode != "import" && mode != "reload" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid mode %q (expected import|reload)", mode))
		return
	}

	// destructive 模式：记录开始时间，便于事后按 updated_at 识别孤儿
	startedAt := time.Now()

	if err := h.reloader.ReloadOne(c.Request.Context(), "param-model"); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var orphansDeleted int64
	if mode == "reload" {
		var err error
		orphansDeleted, err = h.repo.DeleteOrphansSince(c.Request.Context(), startedAt)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("cleanup orphan param_models: %w", err))
			return
		}
	}

	if h.registry != nil {
		if err := h.registry.Refresh(c.Request.Context()); err != nil {
			h.logger.Warn("post-reload param registry refresh failed", zap.Error(err))
		}
	}
	response.OK(c, gin.H{
		"reloaded":         "param-model",
		"mode":             mode,
		"orphans_deleted":  orphansDeleted,
	})
}

func (h *Handler) CacheRefresh(c *gin.Context) {
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("param registry not wired"))
		return
	}
	if err := h.registry.Refresh(c.Request.Context()); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"refreshed": true})
}

// ── helpers ─────────────────────────────────────────────────────────

func (h *Handler) refreshAsync(ctx context.Context, op string) {
	if h.registry == nil {
		return
	}
	if err := h.registry.Refresh(ctx); err != nil {
		h.logger.Warn("param registry refresh after write failed", zap.String("op", op), zap.Error(err))
	}
}

func (h *Handler) invalidateModel(ctx context.Context, paramModelID uuid.UUID, op string) {
	if h.registry == nil {
		return
	}
	if err := h.registry.InvalidateParamModel(ctx, paramModelID); err != nil {
		h.logger.Warn("param registry invalidate after write failed",
			zap.String("op", op),
			zap.String("param_model_id", paramModelID.String()),
			zap.Error(err))
	}
}
