package admin

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
	"github.com/omcgo/omcgo/internal/core/response"
)

// DeadLetterReplayer 是 admin handler 与 runner.Runner 的解耦点。
// runner.Runner 自然实现 Replay(ctx, *dlq.DeadLetter) error，但 handler 不该直接
// 依赖 runner 包（避免上下游环），故在此声明小接口。
type DeadLetterReplayer interface {
	Replay(ctx context.Context, dl *dlq.DeadLetter) error
}

// DeadLetterHandler 提供运维端的死信队列管理 REST 端点。
//
// 端点（全部需要 ops admin 权限，由路由层 permGroup("ops") 保证）：
//
//	GET    /admin/dead-letters
//	GET    /admin/dead-letters/:id
//	DELETE /admin/dead-letters/:id
//	POST   /admin/dead-letters/:id/replay
type DeadLetterHandler struct {
	repo      dlq.Repository
	replayers map[string]DeadLetterReplayer // module → Replayer（PM/MR/Alarm…）
	logger    *zap.Logger
}

// NewDeadLetterHandler 构造 handler。logger 为 nil 时使用 zap.NewNop()。
// 调用 SetReplayer(module, runner) 注册可重放的模块；未注册的 module 重放时
// 返回 503 Service Unavailable（无可用 replayer）。
func NewDeadLetterHandler(repo dlq.Repository, logger *zap.Logger) *DeadLetterHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DeadLetterHandler{
		repo:      repo,
		replayers: make(map[string]DeadLetterReplayer),
		logger:    logger.Named("admin-dlq"),
	}
}

// SetReplayer 注册某 module 的 replayer。同名 module 后注册的覆盖前者。
func (h *DeadLetterHandler) SetReplayer(module string, replayer DeadLetterReplayer) {
	if h == nil || module == "" || replayer == nil {
		return
	}
	h.replayers[module] = replayer
}

// RegisterRoutes 在给定 RouterGroup 下注册 4 个端点。
// 调用方应在 permGroup("ops") 下挂载，确保 RBAC 生效。
func (h *DeadLetterHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/admin/dead-letters")
	{
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/replay", h.Replay)
	}
}

// listQuery 表示 GET 列表的查询参数。
//
// 不嵌入 model.ListRequest（其 binding:"min=1" 会让缺省请求 400）；这里保持
// 宽松绑定，缺省值在 handler 内填默认。
type listQuery struct {
	Module   string `form:"module"`
	Subject  string `form:"subject"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	SortBy   string `form:"sort_by"`
	SortDir  string `form:"sort_dir"`
}

// List 返回分页死信记录。可选过滤 module / subject。
func (h *DeadLetterHandler) List(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}

	filter := dlq.Filter{
		ListRequest: model.ListRequest{
			Page:     q.Page,
			PageSize: q.PageSize,
			SortBy:   q.SortBy,
			SortDir:  q.SortDir,
		},
	}
	if q.Module != "" {
		m := q.Module
		filter.Module = &m
	}
	if q.Subject != "" {
		s := q.Subject
		filter.Subject = &s
	}

	resp, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list dead-letters", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, resp)
}

// Get 返回单条死信详情。404 if not found。
func (h *DeadLetterHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	dl, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("get dead-letter", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dl == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, dl)
}

// Delete 删除一条死信记录。删除后续 Get 同 ID 返回 404。
func (h *DeadLetterHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete dead-letter", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, nil, "deleted")
}

// Replay 把死信 payload 重新 publish 到原 subject，记录不删除（PRD §8.3）。
// 未找到对应模块 replayer → 503；publisher 失败 → 500；记录 not found → 404。
func (h *DeadLetterHandler) Replay(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	dl, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("replay get", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dl == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	replayer := h.replayers[dl.SourceModule]
	if replayer == nil {
		h.logger.Warn("no replayer registered for module", zap.String("module", dl.SourceModule))
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
		return
	}
	if err := replayer.Replay(c.Request.Context(), dl); err != nil {
		h.logger.Error("replay publish", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"id": id.String()}, "republished")
}
