package topology

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// DeviceLister 设备列举器（消费侧 narrow 接口，单方法）。
//
// 用于 ApplyRule 异步 worker 中的 getAllDevices；nil 时 worker 走旧 stub
// 行为返空切片（向后兼容，便于 Day 3 灰度推进）。
//
// 实施位于 PgDeviceLister（同包，pg_device_lister.go）。
type DeviceLister interface {
	// ListAllForRuleEval 返回所有可被规则匹配的设备的最小信息集。
	// 当前 LAC/TAC 字段为 nil（W3 待定点：LAC/TAC 数据源 — devices/device_info
	// 表无对应列，需后续确认 device_parameters TR-069 path 或 sites 关联）。
	ListAllForRuleEval(ctx context.Context) ([]DeviceForMatch, error)
}

// DeviceRuleService 设备规则服务
type DeviceRuleService struct {
	repo         DeviceRuleRepository
	taskRepo     RuleTaskRepository
	groupRepo    DeviceGroupRepository
	matcher      *DeviceMatcher
	pool         *pgxpool.Pool
	deviceLister DeviceLister      // 可选注入；nil 时 getAllDevices 返空切片
	eventBus     event.EventBus    // T-0027 D7 可选注入；nil 时 Start 跳过订阅装配
	cron         *cron.Cron        // T-0027 D6 cron @hourly 调度器，Start 后非 nil
	subscription event.Subscription // T-0027 D7 device.registered 订阅句柄
	metrics      *RuleMetrics      // T-0027 D8 可选注入；nil 时所有埋点 no-op（PRD §12.5）
	taskQueue    chan uuid.UUID    // 任务队列
	workers      int               // Worker 数量
	workerSem    *semaphore.Weighted // 并发控制：限制同时运行的goroutine数
	logger       *zap.Logger
}

// NewDeviceRuleService 创建设备规则服务
func NewDeviceRuleService(
	repo DeviceRuleRepository,
	taskRepo RuleTaskRepository,
	groupRepo DeviceGroupRepository,
	matcher *DeviceMatcher,
	pool *pgxpool.Pool,
	workers int,
	logger *zap.Logger,
) *DeviceRuleService {
	// 并发控制：CPU核心数 * 4，避免goroutine爆炸
	maxConcurrency := int64(runtime.NumCPU() * 4)

	svc := &DeviceRuleService{
		repo:      repo,
		taskRepo:  taskRepo,
		groupRepo: groupRepo,
		matcher:   matcher,
		pool:      pool,
		taskQueue: make(chan uuid.UUID, 100),
		workers:   workers,
		workerSem: semaphore.NewWeighted(maxConcurrency),
		logger:    logger,
	}

	// 启动 Worker
	svc.startWorkers()

	return svc
}

// startWorkers 启动异步任务 Worker
func (s *DeviceRuleService) startWorkers() {
	for i := 0; i < s.workers; i++ {
		go s.worker(i)
	}
}

// worker 异步任务处理协程
func (s *DeviceRuleService) worker(id int) {
	s.logger.Info("device rule worker started", zap.Int("worker_id", id))

	for taskID := range s.taskQueue {
		ctx := context.Background()
		s.processTask(ctx, taskID)
	}
}

