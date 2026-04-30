package topology

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// newTestRuleService 构造仅用于单元测试的 DeviceRuleService。
//
// workers=0 避免 startWorkers 启动 goroutine 泄漏；
// pool=nil — 单测路径不命中 pool（仅 ApplyRule 同步部分），需要 SQL 路径的测试改用 integration test。
//
// 返回 service + 三个 repo mock 句柄供 EXPECT() 使用。
func newTestRuleService(t *testing.T, ctrl *gomock.Controller) (
	*DeviceRuleService,
	*MockDeviceRuleRepository,
	*MockRuleTaskRepository,
	*MockDeviceGroupRepository,
) {
	t.Helper()
	repo := NewMockDeviceRuleRepository(ctrl)
	taskRepo := NewMockRuleTaskRepository(ctrl)
	groupRepo := NewMockDeviceGroupRepository(ctrl)
	matcher := NewDeviceMatcher(groupRepo, nil, zap.NewNop())

	svc := NewDeviceRuleService(repo, taskRepo, groupRepo, matcher, nil, 0, zap.NewNop())
	return svc, repo, taskRepo, groupRepo
}

// makeEnabledRule 构造一条已启用、target_group 指定、LAC 模式的测试规则。
func makeEnabledRule(t *testing.T, targetGroupID uuid.UUID, priority int, lacList []int) *DeviceRule {
	t.Helper()
	return &DeviceRule{
		ID:            uuid.New(),
		Name:          "test-rule",
		Priority:      priority,
		TargetGroupID: &targetGroupID,
		Enabled:       true,
		MatchingMode:  MatchingModeLAC,
		LACList:       lacList,
		CreatedBy:     "tester",
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// A1 — ApplyRule happy path：rule enabled，无并发任务，应创建 pending task 并入队
// ──────────────────────────────────────────────────────────────────────────────
//
// PRD §3 GWT A1 全语义（matched_count: 300 / source_type='rule' 行 300 / EventBus
// topology.rule.applied 发布）的端到端验证依赖 worker 异步路径 + getAllDevices
// 真实实现 + pg_repository UPDATE 写 source_type 列 + EventBus 接入。这些落地后
// 升级为 integration test（命名 TestApplyRule_FullPipeline_AppliesAndPublishes）。
//
// 本单测仅守 ApplyRule 同步部分契约：参数校验 + GetByID + 任务创建 + queue 入队。
func TestApplyRule_HappyPath_CreatesPendingTaskAndQueues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, taskRepo, _ := newTestRuleService(t, ctrl)
	ctx := context.Background()

	// Given: 一条启用规则 R + 当前无 running task
	targetGroup := uuid.New()
	rule := makeEnabledRule(t, targetGroup, 50, []int{100, 101})

	repo.EXPECT().GetByID(ctx, rule.ID).Return(rule, nil)
	taskRepo.EXPECT().GetLatestByRule(ctx, rule.ID).Return(nil, nil) // 无并发任务
	taskRepo.EXPECT().Create(ctx, gomock.Any()).
		Do(func(_ context.Context, task *RuleTask) {
			require.Equal(t, rule.ID, task.RuleID)
			require.Equal(t, "pending", task.Status)
			require.Equal(t, rule.Name, task.RuleName)
		}).
		Return(nil)

	// When: 调 ApplyRule
	task, err := svc.ApplyRule(ctx, rule.ID, ApplyRuleRequest{}, "operator-x")

	// Then: 任务创建成功 + 状态 pending + 已入 queue
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "pending", task.Status)
	assert.Equal(t, rule.ID, task.RuleID)

	// Queue 校验：taskQueue 应有 1 条
	select {
	case taskID := <-svc.taskQueue:
		assert.Equal(t, task.ID, taskID, "ApplyRule 应将 task.ID 入 queue")
	default:
		t.Fatal("ApplyRule 未将 task 入 queue")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// A1.b — ApplyRule rule 未启用 → 返 ErrCodeRuleNotEnabled
// ──────────────────────────────────────────────────────────────────────────────
func TestApplyRule_RuleNotEnabled_ReturnsBusinessError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, _ := newTestRuleService(t, ctrl)
	ctx := context.Background()

	rule := makeEnabledRule(t, uuid.New(), 50, []int{100})
	rule.Enabled = false

	repo.EXPECT().GetByID(ctx, rule.ID).Return(rule, nil)

	// When + Then
	task, err := svc.ApplyRule(ctx, rule.ID, ApplyRuleRequest{}, "operator-x")
	require.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "not enabled")
}

// ──────────────────────────────────────────────────────────────────────────────
// A1.d — getAllDevices wiring：未注入 DeviceLister → 返空切片（兼容 Day 1-2 stub）
// ──────────────────────────────────────────────────────────────────────────────
func TestGetAllDevices_NoListerInjected_ReturnsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _ := newTestRuleService(t, ctrl)
	devices, err := svc.getAllDevices(context.Background())

	require.NoError(t, err)
	assert.Empty(t, devices, "未注入 DeviceLister 时应返空切片")
}

