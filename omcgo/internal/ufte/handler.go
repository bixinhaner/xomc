package ufte

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	service  *Service
	resolver *authz.Resolver
	logger   *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("ufte-handler"),
	}
}

// SetPermissionService 注入数据权限解析器（#63 设备组可见性强制层）。未注入时
// FromContext 走 nil-safe 退化（不过滤），与 device/alarm 模块语义一致。
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	ufte := rg.Group("/ufte")
	ufte.GET("/overview", h.GetOverview)
	ufte.GET("/task-types", h.ListTaskTypes)
	ufte.POST("/task-types", h.CreateTaskType)
	ufte.PUT("/task-types/:typeCode", h.UpdateTaskType)
	ufte.DELETE("/task-types/:typeCode", h.DeleteTaskType)
	ufte.GET("/tasks", h.ListTasks)
	ufte.POST("/tasks", h.CreateTask)
	ufte.PUT("/tasks/:id/start", h.StartTask)
	ufte.PUT("/tasks/:id/suspend", h.SuspendTask)
	ufte.PUT("/tasks/:id/terminate", h.TerminateTask)
	ufte.DELETE("/tasks/:id", h.DeleteTask)
	// 批量删除：body 走 POST 避免 DELETE+body 在某些代理 / WAF 下被吞
	ufte.POST("/tasks/batch-delete", h.BatchDeleteTasks)
	ufte.POST("/tasks/:id/retry", h.RetryTask)
	ufte.GET("/devices", h.ListDevices)
	ufte.GET("/devices/export", h.ExportDevices)
	ufte.GET("/device-candidates", h.ListDeviceCandidates)
}

