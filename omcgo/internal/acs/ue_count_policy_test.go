package acs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	acquired   bool
	err        error
	retryDelay *time.Duration
}

func (g stubUECountProbeGate) Acquire(context.Context, string) (bool, error) {
	return g.acquired, g.err
}

func (g stubUECountProbeGate) RetryAfter(_ context.Context, _ string, delay time.Duration) error {
	if g.retryDelay != nil {
		*g.retryDelay = delay
	}
	return nil
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

func TestUECountPolicy_EnqueueSchedulesShortRetryAfterTaskCreationFailure(t *testing.T) {
	createErr := errors.New("redis queue timeout")
	var retryDelay time.Duration
	policy := NewUECountPolicy(
		&stubUECountPathResolver{paths: []string{"Device.DeviceInfo.UE_Count"}},
		&stubUECountTaskService{createErr: createErr},
		stubUECountProbeGate{acquired: true, retryDelay: &retryDelay},
		zap.NewNop(),
	)

	err := policy.Enqueue(context.Background(), "SN-220")

	require.ErrorIs(t, err, createErr)
	require.Equal(t, 5*time.Minute, retryDelay)
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

func TestRedisUECountProbeGate_ColdStartSpreadsDevicesAcrossCadence(t *testing.T) {
	server := miniredis.RunT(t)
	baseTime := time.Unix(1_800_000_000, 0)
	server.SetTime(baseTime)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	gate := NewRedisUECountProbeGate(client, time.Hour)

	const deviceCount = 120
	admitted := make(map[string]int, deviceCount)
	slotCounts := make([]int, 12)
	for slot := range slotCounts {
		server.SetTime(baseTime.Add(time.Duration(slot) * 5 * time.Minute))
		for i := range deviceCount {
			deviceSN := fmt.Sprintf("SN-%03d", i)
			ok, err := gate.Acquire(context.Background(), deviceSN)
			require.NoError(t, err)
			if ok {
				admitted[deviceSN]++
				slotCounts[slot]++
			}
		}
	}

	require.Len(t, admitted, deviceCount, "one cadence must cover every device")
	for deviceSN, count := range admitted {
		require.Equal(t, 1, count, "device %s must be admitted exactly once per cadence", deviceSN)
	}
	for slot, count := range slotCounts {
		require.LessOrEqual(t, count, 20,
			"cold start slot %d contains a synchronized task wave: %v", slot, slotCounts)
	}
}

func TestRedisUECountProbeGate_RetryAfterShortensFailedProbeDelay(t *testing.T) {
	server := miniredis.RunT(t)
	baseTime := time.Unix(1_800_000_000, 0)
	server.SetTime(baseTime)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	gate := NewRedisUECountProbeGate(client, time.Hour)

	var admittedSN string
	for i := range 120 {
		deviceSN := fmt.Sprintf("SN-%03d", i)
		ok, err := gate.Acquire(context.Background(), deviceSN)
		require.NoError(t, err)
		if ok {
			admittedSN = deviceSN
			break
		}
	}
	require.NotEmpty(t, admittedSN, "fixture must include a cold-start slot-zero device")

	retryGate, ok := gate.(interface {
		RetryAfter(context.Context, string, time.Duration) error
	})
	require.True(t, ok, "Redis UE count gate must support scheduling a short retry")

	server.SetTime(baseTime.Add(time.Minute))
	require.NoError(t, retryGate.RetryAfter(context.Background(), admittedSN, 5*time.Minute))
	admitted, err := gate.Acquire(context.Background(), admittedSN)
	require.NoError(t, err)
	require.False(t, admitted, "retry must not be admitted before its delay")

	server.SetTime(baseTime.Add(6 * time.Minute))
	admitted, err = gate.Acquire(context.Background(), admittedSN)
	require.NoError(t, err)
	require.True(t, admitted, "failed probe must be admitted after the short retry delay")
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
