package device

import (
	"context"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"go.uber.org/zap"
)

// OnlinePruner 周期性维护 acs:online 在线索引（issue #397，加法优先阶段）：
//   - Prune：删除早于 now-PruneWindow 的成员，控制集合规模（不参与离线判定，
//     故保留窗口取得很宽，远大于最大离线阈值，避免误删仍可能在线的设备）。
//   - 记录：把「最近 CountWindow 秒内有上报」的在线总数打到日志，提供运维可观测，
//     同时验证 ACS 侧 Mark 链路确实在写入。
//
// 本期不改离线判定（reconciler 仍读 PG last_inform_at）；OnlinePruner 与
// DeviceStatusReconciler 并存、各司其职。
type OnlinePruner struct {
	index       *redisx.OnlineIndex
	interval    time.Duration
	pruneWindow time.Duration
	countWindow time.Duration
	logger      *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

const (
	defaultOnlinePruneInterval = 60 * time.Second
	defaultOnlinePruneWindow   = time.Hour         // 保留窗口：远大于最大离线阈值（CPE 600s），仅控规模
	defaultOnlineCountWindow   = 600 * time.Second // 在线统计窗口（日志用）
)

// NewOnlinePruner 构造一个 pruner。index 为 nil（无 Redis）时 Start 退化为不启动。
func NewOnlinePruner(index *redisx.OnlineIndex, logger *zap.Logger) *OnlinePruner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OnlinePruner{
		index:       index,
		interval:    defaultOnlinePruneInterval,
		pruneWindow: defaultOnlinePruneWindow,
		countWindow: defaultOnlineCountWindow,
		logger:      logger,
	}
}

// Start 启动后台循环（非阻塞）。index 为 nil 时不启动。
func (p *OnlinePruner) Start() {
	if p == nil || p.index == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.wg.Add(1)
	go p.loop(ctx)
	p.logger.Info("online index pruner started",
		zap.Duration("interval", p.interval),
		zap.Duration("prune_window", p.pruneWindow),
		zap.Duration("count_window", p.countWindow))
}

// Stop 取消并等待循环退出。可多次安全调用。
func (p *OnlinePruner) Stop() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.wg.Wait()
}

func (p *OnlinePruner) loop(ctx context.Context) {
	defer p.wg.Done()
	t := time.NewTicker(p.interval)
	defer t.Stop()
	p.tick(ctx, time.Now().Unix()) // 启动即跑一次
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("online index pruner stopped")
			return
		case <-t.C:
			p.tick(ctx, time.Now().Unix())
		}
	}
}

// tick 执行一轮清理 + 在线计数日志。nowUnix 显式传入便于单测。
func (p *OnlinePruner) tick(ctx context.Context, nowUnix int64) {
	if p.index == nil {
		return
	}
	pruneCutoff := nowUnix - int64(p.pruneWindow.Seconds())
	removed, err := p.index.Prune(ctx, pruneCutoff)
	if err != nil {
		p.logger.Warn("online index prune failed", zap.Error(err))
		return
	}
	countCutoff := nowUnix - int64(p.countWindow.Seconds())
	online, err := p.index.Count(ctx, countCutoff)
	if err != nil {
		p.logger.Warn("online index count failed", zap.Error(err))
		return
	}
	p.logger.Info("online index reconciled",
		zap.Int64("pruned", removed),
		zap.Int64("online_within_window", online),
		zap.Int("count_window_sec", int(p.countWindow.Seconds())))
}
