package provision

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// StaleDeviceLister 是 PeriodicSyncer 需要的"查过期设备"能力（消费者驱动接口，
// device.DeviceRepository 自然满足）。
type StaleDeviceLister interface {
	ListStaleForParamSync(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error)
}

// PathBSyncStarter 是 PeriodicSyncer 需要的"启动参数同步"能力（消费者驱动接口，
// 当前由 *SyncService 适配到 durable parameter_sync 数据面）。
type PathBSyncStarter interface {
	StartPathBSync(ctx context.Context, dev *model.Device, sourceID string, opts ...PathBOption) (bool, int, error)
}

type ReleaseCandidateLister interface {
	ListReleaseCandidates(ctx context.Context, campaignID uuid.UUID, limit int) ([]*model.Device, error)
}

type ReleaseSyncStarter interface {
	StartReleaseSync(
		ctx context.Context,
		dev *model.Device,
		campaignID uuid.UUID,
		attemptID uuid.UUID,
	) (submitted bool, err error)
}

// PeriodicSyncer 周期性参数同步兜底（T-0124 设计 §2）。
//
// 按 Policy.Snapshot().Interval 扫描 active 设备中 last_param_sync_at NULL 或
// 过期的，逐个调 StartPathBSync(WithReason("periodic"))，统一提交到
// durable parameter_sync_* 数据面。
//
// 本同步作为"配置漂移检测"的兜底链路（事件驱动链路 device_online / firmware_changed /
// manual 已覆盖大部分场景；本兜底覆盖"长期在线无变化但本地被改过参数"的盲点）。
//
// 配置源：sys_configs (category='device')，由 FE pages/system/SystemConfig/
// DeviceSettings.tsx 表单 batch upsert 写入；通过 PeriodicSyncPolicy（30s 缓存）
// 读出。Enabled / Interval / BatchSize / MaxConcurrent / StaggerWindow 都在
// runtime 动态生效，最多 30s 内反映到下一轮调度（Enabled / Interval 跨 tick 才会
// 重新应用到 ticker 周期，参 Start 注释）。
//
// 多副本场景：通过 LeaderElector 保证同一时刻只有一个副本执行 runOnce。
type PeriodicSyncer struct {
	lister StaleDeviceLister
	syncer PathBSyncStarter
	leader LeaderElector
	policy *PeriodicSyncPolicy
	logger *zap.Logger

	releaseLister       ReleaseCandidateLister
	releaseStarter      ReleaseSyncStarter
	releaseCampaignID   uuid.UUID
	releaseStaggerDelay func(time.Duration) time.Duration
}

func (p *PeriodicSyncer) SetReleaseSync(
	lister ReleaseCandidateLister,
	starter ReleaseSyncStarter,
	campaignID uuid.UUID,
) {
	if lister == nil || starter == nil || campaignID == uuid.Nil {
		return
	}
	p.releaseLister = lister
	p.releaseStarter = starter
	p.releaseCampaignID = campaignID
}

// NewPeriodicSyncer 创建周期同步器。leader / policy 可为 nil：
//   - leader=nil：单副本部署不需要协调
//   - policy=nil：全部走 default（Enabled=false → Start 立即退出）
func NewPeriodicSyncer(
	lister StaleDeviceLister,
	syncer PathBSyncStarter,
	leader LeaderElector,
	policy *PeriodicSyncPolicy,
	logger *zap.Logger,
) *PeriodicSyncer {
	return &PeriodicSyncer{
		lister: lister,
		syncer: syncer,
		leader: leader,
		policy: policy,
		logger: logger,
	}
}

// pollInterval 是 Start 检查 Policy 的最短轮询周期。Snapshot 命中 30s 缓存
// 不打 DB，所以 60s 轮询代价微乎其微，但能保证 Enabled / Interval 的运行时切换
// 在 ~1 分钟内生效。
const periodicSyncPollInterval = 1 * time.Minute

// Start 阻塞运行到 ctx.Done。
//
// 启动后立即检查一次，此后每分钟检查一次 policy.Snapshot()：
//   - Enabled=false → 跳过本轮（运行中可通过 FE 关 enabled 来临时停掉）
//   - Enabled=true 且距上次执行 ≥ Interval → 执行 runOnce
//
// 调用方通常 go p.Start(ctx)；进程退出 ctx 取消，Start 优雅退出并调 leader.Release。
func (p *PeriodicSyncer) Start(ctx context.Context) error {
	p.logger.Info("periodic syncer scheduler started (waits for policy.enabled)")

	timer := time.NewTimer(0)
	defer timer.Stop()
	defer func() {
		if p.leader != nil {
			if err := p.leader.Release(context.Background()); err != nil {
				p.logger.Warn("periodic syncer: release leader lock failed", zap.Error(err))
			}
		}
	}()

	var lastRunAt time.Time
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("periodic syncer stopped")
			return ctx.Err()
		case <-timer.C:
			snap := p.snapshot(ctx)
			p.runReleaseOnce(ctx, snap)
			if snap.Enabled && p.lister != nil && p.syncer != nil &&
				(lastRunAt.IsZero() || time.Since(lastRunAt) >= snap.Interval) {
				if p.runOnce(ctx, snap) {
					lastRunAt = time.Now()
				}
			}
			timer.Reset(periodicSyncPollInterval)
		}
	}
}

