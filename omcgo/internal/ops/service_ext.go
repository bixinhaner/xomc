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

// Approve 4 眼审批：approver_user_id 不能 == 任务 creator。
func (s *ApprovalService) Approve(ctx context.Context, taskID, approverID uuid.UUID, approve bool, reason string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task.Creator == approverID.String() {
		return ErrSelfApprovalForbidden
	}
	// MVP: 状态机简化 — 任务字段在主表，扩展列已通过 model 层 patch
	// 实际工作中 ops_tasks 的 approval_state 字段需要专门 Update（待集成到 main TaskRepository）。
	// 这里仅写审计，不持久化 approval_state（留待 T-0106-b 二期完善）。
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
	s.logger.Info("task approved",
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

// TaskExecutor 调度引擎（T-0101，MVP stub）。
//
// MVP 行为：
//   - Run：把任务状态从 pending → running，写一条"准备执行"的 ops_task_executions
//   - Cancel / Pause / Resume：直接切状态字段
//   - 实际 RPC dispatching / 步骤路由 / 结果聚合 / 回滚 留待 T-0101-b..i 二期实现
type TaskExecutor struct {
	taskRepo TaskRepository
	execRepo TaskExecutionRepository
	auditSvc *AuditLogService
	logger   *zap.Logger
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
