package task

import (
	"context"
	"encoding/json"
	"strings"
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
	mr := 3
	return &CreateTaskRequest{
		DeviceSN:    sn,
		Method:      method,
		Params:      json.RawMessage("{}"),
		Priority:    5,
		MaxRetries:  &mr,
		Source:      TaskSourceAPI,
		Description: "test task",
		SourceID:    generateUUID(),
	}
}

func createParamSyncRunForTaskTest(t *testing.T, repo *PgTaskRepository, sn, status string) string {
	t.Helper()
	requestID, runID, deviceID := generateUUID(), generateUUID(), generateUUID()
	_, err := repo.pool.Exec(context.Background(), `INSERT INTO parameter_sync_requests
(id, device_id, device_sn, trigger_reason, sync_scope, status)
VALUES ($1, $2, $3, 'manual', 'full', 'running')`, requestID, deviceID, sn)
	require.NoError(t, err)
	_, err = repo.pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status)
VALUES ($1, $2, $3, $4, 'manual', 'full', $5)`, runID, requestID, deviceID, sn, status)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = repo.pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, requestID)
	})
	return runID
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

func TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	tk := freshTaskForPG("terminal-tombstone", "terminal-tombstone")
	largeParams, err := json.Marshal(map[string]string{"payload": strings.Repeat("p", 8*1024)})
	require.NoError(t, err)
	largeResult, err := json.Marshal(map[string]string{"payload": strings.Repeat("r", 16*1024)})
	require.NoError(t, err)
	tk.Params = largeParams
	tk.MarkCompleted(largeResult)
	require.NoError(t, repo.Create(ctx, tk))
	require.NoError(t, q.Update(ctx, tk))
	require.False(t, q.client.HExists(ctx, q.taskKey(tk.ID), "data").Val())

	got, err := svc.GetTask(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, tk.ID, got.ID)
	require.Equal(t, TaskStatusCompleted, got.Status)
	require.JSONEq(t, string(tk.Params), string(got.Params))
	require.JSONEq(t, string(tk.Result), string(got.Result))
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

func TestService_PG_GetQueueLength_IgnoresStaleRedisEntries(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()
	sn := testDeviceSNPrefix + "ql-stale"

	stale := freshTaskForPG("ql-stale-expired", "ql-stale")
	stale.Status = TaskStatusExpired
	past := time.Now().Add(-time.Hour)
	stale.ExpiresAt = &past
	stale.CompletedAt = &past
	require.NoError(t, repo.Create(ctx, stale))
	require.NoError(t, q.Push(ctx, stale))

	length, err := svc.GetQueueLength(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(0), length, "已终态/过期任务即使残留在 Redis 队列，也不应让同步状态保持 syncing")

	active := freshTaskForPG("ql-active", "ql-stale")
	future := time.Now().Add(time.Hour)
	active.ExpiresAt = &future
	require.NoError(t, repo.Create(ctx, active))
	require.NoError(t, q.Push(ctx, active))

	length, err = svc.GetQueueLength(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(1), length)
}

func TestService_PG_CountOpenSyncGPVByDevice_ScopesToRecentSyncTasks(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()
	sn := testDeviceSNPrefix + "sync-count"
	failedSourceID := generateUUID()

	pmTask := freshTaskForPG("pm", "sync-count")
	pmTask.Method = "SetParameterValues"
	pmTask.CommandKey = "pm_upload_setup_on_online"
	pmExpires := time.Now().Add(time.Hour)
	pmTask.ExpiresAt = &pmExpires
	require.NoError(t, repo.Create(ctx, pmTask))

	staleSync := freshTaskForPG("stale-sync", "sync-count")
	staleSync.Method = "GetParameterValues"
	staleSync.CommandKey = "sync-gpv-" + sn + "-0"
	staleSync.Status = TaskStatusSent
	staleSync.CreatedAt = time.Now().Add(-48 * time.Hour)
	sentAt := staleSync.CreatedAt
	staleSync.SentAt = &sentAt
	require.NoError(t, repo.Create(ctx, staleSync))

	orphanSync := freshTaskForPG("orphan-sync", "sync-count")
	orphanSync.Method = "GetParameterValues"
	orphanSync.CommandKey = "sync-gpv-" + sn + "-0-r"
	orphanSync.Status = TaskStatusSent
	orphanSync.ExpiresAt = nil
	orphanSync.CreatedAt = time.Now().Add(-2 * time.Hour)
	orphanSentAt := orphanSync.CreatedAt
	orphanSync.SentAt = &orphanSentAt
	require.NoError(t, repo.Create(ctx, orphanSync))

	oldFailedSync := freshTaskForPG("old-failed-sync", "sync-count")
	oldFailedSync.Method = "GetParameterValues"
	oldFailedSync.CommandKey = "sync-gpv-" + sn + "-1"
	oldFailedSync.Status = TaskStatusExpired
	oldFailedSync.SourceID = failedSourceID
	oldFailedSync.CreatedAt = time.Now().Add(-2 * time.Hour)
	failedAt := oldFailedSync.CreatedAt.Add(time.Minute)
	oldFailedSync.CompletedAt = &failedAt
	require.NoError(t, repo.Create(ctx, oldFailedSync))

	count, err := svc.CountOpenSyncGPVByDevice(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	activeSync := freshTaskForPG("active-sync", "sync-count")
	activeSync.Method = "GetParameterValues"
	activeSync.CommandKey = "sync-gpv-" + sn + "-2"
	activeSync.ExpiresAt = &pmExpires
	activeSync.SourceID = failedSourceID
	require.NoError(t, repo.Create(ctx, activeSync))

	count, err = svc.CountOpenSyncGPVByDevice(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	activeNewBatch := freshTaskForPG("active-new-batch", "sync-count")
	activeNewBatch.Method = "GetParameterValues"
	activeNewBatch.CommandKey = "sync-gpv-" + sn + "-3"
	activeNewBatch.ExpiresAt = &pmExpires
	activeNewBatch.SourceID = generateUUID()
	require.NoError(t, repo.Create(ctx, activeNewBatch))

	count, err = svc.CountOpenSyncGPVByDevice(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestService_PG_CountOpenSyncGPVByDevice_IncludesDurableParamSync(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()
	sn := testDeviceSNPrefix + "durable-sync-count"

	active := freshTaskForPG("active-param-sync", "durable-sync-count")
	active.Method = "GetParameterValues"
	active.CommandKey = "param-sync-" + generateUUID() + "-0"
	active.Source = TaskSourceParamSync
	active.SourceID = createParamSyncRunForTaskTest(t, repo, sn, "executing")
	expires := time.Now().Add(time.Hour)
	active.ExpiresAt = &expires
	require.NoError(t, repo.Create(ctx, active))

	count, err := svc.CountOpenSyncGPVByDevice(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestService_PG_TerminalParamSyncRunCannotRemainOpenOrBeSent(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()
	sn := testDeviceSNPrefix + "terminal-param-sync"

	orphan := freshTaskForPG("terminal-param-sync", "terminal-param-sync")
	orphan.Method = "GetParameterValues"
	orphan.CommandKey = "param-sync-" + generateUUID() + "-0-r"
	orphan.Source = TaskSourceParamSync
	orphan.SourceID = createParamSyncRunForTaskTest(t, repo, sn, "failed")
	expires := time.Now().Add(30 * time.Minute)
	orphan.ExpiresAt = &expires
	require.NoError(t, repo.Create(ctx, orphan))

	count, err := svc.CountOpenSyncGPVByDevice(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "终态 durable run 的竞态遗留 task 不应让参数树保持同步中")

	acquired, err := repo.MarkSentIfPending(ctx, orphan.ID, "cwmp-terminal-run", time.Now())
	require.NoError(t, err)
	assert.False(t, acquired, "终态 durable run 的遗留 task 不得再下发设备")
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
	before, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, before)
	require.Equal(t, TaskStatusPending, before.Status)
	require.Equal(t, TaskSourceAPI, before.Source)
	var fenceAllows bool
	require.NoError(t, repo.pool.QueryRow(ctx, `SELECT COALESCE(source, '') <> 'param_sync' OR EXISTS (
SELECT 1 FROM parameter_sync_runs r WHERE r.id=device_tasks.source_id
  AND r.status IN ('planning','enqueuing','waiting_device','executing','processing'))
