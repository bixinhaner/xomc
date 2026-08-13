package parammodel

import (
	"context"
	"encoding/json"
	"encoding/xml"
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
// XML 管理重构:取消 builtin/custom 自定义目录区分 —— Upload 直接写
// builtin 目录(Loader 扫描的同一目录 param-mappings/),接受升级丢失。
// baseDir 与 Loader 共用,DELETE/Upload 物理路径解析基础;
// builtinDir 是 Upload 落地的绝对路径(NewHandler 一次性算出);
// fileLocks 提供 per-filename 进程内互斥(Upload + Delete + ReloadOne 三方共用,
// 避免同名文件并发写入竞态)。
type Handler struct {
	repo             *PgRepository
	registry         *Registry
	reloader         Reloader
	dictRefresher    DictSourceRefresher
	productRefresher RegistryRefresher
	logger           *zap.Logger
	baseDir          string
	builtinDir       string   // absolute path = filepath.Join(baseDir, BuiltinDirSubdir)
	fileLocks        sync.Map // map[basename]*sync.Mutex
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不强依赖 dictloader 包。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// DictSourceRefresher 在导入 XML 成功后按 source_table 刷新绑定字典(T-0182 / #241)。
// best-effort:刷新失败不阻断导入(daily cron 兜底)。可为 nil(未接入时整段跳过)。
type DictSourceRefresher interface {
	RefreshSourceBoundByTable(ctx context.Context, sourceTable string) (int, error)
}

// RegistryRefresher 用于在参数模型启停后刷新依赖该状态的产品路由缓存。
type RegistryRefresher interface {
	Refresh(ctx context.Context) error
}

// SetProductRegistryRefresher 注入 ProductRegistry，避免 parammodel 包直接依赖 product 包。
func (h *Handler) SetProductRegistryRefresher(refresher RegistryRefresher) {
	h.productRefresher = refresher
}

// NewHandler 构造 Handler；reloader 可为 nil（导入 XML 时 destructiveReload 跳过重载，
// 只写文件，仅 Warn）。baseDir 来自 DictLoaderConfig.XMLBaseDir,用于 XML 物理删除/上传定位。
// dictRefresher 可为 nil(未接入数据字典数据源时);非 nil 时导入成功后刷新 param_models 绑定字典。
func NewHandler(repo *PgRepository, registry *Registry, reloader Reloader, dictRefresher DictSourceRefresher, baseDir string, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		repo:          repo,
		registry:      registry,
		reloader:      reloader,
		dictRefresher: dictRefresher,
		baseDir:       baseDir,
		builtinDir:    filepath.Join(baseDir, BuiltinDirSubdir),
		logger:        logger.Named("parammodel.handler"),
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
	h.RegisterReadRoutes(rg)
	h.RegisterWriteRoutes(rg)
}

// RegisterReadRoutes 挂载参数模型与标准参数的只读接口。
func (h *Handler) RegisterReadRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/param-models")
	// 集中操作（无 :name）
	g.GET("", h.ListModels)
	// 下载 XML 原文件(?loaded_from= query;静态段优先于下方 /:name 参数路由)
	g.GET("/file-content", h.DownloadFile)
	g.POST("/translate", h.Translate)
	// 标准参数树（位于 /param-models/standard 子路径）
	g.GET("/standard", h.ListStandard)
	g.GET("/standard/:path", h.GetStandard)
	// 单 paramModel
	g.GET("/:name", h.GetModel)
	// mappings 子资源
	g.GET("/:name/mappings", h.ListMappings)

	// discovered 视图（按 product 隔离，挂在 products 命名空间下）
	prod := rg.Group("/products")
	prod.GET("/:id/discovered", h.ListDiscovered)
	prod.GET("/:id/discovered/versions", h.ListDiscoveredVersions)
	prod.DELETE("/:id/discovered/versions/:swVersion", h.DeleteDiscoveredVersion)
}

// RegisterWriteRoutes 挂载参数模型与标准参数的写接口，仅供超管管理。
func (h *Handler) RegisterWriteRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/param-models")
	// 导入 XML:名称取自 XML paramModel 属性 + 重复二次确认覆盖(?force=true)→
	// destructive 重载(全量+删孤儿)→ 刷新缓存,在单端点内顺序完成。
	g.POST("/upload-xml", h.UploadXML)
	g.POST("/standard", h.CreateStandard)
	g.PUT("/standard/:path", h.UpdateStandard)
	g.DELETE("/standard/:path", h.DeleteStandard)
	g.PUT("/:name", h.UpdateModel)
	g.DELETE("/:name", h.DeleteModel)
	g.POST("/:name/mappings", h.CreateMapping)
	g.PUT("/:name/mappings/:id", h.UpdateMapping)
	g.DELETE("/:name/mappings/:id", h.DeleteMapping)
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

func toModelView(baseDir string, m *ParamModel) modelView {
	return modelView{
		ID: m.ID, Name: m.Name,
		TotalEntries: m.TotalEntries, TotalObjects: m.TotalObjects, TotalParams: m.TotalParams,
		Description: m.Description, IsActive: m.IsActive, LoadedFrom: m.LoadedFrom,
		Source:    ClassifySource(baseDir, m.LoadedFrom),
		Deletable: IsDeletable(baseDir, m.LoadedFrom),
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
		views = append(views, toModelView(h.baseDir, &models[i]))
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
	response.OK(c, toModelView(h.baseDir, m))
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
	response.OK(c, toModelView(h.baseDir, m))
}

// DeleteModel 删除 paramModel。
//
// XML 管理重构:取消 builtin/custom 删除守门,所有文件均可删除。
//
// 流程:
//  1. 取出 loaded_from 定位物理文件
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

	// 1. 先取出 loaded_from 定位物理文件
	pm, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// 1.5 内置数据守门:builtin(当前目录 XML 加载)+ unknown 不可删(2026-06-04 用户决策)
	if !IsDeletable(h.baseDir, pm.LoadedFrom) {
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("内置数据不允许删除(loaded_from=%q)[code=%d]", pm.LoadedFrom, global.ErrCodeParamModelBuiltinNotDeletable))
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

	// DB 行已删,移除 sidecar 标记(X.xml.custom)。XML 本体已 rename 为 .deleted.<ts> 备份,
	// sidecar 不再有判定意义;残留会让孤儿 sidecar 累积。ENOENT 容忍(可能从未写过)。
	if rmErr := os.Remove(absPath + CustomMarkerSuffix); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		h.logger.Warn("remove sidecar marker failed (non-fatal)",
			zap.String("name", name),
			zap.String("sidecar", filepath.Base(absPath)+CustomMarkerSuffix),
			zap.Error(rmErr))
	}

	h.logger.Info("audit: param-model deleted",
		zap.String("audit_action", "parammodel.delete"),
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

// UploadXML 导入 paramModel XML —— "导入 XML" 合并端点(三库 XML 导入重构 D3/D5/D6)。
//
// POST /api/v1/param-models/upload-xml[?force=true]
// Content-Type: multipart/form-data
// Fields: file(XML 内容)
//
// 名称取消手填(2026-06-05 导入 XML 调整):唯一名称取自 XML <parameterModel paramModel="...">
// 属性,文件名 = <paramModel>.xml,直接写 builtin 目录(Loader 扫描的同一目录 param-mappings/)。
// 上传文件自身的 filename 被忽略。
//
// 重复允许覆盖(2026-06-05 调整,需前端二次确认):
//   - 重复判定:paramModel 名已在 DB(GetParamModelByName 定位归属文件)或 <paramModel>.xml 已在盘上
//   - 无 ?force=true → 409,响应 data.overwritable=true,前端弹确认框
//   - ?force=true → 覆盖归属文件(原文件先备份 .bak.<ts>);覆盖内置保持 builtin 身份
//     (不写 sidecar,仍不可删),覆盖自定义保持 custom
//
// 校验链(顺序敏感,任一失败即 400/409,审计明确拒绝原因):
//  1. 大小:file.Size <= MaxUploadXMLSize (1 MiB)
//  2. 内容:validateUploadXML 根元素 = <parameterModel>
//  3. 名称:paramModel 属性必填;validateUploadFilename(<paramModel>+".xml") 白名单正则 + 保留名拦截
//  4. 路径:目标路径经 pathContainedIn 二次验证不逃逸
//  5. 重复 + force 判定(见上)
//
// 写入流程(全程 per-filename 锁):
//  1. 确保 builtinDir 存在(MkdirAll,首次上传场景)
//  2. 写 tmp 文件:targetPath + .tmp.<uuid> → 原子 rename 上线
//  3. 写 sidecar(<name>.xml.custom 空标记);失败回滚 XML → 500
//  4. destructiveReload:全量 ReloadOne + 删孤儿 + registry.Refresh
//     (文件已落地,reload 失败只 Warn,响应 reloaded 字段反映状态)
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

	// 校验 1: 大小
	if file.Size <= 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("file size must be positive, got %d", file.Size))
		return
	}
	if file.Size > MaxUploadXMLSize {
		h.logger.Info("audit: upload rejected (size limit)",
			zap.String("audit_action", "parammodel.upload.rejected_too_large"),
			zap.String("filename", file.Filename),
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
			zap.String("filename", file.Filename),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 校验 3: 名称提取 —— 唯一来源 XML <parameterModel paramModel="..."> 属性
	// (取消手填 name);内容主键与目标文件名同源。真值源 model.go xmlParameterModel。
	var doc xmlParameterModel
	if err := xml.Unmarshal(raw, &doc); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("解析 parameterModel XML 失败: %w", err))
		return
	}
	modelName := strings.TrimSpace(doc.ParamModel)
	if modelName == "" {
		h.logger.Info("audit: upload rejected (missing paramModel attr)",
			zap.String("audit_action", "parammodel.upload.rejected_missing_param_model"),
			zap.String("filename", file.Filename))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("参数模型 XML 缺少 paramModel 属性,无法确定名称"))
		return
	}
	base := modelName + ".xml"
	if err := validateUploadFilename(base); err != nil {
		h.logger.Info("audit: upload rejected (invalid paramModel-derived name)",
			zap.String("audit_action", "parammodel.upload.rejected_invalid_name"),
			zap.String("model_name", modelName),
			zap.String("basename", base),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("paramModel「%s」不能作为文件名:%w", modelName, err))
		return
	}

	force := c.Query("force") == "true"

	// 校验 4/5: 定位目标文件 + 重复判定。
	// 内容主键(paramModel 名)已在 DB → 归属文件即覆盖目标;否则落 <paramModel>.xml,
	// 盘上已存在(uploaded-but-not-loaded)同样视为重复。
	targetPath := filepath.Join(h.builtinDir, base)
	conflict := false
	existingPM, err := h.repo.GetParamModelByName(c.Request.Context(), modelName)
	switch {
	case err == nil:
		conflict = true
		if existingPM.LoadedFrom != "" {
			if p := filepath.Join(h.baseDir, existingPM.LoadedFrom); pathContainedIn(h.builtinDir, p) {
				targetPath = p
			}
			// 归属路径异常(历史脏数据)时回退派生路径,destructive 重载会清孤儿行
		}
	case errors.Is(err, ErrNoParamModel):
		// 无冲突,落派生路径
	default:
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !pathContainedIn(h.builtinDir, targetPath) {
		h.logger.Error("audit: upload rejected (path traversal detected)",
			zap.String("audit_action", "parammodel.upload.rejected_path_traversal"),
			zap.String("filename", base),
			zap.String("computed_path", targetPath),
			zap.String("builtin_dir", h.builtinDir))
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("path traversal detected for filename %q", base))
		return
	}
	if _, statErr := os.Stat(targetPath); statErr == nil {
		conflict = true
	}

	// 重复且未带 force → 409 + overwritable 标记,前端弹二次确认后带 ?force=true 重试。
	loadedFromRel, _ := filepath.Rel(h.baseDir, targetPath)
	loadedFrom := filepath.ToSlash(loadedFromRel)
	if conflict && !force {
		h.logger.Info("audit: upload conflict (needs confirm)",
			zap.String("audit_action", "parammodel.upload.conflict_needs_confirm"),
			zap.String("model_name", modelName),
			zap.String("loaded_from", loadedFrom))
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"ret": 0,
			"msg": fmt.Sprintf("paramModel「%s」已存在(%s),确认后可覆盖", modelName, loadedFrom),
			"data": gin.H{
				"overwritable": true,
				"model_name":   modelName,
				"loaded_from":  loadedFrom,
			},
		})
		return
	}

	// per-filename 锁(与 Delete / 单文件 Reload 共用)
	unlock := h.acquireFileLock(filepath.Base(targetPath))
	defer unlock()

	// 确保 builtin 目录存在(首次部署 + 0750 = owner rwx,group rx,others -)
	if err := os.MkdirAll(h.builtinDir, 0o750); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("ensure builtin dir: %w", err))
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

	// 1. tmp 写入 → 原子 rename 上线;失败回滚备份
	tmpPath := targetPath + ".tmp." + uuid.New().String()
	defer os.Remove(tmpPath) // 兜底:rename 成功后 tmp 已不存在,Remove 返 ENOENT 无害
	writeErr := os.WriteFile(tmpPath, raw, 0o640)
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

	// 2. sidecar 标记(<name>.xml.custom 空文件):仅"新建"写入(= custom 可删);
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

	// 3. destructive 重载(全量 + 删孤儿)+ 刷新缓存。
	//    文件已落地,reload/cache 失败不阻塞 upload 响应,只 Warn;
	//    响应 reloaded/orphans_deleted 字段反映实际状态,用户可重新导入重试。
	reloaded, orphansDeleted := h.destructiveReload(c.Request.Context(), "upload:"+filepath.Base(targetPath))

	// #241:导入成功落库后,主动刷新绑定 param_models 的字典(param_model_name),
	// 使「新增产品 → 参数模型名称」下拉即时出现新模型(无需等 daily cron / 手动刷新)。
	// 仅在重载真正落库(reloaded)后刷新,与 indicator/alarm 两库口径一致;best-effort,失败只 Warn。
	if reloaded {
		h.refreshBoundDict(c.Request.Context(), "param_models")
	}

	backupName := ""
	if backupPath != "" {
		backupName = filepath.Base(backupPath)
	}
	h.logger.Info("audit: param-model uploaded",
		zap.String("audit_action", "parammodel.upload.success"),
		zap.String("filename", filepath.Base(targetPath)),
		zap.String("model_name", modelName),
		zap.Int64("size", file.Size),
		zap.Bool("overwritten", overwriting),
		zap.String("backup", backupName),
		zap.Bool("reloaded", reloaded),
		zap.Int64("orphans_deleted", orphansDeleted))

	response.OK(c, gin.H{
		"filename":        filepath.Base(targetPath),
		"model_name":      modelName,
		"size":            file.Size,
		"overwritten":     overwriting,
		"backup":          backupName,
		"reloaded":        reloaded,
		"orphans_deleted": orphansDeleted,
	})
}

