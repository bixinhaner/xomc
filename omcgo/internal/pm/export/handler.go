package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
)

// presignURLTTL 下载签名链接有效期（与备份/快照下载一致，短 TTL）。
const presignURLTTL = time.Hour

var (
	errAdhocPermissionCheckerUnavailable = errors.New("adhoc task permission checker not available")
	errInvalidAdhocExportParams          = errors.New("invalid adhoc export params")
	errAdhocTaskNotVisible               = errors.New("adhoc task is not visible to current user")
)

// Presigner 生成对象存储签名 GET 链接的最小契约（便于单测 stub）。
// 真实实现由 minio.Client（PresignedGetObject）满足。
type Presigner interface {
	PresignedGetObject(ctx context.Context, bucket, object string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
}

// PresignClientProvider 抽象"按需取当前 MinIO 预签名 client"的能力（issue #548 切片 4）。
// 主线生产实现是 internal/core/components/minio.PresignBridge；sys_configs 写入
// storage.minio_public_endpoint 后下一次 Get() 拿到新 endpoint 对应的 client。
type PresignClientProvider interface {
	Get() *minio.Client
}

// AdhocTaskReader 是导出创建 adhoc 来源任务时需要的最小可见性检查能力。
type AdhocTaskReader interface {
	Get(ctx context.Context, id uuid.UUID) (*adhoc.Task, error)
}

// Handler 是 KPI 导出的 REST 入口。
type Handler struct {
	svc             *Service
	presigner       Presigner             // 启动期默认；nil 时下载端点返 503
	presignProvider PresignClientProvider // issue #548 切片 4：sys_configs 热改 endpoint 后下次 Download 即生效
	objectClient    *minio.Client         // app 内部可达的 MinIO client；默认用于文件管理同源流式下载
	adhocTasks      AdhocTaskReader       // adhoc 导出需复用任务可见性规则，避免 task_id 旁路私有数据
	logger          *zap.Logger
}

// NewHandler 构造 Handler。presigner 可为 nil（无对象存储环境，下载端点降级返 503）。
func NewHandler(svc *Service, presigner Presigner, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, presigner: presigner, logger: logger.Named("pm.export.handler")}
}

// SetPresignProvider 注入运行期感知 sys_configs 变更的 presign client provider（issue #548 切片 4）。
// nil 时回退使用 h.presigner（启动期静态注入的 client）。
func (h *Handler) SetPresignProvider(p PresignClientProvider) {
	if h == nil {
		return
	}
	h.presignProvider = p
}

// SetObjectClient 注入内部对象存储 client，用于默认同源流式下载。
func (h *Handler) SetObjectClient(c *minio.Client) {
	if h == nil {
		return
	}
	h.objectClient = c
}

// SetAdhocTaskReader 注入 adhoc 任务读取器，用于校验 adhoc 结果类导出的权限。
func (h *Handler) SetAdhocTaskReader(r AdhocTaskReader) {
	if h == nil {
		return
	}
	h.adhocTasks = r
}

// currentPresigner 返回当前 Download 该用的 Presigner：优先 provider.Get()、其次 h.presigner。
func (h *Handler) currentPresigner() Presigner {
	if h.presignProvider != nil {
		if c := h.presignProvider.Get(); c != nil {
			return c
		}
	}
	return h.presigner
}

// RegisterRoutes 把 5 个 REST 端点挂到 router group（不带 /pm 前缀，由调用方决定 group）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/pm/exports")
	{
		g.POST("", h.Create)               // 建导出任务
		g.GET("", h.List)                  // 列导出任务（任务管理 Tab）
		g.GET("/files", h.ListFiles)       // 列已成功的导出文件（文件管理 Tab）
		g.GET("/:id/download", h.Download) // 下载（默认同源流式；?mode=url 返回签名链接）
		g.DELETE("/:id", h.Delete)         // 删除任务记录
	}
}

