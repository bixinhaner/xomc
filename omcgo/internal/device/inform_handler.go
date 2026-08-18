package device

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

const (
	// A 20k-device deployment reporting every minute produces about 333 events/s.
	// Production measurements put one cached lookup + batch submission at roughly
	// 8 events/s per shard, so 64 ordered shards retain headroom while preserving
	// serialization for repeated events from the same device.
	periodicConsumerConcurrency = 64
	periodicConsumerQueueDepth  = 64
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
	EventStructs  []tr069.EventStruct          `json:"event_structs,omitempty"`
	ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
	CurrentTime   string                       `json:"current_time"`
	RetryCount    int                          `json:"retry_count"`
	RemoteIP      string                       `json:"remote_ip"`
	Authenticated bool                         `json:"authenticated"`
	AuthMethod    string                       `json:"auth_method"`
	CredentialID  string                       `json:"credential_id"`
}

// InformHandler subscribes to device Inform events and routes them to DeviceService.
type InformHandler struct {
	service         *DeviceService
	batchProcessor  *BatchInformProcessor
	carrierRegistry *carrier.CarrierRegistry
	defaultCarrier  model.CarrierCode
	accessGate      AccessGate
	groupAssigner   HeartbeatGroupAssigner // 兼容注入点；Periodic 不再触发，归组由领域事件负责
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

// SetGroupAssigner 保留旧 wiring 的源码兼容。Periodic Inform 不再调用该依赖；
// 首次注册和属性变化分别由 device.registered / device.attributes.changed 驱动归组。
func (h *InformHandler) SetGroupAssigner(ga HeartbeatGroupAssigner) {
	h.groupAssigner = ga
}

// licenseEnforcementRecoverable reports whether err is a recoverable license
// enforcement rejection: capacity exhausted / expired / unavailable（per-type
// 容量满、网元类型未授权、ne_type 未知、license 过期/未配置均归入此类）。
//
// 这些都是可恢复的运维状态——删设备腾容量、换更大 license、登记产品、续期后
// 设备即可接入。因此 Inform 消费侧对它们 Ack（不重试单条消息）：
//   - 单条 Inform 重试无意义（几秒内容量不会变化）；
//   - 永久 Term 又危险：容量腾出后设备应能在下次 periodic Inform（新消息）重新
//     注册，Term 会让它错过这次机会。
func licenseEnforcementRecoverable(err error) bool {
	return errors.Is(err, commonerrors.ErrLicenseCapacityExceeded) ||
		errors.Is(err, commonerrors.ErrLicenseExpired) ||
		errors.Is(err, commonerrors.ErrLicenseUnavailable)
}

// ackIfLicenseRejected 在注册被 license enforcement 拒绝（可恢复）时记一条 warn
// 并返回 true，调用方据此 return nil 让 NATS Ack 当前 Inform。其他错误返回 false
// 透传给调用方按原逻辑处理（Error 日志 + 触发 NATS 重试）。
func (h *InformHandler) ackIfLicenseRejected(err error, sn, handler string) bool {
	if !licenseEnforcementRecoverable(err) {
		return false
	}
	h.logger.Warn("auto-register skipped: rejected by license enforcement (recoverable; will retry on next inform)",
		zap.String("handler", handler),
		zap.String("serial_number", sn),
		zap.Error(err))
	return true
}

// SetAccessGate installs the business admission check used before formal
// registration and for reevaluating subsequent Inform sessions.
func (h *InformHandler) SetAccessGate(gate AccessGate) {
	h.accessGate = gate
}

// Subscribe registers event handlers on the event bus for device Inform events.
func (h *InformHandler) Subscribe(bus event.EventBus) error {
	h.logger.Info("subscribing to device inform events...")

	if _, err := bus.QueueSubscribe(event.SubjectDeviceBootstrap, "device-mgr-bootstrap", h.handleBootstrap); err != nil {
		return fmt.Errorf("subscribe bootstrap: %w", err)
	}
	h.logger.Info("subscribed to bootstrap events", zap.String("subject", event.SubjectDeviceBootstrap))

	if keyedBus, ok := bus.(keyedQueueEventBus); ok {
		if _, err := keyedBus.KeyedQueueSubscribe(
			event.SubjectDevicePeriodic,
			event.KeyedQueueConfig{
				Durable:     "device-mgr-periodic",
				Concurrency: periodicConsumerConcurrency,
				QueueDepth:  periodicConsumerQueueDepth,
			},
			periodicDeviceKey,
			h.handlePeriodic,
		); err != nil {
			return fmt.Errorf("subscribe keyed periodic: %w", err)
		}
	} else if _, err := bus.QueueSubscribe(event.SubjectDevicePeriodic, "device-mgr-periodic", h.handlePeriodic); err != nil {
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

func periodicDeviceKey(evt event.Event) (string, error) {
	var payload struct {
		DeviceID tr069.DeviceId `json:"device_id"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return "", fmt.Errorf("decode periodic device key: %w", err)
	}
	serialNumber := strings.TrimSpace(payload.DeviceID.SerialNumber)
	if serialNumber == "" {
		return "", errors.New("periodic device key has empty serial number")
	}
	return serialNumber, nil
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

	// Existing devices keep their persisted carrier. Only a genuinely new
	// device is auto-classified from its TR-069 identity; this prevents a later
	// Bootstrap with incomplete ProductClass from disrupting an onboarded site.
	existing, err := h.service.GetBySerialNumber(ctx, payload.DeviceId.SerialNumber)
	if err != nil {
		return fmt.Errorf("lookup bootstrap device before carrier resolution: %w", err)
	}
	carrierCode := model.CarrierCode("")
	if existing != nil {
		carrierCode = existing.Carrier
	} else {
		carrierCode, err = h.resolveCarrier(payload.DeviceId.OUI, payload.DeviceId.ProductClass)
		if err != nil {
			return fmt.Errorf("resolve bootstrap carrier: %w", err)
		}
	}
	h.logger.Info("handleBootstrap: carrier resolved",
		zap.String("carrier", string(carrierCode)),
		zap.String("oui", payload.DeviceId.OUI))

	h.logger.Info("handleBootstrap: calling DeviceService.RegisterFromInform",
		zap.String("serial_number", payload.DeviceId.SerialNumber))

	registration, admitted, err := h.registerFromInformIfAdmitted(ctx, payload, inform, carrierCode, evt.ID)
	if errors.Is(err, commonerrors.ErrNotFound) {
		h.logger.Info("handleBootstrap: device is in recycle bin, skipping auto-register",
			zap.String("serial_number", payload.DeviceId.SerialNumber),
			zap.String("carrier", string(carrierCode)))
		return nil
	}
	if err != nil {
		if h.ackIfLicenseRejected(err, payload.DeviceId.SerialNumber, "handleBootstrap") {
			return nil
		}
		h.logger.Error("handleBootstrap: RegisterFromInform failed",
			zap.Error(err),
			zap.String("serial_number", payload.DeviceId.SerialNumber),
		)
		return err
	}
	if !admitted {
		return nil
	}
	device := registration.Device

	h.logger.Info("handleBootstrap: device registered successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrierCode)),
	)

	// Always publish device.registered on BOOTSTRAP events,
	// so that provisioning engine triggers auto-discovery/sync
	// for both new and existing devices.
	if err := h.service.PublishDeviceRegistered(ctx, device, registration.Created, evt.ID); err != nil {
		return fmt.Errorf("publish device.registered: %w", err)
	}

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
//   - Publish device.reboot.abnormal when HaltReason matches the abnormal-reboot
//     rule for the device type.
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
		carrierCode, resolveErr := h.resolveCarrier(payload.DeviceId.OUI, payload.DeviceId.ProductClass)
		if resolveErr != nil {
			return fmt.Errorf("resolve reboot carrier: %w", resolveErr)
		}
		registration, admitted, regErr := h.registerFromInformIfAdmitted(ctx, payload, inform, carrierCode, evt.ID)
		if errors.Is(regErr, commonerrors.ErrNotFound) {
			h.logger.Info("handleRebootComplete: device is in recycle bin, skipping auto-register",
				zap.String("serial_number", sn),
				zap.String("carrier", string(carrierCode)))
			return nil
		}
		if regErr != nil {
			if h.ackIfLicenseRejected(regErr, sn, "handleRebootComplete") {
				return nil
			}
			h.logger.Error("handleRebootComplete: auto-register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		if !admitted {
			return nil
		}
		device = registration.Device
		// First-time registration via reboot_complete path is equivalent to bootstrap;
		// publish device.registered so ProvisioningEngine can route productClass and
		// kick off Path B/C (auto-sync / model upload).
		if err := h.service.PublishDeviceRegistered(ctx, device, registration.Created, evt.ID); err != nil {
			return fmt.Errorf("publish device.registered after reboot auto-register: %w", err)
		}
	} else {
		// 读重启前快照（在 UpdateFromInform 写入新值之前）。
		preRebootSnapshot := *device
		preRebootDevice = &preRebootSnapshot
		preRebootRunTime = h.service.GetDevicePreRebootRunTime(ctx, device.ID)
		if _, accessErr := h.evaluateInformAccess(ctx, payload, inform, device.Carrier, evt.ID); accessErr != nil {
			return fmt.Errorf("evaluate reboot access: %w", accessErr)
		}

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

		carrierCode, resolveErr := h.resolveCarrier(payload.DeviceId.OUI, payload.DeviceId.ProductClass)
		if resolveErr != nil {
			return fmt.Errorf("resolve periodic carrier: %w", resolveErr)
		}
		registration, admitted, regErr := h.registerFromInformIfAdmitted(ctx, payload, inform, carrierCode, evt.ID)
		if errors.Is(regErr, commonerrors.ErrNotFound) {
			h.logger.Info("handlePeriodic: device is in recycle bin, skipping auto-register",
				zap.String("serial_number", sn),
				zap.String("carrier", string(carrierCode)))
			return nil
		}
		if regErr != nil {
			if h.ackIfLicenseRejected(regErr, sn, "handlePeriodic") {
				return nil
			}
			h.logger.Error("handlePeriodic: auto-register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		if !admitted {
			return nil
		}
		registered := registration.Device
		h.logger.Info("handlePeriodic: device auto-registered",
			zap.String("device_id", registered.ID.String()),
			zap.String("serial_number", registered.SerialNumber),
			zap.String("carrier", string(carrierCode)))
		if err := h.service.PublishDeviceRegistered(ctx, registered, registration.Created, evt.ID); err != nil {
			return fmt.Errorf("publish device.registered after periodic auto-register: %w", err)
		}
		return nil
	}

	// Step 3: Device exists → update
	// Access state is evaluated independently from online/lifecycle state. A
	// rejected result freezes protected tasks but does not falsify the heartbeat
	// or force the formal device row offline.
	if _, accessErr := h.evaluateInformAccess(ctx, payload, inform, device.Carrier, evt.ID); accessErr != nil {
		return fmt.Errorf("evaluate periodic access: %w", accessErr)
	}

	// 如果启用了批量处理器，走异步批量路径
	if h.batchProcessor != nil {
		// T-0123 / T-0125 batch path 补完：在 prepareDeviceUpdate 覆盖字段前捕获旧值，
		// 让 batch flush 完成后能基于 oldStatus / oldVersion 判断是否发
		// device.online / device.firmware.changed 事件（与 UpdateFromInform 非 batch 路径对齐）。
		oldStatus := device.Status
		oldVersion := device.FirmwareVersion
		oldIsOnline := device.IsOnline
		params, _ := prepareDeviceUpdate(device, inform)
		// license 容量校验（在线口径，issue #316）：prepareDeviceUpdate 已无条件置
		// is_online=true；若设备从离线→在线且该类型在线容量已满，回滚为离线，避免
		// batch 路径绕过容量限制（生产默认启用 batch，此分支是设备上线主路径）。
		if !oldIsOnline && device.IsOnline {
			if capErr := h.service.EnforceOnlineCapacity(ctx, device.ProductClass); capErr != nil {
				device.IsOnline = false
				h.logger.Warn("handlePeriodic: device kept offline by license capacity (batch path)",
					zap.String("serial_number", sn), zap.Error(capErr))
			}
		}
		h.batchProcessor.Submit(device, inform, params, oldStatus, oldVersion)
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
		carrierCode, resolveErr := h.resolveCarrier(payload.DeviceId.OUI, payload.DeviceId.ProductClass)
		if resolveErr != nil {
			return fmt.Errorf("resolve stale-cache carrier: %w", resolveErr)
		}
		registration, admitted, regErr := h.registerFromInformIfAdmitted(ctx, payload, inform, carrierCode, evt.ID)
		if errors.Is(regErr, commonerrors.ErrNotFound) {
			h.logger.Info("handlePeriodic: recycle-bin device skipped during stale-cache fall-through",
				zap.String("serial_number", sn),
				zap.String("carrier", string(carrierCode)))
			return nil
		}
		if regErr != nil {
			if h.ackIfLicenseRejected(regErr, sn, "handlePeriodic(stale-cache)") {
				return nil
			}
			h.logger.Error("handlePeriodic: stale-cache fall-through register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		if !admitted {
			return nil
		}
		if err := h.service.PublishDeviceRegistered(ctx, registration.Device, registration.Created, evt.ID); err != nil {
			return fmt.Errorf("publish device.registered after stale-cache auto-register: %w", err)
		}
		return nil
	}

	h.logger.Debug("handlePeriodic: device updated",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))
	return nil
}

func (h *InformHandler) registerFromInformIfAdmitted(
	ctx context.Context,
	payload InformEventPayload,
	inform *tr069.InformMessage,
	carrierCode model.CarrierCode,
	eventID string,
) (*InformRegistration, bool, error) {
	decision, err := h.evaluateInformAccess(ctx, payload, inform, carrierCode, eventID)
	if err != nil {
		return nil, false, fmt.Errorf("admit device inform: %w", err)
	}
	if !accessDecisionAllowsRegistration(decision) {
		h.logger.Info("device inform held outside formal registration",
			zap.String("serial_number", inform.DeviceId.SerialNumber),
			zap.String("carrier", string(carrierCode)),
			zap.String("decision_state", decision.State),
			zap.String("reason_code", decision.ReasonCode),
		)
		return nil, false, nil
	}

	registration, err := h.service.RegisterFromInformEvent(ctx, inform, carrierCode, eventID)
	if err != nil {
		return nil, false, err
	}
	return registration, true, nil
}

func (h *InformHandler) evaluateInformAccess(
	ctx context.Context,
	payload InformEventPayload,
	inform *tr069.InformMessage,
	carrierCode model.CarrierCode,
	eventID string,
) (AccessDecision, error) {
	if h.accessGate == nil {
		return AccessDecision{State: AccessDecisionBypassed}, nil
	}
	return h.accessGate.Admit(ctx, AccessObservation{
		Carrier:         carrierCode,
		SerialNumber:    inform.DeviceId.SerialNumber,
		OUI:             inform.DeviceId.OUI,
		ProductClass:    inform.DeviceId.ProductClass,
		SoftwareVersion: findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion"),
		RemoteIP:        payload.RemoteIP,
		Authenticated:   payload.Authenticated,
		AuthMethod:      payload.AuthMethod,
		CredentialID:    payload.CredentialID,
		CarrierIdentityResolved: h.carrierIdentityResolved(
			inform.DeviceId.OUI,
			inform.DeviceId.ProductClass,
			carrierCode,
		),
		Inform:  inform,
		EventID: eventID,
	})
}

func (h *InformHandler) carrierIdentityResolved(oui, productClass string, expected model.CarrierCode) bool {
	if h.carrierRegistry == nil || strings.TrimSpace(string(expected)) == "" {
		return false
	}
	resolved, err := h.carrierRegistry.ResolveByIdentity(oui, productClass)
	return err == nil && resolved != "" && resolved == expected
}

// resolveCarrier resolves the carrier from the complete TR-069 device
// identity. Unknown identities retain the deployment default for historical
// compatibility; ambiguous shared OUIs return an error and never fall back.
func (h *InformHandler) resolveCarrier(oui, productClass string) (model.CarrierCode, error) {
	if h.carrierRegistry != nil {
		code, err := h.carrierRegistry.ResolveByIdentity(oui, productClass)
		if err != nil {
			return "", fmt.Errorf("resolve carrier by device identity: %w", err)
		}
		if code != "" {
			return code, nil
		}
	}
	return h.defaultCarrier, nil
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
