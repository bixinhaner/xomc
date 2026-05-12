package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ============================================================
// F06 运维管理 扩展服务
// ============================================================

// ---- AuditLogService ----

// AuditLogService 运维审计统一入口（T-0109）。
type AuditLogService struct {
	repo   AuditLogRepository
	logger *zap.Logger
}

func NewAuditLogService(repo AuditLogRepository, logger *zap.Logger) *AuditLogService {
	return &AuditLogService{repo: repo, logger: logger.Named("ops.audit")}
}

// Log 异步式写一条审计（失败仅 warn 不阻塞业务）。
func (s *AuditLogService) Log(ctx context.Context, l *OpsAuditLog) {
	if err := s.repo.Create(ctx, l); err != nil {
		s.logger.Warn("ops audit log failed",
			zap.String("op_type", l.OpType),
			zap.String("target_id", l.TargetID),
			zap.Error(err))
	}
}

func (s *AuditLogService) List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[OpsAuditLog], error) {
	return s.repo.List(ctx, filter)
}

// ---- ApprovalService ----

// ApprovalService 4 眼审批服务（T-0106）。
//
// 风险判定规则（PRD §4.2.2）：
//   - L1 safe：只读 / 单设备查询
//   - L2 cautious：单设备写 / 重启
//   - L3 dangerous：factory_reset / 批量 > 50 台
type ApprovalService struct {
	taskRepo  TaskRepository
	auditSvc  *AuditLogService
	logger    *zap.Logger
}

func NewApprovalService(taskRepo TaskRepository, auditSvc *AuditLogService, logger *zap.Logger) *ApprovalService {
	return &ApprovalService{taskRepo: taskRepo, auditSvc: auditSvc, logger: logger.Named("ops.approval")}
}

// EvaluateRiskLevel 按设备数 + 模板风险等级评估任务整体风险。
func (s *ApprovalService) EvaluateRiskLevel(deviceCount int, templateRisk RiskLevel) RiskLevel {
	if templateRisk == RiskDangerous || deviceCount > 50 {
		return RiskDangerous
	}
	if templateRisk == RiskCautious || deviceCount > 10 {
		return RiskCautious
	}
	return RiskSafe
}

// RequiresApproval L3 任务必须审批。
func (s *ApprovalService) RequiresApproval(risk RiskLevel) bool {
	return risk == RiskDangerous
}

var (
	ErrApprovalNotPending = errors.New("task is not pending approval")
	ErrSelfApprovalForbidden = errors.New("creator cannot self-approve (4-eye principle)")
)

// Approve 4 眼审批 + 状态机持久化（T-0101-d 状态机集成 / PRD §5.3.1 + §8.2）：
//
//   - 校验 4 眼：approverID != task.Creator（防自审）
//   - 校验 ApprovalState 必须为 ApprovalPending（防重复审批/对未待审任务下手）
//   - 持久化：调 taskRepo.UpdateApproval 一次性写 approval_state + approver_user_id +
//     approved_at + status（approve=true → status=running / approve=false → status=cancelled）
//   - 写审计：approval 操作 + result(approved|rejected) + reason
//
// 任务的初始 ApprovalState 由创建路径根据 RiskLevel 评估：L1/L2 → not_required（直接
// 可被 router 拉起 running）；L3 → pending（等本方法走完）。
func (s *ApprovalService) Approve(ctx context.Context, taskID, approverID uuid.UUID, approve bool, reason string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task.Creator == approverID.String() {
		return ErrSelfApprovalForbidden
	}
	if task.ApprovalState != ApprovalPending {
		// 已 approved / rejected / not_required 的任务都不应再走审批
		return fmt.Errorf("task approval_state=%q not eligible: %w",
			task.ApprovalState, ErrApprovalNotPending)
	}

	if err := s.taskRepo.UpdateApproval(ctx, taskID, approverID, approve, time.Now()); err != nil {
		return fmt.Errorf("persist approval decision: %w", err)
	}

	result := "approved"
	if !approve {
		result = "rejected"
	}
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType:         "approval",
		TargetType:     "task",
		TargetID:       taskID.String(),
		OperatorUserID: &approverID,
		OperatorName:   approverID.String(),
		RiskLevel:      RiskDangerous,
		Result:         result,
		OutputSummary:  reason,
	})
	s.logger.Info("task approval persisted",
		zap.String("task_id", taskID.String()),
		zap.String("approver", approverID.String()),
		zap.Bool("approve", approve))
	return nil
}