// CreateRule 创建规则
func (s *DeviceRuleService) CreateRule(ctx context.Context, req CreateRuleRequest, operator string) (*DeviceRule, error) {
	// 1. 验证目标分组
	groupID, err := uuid.Parse(req.TargetGroupID)
	if err != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeRuleInvalidParameter, "invalid target_group_id", nil)
	}

	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupNotFound, "target group not found", nil)
	}

	if group.Level != 2 {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupLevelInvalid, "target group must be L2", nil)
	}

	// 2. 检查优先级冲突（仅启用时检查）
	if req.Enabled {
		exists, _ := s.repo.ExistsByPriority(ctx, req.Priority, nil)
		if exists {
			return nil, commonerrors.NewBusinessError(global.ErrCodeRulePriorityDuplicate,
				"priority already exists for enabled rule", nil)
		}
	}

	// 3. 生成规则描述
	operators := s.generateOperators(req.MatchingMode, req.NameRuleList, req.LACList, req.TACList, req.SerialNumberList)

	// 4. 创建规则
	rule := &DeviceRule{
		ID:               uuid.New(),
		Name:             req.Name,
		Priority:         req.Priority,
		TargetGroupID:    &groupID,
		Enabled:          req.Enabled,
		MatchingMode:     MatchingMode(req.MatchingMode),
		NameRuleList:     req.NameRuleList,
		LACList:          req.LACList,
		TACList:          req.TACList,
		SerialNumberList: req.SerialNumberList, // migration 000124：SN 模式专用列表
		Description:      req.Description,
		Operators:        operators,
		CreatedBy:        operator,
		UpdatedBy:        operator,
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}

	// 5. 更新分组的绑定关系（仅启用时）
	if req.Enabled {
		if err := s.groupRepo.UpdateBoundRule(ctx, groupID, rule.ID); err != nil {
			s.logger.Warn("failed to update group bound rule",
				zap.String("group_id", groupID.String()),
				zap.Error(err),
			)
		}
	}

	rule.TargetGroupName = group.Name

	// 6. 异步触发一次匹配 + 入组（PRD: 创建规则立即把满足条件的设备入组，
	//    不论 enabled 状态。enabled=false 时仅本次入组生效，后续心跳不会持续
	//    匹配此规则；用户可在前端 ToggleRule 启用后走常规心跳路径）。
	//
	// fire-and-forget goroutine：不阻塞 HTTP 响应；失败 warn 不影响 CreateRule
	// 成功。force=true 绕过 ApplyRule 内部的 !enabled 拒绝。
	s.triggerCreateApply(rule.ID, operator)

	return rule, nil
}

// triggerCreateApply CreateRule 内部异步钩子 — 起一个 goroutine 跑 force apply。
//
// 设计要点：
//   - 独立 ctx + 60s 超时（apply 走 worker 队列 + 全表扫描，比心跳慢）
//   - 触发 force=true 模式：即使 rule.enabled=false 也跑一次（满足 PRD 需求）
//   - apply 本身已是异步（写 task 入队即返回），这里再加一层 goroutine 只是为
//     了不阻塞 HTTP 请求 + 失败仅 warn
func (s *DeviceRuleService) triggerCreateApply(ruleID uuid.UUID, operator string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		task, err := s.applyRuleInternal(ctx, ruleID, ApplyRuleRequest{}, operator, true)
		if err != nil {
			s.logger.Warn("auto apply on rule create failed",
				zap.String("rule_id", ruleID.String()),
				zap.String("operator", operator),
				zap.Error(err))
			return
		}
		s.logger.Info("auto apply on rule create scheduled",
			zap.String("rule_id", ruleID.String()),
			zap.String("task_id", task.ID.String()))
	}()
}

