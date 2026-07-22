package adhoc

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// ---------------------------------------------------------------------------
// Worker.tryRunOne：核心抢-跑-切终态闭环
// ---------------------------------------------------------------------------

type workerStubRepo struct {
	mu            sync.Mutex
	pendingTasks  []*Task
	statusUpdates []struct {
		id     uuid.UUID
		status Status
	}
	inserted []ResultRow

	// T-0186：运行记录捕获
	insertedRuns            []TaskRun           // 跑前 INSERT 的 run（status=running 快照）
	finishedRuns            []finishedRunRecord // 跑后 FinishRun 的终态
	nextSeq                 int                 // NextRunSeq 返回值（默认 1）
	insertResults           func([]ResultRow) error
	updateStatusErrByStatus map[Status]error
}

type finishedRunRecord struct {
	runID     uuid.UUID
	status    Status
	rowsTotal int
	errMsg    string
}

func (s *workerStubRepo) Create(context.Context, CreateRequest) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *workerStubRepo) Update(context.Context, uuid.UUID, UpdateRequest) error { return nil }
func (s *workerStubRepo) Get(context.Context, uuid.UUID) (*Task, error)          { return nil, nil }
func (s *workerStubRepo) List(context.Context, ListFilter) ([]Task, error)       { return nil, nil }
func (s *workerStubRepo) Cancel(context.Context, uuid.UUID) error                { return nil }
func (s *workerStubRepo) Resume(context.Context, uuid.UUID) (Status, error)      { return "", nil }
func (s *workerStubRepo) Delete(context.Context, uuid.UUID) error                { return nil }

func (s *workerStubRepo) LockNextPending(context.Context, string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pendingTasks) == 0 {
		return nil, ErrNoPendingTask
	}
	t := s.pendingTasks[0]
	s.pendingTasks = s.pendingTasks[1:]
	return t, nil
}

func (s *workerStubRepo) UpdateStatus(_ context.Context, id uuid.UUID, status Status, _ *int, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusUpdates = append(s.statusUpdates, struct {
		id     uuid.UUID
		status Status
	}{id, status})
	return s.updateStatusErrByStatus[status]
}

func (s *workerStubRepo) InsertResults(_ context.Context, rows []ResultRow) error {
	if s.insertResults != nil {
		return s.insertResults(rows)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inserted = append(s.inserted, rows...)
	return nil
}

func (s *workerStubRepo) NextRunSeq(context.Context, uuid.UUID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nextSeq == 0 {
		return 1, nil
	}
	return s.nextSeq, nil
}

func (s *workerStubRepo) InsertRun(_ context.Context, run TaskRun) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New()
	run.ID = id
	s.insertedRuns = append(s.insertedRuns, run)
	return id, nil
}

func (s *workerStubRepo) FinishRun(_ context.Context, runID uuid.UUID, status Status, rowsTotal int, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finishedRuns = append(s.finishedRuns, finishedRunRecord{runID, status, rowsTotal, errMsg})
	return nil
}

func (s *workerStubRepo) ListRuns(context.Context, uuid.UUID, int, int) ([]TaskRun, error) {
	return nil, nil
}

func Test_Worker_OneshotSuccess_TerminalStatusSucceeded(t *testing.T) {
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
	}
	repo := &workerStubRepo{pendingTasks: []*Task{task}}
	aggr := &stubAggr{} // 返空 rows，不影响 executor 流程
	pub := &stubPublisher{}
	executor := NewExecutor(aggr, repo, pub, nil)
	w := NewWorker(repo, executor, "test-worker-1", 1*time.Millisecond, nil)

	ran := w.tryRunOne(context.Background())
	assert.True(t, ran)

	// 至少 1 个 running progress + 1 个 succeeded 终态
	statuses := []Status{}
	for _, u := range repo.statusUpdates {
		statuses = append(statuses, u.status)
	}
	require.Contains(t, statuses, StatusSucceeded, "oneshot should reach succeeded")

	// completed 事件已发
	subjects := []string{}
	for _, e := range pub.events {
		subjects = append(subjects, e.subject)
	}
	assert.Contains(t, subjects, SubjectRealtimeCompleted)
	for _, published := range pub.events {
		if published.subject != SubjectRealtimeCompleted {
			continue
		}
		payload, ok := published.payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, task.ID.String(), payload["task_id"])
		assert.Equal(t, string(StatusSucceeded), payload["status"])
		assert.Equal(t, 0, payload["rows_total"])
		assert.NotContains(t, payload, "error")
	}
}

