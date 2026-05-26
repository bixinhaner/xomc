package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Service 是 MR 任务管理的业务用例集合。HTTP handler 与 scheduler 共享同一
// Service 接口，便于把"创建/查询/停止"的业务规则集中维护。
type Service interface {
	// Create 创建任务（含默认值填充 + 入参合法性校验 + 事务写入）。
	// input.Creator 由调用方（handler）从 auth ctx 提前填好。
	Create(ctx context.Context, input CreateTaskInput) (*Task, error)

	// Get 按 ID 查任务详情。
	Get(ctx context.Context, taskID uuid.UUID) (*Task, error)

	// List 列表查询。
	List(ctx context.Context, filter TaskListFilter) (*model.ListResponse[Task], error)

	// Stop 手动停止任务。
	//   - waitting → termination（无需下发，直接置态）
	//   - on       → termination（调用方 scheduler 后续会发关闭 SPV，
	//                 关闭完成后由 scheduler 把状态推进到 off）
	//   - 其它状态 → 返回错误
	Stop(ctx context.Context, taskID uuid.UUID) error

	// Delete 物理删除（仅 off / termination 允许）。
	Delete(ctx context.Context, taskID uuid.UUID) error

	// ListProgress 查询某任务的 cell 进度。
	ListProgress(ctx context.Context, filter ProgressListFilter) (*model.ListResponse[Progress], error)
}

// ErrInvalidInput 表示入参校验失败，handler 应映射到 400。
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

// NewValidationError 创建一个业务校验错误。handler 用 errors.As 判断后映射到 400。
func NewValidationError(format string, a ...interface{}) error {
	return &validationError{msg: fmt.Sprintf(format, a...)}
}

// IsValidationError 判断是否为入参校验错误。
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*validationError)
	return ok
}

// ErrInvalidStateTransition 表示状态机不允许的转换（如对 off 任务再次 stop）。
type stateError struct{ msg string }

func (e *stateError) Error() string { return e.msg }

// NewStateError 创建状态转换错误，handler 应映射到 409 Conflict。
func NewStateError(format string, a ...interface{}) error {
	return &stateError{msg: fmt.Sprintf(format, a...)}
}

// IsStateError 判断是否为状态转换错误。
func IsStateError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*stateError)
	return ok
}

// ---------- 实现 ----------

type service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService 创建 Service 实例。logger 允许 nil（使用 NopLogger）。
func NewService(repo Repository, logger *zap.Logger) Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{repo: repo, logger: logger}
}