// DownloadFile GET /api/v1/param-models/file-content?loaded_from=param-mappings/<file>.xml
//
// 下载 XML 原文件(2026-06-05 操作列下载功能)。builtin 与 custom 均可下载(只读无守门);
// 路径校验:两段 param-mappings/<file>.xml + 防遍历 + pathContainedIn 双重防御。
func (h *Handler) DownloadFile(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("loaded_from"))
	loadedFrom := filepath.ToSlash(filepath.Clean(raw))
	parts := strings.Split(loadedFrom, "/")
	valid := raw != "" && len(parts) == 2 && parts[0] == BuiltinDirSubdir &&
		parts[1] != "" && !strings.Contains(parts[1], "..") &&
		strings.EqualFold(filepath.Ext(parts[1]), ".xml")
	if !valid {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid loaded_from %q (expected %s/<file>.xml)", raw, BuiltinDirSubdir))
		return
	}
	absPath := filepath.Join(h.baseDir, loadedFrom)
	if !pathContainedIn(h.builtinDir, absPath) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("path %q escapes param-mappings dir", loadedFrom))
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

// ── Mappings ────────────────────────────────────────────────────────

type mappingView struct {
	ID                uuid.UUID `json:"id"`
	ParamModelID      uuid.UUID `json:"param_model_id"`
	StandardPath      string    `json:"standard_path"`
	PrivatePath       string    `json:"private_path"`
	EntryType         string    `json:"entry_type"`
	Access            string    `json:"access"`
	DataType          string    `json:"data_type"`
	ChangeApplies     string    `json:"change_applies"`
	MinValue          *int64    `json:"min_value,omitempty"`
	MaxValue          *int64    `json:"max_value,omitempty"`
	EnumValues        *string   `json:"enum_values,omitempty"`
	EnumLabels        *string   `json:"enum_labels,omitempty"`
	ValidationPattern *string   `json:"validation_pattern,omitempty"`
	IsStorable        bool      `json:"is_storable"`
	IsActive          bool      `json:"is_active"`
	IsSupported       bool      `json:"is_supported"`
	SoftwareVer       *string   `json:"software_version,omitempty"`
	// T-PMSRC: 行级来源("builtin"/"custom")与可删标志(仅 custom 可删),前端据此渲染来源 Tag + 删除按钮。
	Source    string `json:"source"`
	Deletable bool   `json:"deletable"`
}