func Test_Worker_ContinuousSuccess_TerminalStatusScheduled(t *testing.T) {
	cron := "5 * * * *"
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeContinuous,
		CronExpr:      &cron,
		Granularities: []string{"hourly"},
	}
	repo := &workerStubRepo{pendingTasks: []*Task{task}}
	executor := NewExecutor(&stubAggr{}, repo, nil, nil)
	w := NewWorker(repo, executor, "test-worker-1", 1*time.Millisecond, nil)

	ran := w.tryRunOne(context.Background())
	assert.True(t, ran)

	last := repo.statusUpdates[len(repo.statusUpdates)-1].status
	assert.Equal(t, StatusScheduled, last, "continuous should end in scheduled")
}

func Test_Worker_DoesNotPublishCompletedWhenTerminalStatusPersistenceFails(t *testing.T) {
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
	}
	repo := &workerStubRepo{
		pendingTasks: []*Task{task},
		updateStatusErrByStatus: map[Status]error{
			StatusSucceeded: errors.New("database unavailable"),
		},
	}
	pub := &stubPublisher{}
	executor := NewExecutor(&stubAggr{}, repo, pub, nil)
	w := NewWorker(repo, executor, "test-worker-1", time.Millisecond, nil)

	require.True(t, w.tryRunOne(context.Background()))

	var subjects []string
	for _, published := range pub.events {
		subjects = append(subjects, published.subject)
		assert.NotEqual(t, SubjectRealtimeCompleted, published.subject,
			"unpersisted terminal status must not be broadcast")
	}
	assert.Contains(t, subjects, SubjectRealtimeProgress,
		"the earlier persisted progress update should still be broadcast")
}

func Test_Worker_PublisherFailureDoesNotFailPersistedTerminalStatus(t *testing.T) {
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
	}
	repo := &workerStubRepo{pendingTasks: []*Task{task}}
	pub := &stubPublisher{err: errors.New("realtime channel unavailable")}
	executor := NewExecutor(&stubAggr{}, repo, pub, nil)
	w := NewWorker(repo, executor, "test-worker-1", time.Millisecond, nil)

	require.True(t, w.tryRunOne(context.Background()))

	var statuses []Status
	for _, update := range repo.statusUpdates {
		statuses = append(statuses, update.status)
	}
	assert.Contains(t, statuses, StatusSucceeded)
}

// T-0186：成功路径写出一行 run 记录（status=running 入库 + succeeded 终态 + rows_total + window/粒度/维度快照）。
func Test_Worker_RunRecord_SuccessPath(t *testing.T) {
	ws := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	we := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
		Dimension:     DimensionDevice,
		WindowStart:   ws,
		WindowEnd:     we,
	}
	// aggr 返回一行 → executor 落库 1 行
	aggr := &stubAggr{rowsByGran: map[metrics.Granularity][]aggregator.Row{
		metrics.Granularity("hourly"): {{MetricPath: "L.RRC.Succ", MetricType: "counter", MetricValue: 1}},
	}}
	repo := &workerStubRepo{pendingTasks: []*Task{task}, nextSeq: 1}
	executor := NewExecutor(aggr, repo, nil, nil)
	w := NewWorker(repo, executor, "test-worker-1", 1*time.Millisecond, nil)

	require.True(t, w.tryRunOne(context.Background()))

	// 跑前写出一行 run（running 快照，run_seq=1，窗口/粒度/维度对齐）
	require.Len(t, repo.insertedRuns, 1)
	run := repo.insertedRuns[0]
	assert.Equal(t, task.ID, run.TaskID)
	assert.Equal(t, 1, run.RunSeq)
	assert.Equal(t, StatusRunning, run.Status)
	assert.Equal(t, "hourly", run.Granularity)
	assert.Equal(t, string(DimensionDevice), run.Dimension)
	require.NotNil(t, run.WindowStart)
	require.NotNil(t, run.WindowEnd)
	assert.True(t, run.WindowStart.Equal(ws))
	assert.True(t, run.WindowEnd.Equal(we))
	require.NotNil(t, run.QueuedAt)

	// 跑后切终态 succeeded，rows_total=1，error 空
	require.Len(t, repo.finishedRuns, 1)
	fin := repo.finishedRuns[0]
	assert.Equal(t, run.ID, fin.runID)
	assert.Equal(t, StatusSucceeded, fin.status)
	assert.Equal(t, 1, fin.rowsTotal)
	assert.Empty(t, fin.errMsg)
}

