package ops

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// TaskCreator is the slice of *Service that ExtHandler needs to enqueue
// an ad-hoc RPC command as an OpsTask (T-0102-b). Defined here as a
// consumer-driven small interface so the handler test can supply a
// stub without standing up the full Service + repos.
type TaskCreator interface {
	CreateTask(ctx context.Context, task *OpsTask) (*OpsTask, error)
}

// ExtHandler F06 运维管理扩展端点（T-0104..T-0111）。
type ExtHandler struct {
	diagSvc        *DiagnosticService
	downloadSvc    *DownloadService
	auditSvc       *AuditLogService
	maintenanceSvc *MaintenanceWindowService
	playbookSvc    *PlaybookService
	executor       TaskExecutorEngine
	approvalSvc    *ApprovalService
	breakGlassSvc  *BreakGlassService
	inspectionSvc  *InspectionService
	taskCreator    TaskCreator
	sseHub         *SSEHub
	logger         *zap.Logger
}

func NewExtHandler(
	diagSvc *DiagnosticService,
	downloadSvc *DownloadService,
	auditSvc *AuditLogService,
	maintenanceSvc *MaintenanceWindowService,
	playbookSvc *PlaybookService,
	executor *TaskExecutor,
	approvalSvc *ApprovalService,
	breakGlassSvc *BreakGlassService,
	inspectionSvc *InspectionService,
	taskCreator TaskCreator,
	sseHub *SSEHub,
	logger *zap.Logger,
) *ExtHandler {
	return &ExtHandler{
		diagSvc:        diagSvc,
		downloadSvc:    downloadSvc,
		auditSvc:       auditSvc,
		maintenanceSvc: maintenanceSvc,
		playbookSvc:    playbookSvc,
		executor:       executor,
		approvalSvc:    approvalSvc,
		breakGlassSvc:  breakGlassSvc,
		inspectionSvc:  inspectionSvc,
		taskCreator:    taskCreator,
		sseHub:         sseHub,
		logger:         logger.Named("ops.ext"),
	}
}

// RegisterRoutes 在 /api/v1/ops 前缀下注册扩展端点。
//
// 调用方应在已有 ops Handler.RegisterRoutes 之后调本方法，共用同一 rg。
func (h *ExtHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ops := rg.Group("/ops")

	// 诊断
	diag := ops.Group("/diagnostics")
	diag.POST("/ping", h.DiagPing)
	diag.POST("/traceroute", h.DiagTraceroute)
	diag.POST("/throughput", h.DiagThroughput)
	diag.GET("", h.ListDiagnostics)
	diag.GET("/:id", h.GetDiagnostic)
	diag.POST("/inspection", h.RunInspection)

	// 下载
	dl := ops.Group("/downloads")
	dl.POST("/collect", h.CollectDownload)
	dl.GET("", h.ListDownloads)
	dl.GET("/:id", h.GetDownload)

	// 审计
	audit := ops.Group("/audit-logs")
	audit.GET("", h.ListAuditLogs)

	// 维护窗口
	mw := ops.Group("/maintenance-windows")
	mw.POST("", h.CreateMaintenanceWindow)
	mw.GET("", h.ListMaintenanceWindows)
	mw.GET("/active", h.ListActiveMaintenanceWindows)
	mw.GET("/:id", h.GetMaintenanceWindow)
	mw.POST("/:id/approve", h.ApproveMaintenanceWindow)

	// 知识库
	pb := ops.Group("/playbooks")
	pb.GET("", h.ListPlaybooks)
	pb.POST("", h.CreatePlaybook)
	pb.POST("/match", h.MatchPlaybook)

	// 任务编排扩展
	tasks := ops.Group("/tasks")
	tasks.POST("/:id/run", h.RunTask)
	tasks.GET("/:id/executions", h.ListTaskExecutions)
	tasks.POST("/:id/approve", h.ApproveTask)
	tasks.GET("/:id/events", h.TaskEventsSSE)

	// 即时命令（SSE 通道）
	cmd := ops.Group("/commands")
	cmd.POST("/rpc", h.ExecuteRPC)
	cmd.GET("/:id/stream", h.CommandStreamSSE)

	// 紧急响应
	bg := ops.Group("/break-glass")
	bg.POST("/activate", h.ActivateBreakGlass)
	bg.POST("/deactivate", h.DeactivateBreakGlass)
	bg.GET("/status", h.BreakGlassStatus)
}

// ============================================================
// 诊断
// ============================================================

