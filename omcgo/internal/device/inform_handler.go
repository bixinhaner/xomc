package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// HeartbeatGroupAssigner 是 inform_handler 的"匹配 + 自动分组"消费侧接口。
//
// 与 device_service.go 既有的 GroupAssigner 接口（BatchAddDevices）刻意分开命名：
// 那个服务于"管理面批量加设备"语义，本接口服务于"心跳时按规则自动归组"。
//
// 仅暴露 1 个方法，避免 device 包反向依赖 topology。在 modules.go wiring 时把
// topology.HeartbeatAssigner 包装成符合本接口的本地 struct。
type HeartbeatGroupAssigner interface {
	AssignDeviceToGroup(ctx context.Context, req GroupAssignRequest) error
}

// GroupAssignRequest — 匹配请求 DTO。
// 字段刻意与 topology.MatchRequest 对应，方便 adapter 直接转发。
type GroupAssignRequest struct {
	DeviceID     uuid.UUID
	DeviceName   string
	SerialNumber string
	LAC          *int
	TAC          *int
}

// InformEventPayload is the structure published by the ACS handler on device.inform.* events.
type InformEventPayload struct {
	DeviceId      tr069.DeviceId               `json:"device_id"`
	Events        []string                     `json:"events"`
	ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
	CurrentTime   string                       `json:"current_time"`
	RetryCount    int                          `json:"retry_count"`
}

// InformHandler subscribes to device Inform events and routes them to DeviceService.
type InformHandler struct {
	service         *DeviceService
	batchProcessor  *BatchInformProcessor
	carrierRegistry *carrier.CarrierRegistry
	defaultCarrier  model.CarrierCode
	groupAssigner   HeartbeatGroupAssigner // 心跳路径自动分组钩子（nil = 禁用）
	logger          *zap.Logger
}

// NewInformHandler creates a new InformHandler.
func NewInformHandler(
	service *DeviceService,
	carrierRegistry *carrier.CarrierRegistry,
	defaultCarrier model.CarrierCode,
	logger *zap.Logger,
) *InformHandler {
	return &InformHandler{
		service:         service,
		carrierRegistry: carrierRegistry,
		defaultCarrier:  defaultCarrier,
		logger:          logger,
	}
}

// SetBatchProcessor enables batch processing for periodic Inform events.
func (h *InformHandler) SetBatchProcessor(bp *BatchInformProcessor) {
	h.batchProcessor = bp
}

// SetGroupAssigner 注入"心跳后自动分组"钩子（migration 000124+ / SN 规则匹配）。
// nil 等价于禁用（保持向后兼容；测试 / 不需要自动分组的部署可跳过 wiring）。
func (h *InformHandler) SetGroupAssigner(ga HeartbeatGroupAssigner) {
	h.groupAssigner = ga
}

// Subscribe registers event handlers on the event bus for device Inform events.
func (h *InformHandler) Subscribe(bus event.EventBus) error {
	h.logger.Info("subscribing to device inform events...")

	if _, err := bus.QueueSubscribe(event.SubjectDeviceBootstrap, "device-mgr-bootstrap", h.handleBootstrap); err != nil {
		return fmt.Errorf("subscribe bootstrap: %w", err)
	}
	h.logger.Info("subscribed to bootstrap events", zap.String("subject", event.SubjectDeviceBootstrap))

	if _, err := bus.QueueSubscribe(event.SubjectDevicePeriodic, "device-mgr-periodic", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe periodic: %w", err)
	}
	h.logger.Info("subscribed to periodic events", zap.String("subject", event.SubjectDevicePeriodic))

	if _, err := bus.QueueSubscribe(event.SubjectDeviceValueChange, "device-mgr-valuechange", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe value_change: %w", err)
	}
	h.logger.Info("subscribed to value_change events", zap.String("subject", event.SubjectDeviceValueChange))

	if _, err := bus.QueueSubscribe(event.SubjectDeviceRebootComplete, "device-mgr-reboot", h.handleRebootComplete); err != nil {
		return fmt.Errorf("subscribe reboot_complete: %w", err)
	}
	h.logger.Info("subscribed to reboot_complete events", zap.String("subject", event.SubjectDeviceRebootComplete))

	h.logger.Info("inform handler subscribed to device events successfully")
	return nil
}