func (h *Handler) GetOverview(c *gin.Context) {
	result, err := h.service.GetOverview(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListTaskTypes(c *gin.Context) {
	items, err := h.service.GetTaskTypes(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, items)
}

func (h *Handler) CreateTaskType(c *gin.Context) {
	var req TaskTypeWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	editor := currentUsername(c)
	item, err := h.service.CreateTaskType(c.Request.Context(), req, editor)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, item)
}

func (h *Handler) UpdateTaskType(c *gin.Context) {
	var req TaskTypeWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.UpdateTaskType(c.Request.Context(), c.Param("typeCode"), req, currentUsername(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) DeleteTaskType(c *gin.Context) {
	if err := h.service.DeleteTaskType(c.Request.Context(), c.Param("typeCode")); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"typeCode": c.Param("typeCode")})
}

// ListTasks returns transfer-center tasks across upgrades, log collection,
// configuration backup and restore, and other unified file-transfer workflows.
//
// @Summary 查询传输中心任务
// @Description 查询统一文件传输任务汇总；日志收集任务使用 category=station_log 和 typeCode=RUNTIME_LOG_COLLECT，结束任务的 result 可区分成功、部分成功和失败。
func (h *Handler) ListTasks(c *gin.Context) {
	var filter TaskListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	visibleGroups, ok := h.resolver.FromContext(c) // #63 设备组可见性
	if !ok {
		return
	}
	result, err := h.service.ListTasks(c.Request.Context(), filter, visibleGroups)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}
	creator := currentUsername(c)
	task, err := h.service.CreateTask(c.Request.Context(), req, creator, visibleGroups)

	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionUpgrade
	entry.ResourceType = audit.ResourceUpgradeTask
	entry.Success = err == nil
	if task != nil {
		entry.ResourceID = task.ID
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.LogAsync(entry)

	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

// ListDevices returns device-level execution details for transfer-center tasks.
//
// @Summary 查询传输任务设备执行明细与失败原因
// @Description 可按 taskId、category、typeCode 和 status 筛选；失败记录返回 failureReason 和设备上报的原始 failureDetail，适用于日志收集、升级、备份和恢复任务排障。
func (h *Handler) ListDevices(c *gin.Context) {
	var filter DeviceListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	visibleGroups, ok := h.resolver.FromContext(c) // #63 设备组可见性
	if !ok {
		return
	}
	result, err := h.service.ListDevices(c.Request.Context(), filter, visibleGroups)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

const failureCodeOperatorTerminated = "OPERATOR_TERMINATED"

// failureReasonZH 把 sub_task.failure_reason 翻成 UI 同款中文（与前端 i18n
// software.failureCode.* + 终止任务历史兜底值对齐）。
// 后端不维护多语言资源；只针对当前已知 code 给出 zh-CN 显示。前端 i18n bundle
// 仍是真值源，本表只在 CSV 导出场景"近似还原"显示文本。
var failureReasonZH = map[string]string{
	"DEVICE_NOT_FOUND":              "任务无法启动，设备不存在",
	"DEVICE_LOCKED":                 "任务无法启动，设备已在其他任务中运行",
	"DEVICE_OFFLINE":                "任务无法启动，设备离线",
	"COMMAND_PUSH_FAILED":           "任务无法启动，下发命令失败",
	"DOWNLOAD_TIMEOUT":              "下载响应超时，未收到设备 DownloadResponse",
	"DOWNLOAD_FILE_ERROR":           "下载失败，找不到目标文件",
	"DOWNLOAD_FAULT":                "下载失败，设备拒绝 Download 请求",
	"UPLOAD_FAULT":                  "上传失败，设备拒绝 Upload / SetParameterValues 请求",
	"ROLLBACK_ENABLE_CHECK_TIMEOUT": "回退能力检查超时，未收到设备 GetParameterValuesResponse",
	"ROLLBACK_APPLY_TIMEOUT":        "回退执行超时，下发 SetParameterValues 后未收到设备重启完成上报",
	"ROLLBACK_ENABLE_CHECK_FAULT":   "回退能力检查失败，设备拒绝 GetParameterValues 请求",
	"ROLLBACK_SET_FAULT":            "回退失败，设备拒绝 SetParameterValues 请求",
	"ROLLBACK_NOT_SUPPORTED":        "回退失败，设备不支持回退",
	"TC_FAULT":                      "文件传输失败，设备 TransferComplete 异常",
	"UPGRADE_5G_FAILED":             "升级失败，5G 升级状态异常",
	"VERSION_MISMATCH":              "升级失败，重启后版本与目标版本不一致",
	"TASK_TIMEOUT":                  "任务超时，未收到设备 TransferComplete",
	"FIRMWARE_NOT_FOUND":            "任务无法启动，固件文件不存在",
	"INTERNAL_ERROR":                "系统内部错误",
	failureCodeOperatorTerminated:   "被操作者终止",
	// SoftwareService.TerminateUpgrade 给被终止 sub_task 写的固定字符串
	"task terminated by operator": "被操作者终止",
	// 旧版 UFTE 设备列表曾在 failure_reason 为空时补中文短标，导出侧保留兼容。
	"终止": "被操作者终止",
}

func translateFailureReason(raw string) string {
	if raw == "" {
		return ""
	}
	if v, ok := failureReasonZH[raw]; ok {
		return v
	}
	return raw
}

// translateDeviceStatus 把 sub_task status 翻成 UI 同款中文标签（与
// omcmb/.../shared.tsx::renderDeviceStatus 对齐），CSV 显示和页面一致。
func translateDeviceStatus(s string) string {
	switch s {
	case "downloading":
		return "下载中"
	case "uploading":
		return "上传中"
	case "awaiting_tc":
		return "等待 TransferComplete"
	case "rollback_checking":
		return "回退能力检查中"
	case "rolling_back":
		return "回退中"
	case "verifying":
		return "校验中"
	case "suspended":
		return "已挂起 / 待上线"
	case "ended":
		return "已完成"
	case "failed":
		return "失败"
	default:
		return "待执行"
	}
}

// ExportDevices 真流式导出设备子任务列表为 CSV。
//
// 复用 ListDevices 过滤条件；走 Service.StreamDeviceItems 不在内存累积全量行，
// 每命中一条立刻写 csv.Writer，每 500 行 Flush 一次把字节推到客户端 HTTP 流，
// 大数据集下浏览器能边下边显进度，服务端内存占用稳定（≈ 1 批 + caches）。
//
// 为什么不让前端 pageSize=10000：① backend 可能限上限 ② JSON 序列化大量行内存
// 膨胀 ③ 客户端拼 CSV 字段含 , " 换行 时转义易错。后端 csv.Writer 标准库自动转义。
func (h *Handler) ExportDevices(c *gin.Context) {
	var filter DeviceListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// #63 设备组可见性——必须在写出 200/header 之前解析，FromContext 失败会 abort 403。
	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}

	stamp := time.Now().Format("20060102-150405")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition",
		fmt.Sprintf("attachment; filename=ufte-devices-%s.csv", stamp))
	// 不设 Content-Length → Go net/http 自动用 Transfer-Encoding: chunked
	// （Flush 才真正生效，否则 ResponseWriter 内部缓冲后才一次性发）
	c.Status(http.StatusOK)

	w := c.Writer
	flusher, _ := w.(http.Flusher) // gin 的 ResponseWriter 实现了 http.Flusher

	// UTF-8 BOM：Excel 直接打开中文不乱码
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	csvW := csv.NewWriter(w)

	// view=upgrade 时按"4G/5G 升级"页签列名输出；其它（备份/恢复/license/log）
	// 按通用页签列名输出。与 FileTransferCenter 表头严格对齐。
	view := c.Query("view")
	upgrade := view == "upgrade"
	var headers []string
	if upgrade {
		headers = []string{
			"基站编码", "任务名称", "源版本", "目标版本", "升级类型",
			"产品名称", "升级进度(%)", "结果", "操作人", "失败原因",
			// issue #655：三皮肤设备列表统一为「开始时间 / 结束时间」，CSV 同步加两列；
			// 保留旧「操作时间」列（= LastReportAt）避免破坏老脚本/北向消费。
			"开始时间", "结束时间", "操作时间",
		}
	} else {
		headers = []string{
			// #529：去掉独立「设备名称」列——v1 标准皮肤设备列表非升级视图把设备名
			// 并进「任务/设备」组合列(不作独立列展示)，CSV 据此对齐，避免「导出比页面多一列」。
			"任务名称", "设备 SN", "产品名称", "当前版本",
			"目标版本/目标文件", "状态", "进度(%)", "失败原因",
			// issue #655：同上，加「开始时间 / 结束时间」并保留旧「上报时间」。
			"开始时间", "结束时间", "上报时间",
		}
	}
	_ = csvW.Write(headers)

	rowCount := 0
	const flushEvery = 500
	streamErr := h.service.StreamDeviceItems(c.Request.Context(), filter, visibleGroups,
		func(it DeviceItem) error {
			var row []string
			// 产品列与页面口径一致（#492/#524）：优先展示 ProductRegistry 解析出的产品英文名
			// （ProductName），解析不到（孤儿 / 未注册设备）回退裸 productClass，对齐前端
			// 设备列表「产品名称」列 render 的 productName || productType。
			productName := it.ProductName
			if productName == "" {
				productName = it.ProductType
			}
			if upgrade {
				startedAt := response.FormatTimeInCurrentLocation(c, it.StartedAt)
				endedAt := response.FormatTimeInCurrentLocation(c, it.EndedAt)
				lastReportAt := response.FormatTimeInCurrentLocation(c, it.LastReportAt)
				// 升级类型显示规则与前端 getUpgradeTypeLabel 一致
				upType := it.TypeDisplayName
				switch it.Category {
				case "gnb_upgrade", "enb_upgrade", "gsm_upgrade", "ups_upgrade", deviceUpgradeVirtualCategory:
					upType = "软件升级"
				case "version_rollback":
					upType = "版本回退"
				}
				row = []string{
					it.DeviceSN, it.TaskName, it.CurrentVersion, it.TargetVersion, upType,
					productName, fmt.Sprintf("%d", it.Progress),
					translateDeviceStatus(it.Status), it.OperatorScope,
					translateFailureReason(it.FailureReason),
					startedAt, endedAt, lastReportAt,
				}
			} else {
				startedAt := response.FormatTimeInCurrentLocation(c, it.StartedAt)
				endedAt := response.FormatTimeInCurrentLocation(c, it.EndedAt)
				lastReportAt := response.FormatTimeInCurrentLocation(c, it.LastReportAt)
				// "目标版本/目标文件" 在 UI 优先显示 targetFile，回退 targetVersion
				tgt := it.TargetFile
				if tgt == "" {
					tgt = it.TargetVersion
				}
				row = []string{
					// #529：列顺序与上方 headers 对齐——已去掉「设备名称」列。
					it.TaskName, it.DeviceSN, productName, it.CurrentVersion,
					tgt, translateDeviceStatus(it.Status),
					fmt.Sprintf("%d", it.Progress),
					translateFailureReason(it.FailureReason),
					startedAt, endedAt, lastReportAt,
				}
			}
			if err := csvW.Write(row); err != nil {
				return err
			}
			rowCount++
			if rowCount%flushEvery == 0 {
				csvW.Flush()
				if flusher != nil {
					flusher.Flush()
				}
			}
			return nil
		},
	)
	// 收尾：把 csv 内部 buffer + HTTP buffer 全部冲出去
	csvW.Flush()
	if flusher != nil {
		flusher.Flush()
	}
	if streamErr != nil {
		// HTTP status 已经发出去了改不了。只能记日志，下载文件末尾可能截断。
		h.logger.Warn("export devices stream error (response may be truncated)",
			zap.Error(streamErr), zap.Int("written_rows", rowCount))
	}
}

func (h *Handler) parseTaskID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return uuid.Nil, false
	}
	return id, true
}

