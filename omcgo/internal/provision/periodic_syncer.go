package provision

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// StaleDeviceLister 是 PeriodicSyncer 需要的"查过期设备"能力（消费者驱动接口，
// device.DeviceRepository 自然满足）。
type StaleDeviceLister interface {
	ListStaleForParamSync(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error)
}

// PathBSyncStarter 是 PeriodicSyncer 需要的"启动 Path B 同步"能力（消费者驱动接口，
// *SyncService 自然满足）。
type PathBSyncStarter interface {
	StartPathBSync(ctx context.Context, dev *model.Device, sourceID string, opts ...PathBOption) (bool, error)
}

// PeriodicSyncer 周期性参数同步兜底（T-0124 设计 §2）。
//
// 按 interval 扫描 active 设备中 last_param_sync_at NULL 或过期的，逐个调
// StartPathBSync(WithReason("periodic")) 入队 Path B 全量同步，作为"配置漂移
// 检测"的兜底链路（事件驱动链路 device_online / firmware_changed / manual 已覆盖
// 大部分场景；本兜底覆盖"长期在线无变化但本地被改过参数"的盲点）。
//
// 多副本场景：通过 LeaderElector 保证同一时刻只有一个副本执行 runOnce。
type PeriodicSyncer struct {
	lister StaleDeviceLister
	syncer PathBSyncStarter
	leader LeaderElector
	cfg    appconfig.PeriodicSyncConfig
	logger *zap.Logger
}

// NewPeriodicSyncer 创建周期同步器。leader 可为 nil — 单副本部署不需要协调。
func NewPeriodicSyncer(
	lister StaleDeviceLister,
	syncer PathBSyncStarter,
	leader LeaderElector,
	cfg appconfig.PeriodicSyncConfig,
	logger *zap.Logger,
) *PeriodicSyncer {
	return &PeriodicSyncer{
		lister: lister,
		syncer: syncer,
		leader: leader,
		cfg:    cfg,
		logger: logger,
	}
}

// Start 启动 ticker，阻塞运行到 ctx.Done。Enabled=false 时立即返回。
// 调用方通常 go p.Start(ctx)；进程退出 ctx 取消，Start 优雅退出并调 leader.Release。
func (p *PeriodicSyncer) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.logger.Info("periodic syncer disabled, not starting")
		return nil
	}
	interval := p.cfg.Interval
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	p.logger.Info("periodic syncer started",
		zap.Duration("interval", interval),
		zap.Int("batch_size", p.batchSize()),
		zap.Int("max_concurrent", p.maxConcurrent()),
		zap.Duration("stagger_window", p.cfg.StaggerWindow))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer func() {
		if p.leader != nil {
			if err := p.leader.Release(context.Background()); err != nil {
				p.logger.Warn("periodic syncer: release leader lock failed", zap.Error(err))
			}
		}
	}()

	// 首次启动延迟一个 interval 再跑（让其他初始化稳定），与 OfflineDetector 不同——
	// OfflineDetector 关心快速冷启动；本兜底不急于第一次跑。
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("periodic syncer stopped")
			return ctx.Err()
		case <-ticker.C:
			p.runOnce(ctx)
		}
	}
}

// runOnce 单轮：leader 检查 → 查 stale 设备 → 并发池入队 Path B 同步。
//
// 任何步骤错误仅 log 不 panic，保证下一 tick 能继续。
func (p *PeriodicSyncer) runOnce(ctx context.Context) {
	// 1. leader 检查（nil leader 视为单副本部署直接放行）
	if p.leader != nil {
		isLeader, err := p.leader.TryAcquire(ctx)
		if err != nil {
			p.logger.Warn("periodic syncer: leader TryAcquire failed, skip run", zap.Error(err))
			return
		}
		if !isLeader {
			p.logger.Debug("periodic syncer: not leader, skip run")
			return
		}
	}

	// 2. 查 stale 设备：last_param_sync_at IS NULL OR < now - interval
	threshold := time.Now().Add(-p.cfg.Interval)
	devices, err := p.lister.ListStaleForParamSync(ctx, threshold, p.batchSize())
	if err != nil {
		p.logger.Warn("periodic syncer: list stale devices failed", zap.Error(err))
		return
	}
	if len(devices) == 0 {
		p.logger.Debug("periodic syncer: no stale devices in this round")
		return
	}

	// 3. 并发池入队（MaxConcurrent 限并发；StaggerWindow > 0 时打散到窗口）
	start := time.Now()
	enqueued, skipped, failed := p.enqueueBatch(ctx, devices)

	p.logger.Info("periodic syncer: batch enqueued",
		zap.Int("device_count", len(devices)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
		zap.Duration("duration", time.Since(start)))
}

// enqueueBatch 并发入队 Path B 同步，返回 (成功入队 / Path B 不可用跳过 / 失败) 计数。
func (p *PeriodicSyncer) enqueueBatch(ctx context.Context, devices []*model.Device) (enqueued, skipped, failed int) {
	maxConc := p.maxConcurrent()
	sem := make(chan struct{}, maxConc)
	var wg sync.WaitGroup
	var countMu sync.Mutex
	stagger := p.cfg.StaggerWindow

	for i, dev := range devices {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(idx int, d *model.Device) {
			defer wg.Done()
			defer func() { <-sem }()

			// Stagger 打散：把入队动作分散到 [0, stagger) 窗口内的随机点
			if stagger > 0 {
				delay := time.Duration(rand.Int63n(int64(stagger)))
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}
			}

			sourceID := fmt.Sprintf("periodic:%s", d.ID.String())
			used, err := p.syncer.StartPathBSync(ctx, d, sourceID, WithReason("periodic"))

			countMu.Lock()
			defer countMu.Unlock()
			switch {
			case err != nil:
				failed++
				p.logger.Warn("periodic syncer: StartPathBSync failed",
					zap.String("device_id", d.ID.String()),
					zap.String("device_sn", d.SerialNumber),
					zap.Error(err))
			case !used:
				// Path B 不可用（无 MappingSet 等）— 计为 skipped 不算失败
				skipped++
			default:
				enqueued++
			}
		}(i, dev)
	}
	wg.Wait()
	return
}

func (p *PeriodicSyncer) batchSize() int {
	if p.cfg.BatchSize <= 0 {
		return 200
	}
	return p.cfg.BatchSize
}

func (p *PeriodicSyncer) maxConcurrent() int {
	if p.cfg.MaxConcurrent <= 0 {
		return 10
	}
	return p.cfg.MaxConcurrent
}
