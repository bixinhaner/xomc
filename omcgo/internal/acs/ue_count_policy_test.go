package acs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

type stubUECountPathResolver struct {
	paths []string
	err   error
	sn    string
}

func (s *stubUECountPathResolver) ResolveUECountPaths(_ context.Context, deviceSN string) ([]string, error) {
	s.sn = deviceSN
	return s.paths, s.err
}

type stubUECountTaskService struct {
	mu        sync.Mutex
	open      *task.Task
	openErr   error
	createErr error
	created   []*task.CreateTaskRequest
	block     bool
}

func (s *stubUECountTaskService) LatestOpenTaskByDeviceAndMethod(
	ctx context.Context,
	_, _, _ string,
) (*task.Task, error) {
	if s.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.open, s.openErr
}

func (s *stubUECountTaskService) CreateTask(
	_ context.Context,
	req *task.CreateTaskRequest,
) (*task.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.created = append(s.created, req)
	return &task.Task{ID: "ue-count-task"}, nil
}

func (s *stubUECountTaskService) createdCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.created)
}

type stubUECountProbeGate struct {
	acquired bool
	err      error
}

func (g stubUECountProbeGate) Acquire(context.Context, string) (bool, error) {
	return g.acquired, g.err
}

func TestUECountPolicy_ShouldTriggerOnlyPeriodicInform(t *testing.T) {
	policy := NewUECountPolicy(
		&stubUECountPathResolver{},
		&stubUECountTaskService{},
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)

	assert.True(t, policy.ShouldTrigger([]string{tr069.EventPeriodic}))
	assert.True(t, policy.ShouldTrigger([]string{tr069.EventBoot, tr069.EventPeriodic}))
	assert.False(t, policy.ShouldTrigger([]string{tr069.EventBoot}))
	assert.False(t, policy.ShouldTrigger(nil))

	var nilPolicy *UECountPolicy
	assert.False(t, nilPolicy.ShouldTrigger([]string{tr069.EventPeriodic}))
}

func TestUECountPolicy_EnqueueCreatesDirectGPVForSupportedPaths(t *testing.T) {
	resolver := &stubUECountPathResolver{paths: []string{
		"Device.DeviceInfo.UE_Count",
		"Device.DeviceInfo.2.UE_Count",
	}}
	tasks := &stubUECountTaskService{}
	policy := NewUECountPolicy(resolver, tasks, stubUECountProbeGate{acquired: true}, zap.NewNop())

	err := policy.Enqueue(context.Background(), "SN-220")

	require.NoError(t, err)
	assert.Equal(t, "SN-220", resolver.sn)
	require.Len(t, tasks.created, 1)
	req := tasks.created[0]
	assert.Equal(t, "SN-220", req.DeviceSN)
	assert.Equal(t, "GetParameterValues", req.Method)
	assert.Equal(t, task.TaskSourceSystem, req.Source)
	assert.Equal(t, ueCountGPVDescription, req.Description)

	var params GPVParams
	require.NoError(t, json.Unmarshal(req.Params, &params))
	assert.Equal(t, []string{
		"Device.DeviceInfo.UE_Count",
		"Device.DeviceInfo.2.UE_Count",
	}, params.Names)
}

func TestUECountPolicy_EnqueueCoalescesOutstandingQuery(t *testing.T) {
	resolver := &stubUECountPathResolver{paths: []string{"Device.DeviceInfo.UE_Count"}}
	tasks := &stubUECountTaskService{
		open: &task.Task{ID: "existing", Status: task.TaskStatusSent},
	}
	policy := NewUECountPolicy(resolver, tasks, stubUECountProbeGate{acquired: true}, zap.NewNop())

	require.NoError(t, policy.Enqueue(context.Background(), "SN-220"))
	assert.Zero(t, tasks.createdCount())
}

func TestUECountPolicy_EnqueueSkipsWhenProductHasNoSupportedPath(t *testing.T) {
	tasks := &stubUECountTaskService{}
	policy := NewUECountPolicy(
		&stubUECountPathResolver{},
		tasks,
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)

	require.NoError(t, policy.Enqueue(context.Background(), "SN-220"))
	assert.Zero(t, tasks.createdCount())
}

func TestUECountPolicy_EnqueueReturnsDependencyErrors(t *testing.T) {
	resolveErr := errors.New("mapping unavailable")
	policy := NewUECountPolicy(
		&stubUECountPathResolver{err: resolveErr},
		&stubUECountTaskService{},
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)
	assert.ErrorIs(t, policy.Enqueue(context.Background(), "SN-220"), resolveErr)

	openErr := errors.New("task lookup unavailable")
	policy = NewUECountPolicy(
		&stubUECountPathResolver{paths: []string{"Device.DeviceInfo.UE_Count"}},
		&stubUECountTaskService{openErr: openErr},
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)
	assert.ErrorIs(t, policy.Enqueue(context.Background(), "SN-220"), openErr)
}

func TestRedisUECountProbeGate_AllowsOnlyOneConcurrentProbe(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	gate := NewRedisUECountProbeGate(client, time.Minute)

	var acquired atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := gate.Acquire(context.Background(), "SN-220")
			assert.NoError(t, err)
			if ok {
				acquired.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), acquired.Load())
}

func TestUECountPolicy_EnqueueHasBoundedDependencyDeadline(t *testing.T) {
	policy := NewUECountPolicy(
		&stubUECountPathResolver{paths: []string{"Device.DeviceInfo.UE_Count"}},
		&stubUECountTaskService{block: true},
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)
	policy.timeout = 20 * time.Millisecond

	started := time.Now()
	err := policy.Enqueue(context.Background(), "SN-220")

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), 250*time.Millisecond)
}
