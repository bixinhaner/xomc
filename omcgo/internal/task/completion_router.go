package task

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// CompletionRouter 把 task.completed / task.failed NATS 事件按 `Task.Source`
// 路由到对应的 TaskCompletionCallback 实现。P1 核心抽象，去除原
// `CompletionEventBridge` 对 MML aggregator 的硬编码依赖。
//
//   - 注册：Register(TaskSourceMML, mmlAggregator)
//   - 未注册的 source 会走 unknownHandler（默认记 warn 日志，可覆盖）
//   - 线程安全：Register / Dispatch 都加 RWMutex 保护
//
// 当前 source 枚举包含 `api` / `scheduler` / `system` / `mml`；未来新增上游
// 只要：① 定义 TaskSource 常量 ② 实现 TaskCompletionCallback ③ 在装配阶段
// `router.Register(source, handler)`——**TaskService、CompletionEventBridge
// 无需改动**。
type CompletionRouter struct {
	mu             sync.RWMutex
	handlers       map[TaskSource][]TaskCompletionCallback
	observers      []TaskCompletionCallback
	unknownHandler TaskCompletionCallback
	logger         *zap.Logger
	metrics        *TaskMetrics
}

// NewCompletionRouter 构造 router。logger 不能为 nil。
func NewCompletionRouter(logger *zap.Logger) *CompletionRouter {
	return &CompletionRouter{
		handlers:       make(map[TaskSource][]TaskCompletionCallback),
		unknownHandler: &warnOnlyHandler{logger: logger.Named("unknown-source")},
		logger:         logger.Named("completion-router"),
	}
}

// SetMetrics 注入 Prometheus 指标（可选；单测时留 nil）。
// router 会在 Dispatch 时按 source/status 累计计数，未命中 handler 时累计
// completion_no_handler_total。
func (r *CompletionRouter) SetMetrics(m *TaskMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = m
}

// Register 为某个 source 追加一个 handler。同一 source 允许挂多个 handler，
// dispatch 时按注册顺序依次调用（任一 handler panic 不影响其它）。
func (r *CompletionRouter) Register(source TaskSource, handler TaskCompletionCallback) {
	if handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[source] = append(r.handlers[source], handler)
}

// SetUnknownHandler 覆盖默认的"未注册 source 处理器"。传 nil 等同于禁用
// 未知处理（事件被静默丢弃）。
func (r *CompletionRouter) SetUnknownHandler(h TaskCompletionCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unknownHandler = h
}

// RegisterObserver 注册一个 source 无关的终态观察者：每个被 Dispatch 的终态
// 任务（无论 source、无论是否命中 per-source handler）都会回调一次。
//
// 与 Register（按 source 路由到业务聚合器）的区别：observer 是横切关注点，
// 用于"对所有终态任务都要做的事"，如 #122 把终态写入 sys_task_logs。observer
// 永远在 per-source handler 之前调用，且同样受 safeCall 的 panic 隔离保护。
func (r *CompletionRouter) RegisterObserver(observer TaskCompletionCallback) {
	if observer == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, observer)
}

// Dispatch 把一个 device_task 终态按 source 分发。
// handler 执行过程中的 panic 会被 recover 并记录，保证其它订阅不受影响。
// 同时按 source/status 累计 Prometheus 指标（metrics 已注入时生效）。
func (r *CompletionRouter) Dispatch(ctx context.Context, t *Task) {
	if err := r.DispatchReliable(ctx, t); err != nil {
		r.logger.Error("completion handler failed", zap.Error(err))
	}
}

// DispatchReliable dispatches a terminal task and returns projection errors so
// the durable event consumer can NAK and retry the transition.
func (r *CompletionRouter) DispatchReliable(ctx context.Context, t *Task) error {
	if t == nil {
		return nil
	}
	r.mu.RLock()
	handlers, ok := r.handlers[t.Source]
	observers := r.observers
	unknown := r.unknownHandler
	metrics := r.metrics
	r.mu.RUnlock()

	if metrics != nil {
		metrics.CompletedTotal.WithLabelValues(string(t.Source), string(t.Status)).Inc()
		if t.SentAt != nil && !t.SentAt.IsZero() && t.CompletedAt != nil && !t.CompletedAt.IsZero() {
			metrics.DurationSeconds.WithLabelValues(string(t.Source)).Observe(t.CompletedAt.Sub(*t.SentAt).Seconds())
		}
	}

	if !ok || len(handlers) == 0 {
		if metrics != nil {
			metrics.NoHandlerTotal.WithLabelValues(string(t.Source)).Inc()
		}
		if unknown != nil {
			r.safeCall(ctx, unknown, t)
		}
		for _, obs := range observers {
			r.safeCall(ctx, obs, t)
		}
		return nil
	}
	var firstErr error
	for _, h := range handlers {
		if err := r.safeCallReliable(ctx, h, t); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	// Cross-cutting terminal observers run only after durable business
	// projection succeeds. Otherwise a NAK/redelivery would append duplicate
	// terminal logs before the retryable handler has committed.
	if firstErr == nil {
		for _, obs := range observers {
			r.safeCall(ctx, obs, t)
		}
	}
	return firstErr
}

func (r *CompletionRouter) safeCall(ctx context.Context, h TaskCompletionCallback, t *Task) {
	defer func() {
		if p := recover(); p != nil {
			r.logger.Error("completion handler panic",
				zap.String("task_id", t.ID),
				zap.String("source", string(t.Source)),
				zap.Any("panic", p),
			)
		}
	}()
	h.OnTaskCompleted(ctx, t)
}

func (r *CompletionRouter) safeCallReliable(ctx context.Context, h TaskCompletionCallback, t *Task) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("completion handler panic for task %s: %v", t.ID, p)
		}
	}()
	if reliable, ok := h.(ReliableTaskCompletionCallback); ok {
		return reliable.OnTaskCompletedReliable(ctx, t)
	}
	h.OnTaskCompleted(ctx, t)
	return nil
}

// warnOnlyHandler 是默认的未知 source 处理器：只记 warn 日志，不中断流程。
type warnOnlyHandler struct {
	logger *zap.Logger
}

func (h *warnOnlyHandler) OnTaskCompleted(_ context.Context, t *Task) {
	h.logger.Warn("no completion handler registered for task source",
		zap.String("task_id", t.ID),
		zap.String("source", string(t.Source)),
		zap.String("source_id", t.SourceID),
		zap.String("status", string(t.Status)),
	)
}
