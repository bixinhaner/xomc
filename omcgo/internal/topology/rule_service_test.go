package topology

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
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
// A1.f — D5.B wiring：processTask matched 路径调 AddDeviceWithSource('rule')
// ──────────────────────────────────────────────────────────────────────────────
//
// PRD §3 GWT A1 + D5.B 决议：rule-driven add 必须传 source_type='rule' +
// source_rule_id，让 cron reEvaluateAll 后续可识别 rule-applied 行；A4
// manual override 通过 SQL 层 `WHERE source_type IS DISTINCT FROM 'manual'`
// 守护，本路径只验证调用契约。
func TestProcessTask_MatchedDevice_CallsAddDeviceWithSourceRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, taskRepo, groupRepo := newTestRuleService(t, ctrl)
	lister := NewMockDeviceLister(ctrl)
	svc.SetDeviceLister(lister)

	// Given: 启用规则 R 按 deviceName equal "dev-001" 匹配
	targetGroup := uuid.New()
	rule := &DeviceRule{
		ID:            uuid.New(),
		Name:          "name-rule",
		Priority:      50,
		TargetGroupID: &targetGroup,
		Enabled:       true,
		MatchingMode:  MatchingModeDeviceName,
		NameRuleList:  []NameRule{{Condition: "contain", Value: "dev-001"}}, // matcher.go 未实现 "equal"（pre-existing 缺口）
	}
	deviceID := uuid.New()
	taskID := uuid.New()

	// Mock 期望（按 processTask 调用顺序）
	taskRepo.EXPECT().GetByID(gomock.Any(), taskID).Return(
		&RuleTask{ID: taskID, RuleID: rule.ID, RuleName: rule.Name, Status: "pending"}, nil,
	)
	taskRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	repo.EXPECT().GetByID(gomock.Any(), rule.ID).Return(rule, nil)
	lister.EXPECT().ListAllForRuleEval(gomock.Any()).Return([]DeviceForMatch{
		{ID: deviceID, Name: "dev-001"},
	}, nil)

	// **核心契约断言**：AddDeviceWithSource 被调用且参数匹配 D5.B
	groupRepo.EXPECT().AddDeviceWithSource(
		gomock.Any(),
		targetGroup,
		deviceID,
		"rule",
		gomock.Cond(func(p any) bool {
			ruleIDPtr, ok := p.(*uuid.UUID)
			return ok && ruleIDPtr != nil && *ruleIDPtr == rule.ID
		}),
	).Return(int64(1), nil)

	// When: 直接驱动 worker 私有方法（workers=0 testing 通道）
	svc.processTask(context.Background(), taskID)

	// Then: gomock EXPECT 自动断言；ctrl.Finish 验证全部期望被满足
}

// ──────────────────────────────────────────────────────────────────────────────
// A2 — device.registered → 自动评估命中规则并入 target group（PRD §3 GWT A2）
// ──────────────────────────────────────────────────────────────────────────────
//
// 注：实际订阅 SubjectDeviceRegistered（项目模式 — 见 commit 2026-03-18 race fix）
// 而非 PRD §12.7 早稿写的 device.inform.bootstrap；保留测试名称含 "Bootstrap"
// 反映 PRD §3 用例语义（新设备 bootstrap 概念）。
func TestHandleDeviceRegistered_MatchedDevice_AssignsToTargetGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, groupRepo := newTestRuleService(t, ctrl)

	deviceID := uuid.New()
	targetGroup := uuid.New()
	// 注册时设备尚无 site_name（站点名称为配置项），名称匹配交由 cron @hourly 接力，
	// 故实时路径用 SN 模式校验注册→入组流程（site_name 名称匹配见下方 _NameMode_ 测试）。
	rule := DeviceRule{
		ID: uuid.New(), Name: "sn-rule", Priority: 50, Enabled: true,
		TargetGroupID: &targetGroup, MatchingMode: MatchingModeSerialNumber,
		SerialNumberList: []string{"SN-ABC123-001"},
	}

	// Mock: GetEnabledByPriority 返一条规则；命中后 AddDeviceWithSource
	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return([]DeviceRule{rule}, nil)
	groupRepo.EXPECT().AddDeviceWithSource(
		gomock.Any(), targetGroup, deviceID, "rule",
		gomock.Cond(func(p any) bool {
			rid, ok := p.(*uuid.UUID)
			return ok && rid != nil && *rid == rule.ID
		}),
	).Return(int64(1), nil)

	// 构造 device.registered 事件
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     deviceID,
		"serial_number": "SN-ABC123-001",
	})
	require.NoError(t, err)

	// When
	err = svc.handleDeviceRegistered(context.Background(), evt)
	require.NoError(t, err)
}

