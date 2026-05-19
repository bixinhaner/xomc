package notification

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
)

// Test_TaskSubscriber_HandleCreated_WritesQueued 验证 task.created 事件触发 status=queued 消息。
func Test_TaskSubscriber_HandleCreated_WritesQueued(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	sub := NewTaskSubscriber(svc, zap.NewNop())

	tk := &task.Task{
		ID:        "task-001",
		DeviceSN:  "SN001",
		Method:    "SetParameterValues",
		CreatorID: "alice",
		Status:    task.TaskStatusPending,
		Params:    json.RawMessage(`{"values":[{"name":"TAC","value":"4"}]}`),
	}
	evt, err := event.NewEvent(event.SubjectTaskCreated, tk)
	require.NoError(t, err)

	require.NoError(t, sub.handleCreated(context.Background(), evt))

	list, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	got := list.Items[0]
	assert.Equal(t, StatusQueued, got.Status)
	assert.Equal(t, PriorityNormal, got.Priority)
	assert.Contains(t, got.Title, "参数设置")
	assert.Contains(t, got.Title, "SN001")
	assert.Contains(t, got.Title, "进行中")
	assert.Contains(t, got.Title, "1 项")
	assert.Contains(t, got.Content, "TAC = 4")
	assert.Equal(t, "/device/detail/SN001?tab=quickSettings", got.Link)
	require.NotNil(t, got.DedupKey)
	assert.Equal(t, "task-001", *got.DedupKey)
}

// Test_TaskSubscriber_Upgrade_QueuedToCompleted 验证同一 task 的 created → completed 升级走 upsert。
func Test_TaskSubscriber_Upgrade_QueuedToCompleted(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	sub := NewTaskSubscriber(svc, zap.NewNop())

	tk := &task.Task{
		ID:        "task-002",
		DeviceSN:  "SN002",
		Method:    "SetParameterValues",
		CreatorID: "bob",
		Status:    task.TaskStatusPending,
		Params:    json.RawMessage(`{"values":[{"name":"PCI","value":"5"}]}`),
	}

	// Step 1: created
	evtCreated, _ := event.NewEvent(event.SubjectTaskCreated, tk)
	require.NoError(t, sub.handleCreated(context.Background(), evtCreated))

	// Step 2: completed
	tk.Status = task.TaskStatusCompleted
	completed := time.Now()
	tk.CompletedAt = &completed
	evtCompleted, _ := event.NewEvent(event.SubjectTaskCompleted, tk)
	require.NoError(t, sub.handleCompleted(context.Background(), evtCompleted))

	// 只有一条消息（同 dedup_key upsert）
	list, err := svc.List(context.Background(), NotificationFilter{UserID: "bob"})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, StatusCompleted, list.Items[0].Status)
	assert.Contains(t, list.Items[0].Title, "已完成")
}

// Test_TaskSubscriber_HandleFailed_MapsExpiredAndCancelled 验证 task.failed 主题三态分流。
func Test_TaskSubscriber_HandleFailed_MapsExpiredAndCancelled(t *testing.T) {
	cases := []struct {
		name      string
		taskStat  task.TaskStatus
		wantNStat NotificationStatus
		wantTitle string
	}{
		{"failed", task.TaskStatusFailed, StatusFailed, "失败"},
		{"expired", task.TaskStatusExpired, StatusExpired, "超时"},
		{"cancelled", task.TaskStatusCancelled, StatusCancelled, "已取消"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo, nil, zap.NewNop())
			sub := NewTaskSubscriber(svc, zap.NewNop())

			tk := &task.Task{
				ID:           "task-" + c.name,
				DeviceSN:     "SN_X",
				Method:       "Reboot",
				CreatorID:    "alice",
				Status:       c.taskStat,
				ErrorMessage: "boom",
			}
			evt, _ := event.NewEvent(event.SubjectTaskFailed, tk)
			require.NoError(t, sub.handleFailed(context.Background(), evt))

			list, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			got := list.Items[0]
			assert.Equal(t, c.wantNStat, got.Status)
			assert.Contains(t, got.Title, c.wantTitle)
			// failed/expired 升级 priority；cancelled 保留 normal
			if c.taskStat == task.TaskStatusCancelled {
				assert.Equal(t, PriorityNormal, got.Priority)
			} else {
				assert.Equal(t, PriorityHigh, got.Priority)
				assert.Contains(t, got.Content, "boom")
			}
		})
	}
}

