package device

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const heartbeatKeyPrefix = "acs:heartbeat:"

// HeartbeatMonitor tracks device liveness via Redis TTL keys and marks
// expired devices as offline.
type HeartbeatMonitor struct {
	redis      redis.UniversalClient
	deviceRepo DeviceRepository
	cron       *cron.Cron
	logger     *zap.Logger
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
	m.cron = cron.New()
	m.cron.AddFunc("@every 60s", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.CheckHeartbeats(ctx)
	})
	m.cron.Start()
	m.logger.Info("heartbeat monitor started")
}

// Stop shuts down the cron scheduler.
func (m *HeartbeatMonitor) Stop() {
	if m.cron != nil {
		m.cron.Stop()
	}
}

// RefreshHeartbeat sets/refreshes a Redis key with TTL = 2 * informInterval.
func (m *HeartbeatMonitor) RefreshHeartbeat(ctx context.Context, deviceSN string, informInterval int) {
	key := heartbeatKeyPrefix + deviceSN
	ttl := time.Duration(informInterval*2) * time.Second
	if ttl < 60*time.Second {
		ttl = 600 * time.Second // minimum 10 minutes
	}

	if err := m.redis.Set(ctx, key, time.Now().Unix(), ttl).Err(); err != nil {
		m.logger.Error("refresh heartbeat", zap.Error(err), zap.String("device_sn", deviceSN))
	}
}

// CheckHeartbeats scans active devices and marks those with expired heartbeats as offline.
func (m *HeartbeatMonitor) CheckHeartbeats(ctx context.Context) {
	// List all active devices
	filter := DeviceFilter{
		Status: statusPtr(model.DeviceActive),
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: 100,
		},
	}

	for {
		result, err := m.deviceRepo.List(ctx, filter)
		if err != nil {
			m.logger.Error("list active devices for heartbeat check", zap.Error(err))
			return
		}

		for _, device := range result.Items {
			key := heartbeatKeyPrefix + device.SerialNumber
			exists, err := m.redis.Exists(ctx, key).Result()
			if err != nil {
				m.logger.Error("check heartbeat key",
					zap.Error(err),
					zap.String("device_sn", device.SerialNumber))
				continue
			}

			if exists == 0 {
				// Heartbeat expired — mark offline
				if err := m.deviceRepo.UpdateStatus(ctx, device.ID, model.DeviceOffline); err != nil {
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

		if filter.Page >= result.TotalPages {
			break
		}
		filter.Page++
	}

	_ = fmt.Sprintf("heartbeat check complete") // avoid unused import
}

func statusPtr(s model.DeviceStatus) *model.DeviceStatus {
	return &s
}
