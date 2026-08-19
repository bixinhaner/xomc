package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/netutil"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// SysConfigLookup 读取系统配置的函数类型（(category, key) → value）。
// 与 provision.NameSyncConfigLookup 签名相同，在 device 包独立定义避免 import cycle
// （provision 依赖 device，device 不能反过来依赖 provision）。
type SysConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

type syncGPVOpenGuard interface {
	HasOpenSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error)
}

type syncGPVDeviceLocker interface {
	AcquireSyncGPVDeviceLock(ctx context.Context, deviceSN string) (release func(), err error)
}

// RenameDeviceResult describes side effects produced by RenameDevice.
type RenameDeviceResult struct {
	TaskID string
}

// GroupAssigner assigns a device to a group.
type GroupAssigner interface {
	BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
}

// DeviceGroupCountsInvalidator clears cached device-group counts after a
// device lifecycle write changes which active devices must be counted.
type DeviceGroupCountsInvalidator interface {
	InvalidateDeviceGroupCounts()
}

// DeviceService provides business logic for device management.
type DeviceService struct {
	redisClient            redis.UniversalClient
	deviceRepo             DeviceRepository
	paramRepo              DeviceParameterRepository
	antennaPlanRepo        AntennaSectorPlanRepository
	deviceInfoRepo         DeviceInfoRepository
	controlSummaryReader   DeviceControlSummaryReader
	disconnectAlarms       disconnectedAlarmStore
	disconnectClearer      disconnectedAlarmClearer
	regRepo                RegistrationRepository
	groupAssigner          GroupAssigner
	groupCountsInvalidator DeviceGroupCountsInvalidator
	infoSyncer             *InfoSyncer
	reconciler             *DeviceStatusReconciler
	eventBus               event.EventBus
	taskSvc                task.Enqueuer
	connReq                ConnectionRequester
	stunUpdater            StunAddressUpdater
	cache                  *DeviceCache
	metrics                *DeviceMetrics
	licenseEnforcer        LicenseEnforcer
	excessOffliner         ExcessOffliner           // license 降容清理：批量置离线超容设备（nil = 禁用）
	carrierRegistry        *carrier.CarrierRegistry // T-0029: RF control path lookup by carrier+tech
	paramSyncStarter       ParamSyncStarter         // T-0126: 注入 *provision.SyncService 触发 Path B 手动同步
	manualOfflineMode      string
	abnormalRecorder       AbnormalRebootRecorder // T-0158: 异常重启识别即落库（nil = 禁用）
	bootEventRecorder      BootEventRecorder      // 普通 1 BOOT 事件日志写入（nil = 禁用）
	productMatcher         ProductClassMatcher    // Phase 6 ModelName 回填（nil = 禁用）
	productBinder          ProductBinder          // T-0176-PR-D：CreateDevice inline match 后写回 product_id（nil = 禁用）
	groupReader            DeviceGroupReader      // 越权校验：读设备组归属（nil = 退化为不校验，见 AuthorizeDeviceGroupAccess）
	sysConfigLookup        SysConfigLookup        // 读系统配置（nameSyncMode 等）
	logger                 *zap.Logger
}

func (s *DeviceService) SetAntennaSectorPlanRepository(repo AntennaSectorPlanRepository) {
	s.antennaPlanRepo = repo
}

type disconnectedAlarmStore interface {
	GetActiveByDeviceAndIdentifier(ctx context.Context, deviceSN string, alarmIdentifier string) (*model.Alarm, error)
}

type disconnectedAlarmClearer interface {
	ClearBySync(ctx context.Context, alarm *model.Alarm) error
}

// LicenseEnforcer is the narrow interface DeviceService consumes from the
// license package. Defined here on the consumer side so DeviceService stays
// independent of the full license model. Wired via SetLicenseEnforcer.
//
// EnforceCapacity 走 per-type gating（issue #316）：deviceType 为网元类型
// (product.alarm_ne_type)，license 按 DevicesSupport[deviceType] 独立限额。
// Both methods may be called as nil-safe gates: SetLicenseEnforcer with a
// nil value is fine and disables enforcement (used in dev/test).
type LicenseEnforcer interface {
	EnforceCapacity(ctx context.Context, deviceType string, additional int) error
	EnforceExpiry(ctx context.Context, operation string) error
}

