package task

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 这个文件用真实 RedisTaskQueue + 真实 PgTaskRepository（依 PG 可达）
// 来覆盖 TaskService 的端到端路径。PG 不可用时所有用例 t.Skip。

// newServiceWithPG 构造一个 queue=miniredis、repo=真实 PG 的 TaskService。
func newServiceWithPG(t *testing.T) (*TaskService, *miniredis.Miniredis, *RedisTaskQueue, *PgTaskRepository) {
	t.Helper()
	pool := newTestPool(t)
	if pool == nil {
		return nil, nil, nil, nil
	}
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	q := NewRedisTaskQueue(client)
	repo := NewPgTaskRepository(pool)
	svc := NewTaskService(q, repo, zap.NewNop())
	return svc, m, q, repo
}

// makeReq 创建一个 SourceID 为有效 UUID 的 CreateTaskRequest。
func makeReq(sn, method string) *CreateTaskRequest {
	return &CreateTaskRequest{
		DeviceSN:    sn,
		Method:      method,
		Params:      json.RawMessage("{}"),
		Priority:    5,
		MaxRetries:  3,
		Source:      TaskSourceAPI,
		Description: "test task",
		SourceID:    generateUUID(),
	}
}

func TestService_PG_CreateTask(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"svccreate", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, tk)
	assert.Equal(t, TaskStatusPending, tk.Status)

	// 应在 queue 里
	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	// 也应在 PG 里
	got2, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got2)
}

func TestService_PG_GetTask_FallbackToPG(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	tk := freshTaskForPG("svcfb", "svcfb")
	require.NoError(t, repo.Create(ctx, tk))

	// 不在 queue → 应 fallback 到 PG
	got, err := svc.GetTask(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tk.ID, got.ID)
}

func TestService_PG_GetTask_NotFoundAnywhere(t *testing.T) {
	svc, _, _, _ := newServiceWithPG(t)
	if svc == nil {
		return
	}
	got, err := svc.GetTask(context.Background(), generateUUID())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_PG_GetTaskByCWMPID_FallbackToPG(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	tk := freshTaskForPG("cwfb", "cwfb")
	tk.CWMPID = "cwmp-svcfb-" + tk.ID[:8]
	require.NoError(t, repo.Create(ctx, tk))

	got, err := svc.GetTaskByCWMPID(ctx, tk.CWMPID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tk.ID, got.ID)
}

func TestService_PG_GetPendingTasks(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	sn := testDeviceSNPrefix + "svcpending"
	for i := 0; i < 3; i++ {
		req := makeReq(sn, "Reboot")
		_, err := svc.CreateTask(ctx, req)
		require.NoError(t, err)
	}

	tasks, err := svc.GetPendingTasks(ctx, sn, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 3)

	// 带 limit
	tasks2, err := svc.GetPendingTasks(ctx, sn, 2)
	require.NoError(t, err)
	assert.Len(t, tasks2, 2)
}

func TestService_PG_MarkTaskSent(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"svcmark", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)

	cwmpID := "cwmp-svc-mark-" + tk.ID[:8]
	require.NoError(t, svc.MarkTaskSent(ctx, tk.ID, cwmpID))

	// PG 应已同步
	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusSent, got.Status)
	assert.Equal(t, cwmpID, got.CWMPID)
}

func TestService_PG_MarkTaskCompleted(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"svcdone", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)

	require.NoError(t, svc.MarkTaskCompleted(ctx, tk.ID, json.RawMessage(`{"ok":true}`)))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusCompleted, got.Status)
}

func TestService_PG_MarkTaskFailed(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"svcfail", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)

	require.NoError(t, svc.MarkTaskFailed(ctx, tk.ID, 9001, "boom"))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusFailed, got.Status)
	assert.Equal(t, 9001, got.ErrorCode)
}

func TestService_PG_CancelTask(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"svccan", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)

	require.NoError(t, svc.CancelTask(ctx, tk.ID))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusCancelled, got.Status)
}