// resolveTaskOp 解析 task id + 调用者可见设备组；二者任一失败已 abort，返回 false。
func (h *Handler) resolveTaskOp(c *gin.Context) (uuid.UUID, []uuid.UUID, bool) {
	id, ok := h.parseTaskID(c)
	if !ok {
		return uuid.Nil, nil, false
	}
	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return uuid.Nil, nil, false
	}
	return id, visibleGroups, true
}

func (h *Handler) StartTask(c *gin.Context) {
	id, visibleGroups, ok := h.resolveTaskOp(c)
	if !ok {
		return
	}
	if err := h.service.StartTask(c.Request.Context(), id, visibleGroups); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) SuspendTask(c *gin.Context) {
	id, visibleGroups, ok := h.resolveTaskOp(c)
	if !ok {
		return
	}
	if err := h.service.SuspendTask(c.Request.Context(), id, visibleGroups); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) TerminateTask(c *gin.Context) {
	id, visibleGroups, ok := h.resolveTaskOp(c)
	if !ok {
		return
	}
	if err := h.service.TerminateTask(c.Request.Context(), id, visibleGroups); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id, visibleGroups, ok := h.resolveTaskOp(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTask(c.Request.Context(), id, visibleGroups); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

// BatchDeleteTasksRequest 批量删除 body。
type BatchDeleteTasksRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required,min=1"`
}

// BatchDeleteTasksResponse 批量删除响应。
type BatchDeleteTasksResponse struct {
	Succeeded []string                     `json:"succeeded"`
	Failed    []BatchDeleteTaskFailureItem `json:"failed"`
}

// BatchDeleteTaskFailureItem 失败明细。
type BatchDeleteTaskFailureItem struct {
	TaskID string `json:"task_id"`
	Error  string `json:"error"`
}

// BatchDeleteTasks 批量删除任务；单条失败不影响其他。
func (h *Handler) BatchDeleteTasks(c *gin.Context) {
	var req BatchDeleteTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}
	ids := make([]uuid.UUID, 0, len(req.TaskIDs))
	invalidIDs := make([]BatchDeleteTaskFailureItem, 0)
	for _, raw := range req.TaskIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			invalidIDs = append(invalidIDs, BatchDeleteTaskFailureItem{
				TaskID: raw, Error: "invalid uuid: " + err.Error(),
			})
			continue
		}
		ids = append(ids, id)
	}
	results := h.service.BatchDeleteTasks(c.Request.Context(), ids, visibleGroups)
	resp := BatchDeleteTasksResponse{
		Succeeded: make([]string, 0, len(results)),
		Failed:    invalidIDs, // 先把 uuid 解析失败的塞进去
	}
	for _, r := range results {
		if r.Success {
			resp.Succeeded = append(resp.Succeeded, r.TaskID.String())
		} else {
			resp.Failed = append(resp.Failed, BatchDeleteTaskFailureItem{
				TaskID: r.TaskID.String(), Error: r.Error,
			})
		}
	}
	response.OK(c, resp)
}

func (h *Handler) RetryTask(c *gin.Context) {
	id, visibleGroups, ok := h.resolveTaskOp(c)
	if !ok {
		return
	}
	if err := h.service.RetryTask(c.Request.Context(), id, visibleGroups); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) ListDeviceCandidates(c *gin.Context) {
	var filter DeviceCandidateFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	visibleGroups, ok := h.resolver.FromContext(c) // #63 设备组可见性
	if !ok {
		return
	}
	result, err := h.service.ListDeviceCandidates(c.Request.Context(), filter, visibleGroups)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func currentUsername(c *gin.Context) string {
	if raw, ok := c.Get("username"); ok {
		if name, ok := raw.(string); ok && name != "" {
			return name
		}
	}
	return "system"
}
