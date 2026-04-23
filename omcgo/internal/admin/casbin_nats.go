package admin

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2/persist"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// natsCasbinWatcher 用 NATS JetStream 替代 Redis Pub/Sub 作为 Casbin
// 策略变更的分布式通知通道，获得持久化 + 断线重连回放 + 与项目主事件
// 总线统一的 at-least-once 交付语义。
//
// 在 EventBus 不可用（未部署 NATS 或 bus=nil）时降级为 no-op：Update
// 只记录日志不报错，StartListener 不启动 goroutine。此时策略同步退化
// 为 CasbinAuthorizer.StartPeriodicRefresh 的周期性全量刷新。
type natsCasbinWatcher struct {
	bus      event.EventBus
	callback func(string)
	sub      event.Subscription
	logger   *zap.Logger
}

var _ persist.Watcher = (*natsCasbinWatcher)(nil)

// newNATSCasbinWatcher 构造 NATS 版 Watcher。bus 为 nil 时返回退化实例。
func newNATSCasbinWatcher(bus event.EventBus, logger *zap.Logger) *natsCasbinWatcher {
	return &natsCasbinWatcher{
		bus:    bus,
		logger: logger.Named("casbin-watcher"),
	}
}

// SetUpdateCallback 由 Casbin Enforcer 注入刷新回调。
func (w *natsCasbinWatcher) SetUpdateCallback(callback func(string)) error {
	w.callback = callback
	return nil
}

// Update 由本地 Enforcer 在策略变更后调用，通过 NATS 广播给全集群。
// Bus 不可用时仅记日志，由周期性刷新兜底最终一致。
func (w *natsCasbinWatcher) Update() error {
	if w.bus == nil {
		w.logger.Debug("event bus unavailable; skip publish, rely on periodic refresh")
		return nil
	}
	evt, err := event.NewEvent(event.SubjectSysCasbinPolicyReload, map[string]string{"signal": "reload"})
	if err != nil {
		return fmt.Errorf("build casbin reload event: %w", err)
	}
	if err := w.bus.Publish(context.Background(), event.SubjectSysCasbinPolicyReload, evt); err != nil {
		return fmt.Errorf("publish casbin reload: %w", err)
	}
	return nil
}

// StartListener 注册全集群策略重载订阅。使用普通 Subscribe（而非
// QueueSubscribe）使每个副本都能收到通知并各自 LoadPolicy——这是广播
// 语义的本意。Bus 不可用则跳过订阅。
func (w *natsCasbinWatcher) StartListener() {
	if w.bus == nil {
		w.logger.Warn("event bus unavailable; casbin policy reload relies solely on periodic refresh")
		return
	}
	sub, err := w.bus.Subscribe(event.SubjectSysCasbinPolicyReload, func(_ context.Context, evt event.Event) error {
		if w.callback != nil {
			w.callback("reload")
		}
		return nil
	})
	if err != nil {
		w.logger.Error("subscribe casbin reload", zap.Error(err))
		return
	}
	w.sub = sub
	w.logger.Info("casbin policy reload listener started",
		zap.String("subject", event.SubjectSysCasbinPolicyReload))
}

// Close 取消订阅；重复调用安全。
func (w *natsCasbinWatcher) Close() {
	if w.sub != nil {
		if err := w.sub.Unsubscribe(); err != nil {
			w.logger.Warn("unsubscribe casbin reload", zap.Error(err))
		}
		w.sub = nil
	}
}

// Notify 向后兼容 CasbinAuthorizer.NotifyPolicyChange 的别名。
func (w *natsCasbinWatcher) Notify() error {
	return w.Update()
}