// ──────────────────────────────────────────────────────────────────────────────
// A1.e — getAllDevices wiring：注入 DeviceLister 后委托调用
// ──────────────────────────────────────────────────────────────────────────────
func TestGetAllDevices_WithListerInjected_DelegatesToLister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _ := newTestRuleService(t, ctrl)
	lister := NewMockDeviceLister(ctrl)
	svc.SetDeviceLister(lister)

	expected := []DeviceForMatch{
		{ID: uuid.New(), Name: "dev-001"},
		{ID: uuid.New(), Name: "dev-002"},
	}
	lister.EXPECT().ListAllForRuleEval(gomock.Any()).Return(expected, nil)

	devices, err := svc.getAllDevices(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expected, devices)
}

// ──────────────────────────────────────────────────────────────────────────────
// A1.c — ApplyRule 已有 running task → 拒绝并发
// ──────────────────────────────────────────────────────────────────────────────
func TestApplyRule_ConcurrentRunningTask_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, taskRepo, _ := newTestRuleService(t, ctrl)
	ctx := context.Background()

	rule := makeEnabledRule(t, uuid.New(), 50, []int{100})
	runningTask := &RuleTask{ID: uuid.New(), RuleID: rule.ID, Status: "running"}

	repo.EXPECT().GetByID(ctx, rule.ID).Return(rule, nil)
	taskRepo.EXPECT().GetLatestByRule(ctx, rule.ID).Return(runningTask, nil)

	task, err := svc.ApplyRule(ctx, rule.ID, ApplyRuleRequest{}, "operator-x")
	require.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "already being applied")
}

// ──────────────────────────────────────────────────────────────────────────────
// A2 — 新设备 bootstrap → 自动评估并入 target group（PRD §3 GWT A2）
// ──────────────────────────────────────────────────────────────────────────────
// 阻塞在 S3 Day 3+ 的 EventBus 订阅 + handleDeviceBootstrap 实现：
//   - 需新增 handleDeviceBootstrap(ctx, evt *event.DeviceInformEvent) error 方法
//   - 需 mock event.EventBus + DeviceLister
// 落地后这里实测：bootstrap event → matcher 单设备 → BatchAddDevices 到 target group + source_type='rule'
func TestHandleDeviceBootstrap_MatchedDevice_AssignsToTargetGroup_TODO(t *testing.T) {
	t.Skip("TODO Day 3+: handleDeviceBootstrap method not yet implemented (PRD §12.1, §12.7)")
}

// ──────────────────────────────────────────────────────────────────────────────
// A3 — 多规则匹配按 priority 数字小者胜（PRD §3 GWT A3）
// ──────────────────────────────────────────────────────────────────────────────
// 阻塞在 handleDeviceBootstrap 实现 — 内部需按 GetEnabledByPriority 顺序遍历
// 找首个 match 的规则，跳过低优先级。落地后：
//   - 准备 R1 priority=10 + R2 priority=20 同时 match D
//   - 调 handleDeviceBootstrap → 仅 R1 target_group 加 D
//   - log 含 "topology.rule.priority_winner" debug
func TestHandleDeviceBootstrap_MultiRuleMatch_LowerPriorityWins_TODO(t *testing.T) {
	t.Skip("TODO Day 3+: priority resolution in handleDeviceBootstrap (PRD §3 A3)")
}

// ──────────────────────────────────────────────────────────────────────────────
// A4 — manual override 在 cron 重评中保留（PRD §3 GWT A4）
// ──────────────────────────────────────────────────────────────────────────────
// 阻塞在 reEvaluateAll 实现 — 需新增 method + cron 装配 + SQL 过滤
// `WHERE source_type != 'manual'`。落地后：
//   - 设备 D 通过 ApplyRule 进 G（source_type='rule'）
//   - 运维 manual MOVE D 到 H（source_type='manual'）
//   - 规则 R 改 LAC 阈值，D 已不再符合
//   - 调 reEvaluateAll
//   - 断言 D 仍在 H + log 含 "topology.rule.manual_skipped"
func TestReEvaluateAll_ManualOverride_NotTouched_TODO(t *testing.T) {
	t.Skip("TODO Day 3+: reEvaluateAll method + source_type column wiring (PRD §3 A4 / §12.1 / §12.3)")
}

// ──────────────────────────────────────────────────────────────────────────────
// A5 — 单条规则失败不影响其他规则（PRD §3 GWT A5）
// ──────────────────────────────────────────────────────────────────────────────
// 当前 ApplyRule 是单规则路径，A5 语义需要 ApplyAllEnabled 或 worker 批量循环；
// PRD §3 A5 用例 "POST /device-rules/all/apply" 端点目前不存在。
// 落地选择：① reEvaluateAll 内对每条规则独立 try-catch；② 新加批量端点；
// 均归 S3 Day 3+ 实施。
func TestApplyAll_SingleRuleFailure_OthersSucceed_TODO(t *testing.T) {
	t.Skip("TODO Day 3+: batch apply path with per-rule isolation (PRD §3 A5)")
}
