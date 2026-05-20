package device

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// HeartbeatMonitor tracks device liveness via Redis TTL keys and marks
// expired devices as offline.
type HeartbeatMonitor struct {
	redis      redis.UniversalClient
	deviceRepo DeviceRepository
	cron       *cron.Cron
	logger     *zap.Logger
	cancel     context.CancelFunc
}

// NewHeartbeatMonitor creates a new HeartbeatMonitor.
func NewHeartbeatMonitor(redis redis.UniversalClient, deviceRepo DeviceRepository, logger *zap.Logger) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		redis:      redis,
		deviceRepo: deviceRepo,
		logger:     logger,
	}
}

// Start begins the periodic heartbeat check using a cron scheduler.
func (m *HeartbeatMonitor) Start() {
	baseCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.cron = cron.New()
	m.cron.AddFunc("@every 60s", func() {
		ctx, timeoutCancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer timeoutCancel()
		m.CheckHeartbeats(ctx)
	})
	m.cron.Start()
	m.logger.Info("heartbeat monitor started")
}

// Stop shuts down the cron scheduler and cancels any in-flight heartbeat checks.
func (m *HeartbeatMonitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.cron != nil {
		m.cron.Stop()
	}
}

// RefreshHeartbeat sets/refreshes a Redis key with TTL = 2 * informInterval.
func (m *HeartbeatMonitor) RefreshHeartbeat(ctx context.Context, deviceSN string, informInterval int) {
	key := redisx.Keys.ACSHeartbeat(deviceSN)
	ttl := time.Duration(informInterval*2) * time.Second
	if ttl < 60*time.Second {
		ttl = 600 * time.Second // minimum 10 minutes
	}

	if err := m.redis.Set(ctx, key, time.Now().Unix(), ttl).Err(); err != nil {
		m.logger.Error("refresh heartbeat", zap.Error(err), zap.String("device_sn", deviceSN))
	}
}

// CheckHeartbeats scans active devices and marks those with expired heartbeats as offline.
// Uses keyset (cursor) pagination ordered by last_inform_at ASC to:
//  1. Prioritize checking devices that haven't reported in the longest time.
//  2. Avoid the sliding-window problem where OFFSET-based pagination skips
//     devices when earlier rows are mutated (marked offline) during iteration.
func (m *HeartbeatMonitor) CheckHeartbeats(ctx context.Context) {
	const batchSize = 100
	var (
		cursorTime *time.Time
		cursorID   *uuid.UUID
		checked    int
	)

	for {
		devices, err := m.deviceRepo.ListActiveByLastInform(ctx, cursorTime, cursorID, batchSize)
		if err != nil {
			m.logger.Error("list active devices for heartbeat check", zap.Error(err))
			return
		}

		for _, device := range devices {
			key := redisx.Keys.ACSHeartbeat(device.SerialNumber)
			exists, err := m.redis.Exists(ctx, key).Result()
			if err != nil {
				m.logger.Error("check heartbeat key",
					zap.Error(err),
					zap.String("device_sn", device.SerialNumber))
				continue
			}

			if exists == 0 {
				// T-0162: 只更 is_online=false，**不动 lifecycle_state**。
				// commissioned + is_online=false 是合法状态（已入网 + 当前掉线）。
				// 老代码这里写 status='offline' 实际等价于"丢失 lifecycle 信息"，是
				// status 字段双重语义的典型 bug 现场。
				if err := m.deviceRepo.UpdateOnlineStatus(ctx, device.ID, false); err != nil {
					m.logger.Error("mark device offline",
						zap.Error(err),
						zap.String("device_sn", device.SerialNumber))
				} else {
					m.logger.Info("device marked offline (heartbeat expired)",
						zap.String("device_sn", device.SerialNumber),
						zap.String("device_id", device.ID.String()))
				}
			}
		}

		checked += len(devices)

		if len(devices) < batchSize {
			break
		}
		// Advance cursor to the last device in this batch.
		last := devices[len(devices)-1]
		cursorTime = last.LastInformAt
		cursorID = &last.ID
	}

	m.logger.Info("heartbeat check complete", zap.Int("devices_checked", checked))
}