func toMappingView(m *ParamMapping) mappingView {
	source := m.Source
	if source == "" {
		source = "builtin" // 兜底:旧数据/未迁移场景按内置处理
	}
	return mappingView{
		ID: m.ID, ParamModelID: m.ParamModelID,
		StandardPath: m.StandardPath, PrivatePath: m.PrivatePath, EntryType: m.EntryType,
		Access: m.Access, DataType: m.DataType, ChangeApplies: m.ChangeApplies,
		MinValue: m.MinValue, MaxValue: m.MaxValue, EnumValues: m.EnumValues, EnumLabels: m.EnumLabels,
		ValidationPattern: m.ValidationPattern,
		IsStorable:        m.IsStorable, IsActive: m.IsActive, IsSupported: m.IsSupported, SoftwareVer: m.SoftwareVersion,
		Source: source, Deletable: source == "custom",
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
	response.OK(c, gin.H{"items": views, "total": len(views), "param_model": toModelView(h.baseDir, m)})
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
	if errors.Is(err, ErrDuplicateStandardPath) {
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
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
	// T-PMSRC：内置(source='builtin',来自 XML)映射不可删 → 403
	if errors.Is(err, ErrBuiltinMappingNotDeletable) {
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("内置映射不允许删除(来自 XML,只能删除自定义映射)[code=%d]", global.ErrCodeParamMappingBuiltinNotDeletable))
		return
	}
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
	StandardPath  string    `json:"standard_path"`
	EntryType     string    `json:"entry_type"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *int64    `json:"min_value,omitempty"`
	MaxValue      *int64    `json:"max_value,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdatedFields []string  `json:"updated_fields"`
}

func toStandardView(sp *StandardParam) standardView {
	return standardView{
		StandardPath: sp.StandardPath, EntryType: sp.EntryType,
		Access: sp.Access, DataType: sp.DataType, ChangeApplies: sp.ChangeApplies,
		MinValue: sp.MinValue, MaxValue: sp.MaxValue,
		UpdatedAt: sp.UpdatedAt, UpdatedFields: sp.UpdatedFields,
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

func (h *Handler) CreateStandard(c *gin.Context) {
	var req upsertStandardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	sp, err := h.repo.CreateStandardParam(c.Request.Context(), UpsertStandardParamInput{
		StandardPath:  req.StandardPath,
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
	})
	if err != nil {
		if errors.Is(err, ErrStandardParamExists) {
			commonerrors.AbortWithError(c, http.StatusConflict, commonerrors.NewBusinessError(
				global.ErrCodeStandardParamDuplicate,
				fmt.Sprintf("standard path %q already exists; update the existing record instead", req.StandardPath),
				err,
			))
			return
		}
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
	sp, err := h.repo.UpdateStandardParam(c.Request.Context(), UpsertStandardParamInput{
		StandardPath:  req.StandardPath,
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
	})
	if err != nil {
		if errors.Is(err, ErrStandardParamNotFound) {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
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

// ── Reload ──────────────────────────────────────────────────────────

// destructiveReload 执行 destructive 全量重载 + 删孤儿 + 刷新缓存,
// 是"导入 XML"端点(UploadXML)在写文件后串联调用的可复用逻辑
// (沿用旧 import-directory ?mode=reload 的语义):
//  1. 全量 ReloadOne("param-model") —— UPSERT 所有 XML 中的模型
//  2. 删除 DB 中所有未被本次加载触及(updated_at < startedAt)的 param_models
//     (孤儿模型);param_mappings CASCADE 删除;products.param_model_id SET NULL
//  3. registry.Refresh 刷新内存映射缓存(cache_version 协调由 Registry 内部处理)
//
// 容错:reload / 删孤儿 / 刷新缓存任一失败只 Warn 不致命(文件已落地,
// 调用方可重新导入重试)。返回 reloaded 标志(reload 成功且未致命失败)与
// orphansDeleted 计数,供端点响应反映实际状态。
func (h *Handler) destructiveReload(ctx context.Context, reason string) (reloaded bool, orphansDeleted int64) {
	if h.reloader == nil {
		h.logger.Warn("destructive reload skipped: dictloader registry not wired",
			zap.String("reason", reason))
		return false, 0
	}

	// 记录开始时间,便于事后按 updated_at 识别孤儿
	startedAt := time.Now()

	if err := h.reloader.ReloadOne(ctx, LoaderName); err != nil {
		h.logger.Warn("destructive reload failed",
			zap.String("reason", reason),
			zap.Error(err))
		return false, 0
	}
	reloaded = true

	deleted, err := h.repo.DeleteOrphansSince(ctx, startedAt)
	if err != nil {
		h.logger.Warn("cleanup orphan param_models failed",
			zap.String("reason", reason),
			zap.Error(err))
	} else {
		orphansDeleted = deleted
	}

	if h.registry != nil {
		if err := h.registry.Refresh(ctx); err != nil {
			h.logger.Warn("post-reload param registry refresh failed",
				zap.String("reason", reason),
				zap.Error(err))
		}
	}
	return reloaded, orphansDeleted
}

// refreshBoundDict 在导入成功后刷新绑定 sourceTable 的数据字典(T-0182 / #241)。
// best-effort:未接入(dictRefresher==nil)或刷新失败均只记日志,不影响导入响应。
func (h *Handler) refreshBoundDict(ctx context.Context, sourceTable string) {
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

// ── helpers ─────────────────────────────────────────────────────────

func (h *Handler) refreshAsync(ctx context.Context, op string) {
	if h.registry != nil {
		if err := h.registry.Refresh(ctx); err != nil {
			h.logger.Warn("param registry refresh after write failed", zap.String("op", op), zap.Error(err))
		}
	}
	if h.productRefresher != nil {
		if err := h.productRefresher.Refresh(ctx); err != nil {
			h.logger.Warn("product registry refresh after param model write failed", zap.String("op", op), zap.Error(err))
		}
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
