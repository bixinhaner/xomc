package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/events"
	"github.com/omcgo/omcgo/internal/task"
)

// Service provides business logic for notification management.
type Service struct {
	repo       Repository
	hub        *events.MessageHub
	logger     *zap.Logger
	staleTask  StaleTaskLookup // T-0157 stale sync; nil=禁用 SyncStaleByUser
}

// NewService creates a new notification Service.
// The hub parameter is optional; if nil, notifications will not be pushed via SSE.
func NewService(repo Repository, hub *events.MessageHub, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		hub:    hub,
		logger: logger.Named("notification"),
	}
}

// List returns a paginated list of notifications for the given user.
func (s *Service) List(ctx context.Context, filter NotificationFilter) (*model.ListResponse[Notification], error) {
	return s.repo.List(ctx, filter)
}

// GetByID returns a single notification by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// CreateNotification creates a new notification and pushes it via SSE if hub is available.
func (s *Service) CreateNotification(ctx context.Context, notif *Notification) (*Notification, error) {
	if err := s.repo.Create(ctx, notif); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	// Push via SSE if hub is available
	if s.hub != nil {
		s.pushSSEEvent(notif)
	}

	return notif, nil
}

// UpsertByDedup 按 dedup_key upsert 一条 notification（T-0157 C3）。
// 用于 task → notification 订阅器：同一 task 多次状态变更 upsert 到同一行。
// 成功后通过 SSE 推送（如已配 hub），与 CreateNotification 行为一致。
func (s *Service) UpsertByDedup(ctx context.Context, notif *Notification) (*Notification, error) {
	if err := s.repo.UpsertByDedup(ctx, notif); err != nil {
		return nil, fmt.Errorf("upsert notification by dedup: %w", err)
	}
	if s.hub != nil {
		s.pushSSEEvent(notif)
	}
	return notif, nil
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID, userID string) error {
	return s.repo.MarkRead(ctx, id, userID)
}

// MarkAllRead marks all notifications for a user as read.
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// GetUnreadCount returns the number of unread notifications for a user.
func (s *Service) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// Delete removes a notification.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

// DeleteAllByUser 删除当前用户的全部消息 (T-0157 C4)。返回删除数。
func (s *Service) DeleteAllByUser(ctx context.Context, userID string) (int64, error) {
	return s.repo.DeleteAllByUser(ctx, userID)
}

// StaleTaskLookup 抽象 task 表查询，便于单测 mock。
// 实现端 = task.PgTaskRepository 的包装。
type StaleTaskLookup interface {
	LookupTaskStatus(ctx context.Context, taskID string) (status string, errMsg string, found bool, err error)
	// LookupTaskStatuses 批量反查多个 task 的状态（#16 消除 SyncStaleByUser 的 N+1）。
	// 返回 map 仅含仍存在的 taskID；缺失的 taskID（已被清理）由调用方按 not-found 处理。
	LookupTaskStatuses(ctx context.Context, taskIDs []string) (map[string]task.TaskStatusInfo, error)
}

// SetStaleTaskLookup 注入 task 状态查询器（T-0157 stale sync）。
// 由 cmd/app/bootstrap.go 在 wire 时调用，注入 task.PgTaskRepository 的适配器。
func (s *Service) SetStaleTaskLookup(l StaleTaskLookup) {
	s.staleTask = l
}

