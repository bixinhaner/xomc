package notification

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// NewCreateFailureNotifier 返回一个 task.CreateFailureNotifier closure，注入到
// TaskService.SetCreateFailureNotifier（T-0157 C6）。
//
// 作用：当 TaskService.CreateTask 入队失败（repo.Create / queue.Push 出错）时，
// 主动写一条 status='failed' 的消息到当前用户的消息中心，避免出现"用户点了
// 保存按钮却没有任何反馈"。
//
// 复用 §4.5 文案规范：标题 / 详情走与终态消息一致的渲染函数，用户感知统一。
// dedup_key 用 task.ID —— 与订阅器路径互斥（入队失败时永远不会触发 task.created
// publish，所以不会与"进行中"消息冲突）。
//
// 系统任务（CreatorID 为空）在 TaskService.notifyCreateFailure 已过滤；
// 本 closure 防御性再检一次，避免单元测试直接调用时漏过。
func NewCreateFailureNotifier(svc *Service, log *zap.Logger) task.CreateFailureNotifier {
	if log == nil {
		log = zap.NewNop()
	}
	logger := log.Named("notification-create-failure")
	return func(ctx context.Context, t *task.Task, errMsg string) {
		if t == nil || t.CreatorID == "" {
			return
		}
		title := fmt.Sprintf("%s · 设备 %s · 入队失败", translateMethod(t.Method), t.DeviceSN)
		content := errMsg
		dedup := t.ID

		notif := &Notification{
			UserID:   t.CreatorID,
			Type:     NotifTypeTaskComplete,
			Status:   StatusFailed,
			Priority: PriorityHigh,
			Title:    title,
			Content:  content,
			Link:     fmt.Sprintf("/device/detail/%s?tab=quickSettings", t.DeviceSN),
			Sender:   "system",
			DedupKey: &dedup,
		}
		if _, err := svc.UpsertByDedup(ctx, notif); err != nil {
			logger.Warn("upsert failed-enqueue notification",
				zap.String("task_id", t.ID),
				zap.String("user_id", t.CreatorID),
				zap.Error(err))
		}
	}
}
