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
	insertedRuns  []TaskRun           // 跑前 INSERT 的 run（status=running 快照）
	finishedRuns  []finishedRunRecord // 跑后 FinishRun 的终态
	nextSeq       int                 // NextRunSeq 返回值（默认 1）
	insertResults func([]ResultRow) error
}

type finishedRunRecord struct {
	runID     uuid.UUID
	status    Status
	rowsTotal int
	errMsg    string
}

func (s *workerStubRepo) Create(context.Context, CreateRequest) (uuid.UUID, error) { return uuid.Nil, nil }
func (s *workerStubRepo) Get(context.Context, uuid.UUID) (*Task, error)            { return nil, nil }
func (s *workerStubRepo) List(context.Context, ListFilter) ([]Task, error)         { return nil, nil }
func (s *workerStubRepo) Cancel(context.Context, uuid.UUID) error                  { return nil }

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
	return nil
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
	assert.Contains(t, subjects, SubjectCompleted)
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
