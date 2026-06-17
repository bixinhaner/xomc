package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/systimezone"
)

// defaultReloadPollInterval 是动态感知轮询周期：worker 每隔此时长重读 sys_configs
// 业务时区，发现变化即重排 cron。30s 与 ACS 背压看门狗同口径——管理员改时区后
// 最多 30s 即对下次聚合生效，对小时级最快的聚合 cron 无业务影响。
const defaultReloadPollInterval = 30 * time.Second

// cronBuilder 用给定业务时区构造并启动一个 *cron.Cron（含其 WithLocation + 各 AddFunc）。
// tzManager 在初次启动与每次时区变更时调用它重建调度。
type cronBuilder func(loc *time.Location) (*cron.Cron, error)

// tzManager 统一管理 worker 的 PM 业务时区（#458 子单C）。
//
// 背景：聚合切桶（daily/weekly/monthly）按业务时区归零点。业务时区的唯一源是
// sys_configs（category='basic', key='timezoneCode'，#456 子单A 的 systimezone.Provider），
// 不再读 YAML PM.Timezone。
//
// 动态感知（不重启 worker）：本管理器后台轮询（默认 30s）重读 sys_configs 业务时区。
//
// 为何不走 sys.config.saved 事件：该 subject 属 NATS SYS 流（WorkQueuePolicy），其唯一
// consumer 已被 ACS 的 transfercfg.HandleSysConfigSavedEvent 占用——WorkQueue 流不允许
// 第二个同 filter subject 的 consumer（NATS 报 "filtered consumer not unique on workqueue
// stream"），worker 再订阅必失败、动态重排被静默关闭。故改用轮询（与 ACS 背压看门狗、
// transfercfg 同套路，零 NATS 改造）。每轮 reload 让 Provider 缓存失效并重读；若时区变了：
//  1. 原子换出当前 *time.Location（adhoc executor 与各 window 闭包经 Current() 实时读到新值）；
//  2. 停掉所有已注册 cron 并用新 loc 重建（cron.WithLocation 不可变，必须重建调度）。
//
// 历史不回溯：改时区只影响之后新触发的桶，已落库的聚合行边界不动。
// 多 worker 都读同一 sys_configs 源，保证跨实例划桶一致。
type tzManager struct {
	provider *systimezone.Provider
	logger   *zap.Logger

	// cur 原子持有当前业务时区。adhoc executor 跑在独立 goroutine，经 Current() 读，无锁安全；
	// 各 cron window 闭包也经 Current() 实时取 loc（重建间隙也用新时区切桶）。
	cur atomic.Pointer[time.Location]

	mu       sync.Mutex
	builders []cronBuilder // 已注册的 cron 构造器（聚合 + 各 retention）
	running  []*cron.Cron  // 当前活跃的 cron 实例（重建时全停后重填）
	stopped  bool          // ctx 取消后不再重建
}

// newTzManager 构造管理器并解析初始业务时区（读 sys_configs；空/非法回落 UTC）。
//
// pool 为 nil（无 DB，理论上不该发生）时回落 UTC，不 panic。
func newTzManager(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) *tzManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.Named("pm-timezone")

	m := &tzManager{logger: logger}

	if pool != nil {
		repo := admin.NewPgSysConfigRepository(pool)
		fetch := func(ctx context.Context, category, key string) (string, bool) {
			row, err := repo.GetByKey(ctx, category, key)
			if err != nil || row == nil {
				return "", false
			}
			return row.Value, true
		}
		m.provider = systimezone.New(fetch, logger)
	}

	loc := m.resolve(ctx)
	m.cur.Store(loc)
	logger.Info("pm aggregation timezone resolved from sys_configs",
		zap.String("timezone", loc.String()))
	return m
}

// resolve 从 Provider 读当前业务时区；无 Provider 回落 UTC。Provider 内部已保证空/非法回落 UTC。
func (m *tzManager) resolve(ctx context.Context) *time.Location {
	if m.provider == nil {
		return time.UTC
	}
	return m.provider.Location(ctx)
}

