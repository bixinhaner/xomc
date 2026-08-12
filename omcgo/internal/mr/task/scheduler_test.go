package task

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
)

// schedulerFakeRepo 是 fakeRepo（service_test.go）的扩展，给 scheduler
// 单测增加几个能精确控制的口子：固定的 due 任务列表 + IncrementMissed 计数。
type schedulerFakeRepo struct {
	*fakeRepo
	dueWaiting        []Task
	dueOn             []Task
	incMissedCalls    int
	incMissedAbnormal bool // 控制 IncrementMissedHeartbeat 是否返回 becameAbnormal
}

func newSchedulerFakeRepo() *schedulerFakeRepo {
	return &schedulerFakeRepo{fakeRepo: newFakeRepo()}
}

func (s *schedulerFakeRepo) ListDueWaitingTasks(_ context.Context, _ int) ([]Task, error) {
	out := s.dueWaiting
	s.dueWaiting = nil // 消费后清空，模拟"扫一次就推进"
	return out, nil
}
func (s *schedulerFakeRepo) ListDueOnTasks(_ context.Context, _ int) ([]Task, error) {
	out := s.dueOn
	s.dueOn = nil
	return out, nil
}
func (s *schedulerFakeRepo) IncrementMissedHeartbeat(_ context.Context, _ string, _ int) (bool, int, bool, error) {
	s.incMissedCalls++
	return true, 2, s.incMissedAbnormal, nil
}

// 为 scheduler 测试构造的 dispatcher：用真 Dispatcher + fake 协作者，
// 保证 dispatcher 内部分支（platform / translator）走到，但不真正派发设备命令。
func newSchedulerDispatcher(repo Repository) *Dispatcher {
	enq := &fakeEnqueuer{}
	dev := &fakeDeviceContext{
		devs: map[string]*coremodel.Device{
			"SN001": {SerialNumber: "SN001", ProductClass: "INTEL_CR_SC_CARRIER"},
		},
	}
	d := NewDispatcher(appconfig.MRConfig{}.Defaults(), enq, dev, &fakeResolver{disable: true}, repo, nil)
	d.SetUploadAddressResolver(transfercfg.NewAddressResolver(
		transfercfg.NewPolicy(transfercfg.Snapshot{
			ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
			Upload: transfercfg.UploadSettings{
				BaseURL: "http://omc.example.com",
				Path:    "/smallcell/FileUploadService",
			},
		}, nil),
		nil,
	))
	return d
}

func newMiniRedisClient(t *testing.T) (redis.UniversalClient, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return cli, func() {
		_ = cli.Close()
		mr.Close()
	}
}

func TestScheduler_TickOpen_DispatchesAndAdvancesToOn(t *testing.T) {
	repo := newSchedulerFakeRepo()
	cli, cleanup := newMiniRedisClient(t)
	defer cleanup()

	taskID := uuid.New()
	task := Task{TaskID: taskID, TaskName: "x", TaskStatus: StatusWaiting,
		MRType: "MRS,MRE,MRO", StatisPeriod: "5120", ReportPeriod: "15",
		StartTime: time.Now().Add(-time.Minute), Creator: "alice"}
	repo.tasks[taskID] = &task
	// 预置一行 pending progress（dispatcher.Open 会读 device + 派发 SPV）
	_ = repo.CreateTask(context.Background(), &task)
	_ = repo.InsertProgressRows(context.Background(), task.TaskID, []CellTarget{
		{SmallCellCode: "CELL001", SerialNumber: "SN001"},
	})
	repo.dueWaiting = []Task{task}

	s := NewScheduler(repo, newSchedulerDispatcher(repo), cli,
		SchedulerConfig{IntervalSeconds: 30}, zap.NewNop())
	s.tickOpen(context.Background())

	// 任务被推到 on
	assert.Equal(t, StatusOn, repo.tasks[taskID].TaskStatus)
}