func (h *InformHandler) handleBootstrap(ctx context.Context, evt event.Event) error {
	h.logger.Info("handleBootstrap: received bootstrap event",
		zap.String("event_id", evt.ID),
		zap.String("subject", evt.Subject),
		zap.Time("timestamp", evt.Timestamp))

	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		h.logger.Error("handleBootstrap: decode payload failed", zap.Error(err))
		return fmt.Errorf("decode bootstrap payload: %w", err)
	}

	h.logger.Info("handleBootstrap: payload decoded",
		zap.String("serial_number", payload.DeviceId.SerialNumber),
		zap.String("oui", payload.DeviceId.OUI),
		zap.String("product_class", payload.DeviceId.ProductClass),
		zap.Strings("events", payload.Events),
		zap.Int("param_count", len(payload.ParameterList)))

	inform := payloadToInform(payload)

	// Determine carrier from OUI via registry, falling back to default
	carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
	h.logger.Info("handleBootstrap: carrier resolved",
		zap.String("carrier", string(carrierCode)),
		zap.String("oui", payload.DeviceId.OUI))

	h.logger.Info("handleBootstrap: calling DeviceService.RegisterFromInform",
		zap.String("serial_number", payload.DeviceId.SerialNumber))

	device, err := h.service.RegisterFromInform(ctx, inform, carrierCode)
	if errors.Is(err, commonerrors.ErrNotFound) {
		h.logger.Info("handleBootstrap: device is in recycle bin, skipping auto-register",
			zap.String("serial_number", payload.DeviceId.SerialNumber),
			zap.String("carrier", string(carrierCode)))
		return nil
	}
	if err != nil {
		h.logger.Error("handleBootstrap: RegisterFromInform failed",
			zap.Error(err),
			zap.String("serial_number", payload.DeviceId.SerialNumber),
		)
		return err
	}

	h.logger.Info("handleBootstrap: device registered successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrierCode)),
	)

	// Always publish device.registered on BOOTSTRAP events,
	// so that provisioning engine triggers auto-discovery/sync
	// for both new and existing devices.
	h.service.PublishDeviceRegistered(ctx, device)

	return nil
}