func (h *ExtHandler) DiagPing(c *gin.Context) {
	var req DiagPingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	d, err := h.diagSvc.Ping(c.Request.Context(), req, currentOperator(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, d)
}

func (h *ExtHandler) DiagTraceroute(c *gin.Context) {
	var req DiagTracerouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	d, err := h.diagSvc.Traceroute(c.Request.Context(), req, currentOperator(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, d)
}

func (h *ExtHandler) DiagThroughput(c *gin.Context) {
	var req DiagThroughputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	d, err := h.diagSvc.Throughput(c.Request.Context(), req, currentOperator(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, d)
}

func (h *ExtHandler) ListDiagnostics(c *gin.Context) {
	filter := DiagnosticFilter{
		DeviceSN:    c.Query("device_sn"),
		DiagType:    c.Query("diag_type"),
		ListRequest: parseListRequest(c),
	}
	if s := c.Query("status"); s != "" {
		st := DiagnosticStatus(s)
		filter.Status = &st
	}
	resp, err := h.diagSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ExtHandler) GetDiagnostic(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	d, err := h.diagSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if stderrors.Is(err, commonerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "diagnostic not found")
			return
		}
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *ExtHandler) RunInspection(c *gin.Context) {
	var req struct {
		Scope string `json:"scope"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Scope == "" {
		req.Scope = "all"
	}
	if err := h.inspectionSvc.RunOnce(c.Request.Context(), req.Scope, currentOperator(c)); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "triggered", "scope": req.Scope})
}

// ============================================================
// 下载
// ============================================================

func (h *ExtHandler) CollectDownload(c *gin.Context) {
	var req CollectDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	items, err := h.downloadSvc.Collect(c.Request.Context(), req, currentOperator(c))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"items": items})
}

func (h *ExtHandler) ListDownloads(c *gin.Context) {
	filter := DownloadFilter{
		DeviceSN:    c.Query("device_sn"),
		ContentType: c.Query("content_type"),
		ListRequest: parseListRequest(c),
	}
	if s := c.Query("status"); s != "" {
		st := DownloadStatus(s)
		filter.Status = &st
	}
	resp, err := h.downloadSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ExtHandler) GetDownload(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	d, err := h.downloadSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if stderrors.Is(err, commonerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "download not found")
			return
		}
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, d)
}

// ============================================================
// 审计
// ============================================================

func (h *ExtHandler) ListAuditLogs(c *gin.Context) {
	filter := AuditLogFilter{
		OpType:      c.Query("op_type"),
		TargetType:  c.Query("target_type"),
		TargetID:    c.Query("target_id"),
		ListRequest: parseListRequest(c),
	}
	if u := c.Query("operator_user_id"); u != "" {
		if uid, err := uuid.Parse(u); err == nil {
			filter.OperatorUserID = &uid
		}
	}
	if r := c.Query("risk_level"); r != "" {
		rl := RiskLevel(r)
		filter.RiskLevel = &rl
	}
	if bg := c.Query("break_glass"); bg == "true" {
		t := true
		filter.BreakGlass = &t
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.FromTime = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.ToTime = &t
		}
	}
	resp, err := h.auditSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================================
// 维护窗口
// ============================================================

func (h *ExtHandler) CreateMaintenanceWindow(c *gin.Context) {
	var req CreateMaintenanceWindowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	creator := currentUserID(c)
	w, err := h.maintenanceSvc.Create(c.Request.Context(), req, creator)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (h *ExtHandler) ListMaintenanceWindows(c *gin.Context) {
	filter := MaintenanceWindowFilter{ListRequest: parseListRequest(c)}
	if s := c.Query("status"); s != "" {
		ws := MaintenanceWindowStatus(s)
		filter.Status = &ws
	}
	resp, err := h.maintenanceSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ExtHandler) ListActiveMaintenanceWindows(c *gin.Context) {
	items, err := h.maintenanceSvc.ListActive(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *ExtHandler) GetMaintenanceWindow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	w, err := h.maintenanceSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if stderrors.Is(err, commonerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "maintenance_window not found")
			return
		}
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *ExtHandler) ApproveMaintenanceWindow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	approver := currentUserID(c)
	if approver == nil {
		response.Fail(c, http.StatusForbidden, "approver not identified")
		return
	}
	if err := h.maintenanceSvc.Approve(c.Request.Context(), id, *approver); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// ============================================================
// 知识库
// ============================================================

func (h *ExtHandler) ListPlaybooks(c *gin.Context) {
	filter := PlaybookFilter{
		Keyword:     c.Query("keyword"),
		ListRequest: parseListRequest(c),
	}
	resp, err := h.playbookSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ExtHandler) CreatePlaybook(c *gin.Context) {
	var p OpsPlaybook
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	out, err := h.playbookSvc.Create(c.Request.Context(), &p)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *ExtHandler) MatchPlaybook(c *gin.Context) {
	var req MatchPlaybookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	items, err := h.playbookSvc.Match(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// ============================================================
// 任务编排扩展
// ============================================================

func (h *ExtHandler) RunTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.executor.Run(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "running"})
}

func (h *ExtHandler) ListTaskExecutions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	filter := TaskExecutionFilter{
		TaskID:      &id,
		DeviceSN:    c.Query("device_sn"),
		Status:      c.Query("status"),
		ListRequest: parseListRequest(c),
	}
	resp, err := h.executor.ListExecutions(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ExtHandler) ApproveTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	approver := currentUserID(c)
	if approver == nil {
		response.Fail(c, http.StatusForbidden, "approver not identified")
		return
	}
	var req struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.approvalSvc.Approve(c.Request.Context(), taskID, *approver, req.Approve, req.Reason); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TaskEventsSSE SSE 推送任务进度（T-0102）。
func (h *ExtHandler) TaskEventsSSE(c *gin.Context) {
	id := c.Param("id")
	streamSSE(c, h.sseHub, "task:"+id)
}

// ============================================================
// 即时命令
// ============================================================

// ExecuteRPC enqueues an ad-hoc RPC command as an OpsTask (T-0102-b).
//
// Wire contract (POST /api/v1/ops/commands/rpc):
//
//	request  : { device_sn | device_sns[], action, params? }
//	response : 202 Accepted
//	           { task_id, status, approval_required, stream_url }
//
// Behavior:
//  1. Risk classify by action (classifyRiskForAction) — combined with
//     device count, ApprovalService.EvaluateRiskLevel produces the task's
//     overall risk.
//  2. Persist an OpsTask via the consumer-driven TaskCreator interface.
//     Inline command spec (action + params) lives in OpsTask.Message as
//     a JSON envelope so T-0102-c's RPC dispatcher can read it back.
//  3. If risk is L3 (dangerous), task is created with approval_state=pending
//     and is NOT auto-dispatched — operator must call POST /tasks/:id/approve.
//     Otherwise approval_state=not_required and a fire-and-forget goroutine
//     calls executor.Run to move task → running.
//  4. Audit log + SSE "command.enqueued" event published on the per-task
//     channel "command:<task_id>" so the existing CommandStreamSSE handler
//     can deliver realtime updates.
func (h *ExtHandler) ExecuteRPC(c *gin.Context) {
	var req struct {
		DeviceSN  string                 `json:"device_sn,omitempty"`
		DeviceSNs []string               `json:"device_sns,omitempty"`
		Action    string                 `json:"action" binding:"required"`
		Params    map[string]interface{} `json:"params,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}

	devices := req.DeviceSNs
	if len(devices) == 0 && req.DeviceSN != "" {
		devices = []string{req.DeviceSN}
	}
	if len(devices) == 0 {
		response.Fail(c, http.StatusBadRequest, "device_sn or device_sns required")
		return
	}

	op := currentOperator(c)
	actionRisk := classifyRiskForAction(req.Action)
	overallRisk := h.approvalSvc.EvaluateRiskLevel(len(devices), actionRisk)
	approvalState := ApprovalNotRequired
	if h.approvalSvc.RequiresApproval(overallRisk) {
		approvalState = ApprovalPending
	}

	// Validate the action up-front: an undispatchable action should fail
	// the request, not get stuck in the queue forever (T-0102-c).
	if _, err := actionToRPCMethod(req.Action); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Inline command spec lives in Message until OpsTask gains a dedicated
	// inline_command column. T-0102-c reads the envelope back to dispatch
	// real RPCs against the ACS engine.
	cmdJSON, _ := json.Marshal(RPCInlineEnvelope{
		Kind:   RPCInlineKind,
		Action: req.Action,
		Params: req.Params,
	})
	deviceSNsJSON, _ := json.Marshal(devices)

	task := &OpsTask{
		TaskName:      buildRPCTaskName(req.Action, devices),
		DeviceSNs:     deviceSNsJSON,
		Creator:       op,
		Message:       string(cmdJSON),
		RiskLevel:     overallRisk,
		ApprovalState: approvalState,
		TotalSteps:    1,
	}

	created, err := h.taskCreator.CreateTask(c.Request.Context(), task)
	if err != nil {
		h.logger.Error("enqueue rpc task failed",
			zap.String("action", req.Action),
			zap.Int("device_count", len(devices)),
			zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "enqueue task failed")
		return
	}

	// Audit log: input carries the original payload (action+params+devices),
	// target_id is the task id so audit can cross-link to executions.
	auditInput, _ := json.Marshal(req)
	h.auditSvc.Log(c.Request.Context(), &OpsAuditLog{
		OpType:       "command_rpc",
		TargetType:   "task",
		TargetID:     created.ID.String(),
		OperatorName: op,
		RiskLevel:    overallRisk,
		Result:       "enqueued",
		Input:        auditInput,
	})

	// SSE notification on the per-task channel so any active subscriber
	// of /commands/:id/stream gets immediate confirmation.
	enqueuedEvt := SSEEvent{
		Event: "command.enqueued",
		Data: mustJSON(map[string]interface{}{
			"task_id":           created.ID,
			"action":            req.Action,
			"devices":           devices,
			"status":            created.Status,
			"approval_required": approvalState == ApprovalPending,
		}),
	}
	h.sseHub.Publish("command:"+created.ID.String(), enqueuedEvt)

	// Auto-dispatch when no approval gating. fire-and-forget; the executor
	// owns task lifecycle from here. T-0102-c will replace the executor's
	// MVP stub with real RPC dispatch; this endpoint's contract does not
	// change either way.
	if approvalState == ApprovalNotRequired {
		go h.runRPCAsync(created.ID)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":           created.ID,
		"status":            created.Status,
		"risk_level":        overallRisk,
		"approval_required": approvalState == ApprovalPending,
		"stream_url":        fmt.Sprintf("/api/v1/ops/commands/%s/stream", created.ID),
	})
}

// runRPCAsync is the fire-and-forget dispatch invoked when an RPC task
// is not gated by approval. A fresh context.Background() is used because
// the HTTP request's context is cancelled when the 202 response returns.
// Errors are logged; the task's status field carries the true outcome.
func (h *ExtHandler) runRPCAsync(taskID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := h.executor.Run(ctx, taskID); err != nil {
		h.logger.Error("rpc executor run failed",
			zap.String("task_id", taskID.String()),
			zap.Error(err))
	}
}

// buildRPCTaskName composes a human-readable task name from the action
// and target devices. Single device shows the SN; multi-device shows a
// count to keep the name within the 200-char column budget.
func buildRPCTaskName(action string, devices []string) string {
	if len(devices) == 1 {
		return fmt.Sprintf("rpc:%s on %s", action, devices[0])
	}
	return fmt.Sprintf("rpc:%s on %d devices", action, len(devices))
}

// mustJSON marshals v and silently returns "null" on error. SSE payloads
// are best-effort observability; a marshal failure should not crash the
// publisher goroutine.
func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

func (h *ExtHandler) CommandStreamSSE(c *gin.Context) {
	id := c.Param("id")
	streamSSE(c, h.sseHub, "command:"+id)
}

// ============================================================
// 紧急响应
// ============================================================

func (h *ExtHandler) ActivateBreakGlass(c *gin.Context) {
	var req BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	userID := currentUserID(c)
	if userID == nil {
		response.Fail(c, http.StatusForbidden, "user not identified")
		return
	}
	if err := h.breakGlassSvc.Activate(c.Request.Context(), *userID, req); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "activated", "expires_in_seconds": 1800})
}