func TestService_PG_CancelTask_NotFound(t *testing.T) {
	svc, _, _, _ := newServiceWithPG(t)
	if svc == nil {
		return
	}
	err := svc.CancelTask(context.Background(), generateUUID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestService_PG_GetTaskHistory(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	sn := testDeviceSNPrefix + "svchist"
	for i := 0; i < 3; i++ {
		_, err := svc.CreateTask(ctx, makeReq(sn, "Reboot"))
		require.NoError(t, err)
	}

	resp, err := svc.GetTaskHistory(ctx, sn, &TaskHistoryOptions{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, int64(3))
	assert.NotEmpty(t, resp.Tasks)
}

func TestService_PG_GetTaskStats(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	sn := testDeviceSNPrefix + "svcstats"
	_, err := svc.CreateTask(ctx, makeReq(sn, "Reboot"))
	require.NoError(t, err)

	stats, err := svc.GetTaskStats(ctx, sn)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats[TaskStatusPending], int64(1))
}

func TestService_PG_RetryTask(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"svcretry", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)

	tk.RetryCount = 1
	require.NoError(t, svc.RetryTask(ctx, tk))

	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.RetryCount)
}

func TestService_PG_RetryTask_NilTask(t *testing.T) {
	svc, _, _, _ := newServiceWithPG(t)
	if svc == nil {
		return
	}
	err := svc.RetryTask(context.Background(), nil)
	require.Error(t, err)
}

func TestService_PG_BatchCreateTasks(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	reqs := []*CreateTaskRequest{
		makeReq(testDeviceSNPrefix+"svcbat", "Reboot"),
		makeReq(testDeviceSNPrefix+"svcbat", "GetParameterValues"),
	}
	tasks, err := svc.BatchCreateTasks(ctx, reqs)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestService_PG_PurgeOldTasks(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	tk := freshTaskForPG("svcpurge", "svcpurge")
	tk.Status = TaskStatusCompleted
	completedAt := time.Now()
	tk.CompletedAt = &completedAt
	require.NoError(t, repo.Create(ctx, tk))

	// retentionDays=-1 → before = now+1day → 命中
	count, err := svc.PurgeOldTasks(ctx, -1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1))
}

func TestService_PG_RecoverPendingTasks(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	sn := testDeviceSNPrefix + "svcrec"
	// 创建一个 sent 状态 + sentAt 较旧的任务，直接进 queue
	tk := freshTaskForPG("rec1", "rec1")
	tk.DeviceSN = sn
	tk.Status = TaskStatusSent
	stale := time.Now().Add(-10 * time.Minute)
	tk.SentAt = &stale
	tk.RetryCount = 0
	tk.MaxRetries = 3
	require.NoError(t, repo.Create(ctx, tk))
	require.NoError(t, q.Push(ctx, tk))

	require.NoError(t, svc.RecoverPendingTasks(ctx, sn))

	// 重置后任务应回到 pending，retry+1
	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusPending, got.Status)
	assert.Equal(t, 1, got.RetryCount)
}

func TestService_PG_RecoverPendingTasks_Exhausted(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	sn := testDeviceSNPrefix + "svcrec2"
	tk := freshTaskForPG("rec2", "rec2")
	tk.DeviceSN = sn
	tk.Status = TaskStatusSent
	stale := time.Now().Add(-10 * time.Minute)
	tk.SentAt = &stale
	tk.RetryCount = 3
	tk.MaxRetries = 3
	require.NoError(t, repo.Create(ctx, tk))
	require.NoError(t, q.Push(ctx, tk))

	require.NoError(t, svc.RecoverPendingTasks(ctx, sn))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusFailed, got.Status)
}

func TestService_PG_RestorePendingQueues(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	tk := freshTaskForPG("restpub", "restpub")
	require.NoError(t, repo.Create(ctx, tk))

	stats, err := svc.RestorePendingQueues(ctx, 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.Scanned, 1)
}