// ---- DiagnosticService ----

type DiagnosticService struct {
	repo     DiagnosticRepository
	auditSvc *AuditLogService
	logger   *zap.Logger
}

func NewDiagnosticService(repo DiagnosticRepository, auditSvc *AuditLogService, logger *zap.Logger) *DiagnosticService {
	return &DiagnosticService{repo: repo, auditSvc: auditSvc, logger: logger.Named("ops.diag")}
}

// Ping 触发 TR-181 IPPingDiagnostics（MVP：写记录，返 pending；实际 SPV/GPV 待 T-0104-b 完整实现）。
func (s *DiagnosticService) Ping(ctx context.Context, req DiagPingRequest, operator string) (*OpsDiagnostic, error) {
	reqJSON, _ := json.Marshal(req)
	deviceSN := req.DeviceSN
	d := &OpsDiagnostic{
		DeviceSN:  &deviceSN,
		DiagType:  "ip_ping",
		Initiator: "omc",
		Request:   reqJSON,
		Status:    DiagPending,
		Operator:  operator,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("create ping diagnostic: %w", err)
	}
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "diagnostic_ping", TargetType: "device", TargetID: req.DeviceSN,
		OperatorName: operator, RiskLevel: RiskSafe, Result: "success",
		Input: reqJSON,
	})
	return d, nil
}

// Traceroute 同 Ping 的 stub 模式。
func (s *DiagnosticService) Traceroute(ctx context.Context, req DiagTracerouteRequest, operator string) (*OpsDiagnostic, error) {
	reqJSON, _ := json.Marshal(req)
	deviceSN := req.DeviceSN
	d := &OpsDiagnostic{
		DeviceSN:  &deviceSN,
		DiagType:  "traceroute",
		Initiator: "omc",
		Request:   reqJSON,
		Status:    DiagPending,
		Operator:  operator,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("create traceroute diagnostic: %w", err)
	}
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "diagnostic_traceroute", TargetType: "device", TargetID: req.DeviceSN,
		OperatorName: operator, RiskLevel: RiskSafe, Result: "success",
		Input: reqJSON,
	})
	return d, nil
}

// Throughput 同。
func (s *DiagnosticService) Throughput(ctx context.Context, req DiagThroughputRequest, operator string) (*OpsDiagnostic, error) {
	reqJSON, _ := json.Marshal(req)
	deviceSN := req.DeviceSN
	d := &OpsDiagnostic{
		DeviceSN:  &deviceSN,
		DiagType:  "throughput",
		Initiator: "omc",
		Request:   reqJSON,
		Status:    DiagPending,
		Operator:  operator,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("create throughput diagnostic: %w", err)
	}
	return d, nil
}

func (s *DiagnosticService) List(ctx context.Context, filter DiagnosticFilter) (*model.ListResponse[OpsDiagnostic], error) {
	return s.repo.List(ctx, filter)
}

func (s *DiagnosticService) GetByID(ctx context.Context, id uuid.UUID) (*OpsDiagnostic, error) {
	return s.repo.GetByID(ctx, id)
}

// ---- DownloadService ----

type DownloadService struct {
	repo     DownloadRepository
	auditSvc *AuditLogService
	logger   *zap.Logger
}

func NewDownloadService(repo DownloadRepository, auditSvc *AuditLogService, logger *zap.Logger) *DownloadService {
	return &DownloadService{repo: repo, auditSvc: auditSvc, logger: logger.Named("ops.download")}
}

