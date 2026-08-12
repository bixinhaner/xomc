package pm

import (
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/paramsync"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/redis/go-redis/v9"
)

// 编译时引用 device 包确保 DeviceOnlineEvent 类型存在（payload decode 用）。
var _ = device.DeviceOnlineEvent{}

// OnlineSubscriber 订阅 device.online 事件，在设备由 offline→active 时
// 自动下发 PM 文件上传配置（KPI 上报参数整理.md 三参数）。
//
// 触发时机（由 device.DeviceService 决定）：
//   - 非批量路径：UpdateFromInform 检测到 oldStatus=offline && device.Status=active
//   - 批量路径：BatchInformProcessor.doFlush 同条件
//   - firmware 变化时不发 device.online（发 firmware.changed，由 provision 补交 durable 全量同步）—
//     本订阅器不订阅 firmware.changed，避免与 provision 的 GPN/GPV 同步抢资源
//
// 节流：device.online 事件在 provision 模块已有 60s Redis token bucket 防抖，
// 本订阅器不重复实现（多 worker 实例共享节流由 QueueSubscribe 配合 token bucket 完成）。
//
// 失败处理：临时 Redis / PostgreSQL / 入队失败返回 error，由 EventBus 受控重投；
// payload 或 URL 配置永久错误仅记录告警。系统级任务无 user_id，不写消息中心。
type OnlineSubscriber struct {
	taskSvc     TaskCreator
	urlTemplate string
	enableValue string
	intervalSec int
	// resolveBaseURL 可选：运行时解析 PM 上传基址（通常读 sys_configs
	// acs_transfer.uploadBaseURL，与「系统配置→ACS 传输」页同源）。非空合法时
	// 覆盖 urlTemplate 的 scheme/host（path+query 仍取自 urlTemplate）；为空回退 urlTemplate。
	resolveBaseURL func(ctx context.Context) string
	resolveAddress PMUploadAddressResolver
	admissionGate  PMSetupAdmissionGate
	deviceLookup   PMDeviceLookup
	parameterRead  PMDeviceParameterReader
	logger         *zap.Logger
}

// defaultPMUploadPathQuery 是 urlTemplate 不可解析时回退用的 PM 上传 path+query，
// 与 config.*.yaml 的 pm.upload_url_template 末段保持一致。
const defaultPMUploadPathQuery = "/smallcell/FileUploadService?fileType=PM&filename="
const automatedPMTaskRetryIntervalSeconds = 30
const pmSetupAdmissionTTL = 30 * time.Second
const pmSetupCommandKey = "pm_upload_setup_on_online"
const pmHTTPSCompensationCommandKey = "pm_upload_https_compensation"
const pmHTTPSCompensationDescription = "Compensate PM upload URL to HTTPS after parameter sync"
const pmUploadURLParameterPath = "Device.FAP.PerfMgmt.Config.1.URL"

// TaskCreator 是 OnlineSubscriber 入队 SPV task 的最小依赖。
// 真实实现是 *task.TaskService。
type TaskCreator interface {
	CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
	LatestOpenTaskByDeviceAndMethod(
		ctx context.Context,
		deviceSN, method, description string,
	) (*task.Task, error)
	ListOpenTasksByDeviceAndMethods(
		ctx context.Context,
		deviceSN string,
		methods []string,
	) ([]*task.Task, error)
	LatestCompletedTaskByDeviceAndCommandKey(
		ctx context.Context,
		deviceSN, commandKey string,
	) (*task.Task, error)
}

// PMSetupAdmissionGate coalesces registered/online events across worker
// instances while one handler checks durable state and creates a task.
type PMSetupAdmissionGate interface {
	Acquire(ctx context.Context, deviceSN string) (leaseToken string, acquired bool, err error)
	Renew(ctx context.Context, deviceSN, leaseToken string) (renewed bool, err error)
	Release(ctx context.Context, deviceSN, leaseToken string) error
}

type PMUploadAddressResolver interface {
	Resolve(
		ctx context.Context,
		deviceID uuid.UUID,
		direction transfercfg.TransferDirection,
	) (transfercfg.AddressDecision, error)
}

type PMDeviceLookup interface {
	GetByID(ctx context.Context, deviceID uuid.UUID) (*model.Device, error)
}

type PMDeviceParameterReader interface {
	GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
}

type redisPMSetupAdmissionGate struct {
	client redis.Cmdable
	ttl    time.Duration
}

