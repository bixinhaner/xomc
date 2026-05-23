package pm

import (
	"context"
	"encoding/json"
	"fmt"
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
	logger      *zap.Logger
}

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

// Subscribe 把 OnlineSubscriber 挂到 EventBus。使用 QueueSubscribe 让多 worker 实例
// 同时启动时只有一个实例处理同一事件（避免重复入队 SPV task）。
//
// queueName 固定 "pm-online-pm-upload-setup" — 与 provision 的 "provision-online-sync"
// 处于不同 queue group，互不抢消息（两个订阅者都会各自收到事件，是设计意图）。
func (s *OnlineSubscriber) Subscribe(bus event.EventBus) error {
	const queueName = "pm-online-pm-upload-setup"
	_, err := bus.QueueSubscribe(event.SubjectDeviceOnline, queueName, s.handle)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceOnline, err)
	}
	s.logger.Info("pm online subscriber registered",
		zap.String("subject", event.SubjectDeviceOnline),
		zap.String("queue", queueName),
		zap.String("url_template", s.urlTemplate),
		zap.Int("interval_seconds", s.intervalSec))
	return nil
}

// handle 是单事件处理：解析 → 渲染 URL → 入队 1 个 SPV task 带 3 个 params。
func (s *OnlineSubscriber) handle(ctx context.Context, evt event.Event) error {
	var payload device.DeviceOnlineEvent
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.online payload failed",
			zap.String("event_id", evt.ID), zap.Error(err))
		// 返 nil 让 EventBus 不重试（payload 损坏重试无意义）
		return nil
	}

	if payload.SerialNumber == "" {
		s.logger.Warn("device.online payload missing serial_number",
			zap.String("event_id", evt.ID))
		return nil
	}

	url := expandEnv(s.urlTemplate)
	if !strings.Contains(url, "://") {
		s.logger.Warn("rendered URL missing scheme; skip SPV",
			zap.String("device_sn", payload.SerialNumber),
			zap.String("url", url))
		return nil
	}

	params := buildSPVParams(s.enableValue, url, s.intervalSec)
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		s.logger.Warn("marshal SPV params failed",
			zap.String("device_sn", payload.SerialNumber), zap.Error(err))
		return nil
	}

	req := &task.CreateTaskRequest{
		DeviceSN:    payload.SerialNumber,
		Method:      "SetParameterValues",
		Params:      paramsJSON,
		Priority:    20, // 低于业务关键命令（默认 10），但高于纯监控类
		Source:      task.TaskSourceSystem,
		CreatorID:   "", // 系统任务，不进消息中心
		Description: "Auto-setup PM file upload on device online",
		CommandKey:  "pm_upload_setup_on_online",
		ExpiresIn:   3600, // 1 小时不下发则视为过期（避免设备长时间离线后积压）
	}

	tsk, err := s.taskSvc.CreateTask(ctx, req)
	if err != nil {
		// 失败 log only，下次自然 offline→online 时自动重发
		s.logger.Warn("enqueue PM upload SPV task failed",
			zap.String("device_sn", payload.SerialNumber),
			zap.String("device_id", payload.DeviceID.String()),
			zap.Error(err))
		return nil
	}

	s.logger.Info("PM upload SPV task enqueued",
		zap.String("device_sn", payload.SerialNumber),
		zap.String("task_id", tsk.ID),
		zap.String("url", url),
		zap.String("enable", s.enableValue),
		zap.Int("interval", s.intervalSec))
	return nil
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