// SyncStaleByUser 拉当前用户所有非终态 (queued/sent) 消息，反查对应 task 实际状态。
// 若 task 已终态且与 notification 状态不一致 → 更新 notification (T-0157 stale sync)。
// 用于 Popover 打开 / 用户主动刷新场景，兜底 subscriber 漏接事件导致的卡死消息。
// 返回更新条数。
func (s *Service) SyncStaleByUser(ctx context.Context, userID string) (int, error) {
	if s.staleTask == nil {
		return 0, nil // 未注入查询器，no-op
	}
	stale, err := s.repo.ListStaleByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("list stale: %w", err)
	}

	// 批量反查 task 状态：先收集所有非空 dedup_key（=task_id），一条 IN 查询取回所有状态，
	// 替代原先逐 notification 调用 LookupTaskStatus 的 N+1（#16）。
	taskIDs := make([]string, 0, len(stale))
	seen := make(map[string]struct{}, len(stale))
	for _, n := range stale {
		if n.DedupKey == nil || *n.DedupKey == "" {
			continue
		}
		if _, dup := seen[*n.DedupKey]; dup {
			continue
		}
		seen[*n.DedupKey] = struct{}{}
		taskIDs = append(taskIDs, *n.DedupKey)
	}
	statuses, err := s.staleTask.LookupTaskStatuses(ctx, taskIDs)
	if err != nil {
		return 0, fmt.Errorf("lookup task statuses for stale sync: %w", err)
	}

	updated := 0
	for _, n := range stale {
		if n.DedupKey == nil || *n.DedupKey == "" {
			continue
		}
		info, found := statuses[*n.DedupKey]
		status, errMsg := info.Status, info.ErrorMsg
		if !found {
			// task 已被 PurgeOldTasks 清掉但消息仍在 → 标 expired（视为运行期超时无人收尾）
			if err := s.repo.UpdateStatusByID(ctx, n.ID, StatusExpired, PriorityHigh,
				n.Title+"（任务记录已过期）", n.Content); err != nil {
				s.logger.Warn("update stale notification (task gone)", zap.Error(err))
				continue
			}
			updated++
			continue
		}
		// task 还在运行（pending/sent），状态没变，跳过
		if status == "pending" || status == "sent" {
			continue
		}
		// task 已终态，映射回 notification 状态 + 升 priority
		var ns NotificationStatus
		var prio NotificationPriority
		switch status {
		case "completed":
			ns = StatusCompleted
			prio = PriorityNormal
		case "failed":
			ns = StatusFailed
			prio = PriorityHigh
		case "expired":
			ns = StatusExpired
			prio = PriorityHigh
		case "cancelled":
			ns = StatusCancelled
			prio = PriorityNormal
		default:
			continue
		}
		// 标题中"进行中"→ 实际状态对应中文（简单替换；UI 视觉以 status 字段为准）
		title := replaceStatusVerb(n.Title, ns)
		content := n.Content
		if (ns == StatusFailed || ns == StatusExpired) && errMsg != "" && content == "" {
			content = "错误：" + errMsg
		}
		if err := s.repo.UpdateStatusByID(ctx, n.ID, ns, prio, title, content); err != nil {
			s.logger.Warn("update stale notification", zap.String("task_id", *n.DedupKey), zap.Error(err))
			continue
		}
		updated++
	}
	return updated, nil
}

// replaceStatusVerb 把标题里的"进行中"替换为终态文案。
// 与 task_subscriber.statusVerb 保持一致。
func replaceStatusVerb(title string, s NotificationStatus) string {
	verb := "已完成"
	switch s {
	case StatusFailed:
		verb = "失败"
	case StatusExpired:
		verb = "超时"
	case StatusCancelled:
		verb = "已取消"
	}
	// 简单字符串替换：原 title 含"· 进行中 ·"
	return replaceOnce(title, "· 进行中 ·", "· "+verb+" ·")
}

func replaceOnce(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// pushSSEEvent publishes a notification event to the user's SSE channel.
func (s *Service) pushSSEEvent(notif *Notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		s.logger.Error("failed to marshal notification for SSE",
			zap.String("notification_id", notif.ID.String()),
			zap.Error(err),
		)
		return
	}

	msg := &events.SSEMessage{
		ID:     notif.ID.String(),
		Event:  "notification",
		Data:   data,
		UserID: notif.UserID,
	}

	if err := s.hub.Publish(notif.UserID, msg); err != nil {
		s.logger.Error("failed to push notification via SSE",
			zap.String("user_id", notif.UserID),
			zap.Error(err),
		)
	}
}

// SendNotification is a convenience method to create and push a notification
// in a single call. Useful for other modules to trigger notifications.
func (s *Service) SendNotification(ctx context.Context, userID string, notifType NotificationType, priority NotificationPriority, title, content, link string) error {
	notif := &Notification{
		UserID:   userID,
		Type:     notifType,
		Priority: priority,
		Title:    title,
		Content:  content,
		Link:     link,
		Sender:   "system",
	}

	_, err := s.CreateNotification(ctx, notif)
	return err
}

// SendGlobalNotification sends a notification to multiple users.
// For bulk notifications, this avoids repeated create+publish cycles.
func (s *Service) SendGlobalNotification(ctx context.Context, userIDs []string, notifType NotificationType, priority NotificationPriority, title, content, link string) error {
	for _, userID := range userIDs {
		if err := s.SendNotification(ctx, userID, notifType, priority, title, content, link); err != nil {
			s.logger.Warn("failed to send notification to user",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}
	return nil
}

// CreateAndBroadcast creates a notification for a user and also sends a global
// SSE event if needed (e.g., for admin dashboards).
func (s *Service) CreateAndBroadcast(ctx context.Context, notif *Notification) error {
	if err := s.repo.Create(ctx, notif); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	// Push to specific user
	if s.hub != nil {
		s.pushSSEEvent(notif)
	}

	return nil
}
