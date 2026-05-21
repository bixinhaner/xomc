package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// GroupAssigner assigns a device to a group.
type GroupAssigner interface {
	BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
}

// DeviceService provides business logic for device management.
type DeviceService struct {
	deviceRepo       DeviceRepository
	paramRepo        DeviceParameterRepository
	deviceInfoRepo   DeviceInfoRepository
	regRepo          RegistrationRepository
	groupAssigner    GroupAssigner
	infoSyncer       *InfoSyncer
	heartbeat        *HeartbeatMonitor
	eventBus         event.EventBus
	taskSvc          task.Enqueuer
	connReq          ConnectionRequester
	stunUpdater      StunAddressUpdater
	cache            *DeviceCache
	metrics          *DeviceMetrics
	licenseEnforcer  LicenseEnforcer
	carrierRegistry  *carrier.CarrierRegistry // T-0029: RF control path lookup by carrier+tech
	paramSyncStarter ParamSyncStarter         // T-0126: 注入 *provision.SyncService 触发 Path B 手动同步
	abnormalRecorder  AbnormalRebootRecorder // T-0158: 异常重启识别即落库（nil = 禁用）
	bootEventRecorder BootEventRecorder      // 普通 1 BOOT 事件日志写入（nil = 禁用）
	logger            *zap.Logger
}

// LicenseEnforcer is the narrow interface DeviceService consumes from the
// license package. Defined here on the consumer side so DeviceService stays
// independent of the full license model. Wired via SetLicenseEnforcer.
//
// Both methods may be called as nil-safe gates: SetLicenseEnforcer with a
// nil value is fine and disables enforcement (used in dev/test).
type LicenseEnforcer interface {
	EnforceCapacity(ctx context.Context, additional int) error
	EnforceExpiry(ctx context.Context, operation string) error
}

// ConnectionRequester sends Connection Request to wake a CPE device.
type ConnectionRequester interface {
	Send(ctx context.Context, deviceSN string, url string) error
}

// StunAddressUpdater syncs device STUN addresses to the address cache.
type StunAddressUpdater interface {
	SetFromInform(ctx context.Context, deviceSN, udpAddr string) error
}

// NewDeviceService creates a new DeviceService.
func NewDeviceService(
	deviceRepo DeviceRepository,
	paramRepo DeviceParameterRepository,
	heartbeat *HeartbeatMonitor,
	eventBus event.EventBus,
	logger *zap.Logger,
) *DeviceService {
	return &DeviceService{
		deviceRepo: deviceRepo,
		paramRepo:  paramRepo,
		heartbeat:  heartbeat,
		eventBus:   eventBus,
		logger:     logger,
	}
}

// SetTaskService sets the unified task service used to enqueue RPC commands.
func (s *DeviceService) SetTaskService(t task.Enqueuer) {
	s.taskSvc = t
}

// SetCarrierRegistry wires the Carrier adapter registry (T-0029). Used by
// SetRFSwitch (and future carrier-aware paths) to resolve RF control paths
// per (carrier, technology) instead of hardcoding TR-181 paths in device
// code. nil-safe: when unset, SetRFSwitch returns a clear error rather
// than queue an unkeyed SetParameterValues.
func (s *DeviceService) SetCarrierRegistry(r *carrier.CarrierRegistry) {
	s.carrierRegistry = r
}

// SetConnectionRequester sets the connection request client.
func (s *DeviceService) SetConnectionRequester(cr ConnectionRequester) {
	s.connReq = cr
}

// SetStunAddressUpdater sets the STUN address updater for syncing
// UDPConnectionRequestAddress from Inform to the STUN address cache.
func (s *DeviceService) SetStunAddressUpdater(u StunAddressUpdater) {
	s.stunUpdater = u
}

// SetDeviceCache sets the Redis device cache for fast serial number lookups.
func (s *DeviceService) SetDeviceCache(c *DeviceCache) {
	s.cache = c
}

// SetDeviceInfoRepo sets the device info repository for extended info management.
func (s *DeviceService) SetDeviceInfoRepo(repo DeviceInfoRepository) {
	s.deviceInfoRepo = repo
}

// SetInfoSyncer sets the parameter-to-device_info syncer.
func (s *DeviceService) SetInfoSyncer(syncer *InfoSyncer) {
	s.infoSyncer = syncer
}

// SetRegistrationRepo sets the device registration repository for pre-registration lookup.
func (s *DeviceService) SetRegistrationRepo(repo RegistrationRepository) {
	s.regRepo = repo
}

// SetGroupAssigner sets the group assigner for assigning devices to groups.
func (s *DeviceService) SetGroupAssigner(ga GroupAssigner) {
	s.groupAssigner = ga
}

// BatchAssignToGroup 把一批设备成员关系一次写入指定分组（UPSERT，device_id 冲
// 突时改写为新 groupID）。供 BatchImportDevices 在 CreateDevice 全部跑完后批
// 量归组使用——避免 N 行 N 次单写。
// 未注入 GroupAssigner 时返回 nil（dev/test 旁路），调用方按"分组写入失败但
// 设备已落库"对待。
func (s *DeviceService) BatchAssignToGroup(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) error {
	if s.groupAssigner == nil || len(deviceIDs) == 0 {
		return nil
	}
	_, err := s.groupAssigner.BatchAddDevices(ctx, groupID, deviceIDs)
	if err != nil {
		return fmt.Errorf("batch assign devices to group %s: %w", groupID, err)
	}
	return nil
}

// SetLicenseEnforcer wires the license enforcer used by CreateDevice to gate
// against capacity/expiry. Pass nil to disable (default in tests).
func (s *DeviceService) SetLicenseEnforcer(e LicenseEnforcer) {
	s.licenseEnforcer = e
}

// RebootDevice queues a Reboot command for the given device via the ACS command queue.
func (s *DeviceService) RebootDevice(ctx context.Context, id uuid.UUID) error {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get device for reboot: %w", err)
	}
	if device == nil {
		return commonerrors.ErrNotFound
	}

	if s.taskSvc == nil {
		return fmt.Errorf("task service not configured")
	}

	commandKey := uuid.New().String()
	created, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   device.SerialNumber,
		Method:     "Reboot",
		Priority:   0, // highest priority
		CommandKey: commandKey,
		Source:     task.TaskSourceAPI,
	})
	if err != nil {
		return fmt.Errorf("queue reboot command: %w", err)
	}

	s.logger.Info("reboot command queued",
		zap.String("device_id", id.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("command_id", created.ID),
	)

	// Optionally trigger Connection Request to wake the device immediately
	if s.connReq != nil && device.ConnectionRequestURL != "" {
		go func() {
			if err := s.connReq.Send(context.Background(), device.SerialNumber, device.ConnectionRequestURL); err != nil {
				s.logger.Warn("connection request for reboot failed",
					zap.String("serial_number", device.SerialNumber),
					zap.Error(err),
				)
			}
		}()
	}

	return nil
}

