package alarm

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

const (
	DefaultOfflineAlarmCleanupInterval  = 5 * time.Minute
	DefaultOfflineAlarmCleanupThreshold = time.Hour
	DefaultOfflineAlarmCleanupBatchSize = 200

	offlineAlarmCleanupClearedBy = "system:offline_timeout"
)

type offlineAlarmCleanupDeviceRepo interface {
	FindOfflineDevicesBefore(ctx context.Context, cutoff time.Time, limit int) ([]*model.Device, error)
}

type offlineAlarmCleanupStore interface {
	GetActiveByDeviceSN(ctx context.Context, deviceSN string) ([]*model.Alarm, error)
}

type offlineAlarmClearer interface {
	ClearBySync(ctx context.Context, alarm *model.Alarm) error
}

// OfflineAlarmCleaner 周期扫描离线超时设备，把其当前告警转入历史告警。
//
// 触发条件：设备当前仍离线，且 last_offline_time 距今超过阈值（默认 1 小时）。
// 清理动作复用 AlarmEngine.ClearBySync，确保活动告警库、历史告警库、Redis 去重键
// 和活跃告警指标一起收敛，避免只删活动表留下运行态漂移。
type OfflineAlarmCleaner struct {
	deviceRepo offlineAlarmCleanupDeviceRepo
	store      offlineAlarmCleanupStore
	clearer    offlineAlarmClearer
	logger     *zap.Logger

	interval  time.Duration
	threshold time.Duration
	batchSize int
}

func NewOfflineAlarmCleaner(
	deviceRepo offlineAlarmCleanupDeviceRepo,
	store offlineAlarmCleanupStore,
	clearer offlineAlarmClearer,
	logger *zap.Logger,
) *OfflineAlarmCleaner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OfflineAlarmCleaner{
		deviceRepo: deviceRepo,
		store:      store,
		clearer:    clearer,
		logger:     logger,
		interval:   DefaultOfflineAlarmCleanupInterval,
		threshold:  DefaultOfflineAlarmCleanupThreshold,
		batchSize:  DefaultOfflineAlarmCleanupBatchSize,
	}
}

// Run 启动后台扫描循环。首次立即执行一次，之后按 interval 周期扫描。
func (c *OfflineAlarmCleaner) Run(ctx context.Context) {
	if c == nil {
		return
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.sweep(ctx)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("offline alarm cleaner stopped")
			return
		case <-ticker.C:
			c.sweep(ctx)
		}
	}
}

func (c *OfflineAlarmCleaner) sweep(ctx context.Context) {
	cutoff := time.Now().Add(-c.threshold)
	devices, err := c.deviceRepo.FindOfflineDevicesBefore(ctx, cutoff, c.batchSize)
	if err != nil {
		c.logger.Error("find offline devices for alarm cleanup failed", zap.Error(err))
		return
	}
	if len(devices) == 0 {
		return
	}

	var (
		clearedCount int
		deviceCount  int
		failureCount int
	)
	for _, device := range devices {
		cleared, clearErr := c.clearDeviceAlarms(ctx, device)
		if clearErr != nil {
			failureCount++
			c.logger.Error("clear offline device alarms failed",
				zap.Error(clearErr),
				zap.String("device_id", device.ID.String()),
				zap.String("serial_number", device.SerialNumber),
			)
			continue
		}
		if cleared > 0 {
			deviceCount++
			clearedCount += cleared
		}
	}

	if clearedCount == 0 && failureCount == 0 {
		return
	}
	c.logger.Info("offline alarm cleanup pass complete",
		zap.Int("offline_devices_scanned", len(devices)),
		zap.Int("devices_cleared", deviceCount),
		zap.Int("alarms_cleared", clearedCount),
		zap.Int("failures", failureCount),
		zap.Duration("threshold", c.threshold),
	)
}

func (c *OfflineAlarmCleaner) clearDeviceAlarms(ctx context.Context, device *model.Device) (int, error) {
	alarms, err := c.store.GetActiveByDeviceSN(ctx, device.SerialNumber)
	if err != nil {
		return 0, fmt.Errorf("get active alarms by device sn: %w", err)
	}
	if len(alarms) == 0 {
		return 0, nil
	}

	clearedBy := offlineAlarmCleanupClearedBy
	clearNote := fmt.Sprintf("device offline for over %s", c.threshold.Round(time.Minute))
	clearedCount := 0
	for _, alarm := range alarms {
		alarm.ClearedBy = &clearedBy
		alarm.ClearNote = &clearNote
		if err := c.clearer.ClearBySync(ctx, alarm); err != nil {
			return clearedCount, fmt.Errorf("clear alarm %s: %w", alarm.ID, err)
		}
		clearedCount++
	}
	return clearedCount, nil
}

func (c *OfflineAlarmCleaner) SetInterval(d time.Duration) {
	if d > 0 {
		c.interval = d
	}
}

func (c *OfflineAlarmCleaner) SetThreshold(d time.Duration) {
	if d > 0 {
		c.threshold = d
	}
}

func (c *OfflineAlarmCleaner) SetBatchSize(n int) {
	if n > 0 {
		c.batchSize = n
	}
}

// Interval 返回当前生效的扫描周期（含 SetInterval 注入后的值），供启动日志/测试断言用。
func (c *OfflineAlarmCleaner) Interval() time.Duration { return c.interval }

// Threshold 返回当前生效的离线收敛阈值（含 SetThreshold 注入后的值），供启动日志/测试断言用。
func (c *OfflineAlarmCleaner) Threshold() time.Duration { return c.threshold }

// BatchSize 返回当前生效的每轮扫描批量上限（含 SetBatchSize 注入后的值），供启动日志/测试断言用。
func (c *OfflineAlarmCleaner) BatchSize() int { return c.batchSize }