func (h *ExtHandler) DeactivateBreakGlass(c *gin.Context) {
	userID := currentUserID(c)
	if userID == nil {
		response.Fail(c, http.StatusForbidden, "user not identified")
		return
	}
	h.breakGlassSvc.Deactivate(c.Request.Context(), *userID)
	c.JSON(http.StatusOK, gin.H{"status": "deactivated"})
}

func (h *ExtHandler) BreakGlassStatus(c *gin.Context) {
	userID := currentUserID(c)
	if userID == nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": h.breakGlassSvc.IsActive(*userID)})
}

// ============================================================
// helpers
// ============================================================

func parseListRequest(c *gin.Context) model.ListRequest {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return model.ListRequest{Page: page, PageSize: pageSize}
}

func currentOperator(c *gin.Context) string {
	if name, ok := c.Get("user_name"); ok {
		if s, ok := name.(string); ok {
			return s
		}
	}
	if uid := currentUserID(c); uid != nil {
		return uid.String()
	}
	return "system"
}

func currentUserID(c *gin.Context) *uuid.UUID {
	if id, ok := admin.UserIDFromCtx(c); ok {
		return &id
	}
	return nil
}

func classifyRiskForAction(action string) RiskLevel {
	switch strings.ToLower(action) {
	case "reboot", "set_param", "set_parameters":
		return RiskCautious
	case "factory_reset", "factoryreset":
		return RiskDangerous
	default:
		return RiskSafe
	}
}

// streamSSE 标准 SSE 推送处理（保持长连接 + 心跳）。
func streamSSE(c *gin.Context, hub *SSEHub, channel string) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // nginx: disable buffering

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, "streaming not supported")
		return
	}

	events, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	// 初始事件标记连接建立
	writeSSE(c.Writer, "open", []byte(`{"channel":"`+channel+`"}`))
	flusher.Flush()

	ctx := c.Request.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			writeSSE(c.Writer, evt.Event, evt.Data)
			flusher.Flush()
		case <-ticker.C:
			// 心跳，防 nginx idle timeout
			writeSSE(c.Writer, "ping", []byte(`{}`))
			flusher.Flush()
		}
	}
}

func writeSSE(w io.Writer, event string, data []byte) {
	if event != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", event)
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}

// 防 unused import 警告
var _ = context.Background