// SyncDeviceParamsManual T-0126: 手动触发设备参数 Path B 全量同步。
//
// 流程：
//  1. 查 device（404 if not found）
//  2. 通过 ParamSyncStarter 调 Path B（reason="manual"）
//  3. used=false 时返 ErrServiceUnavailable（Path B 不可用 — MappingSet 缺失）
//  4. 后台异步唤醒设备（已有 connReq 链路）
//
// 替代旧 TriggerParamSync 方法（Path A 已下线），完整接入 T-0123/T-0124/T-0125/T-0127
// 触发链：reason 通道 + 差异日志 + last_param_sync_at 回写 + Translator 翻译。
//
// sourceID 由 caller 构造（"manual:UUID"），供 HandleSyncResultPathB 写差异日志时
// 通过 Redis hint 读取 reason 标签。
func (s *DeviceService) SyncDeviceParamsManual(ctx context.Context, deviceID uuid.UUID, sourceID string) (used bool, dev *model.Device, err error) {
	dev, err = s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return false, nil, fmt.Errorf("get device for manual sync: %w", err)
	}
	if dev == nil {
		return false, nil, commonerrors.ErrNotFound
	}
	if s.paramSyncStarter == nil {
		return false, dev, fmt.Errorf("paramSyncStarter not configured")
	}

	used, err = s.paramSyncStarter.StartManualSync(ctx, dev, sourceID)
	if err != nil {
		return used, dev, fmt.Errorf("start manual sync: %w", err)
	}

	s.logger.Info("manual sync requested",
		zap.String("device_id", deviceID.String()),
		zap.String("serial_number", dev.SerialNumber),
		zap.String("source_id", sourceID),
		zap.Bool("path_b_used", used))

	// 唤醒设备（与旧 TriggerParamSync 一致；Connection Request 仅在 Path B 入队成功后发起）
	if used && s.connReq != nil && dev.ConnectionRequestURL != "" {
		sn := dev.SerialNumber
		url := dev.ConnectionRequestURL
		go func() {
			_ = s.connReq.Send(context.Background(), sn, url)
		}()
	}

	return used, dev, nil
}

// SetParamSyncStarter T-0126: 注入 Path B 同步 starter（消费者驱动 narrow interface）。
// 唯一实现者 *provision.SyncService。nil 时 SyncDeviceParamsManual 会返错。
func (s *DeviceService) SetParamSyncStarter(starter ParamSyncStarter) {
	s.paramSyncStarter = starter
}

// SetRFSwitch queues a SetParameterValues command to enable/disable the
// device RF. The TR-181 path is resolved through the Carrier adapter
// (T-0029 / R-201) so:
//
//   - CMCC/CTCC use FAPControl.{LTE,NR}.AdminState by technology;
//   - CUCC supports NR only — LTE returns "" and SetRFSwitch fails fast;
//   - future carrier-specific paths (e.g. X_VENDOR_*) plug into the
//     respective adapter's RFControlPath without changing this method.
//
// SetCarrierRegistry must be wired (DI). When the registry is nil or the
// carrier/tech combination has no path, the method returns a clear error
// rather than queuing an unkeyed command.
func (s *DeviceService) SetRFSwitch(ctx context.Context, deviceID uuid.UUID, enabled bool) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device for RF switch: %w", err)
	}
	if device == nil {
		return commonerrors.ErrNotFound
	}
	if s.taskSvc == nil {
		return fmt.Errorf("task service not configured")
	}
	if s.carrierRegistry == nil {
		return fmt.Errorf("carrier registry not configured (T-0029 DI gap)")
	}

	c, err := s.carrierRegistry.Get(device.Carrier)
	if err != nil {
		return fmt.Errorf("resolve carrier=%s for RF switch: %w", device.Carrier, err)
	}
	rfPath := c.RFControlPath(device.Technology)
	if rfPath == "" {
		return fmt.Errorf("carrier=%s does not support RF control for technology=%s: %w",
			device.Carrier, device.Technology, commonerrors.ErrInvalidInput)
	}

	// RF switch value: "1" for enabled, "0" for disabled.
	value := "0"
	if enabled {
		value = "1"
	}

	rfParamsJSON, _ := json.Marshal(map[string]interface{}{
		"ParameterList": []map[string]string{
			{"Name": rfPath, "Value": value},
		},
	})
	commandKey := uuid.New().String()
	if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   device.SerialNumber,
		Method:     "SetParameterValues",
		Params:     rfParamsJSON,
		CommandKey: commandKey,
		Source:     task.TaskSourceAPI,
	}); err != nil {
		return fmt.Errorf("queue RF switch command: %w", err)
	}

	s.logger.Info("RF switch command queued",
		zap.String("device_id", deviceID.String()),
		zap.Bool("enabled", enabled))

	// Wake the device.
	if s.connReq != nil && device.ConnectionRequestURL != "" {
		go func() {
			s.connReq.Send(context.Background(), device.SerialNumber, device.ConnectionRequestURL)
		}()
	}

	return nil
}

// SetParameters queues a SetParameterValues RPC command for the given device.
// 返回任务 ID(T-0146:供前端用 useTaskStatus 轮询真实 CPE 应答状态)。
// creatorID: T-0157 C5 — 任务发起人（JWT user_id），用于消息中心按用户隔离；空串表示系统任务。
func (s *DeviceService) SetParameters(ctx context.Context, deviceID uuid.UUID, params []ParameterValueItem, creatorID string) (string, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return "", fmt.Errorf("get device for set params: %w", err)
	}
	if device == nil {
		return "", commonerrors.ErrNotFound
	}

	if s.taskSvc == nil {
		return "", fmt.Errorf("task service not configured")
	}

	// Build SPV parameter list.
	// 注意:ACS dispatcher `SetParameterValuesHandler.BuildRequest`(rpc/dispatcher.go:103)
	// 期望 JSON 字段名 `values`(不是 parameter_list)。字段错配会导致 json.Unmarshal 静默
	// 拿到空 slice → SOAP ParameterList 渲染为空 → CPE 收到空命令但 Status=0 应答。
	// (T-0147 真机 TR069 报文跟踪发现此 bug)。
	spvParams := make([]map[string]string, 0, len(params))
	for _, p := range params {
		paramType := p.Type
		if paramType == "" {
			paramType = "xsd:string"
		}
		spvParams = append(spvParams, map[string]string{
			"name":  p.Path,
			"value": p.Value,
			"type":  paramType,
		})
	}

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"values":        spvParams,
		"parameter_key": fmt.Sprintf("ui-spv-%d", time.Now().Unix()),
	})
	if err != nil {
		return "", fmt.Errorf("marshal SPV params: %w", err)
	}

	createdTask, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   device.SerialNumber,
		Method:     "SetParameterValues",
		Params:     paramsJSON,
		Priority:   5,
		CommandKey: fmt.Sprintf("ui-spv-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
		CreatorID:  creatorID, // T-0157 C5: 用于消息中心 user_id
	})
	if err != nil {
		return "", fmt.Errorf("queue SPV command: %w", err)
	}

	s.logger.Info("set parameter values command queued",
		zap.String("device_id", deviceID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.Int("param_count", len(params)),
		zap.String("task_id", createdTask.ID),
	)

	return createdTask.ID, nil
}