// T-0186：失败路径也写出一行 run 记录，终态 failed 且 error 非空。
func Test_Worker_RunRecord_FailurePath(t *testing.T) {
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
		Dimension:     DimensionDevice,
		// 非零窗口 → 明确 oneshot（#528 P2 后零窗口会被当持续型走水位口径）。
		WindowStart: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	// aggr.Query 报错 → ExecuteOneshot 返错 → run 终态 failed
	aggr := &failingAggr{err: errors.New("aggregator boom")}
	repo := &workerStubRepo{pendingTasks: []*Task{task}, nextSeq: 1}
	executor := NewExecutor(aggr, repo, nil, nil)
	w := NewWorker(repo, executor, "test-worker-1", 1*time.Millisecond, nil)

	require.True(t, w.tryRunOne(context.Background()))

	require.Len(t, repo.insertedRuns, 1)
	assert.Equal(t, StatusRunning, repo.insertedRuns[0].Status)

	require.Len(t, repo.finishedRuns, 1)
	fin := repo.finishedRuns[0]
	assert.Equal(t, StatusFailed, fin.status)
	assert.NotEmpty(t, fin.errMsg, "失败路径 error 必须非空")
}

// T-0186：continuous 任务整体终态 scheduled，但本次 run 记 succeeded（单次执行视角）。
func Test_Worker_RunRecord_ContinuousRunSucceeded(t *testing.T) {
	cron := "5 * * * *"
	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeContinuous,
		CronExpr:      &cron,
		Granularities: []string{"hourly"},
		Dimension:     DimensionNetwork,
	}
	repo := &workerStubRepo{pendingTasks: []*Task{task}, nextSeq: 1}
	executor := NewExecutor(&stubAggr{}, repo, nil, nil)
	w := NewWorker(repo, executor, "test-worker-1", 1*time.Millisecond, nil)

	require.True(t, w.tryRunOne(context.Background()))

	// 任务终态 scheduled
	last := repo.statusUpdates[len(repo.statusUpdates)-1].status
	assert.Equal(t, StatusScheduled, last)
	// run 终态 succeeded（不是 scheduled）
	require.Len(t, repo.finishedRuns, 1)
	assert.Equal(t, StatusSucceeded, repo.finishedRuns[0].status)
}

// failingAggr 让 ExecuteOneshot 走失败路径。
type failingAggr struct{ err error }

func (a *failingAggr) Query(context.Context, aggregator.QueryRequest) ([]aggregator.Row, error) {
	return nil, a.err
}

func Test_Worker_NoPendingReturnsFalse(t *testing.T) {
	repo := &workerStubRepo{}
	executor := NewExecutor(&stubAggr{}, repo, nil, nil)
	w := NewWorker(repo, executor, "test-worker-1", 1*time.Millisecond, nil)

	ran := w.tryRunOne(context.Background())
	assert.False(t, ran)
}

// ---------------------------------------------------------------------------
// ContinuousScheduler.sweepOnce：scheduled→pending 切换
// ---------------------------------------------------------------------------

type schedRepoStub struct {
	tasks      []ContinuousTask
	markedIDs  []ContinuousTaskID
	markedFire []time.Time
	listErr    error
}

func (s *schedRepoStub) ListReschedulable(context.Context) ([]ContinuousTask, error) {
	return s.tasks, s.listErr
}
func (s *schedRepoStub) MarkPending(_ context.Context, id ContinuousTaskID, fireAt time.Time) error {
	s.markedIDs = append(s.markedIDs, id)
	s.markedFire = append(s.markedFire, fireAt)
	return nil
}

func Test_ContinuousScheduler_MarksDueTasksPending(t *testing.T) {
	// 一个 cron '* * * * *' 每分钟跑；last_fire_at 设 1 小时前 → 早该跑了
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "* * * * *", LastFireAt: time.Now().Add(-time.Hour)},
		},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	s.sweepOnce(context.Background(), time.Now())

	require.Len(t, repo.markedIDs, 1)
	assert.Equal(t, ContinuousTaskID("task-1"), repo.markedIDs[0])
}

func Test_ContinuousScheduler_DoesNotMarkBeforeDue(t *testing.T) {
	// last_fire_at 刚刚，每小时 cron → 还没到下一次
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "0 * * * *", LastFireAt: time.Now()},
		},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	s.sweepOnce(context.Background(), time.Now())

	assert.Empty(t, repo.markedIDs)
}

