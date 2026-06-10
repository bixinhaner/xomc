package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

// stubStaleLookup 是 StaleTaskLookup 的测试桩，记录 LookupTaskStatuses 的调用次数与入参，
// 用于断言 #16 已把逐条反查折叠成一次批量查询。
type stubStaleLookup struct {
	statuses    map[string]task.TaskStatusInfo
	batchErr    error
	batchCalls  int
	lastTaskIDs []string
}

func (s *stubStaleLookup) LookupTaskStatus(_ context.Context, taskID string) (string, string, bool, error) {
	info, ok := s.statuses[taskID]
	if !ok {
		return "", "", false, nil
	}
	return info.Status, info.ErrorMsg, true, nil
}

func (s *stubStaleLookup) LookupTaskStatuses(_ context.Context, taskIDs []string) (map[string]task.TaskStatusInfo, error) {
	s.batchCalls++
	s.lastTaskIDs = taskIDs
	if s.batchErr != nil {
		return nil, s.batchErr
	}
	out := make(map[string]task.TaskStatusInfo, len(taskIDs))
	for _, id := range taskIDs {
		if info, ok := s.statuses[id]; ok {
			out[id] = info
		}
	}
	return out, nil
}

func dedupPtr(s string) *string { return &s }

func seedStale(repo *mockRepository, userID, dedupKey, title string) *Notification {
	return repo.seed(&Notification{
		UserID:   userID,
		Status:   StatusSent,
		Priority: PriorityNormal,
		Title:    title,
		DedupKey: dedupPtr(dedupKey),
	})
}

// TestSyncStaleByUser_BatchesLookup 成功路径：多条 stale 消息只触发一次批量反查（消除 N+1），
// 且终态/缺失的回灌行为与逐条版本一致。
func TestSyncStaleByUser_BatchesLookup(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	const user = "u1"
	nDone := seedStale(repo, user, "task-done", "下发配置进行中")
	nFail := seedStale(repo, user, "task-fail", "重启进行中")
	nGone := seedStale(repo, user, "task-gone", "采集进行中")
	nRunning := seedStale(repo, user, "task-run", "升级进行中")

	stub := &stubStaleLookup{statuses: map[string]task.TaskStatusInfo{
		"task-done": {Status: "completed"},
		"task-fail": {Status: "failed", ErrorMsg: "9002 内部错误"},
		"task-run":  {Status: "sent"},
		// task-gone 故意缺失 → not-found 路径
	}}
	svc.SetStaleTaskLookup(stub)

	updated, err := svc.SyncStaleByUser(context.Background(), user)
	require.NoError(t, err)

	// 一次批量查询覆盖 4 个去重后的 task_id，而非 4 次单查。
	assert.Equal(t, 1, stub.batchCalls, "应只发一次批量反查")
	assert.ElementsMatch(t,
		[]string{"task-done", "task-fail", "task-gone", "task-run"},
		stub.lastTaskIDs)

	// 回灌条数：completed + failed + not-found(expired) = 3；running 跳过。
	assert.Equal(t, 3, updated)

	got, _ := repo.GetByID(context.Background(), nDone.ID)
	assert.Equal(t, StatusCompleted, got.Status)

	got, _ = repo.GetByID(context.Background(), nFail.ID)
	assert.Equal(t, StatusFailed, got.Status)
	assert.Equal(t, PriorityHigh, got.Priority)
	assert.Equal(t, "错误：9002 内部错误", got.Content)

	got, _ = repo.GetByID(context.Background(), nGone.ID)
	assert.Equal(t, StatusExpired, got.Status)

	got, _ = repo.GetByID(context.Background(), nRunning.ID)
	assert.Equal(t, StatusSent, got.Status, "仍在运行的 task 对应消息不应被改")
}

// TestSyncStaleByUser_DedupTaskIDs 多条消息引用同一 task_id 时，批量入参去重，仍一次查询。
func TestSyncStaleByUser_DedupTaskIDs(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	const user = "u2"
	seedStale(repo, user, "shared-task", "操作 A 进行中")
	seedStale(repo, user, "shared-task", "操作 B 进行中")

	stub := &stubStaleLookup{statuses: map[string]task.TaskStatusInfo{
		"shared-task": {Status: "completed"},
	}}
	svc.SetStaleTaskLookup(stub)

	updated, err := svc.SyncStaleByUser(context.Background(), user)
	require.NoError(t, err)
	assert.Equal(t, 1, stub.batchCalls)
	assert.Equal(t, []string{"shared-task"}, stub.lastTaskIDs, "重复 task_id 应去重")
	assert.Equal(t, 2, updated, "两条消息都应被回灌为 completed")
}

// TestSyncStaleByUser_LookupError 失败路径：批量反查报错 → 整体返回错误，不静默吞掉。
func TestSyncStaleByUser_LookupError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	const user = "u3"
	seedStale(repo, user, "task-x", "进行中")

	stub := &stubStaleLookup{batchErr: errBoom}
	svc.SetStaleTaskLookup(stub)

	updated, err := svc.SyncStaleByUser(context.Background(), user)
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, 0, updated)
}

// TestSyncStaleByUser_NoLookup 未注入查询器 → no-op，不打 DB。
func TestSyncStaleByUser_NoLookup(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	updated, err := svc.SyncStaleByUser(context.Background(), "u4")
	require.NoError(t, err)
	assert.Equal(t, 0, updated)
}