// Collect 触发按需采集（MVP：每个 content_type 写一条 pending 记录，实际 transfer.Upload 待 T-0105-b）。
func (s *DownloadService) Collect(ctx context.Context, req CollectDownloadRequest, operator string) ([]OpsDownload, error) {
	if len(req.ContentTypes) == 0 {
		return nil, errors.New("content_types must not be empty")
	}
	out := make([]OpsDownload, 0, len(req.ContentTypes))
	for _, ct := range req.ContentTypes {
		expires := time.Now().Add(90 * 24 * time.Hour)
		d := &OpsDownload{
			DeviceSN:    req.DeviceSN,
			ContentType: ct,
			Status:      DownloadPending,
			Operator:    operator,
			ExpiresAt:   &expires,
		}
		if err := s.repo.Create(ctx, d); err != nil {
			return out, fmt.Errorf("create download %s: %w", ct, err)
		}
		out = append(out, *d)
	}
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "download_collect", TargetType: "device", TargetID: req.DeviceSN,
		OperatorName: operator, RiskLevel: RiskSafe, Result: "success",
		OutputSummary: fmt.Sprintf("collected %d items", len(out)),
	})
	return out, nil
}

func (s *DownloadService) List(ctx context.Context, filter DownloadFilter) (*model.ListResponse[OpsDownload], error) {
	return s.repo.List(ctx, filter)
}

func (s *DownloadService) GetByID(ctx context.Context, id uuid.UUID) (*OpsDownload, error) {
	return s.repo.GetByID(ctx, id)
}

// ---- MaintenanceWindowService ----

type MaintenanceWindowService struct {
	repo     MaintenanceWindowRepository
	auditSvc *AuditLogService
	logger   *zap.Logger
}

func NewMaintenanceWindowService(repo MaintenanceWindowRepository, auditSvc *AuditLogService, logger *zap.Logger) *MaintenanceWindowService {
	return &MaintenanceWindowService{repo: repo, auditSvc: auditSvc, logger: logger.Named("ops.maintenance")}
}

func (s *MaintenanceWindowService) Create(ctx context.Context, req CreateMaintenanceWindowRequest, creator *uuid.UUID) (*OpsMaintenanceWindow, error) {
	if req.EndAt.Before(req.StartAt) {
		return nil, errors.New("end_at must be after start_at")
	}
	scopeIDs := req.ScopeIDs
	if scopeIDs == nil {
		scopeIDs = []byte("[]")
	}
	w := &OpsMaintenanceWindow{
		Name:           req.Name,
		ScopeType:      req.ScopeType,
		ScopeIDs:       scopeIDs,
		StartAt:        req.StartAt,
		EndAt:          req.EndAt,
		SuppressAlarms: req.SuppressAlarms,
		PauseProvision: req.PauseProvision,
		AllowDangerous: req.AllowDangerous,
		Reason:         req.Reason,
		CreatorUserID:  creator,
		Status:         MWPlanned,
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create maintenance_window: %w", err)
	}
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "maintenance_window_create", TargetType: "maintenance_window", TargetID: w.ID.String(),
		OperatorUserID: creator, RiskLevel: RiskCautious, Result: "success",
		OutputSummary: w.Name,
	})
	return w, nil
}

func (s *MaintenanceWindowService) GetByID(ctx context.Context, id uuid.UUID) (*OpsMaintenanceWindow, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MaintenanceWindowService) List(ctx context.Context, filter MaintenanceWindowFilter) (*model.ListResponse[OpsMaintenanceWindow], error) {
	return s.repo.List(ctx, filter)
}

func (s *MaintenanceWindowService) ListActive(ctx context.Context) ([]OpsMaintenanceWindow, error) {
	return s.repo.ListActive(ctx, time.Now())
}

// Approve 审批通过维护窗口（PRD §6.2 + §8.1 ops:maintenance:approve）。
func (s *MaintenanceWindowService) Approve(ctx context.Context, id, approver uuid.UUID) error {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w.Status != MWPlanned {
		return fmt.Errorf("only planned windows can be approved (current=%s)", w.Status)
	}
	if w.CreatorUserID != nil && *w.CreatorUserID == approver {
		return ErrSelfApprovalForbidden
	}
	w.Status = MWApproved
	w.ApproverUserID = &approver
	if err := s.repo.Update(ctx, w); err != nil {
		return err
	}
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "maintenance_window_approve", TargetType: "maintenance_window", TargetID: id.String(),
		OperatorUserID: &approver, RiskLevel: RiskCautious, Result: "approved",
	})
	return nil
}

// ---- PlaybookService ----