FROM device_tasks WHERE id=$1`, tk.ID).Scan(&fenceAllows))
	require.True(t, fenceAllows)

	cwmpID := "cwmp-svc-mark-" + tk.ID[:8]
	require.NoError(t, svc.MarkTaskSent(ctx, tk.ID, cwmpID))

	// PG 应已同步
	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusSent, got.Status)
	assert.Equal(t, cwmpID, got.CWMPID)
}

func TestService_PG_MarkTaskSentReleasesFenceWhenRedisFailsBeforeRPCWrite(t *testing.T) {
	svc, redisServer, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"send-fence-redis-fail", "Reboot")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)
	popped, err := svc.PopTask(ctx, tk.DeviceSN)
	require.NoError(t, err)
	require.NotNil(t, popped)
	redisServer.Close()

	err = svc.MarkTaskSent(ctx, tk.ID, "cwmp-before-write-failure")
	require.Error(t, err)

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusPending, got.Status)
	assert.Empty(t, got.CWMPID)
	assert.Nil(t, got.SentAt)
}

func TestPGReleaseSentClaimDoesNotReviveTerminalTask(t *testing.T) {
	_, _, _, repo := newServiceWithPG(t)
	if repo == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()
	tk := freshTaskForPG("release-terminal", "release-terminal")
	require.NoError(t, repo.Create(ctx, tk))
	cwmpID := "cwmp-release-terminal"
	acquired, err := repo.MarkSentIfPending(ctx, tk.ID, cwmpID, time.Now())
	require.NoError(t, err)
	require.True(t, acquired)
	_, err = repo.pool.Exec(ctx, `UPDATE device_tasks SET status='cancelled', completed_at=now() WHERE id=$1`, tk.ID)
	require.NoError(t, err)

	released, err := repo.ReleaseSentClaimIfUnwritten(ctx, tk.ID, cwmpID)

	require.NoError(t, err)
	assert.False(t, released)
	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusCancelled, got.Status)
}

func TestService_PG_MarkTaskSentDropsRedisCopyWhenFenceRejects(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	req := makeReq(testDeviceSNPrefix+"send-fence-stale", "GetParameterValues")
	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)
	_, err = repo.pool.Exec(ctx, `UPDATE device_tasks SET status='failed', completed_at=now(), error_code=9005,