type admissionRenewalIntervalProvider interface {
	RenewalInterval() time.Duration
}

var releasePMSetupAdmissionScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

var renewPMSetupAdmissionScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

func NewRedisPMSetupAdmissionGate(
	client redis.Cmdable,
	ttl time.Duration,
) PMSetupAdmissionGate {
	if client == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = pmSetupAdmissionTTL
	}
	return &redisPMSetupAdmissionGate{client: client, ttl: ttl}
}

func (g *redisPMSetupAdmissionGate) Acquire(
	ctx context.Context,
	deviceSN string,
) (string, bool, error) {
	leaseToken := uuid.NewString()
	acquired, err := g.client.SetNX(
		ctx,
		redisx.Keys.PMUploadSetupAdmission(deviceSN),
		leaseToken,
		g.ttl,
	).Result()
	if err != nil {
		return "", false, fmt.Errorf("acquire PM setup admission: %w", err)
	}
	if !acquired {
		return "", false, nil
	}
	return leaseToken, true, nil
}

func (g *redisPMSetupAdmissionGate) Release(
	ctx context.Context,
	deviceSN, leaseToken string,
) error {
	if leaseToken == "" {
		return nil
	}
	if err := releasePMSetupAdmissionScript.Run(
		ctx,
		g.client,
		[]string{
			redisx.Keys.PMUploadSetupAdmission(deviceSN),
		},
		leaseToken,
	).Err(); err != nil {
		return fmt.Errorf("release PM setup admission: %w", err)
	}
	return nil
}

func (g *redisPMSetupAdmissionGate) Renew(
	ctx context.Context,
	deviceSN, leaseToken string,
) (bool, error) {
	if leaseToken == "" {
		return false, nil
	}
	result, err := renewPMSetupAdmissionScript.Run(
		ctx,
		g.client,
		[]string{redisx.Keys.PMUploadSetupAdmission(deviceSN)},
		leaseToken,
		g.ttl.Milliseconds(),
	).Int64()
	if err != nil {
		return false, fmt.Errorf("renew PM setup admission: %w", err)
	}
	return result == 1, nil
}

func (g *redisPMSetupAdmissionGate) RenewalInterval() time.Duration {
	return g.ttl / 3
}

// NewOnlineSubscriber 构造订阅器。urlTemplate 支持 ${VAR} 与 ${VAR:-default} 插值。
func NewOnlineSubscriber(
	taskSvc TaskCreator,
	urlTemplate string,
	enableValue string,
	intervalSec int,
	logger *zap.Logger,
) *OnlineSubscriber {
	if enableValue == "" {
		enableValue = "1"
	}
	if intervalSec <= 0 {
		intervalSec = 900
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OnlineSubscriber{
		taskSvc:     taskSvc,
		urlTemplate: urlTemplate,
		enableValue: enableValue,
		intervalSec: intervalSec,
		logger:      logger.Named("pm.online-subscriber"),
	}
}

// SetUploadBaseURLResolver 注入运行时上传基址解析器（通常读 sys_configs
// acs_transfer.uploadBaseURL）。设置后，PM 上传 URL 的 scheme/host 用解析结果、
// path+query 仍取自 urlTemplate；解析为空则回退 urlTemplate（${OMC_PUBLIC_HOST} 插值）。
// 这样上传地址与「系统配置→ACS 传输」页同源，改 IP 无需重建镜像。
func (s *OnlineSubscriber) SetUploadBaseURLResolver(fn func(ctx context.Context) string) {
	s.resolveBaseURL = fn
}

func (s *OnlineSubscriber) SetUploadAddressResolver(resolver PMUploadAddressResolver) {
	s.resolveAddress = resolver
}

func (s *OnlineSubscriber) SetAdmissionGate(gate PMSetupAdmissionGate) {
	s.admissionGate = gate
}

func (s *OnlineSubscriber) SetParamSyncPMCompensationReaders(
	deviceLookup PMDeviceLookup,
	parameterReader PMDeviceParameterReader,
) {
	s.deviceLookup = deviceLookup
	s.parameterRead = parameterReader
}

// Subscribe 把 OnlineSubscriber 挂到 EventBus。使用 QueueSubscribe 让多 worker 实例
// 同时启动时只有一个实例处理同一事件（避免重复入队 SPV task）。
//
// queueName 固定 "pm-online-pm-upload-setup" — 与 provision 的 "provision-online-sync"
// 处于不同 queue group，互不抢消息（两个订阅者都会各自收到事件，是设计意图）。
func (s *OnlineSubscriber) Subscribe(bus event.EventBus) error {
	// device.online 只在 offline→active 翻转时发（重连/重启上线）。
	const onlineQueue = "pm-online-pm-upload-setup"
	if _, err := bus.QueueSubscribe(event.SubjectDeviceOnline, onlineQueue, s.handleOnline); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceOnline, err)
	}
	// device.registered 在「首次 onboard」（BOOTSTRAP/BOOT/未知设备自动注册）发——
	// 与 device.online 并订，确保**首次上线**也下发 PM 上报配置（不只重连）。
	// 独立 queue group，互不抢消息；同 CommandKey 幂等，与 online 重复下发同值无害。
	const registeredQueue = "pm-registered-pm-upload-setup"
	if _, err := bus.QueueSubscribe(event.SubjectDeviceRegistered, registeredQueue, s.handleRegistered); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceRegistered, err)
	}
	if err := s.SubscribeParamSyncCompleted(bus); err != nil {
		return err
	}
	s.logger.Info("pm online subscriber registered",
		zap.String("subjects", event.SubjectDeviceOnline+","+event.SubjectDeviceRegistered+","+event.SubjectParamSyncRunCompleted),
		zap.String("url_template", s.urlTemplate),
		zap.Int("interval_seconds", s.intervalSec))
	return nil
}