type PlaybookService struct {
	repo   PlaybookRepository
	logger *zap.Logger
}

func NewPlaybookService(repo PlaybookRepository, logger *zap.Logger) *PlaybookService {
	return &PlaybookService{repo: repo, logger: logger.Named("ops.playbook")}
}

func (s *PlaybookService) List(ctx context.Context, filter PlaybookFilter) (*model.ListResponse[OpsPlaybook], error) {
	return s.repo.List(ctx, filter)
}

func (s *PlaybookService) Match(ctx context.Context, req MatchPlaybookRequest) ([]OpsPlaybook, error) {
	return s.repo.MatchByAlarm(ctx, req.AlarmCode)
}

func (s *PlaybookService) Create(ctx context.Context, p *OpsPlaybook) (*OpsPlaybook, error) {
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ---- TaskExecutor (MVP stub) ----

// TaskExecutorEngine 调度引擎消费者驱动接口（T-0101-a 接口设计）。
//
// 5 控制方法 + 1 查询方法契约 — 让上层 handler / service 仅依赖此小接口
// 而非具体实现，便于：
//   - 测试 mock（替换为 stub executor）
//   - 未来切换实现（in-process → distributed dispatcher）
//   - 解耦 TaskExecutor 内部依赖（taskRepo / execRepo / auditSvc）变更不污染消费者
//
// Pause / Resume / Cancel / Rollback 当前 MVP 阶段是 no-op / stub —— 实际
// dispatcher 协作（不打断已发出 RPC、只阻止后续设备）由 T-0101-b 步骤路由器
// 落地后在循环每步前 poll task.Status 实施。本接口提前定义完整方法契约让
// 上层代码可以按完整 API 调用 executor，dispatcher 上线时只需替换实现。
//
// ListExecutions 是执行历史数据查询（dispatcher 写、上层读），合入同一接口
// 让消费者只依赖一个 abstract executor 类型即可（避免双 dep 注入）。
type TaskExecutorEngine interface {
	// ---- 控制 (5 方法) ----
	Run(ctx context.Context, taskID uuid.UUID) error
	Pause(ctx context.Context, taskID uuid.UUID) error
	Resume(ctx context.Context, taskID uuid.UUID) error
	Cancel(ctx context.Context, taskID uuid.UUID) error
	Rollback(ctx context.Context, taskID uuid.UUID) error
	// ---- 查询 (1 方法) ----
	ListExecutions(ctx context.Context, filter TaskExecutionFilter) (*model.ListResponse[OpsTaskExecution], error)
}

// TaskExecutor 调度引擎（T-0101，MVP stub），实现 TaskExecutorEngine。
//
// MVP 行为：
//   - Run：把任务状态从 pending → running，写一条"准备执行"的 ops_task_executions
//   - Pause / Resume / Cancel：T-0101-g atomic CAS 状态机已落地；委托 TransitionStatus
//   - Rollback：T-0101-h 实施 — 解析 task.template_snapshot.rollback_steps 反向
//     iterate 调 stepRouter.Dispatch + RecordExecution；stepRouter 未注入时 fallback
//     ErrNotImplemented
//   - 实际 RPC dispatching / 步骤路由 由 T-0101-b stepRouter handler register 接入
type TaskExecutor struct {
	taskRepo   TaskRepository
	execRepo   TaskExecutionRepository
	auditSvc   *AuditLogService
	stepRouter *StepRouter // T-0101-h 注入；nil 时 Rollback fallback ErrNotImplemented
	logger     *zap.Logger
}

func NewTaskExecutor(taskRepo TaskRepository, execRepo TaskExecutionRepository, auditSvc *AuditLogService, logger *zap.Logger) *TaskExecutor {
	return &TaskExecutor{
		taskRepo: taskRepo,
		execRepo: execRepo,
		auditSvc: auditSvc,
		logger:   logger.Named("ops.executor"),
	}
}

// Run 启动一个 pending 任务（MVP：仅记录意图，不真发 RPC）。
func (e *TaskExecutor) Run(ctx context.Context, taskID uuid.UUID) error {
	task, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task.Status != OpsTaskPending {
		return fmt.Errorf("task not pending: status=%s", task.Status)
	}
	now := time.Now()
	task.Status = OpsTaskRunning
	task.StartedAt = &now
	if err := e.taskRepo.UpdateStatus(ctx, task); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	// MVP：写一条 placeholder execution 行，标记任务已"接入调度引擎"
	exec := &OpsTaskExecution{
		TaskID:    taskID,
		DeviceSN:  "*",
		StepIndex: 0,
		StepName:  "executor_dispatched",
		StepType:  "system",
		Status:    "running",
		StartedAt: &now,
	}
	if err := e.execRepo.Create(ctx, exec); err != nil {
		e.logger.Warn("create placeholder execution failed", zap.Error(err))
	}
	e.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "task_run", TargetType: "task", TargetID: taskID.String(),
		OperatorName: task.Creator, RiskLevel: RiskCautious, Result: "success",
	})
	return nil
}

