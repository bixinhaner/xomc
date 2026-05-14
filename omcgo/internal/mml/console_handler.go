package mml

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/response"
)

// ============================================================
// console_handler.go — T-0123-P1 S3-D3 Console HTTP 入口
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §N.4 / §N.5
//
// 5 端点（api_group="mml" inferApiGroup 自动派生）：
//
//	GET  /api/v1/mml/group-tree?root=<code>&lang=<lang>
//	GET  /api/v1/mml/commands/:id/sub-fields?lang=<lang>
//	POST /api/v1/mml/render
//	POST /api/v1/mml/parse
//	POST /api/v1/mml/execute-statements
//
// RBAC：路由组上层 RequireAPIPermission（端点级 Casbin）。lang 透传 zh-CN/en-US，
// 后端 i18n fallback chain 已在 ConsoleService 内处理。
//
// 错误翻译：
//   - ErrCommandNotFound        → 404
//   - bind / 参数校验失败          → 400
//   - 其他                       → 500
// ============================================================

// ConsoleHandler 暴露 ConsoleService 的 5 个端点。
//
// 设计：handler 持 *ConsoleService 与 MMLTaskCreator（=*Service）分离，使
// service 与协议执行通路解耦——ConsoleService 仅做编排与编译，task 持久化 /
// fanout 由上游 *Service 通过 MMLTaskCreator 接口承接。
type ConsoleHandler struct {
	svc         *ConsoleService
	taskCreator MMLTaskCreator
	logger      *zap.Logger
}

// NewConsoleHandler 构造 ConsoleHandler。taskCreator 必须非 nil（执行端点依赖）。
func NewConsoleHandler(svc *ConsoleService, taskCreator MMLTaskCreator, logger *zap.Logger) *ConsoleHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ConsoleHandler{
		svc:         svc,
		taskCreator: taskCreator,
		logger:      logger.Named("mml-console-handler"),
	}
}

// RegisterRoutes 注册 5 端点。父 router group 应为 v1 + RequireAPIPermission。
func (h *ConsoleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	mml := rg.Group("/mml")

	mml.GET("/group-tree", h.GetGroupTree)
	mml.GET("/commands/:id/sub-fields", h.GetCommandSubFields)
	mml.POST("/render", h.PostRender)
	mml.POST("/parse", h.PostParse)
	mml.POST("/execute-statements", h.PostExecuteStatements)
}

// ============================================================
// GET /api/v1/mml/group-tree?root=<code>&lang=<lang>
// ============================================================

// GroupTreeQuery 是 GET /mml/group-tree 的查询参数。
//
// root 缺省时返回全树根节点（按 path 顶级 ltree 自动判定）。
type GroupTreeQuery struct {
	Root string `form:"root"`
	Lang string `form:"lang"`
}

// GetGroupTree 返回 3 列布局左侧命令分组树（含每节点 commands[]）。
func (h *ConsoleHandler) GetGroupTree(c *gin.Context) {
	var q GroupTreeQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}
	lang := normalizeLang(q.Lang)
	tree, err := h.svc.BuildGroupTree(c.Request.Context(), q.Root, lang)
	if err != nil {
		h.logger.Error("build group tree", zap.Error(err), zap.String("root", q.Root))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"tree": tree})
}

// ============================================================
// GET /api/v1/mml/commands/:id/sub-fields?lang=<lang>
// ============================================================

// GetCommandSubFields 返回命令的 sub-fields（含 join mml_params 通用元数据），
// 按 sort_order 排序。lang 派生顶级 label / constraint_text。
func (h *ConsoleHandler) GetCommandSubFields(c *gin.Context) {
	id, ok := parseUUIDPathParam(c, "id")
	if !ok {
		return
	}
	lang := normalizeLang(c.Query("lang"))
	subFields, err := h.svc.GetCommandSubFields(c.Request.Context(), id, lang)
	if err != nil {
		h.logger.Error("get command sub_fields", zap.Error(err), zap.String("command_id", id.String()))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"sub_fields": subFields})
}

// ============================================================
// POST /api/v1/mml/render
// ============================================================

// PostRender 把 RenderRequest 渲染为 MML 字符串（双向绑定 outbound）。
func (h *ConsoleHandler) PostRender(c *gin.Context) {
	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	out, err := h.svc.RenderMML(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommandNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		h.logger.Error("render mml", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"mml_string": out})
}

// ============================================================
// POST /api/v1/mml/parse
// ============================================================

// PostParse 把 MML 字符串解析为 statements（双向绑定 inbound）。
// 单条 statement 解析失败累入 ParseErrors 不中断整体；HTTP 始终 200。
func (h *ConsoleHandler) PostParse(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.svc.ParseMML(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("parse mml", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, resp)
}

// ============================================================
// POST /api/v1/mml/execute-statements
// ============================================================

// PostExecuteStatements 把 N 设备 × M statements 编译为 MMLTask 并 fanout 执行。
// 多 statement → sequential=true；单 statement → 并发。
func (h *ConsoleHandler) PostExecuteStatements(c *gin.Context) {
	var req ExecuteStatementsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Creator == "" {
		if u := c.GetString("username"); u != "" {
			req.Creator = u
		}
	}
	if req.Executor == "" {
		req.Executor = req.Creator
	}

	task, err := h.svc.ExecuteStatements(c.Request.Context(), req, h.taskCreator)
	if err != nil {
		// 错误分级（sentinel）：
		//   ErrCommandNotFound  → 404（command_id 未命中 / op_logical_code 查不到）
		//   ErrInvalidRequest   → 400（空 statements / 空 devices / 编译时 sub_field 缺失 /
		//                              target_object 缺失 / RMV 缺 index 等用户输入错）
		//   其余                  → 500（DB 故障 / fanout 内部错误）
		switch {
		case errors.Is(err, ErrCommandNotFound):
			response.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidRequest):
			response.Fail(c, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("execute statements", zap.Error(err))
			response.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

// ============================================================
// Helpers
// ============================================================

// parseUUIDPathParam 解析 path param 为 UUID；失败 400 并直接终止。
func parseUUIDPathParam(c *gin.Context, key string) (uuid.UUID, bool) {
	raw := c.Param(key)
	id, err := uuid.Parse(raw)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid uuid in path param "+key)
		return uuid.Nil, false
	}
	return id, true
}

// normalizeLang 把 lang 参数规范化为 ConsoleService 期望的形式。
// 空 / 未知 → 默认 zh-CN（OMC 主语料）。
func normalizeLang(lang string) string {
	lang = strings.TrimSpace(lang)
	switch lang {
	case "zh-CN", "zh", "zh_CN":
		return "zh-CN"
	case "en-US", "en", "en_US":
		return "en-US"
	}
	return "zh-CN"
}