func Test_ContinuousScheduler_InvalidCronSkipped(t *testing.T) {
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "garbage", LastFireAt: time.Now().Add(-time.Hour)},
		},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	s.sweepOnce(context.Background(), time.Now())
	assert.Empty(t, repo.markedIDs)
}

func Test_ContinuousScheduler_ListErrorTolerated(t *testing.T) {
	repo := &schedRepoStub{listErr: errors.New("db down")}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	// 不 panic
	s.sweepOnce(context.Background(), time.Now())
}

// G7-Gap-9: lossless catchup
// 每小时 cron '0 * * * *'，last_fire_at 设 4 小时前 → 应识别漏 4 个窗口，
// 且 MarkPending 只推进 1 格（fireAt = LastFireAt 之后第一个 cron 触发时刻）。
func Test_ContinuousScheduler_LosslessCatchup_AdvancesOneWindowPerSweep(t *testing.T) {
	now := time.Date(2026, 5, 23, 4, 30, 0, 0, time.UTC)
	// 头一晚 23:30，cron "0 * * * *" 漏了 00:00 / 01:00 / 02:00 / 03:00 / 04:00 共 5 个窗口
	lastFire := time.Date(2026, 5, 22, 23, 30, 0, 0, time.UTC)
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "0 * * * *", LastFireAt: lastFire},
		},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	s.sweepOnce(context.Background(), now)

	require.Len(t, repo.markedIDs, 1)
	require.Len(t, repo.markedFire, 1)
	// 推进到 LastFireAt 后第一个 cron 触发：00:00 May 23（不是跳到 04:00 当前小时）
	expected := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, repo.markedFire[0],
		"fire_at 必须推进 1 格（不是直接跳到 now），让后续 sweep 继续追平漏桶")
}

// ---------------------------------------------------------------------------
// #528 P3：调度器水位 gate —— 追平上界由真实水位决定（不越水位、不空磨史前格、不读 #479 半成品格）
// ---------------------------------------------------------------------------

// gateStub 模拟「上游完成水位」。bucket=该 (粒度,层级) 当前水位桶起点；ok=false 表示无水位。
type gateStub struct {
	bucket time.Time
	ok     bool
	calls  int
}

func (g *gateStub) CompletedBucketStart(_ context.Context, _ metrics.Granularity, _ aggregator.WatermarkLevel) (time.Time, bool) {
	g.calls++
	return g.bucket, g.ok
}

// 追平上界 == 水位：下游落后多格、水位只到 T，调度器追到 T 即停，不放行 > T 的格。
// 用每小时 cron，last_fire_at 在 4 小时前，墙钟 now 已过去很久（墙钟不挡）；水位停在 T=02:00。
// fire 02:00 处理桶 [01:00,02:00) 起点 01:00 ≤ 水位 02:00 → 放行；
// （下一 sweep 会 fire 03:00 处理桶 [02:00,03:00) 起点 02:00 ≤ 水位 02:00 → 仍放行；
//
//	再下一 sweep fire 04:00 处理桶 [03:00,04:00) 起点 03:00 > 水位 02:00 → 挡住，即追平终点 = 水位 T）。
func Test_ContinuousScheduler_P3_CatchupBoundedByWatermark(t *testing.T) {
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC) // 墙钟远超水位，确保上界由水位卡而非墙钟
	watermarkT := time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC)

	// 逐 sweep 推进 last_fire_at，断言放行的最后一格 fire 对应的处理桶 == 水位 T。
	lastFire := time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC)
	var lastAllowedFire time.Time
	for i := 0; i < 12; i++ {
		repo := &schedRepoStub{
			tasks: []ContinuousTask{{
				ID: "task-1", CronExpr: "0 * * * *", LastFireAt: lastFire,
				Granularities: []string{string(metrics.GranularityHourly)}, Dimension: DimensionNetwork,
			}},
		}
		gate := &gateStub{bucket: watermarkT, ok: true}
		s := NewContinuousScheduler(repo, time.Hour, nil).SetWatermarkGate(gate)
		s.sweepOnce(context.Background(), now)
		if len(repo.markedIDs) == 0 {
			break // 被水位挡住，追平停止
		}
		lastAllowedFire = repo.markedFire[0]
		lastFire = repo.markedFire[0] // 模拟 worker 跑完后游标推进
	}
	// 最后放行的 fire = 03:00（处理桶 [02:00,03:00) 起点 02:00 == 水位 T）；fire 04:00 起被挡。
	require.False(t, lastAllowedFire.IsZero(), "至少应放行若干格")
	wantLastFire := time.Date(2026, 5, 23, 3, 0, 0, 0, time.UTC)
	assert.Equal(t, wantLastFire, lastAllowedFire,
		"追平终点 fire 对应的处理桶起点必须 == 水位 T，绝不越过水位")
	// 处理桶起点 = lastAllowedFire 前一格 = 02:00 == 水位 T → 相差 0 格。
	processedBucket := truncateBucketStart(metrics.GranularityHourly, lastAllowedFire.Add(-time.Nanosecond), time.UTC)
	assert.Equal(t, watermarkT, processedBucket, "追平终点桶 == 水位 T（相差 0 格）")
}

