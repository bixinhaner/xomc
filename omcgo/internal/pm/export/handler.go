package export

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// presignURLTTL 下载签名链接有效期（与备份/快照下载一致，短 TTL）。
const presignURLTTL = time.Hour

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

// Handler 是 KPI 导出的 REST 入口。
type Handler struct {
	svc             *Service
	presigner       Presigner             // 启动期默认；nil 时下载端点返 503
	presignProvider PresignClientProvider // issue #548 切片 4：sys_configs 热改 endpoint 后下次 Download 即生效
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
		g.GET("/:id/download", h.Download) // 下载（签名链接）
		g.DELETE("/:id", h.Delete)         // 删除任务记录
	}
}

// ── 请求/响应 DTO ─────────────────────────────────────────────────────────

type createRequestDTO struct {
	SourceType string          `json:"source_type" binding:"required,oneof=dashboard adhoc"`
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
	task, err := h.svc.Create(c.Request.Context(), CreateRequest{
		TaskName:   defaultTaskName(req.TaskName, SourceType(req.SourceType)),
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

// List GET /pm/exports — 列导出任务（任务管理 Tab）。
func (h *Handler) List(c *gin.Context) {
	filter := h.parseListFilter(c)
	tasks, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
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
	response.OK(c, gin.H{"items": tasksToDTO(tasks)})
}

// Download GET /pm/exports/:id/download — 生成对象存储签名 GET 链接返回。
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
	if task.Status != StatusSucceeded || task.FilePath == "" || task.Bucket == "" {
		// 文件未就绪：合理 4xx（409 Conflict 表示资源当前状态不允许下载），不崩溃。
		response.Fail(c, http.StatusConflict, "export file not ready")
		return
	}
	if h.presigner == nil && (h.presignProvider == nil || h.presignProvider.Get() == nil) {
		response.Fail(c, http.StatusServiceUnavailable, "object storage not available")
		return
	}
	u, err := h.currentPresigner().PresignedGetObject(c.Request.Context(), task.Bucket, task.FilePath, presignURLTTL, url.Values{})
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

func tasksToDTO(tasks []Task) []taskResponseDTO {
	out := make([]taskResponseDTO, 0, len(tasks))
	for i := range tasks {
		out = append(out, taskToDTO(&tasks[i]))
	}
	return out
}

// defaultTaskName 任务名缺省自动生成 KPI导出_{来源}_{时间戳}。
func defaultTaskName(name string, source SourceType) string {
	if name != "" {
		return name
	}
	label := "仪表盘"
	if source == SourceAdhoc {
		label = "任务结果"
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
