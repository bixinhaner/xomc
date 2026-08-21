package mml

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
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
//
// compatibility 是 R-8.5 命令兼容性警告服务（独立 CompatibilityService 以避免
// 改动 ConsoleService 签名连累 5 个测试文件）；nil 时端点降级为 503，不影响
// 其他路由。
type ConsoleHandler struct {
	svc           *ConsoleService
	taskCreator   MMLTaskCreator
	compatibility *CompatibilityService
	logger        *zap.Logger
}

// NewConsoleHandler 构造 ConsoleHandler。taskCreator 必须非 nil（执行端点依赖）。
// compatibility 可为 nil — 端点返回 503 但其他端点不受影响。
func NewConsoleHandler(
	svc *ConsoleService,
	taskCreator MMLTaskCreator,
	compatibility *CompatibilityService,
	logger *zap.Logger,
) *ConsoleHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ConsoleHandler{
		svc:           svc,
		taskCreator:   taskCreator,
		compatibility: compatibility,
		logger:        logger.Named("mml-console-handler"),
	}
}

// RegisterRoutes 注册 5+1+1 端点。父 router group 应为 v1 + RequireAPIPermission。
//
// R-9.2 新增 `/mml/console/execute-statements-structured` — 与旧 `/execute-statements`
// 并存兼容期；前缀 /console/ 物理隔离便于未来下线旧端点。
// R-8.5 新增 `/mml/console/command-compatibility` — 命令兼容性警告。
func (h *ConsoleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	mml := rg.Group("/mml")

	mml.GET("/group-tree", h.GetGroupTree)
	// Bundle C: 命令搜索（按 command_code / logical_name / standardPath / description 联合 ILIKE）。
	// 静态段 "/commands/search" 必须在通配段 "/commands/:id/sub-fields" 之前注册 — 否则
	// gin tree 会把 "search" 当作 :id 路径变量匹配后者，返回 400/404。
	mml.GET("/commands/search", h.SearchCommands)
	mml.GET("/commands/:id/sub-fields", h.GetCommandSubFields)
	// 产品（按设备 product_class）不支持的参数 PATH —— 前端「选择命令 / 配置参数」过滤展示用。
	mml.GET("/unsupported-paths", h.GetUnsupportedPaths)
	mml.POST("/render", h.PostRender)
	mml.POST("/parse", h.PostParse)
	mml.POST("/execute-statements", h.PostExecuteStatements)
	mml.POST("/console/execute-statements-structured", h.PostExecuteStatementsStructured)
	mml.GET("/console/command-compatibility", h.GetCommandCompatibility)
}

// ============================================================
// GET /api/v1/mml/console/command-compatibility?product_class=<class>&lang=<lang>
// R-8.5
// ============================================================

// GetCommandCompatibility 返回指定 product_class 下不兼容的命令 ID 列表。
//
// 错误码：
//   - 400 product_class 缺失
//   - 503 compatibility service 未配置（启动期 nil 注入）
//   - 500 其他内部错误
//
// T-0177：孤儿 productClass（ProductRegistry 未匹配）不再返 404，而是返 200 +
// paramModelID=零值 + 全部 unsupported；前端据 paramModelID 是否零值提示
// "未配置参数模型" vs "全部不兼容"。
func (h *ConsoleHandler) GetCommandCompatibility(c *gin.Context) {
	if h.compatibility == nil {
		response.Fail(c, http.StatusServiceUnavailable, "compatibility service not configured")
		return
	}
	productClass := strings.TrimSpace(c.Query("product_class"))
	if productClass == "" {
		response.Fail(c, http.StatusBadRequest, "product_class is required")
		return
	}
	result, err := h.compatibility.GetCommandCompatibility(c.Request.Context(), productClass)
	if err != nil {
		h.logger.Error("get command compatibility",
			zap.Error(err), zap.String("product_class", productClass))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, result)
}

// ============================================================
// GET /api/v1/mml/group-tree?root=<code>&lang=<lang>
// ============================================================

// GroupTreeQuery 是 GET /mml/group-tree 的查询参数。
//
// root 缺省时返回全树根节点（按 path 顶级 ltree 自动判定）。
// format 缺省为 tree（向后兼容递归树）；format=flat 返 Task #4 扁平响应。
// product_class 缺省时不做产品级过滤（向后兼容）；非空时按 T-0172 方案 X
// 过滤命令并给每条命令挂 supported_path_count / unsupported_paths / product_resolved。
// product_class 非空时优先按产品/站型过滤；device_sn 仅作兼容兜底。
// 控制台多设备选择时同一批设备已被前端约束为同一站型，因此命令树不按设备数量求交集。
type GroupTreeQuery struct {
	Root         string `form:"root"`
	Lang         string `form:"lang"`
	Format       string `form:"format"`
	ProductClass string `form:"product_class"`
	DeviceSN     string `form:"device_sn"`
}