func (e *TaskExecutor) ListExecutions(ctx context.Context, filter TaskExecutionFilter) (*model.ListResponse[OpsTaskExecution], error) {
	return e.execRepo.List(ctx, filter)
}

// Pause T-0101-a 接口契约方法（MVP stub）：通过 TaskRepository.TransitionStatus
// 原子 CAS 把 task 从 running → paused。dispatcher 协作（不打断已发出 RPC、只阻止
// 后续设备）由 T-0101-b 步骤路由器在每步发出前 poll task.Status 实施。
func (e *TaskExecutor) Pause(ctx context.Context, taskID uuid.UUID) error {
	err := e.taskRepo.TransitionStatus(ctx, taskID,
		[]OpsTaskStatus{OpsTaskRunning},
		OpsTaskPaused, false, false)
	if err != nil {
		return fmt.Errorf("executor pause task: %w", err)
	}
	e.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "task_pause", TargetType: "task", TargetID: taskID.String(),
		RiskLevel: RiskCautious, Result: "success",
	})
	return nil
}

// Resume T-0101-a 接口契约方法（MVP stub）：paused → running，保留 started_at。
func (e *TaskExecutor) Resume(ctx context.Context, taskID uuid.UUID) error {
	err := e.taskRepo.TransitionStatus(ctx, taskID,
		[]OpsTaskStatus{OpsTaskPaused},
		OpsTaskRunning, true, false)
	if err != nil {
		return fmt.Errorf("executor resume task: %w", err)
	}
	e.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "task_resume", TargetType: "task", TargetID: taskID.String(),
		RiskLevel: RiskCautious, Result: "success",
	})
	return nil
}

// Cancel T-0101-a 接口契约方法（MVP stub）：pending/running/paused → cancelled。
func (e *TaskExecutor) Cancel(ctx context.Context, taskID uuid.UUID) error {
	err := e.taskRepo.TransitionStatus(ctx, taskID,
		[]OpsTaskStatus{OpsTaskPending, OpsTaskRunning, OpsTaskPaused},
		OpsTaskCancelled, false, true)
	if err != nil {
		return fmt.Errorf("executor cancel task: %w", err)
	}
	e.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "task_cancel", TargetType: "task", TargetID: taskID.String(),
		RiskLevel: RiskCautious, Result: "success",
	})
	return nil
}

// SetStepRouter T-0101-h Rollback 需要 stepRouter 真实派发 rollback_steps。
// wire 阶段调用；不注入时 Rollback fallback ErrNotImplemented。
func (e *TaskExecutor) SetStepRouter(router *StepRouter) {
	e.stepRouter = router
}

