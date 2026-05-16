package trace

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// WhitelistCache 跟踪 SN 白名单（ACS 进程内）。
// M1 阶段通过定时轮询 PG 维护一致性（默认 5s）；M2 改为 NATS 事件实时更新 + 30s 兜底对账。
type WhitelistCache struct {
	repo   Repository
	logger *zap.Logger

	m sync.Map // map[string]uuid.UUID  device_sn → task_id

	refreshInterval time.Duration
	lastRefresh     atomic.Int64 // unix nano，监控用

	shutdownOnce sync.Once
	doneCh       chan struct{}
	stopCh       chan struct{}
}

// WhitelistConfig 缓存配置。
//
//   - M1 模式（bus == nil）：仅 5s 轮询。
//   - M2 模式（注入 EventBus）：NATS 实时同步主导，轮询降为 30s 兜底对账。
type WhitelistConfig struct {
	RefreshInterval time.Duration // M1 默认 5s；M2 自动放宽到 30s
}

// DefaultWhitelistConfig 默认配置（M1 兼容）。
func DefaultWhitelistConfig() WhitelistConfig {
	return WhitelistConfig{RefreshInterval: 5 * time.Second}
}

// M2WhitelistConfig 跨进程 NATS 实时同步模式下的默认配置。
func M2WhitelistConfig() WhitelistConfig {
	return WhitelistConfig{RefreshInterval: 30 * time.Second}
}

// NewWhitelistCache 构造函数。调用方需调 Start 启动后台刷新。
func NewWhitelistCache(repo Repository, cfg WhitelistConfig, logger *zap.Logger) *WhitelistCache {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 5 * time.Second
	}
	return &WhitelistCache{
		repo:            repo,
		logger:          logger,
		refreshInterval: cfg.RefreshInterval,
		doneCh:          make(chan struct{}),
		stopCh:          make(chan struct{}),
	}
}

// Start 阻塞执行一次启动加载，然后后台周期刷新。
func (w *WhitelistCache) Start(ctx context.Context) error {
	if err := w.refresh(ctx); err != nil {
		w.logger.Error("trace: initial whitelist load failed", zap.Error(err))
		return err
	}
	w.logger.Info("trace: whitelist initial load done",
		zap.Int("size", w.Size()),
		zap.Duration("refresh_interval", w.refreshInterval))
	go w.refresher(ctx)
	return nil
}

// Stop 停止后台刷新。
func (w *WhitelistCache) Stop() {
	w.shutdownOnce.Do(func() {
		close(w.stopCh)
		<-w.doneCh
	})
}

// Lookup 命中返回 task_id；未命中返回 uuid.Nil + false。O(1)。
func (w *WhitelistCache) Lookup(sn string) (uuid.UUID, bool) {
	if sn == "" {
		return uuid.Nil, false
	}
	v, ok := w.m.Load(sn)
	if !ok {
		return uuid.Nil, false
	}
	return v.(uuid.UUID), true
}

// Size 当前白名单大小（监控用）。
func (w *WhitelistCache) Size() int {
	n := 0
	w.m.Range(func(_, _ any) bool { n++; return true })
	return n
}

// LastRefresh 上次刷新时间（监控用）。
func (w *WhitelistCache) LastRefresh() time.Time {
	ns := w.lastRefresh.Load()
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

// Add 手动添加（M2 NATS 事件即时回调使用；M1 不直接调用）。
func (w *WhitelistCache) Add(sn string, taskID uuid.UUID) {
	if sn == "" {
		return
	}
	w.m.Store(sn, taskID)
}

// Remove 手动移除。
func (w *WhitelistCache) Remove(sn string) {
	if sn == "" {
		return
	}
	w.m.Delete(sn)
}

// Subscribe 订阅 trace.task.{started,stopped,purged} NATS 事件，实时维护白名单。
// 失败仅记 WARN — 退化为兜底轮询。
//
// 返回的 cleanup 在进程退出时 Unsubscribe，避免 NATS leak。
func (w *WhitelistCache) Subscribe(bus event.EventBus) (func(), error) {
	if bus == nil {
		return func() {}, nil
	}
	subs := make([]event.Subscription, 0, 3)

	startSub, err := bus.Subscribe(event.SubjectTraceTaskStarted, w.handleTaskEvent)
	if err != nil {
		return nil, err
	}
	subs = append(subs, startSub)

	stopSub, err := bus.Subscribe(event.SubjectTraceTaskStopped, w.handleTaskEvent)
	if err != nil {
		_ = startSub.Unsubscribe()
		return nil, err
	}
	subs = append(subs, stopSub)

	purgeSub, err := bus.Subscribe(event.SubjectTraceTaskPurged, w.handleTaskEvent)
	if err != nil {
		_ = startSub.Unsubscribe()
		_ = stopSub.Unsubscribe()
		return nil, err
	}
	subs = append(subs, purgeSub)

	w.logger.Info("trace: whitelist subscribed to NATS task events")
	return func() {
		for _, s := range subs {
			_ = s.Unsubscribe()
		}
	}, nil
}

// handleTaskEvent 解 TaskEvent 后增删 SN。
//
// started → Add；stopped / purged → Remove。
// 注意：CreateTask 替换旧任务时会先发 stopped(old) 再发 started(new)，
// 由于 stopped 携带的是旧 task_id 而非旧 SN…等等，旧 task 的 device_sn 与新
// task 相同 — 这里需小心去重：若 stopped 事件到达但 SN 在白名单里且对应不同
// task_id（已被 started 覆盖），不要误删。
func (w *WhitelistCache) handleTaskEvent(_ context.Context, evt event.Event) error {
	var payload TaskEvent
	if err := evt.DecodePayload(&payload); err != nil {
		w.logger.Warn("trace: decode task event payload failed",
			zap.String("subject", evt.Subject), zap.Error(err))
		return nil // 不返错 — 防止 JetStream 重投放大错误
	}
	if payload.DeviceSN == "" {
		return nil
	}
	switch evt.Subject {
	case event.SubjectTraceTaskStarted:
		w.Add(payload.DeviceSN, payload.TaskID)
	case event.SubjectTraceTaskStopped, event.SubjectTraceTaskPurged:
		// 只在白名单中映射到该 task_id 时才移除（避免误删后到达的 stopped 事件
		// 把新 started 任务的 SN 删掉）。
		if cur, ok := w.Lookup(payload.DeviceSN); ok && cur == payload.TaskID {
			w.Remove(payload.DeviceSN)
		}
	}
	return nil
}

func (w *WhitelistCache) refresher(ctx context.Context) {
	defer close(w.doneCh)
	ticker := time.NewTicker(w.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			rCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			if err := w.refresh(rCtx); err != nil {
				w.logger.Warn("trace: whitelist refresh failed", zap.Error(err))
			}
			cancel()
		}
	}
}

// refresh 全量加载 → 替换。SN 量级 < 1000，全量替换成本可忽略。
func (w *WhitelistCache) refresh(ctx context.Context) error {
	snapshot, err := w.repo.ListRunningSNs(ctx)
	if err != nil {
		return err
	}
	// 先标记当前 keys，最后删除不在新快照里的
	current := map[string]struct{}{}
	w.m.Range(func(k, _ any) bool {
		current[k.(string)] = struct{}{}
		return true
	})
	for sn, taskID := range snapshot {
		w.m.Store(sn, taskID)
		delete(current, sn)
	}
	for sn := range current {
		w.m.Delete(sn)
	}
	w.lastRefresh.Store(time.Now().UnixNano())
	return nil
}