// GetGroupTree 返回命令树。
//   - format=flat：Task #4 扁平格式（章节 + 命令 + 内联 object_path / 约束）
//   - 其他·缺省：3 列布局左侧递归树（含每节点 commands[]，原有合同）
func (h *ConsoleHandler) GetGroupTree(c *gin.Context) {
	var q GroupTreeQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}
	productClass := strings.TrimSpace(q.ProductClass)
	deviceSN := strings.TrimSpace(q.DeviceSN)
	if strings.EqualFold(strings.TrimSpace(q.Format), "flat") {
		groups, err := h.svc.BuildFlatGroupTreeFiltered(c.Request.Context(), productClass, deviceSN)
		if err != nil {
			if errors.Is(err, ErrFlatTreeNotConfigured) {
				response.Fail(c, http.StatusServiceUnavailable, err.Error())
				return
			}
			h.logger.Error("build flat group tree", zap.Error(err))
			response.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.OK(c, FlatGroupTreeResponse{Groups: groups})
		return
	}
	lang := resolveLang(c, q.Lang)
	var tree []GroupTreeNode
	var err error
	if productClass != "" {
		tree, err = h.svc.BuildGroupTreeFiltered(c.Request.Context(), q.Root, lang, productClass)
	} else if deviceSN != "" {
		tree, err = h.svc.BuildGroupTreeFilteredByDevice(c.Request.Context(), q.Root, lang, deviceSN)
	} else {
		tree, err = h.svc.BuildGroupTreeFiltered(c.Request.Context(), q.Root, lang, "")
	}
	if err != nil {
		h.logger.Error("build group tree", zap.Error(err),
			zap.String("root", q.Root),
			zap.String("product_class", productClass),
			zap.String("device_sn", deviceSN))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"tree": tree})
}

// ============================================================
// GET /api/v1/mml/commands/search?q=<keyword>&lang=<lang>&limit=<n>
// Bundle C — 命令搜索（按 command_code / logical_name / standardPath / description）
// ============================================================

// SearchCommands 按关键字模糊匹配命令，复用 CommandTree 搜索 + AddTemplateModal 选 path 场景。
//
// 行为：
//   - q 空 / 仅空白 → 返 200 空数组（不消耗服务端 CPU 做"全表 LIMIT 50"）
//   - q 非空 → ILIKE %q% 联合 SELECT，按 (chapter, display_order, op_type, code) 排序
//   - limit 缺省 50，硬上限 200（由 repo 兜底）
//
// 错误码：
//   - 503 — search service 未配置（启动期 SearchRepo 为 nil；理论上不应发生）
//   - 500 — DB 查询错
func (h *ConsoleHandler) SearchCommands(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	lang := resolveLang(c, c.Query("lang"))
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	items, err := h.svc.SearchCommands(c.Request.Context(), q, lang, limit)
	if err != nil {
		if errors.Is(err, ErrSearchNotConfigured) {
			response.Fail(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		h.logger.Error("search commands", zap.Error(err), zap.String("q", q))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []SearchCommandDTO{}
	}
	response.OK(c, gin.H{"items": items})
}

// ============================================================
// GET /api/v1/mml/commands/:id/sub-fields?lang=<lang>
// ============================================================

// GetCommandSubFields 返回命令的 sub-fields（含 JOIN standard_params 元数据），
// 按 sort_order 排序。lang 派生顶级 label / constraint_text。
//
// 入参优先级（service 内实现）：
//   - ?product_class=<class>  console 主用（与命令树命令名计数口径对齐）
//   - ?device_sn=<sn> / ?device_id=<uuid>  admin / omcctl 兼容路径
//   - 都不传 → admin 视图全集
func (h *ConsoleHandler) GetCommandSubFields(c *gin.Context) {
	id, ok := parseUUIDPathParam(c, "id")
	if !ok {
		return
	}
	lang := resolveLang(c, c.Query("lang"))
	productClass := strings.TrimSpace(c.Query("product_class"))
	deviceKey := strings.TrimSpace(c.Query("device_sn"))
	if deviceKey == "" {
		deviceKey = strings.TrimSpace(c.Query("device_id"))
	}
	subFields, err := h.svc.GetCommandSubFields(c.Request.Context(), id, deviceKey, productClass, lang)
	if err != nil {
		h.logger.Error("get command sub_fields", zap.Error(err), zap.String("command_id", id.String()))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"sub_fields": subFields})
}

// GetUnsupportedPaths 返回某产品已记录的「不支持参数 PATH」列表（含读/写标记）。
//
//	GET /api/v1/mml/unsupported-paths?product_id=<uuid>   （前端主用，产品下拉直给）
//	GET /api/v1/mml/unsupported-paths?device_sn=<sn>      （兼容：后端反算 product_id）
//
// 来源：product_unsupported_paths（MML 执行 path 不支持类故障自学习）。前端据此在「选择命令 /
// 配置参数」按命令读/写类型过滤掉对应 path。
func (h *ConsoleHandler) GetUnsupportedPaths(c *gin.Context) {
	deviceSN := strings.TrimSpace(c.Query("device_sn"))
	var productID *uuid.UUID
	if pidStr := strings.TrimSpace(c.Query("product_id")); pidStr != "" {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid product_id")
			return
		}
		productID = &pid
	}
	paths, err := h.svc.GetUnsupportedPaths(c.Request.Context(), productID, deviceSN)
	if err != nil {
		h.logger.Error("get unsupported paths", zap.Error(err), zap.String("device_sn", deviceSN))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if paths == nil {
		paths = []UnsupportedPath{}
	}
	response.OK(c, gin.H{"paths": paths})
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
		//   ErrCommandNotFound        → 404
		//   ErrInvalidRequest         → 400
		//   ErrMixedProductClass      → 400 (R-8.4)
		//   ErrNoValidDevices         → 400 (R-8.4)
		//   ErrProductClassUnresolved → 422 (R-9.3 孤儿设备)
		//   ErrPathUnsupported        → 422 (T-0170 path 在 paramModel 无映射)
		//   其余                        → 500
		var mixedErr *ErrMixedProductClass
		var orphanErr *ErrProductClassUnresolved
		var unsupportedErr *ErrPathUnsupported
		switch {
		case isTaskAdmissionDenied(err):
			response.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrCommandNotFound):
			response.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidRequest):
			response.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoValidDevices):
			response.Fail(c, http.StatusBadRequest, err.Error())
		case errors.As(err, &mixedErr):
			response.Fail(c, http.StatusBadRequest, mixedErr.Error())
		case errors.As(err, &orphanErr):
			response.Fail(c, http.StatusUnprocessableEntity, orphanErr.Error())
		case errors.As(err, &unsupportedErr):
			response.FailWithData(c, http.StatusUnprocessableEntity, unsupportedErr.Error(),
				gin.H{"unsupported_paths": unsupportedErr.Paths, "param_model_id": unsupportedErr.ParamModelID})
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