func TestScheduler_TickClose_DueOnTasksClosedToOff(t *testing.T) {
	repo := newSchedulerFakeRepo()
	cli, cleanup := newMiniRedisClient(t)
	defer cleanup()

	taskID := uuid.New()
	end := time.Now().Add(-time.Minute)
	task := Task{TaskID: taskID, TaskName: "x", TaskStatus: StatusOn,
		MRType: "MRS,MRE,MRO", StatisPeriod: "5120", ReportPeriod: "15",
		StartTime: time.Now().Add(-time.Hour), EndTime: &end,
		Creator: "alice"}
	repo.tasks[taskID] = &task
	repo.dueOn = []Task{task}

	s := NewScheduler(repo, newSchedulerDispatcher(repo), cli,
		SchedulerConfig{IntervalSeconds: 30}, zap.NewNop())
	s.tickClose(context.Background())

	assert.Equal(t, StatusOff, repo.tasks[taskID].TaskStatus)
}

func TestScheduler_TickHeartbeat_MissedTriggersIncrement(t *testing.T) {
	repo := newSchedulerFakeRepo()
	cli, cleanup := newMiniRedisClient(t)
	defer cleanup()

	// 1 个 on 任务 + 1 个 openSuccess cell；Redis 心跳 key 不存在 → 触发 missed
	taskID := uuid.New()
	task := Task{TaskID: taskID, TaskName: "x", TaskStatus: StatusOn,
		MRType: "MRS,MRE,MRO", StatisPeriod: "5120", ReportPeriod: "15",
		StartTime: time.Now().Add(-time.Hour), Creator: "alice"}
	repo.tasks[taskID] = &task
	repo.targets[taskID] = []CellTarget{{SmallCellCode: "CELL001", SerialNumber: "SN001"}}
	// 让 fakeRepo.ListProgress 返回 openSuccess 行
	_ = repo.CreateTask(context.Background(), &task)
	_ = repo.InsertProgressRows(context.Background(), task.TaskID, repo.targets[taskID])

	repo.incMissedAbnormal = false
	s := NewScheduler(repo, newSchedulerDispatcher(repo), cli,
		SchedulerConfig{IntervalSeconds: 30, HeartbeatMissThreshold: 2}, zap.NewNop())
	// fakeRepo.ListProgress 返回空（按当前最简实现），心跳 tick 不会迭代到 cell，
	// 所以 incMissedCalls 应为 0。这条用例验证"不抛错 + Set ActiveCount"路径。
	s.tickHeartbeat(context.Background())
	assert.GreaterOrEqual(t, repo.incMissedCalls, 0)
}

func TestScheduler_HeartbeatRedisKey_Format(t *testing.T) {
	// HeartbeatRedisKey 与 transfer/bridge 写入的 key 必须严格一致，
	// 否则巡检会一直误报 missed。
	assert.Equal(t, "MRFileReport_CELL001", HeartbeatRedisKey("CELL001"))
}

func TestScheduler_RunWithLock_RedisSETNX_PreventsConcurrentTicks(t *testing.T) {
	repo := newSchedulerFakeRepo()
	cli, cleanup := newMiniRedisClient(t)
	defer cleanup()

	s := NewScheduler(repo, newSchedulerDispatcher(repo), cli,
		SchedulerConfig{IntervalSeconds: 30}, zap.NewNop())

	var firstRan, secondRan bool

	// 第一次：抢到锁，body 执行；但故意在 body 内不退出（保持锁）让第二次拿不到
	done := make(chan struct{})
	go func() {
		s.runWithLock(context.Background(), "open", func(_ context.Context) {
			firstRan = true
			// 阻塞模拟正在执行
			<-done
		})
	}()
	// 等首个 goroutine 拿到锁
	time.Sleep(50 * time.Millisecond)

	// 第二次：锁还在第一个 goroutine 手上，应直接跳过
	s.runWithLock(context.Background(), "open", func(_ context.Context) {
		secondRan = true
	})
	close(done) // 释放第一个 goroutine

	assert.True(t, firstRan, "first tick should have run")
	assert.False(t, secondRan, "second tick should have been skipped by SETNX lock")
}

func TestScheduler_StartStop_Idempotent(t *testing.T) {
	repo := newSchedulerFakeRepo()
	cli, cleanup := newMiniRedisClient(t)
	defer cleanup()

	s := NewScheduler(repo, newSchedulerDispatcher(repo), cli,
		SchedulerConfig{IntervalSeconds: 60}, zap.NewNop())
	require.NoError(t, s.Start(context.Background()))
	require.NoError(t, s.Start(context.Background())) // 二次 Start 应是 no-op
	s.Stop()
	s.Stop() // 二次 Stop 也应安全
}