// ──────────────────────────────────────────────────────────────────────────────
// A3 — 多规则匹配按 priority 升序首个胜（PRD §3 GWT A3）
// ──────────────────────────────────────────────────────────────────────────────
func TestHandleDeviceRegistered_MultiRuleMatch_LowestPriorityWins(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, groupRepo := newTestRuleService(t, ctrl)

	deviceID := uuid.New()
	g1 := uuid.New()
	g2 := uuid.New()
	r1 := DeviceRule{ID: uuid.New(), Name: "r1-pri10", Priority: 10, Enabled: true,
		TargetGroupID: &g1, MatchingMode: MatchingModeSerialNumber,
		SerialNumberList: []string{"SN-ABC-X"}}
	r2 := DeviceRule{ID: uuid.New(), Name: "r2-pri20", Priority: 20, Enabled: true,
		TargetGroupID: &g2, MatchingMode: MatchingModeSerialNumber,
		SerialNumberList: []string{"SN-ABC-X"}}

	// repo 已按 priority 升序返回（GetEnabledByPriority 契约）
	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return([]DeviceRule{r1, r2}, nil)
	// 仅 r1 应触发 AddDeviceWithSource，r2 跳过
	groupRepo.EXPECT().AddDeviceWithSource(
		gomock.Any(), g1, deviceID, "rule", gomock.Any(),
	).Return(int64(1), nil)

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     deviceID,
		"serial_number": "SN-ABC-X",
	})
	require.NoError(t, err)

	require.NoError(t, svc.handleDeviceRegistered(context.Background(), evt))
}

// ──────────────────────────────────────────────────────────────────────────────
// A2.b — 无匹配规则属正常路径，不报错也不调 AddDevice
// ──────────────────────────────────────────────────────────────────────────────
func TestHandleDeviceRegistered_NoMatchingRule_NoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, _ := newTestRuleService(t, ctrl)
	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return([]DeviceRule{}, nil)

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     uuid.New(),
		"serial_number": "SN-X",
	})
	require.NoError(t, err)

	require.NoError(t, svc.handleDeviceRegistered(context.Background(), evt))
}

// ──────────────────────────────────────────────────────────────────────────────
// 反退化 — 名称匹配（deviceName 模式）以 devices.site_name 为匹配字段。
// 设备首次注册时 site_name 尚未配置（站点名称为管理面配置项），实时路径不做名称
// 匹配，DeviceName 传空串 → matchByDeviceName 安全降级不命中 → 不入组。
// 名称匹配交由 cron @hourly 在用户填好 site_name 后接力。
// 守护 rule_service.go handleDeviceRegistered 不再用 SN 冒充设备名称。
// ──────────────────────────────────────────────────────────────────────────────
func TestHandleDeviceRegistered_NameMode_DeferredToCron_NotMatchedAtRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, _ := newTestRuleService(t, ctrl)

	// 名称规则 contain "SN-ABC123" —— 若实时路径仍用 SN 冒充名称，会误命中
	rule := DeviceRule{
		ID: uuid.New(), Name: "name-rule", Priority: 50, Enabled: true,
		TargetGroupID: func() *uuid.UUID { id := uuid.New(); return &id }(),
		MatchingMode:  MatchingModeDeviceName,
		NameRuleList:  []NameRule{{Condition: "contain", Value: "SN-ABC123"}},
	}
	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return([]DeviceRule{rule}, nil)
	// 关键断言：groupRepo.AddDeviceWithSource 不应被调用（无 EXPECT 即不可调用）

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     uuid.New(),
		"serial_number": "SN-ABC123-001", // SN 含规则子串，但名称匹配不应用 SN
	})
	require.NoError(t, err)

	require.NoError(t, svc.handleDeviceRegistered(context.Background(), evt))
}