// UpdateRule 更新规则
func (s *DeviceRuleService) UpdateRule(ctx context.Context, id uuid.UUID, req UpdateRuleRequest, operator string) (*DeviceRule, error) {
	// 1. 获取现有规则
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. 更新字段
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Priority != nil {
		// 检查优先级冲突
		if *req.Priority != rule.Priority && rule.Enabled {
			exists, _ := s.repo.ExistsByPriority(ctx, *req.Priority, &id)
			if exists {
				return nil, commonerrors.NewBusinessError(global.ErrCodeRulePriorityDuplicate,
					"priority already exists for enabled rule", nil)
			}
		}
		rule.Priority = *req.Priority
	}
	if req.TargetGroupID != nil {
		groupID, err := uuid.Parse(*req.TargetGroupID)
		if err != nil {
			return nil, commonerrors.NewBusinessError(global.ErrCodeRuleInvalidParameter, "invalid target_group_id", nil)
		}

		group, err := s.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupNotFound, "target group not found", nil)
		}

		if group.Level != 2 {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupLevelInvalid, "target group must be L2", nil)
		}

		// 清除旧分组的绑定
		if rule.TargetGroupID != nil && *rule.TargetGroupID != groupID {
			if err := s.groupRepo.ClearBoundRule(ctx, *rule.TargetGroupID); err != nil {
				s.logger.Warn("failed to clear old group bound rule", zap.Error(err))
			}
		}

		rule.TargetGroupID = &groupID
		rule.TargetGroupName = group.Name
	}
	if req.Enabled != nil {
		// 启用规则时，检查 priority 是否冲突，如果冲突则自动分配新的 priority
		if *req.Enabled && !rule.Enabled {
			exists, _ := s.repo.ExistsByPriority(ctx, rule.Priority, &rule.ID)
			if exists {
				// 获取下一个可用的 priority
				nextPriority, err := s.repo.GetNextPriority(ctx)
				if err != nil {
					s.logger.Warn("failed to get next priority", zap.Error(err))
				} else {
					rule.Priority = nextPriority
				}
			}
		}
		rule.Enabled = *req.Enabled
	}
	if req.MatchingMode != nil {
		rule.MatchingMode = MatchingMode(*req.MatchingMode)
	}
	if req.NameRuleList != nil {
		rule.NameRuleList = req.NameRuleList
	}
	if req.LACList != nil {
		rule.LACList = req.LACList
	}
	if req.TACList != nil {
		rule.TACList = req.TACList
	}
	if req.SerialNumberList != nil {
		// migration 000124：SerialNumber 模式更新；传 [] 也认为是清空意图。
		rule.SerialNumberList = req.SerialNumberList
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}

	rule.UpdatedBy = operator

	// 3. 重新生成规则描述
	rule.Operators = s.generateOperators(string(rule.MatchingMode), rule.NameRuleList, rule.LACList, rule.TACList, rule.SerialNumberList)

	// 4. 保存更新
	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}

	// 5. 更新分组绑定关系
	if rule.Enabled && rule.TargetGroupID != nil {
		if err := s.groupRepo.UpdateBoundRule(ctx, *rule.TargetGroupID, rule.ID); err != nil {
			s.logger.Warn("failed to update group bound rule", zap.Error(err))
		}
	} else if !rule.Enabled && rule.TargetGroupID != nil {
		// 禁用时解除绑定
		if err := s.groupRepo.ClearBoundRule(ctx, *rule.TargetGroupID); err != nil {
			s.logger.Warn("failed to clear group bound rule", zap.Error(err))
		}
	}

	return rule, nil
}

// DeleteRule 删除规则
func (s *DeviceRuleService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 1. 解除与分组的绑定
	if rule.TargetGroupID != nil {
		if err := s.groupRepo.ClearBoundRule(ctx, *rule.TargetGroupID); err != nil {
			s.logger.Warn("failed to clear group bound rule", zap.Error(err))
		}
	}

	// 2. 删除规则
	return s.repo.Delete(ctx, id)
}

// ToggleRule 切换规则启用状态
func (s *DeviceRuleService) ToggleRule(ctx context.Context, id uuid.UUID, enabled bool, operator string) (*DeviceRule, error) {
	req := UpdateRuleRequest{
		Enabled: &enabled,
	}
	return s.UpdateRule(ctx, id, req, operator)
}

// GetRule 获取规则详情
func (s *DeviceRuleService) GetRule(ctx context.Context, id uuid.UUID) (*DeviceRule, error) {
	return s.repo.GetByID(ctx, id)
}

// ListRules 获取规则列表
func (s *DeviceRuleService) ListRules(ctx context.Context, req RuleListRequest) (*RuleListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	items, total, err := s.repo.List(ctx, &req)
	if err != nil {
		return nil, err
	}

	// 填充 operators 字段（如果为空则动态生成）
	for i := range items {
		if items[i].Operators == "" && len(items[i].NameRuleList) > 0 {
			items[i].Operators = s.generateOperators(string(items[i].MatchingMode), items[i].NameRuleList, items[i].LACList, items[i].TACList, items[i].SerialNumberList)
		}
	}

	return &RuleListResponse{
		Items: items,
		Total: total,
	}, nil
}

