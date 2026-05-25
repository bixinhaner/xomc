package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// PeriodicOnlineMarker 订阅 device.inform.periodic / device.inform.value_change，
// 在 sequencer 锁内即时把 is_online 写 true，并 invalidate cache。
//
// 设计动机（真机验证 2026-05-25 暴露的 batch 路径覆盖问题）：
//   - BatchInformProcessor 把 PERIODIC 处理时设的 device.IsOnline=true 累积 10s 后批量
//     UPDATE devices 表，会覆盖 RebootResponseOfflineMarker 在 10s 内写的 is_online=false；
//   - 解决思路（D 方案）：让 batch 路径不再写 is_online 列，由本订阅器独立路径
//     即时写 is_online=true（毫秒级延迟），与 RebootResponseOfflineMarker / OfflineDetector /
//     HeartbeatMonitor 共用 sequencer 锁 + cache.Delete 模式，事件序由 NATS subject 内 FIFO +
//     sequencer 锁队列共同保证。
//
// 触发时机：每次 PERIODIC / VALUE_CHANGE Inform 到达后异步处理。
// BOOT / Bootstrap / M Reboot 走 InformHandler.handleRebootComplete → DeviceService.UpdateFromInform
// 的同步路径，不走本订阅器（事件主题不同 → device.inform.reboot_complete / .bootstrap）。
//
// 性能：100K 设备 PERIODIC 60s 一次 ≈ 1700 UPDATE/s，pgxpool 默认配置可承受。
// 未来如压力上来可加 "cache.Get(sn).IsOnline == true 时跳过 DB 写" 短路（暂不做）。
type PeriodicOnlineMarker struct {
	repo   RebootOfflineDeviceRepo // 复用 GetBySerialNumber + UpdateOnlineStatus 最小接口
	seq    *Sequencer
	cache  *DeviceCache
	logger *zap.Logger
}

// periodic / value_change 各用独立 queue group——NATS JetStream 不允许同一
// consumer 跨 subject 订阅（subject does not match consumer），所以拆开两个。
// 两个 group 各自独立 dedupe（同 group 内多实例只一个处理）；跨 group 行为相同
// 因为两个 subject 互斥（同一 Inform 不会同时是 PERIODIC 和 VALUE_CHANGE）。
const (
	periodicOnlineMarkerPeriodicQueue    = "device-periodic-online-marker-periodic"
	periodicOnlineMarkerValueChangeQueue = "device-periodic-online-marker-valuechange"
)

// periodicOnlinePayload 取 InformEventPayload 子集，本订阅器仅消费 device_sn。
type periodicOnlinePayload struct {
	DeviceId struct {
		SerialNumber string `json:"SerialNumber"`
	} `json:"device_id"`
}

// NewPeriodicOnlineMarker 构造订阅器。seq / cache 允许 nil（仅 test 场景），
// 生产部署必须注入两者，否则 D 方案的竞态防护不完整。
func NewPeriodicOnlineMarker(repo RebootOfflineDeviceRepo, seq *Sequencer, cache *DeviceCache, logger *zap.Logger) *PeriodicOnlineMarker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PeriodicOnlineMarker{
		repo:   repo,
		seq:    seq,
		cache:  cache,
		logger: logger.Named("device.periodic-online-marker"),
	}
}

// Subscribe 注册到事件总线。订阅 PERIODIC + VALUE_CHANGE 两个 subject，
// 各用独立 queue group（NATS JetStream consumer-subject 1:1 约束）。
func (m *PeriodicOnlineMarker) Subscribe(bus event.EventBus) error {
	if _, err := bus.QueueSubscribe(event.SubjectDevicePeriodic, periodicOnlineMarkerPeriodicQueue, m.handle); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDevicePeriodic, err)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDeviceValueChange, periodicOnlineMarkerValueChangeQueue, m.handle); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceValueChange, err)
	}
	m.logger.Info("periodic online marker subscribed",
		zap.Strings("subjects", []string{event.SubjectDevicePeriodic, event.SubjectDeviceValueChange}))
	return nil
}

func (m *PeriodicOnlineMarker) handle(ctx context.Context, evt event.Event) error {
	var payload periodicOnlinePayload
	if err := evt.DecodePayload(&payload); err != nil {
		m.logger.Warn("decode periodic/value_change payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		return nil
	}
	sn := payload.DeviceId.SerialNumber
	if sn == "" {
		return nil
	}

	if m.seq != nil {
		unlock := m.seq.Lock(sn)
		defer unlock()
	}

	dev, err := m.repo.GetBySerialNumber(ctx, sn)
	if err != nil {
		m.logger.Warn("lookup device by SN failed",
			zap.String("device_sn", sn), zap.Error(err))
		return nil
	}
	if dev == nil {
		return nil
	}

	if err := m.repo.UpdateOnlineStatus(ctx, dev.ID, true); err != nil {
		m.logger.Warn("mark device online on periodic/value_change failed",
			zap.String("device_sn", sn),
			zap.String("device_id", dev.ID.String()),
			zap.Error(err))
		return nil
	}
	if m.cache != nil {
		m.cache.Delete(ctx, sn)
	}
	m.logger.Debug("device marked online on periodic/value_change",
		zap.String("device_sn", sn),
		zap.String("device_id", dev.ID.String()))
	return nil
}

// Compile-time assertion that uuid is used (silence unused import in some build configs).
var _ = uuid.Nil