// ── 请求/响应 DTO ─────────────────────────────────────────────────────────

type createRequestDTO struct {
	SourceType string          `json:"source_type" binding:"required,oneof=dashboard device_view kpi_query pm_dashboard adhoc_result"`
	Params     json.RawMessage `json:"params"`
	TaskName   string          `json:"task_name"`
}

type taskResponseDTO struct {
	ID         string          `json:"id"`
	TaskName   string          `json:"task_name"`
	SourceType string          `json:"source_type"`
	Params     json.RawMessage `json:"params"`
	Format     string          `json:"format"`
	Status     string          `json:"status"`
	RowCount   int64           `json:"row_count"`
	FileSize   int64           `json:"file_size"`
	Error      string          `json:"error,omitempty"`
	CreateUser string          `json:"create_user"`
	CreatedAt  time.Time       `json:"created_at"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
}

func taskToDTO(t *Task) taskResponseDTO {
	params := json.RawMessage(t.Params)
	if len(params) == 0 {
		params = json.RawMessage("{}")
	}
	return taskResponseDTO{
		ID:         t.ID.String(),
		TaskName:   t.TaskName,
		SourceType: string(t.SourceType),
		Params:     params,
		Format:     t.Format,
		Status:     string(t.Status),
		RowCount:   t.RowCount,
		FileSize:   t.FileSize,
		Error:      t.Error,
		CreateUser: t.CreateUser,
		CreatedAt:  t.CreatedAt,
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}

// Create POST /pm/exports
func (h *Handler) Create(c *gin.Context) {
	var req createRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var params []byte
	if len(req.Params) > 0 {
		params = []byte(req.Params)
	}
	locale := appcontext.GetLocale(c.Request.Context())
	params, err := withExportLocale(params, locale)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if requiresDashboardExportLimit(SourceType(req.SourceType)) {
		if err := validateDashboardExportLimits(params); err != nil {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	if SourceType(req.SourceType).usesAdhocResultExport() {
		if _, err := h.canAccessAdhocExport(c, params); err != nil {
			h.respondAdhocExportAccessError(c, err)
			return
		}
	}
	task, err := h.svc.Create(c.Request.Context(), CreateRequest{
		TaskName:   defaultTaskName(req.TaskName, SourceType(req.SourceType), locale),
		SourceType: SourceType(req.SourceType),
		Params:     params,
		CreateUser: extractCreateUser(c),
	})
	if err != nil {
		if errors.Is(err, ErrInvalidSourceType) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrJobRepoNotWired) {
			response.Fail(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, taskToDTO(task))
}

func requiresDashboardExportLimit(source SourceType) bool {
	return source == SourceDashboard || source == SourceDeviceView || source == SourceKpiQuery
}

func (h *Handler) canAccessExportTask(c *gin.Context, task *Task) (bool, error) {
	if task == nil || !task.SourceType.usesAdhocResultExport() {
		return true, nil
	}
	return h.canAccessAdhocExport(c, task.Params)
}

func (h *Handler) canAccessAdhocExport(c *gin.Context, params []byte) (bool, error) {
	if h.adhocTasks == nil {
		return false, errAdhocPermissionCheckerUnavailable
	}
	filter, err := parseAdhocParams(params)
	if err != nil {
		return false, fmt.Errorf("%w: %v", errInvalidAdhocExportParams, err)
	}
	task, err := h.adhocTasks.Get(c.Request.Context(), filter.TaskID)
	if err != nil {
		return false, err
	}
	if canExportAdhocTask(c, task) {
		return true, nil
	}
	return false, errAdhocTaskNotVisible
}

func (h *Handler) respondAdhocExportAccessError(c *gin.Context, err error) {
	if errors.Is(err, errAdhocPermissionCheckerUnavailable) {
		response.Fail(c, http.StatusServiceUnavailable, errAdhocPermissionCheckerUnavailable.Error())
		return
	}
	if errors.Is(err, errInvalidAdhocExportParams) {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, adhoc.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "adhoc task not found")
		return
	}
	if errors.Is(err, errAdhocTaskNotVisible) {
		response.Fail(c, http.StatusForbidden, errAdhocTaskNotVisible.Error())
		return
	}
	commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
}

func canExportAdhocTask(c *gin.Context, task *adhoc.Task) bool {
	if task == nil {
		return false
	}
	if task.IsBuiltin || isPMExportAdmin(c) || task.Visibility == adhoc.VisibilityPublic {
		return true
	}
	return task.Creator == extractCreateUser(c)
}

func isPMExportAdmin(c *gin.Context) bool {
	if v, ok := c.Get("is_super_admin"); ok {
		if b, ok := v.(bool); ok && b {
			return true
		}
	}
	if v, ok := c.Get("roles"); ok {
		if roles, ok := v.([]string); ok {
			for _, r := range roles {
				if r == "admin" || r == "super_admin" {
					return true
				}
			}
		}
	}
	return false
}

// List GET /pm/exports — 列导出任务（任务管理 Tab）。
func (h *Handler) List(c *gin.Context) {
	filter := h.parseListFilter(c)
	tasks, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	tasks, ok := h.filterVisibleExportTasks(c, tasks)
	if !ok {
		return
	}
	response.OK(c, gin.H{"items": tasksToDTO(tasks)})
}

// ListFiles GET /pm/exports/files — 列已成功的导出文件（文件管理 Tab）。
// 只筛 status='succeeded' 且 file_path 非空。
func (h *Handler) ListFiles(c *gin.Context) {
	filter := h.parseListFilter(c)
	filter.OnlyReady = true
	filter.Status = nil // OnlyReady 已隐含 succeeded，忽略外部 status 过滤避免冲突
	tasks, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	tasks, ok := h.filterVisibleExportTasks(c, tasks)
	if !ok {
		return
	}
	response.OK(c, gin.H{"items": tasksToDTO(tasks)})
}

// Download GET /pm/exports/:id/download — 默认同源流式下载 KPI 导出 CSV。
//
// 兼容旧调用：?mode=url 仍返回对象存储签名 GET 链接。
//
// 文件未就绪（非 succeeded 或 file_path 为空）返 4xx，不 500 崩溃。
func (h *Handler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid export task id")
		return
	}
	task, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "export task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := h.canAccessExportTask(c, task); err != nil {
		h.respondAdhocExportAccessError(c, err)
		return
	}
	if task.Status != StatusSucceeded || task.FilePath == "" || task.Bucket == "" {
		// 文件未就绪：合理 4xx（409 Conflict 表示资源当前状态不允许下载），不崩溃。
		response.Fail(c, http.StatusConflict, "export file not ready")
		return
	}
	if c.Query("mode") == "url" {
		h.respondDownloadURL(c, id, task)
		return
	}
	if h.objectClient == nil {
		response.Fail(c, http.StatusServiceUnavailable, "object storage not available")
		return
	}
	obj, err := h.objectClient.GetObject(c.Request.Context(), task.Bucket, task.FilePath, minio.GetObjectOptions{})
	if err != nil {
		h.logger.Warn("open export object failed",
			zap.String("task_id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		h.logger.Warn("stat export object failed",
			zap.String("task_id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	filename := exportDownloadFilename(task)
	c.Header("Content-Disposition", exportContentDisposition(filename))
	c.DataFromReader(http.StatusOK, stat.Size, "text/csv; charset=utf-8", obj, nil)
}

func (h *Handler) respondDownloadURL(c *gin.Context, id uuid.UUID, task *Task) {
	presigner := h.currentPresigner()
	if presigner == nil {
		response.Fail(c, http.StatusServiceUnavailable, "object storage not available")
		return
	}
	u, err := presigner.PresignedGetObject(c.Request.Context(), task.Bucket, task.FilePath, presignURLTTL, url.Values{})
	if err != nil {
		h.logger.Warn("presign export download url failed",
			zap.String("task_id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"download_url": u.String()})
}

// Delete DELETE /pm/exports/:id — 删除任务记录。
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid export task id")
		return
	}
	task, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "export task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := h.canAccessExportTask(c, task); err != nil {
		h.respondAdhocExportAccessError(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "export task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

// ── helper ────────────────────────────────────────────────────────────────

func (h *Handler) filterVisibleExportTasks(c *gin.Context, tasks []Task) ([]Task, bool) {
	out := make([]Task, 0, len(tasks))
	for i := range tasks {
		allowed, err := h.canAccessExportTask(c, &tasks[i])
		if err != nil {
			if errors.Is(err, errAdhocTaskNotVisible) ||
				errors.Is(err, adhoc.ErrNotFound) ||
				errors.Is(err, errInvalidAdhocExportParams) {
				continue
			}
			h.respondAdhocExportAccessError(c, err)
			return nil, false
		}
		if allowed {
			out = append(out, tasks[i])
		}
	}
	return out, true
}

func (h *Handler) parseListFilter(c *gin.Context) ListFilter {
	f := ListFilter{Limit: 20}
	if v := c.Query("source_type"); v != "" {
		st := SourceType(v)
		f.SourceType = &st
	}
	if v := c.Query("status"); v != "" {
		s := Status(v)
		f.Status = &s
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}
	return f
}

func exportDownloadFilename(task *Task) string {
	name := strings.TrimSpace(task.TaskName)
	if name == "" {
		name = "kpi_export_" + task.ID.String()
	}
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == "/" {
		name = "kpi_export_" + task.ID.String()
	}
	if strings.ToLower(filepath.Ext(name)) != ".csv" {
		name += ".csv"
	}
	return name
}

func exportContentDisposition(filename string) string {
	ascii := asciiAttachmentFilename(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(filename))
}

func asciiAttachmentFilename(filename string) string {
	var b strings.Builder
	for _, r := range filename {
		if r <= 31 || r == 127 || r == '"' || r == '\\' || r == ';' || r == '/' {
			b.WriteByte('_')
			continue
		}
		if r > 126 {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	name := strings.TrimSpace(b.String())
	if name == "" {
		return "kpi_export.csv"
	}
	return name
}

func tasksToDTO(tasks []Task) []taskResponseDTO {
	out := make([]taskResponseDTO, 0, len(tasks))
	for i := range tasks {
		out = append(out, taskToDTO(&tasks[i]))
	}
	return out
}

// defaultTaskName 任务名缺省自动生成 KPI导出_{来源}_{时间戳}。
func defaultTaskName(name string, source SourceType, locale appcontext.Locale) string {
	if name != "" {
		return name
	}
	if locale == appcontext.LocaleEN {
		label := "Dashboard"
		if source == SourceDeviceView {
			label = "Device_Performance_View"
		} else if source == SourceKpiQuery {
			label = "KPI_Query"
		} else if source == SourcePMDashboard {
			label = "Performance_Dashboard"
		} else if source == SourceAdhocResult {
			label = "Adhoc_Aggregation_Task"
		}
		return "KPI_Export_" + label + "_" + time.Now().Format("20060102_150405")
	}

	label := "仪表盘"
	if source == SourceDeviceView {
		label = "设备性能查看"
	} else if source == SourceKpiQuery {
		label = "指标查询"
	} else if source == SourcePMDashboard {
		label = "性能仪表盘"
	} else if source == SourceAdhocResult {
		label = "自定义聚合任务"
	}

	return "KPI导出_" + label + "_" + time.Now().Format("20060102_150405")
}

// extractCreateUser 从 gin context 取当前用户名（与 adhoc 模块口径一致）。
func extractCreateUser(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "anonymous"
}
