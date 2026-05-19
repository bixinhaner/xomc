package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// Test_CreateFailureNotifier_WritesFailedMessage 验证 closure 被调时写一条 status=failed 消息。
func Test_CreateFailureNotifier_WritesFailedMessage(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	notifier := NewCreateFailureNotifier(svc, zap.NewNop())

	tk := &task.Task{
		ID: "task-fail-001", DeviceSN: "SN001", Method: "SetParameterValues",
		CreatorID: "alice",
	}
	notifier(context.Background(), tk, "redis connection refused")

	list, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	got := list.Items[0]
	assert.Equal(t, StatusFailed, got.Status)
	assert.Equal(t, PriorityHigh, got.Priority)
	assert.Contains(t, got.Title, "参数设置")
	assert.Contains(t, got.Title, "SN001")
	assert.Contains(t, got.Title, "入队失败")
	assert.Equal(t, "redis connection refused", got.Content)
	assert.Equal(t, "/device/detail/SN001?tab=quickSettings", got.Link)
	require.NotNil(t, got.DedupKey)
	assert.Equal(t, "task-fail-001", *got.DedupKey)
}

// Test_CreateFailureNotifier_SkipsSystemTask 验证 CreatorID 空时跳过。
func Test_CreateFailureNotifier_SkipsSystemTask(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	notifier := NewCreateFailureNotifier(svc, zap.NewNop())

	tk := &task.Task{
		ID: "task-sys-fail", DeviceSN: "SN_SYS", Method: "Reboot",
		CreatorID: "", // 系统任务
	}
	notifier(context.Background(), tk, "any error")

	list, _ := svc.List(context.Background(), NotificationFilter{UserID: ""})
	assert.Empty(t, list.Items, "CreatorID 空的 task 不应写消息")
}

// Test_CreateFailureNotifier_NilTask_NoPanic 验证 nil task 防御性处理。
func Test_CreateFailureNotifier_NilTask_NoPanic(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	notifier := NewCreateFailureNotifier(svc, zap.NewNop())
	assert.NotPanics(t, func() {
		notifier(context.Background(), nil, "boom")
	})
}

// Test_CreateFailureNotifier_RepoErrorLogged 验证 repo 失败时不 panic（仅日志 warn）。
func Test_CreateFailureNotifier_RepoErrorLogged(t *testing.T) {
	repo := newMockRepository()
	repo.upsertErr = errors.New("boom")
	svc := NewService(repo, nil, zap.NewNop())
	notifier := NewCreateFailureNotifier(svc, zap.NewNop())

	tk := &task.Task{
		ID: "task-fail-x", DeviceSN: "SN", Method: "Reboot", CreatorID: "alice",
	}
	assert.NotPanics(t, func() {
		notifier(context.Background(), tk, "queue down")
	})
}

// Test_CreateFailureNotifier_LinkPointsToDeviceDetail 验证链接格式。
func Test_CreateFailureNotifier_LinkPointsToDeviceDetail(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	notifier := NewCreateFailureNotifier(svc, zap.NewNop())

	tk := &task.Task{ID: "t1", DeviceSN: "abc123", Method: "Download", CreatorID: "bob"}
	notifier(context.Background(), tk, "minio unreachable")

	list, _ := svc.List(context.Background(), NotificationFilter{UserID: "bob"})
	require.Len(t, list.Items, 1)
	assert.Equal(t, "/device/detail/abc123?tab=quickSettings", list.Items[0].Link)
}
