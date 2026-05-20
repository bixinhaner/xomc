package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
)

// TaskSubscriber 订阅 task.created / task.completed / task.failed 事件并把任务
// 转换为消息中心条目（T-0157 C5）。
//
// 设计要点：
//   - 同一 task 多次状态变更（pending → completed/failed/expired）按 dedup_key=task_id
//     upsert 到同一行 notification，状态字段升级，user_id 保持隔离
//   - task.CreatorID 空时跳过（系统任务不打扰用户）
//   - task.failed 主题覆盖 failed / expired / cancelled 三种 task 终态，按 task.Status 区分
//   - 文案渲染走 §4.5 简化版（V1 仅显示新值；旧值由前端 Tag 显示 — 详见 §11 L-03）
type TaskSubscriber struct {
	service *Service
	logger  *zap.Logger
}

// NewTaskSubscriber 创建订阅器。
func NewTaskSubscriber(svc *Service, log *zap.Logger) *TaskSubscriber {
	if log == nil {
		log = zap.NewNop()
	}
	return &TaskSubscriber{
		service: svc,
		logger:  log.Named("notification-task-subscriber"),
	}
}

// Subscribe 注册到 EventBus 的三个 task 主题。返回首个失败的 Subscription error。
// 不接受 ctx 参数 —— Subscription 生命周期与 EventBus 一致，进程退出时 bus.Close() 统一回收。
func (s *TaskSubscriber) Subscribe(bus event.EventBus) error {
	for _, sub := range []struct {
		subject string
		handler event.EventHandler
	}{
		{event.SubjectTaskCreated, s.handleCreated},
		{event.SubjectTaskCompleted, s.handleCompleted},
		{event.SubjectTaskFailed, s.handleFailed},
	} {
		if _, err := bus.Subscribe(sub.subject, sub.handler); err != nil {
			return fmt.Errorf("subscribe %s: %w", sub.subject, err)
		}
		s.logger.Info("subscribed", zap.String("subject", sub.subject))
	}
	return nil
}

func (s *TaskSubscriber) handleCreated(ctx context.Context, evt event.Event) error {
	t := decodeTask(&evt)
	if t == nil {
		return nil
	}
	return s.upsertFromTask(ctx, t, StatusQueued)
}

func (s *TaskSubscriber) handleCompleted(ctx context.Context, evt event.Event) error {
	t := decodeTask(&evt)
	if t == nil {
		return nil
	}
	return s.upsertFromTask(ctx, t, StatusCompleted)
}

// handleFailed 处理 task.failed 主题 —— 该主题被 failed / expired / cancelled 三种 task 终态共享，
// 按 task.Status 字段区分映射到消息中心 status。
func (s *TaskSubscriber) handleFailed(ctx context.Context, evt event.Event) error {
	t := decodeTask(&evt)
	if t == nil {
		return nil
	}
	var st NotificationStatus
	switch t.Status {
	case task.TaskStatusExpired:
		st = StatusExpired
	case task.TaskStatusCancelled:
		st = StatusCancelled
	default:
		st = StatusFailed
	}
	return s.upsertFromTask(ctx, t, st)
}

// upsertFromTask 把 task 渲染为 Notification 并 upsert（按 dedup_key=task.ID）。
// CreatorID 为空时跳过 — 系统任务（PeriodicSyncer / F09 等）不进消息中心。
func (s *TaskSubscriber) upsertFromTask(ctx context.Context, t *task.Task, status NotificationStatus) error {
	if t == nil || t.CreatorID == "" {
		return nil
	}
	title := renderNotifTitle(t, status)
	content := renderNotifContent(t, status)
	priority := PriorityNormal
	if status == StatusFailed || status == StatusExpired {
		priority = PriorityHigh
	}
	dedup := t.ID

	notif := &Notification{
		UserID:   t.CreatorID,
		Type:     NotifTypeTaskComplete,
		Status:   status,
		Priority: priority,
		Title:    title,
		Content:  content,
		Link:     fmt.Sprintf("/device/detail/%s?tab=quickSettings", t.DeviceSN),
		Sender:   "system",
		DedupKey: &dedup,
	}
	if _, err := s.service.UpsertByDedup(ctx, notif); err != nil {
		s.logger.Warn("upsert notification from task",
			zap.String("task_id", t.ID),
			zap.String("device_sn", t.DeviceSN),
			zap.String("status", string(status)),
			zap.Error(err))
		return err
	}
	return nil
}