func (s *service) Create(ctx context.Context, input CreateTaskInput) (*Task, error) {
	// 1) 默认值填充（先填默认再校验，让用户省略可选字段）
	if input.MRType == "" {
		input.MRType = "MRS,MRE,MRO"
	}
	if input.StatisPeriod == "" {
		input.StatisPeriod = "5120"
	}
	if input.ReportPeriod == "" {
		input.ReportPeriod = "15"
	}

	// 2) 校验
	if strings.TrimSpace(input.TaskName) == "" {
		return nil, NewValidationError("task_name is required")
	}
	// creator 由 handler 从 auth ctx 注入；缺失退化为 "system"
	if input.Creator == "" {
		input.Creator = "system"
	}
	// 目标设备 SN 列表必填（去重 + 过滤空）
	dedup := make(map[string]struct{}, len(input.TargetDeviceSNs))
	cleaned := make([]string, 0, len(input.TargetDeviceSNs))
	for _, sn := range input.TargetDeviceSNs {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		if _, ok := dedup[sn]; ok {
			continue
		}
		dedup[sn] = struct{}{}
		cleaned = append(cleaned, sn)
	}
	if len(cleaned) == 0 {
		return nil, NewValidationError("target_device_sns is required (at least one device)")
	}
	input.TargetDeviceSNs = cleaned
	// 文档 §3.1：MRS/MRE/MRO 三项强制
	if !hasAllRequiredMRTypes(input.MRType) {
		return nil, NewValidationError("mr_type must contain MRS, MRE and MRO")
	}
	if !IsValidStatisPeriod(input.StatisPeriod) {
		return nil, NewValidationError("invalid statis_period: %s", input.StatisPeriod)
	}
	if _, ok := UploadPeriodSeconds(input.ReportPeriod); !ok {
		return nil, NewValidationError("invalid report_period: %s (must be 15/30/60)", input.ReportPeriod)
	}
	if input.EndTime != nil && !input.EndTime.After(input.StartTime) {
		return nil, NewValidationError("end_time must be after start_time")
	}

	// 3) 构造领域对象。task_id 服务端生成。
	// targets 不直接走入参；scheduler 开启时按 target_device_sns 从
	// mr_device_mappings 展开为 cells（2026-05-26）。
	task := &Task{
		TaskID:          uuid.New(),
		TaskName:        input.TaskName,
		MRType:          input.MRType,
		StatisPeriod:    input.StatisPeriod,
		ReportPeriod:    input.ReportPeriod,
		StartTime:       input.StartTime.UTC(),
		TaskStatus:      StatusWaiting,
		Creator:         input.Creator,
		TargetDeviceSNs: input.TargetDeviceSNs,
	}
	if input.EndTime != nil {
		end := input.EndTime.UTC()
		task.EndTime = &end
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create mr task: %w", err)
	}

	s.logger.Info("mr task created",
		zap.String("task_id", task.TaskID.String()),
		zap.String("creator", task.Creator),
		zap.Time("start_time", task.StartTime),
	)
	return task, nil
}

func (s *service) Get(ctx context.Context, taskID uuid.UUID) (*Task, error) {
	t, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *service) List(ctx context.Context, filter TaskListFilter) (*model.ListResponse[Task], error) {
	resp, err := s.repo.ListTasks(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list mr tasks: %w", err)
	}
	return resp, nil
}

func (s *service) Stop(ctx context.Context, taskID uuid.UUID) error {
	t, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.TaskStatus {
	case StatusWaiting:
		// 未下发的任务，直接 termination；scheduler 后续会跳过
		if err := s.repo.UpdateTaskStatus(ctx, taskID, StatusTermination, nil); err != nil {
			return fmt.Errorf("update mr task status to termination: %w", err)
		}
	case StatusOn:
		// 已下发的任务，置为 termination；下一个 scheduler tick 会触发关闭 SPV
		if err := s.repo.UpdateTaskStatus(ctx, taskID, StatusTermination, nil); err != nil {
			return fmt.Errorf("update mr task status to termination: %w", err)
		}
	default:
		return NewStateError("cannot stop task in status %s", t.TaskStatus)
	}
	s.logger.Info("mr task stopped by user",
		zap.String("task_id", taskID.String()),
		zap.String("from_status", string(t.TaskStatus)),
	)
	return nil
}

func (s *service) Delete(ctx context.Context, taskID uuid.UUID) error {
	t, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	if t.TaskStatus != StatusOff && t.TaskStatus != StatusTermination {
		return NewStateError("cannot delete task in status %s (only off/termination allowed)", t.TaskStatus)
	}
	if err := s.repo.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("delete mr task: %w", err)
	}
	s.logger.Info("mr task deleted",
		zap.String("task_id", taskID.String()),
	)
	return nil
}

func (s *service) ListProgress(ctx context.Context, filter ProgressListFilter) (*model.ListResponse[Progress], error) {
	resp, err := s.repo.ListProgress(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list mr progress: %w", err)
	}
	return resp, nil
}

// ---------- 辅助 ----------

// hasAllRequiredMRTypes 检查字符串里是否含 MRS / MRE / MRO 三项（顺序不限、可含 MDT 等）。
func hasAllRequiredMRTypes(s string) bool {
	required := []string{"MRS", "MRE", "MRO"}
	parts := strings.Split(s, ",")
	set := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		set[strings.ToUpper(strings.TrimSpace(p))] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[r]; !ok {
			return false
		}
	}
	return true
}