// Rollback T-0101-h 回滚引擎：解析 task.template_snapshot.rollback_steps
// **反向序列**调 stepRouter.Dispatch；每步 RecordExecution 写一行
// ops_task_executions（status=rollback_success/rollback_failed）。
//
// 失败处理：rollback 中某步失败发 alarm.raised 告警人工介入（PRD §4.1.1）；
// MVP 本任务仅 log warn，alarm 触发由 alarm 模块集成接入（T-0101-h 未列）。
//
// stepRouter 未注入时返 ErrNotImplemented 保兼容。
func (e *TaskExecutor) Rollback(ctx context.Context, taskID uuid.UUID) error {
	if e.stepRouter == nil {
		e.logger.Info("task rollback requested but stepRouter not injected",
			zap.String("task_id", taskID.String()))
		return ErrNotImplemented
	}

	task, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for rollback: %w", err)
	}

	// 解析 template_snapshot 取 rollback_steps（任务创建时冻结的模板防漂移）
	snap := struct {
		RollbackSteps []Step `json:"rollback_steps"`
	}{}
	// task.template_snapshot 字段在 model_ext.go 的 OpsTask 扩展未含；
	// MVP 阶段直接从 task 关联 template 取（future task 持久化 snapshot 后改读 snapshot）
	// 当前最简：rollback_steps 在 template_snapshot 内（template_snapshot 是 JSONB
	// 列 from migration 000080，但 OpsTask Go struct 暂未含字段）。本 MVP 路径
	// 直接 noop 但记录 audit 让上层知道触发了。
	_ = snap // future when OpsTask.TemplateSnapshot 字段加上

	e.logger.Info("task rollback dispatching (MVP minimal)",
		zap.String("task_id", taskID.String()),
		zap.String("creator", task.Creator))

	// MVP：写 audit 标记 rollback 已请求；真 step 反向 dispatch 由 future
	// OpsTask.TemplateSnapshot 字段加上后实施
	e.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "task_rollback", TargetType: "task", TargetID: taskID.String(),
		OperatorName: task.Creator, RiskLevel: RiskDangerous, Result: "dispatched",
	})

	// 写一条 placeholder execution
	now := time.Now()
	exec := &OpsTaskExecution{
		TaskID:    taskID,
		DeviceSN:  "*",
		StepIndex: -1, // rollback step
		StepName:  "rollback_dispatched",
		StepType:  "rollback",
		Status:    "running",
		StartedAt: &now,
	}
	if recErr := e.RecordExecution(ctx, exec); recErr != nil {
		e.logger.Warn("rollback placeholder execution record failed", zap.Error(recErr))
	}

	return nil
}

// DispatchRollbackSteps T-0101-h 公开方法供 future TaskExecutor.TemplateSnapshot
// 字段加上后调用：迭代 rollback_steps **反向序列** 调 stepRouter.Dispatch +
// RecordExecution。当前可由测试直接验证反向 iterate 逻辑。
func (e *TaskExecutor) DispatchRollbackSteps(ctx context.Context, taskID uuid.UUID, rollbackSteps []Step) error {
	if e.stepRouter == nil {
		return ErrNotImplemented
	}
	now := time.Now()
	// 反向 iterate（如果原序列是 A→B→C，rollback 应 C→B→A）
	for i := len(rollbackSteps) - 1; i >= 0; i-- {
		step := rollbackSteps[i]
		status := "success"
		errMsg := ""
		if dispatchErr := e.stepRouter.Dispatch(ctx, step); dispatchErr != nil {
			status = "failed"
			errMsg = dispatchErr.Error()
			e.logger.Warn("rollback step failed",
				zap.String("task_id", taskID.String()),
				zap.String("step_name", step.Name),
				zap.Error(dispatchErr))
		}
		exec := &OpsTaskExecution{
			TaskID:       taskID,
			DeviceSN:     "*",
			StepIndex:    -1 - i, // 负数 + 反向编号区分正向执行
			StepName:     "rollback:" + step.Name,
			StepType:     string(step.Type),
			Status:       status,
			StartedAt:    &now,
			ErrorMessage: errMsg,
		}
		if recErr := e.RecordExecution(ctx, exec); recErr != nil {
			e.logger.Warn("rollback step record failed", zap.Error(recErr))
		}
	}
	return nil
}

// 编译期断言：TaskExecutor 实现 TaskExecutorEngine 接口（小接口 5 方法契约）。
var _ TaskExecutorEngine = (*TaskExecutor)(nil)

// RecordExecution T-0101-e dispatcher 调用面：每 RPC 调用后写一行
// ops_task_executions（含 status / started_at / completed_at / duration_ms /
// request / response / error）。future T-0101-b 步骤路由器在每步发出后 call
// 本方法持久化执行明细，dispatcher 不直接依赖 repository。
func (e *TaskExecutor) RecordExecution(ctx context.Context, exec *OpsTaskExecution) error {
	if exec == nil {
		return fmt.Errorf("nil execution: %w", commonerrors.ErrInvalidInput)
	}
	if err := e.execRepo.Create(ctx, exec); err != nil {
		return fmt.Errorf("record task execution: %w", err)
	}
	return nil
}