// BatchSortRules 批量调整规则优先级
func (s *DeviceRuleService) BatchSortRules(ctx context.Context, items []RuleSortItem, operator string) error {
	return s.repo.BatchUpdatePriority(ctx, items)
}

// ApplyRule 应用规则（异步）— 公开 API 入口，强制 rule.enabled=true。
//
// handler 端点调用此方法；CreateRule 内部钩子走 applyRuleInternal(force=true)。
func (s *DeviceRuleService) ApplyRule(ctx context.Context, ruleID uuid.UUID, req ApplyRuleRequest, operator string) (*RuleTask, error) {
	return s.applyRuleInternal(ctx, ruleID, req, operator, false)
}

// applyRuleInternal 应用规则的内部实现。
//
// force=true 时绕过 enabled 检查 — 用于 CreateRule 创建后自动跑一次"补全已存在
// 的匹配设备入组"，即使规则状态为未启动（PRD: 创建即异步入组，与心跳路径解耦）。
//
// force=false 时（公开 ApplyRule 入口）严格 gate on rule.Enabled，避免管理员手
// 动 Apply 一个被禁用的规则把不该入的设备拉进去。
func (s *DeviceRuleService) applyRuleInternal(ctx context.Context, ruleID uuid.UUID, req ApplyRuleRequest, operator string, force bool) (*RuleTask, error) {
	s.logger.Info("topology.rule.evaluating", // PRD §12.5 log key
		zap.String("rule_id", ruleID.String()),
		zap.String("operator", operator),
		zap.Bool("force", force),
	)
	// 1. 获取规则
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	// 2. 检查是否启用（force 模式跳过 — 创建规则后允许 enabled=false 时一次性补匹配）
	if !force && !rule.Enabled {
		return nil, commonerrors.NewBusinessError(global.ErrCodeRuleNotEnabled, "rule is not enabled", nil)
	}

	// 3. 检查是否有正在运行的任务
	latestTask, err := s.taskRepo.GetLatestByRule(ctx, ruleID)
	if err == nil && latestTask != nil && latestTask.Status == "running" {
		return nil, commonerrors.NewBusinessError(global.ErrCodeRuleTaskRunning, "rule is already being applied", nil)
	}

	// 4. 创建任务
	task := &RuleTask{
		ID:        uuid.New(),
		RuleID:    ruleID,
		RuleName:  rule.Name,
		Status:    "pending",
		CreatedBy: operator,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// 5. 提交到任务队列
	s.taskQueue <- task.ID

	s.logger.Info("rule apply task created",
		zap.String("task_id", task.ID.String()),
		zap.String("rule_id", ruleID.String()),
		zap.String("rule_name", rule.Name),
		zap.Bool("force", force),
	)

	return task, nil
}

// GetTask 获取任务详情
func (s *DeviceRuleService) GetTask(ctx context.Context, ruleID, taskID uuid.UUID) (*RuleTask, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.RuleID != ruleID {
		return nil, fmt.Errorf("task does not belong to rule")
	}

	return task, nil
}

// ListTasks 获取规则的任务列表
func (s *DeviceRuleService) ListTasks(ctx context.Context, ruleID uuid.UUID, limit int) ([]RuleTask, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.taskRepo.ListByRule(ctx, ruleID, limit)
}

// processTask 处理应用任务（Worker 协程调用）
// 优化：使用semaphore控制并发，避免goroutine爆炸
func (s *DeviceRuleService) processTask(ctx context.Context, taskID uuid.UUID) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("task not found", zap.String("task_id", taskID.String()))
		return
	}

	// 更新状态为运行中
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	if err := s.taskRepo.Update(ctx, task); err != nil {
		s.logger.Error("failed to update task status", zap.Error(err))
		return
	}

	// 获取规则
	rule, err := s.repo.GetByID(ctx, task.RuleID)
	if err != nil {
		s.failTask(ctx, task, "rule not found")
		return
	}

	// 获取所有设备
	devices, err := s.getAllDevices(ctx)
	if err != nil {
		s.failTask(ctx, task, fmt.Sprintf("get devices: %v", err))
		return
	}

	task.TotalDevices = len(devices)
	s.taskRepo.Update(ctx, task)

	s.logger.Info("rule apply task started",
		zap.String("task_id", taskID.String()),
		zap.String("rule_name", rule.Name),
		zap.Int("total_devices", task.TotalDevices),
	)

	// 优化：批量匹配 + 并发控制
	const batchSize = 50 // 减小批次大小，提高并发利用率
	const progressUpdateInterval = 500 // 每500个设备更新一次进度，减少DB写入

	var matchedCount, failedCount int
	var mu sync.Mutex

	// 预编译匹配结果缓存（针对设备名称模式）
	type matchResult struct {
		matched bool
		err     error
	}
	results := make(chan matchResult, len(devices))

	// 处理设备
	for i := 0; i < len(devices); i += batchSize {
		end := i + batchSize
		if end > len(devices) {
			end = len(devices)
		}

		var wg sync.WaitGroup
		for j := i; j < end; j++ {
			// 使用semaphore控制并发
			if err := s.workerSem.Acquire(ctx, 1); err != nil {
				s.logger.Error("failed to acquire semaphore", zap.Error(err))
				break
			}

			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer s.workerSem.Release(1)

				device := devices[idx]
				matched, err := s.matcher.matchRule(ctx, rule, MatchRequest{
					DeviceID:     device.ID,
					DeviceName:   device.Name,
					SerialNumber: device.SerialNumber, // migration 000124 SN 模式
					LAC:          device.LAC,
					TAC:          device.TAC,
				})

				results <- matchResult{matched: matched, err: err}

				if err != nil {
					mu.Lock()
					failedCount++
					mu.Unlock()
					s.logger.Debug("match device failed",
						zap.String("device_id", device.ID.String()),
						zap.Error(err),
					)
					return
				}

				if matched && rule.TargetGroupID != nil {
					// T-0027 D5.B：写 source_type='rule' + source_rule_id
					affected, err := s.groupRepo.AddDeviceWithSource(
						ctx, *rule.TargetGroupID, device.ID, "rule", &rule.ID,
					)
					if err != nil {
						mu.Lock()
						failedCount++
						mu.Unlock()
						if s.metrics != nil {
							s.metrics.RecordEvaluation("failed", "manual")
						}
						return
					}
					if affected == 0 {
						// manual override 跳过
						s.logger.Debug("topology.rule.manual_skipped",
							zap.String("device_id", device.ID.String()),
							zap.String("rule_id", rule.ID.String()),
						)
						if s.metrics != nil {
							s.metrics.RecordEvaluation("skipped", "manual")
						}
						return
					}
					mu.Lock()
					matchedCount++
					mu.Unlock()
					if s.metrics != nil {
						s.metrics.RecordEvaluation("matched", "manual")
					}
				}
			}(j)
		}
		wg.Wait()

		// 定期更新进度，避免频繁DB写入
		if (i+batchSize)%progressUpdateInterval == 0 || end >= len(devices) {
			mu.Lock()
			task.MatchedCount = matchedCount
			task.FailedCount = failedCount
			mu.Unlock()
			if err := s.taskRepo.Update(ctx, task); err != nil {
				s.logger.Warn("failed to update task progress", zap.Error(err))
			}
		}
	}

	close(results)

	// 更新任务完成状态
	completedAt := time.Now()
	task.Status = "completed"
	mu.Lock()
	task.MatchedCount = matchedCount
	task.FailedCount = failedCount
	mu.Unlock()
	task.CompletedAt = &completedAt
	s.taskRepo.Update(ctx, task)

	s.logger.Info("rule apply task completed",
		zap.String("task_id", taskID.String()),
		zap.Int("matched", matchedCount),
		zap.Int("failed", failedCount),
	)
}