func (s *OnlineSubscriber) SubscribeParamSyncCompleted(bus event.EventBus) error {
	if s.deviceLookup != nil && s.parameterRead != nil && s.resolveAddress != nil {
		const paramSyncQueue = "pm-param-sync-https-compensation"
		if _, err := bus.QueueSubscribe(
			event.SubjectParamSyncRunCompleted,
			paramSyncQueue,
			s.handleParamSyncCompleted,
		); err != nil {
			return fmt.Errorf("subscribe %s: %w", event.SubjectParamSyncRunCompleted, err)
		}
	}
	return nil
}

// handleOnline 处理 device.online（offline→active 重连上线）。
func (s *OnlineSubscriber) handleOnline(ctx context.Context, evt event.Event) error {
	var payload device.DeviceOnlineEvent
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.online payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		return nil // payload 损坏，重试无意义
	}
	return s.enqueuePMSetup(ctx, payload.SerialNumber, payload.DeviceID.String(), false)
}

// handleRegistered 处理 device.registered（首次 onboard）。payload 是 map 结构
// （device_service.go PublishDeviceRegistered），取 serial_number / device_id。
func (s *OnlineSubscriber) handleRegistered(ctx context.Context, evt event.Event) error {
	var payload struct {
		SerialNumber string `json:"serial_number"`
		DeviceID     string `json:"device_id"`
		Created      bool   `json:"created"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.registered payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		return nil
	}
	return s.enqueuePMSetup(ctx, payload.SerialNumber, payload.DeviceID, payload.Created)
}

func (s *OnlineSubscriber) handleParamSyncCompleted(ctx context.Context, evt event.Event) error {
	var payload struct {
		RunID     uuid.UUID `json:"run_id"`
		DeviceID  uuid.UUID `json:"device_id"`
		Status    string    `json:"status"`
		SyncScope string    `json:"sync_scope"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode param_sync.run.completed payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		return nil
	}
	if payload.RunID == uuid.Nil {
		s.logger.Info("skip PM HTTPS compensation: parameter sync event missing run_id",
			zap.String("event_id", evt.ID))
		return nil
	}
	if payload.DeviceID == uuid.Nil {
		s.logger.Info("skip PM HTTPS compensation: parameter sync event missing device_id",
			zap.String("event_id", evt.ID),
			zap.String("run_id", payload.RunID.String()))
		return nil
	}
	if payload.Status != string(paramsync.RunStatusSucceeded) {
		s.logger.Info("skip PM HTTPS compensation: parameter sync did not succeed",
			zap.String("event_id", evt.ID),
			zap.String("run_id", payload.RunID.String()),
			zap.String("device_id", payload.DeviceID.String()),
			zap.String("status", payload.Status))
		return nil
	}
	if payload.SyncScope != string(paramsync.SyncScopeFull) {
		s.logger.Info("skip PM HTTPS compensation: parameter sync scope is not full",
			zap.String("event_id", evt.ID),
			zap.String("run_id", payload.RunID.String()),
			zap.String("device_id", payload.DeviceID.String()),
			zap.String("sync_scope", payload.SyncScope))
		return nil
	}
	return s.compensatePMUploadURLToHTTPS(ctx, payload.DeviceID, payload.RunID)
}