// Test_TaskSubscriber_SkipsTaskWithoutCreator 验证 CreatorID 空时跳过（系统任务）。
func Test_TaskSubscriber_SkipsTaskWithoutCreator(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	sub := NewTaskSubscriber(svc, zap.NewNop())

	tk := &task.Task{
		ID:        "task-sys",
		DeviceSN:  "SN_SYS",
		Method:    "GetParameterValues",
		CreatorID: "", // 系统任务（PeriodicSyncer 等）
		Status:    task.TaskStatusCompleted,
	}
	evt, _ := event.NewEvent(event.SubjectTaskCompleted, tk)
	require.NoError(t, sub.handleCompleted(context.Background(), evt))

	list, err := svc.List(context.Background(), NotificationFilter{UserID: ""})
	require.NoError(t, err)
	assert.Empty(t, list.Items, "CreatorID 空的 task 不应写入消息中心")
}

// Test_TaskSubscriber_MultiUserIsolation 验证按 CreatorID 隔离写入。
func Test_TaskSubscriber_MultiUserIsolation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	sub := NewTaskSubscriber(svc, zap.NewNop())

	for _, c := range []struct {
		taskID, user string
	}{
		{"task-alice-1", "alice"},
		{"task-alice-2", "alice"},
		{"task-bob-1", "bob"},
	} {
		tk := &task.Task{
			ID: c.taskID, DeviceSN: "SN", Method: "Reboot",
			CreatorID: c.user, Status: task.TaskStatusCompleted,
		}
		evt, _ := event.NewEvent(event.SubjectTaskCompleted, tk)
		require.NoError(t, sub.handleCompleted(context.Background(), evt))
	}

	aliceList, _ := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	bobList, _ := svc.List(context.Background(), NotificationFilter{UserID: "bob"})
	assert.Len(t, aliceList.Items, 2)
	assert.Len(t, bobList.Items, 1)
}

// Test_TaskSubscriber_MultiParam_DetailFormat 验证多参数详情格式 + 5 行截断。
func Test_TaskSubscriber_MultiParam_DetailFormat(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	sub := NewTaskSubscriber(svc, zap.NewNop())

	// 7 项参数 → 应只列前 5 项 + "...共 7 项"
	params := `{"values":[
		{"name":"TAC","value":"4"},
		{"name":"PCI","value":"5"},
		{"name":"P3","value":"v3"},
		{"name":"P4","value":"v4"},
		{"name":"P5","value":"v5"},
		{"name":"P6","value":"v6"},
		{"name":"P7","value":"v7"}
	]}`
	tk := &task.Task{
		ID: "task-multi", DeviceSN: "SN", Method: "SetParameterValues",
		CreatorID: "alice", Status: task.TaskStatusPending,
		Params: json.RawMessage(params),
	}
	evt, _ := event.NewEvent(event.SubjectTaskCreated, tk)
	require.NoError(t, sub.handleCreated(context.Background(), evt))

	list, _ := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.Len(t, list.Items, 1)
	got := list.Items[0]
	assert.Contains(t, got.Title, "7 项")
	// 详情区前 5 行 + 截断提示
	lines := splitLines(got.Content)
	assert.GreaterOrEqual(t, len(lines), 6)
	assert.Contains(t, lines[5], "...共 7 项")
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(c)
		}
	}
	out = append(out, cur)
	return out
}