// ============================================================
// POST /api/v1/mml/console/execute-statements-structured (R-9.2)
// ============================================================

// PostExecuteStatementsStructured 接收结构化入参（paths/values/instanceIndices）执行。
//
// 与旧 PostExecuteStatements 错误映射的差异：
//   - ErrUnknownPaths / ErrInstanceSelectorsNotImplemented → 422 + unknown_paths 元数据
//   - 其余错误（ErrCommandNotFound / ErrMixedProductClass / ...）复用旧 handler 同款映射
func (h *ConsoleHandler) PostExecuteStatementsStructured(c *gin.Context) {
	var req StructuredExecuteRequest
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

	task, err := h.svc.ExecuteStructured(c.Request.Context(), req, h.taskCreator)
	if err != nil {
		var unknownErr *ErrUnknownPaths
		var mixedErr *ErrMixedProductClass
		var orphanErr *ErrProductClassUnresolved
		var unsupportedErr *ErrPathUnsupported
		switch {
		case isTaskAdmissionDenied(err):
			response.Fail(c, http.StatusConflict, err.Error())
		case errors.As(err, &unknownErr):
			response.FailWithData(c, http.StatusUnprocessableEntity, unknownErr.Error(),
				gin.H{"unknown_paths": unknownErr.Paths, "command_id": unknownErr.CommandID})
		case errors.Is(err, ErrCommandNotFound):
			response.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidRequest):
			response.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoValidDevices):
			response.Fail(c, http.StatusBadRequest, err.Error())
		case errors.As(err, &mixedErr):
			response.Fail(c, http.StatusBadRequest, mixedErr.Error())
		case errors.As(err, &orphanErr):
			response.Fail(c, http.StatusUnprocessableEntity, orphanErr.Error())
		case errors.As(err, &unsupportedErr):
			response.FailWithData(c, http.StatusUnprocessableEntity, unsupportedErr.Error(),
				gin.H{"unsupported_paths": unsupportedErr.Paths, "param_model_id": unsupportedErr.ParamModelID})
		default:
			h.logger.Error("execute structured statements", zap.Error(err))
			response.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

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

// resolveLang 统一 MML console 的语言来源（issue #67 §5）：
//   - 显式 ?lang= 非空 → 作为覆盖，规范化后返回（保留前端按需切换语言能力）；
//   - 缺省 → 读 Accept-Language 注入的 ctx locale（与 PM 一致），未识别再回退 zh-CN。
//
// 这样未带 ?lang= 时仍能按浏览器/前端注入的 Accept-Language 本地化，不再硬退中文。
func resolveLang(c *gin.Context, rawLang string) string {
	if strings.TrimSpace(rawLang) != "" {
		return normalizeLang(rawLang)
	}
	return string(appcontext.GetLocale(c.Request.Context()))
}