func (s *OnlineSubscriber) compensatePMUploadURLToHTTPS(
	ctx context.Context,
	deviceID uuid.UUID,
	runID uuid.UUID,
) error {
	if s.deviceLookup == nil || s.parameterRead == nil || s.resolveAddress == nil {
		return nil
	}
	dev, err := s.deviceLookup.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("lookup device for PM HTTPS compensation: %w", err)
	}
	if dev == nil || strings.TrimSpace(dev.SerialNumber) == "" {
		s.logger.Debug("skip PM HTTPS compensation: device not found or missing serial number",
			zap.String("device_id", deviceID.String()))
		return nil
	}

	decision, err := s.resolveAddress.Resolve(ctx, deviceID, transfercfg.TransferDirectionUpload)
	if err != nil {
		return fmt.Errorf("resolve PM HTTPS compensation upload address: %w", err)
	}
	if decision.Protocol != transfercfg.TransferProtocolHTTPS ||
		decision.Capability != transfercfg.HTTPSCapabilityEnabled {
		s.logger.Info("skip PM HTTPS compensation: transfer policy is not eligible",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("device_id", deviceID.String()),
			zap.String("protocol", string(decision.Protocol)),
			zap.String("capability", string(decision.Capability)))
		return nil
	}

	current, err := s.parameterRead.GetByPath(ctx, deviceID, pmUploadURLParameterPath)
	if err != nil {
		return fmt.Errorf("read current PM upload URL: %w", err)
	}
	if current == nil || !isHTTPURL(current.ParameterValue) {
		s.logger.Info("skip PM HTTPS compensation: current PM upload URL is not HTTP",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("device_id", deviceID.String()),
			zap.String("current_protocol", urlProtocol(currentParameterValue(current))))
		return nil
	}

	desiredURL, err := uploadURLWithBase(expandEnv(s.urlTemplate), decision.BaseURL)
	if err != nil {
		return fmt.Errorf("build PM HTTPS compensation URL: %w", err)
	}
	if !isHTTPSURL(desiredURL) {
		return fmt.Errorf("resolved PM HTTPS compensation URL is not HTTPS")
	}

	paramsJSON, err := json.Marshal(buildURLOnlySPVParams(desiredURL))
	if err != nil {
		return fmt.Errorf("marshal PM HTTPS compensation params: %w", err)
	}
	commandKey := pmHTTPSCompensationCommandKeyForRun(deviceID, runID)

	var leaseToken string
	if s.admissionGate != nil {
		var acquired bool
		leaseToken, acquired, err = s.admissionGate.Acquire(ctx, dev.SerialNumber)
		if err != nil {
			return fmt.Errorf("acquire PM HTTPS compensation admission: %w", err)
		}
		if !acquired {
			s.logger.Info("skip PM HTTPS compensation: another worker is handling device",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("device_id", deviceID.String()),
				zap.String("old_protocol", "http"),
				zap.String("new_protocol", "https"))
			return nil
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer cancel()
			if releaseErr := s.admissionGate.Release(releaseCtx, dev.SerialNumber, leaseToken); releaseErr != nil {
				s.logger.Warn("release PM HTTPS compensation admission failed",
					zap.String("device_sn", dev.SerialNumber), zap.Error(releaseErr))
			}
		}()
	}

	open, err := s.equivalentOpenSetParameterValuesTask(
		ctx,
		dev.SerialNumber,
		desiredURL,
	)
	if err != nil {
		return fmt.Errorf("query open PM HTTPS compensation task: %w", err)
	}
	if open != nil {
		s.logger.Info("skip PM HTTPS compensation: equivalent open task exists",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("device_id", deviceID.String()),
			zap.String("task_id", open.ID),
			zap.String("old_protocol", "http"),
			zap.String("new_protocol", "https"))
		return nil
	}

	completed, err := s.taskSvc.LatestCompletedTaskByDeviceAndCommandKey(ctx, dev.SerialNumber, commandKey)
	if err != nil {
		return fmt.Errorf("query completed PM HTTPS compensation task: %w", err)
	}
	if completed != nil && spvParamsSetPMUploadURL(completed.Params, desiredURL) {
		s.logger.Info("skip PM HTTPS compensation: equivalent completed task exists for this parameter sync run",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("device_id", deviceID.String()),
			zap.String("run_id", runID.String()),
			zap.String("task_id", completed.ID),
			zap.String("old_protocol", "http"),
			zap.String("new_protocol", "https"))
		return nil
	}

	if s.admissionGate != nil {
		renewed, renewErr := s.admissionGate.Renew(ctx, dev.SerialNumber, leaseToken)
		if renewErr != nil {
			return fmt.Errorf("renew PM HTTPS compensation admission before create: %w", renewErr)
		}
		if !renewed {
			return fmt.Errorf("PM HTTPS compensation admission lost before create for %s", dev.SerialNumber)
		}
	}

	maxRetries := task.RetryBudgetCoveringExpiry(3600, automatedPMTaskRetryIntervalSeconds)
	_, err = s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:             dev.SerialNumber,
		Method:               string(soap.MethodSetParameterValues),
		Params:               paramsJSON,
		Priority:             20,
		Source:               task.TaskSourceSystem,
		CreatorID:            "",
		Description:          pmHTTPSCompensationDescription,
		CommandKey:           commandKey,
		ExpiresIn:            3600,
		MaxRetries:           &maxRetries,
		RetryIntervalSeconds: automatedPMTaskRetryIntervalSeconds,
	})
	if err != nil {
		return fmt.Errorf("enqueue PM HTTPS compensation SPV task: %w", err)
	}
	s.logger.Info("PM HTTPS compensation SPV task enqueued",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("device_id", deviceID.String()),
		zap.String("old_protocol", "http"),
		zap.String("new_protocol", "https"))
	return nil
}