// Current 返回当前业务时区（实时、并发安全）。永不返回 nil。
// 供 adhoc executor 的 SetLocationFunc 与 cron 触发时读取。
func (m *tzManager) Current() *time.Location {
	if loc := m.cur.Load(); loc != nil {
		return loc
	}
	return time.UTC
}

// registerCron 注册一个 cron 构造器并立即用当前时区启动它。
// 之后时区变更时本构造器会被用新 loc 重新调用以重建调度。
func (m *tzManager) registerCron(b cronBuilder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return
	}
	m.builders = append(m.builders, b)
	c, err := b(m.Current())
	if err != nil {
		m.logger.Error("build cron failed", zap.Error(err))
		return
	}
	if c != nil {
		c.Start()
		m.running = append(m.running, c)
	}
}

// rebuildCrons 停掉所有活跃 cron 并用 newLoc 全部重建。持有 m.mu 调用。
func (m *tzManager) rebuildCrons(newLoc *time.Location) {
	for _, c := range m.running {
		stopCtx := c.Stop()
		<-stopCtx.Done()
	}
	m.running = m.running[:0]
	if m.stopped {
		return
	}
	for _, b := range m.builders {
		c, err := b(newLoc)
		if err != nil {
			m.logger.Error("rebuild cron failed", zap.Error(err))
			continue
		}
		if c != nil {
			c.Start()
			m.running = append(m.running, c)
		}
	}
	m.logger.Info("cron schedulers rebuilt with new timezone",
		zap.String("timezone", newLoc.String()), zap.Int("count", len(m.running)))
}

// shutdownOnCtx 在 ctx 取消时停掉所有 cron 并禁止后续重建。
func (m *tzManager) shutdownOnCtx(ctx context.Context) {
	go func() {
		<-ctx.Done()
		m.mu.Lock()
		defer m.mu.Unlock()
		m.stopped = true
		for _, c := range m.running {
			stopCtx := c.Stop()
			<-stopCtx.Done()
		}
		m.running = m.running[:0]
		m.logger.Info("all pm cron schedulers stopped")
	}()
}

// reload 让 Provider 缓存失效并重读时区；若变化则原子换出 + 重建所有 cron。
// 由后台轮询周期触发；返回是否发生了变化（便于测试断言）。
func (m *tzManager) reload(ctx context.Context) bool {
	if m.provider != nil {
		m.provider.Invalidate()
	}
	newLoc := m.resolve(ctx)
	old := m.Current()
	if newLoc.String() == old.String() {
		m.logger.Debug("timezone unchanged after reload", zap.String("timezone", newLoc.String()))
		return false
	}
	m.cur.Store(newLoc)
	m.logger.Info("pm aggregation timezone changed; rescheduling cron without restart",
		zap.String("old", old.String()), zap.String("new", newLoc.String()))

	m.mu.Lock()
	m.rebuildCrons(newLoc)
	m.mu.Unlock()
	return true
}

// startReloadPoller 启动后台轮询：每隔 interval 重读 sys_configs 业务时区，
// 发现变化即重排所有 cron（不重启 worker）。ctx 取消时退出。
//
// 为何轮询而非事件订阅：sys.config.saved 属 NATS SYS WorkQueue 流，其唯一 consumer
// 已被 ACS transfercfg 占用，worker 不能在同 filter subject 再开第二个 consumer（必报
// "filtered consumer not unique on workqueue stream"）。轮询是 spec 给的备选方案，
// 与 ACS 背压看门狗 / transfercfg 同套路，零 NATS 改造。
//
// 无 Provider（无 DB）时不启动轮询——时区恒为启动时解析的 UTC，无可重读源。
func (m *tzManager) startReloadPoller(ctx context.Context, interval time.Duration) {
	if m.provider == nil {
		m.logger.Info("no sys_configs provider; dynamic timezone reload poller disabled (timezone fixed at startup)")
		return
	}
	if interval <= 0 {
		interval = defaultReloadPollInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		m.logger.Info("dynamic timezone reload poller started (no restart needed)",
			zap.Duration("interval", interval))
		for {
			select {
			case <-ctx.Done():
				m.logger.Info("dynamic timezone reload poller stopped")
				return
			case <-ticker.C:
				m.reload(ctx)
			}
		}
	}()
}
