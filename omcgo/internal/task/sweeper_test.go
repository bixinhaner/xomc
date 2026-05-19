package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeLister 返回预设的过期候选任务。
type fakeLister struct {
	mu        sync.Mutex
	candidates []*Task
	listErr    error
	callCount  int
	lastLimit  int
	lastNow    time.Time
}

func (f *fakeLister) ListExpiredCandidates(ctx context.Context, now time.Time, limit int) ([]*Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	f.lastLimit = limit
	f.lastNow = now
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.candidates, nil
}

// fakeExpirer 记录每次 ExpireTask 调用，可注入失败响应。
type fakeExpirer struct {
	mu        sync.Mutex
	expired   []*Task
	failNext  bool
	failError error
}

func (f *fakeExpirer) ExpireTask(ctx context.Context, t *Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return f.failError
	}
	f.expired = append(f.expired, t)
	return nil
}

func (f *fakeExpirer) expiredIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.expired))
	for _, t := range f.expired {
		ids = append(ids, t.ID)
	}
	return ids
}

func mkTask(id, sn string) *Task {
	expiresAt := time.Now().Add(-1 * time.Minute) // 1 分钟前过期
	return &Task{
		ID:        id,
		DeviceSN:  sn,
		Method:    "SetParameterValues",
		Status:    TaskStatusPending,
		ExpiresAt: &expiresAt,
	}
}

func Test_SweepOnce_NoCandidates_ReturnsZero(t *testing.T) {
	lister := &fakeLister{candidates: nil}
	expirer := &fakeExpirer{}
	sw := NewExpiredSweeper(lister, expirer, 0, 0, zap.NewNop())

	processed, err := sw.SweepOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, processed)
	assert.Equal(t, 1, lister.callCount)
}

func Test_SweepOnce_AllSucceed(t *testing.T) {
	lister := &fakeLister{candidates: []*Task{mkTask("t1", "SN1"), mkTask("t2", "SN2"), mkTask("t3", "SN3")}}
	expirer := &fakeExpirer{}
	sw := NewExpiredSweeper(lister, expirer, 0, 0, zap.NewNop())

	processed, err := sw.SweepOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, processed)
	assert.ElementsMatch(t, []string{"t1", "t2", "t3"}, expirer.expiredIDs())
}

func Test_SweepOnce_SingleFailureSkippedContinues(t *testing.T) {
	lister := &fakeLister{candidates: []*Task{mkTask("t1", "SN1"), mkTask("t2", "SN2")}}
	expirer := &fakeExpirer{failNext: true, failError: errors.New("db conflict")}
	sw := NewExpiredSweeper(lister, expirer, 0, 0, zap.NewNop())

	processed, err := sw.SweepOnce(context.Background())
	require.NoError(t, err, "lister 成功，单条失败不应是致命错误")
	assert.Equal(t, 1, processed, "失败一条只成功一条")
	assert.Equal(t, []string{"t2"}, expirer.expiredIDs(), "第二条仍被处理")
}

func Test_SweepOnce_ListerErrorReturnsError(t *testing.T) {
	lister := &fakeLister{listErr: errors.New("pg connection refused")}
	expirer := &fakeExpirer{}
	sw := NewExpiredSweeper(lister, expirer, 0, 0, zap.NewNop())

	processed, err := sw.SweepOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, processed)
	assert.Empty(t, expirer.expiredIDs())
}

func Test_SweepOnce_PassesBatchSizeAndNow(t *testing.T) {
	lister := &fakeLister{candidates: nil}
	expirer := &fakeExpirer{}
	sw := NewExpiredSweeper(lister, expirer, time.Second, 50, zap.NewNop())

	before := time.Now()
	_, err := sw.SweepOnce(context.Background())
	require.NoError(t, err)
	after := time.Now()

	assert.Equal(t, 50, lister.lastLimit)
	assert.True(t, !lister.lastNow.Before(before) && !lister.lastNow.After(after),
		"now 应介于 SweepOnce 调用前后")
}

func Test_NewExpiredSweeper_DefaultsApplied(t *testing.T) {
	sw := NewExpiredSweeper(&fakeLister{}, &fakeExpirer{}, 0, 0, nil)
	assert.Equal(t, 10*time.Second, sw.interval)
	assert.Equal(t, 100, sw.batchSize)
	assert.NotNil(t, sw.logger)
}

func Test_Run_StopsOnContextCancel(t *testing.T) {
	lister := &fakeLister{candidates: nil}
	expirer := &fakeExpirer{}
	// 极短 interval 确保至少跑过一轮
	sw := NewExpiredSweeper(lister, expirer, 20*time.Millisecond, 10, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sw.Run(ctx)
		close(done)
	}()

	// 等至少一轮 sweep 触发
	time.Sleep(60 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// good
	case <-time.After(time.Second):
		t.Fatal("Run 未在 ctx 取消后退出")
	}

	assert.GreaterOrEqual(t, lister.callCount, 1, "应至少跑过一轮扫描")
}