// ──────────────────────────────────────────────────────────────────────────────
// A2.c — Start 注入 EventBus 后装配 QueueSubscribe 订阅
// ──────────────────────────────────────────────────────────────────────────────
func TestStart_WithEventBus_RegistersQueueSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _ := newTestRuleService(t, ctrl)
	bus := NewMockEventBus(ctrl)
	sub := NewMockSubscription(ctrl)

	bus.EXPECT().QueueSubscribe(
		event.SubjectDeviceRegistered,
		"topology-rule-engine",
		gomock.Any(),
	).Return(sub, nil)
	sub.EXPECT().Unsubscribe().Return(nil)

	svc.SetEventBus(bus)
	require.NoError(t, svc.Start(context.Background()))
	assert.NotNil(t, svc.subscription)

	// Stop 应取消订阅
	svc.Stop()
	assert.Nil(t, svc.subscription)
}

// ──────────────────────────────────────────────────────────────────────────────
// A4.a — reEvaluateAll：无启用规则时返 nil（边界用例）
// ──────────────────────────────────────────────────────────────────────────────
func TestReEvaluateAll_NoEnabledRules_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, _ := newTestRuleService(t, ctrl)
	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return([]DeviceRule{}, nil)

	err := svc.reEvaluateAll(context.Background())
	require.NoError(t, err)
}

// ──────────────────────────────────────────────────────────────────────────────
// A4.b — reEvaluateAll：repo error 包装返出
// ──────────────────────────────────────────────────────────────────────────────
func TestReEvaluateAll_RepoError_PropagatesWrapped(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, _, _ := newTestRuleService(t, ctrl)
	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return(nil, fmt.Errorf("db down"))

	err := svc.reEvaluateAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list enabled rules for cron re-eval")
	assert.Contains(t, err.Error(), "db down")
}