// #479 回归：上一格刚结束、上游卷数据未跑完（水位未推进到该格）时运行下游 → 不放行（不读半成品格）。
// cron fire 当前小时（处理刚结束的上一格），但水位还停在更早的格 → gate 挡住，不 MarkPending。
func Test_ContinuousScheduler_P3_479Regression_NoDirtyRead(t *testing.T) {
	// fire 03:00 想处理桶 [02:00,03:00)，但上游水位还停在 01:00（[01:00,02:00) 刚卷完，
	// [02:00,03:00) 的卷数据还没跑完）→ 处理桶起点 02:00 > 水位 01:00 → 挡住。
	now := time.Date(2026, 5, 23, 3, 0, 30, 0, time.UTC) // 03:00 刚过（上一格 02:00-03:00 刚结束）
	lastFire := time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC)
	staleWatermark := time.Date(2026, 5, 23, 1, 0, 0, 0, time.UTC) // 水位未推进到刚结束的格

	repo := &schedRepoStub{
		tasks: []ContinuousTask{{
			ID: "task-1", CronExpr: "0 * * * *", LastFireAt: lastFire,
			Granularities: []string{string(metrics.GranularityHourly)}, Dimension: DimensionNetwork,
		}},
	}
	gate := &gateStub{bucket: staleWatermark, ok: true}
	s := NewContinuousScheduler(repo, time.Hour, nil).SetWatermarkGate(gate)
	s.sweepOnce(context.Background(), now)

	assert.Empty(t, repo.markedIDs, "上游卷数据未跑完（水位未到该格）→ 调度器不放行，不读半成品格（clean）")
	assert.Equal(t, 1, gate.calls, "应查过水位一次")
}

// #479 续：等水位推进到该格后，同一 fire 即放行（半成品变成成品才读）。
func Test_ContinuousScheduler_P3_479Regression_FiresAfterWatermarkAdvances(t *testing.T) {
	now := time.Date(2026, 5, 23, 3, 0, 30, 0, time.UTC)
	lastFire := time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC)
	freshWatermark := time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC) // 水位已推进到 [02:00,03:00) 桶

	repo := &schedRepoStub{
		tasks: []ContinuousTask{{
			ID: "task-1", CronExpr: "0 * * * *", LastFireAt: lastFire,
			Granularities: []string{string(metrics.GranularityHourly)}, Dimension: DimensionNetwork,
		}},
	}
	gate := &gateStub{bucket: freshWatermark, ok: true}
	s := NewContinuousScheduler(repo, time.Hour, nil).SetWatermarkGate(gate)
	s.sweepOnce(context.Background(), now)

	require.Len(t, repo.markedIDs, 1, "水位推进到该格后即放行")
	assert.Equal(t, time.Date(2026, 5, 23, 3, 0, 0, 0, time.UTC), repo.markedFire[0])
}

// #124 回归：weekly continuous 在周一 00:15 延迟触发时，处理的是刚结束的上一周。
// 测试环境口径：2026-07-20 00:15 应消费 weekly 水位 2026-07-13 00:00
// （即 [2026-07-13, 2026-07-20)），不能误判成等待 2026-07-20 开始的本周。
func Test_ContinuousScheduler_P3_WeeklyDelayedCronConsumesPreviousWeek(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	repo := &schedRepoStub{
		tasks: []ContinuousTask{{
			ID:            "task-124",
			CronExpr:      "15 0 * * 1",
			LastFireAt:    time.Date(2026, 7, 19, 16, 31, 39, 0, loc),
			Granularities: []string{string(metrics.GranularityWeekly)},
			Dimension:     DimensionNetwork,
		}},
	}
	gate := &gateStub{bucket: time.Date(2026, 7, 13, 0, 0, 0, 0, loc), ok: true}
	s := NewContinuousScheduler(repo, time.Hour, nil).
		SetWatermarkGate(gate).
		SetLocationFunc(func() *time.Location { return loc })

	s.sweepOnce(context.Background(), time.Date(2026, 7, 20, 0, 16, 0, 0, loc))

	require.Len(t, repo.markedIDs, 1, "weekly 水位已到上一周桶，应放行本次周一 00:15 调度")
	assert.Equal(t, ContinuousTaskID("task-124"), repo.markedIDs[0])
	assert.Equal(t, time.Date(2026, 7, 20, 0, 15, 0, 0, loc), repo.markedFire[0])
}

