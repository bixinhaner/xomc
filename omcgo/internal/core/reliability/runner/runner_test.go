package runner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
)

// =============================================================
// Test fixtures
// =============================================================

// memDLQ 是 dlq.Repository 的 in-memory 实现，仅供 runner 测试使用。
type memDLQ struct {
	mu      sync.Mutex
	entries []*dlq.DeadLetter
	failOn  error
}

func newMemDLQ() *memDLQ { return &memDLQ{} }

func (r *memDLQ) Insert(ctx context.Context, e *dlq.DeadLetter) error {
	if r.failOn != nil {
		return r.failOn
	}
	if e == nil {
		return errors.New("nil entry")
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *e
	r.entries = append(r.entries, &cp)
	return nil
}

func (r *memDLQ) List(ctx context.Context, _ dlq.Filter) (*model.ListResponse[dlq.DeadLetter], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]dlq.DeadLetter, 0, len(r.entries))
	for _, e := range r.entries {
		all = append(all, *e)
	}
	return model.NewListResponse(all, int64(len(all)), 1, 20), nil
}

func (r *memDLQ) Get(ctx context.Context, id uuid.UUID) (*dlq.DeadLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.ID == id {
			cp := *e
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *memDLQ) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.entries[:0]
	for _, e := range r.entries {
		if e.ID != id {
			out = append(out, e)
		}
	}
	r.entries = out
	return nil
}

func (r *memDLQ) Count(ctx context.Context, mod string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mod == "" {
		return int64(len(r.entries)), nil
	}
	var n int64
	for _, e := range r.entries {
		if e.SourceModule == mod {
			n++
		}
	}
	return n, nil
}

// stubPublisher records publishes for assertions.
type stubPublisher struct {
	mu       sync.Mutex
	calls    []event.Event
	failNext bool
}

func (p *stubPublisher) Publish(ctx context.Context, subject string, evt event.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failNext {
		p.failNext = false
		return errors.New("publish failed")
	}
	p.calls = append(p.calls, evt)
	return nil
}

func newRunner(t *testing.T, dlqRepo dlq.Repository, pub Publisher, metrics *Metrics) *Runner {
	t.Helper()
	return NewRunner(
		"pm",
		reliability.RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond},
		dlqRepo,
		pub,
		metrics,
		zap.NewNop(),
	)
}

func newMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	return NewMetrics(reg), reg
}

func counterValue(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelsMatch(m.GetLabel(), labels) {
				if m.Counter != nil {
					return m.Counter.GetValue()
				}
				if m.Gauge != nil {
					return m.Gauge.GetValue()
				}
			}
		}
	}
	return 0
}

func labelsMatch(got []*dto.LabelPair, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, lp := range got {
		if want[lp.GetName()] != lp.GetValue() {
			return false
		}
	}
	return true
}

// =============================================================
// Tests — V1..V7 GWT 矩阵
// =============================================================

// V1: Wrap_HandlerSucceedsFirstTry — handler 第一次就成功，无 DLQ
func TestWrap_HandlerSucceedsFirstTry(t *testing.T) {
	repo := newMemDLQ()
	metrics, reg := newMetrics(t)
	r := newRunner(t, repo, nil, metrics)

	called := atomic.Int32{}
	wrapped := r.Wrap("pm.file.received", func(ctx context.Context, e event.Event) error {
		called.Add(1)
		return nil
	})

	require.NoError(t, wrapped(context.Background(), event.Event{Subject: "pm.file.received"}))
	assert.Equal(t, int32(1), called.Load())
	assert.Empty(t, repo.entries, "no DLQ entry on first-try success")
	assert.Equal(t, 1.0, counterValue(t, reg, "worker_retry_attempts_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ResultSuccess,
	}))
}

