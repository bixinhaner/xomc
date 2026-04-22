package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

// RebootCloser 在设备上报 "M Reboot" Inform 时，兜底收敛尚未完成的 Reboot/FactoryReset 任务。
//
// 正常路径下，CPE 收到 Reboot RPC 后会立即返回 RebootResponse，ACS Handler 在
// 1219 行左右匹配 CWMP-ID 调用 MarkTaskCompleted。但存在若干边界路径：
//   - CPE 实现不规范：收到 Reboot 后直接重启，不返回 SOAP ACK；
//   - 网络抖动：RebootResponse 丢失；
//   - ACS 重启：响应已收到但同步落库之前重启。
// 这些情况下，device_tasks 里的 Reboot 任务会停留在 sent/pending 状态。等设备真
// 正重启回来并携带 "M Reboot"（ACS 主动下发 Reboot 后的事件码）时，由本关闭器
// 做兜底结算。
//
// 正常路径会被跳过（任务已 completed，ListOpen 查不到），不会二次操作。
const (
	rebootCloserQueue = "task-reboot-closer"

	// MethodReboot / MethodFactoryReset 是两种会导致设备重启的 TR-069 RPC。
	MethodReboot       = "Reboot"
	MethodFactoryReset = "FactoryReset"

	// EventMReboot 是 TR-069 Amendment 6 规定的 CPE 重启后 Inform 事件码前缀。
	// 保留在任务包以免循环依赖 pkg/tr069。
	eventMReboot = "M Reboot"
)

// RebootClosableTasksRepo 定义关闭器需要的最小 PG 仓库能力子集。
// 保留小接口便于测试。
type RebootClosableTasksRepo interface {
	ListOpenByDeviceAndMethods(ctx context.Context, deviceSN string, methods []string) ([]*Task, error)
}

// TaskCompleter 与 TaskService.MarkTaskCompleted 对齐，便于单元测试替换。
type TaskCompleter interface {
	MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error
}

// RebootCompletePayload 与 device.InformEventPayload 的字段取子集，仅取关闭器需要的部分。
type RebootCompletePayload struct {
	DeviceId struct {
		SerialNumber string `json:"SerialNumber"`
	} `json:"device_id"`
	Events []string `json:"events"`
}

// RebootCloser 订阅 device.inform.reboot_complete，在事件包含 "M Reboot" 时兜底收敛。
type RebootCloser struct {
	repo      RebootClosableTasksRepo
	completer TaskCompleter
	logger    *zap.Logger
}

// NewRebootCloser 构造 RebootCloser。
func NewRebootCloser(repo RebootClosableTasksRepo, completer TaskCompleter, logger *zap.Logger) *RebootCloser {
	return &RebootCloser{
		repo:      repo,
		completer: completer,
		logger:    logger,
	}
}

// Subscribe 注册到事件总线。
func (c *RebootCloser) Subscribe(bus event.EventBus) error {
	if _, err := bus.QueueSubscribe(event.SubjectDeviceRebootComplete, rebootCloserQueue, c.handle); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceRebootComplete, err)
	}
	c.logger.Info("reboot closer subscribed", zap.String("subject", event.SubjectDeviceRebootComplete))
	return nil
}

func (c *RebootCloser) handle(ctx context.Context, evt event.Event) error {
	var payload RebootCompletePayload
	if err := evt.DecodePayload(&payload); err != nil {
		c.logger.Warn("decode reboot_complete payload", zap.Error(err))
		return nil
	}
	sn := payload.DeviceId.SerialNumber
	if sn == "" {
		return nil
	}

	// 仅在 ACS 主动下发 Reboot 的回包场景处理（"M Reboot"）。
	// 纯 "1 BOOT" 属于自主重启，不对应 Reboot/FactoryReset 任务。
	if !containsEvent(payload.Events, eventMReboot) {
		return nil
	}

	open, err := c.repo.ListOpenByDeviceAndMethods(ctx, sn, []string{MethodReboot, MethodFactoryReset})
	if err != nil {
		c.logger.Warn("list open reboot tasks failed",
			zap.String("serial_number", sn), zap.Error(err))
		return nil // best-effort
	}
	if len(open) == 0 {
		return nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"closed_by":  "reboot_complete_inform",
		"closed_at":  time.Now(),
		"events":     payload.Events,
		"note":       "settled via M Reboot Inform (no explicit RebootResponse received)",
	})

	for _, t := range open {
		if err := c.completer.MarkTaskCompleted(ctx, t.ID, result); err != nil {
			c.logger.Warn("mark reboot task completed failed",
				zap.String("task_id", t.ID),
				zap.String("serial_number", sn),
				zap.Error(err))
			continue
		}
		c.logger.Info("reboot task closed by M Reboot inform",
			zap.String("task_id", t.ID),
			zap.String("method", t.Method),
			zap.String("serial_number", sn))
	}
	return nil
}

func containsEvent(events []string, target string) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}
