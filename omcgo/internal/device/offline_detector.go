package device

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// OfflineDetector 离线检测器
// 定期扫描超时未上报 Inform 的设备，将其标记为离线状态
type OfflineDetector struct {
	deviceRepo DeviceRepository
	infoSyncer *InfoSyncer
	eventBus   event.EventBus
	seq        *Sequencer // per-device 串行化，与 DeviceService / marker 共享同一锁
	cache      *DeviceCache // 写 DB 后 invalidate，与 UpdateFromInform 的 cache.GetOrLoad 协调
	logger     *zap.Logger

	// 配置参数
	checkInterval    time.Duration // 扫描间隔，默认 5 分钟
	offlineThreshold time.Duration // 离线阈值，默认 10 分钟（2×默认心跳）
	batchSize        int           // 单次处理最大设备数
}

// DeviceOfflineEvent 设备离线事件
type DeviceOfflineEvent struct {
	DeviceID    uuid.UUID         `json:"device_id"`
	Serial      string            `json:"serial_number"`
	Carrier     model.CarrierCode `json:"carrier"`
	Technology  model.Technology  `json:"technology"`
	OfflineTime time.Time         `json:"offline_time"`
}

// NewOfflineDetector 创建离线检测器
func NewOfflineDetector(
	deviceRepo DeviceRepository,
	infoSyncer *InfoSyncer,
	eventBus event.EventBus,
	logger *zap.Logger,
) *OfflineDetector {
	return &OfflineDetector{
		deviceRepo:       deviceRepo,
		infoSyncer:       infoSyncer,
		eventBus:         eventBus,
		logger:           logger,
		checkInterval:    5 * time.Minute,
		offlineThreshold: 10 * time.Minute,
		batchSize:        1000,
	}
}

// SetSequencer 注入 per-device 串行化锁（与 DeviceService 共享同一实例）。
// nil-safe：未注入则退化为无锁。
func (d *OfflineDetector) SetSequencer(seq *Sequencer) {
	d.seq = seq
}

// SetCache 注入 DeviceCache，UpdateOnlineStatus 后 invalidate（保持与
// DeviceService.UpdateFromInform 的 cache.GetOrLoad 一致）。nil-safe。
func (d *OfflineDetector) SetCache(cache *DeviceCache) {
	d.cache = cache
}

// Start 启动离线检测（阻塞运行）
func (d *OfflineDetector) Start(ctx context.Context) error {
	ticker := time.NewTicker(d.checkInterval)
	defer ticker.Stop()

	d.logger.Info("offline detector started",
		zap.Duration("interval", d.checkInterval),
		zap.Duration("threshold", d.offlineThreshold),
	)

	// 首次立即执行
	d.detect(ctx)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("offline detector stopped")
			return ctx.Err()
		case <-ticker.C:
			d.detect(ctx)
		}
	}
}

// detect 执行一次离线检测
func (d *OfflineDetector) detect(ctx context.Context) {
	start := time.Now()
	threshold := time.Now().Add(-d.offlineThreshold)

	// 查询超时设备
	devices, err := d.deviceRepo.FindStaleDevices(ctx, threshold, d.batchSize)
	if err != nil {
		d.logger.Error("find stale devices failed", zap.Error(err))
		return
	}

	if len(devices) == 0 {
		return
	}

	d.logger.Info("detected stale devices", zap.Int("count", len(devices)))

	// 批量处理离线设备
	successCount := 0
	for _, device := range devices {
		if err := d.markOffline(ctx, device); err != nil {
			d.logger.Error("mark device offline failed",
				zap.Error(err),
				zap.String("serial", device.SerialNumber),
			)
		} else {
			successCount++
		}
	}

	d.logger.Info("offline detection completed",
		zap.Int("total", len(devices)),
		zap.Int("success", successCount),
		zap.Duration("elapsed", time.Since(start)),
	)
}

// markOffline 标记设备离线
func (d *OfflineDetector) markOffline(ctx context.Context, device *model.Device) error {
	now := time.Now()

	// per-device 串行化：与 UpdateFromInform / marker 共享锁，确保不与同 device 的
	// 在飞事件并发写库（避免覆盖刚到的 Inform 写的 is_online=true）。
	if d.seq != nil {
		unlock := d.seq.Lock(device.SerialNumber)
		defer unlock()
	}

	// 1. T-0162: 只更 is_online=false，**不动 lifecycle_state**。
	// commissioned + is_online=false 是合法状态（已入网 + 当前掉线）。
	if err := d.deviceRepo.UpdateOnlineStatus(ctx, device.ID, false); err != nil {
		return fmt.Errorf("update online status: %w", err)
	}
	if d.cache != nil {
		d.cache.Delete(ctx, device.SerialNumber)
	}

	// 2. 记录离线时间
	if err := d.infoSyncer.RecordOffline(ctx, device.ID); err != nil {
		return fmt.Errorf("record offline time: %w", err)
	}

	// 3. 发布离线事件（用于告警通知）
	if d.eventBus != nil {
		evt, err := event.NewEvent("device.offline", DeviceOfflineEvent{
			DeviceID:    device.ID,
			Serial:      device.SerialNumber,
			Carrier:     device.Carrier,
			Technology:  device.Technology,
			OfflineTime: now,
		})
		if err == nil {
			_ = d.eventBus.Publish(ctx, "device.offline", evt)
		}
	}

	d.logger.Info("device marked offline",
		zap.String("serial", device.SerialNumber),
		zap.String("carrier", string(device.Carrier)),
		zap.Time("offline_time", now),
	)

	return nil
}

// SetCheckInterval 设置扫描间隔（用于测试）
func (d *OfflineDetector) SetCheckInterval(interval time.Duration) {
	d.checkInterval = interval
}

// SetOfflineThreshold 设置离线阈值（用于测试）
func (d *OfflineDetector) SetOfflineThreshold(threshold time.Duration) {
	d.offlineThreshold = threshold
}

// SetBatchSize 设置批量处理大小（用于测试）
func (d *OfflineDetector) SetBatchSize(size int) {
	d.batchSize = size
}