// V1: Wrap_TransientFailThenSuccess — 短暂失败 + 重试成功
func TestWrap_TransientFailThenSuccess(t *testing.T) {
	repo := newMemDLQ()
	metrics, reg := newMetrics(t)
	r := newRunner(t, repo, nil, metrics)

	attempts := atomic.Int32{}
	wrapped := r.Wrap("pm.file.received", func(ctx context.Context, e event.Event) error {
		n := attempts.Add(1)
		if n < 2 {
			return errors.New("transient")
		}
		return nil
	})

	require.NoError(t, wrapped(context.Background(), event.Event{Subject: "pm.file.received"}))
	assert.Equal(t, int32(2), attempts.Load())
	assert.Empty(t, repo.entries, "transient retry-then-success should not write DLQ")
	assert.Equal(t, 1.0, counterValue(t, reg, "worker_retry_attempts_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ResultSuccess,
	}))
}

// V2: Wrap_AllAttemptsFail_DLQAndMetrics — 全失败入 DLQ
func TestWrap_AllAttemptsFail_DLQAndMetrics(t *testing.T) {
	repo := newMemDLQ()
	metrics, reg := newMetrics(t)
	r := newRunner(t, repo, nil, metrics)

	wrapped := r.Wrap("pm.file.received", func(ctx context.Context, e event.Event) error {
		return errors.New("permanent")
	})

	err := wrapped(context.Background(), event.Event{
		Subject: "pm.file.received",
		Payload: []byte(`{"x":1}`),
	})
	require.Error(t, err)

	require.Len(t, repo.entries, 1, "expected exactly one DLQ row")
	assert.Equal(t, "pm", repo.entries[0].SourceModule)
	assert.Equal(t, "pm.file.received", repo.entries[0].SourceSubject)
	assert.Equal(t, []byte(`{"x":1}`), repo.entries[0].Payload)
	assert.Equal(t, 3, repo.entries[0].RetryCount)

	assert.Equal(t, 1.0, counterValue(t, reg, "worker_retry_attempts_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ResultFailed,
	}))
	assert.Equal(t, 1.0, counterValue(t, reg, "worker_dlq_entries_total", map[string]string{
		"module": "pm", "subject": "pm.file.received",
	}))
}

// V6: Wrap_ContextCancelStopsImmediately — ctx 取消立即停 + 不入 DLQ
func TestWrap_ContextCancelStopsImmediately(t *testing.T) {
	repo := newMemDLQ()
	metrics, _ := newMetrics(t)
	r := newRunner(t, repo, nil, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wrapped := r.Wrap("pm.file.received", func(ctx context.Context, e event.Event) error {
		return errors.New("inner failure")
	})

	err := wrapped(ctx, event.Event{Subject: "pm.file.received"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
		"err should preserve context cancel sentinel: %v", err)
	assert.Empty(t, repo.entries, "ctx cancel should not produce DLQ entry")
}

// nil dlq 降级路径不 panic
func TestWrap_NilDLQDegradesGracefully(t *testing.T) {
	metrics, _ := newMetrics(t)
	r := newRunner(t, nil, nil, metrics)

	wrapped := r.Wrap("pm.file.received", func(ctx context.Context, e event.Event) error {
		return errors.New("permanent")
	})
	require.NotPanics(t, func() {
		_ = wrapped(context.Background(), event.Event{Subject: "pm.file.received"})
	})
}

// nil handler 不 panic
func TestWrap_NilHandlerIsNoOp(t *testing.T) {
	r := newRunner(t, newMemDLQ(), nil, nil)
	wrapped := r.Wrap("pm.file.received", nil)
	require.NoError(t, wrapped(context.Background(), event.Event{}))
}

// dlq.Insert 失败不传播：不阻塞调用方但留日志
func TestWrap_DLQInsertFailureDoesNotPanic(t *testing.T) {
	repo := &memDLQ{failOn: errors.New("db down")}
	metrics, reg := newMetrics(t)
	r := newRunner(t, repo, nil, metrics)
	wrapped := r.Wrap("pm.file.received", func(ctx context.Context, e event.Event) error {
		return errors.New("permanent")
	})
	err := wrapped(context.Background(), event.Event{Subject: "pm.file.received"})
	require.Error(t, err)
	// failed metric 仍应 +1，但 dlq_entries_total 不能 +1
	assert.Equal(t, 1.0, counterValue(t, reg, "worker_retry_attempts_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ResultFailed,
	}))
	assert.Equal(t, 0.0, counterValue(t, reg, "worker_dlq_entries_total", map[string]string{
		"module": "pm", "subject": "pm.file.received",
	}))
}

// V4: Replay_Success — 重发成功路径
func TestReplay_Success(t *testing.T) {
	pub := &stubPublisher{}
	metrics, reg := newMetrics(t)
	r := newRunner(t, newMemDLQ(), pub, metrics)

	dl := &dlq.DeadLetter{
		ID:            uuid.New(),
		SourceModule:  "pm",
		SourceSubject: "pm.file.received",
		Payload:       []byte(`{"x":1}`),
	}
	require.NoError(t, r.Replay(context.Background(), dl))
	require.Len(t, pub.calls, 1)
	assert.Equal(t, "pm.file.received", pub.calls[0].Subject)
	assert.Equal(t, []byte(`{"x":1}`), []byte(pub.calls[0].Payload))

	assert.Equal(t, 1.0, counterValue(t, reg, "worker_dlq_replays_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ReplayPublished,
	}))
}