// handleRebootComplete handles Inform events that indicate a device has
// rebooted and re-attached (event codes "1 BOOT" and/or "M Reboot") but is
// NOT a fresh bootstrap. Responsibilities:
//   - Ensure the device row exists (auto-register on the rare case a reboot
//     Inform arrives before bootstrap, e.g. ACS cache miss after restart);
//   - Update last_inform_at / status / IP / ConnectionRequestURL / firmware
//     via UpdateFromInform;
//   - Atomically increment boot_count and stamp last_boot_at;
//   - Publish device.reboot.abnormal for watchdog/crash reboots (no M Reboot).
//
// Unlike bootstrap it does NOT re-trigger the provisioning engine: a rebooted
// device keeps its previous configuration identity.
func (h *InformHandler) handleRebootComplete(ctx context.Context, evt event.Event) error {
	h.logger.Info("handleRebootComplete: received reboot event",
		zap.String("event_id", evt.ID),
		zap.String("subject", evt.Subject))

	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		h.logger.Error("handleRebootComplete: decode payload failed", zap.Error(err))
		return fmt.Errorf("decode reboot_complete payload: %w", err)
	}

	inform := payloadToInform(payload)
	sn := payload.DeviceId.SerialNumber

	device, err := h.service.GetBySerialNumber(ctx, sn)
	if err != nil {
		h.logger.Error("handleRebootComplete: device lookup failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	// preRebootRunTime：在 UpdateFromInform 覆盖 device_info.run_time 之前
	// 读取 DB 中存储的上次同步值，作为重启前设备运行时长（秒）写入重启记录。
	// 新设备首次入网无历史记录时为 0，正常写入（显示为 '-'）。
	var preRebootRunTime int64
	var preRebootDevice *model.Device
	if device == nil {
		h.logger.Info("handleRebootComplete: device not found, auto-registering",
			zap.String("serial_number", sn))
		carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
		registered, regErr := h.service.RegisterFromInform(ctx, inform, carrierCode)
		if errors.Is(regErr, commonerrors.ErrNotFound) {
			h.logger.Info("handleRebootComplete: device is in recycle bin, skipping auto-register",
				zap.String("serial_number", sn),
				zap.String("carrier", string(carrierCode)))
			return nil
		}
		if regErr != nil {
			h.logger.Error("handleRebootComplete: auto-register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		device = registered
		// First-time registration via reboot_complete path is equivalent to bootstrap;
		// publish device.registered so ProvisioningEngine can route productClass and
		// kick off Path B/C (auto-sync / model upload).
		h.service.PublishDeviceRegistered(ctx, device)
	} else {
		// 读重启前快照（在 UpdateFromInform 写入新值之前）。
		preRebootSnapshot := *device
		preRebootDevice = &preRebootSnapshot
		preRebootRunTime = h.service.GetDevicePreRebootRunTime(ctx, device.ID)

		// issue #212：收到 BOOT 即无条件强制驱动一次 "下线 → 上线" 翻转。
		// 先把设备显式置离线（即便当前显示在线），再由紧随其后的 UpdateFromInform
		// 把它带回在线 —— 这样 oldStatus==Offline 成立、device.online 必然发出，
		// OMC 上才会如实走出 "下线 → 上线" 过程。与被动超时离线探测彻底解耦：
		// 不读心跳超时链路、不被 "当前仍显示在线" 挡住。ForceBootStateFlip 内部对
		// 已离线设备幂等、且只翻转 is_online 不动 lifecycle。
		h.service.ForceBootStateFlip(ctx, device)

		updated, updErr := h.service.UpdateFromInform(ctx, inform)
		if updErr != nil {
			h.logger.Error("handleRebootComplete: UpdateFromInform failed",
				zap.Error(updErr), zap.String("serial_number", sn))
			return updErr
		}
		if updated != nil {
			device = updated
		}
	}

	if _, err := h.service.recordBootFromInform(ctx, device, preRebootDevice, payload.Events, payload.ParameterList, preRebootRunTime); err != nil {
		h.logger.Error("handleRebootComplete: RecordBootFromInform failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	return nil
}

func (h *InformHandler) handlePeriodic(ctx context.Context, evt event.Event) error {
	h.logger.Debug("handlePeriodic: received periodic event",
		zap.String("event_id", evt.ID),
		zap.String("subject", evt.Subject))

	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		h.logger.Error("handlePeriodic: decode payload failed", zap.Error(err))
		return fmt.Errorf("decode periodic payload: %w", err)
	}

	h.logger.Debug("handlePeriodic: payload decoded",
		zap.String("serial_number", payload.DeviceId.SerialNumber),
		zap.Strings("events", payload.Events))

	inform := payloadToInform(payload)
	sn := payload.DeviceId.SerialNumber

	// Step 1: Check if device exists (Redis cache → PostgreSQL fallback)
	device, err := h.service.GetBySerialNumber(ctx, sn)
	if err != nil {
		h.logger.Error("handlePeriodic: device lookup failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	// Step 2: Device not found → auto-register
	if device == nil {
		h.logger.Info("handlePeriodic: device not found, auto-registering",
			zap.String("serial_number", sn))

		carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
		registered, regErr := h.service.RegisterFromInform(ctx, inform, carrierCode)
		if errors.Is(regErr, commonerrors.ErrNotFound) {
			h.logger.Info("handlePeriodic: device is in recycle bin, skipping auto-register",
				zap.String("serial_number", sn),
				zap.String("carrier", string(carrierCode)))
			return nil
		}
		if regErr != nil {
			h.logger.Error("handlePeriodic: auto-register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		h.logger.Info("handlePeriodic: device auto-registered",
			zap.String("device_id", registered.ID.String()),
			zap.String("serial_number", registered.SerialNumber),
			zap.String("carrier", string(carrierCode)))
		h.service.PublishDeviceRegistered(ctx, registered)
		return nil
	}

	// Step 3: Device exists → update
	// 如果启用了批量处理器，走异步批量路径
	if h.batchProcessor != nil {
		// T-0123 / T-0125 batch path 补完：在 prepareDeviceUpdate 覆盖字段前捕获旧值，
		// 让 batch flush 完成后能基于 oldStatus / oldVersion 判断是否发
		// device.online / device.firmware.changed 事件（与 UpdateFromInform 非 batch 路径对齐）。
		oldStatus := device.Status
		oldVersion := device.FirmwareVersion
		params, _ := prepareDeviceUpdate(device, inform)
		h.batchProcessor.Submit(device, inform, params, oldStatus, oldVersion)
		// 心跳后自动分组（与非 batch 路径行为一致）。device.SN / DeviceName 跨心跳
		// 稳定，用 pre-update 快照触发即可，无需等待 batch flush。
		h.triggerGroupAssign(device)
		h.logger.Debug("handlePeriodic: submitted to batch processor",
			zap.String("device_id", device.ID.String()),
			zap.String("serial_number", device.SerialNumber))
		return nil
	}

	// 回退：原有逐条处理逻辑
	updated, err := h.service.UpdateFromInform(ctx, inform)
	if err != nil {
		h.logger.Error("handlePeriodic: UpdateFromInform failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	// Stale-cache fall-through: UpdateFromInform returned (nil, nil) because the
	// cache pointed at a row that has since been deleted from PG. Recover by
	// running auto-register so the device is re-created and downstream
	// provisioning fires (otherwise this device would be stuck forever).
	if updated == nil {
		h.logger.Info("handlePeriodic: stale cache fall-through, auto-registering",
			zap.String("serial_number", sn))
		carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
		registered, regErr := h.service.RegisterFromInform(ctx, inform, carrierCode)
		if errors.Is(regErr, commonerrors.ErrNotFound) {
			h.logger.Info("handlePeriodic: recycle-bin device skipped during stale-cache fall-through",
				zap.String("serial_number", sn),
				zap.String("carrier", string(carrierCode)))
			return nil
		}
		if regErr != nil {
			h.logger.Error("handlePeriodic: stale-cache fall-through register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		h.service.PublishDeviceRegistered(ctx, registered)
		return nil
	}

	// 心跳后自动分组（migration 000124：device_groups.matching_mode='serialNumber'
	// 等规则）。fire-and-forget goroutine 不阻塞心跳热路径；nil-safe。
	h.triggerGroupAssign(updated)

	h.logger.Debug("handlePeriodic: device updated",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))
	return nil
}

// triggerGroupAssign 异步触发设备分组匹配。
//
// 设计要点：
//   - fire-and-forget goroutine：不阻塞 inform 处理，匹配失败不影响心跳成功
//   - 独立 ctx + 5s 超时：脱离 NATS handler ctx 避免随消息生命周期取消
//   - nil-safe：groupAssigner 未注入时直接返回（dev/test 友好）
//   - 幂等：matcher.AssignDeviceToGroup 用 AddDevice upsert，已在同组重复调用无副作用
func (h *InformHandler) triggerGroupAssign(device *model.Device) {
	if h.groupAssigner == nil || device == nil {
		return
	}
	req := GroupAssignRequest{
		DeviceID:     device.ID,
		DeviceName:   device.DeviceName, // 与现有 matcher 约定 deviceName 字段对齐；若空则用 SN 兜底
		SerialNumber: device.SerialNumber,
	}
	if req.DeviceName == "" {
		req.DeviceName = device.SerialNumber
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.groupAssigner.AssignDeviceToGroup(ctx, req); err != nil {
			h.logger.Warn("auto group-assign failed (heartbeat path)",
				zap.String("device_id", device.ID.String()),
				zap.String("serial_number", device.SerialNumber),
				zap.Error(err))
		}
	}()
}

// resolveCarrier resolves the carrier code by looking up the OUI in the
// carrier registry's known OUI-ProductClass mappings. Falls back to defaultCarrier.
func (h *InformHandler) resolveCarrier(oui string) model.CarrierCode {
	if h.carrierRegistry != nil {
		if code := h.carrierRegistry.ResolveByOUI(oui); code != "" {
			return code
		}
	}
	return h.defaultCarrier
}

func payloadToInform(p InformEventPayload) *tr069.InformMessage {
	events := make([]tr069.EventStruct, 0, len(p.Events))
	for _, e := range p.Events {
		events = append(events, tr069.EventStruct{EventCode: e})
	}

	return &tr069.InformMessage{
		DeviceId:      p.DeviceId,
		Event:         events,
		ParameterList: p.ParameterList,
	}
}