// ParameterValueItem represents a single parameter to set.
type ParameterValueItem struct {
	Path  string `json:"path"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// GetTaskService returns the task service, or nil if not configured.
func (s *DeviceService) GetTaskService() task.Enqueuer {
	return s.taskSvc
}

// SetMetrics attaches Prometheus metrics to the service.
func (s *DeviceService) SetMetrics(m *DeviceMetrics) {
	s.metrics = m
}

// RegisterFromInform creates a new device from a bootstrap Inform message.
func (s *DeviceService) RegisterFromInform(ctx context.Context, inform *tr069.InformMessage, carrier model.CarrierCode) (*model.Device, error) {
	ctx, span := tracing.StartSpan(ctx, tracing.DeviceTracerName, "Device RegisterFromInform",
		attribute.String("device.serial_number", inform.DeviceId.SerialNumber),
		attribute.String("device.oui", inform.DeviceId.OUI),
		attribute.String("device.carrier", string(carrier)),
	)
	defer span.End()

	s.logger.Info("RegisterFromInform: start",
		zap.String("serial_number", inform.DeviceId.SerialNumber),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.String("carrier", string(carrier)),
		zap.Int("param_count", len(inform.ParameterList)))

	// Check if device already exists
	s.logger.Debug("RegisterFromInform: checking if device exists",
		zap.String("serial_number", inform.DeviceId.SerialNumber))
	existing, err := s.getDeviceBySerialNumber(ctx, inform.DeviceId.SerialNumber)
	if err != nil {
		s.logger.Error("RegisterFromInform: GetBySerialNumber failed",
			zap.Error(err),
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if existing != nil {
		// Device already registered, just update it
		s.logger.Info("RegisterFromInform: device already exists, updating instead",
			zap.String("serial_number", inform.DeviceId.SerialNumber),
			zap.String("existing_device_id", existing.ID.String()))
		return s.UpdateFromInform(ctx, inform)
	}

	// Detect technology from parameters
	tech := detectTechnology(inform.ParameterList)
	s.logger.Debug("RegisterFromInform: technology detected",
		zap.String("technology", string(tech)))

	now := time.Now()
	udpAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")

	device := &model.Device{
		ID:                          uuid.New(),
		SerialNumber:                inform.DeviceId.SerialNumber,
		OUI:                         inform.DeviceId.OUI,
		ProductClass:                inform.DeviceId.ProductClass,
		Manufacturer:                inform.DeviceId.Manufacturer,
		Carrier:                     carrier,
		Technology:                  tech,
		Status:                      model.DeviceActive,
		OpState:                     model.DeriveOpState(model.DeviceActive),
		FirmwareVersion:             findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion"),
		ConnectionRequestURL:        findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL"),
		IPAddress:                   udpAddr,
		NatDetected:                 udpAddr != "",
		UDPConnectionRequestAddress: udpAddr,
		LastInformAt:                &now,
		LastInformEvents:            tr069.EventCodes(inform.Event),
		InformInterval:              300,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	s.logger.Info("RegisterFromInform: creating device in DB",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrier)),
		zap.String("technology", string(tech)),
		zap.String("firmware_version", device.FirmwareVersion))

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		s.logger.Error("RegisterFromInform: deviceRepo.Create failed",
			zap.Error(err),
			zap.String("serial_number", device.SerialNumber))
		return nil, fmt.Errorf("create device: %w", err)
	}

	s.logger.Info("RegisterFromInform: device created in DB successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Write-through: cache the new device
	s.cacheDevice(ctx, device)

	// Create device_info record for extended info
	if err := s.CreateDeviceInfo(ctx, device.ID); err != nil {
		s.logger.Warn("create device_info for new device",
			zap.String("device_id", device.ID.String()),
			zap.Error(err))
	}

	// 记录激活时间：新设备首次 bootstrap inform 即激活，写 first_online_time，
	// 使 op_state（激活状态）反映之。UpdateFromInform 的 shouldActivate 分支已有
	// 同样调用；bootstrap 路径在此补齐——否则全程保持 active 的新设备永不写入。
	if s.infoSyncer != nil {
		if err := s.infoSyncer.RecordOnline(ctx, device.ID); err != nil {
			s.logger.Warn("record online time for new device",
				zap.String("device_id", device.ID.String()),
				zap.Error(err))
		}
	}

	// Check pre-registration: assign to specified group if found.
	if s.regRepo != nil && s.groupAssigner != nil {
		preReg, _ := s.regRepo.GetBySerialNumber(ctx, device.SerialNumber)
		if preReg != nil && preReg.GroupID != nil {
			if _, err := s.groupAssigner.BatchAddDevices(ctx, *preReg.GroupID, []uuid.UUID{device.ID}); err != nil {
				s.logger.Warn("assign device to pre-registered group",
					zap.String("device_id", device.ID.String()),
					zap.String("group_id", preReg.GroupID.String()),
					zap.Error(err))
			} else {
				s.logger.Info("device assigned to pre-registered group",
					zap.String("device_id", device.ID.String()),
					zap.String("group_id", preReg.GroupID.String()))
			}
			// Mark registration as online.
			s.regRepo.UpdateStatus(ctx, preReg.ID, string(global.RegistrationOnline))
		} else if preReg == nil {
			// No pre-registration: assign to default L2 group.
			defaultGroupID, _ := uuid.Parse(global.DefaultLevel2GroupID)
			if _, err := s.groupAssigner.BatchAddDevices(ctx, defaultGroupID, []uuid.UUID{device.ID}); err != nil {
				s.logger.Warn("assign device to default group",
					zap.String("device_id", device.ID.String()),
					zap.Error(err))
			}
		}
	}

	// Sync UDP address to STUN cache
	if udpAddr != "" && s.stunUpdater != nil {
		if err := s.stunUpdater.SetFromInform(ctx, device.SerialNumber, udpAddr); err != nil {
			s.logger.Warn("sync udp address to stun store on register",
				zap.String("serial_number", device.SerialNumber),
				zap.Error(err))
		}
	}

	// Record registration metric
	if s.metrics != nil {
		s.metrics.RegistrationsTotal.WithLabelValues(string(carrier)).Inc()
		s.metrics.DevicesTotal.WithLabelValues(string(device.Status), string(carrier)).Inc()
	}

	// Store parameters
	s.logger.Debug("RegisterFromInform: storing inform parameters",
		zap.Int("param_count", len(inform.ParameterList)))
	s.storeInformParameters(ctx, device.ID, inform.ParameterList)

	// Refresh heartbeat
	if s.heartbeat != nil {
		s.heartbeat.RefreshHeartbeat(ctx, device.SerialNumber, device.InformInterval)
	}

	s.logger.Info("device registered from bootstrap",
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrier)),
		zap.String("oui", device.OUI),
	)

	return device, nil
}

// UpdateFromInform updates an existing device from a periodic Inform message.
func (s *DeviceService) UpdateFromInform(ctx context.Context, inform *tr069.InformMessage) (*model.Device, error) {
	s.logger.Debug("UpdateFromInform: looking up device",
		zap.String("serial_number", inform.DeviceId.SerialNumber))

	device, err := s.getDeviceBySerialNumber(ctx, inform.DeviceId.SerialNumber)
	if err != nil {
		s.logger.Error("UpdateFromInform: device lookup failed",
			zap.Error(err),
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if device == nil {
		s.logger.Warn("UpdateFromInform: device not found in DB, caller should register",
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, nil
	}

	s.logger.Debug("UpdateFromInform: device found, updating",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Update fields from Inform
	now := time.Now()
	// T-0123: 在覆盖字段前捕获旧值，UpdateFromInform 收尾时据此判断 firmware 变化 / offline→active。
	oldStatus := device.Status
	oldVersion := device.FirmwareVersion
	device.OUI = inform.DeviceId.OUI
	device.ProductClass = inform.DeviceId.ProductClass
	device.Manufacturer = inform.DeviceId.Manufacturer
	device.FirmwareVersion = findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
	device.ConnectionRequestURL = findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL")
	device.LastInformAt = &now
	device.LastInformEvents = tr069.EventCodes(inform.Event)
	// T-0162: 收到 Inform 即视为在线。对于已 scan 出 lifecycle_state 的设备，
	// normalizeDeviceForPersist 不会再从老 Status 反推新双字段；这里必须显式置 true，
	// 否则设备一旦被 OfflineDetector 标记成 is_online=false，后续正常 Inform 也无法恢复在线展示。
	device.IsOnline = true

	udpAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")
	if udpAddr != "" {
		device.IPAddress = udpAddr
		device.UDPConnectionRequestAddress = udpAddr
		device.NatDetected = true

		// Sync to STUN address cache for UDP Connection Request
		if s.stunUpdater != nil {
			if err := s.stunUpdater.SetFromInform(ctx, device.SerialNumber, udpAddr); err != nil {
				s.logger.Warn("sync udp address to stun store",
					zap.String("serial_number", device.SerialNumber),
					zap.Error(err))
			}
		}
	}

	// Auto-transition to active when device informs (it's communicating, so it's online)
	// This provides fault tolerance for various non-active states:
	// - discovered: new device sending first heartbeat
	// - registered: device completed registration, now active
	// - provisioning: device was being provisioned, now appears to be active
	// - offline: device was offline, now back online
	// - maintenance: device was in maintenance, now communicating (may indicate recovery)
	shouldActivate := device.Status != model.DeviceActive &&
		device.Status != model.DeviceDecommissioned

	if shouldActivate {
		// Validate that the transition is allowed by state machine
		if err := ValidateTransition(device.Status, model.DeviceActive); err == nil {
			// T-0123: 复用外层 oldStatus（line 552 之前已捕获），不再 shadow 内层声明。
			device.Status = model.DeviceActive
			device.OpState = model.DeriveOpState(model.DeviceActive)

			// Record online time: update last_online_time, and first_online_time if this is the first time
			if s.infoSyncer != nil {
				if err := s.infoSyncer.RecordOnline(ctx, device.ID); err != nil {
					s.logger.Warn("record online time failed",
						zap.String("device_id", device.ID.String()),
						zap.Error(err))
				}
			}

			s.logger.Info("UpdateFromInform: device auto-transitioned to active",
				zap.String("serial_number", device.SerialNumber),
				zap.String("previous_status", string(oldStatus)),
			)
		} else {
			// Log when transition is not allowed (e.g., decommissioned devices)
			s.logger.Debug("UpdateFromInform: state transition to active not allowed",
				zap.String("serial_number", device.SerialNumber),
				zap.String("current_status", string(device.Status)),
				zap.Error(err),
			)
		}
	}

	s.logger.Debug("UpdateFromInform: calling deviceRepo.Update",
		zap.String("device_id", device.ID.String()))

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		// stale-cache recovery: cache returned a device whose row no longer
		// exists in PG (manual delete / migration race). Invalidate the cache
		// and let the caller retry as a registration. Without this fall-back
		// the device would be permanently stuck — UpdateFromInform keeps
		// silently writing 0 rows while the cache writes itself back.
		if errors.Is(err, commonerrors.ErrNotFound) {
			s.logger.Warn("UpdateFromInform: stale cache detected (device gone from DB), clearing and signalling re-register",
				zap.String("device_id", device.ID.String()),
				zap.String("serial_number", device.SerialNumber))
			if s.cache != nil {
				s.cache.Delete(ctx, device.SerialNumber)
			}
			return nil, nil
		}
		s.logger.Error("UpdateFromInform: deviceRepo.Update failed",
			zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return nil, fmt.Errorf("update device: %w", err)
	}

	s.logger.Info("UpdateFromInform: device updated in DB successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Write-through: refresh cache with updated device
	s.cacheDevice(ctx, device)

	// Store parameters
	s.storeInformParameters(ctx, device.ID, inform.ParameterList)

	// Sync key parameters to device_info for fast query access
	if s.infoSyncer != nil {
		if err := s.infoSyncer.SyncFromParameters(ctx, device.ID, device.Carrier, device.Technology); err != nil {
			s.logger.Warn("sync device info from parameters",
				zap.String("device_id", device.ID.String()),
				zap.Error(err))
		}
	}

	// Refresh heartbeat
	if s.heartbeat != nil {
		s.heartbeat.RefreshHeartbeat(ctx, device.SerialNumber, device.InformInterval)
	}

	// T-0123/T-0125: 检测 firmware 变化与 offline→active 二选一发布事件。
	// 同一 Inform 满足两者时优先发 firmware.changed（不发 device.online），
	// 由 provision.HandleFirmwareChanged 触发的重新交集 + Path B 同步覆盖 online 的能力，
	// 避免两路 Path B 重复同步。
	newVersion := device.FirmwareVersion
	firmwareChanged := oldVersion != "" && newVersion != "" && oldVersion != newVersion
	becameOnline := oldStatus == model.DeviceOffline && device.Status == model.DeviceActive
	if firmwareChanged {
		s.PublishDeviceFirmwareChangedEvent(ctx, device, oldVersion, newVersion, becameOnline)
	} else if becameOnline {
		s.PublishDeviceOnlineEvent(ctx, device)
	}

	return device, nil
}

// TransitionStatus validates and executes a device state transition.
func (s *DeviceService) TransitionStatus(ctx context.Context, deviceID uuid.UUID, newStatus model.DeviceStatus) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	if err := ValidateTransition(device.Status, newStatus); err != nil {
		return err
	}

	if err := s.deviceRepo.UpdateStatus(ctx, deviceID, newStatus); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// C2 修复：状态变更后清 cache，避免前端读到 stale status 字段（最长一个 inform 周期）
	if s.cache != nil {
		s.cache.Delete(ctx, device.SerialNumber)
	}

	// Update gauge: decrement old status, increment new status
	if s.metrics != nil {
		s.metrics.DevicesTotal.WithLabelValues(string(device.Status), string(device.Carrier)).Dec()
		s.metrics.DevicesTotal.WithLabelValues(string(newStatus), string(device.Carrier)).Inc()
	}

	s.logger.Info("device status transitioned",
		zap.String("device_id", deviceID.String()),
		zap.String("from", string(device.Status)),
		zap.String("to", string(newStatus)),
	)

	return nil
}

// GetDevice retrieves a device by ID.
func (s *DeviceService) GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	return s.deviceRepo.GetByID(ctx, id)
}

// GetBySerialNumber retrieves a device by its serial number (Redis cache → PostgreSQL).
func (s *DeviceService) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	return s.getDeviceBySerialNumber(ctx, sn)
}

// ListDevices returns a paginated list of devices.
func (s *DeviceService) ListDevices(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	return s.deviceRepo.List(ctx, filter)
}

// GetDeviceParameters retrieves all parameters for a device.
func (s *DeviceService) GetDeviceParameters(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return s.paramRepo.GetByDevice(ctx, deviceID)
}

// CountByStatus returns device counts grouped by status.
func (s *DeviceService) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	counts, err := s.deviceRepo.CountByStatus(ctx, carrier)
	if err != nil {
		return nil, err
	}

	// Sync gauge metrics with actual DB counts
	if s.metrics != nil && carrier != nil {
		for status, count := range counts {
			s.metrics.DevicesTotal.WithLabelValues(string(status), string(*carrier)).Set(float64(count))
		}
	}

	return counts, nil
}

// ListDevicesWithInfo returns a paginated list of devices joined with extended info.
//
// T-0162 D5: 返回时附带 Stats 字段（筛选条件下的全量统计），前端直读 stats
// 而非用当前页 items 自行 filter() 估算（修复 Q2 分析里的偏差）。
func (s *DeviceService) ListDevicesWithInfo(ctx context.Context, filter DeviceFilter) (*model.ListResponse[DeviceWithInfo], error) {
	if s.deviceInfoRepo == nil {
		// Fallback: if device_info repo not configured, use standard list
		result, err := s.deviceRepo.List(ctx, filter)
		if err != nil {
			return nil, err
		}
		items := make([]DeviceWithInfo, len(result.Items))
		for i, d := range result.Items {
			items[i] = DeviceWithInfo{Device: d}
		}
		return model.NewListResponse(items, result.Total, result.Page, result.PageSize), nil
	}
	result, err := s.deviceInfoRepo.ListDevicesWithInfo(ctx, filter)
	if err != nil {
		return nil, err
	}
	// T-0162: 同筛选条件下跑全量统计；stats 失败不阻断主 list 返回（降级返回
	// items + Stats=nil，前端会回退到老 fallback 行为，与现状等价）。
	if stats, statsErr := s.deviceInfoRepo.ComputeListStats(ctx, filter); statsErr == nil {
		result.Stats = stats
	}
	return result, nil
}

// GetDeviceWithInfo retrieves a single device joined with extended info, for the
// device detail page. 使详情与列表的 op_state（激活状态）及 device_info 扩展
// 字段口径一致。device_info repo 未配置时回退为裸 Device 包装（扩展字段为空）。
func (s *DeviceService) GetDeviceWithInfo(ctx context.Context, id uuid.UUID) (*DeviceWithInfo, error) {
	if s.deviceInfoRepo == nil {
		d, err := s.deviceRepo.GetByID(ctx, id)
		if err != nil || d == nil {
			return nil, err
		}
		return &DeviceWithInfo{Device: *d}, nil
	}
	return s.deviceInfoRepo.GetByIDWithInfo(ctx, id)
}

// GetDeviceInfo retrieves extended info for a device.
func (s *DeviceService) GetDeviceInfo(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error) {
	if s.deviceInfoRepo == nil {
		return nil, nil
	}
	return s.deviceInfoRepo.GetByDeviceID(ctx, deviceID)
}

// UpdateDeviceInfo updates the manually editable device info fields.
func (s *DeviceService) UpdateDeviceInfo(ctx context.Context, deviceID uuid.UUID, req UpdateDeviceInfoRequest, updater string) error {
	if s.deviceInfoRepo == nil {
		return fmt.Errorf("device info repository not configured")
	}
	return s.deviceInfoRepo.UpdateManualFields(ctx, deviceID, req, updater)
}

// CreateDeviceInfo creates the initial device_info record for a newly registered device.
// Note: first_online_time is NOT set here. It will be set when the device actually
// transitions to active status (sends first heartbeat and becomes online).
func (s *DeviceService) CreateDeviceInfo(ctx context.Context, deviceID uuid.UUID) error {
	if s.deviceInfoRepo == nil {
		return nil
	}
	info := &DeviceInfo{
		DeviceID: deviceID,
		// FirstOnlineTime: nil - will be set when device actually goes online
	}
	return s.deviceInfoRepo.Create(ctx, info)
}

func (s *DeviceService) storeInformParameters(ctx context.Context, deviceID uuid.UUID, params []tr069.ParameterValueStruct) {
	if len(params) == 0 {
		return
	}

	modelParams := make([]model.DeviceParameter, 0, len(params))
	for _, p := range params {
		modelParams = append(modelParams, model.DeviceParameter{
			DeviceID:       deviceID,
			ParameterPath:  p.Name,
			ParameterValue: p.Value,
			ParameterType:  model.ParamString,
			Writable:       false,
			LastUpdatedAt:  time.Now(),
		})
	}

	if err := s.paramRepo.BatchUpsert(ctx, deviceID, modelParams); err != nil {
		s.logger.Error("store inform parameters",
			zap.Error(err),
			zap.String("device_id", deviceID.String()),
		)
	}
}

// DeviceFirmwareChangedEvent 设备 swVersion 变化时发布（T-0125）。
//
// 订阅者：provision.Engine.HandleFirmwareChanged — Redis 串行锁 + RequestModelUpload
// 重新交集 + Path B 同步。BecameOnline 字段记录该次 Inform 是否同时从 offline 恢复 active
// （二选一逻辑下不发 device.online，但下游可参考此字段做指标/审计区分）。
type DeviceFirmwareChangedEvent struct {
	DeviceID     uuid.UUID `json:"device_id"`
	SerialNumber string    `json:"serial_number"`
	ProductClass string    `json:"product_class"`
	OldVersion   string    `json:"old_version"`
	NewVersion   string    `json:"new_version"`
	BecameOnline bool      `json:"became_online"` // 二选一抑制的 online 事件
}

// PublishDeviceFirmwareChangedEvent 发布 device.firmware.changed 事件（T-0125）。
//
// 不阻塞主流程：EventBus nil / 序列化失败 / Publish 失败均仅 log Warn。
//
// 导出供 BatchInformProcessor.doFlush 在 PG 写入成功后调用（T-0125 batch path 补完）。
func (s *DeviceService) PublishDeviceFirmwareChangedEvent(ctx context.Context, device *model.Device,
	oldVersion, newVersion string, becameOnline bool) {
	if s.eventBus == nil {
		return
	}
	payload := DeviceFirmwareChangedEvent{
		DeviceID:     device.ID,
		SerialNumber: device.SerialNumber,
		ProductClass: device.ProductClass,
		OldVersion:   oldVersion,
		NewVersion:   newVersion,
		BecameOnline: becameOnline,
	}
	evt, err := event.NewEvent(event.SubjectDeviceFirmwareChanged, payload)
	if err != nil {
		s.logger.Error("create device.firmware.changed event", zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectDeviceFirmwareChanged, evt); err != nil {
		s.logger.Warn("publish device.firmware.changed event", zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return
	}
	s.logger.Info("firmware.changed event published",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("old_version", oldVersion),
		zap.String("new_version", newVersion),
		zap.Bool("became_online_suppressed", becameOnline),
	)
}

// DeviceOnlineEvent 已存在设备从 offline 恢复 active 时发布（T-0123）。
//
// 与 DeviceOfflineEvent 对称；订阅者：provision.Engine.HandleDeviceOnline。
// 载荷不含 lastOfflineDuration（devices 表无 last_offline_at 字段，超本任务范围）。
type DeviceOnlineEvent struct {
	DeviceID     uuid.UUID `json:"device_id"`
	SerialNumber string    `json:"serial_number"`
	ProductClass string    `json:"product_class"`
	SwVersion    string    `json:"sw_version"`
}

// PublishDeviceOnlineEvent 发布 device.online 事件（T-0123）。
//
// 不阻塞主流程：EventBus nil / 序列化失败 / Publish 失败均仅 log Warn，
// 不影响 UpdateFromInform 后续步骤。
//
// 导出供 BatchInformProcessor.doFlush 在 PG 写入成功后调用（T-0123 batch path 补完）。
func (s *DeviceService) PublishDeviceOnlineEvent(ctx context.Context, device *model.Device) {
	if s.eventBus == nil {
		return
	}
	payload := DeviceOnlineEvent{
		DeviceID:     device.ID,
		SerialNumber: device.SerialNumber,
		ProductClass: device.ProductClass,
		SwVersion:    device.FirmwareVersion,
	}
	evt, err := event.NewEvent(event.SubjectDeviceOnline, payload)
	if err != nil {
		s.logger.Error("create device.online event", zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectDeviceOnline, evt); err != nil {
		s.logger.Warn("publish device.online event", zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return
	}
	s.logger.Info("device.online published",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
	)
}

// PublishDeviceRegistered publishes a device.registered event for the given device.
// Called by InformHandler on BOOTSTRAP/BOOT events to trigger provisioning engine.
func (s *DeviceService) PublishDeviceRegistered(ctx context.Context, device *model.Device) {
	if s.eventBus == nil {
		return
	}
	payload := map[string]interface{}{
		"device_id":     device.ID,
		"serial_number": device.SerialNumber,
		"oui":           device.OUI,
		"product_class": device.ProductClass,
		"carrier":       string(device.Carrier),
		"technology":    string(device.Technology),
	}
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, payload)
	if err != nil {
		s.logger.Error("create device.registered event", zap.Error(err))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectDeviceRegistered, evt); err != nil {
		s.logger.Warn("publish device.registered event", zap.Error(err))
	}
}

// HaltReasonMainPath / HaltReasonDetailPath are the standard TR-181 paths
// carrying the abnormal-reboot signal. Vendor-specific aliases (X_BAICELLS_*,
// X_COM_* etc.) are normalized to these by the param-model Translator before
// reaching this layer; private-path resolution itself is therefore left to
// upstream code and not duplicated here.
const (
	HaltReasonMainPath   = "Device.HaltReason.MainReason"
	HaltReasonDetailPath = "Device.HaltReason.DetailReason"
	// DeviceUpTimeStandardPath is the canonical TR-181 uptime path used as a
	// best-effort source for runtime_before_reboot. Many CPEs report it under
	// a private alias; absence is acceptable (we store 0).
	DeviceUpTimeStandardPath = "Device.DeviceInfo.UpTime"
)

// extractParam looks up a parameter value by exact path. Returns "" when absent.
func extractParam(params []tr069.ParameterValueStruct, path string) string {
	for _, p := range params {
		if p.Name == path {
			return p.Value
		}
	}
	return ""
}

// RecordBootFromInform handles the data updates triggered by a reboot-complete
// Inform (event codes "1 BOOT" or "M Reboot"). It:
//   - atomically increments devices.boot_count and stamps last_boot_at;
//   - refreshes the Redis device cache so the incremented counter is visible;
//   - publishes SubjectDeviceRebootAbnormal when the reboot was not initiated by
//     the ACS (i.e. "1 BOOT" without "M Reboot"), so downstream listeners
//     (alarm engine, audit log) can react.
//
// The caller is expected to have already ensured the device row exists (via
// UpdateFromInform or RegisterFromInform). Returns the updated boot_count; 0
// with no error means the device could not be found and the boot was ignored.
func (s *DeviceService) RecordBootFromInform(ctx context.Context, device *model.Device, events []string, params []tr069.ParameterValueStruct) (int, error) {
	if device == nil {
		return 0, nil
	}
	now := time.Now()
	bootCount, err := s.deviceRepo.RecordBoot(ctx, device.SerialNumber, now)
	if err != nil {
		return 0, fmt.Errorf("record boot: %w", err)
	}
	if bootCount == 0 {
		// Device row vanished between lookup and update — nothing to do.
		return 0, nil
	}

	// Refresh the cached device so subsequent reads see the new counter.
	device.LastBootAt = &now
	device.BootCount = bootCount
	s.cacheDevice(ctx, device)

	// Abnormal reboot 判断（T-0158）：
	//   1 BOOT 事件 + Device.HaltReason.MainReason 非空 → 异常重启
	//
	// 比"1 BOOT && !M Reboot"更准确：受控的看门狗 / 软重启 CPE 不会带 M Reboot
	// 但也不算异常；只有 CPE 明确上报故障主原因（崩溃/死机/异常断电）才算。
	// 当 ParameterList 为空（极少数兼容场景）退回旧的事件码组合规则，保留漏判
	// 优先于误判。
	haltMainReason := extractParam(params, HaltReasonMainPath)
	haltDetailReason := extractParam(params, HaltReasonDetailPath)
	uptimeStr := extractParam(params, DeviceUpTimeStandardPath)
	runtimeBeforeReboot := parseUptimeSeconds(uptimeStr)

	hasBoot := hasEventCode(events, tr069.EventBoot)
	var abnormal bool
	switch {
	case len(params) == 0:
		// Fallback: 没有参数列表时退回旧规则（保 backward-compatible）
		abnormal = hasBoot && !hasEventCode(events, tr069.EventMReboot)
	default:
		abnormal = hasBoot && haltMainReason != ""
	}

	s.logger.Info("device boot recorded",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.Int("boot_count", bootCount),
		zap.Bool("abnormal", abnormal),
		zap.String("halt_main_reason", haltMainReason),
		zap.String("halt_detail_reason", haltDetailReason),
		zap.Int64("runtime_before_reboot", runtimeBeforeReboot),
		zap.Strings("events", events),
	)

	if abnormal {
		// 识别即落库：在事件发布前完成 detected 占位记录写入，
		// 让"设备一上线立即可见"，且不依赖订阅者完成时机。
		if s.abnormalRecorder != nil {
			// snapshot 直接 freeze devices 表当时的字段值，空就是空（人工命名 /
			// IP 长期没回填等都是上游业务流程的事，不在异常重启识别这一步做兜底）。
			snap := AbnormalRebootSnapshot{
				DeviceID:            device.ID,
				DeviceSN:            device.SerialNumber,
				DeviceName:          device.DeviceName,
				OperateIP:           device.IPAddress,
				SoftwareVersion:     device.FirmwareVersion,
				HaltMainReason:      haltMainReason,
				HaltDetailReason:    haltDetailReason,
				RuntimeBeforeReboot: runtimeBeforeReboot,
				IsGNB:               device.Technology == model.TechNR,
				DetectedAt:          now,
			}
			if device.Technology == model.TechNR {
				snap.DeviceType = "gNB"
			} else {
				snap.DeviceType = "eNB"
			}
			if recErr := s.abnormalRecorder.RecordAbnormalReboot(ctx, snap); recErr != nil {
				// 落库失败不阻塞事件发布；告警链路依然能基于事件累计。
				s.logger.Warn("record abnormal reboot",
					zap.String("serial_number", device.SerialNumber),
					zap.Error(recErr))
			}
		}

		if s.eventBus != nil {
			payload := map[string]interface{}{
				"device_id":             device.ID.String(),
				"serial_number":         device.SerialNumber,
				"carrier":               string(device.Carrier),
				"boot_count":            bootCount,
				"last_boot_at":          now,
				"events":                events,
				"halt_main_reason":      haltMainReason,
				"halt_detail_reason":    haltDetailReason,
				"runtime_before_reboot": runtimeBeforeReboot,
			}
			if evt, evtErr := event.NewEvent(event.SubjectDeviceRebootAbnormal, payload); evtErr == nil {
				if pubErr := s.eventBus.Publish(ctx, event.SubjectDeviceRebootAbnormal, evt); pubErr != nil {
					s.logger.Warn("publish device.reboot.abnormal event", zap.Error(pubErr))
				}
			}
		}
	} else if hasBoot && s.bootEventRecorder != nil {
		// 单纯 1 BOOT（无 HaltReason）→ event_logs 普通事件日志。
		// 与异常重启分两个表：station_fault_logs 是异常+文件管理，event_logs 是审计流水。
		bootSnap := BootEventSnapshot{
			DeviceID:        device.ID,
			DeviceSN:        device.SerialNumber,
			DeviceName:      device.DeviceName,
			OperateIP:       device.IPAddress,
			SoftwareVersion: device.FirmwareVersion,
			IsGNB:           device.Technology == model.TechNR,
			BootCount:       bootCount,
			Events:          events,
			OccurredAt:      now,
		}
		if device.Technology == model.TechNR {
			bootSnap.DeviceType = "gNB"
		} else {
			bootSnap.DeviceType = "eNB"
		}
		if recErr := s.bootEventRecorder.RecordBootEvent(ctx, bootSnap); recErr != nil {
			// 失败不阻塞主流程
			s.logger.Warn("record boot event log",
				zap.String("serial_number", device.SerialNumber),
				zap.Error(recErr))
		}
	}

	return bootCount, nil
}

// parseUptimeSeconds turns the TR-181 UpTime string ("seconds since boot")
// into int64 seconds. Empty / unparseable values return 0 (= unknown).
func parseUptimeSeconds(v string) int64 {
	if v == "" {
		return 0
	}
	var n int64
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 {
		return 0
	}
	return n
}

func hasEventCode(events []string, target string) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

// detectTechnology tries to determine the radio technology from Inform parameters.
//
// 设备从来不直接送 Technology = "LTE"/"NR" 这种参数值（TR-069 协议层面没
// 这种参数）；只送 Device.DeviceInfo.ModelName / Description 字符串。这里
// 用 substring 启发式从 model 名字推断 5G/NR vs 4G/LTE。
//
// 大小写不敏感：Ericsson AIR6488 写 "AIR6488 5G NR"，华为 AAU5613 可能写
// "5g nr" 小写 — 全部转 lower 后再匹配。返回的 model.Technology 是内部
// canonical 小写常量。
func detectTechnology(params []tr069.ParameterValueStruct) model.Technology {
	for _, p := range params {
		if p.Name == "Device.DeviceInfo.ModelName" || p.Name == "Device.DeviceInfo.Description" {
			v := strings.ToLower(p.Value)
			if strings.Contains(v, "nr") || strings.Contains(v, "5g") || strings.Contains(v, "gnb") {
				return model.TechNR
			}
		}
	}
	return model.TechLTE
}

func findParamValue(params []tr069.ParameterValueStruct, name string) string {
	for _, p := range params {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

// getDeviceBySerialNumber looks up a device with Redis cache → PostgreSQL fallback.
func (s *DeviceService) getDeviceBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if s.cache != nil {
		return s.cache.GetOrLoad(ctx, sn, func(ctx context.Context, sn string) (*model.Device, error) {
			return s.deviceRepo.GetBySerialNumber(ctx, sn)
		})
	}
	return s.deviceRepo.GetBySerialNumber(ctx, sn)
}

// cacheDevice writes the device to Redis cache after DB mutations.
func (s *DeviceService) cacheDevice(ctx context.Context, device *model.Device) {
	if s.cache != nil {
		s.cache.Set(ctx, device)
	}
}

// GetDeviceDetailComposite assembles a comprehensive device detail view by querying
// device, device_info, and device_parameters (via prefix queries for MME/License/Antenna/Cells).
func (s *DeviceService) GetDeviceDetailComposite(ctx context.Context, deviceID uuid.UUID) (*DeviceDetailComposite, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return nil, nil
	}

	result := &DeviceDetailComposite{
		Device: device,
	}

	// Get device_info
	if s.deviceInfoRepo != nil {
		info, err := s.deviceInfoRepo.GetByDeviceID(ctx, deviceID)
		if err != nil {
			s.logger.Warn("get device info for detail composite",
				zap.String("device_id", deviceID.String()),
				zap.Error(err))
		}
		result.Info = info
	}

	// MME pool — 按分组查询取代 LIKE
	mmeParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "mme_pool")
	if err != nil {
		s.logger.Warn("get mme_pool params",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	result.MMEPool = AssembleMMEPool(mmeParams)

	// License
	licenseParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "license")
	if err != nil {
		s.logger.Warn("get license params",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	result.License = AssembleLicenseDetail(licenseParams)

	// Antenna
	antennaParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "antenna")
	if err != nil {
		s.logger.Warn("get antenna params",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	result.Antenna = AssembleAntennaInfo(antennaParams)

	// Cells — 需要跨多个 group，仍用全量查询
	numOfCells := 1
	if result.Info != nil && result.Info.NumOfCells > 0 {
		numOfCells = result.Info.NumOfCells
	}
	allParams, err := s.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		s.logger.Warn("get all params for cells",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
		return result, nil
	}
	result.Cells = AssembleCells(allParams, numOfCells)

	return result, nil
}

// CreateDevice creates a new device from an API request.
func (s *DeviceService) CreateDevice(ctx context.Context, req CreateDeviceRequest) (*model.Device, error) {
	existing, err := s.deviceRepo.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("check existing device: %w", err)
	}
	if existing != nil {
		return nil, commonerrors.ErrAlreadyExists
	}

	// License enforcement (T-0015 / R-103). nil enforcer = enforcement disabled.
	if s.licenseEnforcer != nil {
		if err := s.licenseEnforcer.EnforceExpiry(ctx, "device.create"); err != nil {
			return nil, err
		}
		if err := s.licenseEnforcer.EnforceCapacity(ctx, 1); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	device := &model.Device{
		ID:           uuid.New(),
		SerialNumber: req.SerialNumber,
		OUI:          req.OUI,
		ProductClass: req.ProductClass,
		Manufacturer: req.Manufacturer,
		ModelName:    req.ModelName,
		Carrier:      req.Carrier,
		Technology:   req.Technology,
		Status:       model.DeviceRegistered,
		IPAddress:    req.IPAddress,
		DeviceName:   req.DeviceName,
		SiteID:       req.SiteID,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	s.logger.Info("device created via API",
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(device.Carrier)),
	)
	return device, nil
}

// UpdateDevice updates an existing device from an API request.
func (s *DeviceService) UpdateDevice(ctx context.Context, id uuid.UUID, req UpdateDeviceRequest) (*model.Device, error) {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return nil, nil
	}

	if req.DeviceName != nil {
		device.DeviceName = *req.DeviceName
	}
	if req.SiteID != nil {
		device.SiteID = *req.SiteID
	}
	if req.ModelName != nil {
		device.ModelName = *req.ModelName
	}
	if req.Latitude != nil {
		device.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		device.Longitude = *req.Longitude
	}
	if req.Status != nil {
		device.Status = *req.Status
	}

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}
	return device, nil
}

// DeleteDevice deletes a device by ID.
func (s *DeviceService) DeleteDevice(ctx context.Context, id uuid.UUID) error {
	// C2 修复：先查 SN 用于删除后清 cache（cache key 是 SN 不是 ID）
	device, _ := s.deviceRepo.GetByID(ctx, id)
	if err := s.deviceRepo.Delete(ctx, id); err != nil {
		return err
	}
	if s.cache != nil && device != nil {
		s.cache.Delete(ctx, device.SerialNumber)
	}
	return nil
}

// BatchDeleteDevices deletes multiple devices by their IDs.
// Returns a BatchOperationResult summarising successes and failures.
func (s *DeviceService) BatchDeleteDevices(ctx context.Context, ids []uuid.UUID, deletedBy string) BatchOperationResult {
	result := BatchOperationResult{Total: len(ids)}

	// C2 修复：先查 SN 列表用于删除后清 cache
	idToSN, _ := s.deviceRepo.ListSerialsByIDs(ctx, ids)

	deleted, err := s.deviceRepo.BatchDelete(ctx, ids, deletedBy)
	if err != nil {
		s.logger.Error("batch delete devices failed",
			zap.Int("total", len(ids)),
			zap.Error(err),
		)
		// All items failed
		result.Failed = len(ids)
		for _, id := range ids {
			result.Errors = append(result.Errors, BatchItemError{
				ID:      id.String(),
				Message: err.Error(),
			})
		}
		return result
	}

	result.Succeeded = int(deleted)
	result.Failed = len(ids) - int(deleted)

	// C2 修复：删除成功后清 cache（cache key 用 SN，cache 不感知 ID）
	if s.cache != nil {
		for _, sn := range idToSN {
			s.cache.Delete(ctx, sn)
		}
	}

	s.logger.Info("batch delete devices",
		zap.Int("total", len(ids)),
		zap.Int64("deleted", deleted),
	)
	return result
}

// BatchRebootDevices queues a Reboot command for each device in the list.
// Returns a BatchOperationResult summarising successes and failures.
func (s *DeviceService) BatchRebootDevices(ctx context.Context, ids []uuid.UUID) BatchOperationResult {
	result := BatchOperationResult{Total: len(ids)}

	for _, id := range ids {
		if err := s.RebootDevice(ctx, id); err != nil {
			s.logger.Warn("batch reboot: reboot device failed",
				zap.String("device_id", id.String()),
				zap.Error(err),
			)
			result.Failed++
			result.Errors = append(result.Errors, BatchItemError{
				ID:      id.String(),
				Message: err.Error(),
			})
		} else {
			result.Succeeded++
		}
	}

	s.logger.Info("batch reboot devices",
		zap.Int("total", len(ids)),
		zap.Int("succeeded", result.Succeeded),
		zap.Int("failed", result.Failed),
	)
	return result
}

// ListGeo returns devices with geographic coordinates for map display.
func (s *DeviceService) ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error) {
	return s.deviceRepo.ListGeo(ctx, filter)
}

// GetGeoStats returns device statistics for map display.
func (s *DeviceService) GetGeoStats(ctx context.Context, groupIDs []string) (*GeoStats, error) {
	return s.deviceRepo.GetGeoStats(ctx, groupIDs)
}

// SearchDevices searches devices by keyword for map display.
func (s *DeviceService) SearchDevices(ctx context.Context, keyword string, limit int) ([]GeoDevice, error) {
	return s.deviceRepo.SearchDevices(ctx, keyword, limit)
}

// ===== Recycle Bin Operations =====

// ListRecycleBin returns soft-deleted devices with filtering.
func (s *DeviceService) ListRecycleBin(ctx context.Context, filter RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return s.deviceRepo.ListRecycleBin(ctx, filter)
}

// RestoreDevices restores soft-deleted devices.
func (s *DeviceService) RestoreDevices(ctx context.Context, ids []uuid.UUID) (int64, error) {
	// C2 修复：恢复后清 cache，让下次 inform 重新走 GetOrLoad 加载干净的 device 行
	idToSN, _ := s.deviceRepo.ListSerialsByIDs(ctx, ids)
	n, err := s.deviceRepo.RestoreDevices(ctx, ids)
	if err != nil {
		return n, err
	}
	if s.cache != nil {
		for _, sn := range idToSN {
			s.cache.Delete(ctx, sn)
		}
	}
	return n, nil
}

// PermanentDeleteDevices permanently removes devices from the database.
func (s *DeviceService) PermanentDeleteDevices(ctx context.Context, ids []uuid.UUID) (int64, error) {
	// C2 修复：先查 SN 用于删除后清 cache
	idToSN, _ := s.deviceRepo.ListSerialsByIDs(ctx, ids)
	n, err := s.deviceRepo.PermanentDelete(ctx, ids)
	if err != nil {
		return n, err
	}
	if s.cache != nil {
		for _, sn := range idToSN {
			s.cache.Delete(ctx, sn)
		}
	}
	return n, nil
}

// GetProductClasses returns distinct product types from the device table,
// merged with mandatory types.
func (s *DeviceService) GetProductClasses(ctx context.Context) ([]string, error) {
	return s.deviceRepo.ListProductClasses(ctx)
}
