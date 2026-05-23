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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inserted = append(s.inserted, rows...)
	return nil
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
	tasks     []ContinuousTask
	markedIDs []ContinuousTaskID
	listErr   error
}

func (s *schedRepoStub) ListReschedulable(context.Context) ([]ContinuousTask, error) {
	return s.tasks, s.listErr
}
func (s *schedRepoStub) MarkPending(_ context.Context, id ContinuousTaskID) error {
	s.markedIDs = append(s.markedIDs, id)
	return nil
}

func Test_ContinuousScheduler_MarksDueTasksPending(t *testing.T) {
	// 一个 cron '* * * * *' 每分钟跑；updated_at 设 1 小时前 → 早该跑了
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "* * * * *", UpdatedAt: time.Now().Add(-time.Hour)},
		},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	s.sweepOnce(context.Background(), time.Now())

	require.Len(t, repo.markedIDs, 1)
	assert.Equal(t, ContinuousTaskID("task-1"), repo.markedIDs[0])
}

func Test_ContinuousScheduler_DoesNotMarkBeforeDue(t *testing.T) {
	// updated_at 刚刚，每小时 cron → 还没到下一次
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "0 * * * *", UpdatedAt: time.Now()},
		},
	}
	s := NewContinuousScheduler(repo, time.Hour, nil)
	s.sweepOnce(context.Background(), time.Now())

	assert.Empty(t, repo.markedIDs)
}

func Test_ContinuousScheduler_InvalidCronSkipped(t *testing.T) {
	repo := &schedRepoStub{
		tasks: []ContinuousTask{
			{ID: "task-1", CronExpr: "garbage", UpdatedAt: time.Now().Add(-time.Hour)},
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