error_message='run already failed' WHERE id=$1`, tk.ID)
	require.NoError(t, err)

	err = svc.MarkTaskSent(ctx, tk.ID, "cwmp-stale-fence")
	require.ErrorContains(t, err, "no longer pending")
	queued, getErr := q.GetByID(ctx, tk.ID)
	require.NoError(t, getErr)
	assert.Nil(t, queued, "被 PG fence 拒绝的 stale Redis task 必须移除，避免设备每次会话重复命中")
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

func TestService_PG_MarkTaskFailed_AutoRetriesMML(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	mr := 2
	req := makeReq(testDeviceSNPrefix+"svcmmlretry", "Reboot")
	req.Source = TaskSourceMML
	req.MaxRetries = &mr
	req.RetryIntervalSeconds = 60

	tk, err := svc.CreateTask(ctx, req)
	require.NoError(t, err)
	require.NoError(t, q.MarkTaskSent(ctx, tk.ID, "cwmp-mml-retry"))

	require.NoError(t, svc.MarkTaskFailed(ctx, tk.ID, 9001, "boom"))

	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusPending, got.Status)
	assert.Equal(t, 1, got.RetryCount)
	assert.Equal(t, 60, got.RetryIntervalSeconds)
	require.NotNil(t, got.NextAttemptAt)
	assert.True(t, got.NextAttemptAt.After(time.Now()))

	popped, err := q.Pop(ctx, req.DeviceSN)
	require.NoError(t, err)
	assert.Nil(t, popped, "MML failed retry must wait until next_attempt_at before popping")
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
	tk.CreatedAt = time.Now().AddDate(-11, 0, 0)
	completedAt := time.Now().AddDate(-11, 0, 0)
	tk.CompletedAt = &completedAt
	require.NoError(t, repo.Create(ctx, tk))

	// Use an old retention boundary so this integration test does not purge
	// fresh local E2E device_tasks from the shared dev database.
	count, err := svc.PurgeOldTasks(ctx, 3650)
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
	// 创建一个 sent 状态 + sentAt 较旧的任务。真实 PopTask 后任务已不在 Redis 队列 ZSET，
	// 恢复逻辑必须能只凭 PG 记录把它放回队列。
	tk := freshTaskForPG("rec1", "rec1")
	tk.DeviceSN = sn
	tk.Status = TaskStatusSent
	stale := time.Now().Add(-10 * time.Minute)
	tk.SentAt = &stale
	tk.RetryCount = 0
	tk.MaxRetries = 3
	require.NoError(t, repo.Create(ctx, tk))

	require.NoError(t, svc.RecoverPendingTasks(ctx, sn))

	// 重置后任务应回到 pending，retry+1
	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusPending, got.Status)
	assert.Equal(t, 1, got.RetryCount)
}

func TestService_PG_RecoverPendingTasks_RecoversFreshSentOnNewInform(t *testing.T) {
	svc, _, q, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	sn := testDeviceSNPrefix + "svcrec-fresh"
	tk := freshTaskForPG("rec-fresh", "rec-fresh")
	tk.DeviceSN = sn
	tk.Status = TaskStatusSent
	now := time.Now()
	tk.SentAt = &now
	tk.RetryCount = 0
	tk.MaxRetries = 3
	require.NoError(t, repo.Create(ctx, tk))

	require.NoError(t, svc.RecoverPendingTasks(ctx, sn))

	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusPending, got.Status)
	assert.Equal(t, 1, got.RetryCount)
	assert.Nil(t, got.SentAt)

	pgGot, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, pgGot)
	assert.Equal(t, TaskStatusPending, pgGot.Status)
	assert.Equal(t, 1, pgGot.RetryCount)
	assert.Nil(t, pgGot.SentAt)
}

func TestService_PG_RecoverPendingTasks_Exhausted(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
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