func Test_ContinuousScheduler_P3_DailyAndMonthlyDelayedCronConsumePreviousBucket(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	cases := []struct {
		name        string
		cron        string
		granularity metrics.Granularity
		lastFire    time.Time
		now         time.Time
		watermark   time.Time
		wantFire    time.Time
	}{
		{
			name:        "daily 00:10 consumes previous day",
			cron:        "10 0 * * *",
			granularity: metrics.GranularityDaily,
			lastFire:    time.Date(2026, 7, 19, 0, 10, 0, 0, loc),
			now:         time.Date(2026, 7, 20, 0, 11, 0, 0, loc),
			watermark:   time.Date(2026, 7, 19, 0, 0, 0, 0, loc),
			wantFire:    time.Date(2026, 7, 20, 0, 10, 0, 0, loc),
		},
		{
			name:        "monthly 00:20 consumes previous month",
			cron:        "20 0 1 * *",
			granularity: metrics.GranularityMonthly,
			lastFire:    time.Date(2026, 7, 1, 0, 20, 0, 0, loc),
			now:         time.Date(2026, 8, 1, 0, 21, 0, 0, loc),
			watermark:   time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantFire:    time.Date(2026, 8, 1, 0, 20, 0, 0, loc),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &schedRepoStub{
				tasks: []ContinuousTask{{
					ID:            "task-delayed",
					CronExpr:      tc.cron,
					LastFireAt:    tc.lastFire,
					Granularities: []string{string(tc.granularity)},
					Dimension:     DimensionNetwork,
				}},
			}
			gate := &gateStub{bucket: tc.watermark, ok: true}
			s := NewContinuousScheduler(repo, time.Hour, nil).
				SetWatermarkGate(gate).
				SetLocationFunc(func() *time.Location { return loc })

			s.sweepOnce(context.Background(), tc.now)

			require.Len(t, repo.markedIDs, 1)
			assert.Equal(t, tc.wantFire, repo.markedFire[0])
		})
	}
}

// 无水位记录（上游一格都没卷完 / 全新环境）→ gate 返回 ok=false → 不放行（不空磨史前格）。
func Test_ContinuousScheduler_P3_NoWatermark_SkipsCatchup(t *testing.T) {
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	lastFire := time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC) // 落后很多格

	repo := &schedRepoStub{
		tasks: []ContinuousTask{{
			ID: "task-1", CronExpr: "0 * * * *", LastFireAt: lastFire,
			Granularities: []string{string(metrics.GranularityHourly)}, Dimension: DimensionNetwork,
		}},
	}
	gate := &gateStub{ok: false} // 上游尚未卷完任何格
	s := NewContinuousScheduler(repo, time.Hour, nil).SetWatermarkGate(gate)
	s.sweepOnce(context.Background(), now)

	assert.Empty(t, repo.markedIDs, "无水位 → 全新环境不空磨史前空格")
}

// 未注入 gate（nil）→ 退化为旧墙钟语义，不回归（已运行任务在 gate 接通前仍可调度）。
func Test_ContinuousScheduler_P3_NilGate_FallbackToWallClock(t *testing.T) {
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	lastFire := time.Date(2026, 5, 23, 8, 30, 0, 0, time.UTC)
	repo := &schedRepoStub{
		tasks: []ContinuousTask{{
			ID: "task-1", CronExpr: "0 * * * *", LastFireAt: lastFire,
			Granularities: []string{string(metrics.GranularityHourly)}, Dimension: DimensionNetwork,
		}},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil) // 不注入 gate
	s.sweepOnce(context.Background(), now)
	require.Len(t, repo.markedIDs, 1, "nil gate 退化墙钟，行为不回归")
	assert.Equal(t, time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC), repo.markedFire[0])
}