// failTask 标记任务失败
func (s *DeviceRuleService) failTask(ctx context.Context, task *RuleTask, errMsg string) {
	task.Status = "failed"
	task.ErrorMessage = errMsg
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	s.taskRepo.Update(ctx, task)

	// PRD §12.5：log key + metric 联动
	s.logger.Warn("topology.rule.apply_failed",
		zap.String("task_id", task.ID.String()),
		zap.String("rule_id", task.RuleID.String()),
		zap.String("error", errMsg),
	)
	s.metrics.RecordApplyFailure("other") // 通用归类；目标组缺失类失败由调用点定向归类（Step 11 优化）
}

// DeviceForMatch 用于匹配的设备信息
type DeviceForMatch struct {
	ID           uuid.UUID
	Name         string
	SerialNumber string // migration 000124：serialNumber 匹配模式需要
	LAC          *int
	TAC          *int
}

// SetDeviceLister 注入设备列举器，替换默认 stub 行为。
// 通常由 modules.go DI 装配阶段调用一次（Day 3+ 接通真实施）。
func (s *DeviceRuleService) SetDeviceLister(l DeviceLister) {
	s.deviceLister = l
}

// SetEventBus 注入 EventBus 让 Start 装配 device.registered 订阅。
// nil 时 Start 跳过订阅装配（仅 cron 路径生效）。
func (s *DeviceRuleService) SetEventBus(bus event.EventBus) {
	s.eventBus = bus
}

