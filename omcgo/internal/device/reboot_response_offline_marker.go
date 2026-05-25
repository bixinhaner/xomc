package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// RebootResponseOfflineMarker 订阅 command.reboot.response（ACS 收到 CPE 的 RebootResponse
// SOAP 时发布），立即把对应设备的 is_online 标为 false。
//
// 设计动机（真机验证 2026-05-25 暴露）：
//   - CPE Reboot 大约 ~4 分钟，短于 OfflineDetector 的 10 分钟离线阈值（每 5 分钟扫一次）；
//   - 重启完成发 BOOT/M Reboot Inform 回连时，is_online 仍是 true → device_service.UpdateFromInform
//     里 `becameOnline := oldStatus==Offline && newStatus==Active` 判定不成立；
//   - 结果：device.online 事件不发，pm.OnlineSubscriber 不入队 PM SPV，
//     设备重启后 PM 上传配置不被重新下发（实际场景 CPE 可能恢复出厂、配置擦除）。
//
// 解法（最小侵入）：CPE 回 RebootResponse 即视作"设备即将进入离线窗口"，主动把
// is_online 写 false。等 BOOT Inform 回来时 oldStatus=Offline → device.online 自然触发
// → PM SPV 入队 → ACS 下发 → 完整重发 PM 配置。不动 device.online 语义，不动 PM 模块。
type RebootResponseOfflineMarker struct {
	repo   RebootOfflineDeviceRepo
	seq    *Sequencer
	cache  *DeviceCache // 写 DB 后 invalidate，否则 UpdateFromInform 走 cache 读到 stale device
	logger *zap.Logger
}

// RebootOfflineDeviceRepo 是订阅器需要的最小 device repository 能力子集。
// 真实实现是 *PgDeviceRepository。
type RebootOfflineDeviceRepo interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
	UpdateOnlineStatus(ctx context.Context, id uuid.UUID, isOnline bool) error
}

// rebootResponseOfflineMarkerQueue 让多 worker 实例共享同一 queue group，
// 同一事件只由一个实例处理（避免重复写库）。
const rebootResponseOfflineMarkerQueue = "device-reboot-response-offline-marker"

// rebootResponsePayload 是 acs/handler.go publishRPCResponseEvent 发的 payload 子集。
// 实际 payload 还含 method / command_key / path 等字段，本订阅器仅消费 device_sn。
type rebootResponsePayload struct {
	DeviceSN string `json:"device_sn"`
	Method   string `json:"method"`
}

// NewRebootResponseOfflineMarker 构造订阅器。
// seq 用于与 DeviceService.UpdateFromInform / OfflineDetector / HeartbeatMonitor
// 共享 per-device 锁，确保同一 device 的事件按 ACS publish 顺序串行处理，
// 避免 PERIODIC Inform 异步写 is_online=true 覆盖本订阅器写的 false（真机验证
// 2026-05-25 暴露的竞态）。seq=nil 时退化为无锁（仅 single-handler 测试场景）。
func NewRebootResponseOfflineMarker(repo RebootOfflineDeviceRepo, seq *Sequencer, logger *zap.Logger) *RebootResponseOfflineMarker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RebootResponseOfflineMarker{
		repo:   repo,
		seq:    seq,
		logger: logger.Named("device.reboot-response-offline-marker"),
	}
}

// SetCache 注入 DeviceCache，让 marker 写 DB 后同步 invalidate 缓存。
// 否则 DeviceService.UpdateFromInform 走 cache.GetOrLoad 会读到旧的
// is_online=true，导致 BOOT 处理时 oldStatus 误判为 Active，不发 device.online
// （真机验证 2026-05-25 暴露的缓存一致性缺口）。nil-safe。
func (m *RebootResponseOfflineMarker) SetCache(cache *DeviceCache) {
	m.cache = cache
}

// Subscribe 注册到事件总线。
func (m *RebootResponseOfflineMarker) Subscribe(bus event.EventBus) error {
	if _, err := bus.QueueSubscribe(event.SubjectCommandRebootResponse, rebootResponseOfflineMarkerQueue, m.handle); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectCommandRebootResponse, err)
	}
	m.logger.Info("reboot response offline marker subscribed",
		zap.String("subject", event.SubjectCommandRebootResponse),
		zap.String("queue", rebootResponseOfflineMarkerQueue))
	return nil
}

func (m *RebootResponseOfflineMarker) handle(ctx context.Context, evt event.Event) error {
	var payload rebootResponsePayload
	if err := evt.DecodePayload(&payload); err != nil {
		m.logger.Warn("decode reboot response payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		return nil
	}
	if payload.DeviceSN == "" {
		m.logger.Warn("reboot response payload missing device_sn",
			zap.String("event_id", evt.ID))
		return nil
	}

	if m.seq != nil {
		unlock := m.seq.Lock(payload.DeviceSN)
		defer unlock()
	}

	dev, err := m.repo.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		m.logger.Warn("lookup device by SN failed",
			zap.String("device_sn", payload.DeviceSN), zap.Error(err))
		return nil
	}
	if dev == nil {
		return nil
	}

	if err := m.repo.UpdateOnlineStatus(ctx, dev.ID, false); err != nil {
		m.logger.Warn("mark device offline on RebootResponse failed",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("device_id", dev.ID.String()),
			zap.Error(err))
		return nil
	}
	if m.cache != nil {
		m.cache.Delete(ctx, payload.DeviceSN)
	}
	m.logger.Info("device marked offline on RebootResponse",
		zap.String("device_sn", payload.DeviceSN),
		zap.String("device_id", dev.ID.String()),
		zap.String("method", payload.Method))
	return nil
}