func (s *OnlineSubscriber) equivalentOpenSetParameterValuesTask(
	ctx context.Context,
	deviceSN string,
	desiredURL string,
) (*task.Task, error) {
	openTasks, err := s.taskSvc.ListOpenTasksByDeviceAndMethods(ctx, deviceSN, []string{string(soap.MethodSetParameterValues)})
	if err != nil {
		return nil, err
	}
	for _, open := range openTasks {
		if open != nil && spvParamsSetPMUploadURL(open.Params, desiredURL) {
			return open, nil
		}
	}
	return nil, nil
}

func pmHTTPSCompensationCommandKeyForRun(deviceID, runID uuid.UUID) string {
	return pmHTTPSCompensationCommandKey + ":" + deviceID.String() + ":" + runID.String()
}

// enqueuePMSetup 解析上传 URL → host 守卫 → 入队 1 个 SPV task 带 3 个 PM 参数。
func (s *OnlineSubscriber) enqueuePMSetup(
	ctx context.Context,
	serialNumber, deviceID string,
	newRegistration bool,
) error {
	if serialNumber == "" {
		s.logger.Warn("PM upload setup skipped: missing serial_number")
		return nil
	}

	url, err := s.resolveUploadURL(ctx, deviceID)
	if err != nil {
		s.logger.Warn("resolve PM upload URL failed",
			zap.String("device_sn", serialNumber),
			zap.String("device_id", deviceID),
			zap.Error(err))
		return fmt.Errorf("resolve PM upload URL: %w", err)
	}
	// host 非空守卫：渲染后若缺 scheme 或 host（如 OMC_PUBLIC_HOST 漏配渲染成
	// "http://:7557/..."，u.Hostname() 返空；或 "${...}" 残留），跳过 SPV 不污染设备。
	if u, perr := neturl.Parse(url); perr != nil || u.Scheme == "" || u.Hostname() == "" {
		s.logger.Warn("rendered PM upload URL missing scheme or host; skip SPV to avoid pushing broken URL",
			zap.String("device_sn", serialNumber),
			zap.String("url", url))
		return nil
	}

	params := buildSPVParams(s.enableValue, url, s.intervalSec)
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		s.logger.Warn("marshal SPV params failed",
			zap.String("device_sn", serialNumber), zap.Error(err))
		return nil
	}
	commandKey := pmSetupCommandKey
	if deviceID != "" {
		commandKey += ":" + deviceID
	}

	var leaseToken string
	var stopRenewal context.CancelFunc
	if s.admissionGate != nil {
		var acquired bool
		var gateErr error
		leaseToken, acquired, gateErr = s.admissionGate.Acquire(ctx, serialNumber)
		if gateErr != nil {
			s.logger.Warn("acquire PM upload setup admission failed",
				zap.String("device_sn", serialNumber), zap.Error(gateErr))
			return fmt.Errorf("acquire PM upload setup admission: %w", gateErr)
		}
		if !acquired {
			s.logger.Debug("skip coalesced PM upload setup event",
				zap.String("device_sn", serialNumber))
			return fmt.Errorf("PM upload setup admission busy for %s", serialNumber)
		}
		var renewalCtx context.Context
		renewalCtx, stopRenewal = context.WithCancel(context.WithoutCancel(ctx))
		go s.renewAdmission(renewalCtx, serialNumber, leaseToken)
	}
	if stopRenewal != nil {
		defer stopRenewal()
	}
	releaseAdmission := func() {
		if s.admissionGate == nil {
			return
		}
		releaseCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			3*time.Second,
		)
		defer cancel()
		if releaseErr := s.admissionGate.Release(
			releaseCtx, serialNumber, leaseToken,
		); releaseErr != nil {
			s.logger.Warn("release PM upload setup admission failed",
				zap.String("device_sn", serialNumber), zap.Error(releaseErr))
		}
	}

	open, err := s.taskSvc.LatestOpenTaskByDeviceAndMethod(
		ctx,
		serialNumber,
		string(soap.MethodSetParameterValues),
		"Auto-setup PM file upload on device onboard/online",
	)
	if err != nil {
		releaseAdmission()
		s.logger.Warn("query open PM upload setup task failed",
			zap.String("device_sn", serialNumber), zap.Error(err))
		return fmt.Errorf("query open PM upload setup task: %w", err)
	}
	openBelongsToCurrentDevice := open != nil &&
		(open.CommandKey == commandKey ||
			(!newRegistration && open.CommandKey == pmSetupCommandKey))
	if openBelongsToCurrentDevice &&
		jsonSemanticallyEqual(open.Params, paramsJSON) {
		s.logger.Debug("skip duplicate open PM upload setup task",
			zap.String("device_sn", serialNumber),
			zap.String("task_id", open.ID))
		return nil
	}

	completed, lookupErr := s.taskSvc.LatestCompletedTaskByDeviceAndCommandKey(
		ctx, serialNumber, commandKey,
	)
	if lookupErr != nil {
		releaseAdmission()
		s.logger.Warn("query completed PM upload setup task failed",
			zap.String("device_sn", serialNumber), zap.Error(lookupErr))
		return fmt.Errorf("query completed PM upload setup task: %w", lookupErr)
	}
	// Before device-scoped command keys were introduced, reconnect tasks used
	// the shared legacy key. Existing devices may trust that durable state;
	// a genuinely new registration must not, because the serial number may
	// have been reused for a different device identity.
	if completed == nil && !newRegistration && commandKey != pmSetupCommandKey {
		completed, lookupErr = s.taskSvc.LatestCompletedTaskByDeviceAndCommandKey(
			ctx, serialNumber, pmSetupCommandKey,
		)
		if lookupErr != nil {
			releaseAdmission()
			s.logger.Warn("query completed PM upload setup task failed",
				zap.String("device_sn", serialNumber), zap.Error(lookupErr))
			return fmt.Errorf("query legacy completed PM upload setup task: %w", lookupErr)
		}
	}
	if completed != nil && jsonSemanticallyEqual(completed.Params, paramsJSON) {
		s.logger.Debug("skip already applied PM upload setup",
			zap.String("device_sn", serialNumber),
			zap.String("task_id", completed.ID))
		return nil
	}

	if s.admissionGate != nil {
		renewed, renewErr := s.admissionGate.Renew(ctx, serialNumber, leaseToken)
		if renewErr != nil || !renewed {
			releaseAdmission()
			if renewErr != nil {
				return fmt.Errorf("renew PM upload setup admission before create: %w", renewErr)
			}
			return fmt.Errorf("PM upload setup admission lost before create for %s", serialNumber)
		}
	}

	maxRetries := task.RetryBudgetCoveringExpiry(3600, automatedPMTaskRetryIntervalSeconds)
	req := &task.CreateTaskRequest{
		DeviceSN:             serialNumber,
		Method:               string(soap.MethodSetParameterValues),
		Params:               paramsJSON,
		Priority:             20, // 低于业务关键命令（默认 10），但高于纯监控类
		Source:               task.TaskSourceSystem,
		CreatorID:            "", // 系统任务，不进消息中心
		Description:          "Auto-setup PM file upload on device onboard/online",
		CommandKey:           commandKey,
		ExpiresIn:            3600, // 1 小时不下发则视为过期（避免设备长时间离线后积压）
		MaxRetries:           &maxRetries,
		RetryIntervalSeconds: automatedPMTaskRetryIntervalSeconds,
	}

	tsk, err := s.taskSvc.CreateTask(ctx, req)
	if err != nil {
		releaseAdmission()
		// 返回 error 交给 EventBus 受控重投，不能 ACK 后等待下一次上下线。
		s.logger.Warn("enqueue PM upload SPV task failed",
			zap.String("device_sn", serialNumber),
			zap.String("device_id", deviceID),
			zap.Error(err))
		return fmt.Errorf("enqueue PM upload SPV task: %w", err)
	}

	s.logger.Info("PM upload SPV task enqueued",
		zap.String("device_sn", serialNumber),
		zap.String("task_id", tsk.ID),
		zap.String("url", url),
		zap.String("enable", s.enableValue),
		zap.Int("interval", s.intervalSec))
	return nil
}