// SetMetrics 注入 RuleMetrics（PRD §12.5）。nil 时所有埋点 no-op。
// 所有 RuleMetrics 方法 nil-safe，setter 不强制 — 测试场景可省。
func (s *DeviceRuleService) SetMetrics(m *RuleMetrics) {
	s.metrics = m
}

// Start 启动后台任务：cron @hourly reEvaluateAll。
// 应在 DI 装配阶段调用一次；幂等（重复调用 no-op）。
// 未来 Day 7+ EventBus 订阅 device.inform.bootstrap 也在此装配。
//
// PRD §12.1 §D2 拍板 default 走 @hourly；§D5.B 重评全规则不区分 default
// vs 其他组（target_group 已是 GroupReader 计算结果），SQL 层 A4 守护
// （AddDeviceWithSource WHERE source_type IS DISTINCT FROM 'manual'）让
// manual override 行天然不被 cron 触动。
func (s *DeviceRuleService) Start(ctx context.Context) error {
	if s.cron == nil {
		s.cron = cron.New()
		if _, err := s.cron.AddFunc("@hourly", func() {
			s.logger.Info("topology.cron.tick") // PRD §12.5 log key
			if err := s.reEvaluateAll(ctx); err != nil {
				s.logger.Error("reEvaluateAll failed", zap.Error(err))
			}
		}); err != nil {
			s.cron = nil
			return fmt.Errorf("register cron @hourly: %w", err)
		}
		s.cron.Start()
		s.logger.Info("topology rule cron started", zap.String("schedule", "@hourly"))
	}
	// T-0027 D7：装配 device.registered 订阅（注：项目模式用 device.registered
	// 而非 PRD §12.7 早稿的 device.inform.bootstrap，避免 InformHandler 与
	// 后续消费者的 race — 与 ProvisioningEngine 同模式，commit 见 2026-03-18
	// REVIEW_adc99fd_chenbo01_device.md）
	if s.eventBus != nil && s.subscription == nil {
		sub, err := s.eventBus.QueueSubscribe(
			event.SubjectDeviceRegistered,
			"topology-rule-engine", // PRD §12.7 NATS Queue group
			func(ctx context.Context, evt event.Event) error {
				return s.handleDeviceRegistered(ctx, evt)
			},
		)
		if err != nil {
			return fmt.Errorf("subscribe device.registered: %w", err)
		}
		s.subscription = sub
		s.logger.Info("topology rule subscribed device.registered",
			zap.String("queue", "topology-rule-engine"))
	}
	return nil
}

// Stop 优雅停止 cron 调度器 + 取消 EventBus 订阅。
// 用于 graceful shutdown 与单元测试清理；幂等。
func (s *DeviceRuleService) Stop() {
	if s.subscription != nil {
		_ = s.subscription.Unsubscribe()
		s.subscription = nil
	}
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done() // 等正在执行的 job 结束
		s.cron = nil
	}
}

