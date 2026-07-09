package asyncjob

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ---------- stub Repository（仅本测试用） ----------

type stubRepo struct {
	mu                 sync.Mutex
	insertCount        int
	pending            []*Job
	heartbeatCalls     int
	heartbeatReturnErr error
	markSucceededCalls int
	markFailedCalls    int
	lastResult         json.RawMessage
	lastErrMsg         string
	zombies            []Job
	resetCalls         int
	resetErr           error
	lockErr            error
}

func (s *stubRepo) Insert(_ context.Context, req InsertRequest) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}
	id := uuid.New()
	s.pending = append(s.pending, &Job{
		ID:          id,
		JobType:     req.JobType,
		Status:      StatusPending,
		Attempt:     1,
		ScheduledAt: req.ScheduledAt,
		Payload:     req.Payload,
		MaxAttempts: maxAttempts,
	})
	s.insertCount++
	return id, nil
}

func (s *stubRepo) GetByID(_ context.Context, id uuid.UUID) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.pending {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, ErrNoPendingJob
}

func (s *stubRepo) LockNextPending(_ context.Context, jobType, lockOwner string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockErr != nil {
		return nil, s.lockErr
	}
	for i, j := range s.pending {
		if j.JobType == jobType && j.Status == StatusPending {
			j.Status = StatusRunning
			j.LockOwner = lockOwner
			s.pending = append(s.pending[:i], s.pending[i+1:]...)
			s.pending = append(s.pending, j) // 留在 pending slice 里便于后续 GetByID
			return j, nil
		}
	}
	return nil, ErrNoPendingJob
}

func (s *stubRepo) UpdateHeartbeat(_ context.Context, _ uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.heartbeatCalls++
	return s.heartbeatReturnErr
}

func (s *stubRepo) MarkSucceeded(_ context.Context, _ uuid.UUID, result json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markSucceededCalls++
	s.lastResult = result
	return nil
}

func (s *stubRepo) MarkFailed(_ context.Context, id uuid.UUID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markFailedCalls++
	s.lastErrMsg = errMsg
	for _, j := range s.pending {
		if j.ID != id {
			continue
		}
		if j.Attempt+1 > j.MaxAttempts {
			j.Status = StatusFailed
		} else {
			j.Status = StatusPending
			j.Attempt++
			j.StartedAt = nil
			j.HeartbeatAt = nil
			j.LockOwner = ""
		}
		break
	}
	return nil
}

func (s *stubRepo) ListZombies(_ context.Context, _ time.Duration) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.zombies, nil
}

func (s *stubRepo) ResetZombie(_ context.Context, _ uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetCalls++
	return s.resetErr
}

// ---------- 测试 Runner ----------

type stubJobRunner struct {
	jobType string
	runFn   func(ctx context.Context, job *Job) (json.RawMessage, error)
	calls   atomic.Int32
}

func (r *stubJobRunner) JobType() string { return r.jobType }
func (r *stubJobRunner) Run(ctx context.Context, job *Job) (json.RawMessage, error) {
	r.calls.Add(1)
	return r.runFn(ctx, job)
}

func TestRegistry_Register_OverridesSameType(t *testing.T) {
	repo := &stubRepo{}
	reg := NewRegistry(repo, "test", nil)

	r1 := &stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) { return nil, nil }}
	r2 := &stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) { return nil, nil }}
	reg.Register(r1)
	reg.Register(r2)
	require.Same(t, r2, reg.runners["x"])

	reg.Register(nil) // nil 不应 panic 或加入 map
	require.Len(t, reg.runners, 1)
}

func TestRegistry_RunNext_NoRunnerRegistered(t *testing.T) {
	repo := &stubRepo{}
	reg := NewRegistry(repo, "test", nil)
	didRun, err := reg.RunNext(context.Background(), "unknown")
	require.False(t, didRun)
	require.Error(t, err)
}

func TestRegistry_RunNext_NoPendingJob(t *testing.T) {
	repo := &stubRepo{}
	reg := NewRegistry(repo, "test", nil)
	reg.Register(&stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) { return nil, nil }})

	didRun, err := reg.RunNext(context.Background(), "x")
	require.False(t, didRun)
	require.NoError(t, err)
}

func TestRegistry_RunNext_SucceedsAndMarksResult(t *testing.T) {
	repo := &stubRepo{}
	_, _ = repo.Insert(context.Background(), InsertRequest{JobType: "x", ScheduledAt: time.Now()})

	reg := NewRegistry(repo, "test", nil)
	resultJSON := json.RawMessage(`{"rows":42}`)
	reg.Register(&stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) { return resultJSON, nil }})

	didRun, err := reg.RunNext(context.Background(), "x")
	require.True(t, didRun)
	require.NoError(t, err)
	require.Equal(t, 1, repo.markSucceededCalls)
	require.Equal(t, 0, repo.markFailedCalls)
	require.JSONEq(t, string(resultJSON), string(repo.lastResult))
}