func (s *OnlineSubscriber) renewAdmission(
	ctx context.Context,
	serialNumber, leaseToken string,
) {
	interval := pmSetupAdmissionTTL / 3
	if provider, ok := s.admissionGate.(admissionRenewalIntervalProvider); ok {
		if configured := provider.RenewalInterval(); configured > 0 {
			interval = configured
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			renewed, err := s.admissionGate.Renew(
				renewCtx, serialNumber, leaseToken,
			)
			cancel()
			if err != nil || !renewed {
				s.logger.Warn("renew PM upload setup admission failed",
					zap.String("device_sn", serialNumber),
					zap.Bool("renewed", renewed),
					zap.Error(err))
				return
			}
		}
	}
}

func jsonSemanticallyEqual(left, right json.RawMessage) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func spvParamsSetPMUploadURL(paramsJSON json.RawMessage, desiredURL string) bool {
	var params spvParams
	if json.Unmarshal(paramsJSON, &params) != nil {
		return false
	}
	for _, value := range params.Values {
		if value.Name == pmUploadURLParameterPath && value.Value == desiredURL {
			return true
		}
	}
	return false
}

// resolveUploadURL 计算 PM 上传 URL：
//   - 优先走统一 transfercfg.AddressResolver，按设备 HTTPS 能力和策略选择上传基址；
//   - 无统一 resolver 时保留旧 resolveBaseURL 兼容路径；
//   - 最终 URL 用所选 base 的 scheme/host，path+query 仍取自 urlTemplate
//     （保留 fileType=PM&filename= 等 PM 端约定）。
func (s *OnlineSubscriber) resolveUploadURL(ctx context.Context, deviceID string) (string, error) {
	rendered := expandEnv(s.urlTemplate)
	if s.resolveAddress != nil {
		parsedDeviceID := uuid.Nil
		if trimmed := strings.TrimSpace(deviceID); trimmed != "" {
			parsed, err := uuid.Parse(trimmed)
			if err != nil {
				s.logger.Warn("PM upload address resolver received invalid device_id; treating capability as unknown",
					zap.String("device_id", deviceID),
					zap.Error(err))
			} else {
				parsedDeviceID = parsed
			}
		}
		decision, err := s.resolveAddress.Resolve(ctx, parsedDeviceID, transfercfg.TransferDirectionUpload)
		if err != nil {
			return "", fmt.Errorf("resolve transfer upload address: %w", err)
		}
		return uploadURLWithBase(rendered, decision.BaseURL)
	}
	if s.resolveBaseURL == nil {
		return rendered, nil
	}
	base := strings.TrimSpace(s.resolveBaseURL(ctx))
	if base == "" {
		return rendered, nil
	}
	resolved, err := uploadURLWithBase(rendered, base)
	if err != nil {
		return rendered, nil // legacy base resolver 非法时仍回退模板
	}
	return resolved, nil
}