// ──────────────────────────────────────────────────────────────────────────────
// A4.c — reEvaluateAll：单条规则 ApplyRule 失败不中断批量（PRD §3 GWT A5 语义在 cron 路径）
// ──────────────────────────────────────────────────────────────────────────────
//
// 设两条启用规则 R1/R2；R1 ApplyRule 因 GetByID 失败抛错，R2 应仍被尝试。
// 验证：repo.GetEnabledByPriority 返 2 规则；第一规则 ApplyRule chain 出错；
// 第二规则 ApplyRule chain 完整跑通；reEvaluateAll 返 nil（per-rule 错误转 log）。
func TestReEvaluateAll_SingleRuleFailure_ContinuesBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, taskRepo, _ := newTestRuleService(t, ctrl)

	r1 := DeviceRule{ID: uuid.New(), Name: "r1-broken", Enabled: true}
	r2 := DeviceRule{ID: uuid.New(), Name: "r2-ok", Enabled: true, MatchingMode: MatchingModeDeviceName,
		NameRuleList: []NameRule{{Condition: "contain", Value: "x"}}, TargetGroupID: ptrUUID(uuid.New())}

	repo.EXPECT().GetEnabledByPriority(gomock.Any()).Return([]DeviceRule{r1, r2}, nil)

	// R1: ApplyRule 入口 GetByID 失败（模拟 db hiccup）
	repo.EXPECT().GetByID(gomock.Any(), r1.ID).Return(nil, fmt.Errorf("rule fetch failed"))
	// R2: ApplyRule 完整链：GetByID rule + GetLatestByRule + Create task
	repo.EXPECT().GetByID(gomock.Any(), r2.ID).Return(&r2, nil)
	taskRepo.EXPECT().GetLatestByRule(gomock.Any(), r2.ID).Return(nil, nil)
	taskRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	err := svc.reEvaluateAll(context.Background())
	require.NoError(t, err, "单条规则失败不应中断批量（per-rule 转 log）")

	// R2 task 入队
	select {
	case <-svc.taskQueue:
	default:
		t.Fatal("R2 task 未入队")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// A4.d — Start: 注册 @hourly cron 后调度器为非 nil
// ──────────────────────────────────────────────────────────────────────────────
func TestStart_RegistersCron(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _ := newTestRuleService(t, ctrl)
	require.Nil(t, svc.cron)

	require.NoError(t, svc.Start(context.Background()))
	require.NotNil(t, svc.cron, "Start 后 cron 应非 nil")
	assert.Len(t, svc.cron.Entries(), 1, "应注册 1 个 cron entry (@hourly)")

	// 幂等：第二次调用 no-op
	require.NoError(t, svc.Start(context.Background()))
	assert.Len(t, svc.cron.Entries(), 1, "Start 幂等：cron entry 数仍为 1")

	svc.Stop()
	assert.Nil(t, svc.cron, "Stop 后 cron 置 nil")
}

// ──────────────────────────────────────────────────────────────────────────────
// D8 — RuleMetrics 全部方法 nil-safe（PRD §12.5；模仿 PolicyMetrics 模式）
// ──────────────────────────────────────────────────────────────────────────────
//
// 测试场景：未注入 metrics 时（生产 reg=nil 或单测路径）service 调埋点
// 不应 panic。RuleMetrics 通过 nil receiver 早 return 实现。
func TestRuleMetrics_NilSafe_AllMethods(t *testing.T) {
	var m *RuleMetrics // nil 指针

	// 应无 panic
	m.RecordEvaluation("matched", "manual")
	m.RecordEvaluation("skipped", "cron")
	m.RecordEvaluation("failed", "inform")
	m.ObserveEvaluationDuration("rule-id", 0.123)
	m.RecordApplyFailure("db_error")
	m.SetDevicesInRuleGroups(42)
	m.SetActiveRules(7)
	m.RecordDefaultGroupMigration("migrated")
}

// ──────────────────────────────────────────────────────────────────────────────
// D8 — RuleMetrics reg=nil 构造 + 方法调用 不 panic（生产模式抽样）
// ──────────────────────────────────────────────────────────────────────────────
func TestNewRuleMetrics_NilRegistry_DoesNotRegister(t *testing.T) {
	m := NewRuleMetrics(nil) // 单测场景常见
	require.NotNil(t, m)
	// 调几个常用方法验证非 nil 字段也不 panic
	m.RecordEvaluation("matched", "manual")
	m.SetActiveRules(3)
	m.ObserveEvaluationDuration("r1", 0.05)
}

// ──────────────────────────────────────────────────────────────────────────────
// A4 — manual override 在 cron 重评中保留（PRD §3 GWT A4 — SQL 层守护）
// ──────────────────────────────────────────────────────────────────────────────
//
// A4 manual override 守护由 AddDeviceWithSource SQL 层 WHERE 子句强制
// （Day 5 commit ba45e854 落地）。reEvaluateAll cron 路径调 ApplyRule
// → processTask → AddDeviceWithSource 自然继承守护。
//
// 单元层验证 SQL 行为需真 PG 或 sqlmock，本测试遗留 SKIP 待 S4 integration
// 实测；reEvaluateAll 调用契约由 A4.a/A4.b/A4.c + Day 5
// TestProcessTask_MatchedDevice_CallsAddDeviceWithSourceRule 联合覆盖。
func TestReEvaluateAll_ManualOverride_NotTouched_TODO(t *testing.T) {
	t.Skip("integration test 待 S4 实测：PG 真验证 source_type='manual' 行 rowsAffected=0")
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