func TestRegistry_RunNext_FailureMarksFailed(t *testing.T) {
	repo := &stubRepo{}
	_, _ = repo.Insert(context.Background(), InsertRequest{JobType: "x", ScheduledAt: time.Now()})

	reg := NewRegistry(repo, "test", nil)
	reg.Register(&stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) {
		return nil, errors.New("intentional failure")
	}})

	didRun, err := reg.RunNext(context.Background(), "x")
	require.True(t, didRun)
	require.NoError(t, err) // 业务失败已写库，infrastructure-level 无 err
	require.Equal(t, 0, repo.markSucceededCalls)
	require.Equal(t, 1, repo.markFailedCalls)
	require.Contains(t, repo.lastErrMsg, "intentional failure")
}

func TestRegistry_RunNext_FailureRetriesUntilMaxAttempts(t *testing.T) {
	repo := &stubRepo{}
	jobID, _ := repo.Insert(context.Background(), InsertRequest{JobType: "x", ScheduledAt: time.Now(), MaxAttempts: 2})

	reg := NewRegistry(repo, "test", nil)
	reg.Register(&stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) {
		return nil, errors.New("retry me")
	}})

	didRun, err := reg.RunNext(context.Background(), "x")
	require.True(t, didRun)
	require.NoError(t, err)
	job, err := repo.GetByID(context.Background(), jobID)
	require.NoError(t, err)
	require.Equal(t, StatusPending, job.Status)
	require.Equal(t, 2, job.Attempt)

	didRun, err = reg.RunNext(context.Background(), "x")
	require.True(t, didRun)
	require.NoError(t, err)
	job, err = repo.GetByID(context.Background(), jobID)
	require.NoError(t, err)
	require.Equal(t, StatusFailed, job.Status)
	require.Equal(t, 2, job.Attempt)
}

func TestRegistry_RunNext_PanicRecoveredAndMarksFailed(t *testing.T) {
	repo := &stubRepo{}
	_, _ = repo.Insert(context.Background(), InsertRequest{JobType: "x", ScheduledAt: time.Now()})

	reg := NewRegistry(repo, "test", nil)
	reg.Register(&stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) {
		panic("kaboom")
	}})

	// 不能让 panic 抛到外层
	didRun, err := reg.RunNext(context.Background(), "x")
	require.True(t, didRun)
	require.NoError(t, err)
	require.Equal(t, 1, repo.markFailedCalls)
	require.Contains(t, repo.lastErrMsg, "kaboom")
}

func TestRegistry_RunNext_LockErrorPropagates(t *testing.T) {
	repo := &stubRepo{lockErr: errors.New("db down")}
	reg := NewRegistry(repo, "test", nil)
	reg.Register(&stubJobRunner{jobType: "x", runFn: func(_ context.Context, _ *Job) (json.RawMessage, error) { return nil, nil }})

	didRun, err := reg.RunNext(context.Background(), "x")
	require.False(t, didRun)
	require.Error(t, err)
}

// ---------- 测试 Sweeper ----------

func TestSweeper_sweepOnce_NoZombies(t *testing.T) {
	repo := &stubRepo{}
	sw := NewSweeper(repo, time.Second, time.Second, nil)
	sw.sweepOnce(context.Background())
	require.Equal(t, 0, repo.resetCalls)
}

func TestSweeper_sweepOnce_ResetsAllZombies(t *testing.T) {
	repo := &stubRepo{
		zombies: []Job{
			{ID: uuid.New(), JobType: "x", Attempt: 1, MaxAttempts: 3},
			{ID: uuid.New(), JobType: "x", Attempt: 0, MaxAttempts: 3},
		},
	}
	sw := NewSweeper(repo, time.Second, time.Second, nil)
	sw.sweepOnce(context.Background())
	require.Equal(t, 2, repo.resetCalls)
}

func TestSweeper_sweepOnce_HandlesAttemptsExhausted(t *testing.T) {
	repo := &stubRepo{
		zombies:  []Job{{ID: uuid.New(), JobType: "x", Attempt: 3, MaxAttempts: 3}},
		resetErr: ErrAttemptsExhausted,
	}
	sw := NewSweeper(repo, time.Second, time.Second, nil)
	sw.sweepOnce(context.Background())
	require.Equal(t, 1, repo.resetCalls) // 不破坏外层
}

func TestNewSweeper_AppliesDefaults(t *testing.T) {
	repo := &stubRepo{}
	sw := NewSweeper(repo, 0, 0, nil)
	require.Equal(t, SweeperInterval, sw.interval)
	require.Equal(t, ZombieThreshold, sw.threshold)
}