func uploadURLWithBase(rendered, base string) (string, error) {
	return transfercfg.BuildTemplateURL(base, pmUploadRelativeReference(rendered))
}

func pmUploadRelativeReference(rendered string) string {
	tu, err := neturl.Parse(rendered)
	if err != nil || tu.Path == "" {
		return defaultPMUploadPathQuery
	}
	relative := tu.EscapedPath()
	if tu.RawQuery != "" || tu.ForceQuery {
		relative += "?" + tu.RawQuery
	}
	return relative
}

// spvParam 是 SetParameterValuesHandler.BuildRequest 解析期望的 params.values[] 项。
// 类型字符串与 soap.ParameterSetData.Type 对齐。
type spvParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type spvParams struct {
	Values []spvParam `json:"values"`
}

// buildSPVParams 构造 3 个 PM 上传参数（KPI上报参数整理.md §1-§3）。
//
// 实例号 {i} 固定 1（用户确认 2026-05-23：永远 1）。
// 平台路径差异由 ACS 端 Translator 处理（standardPath → privatePath）—
// BLQ 平台 standardPath = privatePath，直发即可。
//
// 类型选择：
//   - Enable: xsd:boolean（TR-069 标准 boolean，接受 "1"/"true"）
//   - URL: xsd:string
//   - PeriodicUploadInterval: xsd:unsignedInt（秒为单位的整数）
func buildSPVParams(enableValue string, url string, intervalSec int) spvParams {
	return spvParams{
		Values: []spvParam{
			{Name: "Device.FAP.PerfMgmt.Config.1.Enable", Value: enableValue, Type: "xsd:boolean"},
			{Name: pmUploadURLParameterPath, Value: url, Type: "xsd:string"},
			{
				Name:  "Device.FAP.PerfMgmt.Config.1.PeriodicUploadInterval",
				Value: strconv.Itoa(intervalSec),
				Type:  "xsd:unsignedInt",
			},
		},
	}
}