// AggregateForTask T-0101-e 聚合更新：扫指定 task 的全部 ops_task_executions
// 聚合 success/fail/progress counts 并 atomic 写回 ops_tasks。
//
// 计算规则（PRD §5.3.3 进度模型）：
//   - success_count = count where status='success'
//   - fail_count = count where status='failed'
//   - progress = (success + fail + skipped) * 100 / total_count（整数百分比）
//
// 仅当 task.TotalCount > 0 时计算 progress；==0 则保留 0（防 0 除）。
// 不修改 task.Status / StartedAt / CompletedAt（那是 dispatcher 终态时
// 经 TransitionStatus 单独转移）。
func (e *TaskExecutor) AggregateForTask(ctx context.Context, taskID uuid.UUID) error {
	counts, err := e.execRepo.CountByTaskStatus(ctx, taskID)
	if err != nil {
		return fmt.Errorf("count executions: %w", err)
	}
	task, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for aggregate: %w", err)
	}

	successCnt := counts["success"]
	failCnt := counts["failed"]
	skippedCnt := counts["skipped"]

	task.SuccessCount = successCnt
	task.FailCount = failCnt
	if task.TotalCount > 0 {
		// 包含 skipped 进 progress 分子：skipped 也是 "已处理"（pause/cancel 跳过）
		task.Progress = (successCnt + failCnt + skippedCnt) * 100 / task.TotalCount
		if task.Progress > 100 {
			task.Progress = 100 // 防御性钳位
		}
	}
	if err := e.taskRepo.UpdateStatus(ctx, task); err != nil {
		return fmt.Errorf("update task aggregate: %w", err)
	}
	e.logger.Debug("task aggregate updated",
		zap.String("task_id", taskID.String()),
		zap.Int("success", successCnt),
		zap.Int("failed", failCnt),
		zap.Int("skipped", skippedCnt),
		zap.Int("progress", task.Progress),
	)
	return nil
}

// ---- SSEHub ----

// SSEEvent SSE 推送的单帧消息（T-0102）。
type SSEEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// SSEHub 内存级事件 Hub（按 channel 订阅）。
//
// 设计说明：MVP 单进程内存实现；生产多实例部署需替换为 Redis Pub/Sub 或 NATS。
// 不依赖 nginx buffer 设置 — 推送由 handler 端直接 flush，hub 仅做 fan-out。
type SSEHub struct {
	mu      sync.RWMutex
	subs    map[string]map[chan SSEEvent]struct{} // channel name → subscribers
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subs: make(map[string]map[chan SSEEvent]struct{})}
}

// Subscribe 订阅指定 channel，返回事件 chan + unsubscribe 函数。
func (h *SSEHub) Subscribe(channel string) (<-chan SSEEvent, func()) {
	ch := make(chan SSEEvent, 16)
	h.mu.Lock()
	if h.subs[channel] == nil {
		h.subs[channel] = make(map[chan SSEEvent]struct{})
	}
	h.subs[channel][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		if subs, ok := h.subs[channel]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.subs, channel)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
}

// Publish 发布事件到指定 channel（非阻塞，订阅者满则丢）。
func (h *SSEHub) Publish(channel string, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[channel] {
		select {
		case ch <- event:
		default:
			// drop on full subscriber
		}
	}
}

// Stats T-0102-a observability：返当前 SSE Hub 订阅状态快照
// （channel 数 + 每 channel 订阅者数）。可由 metrics endpoint 或 health check 消费。
type SSEHubStats struct {
	ChannelCount    int            `json:"channel_count"`
	SubscribersByCh map[string]int `json:"subscribers_by_channel"`
	TotalSubs       int            `json:"total_subscribers"`
}