func decodeTask(evt *event.Event) *task.Task {
	var t task.Task
	if err := evt.DecodePayload(&t); err != nil {
		return nil
	}
	return &t
}

// renderNotifTitle V1 简化版（§4.5 旧值 → 新值的"旧值"需前端配合，留 §11 L-03 V2 增强）。
//
// 示例输出：
//
//	"参数设置 · 设备 SN001 · 进行中"
//	"参数设置 · 设备 SN001 · 已完成 · 3 项"
//	"设备重启 · 设备 SN001 · 失败"
func renderNotifTitle(t *task.Task, status NotificationStatus) string {
	method := translateMethod(t.Method)
	verb := statusVerb(status)
	base := fmt.Sprintf("%s · 设备 %s · %s", method, t.DeviceSN, verb)
	switch t.Method {
	case "SetParameterValues":
		n := len(extractSPVParams(t.Params))
		if n > 0 {
			base = fmt.Sprintf("%s · %d 项", base, n)
		}
	case "DeleteObject":
		if name := extractObjectName(t.Params); name != "" {
			base = fmt.Sprintf("%s · %s", base, shortObjectName(name))
		}
	}
	return base
}

func renderNotifContent(t *task.Task, status NotificationStatus) string {
	const maxLines = 5
	var lines []string

	switch t.Method {
	case "SetParameterValues":
		items := extractSPVParams(t.Params)
		for i, p := range items {
			if i >= maxLines {
				lines = append(lines, fmt.Sprintf("...共 %d 项", len(items)))
				break
			}
			lines = append(lines, fmt.Sprintf("%s = %s", p.Name, p.Value))
		}
	case "DeleteObject":
		if name := extractObjectName(t.Params); name != "" {
			lines = append(lines, "对象路径："+name)
		}
	}

	if (status == StatusFailed || status == StatusExpired) && t.ErrorMessage != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "错误："+t.ErrorMessage)
	}

	return strings.Join(lines, "\n")
}

func translateMethod(method string) string {
	switch method {
	case "SetParameterValues":
		return "参数设置"
	case "GetParameterValues":
		return "参数读取"
	case "Reboot":
		return "设备重启"
	case "FactoryReset":
		return "工厂复位"
	case "Download":
		return "文件下载"
	case "Upload":
		return "文件上传"
	case "AddObject":
		return "新增对象"
	case "DeleteObject":
		return "删除对象"
	}
	return method
}

func statusVerb(s NotificationStatus) string {
	switch s {
	case StatusQueued, StatusSent:
		return "进行中"
	case StatusCompleted:
		return "已完成"
	case StatusFailed:
		return "失败"
	case StatusExpired:
		return "超时"
	case StatusCancelled:
		return "已取消"
	}
	return string(s)
}

type spvParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func extractSPVParams(raw json.RawMessage) []spvParam {
	if len(raw) == 0 {
		return nil
	}
	var wrapper struct {
		Values []spvParam `json:"values"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil
	}
	return wrapper.Values
}

// extractObjectName 解析 AddObject / DeleteObject task.Params 中的 object_name。
// task.Params 形如 `{"object_name": "Device.X.Y.{i}."}`。
func extractObjectName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var wrapper struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return ""
	}
	return wrapper.ObjectName
}

// shortObjectName 取对象路径末尾 2 段，用于通知标题显示。
// 例：`Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.14.` → `LTECell.14`
func shortObjectName(path string) string {
	trimmed := strings.TrimSuffix(path, ".")
	parts := strings.Split(trimmed, ".")
	if len(parts) <= 2 {
		return trimmed
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}