func buildURLOnlySPVParams(url string) spvParams {
	return spvParams{
		Values: []spvParam{
			{Name: pmUploadURLParameterPath, Value: url, Type: "xsd:string"},
		},
	}
}

func isHTTPURL(value string) bool {
	u, err := neturl.Parse(strings.TrimSpace(value))
	return err == nil && strings.EqualFold(u.Scheme, "http") && u.Hostname() != ""
}

func isHTTPSURL(value string) bool {
	u, err := neturl.Parse(strings.TrimSpace(value))
	return err == nil && strings.EqualFold(u.Scheme, "https") && u.Hostname() != ""
}

func currentParameterValue(param *model.DeviceParameter) string {
	if param == nil {
		return ""
	}
	return param.ParameterValue
}

func urlProtocol(value string) string {
	u, err := neturl.Parse(strings.TrimSpace(value))
	if err != nil || u.Scheme == "" {
		return "unknown"
	}
	return strings.ToLower(u.Scheme)
}

// expandEnv 展开 ${VAR} 与 ${VAR:-default} 占位符。
//
// Go 标准库 os.ExpandEnv 不支持 ":-" shell 默认值语法，自己实现一个最小版本：
//   - "${VAR}" → os.Getenv("VAR")，env 未设时返空字符串
//   - "${VAR:-default}" → os.Getenv("VAR")，env 未设时返 "default"
//   - 其它字符原样保留
//
// 不支持 "${VAR:+alt}" / "${VAR:?error}" 等其它 shell 语法（用不上）。
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		// "VAR:-default" 形式
		if idx := strings.Index(key, ":-"); idx >= 0 {
			name := key[:idx]
			fallback := key[idx+2:]
			if v := os.Getenv(name); v != "" {
				return v
			}
			return fallback
		}
		return os.Getenv(key)
	})
}

// ValidateUploadURLTemplate 在 worker 启动期对 PM 上传 URL 模板做一次性 host 校验：
// 用与 handle() 完全一致的 expandEnv 渲染（复用同一渲染逻辑，避免漂移），
// 再用 net/url 解析校验 scheme/host 非空。返回渲染后的 url 与错误。
//
// 校验失败（如 OMC_PUBLIC_HOST 漏配渲染成 "http://:7557/..."）由调用方 logger.Error
// 醒目告警但不阻断启动——把「运维漏配」从「设备上线时静默跳过下发」前移到「部署即可见」，
// 同时不让单个配置项阻断整个 worker（worker 还跑 PM 解析/聚合等关键流程）。
func ValidateUploadURLTemplate(urlTemplate string) (rendered string, err error) {
	rendered = expandEnv(urlTemplate)
	u, perr := neturl.Parse(rendered)
	if perr != nil {
		return rendered, fmt.Errorf("parse rendered PM upload URL %q: %w", rendered, perr)
	}
	if u.Scheme == "" || u.Hostname() == "" {
		return rendered, fmt.Errorf("rendered PM upload URL %q missing scheme or host (check OMC_PUBLIC_HOST)", rendered)
	}
	return rendered, nil
}