// runOnce 单轮：leader 检查 → 查 stale 设备 → 并发池入队参数同步。
//
// 返回 true 表示本轮完成了有效扫描，可推进 lastRunAt；leader/DB 等临时失败返回 false，
// 让 Start 在下一轮 poll 尽快重试。任何步骤错误仅 log 不 panic，保证下一 tick 能继续。
func (p *PeriodicSyncer) runOnce(ctx context.Context, snap PeriodicSyncSnapshot) bool {
	// 1. leader 检查（nil leader 视为单副本部署直接放行）
	if p.leader != nil {
		isLeader, err := p.leader.TryAcquire(ctx)
		if err != nil {
			p.logger.Warn("periodic syncer: leader TryAcquire failed, skip run", zap.Error(err))
			return false
		}
		if !isLeader {
			p.logger.Debug("periodic syncer: not leader, skip run")
			return false
		}
	}

	// 2. 查 stale 设备：last_param_sync_at IS NULL OR < now - interval
	threshold := time.Now().Add(-snap.Interval)
	devices, err := p.lister.ListStaleForParamSync(ctx, threshold, snap.BatchSize)
	if err != nil {
		p.logger.Warn("periodic syncer: list stale devices failed", zap.Error(err))
		return false
	}
	if len(devices) == 0 {
		p.logger.Debug("periodic syncer: no stale devices in this round")
		return true
	}

	// 3. 并发池入队（MaxConcurrent 限并发；StaggerWindow > 0 时打散到窗口）
	start := time.Now()
	enqueued, skipped, failed := p.enqueueBatch(ctx, devices, snap)

	p.logger.Info("periodic syncer: batch enqueued",
		zap.Int("device_count", len(devices)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
		zap.Duration("duration", time.Since(start)),
		zap.Duration("interval", snap.Interval),
		zap.Int("batch_size", snap.BatchSize),
		zap.Int("max_concurrent", snap.MaxConcurrent),
		zap.Duration("stagger_window", snap.StaggerWindow))
	return true
}

func (p *PeriodicSyncer) runReleaseOnce(ctx context.Context, snap PeriodicSyncSnapshot) bool {
	if p.releaseLister == nil ||
		p.releaseStarter == nil ||
		p.releaseCampaignID == uuid.Nil {
		return false
	}
	if p.leader != nil {
		isLeader, err := p.leader.TryAcquire(ctx)
		if err != nil {
			p.logger.Warn("release sync: leader TryAcquire failed", zap.Error(err))
			return false
		}
		if !isLeader {
			return false
		}
	}
	devices, err := p.releaseLister.ListReleaseCandidates(
		ctx,
		p.releaseCampaignID,
		snap.BatchSize,
	)
	if err != nil {
		p.logger.Warn("release sync: list candidates failed", zap.Error(err))
		return false
	}
	if len(devices) == 0 {
		return true
	}
	enqueued, skipped, failed := p.enqueueReleaseBatch(
		ctx,
		devices,
		p.releaseCampaignID,
		snap,
	)
	p.logger.Info("release sync: batch submitted",
		zap.String("campaign_id", p.releaseCampaignID.String()),
		zap.Int("device_count", len(devices)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed))
	return true
}

func (p *PeriodicSyncer) enqueueReleaseBatch(
	ctx context.Context,
	devices []*model.Device,
	campaignID uuid.UUID,
	snap PeriodicSyncSnapshot,
) (enqueued, skipped, failed int) {
	sem := make(chan struct{}, snap.MaxConcurrent)
	var wg sync.WaitGroup
	var countMu sync.Mutex

	for _, dev := range devices {
		wg.Add(1)
		go func(d *model.Device) {
			defer wg.Done()

			if snap.StaggerWindow > 0 {
				delay := time.Duration(rand.Int63n(int64(snap.StaggerWindow)))
				if p.releaseStaggerDelay != nil {
					delay = p.releaseStaggerDelay(snap.StaggerWindow)
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}
			}
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			submitted, err := p.releaseStarter.StartReleaseSync(
				ctx,
				d,
				campaignID,
				uuid.New(),
			)
			countMu.Lock()
			defer countMu.Unlock()
			switch {
			case err != nil:
				failed++
				p.logger.Warn("release sync: device submission failed",
					zap.String("device_id", d.ID.String()),
					zap.String("device_sn", d.SerialNumber),
					zap.Error(err))
			case !submitted:
				skipped++
			default:
				enqueued++
			}
		}(dev)
	}
	wg.Wait()
	return
}

// enqueueBatch 并发入队参数同步，返回 (成功入队 / durable 数据面不可用跳过 / 失败) 计数。
func (p *PeriodicSyncer) enqueueBatch(ctx context.Context, devices []*model.Device, snap PeriodicSyncSnapshot) (enqueued, skipped, failed int) {
	sem := make(chan struct{}, snap.MaxConcurrent)
	var wg sync.WaitGroup
	var countMu sync.Mutex
	stagger := snap.StaggerWindow

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

			if !p.isEnabled(ctx) {
				return
			}

			// Stagger 打散：把入队动作分散到 [0, stagger) 窗口内的随机点
			if stagger > 0 {
				delay := time.Duration(rand.Int63n(int64(stagger)))
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}
				if !p.isEnabled(ctx) {
					return
				}
			}

			// Empty sourceID lets SyncService allocate a per-run UUID, avoiding
			// cross-run contamination for recurring automatic syncs.
			used, _, err := p.syncer.StartPathBSync(ctx, d, "", WithReason("periodic"))

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

func (p *PeriodicSyncer) isEnabled(ctx context.Context) bool {
	if p.policy == nil {
		return true
	}
	return p.policy.Snapshot(ctx).Enabled
}

// snapshot 取当前快照；policy=nil 时返 default（Enabled=false）。
func (p *PeriodicSyncer) snapshot(ctx context.Context) PeriodicSyncSnapshot {
	if p.policy == nil {
		return defaultPeriodicSyncSnapshot()
	}
	return p.policy.Snapshot(ctx)
}