// handleDeviceRegistered 处理 device.registered 事件 —— 单设备规则评估路径。
//
// PRD §3 GWT A2/A3：新设备首次注册后，遍历启用规则按 priority 升序找首个匹配，
// 调 AddDeviceWithSource('rule', &rule.ID) 入目标分组；A4 manual 守护已 SQL 层。
// 单设备路径不创建 RuleTask（task 表用于审计批量 ApplyRule）；用 debug log 记录。
//
// LAC/TAC 当前 nil（W3 待定点）；Name = serial_number（与 PgDeviceLister
// fallback 一致）。device_info.device_name 由 cron @hourly 在用户填后接力。
func (s *DeviceRuleService) handleDeviceRegistered(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceID     uuid.UUID `json:"device_id"`
		SerialNumber string    `json:"serial_number"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device.registered payload: %w", err)
	}
	if payload.DeviceID == uuid.Nil {
		return fmt.Errorf("device.registered payload missing device_id")
	}

	s.logger.Debug("topology.event.bootstrap_received", // PRD §12.5 log key
		zap.String("device_id", payload.DeviceID.String()),
		zap.String("serial", payload.SerialNumber))

	rules, err := s.repo.GetEnabledByPriority(ctx)
	if err != nil {
		return fmt.Errorf("list enabled rules for bootstrap eval: %w", err)
	}

	for i := range rules {
		rule := &rules[i]
		matched, err := s.matcher.matchRule(ctx, rule, MatchRequest{
			DeviceID:     payload.DeviceID,
			DeviceName:   payload.SerialNumber,
			SerialNumber: payload.SerialNumber, // migration 000124 SN 模式
		})
		if err != nil {
			s.logger.Warn("match rule failed",
				zap.String("rule_id", rule.ID.String()),
				zap.Error(err))
			continue
		}
		if !matched || rule.TargetGroupID == nil {
			continue
		}
		// PRD §3 GWT A3：priority asc first-match-wins，单组归属
		s.logger.Info("topology.rule.priority_winner",
			zap.String("device_id", payload.DeviceID.String()),
			zap.String("rule_id", rule.ID.String()),
			zap.Int("priority", rule.Priority))

		affected, err := s.groupRepo.AddDeviceWithSource(
			ctx, *rule.TargetGroupID, payload.DeviceID, "rule", &rule.ID,
		)
		if err != nil {
			s.metrics.RecordEvaluation("failed", "inform")
			return fmt.Errorf("add device to group: %w", err)
		}
		if affected == 0 {
			s.logger.Debug("topology.rule.manual_skipped",
				zap.String("device_id", payload.DeviceID.String()),
				zap.String("rule_id", rule.ID.String()))
			s.metrics.RecordEvaluation("skipped", "inform")
		} else {
			s.logger.Info("topology.rule.matched",
				zap.String("device_id", payload.DeviceID.String()),
				zap.String("rule_id", rule.ID.String()),
				zap.String("group_id", rule.TargetGroupID.String()))
			s.metrics.RecordEvaluation("matched", "inform")
		}
		return nil // 找到首个匹配即停（A3 单组归属）
	}
	return nil // 无匹配规则属正常路径，不算错
}

// reEvaluateAll 遍历所有启用规则，对每条触发 ApplyRule。
// 用于 cron @hourly 重评：让 rule-applied 分组随设备状态漂移自动更新。
//
// 单条规则 ApplyRule 失败（如已有 running task / 目标 group 缺失）走
// log warn 不中断批量（PRD §3 GWT A5 — 单失败不影响其他）。
// A4 manual override 守护在 AddDeviceWithSource SQL 层（Day 5）已强制，
// 本路径不需额外过滤。
func (s *DeviceRuleService) reEvaluateAll(ctx context.Context) error {
	rules, err := s.repo.GetEnabledByPriority(ctx)
	if err != nil {
		return fmt.Errorf("list enabled rules for cron re-eval: %w", err)
	}
	// 周期更新 active_rules gauge（PRD §12.5）
	s.metrics.SetActiveRules(float64(len(rules)))
	if len(rules) == 0 {
		return nil
	}
	for i := range rules {
		rule := &rules[i]
		if _, err := s.ApplyRule(ctx, rule.ID, ApplyRuleRequest{}, "system:cron"); err != nil {
			s.logger.Warn("topology.cron.rule_skipped",
				zap.String("rule_id", rule.ID.String()),
				zap.String("rule_name", rule.Name),
				zap.Error(err),
			)
			s.metrics.RecordEvaluation("failed", "cron")
			continue
		}
	}
	return nil
}

// getAllDevices 获取所有设备供 worker 匹配。
// 注入了 deviceLister 时走真实施；未注入时返空切片（兼容 Day 1-2 行为）。
func (s *DeviceRuleService) getAllDevices(ctx context.Context) ([]DeviceForMatch, error) {
	if s.deviceLister == nil {
		return []DeviceForMatch{}, nil
	}
	devices, err := s.deviceLister.ListAllForRuleEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("list devices for rule eval: %w", err)
	}
	return devices, nil
}

// matchRule 匹配规则
func (m *DeviceMatcher) matchRule(ctx context.Context, rule *DeviceRule, req MatchRequest) (bool, error) {
	switch rule.MatchingMode {
	case MatchingModeDeviceName:
		return m.matchByDeviceName(rule.NameRuleList, req.DeviceName), nil
	case MatchingModeLAC:
		if req.LAC == nil {
			return false, nil
		}
		return m.matchByCode(rule.LACList, *req.LAC), nil
	case MatchingModeTAC:
		if req.TAC == nil {
			return false, nil
		}
		return m.matchByCode(rule.TACList, *req.TAC), nil
	case MatchingModeSerialNumber:
		if req.SerialNumber == "" {
			return false, nil
		}
		return m.matchBySerialNumber(rule.SerialNumberList, req.SerialNumber), nil
	default:
		return false, nil
	}
}

// generateOperators 生成规则描述
func (s *DeviceRuleService) generateOperators(mode string, nameRules []NameRule, lacList, tacList []int, snList []string) string {
	switch mode {
	case "deviceName", "and", "or":
		return s.generateNameOperators(nameRules)
	case "lac":
		return fmt.Sprintf("LAC: %v", lacList)
	case "tac":
		return fmt.Sprintf("TAC: %v", tacList)
	case "serialNumber":
		// migration 000124：精确成员列表，截断超长以防 UI 撑爆。
		if len(snList) > 5 {
			return fmt.Sprintf("SN(%d): %v ...", len(snList), snList[:5])
		}
		return fmt.Sprintf("SN(%d): %v", len(snList), snList)
	}
	return ""
}

// generateNameOperators 生成设备名称规则描述
func (s *DeviceRuleService) generateNameOperators(rules []NameRule) string {
	if len(rules) == 0 {
		return ""
	}

	// 按 OR 分组
	orGroups := [][]NameRule{{}}
	for _, rule := range rules {
		if rule.AndOr == "or" {
			orGroups = append(orGroups, []NameRule{rule})
		} else {
			orGroups[len(orGroups)-1] = append(orGroups[len(orGroups)-1], rule)
		}
	}

	conditionMap := map[string]string{
		"contain":     "包含",
		"notContain":  "不包含",
		"startWith":   "开头是",
		"endWith":     "结尾是",
	}

	var groupParts []string
	for _, group := range orGroups {
		var parts []string
		for _, rule := range group {
			cond := conditionMap[rule.Condition]
			if cond == "" {
				cond = rule.Condition
			}
			parts = append(parts, fmt.Sprintf("%s \"%s\"", cond, rule.Value))
		}
		if len(parts) > 0 {
			groupText := strings.Join(parts, " 且 ")
			if len(orGroups) > 1 || len(group) > 1 {
				groupText = "(" + groupText + ")"
			}
			groupParts = append(groupParts, groupText)
		}
	}

	return strings.Join(groupParts, " 或 ")
}
