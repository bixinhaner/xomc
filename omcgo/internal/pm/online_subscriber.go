package pm

import (
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
)

// 编译时引用 device 包确保 DeviceOnlineEvent 类型存在（payload decode 用）。
var _ = device.DeviceOnlineEvent{}

// OnlineSubscriber 订阅 device.online 事件，在设备由 offline→active 时
// 自动下发 PM 文件上传配置（KPI 上报参数整理.md 三参数）。
//
// 触发时机（由 device.DeviceService 决定）：
//   - 非批量路径：UpdateFromInform 检测到 oldStatus=offline && device.Status=active
//   - 批量路径：BatchInformProcessor.doFlush 同条件
//   - firmware 变化时不发 device.online（发 firmware.changed，由 provision 处理 Path B 同步）—
//     本订阅器不订阅 firmware.changed，避免与 provision 的 GPN/GPV 同步抢资源
//
// 节流：device.online 事件在 provision 模块已有 60s Redis token bucket 防抖，
// 本订阅器不重复实现（多 worker 实例共享节流由 QueueSubscribe 配合 token bucket 完成）。
//
// 失败处理：失败仅 log warn，不进重试 / 不写消息中心（PM 配置下发是后台系统行为，
// 系统级任务无 user_id，与 task_subscriber.go §upsertFromTask CreatorID 跳过规则一致）。
// 下次设备 offline→online 自然重发。
type OnlineSubscriber struct {
	taskSvc     TaskCreator
	urlTemplate string
	enableValue string
	intervalSec int
	// resolveBaseURL 可选：运行时解析 PM 上传基址（通常读 sys_configs
	// acs_transfer.uploadBaseURL，与「系统配置→ACS 传输」页同源）。非空合法时
	// 覆盖 urlTemplate 的 scheme/host（path+query 仍取自 urlTemplate）；为空回退 urlTemplate。
	resolveBaseURL func(ctx context.Context) string
	logger         *zap.Logger
}

// defaultPMUploadPathQuery 是 urlTemplate 不可解析时回退用的 PM 上传 path+query，
// 与 config.*.yaml 的 pm.upload_url_template 末段保持一致。
const defaultPMUploadPathQuery = "/smallcell/FileUploadService?fileType=PM&filename="

// TaskCreator 是 OnlineSubscriber 入队 SPV task 的最小依赖。
// 真实实现是 *task.TaskService。
type TaskCreator interface {
	CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
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
	s.logger.Info("pm online subscriber registered",
		zap.String("subjects", event.SubjectDeviceOnline+","+event.SubjectDeviceRegistered),
		zap.String("url_template", s.urlTemplate),
		zap.Int("interval_seconds", s.intervalSec))
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
	return s.enqueuePMSetup(ctx, payload.SerialNumber, payload.DeviceID.String())
}

// handleRegistered 处理 device.registered（首次 onboard）。payload 是 map 结构
// （device_service.go PublishDeviceRegistered），取 serial_number / device_id。
func (s *OnlineSubscriber) handleRegistered(ctx context.Context, evt event.Event) error {
	var payload struct {
		SerialNumber string `json:"serial_number"`
		DeviceID     string `json:"device_id"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.registered payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		return nil
	}
	return s.enqueuePMSetup(ctx, payload.SerialNumber, payload.DeviceID)
}

// enqueuePMSetup 解析上传 URL → host 守卫 → 入队 1 个 SPV task 带 3 个 PM 参数。
func (s *OnlineSubscriber) enqueuePMSetup(ctx context.Context, serialNumber, deviceID string) error {
	if serialNumber == "" {
		s.logger.Warn("PM upload setup skipped: missing serial_number")
		return nil
	}

	url := s.resolveUploadURL(ctx)
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

	req := &task.CreateTaskRequest{
		DeviceSN:    serialNumber,
		Method:      "SetParameterValues",
		Params:      paramsJSON,
		Priority:    20, // 低于业务关键命令（默认 10），但高于纯监控类
		Source:      task.TaskSourceSystem,
		CreatorID:   "", // 系统任务，不进消息中心
		Description: "Auto-setup PM file upload on device onboard/online",
		CommandKey:  "pm_upload_setup_on_online",
		ExpiresIn:   3600, // 1 小时不下发则视为过期（避免设备长时间离线后积压）
	}

	tsk, err := s.taskSvc.CreateTask(ctx, req)
	if err != nil {
		// 失败 log only，下次设备 onboard/online 时自动重发
		s.logger.Warn("enqueue PM upload SPV task failed",
			zap.String("device_sn", serialNumber),
			zap.String("device_id", deviceID),
			zap.Error(err))
		return nil
	}

	s.logger.Info("PM upload SPV task enqueued",
		zap.String("device_sn", serialNumber),
		zap.String("task_id", tsk.ID),
		zap.String("url", url),
		zap.String("enable", s.enableValue),
		zap.Int("interval", s.intervalSec))
	return nil
}

// resolveUploadURL 计算 PM 上传 URL：
//   - 无 resolveBaseURL（未注入）或解析为空/非法 → 回退 expandEnv(urlTemplate)；
//   - 解析到合法 base（如 sys_configs.acs_transfer.uploadBaseURL=http://172.19.1.173:8080）
//     → 用 base 的 scheme/host，path+query 仍取自 urlTemplate（保留 fileType=PM&filename= 等），
//     使上传地址与「系统配置→ACS 传输」页同源、改 IP 即时生效、无需重建镜像。
func (s *OnlineSubscriber) resolveUploadURL(ctx context.Context) string {
	rendered := expandEnv(s.urlTemplate)
	if s.resolveBaseURL == nil {
		return rendered
	}
	base := strings.TrimSpace(s.resolveBaseURL(ctx))
	if base == "" {
		return rendered
	}
	bu, err := neturl.Parse(base)
	if err != nil || bu.Scheme == "" || bu.Host == "" {
		return rendered // base 非法，回退模板
	}
	tu, terr := neturl.Parse(rendered)
	if terr != nil || tu.Path == "" {
		// 模板不可解析/无 path：用 base + 默认 PM 上传 path+query
		return strings.TrimRight(base, "/") + defaultPMUploadPathQuery
	}
	tu.Scheme = bu.Scheme
	tu.Host = bu.Host
	return tu.String()
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
			{Name: "Device.FAP.PerfMgmt.Config.1.URL", Value: url, Type: "xsd:string"},
			{
				Name:  "Device.FAP.PerfMgmt.Config.1.PeriodicUploadInterval",
				Value: strconv.Itoa(intervalSec),
				Type:  "xsd:unsignedInt",
			},
		},
	}
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