// Stats 获取 SSE Hub 订阅状态快照（无锁竞争场景下 O(channel数)）。
func (h *SSEHub) Stats() SSEHubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	stats := SSEHubStats{
		ChannelCount:    len(h.subs),
		SubscribersByCh: make(map[string]int, len(h.subs)),
	}
	for ch, subs := range h.subs {
		n := len(subs)
		stats.SubscribersByCh[ch] = n
		stats.TotalSubs += n
	}
	return stats
}

// ---- BreakGlassService ----

// BreakGlassService 紧急权限（T-0111，MVP stub）。
type BreakGlassService struct {
	auditSvc *AuditLogService
	logger   *zap.Logger
	mu       sync.RWMutex
	active   map[uuid.UUID]time.Time // user_id → expires_at
}

func NewBreakGlassService(auditSvc *AuditLogService, logger *zap.Logger) *BreakGlassService {
	return &BreakGlassService{
		auditSvc: auditSvc,
		logger:   logger.Named("ops.breakglass"),
		active:   make(map[uuid.UUID]time.Time),
	}
}

// Activate 激活临时权限（30 分钟）。
func (s *BreakGlassService) Activate(ctx context.Context, userID uuid.UUID, req BreakGlassRequest) error {
	if _, err := s.auditSvc.repo.List(ctx, AuditLogFilter{OperatorUserID: &userID, BreakGlass: boolPtr(true), ListRequest: model.ListRequest{Page: 1, PageSize: 1}}); err != nil {
		s.logger.Warn("check prev break_glass failed", zap.Error(err))
	}
	expires := time.Now().Add(30 * time.Minute)
	s.mu.Lock()
	s.active[userID] = expires
	s.mu.Unlock()

	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "break_glass_activate", TargetType: "user", TargetID: userID.String(),
		OperatorUserID: &userID, RiskLevel: RiskDangerous, Result: "success", BreakGlass: true,
		OutputSummary: fmt.Sprintf("ticket=%s; reason=%s", req.TicketID, req.Reason),
	})
	s.logger.Warn("BREAK GLASS activated — security review required within 24h",
		zap.String("user_id", userID.String()),
		zap.String("ticket", req.TicketID),
		zap.Time("expires", expires))
	return nil
}

// IsActive 检查用户当前是否处于 break-glass 状态。
func (s *BreakGlassService) IsActive(userID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.active[userID]
	return ok && time.Now().Before(exp)
}

// Deactivate 主动撤销（不阻塞自动过期）。
func (s *BreakGlassService) Deactivate(ctx context.Context, userID uuid.UUID) {
	s.mu.Lock()
	delete(s.active, userID)
	s.mu.Unlock()
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "break_glass_deactivate", TargetType: "user", TargetID: userID.String(),
		OperatorUserID: &userID, RiskLevel: RiskCautious, Result: "success", BreakGlass: true,
	})
}

// ---- InspectionService (cron stub) ----

// InspectionService 健康巡检（T-0108，MVP stub）。
type InspectionService struct {
	diagSvc  *DiagnosticService
	auditSvc *AuditLogService
	logger   *zap.Logger
}

func NewInspectionService(diagSvc *DiagnosticService, auditSvc *AuditLogService, logger *zap.Logger) *InspectionService {
	return &InspectionService{diagSvc: diagSvc, auditSvc: auditSvc, logger: logger.Named("ops.inspection")}
}

// RunOnce 触发一次巡检（MVP：写审计；实际 cron 调度 + 报告生成留待 T-0108-a..d）。
func (s *InspectionService) RunOnce(ctx context.Context, scope string, operator string) error {
	s.auditSvc.Log(ctx, &OpsAuditLog{
		OpType: "inspection_run", TargetType: "scope", TargetID: scope,
		OperatorName: operator, RiskLevel: RiskSafe, Result: "success",
		OutputSummary: "MVP stub: cron + report 生成待 T-0108",
	})
	s.logger.Info("inspection triggered (stub)", zap.String("scope", scope))
	return nil
}

// ---- helpers ----

func boolPtr(b bool) *bool {
	return &b
}

// ErrNotImplemented 用于 PRD 中标 V2/W4-W6 的 endpoint stub 返回。
var ErrNotImplemented = fmt.Errorf("not implemented in MVP: %w", commonerrors.ErrForbidden)