// ExcessOffliner 按各网元类型容量上限批量置离线超容在线设备，供 license 降容
// 清理使用（issue #316，在线口径：离线不占容量）。PgDeviceRepository 实现。
// 返回被置离线设备的 serial_number 列表（用于清 Redis 缓存）。
// 单独窄接口，避免扩大 DeviceRepository（mockgen 生成）。
type ExcessOffliner interface {
	OfflineExcessByTypeCapacity(ctx context.Context, typeCapacity map[string]int) ([]string, error)
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
//
// reconciler 可为 nil（dev/test 模式下不启用心跳刷新),此时所有 RefreshHeartbeat
// 调用静默跳过,不影响 Inform 接收与 PG 写入。

// SetRedis assigns the redis client to DeviceService for caching lookups.
func (s *DeviceService) SetRedis(r redis.UniversalClient) {
	s.redisClient = r
}

// SetDeviceGroupCountsInvalidator wires the topology count-cache invalidator.
func (s *DeviceService) SetDeviceGroupCountsInvalidator(invalidator DeviceGroupCountsInvalidator) {
	s.groupCountsInvalidator = invalidator
}

func (s *DeviceService) SetControlSummaryReader(reader DeviceControlSummaryReader) {
	s.controlSummaryReader = reader
}

func NewDeviceService(
	deviceRepo DeviceRepository,
	paramRepo DeviceParameterRepository,
	reconciler *DeviceStatusReconciler,
	eventBus event.EventBus,
	logger *zap.Logger,
) *DeviceService {
	return &DeviceService{
		deviceRepo: deviceRepo,
		paramRepo:  paramRepo,
		reconciler: reconciler,
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

// ResolveCarrierByOUI 通过 OUI 查询运营商编码（供批量预登记路径使用）。
// carrierRegistry 未注入时返回空串（调用方需额外处理）。
func (s *DeviceService) ResolveCarrierByOUI(oui string) model.CarrierCode {
	if s.carrierRegistry == nil {
		return ""
	}
	return s.carrierRegistry.ResolveByOUI(oui)
}

// SetDisconnectedAlarmCleaner wires OMC disconnected-alarm cleanup for offline→online recovery.
func (s *DeviceService) SetDisconnectedAlarmCleaner(store disconnectedAlarmStore, clearer disconnectedAlarmClearer) {
	s.disconnectAlarms = store
	s.disconnectClearer = clearer
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

// SetDeviceGroupReader wires the per-device group membership reader used by
// AuthorizeDeviceGroupAccess for IDOR protection on by-ID read endpoints.
// nil-safe: when unset, authorization checks degrade to allow (dev/test).
func (s *DeviceService) SetDeviceGroupReader(r DeviceGroupReader) {
	s.groupReader = r
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

// SetExcessOffliner 注入批量置离线能力（PgDeviceRepository），启用 license 降容
// 清理（OfflineExcessDevices）。nil = 禁用降容清理。
func (s *DeviceService) SetExcessOffliner(o ExcessOffliner) {
	s.excessOffliner = o
}

// OfflineExcessDevices 按 license 各网元类型容量上限，把超出（created_at 晚接入的）
// 在线设备置离线（不删除）。实现 license.CapacityOffliner，供 license 降容后清理。
// 在线口径（issue #316）：离线设备不占容量，故腾容量只需置离线。
// 置离线后**清这些设备的 Redis 缓存**——否则缓存里 stale 的 is_online=true 会让被踢
// 设备下次 Inform 读到 oldIsOnline=true、跳过容量校验又上线，降容清理白做。
func (s *DeviceService) OfflineExcessDevices(ctx context.Context, typeCapacity map[string]int) (int, error) {
	if s.excessOffliner == nil || len(typeCapacity) == 0 {
		return 0, nil
	}
	sns, err := s.excessOffliner.OfflineExcessByTypeCapacity(ctx, typeCapacity)
	if err != nil {
		return 0, fmt.Errorf("offline excess devices: %w", err)
	}
	// 清被踢设备的缓存，让下次 Inform 重新从 DB load is_online=false → 触发容量校验。
	if s.cache != nil {
		for _, sn := range sns {
			s.cache.Delete(ctx, sn)
		}
	}
	if len(sns) > 0 {
		s.logger.Info("excess devices offlined after license capacity change", zap.Int("count", len(sns)))
	}
	return len(sns), nil
}

// ProductClassMatcher 是 DeviceService 反查 productClass → product 装配件的最小依赖。
//
// 生产环境由 *product.Registry 满足；测试可注入 stub。
// 通过接口而非直接依赖具体类型，避免在 device 包引入 product 包循环依赖风险。
//
// MatchProductClass 返回的 product 对象上的 `Name` 字段即装配件名（如 "mBS31001"），
// 用作 devices.model_name 的回填来源（设计文档 §4.3 方案 X）。
type ProductClassMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// SetProductMatcher 注入 product 装配件路由器（Phase 6 ModelName 回填）。
// 传 nil 等价于禁用回填——Inform 路径不影响。
func (s *DeviceService) SetProductMatcher(m ProductClassMatcher) {
	s.productMatcher = m
}

// ProductBinder 是 CreateDevice 把命中的 product 装配件写回 devices.product_id
// 的最小依赖（T-0176-PR-D §B）。
//
// 生产环境由 *product.PgRepository 满足；测试可注入 stub。
// 通过接口而非直接依赖避免 device 包对 product 仓储的强耦合。
//
// 注：BindDevice 与 *product.PgRepository.BindDevice 的签名保持一致 —— 当 INSERT
// 已完成（device.ID 已分配）后才被调用，写库失败仅 warn 不回滚 device 行
// （PR-D 事实修正：product_id 列只服务 admin/审计/未来 denorm 消费，
//
//	不再是 Go 运行路径关键 — PR-C 切完 resolver 后 Go 路径不读它）。
type ProductBinder interface {
	BindDevice(ctx context.Context, deviceID, productID uuid.UUID, paramModelID *uuid.UUID) error
}

// SetProductBinder 注入 ProductBinder（T-0176-PR-D）。
// 传 nil 等价于禁用回写——CreateDevice 进 inline match 时退化为"仅 metric + log"。
func (s *DeviceService) SetProductBinder(b ProductBinder) {
	s.productBinder = b
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
//  4. durable task Outbox 完成整轮入队后统一唤醒设备
//
// 替代旧 TriggerParamSync 方法（Path A 已下线），完整接入 T-0123/T-0124/T-0125/T-0127
// 触发链：reason 通道 + 差异日志 + last_param_sync_at 回写 + Translator 翻译。
//
// sourceID 由 caller 构造（"manual:UUID"），供 HandleSyncResultPathB 写差异日志时
// 通过 Redis hint 读取 reason 标签。
func (s *DeviceService) SyncDeviceParamsManual(ctx context.Context, deviceID uuid.UUID, sourceID string, parameterPaths []string) (used bool, dev *model.Device, gpvTaskCount int, err error) {
	result, dev, err := s.SyncDeviceParamsManualDetailed(ctx, deviceID, sourceID, parameterPaths)
	if result == nil {
		return false, dev, 0, err
	}
	return result.Used, dev, result.TaskCount, err
}

func (s *DeviceService) SyncDeviceParamsManualDetailed(ctx context.Context, deviceID uuid.UUID, sourceID string, parameterPaths []string) (result *ManualParamSyncStart, dev *model.Device, err error) {
	dev, err = s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("get device for manual sync: %w", err)
	}
	if dev == nil {
		return nil, nil, commonerrors.ErrNotFound
	}
	if s.paramSyncStarter == nil {
		return nil, dev, fmt.Errorf("paramSyncStarter not configured")
	}
	if !dev.IsOnline && s.manualOfflineModeRejects() {
		return nil, dev, commonerrors.NewBusinessError(
			global.ErrCodeDeviceOffline,
			"device is offline; parameter sync can only be started for online devices",
			commonerrors.ErrUnavailable,
		)
	}
	result, err = s.paramSyncStarter.StartManualSyncDetailed(ctx, dev, sourceID, parameterPaths)
	if err != nil {
		return result, dev, fmt.Errorf("start manual sync: %w", err)
	}

	s.logger.Info("manual sync requested",
		zap.String("device_id", deviceID.String()),
		zap.String("serial_number", dev.SerialNumber),
		zap.String("source_id", sourceID),
		zap.Int("parameter_paths", len(parameterPaths)),
		zap.Int("gpv_tasks", result.TaskCount),
		zap.Bool("path_b_used", result.Used))

	return result, dev, nil
}

// SetParamSyncStarter T-0126: 注入 Path B 同步 starter（消费者驱动 narrow interface）。
// 唯一实现者 *provision.SyncService。nil 时 SyncDeviceParamsManual 会返错。
func (s *DeviceService) SetParamSyncStarter(starter ParamSyncStarter) {
	s.paramSyncStarter = starter
}

// SetParamSyncManualOfflineMode controls whether manual durable requests for
// offline devices are queued or rejected before reaching the durable submitter.
func (s *DeviceService) SetParamSyncManualOfflineMode(mode string) {
	s.manualOfflineMode = strings.TrimSpace(mode)
}

func (s *DeviceService) manualOfflineModeRejects() bool {
	return s.manualOfflineMode != "queue"
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
	_, err := s.QueueRFSwitch(ctx, deviceID, enabled, RFSwitchTaskOptions{})
	return err
}

// RFSwitchTaskOptions carries the unified-task metadata for an RF command.
// Empty options preserve the existing manual API behaviour. Device-access
// containment uses the same RF path resolution and queue, but supplies its
// audited source, idempotency key and security_action admission class.
type RFSwitchTaskOptions struct {
	Source         task.TaskSource
	SourceID       string
	CreatorID      string
	Description    string
	CommandKey     string
	AdmissionClass task.AdmissionClass
	// TargetPaths pins SPV/GPV to the exact RF instances owned by the action.
	TargetPaths []string
}

// QueueRFSwitch resolves the carrier-specific RF path and queues the command
// through TaskService. It returns the durable task so callers can correlate
// dispatch and device response without creating a second RF path.
func (s *DeviceService) QueueRFSwitch(
	ctx context.Context,
	deviceID uuid.UUID,
	enabled bool,
	options RFSwitchTaskOptions,
) (*task.Task, error) {
	device, rfPaths, err := s.resolveRFControlTargets(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if len(options.TargetPaths) > 0 {
		rfPaths, err = pinRFControlTargets(rfPaths, options.TargetPaths)
		if err != nil {
			return nil, err
		}
	}

	// RF switch value: "1" for enabled, "0" for disabled.
	value := "0"
	if enabled {
		value = "1"
	}

	values := make([]map[string]string, 0, len(rfPaths))
	for _, rfPath := range rfPaths {
		values = append(values, map[string]string{
			"name": rfPath, "value": value, "type": "xsd:boolean",
		})
	}
	rfParamsJSON, err := json.Marshal(map[string]interface{}{"values": values})
	if err != nil {
		return nil, fmt.Errorf("encode RF switch task parameters: %w", err)
	}
	if options.Source == "" {
		options.Source = task.TaskSourceAPI
	}
	if options.AdmissionClass == "" {
		options.AdmissionClass = task.AdmissionClassNormal
	}
	if options.CommandKey == "" {
		options.CommandKey = uuid.New().String()
	}
	created, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   device.SerialNumber,
		Method:     "SetParameterValues",
		Params:     rfParamsJSON,
		CommandKey: options.CommandKey,
		Source:     options.Source,
		SourceID:   options.SourceID,
		CreatorID:  options.CreatorID,
		Description: func() string {
			if options.Description != "" {
				return options.Description
			}
			return "RF switch"
		}(),
		AdmissionClass: options.AdmissionClass,
	})
	if err != nil {
		return nil, fmt.Errorf("queue RF switch command: %w", err)
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

	return created, nil
}

// QueueRFReadback queues the correlated GPV used to verify an RF security
// action. It deliberately shares the same target/path resolution as the SPV;
// action ownership is established only after this task returns the requested
// value, not merely after SetParameterValuesResponse.
func (s *DeviceService) QueueRFReadback(
	ctx context.Context,
	deviceID uuid.UUID,
	options RFSwitchTaskOptions,
) (*task.Task, error) {
	device, rfPaths, err := s.resolveRFControlTargets(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if len(options.TargetPaths) > 0 {
		rfPaths, err = pinRFControlTargets(rfPaths, options.TargetPaths)
		if err != nil {
			return nil, err
		}
	}
	params, err := json.Marshal(map[string]interface{}{"names": rfPaths})
	if err != nil {
		return nil, fmt.Errorf("encode RF readback parameters: %w", err)
	}
	if options.Source == "" {
		options.Source = task.TaskSourceAPI
	}
	if options.AdmissionClass == "" {
		options.AdmissionClass = task.AdmissionClassNormal
	}
	if options.CommandKey == "" {
		options.CommandKey = uuid.New().String()
	}
	created, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN: device.SerialNumber, Method: "GetParameterValues", Params: params,
		Priority: 5, CommandKey: options.CommandKey, Source: options.Source,
		SourceID: options.SourceID, CreatorID: options.CreatorID,
		Description: options.Description, AdmissionClass: options.AdmissionClass,
	})
	if err != nil {
		return nil, fmt.Errorf("queue RF readback command: %w", err)
	}
	return created, nil
}

func pinRFControlTargets(resolved, requested []string) ([]string, error) {
	allowed := make(map[string]struct{}, len(resolved))
	for _, path := range resolved {
		if path = strings.TrimSpace(path); path != "" {
			allowed[path] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(requested))
	pinned := make([]string, 0, len(requested))
	for _, path := range requested {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := allowed[path]; !ok {
			return nil, fmt.Errorf("RF target path %q is not writable for current device snapshot: %w", path, commonerrors.ErrInvalidInput)
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		pinned = append(pinned, path)
	}
	if len(pinned) == 0 {
		return nil, fmt.Errorf("RF target paths are empty: %w", commonerrors.ErrInvalidInput)
	}
	sort.Strings(pinned)
	return pinned, nil
}

func (s *DeviceService) resolveRFControlTargets(
	ctx context.Context,
	deviceID uuid.UUID,
) (*model.Device, []string, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("get device for RF control: %w", err)
	}
	if device == nil {
		return nil, nil, commonerrors.ErrNotFound
	}
	if s.taskSvc == nil {
		return nil, nil, fmt.Errorf("task service not configured")
	}
	if s.carrierRegistry == nil {
		return nil, nil, fmt.Errorf("carrier registry not configured (T-0029 DI gap)")
	}
	c, err := s.carrierRegistry.Get(device.Carrier)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve carrier=%s for RF control: %w", device.Carrier, err)
	}
	rfPath := c.RFControlPath(device.Technology)
	if rfPath == "" {
		return nil, nil, fmt.Errorf("carrier=%s does not support RF control for technology=%s: %w",
			device.Carrier, device.Technology, commonerrors.ErrInvalidInput)
	}
	if device.Technology == model.TechLTE && s.paramRepo != nil {
		parameters, loadErr := s.paramRepo.GetByDevice(ctx, deviceID)
		if loadErr != nil {
			return nil, nil, fmt.Errorf("load device parameters for RF control: %w", loadErr)
		}
		if len(parameters) > 0 {
			paths := carrier.ResolveWritableRFControlPathsForProduct(device.ProductClass, parameters)
			if len(paths) > 0 {
				return device, paths, nil
			}
			if carrier.IsMBS31001ProductClass(device.ProductClass) {
				return nil, nil, fmt.Errorf(
					"mBS31001 RF control requires reported Device.DeviceInfo.SAS.RadioEnable: %w",
					commonerrors.ErrInvalidInput,
				)
			}
		}
	}
	return device, []string{rfPath}, nil
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

// InformRegistration reports both the current device and whether this Inform
// created a new device identity.
type InformRegistration struct {
	Device  *model.Device
	Created bool
}

const registrationSourceEventIDKey = "_system_registration_source_event_id"

// RegisterFromInform creates a new device from a bootstrap Inform message.
func (s *DeviceService) RegisterFromInform(ctx context.Context, inform *tr069.InformMessage, carrier model.CarrierCode) (*InformRegistration, error) {
	return s.RegisterFromInformEvent(ctx, inform, carrier, "")
}

// RegisterFromInformEvent preserves the source Inform event identity so a
// redelivered Bootstrap can recover a device.registered publication that
// failed after the device row was committed.
func (s *DeviceService) RegisterFromInformEvent(
	ctx context.Context,
	inform *tr069.InformMessage,
	carrier model.CarrierCode,
	sourceEventID string,
) (*InformRegistration, error) {
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
		registrationSourceEventID, _ := existing.ExtensionData[registrationSourceEventIDKey].(string)
		createdBySourceEvent := sourceEventID != "" && registrationSourceEventID == sourceEventID
		// Device already registered, just update it
		s.logger.Info("RegisterFromInform: device already exists, updating instead",
			zap.String("serial_number", inform.DeviceId.SerialNumber),
			zap.String("existing_device_id", existing.ID.String()))
		updated, err := s.UpdateFromInform(ctx, inform)
		if err != nil {
			return nil, err
		}
		return &InformRegistration{Device: updated, Created: createdBySourceEvent}, nil
	}

	// Device not found as active — check if it's soft-deleted in the recycle bin.
	// Recycle-bin devices are intentionally not auto-restored and must not create
	// a duplicate active row with a new UUID.
	deletedDevice, err := s.deviceRepo.GetDeletedBySerialNumber(ctx, inform.DeviceId.SerialNumber, carrier)
	if err != nil {
		s.logger.Error("RegisterFromInform: GetDeletedBySerialNumber failed",
			zap.Error(err),
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, fmt.Errorf("lookup deleted device: %w", err)
	}
	if deletedDevice != nil {
		s.logger.Info("RegisterFromInform: device found in recycle bin, skipping auto-registration",
			zap.String("serial_number", inform.DeviceId.SerialNumber),
			zap.String("deleted_device_id", deletedDevice.ID.String()))
		return nil, commonerrors.ErrNotFound
	}

	// License enforcement (E-04 + issue #316)：南向 Inform 自动注册同样受 license
	// 过期/fail-closed 约束；容量按网元类型独立限额（per-type）——先解析
	// productClass → ne_type，类型未知/未授权/超配额一律拒绝，与管理面
	// CreateDevice 对齐，避免南向绕过容量限制。仅对全新设备生效——已注册设备的
	// 更新路径不走这里（上方 existing != nil 分支已提前返回）。nil enforcer =
	// 执法未启用（轻量部署/测试）。
	if s.licenseEnforcer != nil {
		if err := s.licenseEnforcer.EnforceExpiry(ctx, "device.inform.register"); err != nil {
			return nil, err
		}
		neType, err := s.resolveNEType(ctx, inform.DeviceId.ProductClass)
		if err != nil {
			return nil, err
		}
		if err := s.licenseEnforcer.EnforceCapacity(ctx, neType, 1); err != nil {
			return nil, err
		}
	}

	modelName := findParamValue(inform.ParameterList, "Device.DeviceInfo.ModelName")
	firmwareVersion := findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")

	// Detect technology from Inform paths first, then known product identities.
	tech := detectTechnologyForInform(inform.DeviceId.ProductClass, modelName, firmwareVersion, inform.ParameterList)
	s.logger.Debug("RegisterFromInform: technology detected",
		zap.String("technology", string(tech)))

	now := time.Now()
	connReqURL := findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL")
	udpAddr := deriveUDPConnectionRequestAddress(inform.ParameterList)

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
		FirmwareVersion:             firmwareVersion,
		ConnectionRequestURL:        connReqURL,
		IPAddress:                   deriveInformIPAddress(udpAddr, connReqURL),
		NatDetected:                 udpAddr != "",
		UDPConnectionRequestAddress: udpAddr,
		LastInformAt:                &now,
		LastInformEvents:            tr069.EventCodes(inform.Event),
		InformInterval:              300,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}
	if sourceEventID != "" {
		device.ExtensionData = map[string]interface{}{
			registrationSourceEventIDKey: sourceEventID,
		}
	}

	// Phase 6: ProductRegistry 回填 model_name（TR-069 DeviceId 不含 ModelName）
	s.applyProductMetadata(ctx, device)

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

	// Assign new devices to the pre-registered group when present; otherwise to
	// the default L2 group. Matching rules may move the device later.
	if s.groupAssigner != nil {
		var preReg *DeviceRegistration
		if s.regRepo != nil {
			preReg, _ = s.regRepo.GetBySerialNumber(ctx, device.SerialNumber)
		}

		targetGroupID := uuid.MustParse(global.DefaultLevel2GroupID)
		if preReg != nil && preReg.GroupID != nil {
			targetGroupID = *preReg.GroupID
		}

		if _, err := s.groupAssigner.BatchAddDevices(ctx, targetGroupID, []uuid.UUID{device.ID}); err != nil {
			s.logger.Warn("assign new device to group",
				zap.String("device_id", device.ID.String()),
				zap.String("group_id", targetGroupID.String()),
				zap.Error(err))
		} else {
			s.logger.Info("new device assigned to group",
				zap.String("device_id", device.ID.String()),
				zap.String("group_id", targetGroupID.String()))
		}

		if preReg != nil {
			// Mark registration as online.
			s.regRepo.UpdateStatus(ctx, preReg.ID, string(global.RegistrationOnline))
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

	// 新设备 device_info 行已在 CreateDeviceInfo 创建（空字段）。这里立刻 sync 一遍把
	// 首次 Inform 携带的 LAC/TAC 等关键字段写入，让按 LAC/TAC 匹配的分组规则在首次
	// 注册就能命中，而不必等下一次周期 Inform。NULL→有值 也算变化，会触发
	// device.attributes.changed 事件 → GroupMatchEngine 异步归组。
	if s.infoSyncer != nil {
		changedAttrs, err := s.infoSyncer.SyncFromParameters(ctx, device.ID, device.Carrier, device.Technology, device.ProductClass)
		if err != nil {
			s.logger.Warn("RegisterFromInform: sync device info from parameters",
				zap.String("device_id", device.ID.String()),
				zap.Error(err))
		} else if len(changedAttrs) > 0 {
			s.PublishDeviceAttributesChangedEvent(ctx, device, changedAttrs)
		}
	}

	// Refresh heartbeat
	if s.reconciler != nil {
		s.reconciler.RefreshHeartbeat(ctx, device.SerialNumber, device.InformInterval)
	}

	s.logger.Info("device registered from bootstrap",
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrier)),
		zap.String("oui", device.OUI),
	)

	return &InformRegistration{Device: device, Created: true}, nil
}

func deriveInformIPAddress(udpAddr, connReqURL string) string {
	if udpAddr != "" {
		if host, _, err := net.SplitHostPort(udpAddr); err == nil {
			return host
		}
		return udpAddr
	}
	if connReqURL == "" {
		return ""
	}
	parsed, err := url.Parse(connReqURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// updateConnectionRequestSummary applies ConnectionRequestURL as a partial
// Inform field: absence, an empty value, or an invalid URL must not erase the
// last known management endpoint. Only a valid HTTP(S) URL updates both the
// persisted URL and its derived management IP address.
func updateConnectionRequestSummary(device *model.Device, params []tr069.ParameterValueStruct) {
	if device == nil {
		return
	}
	for _, param := range params {
		if param.Name != "Device.ManagementServer.ConnectionRequestURL" {
			continue
		}
		candidate := strings.TrimSpace(param.Value)
		if candidate == "" {
			return
		}
		parsed, err := url.ParseRequestURI(candidate)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
			return
		}
		device.ConnectionRequestURL = candidate
		device.IPAddress = parsed.Hostname()
		return
	}
}

func rebootDeviceType(tech model.Technology) string {
	switch tech {
	case model.TechNR:
		return "gNB"
	case model.TechGSM:
		return "GSM"
	default:
		return "eNB"
	}
}

func deriveUDPConnectionRequestAddress(params []tr069.ParameterValueStruct) string {
	udpAddr := strings.TrimSpace(findParamValue(params, "Device.ManagementServer.UDPConnectionRequestAddress"))
	if udpAddr != "" && !netutil.IsUnspecifiedUDPAddress(udpAddr) {
		return udpAddr
	}
	return ""
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
	oldIsOnline := device.IsOnline
	oldVersion := device.FirmwareVersion
	device.OUI = inform.DeviceId.OUI
	device.ProductClass = inform.DeviceId.ProductClass
	device.Manufacturer = inform.DeviceId.Manufacturer
	if inferredTech, ok := detectTechnologyFromPaths(inform.ParameterList); ok && device.Technology != inferredTech {
		s.logger.Info("UpdateFromInform: correcting device technology from inform paths",
			zap.String("serial_number", device.SerialNumber),
			zap.String("old_technology", string(device.Technology)),
			zap.String("new_technology", string(inferredTech)))
		device.Technology = inferredTech
	} else if inferredTech, ok := detectTechnologyFromIdentity(inform.DeviceId.ProductClass, findParamValue(inform.ParameterList, "Device.DeviceInfo.ModelName"), findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")); ok && device.Technology != inferredTech {
		s.logger.Info("UpdateFromInform: correcting device technology from product identity",
			zap.String("serial_number", device.SerialNumber),
			zap.String("old_technology", string(device.Technology)),
			zap.String("new_technology", string(inferredTech)),
			zap.String("product_class", inform.DeviceId.ProductClass))
		device.Technology = inferredTech
	}
	// Phase 6: ProductRegistry 回填 model_name（仅在 ModelName 为空时尝试）
	s.applyProductMetadata(ctx, device)
	device.FirmwareVersion = findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
	updateConnectionRequestSummary(device, inform.ParameterList)
	device.LastInformAt = &now
	device.LastInformEvents = tr069.EventCodes(inform.Event)
	// T-0162: 收到 Inform 即视为在线。对于已 scan 出 lifecycle_state 的设备，
	// normalizeDeviceForPersist 不会再从老 Status 反推新双字段；这里必须显式置 true，
	// 否则设备一旦被 OfflineDetector 标记成 is_online=false，后续正常 Inform 也无法恢复在线展示。
	//
	// license 容量校验（在线口径，issue #316）：仅离线→在线转换时校验，若该类型
	// 在线容量已满则保持离线（拒绝上线，不占容量）。已在线设备的周期 Inform 不重复校验。
	// 走 EnforceOnlineCapacity 与 periodic batch 路径统一，避免口径分裂。
	if !oldIsOnline && s.licenseEnforcer != nil {
		if capErr := s.EnforceOnlineCapacity(ctx, device.ProductClass); capErr != nil {
			s.logger.Warn("UpdateFromInform: device kept offline by license capacity",
				zap.String("serial_number", device.SerialNumber), zap.Error(capErr))
		} else {
			device.IsOnline = true
		}
	} else {
		device.IsOnline = true
	}
	udpAddr := deriveUDPConnectionRequestAddress(inform.ParameterList)
	if udpAddr != "" {
		device.IPAddress = deriveInformIPAddress(udpAddr, device.ConnectionRequestURL)
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
	} else {
		// 用户原则：UDP CR 地址只信设备本次 Inform 上报或 ACS STUN UDP 包源地址。
		// 本次 Inform 未上报有效地址 → 主动清除残留脏值（DB cache 可能携带历史派生兜底）。
		device.UDPConnectionRequestAddress = ""
		device.NatDetected = false
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

	// 设备离线后再次收到 Inform 时，Status 可能已是 active（不会触发 shouldActivate），
	// 但 is_online 会从 false 翻转到 true。该边沿必须刷新 last_online_time，
	// 避免本次在线时长跨越离线区间。
	if !oldIsOnline && device.IsOnline && s.infoSyncer != nil {
		if err := s.infoSyncer.RecordOnline(ctx, device.ID); err != nil {
			s.logger.Warn("record online time failed",
				zap.String("device_id", device.ID.String()),
				zap.Error(err))
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

	// Sync key parameters to device_info for fast query access.
	// 同时拿到 topology 关键列（LAC/TAC）的变化集，若非空发 device.attributes.changed
	// 让 GroupMatchEngine 异步重匹配该设备，避免等 @hourly cron 兜底。
	if s.infoSyncer != nil {
		changedAttrs, err := s.infoSyncer.SyncFromParameters(ctx, device.ID, device.Carrier, device.Technology, device.ProductClass)
		if err != nil {
			s.logger.Warn("sync device info from parameters",
				zap.String("device_id", device.ID.String()),
				zap.Error(err))
		} else if len(changedAttrs) > 0 {
			s.PublishDeviceAttributesChangedEvent(ctx, device, changedAttrs)
		}
	}

	// Refresh heartbeat
	if s.reconciler != nil {
		s.reconciler.RefreshHeartbeat(ctx, device.SerialNumber, device.InformInterval)
	}

	// T-0123/T-0125: 检测 firmware 变化与 offline→active 二选一发布事件。
	// 同一 Inform 满足两者时优先发 firmware.changed（不发 device.online），
	// 由 provision.HandleFirmwareChanged 触发模型刷新 + durable 全量同步覆盖 online 语义，
	// 避免两路全量同步重复触发。
	newVersion := device.FirmwareVersion
	firmwareChanged := oldVersion != "" && newVersion != "" && oldVersion != newVersion
	becameOnline := oldStatus == model.DeviceOffline && device.Status == model.DeviceActive
	if device.Status == model.DeviceActive && device.IsOnline {
		s.ClearDisconnectedAlarmOnOnline(ctx, device)
	}
	if firmwareChanged {
		s.PublishDeviceFirmwareChangedEvent(ctx, device, oldVersion, newVersion, becameOnline)
	} else if becameOnline {
		s.PublishDeviceOnlineEvent(ctx, device)
	}

	return device, nil
}

func (s *DeviceService) clearDisconnectedAlarmOnOnline(ctx context.Context, device *model.Device) {
	if s.disconnectAlarms == nil || s.disconnectClearer == nil || device == nil {
		return
	}
	identifier, _, ok := disconnectedAlarmForTechnology(device.Technology)
	if !ok {
		return
	}
	alarm, err := s.disconnectAlarms.GetActiveByDeviceAndIdentifier(ctx, device.SerialNumber, identifier)
	if err != nil {
		// 设备上线时没有活跃断连告警是最常见的正常情况（没触发过断连/已清除），
		// 不是错误——只对真正的查询失败（DB错误等）打WARN，避免每次设备上线都
		// 产生一条噪音日志（压测/大规模上线场景下会淹没真正的告警日志）。
		if !errors.Is(err, commonerrors.ErrNotFound) {
			s.logger.Warn("lookup disconnected alarm on device online failed",
				zap.Error(err),
				zap.String("device_id", device.ID.String()),
				zap.String("serial_number", device.SerialNumber),
				zap.String("alarm_identifier", identifier))
		}
		return
	}
	if alarm == nil {
		return
	}
	if alarm.AlarmSource != nil && strings.TrimSpace(*alarm.AlarmSource) != "" && strings.TrimSpace(*alarm.AlarmSource) != "OMC" {
		return
	}
	clearedBy := "system:device_online"
	clearNote := "device reported online"
	alarm.ClearedBy = &clearedBy
	alarm.ClearNote = &clearNote
	if err := s.disconnectClearer.ClearBySync(ctx, alarm); err != nil {
		s.logger.Warn("clear disconnected alarm on device online failed",
			zap.Error(err),
			zap.String("device_id", device.ID.String()),
			zap.String("serial_number", device.SerialNumber),
			zap.String("alarm_identifier", identifier))
	}
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

// ClearDisconnectedAlarmOnOnline 清理设备恢复在线时的 OMC 断链告警。
//
// 仅对 OMC 自己抬起的断链告警生效；外部源告警保持不动。
func (s *DeviceService) ClearDisconnectedAlarmOnOnline(ctx context.Context, device *model.Device) {
	s.clearDisconnectedAlarmOnOnline(ctx, device)
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
	var (
		stats    *DeviceListStats
		statsErr error
		wg       sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats, statsErr = s.deviceInfoRepo.ComputeListStats(ctx, filter)
	}()

	result, err := s.deviceInfoRepo.ListDevicesWithInfo(ctx, filter)
	wg.Wait()
	if err != nil {
		return nil, err
	}
	// T-0162: 同筛选条件下跑全量统计；stats 失败不阻断主 list 返回（降级返回
	// items + Stats=nil，前端会回退到老 fallback 行为，与现状等价）。
	if statsErr == nil {
		result.Stats = stats
	}
	if s.controlSummaryReader != nil && len(result.Items) > 0 {
		deviceIDs := make([]uuid.UUID, 0, len(result.Items))
		for index := range result.Items {
			deviceIDs = append(deviceIDs, result.Items[index].ID)
		}
		summaries, summaryErr := s.controlSummaryReader.ListCurrentByDeviceIDs(ctx, deviceIDs)
		if summaryErr != nil {
			// The control summary is an optional device-list decoration. A summary
			// query failure must not make the primary device inventory unavailable.
			if s.logger != nil {
				s.logger.Warn("list device control summaries failed; returning devices without summaries",
					zap.Error(summaryErr),
					zap.Int("device_count", len(deviceIDs)),
				)
			}
			return result, nil
		}
		for index := range result.Items {
			if summary, ok := summaries[result.Items[index].ID]; ok {
				summaryCopy := summary
				result.Items[index].ControlSummary = &summaryCopy
			}
		}
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
	result, err := s.deviceInfoRepo.GetByIDWithInfo(ctx, id)
	if err != nil || result == nil || s.controlSummaryReader == nil {
		return result, err
	}
	summaries, err := s.controlSummaryReader.ListCurrentByDeviceIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, fmt.Errorf("get device control summary: %w", err)
	}
	if summary, ok := summaries[id]; ok {
		result.ControlSummary = &summary
	}
	return result, nil
}

func (s *DeviceService) ListDeviceControlHistory(
	ctx context.Context,
	id uuid.UUID,
	page int,
	pageSize int,
) (*DeviceControlActionHistoryList, error) {
	if s.controlSummaryReader == nil {
		return &DeviceControlActionHistoryList{
			Items: []DeviceControlActionHistory{}, Page: page, PageSize: pageSize,
		}, nil
	}
	result, err := s.controlSummaryReader.ListHistoryByDeviceID(ctx, id, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list device control history: %w", err)
	}
	return result, nil
}

// applyProductMetadata 在 Inform 路径上把 ProductRegistry 装配件元数据
// 回填到 device 结构体（Phase 6 — 设计文档 §4.3 方案 X）。
//
// 当前仅回填 ModelName：TR-069 DeviceId 块不含 ModelName，CPE 参数树通常也
// 不上报 Device.DeviceInfo.ModelName，因此 devices.model_name 列长期为空。
// 通过 ProductRegistry.MatchProductClass 路由 productClass → product 装配件，
// 把 product.Name（装配件友好名称，如 "mBS31001"）回填进 device.ModelName。
//
// 行为：
//   - productMatcher 未注入 / device == nil → noop
//   - device.ModelName 已有值 → noop（不覆盖手填或前次回填）
//   - device.ProductClass 为空 → noop（无法路由）
//   - MatchProductClass 返 ErrOrphan / 任何错误 → fail-soft 仅 WARN，不阻断 Inform
//   - 命中且 product.Name != "" → 写入 device.ModelName（in-memory，caller 负责持久化）
//
// 由 RegisterFromInform 与 UpdateFromInform 在 Create/Update 持久化前各调用一次。
func (s *DeviceService) applyProductMetadata(ctx context.Context, device *model.Device) {
	if s == nil || s.productMatcher == nil || device == nil {
		return
	}
	if device.ProductClass == "" {
		return
	}
	// 不能在 ModelName 已回填时早返回 — Technology 可能还错着(老设备首次 Inform
	// 时 ProductRegistry 还没识别 productClass,Technology 被默认推断成 LTE;后来
	// ModelName 被补上,但 Technology 没人改)。两个字段独立判断。
	matchRes, err := s.productMatcher.MatchProductClass(ctx, device.ProductClass)
	if err != nil {
		// ErrOrphan 是合法业务态（productClass 未登记）；其他错误也都视为 soft fail：
		// Inform 主流程不依赖回填成功，下次 Inform 仍会重试。
		if !errors.Is(err, product.ErrOrphan) {
			s.logger.Warn("applyProductMetadata: MatchProductClass failed (non-fatal)",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_class", device.ProductClass),
				zap.Error(err))
		}
		return
	}
	if matchRes == nil || matchRes.Product == nil {
		return
	}
	if device.ModelName == "" && matchRes.Product.Name != "" {
		device.ModelName = matchRes.Product.Name
	}
	// Technology 回填:以产品字典为权威——
	//   · 登记了 tech("lte"/"nr"/"gsm") → 覆盖(修正首次 Inform 被默认推断成 LTE 的 5G 设备);
	//   · 产品已登记但 tech 为空(非无线产品,如核心网 ImsCore) → 清空,
	//     避免 detectTechnology 兜底推断的 lte 残留(设备列表错误显示 eNB(LTE))。
	if normalized := model.NormalizeTechnology(matchRes.Product.Tech); normalized != "" {
		if device.Technology != normalized {
			s.logger.Info("applyProductMetadata: technology corrected from ProductRegistry",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_class", device.ProductClass),
				zap.String("old", string(device.Technology)),
				zap.String("new", string(normalized)))
			device.Technology = normalized
		}
	} else if device.Technology != "" {
		s.logger.Info("applyProductMetadata: technology cleared (product registered without tech)",
			zap.String("serial_number", device.SerialNumber),
			zap.String("product_class", device.ProductClass),
			zap.String("old", string(device.Technology)))
		device.Technology = ""
	}
}

// resolveNEType 解析 productClass 对应的网元类型（products.alarm_ne_type），
// 供 per-type license 容量校验（EnforceCapacity）使用。
//
// 严格模式（issue #316）：以下任一情况都返回错误 → 调用方拒绝注册/创建：
//   - productMatcher 未注入（生产不应发生）；
//   - productClass 为空；
//   - 匹配返回 ErrOrphan（productClass 未在产品字典登记）；
//   - 命中但产品未配置 alarm_ne_type。
//
// 仅在 licenseEnforcer != nil 时调用，故 dev/test 不注入 enforcer 时不受影响。
func (s *DeviceService) resolveNEType(ctx context.Context, productClass string) (string, error) {
	pc := strings.TrimSpace(productClass)
	if pc == "" {
		return "", fmt.Errorf("product_class empty; cannot resolve ne_type for license enforcement: %w", commonerrors.ErrLicenseCapacityExceeded)
	}
	if s.productMatcher == nil {
		return "", fmt.Errorf("product matcher not configured; cannot resolve ne_type for license enforcement: %w", commonerrors.ErrLicenseCapacityExceeded)
	}
	mr, err := s.productMatcher.MatchProductClass(ctx, pc)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			return "", fmt.Errorf("product_class %q is not registered (orphan); device rejected by license enforcement: %w", pc, commonerrors.ErrLicenseCapacityExceeded)
		}
		return "", fmt.Errorf("resolve product (class=%s) for ne_type: %w", pc, err)
	}
	if mr == nil || mr.Product == nil || strings.TrimSpace(mr.Product.AlarmNeType) == "" {
		return "", fmt.Errorf("product_class %q matched no network-element type (alarm_ne_type); device rejected by license enforcement: %w", pc, commonerrors.ErrLicenseCapacityExceeded)
	}
	return mr.Product.AlarmNeType, nil
}

// EnforceOnlineCapacity 在设备即将从离线转为在线时，校验该网元类型的 license
// 容量（在线口径，issue #316）。返回 nil 表示容量充足、允许上线；返回非 nil
// （license 错误）表示该类型在线数已满、应保持离线。licenseEnforcer 未注入时
// 恒返回 nil。供 UpdateFromInform 与 periodic batch 路径统一调用——避免 batch
// 路径绕过容量限制（生产默认启用 batch processor）。
func (s *DeviceService) EnforceOnlineCapacity(ctx context.Context, productClass string) error {
	if s.licenseEnforcer == nil {
		return nil
	}
	neType, err := s.resolveNEType(ctx, productClass)
	if err != nil {
		return err
	}
	return s.licenseEnforcer.EnforceCapacity(ctx, neType, 1)
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

// ForceBootStateFlip 在收到 BOOT 信号时确定性地强制把设备先置为离线，让随后的
// Inform 驱动更新（UpdateFromInform）必然走出 "offline → active" 翻转、发出
// device.online 事件，从而在 OMC 上如实呈现一次 "下线 → 上线" 过程（issue #212）。
//
// 设计要点（与 issue #212 口径对齐）：
//   - 无条件：不论设备当前是否显示在线、不论被动超时离线探测是否已先行把它标离线，
//     都先显式写 is_online=false。已经离线的设备这一步幂等（无副作用）。
//   - 与被动超时离线探测彻底解耦：不读心跳超时链路，BOOT 是主动确定信号，直接驱动翻转。
//   - 只翻转 is_online（在线/离线维度），不动 lifecycle_state（入网生命周期维度）——
//     Commissioned + is_online=false → Status 派生为 Offline，正是 UpdateFromInform 里
//     becameOnline 判定所需的 oldStatus==Offline 前置。Decommissioned 等非入网态设备
//     的 lifecycle 不受影响。
//   - 发 device.offline 事件让 "下线" 半程对订阅方（前端推送 / 审计）也可见；随后由
//     UpdateFromInform 发 device.online 补齐 "上线" 半程。
//
// 注意：本方法只负责 "强制置离线 + 发 offline 事件 + 失效缓存"，"上线" 由调用方在其后
// 调用 UpdateFromInform 自然完成。EventBus / 仓储错误均仅 log Warn 不阻塞 BOOT 主流程
// （记录侧 RecordBootFromInform 仍会照常落库）。
func (s *DeviceService) ForceBootStateFlip(ctx context.Context, device *model.Device) {
	if device == nil {
		return
	}

	// 已离线则跳过写库与发事件（幂等）：避免对本就离线的设备凭空多发一条 offline。
	// 仍由调用方后续的 UpdateFromInform 把它带回在线并发 device.online。
	if !device.IsOnline {
		s.logger.Debug("ForceBootStateFlip: device already offline, skip forced offline write",
			zap.String("device_id", device.ID.String()),
			zap.String("serial_number", device.SerialNumber))
		return
	}

	if err := s.deviceRepo.UpdateOnlineStatus(ctx, device.ID, false); err != nil {
		s.logger.Warn("ForceBootStateFlip: force offline write failed",
			zap.String("device_id", device.ID.String()),
			zap.String("serial_number", device.SerialNumber),
			zap.Error(err))
		return
	}

	// 失效缓存，否则随后的 UpdateFromInform 走 cache 读到 is_online=true 的旧值，
	// becameOnline 判定不成立、不发 device.online（真机 2026-05-25 已暴露过同类缓存竞态）。
	device.IsOnline = false
	if s.cache != nil {
		s.cache.Delete(ctx, device.SerialNumber)
	}

	// 发 device.offline 让 "下线" 半程可见。reason=reboot 与超时离线（heartbeat_timeout）
	// 区分开，下游可据此知道这是 BOOT 驱动的确定性离线，而非被动超时判定。
	if s.eventBus != nil {
		payload := DeviceOfflineEvent{
			DeviceID:    device.ID,
			Serial:      device.SerialNumber,
			Carrier:     device.Carrier,
			Technology:  device.Technology,
			OfflineTime: time.Now(),
			Reason:      OfflineReasonReboot,
		}
		if evt, evtErr := event.NewEvent(event.SubjectDeviceOffline, payload); evtErr != nil {
			s.logger.Warn("ForceBootStateFlip: create device.offline event failed",
				zap.Error(evtErr), zap.String("device_id", device.ID.String()))
		} else if pubErr := s.eventBus.Publish(ctx, event.SubjectDeviceOffline, evt); pubErr != nil {
			s.logger.Warn("ForceBootStateFlip: publish device.offline failed",
				zap.Error(pubErr), zap.String("device_id", device.ID.String()))
		}
	}

	s.logger.Info("ForceBootStateFlip: device force-marked offline on BOOT",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))
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

// DeviceAttributesChangedEvent — 设备分组关键属性 (LAC/TAC) 实际变化时载荷。
// 详见 event.SubjectDeviceAttributesChanged 的发布/订阅说明。
//
// 订阅者（topology.GroupMatchEngine）拿到事件后会自己从 device_info 表
// 重新读 LAC/TAC 当前值再做匹配 —— 载荷里只带 device_id + serial_number
// 即可，避免把 *string 字段反复序列化又得在 receiver 重新解析。
type DeviceAttributesChangedEvent struct {
	DeviceID      uuid.UUID `json:"device_id"`
	SerialNumber  string    `json:"serial_number"`
	ChangedFields []string  `json:"changed_fields"`
}

// PublishDeviceAttributesChangedEvent 在 SyncFromParameters 检测到 device_info
// 拓扑关键列（LAC/TAC 等）实际变化后被调用，触发 GroupMatchEngine 异步重匹配。
// EventBus nil / Publish 失败均仅 log Warn，不阻塞 Inform 主流程。
func (s *DeviceService) PublishDeviceAttributesChangedEvent(ctx context.Context, device *model.Device, changedFields []string) {
	if s.eventBus == nil || len(changedFields) == 0 {
		return
	}
	payload := DeviceAttributesChangedEvent{
		DeviceID:      device.ID,
		SerialNumber:  device.SerialNumber,
		ChangedFields: changedFields,
	}
	evt, err := event.NewEvent(event.SubjectDeviceAttributesChanged, payload)
	if err != nil {
		s.logger.Error("create device.attributes.changed event", zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectDeviceAttributesChanged, evt); err != nil {
		s.logger.Warn("publish device.attributes.changed event", zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return
	}
	s.logger.Info("device.attributes.changed published",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.Strings("changed_fields", changedFields),
	)
}

// PublishDeviceRegistered publishes a device.registered event for the given device.
// Called by InformHandler on BOOTSTRAP/BOOT events to trigger provisioning engine.
func (s *DeviceService) PublishDeviceRegistered(
	ctx context.Context,
	device *model.Device,
	created bool,
	sourceEventIDs ...string,
) error {
	if s.eventBus == nil {
		return nil
	}
	payload := map[string]interface{}{
		"device_id":     device.ID,
		"serial_number": device.SerialNumber,
		"oui":           device.OUI,
		"product_class": device.ProductClass,
		"carrier":       string(device.Carrier),
		"technology":    string(device.Technology),
		"created":       created,
	}
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, payload)
	if err != nil {
		s.logger.Error("create device.registered event", zap.Error(err))
		return err
	}
	if len(sourceEventIDs) > 0 && sourceEventIDs[0] != "" {
		evt.ID = sourceEventIDs[0]
	}
	if err := s.eventBus.Publish(ctx, event.SubjectDeviceRegistered, evt); err != nil {
		s.logger.Warn("publish device.registered event", zap.Error(err))
		return err
	}
	return nil
}

// HaltReasonMainPath / HaltReasonDetailPath are the standard TR-181 paths
// carrying the abnormal-reboot signal. Vendor-specific aliases (X_BAICELLS_*,
// X_COM_* etc.) are normalized to these by the param-model Translator before
// reaching this layer; private-path resolution itself is therefore left to
// upstream code and not duplicated here.
const (
	HaltReasonMainPath    = "Device.HaltReason.MainReason"
	HaltReasonDetailPath  = "Device.HaltReason.DetailReason"
	gnbAbnormalMainReason = "halt_reboot"
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

// GetDevicePreRebootRunTime 返回 device_info.run_time 当前值（秒），供 1 BOOT 到来时
// 在 UpdateFromInform 覆盖该字段之前快照重启前运行时长。
// deviceInfoRepo 未注入或设备无 info 记录时返回 0。
func (s *DeviceService) GetDevicePreRebootRunTime(ctx context.Context, deviceID uuid.UUID) int64 {
	if s.deviceInfoRepo == nil {
		return 0
	}
	info, err := s.deviceInfoRepo.GetByDeviceID(ctx, deviceID)
	if err != nil || info == nil {
		return 0
	}
	return info.RunTime
}

// RecordBootFromInform handles the data updates triggered by a reboot-complete
// Inform (event codes "1 BOOT" or "M Reboot"). It:
//   - atomically increments devices.boot_count and stamps last_boot_at;
//   - refreshes the Redis device cache so the incremented counter is visible;
//   - publishes SubjectDeviceRebootAbnormal when HaltReason matches the
//     abnormal-reboot rule for the device type, so downstream listeners
//     (alarm engine, audit log) can react.
//
// preRebootRunTime 是调用方在 UpdateFromInform 覆盖 device_info.run_time 之前
// 读取的重启前设备运行时长（秒），来源于 device_info.run_time（DB 存储的上次同步值）。
// 新设备或 DB 无记录时传 0。
//
// The caller is expected to have already ensured the device row exists (via
// UpdateFromInform or RegisterFromInform). Returns the updated boot_count; 0
// with no error means the device could not be found and the boot was ignored.
func (s *DeviceService) RecordBootFromInform(ctx context.Context, device *model.Device, events []string, params []tr069.ParameterValueStruct, preRebootRunTime int64) (int, error) {
	return s.recordBootFromInform(ctx, device, nil, events, params, preRebootRunTime)
}

func (s *DeviceService) recordBootFromInform(ctx context.Context, device *model.Device, preRebootDevice *model.Device, events []string, params []tr069.ParameterValueStruct, preRebootRunTime int64) (int, error) {
	if device == nil {
		return 0, nil
	}
	snapshotDevice := device
	if preRebootDevice != nil {
		snapshotDevice = preRebootDevice
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
	//   - 所有设备都必须带 1 BOOT；
	//   - 5G gNB 设备必须明确上报 MainReason=halt_reboot 才算异常；
	//   - 其它设备沿用既有口径：ParameterList 有值时 MainReason 非空即异常；
	//     ParameterList 为空时退回旧事件码组合规则，保留兼容性。
	haltMainReason := strings.TrimSpace(extractParam(params, HaltReasonMainPath))
	haltDetailReason := strings.TrimSpace(extractParam(params, HaltReasonDetailPath))
	// runtimeBeforeReboot 来自调用方在 UpdateFromInform 覆盖 device_info.run_time 之前
	// 快照的旧值，即设备本次重启前的运行时长（秒）。
	runtimeBeforeReboot := preRebootRunTime

	hasBoot := hasEventCode(events, tr069.EventBoot)
	isGNB := snapshotDevice.Technology == model.TechNR
	abnormal := isAbnormalRebootInform(isGNB, hasBoot, events, params, haltMainReason)

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
			// snapshot 直接 freeze 重启前 devices 表字段值，空就是空（人工命名 /
			// IP 长期没回填等都是上游业务流程的事，不在异常重启识别这一步做兜底）。
			snap := AbnormalRebootSnapshot{
				DeviceID:            device.ID,
				DeviceSN:            device.SerialNumber,
				DeviceName:          snapshotDevice.DeviceName,
				DeviceType:          rebootDeviceType(snapshotDevice.Technology),
				OperateIP:           snapshotDevice.IPAddress,
				SoftwareVersion:     snapshotDevice.FirmwareVersion,
				HaltMainReason:      haltMainReason,
				HaltDetailReason:    haltDetailReason,
				RuntimeBeforeReboot: runtimeBeforeReboot,
				IsGNB:               isGNB,
				DetectedAt:          now,
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
			DeviceID:            device.ID,
			DeviceSN:            device.SerialNumber,
			DeviceName:          snapshotDevice.DeviceName,
			DeviceType:          rebootDeviceType(snapshotDevice.Technology),
			OperateIP:           snapshotDevice.IPAddress,
			SoftwareVersion:     snapshotDevice.FirmwareVersion,
			RuntimeBeforeReboot: runtimeBeforeReboot,
			IsGNB:               isGNB,
			BootCount:           bootCount,
			Events:              events,
			OccurredAt:          now,
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

func isAbnormalRebootInform(isGNB, hasBoot bool, events []string, params []tr069.ParameterValueStruct, haltMainReason string) bool {
	if !hasBoot {
		return false
	}
	if isGNB {
		return haltMainReason == gnbAbnormalMainReason
	}
	if len(params) == 0 {
		return !hasEventCode(events, tr069.EventMReboot)
	}
	return haltMainReason != ""
}

func hasEventCode(events []string, target string) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

var nrCellConfigPathPattern = regexp.MustCompile(`(?i)\.cellconfig\.\d+\.nr\.`)
var baiBNQProductClassPattern = regexp.MustCompile(`(?i)^fap/\w*bsc\w+`)

func detectTechnologyFromPaths(params []tr069.ParameterValueStruct) (model.Technology, bool) {
	hasLTE := false
	hasNR := false

	for _, p := range params {
		path := strings.ToLower(p.Name)
		if strings.Contains(path, ".cellconfig.lte.") || strings.Contains(path, ".fapcontrol.lte.") {
			hasLTE = true
		}
		if nrCellConfigPathPattern.MatchString(path) || strings.Contains(path, ".fapcontrol.nr.") {
			hasNR = true
		}
	}
	if hasNR {
		return model.TechNR, true
	}
	if hasLTE {
		return model.TechLTE, true
	}
	return "", false
}

func detectTechnologyFromIdentity(productClass, modelName, firmwareVersion string) (model.Technology, bool) {
	productClassLower := strings.ToLower(productClass)
	modelNameLower := strings.ToLower(modelName)
	firmwareLower := strings.ToLower(firmwareVersion)

	if baiBNQProductClassPattern.MatchString(productClass) ||
		strings.Contains(productClassLower, "bnq") ||
		strings.Contains(productClassLower, "bnx") ||
		strings.Contains(modelNameLower, "bnq") ||
		strings.Contains(modelNameLower, "bnx") ||
		strings.Contains(firmwareLower, "bnq") ||
		strings.Contains(firmwareLower, "bnx") {
		return model.TechNR, true
	}

	return "", false
}

// detectTechnology tries to determine the radio technology from Inform parameters.
//
// 首次 Inform 通常已经带有比 ModelName/Description 更稳定的路径特征：
//   - LTE: ...CellConfig.LTE... / ...FAPControl.LTE...
//   - NR:  ...CellConfig.{i}.NR... / ...FAPControl.NR...
//
// 因此优先按参数路径判定；只有路径没有提供制式信号时，才退回到
// ModelName/Description 的关键字启发式。
func detectTechnology(params []tr069.ParameterValueStruct) model.Technology {
	if tech, ok := detectTechnologyFromPaths(params); ok {
		return tech
	}

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

func detectTechnologyForInform(productClass, modelName, firmwareVersion string, params []tr069.ParameterValueStruct) model.Technology {
	if tech, ok := detectTechnologyFromPaths(params); ok {
		return tech
	}
	if tech, ok := detectTechnologyFromIdentity(productClass, modelName, firmwareVersion); ok {
		return tech
	}
	return detectTechnology(params)
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
func (s *DeviceService) GetAntennaSectors(ctx context.Context, deviceID uuid.UUID) ([]AntennaSector, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return nil, nil
	}

	params, err := s.paramRepo.GetByGroup(ctx, deviceID, "antenna")
	if err != nil {
		return nil, fmt.Errorf("get antenna params: %w", err)
	}
	sectors := AssembleAntennaSectors(params)
	if s.antennaPlanRepo == nil {
		return sectors, nil
	}
	plans, err := s.antennaPlanRepo.ListByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get antenna sector plans: %w", err)
	}
	return mergeAntennaSectorPlans(sectors, plans), nil
}

func (s *DeviceService) UpdateAntennaSectorPlan(
	ctx context.Context,
	deviceID uuid.UUID,
	sectorNumber int,
	req UpdateAntennaSectorPlanRequest,
) (*AntennaSector, error) {
	if sectorNumber < 1 || sectorNumber > 32767 || !validAntennaPlan(req) {
		return nil, commonerrors.ErrInvalidInput
	}
	if s.antennaPlanRepo == nil {
		return nil, fmt.Errorf("antenna sector plan repository is not configured: %w", commonerrors.ErrUnavailable)
	}
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return nil, commonerrors.ErrNotFound
	}
	plan := AntennaSectorPlan{
		DeviceID:            deviceID,
		SectorNumber:        sectorNumber,
		Azimuth:             req.Azimuth,
		AntennaHeight:       req.AntennaHeight,
		MechanicalDowntilt:  req.MechanicalDowntilt,
		HorizontalBeamwidth: req.HorizontalBeamwidth,
		VerticalBeamwidth:   req.VerticalBeamwidth,
	}
	params, err := s.paramRepo.GetByGroup(ctx, deviceID, "antenna")
	if err != nil {
		return nil, fmt.Errorf("get antenna params for plan validation: %w", err)
	}
	effectiveSectors := mergeAntennaSectorPlans(AssembleAntennaSectors(params), []AntennaSectorPlan{plan})
	for i := range effectiveSectors {
		if effectiveSectors[i].Number != sectorNumber {
			continue
		}
		if effectiveSectors[i].CoverageStatus == antennaCoverageStatusInvalidGeometry {
			return nil, fmt.Errorf(
				"invalid antenna coverage geometry (%s): %w",
				effectiveSectors[i].CoverageIssue,
				commonerrors.ErrInvalidInput,
			)
		}
		break
	}
	if err := s.antennaPlanRepo.Upsert(ctx, plan); err != nil {
		return nil, fmt.Errorf("save antenna sector plan: %w", err)
	}
	sectors, err := s.GetAntennaSectors(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	for i := range sectors {
		if sectors[i].Number == sectorNumber {
			return &sectors[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func validAntennaPlan(req UpdateAntennaSectorPlanRequest) bool {
	return validOptionalRange(req.Azimuth, 0, 360, true) &&
		validOptionalRange(req.AntennaHeight, 0, 10000, false) &&
		validOptionalRange(req.MechanicalDowntilt, 0, 90, true) &&
		validOptionalRange(req.HorizontalBeamwidth, 0, 180, false) &&
		validOptionalRange(req.VerticalBeamwidth, 0, 180, false)
}

func validOptionalRange(value *float64, min, max float64, allowMin bool) bool {
	if value == nil {
		return true
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) || *value >= max {
		return false
	}
	if allowMin {
		return *value >= min
	}
	return *value > min
}

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
	if device.Technology == model.TechLTE {
		paramValues := make(map[string]string, len(allParams))
		for _, param := range allParams {
			paramValues[param.ParameterPath] = param.ParameterValue
		}
		if mmeStatus := CalcMMEStatus(paramValues); mmeStatus != "" {
			if result.Info == nil {
				result.Info = &DeviceInfo{DeviceID: deviceID}
			}
			// 详情页以当前参数真值覆盖可能尚未回填的 device_info 快照。
			result.Info.MMEStatus = mmeStatus
		}
	}
	if device.Technology == model.TechNR {
		if result.Info == nil {
			result.Info = &DeviceInfo{DeviceID: deviceID}
		}
		if amfStatus := AssembleAMFStatus(allParams); amfStatus != "" {
			result.Info.MMEStatus = amfStatus
			result.Info.AMFStatus = amfStatus
		}
		if result.Info.AMFStatus == "" && result.Info.MMEStatus != "" {
			result.Info.AMFStatus = result.Info.MMEStatus
		}
		if multiPlmnEnable := AssembleMultiPlmnEnable(allParams); multiPlmnEnable != "" {
			result.Info.MultiPlmnEnable = multiPlmnEnable
		}
	}
	if gpsVersion := AssembleGPSVersion(allParams); gpsVersion != "" {
		if result.Info == nil {
			result.Info = &DeviceInfo{DeviceID: deviceID}
		}
		result.Info.GPSVersion = gpsVersion
	}
	if ppsTimeMode := AssemblePPSTimeMode(allParams); ppsTimeMode != "" {
		if result.Info == nil {
			result.Info = &DeviceInfo{DeviceID: deviceID}
		}
		result.Info.PPSTimeMode = ppsTimeMode
	}
	if rollbackVersion := AssembleRollbackVersion(allParams); rollbackVersion != "" {
		if result.Info == nil {
			result.Info = &DeviceInfo{DeviceID: deviceID}
		}
		result.Info.RollbackVersion = rollbackVersion
	}
	if wanStatus := AssembleWANStatus(allParams); wanStatus != "" {
		if result.Info == nil {
			result.Info = &DeviceInfo{DeviceID: deviceID}
		}
		result.Info.WANStatus = wanStatus
	}
	// BscLinkStatus 派生（GSM/BTS 专属）：详情接口走的是 *DeviceInfo 直接序列化，
	// 没经过 DeviceWithInfo 列表 DTO 的 SQL CASE，因此在这里手工补一次，
	// 让前端 "BSC连接状态" 字段能与列表页一致。
	if result.Info != nil && result.Info.OmlRemoteIp != nil && *result.Info.OmlRemoteIp != "" {
		if device.IsOnline {
			result.Info.BscLinkStatus = "connected"
		} else {
			result.Info.BscLinkStatus = "disconnected"
		}
	}
	// OmcStatus 派生：设备到 OMC 平台的连接态。基于 device.is_online 直接映射，
	// 前端 "OMC 连接状态" 字段使用。无需依赖 TR-069 参数，所有技术/厂商通用。
	if result.Info == nil {
		result.Info = &DeviceInfo{DeviceID: deviceID}
	}
	if device.IsOnline {
		result.Info.OmcStatus = "connected"
	} else {
		result.Info.OmcStatus = "disconnected"
	}
	result.Cells = AssembleCells(allParams, numOfCells, device.ProductClass)
	result.GSMCells = AssembleGSMCells(allParams)

	return result, nil
}

// CreateDevice creates a new device from an API request.
//
// T-0176-PR-D：INSERT 之前 inline 路由 productClass → product 装配件 →
// INSERT 后通过 ProductBinder 写回 product_id / param_model_id；productClass
// 为空 / productMatcher 未注入 / Registry 返 ErrOrphan 时跳过绑定（孤儿设备走
// 既有 orphan 列表 / RematchOrphan 流程）。绑定写库失败仅 warn，不回滚 device
// 行（PR-D 事实修正：product_id 列只服务 admin/审计/未来 denorm 消费）。
func (s *DeviceService) CreateDevice(ctx context.Context, req CreateDeviceRequest) (*model.Device, error) {
	existing, err := s.deviceRepo.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("check existing device: %w", err)
	}
	if existing != nil {
		return nil, commonerrors.ErrAlreadyExists
	}

	// License enforcement (T-0015 / R-103). nil enforcer = disabled.
	// 仅校验过期：CreateDevice 创建的是初始离线设备（is_online=false），不占在线
	// 容量（issue #316 在线口径）；设备真正上线（Inform）时由 RegisterFromInform /
	// UpdateFromInform 校验容量，故此处不做容量校验。
	if s.licenseEnforcer != nil {
		if err := s.licenseEnforcer.EnforceExpiry(ctx, "device.create"); err != nil {
			return nil, err
		}
	}

	// T-0176-PR-D：INSERT 之前先尝试路由 productClass，命中 / orphan / 真错三态分流。
	// 真错（DB 故障等）直接 return；不写脏数据。orphan 与命中都继续 INSERT，
	// 仅在 INSERT 完成后用拿到的 device.ID 调 BindDevice。
	var matchedProduct *product.Product
	if req.ProductClass != "" && s.productMatcher != nil {
		mr, mErr := s.productMatcher.MatchProductClass(ctx, req.ProductClass)
		switch {
		case mErr == nil && mr != nil && mr.Product != nil:
			matchedProduct = mr.Product
		case errors.Is(mErr, product.ErrOrphan):
			if s.metrics != nil {
				s.metrics.DeviceCreateOrphan.Inc()
			}
			s.logger.Info("device created as orphan; product_id left NULL",
				zap.String("serial_number", req.SerialNumber),
				zap.String("product_class", req.ProductClass))
		case mErr != nil:
			return nil, fmt.Errorf("resolve product for new device %s: %w", req.SerialNumber, mErr)
			// mErr == nil && mr == nil 的极端场景按 orphan 对待但不计 metric（Registry 契约保证
			// orphan 总返 ErrOrphan，到此说明运行期不一致 — 仅日志，不创造假数据）
		default:
			s.logger.Warn("device productClass match returned nil without ErrOrphan; treating as orphan",
				zap.String("serial_number", req.SerialNumber),
				zap.String("product_class", req.ProductClass))
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

	// T-0176-PR-D：INSERT 已成功，device.ID 已确定 — 写回 product_id（命中时）。
	// 写库失败仅 warn，不回滚 device 行（让 orphan 列表+RematchOrphan 兜底）。
	if matchedProduct != nil && s.productBinder != nil {
		if bindErr := s.productBinder.BindDevice(ctx, device.ID, matchedProduct.ID, matchedProduct.ParamModelID); bindErr != nil {
			s.logger.Warn("device created but product binding failed; leaving product_id NULL",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_id", matchedProduct.ID.String()),
				zap.Error(bindErr))
		} else {
			s.logger.Debug("device bound to product on create",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_id", matchedProduct.ID.String()),
				zap.String("product_name", matchedProduct.Name))
		}
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

	// site_name(设备名称)旧值 — 改名后触发分组重匹配用。
	oldDeviceName := device.DeviceName

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
		device.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		device.Longitude = req.Longitude
	}
	if req.LocationSourceMode != nil {
		if !req.LocationSourceMode.Valid() {
			return nil, fmt.Errorf(
				"invalid location source mode %q: %w",
				*req.LocationSourceMode,
				commonerrors.ErrInvalidInput,
			)
		}
		device.LocationSourceMode = *req.LocationSourceMode
	}
	if req.Status != nil {
		device.Status = *req.Status
	}

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}
	// T-0176-PR-D：UpdateDevice 写库后清 cache，让下次读经 GetOrLoad 重新加载。
	// 改 productClass 时尤其关键 — PR-C 切完 resolver 后，DeviceCache 缓存的 product_class
	// 直接喂给 ProductRegistry，stale 会让运行路径选错 product。
	if s.cache != nil {
		s.cache.Delete(ctx, device.SerialNumber)
	}

	// 改名(site_name 变化)后触发分组重匹配:名称是运维配置项、不走 Inform 同步路径,
	// 故在此复用 device.attributes.changed 事件链,让 GroupMatchEngine 按名称规则即时重新归组。
	if req.DeviceName != nil && device.DeviceName != oldDeviceName {
		s.PublishDeviceAttributesChangedEvent(ctx, device, []string{"site_name"})
	}
	return device, nil
}

// DeleteDevice deletes a device by ID.
func (s *DeviceService) DeleteDevice(ctx context.Context, id uuid.UUID) error {
	// C2 修复：先查 SN 用于删除后清 cache（cache key 是 SN 不是 ID）
	device, _ := s.deviceRepo.GetByID(ctx, id)
	if _, err := s.deviceRepo.BatchDelete(ctx, []uuid.UUID{id}, deletedByFromContext(ctx, "")); err != nil {
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
	deletedBy = deletedByFromContext(ctx, deletedBy)

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
	if deleted > 0 && s.groupCountsInvalidator != nil {
		s.groupCountsInvalidator.InvalidateDeviceGroupCounts()
	}

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

func deletedByFromContext(ctx context.Context, explicit string) string {
	if actor := strings.TrimSpace(explicit); actor != "" {
		return actor
	}
	if actor, ok := ctx.Value(admin.CtxKeyUsername).(string); ok {
		return strings.TrimSpace(actor)
	}
	return ""
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

// splitGeoGroupIDs 归一化前端传来的 group_ids。默认组现在是真实设备组，
// 保留 includeUngrouped 仅兼容历史无归属数据的显式空值分支。
func splitGeoGroupIDs(ids []string) (realIDs []string, includeUngrouped bool) {
	if len(ids) == 0 {
		return nil, false
	}
	realIDs = make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			includeUngrouped = true
			continue
		}
		realIDs = append(realIDs, id)
	}
	if len(realIDs) == 0 {
		realIDs = nil
	}
	return realIDs, includeUngrouped
}

// ListGeo returns devices with geographic coordinates for map display.
// filter.VisibleGroups 携带 #64 设备组数据权限，由 handler 解析调用者身份后注入。
// filter.GroupIDs 经 splitGeoGroupIDs 归一后传给 repository。
func (s *DeviceService) ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error) {
	filter.GroupIDs, filter.IncludeUngrouped = splitGeoGroupIDs(filter.GroupIDs)
	devices, total, err := s.deviceRepo.ListGeo(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if s.redisClient != nil && len(devices) > 0 {
		var countCmds []*redis.StringCmd
		var sevCmds []*redis.StringCmd
		var sevCountCmds []*redis.StringCmd
		pipe := s.redisClient.Pipeline()

		for _, d := range devices {
			key := "cache:device:alarm_stats:" + d.ID.String()
			countCmds = append(countCmds, pipe.HGet(ctx, key, "active_count"))
			sevCmds = append(sevCmds, pipe.HGet(ctx, key, "highest_severity"))
			sevCountCmds = append(sevCountCmds, pipe.HGet(ctx, key, "highest_count"))
		}
		_, _ = pipe.Exec(ctx)

		for i := range devices {
			countVal, _ := countCmds[i].Int()
			if countVal > 0 {
				devices[i].AlarmCount = countVal
				sevVal, err := sevCmds[i].Int()
				if err == nil && sevVal > 0 {
					// Copy value because &sevVal points to the loop variable causing bugs
					sevCopy := sevVal
					devices[i].HighestAlarmSeverity = &sevCopy
				}
				sevCountVal, _ := sevCountCmds[i].Int()
				if sevCountVal > 0 {
					devices[i].HighestSeverityAlarmCount = sevCountVal
				}
			} else {
				devices[i].AlarmCount = 0
			}
		}
	}

	return devices, total, nil
}

// GetGeoStats returns device statistics for map display.
// visibleGroups 携带 #64 设备组数据权限（nil 超管 / [] fail-closed / [g...] 限定）。
// statusFilter 与 ListGeo 同口径（handler 已把 onlineActive/onlineInactive/offline 三档翻译为
// model.DeviceStatus 列表），让顶部统计带随状态筛选变化与地图点位一致。
func (s *DeviceService) GetGeoStats(ctx context.Context, groupIDs []string, statusFilter []model.DeviceStatus, visibleGrants []model.DeviceVisibilityGrant) (*GeoStats, error) {
	realIDs, includeUngrouped := splitGeoGroupIDs(groupIDs)
	return s.deviceRepo.GetGeoStats(ctx, GeoStatsFilter{
		GroupIDs:            realIDs,
		IncludeUngrouped:    includeUngrouped,
		Status:              statusFilter,
		VisibleGroups:       flattenVisibleGroupIDs(visibleGrants),
		VisibleDeviceGrants: visibleGrants,
	})
}

// SearchDevices searches devices by keyword for map display.
// visibleGrants 携带 #64 设备数据权限（nil 超管 / [] fail-closed / [grant...] 限定）。
func (s *DeviceService) SearchDevices(ctx context.Context, keyword string, limit int, visibleGrants []model.DeviceVisibilityGrant) ([]GeoDevice, error) {
	devices, err := s.deviceRepo.SearchDevices(ctx, keyword, limit, visibleGrants)
	if err != nil {
		return nil, err
	}

	if s.redisClient != nil && len(devices) > 0 {
		var countCmds []*redis.StringCmd
		var sevCmds []*redis.StringCmd
		var sevCountCmds []*redis.StringCmd
		pipe := s.redisClient.Pipeline()

		for _, d := range devices {
			key := "cache:device:alarm_stats:" + d.ID.String()
			countCmds = append(countCmds, pipe.HGet(ctx, key, "active_count"))
			sevCmds = append(sevCmds, pipe.HGet(ctx, key, "highest_severity"))
			sevCountCmds = append(sevCountCmds, pipe.HGet(ctx, key, "highest_count"))
		}
		_, _ = pipe.Exec(ctx)

		for i := range devices {
			countVal, _ := countCmds[i].Int()
			if countVal > 0 {
				devices[i].AlarmCount = countVal
				sevVal, err := sevCmds[i].Int()
				if err == nil && sevVal > 0 {
					// Copy value because &sevVal points to the loop variable causing bugs
					sevCopy := sevVal
					devices[i].HighestAlarmSeverity = &sevCopy
				}
				sevCountVal, _ := sevCountCmds[i].Int()
				if sevCountVal > 0 {
					devices[i].HighestSeverityAlarmCount = sevCountVal
				}
			} else {
				devices[i].AlarmCount = 0
			}
		}
	}

	return devices, nil
}

// ===== Recycle Bin Operations =====

// ListRecycleBin returns soft-deleted devices with filtering.
// T-2026-07-02: 返回类型改为 DeviceWithInfo 以包含 device_info 字段（MAC、GPS、项目状态等）。
func (s *DeviceService) ListRecycleBin(ctx context.Context, filter RecycleBinFilter) (*model.ListResponse[DeviceWithInfo], error) {
	return s.deviceRepo.ListRecycleBin(ctx, filter)
}

// RestoreDevices restores soft-deleted devices.
// #378: 返回 RestoreResult（恢复数 + 冲突明细），可部分成功；仅真正 DB 错误才返回 err。
func (s *DeviceService) RestoreDevices(ctx context.Context, ids []uuid.UUID) (*RestoreResult, error) {
	// C2 修复：恢复后清 cache，让下次 inform 重新走 GetOrLoad 加载干净的 device 行
	idToSN, _ := s.deviceRepo.ListSerialsByIDs(ctx, ids)
	res, err := s.deviceRepo.RestoreDevices(ctx, ids)
	if err != nil {
		return res, err
	}
	if s.cache != nil {
		for _, sn := range idToSN {
			s.cache.Delete(ctx, sn)
		}
	}
	return res, nil
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

// ===== Device Name Sync (Issue #758) =====

const (
	hnbNameStandardPath = "Device.Services.FAPService.1.AccessMgmt.LTE.HNBName"
	gnbNameStandardPath = "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBName"
)

// lmtDeviceNameStandardPath 返回当前制式的基站侧名称参数。
func lmtDeviceNameStandardPath(dev *model.Device) string {
	if dev != nil && dev.Technology == model.TechNR {
		return gnbNameStandardPath
	}
	return hnbNameStandardPath
}

// ResolveNameSync 处理设备名称同步人工确认。
//
// action:
//   - "use_lmt": 使用 LMT 名称覆盖网管名称
//   - "use_omc": 使用网管名称下发到 LMT
//   - "ignore": 忽略差异（清标记，保持各自名称不变）
func (s *DeviceService) ResolveNameSync(ctx context.Context, deviceID uuid.UUID, action string) error {
	if s.deviceInfoRepo == nil {
		return fmt.Errorf("device info repository not configured")
	}

	// 获取当前 device_info
	info, err := s.deviceInfoRepo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device info: %w", err)
	}
	if info == nil {
		return commonerrors.ErrNotFound
	}

	switch action {
	case "use_lmt":
		// 使用 LMT 名称：同时更新 device_info.device_name（详情页口径）
		// 和 devices.site_name（列表页口径），保证前端两处显示一致（P1-1 修复）。
		if info.LMTDeviceName != "" {
			if err := s.deviceInfoRepo.UpdateDeviceName(ctx, deviceID, info.LMTDeviceName); err != nil {
				return fmt.Errorf("update device_info.device_name: %w", err)
			}
			if err := s.deviceRepo.UpdateSiteName(ctx, deviceID, info.LMTDeviceName); err != nil {
				// 列表名更新失败降级警告，不阻断主流程（详情页已更新）
				s.logger.Warn("update devices.site_name failed after use_lmt",
					zap.String("device_id", deviceID.String()),
					zap.Error(err))
			}
			// 清 cache，让列表下次读到新 site_name
			if s.cache != nil {
				if dev, _ := s.deviceRepo.GetByID(ctx, deviceID); dev != nil {
					s.cache.Delete(ctx, dev.SerialNumber)
				}
			}
		}
		if err := s.deviceInfoRepo.UpdateNameSyncFields(ctx, deviceID, false, info.LMTDeviceName); err != nil {
			return fmt.Errorf("clear name_sync_pending: %w", err)
		}
		s.logger.Info("device name sync resolved: use_lmt",
			zap.String("device_id", deviceID.String()),
			zap.String("lmt_name", info.LMTDeviceName))

	case "use_omc":
		// 使用网管名称：下发 SPV 到设备，成功后清 pending 标记
		if info.DeviceName == "" {
			return fmt.Errorf("omc device_name is empty, cannot push to device")
		}

		// 获取设备 SN 用于下发任务
		dev, err := s.deviceRepo.GetByID(ctx, deviceID)
		if err != nil {
			return fmt.Errorf("get device: %w", err)
		}
		if dev == nil {
			return commonerrors.ErrNotFound
		}

		// 构建 SPV 参数并下发
		if s.taskSvc == nil {
			return fmt.Errorf("task service not configured, cannot push name to device")
		}

		spvParams := []map[string]string{{
			"name":  lmtDeviceNameStandardPath(dev),
			"value": info.DeviceName,
			"type":  "xsd:string",
		}}
		paramsJSON, err := json.Marshal(map[string]interface{}{
			"values":        spvParams,
			"parameter_key": fmt.Sprintf("name-sync-%d", time.Now().Unix()),
		})
		if err != nil {
			return fmt.Errorf("marshal SPV params: %w", err)
		}

		createdTask, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   dev.SerialNumber,
			Method:     "SetParameterValues",
			Params:     paramsJSON,
			Priority:   5,
			CommandKey: fmt.Sprintf("name-sync-%s", uuid.New().String()[:8]),
			Source:     task.TaskSourceAPI,
		})
		if err != nil {
			return fmt.Errorf("queue name sync SPV: %w", err)
		}

		// SPV 已入队，清除 pending 标记
		if err := s.deviceInfoRepo.UpdateNameSyncFields(ctx, deviceID, false, info.LMTDeviceName); err != nil {
			return fmt.Errorf("clear name_sync_pending: %w", err)
		}
		s.logger.Info("device name sync resolved: use_omc (SPV queued)",
			zap.String("device_id", deviceID.String()),
			zap.String("omc_name", info.DeviceName),
			zap.String("task_id", createdTask.ID))

	case "ignore":
		// 忽略差异：清除 pending 标记
		if err := s.deviceInfoRepo.UpdateNameSyncFields(ctx, deviceID, false, info.LMTDeviceName); err != nil {
			return fmt.Errorf("clear name_sync_pending: %w", err)
		}
		s.logger.Info("device name sync resolved: ignore",
			zap.String("device_id", deviceID.String()))

	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	return nil
}

// SetSysConfigLookup 注入系统配置查询函数（nameSyncMode 读取用）。
func (s *DeviceService) SetSysConfigLookup(fn SysConfigLookup) {
	s.sysConfigLookup = fn
}

// loadNameSyncMode 从 sys_configs 读取 nameSyncMode，未配置或未知值均回落 "prompt"。
func (s *DeviceService) loadNameSyncMode(ctx context.Context) string {
	if s.sysConfigLookup == nil {
		return "prompt"
	}
	v, ok := s.sysConfigLookup(ctx, "device", "nameSyncMode")
	if !ok {
		return "prompt"
	}
	switch v {
	case "auto_lmt_to_omc", "auto_omc_to_lmt", "prompt":
		return v
	default:
		return "prompt"
	}
}

// RenameDevice 从网管侧修改设备名称并按 nameSyncMode 策略决定是否下发到基站。
//
// 行为矩阵（见设计文档 §2.1）：
//   - auto_lmt_to_omc → 仅修改网管名称；后续设备上报时仍按策略以 LMT 名称覆盖
//   - auto_omc_to_lmt → 双写网管库 + 建 SPV 下发任务 + 清 pending
//   - prompt          → 双写网管库 + 置 pending=true（lmtName 取旧值原样回写）
//
// 下发失败不回滚网管库（与现有人工下发语义一致）。
func (s *DeviceService) RenameDevice(ctx context.Context, id uuid.UUID, newName string, creatorID string) (*RenameDeviceResult, error) {
	if strings.TrimSpace(newName) == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", commonerrors.ErrInvalidInput)
	}

	result := &RenameDeviceResult{}
	mode := s.loadNameSyncMode(ctx)

	// 1. 读设备（需要 SN 做缓存清理和 SPV 下发）
	dev, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if dev == nil {
		return nil, commonerrors.ErrNotFound
	}

	// 2. 读 device_info（prompt 模式下需要 lmt_device_name 旧值防止污染缓存）
	info, err := s.deviceInfoRepo.GetByDeviceID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get device info: %w", err)
	}
	if info == nil {
		return nil, commonerrors.ErrNotFound
	}

	// 3. 双写：device_info.device_name（详情口径）+ devices.site_name（列表口径）
	if err := s.deviceInfoRepo.UpdateDeviceName(ctx, id, newName); err != nil {
		return nil, fmt.Errorf("update device_info.device_name: %w", err)
	}
	if err := s.deviceRepo.UpdateSiteName(ctx, id, newName); err != nil {
		// 列表名更新失败降级警告，不全量回滚（详情页已更新）
		s.logger.Warn("rename: update devices.site_name failed",
			zap.String("device_id", id.String()),
			zap.Error(err))
	}
	// 清缓存，让列表下次读到新 site_name
	if s.cache != nil {
		s.cache.Delete(ctx, dev.SerialNumber)
	}
	// 触发分组重匹配（名称规则）
	dev.DeviceName = newName
	s.PublishDeviceAttributesChangedEvent(ctx, dev, []string{"site_name"})

	// 4. 按策略决定下发与 pending
	lmtName := info.LMTDeviceName // 旧值，用于 pending 时回写和 auto 路径清 pending

	switch mode {
	case "auto_lmt_to_omc":
		// 人工修改只作用于网管名称，不下发 LMT，也不制造人工确认 pending。
		// 后续设备名称上报时，自动同步逻辑仍会按当前策略采用 LMT 名称。
	case "auto_omc_to_lmt":
		// 建 SPV 下发任务（按制式选择 HNBName/gNBName，ACS 侧翻私有路径）
		if s.taskSvc != nil {
			spvParams := []map[string]string{{
				"name":  lmtDeviceNameStandardPath(dev),
				"value": newName,
				"type":  "xsd:string",
			}}
			paramsJSON, merr := json.Marshal(map[string]interface{}{
				"values":        spvParams,
				"parameter_key": fmt.Sprintf("rename-%d", time.Now().Unix()),
			})
			if merr == nil {
				createdTask, derr := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
					DeviceSN:   dev.SerialNumber,
					Method:     "SetParameterValues",
					Params:     paramsJSON,
					Priority:   5,
					CommandKey: fmt.Sprintf("rename-%s", uuid.New().String()[:8]),
					Source:     task.TaskSourceAPI,
					CreatorID:  creatorID, // T-0157 C5: 让 rename 下发任务进入消息中心
				})
				if derr != nil {
					// 下发失败不回滚网管库，仅 Warn
					s.logger.Warn("rename: SPV dispatch failed (network name already updated)",
						zap.String("device_id", id.String()),
						zap.Error(derr))
				} else if createdTask != nil {
					result.TaskID = createdTask.ID
				}
			}
		}
		// 清 pending（lmtName 传旧值，不污染"基站侧最后上报名"缓存）
		if cerr := s.deviceInfoRepo.UpdateNameSyncFields(ctx, id, false, lmtName); cerr != nil {
			s.logger.Warn("rename: clear name_sync_pending failed",
				zap.String("device_id", id.String()),
				zap.Error(cerr))
		}

	case "prompt":
		// 不下发，置 pending=true；lmtName 必须传当前数据库旧值，不能传新名或空
		if perr := s.deviceInfoRepo.UpdateNameSyncFields(ctx, id, true, lmtName); perr != nil {
			return nil, fmt.Errorf("rename: set name_sync_pending: %w", perr)
		}
	}

	s.logger.Info("device renamed",
		zap.String("device_id", id.String()),
		zap.String("new_name", newName),
		zap.String("mode", mode))
	return result, nil
}