// V4 negative: Replay publisher 失败
func TestReplay_PublisherFails(t *testing.T) {
	pub := &stubPublisher{failNext: true}
	metrics, reg := newMetrics(t)
	r := newRunner(t, newMemDLQ(), pub, metrics)

	dl := &dlq.DeadLetter{
		ID:            uuid.New(),
		SourceModule:  "pm",
		SourceSubject: "pm.file.received",
		Payload:       []byte(`{}`),
	}
	err := r.Replay(context.Background(), dl)
	require.Error(t, err)
	assert.Equal(t, 1.0, counterValue(t, reg, "worker_dlq_replays_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ReplayFailed,
	}))
}

// Replay nil publisher 应返回错误并计入 failed metric
func TestReplay_NilPublisher(t *testing.T) {
	metrics, reg := newMetrics(t)
	r := newRunner(t, newMemDLQ(), nil, metrics)
	err := r.Replay(context.Background(), &dlq.DeadLetter{
		ID:            uuid.New(),
		SourceModule:  "pm",
		SourceSubject: "pm.file.received",
	})
	require.Error(t, err)
	assert.Equal(t, 1.0, counterValue(t, reg, "worker_dlq_replays_total", map[string]string{
		"module": "pm", "subject": "pm.file.received", "result": ReplayFailed,
	}))
}

// Replay nil dl 应返回错误，不写 metric
func TestReplay_NilDeadLetterReturnsErr(t *testing.T) {
	metrics, _ := newMetrics(t)
	r := newRunner(t, newMemDLQ(), &stubPublisher{}, metrics)
	require.Error(t, r.Replay(context.Background(), nil))
}

// nil-receiver Metrics 全部方法降级 no-op
func TestMetrics_NilSafeNoOps(t *testing.T) {
	var m *Metrics
	require.NotPanics(t, func() {
		m.RecordRetryResult("pm", "pm.file.received", ResultSuccess)
		m.RecordDLQEntry("pm", "pm.file.received")
		m.RecordReplayResult("pm", "pm.file.received", ReplayFailed)
		m.SetDLQSize("pm", 0)
		_ = m.RefreshDLQSize(context.Background(), nil, "pm")
	})
}

// Module() 暴露给调用方做日志/聚合
func TestRunnerModuleAccessor(t *testing.T) {
	r := newRunner(t, newMemDLQ(), nil, nil)
	assert.Equal(t, "pm", r.Module())
}

// SetDLQSize / RefreshDLQSize 走通
func TestMetrics_RefreshDLQSize(t *testing.T) {
	metrics, reg := newMetrics(t)
	repo := newMemDLQ()
	_ = repo.Insert(context.Background(), &dlq.DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"})
	_ = repo.Insert(context.Background(), &dlq.DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"})

	require.NoError(t, metrics.RefreshDLQSize(context.Background(), repo, "pm"))
	assert.Equal(t, 2.0, counterValue(t, reg, "worker_dlq_size", map[string]string{"module": "pm"}))
}
